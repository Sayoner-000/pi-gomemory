package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// HCD-039: seguimiento de los paquetes de Pi fijados (plugin.go) frente a npm.
// El pin sigue siendo la única versión que se instala (HCD-005); aquí solo se
// calcula qué publica npm para avisar de lo que pi-gomemory aún no validó.

// maxRegistryDoc acota el documento abreviado de un paquete. Crece con cada
// versión publicada, así que no se usa httpJSON, que corta en 4 MiB.
const maxRegistryDoc = 16 << 20

// npmInstallAccept pide el documento abreviado del registro, que incluye
// versions.<v>.dist con el tarball y su integridad.
const npmInstallAccept = "application/vnd.npm.install-v1+json"

// RejectedVersion es una versión de npm revisada y descartada a propósito.
type RejectedVersion struct {
	Package string
	Version string
	Reason  string
}

// RejectedVersions son las versiones revisadas y descartadas, con su motivo
// (HCD-039, examen socrático P1). Una versión rechazada no cuenta como desfase
// en pins-check ni en doctor; una más nueva vuelve a contar. Para registrar un
// rechazo, añade aquí paquete, versión y el contrato o la razón del descarte.
var RejectedVersions = []RejectedVersion{}

// PinnedPackage es un paquete de Pi que pi-gomemory instala con versión exacta.
type PinnedPackage struct {
	Name    string
	Version string
}

// PinnedPackages devuelve los paquetes fijados en plugin.go, con su pin.
func PinnedPackages() []PinnedPackage {
	return []PinnedPackage{
		{"pi-cc-extensions", CCExtensionsVersion},
	}
}

// rejectedAbove devuelve, de menor a mayor, las versiones de pkg publicadas por
// encima del pin que están en rejected.
func rejectedAbove(rejected []RejectedVersion, pkg, pin string, versions []string) []RejectedVersion {
	var out []RejectedVersion
	for _, r := range rejected {
		if r.Package != pkg || !IsNewer(r.Version, pin) {
			continue
		}
		for _, v := range versions {
			if v == r.Version {
				out = append(out, r)
				break
			}
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && IsNewer(out[j-1].Version, out[j].Version); j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// registryDist es el bloque dist de una versión en el documento abreviado.
type registryDist struct {
	Tarball   string `json:"tarball"`
	Integrity string `json:"integrity"`
}

type registryDoc struct {
	Versions map[string]struct {
		Dist registryDist `json:"dist"`
	} `json:"versions"`
}

var registryClient = &http.Client{Timeout: depsProbeTimeout}

// fetchRegistryDoc pide el documento abreviado de pkg a NPMRegistryURL.
func fetchRegistryDoc(ctx context.Context, pkg string) (*registryDoc, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, NPMRegistryURL+"/"+pkg, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", npmInstallAccept)
	req.Header.Set("User-Agent", "pi-gomemory-setup")
	resp, err := registryClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", pkg, resp.StatusCode)
	}
	// Un byte de más distingue «cabe justo» de «excede»: truncar daría un JSON
	// ilegible en vez de un error que explique la causa.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRegistryDoc+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxRegistryDoc {
		return nil, fmt.Errorf("%s: documento del registro mayor que 16 MiB; se descarta", pkg)
	}
	var doc registryDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("%s: documento del registro ilegible: %w", pkg, err)
	}
	if len(doc.Versions) == 0 {
		return nil, fmt.Errorf("%s: el registro no devuelve versions", pkg)
	}
	return &doc, nil
}

// registryVersions devuelve las versiones publicadas de pkg.
func registryVersions(ctx context.Context, pkg string) ([]string, error) {
	doc, err := fetchRegistryDoc(ctx, pkg)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(doc.Versions))
	for v := range doc.Versions {
		out = append(out, v)
	}
	return out, nil
}

// publishedVersion es la versión estable más alta de versions que no está en
// rejected; puede ser igual al pin. Es un error que no haya versiones, que no
// haya ninguna estable o que el pin no figure entre las publicadas: en ningún
// caso se puede afirmar que el pin está al día.
func publishedVersion(pkg string, versions []string, pin string, rejected []RejectedVersion) (string, error) {
	if len(versions) == 0 {
		return "", fmt.Errorf("%s: el registro no devuelve versions", pkg)
	}
	pinFound := false
	best := ""
	for _, v := range versions {
		if v == pin {
			pinFound = true
		}
		if _, ok := parseSemver(v); !ok || isRejected(rejected, pkg, v) {
			continue
		}
		if best == "" || IsNewer(v, best) {
			best = v
		}
	}
	if !pinFound {
		return "", fmt.Errorf("%s: la versión fijada %s no aparece en el registro", pkg, pin)
	}
	if best == "" {
		return "", fmt.Errorf("%s: el registro no publica ninguna versión estable", pkg)
	}
	return best, nil
}

// npmPublished es el DepProbe.Published de un paquete fijado: la misma regla
// que pins-check, guardada en la caché para que doctor avise sin red.
func npmPublished(pkg, pin string) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		versions, err := registryVersions(ctx, pkg)
		if err != nil {
			return "", err
		}
		return publishedVersion(pkg, versions, pin, RejectedVersions)
	}
}

// pinsDoctorItems resume para doctor, sin red, lo que la caché de dependencias
// dice de cada paquete fijado frente a npm (HCD-039, Q1). Compara con el pin
// compilado en el binario, no con lo guardado en la caché.
func pinsDoctorItems(cachePath string) []CheckItem {
	var c depsCache
	data, err := os.ReadFile(cachePath)
	if err == nil && json.Unmarshal(data, &c) != nil {
		c = depsCache{}
	}
	noData := "sin datos de npm; ejecuta pi-gomemory-setup deps-refresh"
	if UpdateCheckDisabled() {
		noData = "sin datos de npm (comprobación desactivada)"
	}
	checked := c.CheckedAt.Local().Format("2006-01-02 15:04")
	var items []CheckItem
	for _, p := range PinnedPackages() {
		item := CheckItem{Name: "Paquete fijado " + p.Name, Status: "INFO", Details: noData}
		var st DepState
		for _, s := range c.Deps {
			if s.Name == p.Name {
				st = s
			}
		}
		var rejected []string
		for _, r := range RejectedVersions {
			if r.Package == p.Name && IsNewer(r.Version, p.Version) {
				rejected = append(rejected, fmt.Sprintf("%s rechazada: %s", r.Version, r.Reason))
			}
		}
		switch {
		case st.Published == "":
		case IsNewer(st.Published, p.Version):
			item.Status = "WARN"
			item.Details = fmt.Sprintf("fijado %s · npm publica %s, no validada por pi-gomemory (consulta %s); revísala con quien mantiene pi-gomemory", p.Version, st.Published, checked)
		case len(rejected) > 0:
			item.Details = fmt.Sprintf("fijado %s · %s (consulta %s)", p.Version, strings.Join(rejected, "; "), checked)
		default:
			item.Status = "OK"
			item.Details = fmt.Sprintf("%s al día con npm (consulta %s)", p.Version, checked)
		}
		items = append(items, item)
	}
	return items
}

// PinCheck es el resultado de comparar un paquete fijado con npm.
type PinCheck struct {
	Package   string
	Pin       string
	Published string
	// Rejected son las versiones por encima del pin que están en RejectedVersions.
	Rejected []RejectedVersion
	Err      error
}

// Stale indica que npm publica una versión estable no rechazada por encima del pin.
func (c PinCheck) Stale() bool {
	return c.Err == nil && IsNewer(c.Published, c.Pin)
}

// CheckPins compara cada paquete fijado con su lista de versiones en npm.
func CheckPins(ctx context.Context) []PinCheck {
	var out []PinCheck
	for _, p := range PinnedPackages() {
		c := PinCheck{Package: p.Name, Pin: p.Version}
		versions, err := registryVersions(ctx, p.Name)
		if err == nil {
			c.Published, err = publishedVersion(p.Name, versions, p.Version, RejectedVersions)
			c.Rejected = rejectedAbove(RejectedVersions, p.Name, p.Version, versions)
		}
		c.Err = err
		out = append(out, c)
	}
	return out
}

// pinsNextSteps explica cómo seguir cuando pins-check detiene la release.
const pinsNextSteps = "Para seguir: valida el candidato con " +
	"PINS_CANDIDATE=<paquete>@<versión> go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$', " +
	"regístralo como rechazado en RejectedVersions (pkg/installer/pins.go) " +
	"o publica igualmente con PI_GOMEMORY_ALLOW_STALE_PINS=1."

// FormatPinChecks redacta el informe de pins-check. fail es verdadero si algún
// paquete va atrás o no se pudo consultar y no se aceptó con allowStale.
func FormatPinChecks(checks []PinCheck, allowStale bool) (lines []string, fail bool) {
	accept := func(s string) string {
		if allowStale {
			return "aceptado: " + s
		}
		fail = true
		return s
	}
	for _, c := range checks {
		for _, r := range c.Rejected {
			lines = append(lines, fmt.Sprintf("%s: %s rechazada: %s", c.Package, r.Version, r.Reason))
		}
		switch {
		case c.Err != nil:
			if allowStale {
				lines = append(lines, fmt.Sprintf("aceptado: %s: no comprobado: error de red (%v)", c.Package, c.Err))
			} else {
				fail = true
				lines = append(lines, fmt.Sprintf("%s: error consultando npm: %v", c.Package, c.Err))
			}
		case c.Stale():
			lines = append(lines, accept(fmt.Sprintf("%s: fijado %s · npm publica %s sin validar", c.Package, c.Pin, c.Published)))
		default:
			lines = append(lines, fmt.Sprintf("%s: al día (%s)", c.Package, c.Pin))
		}
	}
	if fail {
		lines = append(lines, pinsNextSteps)
	}
	return lines, fail
}

func isRejected(rejected []RejectedVersion, pkg, v string) bool {
	for _, r := range rejected {
		if r.Package == pkg && r.Version == v {
			return true
		}
	}
	return false
}

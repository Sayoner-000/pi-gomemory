package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	depsCacheFile    = "pi-gomemory-deps.json"
	depsProbeTimeout = 20 * time.Second
	// depsCacheTTL limita las consultas en red a una por hora: una release se
	// anuncia como mucho una hora después de publicarse (HCD-031) y las ~7
	// consultas quedan lejos del límite de la API de GitHub sin autenticar.
	depsCacheTTL = time.Hour
	// liveVersionTimeout acota la relectura local de la versión instalada,
	// que corre en el arranque del launcher antes de abrir Pi.
	liveVersionTimeout = 3 * time.Second
)

// DepProbe describe un componente del meta-harness: cómo leer su versión
// instalada, cómo conocer la última publicada y cómo actualizarlo.
type DepProbe struct {
	Name    string
	Current func(ctx context.Context) (string, error)
	Latest  func(ctx context.Context) (string, error)
	Update  []string
	// Manual, si no está vacío, es la instrucción para actualizar a mano: el
	// componente no tiene comando automático seguro.
	Manual string
	// Published, opcional, devuelve lo que publica npm para un paquete fijado.
	// Solo informa (doctor): Latest sigue siendo el pin y deps-update nunca
	// instala otra versión (HCD-039).
	Published func(ctx context.Context) (string, error)
}

// DepState es el resultado guardado en caché para un componente.
type DepState struct {
	Name    string   `json:"name"`
	Current string   `json:"current,omitempty"`
	Latest  string   `json:"latest,omitempty"`
	Update  []string `json:"update,omitempty"`
	Manual  string   `json:"manual,omitempty"`
	Error   string   `json:"error,omitempty"`
	// Published y PublishedError vienen de DepProbe.Published. El error de npm
	// no va en Error: Outdated() devuelve falso con Error y un corte de red
	// dejaría de ofrecer el pin (HCD-039, Splash 1).
	Published      string `json:"published,omitempty"`
	PublishedError string `json:"published_error,omitempty"`
}

// Outdated indica si hay una versión estable más nueva que la instalada.
func (d DepState) Outdated() bool {
	return d.Error == "" && IsNewer(d.Latest, d.Current)
}

type depsCache struct {
	CheckedAt time.Time  `json:"checked_at"`
	Deps      []DepState `json:"deps"`
}

// DefaultDepsCachePath devuelve la ruta de la caché de dependencias.
func DefaultDepsCachePath(piAgentDir string) string {
	return filepath.Join(piAgentDir, depsCacheFile)
}

var semverRe = regexp.MustCompile(`v?(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)`)

// firstSemver extrae la primera versión X.Y.Z de un texto, o "" si no hay.
func firstSemver(s string) string {
	if m := semverRe.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// matchSemver extrae la versión capturada por re, o error si no aparece.
func matchSemver(re *regexp.Regexp, s string) (string, error) {
	if m := re.FindStringSubmatch(s); m != nil {
		return m[1], nil
	}
	return "", fmt.Errorf("no se reconoce la versión en %q", strings.TrimSpace(firstLine(s)))
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func runOutput(ctx context.Context, name string, args ...string) (string, error) {
	if _, err := exec.LookPath(name); err != nil {
		return "", fmt.Errorf("%s no está en PATH", name)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = nil
	// Tras cancelar, no se espera a que un descendiente cierre la salida: sin
	// WaitDelay, `sh -c 'echo; sleep'` retenía la respuesta hasta el sleep.
	cmd.WaitDelay = time.Second
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Un comando fallido o cancelado nunca da versión, aunque haya impreso una
		// (HCD-041, H-03, Q1). La salida no se incluye en el error.
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return string(out), nil
}

func versionFromCmd(name string, args ...string) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		out, err := runOutput(ctx, name, args...)
		if err != nil {
			return "", err
		}
		if v := firstSemver(out); v != "" {
			return v, nil
		}
		return "", fmt.Errorf("%s: versión no reconocida", name)
	}
}

func latestFromCmd(re *regexp.Regexp, name string, args ...string) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		out, err := runOutput(ctx, name, args...)
		if err != nil {
			return "", err
		}
		return matchSemver(re, out)
	}
}

func npmPackageVersion(piAgentDir, pkg string) func(context.Context) (string, error) {
	return func(context.Context) (string, error) {
		data, err := os.ReadFile(filepath.Join(piAgentDir, "npm", "node_modules", pkg, "package.json"))
		if err != nil {
			return "", fmt.Errorf("%s no está instalado", pkg)
		}
		var p struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(data, &p); err != nil || p.Version == "" {
			return "", fmt.Errorf("%s: package.json sin versión", pkg)
		}
		return p.Version, nil
	}
}

// depsHTTPClient es el cliente de httpJSON. Su timeout propio acota la espera
// aunque el llamador no pase un contexto con plazo (HCD-033, C-002).
var depsHTTPClient = &http.Client{Timeout: depsProbeTimeout}

func httpJSON(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "pi-gomemory-setup")
	resp, err := depsHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(dst)
}

// NPMRegistryURL es el registro npm consultado; se puede sustituir en pruebas.
var NPMRegistryURL = "https://registry.npmjs.org"

func npmLatest(pkg string) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		var body struct {
			Version string `json:"version"`
		}
		if err := httpJSON(ctx, NPMRegistryURL+"/"+pkg+"/latest", &body); err != nil {
			return "", err
		}
		if body.Version == "" {
			return "", fmt.Errorf("%s: versión vacía en npm", pkg)
		}
		return body.Version, nil
	}
}

// githubLatest devuelve la etiqueta de la última release de un repositorio de
// GitHub a partir de la redirección de github.com/<repo>/releases/latest.
func githubLatest(url string) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		timeout := depsProbeTimeout
		if d, ok := ctx.Deadline(); ok {
			timeout = time.Until(d)
		}
		return fetchLatestTag(url, timeout)
	}
}

// fixedVersion devuelve la versión fijada por pi-gomemory: los paquetes de Pi
// solo cambian con una release nueva de pi-gomemory.
func fixedVersion(v string) func(context.Context) (string, error) {
	return func(context.Context) (string, error) { return v, nil }
}

// DefaultDepProbes devuelve los componentes del meta-harness con los comandos
// oficiales de actualización de cada uno.
func DefaultDepProbes(version, piAgentDir string) []DepProbe {
	return []DepProbe{
		{
			Name:    "pi-gomemory",
			Current: func(context.Context) (string, error) { return version, nil },
			Latest:  githubLatest(DefaultReleasesAPI),
			Update:  []string{"pi-gomemory-setup", "update"},
		},
		{
			Name:    "pi",
			Current: versionFromCmd("pi", "--version"),
			Latest:  npmLatest("@earendil-works/pi-coding-agent"),
			Update:  []string{"pi", "update", "--self"},
		},
		{
			Name:    "mem",
			Current: versionFromCmd("mem", "version"),
			Latest:  latestFromCmd(regexp.MustCompile(`Disponible:\s*v?(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)`), "mem", "update", "-check"),
			Update:  []string{"mem", "update", "-y"},
		},
		{
			Name:    "codegraph",
			Current: versionFromCmd("codegraph", "--version"),
			Latest:  latestFromCmd(regexp.MustCompile(`latest\s+v?(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)`), "codegraph", "upgrade", "--check"),
			Update:  []string{"codegraph", "upgrade"},
		},
		{
			Name:    "codebase-memory-mcp",
			Current: versionFromCmd("codebase-memory-mcp", "--version"),
			Latest:  githubLatest("https://github.com/DeusData/codebase-memory-mcp/releases/latest"),
			Manual:  "https://github.com/DeusData/codebase-memory-mcp/releases/latest",
		},
		{
			Name:      "pi-antigravity",
			Current:   npmPackageVersion(piAgentDir, "pi-antigravity"),
			Latest:    fixedVersion(AntigravityVersion),
			Update:    []string{"pi", "install", AntigravityPackageSource},
			Published: npmPublished("pi-antigravity", AntigravityVersion),
		},
		{
			Name:      "pi-cc-extensions",
			Current:   npmPackageVersion(piAgentDir, "pi-cc-extensions"),
			Latest:    fixedVersion(CCExtensionsVersion),
			Update:    []string{"pi", "install", CCExtensionsSource},
			Published: npmPublished("pi-cc-extensions", CCExtensionsVersion),
		},
	}
}

// RefreshDeps consulta en paralelo cada componente y guarda el resultado en
// cachePath. Si la caché existe y tiene menos de depsCacheTTL (una hora), se
// omite la consulta en red para no saturar en cada arranque.
func RefreshDeps(probes []DepProbe, cachePath string, now time.Time) []DepState {
	if cachePath != "" {
		if data, err := os.ReadFile(cachePath); err == nil {
			var c depsCache
			if json.Unmarshal(data, &c) == nil && !c.CheckedAt.IsZero() {
				// Con la comprobación desactivada se devuelve la caché tal cual,
				// sin red y sin reescribirla (HCD-037, Q2).
				fresh := now.Sub(c.CheckedAt) < depsCacheTTL && !missingPublished(probes, c.Deps)
				if fresh || UpdateCheckDisabled() {
					return c.Deps
				}
			}
		}
	}
	if UpdateCheckDisabled() {
		return nil
	}

	states := make([]DepState, len(probes))
	var wg sync.WaitGroup
	for i, p := range probes {
		wg.Add(1)
		go func(i int, p DepProbe) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), depsProbeTimeout)
			defer cancel()
			st := DepState{Name: p.Name, Update: p.Update, Manual: p.Manual}
			// Se consulta aunque Current falle, para saber qué publica npm
			// también sin el paquete instalado (HCD-039, C-25).
			if p.Published != nil {
				if pub, err := p.Published(ctx); err != nil {
					st.PublishedError = err.Error()
				} else {
					st.Published = pub
				}
			}
			cur, err := p.Current(ctx)
			if err != nil {
				st.Error = err.Error()
				states[i] = st
				return
			}
			st.Current = cur
			latest, err := p.Latest(ctx)
			if err != nil {
				st.Error = err.Error()
			}
			st.Latest = latest
			states[i] = st
		}(i, p)
	}
	wg.Wait()
	if cachePath != "" {
		if data, err := json.MarshalIndent(depsCache{CheckedAt: now, Deps: states}, "", "  "); err == nil {
			_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
			_ = writeFileAtomic(cachePath, append(data, '\n'), 0644)
		}
	}
	return states
}

// missingPublished indica que algún probe con Published no tiene en la caché ni
// published ni published_error, como la que escribe un binario anterior a
// HCD-039: se refresca sin esperar a depsCacheTTL (C-24).
func missingPublished(probes []DepProbe, states []DepState) bool {
	for _, p := range probes {
		if p.Published == nil {
			continue
		}
		found := false
		for _, s := range states {
			if s.Name == p.Name {
				found = s.Published != "" || s.PublishedError != ""
				break
			}
		}
		if !found {
			return true
		}
	}
	return false
}

// cachedDeps lee todas las entradas de la caché; nil si falta o es ilegible.
func cachedDeps(cachePath string) []DepState {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil
	}
	var c depsCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	return c.Deps
}

// pinnedFirst deja pi-gomemory delante del resto.
func pinnedFirst(out []DepState) []DepState {
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name == "pi-gomemory" && out[j].Name != "pi-gomemory" })
	return out
}

// DepsStatus lee la caché y devuelve los componentes desactualizados.
func DepsStatus(cachePath string) []DepState {
	var out []DepState
	for _, d := range cachedDeps(cachePath) {
		if d.Outdated() {
			out = append(out, d)
		}
	}
	return pinnedFirst(out)
}

// DepsStatusFor devuelve lo desactualizado según la caché, pero con el comando
// y la instrucción manual del probe actual del mismo nombre: la caché es un
// archivo editable y nunca decide qué se ejecuta. Los nombres sin probe se omiten.
// La versión instalada se relee del probe (local y rápido): tras actualizar por
// otra vía la caché aún guarda la anterior y anunciaría algo ya instalado. Si no
// se puede leer, se conserva la de la caché.
//
// Para los paquetes fijados, el objetivo es el pin compilado en este binario y
// no la latest guardada: una caché escrita antes de subir un pin lo ocultaría
// (HCD-040, H-01 de la ACR/ADM 2026-11-17). Por eso se recorre la caché entera
// y se decide con Outdated() después de sustituir objetivo y versión instalada.
func DepsStatusFor(cachePath string, probes []DepProbe) []DepState {
	byName := make(map[string]DepProbe, len(probes))
	for _, p := range probes {
		byName[p.Name] = p
	}
	var out []DepState
	for _, d := range cachedDeps(cachePath) {
		p, ok := byName[d.Name]
		if !ok {
			continue
		}
		d = effectiveDepState(d, p)
		if d.Outdated() {
			out = append(out, d)
		}
	}
	return pinnedFirst(out)
}

// effectiveDepState aplica los comandos de confianza, el pin compilado y la
// versión local antes de comparar o diagnosticar el estado (HCD-042, F-01).
func effectiveDepState(d DepState, p DepProbe) DepState {
	d.Update, d.Manual = p.Update, p.Manual
	for _, pin := range PinnedPackages() {
		if d.Name == pin.Name {
			d.Latest = pin.Version
			break
		}
	}
	if cur := liveVersion(p); cur != "" {
		d.Current = cur
	}
	return d
}

// liveVersion lee la versión instalada con el probe; "" si no hay lector o falla.
func liveVersion(p DepProbe) string {
	if p.Current == nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), liveVersionTimeout)
	defer cancel()
	v, err := p.Current(ctx)
	if err != nil {
		return ""
	}
	return v
}

// RunDepsUpdate actualiza lo desactualizado con los comandos de los probes y
// devuelve el código de salida. Los componentes manuales solo se anuncian.
func RunDepsUpdate(cachePath string, probes []DepProbe, stdout, stderr io.Writer) int {
	// No confundir una caché ausente o ilegible con una consulta sin novedades.
	data, err := os.ReadFile(cachePath)
	var cache depsCache
	if err != nil || json.Unmarshal(data, &cache) != nil || cache.CheckedAt.IsZero() {
		fmt.Fprintln(stderr, "✗ Estado desconocido; ejecuta pi-gomemory-setup deps-refresh.")
		if UpdateCheckDisabled() {
			fmt.Fprintln(stderr, "La comprobación está desactivada por PI_GOMEMORY_NO_UPDATE_CHECK.")
		}
		return 1
	}
	states := make(map[string]DepState, len(cache.Deps))
	for _, d := range cache.Deps {
		states[d.Name] = d
	}
	unknown := false
	var outdated []DepState
	for _, p := range probes {
		d, ok := states[p.Name]
		if ok {
			d = effectiveDepState(d, p)
		}
		reason := d.Error
		if !ok {
			reason = "sin datos en la caché"
		} else if reason == "" {
			_, currentOK := parseSemver(d.Current)
			_, latestOK := parseSemver(d.Latest)
			switch {
			case d.Current == "" || d.Latest == "":
				reason = "datos incompletos en la caché"
			case !currentOK || !latestOK:
				reason = "versiones no comparables"
			}
		}
		if reason != "" {
			fmt.Fprintf(stderr, "✗ %s: estado desconocido (%s)\n", p.Name, reason)
			unknown = true
		} else if d.Outdated() {
			outdated = append(outdated, d)
		}
	}
	// La misma instantánea sirve para diagnóstico y actualización: no se relee
	// la caché ni el probe después de validar las versiones efectivas.
	outdated = pinnedFirst(outdated)
	if len(outdated) == 0 && unknown {
		fmt.Fprintln(stderr, "Ejecuta pi-gomemory-setup deps-refresh para renovar el estado.")
		return 1
	}
	if len(outdated) == 0 {
		fmt.Fprintln(stdout, "✓ El meta-harness y sus dependencias están al día.")
		return 0
	}
	var manual []string
	for _, d := range outdated {
		if len(d.Update) == 0 && d.Manual != "" {
			fmt.Fprintf(stdout, "==> %s (%s → %s): actualízalo manualmente: %s\n", d.Name, d.Current, d.Latest, d.Manual)
			manual = append(manual, d.Name)
		}
	}
	failed := UpdateDeps(outdated, stdout, stderr)
	_ = os.Remove(cachePath)
	if len(failed) > 0 {
		fmt.Fprintf(stderr, "✗ No se pudieron actualizar: %v\n", failed)
		return 1
	}
	if unknown {
		fmt.Fprintln(stderr, "Ejecuta pi-gomemory-setup deps-refresh para renovar el estado desconocido.")
		return 1
	}
	if len(manual) > 0 {
		fmt.Fprintf(stdout, "⚠ Pendientes de actualización manual: %v\n", manual)
		return 0
	}
	fmt.Fprintln(stdout, "✓ Dependencias actualizadas.")
	return 0
}

// FormatDepsNotice compone el aviso de componentes desactualizados.
func FormatDepsNotice(outdated []DepState) string {
	if len(outdated) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s↑ Actualizaciones disponibles del meta-harness:%s\n", ColorBrillo, ColorReset)
	for _, d := range outdated {
		how := strings.Join(d.Update, " ")
		if how == "" && d.Manual != "" {
			how = "actualízalo manualmente: " + d.Manual
		}
		fmt.Fprintf(&b, "  %-20s %s → %s   %s(%s)%s\n", d.Name, d.Current, d.Latest, ColorMuted, how, ColorReset)
	}
	return b.String()
}

// UpdateDeps ejecuta el comando de actualización de cada componente
// desactualizado. Devuelve los nombres que fallaron.
func UpdateDeps(outdated []DepState, stdout, stderr io.Writer) []string {
	var failed []string
	for _, d := range outdated {
		if len(d.Update) == 0 {
			continue
		}
		fmt.Fprintf(stdout, "==> Actualizando %s (%s → %s)...\n", d.Name, d.Current, d.Latest)
		cmd := exec.Command(d.Update[0], d.Update[1:]...)
		cmd.Stdout, cmd.Stderr, cmd.Stdin = stdout, stderr, os.Stdin
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(stderr, "%s✗ %s: %v%s\n", ColorError, d.Name, err, ColorReset)
			failed = append(failed, d.Name)
		}
	}
	return failed
}

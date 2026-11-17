package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Límites de pins-validate (HCD-039, C-15). Son variables para que las pruebas
// los reduzcan; en producción no cambian.
var (
	candidateDownloadLimit int64 = 50 << 20
	candidateExtractLimit  int64 = 200 << 20
	// candidateEntryLimit y candidateTarStreamLimit acotan lo que el límite de
	// contenido no cubre: entradas vacías, directorios y cabeceras (HCD-041,
	// H-05, Q3). Los paquetes reales tienen menos de 100 entradas y 1 MiB.
	candidateEntryLimit           = 10_000
	candidateTarStreamLimit int64 = 256 << 20
	// candidateTempRoot es la carpeta donde se crea el directorio temporal; ""
	// usa la carpeta temporal del sistema.
	candidateTempRoot = ""
)

// validateCandidate valida spec («<paquete>@<versión>»): el paquete debe estar
// fijado y la versión publicada; descarga su tarball, lo verifica, lo extrae en
// un directorio temporal que se borra al terminar y aplica sus contratos.
func validateCandidate(ctx context.Context, spec string) ([]ContractResult, error) {
	pkg, version, ok := strings.Cut(spec, "@")
	if !ok || pkg == "" || version == "" {
		return nil, fmt.Errorf("PINS_CANDIDATE=%q: usa <paquete>@<versión>, por ejemplo pi-antigravity@0.14.0", spec)
	}
	pinned := false
	for _, p := range PinnedPackages() {
		pinned = pinned || p.Name == pkg
	}
	if !pinned {
		return nil, fmt.Errorf("%s no está fijado por pi-gomemory; solo se validan pi-antigravity y pi-cc-extensions", pkg)
	}
	doc, err := fetchRegistryDoc(ctx, pkg)
	if err != nil {
		return nil, err
	}
	v, ok := doc.Versions[version]
	if !ok {
		return nil, fmt.Errorf("%s@%s no aparece en el registro", pkg, version)
	}
	dir, cleanup, err := fetchCandidate(ctx, v.Dist.Tarball, v.Dist.Integrity)
	if err != nil {
		return nil, fmt.Errorf("%s@%s: %w", pkg, version, err)
	}
	defer cleanup()
	return checkContracts(pkg, dir)
}

// FormatContracts redacta el resultado de pins-validate; fail si alguno se rompe.
func FormatContracts(spec string, results []ContractResult) (lines []string, fail bool) {
	for _, r := range results {
		state := "cumplido"
		if !r.OK {
			state, fail = "roto", true
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s (%s)", spec, r.ID, state, r.Detail))
	}
	return lines, fail
}

// contractRule es una regla textual de la tabla de contratos de HCD-039: alguno
// de los archivos que casan con Files debe contener todas las cadenas de Needs.
type contractRule struct {
	ID    string
	Files string // ruta dentro del tarball; con «**/*.ts», cualquier .ts bajo ese prefijo
	Needs []string
}

// packageContracts son los contratos de cada paquete fijado de los que depende
// pi-gomemory. Detectan un renombre o una retirada, no un cambio de significado.
var packageContracts = map[string][]contractRule{
	"pi-antigravity": {
		// usage.go lee activeAccountId y accounts.<id> de este archivo.
		{"A1", "package/src/auth/accounts.ts", []string{`"antigravity-accounts.json"`, "activeAccountId", "accounts: Record<"}},
		// El sidebar detecta el proveedor por este id.
		{"A3", "package/src/models/models.ts", []string{`PROVIDER_ID = "antigravity"`}},
	},
	"pi-cc-extensions": {
		// managedCCKeys (plugin.go) impone esta clave de pi-cc-extensions.json.
		{"C1", "package/extensions/**/*.ts", []string{"showStartupHeader"}},
	},
}

// ContractResult es el resultado de una regla sobre un candidato extraído.
type ContractResult struct {
	ID     string
	OK     bool
	Detail string
}

// checkContracts aplica a dir, el tarball extraído, las reglas de pkg.
func checkContracts(pkg, dir string) ([]ContractResult, error) {
	rules, ok := packageContracts[pkg]
	if !ok {
		return nil, fmt.Errorf("%s no tiene contratos definidos; solo se validan paquetes fijados", pkg)
	}
	out := make([]ContractResult, 0, len(rules))
	for _, r := range rules {
		out = append(out, checkRule(r, dir))
	}
	return out, nil
}

func checkRule(r contractRule, dir string) ContractResult {
	files, err := ruleFiles(r.Files, dir)
	if err != nil || len(files) == 0 {
		return ContractResult{ID: r.ID, Detail: "falta " + r.Files}
	}
	var missing []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		missing = missing[:0]
		for _, need := range r.Needs {
			if !bytes.Contains(b, []byte(need)) {
				missing = append(missing, need)
			}
		}
		if len(missing) == 0 {
			rel, _ := filepath.Rel(dir, f)
			return ContractResult{ID: r.ID, OK: true, Detail: filepath.ToSlash(rel)}
		}
	}
	if len(files) > 1 {
		return ContractResult{ID: r.ID, Detail: fmt.Sprintf("ningún archivo de %s contiene %s", r.Files, strings.Join(r.Needs, ", "))}
	}
	return ContractResult{ID: r.ID, Detail: fmt.Sprintf("%s no contiene %s", r.Files, strings.Join(missing, ", "))}
}

// ruleFiles resuelve el patrón de una regla dentro de dir.
func ruleFiles(pattern, dir string) ([]string, error) {
	prefix, ok := strings.CutSuffix(pattern, "**/*.ts")
	if !ok {
		f := filepath.Join(dir, filepath.FromSlash(pattern))
		if info, err := os.Stat(f); err != nil || !info.Mode().IsRegular() {
			return nil, err
		}
		return []string{f}, nil
	}
	var out []string
	err := filepath.WalkDir(filepath.Join(dir, filepath.FromSlash(prefix)), func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() && strings.HasSuffix(p, ".ts") {
			out = append(out, p)
		}
		return nil
	})
	return out, err
}

// fetchCandidate descarga el tarball de un candidato, comprueba su integridad
// sha512 y lo extrae en un directorio temporal nuevo. Devuelve ese directorio y
// la función que lo borra; si algo falla, lo borra antes de devolver el error.
func fetchCandidate(ctx context.Context, url, integrity string) (string, func(), error) {
	want, err := parseIntegrity(integrity)
	if err != nil {
		return "", nil, err
	}
	data, err := downloadCandidate(ctx, url)
	if err != nil {
		return "", nil, err
	}
	got := sha512.Sum512(data)
	if subtle.ConstantTimeCompare(got[:], want) != 1 {
		return "", nil, errors.New("el sha512 del tarball no coincide con dist.integrity")
	}
	dir, err := os.MkdirTemp(candidateTempRoot, "pi-gomemory-pins-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	if err := extractCandidate(ctx, data, dir); err != nil {
		cleanup()
		return "", nil, err
	}
	return dir, cleanup, nil
}

// parseIntegrity acepta solo «sha512-<base64>», el formato de npm.
func parseIntegrity(integrity string) ([]byte, error) {
	if integrity == "" {
		return nil, errors.New("el candidato no trae dist.integrity; no se puede verificar")
	}
	b64, ok := strings.CutPrefix(integrity, "sha512-")
	if !ok {
		return nil, fmt.Errorf("dist.integrity no es sha512: %q", integrity)
	}
	sum, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(sum) != sha512.Size {
		return nil, fmt.Errorf("dist.integrity sha512 ilegible: %q", integrity)
	}
	return sum, nil
}

func downloadCandidate(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "pi-gomemory-setup")
	resp, err := registryClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("descarga del tarball: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, candidateDownloadLimit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > candidateDownloadLimit {
		return nil, fmt.Errorf("la descarga del tarball supera %d bytes; se descarta", candidateDownloadLimit)
	}
	return data, nil
}

// extractCandidate extrae un tar.gz en dir admitiendo solo archivos normales y
// directorios con rutas relativas que no salgan de dir. Además del contenido,
// acota el número de entradas y el flujo tar descomprimido, cabeceras incluidas,
// y se detiene si ctx se cancela (HCD-041, H-05).
func extractCandidate(ctx context.Context, data []byte, dir string) error {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("tarball ilegible: %w", err)
	}
	defer gz.Close()
	// Un byte de más distingue «cabe justo» de «excede».
	stream := &io.LimitedReader{R: gz, N: candidateTarStreamLimit + 1}
	streamExceeded := func() error {
		return fmt.Errorf("el tarball descomprimido supera %d bytes; se descarta", candidateTarStreamLimit)
	}
	tr := tar.NewReader(stream)
	var total int64
	entries := 0
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("extracción cancelada: %w", err)
		}
		h, err := tr.Next()
		if stream.N <= 0 {
			return streamExceeded()
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("tarball ilegible: %w", err)
		}
		if entries++; entries > candidateEntryLimit {
			return fmt.Errorf("el tarball supera %d entradas; se descarta", candidateEntryLimit)
		}
		clean := path.Clean(h.Name)
		if path.IsAbs(h.Name) || clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("ruta no permitida en el tarball: %q", h.Name)
		}
		target := filepath.Join(dir, filepath.FromSlash(clean))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if total += h.Size; total > candidateExtractLimit {
				return fmt.Errorf("la extracción del tarball supera %d bytes; se descarta", candidateExtractLimit)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := writeCandidateFile(target, tr, h.Size); err != nil {
				if stream.N <= 0 {
					return streamExceeded()
				}
				return err
			}
		default:
			return fmt.Errorf("tipo de entrada no permitido en el tarball (%q, tipo %q): solo archivos y directorios", h.Name, h.Typeflag)
		}
	}
}

func writeCandidateFile(target string, r io.Reader, size int64) error {
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(r, size))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n != size {
		err = fmt.Errorf("entrada truncada en el tarball: %s", target)
	}
	return err
}

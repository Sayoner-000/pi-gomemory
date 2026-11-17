package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hcd039Entry es una entrada de un tarball sintético.
type hcd039Entry struct {
	name     string
	body     string
	typeflag byte
	link     string
}

// hcd039Tarball empaqueta entries como hace npm (tar.gz).
func hcd039Tarball(t *testing.T, entries ...hcd039Entry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		typ := e.typeflag
		if typ == 0 {
			typ = tar.TypeReg
		}
		h := &tar.Header{Name: e.name, Mode: 0644, Typeflag: typ, Linkname: e.link}
		if typ == tar.TypeReg {
			h.Size = int64(len(e.body))
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if typ == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func hcd039Integrity(data []byte) string {
	sum := sha512.Sum512(data)
	return "sha512-" + base64.StdEncoding.EncodeToString(sum[:])
}

// hcd039TarballServer sirve data en /pkg.tgz.
func hcd039TarballServer(t *testing.T, data []byte) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/pkg.tgz"
}

// Tarea 8 (C-12, C-14): validateCandidate toma dist.tarball y dist.integrity
// del registro, valida el formato de PINS_CANDIDATE y aplica los contratos.
func TestHCD039ValidateCandidate(t *testing.T) {
	root := t.TempDir()
	oldRoot := candidateTempRoot
	candidateTempRoot = root
	t.Cleanup(func() { candidateTempRoot = oldRoot })
	good := hcd039Tarball(t,
		hcd039Entry{name: "package/src/auth/accounts.ts", body: "\"antigravity-accounts.json\" activeAccountId accounts: Record<string, X>"},
		hcd039Entry{name: "package/src/models/models.ts", body: `PROVIDER_ID = "antigravity"`},
	)
	broken := hcd039Tarball(t,
		hcd039Entry{name: "package/src/auth/accounts.ts", body: "\"antigravity-accounts.json\" activeAccountId accounts: Record<string, X>"},
		hcd039Entry{name: "package/src/models/models.ts", body: `PROVIDER_ID = "otro"`},
	)
	goodURL, brokenURL := hcd039TarballServer(t, good), hcd039TarballServer(t, broken)
	hcd039Registry(t, map[string]string{
		"pi-antigravity": `{"versions":{` +
			`"9.9.9":{"dist":{"tarball":"` + goodURL + `","integrity":"` + hcd039Integrity(good) + `"}},` +
			`"9.9.8":{"dist":{"tarball":"` + brokenURL + `","integrity":"` + hcd039Integrity(broken) + `"}}}}`,
	})
	ctx := context.Background()

	rs, err := validateCandidate(ctx, "pi-antigravity@9.9.9")
	lines, fail := FormatContracts("pi-antigravity@9.9.9", rs)
	if err != nil || fail || len(rs) != 2 {
		t.Fatalf("C-12: el candidato correcto debe cumplir A1 y A3: %v %v\n%s", rs, err, strings.Join(lines, "\n"))
	}
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "A1: cumplido") || !strings.Contains(out, "A3: cumplido") {
		t.Errorf("C-12: el informe nombra cada contrato por su id:\n%s", out)
	}

	rs, err = validateCandidate(ctx, "pi-antigravity@9.9.8")
	lines, fail = FormatContracts("pi-antigravity@9.9.8", rs)
	if err != nil || !fail || !strings.Contains(strings.Join(lines, "\n"), "A3: roto") {
		t.Errorf("C-13: el candidato roto falla nombrando A3: %v\n%s", err, strings.Join(lines, "\n"))
	}

	for spec, want := range map[string]string{
		"pi-antigravity":       "@<versión>",
		"otro-paquete@1.0.0":   "no está fijado",
		"pi-antigravity@1.2.3": "no aparece",
	} {
		if _, err := validateCandidate(ctx, spec); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("C-14: %q debe fallar explicando «%s»: %v", spec, want, err)
		}
	}
	if left, _ := os.ReadDir(root); len(left) != 0 {
		t.Errorf("C-16: quedaron directorios temporales: %v", left)
	}
}

// hcd039Tree escribe files (ruta relativa → contenido) en un directorio nuevo.
func hcd039Tree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		writeFile(t, filepath.Join(dir, filepath.FromSlash(name)), body)
	}
	return dir
}

// Tarea 7 (C-12, C-13): las reglas de la tabla de contratos se aplican sobre
// el árbol extraído y un fallo nombra el id del contrato roto.
func TestHCD039Contracts(t *testing.T) {
	accounts := "const ACCOUNTS_FILE = \"antigravity-accounts.json\";\ntype AccountsFile = {\n  activeAccountId?: string;\n  accounts: Record<string, StoredAccount>;\n};\n"
	models := "export const PROVIDER_ID = \"antigravity\";\n"
	byID := func(rs []ContractResult) map[string]ContractResult {
		m := map[string]ContractResult{}
		for _, r := range rs {
			m[r.ID] = r
		}
		return m
	}

	good := hcd039Tree(t, map[string]string{"package/src/auth/accounts.ts": accounts, "package/src/models/models.ts": models})
	rs, err := checkContracts("pi-antigravity", good)
	if err != nil || len(rs) != 2 || !byID(rs)["A1"].OK || !byID(rs)["A3"].OK {
		t.Fatalf("C-12: pi-antigravity correcto debe cumplir A1 y A3: %+v %v", rs, err)
	}

	// «accounts» solo dentro del nombre del archivo no basta (Splash 3, H3).
	onlyName := strings.Replace(accounts, "accounts: Record<", "cuentas: Record<", 1)
	rs, _ = checkContracts("pi-antigravity", hcd039Tree(t, map[string]string{"package/src/auth/accounts.ts": onlyName, "package/src/models/models.ts": models}))
	if a1 := byID(rs)["A1"]; a1.OK || !strings.Contains(a1.Detail, "accounts: Record<") {
		t.Errorf("C-13: A1 debe fallar si falta la declaración de accounts: %+v", a1)
	}
	rs, _ = checkContracts("pi-antigravity", hcd039Tree(t, map[string]string{"package/src/auth/accounts.ts": accounts, "package/src/models/models.ts": `export const PROVIDER_ID = "otro";`}))
	if a3 := byID(rs)["A3"]; a3.OK || !byID(rs)["A1"].OK {
		t.Errorf("C-13: A3 debe fallar con otro PROVIDER_ID y A1 seguir cumplido: %+v", rs)
	}
	rs, _ = checkContracts("pi-antigravity", hcd039Tree(t, map[string]string{"package/src/models/models.ts": models}))
	if a1 := byID(rs)["A1"]; a1.OK || !strings.Contains(a1.Detail, "falta") {
		t.Errorf("C-13: sin el archivo que la regla cita, A1 falla diciendo que falta: %+v", a1)
	}

	rs, err = checkContracts("pi-cc-extensions", hcd039Tree(t, map[string]string{"package/extensions/config/config.ts": "showStartupHeader: false"}))
	if err != nil || len(rs) != 1 || !byID(rs)["C1"].OK {
		t.Errorf("C-12: pi-cc-extensions correcto debe cumplir C1: %+v %v", rs, err)
	}
	rs, _ = checkContracts("pi-cc-extensions", hcd039Tree(t, map[string]string{"package/extensions/index.ts": "export {}", "package/README.md": "showStartupHeader"}))
	if c1 := byID(rs)["C1"]; c1.OK {
		t.Errorf("C-13: C1 solo cuenta archivos .ts de extensions/: %+v", c1)
	}
	if _, err := checkContracts("otro-paquete", good); err == nil {
		t.Error("un paquete sin contratos definidos debe ser un error")
	}
}

// Tarea 6 (C-15, C-16): el tarball se verifica con dist.integrity antes de
// extraer, con límites de tamaño, y solo admite archivos normales y
// directorios dentro del directorio temporal, que se borra al terminar.
func TestHCD039SafeExtract(t *testing.T) {
	root := t.TempDir()
	oldRoot, oldDL, oldEx := candidateTempRoot, candidateDownloadLimit, candidateExtractLimit
	candidateTempRoot = root
	t.Cleanup(func() { candidateTempRoot, candidateDownloadLimit, candidateExtractLimit = oldRoot, oldDL, oldEx })
	ctx := context.Background()
	assertEmptyRoot := func(name string) {
		t.Helper()
		if left, _ := os.ReadDir(root); len(left) != 0 {
			t.Errorf("%s: el directorio temporal no se borró: %v", name, left)
		}
	}

	good := hcd039Tarball(t,
		hcd039Entry{name: "package/", typeflag: tar.TypeDir},
		hcd039Entry{name: "package/src/models/models.ts", body: `export const PROVIDER_ID = "antigravity";`},
	)
	dir, cleanup, err := fetchCandidate(ctx, hcd039TarballServer(t, good), hcd039Integrity(good))
	if err != nil {
		t.Fatalf("un tarball correcto debe extraerse: %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "package", "src", "models", "models.ts")); err != nil || !strings.Contains(string(b), "PROVIDER_ID") {
		t.Fatalf("contenido extraído = %q, %v", b, err)
	}
	cleanup()
	assertEmptyRoot("tarball correcto")

	reject := func(name string, data []byte, integrity, want string) {
		t.Helper()
		_, cleanup, err := fetchCandidate(ctx, hcd039TarballServer(t, data), integrity)
		if err == nil {
			cleanup()
			t.Errorf("%s: debía rechazarse", name)
			return
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s: el error debe explicar «%s»: %v", name, want, err)
		}
		assertEmptyRoot(name)
	}

	reject("sin integrity", good, "", "integrity")
	reject("integrity sha1", good, "sha1-abc", "sha512")
	reject("integrity distinta", good, hcd039Integrity([]byte("otro")), "no coincide")
	for _, c := range []struct {
		name  string
		entry hcd039Entry
		want  string
	}{
		{"enlace simbólico", hcd039Entry{name: "package/x", typeflag: tar.TypeSymlink, link: "/etc/passwd"}, "tipo"},
		{"enlace duro", hcd039Entry{name: "package/x", typeflag: tar.TypeLink, link: "package/y"}, "tipo"},
		{"FIFO", hcd039Entry{name: "package/x", typeflag: tar.TypeFifo}, "tipo"},
		{"dispositivo", hcd039Entry{name: "package/x", typeflag: tar.TypeChar}, "tipo"},
		{"ruta con ..", hcd039Entry{name: "package/../../evil", body: "x"}, "ruta"},
		{"ruta absoluta", hcd039Entry{name: "/tmp/hcd039-evil", body: "x"}, "ruta"},
	} {
		data := hcd039Tarball(t, c.entry)
		reject(c.name, data, hcd039Integrity(data), c.want)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "evil")); err == nil {
		t.Error("una ruta con .. escribió fuera del directorio temporal")
	}

	candidateDownloadLimit = 64
	big := hcd039Tarball(t, hcd039Entry{name: "package/a", body: strings.Repeat("a", 4096)})
	reject("descarga sobre el límite", big, hcd039Integrity(big), "descarga")
	candidateDownloadLimit = oldDL

	candidateExtractLimit = 1024
	reject("extracción sobre el límite", big, hcd039Integrity(big), "extra")
}

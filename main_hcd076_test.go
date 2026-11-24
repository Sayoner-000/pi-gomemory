package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"pi-gomemory/pkg/installer"
)

// fakeSetupScript es un pi-gomemory-setup falso: `components --names` lista
// los tool dados e `install` termina bien.
func fakeSetupScript(tools ...string) string {
	return "#!/bin/sh\nif [ \"$1\" = components ]; then printf '" + strings.Join(tools, "\\n") + "\\n'; exit 0; fi\necho \"setup $*\"\nexit 0\n"
}

func tarballWith(t *testing.T, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "pi-gomemory-setup", Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte(content))
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

// releaseServer sirve /latest (redirección a latestTag, o 404 si está vacío)
// y los tarballs de las etiquetas dadas, y registra las rutas pedidas.
type releaseServer struct {
	srv  *httptest.Server
	mu   sync.Mutex
	hits []string
}

func newReleaseServer(t *testing.T, latestTag, setup string, tags ...string) *releaseServer {
	t.Helper()
	rs := &releaseServer{}
	files := map[string][]byte{}
	var sums strings.Builder
	for _, tag := range tags {
		name := fmt.Sprintf("pi-gomemory-setup_%s_%s_%s.tar.gz", tag, runtime.GOOS, runtime.GOARCH)
		data := tarballWith(t, setup)
		sum := sha256.Sum256(data)
		files["/"+name] = data
		fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(sum[:]), name)
	}
	rs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rs.mu.Lock()
		rs.hits = append(rs.hits, r.URL.Path)
		rs.mu.Unlock()
		switch {
		case r.URL.Path == "/latest" && latestTag != "":
			http.Redirect(w, r, "/Sayoner-000/pi-gomemory/releases/tag/"+latestTag, http.StatusFound)
		case r.URL.Path == "/SHA256SUMS":
			fmt.Fprint(w, sums.String())
		case files[r.URL.Path] != nil:
			_, _ = w.Write(files[r.URL.Path])
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(rs.srv.Close)
	return rs
}

func (rs *releaseServer) reset() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.hits = nil
}

func (rs *releaseServer) count() int {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return len(rs.hits)
}

func (rs *releaseServer) hit(path string) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	for _, h := range rs.hits {
		if h == path {
			return true
		}
	}
	return false
}

// runInstallSh ejecuta install.sh en dir con el entorno dado.
func runInstallSh(t *testing.T, dir string, env ...string) (string, int) {
	t.Helper()
	script, err := filepath.Abs("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), code
}

// HCD-076 T18 · C-02: sin PI_GOMEMORY_VERSION, install.sh descarga la etiqueta
// de PI_GOMEMORY_LATEST_URL; con ella, esa etiqueta; si la resolución falla,
// aborta mencionando PI_GOMEMORY_VERSION; compilando desde el código fuente
// (opción A) no consulta la red.
func TestHCD076InstallShLatest(t *testing.T) {
	sysPath := "PATH=/usr/bin:/bin:/usr/sbin:/sbin"
	rs := newReleaseServer(t, "v9.8.7", fakeSetupScript("pi"), "v9.8.7", "v1.2.3")
	base := []string{"HOME=" + t.TempDir(), sysPath, "PI_GOMEMORY_BASE_URL=" + rs.srv.URL, "PI_GOMEMORY_LATEST_URL=" + rs.srv.URL + "/latest"}

	out, code := runInstallSh(t, t.TempDir(), base...)
	if code != 0 || !strings.Contains(out, "pi-gomemory-setup_v9.8.7_") {
		t.Fatalf("sin versión debe descargar la última (v9.8.7): código %d\n%s", code, out)
	}

	rs.reset()
	out, code = runInstallSh(t, t.TempDir(), append(base, "PI_GOMEMORY_VERSION=v1.2.3")...)
	if code != 0 || !strings.Contains(out, "pi-gomemory-setup_v1.2.3_") || rs.hit("/latest") {
		t.Fatalf("con PI_GOMEMORY_VERSION debe usar esa etiqueta sin consultar latest: código %d\n%s", code, out)
	}

	broken := newReleaseServer(t, "", fakeSetupScript("pi"))
	out, code = runInstallSh(t, t.TempDir(), "HOME="+t.TempDir(), sysPath, "PI_GOMEMORY_LATEST_URL="+broken.srv.URL+"/latest")
	if code == 0 || !strings.Contains(out, "PI_GOMEMORY_VERSION") {
		t.Fatalf("si no se resuelve la última debe abortar y sugerir PI_GOMEMORY_VERSION: código %d\n%s", code, out)
	}

	// Opción A: un árbol con go.mod de pi-gomemory y un go falso que deja el
	// binario; no debe pedirse /latest.
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "go.mod"), []byte("module pi-gomemory\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeBin := t.TempDir()
	goScript := "#!/bin/sh\nwhile [ $# -gt 0 ]; do if [ \"$1\" = -o ]; then shift; printf '%s' \"$SETUP\" > \"$1\"; fi; shift; done\n"
	if err := os.WriteFile(filepath.Join(fakeBin, "go"), []byte(goScript), 0o755); err != nil {
		t.Fatal(err)
	}
	rs.reset()
	out, code = runInstallSh(t, src, "HOME="+t.TempDir(), "PATH="+fakeBin+":/usr/bin:/bin:/usr/sbin:/sbin", "SETUP="+fakeSetupScript("pi"), "PI_GOMEMORY_LATEST_URL="+rs.srv.URL+"/latest")
	if code != 0 || !strings.Contains(out, "compilado en") || rs.count() != 0 {
		t.Fatalf("la compilación local no debe consultar la red: código %d peticiones %d\n%s", code, rs.count(), out)
	}
}

// HCD-076 T19 · C-08: la nota de PATH de install.sh usa `components --names`
// en lugar de una lista fija.
func TestHCD076InstallShPathFromCatalog(t *testing.T) {
	src, err := os.ReadFile("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "pi mem codegraph codebase-memory-mcp") || !strings.Contains(string(src), "components --names") {
		t.Fatal("install.sh debe obtener los tool de `components --names`, no de una lista fija")
	}
	home := t.TempDir()
	herdrDir := filepath.Join(home, ".gomemory", "bin")
	if err := os.MkdirAll(herdrDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(herdrDir, "herdr"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	rs := newReleaseServer(t, "v9.8.7", fakeSetupScript("pi", "herdr"), "v9.8.7")
	out, code := runInstallSh(t, t.TempDir(), "HOME="+home, "PATH=/usr/bin:/bin:/usr/sbin:/sbin", "PI_GOMEMORY_BASE_URL="+rs.srv.URL, "PI_GOMEMORY_LATEST_URL="+rs.srv.URL+"/latest")
	if code != 0 || !strings.Contains(out, herdrDir) {
		t.Fatalf("la nota de PATH debe nombrar el directorio de herdr: código %d\n%s", code, out)
	}
}

// HCD-076 T14 · C-15: update reconcilia los tool una sola vez: al día,
// ejecuta sync; con release nueva, el install relanzado lleva
// PI_GOMEMORY_SYNC_YES=1 y no hay un segundo sync; si la consulta falla,
// termina con error, no ejecuta sync y sugiere `pi-gomemory-setup sync`.
func TestHCD076UpdateRunsSync(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")
	h := newUpdateHarness(t, "v1.1.0", "v1.1.0")
	if err := runUpdateWith(h.opts, h.deps()); err != nil {
		t.Fatalf("al día: %v", err)
	}
	if h.syncs != 1 || h.ran {
		t.Fatalf("al día debe ejecutar sync una vez y no relanzar install: syncs=%d ran=%v", h.syncs, h.ran)
	}

	h = newUpdateHarness(t, "v1.0.0", "v1.1.0")
	if err := runUpdateWith(h.opts, h.deps()); err != nil {
		t.Fatalf("release nueva: %v", err)
	}
	if v, _ := envValue(h.runEnv, "PI_GOMEMORY_SYNC_YES"); !h.ran || v != "1" || h.syncs != 0 {
		t.Fatalf("release nueva: ran=%v SYNC_YES=%q syncs=%d", h.ran, v, h.syncs)
	}

	h = newUpdateHarness(t, "v1.0.0", "v1.1.0")
	d := h.deps()
	d.apiURL = h.release.srv.URL + "/no-existe"
	err := runUpdateWith(h.opts, d)
	if err == nil || !strings.Contains(err.Error(), "pi-gomemory-setup sync") || h.syncs != 0 {
		t.Fatalf("consulta fallida: err=%v syncs=%d", err, h.syncs)
	}
}

// HCD-076 T6 · C-08: `components --names` imprime los tool del catálogo, uno
// por línea, sin red; una opción desconocida termina con código 2.
func TestHCD076ComponentsNames(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "1")
	var out, errOut bytes.Buffer
	if code := runComponents(&installer.Options{HomeDir: t.TempDir(), PiAgentDir: t.TempDir()}, []string{"--names"}, &out, &errOut); code != 0 {
		t.Fatalf("código %d: %s", code, errOut.String())
	}
	want := strings.Join(installer.ToolNames(), "\n") + "\n"
	if out.String() != want {
		t.Fatalf("salida = %q, quiero %q", out.String(), want)
	}
	if code := runComponents(&installer.Options{}, []string{"--bogus"}, &out, &errOut); code != 2 {
		t.Fatalf("opción desconocida: código %d", code)
	}
}

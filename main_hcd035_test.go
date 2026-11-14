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
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"pi-gomemory/pkg/installer"
)

// fakeReleases sirve la última etiqueta en /latest y su tarball verificado en
// /download/<tag>/, contando consultas y descargas.
type fakeReleases struct {
	srv       *httptest.Server
	queries   int32
	downloads int32
}

func newFakeReleases(t *testing.T, tag string, binary []byte) *fakeReleases {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "pi-gomemory-setup", Mode: 0755, Size: int64(len(binary))}); err != nil {
		t.Fatal(err)
	}
	tw.Write(binary)
	tw.Close()
	gz.Close()
	tarball := buf.Bytes()
	sum := sha256.Sum256(tarball)
	name := fmt.Sprintf("pi-gomemory-setup_%s_%s_%s.tar.gz", tag, runtime.GOOS, runtime.GOARCH)
	f := &fakeReleases{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			atomic.AddInt32(&f.queries, 1)
			fmt.Fprintf(w, `{"tag_name":%q}`, tag)
		case "/download/" + tag + "/SHA256SUMS":
			atomic.AddInt32(&f.downloads, 1)
			fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), name)
		case "/download/" + tag + "/" + name:
			atomic.AddInt32(&f.downloads, 1)
			w.Write(tarball)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// updateHarness prepara un HOME y un ejecutable temporales; nunca usa el
// binario de la persona ni el de las pruebas.
type updateHarness struct {
	opts    *installer.Options
	exe     string
	out     bytes.Buffer
	ran     bool
	runEnv  []string
	runExe  string
	release *fakeReleases
}

func newUpdateHarness(t *testing.T, current, latest string) *updateHarness {
	t.Helper()
	old := version
	version = current
	t.Cleanup(func() { version = old })
	home := t.TempDir()
	h := &updateHarness{
		opts:    &installer.Options{HomeDir: home, PiAgentDir: filepath.Join(home, ".pi", "agent"), LocalBinDir: filepath.Join(home, ".local", "bin")},
		exe:     filepath.Join(t.TempDir(), "pi-gomemory-setup"),
		release: newFakeReleases(t, latest, []byte("NEW-BINARY")),
	}
	if err := os.WriteFile(h.exe, []byte("OLD-BINARY"), 0755); err != nil {
		t.Fatal(err)
	}
	return h
}

func (h *updateHarness) deps() updateDeps {
	return updateDeps{
		executable: func() (string, error) { return h.exe, nil },
		apiURL:     h.release.srv.URL + "/latest",
		baseURL:    h.release.srv.URL + "/download",
		run: func(exe string, env []string) error {
			h.ran, h.runExe, h.runEnv = true, exe, env
			return nil
		},
		out: &h.out,
	}
}

func (h *updateHarness) exeContent(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(h.exe)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// envValue devuelve el último valor de key en env, como lo resuelve exec.
func envValue(env []string, key string) (string, bool) {
	val, found := "", false
	for _, kv := range env {
		if strings.HasPrefix(kv, key+"=") {
			val, found = strings.TrimPrefix(kv, key+"="), true
		}
	}
	return val, found
}

// Tarea 10: runUpdate usa el ejecutable, las URL y el lanzador inyectados.
func TestHCD035RunUpdateInjected(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")
	h := newUpdateHarness(t, "v1.0.0", "v1.1.0")
	if err := runUpdateWith(h.opts, h.deps()); err != nil {
		t.Fatalf("update: %v\n%s", err, h.out.String())
	}
	if h.release.queries == 0 || h.release.downloads == 0 {
		t.Fatalf("debe consultar y descargar del servidor inyectado: q=%d d=%d", h.release.queries, h.release.downloads)
	}
	if got := h.exeContent(t); got != "NEW-BINARY" {
		t.Fatalf("el ejecutable inyectado no se reemplazó: %q", got)
	}
	want, err := filepath.EvalSymlinks(h.exe)
	if err != nil {
		t.Fatal(err)
	}
	if !h.ran || h.runExe != want {
		t.Fatalf("el hijo debe lanzarse con el ejecutable inyectado: ran=%v exe=%q", h.ran, h.runExe)
	}
}

// C-13: con la comprobación desactivada, update consulta y actualiza igual.
func TestHCD035C13(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "1")
	h := newUpdateHarness(t, "v1.0.0", "v1.1.0")
	if err := runUpdateWith(h.opts, h.deps()); err != nil {
		t.Fatalf("update: %v\n%s", err, h.out.String())
	}
	if h.release.queries == 0 {
		t.Fatalf("C-13: update no consultó con la comprobación desactivada:\n%s", h.out.String())
	}
	if got := h.exeContent(t); got != "NEW-BINARY" || !h.ran {
		t.Fatalf("C-13: update no actualizó: exe=%q ran=%v\n%s", got, h.ran, h.out.String())
	}
}

// C-14: «al día» solo después de consultar.
func TestHCD035C14(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "1")
	h := newUpdateHarness(t, "v1.1.0", "v1.1.0")
	if err := runUpdateWith(h.opts, h.deps()); err != nil {
		t.Fatalf("update: %v\n%s", err, h.out.String())
	}
	if h.release.queries == 0 {
		t.Fatalf("C-14: dijo «al día» sin consultar:\n%s", h.out.String())
	}
	if !strings.Contains(h.out.String(), "ya está al día (v1.1.0)") || h.ran {
		t.Fatalf("C-14: con la misma versión debe decir «al día» sin reinstalar:\n%s", h.out.String())
	}
}

func writeManifestFixture(t *testing.T, home, content string) {
	t.Helper()
	path := installer.ManifestPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// C-10: sin variable, el hijo recibe el bin_dir registrado en el manifiesto.
func TestHCD035C10(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")
	t.Setenv("PI_GOMEMORY_BIN_DIR", "")
	h := newUpdateHarness(t, "v1.0.0", "v1.1.0")
	custom := filepath.Join(t.TempDir(), "custom-bin")
	writeManifestFixture(t, h.opts.HomeDir, fmt.Sprintf(`{"version":"v1.0.0","bin_dir":%q,"files":{}}`, custom))
	if err := runUpdateWith(h.opts, h.deps()); err != nil {
		t.Fatalf("update: %v\n%s", err, h.out.String())
	}
	if got, _ := envValue(h.runEnv, "PI_GOMEMORY_BIN_DIR"); got != custom {
		t.Fatalf("C-10: el hijo recibe PI_GOMEMORY_BIN_DIR=%q, quiero %q", got, custom)
	}
}

// C-11: con la variable no vacía, el hijo la recibe tal cual y el manifiesto
// no se lee (uno inválido no hace fallar update).
func TestHCD035C11(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")
	explicit := filepath.Join(t.TempDir(), "explicit-bin")
	t.Setenv("PI_GOMEMORY_BIN_DIR", explicit)
	h := newUpdateHarness(t, "v1.0.0", "v1.1.0")
	writeManifestFixture(t, h.opts.HomeDir, "{no es json")
	if err := runUpdateWith(h.opts, h.deps()); err != nil {
		t.Fatalf("C-11: con la variable puesta no debe leerse el manifiesto: %v\n%s", err, h.out.String())
	}
	if got, _ := envValue(h.runEnv, "PI_GOMEMORY_BIN_DIR"); got != explicit {
		t.Fatalf("C-11: el hijo recibe PI_GOMEMORY_BIN_DIR=%q, quiero %q", got, explicit)
	}
}

// C-12: sin manifiesto, con bin_dir vacío o ENOTDIR no se añade BIN_DIR; la
// marca de desinstalado no cambia nada.
func TestHCD035C12(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")
	t.Setenv("PI_GOMEMORY_BIN_DIR", "")
	custom := filepath.Join(t.TempDir(), "custom-bin")
	cases := map[string]struct {
		prepare func(t *testing.T, home string)
		want    string
	}{
		"sin manifiesto": {func(*testing.T, string) {}, ""},
		"bin_dir vacío": {func(t *testing.T, home string) {
			writeManifestFixture(t, home, `{"version":"v1.0.0","bin_dir":"","files":{}}`)
		}, ""},
		"ENOTDIR": {func(t *testing.T, home string) {
			if err := os.WriteFile(filepath.Join(home, ".pi-gomemory"), []byte("archivo"), 0644); err != nil {
				t.Fatal(err)
			}
		}, ""},
		"marca de desinstalado": {func(t *testing.T, home string) {
			writeManifestFixture(t, home, fmt.Sprintf(`{"version":"v1.0.0","bin_dir":%q,"files":{}}`, custom))
			if err := os.WriteFile(installer.UninstalledMarkerPath(home), []byte("v1.0.0\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}, custom},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			h := newUpdateHarness(t, "v1.0.0", "v1.1.0")
			c.prepare(t, h.opts.HomeDir)
			if err := runUpdateWith(h.opts, h.deps()); err != nil {
				t.Fatalf("C-12: update: %v\n%s", err, h.out.String())
			}
			if got, _ := envValue(h.runEnv, "PI_GOMEMORY_BIN_DIR"); got != c.want {
				t.Fatalf("C-12: el hijo recibe PI_GOMEMORY_BIN_DIR=%q, quiero %q", got, c.want)
			}
		})
	}
}

// C-24: con versión nueva y un manifiesto ilegible, update falla antes de
// descargar y el ejecutable queda intacto; sin versión nueva no se lee.
func TestHCD035C24(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")
	t.Setenv("PI_GOMEMORY_BIN_DIR", "")
	h := newUpdateHarness(t, "v1.0.0", "v1.1.0")
	writeManifestFixture(t, h.opts.HomeDir, "{no es json")
	if err := runUpdateWith(h.opts, h.deps()); err == nil {
		t.Fatalf("C-24: un manifiesto ilegible debe hacer fallar update:\n%s", h.out.String())
	}
	if h.release.downloads != 0 || h.ran || h.exeContent(t) != "OLD-BINARY" {
		t.Fatalf("C-24: debe fallar antes de descargar: d=%d ran=%v exe=%q", h.release.downloads, h.ran, h.exeContent(t))
	}

	same := newUpdateHarness(t, "v1.1.0", "v1.1.0")
	writeManifestFixture(t, same.opts.HomeDir, "{no es json")
	if err := runUpdateWith(same.opts, same.deps()); err != nil || !strings.Contains(same.out.String(), "ya está al día") {
		t.Fatalf("C-24: sin versión nueva el manifiesto no se lee: %v\n%s", err, same.out.String())
	}
}

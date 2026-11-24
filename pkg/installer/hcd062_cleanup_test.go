package installer

import (
	"bytes"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// HCD-062 T8: Ctrl-C durante el paso borra los temporales y sale con 130.
func TestHCD062StaleSignalCleansTemp(t *testing.T) {
	opts := toolSandbox(t)
	got := make(chan int, 1)
	old := signalExit
	signalExit = func(code int) { got <- code }
	t.Cleanup(func() { signalExit = old })

	// Receptor propio: sin el manejador del paso, la prueba falla por
	// aserción en vez de morir por la señal.
	guard := make(chan os.Signal, 1)
	signal.Notify(guard, syscall.SIGINT)
	defer signal.Stop(guard)

	dir, cleanup, err := depsTempDir(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-got:
		if code != 130 {
			t.Fatalf("código = %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("SIGINT no se capturó")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("el temporal %s debe borrarse: %v", dir, err)
	}
}

// T8: install retira los temporales huérfanos y no toca nada más.
func TestHCD062StaleTempsRemovedByInstall(t *testing.T) {
	opts := toolSandbox(t)
	stale := filepath.Join(opts.LocalBinDir, depsTempPrefix+"abc")
	keep := filepath.Join(opts.LocalBinDir, "otro-dir")
	for _, d := range []string{stale, keep} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	fakeTool(t, opts.LocalBinDir, "pi")
	withTools(t, []Tool{{Name: "pi", Required: true}})
	if err := InstallAll(repoAssets(t), opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("debe retirarse %s: %v", stale, err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("no debe tocarse %s: %v", keep, err)
	}
}

// T8 y T11: uninstall retira temporales huérfanos y conserva las cuatro
// herramientas diciendo cómo quitarlas.
func TestHCD062UninstallKeepsToolsAndCleansTemps(t *testing.T) {
	opts := installSandbox(t)
	t.Setenv("PATH", "/usr/bin:/bin")
	stale := filepath.Join(opts.LocalBinDir, depsTempPrefix+"xyz")
	if err := os.MkdirAll(stale, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"pi", "mem", "codegraph", "codebase-memory-mcp"} {
		fakeTool(t, opts.LocalBinDir, n)
	}
	var out bytes.Buffer
	if err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true}, &out); err != nil {
		t.Fatalf("Uninstall: %v\n%s", err, out.String())
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("debe retirarse %s: %v", stale, err)
	}
	for _, n := range []string{"pi", "mem", "codegraph"} {
		if _, err := os.Stat(filepath.Join(opts.LocalBinDir, n)); err != nil {
			t.Fatalf("%s debe conservarse: %v", n, err)
		}
	}
	for _, want := range []string{"Se conservan pi, mem, codegraph y herdr", "codegraph/main/install.sh | sh -s -- --uninstall", "gomemory/main/scripts/install.sh | bash -s -- --uninstall"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("falta %q en:\n%s", want, out.String())
		}
	}
}

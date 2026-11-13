package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// HCD-030 · C-004: uninstall solo retira el launcher que pertenece a la
// instalación de su HOME. Junto al ejecutable pero fuera de HOME y sin respaldo
// del manifiesto ni de PI_GOMEMORY_BIN_DIR, se omite e informa (Q1 → A).
func TestHCD030LauncherOutsideHome(t *testing.T) {
	t.Setenv("PI_GOMEMORY_BIN_DIR", "")
	launcher := GenerateHarnessScript()

	newOpts := func(t *testing.T) *Options {
		home := t.TempDir()
		return &Options{HomeDir: home, PiAgentDir: filepath.Join(home, ".pi", "agent"), LocalBinDir: filepath.Join(home, ".local", "bin")}
	}
	// exeIn simula que pi-gomemory-setup se ejecuta desde dir, con su launcher al lado.
	exeIn := func(t *testing.T, dir string) string {
		t.Helper()
		writeFile(t, filepath.Join(dir, "pi-gomemory"), launcher)
		prev := executablePath
		executablePath = func() (string, error) { return filepath.Join(dir, "pi-gomemory-setup"), nil }
		t.Cleanup(func() { executablePath = prev })
		return filepath.Join(dir, "pi-gomemory")
	}
	planFor := func(t *testing.T, opts *Options) *UninstallPlan {
		t.Helper()
		plan, err := BuildUninstallPlan(repoAssets(t), opts, UninstallFlags{})
		if err != nil {
			t.Fatal(err)
		}
		return plan
	}

	t.Run("fuera de HOME y sin manifiesto: se omite e informa", func(t *testing.T) {
		opts := newOpts(t)
		target := exeIn(t, t.TempDir())
		a := actionFor(planFor(t, opts), target)
		if a == nil || a.Kind != ActionSkip || !strings.Contains(a.Reason, "fuera de HOME y sin manifiesto") {
			t.Fatalf("quiero [%s] «fuera de HOME y sin manifiesto…», obtengo %+v", ActionSkip, a)
		}
		if err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true}, &strings.Builder{}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(target); err != nil {
			t.Fatalf("uninstall borró el launcher fuera de HOME: %v", err)
		}
	})

	t.Run("fuera de HOME con el BIN_DIR del manifiesto: se retira", func(t *testing.T) {
		opts := newOpts(t)
		bin := t.TempDir()
		target := exeIn(t, bin)
		writeFile(t, ManifestPath(opts.HomeDir), `{"version":"v1.12.1","bin_dir":"`+bin+`","files":{}}`)
		if a := actionFor(planFor(t, opts), target); a == nil || a.Kind != ActionRemove {
			t.Fatalf("con el manifiesto que registra %s quiero [%s], obtengo %+v", bin, ActionRemove, a)
		}
	})

	t.Run("junto al ejecutable dentro de HOME: se retira como antes", func(t *testing.T) {
		opts := newOpts(t)
		target := exeIn(t, filepath.Join(opts.HomeDir, "bin-propio"))
		if a := actionFor(planFor(t, opts), target); a == nil || a.Kind != ActionRemove {
			t.Fatalf("quiero [%s], obtengo %+v", ActionRemove, a)
		}
	})
}

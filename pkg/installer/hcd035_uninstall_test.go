package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const hcd035ManifestReason = "manifiesto de instalación (se retira si no hay fallos)"

func skipIfRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("como root los permisos no provocan el fallo (Q21)")
	}
}

// chmodFor cambia el modo de path y lo restaura al terminar la prueba.
func chmodFor(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, info.Mode().Perm()) })
}

func manifestExists(t *testing.T, home string) bool {
	t.Helper()
	_, err := os.Lstat(ManifestPath(home))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return err == nil
}

// C-01, C-21: si falla una retirada, el manifiesto se conserva, la salida
// normal lo avisa y el comando termina con error y sin «✓».
func TestHCD035C01(t *testing.T) {
	skipIfRoot(t)
	opts := installSandbox(t)
	chmodFor(t, opts.LocalBinDir, 0555)
	out, err := runUninstall(t, opts, UninstallFlags{Yes: true})
	if err == nil {
		t.Fatalf("la retirada del launcher debía fallar:\n%s", out)
	}
	if !manifestExists(t, opts.HomeDir) {
		t.Fatalf("C-01: un uninstall fallido retiró el manifiesto:\n%s", out)
	}
	if !strings.Contains(out, line(ActionKeep, ManifestPath(opts.HomeDir), "conservado por fallos previos")) {
		t.Fatalf("C-21: la salida normal no avisa del manifiesto conservado:\n%s", out)
	}
	if strings.Contains(out, "✓") {
		t.Fatalf("C-21: no debe anunciar éxito:\n%s", out)
	}
}

// C-02: sin ActionFail, el manifiesto es la última retirada, también tras
// --self y aunque un archivo cambiara después del plan (salida 0, HCD-034).
func TestHCD035C02(t *testing.T) {
	opts := installSandbox(t)
	self := filepath.Join(opts.LocalBinDir, "pi-gomemory-setup")
	writeFile(t, self, "binario sintético")
	status := filepath.Join(opts.PiAgentDir, "extensions", "gomemory-status.ts")
	afterPlan(t, func() { writeFile(t, status, "EDICION-PROPIA") })
	out, err := runUninstall(t, opts, UninstallFlags{Self: true, Verbose: true})
	if err != nil {
		t.Fatalf("un cambio tras el plan no es un fallo: %v\n%s", err, out)
	}
	if manifestExists(t, opts.HomeDir) {
		t.Fatalf("C-02: sin fallos el manifiesto debe retirarse:\n%s", out)
	}
	if !strings.Contains(out, "✓ pi-gomemory desinstalado; se conservaron 1 elementos que cambiaron después del plan\n") {
		t.Fatalf("C-02: debe mantenerse el éxito de HCD-034:\n%s", out)
	}
	applied := out[strings.Index(out, "¿Retirar"):]
	var last string
	for _, l := range strings.Split(applied, "\n") {
		if strings.HasPrefix(l, "  [") {
			last = l
		}
	}
	if want := strings.TrimSuffix(line(ActionRemove, ManifestPath(opts.HomeDir), hcd035ManifestReason), "\n"); last != want {
		t.Fatalf("C-02: la última acción aplicada debe ser el manifiesto; fue %q\n%s", last, out)
	}
}

// C-03: la simulación muestra la retirada condicional y no borra nada.
func TestHCD035C03(t *testing.T) {
	opts := installSandbox(t)
	out, err := runUninstall(t, opts, UninstallFlags{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, out)
	}
	if !strings.Contains(out, line(ActionRemove, ManifestPath(opts.HomeDir), hcd035ManifestReason)) {
		t.Fatalf("C-03: la simulación no muestra la retirada condicional del manifiesto:\n%s", out)
	}
	if !manifestExists(t, opts.HomeDir) {
		t.Fatal("C-03: la simulación borró el manifiesto")
	}
}

// C-16: un ActionFail del plan ajeno al manifiesto lo conserva.
func TestHCD035C16(t *testing.T) {
	skipIfRoot(t)
	opts := installSandbox(t)
	chmodFor(t, filepath.Join(opts.PiAgentDir, "themes", "gomemory.json"), 0000)
	out, err := runUninstall(t, opts, UninstallFlags{Yes: true})
	if err == nil {
		t.Fatalf("el tema ilegible debía fallar en el plan:\n%s", out)
	}
	if !manifestExists(t, opts.HomeDir) {
		t.Fatalf("C-16: un ActionFail del plan retiró el manifiesto:\n%s", out)
	}
}

// C-23 (guarda sin rojo): con PI_GOMEMORY_BIN_DIR, DefaultOptions → InstallAll
// deja launcher y manifiesto en ese directorio y nada en ~/.local/bin.
func TestHCD035C23(t *testing.T) {
	home := sandbox(t).HomeDir
	custom := filepath.Join(t.TempDir(), "custom-bin")
	t.Setenv("PI_GOMEMORY_BIN_DIR", custom)
	opts, err := DefaultOptions()
	if err != nil {
		t.Fatal(err)
	}
	if err := InstallAll(repoAssets(t), opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(custom, "pi-gomemory")); err != nil {
		t.Fatalf("C-23: launcher ausente en %s: %v", custom, err)
	}
	if _, err := os.Lstat(filepath.Join(home, ".local", "bin", "pi-gomemory")); !os.IsNotExist(err) {
		t.Fatalf("C-23: no debe crearse launcher en ~/.local/bin: %v", err)
	}
	m, err := ReadManifest(home)
	if err != nil || m == nil || m.BinDir != custom {
		t.Fatalf("C-23: el manifiesto debe registrar %s: %+v %v", custom, m, err)
	}
}

// C-22: un manifiesto ilegible se retira aunque su inspección falle; el
// primer uninstall sale con error y el segundo ya no falla por él.
func TestHCD035C22(t *testing.T) {
	opts := installSandbox(t)
	writeFile(t, ManifestPath(opts.HomeDir), "{no es json")
	out, err := runUninstall(t, opts, UninstallFlags{Yes: true})
	if err == nil {
		t.Fatalf("C-22: la inspección fallida del manifiesto debe salir con error:\n%s", out)
	}
	if manifestExists(t, opts.HomeDir) {
		t.Fatalf("C-22: el manifiesto ilegible debe retirarse:\n%s", out)
	}
	out, err = runUninstall(t, opts, UninstallFlags{Yes: true})
	if err != nil {
		t.Fatalf("C-22: el segundo uninstall no debe fallar: %v\n%s", err, out)
	}
}

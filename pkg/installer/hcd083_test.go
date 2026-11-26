package installer

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// H-01: scripts/clean-install-check.sh no debe exigir codebase-memory-mcp
// (retirado del catálogo) y debe comprobar exactamente los tools de Catalog().
func TestHCD083CleanInstallCheckMatchesCatalog(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scripts", "clean-install-check.sh"))
	if err != nil {
		t.Fatalf("leer scripts/clean-install-check.sh: %v", err)
	}
	content := string(raw)
	if strings.Contains(content, "codebase-memory-mcp") {
		t.Fatalf("scripts/clean-install-check.sh todavía referencia codebase-memory-mcp (retirado)")
	}
	for _, name := range ToolNames() {
		if !strings.Contains(content, name) {
			t.Fatalf("scripts/clean-install-check.sh no comprueba el tool del catálogo %q", name)
		}
	}
}

// H-03: Sync no debe entrar en pánico ni leer states[0] cuando falta la entrada
// de una herramienta presente en byState (p. ej. sondas vacías).
func TestHCD083SyncMissingProbeNoPanic(t *testing.T) {
	opts := toolSandbox(t)
	fakeTool(t, opts.LocalBinDir, "pi")
	withTools(t, []Tool{
		{Name: "pi", Required: true, Script: "https://invalid.example/pi.sh", Interp: "sh", Manual: "m"},
	})
	withSyncProbes(t, func(*Options) []DepProbe { return nil })

	var out bytes.Buffer
	res, err := Sync(opts, SyncOptions{Yes: true}, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("Sync con sondas vacías falló: %v\n%s", err, out.String())
	}
	if len(res) != 1 || res[0].Action != "current" {
		t.Fatalf("resultado = %+v, quiero [{Name:pi Action:current}]", res)
	}
}

// H-04: InstallCCExtensions y RetireAntigravityPackage (HCD-085) deben ejecutar
// pi con CommandContext (timeout) y stdin aislado para no colgar indefinidamente.
func TestHCD083PluginInstallTimeout(t *testing.T) {
	opts := sandbox(t)
	binDir := t.TempDir()
	fakePi := filepath.Join(binDir, "pi")
	if err := os.WriteFile(fakePi, []byte("#!/bin/sh\nexec /bin/sleep 10\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	prev := pluginInstallTimeout
	pluginInstallTimeout = 150 * time.Millisecond
	t.Cleanup(func() { pluginInstallTimeout = prev })

	writeFile(t, filepath.Join(opts.PiAgentDir, "settings.json"), `{"packages":["npm:pi-antigravity@0.15.0"]}`)
	start := time.Now()
	if err := RetireAntigravityPackage(opts.PiAgentDir, io.Discard); err != nil {
		t.Fatalf("RetireAntigravityPackage: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond || elapsed > 3*time.Second {
		t.Fatalf("RetireAntigravityPackage tardó %v; debía cortar pi remove por timeout (~150ms)", elapsed)
	}

	start = time.Now()
	if err := InstallCCExtensions(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatalf("InstallCCExtensions: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond || elapsed > 3*time.Second {
		t.Fatalf("InstallCCExtensions tardó %v; debía cortar por timeout (~150ms)", elapsed)
	}
}

// H-06: Sync y EnsureTools deben propagar el error cuando lite.json está
// corrupto en lugar de asumir silenciosamente lite=false.
func TestHCD083CorruptLiteJSONFails(t *testing.T) {
	opts := toolSandbox(t)
	writeFile(t, LiteStatePath(opts.HomeDir), "{json-roto")
	withTools(t, []Tool{
		{Name: "pi", Required: true, Script: "https://invalid.example/pi.sh", Interp: "sh", Manual: "m"},
	})
	withSyncProbes(t, func(*Options) []DepProbe { return nil })

	var out bytes.Buffer
	if _, err := Sync(opts, SyncOptions{Yes: true}, strings.NewReader(""), &out); err == nil {
		t.Fatalf("Sync con lite.json corrupto devolvió nil; debe devolver error")
	}
	if err := EnsureTools(opts, &out); err == nil {
		t.Fatalf("EnsureTools con lite.json corrupto devolvió nil; debe devolver error")
	}
}

// H-08: scripts/build-release.sh no debe llevar una versión obsoleta
// hardcodeada (v1.26.1) y debe rechazar versiones inválidas antes de compilar.
func TestHCD083BuildReleaseVersionGuard(t *testing.T) {
	script := filepath.Join("..", "..", "scripts", "build-release.sh")
	raw, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("leer scripts/build-release.sh: %v", err)
	}
	if strings.Contains(string(raw), "v1.26.1") {
		t.Fatalf("scripts/build-release.sh mantiene versión de fallback obsoleta v1.26.1")
	}
	cmd := exec.Command("bash", script, "version-invalida")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("scripts/build-release.sh aceptó versión inválida:\n%s", out)
	}
	if !strings.Contains(string(out), "versión válida") {
		t.Fatalf("se esperaba rechazo inmediato por versión inválida, salida:\n%s", out)
	}
}

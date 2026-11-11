package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeTerminal simula la terminal: tty indica si es interactiva y answer lo que se teclea.
func fakeTerminal(t *testing.T, tty bool, answer string) {
	t.Helper()
	oldTTY, oldIn := stdinIsTerminal, confirmInput
	stdinIsTerminal = func() bool { return tty }
	confirmInput = strings.NewReader(answer)
	t.Cleanup(func() { stdinIsTerminal, confirmInput = oldTTY, oldIn })
}

// Tarea 5 · C-02, C-04, C-08, C-10: retira lo propio, conserva la memoria,
// sigue ante fallos y es repetible.
func TestHCD011Apply(t *testing.T) {
	opts := installSandbox(t)
	fakeTerminal(t, false, "")
	memDB := filepath.Join(opts.HomeDir, ".gomemory", "memory.db")
	hcdDoc := filepath.Join(opts.HomeDir, "proyecto", "hcd", "HCD-001.md")
	writeFile(t, memDB, "db")
	writeFile(t, hcdDoc, "doc")

	var out bytes.Buffer
	if err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true}, &out); err != nil {
		t.Fatalf("uninstall: %v\n%s", err, out.String())
	}
	ext := filepath.Join(opts.PiAgentDir, "extensions")
	for _, p := range []string{
		filepath.Join(ext, "gomemory-status.ts"),
		filepath.Join(opts.PiAgentDir, "themes", "gomemory.json"),
		filepath.Join(opts.LocalBinDir, "pi-gomemory"),
		filepath.Join(SkillRepoDir(opts.HomeDir), "hcd"),
		ManifestPath(opts.HomeDir),
	} {
		if _, err := os.Lstat(p); err == nil {
			t.Errorf("sigue presente: %s", p)
		}
	}
	mcp := readFile(t, filepath.Join(opts.PiAgentDir, "mcp.json"))
	if strings.Contains(mcp, `"gomemory"`) || strings.Contains(mcp, "codebase-memory") {
		t.Errorf("mcp.json conserva entradas de pi-gomemory:\n%s", mcp)
	}
	if _, err := os.Stat(filepath.Join(opts.PiAgentDir, "AGENTS.md")); err == nil {
		t.Error("AGENTS.md solo tenía el bloque y debía borrarse")
	}
	settings := readFile(t, filepath.Join(opts.PiAgentDir, "settings.json"))
	if strings.Contains(settings, `"theme"`) || strings.Contains(settings, SkillRepoSetting) {
		t.Errorf("settings.json conserva tema o registro:\n%s", settings)
	}
	if !IsUninstalled(opts.HomeDir) {
		t.Error("falta la marca de desinstalado")
	}
	if readFile(t, memDB) != "db" || readFile(t, hcdDoc) != "doc" {
		t.Error("se tocó la memoria de gomemory")
	}

	// C-04: segunda ejecución sin error y sin nada que retirar.
	out.Reset()
	if err := Uninstall(repoAssets(t), opts, UninstallFlags{}, &out); err != nil {
		t.Fatalf("segundo uninstall: %v", err)
	}
	if !strings.Contains(out.String(), "nada que retirar") {
		t.Errorf("salida del segundo uninstall:\n%s", out.String())
	}

	// C-10: un JSON ilegible no se toca, el resto continúa y se informa el error.
	opts2 := installSandbox(t)
	bad := filepath.Join(opts2.PiAgentDir, "mcp.json")
	writeFile(t, bad, "{roto")
	out.Reset()
	err := Uninstall(repoAssets(t), opts2, UninstallFlags{Yes: true}, &out)
	if err == nil {
		t.Fatal("con un fallo, uninstall debe devolver error")
	}
	if readFile(t, bad) != "{roto" {
		t.Error("se modificó el JSON ilegible")
	}
	if _, err := os.Stat(filepath.Join(opts2.PiAgentDir, "extensions", "gomemory-status.ts")); err == nil {
		t.Error("el fallo detuvo el resto de acciones")
	}
	if !strings.Contains(out.String(), ActionFail) {
		t.Errorf("el resumen no informa el fallo:\n%s", out.String())
	}
}

// Tarea 6 · C-15: sin terminal ni --yes no se toca nada; en terminal hay que confirmar.
func TestHCD011Confirm(t *testing.T) {
	opts := installSandbox(t)
	before := snapshotTree(t, opts.HomeDir)
	same := func(label string) {
		after := snapshotTree(t, opts.HomeDir)
		if len(after) != len(before) {
			t.Fatalf("%s: cambió el árbol", label)
		}
		for p, v := range before {
			if after[p] != v {
				t.Fatalf("%s: se modificó %s", label, p)
			}
		}
	}
	var out bytes.Buffer

	fakeTerminal(t, false, "")
	err := Uninstall(repoAssets(t), opts, UninstallFlags{}, &out)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("sin terminal ni --yes: quiero error que mencione --yes, obtengo %v", err)
	}
	same("sin terminal")

	fakeTerminal(t, true, "n\n")
	if err := Uninstall(repoAssets(t), opts, UninstallFlags{}, &out); err == nil {
		t.Fatal("al no confirmar debe salir con error")
	}
	same("sin confirmar")

	fakeTerminal(t, true, "s\n")
	if err := Uninstall(repoAssets(t), opts, UninstallFlags{}, &out); err != nil {
		t.Fatalf("confirmado: %v", err)
	}
	if !IsUninstalled(opts.HomeDir) {
		t.Error("tras confirmar no se desinstaló")
	}
}

// Tarea 7 · C-09: --verbose imprime cada acción y la guarda en un log.
func TestHCD011Verbose(t *testing.T) {
	opts := installSandbox(t)
	fakeTerminal(t, false, "")
	writeFile(t, filepath.Join(SkillRepoDir(opts.HomeDir), "mia", "SKILL.md"), "x")
	var out bytes.Buffer
	if err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true, Verbose: true}, &out); err != nil {
		t.Fatal(err)
	}
	logPath := UninstallLogPath(opts.HomeDir)
	if !strings.Contains(out.String(), logPath) {
		t.Fatalf("no se muestra la ruta del log:\n%s", out.String())
	}
	log := readFile(t, logPath)
	for _, want := range []string{"[" + ActionRemove + "]", "[" + ActionSkip + "]", "gomemory-status.ts", "mia"} {
		if !strings.Contains(log, want) || !strings.Contains(out.String(), want) {
			t.Errorf("falta %q en la salida o el log", want)
		}
	}
}

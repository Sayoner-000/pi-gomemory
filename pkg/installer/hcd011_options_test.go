package installer

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func settingsPackages(t *testing.T, opts *Options) []string {
	t.Helper()
	var s struct {
		Packages []string `json:"packages"`
	}
	_ = json.Unmarshal([]byte(readFile(t, filepath.Join(opts.PiAgentDir, "settings.json"))), &s)
	return s.Packages
}

// Tarea 8 · C-13: sin opciones se conservan paquetes y binario; --packages y --self los retiran.
func TestHCD011Options(t *testing.T) {
	fakeTerminal(t, false, "")
	var out bytes.Buffer

	opts := installSandbox(t)
	cc := filepath.Join(opts.PiAgentDir, "pi-cc-extensions.json")
	self := filepath.Join(opts.LocalBinDir, "pi-gomemory-setup")
	writeFile(t, self, "binario")
	setExecutable(t, self)
	if err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true}, &out); err != nil {
		t.Fatal(err)
	}
	if len(settingsPackages(t, opts)) != 2 {
		t.Errorf("sin --packages se tocaron los paquetes: %v", settingsPackages(t, opts))
	}
	if _, err := os.Stat(cc); err != nil {
		t.Error("sin --packages se borró pi-cc-extensions.json")
	}
	if _, err := os.Stat(self); err != nil {
		t.Error("sin --self se borró el binario")
	}

	opts = installSandbox(t)
	cc = filepath.Join(opts.PiAgentDir, "pi-cc-extensions.json")
	settings := filepath.Join(opts.PiAgentDir, "settings.json")
	writeFile(t, settings, strings.Replace(readFile(t, settings), `"packages": [`, `"packages": [
    "npm:otro-paquete@1.0.0",`, 1))
	self = filepath.Join(opts.LocalBinDir, "pi-gomemory-setup")
	writeFile(t, self, "binario")
	setExecutable(t, self)
	if err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true, Packages: true, Self: true}, &out); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if pk := settingsPackages(t, opts); len(pk) != 1 || pk[0] != "npm:otro-paquete@1.0.0" {
		t.Errorf("--packages: quiero solo el paquete ajeno, obtengo %v", pk)
	}
	if _, err := os.Stat(cc); err == nil {
		t.Error("--packages no borró pi-cc-extensions.json con valores por defecto")
	}
	if _, err := os.Stat(self); err == nil {
		t.Error("--self no borró el binario del BIN_DIR")
	}

	// pi-cc-extensions.json editado se conserva aun con --packages.
	opts = installSandbox(t)
	cc = filepath.Join(opts.PiAgentDir, "pi-cc-extensions.json")
	writeFile(t, cc, `{"mia": true}`)
	plan, _ := BuildUninstallPlan(repoAssets(t), opts, UninstallFlags{Packages: true})
	if a := actionFor(plan, cc); a == nil || a.Kind != ActionSkip {
		t.Errorf("pi-cc-extensions.json editado: quiero omitir, obtengo %+v", a)
	}
}

func setExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatal(err)
	}
}

// Tarea 9 · C-12: tras desinstalar, doctor dice «pi-gomemory no instalado» sin exigir pi.
func TestHCD011Doctor(t *testing.T) {
	fakeTerminal(t, false, "")
	opts := installSandbox(t)
	var out bytes.Buffer
	if err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true}, &out); err != nil {
		t.Fatal(err)
	}
	items := RunDoctor(opts)
	found := false
	for _, it := range items {
		if it.Status == "INFO" && it.Name == "pi-gomemory" && strings.Contains(it.Details, "no instalado") {
			found = true
		}
		if it.Status == "FAIL" && (strings.Contains(it.Name, "Tema") || strings.Contains(it.Name, "Skills")) {
			t.Errorf("doctor sigue exigiendo componentes desinstalados: %+v", it)
		}
	}
	if !found {
		t.Errorf("doctor no muestra «pi-gomemory no instalado»: %+v", items)
	}
}

// Tarea 9 · C-03: reinstalar tras desinstalar deja doctor como una instalación nueva.
func TestHCD011Reinstall(t *testing.T) {
	fakeTerminal(t, false, "")
	fresh := installSandbox(t)
	want := doctorSummary(RunDoctor(fresh))

	opts := installSandbox(t)
	var out bytes.Buffer
	if err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true}, &out); err != nil {
		t.Fatal(err)
	}
	if err := InstallAll(repoAssets(t), opts); err != nil {
		t.Fatalf("reinstalación: %v", err)
	}
	if IsUninstalled(opts.HomeDir) {
		t.Error("la reinstalación no retiró la marca")
	}
	if got := doctorSummary(RunDoctor(opts)); got != want {
		t.Errorf("doctor tras reinstalar:\n%s\nnuevo:\n%s", got, want)
	}
}

func doctorSummary(items []CheckItem) string {
	var b strings.Builder
	for _, it := range items {
		b.WriteString(it.Name + "=" + it.Status + "\n")
	}
	return b.String()
}

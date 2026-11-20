package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// HCD-032 · C-001: install, update y uninstall conservan el modo de
// ~/.pi/agent/AGENTS.md; uno nuevo se crea con 0644 como hasta ahora.
func TestHCD032AgentsModePreserved(t *testing.T) {
	t.Parallel()

	modeOf := func(t *testing.T, path string) os.FileMode {
		t.Helper()
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		return info.Mode().Perm()
	}

	// HCD-045: install ya no escribe el protocolo; al retirar el bloque
	// heredado conserva el modo, y sin AGENTS.md no crea ninguno.
	t.Run("install conserva 0600", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "AGENTS.md")
		writeFile(t, path, "mis reglas\n\n"+instructionsStart+"\nbloque\n"+instructionsEnd+"\n")
		if err := os.Chmod(path, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := RetireLegacyInstructions(dir); err != nil {
			t.Fatal(err)
		}
		if m := modeOf(t, path); m != 0600 {
			t.Fatalf("install dejó AGENTS.md en %o; quiero 600", m)
		}
		if got := readFile(t, path); strings.Contains(got, instructionsStart) || !strings.Contains(got, "mis reglas") {
			t.Fatalf("contenido inesperado tras install: %q", got)
		}
	})

	t.Run("install no crea AGENTS.md", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := RetireLegacyInstructions(dir); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(err) {
			t.Fatalf("install creó AGENTS.md: %v", err)
		}
	})

	t.Run("uninstall conserva 0600", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "AGENTS.md")
		writeFile(t, path, "mis reglas\n\n"+instructionsStart+"\nbloque\n"+instructionsEnd+"\n")
		if err := os.Chmod(path, 0600); err != nil {
			t.Fatal(err)
		}
		var plan UninstallPlan
		plan.planAgents(dir)
		a := actionFor(&plan, path+" › bloque pi-gomemory")
		if a == nil || a.apply == nil {
			t.Fatalf("el plan no retira el bloque: %+v", plan.Actions)
		}
		if err := a.apply(); err != nil {
			t.Fatal(err)
		}
		if m := modeOf(t, path); m != 0600 {
			t.Fatalf("uninstall dejó AGENTS.md en %o; quiero 600", m)
		}
		if got := readFile(t, path); strings.Contains(got, instructionsStart) || !strings.Contains(got, "mis reglas") {
			t.Fatalf("contenido inesperado tras uninstall: %q", got)
		}
	})
}

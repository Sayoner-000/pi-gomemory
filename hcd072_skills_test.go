package main

import (
	"os"
	"path/filepath"
	"pi-gomemory/pkg/installer"
	"strings"
	"testing"
)

func TestHCD072EmbeddedScheduling(t *testing.T) {
	home := t.TempDir()
	if _, err := installer.SyncSkillRepo(EmbeddedAssets, home); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"hcd-implement", "hcd-verify"} {
		path := filepath.Join(home, ".pi-gomemory", "skills", name, "SKILL.md")
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		// HCD-079: reglas de gomemory 2.59.0 (stale como progreso con pendientes; --stale sin comando).
		for _, rule := range []string{"No revalides tareas anteriores solo por caducidad", "No ejecutes la misma comprobación fuera y dentro del motor", "Antes del cierre", "cuentan como progreso", "en una sola llamada", "mem hcd verify --stale --json --client <agente>"} {
			if !strings.Contains(string(b), rule) {
				t.Errorf("%s missing %s", name, rule)
			}
		}
		if err := os.WriteFile(path, []byte("personalizada"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := installer.SyncSkillRepo(EmbeddedAssets, home); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"hcd-implement", "hcd-verify"} {
		b, err := os.ReadFile(filepath.Join(home, ".pi-gomemory", "skills", name, "SKILL.md"))
		if err != nil || string(b) != "personalizada" {
			t.Fatalf("custom skill overwritten: %s %v", name, err)
		}
	}
}

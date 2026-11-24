package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreSkillsFailsWhenUninstalled(t *testing.T) {
	home := t.TempDir()
	assets := hcd073Assets()
	opts := &Options{
		HomeDir:     home,
		PiAgentDir:  filepath.Join(home, ".pi", "agent"),
		LocalBinDir: filepath.Join(home, ".local", "bin"),
	}
	if _, err := InstallSkills(assets, opts); err != nil {
		t.Fatal(err)
	}
	skillFile := filepath.Join(SkillRepoDir(home), "hcd-close", "SKILL.md")
	hcd073Write(t, skillFile, "edición local\n")

	marker := UninstalledMarkerPath(home)
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("1.29.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := RestoreSkills(assets, opts, []string{"hcd-close"})
	if err == nil {
		t.Fatal("RestoreSkills debe fallar si pi-gomemory está desinstalado")
	}
	if got := hcd073Read(t, skillFile); got != "edición local\n" {
		t.Fatalf("RestoreSkills borró o modificó la skill en estado desinstalado: %q", got)
	}
}

func TestRunDoctorLiteModeSkipsCodegraphMCP(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	piFake := filepath.Join(binDir, "pi")
	script := "#!/bin/sh\nif [ \"$1\" = \"mcp\" ] && [ \"$2\" = \"list\" ]; then\n  echo '{\"servers\":[{\"name\":\"gomemory\",\"state\":\"connected\"}]}'\n  exit 0\nfi\nexit 0\n"
	if err := os.WriteFile(piFake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := writeLiteState(home, liteState{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	opts := &Options{
		HomeDir:     home,
		PiAgentDir:  filepath.Join(home, ".pi", "agent"),
		LocalBinDir: binDir,
	}

	items := RunDoctor(opts)
	var found *CheckItem
	for i := range items {
		if items[i].Name == "MCP codegraph" {
			found = &items[i]
			break
		}
	}
	if found == nil {
		t.Fatal("falta CheckItem MCP codegraph en RunDoctor")
	}
	if found.Status != "INFO" {
		t.Fatalf("en modo lite, MCP codegraph debe ser INFO, no %s (%s)", found.Status, found.Details)
	}
}

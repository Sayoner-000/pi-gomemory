package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// HCD-003, adaptada a HCD-006: ni la skill propia en ~/.pi/agent/skills ni una
// copia embebida editada en el repositorio se borran o se pisan.
func TestPruneKeepsUserSkill(t *testing.T) {
	opts := sandbox(t)
	writeFile(t, filepath.Join(opts.HomeDir, ".config", "opencode", "skills", "hcd", "SKILL.md"), "host")
	userSkill := filepath.Join(opts.PiAgentDir, "skills", "hcd", "SKILL.md")
	writeFile(t, userSkill, "skill propia del usuario")

	assets := fstest.MapFS{"skills/verify-loop/SKILL.md": {Data: []byte("embebida")}}
	if _, err := InstallSkills(assets, opts); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, userSkill); got != "skill propia del usuario" {
		t.Fatalf("se borró o cambió la skill del usuario: %q", got)
	}

	edited := filepath.Join(SkillRepoDir(opts.HomeDir), "verify-loop", "SKILL.md")
	writeFile(t, edited, "editada")
	if _, err := InstallSkills(assets, opts); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, edited); got != "editada" {
		t.Fatal("se pisó una copia embebida editada por la persona")
	}
}

func TestBinDirOverride(t *testing.T) {
	t.Setenv("PI_GOMEMORY_BIN_DIR", "/opt/pg/bin")
	opts, err := DefaultOptions()
	if err != nil {
		t.Fatal(err)
	}
	if opts.LocalBinDir != "/opt/pg/bin" {
		t.Fatalf("LocalBinDir = %s", opts.LocalBinDir)
	}
	t.Setenv("PI_GOMEMORY_BIN_DIR", "")
	opts, _ = DefaultOptions()
	if opts.LocalBinDir != filepath.Join(opts.HomeDir, ".local", "bin") {
		t.Fatalf("LocalBinDir por defecto = %s", opts.LocalBinDir)
	}
	if !strings.Contains(GenerateHarnessScript(), `export PATH="${SELF_DIR}:`) {
		t.Fatal("el launcher no añade su propio directorio al PATH")
	}
}

func TestWriteJSONObjectAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	writeFile(t, path, `{"a":1}`)
	if err := writeJSONObject(path, map[string]int{"b": 2}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); !strings.Contains(got, `"b": 2`) {
		t.Fatalf("contenido = %s", got)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("quedaron temporales: %v", entries)
	}
	// Un valor no serializable no toca el archivo previo.
	if err := writeJSONObject(path, map[string]any{"f": func() {}}); err == nil {
		t.Fatal("se esperaba error")
	}
	if got := readFile(t, path); !strings.Contains(got, `"b": 2`) {
		t.Fatal("el archivo previo cambió tras un error")
	}
	// Un destino sin permiso de escritura en el directorio falla sin truncar.
	ro := filepath.Join(dir, "ro")
	writeFile(t, filepath.Join(ro, "x.json"), `{"keep":true}`)
	if err := os.Chmod(ro, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(ro, 0755)
	if os.Getuid() != 0 {
		if err := writeJSONObject(filepath.Join(ro, "x.json"), map[string]int{}); err == nil {
			t.Fatal("se esperaba error en un directorio de solo lectura")
		}
		if got := readFile(t, filepath.Join(ro, "x.json")); got != `{"keep":true}` {
			t.Fatal("el archivo se truncó")
		}
	}
}

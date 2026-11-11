package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSkillRepoScan(t *testing.T) {
	opts := sandbox(t)
	h := opts.HomeDir
	writeFile(t, filepath.Join(h, ".agents", "skills", "humanizer", "SKILL.md"), "agents")
	writeFile(t, filepath.Join(h, ".config", "opencode", "skills", "hcd", "SKILL.md"), "oc")
	writeFile(t, filepath.Join(h, ".config", "opencode", "skills", "humanizer", "SKILL.md"), "oc")
	writeFile(t, filepath.Join(h, ".claude", "skills", "semantica", "SKILL.md"), "cl")
	writeFile(t, filepath.Join(h, ".claude", "skills", "synced", "abc", "x", "SKILL.md"), "cl")
	writeFile(t, filepath.Join(h, ".codex", "skills", "codex-only", "SKILL.md"), "cx")
	assets := fstest.MapFS{"skills/hcd/SKILL.md": {Data: []byte("embebida")}}

	repo := SkillRepoDir(h)
	for i := 0; i < 2; i++ {
		res, err := SyncSkillRepo(assets, h)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 && len(res.Collisions) == 0 {
			t.Fatal("se esperaban colisiones resueltas")
		}
	}
	if got := readFile(t, filepath.Join(repo, "hcd", "SKILL.md")); got != "embebida\n" {
		t.Fatalf("hcd debe ser la embebida: %q", got)
	}
	if fi, _ := os.Lstat(filepath.Join(repo, "hcd")); fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("la embebida debe ser una copia real")
	}
	want := map[string]string{
		"humanizer":  filepath.Join(h, ".agents", "skills", "humanizer"),
		"semantica":  filepath.Join(h, ".claude", "skills", "semantica"),
		"codex-only": filepath.Join(h, ".codex", "skills", "codex-only"),
	}
	for n, target := range want {
		if got, err := os.Readlink(filepath.Join(repo, n)); err != nil || got != target {
			t.Errorf("%s → %q %v, want %s", n, got, err, target)
		}
	}
	if _, err := os.Lstat(filepath.Join(repo, "synced")); err == nil {
		t.Error("una carpeta sin SKILL.md directo no es una skill")
	}

	// Una skill que desaparece del anfitrión se retira del repositorio.
	if err := os.RemoveAll(filepath.Join(h, ".codex", "skills", "codex-only")); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncSkillRepo(assets, h); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(repo, "codex-only")); err == nil {
		t.Error("el enlace roto debía retirarse")
	}
	if readFile(t, filepath.Join(h, ".claude", "skills", "semantica", "SKILL.md")) != "cl" {
		t.Fatal("se modificó una skill del anfitrión")
	}
}

func TestSkillRepoKeepsUserContent(t *testing.T) {
	opts := sandbox(t)
	repo := SkillRepoDir(opts.HomeDir)
	writeFile(t, filepath.Join(repo, "mia", "SKILL.md"), "propia")
	writeFile(t, filepath.Join(opts.HomeDir, ".claude", "skills", "mia", "SKILL.md"), "cl")
	if _, err := SyncSkillRepo(fstest.MapFS{"skills/hcd/SKILL.md": {Data: []byte("e")}}, opts.HomeDir); err != nil {
		t.Fatal(err)
	}
	if readFile(t, filepath.Join(repo, "mia", "SKILL.md")) != "propia" {
		t.Fatal("se pisó una skill propia del repositorio")
	}
}

func TestSkillRepoPiSettings(t *testing.T) {
	opts := sandbox(t)
	piSkills := filepath.Join(opts.PiAgentDir, "skills")
	// Estado heredado de HCD-002/003: copia embebida intacta, enlace a Claude y skill propia.
	writeFile(t, filepath.Join(piSkills, "hcd", "SKILL.md"), "x")
	writeFile(t, filepath.Join(piSkills, "hcd", skillMarkerFile), contentHash([]byte("x")))
	writeFile(t, filepath.Join(opts.HomeDir, ".claude", "skills", "semantica", "SKILL.md"), "cl")
	if err := os.Symlink(filepath.Join(opts.HomeDir, ".claude", "skills", "semantica"), filepath.Join(piSkills, "semantica")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(piSkills, "mia", "SKILL.md"), "propia")
	writeFile(t, filepath.Join(opts.PiAgentDir, "settings.json"), `{"theme":"t","skills":["~/.config/opencode/skills","~/.claude/skills","/mis/skills"]}`)

	if _, err := SyncSkillRepo(fstest.MapFS{"skills/hcd/SKILL.md": {Data: []byte("e")}}, opts.HomeDir); err != nil {
		t.Fatal(err)
	}
	if err := RegisterSkillRepo(opts.PiAgentDir, opts.HomeDir); err != nil {
		t.Fatal(err)
	}
	got := readSettingsSkills(t, opts.PiAgentDir)
	if strings.Join(got, ",") != "/mis/skills,"+SkillRepoSetting {
		t.Fatalf("skills = %v", got)
	}
	if !strings.Contains(readFile(t, filepath.Join(opts.PiAgentDir, "settings.json")), `"theme": "t"`) {
		t.Fatal("se perdió una clave del usuario")
	}
	for _, n := range []string{"hcd", "semantica"} {
		if _, err := os.Lstat(filepath.Join(piSkills, n)); err == nil {
			t.Errorf("%s debía retirarse de ~/.pi/agent/skills", n)
		}
	}
	if readFile(t, filepath.Join(piSkills, "mia", "SKILL.md")) != "propia" {
		t.Fatal("se borró la skill propia de ~/.pi/agent/skills")
	}

	// Carpeta del anfitrión que es un enlace: el enlace heredado apunta a la ruta real.
	real := filepath.Join(t.TempDir(), "codex-real")
	writeFile(t, filepath.Join(real, "cx", "SKILL.md"), "cx")
	if err := os.Symlink(real, filepath.Join(opts.HomeDir, ".codex", "skills")); err != nil {
		if err := os.MkdirAll(filepath.Join(opts.HomeDir, ".codex"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, filepath.Join(opts.HomeDir, ".codex", "skills")); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(real, "cx"), filepath.Join(piSkills, "cx")); err != nil {
		t.Fatal(err)
	}
	if err := RegisterSkillRepo(opts.PiAgentDir, opts.HomeDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(piSkills, "cx")); err == nil {
		t.Error("el enlace heredado a la ruta real del anfitrión debía retirarse")
	}
}

func TestSkillRepoPiSettingsLegacyWithoutMarker(t *testing.T) {
	opts := sandbox(t)
	piSkills := filepath.Join(opts.PiAgentDir, "skills")
	// Copia embebida antigua sin marker pero con contenido idéntico al repo central
	writeFile(t, filepath.Join(piSkills, "hcd", "SKILL.md"), "contenido-identico")
	if _, err := SyncSkillRepo(fstest.MapFS{"skills/hcd/SKILL.md": {Data: []byte("contenido-identico")}}, opts.HomeDir); err != nil {
		t.Fatal(err)
	}
	if err := RegisterSkillRepo(opts.PiAgentDir, opts.HomeDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(piSkills, "hcd")); err == nil {
		t.Fatal("hcd sin marker pero idéntico debía retirarse de ~/.pi/agent/skills")
	}
}

func TestHarnessSyncsSkills(t *testing.T) {
	s := GenerateHarnessScript()
	if !strings.Contains(s, "pi-gomemory-setup skills-sync") {
		t.Fatal("el launcher debe sincronizar el repositorio de skills")
	}
	if strings.Index(s, "skills-sync") > strings.Index(s, "exec pi ") {
		t.Fatal("la sincronización debe ocurrir antes de lanzar pi")
	}
}

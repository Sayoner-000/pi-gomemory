package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func readSettingsSkills(t *testing.T, piAgentDir string) []string {
	t.Helper()
	var s struct {
		Skills []string `json:"skills"`
	}
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(piAgentDir, "settings.json"))), &s); err != nil {
		t.Fatal(err)
	}
	return s.Skills
}

func TestRegisterHostSkills(t *testing.T) {
	opts := sandbox(t)
	writeFile(t, filepath.Join(opts.HomeDir, ".config", "opencode", "skills", "impeccable", "SKILL.md"), "x")
	writeFile(t, filepath.Join(opts.PiAgentDir, "settings.json"), `{"theme":"gomemory","skills":["/mis/skills"]}`)

	if err := RegisterHostSkills(opts.PiAgentDir, opts.HomeDir); err != nil {
		t.Fatal(err)
	}
	got := readSettingsSkills(t, opts.PiAgentDir)
	want := []string{"/mis/skills", "~/.config/opencode/skills"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("skills = %v, want %v (C-06, C-07: ~/.claude/skills no existe)", got, want)
	}
	if !strings.Contains(readFile(t, filepath.Join(opts.PiAgentDir, "settings.json")), `"theme": "gomemory"`) {
		t.Fatal("se perdió una clave del usuario")
	}
}

func TestInstallAllHostSkillsIdempotent(t *testing.T) {
	opts := sandbox(t)
	assets := repoAssets(t)
	t.Chdir(t.TempDir())
	host := filepath.Join(opts.HomeDir, ".claude", "skills", "gitlab-release", "SKILL.md")
	writeFile(t, host, "original")
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	os.Chtimes(host, old, old)

	for i := 0; i < 2; i++ {
		if err := InstallAll(assets, opts); err != nil {
			t.Fatal(err)
		}
	}
	if got := readSettingsSkills(t, opts.PiAgentDir); len(got) != 1 || got[0] != "~/.claude/skills" {
		t.Fatalf("skills = %v, se espera una sola entrada (C-08)", got)
	}
	info, _ := os.Stat(host)
	if readFile(t, host) != "original" || !info.ModTime().Equal(old) {
		t.Fatal("se modificó una skill del anfitrión (C-08)")
	}
}

// Las skills repetidas entre OpenCode y Claude no deben llegar dos veces a Pi.
func TestRegisterHostSkillsNoCollisions(t *testing.T) {
	opts := sandbox(t)
	oc := filepath.Join(opts.HomeDir, ".config", "opencode", "skills")
	cl := filepath.Join(opts.HomeDir, ".claude", "skills")
	for _, n := range []string{"hcd", "verify-loop"} {
		writeFile(t, filepath.Join(oc, n, "SKILL.md"), "x")
		writeFile(t, filepath.Join(cl, n, "SKILL.md"), "x")
	}
	writeFile(t, filepath.Join(cl, "semantica", "SKILL.md"), "x")
	writeFile(t, filepath.Join(cl, "synced", "abc", "semantica", "SKILL.md"), "x")
	writeFile(t, filepath.Join(opts.HomeDir, ".agents", "skills", "humanizer", "SKILL.md"), "x")
	writeFile(t, filepath.Join(cl, "humanizer", "SKILL.md"), "x")
	// Copia embebida previa que ya ofrece OpenCode, y una entrada de Claude heredada.
	writeFile(t, filepath.Join(opts.PiAgentDir, "skills", "hcd", "SKILL.md"), "x")
	writeFile(t, filepath.Join(opts.PiAgentDir, "settings.json"), `{"skills":["~/.config/opencode/skills","~/.claude/skills"]}`)

	for i := 0; i < 2; i++ {
		if err := RegisterHostSkills(opts.PiAgentDir, opts.HomeDir); err != nil {
			t.Fatal(err)
		}
	}
	if got := readSettingsSkills(t, opts.PiAgentDir); strings.Join(got, ",") != "~/.config/opencode/skills" {
		t.Fatalf("skills = %v, se espera solo OpenCode", got)
	}
	piSkills := filepath.Join(opts.PiAgentDir, "skills")
	if target, err := os.Readlink(filepath.Join(piSkills, "semantica")); err != nil || target != filepath.Join(cl, "semantica") {
		t.Fatalf("semantica debe enlazarse desde Claude: %q %v", target, err)
	}
	for _, n := range []string{"hcd", "verify-loop", "humanizer", "synced"} {
		if _, err := os.Lstat(filepath.Join(piSkills, n)); err == nil {
			t.Errorf("%s no debe estar en ~/.pi/agent/skills: ya lo ofrece otra carpeta", n)
		}
	}
	if readFile(t, filepath.Join(cl, "hcd", "SKILL.md")) != "x" {
		t.Fatal("se modificó una skill del anfitrión")
	}
}

func TestDoctorHostSkills(t *testing.T) {
	opts := sandbox(t)
	for _, n := range []string{"impeccable", "gitlab-release"} {
		writeFile(t, filepath.Join(opts.HomeDir, ".config", "opencode", "skills", n, "SKILL.md"), "x")
	}
	if err := RegisterHostSkills(opts.PiAgentDir, opts.HomeDir); err != nil {
		t.Fatal(err)
	}
	var host, embedded bool
	for _, it := range RunDoctor(opts) {
		if strings.Contains(it.Name, "Skills del anfitrión") && strings.Contains(it.Details, "~/.config/opencode/skills") && strings.Contains(it.Details, "2 skills") {
			host = true
		}
		if it.Name == "Skills gomemory en Pi" {
			embedded = true
		}
	}
	if !host || !embedded {
		t.Fatalf("doctor: host=%v embedded=%v (C-09)", host, embedded)
	}
}

func TestHarnessExportsRunningVersion(t *testing.T) {
	s := GenerateHarnessScript()
	if !strings.Contains(s, `export PI_GOMEMORY_RUNNING_VERSION="$(pi-gomemory-setup version`) {
		t.Fatal("el launcher debe exportar PI_GOMEMORY_RUNNING_VERSION")
	}
	if strings.Contains(s, "export PI_GOMEMORY_VERSION") {
		t.Fatal("no debe exportar PI_GOMEMORY_VERSION (la usa install.sh)")
	}
	if strings.Index(s, "PI_GOMEMORY_RUNNING_VERSION") > strings.Index(s, "exec pi ") {
		t.Fatal("la variable debe exportarse antes de lanzar pi")
	}
}

func readAsset(t *testing.T, name string) string {
	t.Helper()
	return readFile(t, filepath.Join("..", "..", name))
}

func TestBannerExtensionEmbedded(t *testing.T) {
	s := readAsset(t, "extensions/gomemory-banner.ts")
	for _, want := range []string{"setHeader", "session_start", "pi - gomemory", "PI_GOMEMORY_RUNNING_VERSION", "NO_COLOR", `"dev"`} {
		if !strings.Contains(s, want) {
			t.Errorf("gomemory-banner.ts no contiene %q", want)
		}
	}
	opts := sandbox(t)
	if err := InstallTUIExtension(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	if readFile(t, filepath.Join(opts.PiAgentDir, "extensions", "gomemory-banner.ts")) != s {
		t.Fatal("la extensión del banner no se instala tal cual")
	}
}

type planPatterns struct {
	Tools []string `json:"tools"`
	Strip []string `json:"strip"`
	Block []string `json:"block"`
}

func planBlocks(p planPatterns, cmd string) bool {
	for _, s := range p.Strip {
		cmd = regexp.MustCompile(s).ReplaceAllString(cmd, " ")
	}
	for _, b := range p.Block {
		if regexp.MustCompile(b).MatchString(cmd) {
			return true
		}
	}
	return false
}

func TestPlanPatterns(t *testing.T) {
	var p planPatterns
	if err := json.Unmarshal([]byte(readAsset(t, "extensions/gomemory-plan-patterns.json")), &p); err != nil {
		t.Fatal(err)
	}
	if strings.Join(p.Tools, ",") != "edit,write" {
		t.Errorf("tools = %v", p.Tools)
	}
	block := []string{"rm x", "mv a b", "cp a b", "mkdir d", "touch f", "chmod +x f", "ln -s a b", "echo x | tee f",
		"echo x > f", "echo x >> f", "sed -i s/a/b/ f", "git commit -m x", "git push", "git checkout -b x", "git add .",
		"npm install", "pnpm add x", "go mod tidy", "cd a && rm x", "sudo rm -rf x", "ls; mv a b"}
	allow := []string{"ls -la", "git status", "git diff", "scp a b", "cat f", "grep -r x .", "echo x > /dev/null",
		"go test ./... 2>&1", "echo cp", "git log --oneline", "sed s/a/b/ f", "mem plan-context"}
	for _, c := range block {
		if !planBlocks(p, c) {
			t.Errorf("debería bloquear %q", c)
		}
	}
	for _, c := range allow {
		if planBlocks(p, c) {
			t.Errorf("no debería bloquear %q", c)
		}
	}
}

func TestPlanExtensionEmbedded(t *testing.T) {
	s := readAsset(t, "extensions/gomemory-plan.ts")
	for _, want := range []string{`registerCommand("plan"`, `"tool_call"`, "block: true", "gomemory-plan-patterns.json", `"run"`, `"PLAN"`, "setStatus"} {
		if !strings.Contains(s, want) {
			t.Errorf("gomemory-plan.ts no contiene %q", want)
		}
	}
	opts := sandbox(t)
	if err := InstallTUIExtension(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"gomemory-plan.ts", "gomemory-plan-patterns.json"} {
		if _, err := os.Stat(filepath.Join(opts.PiAgentDir, "extensions", f)); err != nil {
			t.Errorf("%s no se instala", f)
		}
	}
}

func TestPlanExtensionFallbackMarker(t *testing.T) {
	s := readAsset(t, "extensions/gomemory-plan.ts")
	for _, want := range []string{"before_agent_start", `"plan-context"`, "timeout: 5000", "[gomemory-plan:fallback]", ".trim()"} {
		if !strings.Contains(s, want) {
			t.Errorf("gomemory-plan.ts no contiene %q", want)
		}
	}
}

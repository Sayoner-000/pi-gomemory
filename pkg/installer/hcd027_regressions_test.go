package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Each subtest asserts the corrected behavior against a synthetic reproduction.
// Missing executables and setup errors must not be mistaken for reproduced defects.
func TestHCD027Reproductions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		run  func(*testing.T)
	}{
		{"C001_marked_skill_preserves_added_file", hcd027ReproMarkedSkill},
		{"C002_mcp_preserves_private_mode", hcd027ReproMCPMode},
		// C003–C006 reproducían defectos de /tdd, retirado en HCD-085.
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func hcd027ReproMarkedSkill(t *testing.T) {
	home := t.TempDir()
	piDir := filepath.Join(home, ".pi", "agent")
	legacy := filepath.Join(piDir, "skills", "sample")
	content := []byte("# Synthetic skill\n")
	writeFile(t, filepath.Join(legacy, "SKILL.md"), string(content))
	writeFile(t, filepath.Join(legacy, skillMarkerFile), contentHash(content)+"\n")
	extra := filepath.Join(legacy, "scripts", "mine.sh")
	want := []byte("echo synthetic-user-content\n")
	writeFile(t, extra, string(want))
	if err := RegisterSkillRepo(piDir, home); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(extra)
	if err != nil {
		t.Fatalf("C001: user-added script was deleted: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("C001: user-added script was changed: %q", got)
	}
}

func hcd027ReproMCPMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	const config = `{"mcpServers":{"foreign":{"command":"synthetic","env":{"API_KEY":"SYNTH-TOKEN-0001"}}}}`
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if err := InstallMCPServers(dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("C002: private MCP permissions widened: got %04o, want 0600", got)
	}
}

func TestHCD027SkillCleanupCompatibility(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	piDir := filepath.Join(home, ".pi", "agent")
	skillsDir := filepath.Join(piDir, "skills")
	repoDir := filepath.Join(home, ".pi-gomemory", "skills")

	// 1. Managed pristine skill with only SKILL.md and .pi-gomemory
	pristineSkill := filepath.Join(skillsDir, "managed_pristine")
	pristineContent := []byte("# Pristine Skill\n")
	writeFile(t, filepath.Join(pristineSkill, "SKILL.md"), string(pristineContent))
	writeFile(t, filepath.Join(pristineSkill, skillMarkerFile), contentHash(pristineContent)+"\n")

	// 2. Managed unmarked skill matching central repo SKILL.md with only SKILL.md
	repoSkill := filepath.Join(repoDir, "managed_unmarked")
	repoContent := []byte("# Repo Skill\n")
	writeFile(t, filepath.Join(repoSkill, "SKILL.md"), string(repoContent))
	unmarkedSkill := filepath.Join(skillsDir, "managed_unmarked")
	writeFile(t, filepath.Join(unmarkedSkill, "SKILL.md"), string(repoContent))

	// 3. User custom skill (not in repo, not marked)
	customSkill := filepath.Join(skillsDir, "user_custom")
	customContent := []byte("# User Custom Skill\n")
	writeFile(t, filepath.Join(customSkill, "SKILL.md"), string(customContent))
	writeFile(t, filepath.Join(customSkill, "custom_file.txt"), "user payload")

	// 4. Host skill symlink pointing to an external directory
	hostTargetDir := filepath.Join(home, ".claude", "skills", "host_skill")
	hostContent := []byte("# Host Skill\n")
	writeFile(t, filepath.Join(hostTargetDir, "SKILL.md"), string(hostContent))
	writeFile(t, filepath.Join(hostTargetDir, "tool.sh"), "#!/bin/sh\n")
	symlinkPath := filepath.Join(skillsDir, "host_link")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(hostTargetDir, symlinkPath); err != nil {
		t.Fatal(err)
	}

	// 5. User custom folder in repoDir (not symlink)
	userRepoDir := filepath.Join(repoDir, "my_custom_repo_skill")
	writeFile(t, filepath.Join(userRepoDir, "SKILL.md"), "# Repo Custom\n")

	// Run RegisterSkillRepo
	if err := RegisterSkillRepo(piDir, home); err != nil {
		t.Fatal(err)
	}

	// 1: Pristine managed copy must be pruned
	if _, err := os.Stat(pristineSkill); !os.IsNotExist(err) {
		t.Errorf("managed pristine skill was not cleaned up: %v", err)
	}

	// 2: Unmarked matching legacy copy must be pruned
	if _, err := os.Stat(unmarkedSkill); !os.IsNotExist(err) {
		t.Errorf("unmarked legacy skill was not cleaned up: %v", err)
	}

	// 3: User custom skill must NOT be deleted
	if _, err := os.Stat(customSkill); err != nil {
		t.Errorf("user custom skill was deleted: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(customSkill, "custom_file.txt")); err != nil || string(got) != "user payload" {
		t.Errorf("user custom file was lost: %s, %v", string(got), err)
	}

	// 4: Host target directory MUST NOT be deleted (only legacy symlink is pruned from piDir)
	if _, err := os.Stat(hostTargetDir); err != nil {
		t.Errorf("host skill target directory was deleted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(hostTargetDir, "tool.sh")); err != nil {
		t.Errorf("host skill file was deleted: %v", err)
	}

	// 5: User folder in repoDir must NOT be deleted
	if _, err := os.Stat(userRepoDir); err != nil {
		t.Errorf("user repo directory was deleted: %v", err)
	}
}

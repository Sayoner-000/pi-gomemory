package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestHCD027SkillExtras(t *testing.T) {
	home := t.TempDir()
	piDir := filepath.Join(home, ".pi", "agent")
	skillsDir := filepath.Join(piDir, "skills")

	// Case 1: marked skill with added script
	skill1 := filepath.Join(skillsDir, "skill1")
	skill1Content := []byte("# Skill One\n")
	writeFile(t, filepath.Join(skill1, "SKILL.md"), string(skill1Content))
	writeFile(t, filepath.Join(skill1, skillMarkerFile), contentHash(skill1Content)+"\n")
	scriptPath := filepath.Join(skill1, "scripts", "run.sh")
	scriptBytes := []byte("#!/bin/sh\necho test-script\n")
	writeFile(t, scriptPath, string(scriptBytes))

	// Case 2: marked skill with added note
	skill2 := filepath.Join(skillsDir, "skill2")
	skill2Content := []byte("# Skill Two\n")
	writeFile(t, filepath.Join(skill2, "SKILL.md"), string(skill2Content))
	writeFile(t, filepath.Join(skill2, skillMarkerFile), contentHash(skill2Content)+"\n")
	notePath := filepath.Join(skill2, "notes.txt")
	noteBytes := []byte("important personal notes\n")
	writeFile(t, notePath, string(noteBytes))

	// Case 3: marked skill with added nested subdirectory
	skill3 := filepath.Join(skillsDir, "skill3")
	skill3Content := []byte("# Skill Three\n")
	writeFile(t, filepath.Join(skill3, "SKILL.md"), string(skill3Content))
	writeFile(t, filepath.Join(skill3, skillMarkerFile), contentHash(skill3Content)+"\n")
	subPath := filepath.Join(skill3, "nested", "sub", "config.json")
	subBytes := []byte("{\"preserved\": true}\n")
	writeFile(t, subPath, string(subBytes))

	if err := RegisterSkillRepo(piDir, home); err != nil {
		t.Fatal(err)
	}

	// Verify all extra files and their bytes were preserved
	for _, tc := range []struct {
		name string
		path string
		want []byte
	}{
		{"script", scriptPath, scriptBytes},
		{"note", notePath, noteBytes},
		{"nested_file", subPath, subBytes},
	} {
		got, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatalf("extra %s was removed: %v", tc.name, err)
		}
		if !bytes.Equal(got, tc.want) {
			t.Fatalf("extra %s bytes altered: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

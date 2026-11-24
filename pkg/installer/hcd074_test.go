package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHCD074SkillsSinCR: las skills embebidas usan solo LF, como sus fuentes
// en gomemory (HCD-074).
func TestHCD074SkillsSinCR(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "skills", "*", "SKILL.md"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no se encontraron skills embebidas: %v", err)
	}
	var bad []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.IndexByte(b, '\r') >= 0 {
			bad = append(bad, filepath.Base(filepath.Dir(f)))
		}
	}
	if len(bad) > 0 {
		t.Fatalf("skills embebidas con CR: %s", strings.Join(bad, ", "))
	}
}

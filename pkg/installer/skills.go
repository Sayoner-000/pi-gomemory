package installer

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SanitizeFrontmatter asegura que los campos de YAML no contengan mappings anidados inválidos.
func SanitizeFrontmatter(content []byte) []byte {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var buf bytes.Buffer
	inFrontmatter := false
	frontmatterDashes := 0

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			frontmatterDashes++
			if frontmatterDashes == 1 {
				inFrontmatter = true
			} else if frontmatterDashes == 2 {
				inFrontmatter = false
			}
			buf.WriteString(line + "\n")
			continue
		}

		if inFrontmatter && strings.HasPrefix(trimmed, "description:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			if !strings.HasPrefix(val, `"`) && !strings.HasPrefix(val, `'`) && strings.Contains(val, ":") {
				cleanVal := strings.ReplaceAll(val, `"`, `\"`)
				line = fmt.Sprintf(`description: "%s"`, cleanVal)
			}
		}

		buf.WriteString(line + "\n")
	}

	return buf.Bytes()
}

// InstallSkills copia y sanitiza las skills embebidas en el binario a
// ~/.pi/agent/skills/. Solo usa el embebido: no copia ni modifica skills de otros
// agentes (RegisterHostSkills solo registra sus carpetas para que Pi las lea).
func InstallSkills(assets fs.FS, piAgentDir string) (int, error) {
	entries, err := fs.ReadDir(assets, "skills")
	if err != nil {
		return 0, fmt.Errorf("leer skills embebidas: %w", err)
	}

	targetSkillsDir := filepath.Join(piAgentDir, "skills")
	if err := os.MkdirAll(targetSkillsDir, 0755); err != nil {
		return 0, fmt.Errorf("crear dir de skills en pi: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()
		rawContent, err := fs.ReadFile(assets, "skills/"+skillName+"/SKILL.md")
		if err != nil {
			return count, fmt.Errorf("leer skill embebida %s: %w", skillName, err)
		}

		dstSkillDir := filepath.Join(targetSkillsDir, skillName)
		if err := os.MkdirAll(dstSkillDir, 0755); err != nil {
			return count, fmt.Errorf("crear dir para skill %s: %w", skillName, err)
		}
		dstSkillFile := filepath.Join(dstSkillDir, "SKILL.md")
		if err := os.WriteFile(dstSkillFile, SanitizeFrontmatter(rawContent), 0644); err != nil {
			return count, fmt.Errorf("escribir SKILL.md de %s: %w", skillName, err)
		}
		count++
	}

	if count == 0 {
		return 0, fmt.Errorf("el binario no contiene skills embebidas")
	}
	return count, nil
}

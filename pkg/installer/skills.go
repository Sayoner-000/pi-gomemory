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

// InstallSkills copia y sanitiza todas las skills de gomemory a ~/.pi/agent/skills/
// Si embedFS está disponible, lee directamente del binario embebido.
func InstallSkills(embedFS fs.FS, packageRootDir, piAgentDir string) (int, error) {
	targetSkillsDir := filepath.Join(piAgentDir, "skills")
	if err := os.MkdirAll(targetSkillsDir, 0755); err != nil {
		return 0, fmt.Errorf("crear dir de skills en pi: %w", err)
	}

	// 1. Intentar desde FS embebido en el binario
	if embedFS != nil {
		entries, err := fs.ReadDir(embedFS, "skills")
		if err == nil && len(entries) > 0 {
			count := 0
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				skillName := entry.Name()
				rawContent, err := fs.ReadFile(embedFS, "skills/"+skillName+"/SKILL.md")
				if err != nil {
					continue
				}

				sanitized := SanitizeFrontmatter(rawContent)
				dstSkillDir := filepath.Join(targetSkillsDir, skillName)
				if err := os.MkdirAll(dstSkillDir, 0755); err != nil {
					continue
				}

				dstSkillFile := filepath.Join(dstSkillDir, "SKILL.md")
				if err := os.WriteFile(dstSkillFile, sanitized, 0644); err == nil {
					count++
				}
			}
			if count > 0 {
				return count, nil
			}
		}
	}

	// 2. Fallback a filesystem en disco
	sourceDir := filepath.Join(packageRootDir, "skills")
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		home, _ := os.UserHomeDir()
		sourceDir = filepath.Join(home, ".config", "opencode", "skills")
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return 0, fmt.Errorf("leer directorio origen de skills %s: %w", sourceDir, err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()
		srcSkillFile := filepath.Join(sourceDir, skillName, "SKILL.md")
		rawContent, err := os.ReadFile(srcSkillFile)
		if err != nil {
			continue
		}

		sanitized := SanitizeFrontmatter(rawContent)
		_ = os.WriteFile(srcSkillFile, sanitized, 0644)

		dstSkillDir := filepath.Join(targetSkillsDir, skillName)
		if err := os.MkdirAll(dstSkillDir, 0755); err != nil {
			return count, fmt.Errorf("crear dir para skill %s: %w", skillName, err)
		}

		dstSkillFile := filepath.Join(dstSkillDir, "SKILL.md")
		if err := os.WriteFile(dstSkillFile, sanitized, 0644); err != nil {
			return count, fmt.Errorf("escribir SKILL.md sanitizado de %s: %w", skillName, err)
		}
		count++
	}

	return count, nil
}

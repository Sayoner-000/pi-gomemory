package installer

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// skillMarkerFile marca una carpeta de skill instalada por pi-gomemory y guarda
// el SHA256 del SKILL.md escrito, para no borrar nunca una skill ajena o editada.
const skillMarkerFile = ".pi-gomemory"

func contentHash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// isPristineEmbeddedCopy indica si dir es una skill instalada por pi-gomemory
// que la persona no ha modificado.
func isPristineEmbeddedCopy(dir string) bool {
	marker, err := os.ReadFile(filepath.Join(dir, skillMarkerFile))
	if err != nil {
		return false
	}
	content, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(marker)) == contentHash(content)
}

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

// installEmbeddedSkills copia y sanitiza las skills embebidas en dir, cada una
// con su marca. Sustituye enlaces y copias embebidas intactas; nunca pisa una
// carpeta propia de la persona ni una copia embebida que haya editado.
func installEmbeddedSkills(assets fs.FS, dir string) ([]string, error) {
	entries, err := fs.ReadDir(assets, "skills")
	if err != nil {
		return nil, fmt.Errorf("leer skills embebidas: %w", err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("crear %s: %w", dir, err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()
		rawContent, err := fs.ReadFile(assets, "skills/"+skillName+"/SKILL.md")
		if err != nil {
			return names, fmt.Errorf("leer skill embebida %s: %w", skillName, err)
		}
		names = append(names, skillName)

		dstSkillDir := filepath.Join(dir, skillName)
		if fi, err := os.Lstat(dstSkillDir); err == nil {
			switch {
			case fi.Mode()&os.ModeSymlink != 0:
				if err := os.Remove(dstSkillDir); err != nil {
					return names, err
				}
			case !isPristineEmbeddedCopy(dstSkillDir):
				continue
			}
		}
		if err := os.MkdirAll(dstSkillDir, 0755); err != nil {
			return names, fmt.Errorf("crear dir para skill %s: %w", skillName, err)
		}
		content := SanitizeFrontmatter(rawContent)
		if err := os.WriteFile(filepath.Join(dstSkillDir, "SKILL.md"), content, 0644); err != nil {
			return names, fmt.Errorf("escribir SKILL.md de %s: %w", skillName, err)
		}
		if err := os.WriteFile(filepath.Join(dstSkillDir, skillMarkerFile), []byte(contentHash(content)+"\n"), 0644); err != nil {
			return names, fmt.Errorf("escribir marca de %s: %w", skillName, err)
		}
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("el binario no contiene skills embebidas")
	}
	return names, nil
}

// InstallSkills reúne las skills embebidas y las del anfitrión en el
// repositorio central (~/.pi-gomemory/skills) y hace que Pi lo cargue.
func InstallSkills(assets fs.FS, opts *Options) (SkillRepoResult, error) {
	res, err := SyncSkillRepo(assets, opts.HomeDir)
	if err != nil {
		return res, err
	}
	return res, RegisterSkillRepo(opts.PiAgentDir, opts.HomeDir)
}

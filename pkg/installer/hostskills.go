package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hostSkillDirs son las carpetas de skills de otros agentes que Pi lee en su sitio.
// ~/.agents/skills no se registra: Pi la descubre por sí mismo.
var hostSkillDirs = []string{"~/.config/opencode/skills", "~/.claude/skills"}

func expandHome(p, home string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// RegisterHostSkills hace que Pi lea las skills de otros agentes sin colisiones.
// La primera carpeta del anfitrión que exista se registra en el array `skills` del
// settings.json. De las siguientes solo se enlazan, en ~/.pi/agent/skills, las
// skills que no ofrezca ya otra carpeta, y su entrada sale de settings.json: si se
// registraran enteras, Pi avisaría de una colisión por cada skill repetida. Por
// último se retiran de ~/.pi/agent/skills las copias embebidas que el anfitrión
// ya ofrece. No copia ni modifica nada dentro de las carpetas del anfitrión.
func RegisterHostSkills(piAgentDir, home string) error {
	path := filepath.Join(piAgentDir, "settings.json")
	obj, err := readJSONObject(path)
	if err != nil {
		return err
	}
	var skills []string
	if raw, ok := obj["skills"]; ok {
		if err := json.Unmarshal(raw, &skills); err != nil {
			return fmt.Errorf("%s: `skills` no es una lista de rutas; no se modifica: %w", path, err)
		}
	}
	registered := func(abs string) int {
		for i, s := range skills {
			if filepath.Clean(expandHome(s, home)) == filepath.Clean(abs) {
				return i
			}
		}
		return -1
	}

	linksDir := filepath.Join(piAgentDir, "skills")
	provided := map[string]bool{} // skills que Pi ya recibe de una carpeta del anfitrión
	for _, n := range skillNames(filepath.Join(home, ".agents", "skills")) {
		provided[n] = true
	}
	changed, primary := false, false
	for _, dir := range hostSkillDirs {
		abs := expandHome(dir, home)
		if info, err := os.Stat(abs); err != nil || !info.IsDir() {
			continue
		}
		names := skillNames(abs)
		if !primary {
			primary = true
			if registered(abs) < 0 {
				skills = append(skills, dir)
				changed = true
			}
			for _, n := range names {
				provided[n] = true
			}
			continue
		}
		if i := registered(abs); i >= 0 {
			skills = append(skills[:i], skills[i+1:]...)
			changed = true
		}
		for _, n := range names {
			if provided[n] {
				continue
			}
			if err := linkHostSkill(filepath.Join(abs, n), filepath.Join(linksDir, n)); err != nil {
				return err
			}
			provided[n] = true
		}
	}
	if err := pruneProvidedCopies(linksDir, provided); err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if err := setJSONField(obj, "skills", skills); err != nil {
		return err
	}
	if err := os.MkdirAll(piAgentDir, 0755); err != nil {
		return err
	}
	return writeJSONObject(path, obj)
}

// skillNames devuelve las subcarpetas directas de dir (o enlaces a carpetas) que
// contienen SKILL.md.
func skillNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "SKILL.md")); err == nil {
			names = append(names, e.Name())
		}
	}
	return names
}

// linkHostSkill enlaza una skill del anfitrión en Pi. Si ya hay algo con ese
// nombre, lo deja: puede ser una skill embebida o una del usuario.
func linkHostSkill(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.Symlink(src, dst)
}

// pruneProvidedCopies borra de ~/.pi/agent/skills las carpetas reales (no los
// enlaces) cuya skill ya ofrece una carpeta del anfitrión.
func pruneProvidedCopies(linksDir string, provided map[string]bool) error {
	entries, err := os.ReadDir(linksDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !provided[e.Name()] || e.Type()&os.ModeSymlink != 0 || !e.IsDir() {
			continue
		}
		if err := os.RemoveAll(filepath.Join(linksDir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// AvailableSkillCount cuenta las skills distintas que Pi ve: las de
// ~/.pi/agent/skills, las de ~/.agents/skills y las de las carpetas del anfitrión
// registradas en settings.json.
func AvailableSkillCount(piAgentDir, home string) int {
	seen := map[string]bool{}
	for _, n := range skillNames(filepath.Join(piAgentDir, "skills")) {
		seen[n] = true
	}
	for _, n := range skillNames(filepath.Join(home, ".agents", "skills")) {
		seen[n] = true
	}
	for dir := range HostSkillCounts(piAgentDir, home) {
		for _, n := range skillNames(expandHome(dir, home)) {
			seen[n] = true
		}
	}
	return len(seen)
}

// HostSkillCounts devuelve, para cada carpeta del anfitrión registrada en settings.json,
// cuántas subcarpetas directas contienen SKILL.md. Solo lee.
func HostSkillCounts(piAgentDir, home string) map[string]int {
	counts := map[string]int{}
	obj, err := readJSONObject(filepath.Join(piAgentDir, "settings.json"))
	if err != nil {
		return counts
	}
	var skills []string
	if raw, ok := obj["skills"]; ok {
		_ = json.Unmarshal(raw, &skills)
	}
	registered := map[string]bool{}
	for _, s := range skills {
		registered[s] = true
	}
	for _, dir := range hostSkillDirs {
		if !registered[dir] {
			continue
		}
		entries, err := os.ReadDir(expandHome(dir, home))
		if err != nil {
			continue
		}
		n := 0
		for _, e := range entries {
			if _, err := os.Stat(filepath.Join(expandHome(dir, home), e.Name(), "SKILL.md")); err == nil {
				n++
			}
		}
		counts[dir] = n
	}
	return counts
}

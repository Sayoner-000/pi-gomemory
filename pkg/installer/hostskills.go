package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SkillRepoSetting es la ruta del repositorio central tal como se registra en
// el array `skills` de ~/.pi/agent/settings.json.
const SkillRepoSetting = "~/.pi-gomemory/skills"

// hostSkillSources son las carpetas de skills de los agentes del anfitrión, en
// orden de prioridad: ante un mismo nombre gana la primera. Las skills
// embebidas de pi-gomemory ganan a todas.
var hostSkillSources = []string{
	"~/.agents/skills",
	"~/.config/opencode/skills",
	"~/.claude/skills",
	"~/.codex/skills",
	"~/.gemini/skills",
}

// legacyHostSettings son las rutas que HCD-002 registraba en settings.json;
// ahora llegan a Pi a través del repositorio central.
var legacyHostSettings = []string{"~/.config/opencode/skills", "~/.claude/skills"}

// SkillRepoDir devuelve la ruta absoluta del repositorio central de skills.
func SkillRepoDir(home string) string {
	return expandHome(SkillRepoSetting, home)
}

func expandHome(p, home string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// SkillCollision registra una skill que existe en varias fuentes.
type SkillCollision struct {
	Name   string
	Winner string
	Loser  string
}

// SkillRepoResult resume una sincronización del repositorio.
type SkillRepoResult struct {
	Embedded   int
	Linked     int
	Collisions []SkillCollision
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

// planHostLinks decide qué skill del anfitrión entra en el repositorio. owner
// indica, por nombre, quién ocupa ya ese nombre (embebida o carpeta propia).
// Solo lee.
func planHostLinks(home string, owner map[string]string) (map[string]string, []SkillCollision) {
	links := map[string]string{}
	var collisions []SkillCollision
	for _, src := range hostSkillSources {
		abs := expandHome(src, home)
		for _, n := range skillNames(abs) {
			if w, ok := owner[n]; ok {
				collisions = append(collisions, SkillCollision{Name: n, Winner: w, Loser: src})
				continue
			}
			owner[n] = src
			links[n] = filepath.Join(abs, n)
		}
	}
	return links, collisions
}

// SyncSkillRepo reúne en ~/.pi-gomemory/skills las skills embebidas (copias
// reales con marca) y las de los agentes del anfitrión (enlaces simbólicos).
// Nunca modifica las carpetas del anfitrión ni las carpetas reales que la
// persona tenga en el repositorio; retira los enlaces que ya no corresponden.
func SyncSkillRepo(assets fs.FS, home string) (SkillRepoResult, error) {
	var res SkillRepoResult
	repo := SkillRepoDir(home)
	if err := os.MkdirAll(repo, 0755); err != nil {
		return res, fmt.Errorf("crear %s: %w", repo, err)
	}

	embedded, err := installEmbeddedSkills(assets, repo)
	if err != nil {
		return res, err
	}
	res.Embedded = len(embedded)

	owner := map[string]string{}
	for _, n := range embedded {
		owner[n] = "pi-gomemory"
	}
	entries, _ := os.ReadDir(repo)
	for _, e := range entries {
		if _, ok := owner[e.Name()]; !ok && e.Type()&os.ModeSymlink == 0 && e.IsDir() {
			owner[e.Name()] = SkillRepoSetting // carpeta propia de la persona
		}
	}

	links, collisions := planHostLinks(home, owner)
	res.Collisions = collisions
	for n, target := range links {
		dst := filepath.Join(repo, n)
		if cur, err := os.Readlink(dst); err == nil {
			if cur == target {
				res.Linked++
				continue
			}
			if err := os.Remove(dst); err != nil {
				return res, err
			}
		} else if _, err := os.Lstat(dst); err == nil {
			continue // existe algo que no es un enlace: no se toca
		}
		if err := os.Symlink(target, dst); err != nil {
			return res, fmt.Errorf("enlazar %s: %w", n, err)
		}
		res.Linked++
	}

	entries, _ = os.ReadDir(repo)
	for _, e := range entries {
		if e.Type()&os.ModeSymlink == 0 {
			continue
		}
		if _, ok := links[e.Name()]; ok {
			continue
		}
		// Solo borrar enlaces huérfanos cuyo destino esté bajo las fuentes del anfitrión (hostSkillSources),
		// nunca enlaces creados voluntariamente por la persona apuntando a otros lugares.
		p := filepath.Join(repo, e.Name())
		if target, err := os.Readlink(p); err == nil {
			isHostSource := false
			for _, src := range hostSkillSources {
				if isUnder(target, expandHome(src, home)) {
					isHostSource = true
					break
				}
			}
			if !isHostSource {
				continue
			}
		}
		if err := os.Remove(p); err != nil {
			return res, err
		}
	}
	return res, nil
}

// RegisterSkillRepo hace que Pi cargue el repositorio central: lo registra en
// settings.json, retira las rutas del anfitrión que registraba HCD-002 y saca
// de ~/.pi/agent/skills las copias embebidas intactas y los enlaces a carpetas
// del anfitrión. Las skills propias de la persona se conservan.
func RegisterSkillRepo(piAgentDir, home string) error {
	path, err := configTarget(filepath.Join(piAgentDir, "settings.json"))
	if err != nil {
		return err
	}
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
	isLegacy := func(s string) bool {
		clean := filepath.Clean(expandHome(strings.TrimPrefix(s, "-"), home))
		for _, l := range legacyHostSettings {
			root := filepath.Clean(expandHome(l, home))
			if clean == root || (strings.HasPrefix(s, "-") && strings.HasPrefix(clean, root+string(filepath.Separator))) {
				return true
			}
		}
		return false
	}
	repoAbs := filepath.Clean(SkillRepoDir(home))
	var out []string
	hasRepo := false
	for _, s := range skills {
		if isLegacy(s) {
			continue
		}
		if filepath.Clean(expandHome(s, home)) == repoAbs {
			if hasRepo {
				continue
			}
			hasRepo = true
		}
		out = append(out, s)
	}
	if !hasRepo {
		out = append(out, SkillRepoSetting)
	}
	if strings.Join(out, "\x00") != strings.Join(skills, "\x00") {
		if err := setJSONField(obj, "skills", out); err != nil {
			return err
		}
		if err := os.MkdirAll(piAgentDir, 0755); err != nil {
			return err
		}
		if err := writeJSONObject(path, obj); err != nil {
			return err
		}
	}
	return pruneLegacyPiSkills(filepath.Join(piAgentDir, "skills"), home)
}

// onlySkillFile devuelve true si dir sólo contiene SKILL.md y opcionalmente
// la marca .pi-gomemory. Si contiene scripts, notas o carpetas del usuario, false.
func onlySkillFile(dir string) bool {
	es, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range es {
		if e.Name() != "SKILL.md" && e.Name() != skillMarkerFile {
			return false
		}
	}
	return true
}

// pruneLegacyPiSkills retira de ~/.pi/agent/skills lo que ahora ofrece el
// repositorio: copias embebidas intactas y enlaces a carpetas del anfitrión.
func pruneLegacyPiSkills(dir, home string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	repoDir := SkillRepoDir(home)
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		remove := false
		if e.Type()&os.ModeSymlink != 0 {
			if target, err := os.Readlink(p); err == nil {
				for _, src := range hostSkillSources {
					if isUnder(target, expandHome(src, home)) {
						remove = true
					}
				}
			}
		} else if e.IsDir() {
			if isPristineEmbeddedCopy(p) && onlySkillFile(p) {
				remove = true
			} else {
				// También retira copias heredadas de versiones previas que no tenían .pi-gomemory
				// pero cuyo contenido en SKILL.md es idéntico al del repositorio central
				// Y no contienen ningún otro archivo o carpeta del usuario.
				repoSkillFile := filepath.Join(repoDir, e.Name(), "SKILL.md")
				piSkillFile := filepath.Join(p, "SKILL.md")
				if repoContent, err := os.ReadFile(repoSkillFile); err == nil {
					if piContent, err := os.ReadFile(piSkillFile); err == nil {
						if bytes.Equal(bytes.TrimSpace(repoContent), bytes.TrimSpace(piContent)) && onlySkillFile(p) {
							remove = true
						}
					}
				}
			}
		}
		if remove {
			if err := os.RemoveAll(p); err != nil {
				return err
			}
		}
	}
	return nil
}

// isUnder indica si target está dentro de root, comparando también las rutas
// con los enlaces resueltos (p. ej. ~/.claude enlazado a otro disco).
func isUnder(target, root string) bool {
	under := func(t, r string) bool {
		return strings.HasPrefix(filepath.Clean(t), filepath.Clean(r)+string(filepath.Separator))
	}
	if under(target, root) {
		return true
	}
	rr, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	if under(target, rr) {
		return true
	}
	if rt, err := filepath.EvalSymlinks(filepath.Dir(target)); err == nil {
		return under(filepath.Join(rt, filepath.Base(target)), rr)
	}
	return false
}

// SkillRepoSummary describe el repositorio para doctor. Solo lee.
type SkillRepoSummary struct {
	Total      int
	Embedded   int
	Linked     int
	Collisions int
	Registered bool
}

// InspectSkillRepo cuenta lo que Pi recibe del repositorio central.
func InspectSkillRepo(piAgentDir, home string) SkillRepoSummary {
	var s SkillRepoSummary
	repo := SkillRepoDir(home)
	owner := map[string]string{}
	entries, _ := os.ReadDir(repo)
	for _, e := range entries {
		p := filepath.Join(repo, e.Name())
		if _, err := os.Stat(filepath.Join(p, "SKILL.md")); err != nil {
			continue
		}
		s.Total++
		if e.Type()&os.ModeSymlink != 0 {
			s.Linked++
			continue
		}
		if _, err := os.Stat(filepath.Join(p, skillMarkerFile)); err == nil {
			s.Embedded++
		}
		owner[e.Name()] = "repo"
	}
	_, collisions := planHostLinks(home, owner)
	s.Collisions = len(collisions)
	if obj, err := readJSONObject(filepath.Join(piAgentDir, "settings.json")); err == nil {
		var skills []string
		_ = json.Unmarshal(obj["skills"], &skills)
		for _, x := range skills {
			if filepath.Clean(expandHome(x, home)) == filepath.Clean(repo) {
				s.Registered = true
			}
		}
	}
	return s
}

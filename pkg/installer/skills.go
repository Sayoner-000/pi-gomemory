package installer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// skillMarkerFile marca una carpeta de skill instalada por pi-gomemory y guarda
// el SHA256 del SKILL.md escrito, para no borrar nunca una skill ajena o editada.
const skillMarkerFile = ".pi-gomemory"

// memSkillSources son las carpetas donde `mem` instala sus skills, en orden de
// preferencia. Su copia es la vigente: gana a la embebida, que solo queda como
// respaldo si `mem` no instaló la skill en ningún agente (HCD-073).
var memSkillSources = []string{
	"~/.claude/skills",
	"~/.config/opencode/skills",
	"~/.codex/skills",
	"~/.gemini/config/skills",
}

// maxSkillBytes acota la lectura de un SKILL.md ajeno.
const maxSkillBytes = 1 << 20

// memSkillContent devuelve el SKILL.md que `mem` instaló para name. Ignora
// rutas que resuelven dentro de repo (evita leer su propia copia) y archivos
// que no son regulares o superan maxSkillBytes.
func memSkillContent(home, repo, name string) ([]byte, bool) {
	for _, src := range memSkillSources {
		p := filepath.Join(expandHome(src, home), name, "SKILL.md")
		if real, err := filepath.EvalSymlinks(p); err != nil || isUnder(real, repo) {
			continue
		}
		fi, err := os.Stat(p)
		if err != nil || !fi.Mode().IsRegular() || fi.Size() > maxSkillBytes {
			continue
		}
		if b, err := os.ReadFile(p); err == nil && len(bytes.TrimSpace(b)) > 0 {
			return b, true
		}
	}
	return nil, false
}

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
	var buf bytes.Buffer
	inFrontmatter := false
	frontmatterDashes := 0

	rawLines := bytes.SplitAfter(content, []byte("\n"))
	for i, raw := range rawLines {
		// bytes.SplitAfter deja un elemento vacío al final si content termina en "\n".
		if i == len(rawLines)-1 && len(raw) == 0 {
			continue
		}
		line := strings.TrimSuffix(string(bytes.TrimSuffix(raw, []byte("\n"))), "\r")
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			frontmatterDashes++
			if frontmatterDashes == 1 {
				inFrontmatter = true
			} else if frontmatterDashes == 2 {
				inFrontmatter = false
			}
		} else if inFrontmatter && strings.HasPrefix(trimmed, "description:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			if !strings.HasPrefix(val, `"`) && !strings.HasPrefix(val, `'`) && strings.Contains(val, ":") {
				cleanVal := strings.ReplaceAll(val, `"`, `\"`)
				line = fmt.Sprintf(`description: "%s"`, cleanVal)
			}
		}

		buf.WriteString(line)
		buf.WriteByte('\n')
	}

	return buf.Bytes()
}

// installEmbeddedSkills copia y sanitiza las skills de gomemory en dir, cada
// una con su marca. El contenido sale de la copia que `mem` instaló en el
// anfitrión y, si no hay, del binario (HCD-073). Sustituye enlaces y copias
// intactas; nunca pisa una carpeta propia de la persona ni una copia editada.
func installEmbeddedSkills(assets fs.FS, dir, home string) ([]string, error) {
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
		content, err := wantedSkillContent(assets, dir, home, skillName)
		if err != nil {
			return names, err
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
		if err := writeFileAtomic(filepath.Join(dstSkillDir, "SKILL.md"), content, 0644); err != nil {
			return names, fmt.Errorf("escribir SKILL.md de %s: %w", skillName, err)
		}
		if err := writeFileAtomic(filepath.Join(dstSkillDir, skillMarkerFile), []byte(contentHash(content)+"\n"), 0644); err != nil {
			return names, fmt.Errorf("escribir marca de %s: %w", skillName, err)
		}
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("el binario no contiene skills embebidas")
	}
	return names, nil
}

// wantedSkillContent es el SKILL.md que la sincronización escribe para name:
// la copia que instaló `mem` (si hay) o la embebida, saneada (HCD-073).
func wantedSkillContent(assets fs.FS, repo, home, name string) ([]byte, error) {
	raw, err := fs.ReadFile(assets, "skills/"+name+"/SKILL.md")
	if err != nil {
		return nil, fmt.Errorf("leer skill embebida %s: %w", name, err)
	}
	if home != "" {
		if b, ok := memSkillContent(home, repo, name); ok {
			raw = b
		}
	}
	return SanitizeFrontmatter(raw), nil
}

// SkillRepoDrift lista, en orden, las skills de gomemory que una sincronización
// cambiaría en el repositorio central: faltan, son un enlace o su copia intacta
// difiere de la vigente (normalmente tras `mem update`). Ignora las copias
// editadas por la persona, igual que la sincronización. Solo lee (HCD-080).
func SkillRepoDrift(assets fs.FS, home string) ([]string, error) {
	if IsUninstalled(home) {
		return nil, nil
	}
	entries, err := fs.ReadDir(assets, "skills")
	if err != nil {
		return nil, fmt.Errorf("leer skills embebidas: %w", err)
	}
	repo := SkillRepoDir(home)
	var drift []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		dst := filepath.Join(repo, name)
		fi, err := os.Lstat(dst)
		switch {
		case err != nil, fi.Mode()&os.ModeSymlink != 0:
			drift = append(drift, name)
			continue
		case !isPristineEmbeddedCopy(dst):
			continue // carpeta propia o copia editada: la sincronización no la toca
		}
		want, err := wantedSkillContent(assets, repo, home, name)
		if err != nil {
			return drift, err
		}
		if have, err := os.ReadFile(filepath.Join(dst, "SKILL.md")); err != nil || !bytes.Equal(have, want) {
			drift = append(drift, name)
		}
	}
	return drift, nil
}

// SkillRepoEdited lista, en orden, las skills de gomemory que pi-gomemory
// instaló (tienen marca) y alguien editó después, cuando su contenido ya no
// coincide con el vigente. La sincronización no las toca; se informan para que
// la persona decida reponerlas (HCD-080, Q1). Las carpetas propias sin marca no
// cuentan. Solo lee.
func SkillRepoEdited(assets fs.FS, home string) ([]string, error) {
	if IsUninstalled(home) {
		return nil, nil
	}
	entries, err := fs.ReadDir(assets, "skills")
	if err != nil {
		return nil, fmt.Errorf("leer skills embebidas: %w", err)
	}
	repo := SkillRepoDir(home)
	var edited []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		dst := filepath.Join(repo, name)
		if fi, err := os.Lstat(dst); err != nil || !fi.IsDir() || isPristineEmbeddedCopy(dst) {
			continue
		}
		if _, err := os.Stat(filepath.Join(dst, skillMarkerFile)); err != nil {
			continue // carpeta propia de la persona
		}
		want, err := wantedSkillContent(assets, repo, home, name)
		if err != nil {
			return edited, err
		}
		if have, err := os.ReadFile(filepath.Join(dst, "SKILL.md")); err != nil || !bytes.Equal(have, want) {
			edited = append(edited, name)
		}
	}
	return edited, nil
}

// ErrUnknownSkill indica que un nombre no es una skill de gomemory embebida.
var ErrUnknownSkill = errors.New("no es una skill de gomemory")

// RestoreSkills repone en el repositorio central las skills de gomemory names,
// descartando las ediciones locales, y vuelve a sincronizar (HCD-080, Q1).
// Valida todos los nombres antes de tocar nada y se niega si una carpeta
// contiene algo más que SKILL.md y la marca, para no borrar archivos propios.
func RestoreSkills(assets fs.FS, opts *Options, names []string) error {
	if IsUninstalled(opts.HomeDir) {
		return errors.New("pi-gomemory está desinstalado; ejecuta `pi-gomemory-setup install` antes de reponer skills")
	}
	repo := SkillRepoDir(opts.HomeDir)
	var dirs []string
	for _, n := range names {
		if n == "" || n != filepath.Base(n) || strings.HasPrefix(n, ".") {
			return fmt.Errorf("%q: %w", n, ErrUnknownSkill)
		}
		if _, err := fs.Stat(assets, "skills/"+n+"/SKILL.md"); err != nil {
			return fmt.Errorf("%q: %w", n, ErrUnknownSkill)
		}
		dir := filepath.Join(repo, n)
		fi, err := os.Lstat(dir)
		if err != nil {
			continue // ausente: la sincronización la crea
		}
		if fi.IsDir() && !onlySkillFile(dir) {
			return fmt.Errorf("%s contiene archivos propios; muévelos antes de reponer la skill", dir)
		}
		dirs = append(dirs, dir)
	}
	for _, d := range dirs {
		if err := os.RemoveAll(d); err != nil {
			return err
		}
	}
	_, err := InstallSkills(assets, opts)
	return err
}

// EmbeddedSkillDrift compara las skills embebidas en assets con las copias que
// `mem` instaló en los agentes de home. Devuelve, en orden, las que difieren y
// cuántas pudo comparar (0 = no hay copias de mem). Sirve para que el repositorio
// detecte que sus skills/ se quedaron por detrás de gomemory (HCD-080).
func EmbeddedSkillDrift(assets fs.FS, home string) ([]string, int) {
	entries, err := fs.ReadDir(assets, "skills")
	if err != nil {
		return nil, 0
	}
	repo := SkillRepoDir(home)
	var drift []string
	compared := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		mem, ok := memSkillContent(home, repo, name)
		if !ok {
			continue
		}
		compared++
		if embedded, err := fs.ReadFile(assets, "skills/"+name+"/SKILL.md"); err != nil || !bytes.Equal(embedded, mem) {
			drift = append(drift, name)
		}
	}
	return drift, compared
}

// InstallSkills reúne las skills embebidas y las del anfitrión en el
// repositorio central (~/.pi-gomemory/skills) y hace que Pi lo cargue.
func InstallSkills(assets fs.FS, opts *Options) (SkillRepoResult, error) {
	if IsUninstalled(opts.HomeDir) {
		return SkillRepoResult{}, nil
	}
	res, err := SyncSkillRepo(assets, opts.HomeDir)
	if err != nil {
		return res, err
	}
	return res, RegisterSkillRepo(opts.PiAgentDir, opts.HomeDir)
}

package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"
)

// Motivos de las acciones que se revalidan al aplicar (HCD-034).
const (
	reasonChanged       = "cambió después del plan"
	reasonGone          = "ya no existe"
	reasonUninspectable = "no se pudo inspeccionar: "
)

// errChangedAfterPlan y errGoneAfterPlan los devuelve una acción que, al
// revalidar justo antes de escribir, encuentra el recurso cambiado o ausente.
var (
	errChangedAfterPlan = errors.New(reasonChanged)
	errGoneAfterPlan    = errors.New(reasonGone)
)

// isAbsent indica si err significa que el recurso no existe (Q5).
func isAbsent(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}

// goneOr traduce la ausencia a errGoneAfterPlan y deja pasar el resto.
func goneOr(err error) error {
	if isAbsent(err) {
		return errGoneAfterPlan
	}
	return err
}

// readLink y selfExecutable son os.Readlink y os.Executable; las pruebas los
// sustituyen para provocar errores que no se pueden fabricar en disco.
var (
	readLink       = os.Readlink
	selfExecutable = os.Executable
)

// Tipos de acción de un plan de desinstalación.
const (
	ActionRemove = "retirar"
	ActionKeep   = "conservar"
	ActionSkip   = "omitir"
	ActionFail   = "fallido"
)

// UninstallFlags son las opciones de `pi-gomemory-setup uninstall`.
type UninstallFlags struct {
	DryRun   bool
	Yes      bool
	Verbose  bool
	Packages bool
	Self     bool
}

// UninstallAction es un paso del plan. apply solo existe en las acciones que retiran.
// notice marca las omisiones que se muestran aunque no quede nada que retirar.
type UninstallAction struct {
	Kind   string
	Target string
	Reason string
	apply  func() error
	notice bool
	// ifClean marca la retirada del manifiesto: se aplica la última y solo si
	// no falló ninguna acción ajena a él (HCD-035, Q12, Q17).
	ifClean bool
}

// Motivos de la retirada condicional del manifiesto (HCD-035).
const (
	reasonManifest     = "manifiesto de instalación (se retira si no hay fallos)"
	reasonManifestKept = "conservado por fallos previos"
)

// UninstallPlan es la lista ordenada de acciones; construirlo no escribe nada.
type UninstallPlan struct {
	Actions []UninstallAction
}

func (p *UninstallPlan) add(kind, target, reason string, apply func() error) {
	p.Actions = append(p.Actions, UninstallAction{Kind: kind, Target: target, Reason: reason, apply: apply})
}

// addNotice añade una omisión que se imprime siempre (Q7).
func (p *UninstallPlan) addNotice(target, reason string) {
	p.Actions = append(p.Actions, UninstallAction{Kind: ActionSkip, Target: target, Reason: reason, notice: true})
}

// inspectFailed registra un error de inspección: la ausencia es inocua y el
// resto es fallido (Q2).
func (p *UninstallPlan) inspectFailed(target string, err error) {
	if !isAbsent(err) {
		p.add(ActionFail, target, err.Error(), nil)
	}
}

// Pending cuenta las acciones que retirarían algo.
func (p *UninstallPlan) Pending() int {
	return p.Count(ActionRemove)
}

// Count cuenta las acciones de un tipo dado en el plan.
func (p *UninstallPlan) Count(kind string) int {
	n := 0
	for _, a := range p.Actions {
		if a.Kind == kind {
			n++
		}
	}
	return n
}

// ownedHash devuelve el SHA256 que prueba la propiedad de path: el del
// manifiesto si lo registra; si no, el del contenido que escribe esta versión.
func ownedHash(m *Manifest, path string, fallback []byte) string {
	if m != nil {
		if h, ok := m.Files[path]; ok {
			return h
		}
	}
	if fallback == nil {
		return ""
	}
	return contentHash(fallback)
}

// planOwnedFile añade la retirada de path si su contenido coincide con el instalado.
func (p *UninstallPlan) planOwnedFile(m *Manifest, path string, fallback []byte) string {
	return p.planOwned(m, path, fallback, false)
}

// planOwned es planOwnedFile; candidate indica una carpeta de launcher que no
// respaldan el manifiesto ni PI_GOMEMORY_BIN_DIR, cuyo archivo ilegible se
// omite porque puede no ser de pi-gomemory (Q5).
func (p *UninstallPlan) planOwned(m *Manifest, path string, fallback []byte, candidate bool) string {
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
	case isAbsent(err):
		return ""
	case candidate:
		p.addNotice(path, reasonUninspectable+err.Error())
		return ""
	default:
		p.add(ActionFail, path, err.Error(), nil)
		return ActionFail
	}
	if want := ownedHash(m, path, fallback); want != "" && contentHash(data) == want {
		planned := contentHash(data)
		p.add(ActionRemove, path, "instalado por pi-gomemory", func() error {
			if err := ownedFileUnchanged(path, planned); err != nil {
				return err
			}
			return os.Remove(path)
		})
		return ActionRemove
	}
	p.add(ActionSkip, path, "editado: no coincide con lo instalado", nil)
	return ActionSkip
}

// ownedFileUnchanged comprueba que path sigue siendo un archivo regular con
// la huella que examinó el plan; un symlink en su lugar cuenta como cambio.
func ownedFileUnchanged(path, planned string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return goneOr(err)
	}
	if !info.Mode().IsRegular() {
		return errChangedAfterPlan
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return goneOr(err)
	}
	if contentHash(data) != planned {
		return errChangedAfterPlan
	}
	return nil
}

// executablePath es os.Executable; los tests lo sustituyen para simular dónde
// se ejecuta pi-gomemory-setup.
var executablePath = os.Executable

// launcherDirs son las carpetas donde puede estar el launcher de esta
// instalación, sin repetir. La carpeta del ejecutable en curso solo cuenta si
// está dentro de HOME o ya la respalda el manifiesto o PI_GOMEMORY_BIN_DIR; si
// no, se devuelve aparte en foreign: con otro HOME pertenece a otra instalación
// y no se borra (HCD-030, C-004).
func launcherDirs(m *Manifest, opts *Options) (dirs []string, foreign string) {
	var candidates []string
	if m != nil && m.BinDir != "" {
		candidates = append(candidates, m.BinDir)
	}
	candidates = append(candidates, opts.LocalBinDir)
	if v := os.Getenv("PI_GOMEMORY_BIN_DIR"); v != "" {
		candidates = append(candidates, v)
	}
	candidates = append(candidates, filepath.Join(opts.HomeDir, ".local", "bin"))
	seen := map[string]bool{}
	for _, d := range candidates {
		c := filepath.Clean(d)
		if !seen[c] {
			seen[c] = true
			dirs = append(dirs, c)
		}
	}
	if exe, err := executablePath(); err == nil {
		d := filepath.Clean(filepath.Dir(exe))
		switch {
		case seen[d]:
		case isUnder(d, opts.HomeDir):
			dirs = append(dirs, d)
		default:
			foreign = d
		}
	}
	return dirs, foreign
}

func isHostSkillLink(target, home string) bool {
	for _, src := range hostSkillSources {
		if isUnder(target, expandHome(src, home)) {
			return true
		}
	}
	return false
}

// isRemovableEmbeddedCopy exige una copia intacta con solo SKILL.md y la marca.
// Devuelve error si no pudo inspeccionarla por algo distinto de la ausencia.
func isRemovableEmbeddedCopy(dir string) (bool, error) {
	read := func(name string) ([]byte, error) {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if isAbsent(err) {
			return nil, nil
		}
		return data, err
	}
	marker, err := read(skillMarkerFile)
	if err != nil || marker == nil {
		return false, err
	}
	content, err := read("SKILL.md")
	if err != nil || content == nil {
		return false, err
	}
	if strings.TrimSpace(string(marker)) != contentHash(content) {
		return false, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if isAbsent(err) {
			return false, nil
		}
		return false, err
	}
	if len(entries) != 2 {
		return false, nil
	}
	for _, e := range entries {
		if e.Name() != "SKILL.md" && e.Name() != skillMarkerFile {
			return false, nil
		}
	}
	return true, nil
}

// planSkillDir planifica las skills de dir (repositorio central o ruta antigua de Pi).
func (p *UninstallPlan) planSkillDir(dir, home string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		p.inspectFailed(dir, err)
		return
	}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if e.Type()&os.ModeSymlink != 0 {
			target, err := readLink(path)
			switch {
			case err != nil:
				p.inspectFailed(path, err)
			case isHostSkillLink(target, home):
				p.add(ActionRemove, path, "enlace a skill del anfitrión (no se sigue)", func() error {
					if err := hostLinkUnchanged(path, target); err != nil {
						return err
					}
					return os.Remove(path)
				})
			default:
				p.add(ActionSkip, path, "enlace propio", nil)
			}
			continue
		}
		if !e.IsDir() {
			continue
		}
		removable, err := isRemovableEmbeddedCopy(path)
		if err != nil {
			p.add(ActionFail, path, err.Error(), nil)
			continue
		}
		if removable {
			p.add(ActionRemove, path, "skill embebida intacta", func() error {
				if _, err := os.Lstat(path); err != nil {
					return goneOr(err)
				}
				ok, err := isRemovableEmbeddedCopy(path)
				if err != nil {
					return err
				}
				if !ok {
					return errChangedAfterPlan
				}
				return os.RemoveAll(path)
			})
			continue
		}
		marker := filepath.Join(path, skillMarkerFile)
		switch _, err := os.Stat(marker); {
		case err == nil:
			p.add(ActionSkip, path, "skill embebida editada o con archivos añadidos", nil)
		case isAbsent(err):
			p.add(ActionSkip, path, "skill propia", nil)
		default:
			p.add(ActionFail, marker, err.Error(), nil)
		}
	}
}

// hostLinkUnchanged comprueba que path sigue siendo un enlace al mismo destino.
func hostLinkUnchanged(path, target string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return goneOr(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return errChangedAfterPlan
	}
	cur, err := readLink(path)
	if err != nil {
		return goneOr(err)
	}
	if cur != target {
		return errChangedAfterPlan
	}
	return nil
}

func sameJSON(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// editJSON relee path, aplica fn y lo escribe de forma atómica. Si path es un
// enlace, edita su destino sin sustituir el enlace (HCD-036).
func editJSON(path string, fn func(map[string]json.RawMessage) error) error {
	target, err := configTarget(path)
	if err != nil {
		return err
	}
	obj, err := readJSONObject(target)
	if err != nil {
		return err
	}
	if err := fn(obj); err != nil {
		return err
	}
	return writeJSONObject(target, obj)
}

// editPlannedJSON es editJSON para una retirada planificada: si el archivo ya
// no existe devuelve errGoneAfterPlan en lugar de crearlo vacío. fn revalida
// y devuelve errChangedAfterPlan o errGoneAfterPlan sin escribir nada. Un
// enlace que ya no lleva a un archivo es un fallo, no una ausencia (C-16).
func editPlannedJSON(path string, fn func(map[string]json.RawMessage) error) error {
	target, err := configTarget(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(target); err != nil {
		return goneOr(err)
	}
	return editJSON(target, fn)
}

// planConfig comprueba una configuración de Pi antes de inspeccionarla: un
// enlace que no lleva a un archivo es un fallo del plan y no una ausencia
// inocua (HCD-036, Q6). Devuelve false si no hay que seguir.
func (p *UninstallPlan) planConfig(path string) bool {
	if _, err := configTarget(path); err != nil {
		p.add(ActionFail, path, err.Error(), nil)
		return false
	}
	return true
}

func (p *UninstallPlan) planMCP(piAgentDir string) {
	path := filepath.Join(piAgentDir, "mcp.json")
	if !p.planConfig(path) {
		return
	}
	if _, err := os.Stat(path); err != nil {
		p.inspectFailed(path, err)
		return
	}
	obj, err := readJSONObject(path)
	if err != nil {
		p.add(ActionFail, path, err.Error(), nil)
		return
	}
	servers := map[string]json.RawMessage{}
	if raw, ok := obj["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			p.add(ActionFail, path, "mcpServers no es un objeto", nil)
			return
		}
	}
	defs := DefaultMCPServers()
	for _, name := range []string{"gomemory", "codegraph", "codebase-memory"} {
		cur, ok := servers[name]
		if !ok {
			continue
		}
		target := path + " › mcpServers." + name
		want, _ := json.Marshal(defs[name])
		if !sameJSON(cur, want) {
			p.add(ActionSkip, target, "no coincide con la definición de pi-gomemory", nil)
			continue
		}
		n := name
		p.add(ActionRemove, target, "servidor MCP registrado por pi-gomemory", func() error {
			return editPlannedJSON(path, func(o map[string]json.RawMessage) error {
				raw, ok := o["mcpServers"]
				if !ok {
					return errGoneAfterPlan
				}
				s := map[string]json.RawMessage{}
				if err := json.Unmarshal(raw, &s); err != nil {
					return errChangedAfterPlan
				}
				cur, ok := s[n]
				if !ok {
					return errGoneAfterPlan
				}
				if !sameJSON(cur, want) {
					return errChangedAfterPlan
				}
				delete(s, n)
				return setJSONField(o, "mcpServers", s)
			})
		})
	}
}

// stripInstructions quita el bloque de pi-gomemory (o el protocolo sin marcadores)
// y el separador que añadió el instalador. ok indica si había algo que quitar.
func stripInstructions(content string) (string, bool) {
	start := strings.Index(content, instructionsStart)
	end := strings.Index(content, instructionsEnd)
	var before, after string
	switch {
	case start >= 0 && end > start:
		before, after = content[:start], content[end+len(instructionsEnd):]
	case strings.Contains(content, GomemoryPiInstructions):
		i := strings.Index(content, GomemoryPiInstructions)
		before, after = content[:i], content[i+len(GomemoryPiInstructions):]
	default:
		return content, false
	}
	after = strings.TrimPrefix(after, "\n")
	if strings.TrimSpace(after) == "" {
		before = strings.TrimRight(before, "\n")
		if before != "" {
			before += "\n"
		}
	}
	return before + after, true
}

// instructionsBlock devuelve el bloque de pi-gomemory que stripInstructions quitaría.
func instructionsBlock(content string) (string, bool) {
	start := strings.Index(content, instructionsStart)
	end := strings.Index(content, instructionsEnd)
	switch {
	case start >= 0 && end > start:
		return content[start : end+len(instructionsEnd)], true
	case strings.Contains(content, GomemoryPiInstructions):
		return GomemoryPiInstructions, true
	}
	return "", false
}

func (p *UninstallPlan) planAgents(piAgentDir string) {
	path := filepath.Join(piAgentDir, "AGENTS.md")
	if !p.planConfig(path) {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		p.inspectFailed(path, err)
		return
	}
	planned, ok := instructionsBlock(string(data))
	if !ok {
		return
	}
	p.add(ActionRemove, path+" › bloque pi-gomemory", "protocolo escrito por pi-gomemory", func() error {
		// Un AGENTS.md enlazado se edita en su destino; el enlace nunca se
		// borra ni se sustituye (HCD-036, Q4).
		target, err := configTarget(path)
		if err != nil {
			return err
		}
		cur, err := os.ReadFile(target)
		if err != nil {
			return goneOr(err)
		}
		block, ok := instructionsBlock(string(cur))
		if !ok {
			return errGoneAfterPlan
		}
		if block != planned {
			return errChangedAfterPlan
		}
		out, _ := stripInstructions(string(cur))
		if strings.TrimSpace(out) == "" {
			if target == path {
				return os.Remove(path)
			}
			out = ""
		}
		mode, err := modeOrDefault(target, 0644)
		if err != nil {
			return err
		}
		return writeFileAtomic(target, []byte(out), mode)
	})
}

func (p *UninstallPlan) planSettings(piAgentDir, home, themeFile string) {
	path := filepath.Join(piAgentDir, "settings.json")
	if !p.planConfig(path) {
		return
	}
	if _, err := os.Stat(path); err != nil {
		p.inspectFailed(path, err)
		return
	}
	obj, err := readJSONObject(path)
	if err != nil {
		p.add(ActionFail, path, err.Error(), nil)
		return
	}
	var theme string
	_ = json.Unmarshal(obj["theme"], &theme)
	if theme == "gomemory" {
		target := path + " › theme"
		switch themeFile {
		case ActionSkip:
			p.add(ActionKeep, target, "el archivo del tema está editado y se conserva", nil)
		case ActionFail:
			p.add(ActionKeep, target, "el archivo del tema no se pudo inspeccionar y se conserva", nil)
		default:
			p.add(ActionRemove, target, "tema gomemory activado por pi-gomemory", func() error {
				return editPlannedJSON(path, func(o map[string]json.RawMessage) error {
					raw, ok := o["theme"]
					if !ok {
						return errGoneAfterPlan
					}
					var cur string
					if json.Unmarshal(raw, &cur) != nil || cur != "gomemory" {
						return errChangedAfterPlan
					}
					delete(o, "theme")
					return nil
				})
			})
		}
	}

	var skills []string
	if raw, ok := obj["skills"]; ok && json.Unmarshal(raw, &skills) == nil {
		repo := filepath.Clean(SkillRepoDir(home))
		for _, s := range skills {
			if filepath.Clean(expandHome(s, home)) != repo {
				continue
			}
			p.add(ActionRemove, path+" › skills ("+SkillRepoSetting+")", "registro del repositorio de skills", func() error {
				return editPlannedJSON(path, func(o map[string]json.RawMessage) error {
					raw, ok := o["skills"]
					if !ok {
						return errGoneAfterPlan
					}
					var cur, out []string
					if json.Unmarshal(raw, &cur) != nil {
						return errChangedAfterPlan
					}
					found := false
					for _, x := range cur {
						if filepath.Clean(expandHome(x, home)) == repo {
							found = true
						}
					}
					if !found {
						return errGoneAfterPlan
					}
					for _, x := range cur {
						if filepath.Clean(expandHome(x, home)) != repo {
							out = append(out, x)
						}
					}
					if out == nil {
						out = []string{}
					}
					return setJSONField(o, "skills", out)
				})
			})
			break
		}
	}
}

func (p *UninstallPlan) planPlainFile(path, reason string) {
	if _, err := os.Lstat(path); err != nil {
		p.inspectFailed(path, err)
		return
	}
	p.add(ActionRemove, path, reason, func() error { return os.Remove(path) })
}

// backedLauncherDirs son las carpetas de launcher que respaldan el manifiesto
// (o, sin manifiesto, LocalBinDir) y PI_GOMEMORY_BIN_DIR; las demás son
// candidatas (HCD-034, Q5).
func backedLauncherDirs(m *Manifest, opts *Options) map[string]bool {
	backed := map[string]bool{}
	if m != nil && m.BinDir != "" {
		backed[filepath.Clean(m.BinDir)] = true
	} else {
		backed[filepath.Clean(opts.LocalBinDir)] = true
	}
	if v := os.Getenv("PI_GOMEMORY_BIN_DIR"); v != "" {
		backed[filepath.Clean(v)] = true
	}
	return backed
}

// BuildUninstallPlan decide, sin escribir nada, qué se retira, conserva u omite.
func BuildUninstallPlan(assets fs.FS, opts *Options, flags UninstallFlags) (*UninstallPlan, error) {
	plan := &UninstallPlan{}
	m, err := ReadManifest(opts.HomeDir)
	if err != nil {
		plan.inspectFailed(ManifestPath(opts.HomeDir), err)
		m = nil
	}

	for _, r := range tuiResources {
		data, _ := fs.ReadFile(assets, r.source)
		plan.planOwnedFile(m, filepath.Join(opts.PiAgentDir, "extensions", r.destination), data)
	}
	themeData, _ := fs.ReadFile(assets, "themes/gomemory.json")
	themeFile := plan.planOwnedFile(m, filepath.Join(opts.PiAgentDir, "themes", "gomemory.json"), themeData)
	dirs, foreign := launcherDirs(m, opts)
	backed := backedLauncherDirs(m, opts)
	for _, dir := range dirs {
		plan.planOwned(m, filepath.Join(dir, "pi-gomemory"), []byte(GenerateHarnessScript()), !backed[dir])
	}
	if foreign != "" {
		path := filepath.Join(foreign, "pi-gomemory")
		switch _, err := os.Lstat(path); {
		case err == nil:
			plan.addNotice(path, "fuera de HOME y sin manifiesto: retíralo a mano o usa PI_GOMEMORY_BIN_DIR")
		case !isAbsent(err):
			plan.addNotice(path, reasonUninspectable+err.Error())
		}
	}

	plan.planSkillDir(SkillRepoDir(opts.HomeDir), opts.HomeDir)
	plan.planSkillDir(filepath.Join(opts.PiAgentDir, "skills"), opts.HomeDir)
	plan.planMCP(opts.PiAgentDir)
	plan.planAgents(opts.PiAgentDir)
	plan.planSettings(opts.PiAgentDir, opts.HomeDir, themeFile)

	plan.planPlainFile(DefaultDepsCachePath(opts.PiAgentDir), "caché de dependencias")
	plan.planPlainFile(DefaultUpdateCachePath(opts.PiAgentDir), "caché de actualizaciones")
	if flags.Packages {
		plan.planPackages(assets, opts.PiAgentDir)
	}
	if flags.Self {
		plan.planSelf(m, opts)
	}
	plan.planManifest(ManifestPath(opts.HomeDir))
	return plan, nil
}

// planManifest añade, como última acción, la retirada del manifiesto. Guarda
// el BIN_DIR y las huellas que necesita un reintento, por eso Uninstall solo
// la aplica si nada ajeno a él falló (HCD-035, H-01).
func (p *UninstallPlan) planManifest(path string) {
	if _, err := os.Lstat(path); err != nil {
		p.inspectFailed(path, err)
		return
	}
	p.Actions = append(p.Actions, UninstallAction{Kind: ActionRemove, Target: path, Reason: reasonManifest, ifClean: true, apply: func() error { return os.Remove(path) }})
}

// planPackages retira solo las entradas con la fuente fijada exacta y
// pi-cc-extensions.json si coincide con los valores por defecto embebidos.
func (p *UninstallPlan) planPackages(assets fs.FS, piAgentDir string) {
	if settings := filepath.Join(piAgentDir, "settings.json"); p.planConfig(settings) {
		p.planPinnedPackages(settings)
	}
	p.planCCExtensions(assets, piAgentDir)
}

// planPinnedPackages planifica la retirada de las entradas packages fijadas.
func (p *UninstallPlan) planPinnedPackages(settings string) {
	if obj, err := readJSONObject(settings); err != nil {
		p.add(ActionFail, settings, err.Error(), nil)
	} else {
		var pkgs []string
		_ = json.Unmarshal(obj["packages"], &pkgs)
		for _, pp := range pinnedPackages {
			for _, s := range pkgs {
				if s != pp.source {
					continue
				}
				src, name := pp.source, pp.name
				p.add(ActionRemove, settings+" › packages."+pp.name, "paquete fijado por pi-gomemory", func() error {
					// Se revalida antes de pi remove, que desinstalaría el paquete.
					if err := pinnedEntryUnchanged(settings, name, src); err != nil {
						return err
					}
					if piBin, err := exec.LookPath("pi"); err == nil {
						ctxRem, cancelRem := context.WithTimeout(context.Background(), 15*time.Second)
						defer cancelRem()
						if err := exec.CommandContext(ctxRem, piBin, "remove", src).Run(); err != nil {
							return fmt.Errorf("pi remove %s: %w", src, err)
						}
					}
					return editJSON(settings, func(o map[string]json.RawMessage) error {
						var cur []json.RawMessage
						_ = json.Unmarshal(o["packages"], &cur)
						out := []json.RawMessage{}
						for _, c := range cur {
							var str string
							if json.Unmarshal(c, &str) == nil && str == src {
								continue
							}
							out = append(out, c)
						}
						return setJSONField(o, "packages", out)
					})
				})
			}
		}
	}
}

// planCCExtensions planifica pi-cc-extensions.json; solo se llama con --packages.
func (p *UninstallPlan) planCCExtensions(assets fs.FS, piAgentDir string) {
	cfg := filepath.Join(piAgentDir, "pi-cc-extensions.json")
	if !p.planConfig(cfg) {
		return
	}
	if target, _ := configTarget(cfg); target != cfg {
		// Enlazado: es configuración de la persona y nunca se borra (HCD-036, Q4).
		p.addNotice(cfg, "enlace a una configuración propia: se conserva")
		return
	}
	cur, err := os.ReadFile(cfg)
	if err != nil {
		p.inspectFailed(cfg, err)
		return
	}
	defaults, _ := fs.ReadFile(assets, "config/pi-cc-extensions.json")
	if sameJSON(cur, defaults) {
		p.add(ActionRemove, cfg, "configuración por defecto de pi-cc-extensions", func() error {
			now, err := os.ReadFile(cfg)
			if err != nil {
				return goneOr(err)
			}
			if !sameJSON(now, defaults) {
				return errChangedAfterPlan
			}
			return os.Remove(cfg)
		})
	} else {
		p.add(ActionSkip, cfg, "configuración de pi-cc-extensions personalizada", nil)
	}
}

// pinnedEntryUnchanged comprueba que settings.json sigue teniendo la entrada
// packages con la fuente fijada src; otra versión del mismo paquete es un cambio.
func pinnedEntryUnchanged(settings, name, src string) error {
	settings, err := configTarget(settings)
	if err != nil {
		return err
	}
	if _, err := os.Stat(settings); err != nil {
		return goneOr(err)
	}
	obj, err := readJSONObject(settings)
	if err != nil {
		return err
	}
	raw, ok := obj["packages"]
	if !ok {
		return errGoneAfterPlan
	}
	var cur []json.RawMessage
	if json.Unmarshal(raw, &cur) != nil {
		return errChangedAfterPlan
	}
	other := false
	for _, c := range cur {
		var s string
		if json.Unmarshal(c, &s) == nil && s == src {
			return nil
		}
		if npmPackageName(c) == name {
			other = true
		}
	}
	if other {
		return errChangedAfterPlan
	}
	return errGoneAfterPlan
}

// planSelf borra el ejecutable en curso sin resolver enlaces y solo si está en el BIN_DIR efectivo.
func (p *UninstallPlan) planSelf(m *Manifest, opts *Options) {
	exe, err := selfExecutable()
	if err != nil {
		p.add(ActionFail, "pi-gomemory-setup (--self)", err.Error(), nil)
		return
	}
	if base := filepath.Base(exe); !strings.HasPrefix(base, "pi-gomemory-setup") {
		// En tests el ejecutable es el binario de go test: se usa el del BIN_DIR.
		exe = filepath.Join(opts.LocalBinDir, "pi-gomemory-setup")
	}
	if _, err := os.Lstat(exe); err != nil {
		p.inspectFailed(exe, err)
		return
	}
	dir := filepath.Clean(filepath.Dir(exe))
	binDir := filepath.Clean(opts.LocalBinDir)
	if m != nil && m.BinDir != "" {
		binDir = filepath.Clean(m.BinDir)
	}
	if dir != binDir {
		p.add(ActionSkip, exe, "fuera del BIN_DIR de pi-gomemory ("+binDir+")", nil)
		return
	}
	p.add(ActionRemove, exe, "binario pi-gomemory-setup (--self)", func() error { return os.Remove(exe) })
}

// PrintPlan escribe una línea por acción.
func PrintPlan(w io.Writer, plan *UninstallPlan) {
	for _, a := range plan.Actions {
		fmt.Fprintf(w, "  [%s] %s — %s\n", a.Kind, a.Target, a.Reason)
	}
}

// Uninstall construye el plan y lo muestra; con DryRun no escribe nada.
// Sin DryRun pide confirmación (o --yes), ejecuta cada acción sin detenerse
// ante fallos y devuelve error si alguna falló.
func Uninstall(assets fs.FS, opts *Options, flags UninstallFlags, w io.Writer) error {
	plan, err := BuildUninstallPlan(assets, opts, flags)
	if err != nil {
		return err
	}
	if flags.DryRun {
		fmt.Fprintln(w, "Plan de desinstalación (dry-run: no se modifica nada):")
		PrintPlan(w, plan)
		fmt.Fprintf(w, "%d acciones retirarían algo.\n", plan.Pending())
		if n := plan.Count(ActionFail); n > 0 {
			return fmt.Errorf("%d comprobaciones fallaron en plan de desinstalación", n)
		}
		return nil
	}
	if plan.Pending() == 0 {
		if n := plan.Count(ActionFail); n > 0 {
			PrintPlan(w, plan)
			return fmt.Errorf("%d comprobaciones fallaron; no se puede confirmar que no quede nada", n)
		}
		for _, a := range plan.Actions {
			if a.notice {
				fmt.Fprintf(w, "  [%s] %s — %s\n", a.Kind, a.Target, a.Reason)
			}
		}
		fmt.Fprintln(w, "✓ No queda nada que retirar de pi-gomemory.")
		return nil
	}

	fmt.Fprintln(w, "Plan de desinstalación:")
	PrintPlan(w, plan)
	if !flags.Yes {
		if !stdinIsTerminal() {
			return fmt.Errorf("no hay terminal interactiva: revisa el plan con --dry-run y confirma con --yes")
		}
		fmt.Fprintf(w, "¿Retirar %d elementos? [s/N]: ", plan.Pending())
		var answer string
		_, _ = fmt.Fscanln(confirmInput, &answer)
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "s" && a != "si" && a != "sí" && a != "y" && a != "yes" {
			return fmt.Errorf("desinstalación cancelada: no se modificó nada")
		}
	}

	var log strings.Builder
	record := func(kind, target, reason string) {
		line := fmt.Sprintf("  [%s] %s — %s\n", kind, target, reason)
		log.WriteString(line)
		if flags.Verbose {
			fmt.Fprint(w, line)
		}
	}

	if err := os.MkdirAll(filepath.Dir(UninstalledMarkerPath(opts.HomeDir)), 0755); err == nil {
		_ = os.WriteFile(UninstalledMarkerPath(opts.HomeDir), []byte(Version+"\n"), 0644)
	}

	counts := map[string]int{}
	changed := 0
	blocking := 0 // fallos ajenos al manifiesto (Q12)
	manifest := ManifestPath(opts.HomeDir)
	for _, a := range plan.Actions {
		kind, reason := a.Kind, a.Reason
		revalidated := false
		kept := a.ifClean && blocking > 0
		if kept {
			kind, reason = ActionKeep, reasonManifestKept
		} else if a.apply != nil {
			if err := a.apply(); err != nil {
				switch {
				case errors.Is(err, errChangedAfterPlan):
					kind, reason, revalidated = ActionSkip, reasonChanged, true
					changed++
				case errors.Is(err, errGoneAfterPlan) || isAbsent(err):
					kind, reason, revalidated = ActionSkip, reasonGone, true
				default:
					kind, reason = ActionFail, err.Error()
				}
			}
		}
		if kind == ActionFail && a.Target != manifest {
			blocking++
		}
		counts[kind]++
		record(kind, a.Target, reason)
		if !flags.Verbose && (kind == ActionFail || revalidated || kept) {
			fmt.Fprintf(w, "  [%s] %s — %s\n", kind, a.Target, reason)
		}
	}
	removeIfEmpty(SkillRepoDir(opts.HomeDir))

	if flags.Verbose {
		path := UninstallLogPath(opts.HomeDir)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err == nil && os.WriteFile(path, []byte(log.String()), 0644) == nil {
			fmt.Fprintf(w, "Log: %s\n", path)
		} else {
			fmt.Fprintf(w, "No se pudo escribir el log en %s\n", path)
		}
	}
	fmt.Fprintf(w, "Resumen: %d retirados, %d conservados, %d omitidos, %d fallidos.\n",
		counts[ActionRemove], counts[ActionKeep], counts[ActionSkip], counts[ActionFail])
	if counts[ActionFail] > 0 {
		return fmt.Errorf("%d acciones fallaron", counts[ActionFail])
	}
	if changed > 0 {
		fmt.Fprintf(w, "✓ pi-gomemory desinstalado; se conservaron %d elementos que cambiaron después del plan\n", changed)
		return nil
	}
	fmt.Fprintln(w, "✓ pi-gomemory desinstalado. Para reinstalar: pi-gomemory-setup install")
	return nil
}

// UninstallLogPath es el log de --verbose; vive junto a la marca de desinstalado.
func UninstallLogPath(home string) string {
	return filepath.Join(home, ".pi-gomemory", "uninstall.log")
}

func removeIfEmpty(dir string) {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
		_ = os.Remove(dir)
	}
}

var (
	confirmInput    io.Reader = os.Stdin
	stdinIsTerminal           = func() bool {
		fi, err := os.Stdin.Stat()
		return err == nil && fi.Mode()&os.ModeCharDevice != 0
	}
)

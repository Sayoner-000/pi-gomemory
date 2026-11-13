package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"
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
type UninstallAction struct {
	Kind   string
	Target string
	Reason string
	apply  func() error
}

// UninstallPlan es la lista ordenada de acciones; construirlo no escribe nada.
type UninstallPlan struct {
	Actions []UninstallAction
}

func (p *UninstallPlan) add(kind, target, reason string, apply func() error) {
	p.Actions = append(p.Actions, UninstallAction{Kind: kind, Target: target, Reason: reason, apply: apply})
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
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if want := ownedHash(m, path, fallback); want != "" && contentHash(data) == want {
		p.add(ActionRemove, path, "instalado por pi-gomemory", func() error { return os.Remove(path) })
		return ActionRemove
	}
	p.add(ActionSkip, path, "editado: no coincide con lo instalado", nil)
	return ActionSkip
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
func isRemovableEmbeddedCopy(dir string) bool {
	if !isPristineEmbeddedCopy(dir) {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		return false
	}
	for _, e := range entries {
		if e.Name() != "SKILL.md" && e.Name() != skillMarkerFile {
			return false
		}
	}
	return true
}

// planSkillDir planifica las skills de dir (repositorio central o ruta antigua de Pi).
func (p *UninstallPlan) planSkillDir(dir, home string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		switch {
		case e.Type()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err == nil && isHostSkillLink(target, home) {
				p.add(ActionRemove, path, "enlace a skill del anfitrión (no se sigue)", func() error { return os.Remove(path) })
			} else {
				p.add(ActionSkip, path, "enlace propio", nil)
			}
		case !e.IsDir():
			continue
		case isRemovableEmbeddedCopy(path):
			p.add(ActionRemove, path, "skill embebida intacta", func() error { return os.RemoveAll(path) })
		default:
			if _, err := os.Stat(filepath.Join(path, skillMarkerFile)); err == nil {
				p.add(ActionSkip, path, "skill embebida editada o con archivos añadidos", nil)
			} else {
				p.add(ActionSkip, path, "skill propia", nil)
			}
		}
	}
}

func sameJSON(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// editJSON relee path, aplica fn y lo escribe de forma atómica.
func editJSON(path string, fn func(map[string]json.RawMessage) error) error {
	obj, err := readJSONObject(path)
	if err != nil {
		return err
	}
	if err := fn(obj); err != nil {
		return err
	}
	return writeJSONObject(path, obj)
}

func (p *UninstallPlan) planMCP(piAgentDir string) {
	path := filepath.Join(piAgentDir, "mcp.json")
	if _, err := os.Stat(path); err != nil {
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
			return editJSON(path, func(o map[string]json.RawMessage) error {
				s := map[string]json.RawMessage{}
				if err := json.Unmarshal(o["mcpServers"], &s); err != nil {
					return err
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

func (p *UninstallPlan) planAgents(piAgentDir string) {
	path := filepath.Join(piAgentDir, "AGENTS.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if _, ok := stripInstructions(string(data)); !ok {
		return
	}
	p.add(ActionRemove, path+" › bloque pi-gomemory", "protocolo escrito por pi-gomemory", func() error {
		cur, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out, _ := stripInstructions(string(cur))
		if strings.TrimSpace(out) == "" {
			return os.Remove(path)
		}
		return writeFileAtomic(path, []byte(out), 0644)
	})
}

func (p *UninstallPlan) planSettings(piAgentDir, home, themeFile string) {
	path := filepath.Join(piAgentDir, "settings.json")
	if _, err := os.Stat(path); err != nil {
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
		if themeFile == ActionSkip {
			p.add(ActionKeep, target, "el archivo del tema está editado y se conserva", nil)
		} else {
			p.add(ActionRemove, target, "tema gomemory activado por pi-gomemory", func() error {
				return editJSON(path, func(o map[string]json.RawMessage) error {
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
				return editJSON(path, func(o map[string]json.RawMessage) error {
					var cur, out []string
					_ = json.Unmarshal(o["skills"], &cur)
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
	if _, err := os.Lstat(path); err == nil {
		p.add(ActionRemove, path, reason, func() error { return os.Remove(path) })
	}
}

// BuildUninstallPlan decide, sin escribir nada, qué se retira, conserva u omite.
func BuildUninstallPlan(assets fs.FS, opts *Options, flags UninstallFlags) (*UninstallPlan, error) {
	plan := &UninstallPlan{}
	m, err := ReadManifest(opts.HomeDir)
	if err != nil {
		plan.add(ActionFail, ManifestPath(opts.HomeDir), err.Error(), nil)
		m = nil
	}

	for _, r := range tuiResources {
		data, _ := fs.ReadFile(assets, r.source)
		plan.planOwnedFile(m, filepath.Join(opts.PiAgentDir, "extensions", r.destination), data)
	}
	themeData, _ := fs.ReadFile(assets, "themes/gomemory.json")
	themeFile := plan.planOwnedFile(m, filepath.Join(opts.PiAgentDir, "themes", "gomemory.json"), themeData)
	dirs, foreign := launcherDirs(m, opts)
	for _, dir := range dirs {
		plan.planOwnedFile(m, filepath.Join(dir, "pi-gomemory"), []byte(GenerateHarnessScript()))
	}
	if foreign != "" {
		path := filepath.Join(foreign, "pi-gomemory")
		if _, err := os.Lstat(path); err == nil {
			plan.add(ActionSkip, path, "fuera de HOME y sin manifiesto: retíralo a mano o usa PI_GOMEMORY_BIN_DIR", nil)
		}
	}

	plan.planSkillDir(SkillRepoDir(opts.HomeDir), opts.HomeDir)
	plan.planSkillDir(filepath.Join(opts.PiAgentDir, "skills"), opts.HomeDir)
	plan.planMCP(opts.PiAgentDir)
	plan.planAgents(opts.PiAgentDir)
	plan.planSettings(opts.PiAgentDir, opts.HomeDir, themeFile)

	plan.planPlainFile(DefaultDepsCachePath(opts.PiAgentDir), "caché de dependencias")
	plan.planPlainFile(DefaultUpdateCachePath(opts.PiAgentDir), "caché de actualizaciones")
	plan.planPlainFile(ManifestPath(opts.HomeDir), "manifiesto de instalación")
	if flags.Packages {
		plan.planPackages(assets, opts.PiAgentDir)
	}
	if flags.Self {
		plan.planSelf(m, opts)
	}
	return plan, nil
}

// planPackages retira solo las entradas con la fuente fijada exacta y
// pi-cc-extensions.json si coincide con los valores por defecto embebidos.
func (p *UninstallPlan) planPackages(assets fs.FS, piAgentDir string) {
	settings := filepath.Join(piAgentDir, "settings.json")
	if obj, err := readJSONObject(settings); err == nil {
		var pkgs []string
		_ = json.Unmarshal(obj["packages"], &pkgs)
		for _, pp := range pinnedPackages {
			for _, s := range pkgs {
				if s != pp.source {
					continue
				}
				src := pp.source
				p.add(ActionRemove, settings+" › packages."+pp.name, "paquete fijado por pi-gomemory", func() error {
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

	cfg := filepath.Join(piAgentDir, "pi-cc-extensions.json")
	cur, err := os.ReadFile(cfg)
	if err != nil {
		return
	}
	defaults, _ := fs.ReadFile(assets, "config/pi-cc-extensions.json")
	if sameJSON(cur, defaults) {
		p.add(ActionRemove, cfg, "configuración por defecto de pi-cc-extensions", func() error { return os.Remove(cfg) })
	} else {
		p.add(ActionSkip, cfg, "configuración de pi-cc-extensions personalizada", nil)
	}
}

// planSelf borra el ejecutable en curso sin resolver enlaces y solo si está en el BIN_DIR efectivo.
func (p *UninstallPlan) planSelf(m *Manifest, opts *Options) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if base := filepath.Base(exe); !strings.HasPrefix(base, "pi-gomemory-setup") {
		// En tests el ejecutable es el binario de go test: se usa el del BIN_DIR.
		exe = filepath.Join(opts.LocalBinDir, "pi-gomemory-setup")
	}
	if _, err := os.Lstat(exe); err != nil {
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
	for _, a := range plan.Actions {
		kind, reason := a.Kind, a.Reason
		if a.apply != nil {
			if err := a.apply(); err != nil {
				kind, reason = ActionFail, err.Error()
			}
		}
		counts[kind]++
		record(kind, a.Target, reason)
		if kind == ActionFail && !flags.Verbose {
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

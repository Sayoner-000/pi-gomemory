package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HCD-076: `pi-gomemory-setup sync` reconcilia los tool del catálogo: instala
// lo que falta por su canal oficial y actualiza lo desactualizado con su
// comando oficial, sin preguntar. Es una acción explícita: refresca a la
// fuerza aunque PI_GOMEMORY_NO_UPDATE_CHECK esté puesta (como update,
// HCD-035); PI_GOMEMORY_SKIP_DEPS lo omite entero.

// SyncOptions son las opciones de sync.
type SyncOptions struct {
	Yes  bool
	JSON bool
	Only []string
	Skip []string
}

// NonInteractive indica que las opciones evitan el selector.
func (s SyncOptions) NonInteractive() bool {
	return s.Yes || len(s.Only) > 0 || len(s.Skip) > 0
}

// SyncResult es lo que sync hizo con un tool.
type SyncResult struct {
	Name   string `json:"name"`
	Action string `json:"action"` // installed, updated, current, failed o skipped
	From   string `json:"from"`
	To     string `json:"to"`
	Error  string `json:"error,omitempty"`
}

// SyncUsageError es un uso incorrecto de sync: termina con código 2.
type SyncUsageError struct{ msg string }

func (e *SyncUsageError) Error() string { return e.msg }

const syncUsage = "Uso: pi-gomemory-setup sync [--yes] [--json] [--only=a,b | --skip=c]"

// ParseSyncArgs interpreta los argumentos de sync.
func ParseSyncArgs(args []string) (SyncOptions, error) {
	var so SyncOptions
	for _, a := range args {
		switch {
		case a == "--yes" || a == "-y":
			so.Yes = true
		case a == "--json":
			so.JSON = true
		case strings.HasPrefix(a, "--only="):
			names := splitNames(strings.TrimPrefix(a, "--only="))
			if len(names) == 0 {
				return so, &SyncUsageError{"--only requiere al menos un componente\n" + syncUsage}
			}
			so.Only = names
		case strings.HasPrefix(a, "--skip="):
			names := splitNames(strings.TrimPrefix(a, "--skip="))
			if len(names) == 0 {
				return so, &SyncUsageError{"--skip requiere al menos un componente\n" + syncUsage}
			}
			so.Skip = names
		default:
			return so, &SyncUsageError{fmt.Sprintf("opción desconocida para sync: %s\n%s", a, syncUsage)}
		}
	}
	if len(so.Only) > 0 && len(so.Skip) > 0 {
		return so, &SyncUsageError{"--only y --skip no se pueden combinar\n" + syncUsage}
	}
	kinds := map[string]ComponentKind{}
	for _, c := range Catalog() {
		kinds[c.Name] = c.Kind
	}
	for _, n := range append(append([]string{}, so.Only...), so.Skip...) {
		k, ok := kinds[n]
		switch {
		case !ok:
			return so, &SyncUsageError{fmt.Sprintf("componente desconocido: %s (tool del catálogo: %s)", n, ToolNamesText())}
		case k != KindTool:
			return so, &SyncUsageError{fmt.Sprintf("%s no es un tool: sync solo reconcilia %s (pi-gomemory se actualiza con update y los paquetes fijados con install)", n, ToolNamesText())}
		}
	}
	return so, nil
}

func splitNames(s string) []string {
	var out []string
	for _, n := range strings.Split(s, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// selectTools devuelve qué tool reconcilia sync por defecto: los obligatorios
// siempre; en lite, los opcionales solo si se piden con --only.
func selectTools(tools []Tool, so SyncOptions, lite bool) map[string]bool {
	sel := map[string]bool{}
	for _, t := range tools {
		on := !lite || t.Required
		if len(so.Only) > 0 {
			on = contains(so.Only, t.Name)
		}
		if len(so.Skip) > 0 && contains(so.Skip, t.Name) {
			on = false
		}
		sel[t.Name] = on || t.Required
	}
	return sel
}

// syncProbes son las sondas que usa sync; las pruebas las sustituyen.
var syncProbes = func(o *Options) []DepProbe { return DefaultDepProbes(Version, o.PiAgentDir) }

// nodeRuntime devuelve la versión de node ("" si falta) y si npm está.
var nodeRuntime = func() (string, bool) {
	var v string
	if out, err := exec.Command("node", "--version").Output(); err == nil {
		v = strings.TrimSpace(string(out))
	}
	_, err := exec.LookPath("npm")
	return v, err == nil
}

const minNodeVersion = "22.19.0"

// checkNode exige Node.js ≥ 22.19 y npm antes de instalar pi (HCD-076, C-03).
func checkNode() error {
	v, npm := nodeRuntime()
	_, ok := parseSemver(v)
	if !ok || IsNewer(minNodeVersion, v) || !npm {
		have := v
		if have == "" {
			have = "no instalado"
		}
		return fmt.Errorf("pi necesita Node.js 22.19 o superior y npm (Node.js: %s, npm: %v). Instálalos desde https://nodejs.org o con tu gestor de paquetes (brew install node, nvm install 22) y repite la instalación", have, npm)
	}
	return nil
}

// Sync reconcilia los tool del catálogo. in es la terminal del selector (nil
// sin terminal). Solo devuelve error si no se puede instalar un obligatorio
// que falta.
func Sync(opts *Options, so SyncOptions, in io.Reader, out io.Writer) ([]SyncResult, error) {
	if SkipDepsRequested() {
		fmt.Fprintf(out, "%s    · %s: no se comprueban %s%s\n", ColorMuted, skipDepsEnv, ToolNamesText(), ColorReset)
		return nil, nil
	}
	extendToolPath(opts)
	now := time.Now()
	probes := syncProbes(opts)
	cachePath := DefaultDepsCachePath(opts.PiAgentDir)
	// Se refresca sin escribir: si un obligatorio no se puede instalar, no se
	// crea nada en ~/.pi/agent (HCD-062 T6). La caché se escribe al final.
	states := ForceRefreshDeps(probes, "", now)
	byState := map[string]int{}
	for i, d := range states {
		byState[d.Name] = i
	}
	byProbe := map[string]DepProbe{}
	for _, p := range probes {
		byProbe[p.Name] = p
	}

	lite, err := GetLiteStatus(opts.HomeDir)
	if err != nil {
		return nil, err
	}
	sel := selectTools(installTools, so, lite)
	if !so.NonInteractive() && in != nil {
		var err error
		if sel, err = runSelector(installTools, sel, states, in, out); err != nil {
			return nil, err
		}
	}

	var results []SyncResult
	installed := false
	for _, t := range installTools {
		r := SyncResult{Name: t.Name}
		p := byProbe[t.Name]
		switch {
		case !sel[t.Name]:
			r.Action = "skipped"
		case !toolPresent(t.Name):
			fmt.Fprintf(out, "%s    → instalando %s desde su canal oficial…%s\n", ColorMuted, t.Name, ColorReset)
			err := error(nil)
			if t.Name == "pi" {
				err = checkNode()
			}
			if err == nil {
				err = installTool(t, opts, out)
			}
			if err == nil && !toolPresent(t.Name) {
				err = fmt.Errorf("el instalador terminó pero %s no aparece en %s", t.Name, strings.Join(knownToolDirs(opts), ", "))
			}
			if err != nil {
				r.Action, r.Error = "failed", err.Error()
				if t.Required {
					results = append(results, r)
					hint := ""
					if t.Hint != "" {
						hint = " (" + t.Hint + ")"
					}
					return results, fmt.Errorf("no se pudo instalar %s: %w%s. Instálalo a mano con: %s y repite la instalación", t.Name, err, hint, t.Manual)
				}
				fmt.Fprintf(out, "%s    ⚠ no se pudo instalar %s (%v). Instálalo a mano con: %s%s\n", ColorWarning, t.Name, err, t.Manual, ColorReset)
				break
			}
			installed = true
			r.Action, r.To = "installed", liveVersion(p)
			fmt.Fprintf(out, "%s    ✓ %s instalado %s%s\n", ColorMuted, t.Name, r.To, ColorReset)
		default:
			var d DepState
			if i, ok := byState[t.Name]; ok {
				d = effectiveDepState(states[i], p)
			} else {
				d = effectiveDepState(DepState{Name: t.Name}, p)
			}
			r.From = d.Current
			if !d.Outdated() {
				r.Action, r.To = "current", d.Current
				fmt.Fprintf(out, "%s    ✓ %s al día (%s)%s\n", ColorMuted, t.Name, d.Current, ColorReset)
				break
			}
			fmt.Fprintf(out, "%s    → actualizando %s (%s → %s)…%s\n", ColorMuted, t.Name, d.Current, d.Latest, ColorReset)
			if err := updateTool(t, p, opts, out); err != nil {
				fmt.Fprintf(out, "%s    ⚠ no se pudo actualizar %s (%v). Actualízalo a mano con: %s, o reinstálalo con: %s%s\n",
					ColorWarning, t.Name, err, strings.Join(p.Update, " "), t.Manual, ColorReset)
				r.Action, r.Error = "failed", err.Error()
				break
			}
			r.Action, r.To = "updated", liveVersion(p)
			fmt.Fprintf(out, "%s    ✓ %s actualizado a %s%s\n", ColorMuted, t.Name, r.To, ColorReset)
		}
		results = append(results, r)
		if r.To != "" {
			if i, ok := byState[t.Name]; ok {
				states[i].Current, states[i].Error = r.To, ""
			}
		}
	}
	saveDepsCache(cachePath, now, states)
	if installed {
		fmt.Fprintf(out, "%s    · Los instaladores oficiales pueden añadir su directorio al PATH en el archivo de arranque de tu shell (~/.zshrc, ~/.bashrc).%s\n", ColorMuted, ColorReset)
	}
	return results, nil
}

// openTTY abre la terminal para el selector, o nil si no hay. La entrada
// estándar no se usa: con `curl | bash` es el propio script (HCD-062).
var openTTY = func() io.ReadCloser {
	if hooks.TTY == nil || !hooks.TTY() {
		return nil
	}
	f, err := os.Open("/dev/tty")
	if err != nil {
		return nil
	}
	return f
}

// syncTerminal devuelve la terminal del selector si las opciones lo permiten.
func syncTerminal(so SyncOptions) (io.Reader, func()) {
	if so.NonInteractive() {
		return nil, func() {}
	}
	if f := openTTY(); f != nil {
		return f, func() { _ = f.Close() }
	}
	return nil, func() {}
}

// RunSync implementa `pi-gomemory-setup sync` y devuelve el código de salida:
// 2 por uso incorrecto, 1 si no se pudo instalar un obligatorio. Con --json
// el resultado va a stdout y el progreso a stderr.
func RunSync(opts *Options, args []string, stdout, stderr io.Writer) int {
	so, err := ParseSyncArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	progress := stdout
	if so.JSON {
		progress = stderr
	}
	in, done := syncTerminal(so)
	defer done()
	res, err := Sync(opts, so, in, progress)
	if so.JSON {
		if res == nil {
			res = []SyncResult{}
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(stdout, string(data))
	}
	if err != nil {
		fmt.Fprintf(stderr, "%s✗ %v%s\n", ColorError, err, ColorReset)
		return 1
	}
	return 0
}

// updateTool actualiza un tool presente con su comando oficial, sin entrada
// y con la salida por tubería. codebase-memory-mcp se reinstala desde su
// release verificado (HCD-062).
func updateTool(t Tool, p DepProbe, opts *Options, out io.Writer) error {
	if t.Release {
		return installTool(t, opts, out)
	}
	if len(p.Update) == 0 {
		return errors.New("sin comando de actualización")
	}
	var env []string
	if t.Env != nil {
		env = t.Env(opts)
	}
	return runInstaller(p.Update[0], p.Update[1:], env, "", out, hooks.ScriptTimeout)
}

func saveDepsCache(cachePath string, now time.Time, states []DepState) {
	if cachePath == "" {
		return
	}
	if data, err := json.MarshalIndent(depsCache{CheckedAt: now, Deps: states}, "", "  "); err == nil {
		_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
		_ = writeFileAtomic(cachePath, append(data, '\n'), 0o644)
	}
}

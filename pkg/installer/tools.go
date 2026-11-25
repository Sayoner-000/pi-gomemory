package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HCD-062: pi-gomemory instala las herramientas que necesita (pi, mem,
// codegraph y codebase-memory-mcp) desde sus canales oficiales cuando faltan,
// para que el instalador de una línea deje el harness completo.

// skipDepsEnv omite el paso; sigue la regla de PI_GOMEMORY_NO_UPDATE_CHECK.
const skipDepsEnv = "PI_GOMEMORY_SKIP_DEPS"

// Canales oficiales, observados el 2026-11-20 (HCD-062, «Evidencia de los
// canales»).
const (
	PiInstallScriptURL        = "https://pi.dev/install.sh"
	MemInstallScriptURL       = "https://raw.githubusercontent.com/Sayoner-000/gomemory/main/scripts/install.sh"
	CodegraphInstallScriptURL = "https://raw.githubusercontent.com/colbymchenry/codegraph/main/install.sh"
)

// Tool describe una herramienta y su canal oficial. Una herramienta con
// Release se instala desde el release verificado; las demás, por su script.
type Tool struct {
	Name     string
	Required bool
	Script   string
	Interp   string
	Env      func(o *Options) []string
	Manual   string
	// Hint es el requisito previo que se recuerda si la instalación falla.
	Hint    string
	Release bool
}

func memEnv(o *Options) []string { return []string{"GOMEMORY_BIN_DIR=" + o.LocalBinDir} }

// piEnv fija el directorio de agente de esta instalación: dentro de una
// sesión de Pi el entorno trae el de la instalación en uso.
func piEnv(o *Options) []string { return []string{"PI_CODING_AGENT_DIR=" + o.PiAgentDir} }

func codegraphEnv(o *Options) []string { return []string{"CODEGRAPH_BIN_DIR=" + o.LocalBinDir} }

// installTools es la lista del paso, en orden de instalación. Es variable
// para que las pruebas la sustituyan.
var installTools = productionTools()

// productionTools deriva del catálogo los componentes de tipo tool (HCD-076).
func productionTools() []Tool {
	var out []Tool
	for _, c := range catalogTools() {
		out = append(out, Tool{Name: c.Name, Required: c.Required, Script: c.Script, Interp: c.Interp, Env: c.Env, Manual: c.Manual, Hint: c.Hint, Release: c.Release})
	}
	return out
}

// toolHooks son los puntos de sustitución del paso (HCD-062, «Pruebas»).
type toolHooks struct {
	// Client es el cliente HTTP base; se endurece con hardenedClient.
	Client *http.Client
	// TTY informa si hay una terminal controladora (/dev/tty se puede abrir).
	TTY             func() bool
	ScriptTimeout   time.Duration
	DownloadTimeout time.Duration
	GOOS, GOARCH    string
	// SystemDirs son los directorios del sistema de la lista conocida.
	SystemDirs []string
}

func defaultToolHooks() toolHooks {
	return toolHooks{
		Client:          &http.Client{},
		TTY:             ttyAvailable,
		ScriptTimeout:   10 * time.Minute,
		DownloadTimeout: 2 * time.Minute,
		SystemDirs:      []string{"/usr/local/bin"},
	}
}

var hooks = defaultToolHooks()

func ttyAvailable() bool {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

// SkipDepsRequested aplica a PI_GOMEMORY_SKIP_DEPS la regla de
// PI_GOMEMORY_NO_UPDATE_CHECK: vacía, 0 o false dejan el paso activo.
func SkipDepsRequested() bool {
	v := strings.TrimSpace(os.Getenv(skipDepsEnv))
	return v != "" && v != "0" && !strings.EqualFold(v, "false")
}

// knownToolDirs son los directorios donde quedan las herramientas, en orden
// de preferencia (HCD-062, «PATH»).
func knownToolDirs(opts *Options) []string {
	return append([]string{
		opts.LocalBinDir,
		filepath.Join(opts.HomeDir, ".local", "bin"),
		filepath.Join(opts.PiAgentDir, "bin"),
		filepath.Join(opts.HomeDir, ".gomemory", "bin"),
	}, hooks.SystemDirs...)
}

// extendToolPath crea el directorio de binarios y antepone al PATH del
// proceso los directorios conocidos que aún no estén, sin duplicar.
func extendToolPath(opts *Options) {
	_ = os.MkdirAll(opts.LocalBinDir, 0o755)
	prependToolDirs(opts)
}

// PrependToolDirs antepone al PATH del proceso los directorios conocidos de
// herramientas que aún no estén, sin crear directorios en disco (HCD-077, C-001).
func PrependToolDirs(opts *Options) { prependToolDirs(opts) }

// prependToolDirs es extendToolPath sin crear nada: la usa doctor y components.
func prependToolDirs(opts *Options) {
	current := filepath.SplitList(os.Getenv("PATH"))
	have := map[string]bool{}
	for _, d := range current {
		have[filepath.Clean(d)] = true
	}
	var add []string
	for _, d := range knownToolDirs(opts) {
		if d == "" || have[filepath.Clean(d)] {
			continue
		}
		have[filepath.Clean(d)] = true
		add = append(add, d)
	}
	if len(add) > 0 {
		_ = os.Setenv("PATH", strings.Join(append(add, current...), string(os.PathListSeparator)))
	}
}

func toolPresent(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// EnsureTools instala desde su canal oficial las herramientas que faltan.
// pi y mem son obligatorias: si fallan se devuelve un error con el comando
// manual. codegraph y herdr son opcionales: se avisa y se sigue; con el modo
// lite no se comprueban.
func EnsureTools(opts *Options, out io.Writer) error {
	if SkipDepsRequested() {
		fmt.Fprintf(out, "%s    · %s: no se comprueban %s%s\n", ColorMuted, skipDepsEnv, ToolNamesText(), ColorReset)
		return nil
	}
	extendToolPath(opts)
	lite, err := GetLiteStatus(opts.HomeDir)
	if err != nil {
		return err
	}
	installed := false
	for _, t := range installTools {
		if lite && !t.Required {
			continue
		}
		if toolPresent(t.Name) {
			fmt.Fprintf(out, "%s    ✓ %s ya está instalado%s\n", ColorMuted, t.Name, ColorReset)
			continue
		}
		fmt.Fprintf(out, "%s    → instalando %s desde su canal oficial…%s\n", ColorMuted, t.Name, ColorReset)
		err := installTool(t, opts, out)
		if err == nil && !toolPresent(t.Name) {
			err = fmt.Errorf("el instalador terminó pero %s no aparece en %s", t.Name, strings.Join(knownToolDirs(opts), ", "))
		}
		if err != nil {
			if t.Required {
				hint := ""
				if t.Hint != "" {
					hint = " (" + t.Hint + ")"
				}
				return fmt.Errorf("no se pudo instalar %s: %w%s. Instálalo a mano con: %s y repite la instalación", t.Name, err, hint, t.Manual)
			}
			fmt.Fprintf(out, "%s    ⚠ no se pudo instalar %s (%v). Instálalo a mano con: %s%s\n", ColorWarning, t.Name, err, t.Manual, ColorReset)
			continue
		}
		installed = true
		fmt.Fprintf(out, "%s    ✓ %s instalado%s\n", ColorMuted, t.Name, ColorReset)
	}
	if installed {
		fmt.Fprintf(out, "%s    · Los instaladores oficiales pueden añadir su directorio al PATH en el archivo de arranque de tu shell (~/.zshrc, ~/.bashrc).%s\n", ColorMuted, ColorReset)
	}
	return nil
}

// RunDepsInstall informa que codebase-memory-mcp fue retirado y las herramientas
// actuales se actualizan por su propio canal oficial.
func RunDepsInstall(_ *Options, _ []string, _, stderr io.Writer) int {
	fmt.Fprintln(stderr, "codebase-memory-mcp fue retirado; CodeGraph es el único grafo externo.")
	fmt.Fprintf(stderr, "%s se actualizan con su propio comando (%s).\n", scriptToolNamesText(), scriptToolUpdates())
	return 2
}

func installTool(t Tool, opts *Options, out io.Writer) error {
	return installByScript(t, opts, out)
}

// installByScript descarga el script oficial a un temporal privado junto al
// directorio de binarios y lo ejecuta con su intérprete.
func installByScript(t Tool, opts *Options, out io.Writer) error {
	tmp, cleanup, err := depsTempDir(opts)
	if err != nil {
		return err
	}
	defer cleanup()
	body, err := fetchHTTPS(hardenedClient(hooks.Client, hooks.DownloadTimeout), t.Script, maxScriptBytes)
	if err != nil {
		return fmt.Errorf("descargar %s: %w", t.Script, err)
	}
	script := filepath.Join(tmp, "install.sh")
	if err := writeFileAtomic(script, body, 0o600); err != nil {
		return err
	}
	var env []string
	if t.Env != nil {
		env = t.Env(opts)
	}
	return runInstaller(t.Interp, []string{script}, env, tmp, out, hooks.ScriptTimeout)
}

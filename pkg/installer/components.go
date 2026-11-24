package installer

import (
	"context"
	"regexp"
	"strings"
)

// HCD-076: catálogo único de componentes del meta-harness. De él se derivan
// las herramientas que instala el paso [1/9] (productionTools), las sondas de
// versión (DefaultDepProbes), los binarios de doctor, lo que uninstall
// conserva y `pi-gomemory-setup components`.

// ComponentKind clasifica un componente del catálogo.
type ComponentKind string

const (
	// KindSelf es pi-gomemory: se actualiza con `pi-gomemory-setup update`.
	KindSelf ComponentKind = "self"
	// KindTool es un binario externo que sync instala y actualiza.
	KindTool ComponentKind = "tool"
	// KindPackage es un paquete de Pi fijado por pi-gomemory (HCD-039).
	KindPackage ComponentKind = "package"
)

// HerdrInstallScriptURL es el canal oficial de herdr, observado el
// 2026-11-24 (HCD-076, «Evidencia de los canales»): responde HTTP 200, es
// `#!/bin/sh`, verifica SHA-256 y acepta HERDR_INSTALL_DIR. La última versión
// sale de la redirección de HerdrReleasesLatestURL y `herdr update` solo
// admite --handoff.
const (
	HerdrInstallScriptURL  = "https://herdr.dev/install.sh"
	HerdrReleasesLatestURL = "https://github.com/herdrdev/herdr/releases/latest"
)

// Component describe un componente del meta-harness.
type Component struct {
	Name     string
	Kind     ComponentKind
	Required bool
	// Canal de instalación de un tool: script oficial o release verificado.
	Script  string
	Interp  string
	Env     func(o *Options) []string
	Release bool
	// Manual es la instalación a mano; Hint, el requisito previo.
	Manual string
	Hint   string
	// Uninstall es cómo retirar un tool, que uninstall conserva.
	Uninstall string
	// probe construye la sonda de versión del componente.
	probe func(version, piAgentDir string) DepProbe
}

func herdrEnv(o *Options) []string { return []string{"HERDR_INSTALL_DIR=" + o.LocalBinDir} }

// Catalog devuelve los componentes en su orden canónico.
func Catalog() []Component {
	return []Component{
		{
			Name: "pi-gomemory", Kind: KindSelf, Required: true,
			probe: func(version, _ string) DepProbe {
				return DepProbe{
					Current: func(context.Context) (string, error) { return version, nil },
					Latest:  githubLatest(DefaultReleasesAPI),
					Update:  []string{"pi-gomemory-setup", "update"},
				}
			},
		},
		{
			Name: "pi", Kind: KindTool, Required: true,
			Script: PiInstallScriptURL, Interp: "sh", Env: piEnv,
			Manual:    "curl -fsSL " + PiInstallScriptURL + " | sh",
			Hint:      "pi necesita Node.js 22.19 o superior y npm: instálalos si faltan",
			Uninstall: "rm -rf ~/.pi/agent/install ~/.pi/agent/bin/pi (y su enlace en ~/.local/bin)",
			probe: func(string, string) DepProbe {
				return DepProbe{
					Current: versionFromCmd("pi", "--version"),
					Latest:  npmLatest("@earendil-works/pi-coding-agent"),
					Update:  []string{"pi", "update", "--self"},
				}
			},
		},
		{
			Name: "mem", Kind: KindTool, Required: true,
			Script: MemInstallScriptURL, Interp: "bash", Env: memEnv,
			Manual:    "curl -fsSL " + MemInstallScriptURL + " | bash",
			Uninstall: "curl -fsSL " + MemInstallScriptURL + " | bash -s -- --uninstall",
			probe: func(string, string) DepProbe {
				return DepProbe{
					Current: versionFromCmd("mem", "version"),
					Latest:  latestFromCmd(regexp.MustCompile(`Disponible:\s*v?(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)`), "mem", "update", "-check"),
					Update:  []string{"mem", "update", "-y"},
				}
			},
		},
		{
			Name: "codegraph", Kind: KindTool,
			Script: CodegraphInstallScriptURL, Interp: "sh", Env: codegraphEnv,
			Manual:    "curl -fsSL " + CodegraphInstallScriptURL + " | sh",
			Uninstall: "curl -fsSL " + CodegraphInstallScriptURL + " | sh -s -- --uninstall",
			probe: func(string, string) DepProbe {
				return DepProbe{
					Current: versionFromCmd("codegraph", "--version"),
					Latest:  latestFromCmd(regexp.MustCompile(`latest\s+v?(\d+\.\d+\.\d+[0-9A-Za-z.+-]*)`), "codegraph", "upgrade", "--check"),
					Update:  []string{"codegraph", "upgrade"},
				}
			},
		},
		{
			Name: "herdr", Kind: KindTool,
			Script: HerdrInstallScriptURL, Interp: "sh", Env: herdrEnv,
			Manual:    "curl -fsSL " + HerdrInstallScriptURL + " | sh",
			Uninstall: `rm "$(command -v herdr)" (herdr no tiene subcomando de desinstalación)`,
			probe: func(string, string) DepProbe {
				return DepProbe{
					Current: versionFromCmd("herdr", "--version"),
					Latest:  githubLatest(HerdrReleasesLatestURL),
					Update:  []string{"herdr", "update"},
				}
			},
		},
		{
			Name: "pi-antigravity", Kind: KindPackage,
			probe: func(_ string, piAgentDir string) DepProbe {
				return DepProbe{
					Current:   npmPackageVersion(piAgentDir, "pi-antigravity"),
					Latest:    fixedVersion(AntigravityVersion),
					Update:    []string{"pi", "install", AntigravityPackageSource},
					Published: npmPublished("pi-antigravity", AntigravityVersion),
				}
			},
		},
		{
			Name: "pi-cc-extensions", Kind: KindPackage,
			probe: func(_ string, piAgentDir string) DepProbe {
				return DepProbe{
					Current:   npmPackageVersion(piAgentDir, "pi-cc-extensions"),
					Latest:    fixedVersion(CCExtensionsVersion),
					Update:    []string{"pi", "install", CCExtensionsSource},
					Published: npmPublished("pi-cc-extensions", CCExtensionsVersion),
				}
			},
		},
	}
}

// catalogTools devuelve los componentes de tipo tool, en orden.
func catalogTools() []Component {
	var out []Component
	for _, c := range Catalog() {
		if c.Kind == KindTool {
			out = append(out, c)
		}
	}
	return out
}

// ToolNames devuelve los nombres de los tool del catálogo, en orden.
func ToolNames() []string {
	var out []string
	for _, c := range catalogTools() {
		out = append(out, c.Name)
	}
	return out
}

// joinSpanish une nombres como «a, b y c».
func joinSpanish(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " y " + names[len(names)-1]
}

// ToolNamesText es la lista de tool para los mensajes: «pi, mem, … y herdr».
func ToolNamesText() string { return joinSpanish(ToolNames()) }

// scriptToolNamesText nombra los tool que se instalan por script (los que
// deps-install no gestiona).
func scriptToolNamesText() string {
	var out []string
	for _, c := range catalogTools() {
		if !c.Release {
			out = append(out, c.Name)
		}
	}
	return joinSpanish(out)
}

// scriptToolUpdates lista los comandos oficiales de actualización de los tool
// por script: «pi update --self, mem update -y, …».
func scriptToolUpdates() string {
	var out []string
	for _, c := range catalogTools() {
		if !c.Release {
			out = append(out, strings.Join(c.probe("", "").Update, " "))
		}
	}
	return strings.Join(out, ", ")
}

// InstallStepToolsText es el título del paso [1/9] de InstallAll.
func InstallStepToolsText() string {
	return "==> [1/9] Reconciliando " + ToolNamesText() + " (se instala lo que falte y se actualiza lo desactualizado desde su canal oficial)..."
}

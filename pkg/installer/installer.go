package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Options contiene los parámetros de instalación del harness.
type Options struct {
	HomeDir     string
	PiAgentDir  string
	LocalBinDir string
}

func DefaultOptions() (*Options, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("obtener home: %w", err)
	}

	binDir := filepath.Join(home, ".local", "bin")
	if v := os.Getenv("PI_GOMEMORY_BIN_DIR"); v != "" {
		binDir = v
	}
	return &Options{
		HomeDir:     home,
		PiAgentDir:  filepath.Join(home, ".pi", "agent"),
		LocalBinDir: binDir,
	}, nil
}

// InstallAll orquesta la instalación completa a partir de los recursos
// embebidos en el binario. No escribe nada en el directorio actual.
func InstallAll(assets fs.FS, opts *Options) error {
	fmt.Printf("%s==> [1/8] Registrando servidores MCP (gomemory, codegraph, codebase-memory)...%s\n", ColorCyan, ColorReset)
	if err := InstallMCPServers(opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso MCP: %w", err)
	}

	fmt.Printf("%s==> [2/8] Instalando paleta y tema gomemory en Pi...%s\n", ColorCyan, ColorReset)
	if err := InstallTheme(assets, opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso Tema: %w", err)
	}

	fmt.Printf("%s==> [3/8] Configurando plugin pi-antigravity para conexión con Google...%s\n", ColorCyan, ColorReset)
	if err := InstallAntigravityPlugin(opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso Plugin Antigravity: %w", err)
	}

	fmt.Printf("%s==> [4/8] Instalando paquete TUI Claude Code Style (pi-cc-extensions)...%s\n", ColorCyan, ColorReset)
	if err := InstallCCExtensions(assets, opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso Plugin CC Extensions: %w", err)
	}

	fmt.Printf("%s==> [5/8] Escribiendo instrucciones y protocolo de memoria en Pi...%s\n", ColorCyan, ColorReset)
	if err := InstallInstructions(opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso Instrucciones: %w", err)
	}

	fmt.Printf("%s==> [6/8] Reuniendo skills de gomemory y del anfitrión en %s...%s\n", ColorCyan, SkillRepoSetting, ColorReset)
	res, err := InstallSkills(assets, opts)
	if err != nil {
		return fmt.Errorf("paso Skills: %w", err)
	}
	fmt.Printf("%s    ✓ %d skills embebidas y %d enlazadas del anfitrión (%d colisiones resueltas)%s\n", ColorMuted, res.Embedded, res.Linked, len(res.Collisions), ColorReset)

	fmt.Printf("%s==> [7/8] Instalando extensión TUI Status Bar y Monitor de Tokens gomemory...%s\n", ColorCyan, ColorReset)
	if err := InstallTUIExtension(assets, opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso TUI Extension: %w", err)
	}

	fmt.Printf("%s==> [8/8] Generando harness launcher pi-gomemory para Herdr...%s\n", ColorCyan, ColorReset)
	if err := InstallHarness(opts.LocalBinDir); err != nil {
		return fmt.Errorf("paso Harness: %w", err)
	}

	fmt.Printf("\n%s✓ Instalación completa.%s\n", ColorBrillo, ColorReset)

	PrintDoctor(RunDoctor(opts))
	return nil
}

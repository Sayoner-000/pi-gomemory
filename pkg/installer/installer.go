package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Options contiene los parámetros de instalación del harness.
type Options struct {
	HomeDir        string
	PiAgentDir     string
	LocalBinDir    string
	PackageRootDir string
}

func DefaultOptions() (*Options, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("obtener home: %w", err)
	}

	cwd, _ := os.Getwd()

	return &Options{
		HomeDir:        home,
		PiAgentDir:     filepath.Join(home, ".pi", "agent"),
		LocalBinDir:    filepath.Join(home, ".local", "bin"),
		PackageRootDir: cwd,
	}, nil
}

// InstallAll orquesta el combo completo con una sola instrucción:
func InstallAll(embedFS fs.FS, opts *Options) error {
	fmt.Printf("%s==> [1/8] Registrando servidores MCP (gomemory, codegraph, codebase-memory)...%s\n", ColorCyan, ColorReset)
	if err := InstallMCPServers(opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso MCP: %w", err)
	}

	fmt.Printf("%s==> [2/8] Instalando paleta y tema gomemory en Pi...%s\n", ColorCyan, ColorReset)
	if err := InstallTheme(opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso Tema: %w", err)
	}

	fmt.Printf("%s==> [3/8] Configurando plugin pi-antigravity para conexión con Google...%s\n", ColorCyan, ColorReset)
	if err := InstallAntigravityPlugin(opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso Plugin Antigravity: %w", err)
	}

	fmt.Printf("%s==> [4/8] Instalando paquete TUI Claude Code Style (pi-cc-extensions)...%s\n", ColorCyan, ColorReset)
	if err := InstallCCExtensions(opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso Plugin CC Extensions: %w", err)
	}

	fmt.Printf("%s==> [5/8] Escribiendo instrucciones y protocolo de memoria en Pi...%s\n", ColorCyan, ColorReset)
	if err := InstallInstructions(opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso Instrucciones: %w", err)
	}

	fmt.Printf("%s==> [6/8] Instalando y sanitizando skills de gomemory en Pi...%s\n", ColorCyan, ColorReset)
	count, err := InstallSkills(embedFS, opts.PackageRootDir, opts.PiAgentDir)
	if err != nil {
		return fmt.Errorf("paso Skills: %w", err)
	}
	fmt.Printf("%s    ✓ %d skills instaladas y sanitizadas en %s%s\n", ColorMuted, count, filepath.Join(opts.PiAgentDir, "skills"), ColorReset)

	fmt.Printf("%s==> [7/8] Instalando extensión TUI Status Bar y Monitor de Tokens gomemory...%s\n", ColorCyan, ColorReset)
	if err := InstallTUIExtension(embedFS, opts.PackageRootDir, opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso TUI Extension: %w", err)
	}

	fmt.Printf("%s==> [8/8] Generando harness launcher pi-gomemory para Herdr...%s\n", ColorCyan, ColorReset)
	packageBin := filepath.Join(opts.PackageRootDir, "bin")
	if err := InstallHarness(opts.LocalBinDir, packageBin); err != nil {
		return fmt.Errorf("paso Harness: %w", err)
	}

	fmt.Printf("\n%s✓ Combo completo instalado exitosamente con una sola instrucción.%s\n", ColorBrillo, ColorReset)

	// Ejecutar diagnóstico
	items := RunDoctor(opts.PiAgentDir)
	PrintDoctor(items)

	return nil
}

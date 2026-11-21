package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Version es la versión del binario en curso; la fija main al arrancar.
var Version = "dev"

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
	// Las herramientas van primero: sin pi no se puede configurar nada, y si
	// una obligatoria falla no se toca ni la marca de desinstalado (HCD-062).
	fmt.Printf("%s==> [1/9] Comprobando pi, mem, codegraph y codebase-memory-mcp (se instala lo que falte desde su canal oficial)...%s\n", ColorCyan, ColorReset)
	RemoveStaleDepsTemps(opts.LocalBinDir)
	if err := EnsureTools(opts, os.Stdout); err != nil {
		return fmt.Errorf("paso Herramientas: %w", err)
	}

	if err := os.Remove(UninstalledMarkerPath(opts.HomeDir)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("retirar marca de desinstalado: %w", err)
	}
	// Modo MCP directo (codemode off) es el valor por defecto, salvo `codemode on` (HCD-050).
	fmt.Printf("%s==> [2/9] Registrando servidores MCP (gomemory, codegraph, codebase-memory) en modo directo...%s\n", ColorCyan, ColorReset)
	if err := applyCodemodeDefault(opts, os.Stdout); err != nil {
		return fmt.Errorf("paso MCP: %w", err)
	}

	fmt.Printf("%s==> [3/9] Instalando paleta y tema gomemory en Pi...%s\n", ColorCyan, ColorReset)
	if err := InstallTheme(assets, opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso Tema: %w", err)
	}

	fmt.Printf("%s==> [4/9] Configurando plugin pi-antigravity para conexión con Google...%s\n", ColorCyan, ColorReset)
	if err := InstallAntigravityPlugin(opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso Plugin Antigravity: %w", err)
	}

	fmt.Printf("%s==> [5/9] Instalando paquete TUI Claude Code Style (pi-cc-extensions)...%s\n", ColorCyan, ColorReset)
	if err := InstallCCExtensions(assets, opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso Plugin CC Extensions: %w", err)
	}

	fmt.Printf("%s==> [6/9] Reuniendo skills de gomemory y del anfitrión en %s...%s\n", ColorCyan, SkillRepoSetting, ColorReset)
	res, err := InstallSkills(assets, opts)
	if err != nil {
		return fmt.Errorf("paso Skills: %w", err)
	}
	fmt.Printf("%s    ✓ %d skills embebidas y %d enlazadas del anfitrión (%d colisiones resueltas)%s\n", ColorMuted, res.Embedded, res.Linked, len(res.Collisions), ColorReset)

	fmt.Printf("%s==> [7/9] Instalando extensión TUI Status Bar y Monitor de Tokens gomemory...%s\n", ColorCyan, ColorReset)
	if err := InstallTUIExtension(assets, opts.PiAgentDir); err != nil {
		return fmt.Errorf("paso TUI Extension: %w", err)
	}

	// El protocolo ya no se escribe en AGENTS.md: llega en vivo con la extensión
	// gomemory-context.ts (HCD-045). Se retira el bloque heredado solo después
	// de instalar ese reemplazo, para no perder ambos si falla (HCD-047, C-03).
	fmt.Printf("%s==> [8/9] Retirando el protocolo heredado de AGENTS.md (el contexto llega en vivo desde gomemory)...%s\n", ColorCyan, ColorReset)
	if err := retireLegacyStep(opts.PiAgentDir, os.Stdout); err != nil {
		return fmt.Errorf("paso Instrucciones: %w", err)
	}

	fmt.Printf("%s==> [9/9] Generando harness launcher pi-gomemory para Herdr...%s\n", ColorCyan, ColorReset)
	if err := InstallHarness(opts.LocalBinDir); err != nil {
		return fmt.Errorf("paso Harness: %w", err)
	}

	if err := WriteManifest(opts, Version); err != nil {
		return fmt.Errorf("paso Manifiesto: %w", err)
	}

	fmt.Printf("\n%s✓ Instalación completa.%s\n", ColorBrillo, ColorReset)

	PrintDoctor(RunDoctor(opts))
	return nil
}

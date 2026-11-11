package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"pi-gomemory/pkg/installer"
)

// version se inyecta al compilar con -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	opts, err := installer.DefaultOptions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error obteniendo rutas base: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 || os.Args[1] == "install" {
		if err := installer.InstallAll(EmbeddedAssets, opts); err != nil {
			fmt.Fprintf(os.Stderr, "\n%s✗ Falló la instalación: %v%s\n", installer.ColorError, err, installer.ColorReset)
			os.Exit(1)
		}
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "doctor", "status", "check":
		installer.PrintDoctor(installer.RunDoctor(opts))
	case "usage", "quota", "antigravity.usage":
		out, err := installer.FetchAndFormatUsage(opts.PiAgentDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError consultando cuota: %v%s\n", installer.ColorWarning, err, installer.ColorReset)
			os.Exit(1)
		}
		fmt.Print(out)
	case "theme":
		if err := installer.InstallTheme(EmbeddedAssets, opts.PiAgentDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error instalando tema: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Tema gomemory instalado y activado en Pi.")
	case "mcp":
		if err := installer.InstallMCPServers(opts.PiAgentDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error configurando MCP: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Servidores MCP gomemory, codegraph y codebase-memory registrados en Pi.")
	case "plugin":
		if err := installer.InstallAntigravityPlugin(opts.PiAgentDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error configurando plugin: %v\n", err)
			os.Exit(1)
		}
		if err := installer.InstallCCExtensions(EmbeddedAssets, opts.PiAgentDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error configurando pi-cc-extensions: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Plugins pi-antigravity y pi-cc-extensions configurados en Pi.")
	case "skills":
		count, err := installer.InstallSkills(EmbeddedAssets, opts.PiAgentDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error instalando skills: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ %d skills de gomemory instaladas en Pi.\n", count)
	case "harness":
		if err := installer.InstallHarness(opts.LocalBinDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generando harness launcher: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Launcher pi-gomemory instalado.")
	case "version", "-v", "--version":
		fmt.Println(version)
	case "check-update":
		res := installer.CheckUpdate(version, installer.UpdateCheckConfig{
			CachePath: installer.DefaultUpdateCachePath(opts.PiAgentDir),
		})
		if n := res.Notice(); n != "" {
			fmt.Fprintln(os.Stderr, n)
		}
	case "update":
		if err := runUpdate(opts); err != nil {
			fmt.Fprintf(os.Stderr, "%s✗ Falló la actualización: %v%s\n", installer.ColorError, err, installer.ColorReset)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Comando desconocido: %s\n\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf("%spi-gomemory-setup%s — Instalador de pi-gomemory para Pi Coding Agent\n\n", installer.ColorBrillo, installer.ColorReset)
	fmt.Println("Uso:")
	fmt.Println("  pi-gomemory-setup [install]   Instalación completa desde los recursos embebidos")
	fmt.Println("  pi-gomemory-setup usage       Cuota real de Google AI Pro / Antigravity")
	fmt.Println("  pi-gomemory-setup doctor      Diagnóstico de Pi, MCP, plugins, launcher y otros agentes")
	fmt.Println("  pi-gomemory-setup mcp         Registra los servidores MCP (gomemory, codegraph, codebase-memory)")
	fmt.Println("  pi-gomemory-setup theme       Instala y activa el tema gomemory en Pi")
	fmt.Println("  pi-gomemory-setup plugin      Configura pi-antigravity y pi-cc-extensions")
	fmt.Println("  pi-gomemory-setup skills      Instala las 21 skills de gomemory en Pi")
	fmt.Println("  pi-gomemory-setup harness     Instala el launcher pi-gomemory en ~/.local/bin")
	fmt.Println("  pi-gomemory-setup version     Muestra la versión instalada")
	fmt.Println("  pi-gomemory-setup check-update  Avisa si hay una release nueva (caché de 24 h)")
	fmt.Println("  pi-gomemory-setup update      Descarga la última release, verifica SHA256SUMS y reinstala")
}

// runUpdate reemplaza el binario por la última release verificada y relanza la instalación con él.
func runUpdate(opts *installer.Options) error {
	res := installer.CheckUpdate(version, installer.UpdateCheckConfig{Timeout: 10 * time.Second})
	if res.Err != nil {
		return fmt.Errorf("consultar la última release: %w", res.Err)
	}
	if !res.Newer {
		fmt.Printf("✓ pi-gomemory-setup ya está al día (%s).\n", version)
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	fmt.Printf("==> Actualizando %s → %s...\n", version, res.Latest)
	if err := installer.DownloadAndReplace(installer.UpdateConfig{Tag: res.Latest, Target: exe}); err != nil {
		return err
	}
	_ = os.Remove(installer.DefaultUpdateCachePath(opts.PiAgentDir))
	cmd := exec.Command(exe, "install")
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("reinstalar con %s: %w", res.Latest, err)
	}
	fmt.Printf("✓ pi-gomemory actualizado a %s.\n", res.Latest)
	return nil
}

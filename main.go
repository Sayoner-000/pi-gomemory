package main

import (
	"fmt"
	"os"

	"pi-gomemory/pkg/installer"
)

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
		items := installer.RunDoctor(opts.PiAgentDir)
		installer.PrintDoctor(items)
	case "usage", "quota", "antigravity.usage":
		out, err := installer.FetchAndFormatUsage(opts.PiAgentDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError consultando cuota: %v%s\n", installer.ColorWarning, err, installer.ColorReset)
		}
		fmt.Print(out)
	case "theme":
		if err := installer.InstallTheme(opts.PiAgentDir); err != nil {
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
		fmt.Println("✓ Plugin pi-antigravity (Google) configurado en Pi.")
	case "skills":
		count, err := installer.InstallSkills(EmbeddedAssets, opts.PackageRootDir, opts.PiAgentDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error instalando skills: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ %d skills de gomemory instaladas en Pi.\n", count)
	case "harness":
		if err := installer.InstallHarness(opts.LocalBinDir, "bin"); err != nil {
			fmt.Fprintf(os.Stderr, "Error generando harness launcher: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Launcher pi-gomemory instalado.")
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Comando desconocido: %s\n\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf("%spi-gomemory%s — Instalador y Harness para Pi Coding Agent con gomemory\n\n", installer.ColorBrillo, installer.ColorReset)
	fmt.Println("Uso:")
	fmt.Println("  pi-gomemory [install]   Instala el combo completo en una sola instrucción (autocontenido)")
	fmt.Println("  pi-gomemory usage       Muestra la barra lateral y cuota de Google AI Pro / Antigravity")
	fmt.Println("  pi-gomemory doctor      Diagnostica estado de Pi, MCPs, plugins y launcher")
	fmt.Println("  pi-gomemory mcp         Registra los servidores MCP (gomemory, codegraph, codebase-memory)")
	fmt.Println("  pi-gomemory theme       Instala y activa la paleta gomemory en Pi")
	fmt.Println("  pi-gomemory plugin      Configura los plugins pi-antigravity y pi-cc-extensions")
	fmt.Println("  pi-gomemory skills      Instala las 21 skills oficiales de gomemory en Pi")
	fmt.Println("  pi-gomemory harness     Instala el launcher ejecutable pi-gomemory para Herdr")
}

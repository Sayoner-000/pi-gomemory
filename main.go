package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"pi-gomemory/pkg/installer"
)

// version se inyecta al compilar con -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	installer.Version = version
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
		if len(os.Args) > 2 && os.Args[2] == "--json" {
			summary, err := installer.FetchQuotaSummary(opts.PiAgentDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "{\"error\": %q}\n", err.Error())
				os.Exit(1)
			}
			data, _ := json.MarshalIndent(summary, "", "  ")
			fmt.Println(string(data))
			return
		}
		out, err := installer.FetchAndFormatUsage(opts.PiAgentDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sError consultando cuota: %v%s\n", installer.ColorWarning, err, installer.ColorReset)
			os.Exit(1)
		}
		fmt.Print(out)
	case "claude-usage", "claude.usage":
		summary, err := installer.FetchClaudeUsage("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "{\"error\": %q}\n", err.Error())
			os.Exit(1)
		}
		data, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Println(string(data))
		return
	case "anthropic-usage":
		usage, err := installer.FetchAnthropicUsage(opts.PiAgentDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "{\"error\": %q}\n", err.Error())
			os.Exit(1)
		}
		data, _ := json.MarshalIndent(usage, "", "  ")
		fmt.Println(string(data))
		return
	case "codex-usage":
		usage, err := installer.FetchCodexUsage(opts.PiAgentDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "{\"error\": %q}\n", err.Error())
			os.Exit(1)
		}
		data, _ := json.MarshalIndent(usage, "", "  ")
		fmt.Println(string(data))
		return
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
	case "skills", "skills-sync":
		res, err := installer.InstallSkills(EmbeddedAssets, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error sincronizando skills: %v\n", err)
			os.Exit(1)
		}
		if cmd == "skills" {
			fmt.Printf("✓ %d skills de gomemory y %d del anfitrión en %s (%d colisiones resueltas).\n", res.Embedded, res.Linked, installer.SkillRepoSetting, len(res.Collisions))
			for _, c := range res.Collisions {
				fmt.Printf("    %s: gana %s, se omite %s\n", c.Name, c.Winner, c.Loser)
			}
		}
	case "harness":
		if err := installer.InstallHarness(opts.LocalBinDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generando harness launcher: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Launcher pi-gomemory instalado.")
	case "version", "-v", "--version":
		fmt.Println(version)
	case "deps-refresh":
		installer.RefreshDeps(installer.DefaultDepProbes(version, opts.PiAgentDir),
			installer.DefaultDepsCachePath(opts.PiAgentDir), time.Now())
	case "deps-status":
		if n := installer.FormatDepsNotice(installer.DepsStatusFor(installer.DefaultDepsCachePath(opts.PiAgentDir),
			installer.DefaultDepProbes(version, opts.PiAgentDir))); n != "" {
			fmt.Fprint(os.Stderr, n)
			os.Exit(10)
		}
	case "deps-update":
		if code := installer.RunDepsUpdate(installer.DefaultDepsCachePath(opts.PiAgentDir),
			installer.DefaultDepProbes(version, opts.PiAgentDir), os.Stdout, os.Stderr); code != 0 {
			os.Exit(code)
		}
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
	case "uninstall":
		flags := installer.UninstallFlags{}
		for _, a := range os.Args[2:] {
			switch a {
			case "--dry-run":
				flags.DryRun = true
			case "--yes", "-y":
				flags.Yes = true
			case "--verbose", "-v":
				flags.Verbose = true
			case "--packages":
				flags.Packages = true
			case "--self":
				flags.Self = true
			default:
				fmt.Fprintf(os.Stderr, "Opción desconocida para uninstall: %s\n", a)
				os.Exit(2)
			}
		}
		if err := installer.Uninstall(EmbeddedAssets, opts, flags, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "%s✗ %v%s\n", installer.ColorError, err, installer.ColorReset)
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
	fmt.Println("  pi-gomemory-setup anthropic-usage  Límites del plan de Anthropic (sesión y semana) en JSON")
	fmt.Println("  pi-gomemory-setup codex-usage      Límites del plan de OpenAI Codex (5 horas y semana) en JSON")
	fmt.Println("  pi-gomemory-setup doctor      Diagnóstico de Pi, MCP, plugins, launcher y otros agentes")
	fmt.Println("  pi-gomemory-setup mcp         Registra los servidores MCP (gomemory, codegraph, codebase-memory)")
	fmt.Println("  pi-gomemory-setup theme       Instala y activa el tema gomemory en Pi")
	fmt.Println("  pi-gomemory-setup plugin      Configura pi-antigravity y pi-cc-extensions")
	fmt.Println("  pi-gomemory-setup skills      Reúne las skills de gomemory y del anfitrión en ~/.pi-gomemory/skills")
	fmt.Println("  pi-gomemory-setup skills-sync Sincroniza el repositorio central y purga copias legadas")
	fmt.Println("  pi-gomemory-setup harness     Instala el launcher pi-gomemory en ~/.local/bin")
	fmt.Println("  pi-gomemory-setup version     Muestra la versión instalada")
	fmt.Println("  pi-gomemory-setup check-update  Avisa si hay una release nueva (caché de 24 h)")
	fmt.Println("  pi-gomemory-setup update      Descarga la última release, verifica SHA256SUMS y reinstala")
	fmt.Println("  pi-gomemory-setup deps-refresh  Consulta las últimas versiones de pi-gomemory y sus dependencias")
	fmt.Println("  pi-gomemory-setup deps-status   Muestra lo desactualizado según la última consulta (sale con 10)")
	fmt.Println("  pi-gomemory-setup deps-update   Actualiza los componentes desactualizados")
	fmt.Println("  pi-gomemory-setup uninstall [--dry-run] [--yes] [--verbose] [--packages] [--self]")
	fmt.Println("                                Retira lo instalado por pi-gomemory; conserva la memoria de gomemory")
}

// updateDeps son las dependencias externas de runUpdate. En producción son
// siempre las reales (defaultUpdateDeps); solo las pruebas de Go las sustituyen,
// sin variables de entorno ni opciones en el binario (HCD-035, Q8).
type updateDeps struct {
	executable func() (string, error)
	apiURL     string
	baseURL    string
	run        func(exe string, env []string) error
	out        io.Writer
}

func defaultUpdateDeps() updateDeps {
	return updateDeps{
		executable: os.Executable,
		apiURL:     installer.DefaultReleasesAPI,
		baseURL:    installer.DefaultReleasesBaseURL,
		run: func(exe string, env []string) error {
			cmd := exec.Command(exe, "install")
			cmd.Env = env
			cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
			return cmd.Run()
		},
		out: os.Stdout,
	}
}

// runUpdate reemplaza el binario por la última release verificada y relanza la instalación con él.
func runUpdate(opts *installer.Options) error {
	return runUpdateWith(opts, defaultUpdateDeps())
}

func runUpdateWith(opts *installer.Options, d updateDeps) error {
	res := installer.CheckUpdate(version, installer.UpdateCheckConfig{APIURL: d.apiURL, Timeout: 10 * time.Second, Force: true})
	if res.Err != nil {
		return fmt.Errorf("consultar la última release: %w", res.Err)
	}
	if !res.Newer {
		fmt.Fprintf(d.out, "✓ pi-gomemory-setup ya está al día (%s).\n", version)
		return nil
	}
	env, err := childInstallEnv(opts.HomeDir)
	if err != nil {
		return err
	}
	exe, err := d.executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	fmt.Fprintf(d.out, "==> Actualizando %s → %s...\n", version, res.Latest)
	if err := installer.DownloadAndReplace(installer.UpdateConfig{BaseURL: d.baseURL, Tag: res.Latest, Target: exe}); err != nil {
		return err
	}
	_ = os.Remove(installer.DefaultUpdateCachePath(opts.PiAgentDir))
	if err := d.run(exe, env); err != nil {
		return fmt.Errorf("reinstalar con %s: %w", res.Latest, err)
	}
	fmt.Fprintf(d.out, "✓ pi-gomemory actualizado a %s.\n", res.Latest)
	return nil
}

// childInstallEnv devuelve el entorno del install relanzado por update. Una
// PI_GOMEMORY_BIN_DIR no vacía gana y el manifiesto no se lee; si no, se pasa
// el bin_dir registrado. Sin manifiesto (o ENOTDIR) o con bin_dir vacío se
// usa el directorio por defecto; cualquier otro error de lectura impide
// actualizar antes de descargar nada (HCD-035, H-04, Q1, Q14).
func childInstallEnv(home string) ([]string, error) {
	env := os.Environ()
	if os.Getenv("PI_GOMEMORY_BIN_DIR") != "" {
		return env, nil
	}
	m, err := installer.ReadManifest(home)
	if errors.Is(err, syscall.ENOTDIR) {
		return env, nil
	}
	if err != nil {
		return nil, fmt.Errorf("leer el manifiesto de la instalación antes de actualizar: %w", err)
	}
	if m == nil || m.BinDir == "" {
		return env, nil
	}
	return append(env, "PI_GOMEMORY_BIN_DIR="+m.BinDir), nil
}

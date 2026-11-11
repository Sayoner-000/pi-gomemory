package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	ColorCyan    = "\033[38;2;6;193;238m"
	ColorBrillo  = "\033[38;2;34;227;247m"
	ColorWarning = "\033[38;2;244;198;107m"
	ColorError   = "\033[38;2;255;119;127m"
	ColorMuted   = "\033[38;2;145;169;197m"
	ColorReset   = "\033[0m"
)

type CheckItem struct {
	Name    string
	Status  string // OK, WARN, FAIL
	Details string
}

// otherAgents son los agentes que gomemory integra por su cuenta (`mem install`,
// `mem setup-mcp`). pi-gomemory no los configura: solo informa.
var otherAgents = []struct{ bin, setupName string }{
	{"claude", "claude"},
	{"codex", "codex"},
	{"agy", "agy"},
	{"opencode", "opencode"},
}

func RunDoctor(opts *Options) []CheckItem {
	piAgentDir := opts.PiAgentDir
	var items []CheckItem
	status, details := clipboardDiagnostic(runtime.GOOS, os.Getenv("DISPLAY"), os.Getenv("WAYLAND_DISPLAY"), exec.LookPath)
	items = append(items, CheckItem{Name: "Portapapeles del sistema", Status: status, Details: details})

	// 1. Binarios
	bins := []string{"pi", "mem", "codegraph", "codebase-memory-mcp"}
	for _, b := range bins {
		path, err := exec.LookPath(b)
		if err == nil {
			items = append(items, CheckItem{
				Name:    fmt.Sprintf("Binario %s", b),
				Status:  "OK",
				Details: path,
			})
		} else {
			items = append(items, CheckItem{
				Name:    fmt.Sprintf("Binario %s", b),
				Status:  "FAIL",
				Details: "no encontrado en PATH",
			})
		}
	}

	// 2. Tema
	themePath := filepath.Join(piAgentDir, "themes", "gomemory.json")
	if _, err := os.Stat(themePath); err == nil {
		items = append(items, CheckItem{
			Name:    "Tema gomemory en Pi",
			Status:  "OK",
			Details: themePath,
		})
	} else {
		items = append(items, CheckItem{
			Name:    "Tema gomemory en Pi",
			Status:  "FAIL",
			Details: "archivo de tema ausente",
		})
	}

	// 3. MCPs en Pi
	mcpListCmd := exec.Command("pi", "mcp", "list", "--json")
	out, err := mcpListCmd.Output()
	if err == nil {
		var listResp struct {
			Servers []struct {
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"servers"`
		}
		if jsonErr := json.Unmarshal(out, &listResp); jsonErr == nil {
			states := make(map[string]string)
			for _, s := range listResp.Servers {
				states[s.Name] = s.State
			}
			for _, req := range []string{"gomemory", "codegraph", "codebase-memory"} {
				st, found := states[req]
				if found && st == "connected" {
					items = append(items, CheckItem{
						Name:    fmt.Sprintf("MCP %s", req),
						Status:  "OK",
						Details: "conectado",
					})
				} else if found {
					items = append(items, CheckItem{
						Name:    fmt.Sprintf("MCP %s", req),
						Status:  "WARN",
						Details: fmt.Sprintf("estado: %s", st),
					})
				} else {
					items = append(items, CheckItem{
						Name:    fmt.Sprintf("MCP %s", req),
						Status:  "FAIL",
						Details: "no registrado en Pi",
					})
				}
			}
		}
	}

	// 4. Plugin pi-antigravity
	settingsPath := filepath.Join(piAgentDir, "settings.json")
	if data, err := os.ReadFile(settingsPath); err == nil {
		var s map[string]any
		_ = json.Unmarshal(data, &s)
		pkgs, _ := s["packages"].([]any)
		hasAntigravity := false
		for _, p := range pkgs {
			if str, ok := p.(string); ok && (str == AntigravityPackageSource || str == "npm:pi-antigravity") {
				hasAntigravity = true
				break
			}
		}
		if hasAntigravity {
			items = append(items, CheckItem{
				Name:    "Plugin pi-antigravity (Google)",
				Status:  "OK",
				Details: "registrado en settings.json",
			})
		} else {
			items = append(items, CheckItem{
				Name:    "Plugin pi-antigravity (Google)",
				Status:  "WARN",
				Details: "no encontrado en settings.json",
			})
		}
	}

	// 5. Plugin pi-cc-extensions
	cfgCC := filepath.Join(piAgentDir, "pi-cc-extensions.json")
	if _, err := os.Stat(cfgCC); err == nil {
		items = append(items, CheckItem{
			Name:    "Claude Code TUI (pi-cc-extensions)",
			Status:  "OK",
			Details: "activo (diffs, summaries, /context)",
		})
	} else {
		items = append(items, CheckItem{
			Name:    "Claude Code TUI (pi-cc-extensions)",
			Status:  "WARN",
			Details: "no configurado",
		})
	}

	// 6. Skills en Pi (propias, enlazadas o de las carpetas del anfitrión)
	skillCount := AvailableSkillCount(piAgentDir, opts.HomeDir)
	if skillCount >= 20 {
		items = append(items, CheckItem{
			Name:    "Skills gomemory en Pi",
			Status:  "OK",
			Details: fmt.Sprintf("%d skills cargadas", skillCount),
		})
	} else if skillCount > 0 {
		items = append(items, CheckItem{
			Name:    "Skills gomemory en Pi",
			Status:  "WARN",
			Details: fmt.Sprintf("%d skills encontradas (se esperan 21)", skillCount),
		})
	} else {
		items = append(items, CheckItem{
			Name:    "Skills gomemory en Pi",
			Status:  "FAIL",
			Details: "no se encontraron skills instaladas",
		})
	}

	for _, dir := range hostSkillDirs {
		if n, ok := HostSkillCounts(piAgentDir, opts.HomeDir)[dir]; ok {
			items = append(items, CheckItem{
				Name:    "Skills del anfitrión",
				Status:  "INFO",
				Details: fmt.Sprintf("%s · %d skills (solo lectura)", dir, n),
			})
		}
	}

	// 6. Extensión TUI Status Bar
	extPath := filepath.Join(piAgentDir, "extensions", "gomemory-status.ts")
	if _, err := os.Stat(extPath); err == nil {
		items = append(items, CheckItem{
			Name:    "TUI Status Bar & Tokens",
			Status:  "OK",
			Details: "extensión gomemory-status activa",
		})
	} else {
		items = append(items, CheckItem{
			Name:    "TUI Status Bar & Tokens",
			Status:  "WARN",
			Details: "extensión no instalada",
		})
	}

	// 7. Harness launcher
	launcherPath := filepath.Join(opts.LocalBinDir, "pi-gomemory")
	if info, err := os.Stat(launcherPath); err == nil && info.Mode()&0111 != 0 {
		items = append(items, CheckItem{
			Name:    "Launcher Herdr (pi-gomemory)",
			Status:  "OK",
			Details: launcherPath,
		})
	} else {
		items = append(items, CheckItem{
			Name:    "Launcher Herdr (pi-gomemory)",
			Status:  "WARN",
			Details: "no instalado en " + launcherPath,
		})
	}

	// 8. Otros agentes detectados: los integra gomemory, no este instalador
	for _, a := range otherAgents {
		if _, err := exec.LookPath(a.bin); err != nil {
			continue
		}
		items = append(items, CheckItem{
			Name:    fmt.Sprintf("Agente %s (vía gomemory)", a.bin),
			Status:  "INFO",
			Details: "mem setup-mcp --scope global --agents " + a.setupName,
		})
	}

	return items
}

func PrintDoctor(items []CheckItem) {
	fmt.Printf("\n%s╔════════════════════════════════════════════════════════════════════╗%s\n", ColorCyan, ColorReset)
	fmt.Printf("%s║           DIAGNÓSTICO DEL HARNESS PI-GOMEMORY                      ║%s\n", ColorBrillo, ColorReset)
	fmt.Printf("%s╚════════════════════════════════════════════════════════════════════╝%s\n\n", ColorCyan, ColorReset)

	for _, item := range items {
		var symbol, color string
		switch item.Status {
		case "OK":
			symbol = "✓"
			color = ColorBrillo
		case "WARN":
			symbol = "⚠"
			color = ColorWarning
		case "INFO":
			symbol = "→"
			color = ColorMuted
		default:
			symbol = "✗"
			color = ColorError
		}
		fmt.Printf("  %s%s%s %-32s %s%s%s\n", color, symbol, ColorReset, item.Name, ColorMuted, item.Details, ColorReset)
	}
	fmt.Println()
}

package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// CodemodeStatus describe el estado actual del modo codemode híbrido en Pi.
type CodemodeStatus struct {
	Enabled         bool `json:"enabled"`
	SettingsEnabled bool `json:"settings_enabled"`
	MCPEnabled      bool `json:"mcp_enabled"`
	Partial         bool `json:"partial"`
}

// SetCodemode activa (enabled=true) o desactiva (enabled=false) el modo
// codemode híbrido en ~/.pi/agent/settings.json y ~/.pi/agent/mcp.json.
func SetCodemode(piAgentDir string, enabled bool) error {
	if err := os.MkdirAll(piAgentDir, 0755); err != nil {
		return fmt.Errorf("crear %s: %w", piAgentDir, err)
	}
	if err := setCodemodeSettings(piAgentDir, enabled); err != nil {
		return err
	}
	servers := DefaultMCPServers()
	if enabled {
		servers = CodemodeMCPServers()
	}
	return writeMCPServers(piAgentDir, servers)
}

// GetCodemodeStatus inspecciona settings.json y mcp.json para informar si el
// modo codemode híbrido está activo, desactivado o parcialmente configurado.
func GetCodemodeStatus(piAgentDir string) (CodemodeStatus, error) {
	settingsOn, err := hasCodemodeInSettings(piAgentDir)
	if err != nil {
		return CodemodeStatus{}, err
	}
	mcpOn, err := hasCodemodeInMCP(piAgentDir)
	if err != nil {
		return CodemodeStatus{}, err
	}
	return CodemodeStatus{
		Enabled:         settingsOn && mcpOn,
		SettingsEnabled: settingsOn,
		MCPEnabled:      mcpOn,
		Partial:         settingsOn != mcpOn,
	}, nil
}

func setCodemodeSettings(piAgentDir string, enabled bool) error {
	settingsPath, err := configTarget(filepath.Join(piAgentDir, "settings.json"))
	if err != nil {
		return err
	}
	config, err := readJSONObject(settingsPath)
	if err != nil {
		return err
	}

	var tools []string
	if raw, ok := config["defaultTools"]; ok {
		if err := json.Unmarshal(raw, &tools); err != nil {
			return fmt.Errorf("%s: defaultTools no es una lista de cadenas: %w", settingsPath, err)
		}
	}

	if enabled {
		if !slices.Contains(tools, "+codemode") && !slices.Contains(tools, "codemode") {
			tools = append(tools, "+codemode")
		}
		if err := setJSONField(config, "defaultTools", tools); err != nil {
			return err
		}
	} else {
		filtered := make([]string, 0, len(tools))
		for _, t := range tools {
			if t == "+codemode" || t == "codemode" {
				continue
			}
			filtered = append(filtered, t)
		}
		if len(filtered) == 0 {
			delete(config, "defaultTools")
		} else if err := setJSONField(config, "defaultTools", filtered); err != nil {
			return err
		}
	}

	return writeJSONObject(settingsPath, config)
}

func hasCodemodeInSettings(piAgentDir string) (bool, error) {
	settingsPath, err := configTarget(filepath.Join(piAgentDir, "settings.json"))
	if err != nil {
		return false, err
	}
	config, err := readJSONObject(settingsPath)
	if err != nil {
		return false, err
	}
	raw, ok := config["defaultTools"]
	if !ok {
		return false, nil
	}
	var tools []string
	if err := json.Unmarshal(raw, &tools); err != nil {
		return false, fmt.Errorf("%s: defaultTools no es una lista de cadenas: %w", settingsPath, err)
	}
	return slices.Contains(tools, "+codemode") || slices.Contains(tools, "codemode"), nil
}

func hasCodemodeInMCP(piAgentDir string) (bool, error) {
	mcpPath, err := configTarget(filepath.Join(piAgentDir, "mcp.json"))
	if err != nil {
		return false, err
	}
	config, err := readJSONObject(mcpPath)
	if err != nil {
		return false, err
	}
	rawServers, ok := config["mcpServers"]
	if !ok {
		return false, nil
	}
	var servers map[string]MCPServerDef
	if err := json.Unmarshal(rawServers, &servers); err != nil {
		return false, fmt.Errorf("%s: mcpServers no es un objeto válido: %w", mcpPath, err)
	}
	return servers["gomemory"].Exposure == "codemode" && servers["codebase-memory"].Exposure == "codemode", nil
}

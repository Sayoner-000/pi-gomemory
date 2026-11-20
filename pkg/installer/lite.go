package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type liteState struct {
	Enabled bool `json:"enabled"`
}

// LiteStatePath devuelve ~/.pi-gomemory/lite.json.
func LiteStatePath(homeDir string) string {
	return filepath.Join(homeDir, ".pi-gomemory", "lite.json")
}

// IsLiteMode indica si el modo liviano está activo en ~/.pi-gomemory/lite.json.
func IsLiteMode(homeDir string) bool {
	data, err := os.ReadFile(LiteStatePath(homeDir))
	if err != nil {
		return false
	}
	var st liteState
	if err := json.Unmarshal(data, &st); err != nil {
		return false
	}
	return st.Enabled
}

// LiteMCPServers devuelve solo gomemory en mcp.json (sin codegraph ni codebase-memory)
// para reducir el consumo de RAM cuando corren múltiples sesiones Pi.
func LiteMCPServers() map[string]MCPServerDef {
	def := DefaultMCPServers()
	return map[string]MCPServerDef{
		"gomemory": def["gomemory"],
	}
}

// SetLiteMode activa o desactiva el modo liviano en ~/.pi-gomemory/lite.json y
// actualiza ~/.pi/agent/mcp.json en consecuencia.
func SetLiteMode(opts *Options, enabled bool) error {
	if err := os.MkdirAll(filepath.Dir(LiteStatePath(opts.HomeDir)), 0755); err != nil {
		return fmt.Errorf("crear directorio de estado lite: %w", err)
	}
	if err := writeJSONObject(LiteStatePath(opts.HomeDir), liteState{Enabled: enabled}); err != nil {
		return fmt.Errorf("guardar estado lite: %w", err)
	}
	if !enabled {
		return InstallMCPServers(opts.PiAgentDir)
	}

	if err := os.MkdirAll(opts.PiAgentDir, 0755); err != nil {
		return fmt.Errorf("crear %s: %w", opts.PiAgentDir, err)
	}
	mcpPath, err := configTarget(filepath.Join(opts.PiAgentDir, "mcp.json"))
	if err != nil {
		return err
	}
	config, err := readJSONObject(mcpPath)
	if err != nil {
		return err
	}
	servers := map[string]json.RawMessage{}
	if raw, ok := config["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil || servers == nil {
			return fmt.Errorf("%s: mcpServers no es un objeto; no se modifica", mcpPath)
		}
	}
	delete(servers, "codegraph")
	delete(servers, "codebase-memory")
	if err := setJSONField(servers, "gomemory", LiteMCPServers()["gomemory"]); err != nil {
		return err
	}
	if err := setJSONField(config, "mcpServers", servers); err != nil {
		return err
	}
	return writeJSONObject(mcpPath, config)
}

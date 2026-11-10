package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MCPServerDef define la especificación de un servidor MCP en Pi.
type MCPServerDef struct {
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Exposure    string            `json:"exposure,omitempty"`
	Description string            `json:"description,omitempty"`
}

// MCPConfig define el esquema de ~/.pi/agent/mcp.json.
type MCPConfig struct {
	MCPServers map[string]MCPServerDef `json:"mcpServers"`
}

// DefaultMCPServers devuelve los tres servidores requeridos: gomemory, codegraph y codebase-memory.
func DefaultMCPServers() map[string]MCPServerDef {
	return map[string]MCPServerDef{
		"gomemory": {
			Command:     "mem",
			Args:        []string{"mcp"},
			Exposure:    "direct",
			Description: "Gomemory persistent project memory, semantic graphs, checkpoints, HCD and review protocol",
		},
		"codegraph": {
			Command:     "codegraph",
			Args:        []string{"serve", "--mcp"},
			Exposure:    "direct",
			Description: "CodeGraph SQLite AST code intelligence and structural exploration",
		},
		"codebase-memory": {
			Command:     "codebase-memory-mcp",
			Exposure:    "direct",
			Description: "Codebase memory knowledge graph for deep structural queries and call tracing",
		},
	}
}

// InstallMCPServers inyecta y actualiza los 3 servidores MCP en ~/.pi/agent/mcp.json
// sin sobreescribir ni destruir otros servidores preexistentes que la persona tenga configurados.
func InstallMCPServers(piAgentDir string) error {
	mcpPath := filepath.Join(piAgentDir, "mcp.json")

	var config MCPConfig
	if data, err := os.ReadFile(mcpPath); err == nil {
		_ = json.Unmarshal(data, &config)
	}
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]MCPServerDef)
	}

	for name, def := range DefaultMCPServers() {
		config.MCPServers[name] = def
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar mcp.json: %w", err)
	}

	if err := os.WriteFile(mcpPath, data, 0644); err != nil {
		return fmt.Errorf("escribir mcp.json en %s: %w", mcpPath, err)
	}

	return nil
}

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

// InstallMCPServers registra los 3 servidores MCP en ~/.pi/agent/mcp.json.
// Solo reemplaza esas tres entradas: los demás servidores y claves del archivo
// se conservan byte a byte, y un mcp.json ilegible aborta sin escribir.
func InstallMCPServers(piAgentDir string) error {
	if err := os.MkdirAll(piAgentDir, 0755); err != nil {
		return fmt.Errorf("crear %s: %w", piAgentDir, err)
	}
	mcpPath, err := configTarget(filepath.Join(piAgentDir, "mcp.json"))
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

	for name, def := range DefaultMCPServers() {
		if err := setJSONField(servers, name, def); err != nil {
			return err
		}
	}
	if err := setJSONField(config, "mcpServers", servers); err != nil {
		return err
	}

	return writeJSONObject(mcpPath, config)
}

package installer

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// legacyHybridV1140 es el perfil híbrido exacto que escribía v1.14.0 (extraído
// del tag). uninstall lo reconoce como propio sin relajar la comparación de
// perfiles personalizados (HCD-047, C-06).
//
//go:embed legacy/mcp-hybrid-v1.14.0.json
var legacyHybridV1140 []byte

// managedMCPProfiles devuelve, por servidor, las definiciones exactas que
// pi-gomemory ha escrito: directa, híbrida vigente e híbridas publicadas.
func managedMCPProfiles() map[string][]json.RawMessage {
	profiles := map[string][]json.RawMessage{}
	for _, set := range []map[string]MCPServerDef{DefaultMCPServers(), CodemodeMCPServers()} {
		for name, def := range set {
			raw, _ := json.Marshal(def)
			profiles[name] = append(profiles[name], raw)
		}
	}
	var legacy map[string]json.RawMessage
	_ = json.Unmarshal(legacyHybridV1140, &legacy)
	for name, raw := range legacy {
		profiles[name] = append(profiles[name], raw)
	}
	return profiles
}

// MCPServerDef define la especificación de un servidor MCP en Pi.
type MCPServerDef struct {
	Command      string            `json:"command"`
	Args         []string          `json:"args,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	Exposure     string            `json:"exposure,omitempty"`
	ToolExposure map[string]string `json:"toolExposure,omitempty"`
	Description  string            `json:"description,omitempty"`
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

// CodemodeMCPServers devuelve la configuración MCP en modo codemode híbrido:
// gomemory y codebase-memory pasan a exposure "codemode" manteniendo como "direct"
// únicamente las herramientas de protocolo y consulta diaria, mientras codegraph
// permanece "direct" por tener una sola herramienta. El bloque Pi de
// gomemory-context.ts nombra las directas a partir de las tools efectivas
// de cada ejecución, así que no hay lista que mantener sincronizada (HCD-045).
func CodemodeMCPServers() map[string]MCPServerDef {
	servers := DefaultMCPServers()

	gm := servers["gomemory"]
	gm.Exposure = "codemode"
	gm.ToolExposure = map[string]string{
		"get_context":          "direct",
		"get_plan_context":     "direct",
		"search_memories":      "direct",
		"get_memory":           "direct",
		"save_memory":          "direct",
		"start_session":        "direct",
		"save_session_summary": "direct",
		"end_session":          "direct",
		"search_code":          "direct",
		"get_symbol":           "direct",
		"octopus_route_task":   "direct",
		"judge_memories":       "direct",
	}
	servers["gomemory"] = gm

	cbm := servers["codebase-memory"]
	cbm.Exposure = "codemode"
	cbm.ToolExposure = map[string]string{
		"search_graph":         "direct",
		"trace_path":           "direct",
		"get_code_snippet":     "direct",
		"check_index_coverage": "direct",
	}
	servers["codebase-memory"] = cbm

	return servers
}

// InstallMCPServers registra los 3 servidores MCP en ~/.pi/agent/mcp.json.
// Solo reemplaza esas tres entradas: los demás servidores y claves del archivo
// se conservan byte a byte, y un mcp.json ilegible aborta sin escribir.
// Si codemode está activo en settings.json o mcp.json, preserva el perfil híbrido.
func InstallMCPServers(piAgentDir string) error {
	st, err := GetCodemodeStatus(piAgentDir)
	if err != nil {
		return err
	}
	servers := DefaultMCPServers()
	if st.SettingsEnabled || st.MCPEnabled {
		servers = CodemodeMCPServers()
	}
	return writeMCPServers(piAgentDir, servers)
}

func writeMCPServers(piAgentDir string, targetServers map[string]MCPServerDef) error {
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

	for name, def := range targetServers {
		if err := setJSONField(servers, name, def); err != nil {
			return err
		}
	}
	if err := setJSONField(config, "mcpServers", servers); err != nil {
		return err
	}

	return writeJSONObject(mcpPath, config)
}

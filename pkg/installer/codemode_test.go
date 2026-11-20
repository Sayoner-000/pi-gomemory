package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// El protocolo de Pi ya no es una lista copiada en AGENTS.md: el bloque Pi de
// gomemory-context.ts se deriva de las tools efectivas (TestHCD045PiBlock), así
// que la coherencia que blindaba HCD-044 la garantiza la propia derivación.

func TestCodemodeMCP(t *testing.T) {
	t.Parallel()

	direct := DefaultMCPServers()
	if direct["gomemory"].Exposure != "direct" {
		t.Fatalf("DefaultMCPServers gomemory exposure = %q, want direct", direct["gomemory"].Exposure)
	}
	if len(direct["gomemory"].ToolExposure) != 0 {
		t.Fatalf("DefaultMCPServers gomemory ToolExposure should be empty, got %v", direct["gomemory"].ToolExposure)
	}

	hybrid := CodemodeMCPServers()
	gm, ok := hybrid["gomemory"]
	if !ok {
		t.Fatal("CodemodeMCPServers missing gomemory")
	}
	if gm.Exposure != "codemode" {
		t.Fatalf("gomemory exposure = %q, want codemode", gm.Exposure)
	}
	for _, tool := range []string{"get_context", "get_plan_context", "search_memories", "get_memory", "save_memory", "start_session", "save_session_summary", "end_session", "search_code", "get_symbol", "octopus_route_task", "judge_memories"} {
		if gm.ToolExposure[tool] != "direct" {
			t.Errorf("gomemory ToolExposure[%q] = %q, want direct", tool, gm.ToolExposure[tool])
		}
	}

	cbm, ok := hybrid["codebase-memory"]
	if !ok {
		t.Fatal("CodemodeMCPServers missing codebase-memory")
	}
	if cbm.Exposure != "codemode" {
		t.Fatalf("codebase-memory exposure = %q, want codemode", cbm.Exposure)
	}
	for _, tool := range []string{"search_graph", "trace_path", "get_code_snippet", "check_index_coverage"} {
		if cbm.ToolExposure[tool] != "direct" {
			t.Errorf("codebase-memory ToolExposure[%q] = %q, want direct", tool, cbm.ToolExposure[tool])
		}
	}

	cg := hybrid["codegraph"]
	if cg.Exposure != "direct" {
		t.Fatalf("codegraph exposure = %q, want direct", cg.Exposure)
	}

	raw, err := json.Marshal(gm)
	if err != nil {
		t.Fatalf("json.Marshal(gm): %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if _, ok := decoded["toolExposure"]; !ok {
		t.Fatal("expected JSON key \"toolExposure\" on MCPServerDef")
	}
}

func TestCodemodeSettings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	initial := `{
  "theme": "gomemory",
  "packages": ["npm:pi-antigravity@0.14.0"],
  "defaultTools": ["+grep"]
}
`
	if err := os.WriteFile(settingsPath, []byte(initial), 0600); err != nil {
		t.Fatalf("write initial settings: %v", err)
	}

	if err := SetCodemode(dir, true); err != nil {
		t.Fatalf("SetCodemode(true): %v", err)
	}
	// Idempotente: segunda llamada no duplica +codemode
	if err := SetCodemode(dir, true); err != nil {
		t.Fatalf("SetCodemode(true) second call: %v", err)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings after on: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal settings: %v", err)
	}
	if got["theme"] != "gomemory" {
		t.Fatalf("theme lost: %v", got["theme"])
	}
	tools, ok := got["defaultTools"].([]any)
	if !ok || len(tools) != 2 || tools[0] != "+grep" || tools[1] != "+codemode" {
		t.Fatalf("defaultTools after on = %#v, want [+grep +codemode]", got["defaultTools"])
	}

	st, err := GetCodemodeStatus(dir)
	if err != nil {
		t.Fatalf("GetCodemodeStatus(on): %v", err)
	}
	if !st.Enabled || !st.SettingsEnabled || !st.MCPEnabled || st.Partial {
		t.Fatalf("unexpected status on: %+v", st)
	}

	if err := SetCodemode(dir, false); err != nil {
		t.Fatalf("SetCodemode(false): %v", err)
	}
	dataOff, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings after off: %v", err)
	}
	var gotOff map[string]any
	if err := json.Unmarshal(dataOff, &gotOff); err != nil {
		t.Fatalf("unmarshal settings off: %v", err)
	}
	toolsOff, ok := gotOff["defaultTools"].([]any)
	if !ok || len(toolsOff) != 1 || toolsOff[0] != "+grep" {
		t.Fatalf("defaultTools after off = %#v, want [+grep]", gotOff["defaultTools"])
	}

	// Si defaultTools queda vacío al apagar, se elimina la clave defaultTools.
	dirEmpty := t.TempDir()
	if err := SetCodemode(dirEmpty, true); err != nil {
		t.Fatalf("dirEmpty on: %v", err)
	}
	if err := SetCodemode(dirEmpty, false); err != nil {
		t.Fatalf("dirEmpty off: %v", err)
	}
	dataEmpty, err := os.ReadFile(filepath.Join(dirEmpty, "settings.json"))
	if err != nil {
		t.Fatalf("read dirEmpty settings: %v", err)
	}
	var gotEmpty map[string]any
	if err := json.Unmarshal(dataEmpty, &gotEmpty); err != nil {
		t.Fatalf("unmarshal dirEmpty: %v", err)
	}
	if _, exists := gotEmpty["defaultTools"]; exists {
		t.Fatalf("expected defaultTools to be removed when empty, got %#v", gotEmpty["defaultTools"])
	}
}

func TestInstallMCPServersPreservesCodemode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Pre-popular otro servidor MCP para comprobar que tampoco se pierde.
	customMCP := `{
  "mcpServers": {
    "custom": { "command": "echo", "args": ["ok"] }
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "mcp.json"), []byte(customMCP), 0600); err != nil {
		t.Fatalf("write custom mcp.json: %v", err)
	}

	if err := SetCodemode(dir, true); err != nil {
		t.Fatalf("SetCodemode(true): %v", err)
	}
	if err := InstallMCPServers(dir); err != nil {
		t.Fatalf("InstallMCPServers: %v", err)
	}

	st, err := GetCodemodeStatus(dir)
	if err != nil {
		t.Fatalf("GetCodemodeStatus: %v", err)
	}
	if !st.Enabled || !st.MCPEnabled {
		t.Fatalf("InstallMCPServers reverted codemode: %+v", st)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "mcp.json"))
	if err != nil {
		t.Fatalf("read mcp.json: %v", err)
	}
	var parsed struct {
		MCPServers map[string]MCPServerDef `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal mcp.json: %v", err)
	}
	if parsed.MCPServers["custom"].Command != "echo" {
		t.Fatalf("custom server lost: %+v", parsed.MCPServers["custom"])
	}
	if parsed.MCPServers["gomemory"].ToolExposure["get_context"] != "direct" {
		t.Fatalf("gomemory toolExposure lost after InstallMCPServers: %+v", parsed.MCPServers["gomemory"].ToolExposure)
	}
}

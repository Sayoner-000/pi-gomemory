package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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

func readLiteState(homeDir string) (liteState, error) {
	var st liteState
	target, err := configTarget(LiteStatePath(homeDir))
	if err != nil {
		return st, err
	}
	data, err := os.ReadFile(target)
	if errors.Is(err, fs.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return st, fmt.Errorf("%s no es JSON válido: %w", target, err)
	}
	return st, nil
}

func writeLiteState(homeDir string, st liteState) error {
	target, err := configTarget(LiteStatePath(homeDir))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("crear directorio de estado lite: %w", err)
	}
	if err := writeJSONObject(target, st); err != nil {
		return fmt.Errorf("guardar estado lite: %w", err)
	}
	return nil
}

// GetLiteStatus devuelve si el modo liviano está activo o un error si el archivo
// de estado es ilegible o inválido.
func GetLiteStatus(homeDir string) (bool, error) {
	st, err := readLiteState(homeDir)
	return st.Enabled, err
}

// IsLiteMode indica si el modo liviano está activo en ~/.pi-gomemory/lite.json.
func IsLiteMode(homeDir string) bool {
	enabled, err := GetLiteStatus(homeDir)
	return err == nil && enabled
}

// LiteMCPServers devuelve solo gomemory en mcp.json (sin codegraph)
// para reducir el consumo de RAM cuando corren múltiples sesiones Pi.
func LiteMCPServers() map[string]MCPServerDef {
	def := DefaultMCPServers()
	return map[string]MCPServerDef{
		"gomemory": def["gomemory"],
	}
}

// SetLiteMode activa o desactiva el modo liviano en ~/.pi-gomemory/lite.json y
// actualiza ~/.pi/agent/mcp.json preservando entradas personalizadas (C-003),
// componiendo con codemode (C-002) y restaurando ambos archivos si algo falla (C-001).
func SetLiteMode(opts *Options, enabled bool) error {
	// Validar estado y configuración actuales antes de mutar ningún archivo.
	if _, err := readLiteState(opts.HomeDir); err != nil {
		return err
	}
	cmState, err := readCodemodeState(opts.HomeDir)
	if err != nil {
		return err
	}
	cmStatus, err := GetCodemodeStatus(opts.PiAgentDir)
	if err != nil {
		return err
	}
	codemodeEnabled := cmState.Preference == "on" || (cmState.Preference == "" && (cmStatus.SettingsEnabled || cmStatus.MCPEnabled))

	return withFileRollback([]string{
		filepath.Join(opts.PiAgentDir, "mcp.json"),
		LiteStatePath(opts.HomeDir),
	}, func() error {
		if err := reconcileMCPServers(opts.PiAgentDir, codemodeEnabled, enabled); err != nil {
			return err
		}
		return writeLiteState(opts.HomeDir, liteState{Enabled: enabled})
	})
}

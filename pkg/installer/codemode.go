package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	_, err := setCodemode(piAgentDir, enabled)
	return err
}

// setCodemode es SetCodemode e informa además si fue esta llamada la que
// añadió `+codemode` a defaultTools (no estaba antes).
func setCodemode(piAgentDir string, enabled bool) (bool, error) {
	if err := os.MkdirAll(piAgentDir, 0755); err != nil {
		return false, fmt.Errorf("crear %s: %w", piAgentDir, err)
	}
	added, err := setCodemodeSettings(piAgentDir, enabled)
	if err != nil {
		return false, err
	}
	servers := DefaultMCPServers()
	if enabled {
		servers = CodemodeMCPServers()
	}
	return added, writeMCPServers(piAgentDir, servers)
}

// codemodeState es la preferencia de codemode que recuerda pi-gomemory
// (HCD-046). Preference es "on" u "off" solo tras una orden explícita
// `codemode on|off`; vacía, install aplica el valor por defecto (on).
// AddedPlusCodemode prueba que `+codemode` lo añadió pi-gomemory, para que
// uninstall lo retire sin tocar uno que ya tuviera la persona (como HCD-011).
type codemodeState struct {
	Preference        string `json:"preference,omitempty"`
	AddedPlusCodemode bool   `json:"added_plus_codemode,omitempty"`
}

// CodemodeStatePath devuelve ~/.pi-gomemory/codemode.json.
func CodemodeStatePath(home string) string {
	return filepath.Join(home, ".pi-gomemory", "codemode.json")
}

func readCodemodeState(home string) (codemodeState, error) {
	var st codemodeState
	data, err := os.ReadFile(CodemodeStatePath(home))
	if errors.Is(err, fs.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return st, fmt.Errorf("%s no es JSON válido: %w", CodemodeStatePath(home), err)
	}
	return st, nil
}

func writeCodemodeState(home string, st codemodeState) error {
	path := CodemodeStatePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("crear %s: %w", filepath.Dir(path), err)
	}
	return writeJSONObject(path, st)
}

// enableCodemode activa el perfil híbrido y anota en st si pi-gomemory
// añadió `+codemode`; nunca borra una propiedad ya probada.
func enableCodemode(opts *Options, st *codemodeState) error {
	added, err := setCodemode(opts.PiAgentDir, true)
	if err != nil {
		return err
	}
	st.AddedPlusCodemode = st.AddedPlusCodemode || added
	return nil
}

// SetCodemodePreference aplica `codemode on|off` y recuerda la elección para
// que install y update la respeten (HCD-046).
func SetCodemodePreference(opts *Options, enabled bool) error {
	st, err := readCodemodeState(opts.HomeDir)
	if err != nil {
		return err
	}
	if enabled {
		if err := enableCodemode(opts, &st); err != nil {
			return err
		}
		st.Preference = "on"
	} else {
		if err := SetCodemode(opts.PiAgentDir, false); err != nil {
			return err
		}
		st = codemodeState{Preference: "off"}
	}
	return writeCodemodeState(opts.HomeDir, st)
}

// applyCodemodeDefault es el paso MCP de install: codemode híbrido por
// defecto, salvo que la persona lo desactivara con `codemode off`.
func applyCodemodeDefault(opts *Options, w io.Writer) error {
	st, err := readCodemodeState(opts.HomeDir)
	if err != nil {
		return err
	}
	if st.Preference == "off" {
		fmt.Fprintf(w, "%s    · codemode desactivado por la persona (pi-gomemory-setup codemode off): se conserva el modo directo%s\n", ColorMuted, ColorReset)
		return InstallMCPServers(opts.PiAgentDir)
	}
	if err := enableCodemode(opts, &st); err != nil {
		return err
	}
	return writeCodemodeState(opts.HomeDir, st)
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

func setCodemodeSettings(piAgentDir string, enabled bool) (bool, error) {
	settingsPath, err := configTarget(filepath.Join(piAgentDir, "settings.json"))
	if err != nil {
		return false, err
	}
	config, err := readJSONObject(settingsPath)
	if err != nil {
		return false, err
	}

	var tools []string
	if raw, ok := config["defaultTools"]; ok {
		if err := json.Unmarshal(raw, &tools); err != nil {
			return false, fmt.Errorf("%s: defaultTools no es una lista de cadenas: %w", settingsPath, err)
		}
	}

	added := false
	if enabled {
		if !slices.Contains(tools, "+codemode") && !slices.Contains(tools, "codemode") {
			tools = append(tools, "+codemode")
			added = true
		}
		if err := setJSONField(config, "defaultTools", tools); err != nil {
			return false, err
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
			return false, err
		}
	}

	return added, writeJSONObject(settingsPath, config)
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

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
	return withConfigRollback(piAgentDir, func() error {
		_, _, err := setCodemode(piAgentDir, enabled)
		return err
	})
}

// setCodemode es SetCodemode sin reversión: informa además si fue esta llamada
// la que añadió `+codemode` a defaultTools y el destino real de settings.json.
func setCodemode(piAgentDir string, enabled bool) (bool, string, error) {
	return setCodemodeWithLite(piAgentDir, enabled, false)
}

func setCodemodeWithLite(piAgentDir string, enabled, liteEnabled bool) (bool, string, error) {
	if err := os.MkdirAll(piAgentDir, 0755); err != nil {
		return false, "", fmt.Errorf("crear %s: %w", piAgentDir, err)
	}
	added, target, err := setCodemodeSettings(piAgentDir, enabled)
	if err != nil {
		return false, "", err
	}
	return added, target, reconcileMCPServers(piAgentDir, enabled, liteEnabled)
}

// configSnapshot conserva los bytes de una configuración de Pi para
// restaurarla si la conmutación de codemode falla a medias (HCD-047, C-01).
type configSnapshot struct {
	path   string
	data   []byte
	mode   os.FileMode
	exists bool
}

func takeConfigSnapshot(path string) (configSnapshot, error) {
	target, err := configTarget(path)
	if err != nil {
		return configSnapshot{}, err
	}
	s := configSnapshot{path: target}
	info, err := os.Stat(target)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if s.data, err = os.ReadFile(target); err != nil {
		return s, err
	}
	s.mode, s.exists = info.Mode().Perm(), true
	return s, nil
}

func (s configSnapshot) restore() error {
	if !s.exists {
		if err := os.Remove(s.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	return writeFileAtomic(s.path, s.data, s.mode)
}

// withFileRollback ejecuta fn y, si falla, devuelve todas las rutas indicadas
// a sus bytes anteriores; un error al restaurar se informa junto al original.
func withFileRollback(paths []string, fn func() error) error {
	var snaps []configSnapshot
	for _, p := range paths {
		s, err := takeConfigSnapshot(p)
		if err != nil {
			return err
		}
		snaps = append(snaps, s)
	}
	err := fn()
	if err == nil {
		return nil
	}
	var restoreErr error
	for i := len(snaps) - 1; i >= 0; i-- {
		restoreErr = errors.Join(restoreErr, snaps[i].restore())
	}
	if restoreErr != nil {
		return fmt.Errorf("%w; no se pudo restaurar la configuración anterior de Pi: %v", err, restoreErr)
	}
	return err
}

// withConfigRollback ejecuta fn y, si falla, devuelve settings.json y
// mcp.json a sus bytes anteriores; un error al restaurar se informa junto al
// original. Así un fallo no deja una activación sin registro de propiedad.
func withConfigRollback(piAgentDir string, fn func() error) error {
	return withFileRollback([]string{
		filepath.Join(piAgentDir, "settings.json"),
		filepath.Join(piAgentDir, "mcp.json"),
	}, fn)
}

// codemodeState es la preferencia de codemode que recuerda pi-gomemory
// (HCD-046, HCD-050). Preference es "on" u "off" solo tras una orden explícita
// `codemode on|off`; vacía, install aplica el valor por defecto (off, modo directo).
// AddedPlusCodemode prueba que `+codemode` lo añadió pi-gomemory, para que
// uninstall lo retire sin tocar uno que ya tuviera la persona (como HCD-011).
// SettingsTarget es el archivo donde lo añadió: la prueba solo vale para ese
// destino (HCD-047, C-08). Un registro de v1.15.0 no lo incluye.
type codemodeState struct {
	Preference        string `json:"preference,omitempty"`
	AddedPlusCodemode bool   `json:"added_plus_codemode,omitempty"`
	SettingsTarget    string `json:"settings_target,omitempty"`
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
// añadió `+codemode` y en qué destino. Conserva una propiedad ya probada en el
// mismo destino; si el destino cambió, esa prueba ya no vale (HCD-047, C-08).
func enableCodemode(opts *Options, st *codemodeState) error {
	liteSt, err := readLiteState(opts.HomeDir)
	if err != nil {
		return err
	}
	return enableCodemodeWithLite(opts, st, liteSt.Enabled)
}

func enableCodemodeWithLite(opts *Options, st *codemodeState, liteEnabled bool) error {
	added, target, err := setCodemodeWithLite(opts.PiAgentDir, true, liteEnabled)
	if err != nil {
		return err
	}
	switch {
	case added:
		st.AddedPlusCodemode, st.SettingsTarget = true, target
	case st.AddedPlusCodemode && st.SettingsTarget != "" && st.SettingsTarget != target:
		st.AddedPlusCodemode, st.SettingsTarget = false, ""
	}
	return nil
}

// SetCodemodePreference aplica `codemode on|off` y recuerda la elección para
// que install y update la respeten (HCD-046), componiendo con el modo lite (C-002).
// Si algo falla, settings.json y mcp.json vuelven a su estado anterior (HCD-047, C-01).
func SetCodemodePreference(opts *Options, enabled bool) error {
	st, err := readCodemodeState(opts.HomeDir)
	if err != nil {
		return err
	}
	liteSt, err := readLiteState(opts.HomeDir)
	if err != nil {
		return err
	}
	return withConfigRollback(opts.PiAgentDir, func() error {
		if enabled {
			if err := enableCodemodeWithLite(opts, &st, liteSt.Enabled); err != nil {
				return err
			}
			st.Preference = "on"
		} else {
			if _, _, err := setCodemodeWithLite(opts.PiAgentDir, false, liteSt.Enabled); err != nil {
				return err
			}
			st = codemodeState{Preference: "off"}
		}
		return writeCodemodeState(opts.HomeDir, st)
	})
}

// applyCodemodeDefault es el paso MCP de install y update: codemode híbrido
// por defecto (HCD-071, que revierte HCD-050) y modo directo solo si la
// persona eligió `codemode off`, componiendo con el modo lite (C-002). En
// híbrido registra si `+codemode` lo añadió pi-gomemory y en qué destino
// (C-08), sin fijar preferencia. En directo no toca defaultTools, que puede
// llevar un `+codemode` de la persona (HCD-047, C-02).
// Tras aplicar el modo avisa, sin cambiar nada, del estado parcial y de las
// tools de codebase-memory duplicadas por una extensión ajena (HCD-070).
func applyCodemodeDefault(opts *Options, w io.Writer) error {
	if err := applyCodemodeMode(opts, w); err != nil {
		return err
	}
	warnExposure(opts.PiAgentDir, w)
	return nil
}

func applyCodemodeMode(opts *Options, w io.Writer) error {
	st, err := readCodemodeState(opts.HomeDir)
	if err != nil {
		return err
	}
	liteSt, err := readLiteState(opts.HomeDir)
	if err != nil {
		return err
	}
	if st.Preference != "off" {
		switch {
		case st.Preference == "on":
			fmt.Fprintf(w, "%s    · codemode activado por la persona (pi-gomemory-setup codemode on): se conserva el perfil híbrido%s\n", ColorMuted, ColorReset)
		default:
			fmt.Fprintf(w, "%s    · codemode híbrido por defecto: tools de protocolo directas y el resto por codemode (vuelve a directo con pi-gomemory-setup codemode off)%s\n", ColorMuted, ColorReset)
		}
		if liteSt.Enabled {
			fmt.Fprintf(w, "%s    · modo lite activo: gomemory en perfil híbrido, sin servidores de grafo%s\n", ColorMuted, ColorReset)
		}
		return withConfigRollback(opts.PiAgentDir, func() error {
			if err := enableCodemodeWithLite(opts, &st, liteSt.Enabled); err != nil {
				return err
			}
			return writeCodemodeState(opts.HomeDir, st)
		})
	}
	fmt.Fprintf(w, "%s    · codemode off elegido por la persona: servidores MCP en modo directo%s\n", ColorMuted, ColorReset)
	if liteSt.Enabled {
		fmt.Fprintf(w, "%s    · modo lite activo (pi-gomemory-setup lite on): se conserva perfil liviano%s\n", ColorMuted, ColorReset)
	}
	return reconcileMCPServers(opts.PiAgentDir, false, liteSt.Enabled)
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

func setCodemodeSettings(piAgentDir string, enabled bool) (bool, string, error) {
	settingsPath, err := configTarget(filepath.Join(piAgentDir, "settings.json"))
	if err != nil {
		return false, "", err
	}
	added, err := editCodemodeSettings(settingsPath, enabled)
	return added, settingsPath, err
}

func editCodemodeSettings(settingsPath string, enabled bool) (bool, error) {
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
	gm, hasGM := servers["gomemory"]
	if !hasGM {
		return false, nil
	}
	if cbm, hasCBM := servers["codebase-memory"]; hasCBM {
		return gm.Exposure == "codemode" && cbm.Exposure == "codemode", nil
	}
	return gm.Exposure == "codemode", nil
}

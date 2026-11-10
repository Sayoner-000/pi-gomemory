package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// InstallTheme copia el tema embebido themes/gomemory.json a
// ~/.pi/agent/themes/ y lo activa en ~/.pi/agent/settings.json.
func InstallTheme(assets fs.FS, piAgentDir string) error {
	data, err := fs.ReadFile(assets, "themes/gomemory.json")
	if err != nil {
		return fmt.Errorf("leer tema embebido: %w", err)
	}

	themesDir := filepath.Join(piAgentDir, "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return fmt.Errorf("crear dir themes en pi: %w", err)
	}
	if err := os.WriteFile(filepath.Join(themesDir, "gomemory.json"), data, 0644); err != nil {
		return fmt.Errorf("escribir tema: %w", err)
	}

	settingsPath := filepath.Join(piAgentDir, "settings.json")
	settings, err := readJSONObject(settingsPath)
	if err != nil {
		return err
	}
	if err := setJSONField(settings, "theme", "gomemory"); err != nil {
		return err
	}
	return writeJSONObject(settingsPath, settings)
}

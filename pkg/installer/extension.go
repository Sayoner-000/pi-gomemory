package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// InstallTUIExtension instala las extensiones embebidas y la definición neutral de Splash.
func InstallTUIExtension(assets fs.FS, piAgentDir string) error {
	for _, resource := range []struct{ source, destination string }{
		{"extensions/gomemory-status.ts", "gomemory-status.ts"},
		{"extensions/gomemory-splash.ts", "gomemory-splash.ts"},
		{"config/splash.md", "gomemory-splash-prompt.md"},
	} {
		data, err := fs.ReadFile(assets, resource.source)
		if err != nil {
			return fmt.Errorf("leer extensión embebida: %w", err)
		}

		extDir := filepath.Join(piAgentDir, "extensions")
		if err := os.MkdirAll(extDir, 0755); err != nil {
			return fmt.Errorf("crear dir extensions en pi: %w", err)
		}

		dstFile := filepath.Join(extDir, resource.destination)
		if err := os.WriteFile(dstFile, data, 0644); err != nil {
			return fmt.Errorf("escribir extensión %s: %w", dstFile, err)
		}
	}
	return nil
}

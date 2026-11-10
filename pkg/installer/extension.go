package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// InstallTUIExtension instala la extensión embebida gomemory-status.ts en
// ~/.pi/agent/extensions/.
func InstallTUIExtension(assets fs.FS, piAgentDir string) error {
	data, err := fs.ReadFile(assets, "extensions/gomemory-status.ts")
	if err != nil {
		return fmt.Errorf("leer extensión embebida: %w", err)
	}

	extDir := filepath.Join(piAgentDir, "extensions")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		return fmt.Errorf("crear dir extensions en pi: %w", err)
	}

	dstFile := filepath.Join(extDir, "gomemory-status.ts")
	if err := os.WriteFile(dstFile, data, 0644); err != nil {
		return fmt.Errorf("escribir extensión %s: %w", dstFile, err)
	}
	return nil
}

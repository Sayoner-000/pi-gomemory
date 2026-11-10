package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// InstallTUIExtension instala la extensión gomemory-status.ts en ~/.pi/agent/extensions/
// utilizando el FS embebido si está disponible, o el filesystem local.
func InstallTUIExtension(embedFS fs.FS, packageRootDir, piAgentDir string) error {
	extDir := filepath.Join(piAgentDir, "extensions")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		return fmt.Errorf("crear dir extensions en pi: %w", err)
	}

	dstFile := filepath.Join(extDir, "gomemory-status.ts")

	var data []byte
	var err error

	// 1. Intentar desde FS embebido
	if embedFS != nil {
		data, err = fs.ReadFile(embedFS, "extensions/gomemory-status.ts")
	}

	// 2. Si no hay embebido o falló, buscar en disco
	if len(data) == 0 {
		srcFile := filepath.Join(packageRootDir, "extensions", "gomemory-status.ts")
		data, err = os.ReadFile(srcFile)
		if err != nil {
			return fmt.Errorf("leer extension origen %s: %w", srcFile, err)
		}
	}

	if err := os.WriteFile(dstFile, data, 0644); err != nil {
		return fmt.Errorf("escribir extension destino %s: %w", dstFile, err)
	}

	return nil
}

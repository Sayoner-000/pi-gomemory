package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Manifest registra lo que escribió install para que uninstall pueda
// demostrar la propiedad de cada archivo sea cual sea la versión del binario.
type Manifest struct {
	Version string            `json:"version"`
	BinDir  string            `json:"bin_dir"`
	Files   map[string]string `json:"files"` // ruta absoluta → SHA256
}

// ManifestPath devuelve ~/.pi-gomemory/manifest.json.
func ManifestPath(home string) string {
	return filepath.Join(home, ".pi-gomemory", "manifest.json")
}

// UninstalledMarkerPath devuelve la marca que deja uninstall y que detiene skills-sync.
func UninstalledMarkerPath(home string) string {
	return filepath.Join(home, ".pi-gomemory", "uninstalled")
}

// IsUninstalled indica si existe la marca de desinstalado.
func IsUninstalled(home string) bool {
	_, err := os.Lstat(UninstalledMarkerPath(home))
	return err == nil
}

// installedFiles enumera los archivos completos que escribe install.
func installedFiles(opts *Options) []string {
	files := []string{
		filepath.Join(opts.PiAgentDir, "themes", "gomemory.json"),
		filepath.Join(opts.LocalBinDir, "pi-gomemory"),
	}
	for _, r := range tuiResources {
		files = append(files, filepath.Join(opts.PiAgentDir, "extensions", r.destination))
	}
	return files
}

// WriteManifest registra el SHA256 de cada archivo instalado y el BIN_DIR usado.
func WriteManifest(opts *Options, version string) error {
	m := Manifest{Version: version, BinDir: opts.LocalBinDir, Files: map[string]string{}}
	for _, p := range installedFiles(opts) {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		m.Files[p] = contentHash(data)
	}
	path := ManifestPath(opts.HomeDir)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("crear %s: %w", filepath.Dir(path), err)
	}
	return writeJSONObject(path, m)
}

// ReadManifest lee el manifiesto; devuelve nil sin error si no existe.
func ReadManifest(home string) (*Manifest, error) {
	data, err := os.ReadFile(ManifestPath(home))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s no es JSON válido: %w", ManifestPath(home), err)
	}
	return &m, nil
}

package installer

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Paquetes soportados
const (
	AntigravityPackageSource = "npm:pi-antigravity"
	CCExtensionsSource       = "npm:pi-cc-extensions"
)

// PatchAntigravityPackageJSON corrige el aviso de TypeBox como host-provided peerDependency en Pi.
func PatchAntigravityPackageJSON(piAgentDir string) error {
	pkgPath := filepath.Join(piAgentDir, "npm", "node_modules", "pi-antigravity", "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil // No instalado todavía, nada que parchar
	}

	var pkg map[string]any
	if err := json.Unmarshal(data, &pkg); err != nil {
		return err
	}

	changed := false

	// 1. Asegurar peerDependencies con typebox: "*"
	peerDeps, ok := pkg["peerDependencies"].(map[string]any)
	if !ok || peerDeps == nil {
		peerDeps = make(map[string]any)
		pkg["peerDependencies"] = peerDeps
	}
	if peerDeps["typebox"] != "*" {
		peerDeps["typebox"] = "*"
		changed = true
	}

	// 2. Retirar typebox de dependencies
	if deps, ok := pkg["dependencies"].(map[string]any); ok && deps != nil {
		if _, exists := deps["typebox"]; exists {
			delete(deps, "typebox")
			changed = true
		}
	}

	if changed {
		out, err := json.MarshalIndent(pkg, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(pkgPath, out, 0644); err != nil {
			return err
		}
	}

	return nil
}

// InstallAntigravityPlugin instala o asegura el plugin pi-antigravity en Pi y corrige sus peerDependencies.
func InstallAntigravityPlugin(piAgentDir string) error {
	// 1. Intentar instalar a través de la CLI de pi si está en PATH
	if piBin, err := exec.LookPath("pi"); err == nil {
		cmd := exec.Command(piBin, "install", AntigravityPackageSource)
		out, err := cmd.CombinedOutput()
		if err == nil || strings.Contains(string(out), "already installed") || strings.Contains(string(out), "updated") {
			_ = PatchAntigravityPackageJSON(piAgentDir)
			return nil
		}
	}

	// 2. Fallback declarativo: asegurar que esté listado en ~/.pi/agent/settings.json
	settingsPath := filepath.Join(piAgentDir, "settings.json")
	settings, err := readJSONObject(settingsPath)
	if err != nil {
		return err
	}

	// Se conservan las entradas tal cual: un paquete puede ser un string o un objeto.
	var packages []json.RawMessage
	if raw, ok := settings["packages"]; ok {
		if err := json.Unmarshal(raw, &packages); err != nil {
			return fmt.Errorf("%s: packages no es una lista; no se modifica", settingsPath)
		}
	}

	found := false
	for _, p := range packages {
		if strings.Contains(string(p), "pi-antigravity") {
			found = true
			break
		}
	}

	if !found {
		entry, _ := json.Marshal(AntigravityPackageSource)
		packages = append(packages, entry)
		if err := setJSONField(settings, "packages", packages); err != nil {
			return err
		}
		if err := writeJSONObject(settingsPath, settings); err != nil {
			return err
		}
	}

	_ = PatchAntigravityPackageJSON(piAgentDir)
	return nil
}

// InstallCCExtensions instala el paquete pi-cc-extensions y escribe
// ~/.pi/agent/pi-cc-extensions.json a partir de config/pi-cc-extensions.json
// embebido. Las claves que la persona ya tenga configuradas se respetan; solo
// se añaden las que falten.
func InstallCCExtensions(assets fs.FS, piAgentDir string) error {
	defaults := map[string]json.RawMessage{}
	data, err := fs.ReadFile(assets, "config/pi-cc-extensions.json")
	if err != nil {
		return fmt.Errorf("leer config embebida de pi-cc-extensions: %w", err)
	}
	if err := json.Unmarshal(data, &defaults); err != nil {
		return fmt.Errorf("config embebida de pi-cc-extensions inválida: %w", err)
	}

	cfgPath := filepath.Join(piAgentDir, "pi-cc-extensions.json")
	config, err := readJSONObject(cfgPath)
	if err != nil {
		return err
	}

	// Instalar paquete vía pi install si pi está disponible
	if piBin, err := exec.LookPath("pi"); err == nil {
		_, _ = exec.Command(piBin, "install", CCExtensionsSource).CombinedOutput()
	}

	for key, value := range defaults {
		if _, exists := config[key]; !exists {
			config[key] = value
		}
	}
	return writeJSONObject(cfgPath, config)
}

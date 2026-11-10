package installer

import (
	"encoding/json"
	"fmt"
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
	var settings map[string]any
	if data, err := os.ReadFile(settingsPath); err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	var packages []string
	if pkgsRaw, ok := settings["packages"].([]any); ok {
		for _, p := range pkgsRaw {
			if str, ok := p.(string); ok {
				packages = append(packages, str)
			}
		}
	}

	found := false
	for _, p := range packages {
		if p == AntigravityPackageSource || strings.Contains(p, "pi-antigravity") {
			found = true
			break
		}
	}

	if !found {
		packages = append(packages, AntigravityPackageSource)
		settings["packages"] = packages

		data, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return fmt.Errorf("serializar settings.json: %w", err)
		}
		if err := os.WriteFile(settingsPath, data, 0644); err != nil {
			return fmt.Errorf("guardar settings.json: %w", err)
		}
	}

	_ = PatchAntigravityPackageJSON(piAgentDir)
	return nil
}

// InstallCCExtensions instala y configura el paquete de estilo Claude Code (pi-cc-extensions)
// con configuraciones optimizadas para gomemory.
func InstallCCExtensions(piAgentDir string) error {
	// 1. Instalar paquete vía pi install si pi está disponible
	if piBin, err := exec.LookPath("pi"); err == nil {
		cmd := exec.Command(piBin, "install", CCExtensionsSource)
		_, _ = cmd.CombinedOutput()
	}

	// 2. Configurar ~/.pi/agent/pi-cc-extensions.json
	cfgPath := filepath.Join(piAgentDir, "pi-cc-extensions.json")
	config := map[string]any{
		"mode":                    "on",
		"enableContextCommand":    true,
		"enableAgentSummary":      true,
		"enableWorkingMessage":    true,
		"enableEditorPrefix":      true,
		"showStartupHeader":       true,
		"diffViewMode":            "auto",
		"diffIndicatorMode":       "bars",
		"enableCustomFooter":      true,
		"footerNerdIcons":         true,
		"footerTokenRate":         true,
		"footerTokenRateMode":     "average",
		"footerSessionTokens":     true,
		"footerCurrency":          "USD",
		"footerLine1Keys":         []string{"model", "context", "cache", "rate", "cost", "tokens"},
		"footerLine2Keys":         []string{"cwd", "git"},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar pi-cc-extensions.json: %w", err)
	}

	return os.WriteFile(cfgPath, data, 0644)
}

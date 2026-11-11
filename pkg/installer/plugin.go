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

// Paquetes soportados, fijados a la versión publicada en el catálogo oficial de
// Pi (pi.dev/packages). Pi no reinstala un paquete fijado a una versión exacta.
const (
	AntigravityVersion       = "0.12.0"
	CCExtensionsVersion      = "1.0.16"
	AntigravityPackageSource = "npm:pi-antigravity@" + AntigravityVersion
	CCExtensionsSource       = "npm:pi-cc-extensions@" + CCExtensionsVersion
)

// managedCCKeys son las claves de pi-cc-extensions.json que pi-gomemory
// impone en cada instalación, no solo cuando faltan.
var managedCCKeys = []string{"showStartupHeader"}

// pinnedPackages relaciona cada paquete con su fuente fijada.
var pinnedPackages = []struct{ name, source string }{
	{"pi-antigravity", AntigravityPackageSource},
	{"pi-cc-extensions", CCExtensionsSource},
}

// npmPackageName devuelve el nombre de un paquete npm:<nombre>[@versión], o "".
func npmPackageName(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) != nil || !strings.HasPrefix(s, "npm:") {
		return ""
	}
	s = strings.TrimPrefix(s, "npm:")
	if i := strings.LastIndex(s, "@"); i > 0 {
		s = s[:i]
	}
	return s
}

// EnsurePinnedPackages deja en settings.json una sola entrada por paquete
// soportado, con su versión fijada; el resto de entradas se conserva.
func EnsurePinnedPackages(piAgentDir string) error {
	settingsPath := filepath.Join(piAgentDir, "settings.json")
	settings, err := readJSONObject(settingsPath)
	if err != nil {
		return err
	}
	var packages []json.RawMessage
	if raw, ok := settings["packages"]; ok {
		if err := json.Unmarshal(raw, &packages); err != nil {
			return fmt.Errorf("%s: packages no es una lista; no se modifica", settingsPath)
		}
	}
	var out []json.RawMessage
	placed := map[string]bool{}
	for _, p := range packages {
		name := npmPackageName(p)
		pinned := ""
		for _, pp := range pinnedPackages {
			if pp.name == name {
				pinned = pp.source
			}
		}
		if pinned == "" {
			out = append(out, p)
			continue
		}
		if placed[name] {
			continue
		}
		placed[name] = true
		entry, _ := json.Marshal(pinned)
		out = append(out, entry)
	}
	for _, pp := range pinnedPackages {
		if !placed[pp.name] {
			entry, _ := json.Marshal(pp.source)
			out = append(out, entry)
		}
	}
	if string(mustJSON(out)) == string(mustJSON(packages)) {
		return nil
	}
	if err := setJSONField(settings, "packages", out); err != nil {
		return err
	}
	if err := os.MkdirAll(piAgentDir, 0755); err != nil {
		return err
	}
	return writeJSONObject(settingsPath, settings)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

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
		if err := writeFileAtomic(pkgPath, out, 0644); err != nil {
			return err
		}
	}

	return nil
}

// InstallAntigravityPlugin instala pi-antigravity en su versión fijada y corrige sus peerDependencies.
func InstallAntigravityPlugin(piAgentDir string) error {
	if piBin, err := exec.LookPath("pi"); err == nil {
		if out, err := exec.Command(piBin, "install", AntigravityPackageSource).CombinedOutput(); err != nil {
			fmt.Printf("%s    ⚠ pi install %s falló; se registra en settings.json: %s%s\n", ColorWarning, AntigravityPackageSource, strings.TrimSpace(firstLine(string(out))), ColorReset)
		}
	}
	// pi install puede dejar la entrada sin versión o duplicada: se normaliza siempre.
	if err := EnsurePinnedPackages(piAgentDir); err != nil {
		return err
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
		if out, err := exec.Command(piBin, "install", CCExtensionsSource).CombinedOutput(); err != nil {
			fmt.Printf("%s    ⚠ pi install %s falló; se registra en settings.json: %s%s\n", ColorWarning, CCExtensionsSource, strings.TrimSpace(firstLine(string(out))), ColorReset)
		}
	}
	if err := EnsurePinnedPackages(piAgentDir); err != nil {
		return err
	}

	for key, value := range defaults {
		if _, exists := config[key]; !exists {
			config[key] = value
		}
	}
	// El header de pi-cc-extensions sustituye al banner de gomemory: ambos usan
	// el único hueco de setHeader. Se fuerza aunque una instalación anterior lo dejara activo.
	for _, key := range managedCCKeys {
		if value, ok := defaults[key]; ok {
			config[key] = value
		}
	}
	return writeJSONObject(cfgPath, config)
}

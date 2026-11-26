package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// pluginInstallTimeout acota `pi install` para evitar bloqueos indefinidos ante
// cuelgues de red o prompts interactivos (HCD-083, H-04). Es variable para tests.
var pluginInstallTimeout = 2 * time.Minute

func runPiInstall(piBin, pkgSource string) ([]byte, error) {
	return runPi(piBin, "install", pkgSource)
}

// runPi ejecuta `pi <verb> <source>` acotado por pluginInstallTimeout.
func runPi(piBin, verb, pkgSource string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pluginInstallTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, piBin, verb, pkgSource)
	cmd.Stdin = nil
	return cmd.CombinedOutput()
}

// Paquetes soportados, fijados a la versión publicada en el catálogo oficial de
// Pi (pi.dev/packages). Pi no reinstala un paquete fijado a una versión exacta.
const (
	CCExtensionsVersion = "1.0.22"
	CCExtensionsSource  = "npm:pi-cc-extensions@" + CCExtensionsVersion
)

// historicPins son los pins que pi-gomemory escribió en versiones anteriores.
// Al subir un pin, el anterior se añade aquí: uninstall --packages retira esas
// entradas como el pin actual, porque solo pi-gomemory escribió esa fuente
// exacta (HCD-040, Q1). pi-antigravity dejó de distribuirse en HCD-085: todos
// sus pins son históricos.
var historicPins = []struct{ name, source string }{
	{"pi-antigravity", "npm:pi-antigravity@0.12.0"},
	{"pi-antigravity", "npm:pi-antigravity@0.14.0"},
	{"pi-antigravity", "npm:pi-antigravity@0.15.0"},
	{"pi-cc-extensions", "npm:pi-cc-extensions@1.0.16"},
	{"pi-cc-extensions", "npm:pi-cc-extensions@1.0.20"},
	{"pi-cc-extensions", "npm:pi-cc-extensions@1.0.21"},
}

// isHistoricPin indica si source es un pin anterior de pi-gomemory para name.
func isHistoricPin(name, source string) bool {
	for _, h := range historicPins {
		if h.name == name && h.source == source {
			return true
		}
	}
	return false
}

// historicPinnedSources devuelve las fuentes de historicPins.
func historicPinnedSources() []string {
	out := make([]string, 0, len(historicPins))
	for _, h := range historicPins {
		out = append(out, h.source)
	}
	return out
}

// managedCCKeys son las claves de pi-cc-extensions.json que pi-gomemory
// impone en cada instalación, no solo cuando faltan.
var managedCCKeys = []string{"showStartupHeader"}

// pinnedPackages relaciona cada paquete con su fuente fijada.
var pinnedPackages = []struct{ name, source string }{
	{"pi-cc-extensions", CCExtensionsSource},
}

// managedPackages son los paquetes cuyas entradas escribió pi-gomemory: los
// fijados hoy y, sin fuente actual, los retirados que solo tienen pins
// históricos (pi-antigravity desde HCD-085). uninstall --packages los recorre.
func managedPackages() []struct{ name, source string } {
	out := append([]struct{ name, source string }{}, pinnedPackages...)
	seen := map[string]bool{}
	for _, p := range pinnedPackages {
		seen[p.name] = true
	}
	for _, h := range historicPins {
		if !seen[h.name] {
			seen[h.name] = true
			out = append(out, struct{ name, source string }{h.name, ""})
		}
	}
	return out
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
	settingsPath, err := configTarget(filepath.Join(piAgentDir, "settings.json"))
	if err != nil {
		return err
	}
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

// antigravityAccountsFile guarda la sesión OAuth de Google de pi-antigravity.
const antigravityAccountsFile = "antigravity-accounts.json"

// RetireAntigravityPackage retira de settings.json las entradas de
// pi-antigravity que escribió pi-gomemory (sus pins actuales o históricos) y,
// si pi está disponible, se lo pide también a `pi remove` (HCD-085). Una entrada
// de pi-antigravity con otra fuente la añadió la persona: se conserva con aviso.
// Nunca toca antigravity-accounts.json: son credenciales de la persona.
func RetireAntigravityPackage(piAgentDir string, out io.Writer) error {
	settingsPath, err := configTarget(filepath.Join(piAgentDir, "settings.json"))
	if err != nil {
		return err
	}
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
	kept := []json.RawMessage{}
	var retired []string
	for _, p := range packages {
		var source string
		_ = json.Unmarshal(p, &source)
		if isHistoricPin("pi-antigravity", source) {
			retired = append(retired, source)
			continue
		}
		if npmPackageName(p) == "pi-antigravity" {
			fmt.Fprintf(out, "%s    ⚠ %s no lo instaló pi-gomemory: se conserva en settings.json%s\n", ColorWarning, source, ColorReset)
		}
		kept = append(kept, p)
	}
	if len(retired) > 0 {
		if piBin, err := exec.LookPath("pi"); err == nil {
			for _, source := range retired {
				_, _ = runPi(piBin, "remove", source)
			}
		}
		// pi remove puede faltar o fallar: settings.json queda coherente siempre.
		if settings, err = readJSONObject(settingsPath); err != nil {
			return err
		}
		if err := setJSONField(settings, "packages", kept); err != nil {
			return err
		}
		if err := writeJSONObject(settingsPath, settings); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s    ✓ retirado %s de settings.json%s\n", ColorMuted, strings.Join(retired, ", "), ColorReset)
	}
	accounts := filepath.Join(piAgentDir, antigravityAccountsFile)
	if _, err := os.Stat(accounts); err == nil {
		fmt.Fprintf(out, "%s    ⚠ %s guarda tu sesión OAuth de Google y no se toca: bórralo o revoca el acceso si ya no lo usas%s\n", ColorWarning, accounts, ColorReset)
	}
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

	cfgPath, err := configTarget(filepath.Join(piAgentDir, "pi-cc-extensions.json"))
	if err != nil {
		return err
	}
	config, err := readJSONObject(cfgPath)
	if err != nil {
		return err
	}
	// settings.json se valida antes de que pi install pueda escribir en él (C-15).
	if _, err := configTarget(filepath.Join(piAgentDir, "settings.json")); err != nil {
		return err
	}

	// Instalar paquete vía pi install si pi está disponible
	if piBin, err := exec.LookPath("pi"); err == nil {
		if out, err := runPiInstall(piBin, CCExtensionsSource); err != nil {
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

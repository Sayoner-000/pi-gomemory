package installer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// extStubs sustituye los paquetes de Pi por módulos mínimos para ejecutar la
// lógica de las extensiones con node sin instalar Pi.
var extStubs = map[string]string{
	"@earendil-works/pi-ai": `export const Type = new Proxy({}, { get: () => (...a) => ({ a }) });`,
	"@earendil-works/pi-coding-agent": `
export const calls = [];
export const defineTool = (t) => t;
export const createExtensionRuntime = () => ({});
export const ModelRuntime = { create: async () => { calls.push("ModelRuntime"); return {}; } };
export const createAgentSession = async () => { calls.push("createAgentSession"); throw new Error("no provider in tests"); };
export const SessionManager = { inMemory: () => ({}) };
export const SettingsManager = { inMemory: () => ({}) };`,
	"@earendil-works/pi-tui": `
export const visibleWidth = (s) => s.replace(/\x1b\[[0-9;]*m/g, "").length;
export const truncateToWidth = (s) => s;`,
}

// runNodeExt copia las extensiones indicadas a un directorio temporal con los
// stubs y ejecuta script (ESM). Falla, no se omite, si node no está.
func runNodeExt(t *testing.T, files []string, script string) string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("node (>= 22.18) es obligatorio para las pruebas de extensiones")
	}
	dir := t.TempDir()
	for pkg, src := range extStubs {
		pdir := filepath.Join(dir, "node_modules", filepath.FromSlash(pkg))
		writeFile(t, filepath.Join(pdir, "package.json"), `{"type":"module","main":"index.js"}`)
		writeFile(t, filepath.Join(pdir, "index.js"), src)
	}
	writeFile(t, filepath.Join(dir, "package.json"), `{"type":"module"}`)
	for _, f := range files {
		writeFile(t, filepath.Join(dir, filepath.Base(f)), readAsset(t, f))
	}
	writeFile(t, filepath.Join(dir, "gomemory-splash-prompt.md"), readAsset(t, "config/splash.md"))
	writeFile(t, filepath.Join(dir, "test.mjs"), script)
	cmd := exec.Command(node, "test.mjs")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "NODE_NO_WARNINGS=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node falló: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

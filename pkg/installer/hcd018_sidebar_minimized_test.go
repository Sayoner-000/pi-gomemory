package installer

import (
	"strings"
	"testing"
)

// TestHCD018SidebarMinimized verifica que el sidebar minimizado ocupa una sola fila
// con indicador [+] y que /sidebar min está registrado.
func TestHCD018SidebarMinimized(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import sidebarExtensionFactory, { renderSidebarMinimized } from "./gomemory-sidebar.ts";

const theme = { fg: (_c, s) => s, style: (s) => s };
const state = {
	title: "t", tokens: 12000, contextLimit: 200000, cost: 0.5, servers: [],
	providerName: "p", isAntigravityProvider: false, quota: null, quotaError: undefined,
	isClaudeProvider: false, claudeUsage: null,
};
const lines = renderSidebarMinimized(state, 36, theme);

const fails = [];
if (lines.length !== 1) fails.push("minimizado debe ocupar 1 fila, tiene " + lines.length);
if (!lines[0].includes("[+]")) fails.push("falta indicador [+] para restaurar");

const commands = {};
const fakePi = {
	registerCommand: (name, def) => { commands[name] = def; },
	registerTool: () => {},
	on: () => () => {},
	exec: async () => ({ code: 0, stdout: "" }),
};
sidebarExtensionFactory(fakePi);
if (!commands["sidebar"]) fails.push("comando /sidebar no registrado");

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD018SidebarMinimized fallo: %s", out)
	}
}

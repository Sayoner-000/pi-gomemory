package installer

import (
	"strings"
	"testing"
)

// Desde HCD-085 no hay panel /tdd: /min alterna solo el sidebar, y /tdd report
// se retiró con /tdd.
func TestHCD022MinBothPanels(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import sidebarExt from "./gomemory-sidebar.ts";
const commands = {}, notes = [], overlays = {};
const pi = {
	registerCommand: (n, d) => (commands[n] = d),
	registerTool: () => {},
	on: () => () => {},
	exec: async () => ({ code: 1, stdout: "", stderr: "" }),
	sendMessage: () => {},
};
sidebarExt(pi);
const theme = { fg: (_c, s) => s, style: (s) => s };
const tui = { terminal: { rows: 60, columns: 160 }, requestRender: () => {} };
const ctx = {
	hasUI: true, cwd: process.cwd(),
	ui: {
		notify: (m) => notes.push(m), setWidget: () => {},
		custom: (factory, opts) => new Promise((resolve) => { overlays.sidebar = factory(tui, theme, {}, resolve); }),
	},
};
const rows = (w) => overlays.sidebar.render(w).length;
const fails = [];
await commands.sidebar.handler("", ctx);
await new Promise((r) => setTimeout(r, 20));
if (commands.tdd) fails.push("HCD-085: /tdd ya no debe registrarse");
if (rows(36) < 3) fails.push("al empezar el sidebar debe estar expandido");
await commands.min.handler("", ctx);
if (rows(36) !== 1) fails.push("/min debe minimizar el sidebar");
await commands.min.handler("", ctx);
if (rows(36) < 3) fails.push("un segundo /min debe restaurarlo");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD022MinBothPanels: %s", out)
	}
}

func TestHCD022StatusNoEmoji(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-status.ts"}, `
import statusExt from "./gomemory-status.ts";
const handlers = {};
statusExt({ registerCommand: () => {}, on: (e, h) => { handlers[e] = h; return () => {}; } });
let footer;
const ctx = {
	thinkingLevel: "high",
	model: { id: "claude-test", contextWindow: 200000 },
	sessionManager: { getBranch: () => [{ type: "message", message: { role: "assistant", usage: { input: 1200, output: 300, cacheRead: 5000, cost: { total: 0.01 } } } }] },
	ui: { setFooter: (f) => (footer = f) },
};
await handlers.session_start({}, ctx);
const theme = { fg: (_c, s) => s, style: (s) => s };
const comp = footer({ requestRender() {} }, theme, { getGitBranch: () => "main" });
const line = comp.render(200).join("");
const fails = [];
const emoji = line.match(/[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}\u{231A}-\u{23FF}\u{2B00}-\u{2BFF}]/gu) || [];
if (emoji.length) fails.push("la barra inferior usa emoji: " + [...new Set(emoji)].join(" "));
if (!/5\.0k/.test(line)) fails.push("debe seguir mostrando las lecturas de caché");
if (!/high/.test(line)) fails.push("debe seguir mostrando el nivel de thinking");
console.log(fails.length ? "FAIL " + fails.join(" | ") + " :: " + line : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD022StatusNoEmoji: %s", out)
	}
}

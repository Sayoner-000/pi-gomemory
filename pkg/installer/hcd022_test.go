package installer

import (
	"path/filepath"
	"strings"
	"testing"
)

// hcd022Panels abre /sidebar y /tdd en el mismo proceso (como en Pi: dos
// extensiones, un globalThis) con ctx.ui.custom simulado.
const hcd022Panels = `
import sidebarExt from "./gomemory-sidebar.ts";
import tddExt from "./gomemory-tdd.ts";
const commands = {}, notes = [], overlays = {};
const pi = {
	registerCommand: (n, d) => (commands[n] = d),
	registerTool: () => {},
	on: () => () => {},
	exec: async () => ({ code: 1, stdout: "", stderr: "" }),
	sendMessage: () => {},
};
sidebarExt(pi);
tddExt(pi);
const theme = { fg: (_c, s) => s, style: (s) => s };
const tui = { terminal: { rows: 60, columns: 160 }, requestRender: () => {} };
let opening = "";
const ctx = {
	hasUI: true, cwd: process.cwd(),
	ui: {
		notify: (m) => notes.push(m), setWidget: () => {},
		custom: (factory, opts) => new Promise((resolve) => { overlays[opening] = factory(tui, theme, {}, resolve); }),
	},
};
const rows = (name, w) => overlays[name].render(w).length;
opening = "sidebar"; await commands.sidebar.handler("", ctx);
opening = "tdd"; await commands.tdd.handler("min", ctx); await commands.tdd.handler("min", ctx); // abre y deja expandido
await new Promise((r) => setTimeout(r, 20));
const fails = [];
`

func TestHCD022MinBothPanels(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts", "extensions/gomemory-tdd.ts"}, hcd022Panels+`
if (rows("sidebar", 36) < 3 || rows("tdd", 44) < 3) fails.push("al empezar ambos paneles deben estar expandidos");

await commands.min.handler("", ctx);
if (rows("sidebar", 36) !== 1) fails.push("/min debe minimizar el sidebar");
if (rows("tdd", 44) !== 1) fails.push("/min debe minimizar también el panel TDD");

await commands.min.handler("", ctx);
if (rows("sidebar", 36) < 3 || rows("tdd", 44) < 3) fails.push("con ambos minimizados, /min debe restaurar los dos");

// Estados mezclados: si alguno está expandido, /min minimiza ambos.
await commands.tdd.handler("min", ctx);
await commands.min.handler("", ctx);
if (rows("sidebar", 36) !== 1 || rows("tdd", 44) !== 1) fails.push("con uno expandido, /min debe dejar ambos minimizados");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD022MinBothPanels: %s", out)
	}
}

func TestHCD022TddReport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module rep\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "calc", "calc_test.go"), `package calc

import "testing"

func TestRoto(t *testing.T) { t.Fatalf("got 1, want 2") }
`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import factory from "./gomemory-tdd.ts";
const commands = {}, notes = [], sent = [];
let tool;
factory({
	registerCommand: (n, d) => (commands[n] = d),
	registerTool: (d) => (tool = d),
	on: () => () => {},
	exec: async () => { throw new Error("pi.exec no se usa"); },
	sendMessage: (msg, opts) => sent.push({ msg, opts }),
});
const ctx = { hasUI: false, cwd: `+jsStr(dir)+`, ui: { notify: (m) => notes.push(m) } };
const fails = [];

await commands.tdd.handler("report", ctx);
if (sent.length) fails.push("sin ejecución previa no debe enviar un informe vacío");
if (!notes.some((m) => /tdd run/.test(m))) fails.push("sin ejecución previa debe indicar /tdd run: " + JSON.stringify(notes));

await tool.execute("c1", { suite: "backend" }, undefined, undefined, ctx);
await commands.tdd.handler("report", ctx);
const r = sent[0];
if (!r) fails.push("/tdd report no envió el informe");
else {
	if (r.msg.display !== true) fails.push("el informe debe mostrarse en el chat (display: true)");
	if (r.opts?.triggerTurn === true) fails.push("el informe no debe disparar un turno del agente");
	for (const want of ["TestRoto", "calc_test.go:", "want 2"]) if (!String(r.msg.content).includes(want)) fails.push("el informe no incluye " + want);
}
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD022TddReport: %s", out)
	}
}

func TestHCD022StatusNoEmoji(t *testing.T) {
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

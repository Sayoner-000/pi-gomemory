package installer

import (
	"strings"
	"testing"
)

// hcd019Harness monta una extensión con un ctx.ui.custom simulado que guarda el
// componente del overlay y sus overlayOptions, y emula la secuencia de ratón de
// pi-tui en modo fullscreen: un clic llega como press → release → click.
const hcd019Harness = `
const DOCK = Symbol.for("pi-gomemory.dock");
const commands = {};
const fakePi = {
	registerCommand: (name, def) => { commands[name] = def; },
	registerTool: () => {},
	on: () => () => {},
	exec: async () => ({ code: 1, stdout: "", stderr: "" }),
};
const theme = { fg: (_c, s) => s, style: (s) => s };
const forced = [];
const tui = {
	terminal: { rows: 40, columns: 160 },
	requestRender: (force) => { forced.push(!!force); },
};
let overlay = null;
const ctx = {
	hasUI: true,
	cwd: process.cwd(),
	ui: {
		notify: () => {},
		setWidget: () => {},
		custom: (factory, opts) => new Promise((resolve) => {
			overlay = { component: factory(tui, theme, {}, resolve), opts };
		}),
	},
};
const ev = (type, y) => ({ type, button: "left", x: 2, y, screenX: 0, screenY: y, width: 36, height: 30, shift: false, alt: false, ctrl: false });
function click(y) {
	const c = overlay.component;
	const r = c.handleMouse(ev("press", y));
	if (!r || !r.handled) return "press no manejado";
	c.handleMouse(ev("release", y));
	c.handleMouse(ev("click", y));
	return "";
}
const fails = [];
`

// TestHCD019SidebarClickToggle: un clic en la cabecera minimiza una sola vez,
// el minimizado ocupa 1 fila y un clic sobre ella restaura. El sidebar publica
// su altura en el dock compartido.
func TestHCD019SidebarClickToggle(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import factory from "./gomemory-sidebar.ts";
`+hcd019Harness+`
factory(fakePi);
await commands.sidebar.handler("", ctx);
await new Promise((r) => setTimeout(r, 10));
if (!overlay) { console.log("FAIL overlay no abierto"); process.exit(0); }

const expanded = overlay.component.render(36);
if (expanded.length < 3) fails.push("expandido debe tener varias filas");
const dock = globalThis[DOCK];
if (!dock || dock.sidebarHeight !== expanded.length) fails.push("dock.sidebarHeight debe ser " + expanded.length + ", es " + (dock && dock.sidebarHeight));

let e = click(1);
if (e) fails.push(e);
const min = overlay.component.render(36);
if (min.length !== 1) fails.push("tras un clic en la cabecera debe quedar minimizado en 1 fila, tiene " + min.length);
if (!min[0].includes("[+]")) fails.push("minimizado sin indicador [+]");
if (globalThis[DOCK]?.sidebarHeight !== 1) fails.push("dock no refleja la altura minimizada");

e = click(0);
if (e) fails.push(e);
if (overlay.component.render(36).length !== expanded.length) fails.push("un clic en [+] debe restaurar el panel");

if (overlay.component.handleMouse(ev("click", 8)) !== undefined) fails.push("un clic fuera de la cabecera no debe alternar");
if (!overlay.opts.overlayOptions.visible(160, 40) || !globalThis[DOCK]?.sidebarVisible) fails.push("sidebar visible debe publicarse en el dock");

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD019SidebarClickToggle: %s", out)
	}
}

// TestHCD019TDDClickToggleAndDock se retiró con /tdd en HCD-085.

package installer

import (
	"strings"
	"testing"
)

// HCD-076 T20 · C-12: /componentes lista los componentes de `components
// --json`, deja marcar y desmarcar los tool (no los obligatorios), ejecuta
// `sync --json --only=…` con un plazo de 60 min, avisa de reiniciar Pi si pi
// quedó actualizado aunque no estuviera seleccionado, y notifica el plazo
// vencido.
func TestHCD076ComponentesCommand(t *testing.T) {
	t.Parallel()
	out := runNodeExt(t, []string{"extensions/gomemory-status.ts"}, `
import ext from "./gomemory-status.ts";

const commands = {}, execs = [], notes = [], menus = [];
const components = [
	{ name: "pi-gomemory", kind: "self", required: true, installed: true, current: "v1.0.0", latest: "v1.1.0", outdated: true, error: "" },
	{ name: "pi", kind: "tool", required: true, installed: true, current: "1.0.0", latest: "1.2.0", outdated: true, error: "" },
	{ name: "codegraph", kind: "tool", required: false, installed: true, current: "1.0.0", latest: "1.0.0", outdated: false, error: "" },
	{ name: "herdr", kind: "tool", required: false, installed: false, current: "", latest: "v0.9.5", outdated: false, error: "" },
	{ name: "pi-antigravity", kind: "package", required: false, installed: true, current: "0.15.0", latest: "0.15.0", outdated: false, error: "" },
];
let syncResult = { code: 0, killed: false, stdout: JSON.stringify([
	{ name: "pi", action: "updated", from: "1.0.0", to: "1.2.0" },
	{ name: "codegraph", action: "current", from: "1.0.0", to: "1.0.0" },
]), stderr: "" };
const pi = {
	on: () => {},
	registerCommand: (name, def) => { commands[name] = def; },
	exec: async (cmd, args, opts) => {
		execs.push({ line: cmd + " " + args.join(" "), timeout: opts?.timeout });
		if (args[0] === "components") return { code: 0, stdout: JSON.stringify(components), stderr: "" };
		if (args[0] === "sync") return syncResult;
		return { code: 1, stdout: "", stderr: "" };
	},
};
// El usuario pulsa pi (obligatorio, no se desmarca), desmarca herdr y aplica.
let script = [];
const pick = (needle) => (opts) => opts.find((o) => o.includes(needle));
const ctx = {
	hasUI: true,
	ui: {
		select: async (title, opts) => { menus.push(opts); const f = script.shift(); return f ? f(opts) : undefined; },
		notify: (msg, level) => notes.push([level, msg]),
		setStatus: () => {}, setFooter: () => {},
	},
};
ext(pi);
script = [pick(" pi "), pick("herdr"), pick("Aplicar")];
await commands.componentes.handler("", ctx);
console.log("MENU=" + JSON.stringify(menus[0]));
console.log("LASTMENU=" + JSON.stringify(menus.at(-1)));
console.log("EXECS=" + JSON.stringify(execs));
console.log("NOTES=" + JSON.stringify(notes));

execs.length = 0; notes.length = 0; menus.length = 0;
syncResult = { code: 143, killed: true, stdout: "", stderr: "" };
script = [pick("Aplicar")];
await commands.componentes.handler("", ctx);
console.log("TIMEOUT=" + JSON.stringify(notes));
`)
	lines := map[string]string{}
	for _, l := range strings.Split(out, "\n") {
		if k, v, ok := strings.Cut(l, "="); ok {
			lines[k] = v
		}
	}
	menu := lines["MENU"]
	for _, want := range []string{"pi-gomemory", "v1.0.0 → v1.1.0", "[x] pi ", "1.0.0 → 1.2.0", "obligatorio", "[x] codegraph", "opcional", "[x] herdr", "no instalado", "pi-antigravity", "Aplicar"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("falta %q en el menú:\n%s", want, menu)
		}
	}
	if !strings.Contains(lines["LASTMENU"], "[ ] herdr") || !strings.Contains(lines["LASTMENU"], "[x] pi ") {
		t.Fatalf("herdr debe quedar desmarcado y pi marcado:\n%s", lines["LASTMENU"])
	}
	execs := lines["EXECS"]
	if !strings.Contains(execs, `"line":"pi-gomemory-setup components --json"`) ||
		!strings.Contains(execs, `"line":"pi-gomemory-setup sync --json --only=pi,codegraph","timeout":3600000`) {
		t.Fatalf("ejecuciones = %s", execs)
	}
	notes := lines["NOTES"]
	for _, want := range []string{"obligatorio", "pi actualizado", "Reinicia Pi"} {
		if !strings.Contains(notes, want) {
			t.Fatalf("falta %q en las notificaciones: %s", want, notes)
		}
	}
	if !strings.Contains(lines["TIMEOUT"], "ejecuta pi-gomemory-setup sync en la terminal") {
		t.Fatalf("plazo vencido: %s", lines["TIMEOUT"])
	}
}

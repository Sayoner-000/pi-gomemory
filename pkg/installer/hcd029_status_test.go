package installer

import (
	"strings"
	"testing"
)

// HCD-029: el pie de Pi anuncia «↑ actualizaciones disponibles» en su extremo
// derecho cuando deps-status sale con 10. Al iniciar sesión consulta, refresca
// en segundo plano y vuelve a consultar, así que una release publicada después
// de la última consulta aparece en la misma apertura. Con /tokens el pie propio
// se apaga y el aviso pasa a setStatus para el pie por defecto de Pi.
func TestHCD029StatusExtensionUpdateIndicator(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-status.ts"}, `
import ext from "./gomemory-status.ts";

const handlers = {}, commands = {}, execs = [], statuses = [];
// La caché anterior estaba al día (0); tras deps-refresh hay una release (10).
const codes = { "deps-status": [0, 10], "deps-refresh": [0] };
const pi = {
	on: (ev, fn) => { handlers[ev] = fn; },
	registerCommand: (name, def) => { commands[name] = def; },
	exec: async (cmd, args) => {
		execs.push(cmd + " " + args.join(" "));
		const q = codes[args[0]] ?? [1];
		return { code: q.length > 1 ? q.shift() : q[0], stdout: "", stderr: "" };
	},
};
let footer;
const ctx = {
	hasUI: true,
	ui: {
		setFooter: (f) => { footer = f; },
		setStatus: (k, v) => statuses.push([k, v ?? null]),
		notify: () => {},
	},
	sessionManager: { getBranch: () => [] },
};
const theme = { fg: (_r, s) => s, style: (s) => s };
const tui = { requestRender: () => {} };
const settle = () => new Promise((r) => setTimeout(r, 20));
const render = () => footer(tui, theme, {}).render(140)[0];

ext(pi);
await handlers.session_start({}, ctx);
await settle();
console.log("EXECS=" + JSON.stringify(execs));
console.log("FOOTER=" + render());

await commands.tokens.handler("", ctx);
console.log("OFF=" + JSON.stringify(statuses.at(-1)));
await commands.tokens.handler("", ctx);
console.log("ON=" + JSON.stringify(statuses.at(-1)));
`)
	lines := map[string]string{}
	for _, l := range strings.Split(out, "\n") {
		if k, v, ok := strings.Cut(l, "="); ok {
			lines[k] = v
		}
	}

	const label = "↑ actualizaciones disponibles"
	want := `["pi-gomemory-setup deps-status","pi-gomemory-setup deps-refresh","pi-gomemory-setup deps-status"]`
	if lines["EXECS"] != want {
		t.Errorf("secuencia de consultas = %s; quiero %s", lines["EXECS"], want)
	}
	footer := lines["FOOTER"]
	i := strings.Index(footer, label)
	if i < 0 {
		t.Fatalf("el pie no anuncia la actualización:\n%s", footer)
	}
	if left := strings.Index(footer, "⟦mem⟧"); left > i || !strings.Contains(footer[i:], "default") {
		t.Errorf("el aviso debe ir en el lado derecho, delante del modelo:\n%s", footer)
	}
	if lines["OFF"] != `["gomemory-updates","`+label+`"]` {
		t.Errorf("con /tokens apagado el aviso pasa a setStatus; obtuve %s", lines["OFF"])
	}
	if lines["ON"] != `["gomemory-updates",null]` {
		t.Errorf("con el pie propio de vuelta, setStatus se limpia; obtuve %s", lines["ON"])
	}
}

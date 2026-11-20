package installer

import (
	"strings"
	"testing"
)

// HCD-061: con el panel visible, /tasks vuelve a consultar mem hcd show cada
// 10 s y se oculta cuando la unidad ya no está activa, aunque se cerrara desde
// otra sesión. El latido se detiene al ocultarse y al cerrar la sesión. La
// prueba captura el temporizador en lugar de esperar 10 s reales.
func TestHCD061Heartbeat(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
const timers = [];
const realSet = globalThis.setInterval, realClear = globalThis.clearInterval;
globalThis.setInterval = (fn, ms) => { const t = { fn, ms, cleared: false, unref() { return t; } }; timers.push(t); return t; };
globalThis.clearInterval = (t) => { if (t && typeof t === "object" && "cleared" in t) t.cleared = true; else realClear(t); };
const { default: registerTasks } = await import("./gomemory-tasks.ts");
const fails = [];
let show = { ok: true, unit_id: "HCD-061", kind: "fix", stage: "implement", state: "active", tasks: [{ id: "1", text: "uno", status: "pending" }] };
let calls = 0;
const h = {};
const fakePi = { registerCommand() {}, registerShortcut() {}, on(n, fn) { h[n] = fn; }, async exec(cmd, args) { if (cmd === "mem") calls++; return { code: 0, stdout: JSON.stringify(show) }; } };
registerTasks(fakePi);
let comp = null;
const ctx = { cwd: process.cwd(), ui: { notify() {}, setWidget(_id, c) { comp = typeof c === "function" ? c({ requestRender() {} }, { fg: (_c, s) => s }) : null; } } };
const beats = () => timers.filter((t) => t.ms === 10000);
const live = () => beats().filter((t) => !t.cleared);

await h.session_start({}, ctx);
if (!comp) fails.push("session_start debe mostrar el panel con una unidad activa");
if (live().length !== 1) fails.push("con el panel visible debe haber un latido de 10 s activo: " + live().length);

// Otra sesión cierra la unidad; esta no ejecuta nada. Al llegar el latido, se oculta.
const before = calls;
show = { ok: true, unit_id: "HCD-061", kind: "fix", stage: "close", state: "closed", tasks: [{ id: "1", text: "uno", status: "verified" }] };
await live()[0]?.fn();
await new Promise((r) => setTimeout(r, 10));
if (calls === before) fails.push("el latido debe consultar mem hcd show");
if (comp) fails.push("con la unidad cerrada el panel debe ocultarse");
if (live().length !== 0) fails.push("oculto el panel, el latido debe detenerse: " + live().length);

// Una unidad activa de nuevo (turno propio) vuelve a montar el panel y su latido; cerrar la sesión lo detiene.
show = { ok: true, unit_id: "HCD-062", kind: "fix", stage: "implement", state: "active", tasks: [{ id: "1", text: "dos", status: "pending" }] };
await h.agent_end({}, ctx);
if (!comp || live().length !== 1) fails.push("una unidad activa debe volver a montar el panel con un latido: " + live().length);
await h.session_shutdown({}, ctx);
if (live().length !== 0) fails.push("session_shutdown debe detener el latido");
globalThis.setInterval = realSet;
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD061Heartbeat: %s", out)
	}
}

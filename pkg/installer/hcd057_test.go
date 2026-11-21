package installer

import (
	"strings"
	"testing"
)

// hcd057Unit es una unidad HCD con tareas en todos los estados que pinta /tasks.
const hcd057Unit = `
const unit = {
	ok: true, unit_id: "HCD-057", kind: "feature", stage: "implement", state: "active",
	tasks: [
		{ id: "1", text: "Verificada", status: "verified" },
		{ id: "2", text: "En curso", status: "pending" },
		{ id: "3", text: "Caducada", status: "unverified" },
		{ id: "4", text: "Fallida", status: "failed" },
		{ id: "5", text: "Pendiente", status: "pending" },
	],
};
`

// HCD-057 tarea 1: spinner animado para tarea y llamada en curso, color del tema
// por estado sin quitar el glifo, y conectores ├─/└─ con el último hijo en └─.
func TestHCD057Glyphs(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import registerTasks, { createTaskActivityTracker, renderTasksTree, SPINNER_FRAMES, TASK_FRAMES } from "./gomemory-tasks.ts";
`+hcd057Unit+`
const fails = [];
// Q5: círculo que se llena; la tarea activa se anima aunque no haya llamada en curso.
// Desde HCD-060 parpadea ●/○ cada 500 ms en lugar de girar ◐◓◑◒.
if (!Array.isArray(TASK_FRAMES) || TASK_FRAMES.join("") !== "●○") fails.push("TASK_FRAMES debe ser ●○: " + TASK_FRAMES);
{
	const tr = createTaskActivityTracker();
	tr.syncUnit(unit);
	const a = renderTasksTree(unit, tr, 0)?.find((l) => / 2\. /.test(l.text))?.text ?? "";
	const b = renderTasksTree(unit, tr, 500)?.find((l) => / 2\. /.test(l.text))?.text ?? "";
	if (!(TASK_FRAMES ?? []).some((f) => a.includes(f + " 2."))) fails.push("la tarea activa sin llamadas debe girar: " + a);
	if (a === b) fails.push("el giro de la tarea activa debe avanzar con el reloj");
}
if (!Array.isArray(SPINNER_FRAMES) || SPINNER_FRAMES.length !== 10) fails.push("debe exportar los 10 frames del spinner de /tdd");
const frames = SPINNER_FRAMES ?? [];
const tracker = createTaskActivityTracker();
tracker.syncUnit(unit);
tracker.recordStart("c1", "read", { path: "a.ts" }, unit, 1000);
tracker.recordFinish("c1", false, "x\n", 1010);
tracker.recordStart("c2", "bash", { command: "go test ./..." }, unit, 1020);

const at = (now) => renderTasksTree(unit, tracker, now) ?? [];
const row = (tree, id) => tree.find((l) => new RegExp("^\\s+\\S+ " + id + "\\. ").test(l.text));
const t0 = at(0), t1 = at(500);

// Tarea en curso: círculo que gira con el reloj; la llamada, spinner de /tdd. Nunca ⟳ fijo.
const r0 = row(t0, "2")?.text ?? "", r1 = row(t1, "2")?.text ?? "";
if (!(TASK_FRAMES ?? []).some((f) => r0.includes(f + " 2."))) fails.push("la tarea en curso debe girar ◐◓◑◒: " + r0);
if (r0 === r1) fails.push("el giro de la tarea debe avanzar con el reloj");
if (t0.some((l) => l.text.includes("⟳"))) fails.push("no debe quedar ⟳ fijo");
const call = t0.find((l) => l.text.includes("Bash go test"))?.text ?? "";
if (!frames.some((f) => call.includes("├─ " + f + " Bash"))) fails.push("la llamada en curso debe ir con ├─ y spinner: " + call);
if (!t0.some((l) => l.text.includes("├─ ✓ Read a.ts"))) fails.push("la llamada terminada debe ir con ├─ ✓");
const kids = t0.filter((l) => /^\s+[├└]─ /.test(l.text));
if (!kids.length || !kids[kids.length - 1].text.trimStart().startsWith("└─ ")) fails.push("el último hijo debe cerrar con └─: " + JSON.stringify(kids.map((k) => k.text)));
if (t0.some((l) => /^\s+[├└] /.test(l.text))) fails.push("no deben quedar conectores sin ─");

// Tono por estado; el glifo sigue presente (el color no es el único portador).
const tone = (id) => row(t0, id)?.tone;
const want = { "1": "success", "2": "accent", "3": "warning", "4": "error", "5": "dim" };
for (const [id, tn] of Object.entries(want)) if (tone(id) !== tn) fails.push("tarea " + id + " tono " + tone(id) + ", se esperaba " + tn);
const glyphs = { "1": "●", "3": "↻", "4": "✗", "5": "○" }; // HCD-068: caducada ↻
for (const [id, g] of Object.entries(glyphs)) if (!row(t0, id)?.text.includes(g + " " + id + ".")) fails.push("tarea " + id + " sin glifo " + g);

// El widget colorea con el tema y anima el spinner mientras hay algo en curso.
let comp, renders = 0;
const fakePi = { registerCommand() {}, on(n, fn) { (fakePi.h ||= {})[n] = fn; }, async exec() { return { code: 0, stdout: JSON.stringify(unit) }; } };
registerTasks(fakePi);
const ctx = { cwd: process.cwd(), ui: { notify() {}, setWidget(_id, c) { comp = typeof c === "function" ? c({ requestRender() { renders++; } }, { fg: (c2, s) => "<" + c2 + ">" + s }) : null; } } };
await fakePi.h.session_start({}, ctx);
await fakePi.h.tool_execution_start({ toolCallId: "w1", toolName: "bash", args: { command: "go test ./..." } }, ctx);
const lines = comp?.render(160) ?? [];
if (!lines.some((l) => l.includes("<success>") && l.includes("● 1."))) fails.push("la verificada debe pintarse con success y conservar ●: " + JSON.stringify(lines));
const before = renders;
await new Promise((r) => setTimeout(r, 300));
if (renders - before < 2) fails.push("con algo en curso el widget debe repintar solo para animar: " + (renders - before));
await fakePi.h.tool_execution_end({ toolCallId: "w1", toolName: "bash", isError: false, result: {} }, ctx);
// Sin llamadas en curso, la tarea activa sigue girando: el widget sigue repintando.
const idle = renders;
await new Promise((r) => setTimeout(r, 1100));
if (renders - idle < 2) fails.push("con una tarea activa el widget debe seguir animando sin llamadas: " + (renders - idle));
await fakePi.h.session_shutdown?.({}, ctx);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD057Glyphs: %s", out)
	}
}

// HCD-057 tarea 2: la cabecera de /tasks muestra una barra de 10 celdas con
// hechas/total y porcentaje, sin perder unidad, tipo y paso.
func TestHCD057Progress(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import { renderTasksTree } from "./gomemory-tasks.ts";
const fails = [];
const head = (statuses) => renderTasksTree({
	ok: true, unit_id: "HCD-057", kind: "feature", stage: "implement", state: "active",
	tasks: statuses.map((s, i) => ({ id: String(i + 1), text: "t" + i, status: s })),
})?.[0]?.text ?? "";
const cases = [
	// Barra y números separados por dos espacios (la persona: «muy pegado»).
	// HCD-068: línea continua ━ (lleno) / ─ (vacío).
	[["verified", "pending", "pending", "pending", "pending"], "━━────────  1/5 · 20 %"],
	[["pending", "pending", "pending"], "──────────  0/3 · 0 %"],
	[["verified", "verified"], "━━━━━━━━━━  2/2 · 100 %"],
	[Array.from({ length: 13 }, (_, i) => (i < 5 ? "verified" : "pending")), "━━━━──────  5/13 · 38 %"],
];
for (const [st, want] of cases) {
	const h = head(st);
	if (!h.includes(want)) fails.push("esperaba «" + want + "» en «" + h + "»");
	if (!h.startsWith("HCD-057 · feature · paso implement")) fails.push("la cabecera pierde unidad, tipo o paso: " + h);
}
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD057Progress: %s", out)
	}
}

// hcd057Many construye una unidad de n tareas con las k primeras verificadas.
const hcd057Many = `
const many = (n, k) => ({
	ok: true, unit_id: "HCD-057", kind: "feature", stage: "implement", state: "active",
	tasks: Array.from({ length: n }, (_, i) => ({ id: String(i + 1), text: "Tarea " + (i + 1), status: i < k ? "verified" : "pending" })),
});
const taskIds = (tree) => tree.map((l) => l.text.match(/^\s+\S+ (\d+)\. /)?.[1]).filter(Boolean);
`

// HCD-057 tarea 3: con más de 8 tareas, /tasks muestra una ventana de 8 que
// sigue a la tarea activa, con «↑ N más» y «↓ N más» para las ocultas.
func TestHCD057TaskWindow(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import { renderTasksTree, createTaskActivityTracker, TASK_WINDOW } from "./gomemory-tasks.ts";
`+hcd057Many+`
const fails = [];
if (TASK_WINDOW !== 8) fails.push("la ventana debe ser de 8 tareas: " + TASK_WINDOW);
const view = (u) => { const tr = createTaskActivityTracker(); tr.syncUnit(u); return renderTasksTree(u, tr) ?? []; };

// 13 tareas, activa la 6: tareas 4..11, 3 ocultas arriba y 2 abajo.
const t13 = view(many(13, 5));
if (taskIds(t13).join(",") !== "4,5,6,7,8,9,10,11") fails.push("ventana con activa 6: " + taskIds(t13).join(","));
const up = t13.find((l) => l.text.includes("↑"));
const down = t13.find((l) => l.text.includes("↓"));
if (!up?.text.includes("↑ 3 más")) fails.push("falta «↑ 3 más»: " + up?.text);
if (!down?.text.includes("↓ 2 más")) fails.push("falta «↓ 2 más»: " + down?.text);
if (t13.indexOf(up) !== 1) fails.push("«↑» va justo bajo la cabecera");
if (t13.indexOf(down) !== t13.length - 1) fails.push("«↓» va al final");
if (up?.tone !== "dim" || down?.tone !== "dim") fails.push("los indicadores van atenuados");
if (up?.action?.kind !== "scroll" || down?.action?.kind !== "scroll") fails.push("los indicadores deben ser clicables (scroll)");

// Activa al final: la ventana no se sale de la lista.
const end = view(many(13, 12));
if (taskIds(end).join(",") !== "6,7,8,9,10,11,12,13") fails.push("ventana con activa 13: " + taskIds(end).join(","));
if (end.some((l) => l.text.includes("↓"))) fails.push("sin ocultas abajo no hay «↓»");

// 8 tareas o menos: todo visible, sin indicadores.
const eight = view(many(8, 0));
if (taskIds(eight).length !== 8 || eight.some((l) => /[↑↓]/.test(l.text))) fails.push("con 8 tareas se ven todas sin indicadores");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD057TaskWindow: %s", out)
	}
}

// HCD-057 tarea 4: la ventana se desplaza con la rueda sobre el panel, con
// Alt+Shift+↑/↓ (Q4) y con clic en «↑/↓ N más»; ↑/↓ y Alt+↑ quedan para Pi.
func TestHCD057Scroll(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import registerTasks from "./gomemory-tasks.ts";
`+hcd057Many+`
const fails = [];
const unit = many(13, 5);
const shortcuts = {}, h = {};
let comp;
const fakePi = {
	registerCommand() {},
	registerShortcut(key, def) { shortcuts[key] = def; },
	on(n, fn) { h[n] = fn; },
	async exec() { return { code: 0, stdout: JSON.stringify(unit) }; },
};
registerTasks(fakePi);
const ctx = { cwd: process.cwd(), ui: { notify() {}, setWidget(_id, c) { comp = typeof c === "function" ? c({ requestRender() {} }, { fg: (_c, s) => s }) : null; } } };
await h.session_start({}, ctx);
const ids = () => (comp?.render(160) ?? []).map((l) => l.match(/^\s+\S+ (\d+)\. /)?.[1]).filter(Boolean).join(",");
const wheel = (d) => comp?.handleMouse({ type: "wheel", button: "none", wheelDelta: d, x: 2, y: 3, shift: false, alt: false, ctrl: false });

if (ids() !== "4,5,6,7,8,9,10,11") fails.push("inicio: " + ids());
if (!wheel(1)?.handled) fails.push("la rueda sobre el panel debe manejarse");
if (ids() !== "5,6,7,8,9,10,11,12") fails.push("rueda abajo 1: " + ids());
wheel(-10);
if (ids() !== "1,2,3,4,5,6,7,8") fails.push("rueda arriba hasta el tope: " + ids());
wheel(50);
if (ids() !== "6,7,8,9,10,11,12,13") fails.push("rueda abajo hasta el tope: " + ids());

for (const k of ["alt+shift+up", "alt+shift+down"]) if (typeof shortcuts[k]?.handler !== "function") fails.push("falta el atajo " + k);
for (const k of ["up", "down", "alt+up", "alt+down"]) if (shortcuts[k]) fails.push("no debe tomar " + k);
await shortcuts["alt+shift+up"]?.handler(ctx);
if (ids() !== "5,6,7,8,9,10,11,12") fails.push("Alt+Shift+↑: " + ids());
await shortcuts["alt+shift+down"]?.handler(ctx);
if (ids() !== "6,7,8,9,10,11,12,13") fails.push("Alt+Shift+↓: " + ids());

// Clic en «↑ N más» sube una tarea.
const lines = comp?.render(160) ?? [];
const upRow = lines.findIndex((l) => l.includes("↑"));
const res = comp?.handleMouse({ type: "click", button: "left", x: 3, y: upRow, shift: false, alt: false, ctrl: false });
if (!res?.handled) fails.push("el clic en «↑ N más» debe manejarse");
if (ids() !== "5,6,7,8,9,10,11,12") fails.push("clic en ↑: " + ids());
if (!lines.some((l) => l.includes("Alt+Shift+↑"))) fails.push("el indicador debe nombrar Alt+Shift+↑: " + lines.find((l) => l.includes("↑")));
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD057Scroll: %s", out)
	}
}

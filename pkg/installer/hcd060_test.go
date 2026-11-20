package installer

import (
	"strings"
	"testing"
)

// HCD-060: la tarea en curso de /tasks parpadea alternando ● y ○ cada 500 ms
// (antes giraba ◐◓◑◒); la verificada sigue siendo ● fijo, y el panel repinta
// a ese ritmo cuando solo hay una tarea activa.
func TestHCD060Blink(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import registerTasks, { createTaskActivityTracker, renderTasksTree, TASK_FRAMES } from "./gomemory-tasks.ts";
`+hcd057Unit+`
const fails = [];
if (TASK_FRAMES?.join("") !== "●○") fails.push("TASK_FRAMES debe ser ●○: " + TASK_FRAMES);
const tr = createTaskActivityTracker();
tr.syncUnit(unit);
const row = (now, id) => renderTasksTree(unit, tr, now)?.find((l) => new RegExp(" " + id + "\\. ").test(l.text));
// La activa (2) alterna cada 500 ms; dentro del mismo medio segundo no cambia.
const g = (now) => row(now, "2")?.text.trim()[0];
if (g(0) !== "●" || g(499) !== "●") fails.push("de 0 a 499 ms debe verse ●: " + g(0) + g(499));
if (g(500) !== "○" || g(999) !== "○") fails.push("de 500 a 999 ms debe verse ○: " + g(500) + g(999));
if (g(1000) !== "●") fails.push("a los 1000 ms vuelve ●: " + g(1000));
if (row(0, "2")?.tone !== "accent" || row(500, "2")?.tone !== "accent") fails.push("la activa parpadea en accent");
// La verificada es ● fijo en success, en cualquier instante.
if (row(500, "1")?.text.trim()[0] !== "●" || row(500, "1")?.tone !== "success") fails.push("la verificada es ● fijo en success");

// El panel repinta al ritmo del parpadeo (500 ms), no más deprisa.
let comp, renders = 0;
const fakePi = { registerCommand() {}, registerShortcut() {}, on(n, fn) { (fakePi.h ||= {})[n] = fn; }, async exec() { return { code: 0, stdout: JSON.stringify(unit) }; } };
registerTasks(fakePi);
const ctx = { cwd: process.cwd(), ui: { notify() {}, setWidget(_id, c) { comp = typeof c === "function" ? c({ requestRender() { renders++; } }, { fg: (_c, s) => s }) : null; } } };
await fakePi.h.session_start({}, ctx);
const before = renders;
await new Promise((r) => setTimeout(r, 1100));
const n = renders - before;
if (n < 2 || n > 3) fails.push("en 1,1 s debe repintar 2 veces (cada 500 ms): " + n);
await fakePi.h.session_shutdown?.({}, ctx);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD060Blink: %s", out)
	}
}

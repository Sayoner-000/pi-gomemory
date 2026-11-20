package installer

import "testing"

func TestHCD054ActivityTracking(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import assert from "node:assert/strict";
import {
	formatToolCallLabel,
	countOutputLines,
	extractVerifyTaskIds,
	resolveActiveTaskId,
	createTaskActivityTracker,
} from "./gomemory-tasks.ts";

// 1. Formateo de etiquetas por herramienta (Read con offset/limit, Edit, Bash acotado)
assert.equal(
	formatToolCallLabel("read", { path: "extensions/gomemory-tdd.ts", offset: 1430, limit: 52 }),
	"Read extensions/gomemory-tdd.ts (offset=1430, limit=52)",
);
assert.equal(formatToolCallLabel("edit", { path: "extensions/gomemory-tasks.ts" }), "Edit extensions/gomemory-tasks.ts");
assert.equal(formatToolCallLabel("write", { path: "pkg/installer/hcd054_test.go" }), "Write pkg/installer/hcd054_test.go");
assert.ok(formatToolCallLabel("bash", { command: "go test ./pkg/installer -run TestFoo" }).startsWith("Bash go test"));
assert.ok(!formatToolCallLabel("bash", { command: "API_KEY=secret123 go test ./..." }).includes("secret123"), "no debe exponer secretos en etiquetas");

// 2. Conteo de líneas de salida
assert.equal(countOutputLines(""), 0);
assert.equal(countOutputLines("a\nb\nc\n"), 3);
assert.equal(countOutputLines([{ type: "text", text: "l1\nl2" }, { type: "text", text: "l3\n" }]), 3);
assert.equal(countOutputLines({ content: [{ type: "text", text: "x\ny\n" }] }), 2);

// 3. Extracción de --task en mem hcd verify (ignorando texto citado)
assert.deepEqual(extractVerifyTaskIds("mem hcd verify --task 2 --json -- 'go test'"), ["2"]);
assert.deepEqual(extractVerifyTaskIds("mem hcd verify --task=3,4 --json -- 'ok'"), ["3", "4"]);
assert.deepEqual(extractVerifyTaskIds("printf 'mem hcd verify --task 9'"), []);

// 4. Atribución Q2: primera tarea no verificada, re-fijado con verify --task N y foco manual
const unit = {
	ok: true,
	unit_id: "HCD-054",
	kind: "feature",
	stage: "implement",
	state: "active",
	tasks: [
		{ id: "1", text: "hecha", status: "verified" },
		{ id: "2", text: "en curso", status: "pending" },
		{ id: "3", text: "siguiente", status: "pending" },
	],
};
assert.equal(resolveActiveTaskId(unit, null), "2");
assert.equal(resolveActiveTaskId(unit, "3"), "3");

const tracker = createTaskActivityTracker();
tracker.syncUnit(unit);
assert.equal(tracker.recordStart("c1", "read", { path: "extensions/gomemory-tdd.ts", offset: 1430, limit: 52 }, unit, 1000), "2");
assert.equal(tracker.recordFinish("c1", false, [{ type: "text", text: "a\nb\nc\n" }], 1010), true);
const act2 = tracker.getTaskActivity("2");
assert.equal(act2.totalCalls, 1);
assert.equal(act2.totalOutputLines, 3);
assert.equal(act2.calls[0].status, "done");
assert.equal(act2.calls[0].durationMs, 10);

// verify --task 3 atribuye directamente a la tarea 3
assert.equal(tracker.recordStart("c2", "bash", { command: "mem hcd verify --task 3 --json -- 'go test'" }, unit, 2000), "3");
assert.equal(tracker.recordFinish("c2", true, [{ type: "text", text: "FAIL\n" }], 2045), true);
const act3 = tracker.getTaskActivity("3");
assert.equal(act3.totalCalls, 1);
assert.equal(act3.calls[0].status, "error");
assert.equal(act3.calls[0].durationMs, 45);

// Al cambiar de unidad se limpia el estado anterior en memoria
tracker.syncUnit({ ...unit, unit_id: "HCD-055" });
assert.equal(tracker.getTaskActivity("2").totalCalls, 0);
`)
	t.Log(out)
}

func TestHCD054TreeRendering(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import assert from "node:assert/strict";
import { createTaskActivityTracker, renderTasks, renderTasksTree } from "./gomemory-tasks.ts";

const unit = {
	ok: true,
	unit_id: "HCD-054",
	kind: "feature",
	stage: "implement",
	state: "active",
	tasks: [
		{ id: "1", text: "Tarea completada", status: "verified" },
		{ id: "2", text: "Tarea en curso", status: "pending" },
	],
};

const tracker = createTaskActivityTracker();
tracker.syncUnit(unit);

// 6 llamadas en la tarea 2 (en curso) -> deben mostrarse las últimas 5 + pie con "click to show more"
for (let i = 1; i <= 6; i++) {
	tracker.recordStart("a" + i, "read", { path: "extensions/gomemory-tdd.ts", offset: i * 100, limit: 50 }, unit, 1000 + i * 20);
	tracker.recordFinish("a" + i, false, "line1\nline2\nline3\n", 1000 + i * 20 + 9);
}
// 1 llamada en curso en tarea 2 para comprobar glifo ⟳
tracker.recordStart("a7", "bash", { command: "go test ./pkg/installer" }, unit, 2000);

// 2 llamadas en la tarea 1 (completada), atribuidas con foco temporal
tracker.setFocus("1");
tracker.recordStart("done1", "read", { path: "extensions/gomemory-tasks.ts", offset: 1, limit: 20 }, unit, 3000);
tracker.recordFinish("done1", false, "a\nb\n", 3010);
tracker.setFocus(null);

const tree = renderTasksTree(unit, tracker, 2050);
assert.ok(tree && tree.length > 0);
const lines = tree.map((l) => l.text);

// Q1: tarea 1 (completada) colapsada en una línea con resumen de llamadas y líneas
const task1Row = lines.find((l) => l.includes("✓ 1. Tarea completada"));
assert.ok(task1Row && task1Row.includes("1 call") && task1Row.includes("2 lines output"), "tarea completada debe resumirse colapsada: " + task1Row);
assert.ok(!lines.some((l) => l.includes("gomemory-tasks.ts (offset=1")), "tarea completada colapsada no debe mostrar ramas");

// Q1: tarea 2 (en curso) expandida por defecto con 5 llamadas visibles + pie "click to show more"
const branchLines = lines.filter((l) => l.includes("├ "));
assert.equal(branchLines.length, 5, "debe mostrar tope de 5 llamadas visibles: " + JSON.stringify(branchLines));
assert.ok(branchLines.some((l) => l.includes("⟳ Bash go test ./pkg/installer 50ms")), "llamada en curso debe mostrar ⟳ y duración en vivo: " + JSON.stringify(branchLines));
assert.ok(lines.some((l) => l.includes("└ 7 calls · 18 lines output · click to show more")), "debe mostrar pie resumen con click to show more: " + JSON.stringify(lines));

// Al activar showMore en tarea 2 se muestran todas las llamadas (7) y cambia a "click to show less"
tracker.toggleShowMore("2");
const expandedLines = renderTasks(unit, tracker, 2050);
assert.equal(expandedLines.filter((l) => l.includes("├ ")).length, 7);
assert.ok(expandedLines.some((l) => l.includes("click to show less")));

// Al expandir la tarea 1 completada aparecen sus ramas
tracker.expandTask("1");
const withTask1 = renderTasks(unit, tracker, 2050);
assert.ok(withTask1.some((l) => l.includes("├ ✓ Read extensions/gomemory-tasks.ts (offset=1, limit=20) 10ms")));
`)
	t.Log(out)
}

func TestHCD054WidgetInteraction(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import assert from "node:assert/strict";
import registerTasks from "./gomemory-tasks.ts";

const commands = {};
const handlers = {};
const unitPayload = {
	ok: true,
	unit_id: "HCD-054",
	kind: "feature",
	stage: "implement",
	state: "active",
	tasks: [
		{ id: "1", text: "Primera tarea", status: "verified" },
		{ id: "2", text: "Segunda tarea", status: "pending" },
	],
};

const fakePi = {
	registerCommand(name, def) { commands[name] = def; },
	on(name, fn) { handlers[name] = fn; },
	async exec() {
		return { code: 0, stdout: JSON.stringify(unitPayload), stderr: "" };
	},
};

let mountedComponent = null;
let renderRequests = 0;
const tui = { requestRender() { renderRequests++; } };
const theme = { fg: (_c, s) => s };
const notifications = [];
const ctx = {
	cwd: process.cwd(),
	ui: {
		notify(msg) { notifications.push(msg); },
		setWidget(_id, content) {
			if (typeof content === "function") {
				mountedComponent = content(tui, theme);
			} else if (content === undefined) {
				mountedComponent = null;
			}
		},
	},
};

registerTasks(fakePi);
await handlers.session_start({}, ctx);
assert.ok(mountedComponent, "session_start debe montar el componente interactivo del widget");

// Simular 6 llamadas Read en tarea 2 (activa)
for (let i = 1; i <= 6; i++) {
	const id = "call-" + i;
	await handlers.tool_execution_start({ toolCallId: id, toolName: "read", args: { path: "extensions/gomemory-tdd.ts", offset: i * 10, limit: 20 } }, ctx);
	await handlers.tool_result({ toolCallId: id, toolName: "read", input: { path: "extensions/gomemory-tdd.ts" }, isError: false, content: [{ type: "text", text: "l1\nl2\n" }] }, ctx);
	await handlers.tool_execution_end({ toolCallId: id, toolName: "read", isError: false, result: { content: [{ type: "text", text: "l1\nl2\n" }] } }, ctx);
}

const initialLines = mountedComponent.render(120);
const moreIdx = initialLines.findIndex((l) => l.includes("click to show more"));
assert.ok(moreIdx >= 0, "debe renderizar línea click to show more: " + JSON.stringify(initialLines));
assert.equal(initialLines.filter((l) => l.includes("├ ")).length, 5);

// Clic izquierdo sobre "click to show more" expande las 6 llamadas
const clickRes = mountedComponent.handleMouse({ type: "click", button: "left", y: moreIdx, x: 5 });
assert.equal(clickRes?.handled, true, "el clic sobre click to show more debe ser manejado");
const expandedLines = mountedComponent.render(120);
assert.equal(expandedLines.filter((l) => l.includes("├ ")).length, 6, "tras clic debe mostrar las 6 llamadas");
assert.ok(expandedLines.some((l) => l.includes("click to show less")));

// Subcomando /tasks collapse 2 colapsa la tarea 2
await commands.tasks.handler("collapse 2", ctx);
const collapsedLines = mountedComponent.render(120);
assert.equal(collapsedLines.filter((l) => l.includes("├ ")).length, 0, "collapse 2 debe ocultar las ramas");
assert.ok(collapsedLines.some((l) => l.includes("click to expand")));

// Subcomando /tasks expand 2 vuelve a mostrar las ramas
await commands.tasks.handler("expand 2", ctx);
assert.ok(mountedComponent.render(120).some((l) => l.includes("├ ")));

// Subcomando /tasks focus 1 atribuye la siguiente llamada a la tarea 1
await commands.tasks.handler("focus 1", ctx);
await handlers.tool_execution_start({ toolCallId: "focus-1", toolName: "edit", args: { path: "extensions/gomemory-tasks.ts" } }, ctx);
await handlers.tool_execution_end({ toolCallId: "focus-1", toolName: "edit", isError: false, result: { content: [{ type: "text", text: "ok\n" }] } }, ctx);
await commands.tasks.handler("expand 1", ctx);
assert.ok(mountedComponent.render(120).some((l) => l.includes("Edit extensions/gomemory-tasks.ts")));

// /tasks sin argumentos sigue ocultando el widget por completo
await commands.tasks.handler("", ctx);
assert.equal(mountedComponent, null, "/tasks sin argumentos debe ocultar el widget");
`)
	t.Log(out)
}

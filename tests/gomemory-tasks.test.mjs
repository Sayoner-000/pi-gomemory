import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";

registerHooks({
	resolve(specifier, context, nextResolve) {
		if (specifier === "@earendil-works/pi-tui") {
			return {
				shortCircuit: true,
				url: "data:text/javascript,export function truncateToWidth(s){return s;}",
			};
		}
		return nextResolve(specifier, context);
	},
});

const {
	createTaskActivityTracker,
	renderTasksTree,
	resolveActiveTaskId,
} = await import("../extensions/gomemory-tasks.ts");

function makeUnit(tasks) {
	return {
		ok: true,
		unit_id: "HCD-099",
		kind: "feature",
		stage: "implement",
		state: "active",
		tasks,
	};
}

test("resolveActiveTaskId: no salta atrás a tareas stale/unverified mientras haya pending", () => {
	const u = makeUnit([
		{ id: "1", text: "Primera ya pasada y caducada", status: "stale" },
		{ id: "2", text: "Segunda caducada", status: "unverified" },
		{ id: "3", text: "Tercera en curso", status: "pending" },
		{ id: "4", text: "Cuarta pendiente", status: "pending" },
	]);
	assert.equal(resolveActiveTaskId(u, null), "3");
});

test("resolveActiveTaskId: sin pending, prioriza failed antes que stale", () => {
	const u = makeUnit([
		{ id: "1", text: "Caducada", status: "stale" },
		{ id: "2", text: "Fallida", status: "failed" },
		{ id: "3", text: "Verificada", status: "verified" },
	]);
	assert.equal(resolveActiveTaskId(u, null), "2");
});

test("resolveActiveTaskId: cuando no quedan pending ni failed, enfoca la primera stale", () => {
	const u = makeUnit([
		{ id: "1", text: "Verificada", status: "verified" },
		{ id: "2", text: "Caducada", status: "stale" },
		{ id: "3", text: "Otra caducada", status: "stale" },
	]);
	assert.equal(resolveActiveTaskId(u, null), "2");
});

test("renderTasksTree: stale inactiva usa warning ↻ y llamadas completadas no quedan grises (dim)", () => {
	const u = makeUnit([
		{ id: "1", text: "Caducada por edición", status: "stale" },
		{ id: "2", text: "Tarea en curso", status: "pending" },
	]);
	const tracker = createTaskActivityTracker();
	tracker.syncUnit(u);
	tracker.recordStart("c1", "read", { path: "domain/hcd.go" }, u, 1000);
	tracker.recordFinish("c1", false, "line1\nline2\n", 1020);

	const lines = renderTasksTree(u, tracker, 1050, true);
	assert.ok(lines, "debe renderizar árbol");

	const task1 = lines.find((l) => l.text.includes("1. Caducada"));
	assert.ok(task1, "debe mostrar tarea 1");
	assert.equal(task1.tone, "warning");
	assert.ok(task1.text.includes("↻"), "stale debe usar glifo ↻");

	const task2 = lines.find((l) => l.text.includes("2. Tarea en curso"));
	assert.ok(task2, "debe mostrar tarea 2 activa");
	assert.equal(task2.tone, "accent");
	assert.ok(task2.text.includes("en curso"), "tarea activa mientras agente trabaja debe indicar 'en curso'");

	const callLine = lines.find((l) => l.text.includes("Read domain/hcd.go"));
	assert.ok(callLine, "la llamada Read debe atribuirse a la tarea 2, no a la 1");
	assert.notEqual(callLine.tone, "dim", "llamada completada no debe verse gris (dim)");
});

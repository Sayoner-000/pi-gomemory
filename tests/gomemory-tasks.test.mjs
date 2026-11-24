import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

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

// HCD-079 (gomemory 2.59.0, Regla A): con tareas pendientes el motor sigue en
// implement y una stale cuenta como progreso; solo en verify/close es alerta.
test("HCD-079 stale en implement: se pinta como progreso, sin warning ni ↻", () => {
	const u = makeUnit([
		{ id: "1", text: "Verificada y caducada por la tarea 2", status: "stale" },
		{ id: "2", text: "Tarea en curso", status: "pending" },
	]);
	const lines = renderTasksTree(u, createTaskActivityTracker(), 1000, false);
	const task1 = lines.find((l) => l.text.includes("1. Verificada"));
	assert.ok(task1, "debe mostrar la tarea 1");
	assert.notEqual(task1.tone, "warning", "stale en implement no es alerta");
	assert.equal(task1.tone, "dim");
	assert.ok(task1.text.includes("●"), "stale en implement conserva el círculo lleno de progreso");
	assert.ok(!task1.text.includes("↻"), "stale en implement no usa ↻");
	assert.ok(lines[0].text.includes("1/2"), `la barra cuenta la stale como progreso: ${lines[0].text}`);
});

for (const stage of ["verify", "close"]) {
	test(`HCD-079 stale en ${stage}: mantiene ↻ warning`, () => {
		const u = { ...makeUnit([
			{ id: "1", text: "Caducada", status: "stale" },
			{ id: "2", text: "Otra caducada", status: "stale" },
		]), stage };
		const lines = renderTasksTree(u, createTaskActivityTracker(), 1000, false);
		const task2 = lines.find((l) => l.text.includes("2. Otra caducada"));
		assert.ok(task2, "debe mostrar la tarea 2 (inactiva)");
		assert.equal(task2.tone, "warning");
		assert.ok(task2.text.includes("↻"));
	});
}

test("renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)", () => {
	const u = makeUnit([
		{ id: "1", text: "Caducada por edición", status: "stale" },
		{ id: "2", text: "Tarea en curso", status: "pending" },
	]);
	u.stage = "verify";
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

// HCD-079 tarea 5: el panel contra el motor real. Requiere `mem` ≥ 2.59.0 en el PATH;
// sin él se omite. Corre en un repo git y un HOME temporales.
function memVersion() {
	try {
		const m = /(\d+)\.(\d+)\.(\d+)/.exec(execFileSync("mem", ["--version"], { encoding: "utf8" }));
		return m ? m.slice(1).map(Number) : null;
	} catch {
		return null;
	}
}
const realMem = memVersion();
const memOk = Boolean(realMem && (realMem[0] > 2 || (realMem[0] === 2 && realMem[1] >= 59)));

test("HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida", { skip: memOk ? false : "requiere mem >= 2.59.0" }, () => {
	const repo = mkdtempSync(join(tmpdir(), "hcd079-repo-"));
	const home = mkdtempSync(join(tmpdir(), "hcd079-home-"));
	try {
		const env = { ...process.env, HOME: home, GIT_AUTHOR_NAME: "t", GIT_AUTHOR_EMAIL: "t@t", GIT_COMMITTER_NAME: "t", GIT_COMMITTER_EMAIL: "t@t" };
		const run = (cmd, args) => execFileSync(cmd, args, { cwd: repo, env, encoding: "utf8" });
		// Los flags van antes de `--`: lo que sigue es el comando de la comprobación.
		const mem = (...args) => {
			const sep = args.indexOf("--");
			const head = sep < 0 ? args : args.slice(0, sep);
			const tail = sep < 0 ? [] : args.slice(sep);
			return JSON.parse(run("mem", ["hcd", ...head, "--json", "--client", "pi", ...tail]));
		};
		const panel = () => {
			const u = mem("show");
			return { u, lines: renderTasksTree(u, createTaskActivityTracker(), 1000, false) };
		};
		const lineOf = (lines, id) => lines.find((l) => l.text.includes(` ${id}. `));

		run("git", ["init", "-q"]);
		writeFileSync(join(repo, "a.txt"), "a\n");
		run("git", ["add", "."]);
		run("git", ["commit", "-qm", "init"]);

		const started = mem("start", "--kind", "feature", "--lane", "fast", "--title", "Prueba", "--why", "prueba", "--intent", "prueba");
		const doc = join(repo, started.doc_path);
		writeFileSync(
			doc,
			readFileSync(doc, "utf8")
				.replace("Incluido: pendiente", "Incluido: uno y dos")
				.replace("Excluido: pendiente", "Excluido: nada")
				.replace("## ¿Qué pasa si no lo hacemos?\npendiente", "## ¿Qué pasa si no lo hacemos?\nnada")
				.replace("## Alternativa más barata\npendiente", "## Alternativa más barata\nninguna")
				.replace("## Checklist\n", "## Checklist\n- [ ] 1. Crear uno → `test -f uno.txt`\n- [ ] 2. Crear dos → `test -f dos.txt`\n"),
		);

		writeFileSync(join(repo, "uno.txt"), "1\n");
		mem("verify", "--task", "1", "--", "test -f uno.txt");

		// Editar código para la tarea 2 caduca la 1, pero con la 2 pendiente es progreso.
		writeFileSync(join(repo, "codigo.txt"), "trabajo de la tarea 2\n");
		let { u, lines } = panel();
		assert.equal(u.stage, "implement");
		assert.equal(u.tasks[0].status, "stale", "el motor sigue informando stale por tarea");
		let t1 = lineOf(lines, "1");
		assert.notEqual(t1.tone, "warning", `stale en implement no es alerta: ${t1.text}`);
		assert.ok(!t1.text.includes("↻"), t1.text);

		// Sin pendientes el motor pasa a verify y la caducada es el trabajo en curso:
		// el panel la enfoca (resolveActiveTaskId, sin cambios) y deja de contarla.
		writeFileSync(join(repo, "dos.txt"), "2\n");
		mem("verify", "--task", "2", "--", "test -f dos.txt");
		({ u, lines } = panel());
		assert.equal(u.stage, "verify");
		assert.equal(resolveActiveTaskId(u, null), "1");
		assert.equal(lineOf(lines, "1").tone, "accent");
		assert.ok(lines[0].text.includes("1/2"), lines[0].text);

		// `verify --stale` sin comando reejecuta el último comando verde de la tarea 1.
		mem("verify", "--stale");
		({ u, lines } = panel());
		assert.deepEqual(u.tasks.map((t) => t.status), ["verified", "verified"]);
		assert.ok(lines[0].text.includes("2/2"), lines[0].text);
		assert.equal(lineOf(lines, "1").tone, "success");
	} finally {
		rmSync(repo, { recursive: true, force: true });
		rmSync(home, { recursive: true, force: true });
	}
});

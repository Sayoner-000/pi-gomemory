import assert from "node:assert/strict";
import test from "node:test";
import registerPlan, { PLAN_SECTION, PLAN_STATE_KEY, planShortcutKeys } from "../extensions/gomemory-plan.ts";

function setup(exec = async () => ({ code: 0, stdout: "Método de prueba" })) {
	const hooks = new Map();
	let command;
	const messages = [];
	const notices = [];
	const status = new Map();
	const shortcuts = new Map();
	const ctx = {
		cwd: process.cwd(),
		isIdle: () => true,
		ui: {
			notify: (message) => notices.push(message),
			setStatus: (key, value) => status.set(key, value),
		},
	};
	registerPlan({
		registerCommand: (name, options) => {
			assert.equal(name, "plan");
			command = options.handler;
		},
		registerShortcut: (key, options) => shortcuts.set(key, options.handler),
		on: (name, handler) => hooks.set(name, handler),
		exec,
		sendUserMessage: (...args) => messages.push(args),
	});
	return {
		plan: (args = "") => command(args, ctx),
		tool: (toolName, input) => hooks.get("tool_call")({ toolName, input }),
		turn: async (sections) => {
			await hooks.get("before_agent_start")({ systemPromptOptions: { sections } });
		},
		key: (k) => shortcuts.get(k)(ctx),
		ctx, messages, notices, status, shortcuts,
	};
}

test("on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados", async () => {
	let loads = 0;
	const pi = setup(async () => ({ code: 0, stdout: `Método ${++loads}` }));
	const sections = { gomemory: "Otra sección" };
	await pi.plan("status");
	assert.equal(pi.notices.at(-1), "El modo plan no está activo");
	assert.equal(loads, 0);
	await pi.plan("on");
	await pi.plan("on");
	await pi.plan("status");
	assert.match(pi.notices.at(-1), /^Modo plan activo\b/);
	assert.equal(pi.status.get("gomemory-plan"), "PLAN");
	assert.equal(loads, 1);
	await pi.turn(sections);
	assert.match(sections[PLAN_SECTION], /Método 1/);
	for (const tool of ["edit", "write"]) assert.equal((await pi.tool(tool)).block, true);
	assert.equal((await pi.tool("bash", { command: "touch x" })).block, true);
	assert.equal(await pi.tool("bash", { command: "git status" }), undefined);

	await pi.plan("off");
	await pi.plan("off");
	await pi.turn(sections);
	await pi.plan("status");
	assert.equal(pi.notices.at(-1), "El modo plan no está activo");
	assert.equal(pi.status.get("gomemory-plan"), undefined);
	assert.deepEqual(sections, { gomemory: "Otra sección" });
	assert.equal(await pi.tool("write"), undefined);
	assert.equal(await pi.tool("edit"), undefined);
	assert.equal(await pi.tool("bash", { command: "touch x" }), undefined);
	assert.deepEqual(pi.messages, []);
});

test("run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto", async () => {
	const pi = setup();
	await pi.plan("run");
	assert.equal(pi.notices.at(-1), "El modo plan no está activo");
	await pi.plan();
	assert.equal((await pi.tool("write")).block, true);
	await pi.plan();
	assert.equal(await pi.tool("write"), undefined);
	await pi.plan("on y off son opciones");
	pi.ctx.isIdle = () => false;
	await pi.plan("Analiza el repositorio");
	assert.deepEqual(pi.messages, [
		["on y off son opciones"],
		["Analiza el repositorio", { deliverAs: "followUp" }],
	]);
	assert.equal((await pi.tool("write")).block, true);
	await pi.plan("run");
	assert.equal(pi.notices.at(-1), "Modo plan desactivado: ya se permiten modificaciones");
	assert.equal(await pi.tool("write"), undefined);
	assert.equal(pi.messages.length, 2);
});

test("off no deja que una carga tardía vuelva a activar el modo", async () => {
	const { promise, resolve } = Promise.withResolvers();
	const pi = setup(() => promise);
	const on = pi.plan("on");
	assert.equal((await pi.tool("write"))?.block, true);
	const sections = {};
	const turn = pi.turn(sections);
	await pi.plan("off");
	resolve({ code: 0, stdout: "Método tardío" });
	await on;
	await turn;
	await pi.turn(sections);
	assert.deepEqual(sections, {});
	assert.equal(pi.status.get("gomemory-plan"), undefined);
	assert.equal(await pi.tool("write"), undefined);
	assert.deepEqual(pi.messages, []);
});

test("modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes", async () => {
	const pi = setup();
	await pi.plan("on");

	// Permitidas: consultas de grafos e index_repository sin persistence en el repo (3B).
	for (const [tool, input] of [
		["codegraph_explore", { query: "InstallAll" }],
		["mcp__codegraph__codegraph_explore", { query: "InstallAll" }],
		["search_graph", { project: "p", name_pattern: "Install.*" }],
		["mcp__codebase_memory__trace_path", { project: "p", function_name: "InstallAll", direction: "both" }],
		["check_index_coverage", { project: "p", paths: ["main.go"] }],
		["index_repository", { repo_path: ".", mode: "moderate" }],
		["mcp__codebase_memory__index_repository", { repo_path: ".", mode: "moderate", persistence: false }],
		["manage_adr", { project: "p", mode: "outline" }],
		["mcp__codebase_memory__manage_adr", { project: "p", mode: "get" }],
		["bash", { command: "mem hcd show --json" }],
		["bash", { command: "mem hcd history --file main.go" }],
		["bash", { command: "mem hcd analyze --json" }],
		["bash", { command: "codegraph explore 'InstallAll'" }],
	]) {
		assert.equal(await pi.tool(tool, input), undefined, `no debía bloquear ${tool} ${JSON.stringify(input)}`);
	}

	// Bloqueadas: mutaciones de código, HCD o escritura en el repo.
	for (const [tool, input] of [
		["delete_project", { project: "p" }],
		["mcp__codebase_memory__delete_project", { project: "p" }],
		["index_repository", { repo_path: ".", persistence: true }],
		["mcp__codebase_memory__index_repository", { repo_path: ".", persistence: true }],
		["manage_adr", { project: "p", mode: "update", content: "x" }],
		["mcp__codebase_memory__manage_adr", { project: "p", mode: "set_sections", section_updates: {} }],
		["bash", { command: "mem hcd start --kind fix --title x" }],
		["bash", { command: "mem hcd verify --task 1 -- go test" }],
		["bash", { command: "mem hcd close --quote ok" }],
		["bash", { command: "mem hcd cancel --quote no" }],
		["bash", { command: "mem hcd clarify --ask x" }],
		["bash", { command: "mem hcd approve --quote ok" }],
		["bash", { command: "mem hcd promote" }],
		["bash", { command: "mem hcd rename nuevo" }],
		["bash", { command: "codegraph index ." }],
	]) {
		assert.equal((await pi.tool(tool, input))?.block, true, `debía bloquear ${tool} ${JSON.stringify(input)}`);
	}
});

// HCD-067: Alt+P (y Ctrl+Alt+P de respaldo) alternan el modo y el estado se publica al header.
test("planShortcutKeys: Alt+P y Ctrl+Alt+P; en Windows y WSL solo Ctrl+Alt+P", () => {
	assert.deepEqual(planShortcutKeys("darwin", "25.0.0", {}), ["alt+p", "ctrl+alt+p"]);
	assert.deepEqual(planShortcutKeys("linux", "6.8.0", {}), ["alt+p", "ctrl+alt+p"]);
	assert.deepEqual(planShortcutKeys("win32", "10.0", {}), ["ctrl+alt+p"]);
	assert.deepEqual(planShortcutKeys("linux", "5.15.0-microsoft-standard-WSL2", {}), ["ctrl+alt+p"]);
	assert.deepEqual(planShortcutKeys("linux", "6.8.0", { WSL_DISTRO_NAME: "Ubuntu" }), ["ctrl+alt+p"]);
});

test("Alt+P alterna el modo plan, bloquea y publica el estado para el header", async () => {
	const pi = setup();
	const shared = globalThis[PLAN_STATE_KEY];
	assert.ok(shared, "el estado debe publicarse en Symbol.for('pi-gomemory.plan')");
	let changes = 0;
	const off = shared.subscribe(() => changes++);
	assert.ok(pi.shortcuts.has("ctrl+alt+p"));
	const key = pi.shortcuts.has("alt+p") ? "alt+p" : "ctrl+alt+p";

	await pi.key(key);
	assert.equal(shared.on, true);
	assert.equal(pi.status.get("gomemory-plan"), "PLAN");
	assert.equal((await pi.tool("edit")).block, true);
	assert.equal(pi.messages.length, 0, "el atajo no debe enviar mensajes");

	await pi.key(key);
	assert.equal(shared.on, false);
	assert.equal(pi.status.get("gomemory-plan"), undefined);
	assert.equal(await pi.tool("edit"), undefined);

	await pi.plan("on");
	assert.equal(shared.on, true, "/plan on también publica el estado");
	await pi.plan("off");
	assert.equal(shared.on, false);
	assert.equal(changes, 4);
	off();
});

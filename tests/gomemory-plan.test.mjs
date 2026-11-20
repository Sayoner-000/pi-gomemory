import assert from "node:assert/strict";
import test from "node:test";
import registerPlan, { PLAN_SECTION } from "../extensions/gomemory-plan.ts";

function setup(exec = async () => ({ code: 0, stdout: "Método de prueba" })) {
	const hooks = new Map();
	let command;
	const messages = [];
	const notices = [];
	const status = new Map();
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
		ctx, messages, notices, status,
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

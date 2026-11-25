import assert from "node:assert/strict";
import test from "node:test";
import registerContext, { SECTION } from "../extensions/gomemory-context.ts";

const PLAN_STATE_KEY = Symbol.for("pi-gomemory.plan");

function setup(exec) {
	const hooks = new Map();
	const notices = [];
	registerContext({
		on: (name, handler) => hooks.set(name, handler),
		exec,
	});
	const ctx = {
		cwd: process.cwd(),
		ui: {
			notify: (msg, level) => notices.push({ msg, level }),
		},
	};
	return { hooks, notices, ctx };
}

test("H-02: agent_end no ejecuta mem code-refresh mientras /plan está activo", async () => {
	const calls = [];
	const { hooks, ctx } = setup(async (bin, args) => {
		calls.push([bin, ...args]);
		return { code: 0, stdout: "", stderr: "" };
	});

	const prev = globalThis[PLAN_STATE_KEY];
	try {
		globalThis[PLAN_STATE_KEY] = { on: true };
		await hooks.get("agent_end")({}, ctx);
		assert.equal(calls.length, 0, `no debía invocar mem code-refresh en modo plan: ${JSON.stringify(calls)}`);

		globalThis[PLAN_STATE_KEY] = { on: false };
		await hooks.get("agent_end")({}, ctx);
		assert.deepEqual(calls, [["mem", "code-refresh"]]);
	} finally {
		if (prev === undefined) delete globalThis[PLAN_STATE_KEY];
		else globalThis[PLAN_STATE_KEY] = prev;
	}
});

test("H-07: before_agent_start reintenta mem context tras un fallo transitorio", async () => {
	let attempts = 0;
	const { hooks, ctx } = setup(async (_bin, args) => {
		if (args[0] === "context") {
			attempts++;
			if (attempts === 1) {
				return { code: 1, stdout: "", stderr: "timeout temporal" };
			}
			return { code: 0, stdout: "# Contexto recuperado", stderr: "" };
		}
		return { code: 0, stdout: "", stderr: "" };
	});

	await hooks.get("session_start")({}, ctx);

	const turn1 = { systemPromptOptions: { sections: {}, selectedTools: [] } };
	await hooks.get("before_agent_start")(turn1, ctx);
	assert.equal(attempts, 1);
	assert.ok(!turn1.systemPromptOptions.sections[SECTION].includes("# Contexto recuperado"));

	const turn2 = { systemPromptOptions: { sections: {}, selectedTools: [] } };
	await hooks.get("before_agent_start")(turn2, ctx);
	assert.equal(attempts, 2, "debe reintentar mem context en el siguiente turno tras fallo transitorio");
	assert.ok(turn2.systemPromptOptions.sections[SECTION].includes("# Contexto recuperado"));
});

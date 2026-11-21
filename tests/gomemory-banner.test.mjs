import assert from "node:assert/strict";
import test from "node:test";
import registerBanner, { planHeaderLine } from "../extensions/gomemory-banner.ts";
import { planState } from "../extensions/gomemory-plan.ts";

const AMBER = "38;2;244;198;107";
const strip = (s) => s.replace(/\x1b\[[0-9;]*m/g, "");

// HCD-067: el header avisa del modo plan: ámbar ON, atenuado OFF.
test("planHeaderLine: ON en ámbar, OFF atenuado, sin escapes con NO_COLOR, centrada", () => {
	const on = planHeaderLine(80, false, true);
	assert.match(on, /MODO PLAN ON/);
	assert.ok(on.includes(AMBER), "ON debe usar el ámbar del theme (#f4c66b)");
	assert.match(on, /Alt\+P/);

	const off = planHeaderLine(80, false, false);
	assert.match(off, /plan off/);
	assert.ok(!off.includes(AMBER), "OFF no debe ser ámbar");
	assert.ok(off.includes("\x1b[2;"), "OFF debe ir atenuado");

	for (const on of [true, false]) {
		const plain = planHeaderLine(80, true, on);
		assert.ok(!plain.includes("\x1b"), "NO_COLOR no lleva escapes");
		const text = plain.trim();
		assert.equal(plain.length - text.length, Math.floor((80 - text.length) / 2));
	}
});

test("el header muestra el modo actual y se repinta al cambiarlo", async () => {
	const prevNoColor = process.env.NO_COLOR;
	process.env.NO_COLOR = "1";
	process.env.PI_GOMEMORY_RUNNING_VERSION ??= "v0.0.0";
	process.env.GOMEMORY_INSTALLED_VERSION ??= "gomemory 0.0.0";
	try {
		let onStart;
		registerBanner({ on: (name, h) => name === "session_start" && (onStart = h), exec: async () => ({ code: 1, stdout: "" }) });
		let factory;
		await onStart({}, { mode: "tui", ui: { setHeader: (f) => (factory = f) } });
		let renders = 0;
		const header = factory({ requestRender: () => renders++ }, {});
		const shared = planState();
		shared.set(false);
		assert.match(strip(header.render(80).join("\n")), /plan off/);

		shared.set(true);
		assert.ok(renders >= 1, "cambiar el modo debe pedir repintado");
		assert.match(strip(header.render(80).join("\n")), /MODO PLAN ON/);

		shared.set(false);
		assert.match(strip(header.render(80).join("\n")), /plan off/);
		header.dispose?.();
	} finally {
		if (prevNoColor === undefined) delete process.env.NO_COLOR;
		else process.env.NO_COLOR = prevNoColor;
	}
});

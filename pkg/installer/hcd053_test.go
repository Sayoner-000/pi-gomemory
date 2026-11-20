package installer

import "testing"

func TestHCD053LiveHCDRows(t *testing.T) {
	t.Parallel()

	runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import { chmodSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import registerTDD, { SPINNER_FRAMES } from "./gomemory-tdd.ts";

const handlers = new Map();
let command, tool, panel, renders = 0;
const theme = { fg: (_, text) => text, style: (text) => text };
const pi = {
  on(name, fn) { handlers.set(name, fn); },
  registerCommand(_name, cmd) { command = cmd.handler; },
  registerTool(spec) { tool = spec; },
};
const ctx = { hasUI: true, cwd: process.cwd(), ui: {
  notify() {}, custom(factory) {
    panel = factory({ requestRender() { renders++; } }, theme, {}, () => {});
    return new Promise(() => {});
  },
} };
// /tdd min opens the panel without running a suite; the second call expands it.
registerTDD(pi);
await command("min", ctx);
await command("min", ctx);
await command("watch", ctx);
const screen = () => panel.render(46).join("\n");
const start = (id, cmd) => handlers.get("tool_execution_start")?.({ toolName: "bash", toolCallId: id, args: { command: cmd } }, ctx);
const end = (id, passed, isError = !passed, durationMs = 10) => handlers.get("tool_execution_end")?.({
  toolName: "bash", toolCallId: id, isError, durationMs,
  result: { content: [{ type: "text", text: JSON.stringify({ ok: passed, unit_id: "HCD-053", data: { evidence: { passed } } }) }] },
}, ctx);
const cmd = (task) => "mem hcd verify --task " + task + " --json -- 'go test ./...'";
const row = (id) => screen().split("\n").find((line) => new RegExp("Tarea " + id + "\\b").test(line));
const before = renders;
await start("irrelevant", "printf '%s' 'mem hcd verify --task 1'");
assert.equal(row(1), undefined);
await start("one", cmd(1));
assert.ok(renders > before, "debe repintar sin esperar el resultado de Bash");
assert.ok(SPINNER_FRAMES.some((glyph) => row(1)?.includes(glyph)), "debe mostrar la tarea en curso");
const resultText = '{"ok":true,"unit_id":"HCD-053","data":{"evidence":{"passed":true}}}';
await handlers.get("tool_result")({ toolName: "bash", toolCallId: "one", input: { command: cmd(1) }, isError: false, content: [{ type: "text", text: resultText }] }, ctx);
await end("one", true);
assert.match(screen(), /Data & Graph \(0\/0\)/, "el resultado HCD no se escribe en Data & Graph");
assert.match(row(1) ?? "", /✓.*Tarea 1/);
await start("two", cmd("2,3"));
assert.match(row(1) ?? "", /✓/);
assert.ok(SPINNER_FRAMES.some((glyph) => row(2)?.includes(glyph)));
assert.ok(SPINNER_FRAMES.some((glyph) => row(3)?.includes(glyph)));
await end("two", true, true); // el error del tool prevalece sobre la evidencia positiva
assert.match(row(2) ?? "", /✗.*Tarea 2/);
assert.match(row(3) ?? "", /✗.*Tarea 3/);
await start("old-attempt", cmd(2));
await start("new-attempt", cmd(2));
await end("new-attempt", true);
await end("old-attempt", false);
assert.match(row(2) ?? "", /✓.*Tarea 2/);
assert.equal(screen().match(/Tarea 2\b/g)?.length, 1, "cada tarea ocupa una sola fila");

// Watch corre el flujo real del panel contra un ejecutable local determinista.
const mem = join(ctx.cwd, "mem");
writeFileSync(mem, '#!/bin/sh\nprintf x >> watch-runs\nprintf \'%s\\n\' \'{"ok":true,"unit_id":"HCD-053"}\'\n');
chmodSync(mem, 0o700);
process.env.PATH = ctx.cwd + ":" + process.env.PATH;
await handlers.get("agent_settled")({}, ctx);
// Desde HCD-056 una ejecución pedida aborta watch en cola: se espera a que watch termine.
for (let i = 0; i < 200 && !/Data & Graph \(1\/1\)/.test(screen()); i++) await new Promise((r) => setTimeout(r, 25));
assert.equal(readFileSync(join(ctx.cwd, "watch-runs"), "utf8"), "x", "Watch debe correr el flujo real");
await tool.execute("after-watch", { suite: "data" }, undefined, undefined, ctx);
assert.equal(readFileSync(join(ctx.cwd, "watch-runs"), "utf8"), "xx", "Watch y la ejecución encolada deben correr");
assert.match(screen(), /Data & Graph \(1\/1\)/);
assert.match(row(1) ?? "", /✓/);
assert.match(row(2) ?? "", /✓/);
assert.match(row(3) ?? "", /✗/);
assert.match(screen(), /100% \(1\/1\)/, "las filas HCD no alteran los totales de suites");

await start("blocked", cmd(4));
await handlers.get("tool_execution_end")({ toolName: "bash", toolCallId: "blocked", isError: true, result: {} }, ctx);
assert.match(row(4) ?? "", /✗/, "un tool abortado o bloqueado no deja spinner huérfano");
`)
}

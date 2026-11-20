package installer

import (
	"path/filepath"
	"strings"
	"testing"
)

// HCD-058 tarea 1 (C-001): si llegó alguna línea por tool_execution_update, el
// texto final del resultado se descartaba y lo posterior al último update se
// perdía (tests en curso para siempre, fallo real cambiado por bash-error).
func TestHCD058FinalTextReconcile(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { createTDDStore, createBashLiveIngest } from "./gomemory-tdd.ts";
const fails = [];
const upd = (lines) => ({ content: [{ type: "text", text: lines.join("\n") }] });
const suite = (s) => s.getState().suites.find((x) => x.id === "backend");
const test = (s, n) => suite(s)?.tests.find((x) => x.name === n);
const live = [
	"=== RUN   TestA",
	"--- PASS: TestA (0.00s)",
	"=== RUN   TestB",
];
const full = [...live, "    b_test.go:7: boom", "--- FAIL: TestB (0.01s)", "FAIL", "FAIL\tdemo\t0.02s", ""];

// 1) El final completa lo que el último update no trajo.
{
	const store = createTDDStore();
	const ing = createBashLiveIngest(store, process.cwd());
	ing.start("a", "go test -v ./...");
	ing.update("a", upd(live));
	ing.end("a", true, full.join("\n"));
	if (test(store, "TestA")?.status !== "passed") fails.push("1: TestA debió pasar");
	if (test(store, "TestB")?.status !== "failed") fails.push("1: TestB debió fallar con su salida real: " + test(store, "TestB")?.status);
	if (suite(store)?.tests.some((x) => x.name.includes("error de ejecución"))) fails.push("1: no debe sustituirse por bash-error");
	if (suite(store)?.tests.filter((x) => x.name === "TestA").length !== 1) fails.push("1: TestA no debe duplicarse");
	if (suite(store)?.partial) fails.push("1: sin huecos la suite no es parcial");
}

// 2) Final truncado (solo la cola) que sigue cubriendo lo visto: se reconcilia por línea absoluta.
{
	const store = createTDDStore();
	const ing = createBashLiveIngest(store, process.cwd());
	ing.start("b", "go test -v ./...");
	ing.update("b", upd(live));
	const tail = full.slice(2); // pierde las 2 primeras, ya procesadas
	ing.end("b", true, tail.join("\n"), { truncated: true, totalLines: full.length });
	if (test(store, "TestB")?.status !== "failed") fails.push("2: con cola truncada TestB debió fallar: " + test(store, "TestB")?.status);
	if (suite(store)?.partial) fails.push("2: la cola cubre lo no visto: no hay hueco");
}

// 3) Un final que no cubre lo ya visto (más corto) no borra lo ingerido en vivo.
{
	const store = createTDDStore();
	const ing = createBashLiveIngest(store, process.cwd());
	ing.start("c", "go test -v ./...");
	ing.update("c", upd(live));
	ing.end("c", false, "ok");
	if (test(store, "TestA")?.status !== "passed") fails.push("3: lo ingerido en vivo debe conservarse");
}
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD058FinalTextReconcile: %s", out)
	}
}

// HCD-058 tarea 2 (C-002): el trabajo de agent_end no debe escribir la
// sugerencia ni pedirla al modelo si mientras tanto empezó otro turno o se
// cerró la sesión; y agent_end no espera a la petición al modelo.
func TestHCD058SuggestEpoch(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-suggest.ts"}, hcd057Harness+`
await h.session_start({}, ctx);
const ed = factory(tui, {}, {});

// a) Motor HCD: otro turno empieza mientras mem hcd show está en vuelo.
show = { ok: true, state: "active", unit_id: "HCD-058", next_action: { skill: "hcd-implement" } };
const a = h.agent_end({}, ctx);
await h.before_agent_start({}, ctx);
await a;
if (ghostOf(ed)) fails.push("a) la sugerencia del turno anterior no debe aparecer en el nuevo: " + ghostOf(ed));

// b) Modelo: otro turno empieza antes de lanzar la petición.
show = { ok: false };
const before = completeCalls.length;
const b = h.agent_end({}, ctx);
await h.before_agent_start({}, ctx);
await b;
await new Promise((r) => setTimeout(r, 20));
if (completeCalls.length !== before) fails.push("b) no debe pedirse al modelo en el turno siguiente");
if (ghostOf(ed)) fails.push("b) sin sugerencia en el turno nuevo");

// c) Sesión cerrada mientras se consultaba el motor.
show = { ok: true, state: "active", unit_id: "HCD-058", next_action: { skill: "hcd-close" } };
const c = h.agent_end({}, ctx);
await h.session_shutdown({}, ctx);
await c;
if (ghostOf(ed)) fails.push("c) tras cerrar la sesión no se escribe sugerencia");

// d) agent_end no espera a la petición al modelo.
show = { ok: false };
completeImpl = () => new Promise(() => {}); // nunca responde
const t0 = Date.now();
const done = await Promise.race([h.agent_end({}, ctx).then(() => "done"), new Promise((r) => setTimeout(() => r("timeout"), 300))]);
if (done !== "done") fails.push("d) agent_end no debe esperar a modelNext (" + (Date.now() - t0) + " ms)");
await h.before_agent_start({}, ctx);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD058SuggestEpoch: %s", out)
	}
}

// HCD-058 tarea 3 (C-003): un mem hcd verify detrás de otros comandos
// encadenados con && también muestra en /tdd los tests de su comprobación.
func TestHCD058VerifyCompound(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { createTDDStore, ingestHCDVerifyTests } from "./gomemory-tdd.ts";
const fails = [];
const excerpt = ["=== RUN   TestA", "--- PASS: TestA (0.00s)", "=== RUN   TestB", "--- FAIL: TestB (0.00s)", "FAIL"].join("\n");
const result = JSON.stringify({ ok: true, unit_id: "HCD-058", data: { evidence: { passed: false, excerpt } } });
for (const cmd of [
	"mem hcd verify --task 1 --json -- 'go test -v ./...'",
	"cd sub && mem hcd verify --task 1 --json -- 'go test -v ./...'",
	"export X=1 && cd sub && mem hcd verify --task 1 -- 'go test -v ./...'",
]) {
	const store = createTDDStore();
	const ok = ingestHCDVerifyTests(store, cmd, result, false, process.cwd());
	const tests = store.getState().suites.find((s) => s.id === "backend")?.tests ?? [];
	if (!ok) fails.push("no se ingirió: " + cmd);
	else if (tests.find((x) => x.name === "TestB")?.status !== "failed") fails.push("TestB sin ingerir en: " + cmd);
}
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD058VerifyCompound: %s", out)
	}
}

// HCD-058 tarea 4 (C-005, Q1): un Bash de tests completo solo deja a watch sin
// trabajo si su suite es la única suite de tests activa. Con Go y un frontend
// con tests, un go test ./... no cubre el frontend y watch debe correr.
func TestHCD058WatchDirtyScope(t *testing.T) {
	t.Parallel()

	proj := t.TempDir()
	bin := t.TempDir()
	count := filepath.Join(bin, "count.log")
	writeFile(t, filepath.Join(proj, "go.mod"), "module w\n")
	writeFile(t, filepath.Join(proj, "package.json"), `{"scripts":{"test":"echo ok"}}`)
	writeExec(t, filepath.Join(bin, "go"), `[ "$1" = "test" ] && echo go >> "`+count+`"
exit 0`)
	writeExec(t, filepath.Join(bin, "npm"), `echo npm >> "`+count+`"
exit 0`)
	writeExec(t, filepath.Join(bin, "npx"), `exit 0`)
	writeExec(t, filepath.Join(bin, "mem"), `exit 3`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import { readFileSync } from "node:fs";
import factory from "./gomemory-tdd.ts";
const commands = {}, handlers = {};
factory({ registerCommand: (n, d) => (commands[n] = d), registerTool: () => {}, on: (e, h) => { (handlers[e] ||= []).push(h); return () => {}; }, exec: async () => {} });
const ctx = { cwd: `+jsStr(proj)+`, hasUI: false, ui: { notify: () => {} } };
const emit = async (e, ev) => { for (const h of handlers[e] || []) await h(ev, ctx); };
const runs = () => { try { return readFileSync(`+jsStr(count)+`, "utf8").split("\n").filter(Boolean); } catch { return []; } };
await commands.tdd.handler("watch", ctx);
await emit("tool_execution_end", { toolName: "edit", isError: false });
await emit("tool_execution_start", { toolName: "bash", toolCallId: "g1", args: { command: "go test ./..." } });
await emit("tool_execution_end", { toolName: "bash", toolCallId: "g1", isError: false, durationMs: 5, result: { content: [] } });
await emit("agent_settled", { aborted: false });
await new Promise((r) => setTimeout(r, 1200));
const r = runs();
console.log(r.includes("npm") ? "PASS" : "FAIL watch debió comprobar el frontend tras un go test: " + JSON.stringify(r));
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD058WatchDirtyScope: %s", out)
	}
}

// HCD-058 tarea 5 (C-004): los temporizadores que session_start programa para
// reenvolver el editor se cancelan al cerrar la sesión.
func TestHCD058SuggestTimers(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-suggest.ts"}, hcd057Harness+`
await h.session_start({}, ctx);
await h.session_shutdown({}, ctx);
factory = null; // cualquier reinstalación posterior vendría de un temporizador vivo
await new Promise((r) => setTimeout(r, 1200));
console.log(factory ? "FAIL un temporizador de session_start reinstaló el editor tras session_shutdown" : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD058SuggestTimers: %s", out)
	}
}

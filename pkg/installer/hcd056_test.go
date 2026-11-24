package installer

import (
	"path/filepath"
	"strings"
	"testing"
)

// HCD-056 tarea 3: run_test_suite acepta `filter` y lo traduce al ejecutor de
// cada lenguaje; los checks que no admiten filtro se omiten con su motivo y la
// suite queda marcada como parcial.
func TestHCD056Filter(t *testing.T) {
	t.Parallel()

	proj := t.TempDir()
	bin := t.TempDir()
	argsLog := filepath.Join(bin, "args.log")
	writeFile(t, filepath.Join(proj, "go.mod"), "module f\n")
	writeExec(t, filepath.Join(bin, "go"), `echo "$*" >> "`+argsLog+`"
if [ "$1" = "test" ]; then
  echo '{"Action":"run","Package":"f","Test":"TestA"}'
  echo '{"Action":"pass","Package":"f","Test":"TestA","Elapsed":0.01}'
fi
exit 0`)
	writeExec(t, filepath.Join(bin, "mem"), `exit 3`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import { readFileSync } from "node:fs";
import factory, { filterCheck } from "./gomemory-tdd.ts";
const fails = [];
const spec = (id, cmd, args, parser) => ({ id, name: id, cmd, args, parser });
const cases = [
	[spec("go-test", "go", ["test", "-json", "./..."], "go-test"), ["test", "-json", "-run", "TestA", "./..."]],
	[spec("pytest", "pytest", ["-q"], "pytest"), ["-q", "-k", "TestA"]],
	[spec("pytest", "uv", ["run", "pytest", "-q"], "pytest"), ["run", "pytest", "-q", "-k", "TestA"]],
	[spec("vitest", "npx", ["vitest", "run"], "vitest"), ["vitest", "run", "-t", "TestA"]],
	[spec("pw", "npx", ["playwright", "test", "--reporter=line"], "playwright-line"), ["playwright", "test", "--reporter=line", "-g", "TestA"]],
	[spec("phpunit", "vendor/bin/phpunit", [], "phpunit"), ["--filter", "TestA"]],
	[spec("phpunit", "composer", ["test"], "phpunit"), ["test", "--", "--filter", "TestA"]],
	[spec("mvn-test", "mvn", ["test"], "java-test"), ["-Dtest=TestA", "-Dsurefire.failIfNoSpecifiedTests=false", "test"]],
	[spec("gradle-test", "gradle", ["test"], "java-test"), ["test", "--tests", "TestA"]],
];
for (const [check, want] of cases) {
	const got = filterCheck(check, "TestA");
	if ("skip" in got) fails.push(check.cmd + ": no debió omitirse (" + got.skip + ")");
	else if (JSON.stringify(got.args) !== JSON.stringify(want)) fails.push(check.cmd + ": args " + JSON.stringify(got.args));
}
for (const check of [spec("go-vet", "go", ["vet", "./..."], "exit-code"), spec("npm-test", "npm", ["test"], "exit-code"), spec("hcd-show", "mem", ["hcd", "show", "--json"], "hcd")]) {
	const got = filterCheck(check, "TestA");
	if (!("skip" in got)) fails.push(check.id + ": sin filtro traducible debe omitirse");
}

let tool;
factory({ registerCommand: () => {}, registerTool: (d) => (tool = d), on: () => () => {}, exec: async () => { throw new Error("no"); } });
const ctx = { cwd: `+jsStr(proj)+`, hasUI: false, ui: { notify: () => {} } };
const res = await tool.execute("c1", { suite: "all", filter: "TestA" }, undefined, undefined, ctx);
const log = readFileSync(`+jsStr(argsLog)+`, "utf8").trim().split("\n");
if (JSON.stringify(log) !== JSON.stringify(["test -json -run TestA ./..."])) fails.push("go debió ejecutarse solo con -run: " + JSON.stringify(log));
const backend = res.details.suites.find((s) => s.id === "backend");
if (!backend?.partial) fails.push("la suite filtrada debe quedar parcial");
if (backend?.status !== "passed") fails.push("backend: " + backend?.status);
const ci = res.details.suites.find((s) => s.id === "ci");
if (ci?.status !== "skipped") fails.push("ci (go vet) debe omitirse con filtro: " + ci?.status);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD056Filter: %s", out)
	}
}

// HCD-056 tarea 4: la descripción guía al agente al ciclo acotado: filter para
// rojo→verde y la suite completa solo para la regresión final.
func TestHCD056ToolDescription(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import factory from "./gomemory-tdd.ts";
let tool;
factory({ registerCommand: () => {}, registerTool: (d) => (tool = d), on: () => () => {}, exec: async () => {} });
const d = tool.description;
const fails = [];
if (!/filter/.test(d)) fails.push("no menciona filter");
if (!/rojo.{0,3}verde/i.test(d)) fails.push("no menciona el ciclo rojo→verde");
if (!/regresi[oó]n final/i.test(d)) fails.push("no reserva la suite completa para la regresión final");
if (/go\.mod, package\.json, tsconfig\.json y playwright\.config\)/.test(d)) fails.push("la lista de detección omite Python, PHP y Java");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD056ToolDescription: %s", out)
	}
}

// HCD-056 tarea 6: /tdd ingiere en vivo los tests de un Bash a partir de
// tool_execution_update. Pi entrega la salida acumulada con la cola truncada:
// sin truncado empieza en la línea 0; con truncado la primera línea absoluta es
// totalLines - líneas visibles. La última línea puede estar abierta y se retiene.
func TestHCD056BashLiveIngest(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import factory, { createTDDStore, createBashLiveIngest } from "./gomemory-tdd.ts";
const fails = [];
const ev = (a, test, extra = "") => '{"Action":"' + a + '","Package":"p","Test":"' + test + '"' + extra + '}';
const upd = (lines, truncation) => ({ content: [{ type: "text", text: lines.join("\n") }], details: truncation ? { truncation } : undefined });
const suite = (store) => store.getState().suites.find((s) => s.id === "backend");
const test = (store, name) => suite(store)?.tests.find((t) => t.name === name);

// 1) Sin truncado: los tests aparecen mientras corre; la última línea se retiene.
{
	const store = createTDDStore();
	const live = createBashLiveIngest(store, process.cwd());
	if (!live.start("b1", "go test -json ./...")) fails.push("1: no reconoció go test");
	live.update("b1", upd([ev("run", "TestA"), ev("pass", "TestA", ',"Elapsed":0.1'), ev("run", "TestB")]));
	if (test(store, "TestA")?.status !== "passed") fails.push("1: TestA debió pasar en vivo");
	if (test(store, "TestB")) fails.push("1: la última línea (posiblemente abierta) no debe procesarse aún");
	if (suite(store)?.status !== "running") fails.push("1: la suite debe estar en curso: " + suite(store)?.status);
	live.update("b1", upd([ev("run", "TestA"), ev("pass", "TestA", ',"Elapsed":0.1'), ev("run", "TestB"), ev("fail", "TestB", ',"Elapsed":0.2')]));
	if (test(store, "TestB")?.status !== "running") fails.push("1: TestB debió empezar: " + test(store, "TestB")?.status);
	if (!live.end("b1", true)) fails.push("1: end debió consumir la ejecución en vivo");
	if (test(store, "TestB")?.status !== "failed") fails.push("1: TestB debió fallar al terminar");
	if (suite(store)?.status !== "failed") fails.push("1: suite " + suite(store)?.status);
	if (suite(store)?.partial) fails.push("1: sin huecos la suite no es parcial");
	if (suite(store)?.tests.filter((t) => t.name === "TestA").length !== 1) fails.push("1: TestA no debe duplicarse");
}

// 2) Con truncado y hueco: lo que se perdió marca la suite como parcial.
{
	const store = createTDDStore();
	const live = createBashLiveIngest(store, process.cwd());
	live.start("b2", "go test -json ./...");
	live.update("b2", upd([ev("run", "TestA")]));
	live.update("b2", upd([ev("run", "TestY"), ev("pass", "TestY", ',"Elapsed":0.1'), "ok p 1s"], { truncated: true, totalLines: 50, outputLines: 3 }));
	if (test(store, "TestY")?.status !== "passed") fails.push("2: TestY debió ingerirse desde la cola truncada");
	live.end("b2", false);
	if (!suite(store)?.partial) fails.push("2: con líneas perdidas la suite debe quedar parcial");
}

// 3) Lo que no es un test o es mem hcd verify no se ingiere por esta vía.
{
	const store = createTDDStore();
	const live = createBashLiveIngest(store, process.cwd());
	if (live.start("b3", "ls -la")) fails.push("3: ls no es un test");
	if (live.start("b4", "mem hcd verify --task 1 --json -- 'go test ./...'")) fails.push("3: mem hcd verify va por su propia vía");
	if (live.end("b3", false)) fails.push("3: end sin start no consume nada");
}

// 3b) Sin entregas con contenido, end ingiere el texto final del resultado.
{
	const store = createTDDStore();
	const live = createBashLiveIngest(store, process.cwd());
	live.start("b5", "go test -json ./...");
	live.update("b5", { content: [] });
	live.end("b5", true, [ev("run", "TestZ"), ev("fail", "TestZ", ',"Elapsed":0.1')].join("\n"));
	if (test(store, "TestZ")?.status !== "failed") fails.push("3b: sin entregas debió ingerirse el texto final");
}

// 4) La extensión escucha tool_execution_update.
{
	const handlers = {};
	factory({ registerCommand: () => {}, registerTool: () => {}, on: (e, h) => { (handlers[e] ||= []).push(h); return () => {}; }, exec: async () => {} });
	if (!handlers.tool_execution_update?.length) fails.push("4: falta el listener de tool_execution_update");
}
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD056BashLiveIngest: %s", out)
	}
}

// HCD-056 tarea 7: el resultado de `mem hcd verify` trae en evidence.excerpt la
// cola de la salida de su comprobación; /tdd muestra esos tests en la suite del
// ejecutor (parcial) sin cambiar el veredicto de verify ni la fila por tarea.
func TestHCD056HCDVerifyIngest(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { createTDDStore, ingestHCDVerifyTests } from "./gomemory-tdd.ts";
const fails = [];
const verify = (passed, excerpt) => JSON.stringify({ ok: true, unit_id: "HCD-9", data: { evidence: { passed, exit_code: passed ? 0 : 1, excerpt } } });
const suite = (store, id) => store.getState().suites.find((s) => s.id === id);
const test = (store, id, name) => suite(store, id)?.tests.find((t) => t.name === name);

// 1) go test -v: un test pasa, otro falla con su mensaje; verify falló → suite fallida y parcial.
{
	const store = createTDDStore();
	const excerpt = ["=== RUN   TestA", "--- PASS: TestA (0.01s)", "=== RUN   TestB", "--- FAIL: TestB (0.02s)", "    x_test.go:5: boom", "FAIL", "FAIL\tp\t0.1s"].join("\n");
	if (!ingestHCDVerifyTests(store, "mem hcd verify --task 2 --json -- 'go test ./p -run Test -v'", verify(false, excerpt), true, process.cwd())) fails.push("1: no ingirió go test -v");
	if (test(store, "backend", "TestA")?.status !== "passed") fails.push("1: TestA " + test(store, "backend", "TestA")?.status);
	const b = test(store, "backend", "TestB");
	if (b?.status !== "failed" || !b?.failure?.message.includes("boom")) fails.push("1: TestB " + JSON.stringify(b));
	if (suite(store, "backend")?.status !== "failed") fails.push("1: suite " + suite(store, "backend")?.status);
	if (!suite(store, "backend")?.partial) fails.push("1: la suite de un extracto debe ser parcial");
}

// 2) pytest en verde: la suite sigue el veredicto de verify.
{
	const store = createTDDStore();
	const excerpt = ["tests/test_x.py::test_ok PASSED", "1 passed in 0.01s"].join("\n");
	ingestHCDVerifyTests(store, "mem hcd verify --task 1 --json -- 'pytest tests/test_x.py -k test_ok -v'", verify(true, excerpt), false, process.cwd());
	if (test(store, "backend", "tests/test_x.py::test_ok")?.status !== "passed") fails.push("2: pytest no ingerido");
	if (suite(store, "backend")?.status !== "passed") fails.push("2: suite " + suite(store, "backend")?.status);
}

// 3) Sin comprobación reconocible (o sin JSON) no se inventa nada.
{
	const store = createTDDStore();
	if (ingestHCDVerifyTests(store, "mem hcd verify --task 1 --json -- 'make check'", verify(true, "ok"), false, process.cwd())) fails.push("3: make check no es un ejecutor reconocido");
	if (ingestHCDVerifyTests(store, "mem hcd verify --task 1 --manual 'visto'", verify(true, ""), false, process.cwd())) fails.push("3: --manual no tiene comprobación");
}
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD056HCDVerifyIngest: %s", out)
	}
}

// HCD-056 tarea 8: una ejecución pedida (run_test_suite) aborta la de watch en
// curso en lugar de esperar detrás de ella; lo abortado no se muestra como fallo.
func TestHCD056WatchAbort(t *testing.T) {
	t.Parallel()

	proj := t.TempDir()
	bin := t.TempDir()
	count := filepath.Join(bin, "count.log")
	writeFile(t, filepath.Join(proj, "go.mod"), "module w\n")
	// La primera ejecución de go test (la de watch) tarda 8 s; las siguientes, nada.
	writeExec(t, filepath.Join(bin, "go"), `if [ "$1" = "test" ]; then
  echo test >> "`+count+`"
  n=$(wc -l < "`+count+`")
  echo '{"Action":"run","Package":"w","Test":"TestA"}'
  [ "$n" -eq 1 ] && sleep 8
  echo '{"Action":"pass","Package":"w","Test":"TestA","Elapsed":0.01}'
fi
exit 0`)
	writeExec(t, filepath.Join(bin, "mem"), `exit 3`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import factory from "./gomemory-tdd.ts";
const commands = {}, handlers = {};
let tool;
factory({
	registerCommand: (n, d) => (commands[n] = d),
	registerTool: (d) => (tool = d),
	on: (e, h) => { (handlers[e] ||= []).push(h); return () => {}; },
	exec: async () => { throw new Error("no"); },
});
const ctx = { cwd: `+jsStr(proj)+`, hasUI: false, ui: { notify: () => {} } };
const emit = async (e, ev) => { for (const h of handlers[e] || []) await h(ev, ctx); };
const fails = [];
await commands.tdd.handler("watch", ctx);
await emit("tool_execution_end", { toolName: "edit", isError: false });
await emit("agent_settled", { aborted: false });
await new Promise((r) => setTimeout(r, 400)); // watch ya está dentro de go test (8 s)
const t0 = Date.now();
const res = await tool.execute("c1", { suite: "backend" }, undefined, undefined, ctx);
const ms = Date.now() - t0;
if (ms > 3000) fails.push("run_test_suite esperó a watch: " + ms + " ms");
const backend = res.details.suites.find((s) => s.id === "backend");
if (backend?.status !== "passed") fails.push("backend tras la ejecución pedida: " + backend?.status);
const failedSuites = res.details.suites.filter((s) => s.status === "failed").map((s) => s.id);
if (failedSuites.length) fails.push("lo abortado no debe verse como fallo: " + failedSuites.join(","));
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD056WatchAbort: %s", out)
	}
}

// HCD-056 tarea 9: un Bash de tests completo (no parcial) cubre los cambios
// previos, como run_test_suite all: watch no lo repite. Uno parcial o un cambio
// posterior sí dejan trabajo a watch.
func TestHCD056WatchDirtyBash(t *testing.T) {
	t.Parallel()

	proj := t.TempDir()
	bin := t.TempDir()
	count := filepath.Join(bin, "count.log")
	writeFile(t, filepath.Join(proj, "go.mod"), "module w\n")
	writeExec(t, filepath.Join(bin, "go"), `[ "$1" = "test" ] && echo test >> "`+count+`"
exit 0`)
	writeExec(t, filepath.Join(bin, "mem"), `exit 3`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import { readFileSync } from "node:fs";
import factory from "./gomemory-tdd.ts";
const commands = {}, handlers = {};
factory({ registerCommand: (n, d) => (commands[n] = d), registerTool: () => {}, on: (e, h) => { (handlers[e] ||= []).push(h); return () => {}; }, exec: async () => {} });
const ctx = { cwd: `+jsStr(proj)+`, hasUI: false, ui: { notify: () => {} } };
const emit = async (e, ev) => { for (const h of handlers[e] || []) await h(ev, ctx); };
const runs = () => { try { return readFileSync(`+jsStr(count)+`, "utf8").split("\n").filter(Boolean).length; } catch { return 0; } };
const idle = () => new Promise((r) => setTimeout(r, 800));
let id = 0;
const bash = async (command) => {
	const toolCallId = "b" + ++id;
	await emit("tool_execution_start", { toolName: "bash", toolCallId, args: { command } });
	await emit("tool_execution_end", { toolName: "bash", toolCallId, isError: false, durationMs: 5, result: { content: [] } });
};
const fails = [];
await commands.tdd.handler("watch", ctx);

// a) edit y luego Bash completo: watch no repite.
await emit("tool_execution_end", { toolName: "edit", isError: false });
await bash("go test ./...");
await emit("agent_settled", { aborted: false });
await idle();
if (runs() !== 0) fails.push("a) tras un Bash completo watch no debe ejecutar; ejecutó " + runs());

// b) Bash parcial: watch sí comprueba el resto.
await emit("tool_execution_end", { toolName: "edit", isError: false });
await bash("go test ./... -run TestA");
await emit("agent_settled", { aborted: false });
await idle();
if (runs() !== 1) fails.push("b) tras un Bash parcial watch debe ejecutar una vez; ejecutó " + runs());

// c) Bash completo y luego un edit: el cambio posterior sí merece watch.
await bash("go test ./...");
await emit("tool_execution_end", { toolName: "edit", isError: false });
await emit("agent_settled", { aborted: false });
await idle();
if (runs() !== 2) fails.push("c) un edit posterior al Bash debe disparar watch; ejecutó " + runs());

// d) Un Bash cualquiera (no de tests) sigue marcando cambios por prudencia.
await bash("touch x");
await emit("agent_settled", { aborted: false });
await idle();
if (runs() !== 3) fails.push("d) un Bash que no es de tests debe disparar watch; ejecutó " + runs());
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD056WatchDirtyBash: %s", out)
	}
}

// HCD-056 tarea 12 (Q1): /tdd activa el paralelismo nativo del ejecutor solo si
// el proyecto ya lo tiene instalado; sin él, el comando no cambia.
func TestHCD056RunnerParallel(t *testing.T) {
	t.Parallel()

	venvXdist := t.TempDir()
	writeFile(t, filepath.Join(venvXdist, "pyproject.toml"), "[project]\nname='x'\n")
	writeExec(t, filepath.Join(venvXdist, ".venv", "bin", "pytest"), "exit 0")
	writeFile(t, filepath.Join(venvXdist, ".venv", "lib", "python3.12", "site-packages", "xdist", "__init__.py"), "")

	uvXdist := t.TempDir()
	writeFile(t, filepath.Join(uvXdist, "pyproject.toml"), "[project]\nname='x'\n")
	writeFile(t, filepath.Join(uvXdist, "uv.lock"), "[[package]]\nname = \"pytest-xdist\"\nversion = \"3.6.1\"\n")

	declaredOnly := t.TempDir()
	writeFile(t, filepath.Join(declaredOnly, "requirements-dev.txt"), "pytest\npytest-xdist\n")

	paratest := t.TempDir()
	writeFile(t, filepath.Join(paratest, "composer.json"), "{}")
	writeExec(t, filepath.Join(paratest, "vendor", "bin", "phpunit"), "exit 0")
	writeExec(t, filepath.Join(paratest, "vendor", "bin", "paratest"), "exit 0")

	phpunitOnly := t.TempDir()
	writeFile(t, filepath.Join(phpunitOnly, "composer.json"), "{}")
	writeExec(t, filepath.Join(phpunitOnly, "vendor", "bin", "phpunit"), "exit 0")

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { detectSuites, filterCheck } from "./gomemory-tdd.ts";
const fails = [];
const check = (dir, id) => detectSuites(dir).find((p) => p.id === "backend")?.checks.find((c) => c.id === id);
const eq = (label, got, want) => { if (JSON.stringify(got) !== JSON.stringify(want)) fails.push(label + ": " + JSON.stringify(got)); };
eq("venv con xdist", check(`+jsStr(venvXdist)+`, "pytest")?.args, ["-q", "-n", "auto"]);
eq("uv.lock con xdist", check(`+jsStr(uvXdist)+`, "pytest")?.args, ["run", "pytest", "-q", "-n", "auto"]);
eq("xdist solo declarado", check(`+jsStr(declaredOnly)+`, "pytest")?.args, ["-q"]);
const pt = check(`+jsStr(paratest)+`, "phpunit");
if (!pt?.cmd.endsWith("vendor/bin/paratest") || pt?.parser !== "phpunit") fails.push("paratest: " + JSON.stringify(pt));
const pu = check(`+jsStr(phpunitOnly)+`, "phpunit");
if (!pu?.cmd.endsWith("vendor/bin/phpunit")) fails.push("sin paratest se usa phpunit: " + JSON.stringify(pu));
// El filtro sigue funcionando con el paralelismo activado.
const f = filterCheck(check(`+jsStr(venvXdist)+`, "pytest"), "test_y");
eq("filtro con xdist", f.args, ["-q", "-n", "auto", "-k", "test_y"]);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD056RunnerParallel: %s", out)
	}
}

// HCD-056 tarea 5 (Q3): las skills HCD que llegan a Pi exigen una comprobación
// acotada por tarea, una única regresión final y el rojo en toda tarea con un
// test automatizable, con ejemplos de varios lenguajes.
func TestHCD056SkillsTDD(t *testing.T) {
	t.Parallel()

	tasks := readAsset(t, "skills/hcd-tasks/SKILL.md")
	for _, want := range []string{"comprobación acotada", "regresión completa", "go test ./pkg/x -run", "pytest", "vitest"} {
		if !strings.Contains(tasks, want) {
			t.Errorf("hcd-tasks/SKILL.md no contiene %q", want)
		}
	}
	impl := readAsset(t, "skills/hcd-implement/SKILL.md")
	for _, want := range []string{"Rojo antes del cambio en toda tarea con un test automatizable", "--expect-fail", "comprobación acotada", "No ejecutes la misma comprobación fuera y dentro del motor", "no automatizable"} {
		if !strings.Contains(impl, want) {
			t.Errorf("hcd-implement/SKILL.md no contiene %q", want)
		}
	}
}

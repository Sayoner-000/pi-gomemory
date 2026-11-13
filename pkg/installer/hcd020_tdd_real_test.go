package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// jsStr devuelve s como literal de cadena JavaScript válido.
func jsStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// writeExec escribe un ejecutable de shell (comando falso en PATH).
func writeExec(t *testing.T, path, body string) {
	t.Helper()
	writeFile(t, path, "#!/bin/sh\n"+body+"\n")
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatal(err)
	}
}

// goFailProject crea un módulo Go real con un test que falla, uno que pasa y uno lento.
func goFailProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module failproj\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "calc", "calc_test.go"), `package calc

import (
	"testing"
	"time"
)

func Suma(a, b int) int { return a - b }

func TestSuma(t *testing.T) {
	if got := Suma(2, 3); got != 5 {
		t.Fatalf("Suma(2,3) = %d, want 5", got)
	}
}

func TestOK(t *testing.T) {}

func TestLento(t *testing.T) { time.Sleep(150 * time.Millisecond) }
`)
	return dir
}

// playwrightLineFixture es la salida REAL de Playwright 1.61 con --reporter=line
// (capturada el 2026-11-11), con la ruta absoluta sustituida por __ROOT__.
const playwrightLineFixture = `
Running 3 tests using 1 worker

[1/3] tests/demo.spec.mjs:3:1 › suma correcta
[2/3] tests/demo.spec.mjs:7:1 › total del carrito
  1) tests/demo.spec.mjs:7:1 › total del carrito › calcular total ──────────────────────────────────

    Error: expect(received).toBe(expected) // Object.is equality

    Expected: 13
    Received: 7

      11 | 	await test.step("calcular total", async () => {
      12 | 		const total = 10 - 3;
    > 13 | 		expect(total).toBe(13);
         | 		              ^
      14 | 	});
      15 | });
      16 |
        at __ROOT__/tests/demo.spec.mjs:13:17
        at __ROOT__/tests/demo.spec.mjs:11:2

    Error Context: test-results/demo-total-del-carrito/error-context.md


[3/3] tests/demo.spec.mjs:17:1 › espera lenta
  1 failed
    tests/demo.spec.mjs:7:1 › total del carrito ────────────────────────────────────────────────────
  2 passed (786ms)
`

// tddToolHarness registra la extensión y deja la tool run_test_suite en `tool`.
const tddToolHarness = `
import factory, * as tdd from "./gomemory-tdd.ts";
let tool = null;
const fakePi = {
	registerCommand: () => {},
	registerTool: (t) => { tool = t; },
	on: () => () => {},
	exec: async () => { throw new Error("pi.exec no debe usarse: no transmite en vivo"); },
};
factory(fakePi);
const fails = [];
const updates = [];
const run = (suite, cwd) => tool.execute("c1", { suite }, undefined, (u) => updates.push(u), { cwd });
const suiteOf = (st, id) => st.suites.find((s) => s.id === id);
`

func TestHCD020SuiteDetection(t *testing.T) {
	empty := t.TempDir()
	full := t.TempDir()
	writeFile(t, filepath.Join(full, "go.mod"), "module x\n")
	writeFile(t, filepath.Join(full, "package.json"), `{"scripts":{"test":"vitest run"}}`)
	writeFile(t, filepath.Join(full, "tsconfig.json"), `{}`)
	writeFile(t, filepath.Join(full, "playwright.config.ts"), `export default {}`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { detectSuites } from "./gomemory-tdd.ts";
const fails = [];
const byId = (plans, id) => plans.find((p) => p.id === id);

const none = detectSuites(`+jsStr(empty)+`);
for (const id of ["backend", "frontend", "playwright", "ci"]) {
	const p = byId(none, id);
	if (!p) { fails.push("falta la suite " + id); continue; }
	if (!p.skipReason) fails.push(id + " debe quedar omitida con motivo en un proyecto vacío");
	if (p.checks.length) fails.push(id + " no debe ejecutar nada en un proyecto vacío");
}

const all = detectSuites(`+jsStr(full)+`);
const cmd = (id) => (byId(all, id)?.checks || []).map((c) => [c.cmd, ...c.args].join(" "));
if (!cmd("backend").includes("go test -json ./...")) fails.push("backend debe ser go test -json ./..., es " + cmd("backend"));
if (!cmd("frontend").includes("npm test")) fails.push("frontend debe ser npm test, es " + cmd("frontend"));
if (!cmd("playwright").includes("npx playwright test --reporter=line")) fails.push("playwright: " + cmd("playwright"));
if (!cmd("ci").includes("go vet ./...") || !cmd("ci").includes("npx tsc --noEmit")) fails.push("ci: " + cmd("ci"));
if (all.some((p) => p.id !== "data" && p.skipReason)) fails.push("con todo presente ninguna suite debe quedar omitida");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD020SuiteDetection: %s", out)
	}
}

func TestHCD020LineStreaming(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { streamLines } from "./gomemory-tdd.ts";
const fails = [];
const seen = [];
const t0 = Date.now();
const res = await streamLines(process.execPath, ["-e", "console.log('uno'); setTimeout(() => console.log('dos'), 400)"], {}, (line) => seen.push([line, Date.now() - t0]));
const end = Date.now() - t0;
if (seen.map((s) => s[0]).join(",") !== "uno,dos") fails.push("líneas recibidas: " + JSON.stringify(seen));
else if (end - seen[0][1] < 300) fails.push("la primera línea debe llegar mientras el proceso sigue vivo (llegó " + seen[0][1] + "ms, fin " + end + "ms)");
if (res.code !== 0 || res.missing) fails.push("resultado: " + JSON.stringify(res));
const miss = await streamLines("comando-que-no-existe-hcd020", [], {}, () => {});
if (!miss.missing) fails.push("un comando inexistente debe marcar missing");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD020LineStreaming: %s", out)
	}
}

func TestHCD020GoFailureDetail(t *testing.T) {
	dir := goFailProject(t)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, tddToolHarness+`
const res = await run("backend", `+jsStr(dir)+`);
const b = suiteOf(res.details, "backend");
const byName = (n) => b?.tests.find((t) => t.name === n);
const suma = byName("TestSuma"), ok = byName("TestOK"), lento = byName("TestLento");
if (!suma || suma.status !== "failed") fails.push("TestSuma debe fallar");
else {
	if (!/calc_test\.go:\d+/.test(suma.failure?.location || "")) fails.push("falta archivo:línea, es " + suma.failure?.location);
	if (!(suma.failure?.message || "").includes("Suma(2,3) = -1, want 5")) fails.push("falta el mensaje real, es " + suma.failure?.message);
}
if (!ok || ok.status !== "passed") fails.push("TestOK debe pasar");
if (!lento || lento.durationMs < 100 || lento.durationMs === 10) fails.push("TestLento debe tener su duración real (≥100ms), tiene " + lento?.durationMs);
if (b?.status !== "failed") fails.push("la suite backend debe quedar failed");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD020GoFailureDetail: %s", out)
	}
}

func TestHCD020PlaywrightLine(t *testing.T) {
	proj := t.TempDir()
	bin := t.TempDir()
	writeFile(t, filepath.Join(proj, "playwright.config.mjs"), "export default {}\n")
	fixture := strings.ReplaceAll(playwrightLineFixture, "__ROOT__", proj)
	writeFile(t, filepath.Join(bin, "pw-out.txt"), fixture)
	// npx falso: emite la salida real con una pausa a mitad para observar el avance en vivo.
	writeExec(t, filepath.Join(bin, "npx"), `[ "$1 $2" = "playwright test" ] || exit 9
if [ -n "$PLAYWRIGHT_JSON_OUTPUT_NAME" ]; then
  cat << 'EOF' > "$PLAYWRIGHT_JSON_OUTPUT_NAME"
{
  "suites": [
    {
      "file": "tests/demo.spec.mjs",
      "line": 3,
      "column": 1,
      "specs": [
        {
          "title": "suma correcta",
          "file": "tests/demo.spec.mjs",
          "line": 3,
          "column": 1,
          "tests": [{ "status": "expected", "results": [{ "status": "passed", "duration": 50 }] }]
        },
        {
          "title": "total del carrito",
          "file": "tests/demo.spec.mjs",
          "line": 7,
          "column": 1,
          "tests": [{ "status": "unexpected", "results": [{ "status": "failed", "duration": 30, "errors": [{ "message": "Expected: 13\nReceived: 7" }] }] }]
        },
        {
          "title": "espera lenta",
          "file": "tests/demo.spec.mjs",
          "line": 17,
          "column": 1,
          "tests": [{ "status": "expected", "results": [{ "status": "passed", "duration": 150 }] }]
        }
      ]
    }
  ],
  "stats": { "expected": 2, "unexpected": 1 }
}
EOF
fi
head -5 "`+filepath.Join(bin, "pw-out.txt")+`"; sleep 0.4; tail -n +6 "`+filepath.Join(bin, "pw-out.txt")+`"; exit 1`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
`+tddToolHarness+`
const res = await run("playwright", `+jsStr(proj)+`);
const p = suiteOf(res.details, "playwright");
const t = (n) => p?.tests.find((x) => x.name === n);
if (!p || p.tests.length !== 3) fails.push("deben registrarse 3 tests, hay " + p?.tests.length);
const carrito = t("total del carrito");
if (carrito?.status !== "failed") fails.push("'total del carrito' debe fallar");
if (carrito?.failure?.location !== "tests/demo.spec.mjs:13:17") fails.push("ubicación: " + carrito?.failure?.location);
if (carrito?.failure?.step !== "calcular total") fails.push("paso fallido: " + carrito?.failure?.step);
if (!/Expected: 13/.test(carrito?.failure?.message || "") || !/Received: 7/.test(carrito?.failure?.message || "")) fails.push("mensaje: " + carrito?.failure?.message);
if (t("suma correcta")?.status !== "passed" || t("espera lenta")?.status !== "passed") fails.push("los otros dos deben pasar");
if (!updates.some((u) => /suma correcta|total del carrito/.test(JSON.stringify(u.details?.suites?.find((s) => s.id === "playwright")?.tests || [])) && u.details?.status === "running")) fails.push("debe verse el avance en vivo mientras corre");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD020PlaywrightLine: %s", out)
	}
}

func TestHCD020NoFakeResults(t *testing.T) {
	empty := t.TempDir()
	bin := t.TempDir()
	writeExec(t, filepath.Join(bin, "mem"), `echo '{"ok":false,"error":{"code":"not_found"}}'; exit 3`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
`+tddToolHarness+`
const res = await run("all", `+jsStr(empty)+`);
const st = res.details;
if (st.stats.passed !== 0 || st.stats.total !== 0) fails.push("en un proyecto vacío nada debe pasar ni contarse: " + JSON.stringify(st.stats));
for (const s of st.suites) if (s.status !== "skipped" || !s.skipReason) fails.push(s.id + " debe quedar omitida con motivo, está " + s.status);
if (!/unidad HCD/.test(suiteOf(st, "data")?.skipReason || "")) fails.push("data sin unidad HCD activa: " + suiteOf(st, "data")?.skipReason);
if (st.status === "passed" || st.status === "failed") fails.push("estado global no debe inventar resultado: " + st.status);
const syn = tdd.classifyFailure("SyntaxError: Unexpected token '{'");
if (/contrato/i.test(syn.summary) || !/sintaxis|compilaci/i.test(syn.summary)) fails.push("SyntaxError mal diagnosticado: " + syn.summary);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD020NoFakeResults: %s", out)
	}
}

// panelState es un estado con una suite activa (un test corriendo, uno fallido,
// dos aprobados y uno t.Skip) y tres suites omitidas.
const panelState = `
const theme = { fg: (_c, s) => s, style: (s) => s };
const state = {
	status: "running", progressPercent: 50, durationMs: 0, watchMode: false,
	stats: { total: 4, passed: 2, failed: 1, running: 1, queued: 0 },
	suites: [
		{ id: "backend", name: "Backend", domain: "backend", status: "running", tests: [
			{ id: "a", name: "TestAprobadoUno", domain: "backend", status: "passed", durationMs: 4, steps: [] },
			{ id: "b", name: "TestAprobadoDos", domain: "backend", status: "passed", durationMs: 4, steps: [] },
			{ id: "c", name: "TestEnCurso", domain: "backend", status: "running", durationMs: 0, steps: [] },
			{ id: "d", name: "TestRoto", domain: "backend", status: "failed", durationMs: 7, steps: [],
				failure: { message: "Suma(2,3) = -1, want 5", location: "calc/calc_test.go:12" },
				diagnostic: { category: "code_defect", summary: "Discrepancia", suggestedFix: "x" } },
			{ id: "e", name: "TestSaltado", domain: "backend", status: "skipped", durationMs: 0, steps: [] },
		] },
		{ id: "frontend", name: "Frontend", domain: "frontend", status: "skipped", skipReason: "sin script test en package.json", tests: [] },
		{ id: "playwright", name: "Playwright E2E", domain: "playwright", status: "skipped", skipReason: "sin playwright.config", tests: [] },
		{ id: "data", name: "Data & Graph", domain: "data", status: "skipped", skipReason: "sin unidad HCD activa", tests: [] },
	],
};
`

func TestHCD020PanelShowsEnabled(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { renderTDDOverlay } from "./gomemory-tdd.ts";
`+panelState+`
const text = renderTDDOverlay(state, 60, theme).join("\n");
const fails = [];
if (!text.includes("TestEnCurso")) fails.push("debe listar el test en ejecución");
if (!text.includes("TestRoto") || !text.includes("calc/calc_test.go:12") || !text.includes("want 5")) fails.push("debe listar el fallido con ubicación y mensaje");
if (text.includes("TestAprobadoUno") || text.includes("TestAprobadoDos")) fails.push("los aprobados se resumen, no se listan uno a uno");
if (!/2 aprobados/.test(text)) fails.push("falta el resumen de aprobados");
if (text.includes("TestSaltado")) fails.push("los tests t.Skip no se listan");
const all = text.split("\n");
const head = all.findIndex((l) => /Omitidas \(3\)/.test(l));
if (head < 0) fails.push("falta la cabecera «Omitidas (3)»");
for (const [n, why] of [["Frontend", "sin script test"], ["Playwright E2E", "sin playwright.config"], ["Data & Graph", "sin unidad HCD activa"]]) {
	const idx = all.map((l, i) => (l.includes(n) ? i : -1)).filter((i) => i >= 0);
	if (idx.length !== 1 || idx[0] < head || !all[idx[0]].includes("–") || !all[idx[0]].includes(why)) fails.push(n + " debe aparecer una vez, bajo «Omitidas», con su motivo");
}
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD020PanelShowsEnabled: %s", out)
	}
}

func TestHCD020PanelIcons(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { renderTDDOverlay, renderTDDMinimized } from "./gomemory-tdd.ts";
`+panelState+`
const text = [...renderTDDOverlay(state, 60, theme), ...renderTDDMinimized(state, 60, theme)].join("\n");
const fails = [];
const emoji = text.match(/[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}\u{231A}-\u{23FF}\u{2B00}-\u{2BFF}]/gu)?.filter((c) => c !== "✓" && c !== "✗") || [];
if (emoji.length) fails.push("quedan emojis: " + [...new Set(emoji)].join(" "));
const BRAILLE = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];
const running = text.split("\n").find((l) => l.includes("TestEnCurso")) || "";
if (!BRAILLE.some((f) => running.includes(f))) fails.push("el test en curso debe usar el spinner braille de pi-tui: " + running.trim());
const failed = text.split("\n").find((l) => l.includes("TestRoto")) || "";
if (!failed.includes("✗")) fails.push("el fallido debe usar ✗");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD020PanelIcons: %s", out)
	}
}

func TestHCD020AgentLoop(t *testing.T) {
	dir := goFailProject(t)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, tddToolHarness+`
const res = await run("backend", `+jsStr(dir)+`);
const text = res.content.map((c) => c.text).join("\n");
for (const want of ["TestSuma", "calc_test.go:", "Suma(2,3) = -1, want 5", "Backend"]) {
	if (!text.includes(want)) fails.push("el informe al agente no incluye " + JSON.stringify(want));
}
if (!/sugerencia|Sugerencia/.test(text)) fails.push("el informe debe incluir la sugerencia del diagnóstico");
if (updates.length === 0) fails.push("onUpdate no se llamó durante la ejecución");
for (const u of updates) if (!Array.isArray(u.content) || !("details" in u)) fails.push("onUpdate sin content/details");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD020AgentLoop: %s", out)
	}
}

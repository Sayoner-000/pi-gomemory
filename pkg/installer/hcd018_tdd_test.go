package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHCD018TDDTypesAndStore verifica que TDDStore gestiona el estado, suites, tests, pasos y emite eventos
func TestHCD018TDDTypesAndStore(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { createTDDStore } from "./gomemory-tdd.ts";

const store = createTDDStore();
const fails = [];

let state = store.getState();
if (!state || state.status !== "idle" || !Array.isArray(state.suites)) {
	fails.push("estado inicial invalido");
}

let notificationCount = 0;
const unsubscribe = store.subscribe((newState) => {
	notificationCount++;
	state = newState;
});

// Registrar suite y paso
store.initSuites([
	{ id: "backend", name: "Backend API", domain: "backend" },
	{ id: "playwright", name: "Playwright E2E", domain: "playwright" }
]);

store.startTest("backend", "test-auth", "User Authentication");
store.startStep("backend", "test-auth", "step-1", "Generate JWT");
store.finishStep("backend", "test-auth", "step-1", "passed", 45);

state = store.getState();
const backendSuite = state.suites.find(s => s.id === "backend");
if (!backendSuite || backendSuite.tests.length !== 1) {
	fails.push("no se registro el test en backend");
} else {
	const step = backendSuite.tests[0].steps.find(st => st.id === "step-1");
	if (!step || step.status !== "passed" || step.durationMs !== 45) {
		fails.push("el paso step-1 no paso a estado passed con duracion 45");
	}
}

if (notificationCount < 3) {
	fails.push("no se emitieron notificaciones a los suscriptores: " + notificationCount);
}

unsubscribe();
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)

	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD018TDDTypesAndStore fallo: %s", out)
	}
}

// TestHCD018TDDAdapters verifica la deteccion y adaptadores para backend, playwright, frontend, ci y data
func TestHCD018TDDAdapters(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { 
	getAvailableDomains, 
	parseGoTestJsonLine, 
	parsePlaywrightStepLine,
	classifyFailure
} from "./gomemory-tdd.ts";

const fails = [];
const domains = getAvailableDomains();
const expected = ["backend", "frontend", "playwright", "ci", "data"];
for (const exp of expected) {
	if (!domains.includes(exp)) {
		fails.push("falta dominio: " + exp);
	}
}

// Parser de go test JSON
const goPassed = parseGoTestJsonLine('{"Action":"pass","Test":"TestLogin","Elapsed":0.05}');
if (!goPassed || goPassed.status !== "passed" || goPassed.testName !== "TestLogin") {
	fails.push("parseGoTestJsonLine no reconocio pass");
}

const goFailed = parseGoTestJsonLine('{"Action":"fail","Test":"TestOrder","Elapsed":0.12}');
if (!goFailed || goFailed.status !== "failed") {
	fails.push("parseGoTestJsonLine no reconocio fail");
}

// Parser de Playwright Step
const pwStep = parsePlaywrightStepLine('[playwright] Step: wait for #login-button (locator.waitFor)');
if (!pwStep || pwStep.stepName !== "wait for #login-button (locator.waitFor)") {
	fails.push("parsePlaywrightStepLine no reconocio step de locator");
}

// Clasificador de causa raiz (Inspirado en TestSprite)
const diagTimeout = classifyFailure("Timeout 5000ms waiting for locator('#submit')");
if (!diagTimeout || diagTimeout.category !== "selector_or_timeout") {
	fails.push("classifyFailure no clasifico timeout de selector");
}

const diagAssertion = classifyFailure("AssertionError: expected 200 to equal 401");
if (!diagAssertion || diagAssertion.category !== "code_defect") {
	fails.push("classifyFailure no clasifico defecto de codigo");
}

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)

	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD018TDDAdapters fallo: %s", out)
	}
}

// TestHCD018TDDUIRenderers verifica renderizado de overlay lateral flotante, widget y vista modal
func TestHCD018TDDUIRenderers(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { 
	renderTDDOverlay, 
	renderTDDWidget, 
	formatProgressBar 
} from "./gomemory-tdd.ts";

const fails = [];
const fakeTheme = {
	fg: (color, text) => text,
	style: (text, opts) => text,
};

// Barra de progreso
const bar = formatProgressBar(0.75, 10);
if (!bar || bar.length !== 10) {
	fails.push("formatProgressBar longitud incorrecta: " + bar);
}

const mockState = {
	status: "running",
	progressPercent: 75,
	durationMs: 1250,
	suites: [
		{
			id: "backend",
			name: "Backend",
			domain: "backend",
			status: "passed",
			tests: [
				{
					id: "t1",
					name: "TestAuth",
					status: "passed",
					durationMs: 12,
					steps: [{ id: "s1", name: "Issue Token", status: "passed", durationMs: 12 }]
				}
			]
		},
		{
			id: "playwright",
			name: "Playwright E2E",
			domain: "playwright",
			status: "running",
			tests: [
				{
					id: "t2",
					name: "US-01 Checkout",
					status: "running",
					durationMs: 450,
					steps: [
						{ id: "s2-1", name: "Add to cart", status: "passed", durationMs: 120 },
						{ id: "s2-2", name: "Wait for button", status: "running", durationMs: 330 }
					]
				}
			]
		}
	],
	stats: { total: 2, passed: 1, failed: 0, running: 1, queued: 0 }
};

// Overlay lateral flotante (ancho 44)
const overlayLines = renderTDDOverlay(mockState, 44, fakeTheme);
if (!Array.isArray(overlayLines) || overlayLines.length < 5) {
	fails.push("renderTDDOverlay lineas insuficientes");
}
const overlayText = overlayLines.join("\n");
if (!overlayText.includes("TDD") || !overlayText.includes("Backend") || !overlayText.includes("Playwright")) {
	fails.push("renderTDDOverlay no contiene encabezados clave");
}

// Widget compacto (para ctx.ui.setWidget)
const widgetLines = renderTDDWidget(mockState, fakeTheme);
if (!Array.isArray(widgetLines) || widgetLines.length === 0) {
	fails.push("renderTDDWidget no devolvio lineas");
}
const widgetText = widgetLines.join(" ");
if (!widgetText.includes("75%") || !widgetText.includes("Backend")) {
	fails.push("renderTDDWidget no incluye métricas esenciales");
}

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)

	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD018TDDUIRenderers fallo: %s", out)
	}
}

// TestHCD018TDDCommandRegistration verifica registro de /tdd y tool run_test_suite
func TestHCD018TDDCommandRegistration(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import tddExtensionFactory from "./gomemory-tdd.ts";

const commands = {};
const tools = [];
const events = [];

const fakePi = {
	registerCommand: (name, def) => {
		commands[name] = def;
	},
	registerTool: (def) => {
		tools.push(def);
	},
	on: (event, handler) => {
		events.push(event);
		return () => {};
	},
	exec: async () => ({ code: 0, stdout: "" }),
};

tddExtensionFactory(fakePi);

const fails = [];
if (!commands["tdd"]) {
	fails.push("comando /tdd no registrado");
}

if (!tools.some(t => t.name === "run_test_suite")) {
	fails.push("tool run_test_suite no registrada");
}

for (const e of ["tool_execution_end", "agent_settled"]) {
	if (!events.includes(e)) fails.push("evento " + e + " no suscrito para watch mode");
}
// turn_end se espera en cada turno: suscribir watch ahí bloquea al agente en bucle.
if (events.includes("turn_end")) fails.push("watch no debe suscribirse a turn_end");

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)

	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD018TDDCommandRegistration fallo: %s", out)
	}
}

// TestHCD018TDDInstallerIntegration valida que gomemory-tdd.ts se empaqueta e instala
func TestHCD018TDDInstallerIntegration(t *testing.T) {
	opts := sandbox(t)
	if err := InstallTUIExtension(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	destPath := filepath.Join(opts.PiAgentDir, "extensions", "gomemory-tdd.ts")
	if _, err := os.Stat(destPath); err != nil {
		t.Fatalf("gomemory-tdd.ts no se instala en %s: %v", destPath, err)
	}
}

// TestHCD018TDDMinimizedPanel verifica que el panel TDD minimizado ocupa una sola fila
// y que /tdd min está registrado, para no solapar el sidebar en la esquina superior derecha.
func TestHCD018TDDMinimizedPanel(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import tddExtensionFactory, { renderTDDMinimized, createTDDStore } from "./gomemory-tdd.ts";

const theme = { fg: (_c, s) => s, style: (s) => s };
const store = createTDDStore();
store.initSuites([{ id: "backend", name: "Backend", domain: "backend" }]);
const lines = renderTDDMinimized(store.getState(), 44, theme);

const fails = [];
if (lines.length !== 1) fails.push("minimizado debe ocupar 1 fila, tiene " + lines.length);
if (!lines[0].includes("[+]")) fails.push("falta indicador [+] para restaurar");

const commands = {};
const fakePi = {
	registerCommand: (name, def) => { commands[name] = def; },
	registerTool: () => {},
	on: () => () => {},
	exec: async () => ({ code: 0, stdout: "" }),
};
tddExtensionFactory(fakePi);
if (!commands["tdd"]) fails.push("comando /tdd no registrado");

console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)

	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD018TDDMinimizedPanel fallo: %s", out)
	}
}

/**
 * gomemory-tdd.ts — Extensión y comando /tdd de pi-gomemory
 *
 * TDD Live Inspector:
 * - Visualización paso a paso de pruebas (Setup -> Action -> Assertion -> Evidence)
 * - Multi-dominio unificado: Backend, Frontend, CLI / Playwright, CI, Data Testing
 * - Estados en vivo con latencia, barra de progreso y spinners
 * - Diagnóstico heurístico de causa raíz (Bug vs Selector/Timeout vs Entorno)
 * - Panel lateral flotante overlay no bloqueante (anchor: top-right) y visor modal
 */

import { spawn, type ChildProcess } from "node:child_process";
import { randomUUID } from "node:crypto";
import { existsSync, readdirSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { isAbsolute, join, relative } from "node:path";
import { truncateToWidth, visibleWidth } from "@earendil-works/pi-tui";
import type { ExtensionAPI, ExtensionCommandContext, Theme } from "@earendil-works/pi-coding-agent";

// ── 1. Tipos y Modelos de Dominio ──────────────────────────────────────────

export type SuiteDomain = "backend" | "frontend" | "playwright" | "ci" | "data";
export type StepStatus = "pending" | "running" | "passed" | "failed" | "flaky" | "skipped";

export interface Step {
	id: string;
	name: string;
	status: StepStatus;
	durationMs: number;
	error?: string;
	evidence?: string;
}

/** Fallo real de un test: mensaje del runner, ubicación archivo:línea y, si lo hay, el paso fallido. */
export interface TestFailure {
	message: string;
	location?: string;
	step?: string;
}

export interface TestCase {
	id: string;
	name: string;
	domain: SuiteDomain;
	status: StepStatus;
	durationMs: number;
	steps: Step[];
	failure?: TestFailure;
	diagnostic?: FailureDiagnostic;
}

export interface TestSuite {
	id: string;
	name: string;
	domain: SuiteDomain;
	status: StepStatus;
	durationMs?: number;
	/** Indica si el último resultado proviene de una ejecución parcial/filtrada. */
	partial?: boolean;
	/** Motivo por el que la suite no se ejecutó (no aplica al proyecto o falta la herramienta). */
	skipReason?: string;
	tests: TestCase[];
}

export interface FailureDiagnostic {
	category: "code_defect" | "selector_or_timeout" | "env_or_service" | "unknown";
	summary: string;
	suggestedFix: string;
}

export interface TDDStats {
	total: number;
	passed: number;
	failed: number;
	flaky: number;
	running: number;
	queued: number;
}

export interface TDDState {
	status: "idle" | "running" | "passed" | "failed" | "flaky";
	progressPercent: number;
	durationMs: number;
	suites: TestSuite[];
	activeStep?: string;
	stats: TDDStats;
	lastUpdated?: number;
	watchMode?: boolean;
}

export interface TDDStore {
	getState(): TDDState;
	subscribe(listener: (state: TDDState) => void): () => void;
	initSuites(suites: Array<{ id: string; name: string; domain: SuiteDomain }>): void;
	startTest(suiteId: string, testId: string, testName: string): void;
	startStep(suiteId: string, testId: string, stepId: string, stepName: string): void;
	finishStep(
		suiteId: string,
		testId: string,
		stepId: string,
		status: StepStatus,
		durationMs: number,
		error?: string,
		evidence?: string,
	): void;
	finishTest(suiteId: string, testId: string, status: StepStatus, durationMs: number, failure?: TestFailure): void;
	finishSuite(suiteId: string, status: StepStatus, durationMs?: number, partial?: boolean): void;
	/** Vacía los tests de una suite antes de una nueva ejecución. */
	clearSuite(suiteId: string): void;
	/** Marca la suite como omitida con su motivo: ni éxito ni fallo. */
	skipSuite(suiteId: string, reason: string): void;
	setWatchMode(enabled: boolean): void;
	reset(): void;
}

// ── 2. Almacén Reactivo de Estado (TDDStore) ────────────────────────────────

export function createTDDStore(): TDDStore {
	let state: TDDState = {
		status: "idle",
		progressPercent: 0,
		durationMs: 0,
		suites: [],
		stats: { total: 0, passed: 0, failed: 0, flaky: 0, running: 0, queued: 0 },
		watchMode: false,
	};

	const listeners = new Set<(state: TDDState) => void>();

	function notify() {
		recalculateMetrics();
		const snapshot = { ...state };
		listeners.forEach((fn) => {
			try {
				fn(snapshot);
			} catch {
				// Suscriptor falló
			}
		});
	}

	function recalculateMetrics() {
		let total = 0;
		let passed = 0;
		let failed = 0;
		let flaky = 0;
		let running = 0;
		let queued = 0;

		for (const suite of state.suites) {
			for (const test of suite.tests) {
				if (test.status === "skipped") continue; // t.Skip / test.skip: ni pendiente ni resultado
				total++;
				if (test.status === "passed") passed++;
				else if (test.status === "failed") failed++;
				else if (test.status === "flaky") flaky++;
				else if (test.status === "running") running++;
				else queued++;
			}
		}

		state.stats = { total, passed, failed, flaky, running, queued };
		state.progressPercent = total > 0 ? Math.round(((passed + flaky) / total) * 100) : 0;
		state.lastUpdated = Date.now();

		// Entre un test y el siguiente no hay ninguno en curso, pero la suite sigue viva.
		if (running > 0 || state.suites.some((s) => s.status === "running")) {
			state.status = "running";
		} else if (failed > 0 || state.suites.some((s) => s.status === "failed")) {
			state.status = "failed";
		} else if (flaky > 0 || state.suites.some((s) => s.status === "flaky")) {
			state.status = "flaky";
		} else if (total > 0 && passed === total) {
			state.status = "passed";
		} else {
			state.status = "idle";
		}
	}

	return {
		getState() {
			return state;
		},

		subscribe(listener) {
			listeners.add(listener);
			return () => listeners.delete(listener);
		},

		initSuites(suiteDefs) {
			state.suites = suiteDefs.map((def) => ({
				id: def.id,
				name: def.name,
				domain: def.domain,
				status: "pending",
				tests: [],
			}));
			notify();
		},

		startTest(suiteId, testId, testName) {
			const suite = state.suites.find((s) => s.id === suiteId);
			if (!suite) return;

			let test = suite.tests.find((t) => t.id === testId);
			if (!test) {
				test = {
					id: testId,
					name: testName,
					domain: suite.domain,
					status: "running",
					durationMs: 0,
					steps: [],
				};
				suite.tests.push(test);
			} else {
				test.status = "running";
			}
			suite.status = "running";
			notify();
		},

		startStep(suiteId, testId, stepId, stepName) {
			const suite = state.suites.find((s) => s.id === suiteId);
			const test = suite?.tests.find((t) => t.id === testId);
			if (!test) return;

			let step = test.steps.find((st) => st.id === stepId);
			if (!step) {
				step = {
					id: stepId,
					name: stepName,
					status: "running",
					durationMs: 0,
				};
				test.steps.push(step);
			} else {
				step.status = "running";
			}
			state.activeStep = `${test.name} → ${stepName}`;
			notify();
		},

		finishStep(suiteId, testId, stepId, status, durationMs, error, evidence) {
			const suite = state.suites.find((s) => s.id === suiteId);
			const test = suite?.tests.find((t) => t.id === testId);
			const step = test?.steps.find((st) => st.id === stepId);
			if (!step) return;

			step.status = status;
			step.durationMs = durationMs;
			if (error) step.error = error;
			if (evidence) step.evidence = evidence;

			if (status === "failed" && error && test) {
				test.diagnostic = classifyFailure(error);
			}

			notify();
		},

		finishTest(suiteId, testId, status, durationMs, failure) {
			const suite = state.suites.find((s) => s.id === suiteId);
			const test = suite?.tests.find((t) => t.id === testId);
			if (!test) return;

			test.status = status;
			test.durationMs = durationMs;
			if (failure) {
				test.failure = failure;
				test.diagnostic = classifyFailure(failure.message);
			}
			notify();
		},

		clearSuite(suiteId) {
			const suite = state.suites.find((s) => s.id === suiteId);
			if (!suite) return;
			suite.tests = [];
			suite.status = "pending";
			suite.partial = undefined;
			suite.skipReason = undefined;
			suite.durationMs = undefined;
			notify();
		},

		skipSuite(suiteId, reason) {
			const suite = state.suites.find((s) => s.id === suiteId);
			if (!suite) return;
			suite.status = "skipped";
			suite.partial = undefined;
			suite.skipReason = reason;
			notify();
		},

		finishSuite(suiteId, status, durationMs, partial) {
			const suite = state.suites.find((s) => s.id === suiteId);
			if (!suite) return;

			suite.status = status;
			if (durationMs !== undefined) suite.durationMs = durationMs;
			if (partial !== undefined) suite.partial = partial;
			notify();
		},

		setWatchMode(enabled) {
			state.watchMode = enabled;
			notify();
		},

		reset() {
			state = {
				status: "idle",
				progressPercent: 0,
				durationMs: 0,
				suites: [],
				stats: { total: 0, passed: 0, failed: 0, flaky: 0, running: 0, queued: 0 },
				watchMode: state.watchMode,
			};
			notify();
		},
	};
}

// ── 3. Adaptadores de Ejecución y Diagnóstico Heurístico ────────────────────

export function getAvailableDomains(): SuiteDomain[] {
	return ["backend", "frontend", "playwright", "ci", "data"];
}

export function parseGoTestJsonLine(line: string): { status: StepStatus; testName?: string; elapsed?: number } | null {
	try {
		const parsed = JSON.parse(line.trim());
		if (!parsed.Test) return null;

		const elapsedMs = typeof parsed.Elapsed === "number" ? Math.round(parsed.Elapsed * 1000) : undefined;
		if (parsed.Action === "pass") {
			return { status: "passed", testName: parsed.Test, elapsed: elapsedMs };
		}
		if (parsed.Action === "fail") {
			return { status: "failed", testName: parsed.Test, elapsed: elapsedMs };
		}
		if (parsed.Action === "run") {
			return { status: "running", testName: parsed.Test };
		}
		if (parsed.Action === "skip") {
			return { status: "skipped", testName: parsed.Test };
		}
	} catch {
		// No es JSON válido de go test
	}
	return null;
}

export function parsePlaywrightStepLine(line: string): { stepName: string; status?: StepStatus } | null {
	const stepMatch = line.match(/(?:\[playwright\]\s+Step:|\bStep:)\s*(.+)/i);
	if (stepMatch) {
		return { stepName: stepMatch[1].trim(), status: "running" };
	}
	return null;
}

export function classifyFailure(errorMessage: string): FailureDiagnostic {
	const msg = errorMessage.toLowerCase();

	if (msg.includes("timeout") || msg.includes("waiting for locator") || msg.includes("selector")) {
		return {
			category: "selector_or_timeout",
			summary: "Fragilidad en selector UI o tiempo de espera excedido",
			suggestedFix: "Verifica si el elemento tarda en renderizar (añade page.waitForLoadState() o revisa el selector)",
		};
	}

	// Antes que las aserciones: «unexpected token» contiene «expected» y no es un fallo de contrato.
	if (/syntaxerror|unexpected token|cannot find module|build failed|\[build failed\]|undefined: |declared and not used|cannot use .* as/.test(msg)) {
		return {
			category: "code_defect",
			summary: "Error de compilación o sintaxis",
			suggestedFix: "Corrige el error de compilación en la ubicación indicada antes de volver a ejecutar",
		};
	}

	if (/assertionerror|\bexpected\b|\bequal\b|\bwant\b|\bgot\b|\breceived\b/.test(msg)) {
		return {
			category: "code_defect",
			summary: "Discrepancia en contrato o aserción lógica de la prueba",
			suggestedFix: "Revisa la lógica de la función o el valor de retorno esperado en el contrato de API",
		};
	}

	if (
		msg.includes("econnrefused") ||
		msg.includes("connection refused") ||
		msg.includes("no such file") ||
		msg.includes("enotfound")
	) {
		return {
			category: "env_or_service",
			summary: "Servicio dependiente inaccesible o fallo de entorno",
			suggestedFix: "Asegúrate de que la base de datos o el servidor mock local esté activo y escuchando en el puerto",
		};
	}

	return {
		category: "unknown",
		summary: "Fallo durante la ejecución de la prueba",
		suggestedFix: "Inspecciona la traza completa de logs para aislar el punto de ruptura",
	};
}

// ── 3b. Detección de suites y ejecución real con streaming ─────────────────

export type CheckParser = "go-test" | "go-text" | "exit-code" | "playwright-line" | "hcd" | "pytest" | "vitest" | "phpunit" | "java-test";

/** Un comando real que verifica algo del proyecto. */
export interface CheckSpec {
	id: string;
	name: string;
	cmd: string;
	args: string[];
	parser: CheckParser;
}

/** Qué ejecutar en una suite, o por qué no aplica al proyecto. */
export interface SuitePlan {
	id: SuiteDomain;
	name: string;
	checks: CheckSpec[];
	skipReason?: string;
}

const PLAYWRIGHT_CONFIGS = ["ts", "js", "mjs", "cjs", "mts", "cts"].map((e) => `playwright.config.${e}`);
const VITEST_CONFIGS = ["ts", "js", "mts", "mjs", "cts", "cjs"].map((e) => `vitest.config.${e}`);
const ESLINT_CONFIGS = ["eslint.config.js", "eslint.config.mjs", "eslint.config.cjs", "eslint.config.ts", ".eslintrc", ".eslintrc.json", ".eslintrc.js", ".eslintrc.cjs", ".eslintrc.yml", ".eslintrc.yaml"];
const STYLELINT_CONFIGS = [".stylelintrc", ".stylelintrc.json", ".stylelintrc.js", ".stylelintrc.cjs", ".stylelintrc.yml", ".stylelintrc.yaml", "stylelint.config.js", "stylelint.config.mjs", "stylelint.config.cjs"];
const NPM_DEFAULT_TEST = /no test specified/;

function packageTestScript(cwd: string): string | undefined {
	try {
		const pkg = JSON.parse(readFileSync(join(cwd, "package.json"), "utf8"));
		const script = pkg?.scripts?.test;
		return typeof script === "string" && script.trim() && !NPM_DEFAULT_TEST.test(script) ? script : undefined;
	} catch {
		return undefined;
	}
}

function readTextSafe(path: string): string {
	try {
		return readFileSync(path, "utf8");
	} catch {
		return "";
	}
}

function composerHasTestScript(cwd: string): boolean {
	try {
		const pkg = JSON.parse(readFileSync(join(cwd, "composer.json"), "utf8"));
		return typeof pkg?.scripts?.test === "string" && pkg.scripts.test.trim().length > 0;
	} catch {
		return false;
	}
}

/**
 * pytest-xdist instalado (no solo declarado) en la .venv o fijado en uv.lock:
 * `-n auto` con xdist ausente rompería pytest y cambiaría el veredicto (HCD-056, Q1).
 */
function hasPytestXdist(cwd: string): boolean {
	if (/name\s*=\s*"pytest-xdist"/.test(readTextSafe(join(cwd, "uv.lock")))) return true;
	try {
		const lib = join(cwd, ".venv", "lib");
		return readdirSync(lib).some((py) => existsSync(join(lib, py, "site-packages", "xdist")));
	} catch {
		return false;
	}
}

/** Decide las suites según los archivos del proyecto; lo que no aplica queda omitido con su motivo. */
export function detectSuites(cwd: string): SuitePlan[] {
	const has = (f: string) => existsSync(join(cwd, f));
	const goMod = has("go.mod");
	const tsconfig = has("tsconfig.json");
	const pwConfig = PLAYWRIGHT_CONFIGS.some(has);
	const pyProject = has("pyproject.toml");
	const pyProjectText = pyProject ? readTextSafe(join(cwd, "pyproject.toml")) : "";
	const pythonProject = pyProject || has("pytest.ini") || has("setup.cfg") || has("requirements.txt") || has("requirements-dev.txt");
	const phpProject = has("composer.json") || has("phpunit.xml") || has("phpunit.xml.dist");
	const javaMaven = has("pom.xml");
	const javaGradle = has("build.gradle") || has("build.gradle.kts");

	const plan = (id: SuiteDomain, name: string, checks: CheckSpec[], skipReason: string): SuitePlan =>
		checks.length ? { id, name, checks } : { id, name, checks, skipReason };

	const backend: CheckSpec[] = [];
	if (goMod) {
		backend.push({ id: "go-test", name: "go test", cmd: "go", args: ["test", "-json", "./..."], parser: "go-test" });
	}
	if (pythonProject) {
		// Paralelismo nativo solo si ya está instalado (HCD-056, Q1).
		const xdist = hasPytestXdist(cwd) ? ["-n", "auto"] : [];
		const venvPytest = join(cwd, ".venv", "bin", "pytest");
		if (existsSync(venvPytest)) {
			backend.push({ id: "pytest", name: "pytest", cmd: venvPytest, args: ["-q", ...xdist], parser: "pytest" });
		} else if (has("uv.lock")) {
			backend.push({ id: "pytest", name: "uv run pytest", cmd: "uv", args: ["run", "pytest", "-q", ...xdist], parser: "pytest" });
		} else {
			backend.push({ id: "pytest", name: "pytest", cmd: "pytest", args: ["-q", ...xdist], parser: "pytest" });
		}
	}
	if (phpProject) {
		const vendorPhpunit = join(cwd, "vendor", "bin", "phpunit");
		const vendorParatest = join(cwd, "vendor", "bin", "paratest");
		if (existsSync(vendorParatest)) {
			// paratest ejecuta phpunit en procesos paralelos con la misma salida.
			backend.push({ id: "phpunit", name: "paratest", cmd: vendorParatest, args: [], parser: "phpunit" });
		} else if (existsSync(vendorPhpunit)) {
			backend.push({ id: "phpunit", name: "phpunit", cmd: vendorPhpunit, args: [], parser: "phpunit" });
		} else if (composerHasTestScript(cwd)) {
			backend.push({ id: "phpunit", name: "composer test", cmd: "composer", args: ["test"], parser: "phpunit" });
		} else if (has("phpunit.xml") || has("phpunit.xml.dist")) {
			backend.push({ id: "phpunit", name: "phpunit", cmd: "phpunit", args: [], parser: "phpunit" });
		}
	}
	if (javaMaven) {
		backend.push({ id: "mvn-test", name: "mvn test", cmd: has("mvnw") ? "./mvnw" : "mvn", args: ["test"], parser: "java-test" });
	} else if (javaGradle) {
		backend.push({ id: "gradle-test", name: "gradle test", cmd: has("gradlew") ? "./gradlew" : "gradle", args: ["test"], parser: "java-test" });
	}

	const ci: CheckSpec[] = [];
	if (goMod) ci.push({ id: "go-vet", name: "go vet ./...", cmd: "go", args: ["vet", "./..."], parser: "exit-code" });
	if (tsconfig) ci.push({ id: "tsc", name: "tsc --noEmit", cmd: "npx", args: ["tsc", "--noEmit"], parser: "exit-code" });
	if (pythonProject && (has("ruff.toml") || has(".ruff.toml") || /\[tool\.ruff\]/.test(pyProjectText))) {
		ci.push({ id: "ruff", name: "ruff check .", cmd: "ruff", args: ["check", "."], parser: "exit-code" });
	}
	if (pythonProject && (has("mypy.ini") || /\[tool\.mypy\]/.test(pyProjectText))) {
		ci.push({ id: "mypy", name: "mypy .", cmd: "mypy", args: ["."], parser: "exit-code" });
	}
	if (phpProject) {
		ci.push({ id: "php-lint", name: "php -l", cmd: "sh", args: ["-c", "find . -maxdepth 4 -name '*.php' -not -path './vendor/*' -print0 | xargs -0 -r -n1 php -l"], parser: "exit-code" });
	}
	if (ESLINT_CONFIGS.some(has)) {
		ci.push({ id: "eslint", name: "eslint .", cmd: "npx", args: ["eslint", "."], parser: "exit-code" });
	}
	if (has(".htmlhintrc")) {
		ci.push({ id: "htmlhint", name: "htmlhint", cmd: "npx", args: ["htmlhint"], parser: "exit-code" });
	}
	if (STYLELINT_CONFIGS.some(has)) {
		ci.push({ id: "stylelint", name: "stylelint", cmd: "npx", args: ["stylelint", "**/*.css"], parser: "exit-code" });
	}

	const frontend: CheckSpec[] = [];
	if (packageTestScript(cwd)) {
		frontend.push({ id: "npm-test", name: "npm test", cmd: "npm", args: ["test"], parser: "exit-code" });
	} else if (VITEST_CONFIGS.some(has)) {
		frontend.push({ id: "vitest", name: "vitest run", cmd: "npx", args: ["vitest", "run"], parser: "vitest" });
	}

	return [
		plan("backend", "Backend", backend, "sin go.mod, Python, PHP ni Java"),
		plan("frontend", "Frontend", frontend, "sin script test en package.json ni vitest.config"),
		plan("playwright", "Playwright E2E", pwConfig ? [{ id: "pw", name: "playwright test", cmd: "npx", args: ["playwright", "test", "--reporter=line"], parser: "playwright-line" }] : [], "sin playwright.config"),
		plan("ci", "CI & Static", ci, "sin checks estáticos configurados"),
		{ id: "data", name: "Data & Graph", checks: [{ id: "hcd-show", name: "Unidad HCD activa", cmd: "mem", args: ["hcd", "show", "--json"], parser: "hcd" }] },
	];
}

/**
 * Traduce un filtro de nombre de test a la sintaxis del ejecutor (HCD-056).
 * Los checks sin filtro traducible (estáticos, `npm test`, HCD) se omiten con su
 * motivo: ejecutarlos completos rompería el ciclo rojo→verde acotado.
 */
export function filterCheck(check: CheckSpec, filter: string): CheckSpec | { skip: string } {
	const withArgs = (args: string[]): CheckSpec => ({ ...check, name: `${check.name} (${filter})`, args });
	switch (check.parser) {
		case "go-test": {
			const pkgs = check.args.indexOf("./...");
			const at = pkgs >= 0 ? pkgs : check.args.length;
			return withArgs([...check.args.slice(0, at), "-run", filter, ...check.args.slice(at)]);
		}
		case "pytest":
			return withArgs([...check.args, "-k", filter]);
		case "vitest":
			return withArgs([...check.args, "-t", filter]);
		case "playwright-line":
			return withArgs([...check.args, "-g", filter]);
		case "phpunit":
			return withArgs(check.cmd === "composer" ? [...check.args, "--", "--filter", filter] : [...check.args, "--filter", filter]);
		case "java-test":
			return check.id === "mvn-test"
				? withArgs([`-Dtest=${filter}`, "-Dsurefire.failIfNoSpecifiedTests=false", ...check.args])
				: withArgs([...check.args, "--tests", filter]);
		default:
			return { skip: `${check.name}: no admite filtro` };
	}
}

export interface StreamResult {
	code: number;
	/** El comando no existe en PATH. */
	missing: boolean;
	/** La ejecución se interrumpió por timeout. */
	timedOut?: boolean;
	/** La ejecución fue cancelada por señal de aborto. */
	aborted?: boolean;
	/** Últimas líneas de salida, para diagnosticar un fallo sin tests reconocidos. */
	tail: string[];
}

/**
 * Mata el grupo de procesos acotado del hijo en POSIX (detached: true)
 * o el proceso individual si no tiene grupo propio o está en Windows.
 * Nunca envía señales a grupos no positivos (<= 1) para evitar señalizar al anfitrión.
 */
function killProcessGroup(child: ChildProcess, signal: NodeJS.Signals = "SIGTERM"): void {
	const pid = child?.pid;
	if (!pid || pid <= 1) return;
	if (process.platform !== "win32") {
		try {
			process.kill(-pid, signal);
			return;
		} catch {
			// El grupo puede haber terminado ya; fallback al proceso
		}
	}
	try {
		child.kill(signal);
	} catch {
		// proceso ya terminado
	}
}

/** Línea más larga que admite streamLines; por encima la ejecución falla (HCD-038, Q2). */
export const MAX_LINE_CHARS = 64 * 1024;
const MAX_TAIL_LINES = 40;
const MAX_TAIL_LINE_CHARS = 1024;
const MAX_TAIL_BYTES = 64 * 1024;

/**
 * Ejecuta un comando sin shell y entrega cada línea (stdout y stderr) en cuanto llega.
 * pi.exec no sirve aquí: solo devuelve la salida cuando el proceso termina.
 */
export function streamLines(
	cmd: string,
	args: string[],
	opts: { cwd?: string; signal?: AbortSignal; timeoutMs?: number; env?: NodeJS.ProcessEnv },
	onLine: (line: string) => void,
): Promise<StreamResult> {
	return new Promise((resolve) => {
		// tail es solo diagnóstico: extractos acotados en número, longitud y bytes
		// UTF-8. El parser recibe la línea completa por onLine (HCD-038, Q2).
		const tail: string[] = [];
		let tailBytes = 0;
		const push = (line: string) => {
			const excerpt = line.length > MAX_TAIL_LINE_CHARS ? line.slice(-MAX_TAIL_LINE_CHARS) : line;
			tail.push(excerpt);
			tailBytes += Buffer.byteLength(excerpt);
			while (tail.length > MAX_TAIL_LINES || (tailBytes > MAX_TAIL_BYTES && tail.length > 1)) {
				tailBytes -= Buffer.byteLength(tail.shift()!);
			}
			try {
				onLine(line);
			} catch {
				// un parser defectuoso no debe tumbar la ejecución
			}
		};

		if (opts.signal?.aborted) {
			resolve({ code: 1, missing: false, aborted: true, timedOut: false, tail });
			return;
		}

		let child: ReturnType<typeof spawn>;
		try {
			child = spawn(cmd, args, {
				cwd: opts.cwd,
				shell: false,
				stdio: ["ignore", "pipe", "pipe"],
				detached: process.platform !== "win32",
				env: opts.env ? { ...process.env, ...opts.env } : process.env,
			});
		} catch (err: any) {
			const isMissing = err?.code === "ENOENT";
			resolve({
				code: isMissing ? 127 : 1,
				missing: isMissing,
				aborted: Boolean(opts.signal?.aborted),
				timedOut: false,
				tail,
			});
			return;
		}

		let timedOut = false;
		let aborted = false;
		let missing = false;
		let terminated = false;
		let killTimer: NodeJS.Timeout | undefined;
		let timeoutTimer: NodeJS.Timeout | undefined;

		const terminate = (sig: NodeJS.Signals = "SIGTERM") => {
			if (terminated) return;
			terminated = true;
			killProcessGroup(child, sig);
			killTimer = setTimeout(() => {
				killProcessGroup(child, "SIGKILL");
				child.stdout?.destroy();
				child.stderr?.destroy();
			}, 800);
		};

		if (opts.timeoutMs) {
			timeoutTimer = setTimeout(() => {
				timedOut = true;
				terminate("SIGTERM");
			}, opts.timeoutMs);
		}

		const onAbort = () => {
			aborted = true;
			terminate("SIGTERM");
		};

		if (opts.signal) {
			opts.signal.addEventListener("abort", onAbort, { once: true });
		}

		// Una línea más larga que MAX_LINE_CHARS no se trunca para seguir: se corta
		// el proceso y la ejecución falla, para que ningún resultado salga de una
		// salida que no se pudo leer entera (HCD-038, Q1).
		let outputLimited = false;
		const rejectLongLine = () => {
			if (outputLimited) return;
			outputLimited = true;
			push(`Salida rechazada: una línea excede el límite de ${MAX_LINE_CHARS / 1024} Ki caracteres`);
			terminate("SIGTERM");
		};

		for (const stream of [child.stdout, child.stderr]) {
			let buf = "";
			stream?.setEncoding("utf8");
			stream?.on("data", (chunk: string) => {
				if (outputLimited) return;
				buf += chunk;
				let nl: number;
				while ((nl = buf.indexOf("\n")) >= 0) {
					const line = buf.slice(0, nl).replace(/\r$/, "");
					if (line.length > MAX_LINE_CHARS) {
						buf = "";
						rejectLongLine();
						return;
					}
					push(line);
					buf = buf.slice(nl + 1);
				}
				if (buf.length > MAX_LINE_CHARS) {
					buf = "";
					rejectLongLine();
				}
			});
			stream?.on("end", () => {
				if (buf && !outputLimited) push(buf);
				buf = "";
			});
		}

		child.on("error", (err: any) => {
			if (err?.code === "ENOENT") missing = true;
		});

		child.on("close", (code, signal) => {
			if (timeoutTimer) clearTimeout(timeoutTimer);
			// Si ya se pidió terminar, el SIGKILL al grupo sigue programado: el cierre
			// del hijo directo no implica que sus descendientes hayan muerto (HCD-035, H-03).
			if (killTimer && !terminated) clearTimeout(killTimer);
			if (opts.signal) {
				opts.signal.removeEventListener("abort", onAbort);
			}

			let exitCode = code ?? (signal ? 1 : 0);
			if (missing) {
				exitCode = 127;
			} else if (timedOut) {
				exitCode = exitCode !== 0 ? exitCode : 124;
			} else if (aborted) {
				exitCode = exitCode !== 0 ? exitCode : 1;
			}
			if (outputLimited && exitCode === 0) exitCode = 1;

			resolve({
				code: exitCode,
				missing,
				timedOut,
				aborted,
				tail,
			});
		});
	});
}

const ANSI = /\x1b\[[0-9;?]*[A-Za-z]/g;
const clean = (s: string) => s.replace(ANSI, "");
const clip = (s: string, max = 600) => (s.length > max ? `${s.slice(0, max - 1)}…` : s);

/** Ruta relativa al proyecto cuando es posible, para que archivo:línea sea legible. */
function relLocation(loc: string, cwd?: string): string {
	if (!cwd || !isAbsolute(loc)) return loc;
	const r = relative(cwd, loc);
	return r.startsWith("..") ? loc : r;
}

/** Extrae mensaje y archivo:línea de la salida de un test Go fallido. */
export function goFailureFromOutput(lines: string[]): TestFailure {
	const body = lines.map((l) => l.replace(/\n$/, "")).filter((l) => !/^\s*(=== (RUN|PAUSE|CONT)|--- (FAIL|PASS|SKIP))/.test(l));
	const at = body.findIndex((l) => /^\s+[\w./-]+\.go:\d+: /.test(l));
	if (at >= 0) {
		const m = body[at].match(/^\s+([\w./-]+\.go:\d+): (.*)$/)!;
		const more = body.slice(at + 1).filter((l) => /^\s{8,}\S/.test(l)).map((l) => l.trim());
		return { location: m[1], message: clip([m[2], ...more].join("\n")) };
	}
	const panicLoc = body.map((l) => l.match(/([\w-]+_test\.go:\d+)/)?.[1]).find(Boolean);
	return { location: panicLoc, message: clip(body.map((l) => l.trim()).filter(Boolean).join("\n") || "Fallo sin salida") };
}

interface LineParser {
	onLine(line: string): void;
	/** Cierra lo pendiente al terminar el proceso; devuelve si hubo algún fallo. */
	finish(res: StreamResult): boolean;
}

/** Parser en vivo de `go test -json`: un test por evento run/pass/fail con su Output real. */
export function goTestParser(store: TDDStore, suiteId: SuiteDomain): LineParser {
	const output = new Map<string, string[]>();
	const pkgOutput = new Map<string, string[]>();
	const runningTests = new Set<string>();
	let failed = false;
	const keyOf = (ev: any) => `${ev.Package ?? ""}::${ev.Test}`;
	return {
		onLine(line) {
			let ev: any;
			try {
				ev = JSON.parse(line);
			} catch {
				return;
			}
			const pkg = ev.Package ?? ev.ImportPath ?? "";
			if (!ev.Test) {
				if (ev.Action === "output" || ev.Action === "build-output") {
					const arr = pkgOutput.get(pkg) ?? [];
					arr.push(ev.Output ?? "");
					pkgOutput.set(pkg, arr.slice(-60));
				} else if (ev.Action === "fail" && pkg) {
					// Fallo de paquete sin tests fallidos: compilación o setup.
					const anyTest = [...output.keys()].some((k) => k.startsWith(`${pkg}::`));
					if (!anyTest) {
						const id = `${pkg}::(paquete)`;
						store.startTest(suiteId, id, `${pkg} (compilación)`);
						store.finishTest(suiteId, id, "failed", Math.round((ev.Elapsed ?? 0) * 1000), goFailureFromOutput(pkgOutput.get(pkg) ?? []));
						failed = true;
					}
				}
				return;
			}
			const key = keyOf(ev);
			if (ev.Action === "run") {
				output.set(key, []);
				runningTests.add(key);
				store.startTest(suiteId, key, ev.Test);
			} else if (ev.Action === "output") {
				const arr = output.get(key) ?? [];
				arr.push(ev.Output ?? "");
				output.set(key, arr.slice(-60));
			} else if (ev.Action === "pass" || ev.Action === "skip" || ev.Action === "fail") {
				runningTests.delete(key);
				const ms = Math.round((ev.Elapsed ?? 0) * 1000);
				if (ev.Action === "fail") {
					failed = true;
					store.finishTest(suiteId, key, "failed", ms, goFailureFromOutput(output.get(key) ?? []));
				} else {
					store.finishTest(suiteId, key, ev.Action === "pass" ? "passed" : "skipped", ms);
				}
			}
		},
		finish(res) {
			let interruptedReason = "";
			if (res.timedOut) {
				interruptedReason = "interrumpido por límite de tiempo (timeout)";
			} else if (res.aborted) {
				interruptedReason = "interrumpido por cancelación (abort)";
			} else if (res.code !== 0) {
				interruptedReason = `proceso terminado anormalmente con código ${res.code}`;
			}

			if (runningTests.size > 0) {
				for (const key of runningTests) {
					failed = true;
					const testOutput = output.get(key) ?? [];
					let failure = goFailureFromOutput(testOutput);
					if (!failure || failure.message === "Fallo sin salida") {
						const tailMsg = res.tail.filter((l) => l.trim()).slice(-5).join("\n");
						const msg = interruptedReason || (tailMsg ? `go test: ${tailMsg}` : `go test terminó sin emitir resultado (código ${res.code})`);
						failure = { message: clip(msg) };
					} else if (interruptedReason) {
						failure.message = clip(`${failure.message}\n(${interruptedReason})`);
					}
					store.finishTest(suiteId, key, "failed", 0, failure);
				}
				runningTests.clear();
			}

			if (res.code !== 0 && !failed) {
				const id = "go-test::(proceso)";
				store.startTest(suiteId, id, "go test");
				const reason = interruptedReason || clip(res.tail.join("\n") || `go test salió con código ${res.code}`);
				store.finishTest(suiteId, id, "failed", 0, { message: reason });
				failed = true;
			}
			store.finishSuite(suiteId, failed ? "failed" : "passed");
			return failed;
		},
	};
}

function reconcilePlaywrightJson(
	store: TDDStore,
	suiteId: SuiteDomain,
	report: any,
	cwd?: string,
	lineFailures?: Map<string, TestFailure>,
): boolean {
	if (!report || !Array.isArray(report.suites)) {
		return false;
	}

	const existingTests = store.getState().suites.find((s) => s.id === suiteId)?.tests ?? [];

	const walkSuite = (s: any) => {
		if (Array.isArray(s.suites)) {
			for (const childSuite of s.suites) walkSuite(childSuite);
		}
		if (Array.isArray(s.specs)) {
			for (const spec of s.specs) {
				const file = spec.file || s.file || "";
				const line = spec.line ?? s.line ?? 0;
				const col = spec.column ?? s.column ?? 0;
				const loc = `${file}:${line}:${col}`;
				const title = spec.title || "";

				if (Array.isArray(spec.tests)) {
					for (const test of spec.tests) {
						const project = test.projectName || "";
						const id = project ? `[${project}]::${loc}::${title}` : `${loc}::${title}`;
						const displayName = project ? `[${project}] › ${title}` : title;

						const matched = existingTests.find((t) => {
							if (t.id === id) return true;
							const projMatch = project ? t.id.includes(`[${project}]`) : !t.id.includes("[");
							const titleMatch = t.name.includes(title) || t.id.endsWith(`::${title}`);
							const lineMatch = line > 0 ? t.id.includes(`:${line}:`) : true;
							return projMatch && titleMatch && lineMatch;
						});
						const finalId = matched ? matched.id : id;

						let status: StepStatus = "passed";
						let failure: TestFailure | undefined;
						let durationMs = 0;

						if (Array.isArray(test.results)) {
							for (const r of test.results) {
								durationMs += r.duration ?? 0;
							}
						}

						if (test.status === "flaky") {
							status = "flaky";
							const lineFail = lineFailures?.get(finalId);
							const firstFail = test.results?.find(
								(r: any) => r.status === "failed" || r.status === "timedOut" || (r.errors && r.errors.length > 0),
							);
							const errMsg = firstFail?.errors?.[0]?.message || firstFail?.error?.message;
							if (errMsg) {
								failure = {
									message: clip(clean(errMsg)),
									location: lineFail?.location || relLocation(loc, cwd),
									step: lineFail?.step,
								};
							} else {
								failure = lineFail || { message: "Fallo inicial en prueba inestable (flaky)" };
							}
						} else if (test.status === "unexpected") {
							status = "failed";
							const lineFail = lineFailures?.get(finalId);
							const lastFail = test.results?.find(
								(r: any) => r.status === "failed" || r.status === "timedOut" || (r.errors && r.errors.length > 0),
							);
							const errMsg = lastFail?.errors?.[0]?.message || lastFail?.error?.message;
							if (errMsg) {
								failure = {
									message: clip(clean(errMsg)),
									location: lineFail?.location || relLocation(loc, cwd),
									step: lineFail?.step,
								};
							} else {
								failure = lineFail || { message: "Fallo inesperado de prueba" };
							}
						} else if (test.status === "skipped") {
							status = "skipped";
						} else if (test.status === "expected") {
							status = "passed";
						}

						if (!matched) {
							store.startTest(suiteId, finalId, displayName);
						}
						store.finishTest(suiteId, finalId, status, durationMs, failure);
					}
				}
			}
		}
	};

	for (const s of report.suites) {
		walkSuite(s);
	}
	return true;
}

/**
 * Parser en vivo del reporter `line` de Playwright, reconciliado con el informe JSON oficial.
 * «[n/t] [proyecto] › archivo:l:c › título» inicia el test.
 * «N) [proyecto] › archivo:l:c › título › paso ───» abre su fallo.
 */
export function playwrightLineParser(
	store: TDDStore,
	suiteId: SuiteDomain,
	cwd?: string,
	jsonPath?: string,
): LineParser {
	const START = /^\[\d+\/\d+\]\s+(?:\[([^\]]+)\]\s+›\s+)?(\S+:\d+:\d+)\s+›\s+(.+?)\s*$/;
	const FAIL_HEAD = /^\s+\d+\)\s+(?:\[([^\]]+)\]\s+›\s+)?(\S+:\d+:\d+)\s+›\s+(.+?)\s*─*\s*$/;
	const SUMMARY = /^\s+\d+\s+(failed|passed|flaky|skipped|interrupted|did not run)\b/;
	const titles = new Map<string, string>(); // id → cleanTitle
	const locs = new Map<string, string>(); // id → loc
	const lineFailures = new Map<string, TestFailure>();
	const started = new Set<string>();
	let sawPassedSummary = false;
	let current: { id: string; step?: string; lines: string[] } | null = null;

	const idOf = (project: string | undefined, loc: string, title: string) => {
		const cleanTitle = title.replace(/\s+\(retry\s+#\d+\)$/, "").trim();
		return project ? `[${project}]::${loc}::${cleanTitle}` : `${loc}::${cleanTitle}`;
	};

	const closeFailure = () => {
		if (!current) return;
		const body = current.lines.map((l) => l.trim());
		const msg = body.filter((l) => /^(Error|Expected|Received|Timeout|TimeoutError|expect)\b|^Call log:/.test(l));
		const at = body.map((l) => l.match(/^at\s+(.+:\d+:\d+)$/)?.[1]).find(Boolean);
		const loc = locs.get(current.id) || current.id;
		const failure: TestFailure = {
			message: clip((msg.length ? msg : body.filter(Boolean).slice(0, 6)).join("\n")),
			location: at ? relLocation(at, cwd) : relLocation(loc, cwd),
			step: current.step,
		};
		lineFailures.set(current.id, failure);
		store.finishTest(suiteId, current.id, "failed", 0, failure);
		current = null;
	};

	return {
		onLine(raw) {
			const line = clean(raw);
			const start = line.match(START);
			if (start) {
				closeFailure();
				const [, project, loc, title] = start;
				const cleanTitle = title.replace(/\s+\(retry\s+#\d+\)$/, "").trim();
				const id = idOf(project, loc, cleanTitle);
				titles.set(id, cleanTitle);
				locs.set(id, loc);
				started.add(id);
				const displayName = project ? `[${project}] › ${cleanTitle}` : cleanTitle;
				if (!lineFailures.has(id)) {
					store.startTest(suiteId, id, displayName);
				}
				return;
			}
			const head = line.match(FAIL_HEAD);
			if (head) {
				closeFailure();
				const [, project, loc, rest] = head;
				const titlePart = rest.split(" › ")[0].replace(/\s+\(retry\s+#\d+\)$/, "").trim();
				const id = idOf(project, loc, titlePart);
				if (!titles.has(id)) {
					titles.set(id, titlePart);
					locs.set(id, loc);
					started.add(id);
					const displayName = project ? `[${project}] › ${titlePart}` : titlePart;
					store.startTest(suiteId, id, displayName);
				}
				const cleanTitle = titles.get(id) || titlePart;
				const step = rest.startsWith(`${cleanTitle} › `) ? rest.slice(cleanTitle.length + 3) : undefined;
				current = { id, step, lines: [] };
				return;
			}
			if (SUMMARY.test(line)) {
				closeFailure();
				if (/\b\d+\s+passed\b/.test(line)) {
					sawPassedSummary = true;
				}
				return;
			}
			current?.lines.push(line);
		},
		finish(res) {
			closeFailure();

			let jsonReportParsed = false;
			if (jsonPath && existsSync(jsonPath)) {
				try {
					const content = readFileSync(jsonPath, "utf8");
					if (content.trim()) {
						const data = JSON.parse(content);
						jsonReportParsed = reconcilePlaywrightJson(store, suiteId, data, cwd, lineFailures);
					}
				} catch {
					jsonReportParsed = false;
				} finally {
					try {
						rmSync(jsonPath, { force: true });
					} catch {}
				}
			}

			if (!jsonReportParsed) {
				// Si no hay JSON oficial, solo se aprueban tests si la ejecución concluyó
				// con la línea de resumen de aprobados. Una interrupción sin resumen NUNCA aprueba pruebas (C-006 / C-11).
				if (sawPassedSummary) {
					for (const id of started) {
						if (lineFailures.has(id)) {
							store.finishTest(suiteId, id, "failed", 0, lineFailures.get(id));
						} else {
							store.finishTest(suiteId, id, "passed", 0);
						}
					}
				} else {
					for (const id of started) {
						const existing = lineFailures.get(id);
						const failure = existing || {
							message: clip(res.tail.join("\n") || `playwright finalizó sin informe de resultados (código ${res.code})`),
							location: locs.get(id) ? relLocation(locs.get(id)!, cwd) : undefined,
						};
						store.finishTest(suiteId, id, "failed", 0, failure);
					}
				}
				if (started.size === 0 && res.code !== 0) {
					const id = "pw::(proceso)";
					store.startTest(suiteId, id, "playwright test");
					store.finishTest(suiteId, id, "failed", 0, {
						message: clip(res.tail.join("\n") || `playwright salió con código ${res.code}`),
					});
				}
			}

			const suiteBefore = store.getState().suites.find((s) => s.id === suiteId);
			if (res.code !== 0 && !suiteBefore?.tests.some((t) => t.status === "failed")) {
				const id = "pw::(proceso)";
				store.startTest(suiteId, id, "playwright test");
				store.finishTest(suiteId, id, "failed", 0, {
					message: clip(res.tail.join("\n") || `playwright salió con código ${res.code}`),
				});
			}

			const suite = store.getState().suites.find((s) => s.id === suiteId);
			const anyFailed = suite?.tests.some((t) => t.status === "failed");
			const anyFlaky = suite?.tests.some((t) => t.status === "flaky");
			const allPassed = suite?.tests.length ? suite.tests.every((t) => t.status === "passed") : false;
			const suiteStatus: StepStatus = anyFailed ? "failed" : anyFlaky ? "flaky" : res.code !== 0 ? "failed" : allPassed ? "passed" : (jsonReportParsed ? "passed" : "failed");
			store.finishSuite(suiteId, suiteStatus);

			return Boolean(anyFailed) || res.code !== 0;
		},
	};
}

/** Parser para salida de pytest (-v, -q o resumen). */
export function pytestParser(store: TDDStore, suiteId: SuiteDomain, fallbackName = "pytest"): LineParser {
	let recorded = 0;
	let failed = false;
	const seen = new Set<string>();
	const failureLines: string[] = [];
	let lastLocation: string | undefined;

	return {
		onLine(raw) {
			const line = clean(raw).trimEnd();
			const locMatch = line.match(/^([\w./-]+\.py:\d+):/);
			if (locMatch) lastLocation = locMatch[1];

			// Formato: "tests/test_api.py::test_ok PASSED" o "FAILED tests/test_api.py::test_fail - msg"
			const prefixMatch = line.match(/^(PASSED|FAILED|SKIPPED|ERROR)\s+([^\s]+)(?:\s+-\s+(.*))?$/);
			const suffixMatch = !prefixMatch ? line.match(/^([^\s]+::[^\s]+)\s+(PASSED|FAILED|SKIPPED|ERROR)\b(.*)$/) : null;
			const statusWord = prefixMatch?.[1] ?? suffixMatch?.[2];
			const testId = prefixMatch?.[2] ?? suffixMatch?.[1];
			const detail = (prefixMatch?.[3] ?? suffixMatch?.[3] ?? "").trim();

			if (statusWord && testId && !seen.has(testId)) {
				seen.add(testId);
				recorded++;
				store.startTest(suiteId, testId, testId);
				if (statusWord === "PASSED") {
					store.finishTest(suiteId, testId, "passed", 0);
				} else if (statusWord === "SKIPPED") {
					store.finishTest(suiteId, testId, "skipped", 0);
				} else {
					failed = true;
					store.finishTest(suiteId, testId, "failed", 0, {
						message: clip(detail || `${statusWord} ${testId}`),
						location: lastLocation,
					});
				}
				return;
			}
			if (line.trim()) failureLines.push(line);
		},
		finish(res) {
			if (res.code !== 0) failed = true;
			if (recorded === 0) {
				const id = `pytest::${fallbackName}`;
				store.startTest(suiteId, id, fallbackName);
				if (!failed) {
					store.finishTest(suiteId, id, "passed", 0);
				} else {
					const tail = failureLines.slice(-12).join("\n") || res.tail.map(clean).slice(-12).join("\n") || `salió con código ${res.code}`;
					store.finishTest(suiteId, id, "failed", 0, { message: clip(tail), location: lastLocation });
				}
			}
			return failed;
		},
	};
}

/** Parser para salida de Vitest (reporter default/verbose). */
export function vitestParser(store: TDDStore, suiteId: SuiteDomain, fallbackName = "vitest"): LineParser {
	let recorded = 0;
	let failed = false;
	const failureLines: string[] = [];

	return {
		onLine(raw) {
			const line = clean(raw);
			const m = line.match(/^\s*([✓×↓])\s+(.+?)(?:\s+(\d+(?:\.\d+)?)\s*m?s)?\s*$/);
			if (m) {
				const [, glyph, name, dur] = m;
				const id = `vitest::${name.trim()}`;
				const ms = dur ? Math.round(parseFloat(dur)) : 0;
				recorded++;
				store.startTest(suiteId, id, name.trim());
				if (glyph === "✓") {
					store.finishTest(suiteId, id, "passed", ms);
				} else if (glyph === "↓") {
					store.finishTest(suiteId, id, "skipped", ms);
				} else {
					failed = true;
					store.finishTest(suiteId, id, "failed", ms, { message: clip(failureLines.slice(-6).join("\n") || name.trim()) });
				}
				return;
			}
			if (line.trim()) failureLines.push(line.trim());
		},
		finish(res) {
			if (res.code !== 0) failed = true;
			if (recorded === 0) {
				const id = `vitest::${fallbackName}`;
				store.startTest(suiteId, id, fallbackName);
				if (!failed) {
					store.finishTest(suiteId, id, "passed", 0);
				} else {
					store.finishTest(suiteId, id, "failed", 0, { message: clip(failureLines.slice(-12).join("\n") || res.tail.map(clean).slice(-12).join("\n") || `salió con código ${res.code}`) });
				}
			}
			return failed;
		},
	};
}

/** Parser para salida de PHPUnit. */
export function phpunitParser(store: TDDStore, suiteId: SuiteDomain, fallbackName = "phpunit"): LineParser {
	let recorded = 0;
	let failed = false;
	const lines: string[] = [];

	return {
		onLine(raw) {
			const line = clean(raw).trim();
			if (!line) return;
			lines.push(line);
			const failCase = line.match(/^\d+\)\s+([^\s]+::[^\s]+)$/);
			if (failCase) {
				const id = failCase[1];
				recorded++;
				failed = true;
				store.startTest(suiteId, id, id);
				store.finishTest(suiteId, id, "failed", 0, { message: `Fallo en ${id}` });
			}
		},
		finish(res) {
			if (res.code !== 0) failed = true;
			if (recorded === 0) {
				const id = `phpunit::${fallbackName}`;
				store.startTest(suiteId, id, fallbackName);
				if (!failed) {
					store.finishTest(suiteId, id, "passed", 0);
				} else {
					store.finishTest(suiteId, id, "failed", 0, { message: clip(lines.slice(-12).join("\n") || res.tail.map(clean).slice(-12).join("\n") || `salió con código ${res.code}`) });
				}
			}
			return failed;
		},
	};
}

/** Parser para salida de Maven Surefire y Gradle test. */
export function javaTestParser(store: TDDStore, suiteId: SuiteDomain, fallbackName = "java test"): LineParser {
	let recorded = 0;
	let failed = false;
	const lines: string[] = [];

	return {
		onLine(raw) {
			const line = clean(raw).trim();
			if (!line) return;
			lines.push(line);
			// Maven: [ERROR]   OrderServiceTest.rejectsEmpty:34 expected: ...
			const mvnFail = line.match(/^\[ERROR\]\s+([A-Za-z0-9_.$]+):(\d+)\s+(.*)$/);
			if (mvnFail) {
				const [, name, ln, msg] = mvnFail;
				recorded++;
				failed = true;
				store.startTest(suiteId, name, name);
				store.finishTest(suiteId, name, "failed", 0, { message: clip(msg), location: `${name}:${ln}` });
				return;
			}
			// Gradle: ClassName > methodName FAILED
			const gradleFail = line.match(/^([A-Za-z0-9_.$]+\s+>\s+[^\s]+)\s+FAILED$/);
			if (gradleFail) {
				const name = gradleFail[1];
				recorded++;
				failed = true;
				store.startTest(suiteId, name, name);
				store.finishTest(suiteId, name, "failed", 0, { message: `Fallo en ${name}` });
			}
		},
		finish(res) {
			if (res.code !== 0) failed = true;
			if (recorded === 0) {
				const id = `java::${fallbackName}`;
				store.startTest(suiteId, id, fallbackName);
				if (!failed) {
					store.finishTest(suiteId, id, "passed", 0);
				} else {
					store.finishTest(suiteId, id, "failed", 0, { message: clip(lines.slice(-12).join("\n") || res.tail.map(clean).slice(-12).join("\n") || `salió con código ${res.code}`) });
				}
			}
			return failed;
		},
	};
}

export interface BashTestMatch {
	suiteId: SuiteDomain;
	parser: CheckParser;
	name: string;
	hcdTasks?: readonly string[];
	partial: boolean;
}

interface ShellToken {
	text: string;
	quoted: boolean;
}

interface ShellSegment {
	tokens: ShellToken[];
	nextOp?: string;
}

/** Divide una línea de shell en segmentos respetando comillas, escapes y comentarios. */
function parseShellSegments(command: string): ShellSegment[] | null {
	const segments: ShellSegment[] = [];
	let tokens: ShellToken[] = [];
	let cur = "";
	let curQuoted = false;
	let hasToken = false;
	let quote: "'" | '"' | null = null;

	const flushToken = () => {
		if (!hasToken) return;
		tokens.push({ text: cur, quoted: curQuoted });
		cur = "";
		curQuoted = false;
		hasToken = false;
	};
	const flushSegment = (op?: string) => {
		flushToken();
		if (tokens.length > 0 || op) {
			segments.push({ tokens, nextOp: op });
			tokens = [];
		}
	};

	for (let i = 0; i < command.length; i++) {
		const ch = command[i];
		if (quote === "'") {
			if (ch === "'") {
				quote = null;
			} else {
				cur += ch;
			}
			continue;
		}
		if (quote === '"') {
			if (ch === "\\" && i + 1 < command.length) {
				cur += command[++i];
			} else if (ch === '"') {
				quote = null;
			} else {
				cur += ch;
			}
			continue;
		}
		if (ch === "'" || ch === '"') {
			quote = ch;
			curQuoted = true;
			hasToken = true;
			continue;
		}
		if (ch === "\\" && i + 1 < command.length) {
			cur += command[++i];
			hasToken = true;
			continue;
		}
		if (ch === "#" && !hasToken) {
			while (i < command.length && command[i] !== "\n") i++;
			flushSegment("\n");
			continue;
		}
		if (ch === "<" && command[i + 1] === "<") {
			// Heredoc: no se infiere resultado de tests de bloques compuestos.
			return null;
		}
		if (ch === "\n" || ch === ";") {
			flushSegment(ch);
			continue;
		}
		if (ch === "&" || ch === "|") {
			const op = command[i + 1] === ch ? ch + ch : ch;
			if (op.length === 2) i++;
			flushSegment(op);
			continue;
		}
		if (/\s/.test(ch)) {
			flushToken();
			continue;
		}
		cur += ch;
		hasToken = true;
	}
	if (quote !== null) return null;
	flushSegment();
	return segments.filter((s) => s.tokens.length > 0 || s.nextOp);
}

function baseCommandName(token: string): string {
	const slash = token.lastIndexOf("/");
	return slash >= 0 ? token.slice(slash + 1) : token;
}

function stripCommandPrefixes(tokens: ShellToken[]): ShellToken[] {
	let i = 0;
	while (i < tokens.length) {
		const t = tokens[i];
		if (!t.quoted && /^[A-Za-z_][A-Za-z0-9_]*=/.test(t.text)) {
			i++;
			continue;
		}
		if (!t.quoted && (t.text === "env" || t.text === "time" || t.text === "command")) {
			i++;
			continue;
		}
		break;
	}
	return tokens.slice(i);
}

function hasPositionalTarget(args: string[], valueFlags: Set<string>, ignored?: Set<string>): boolean {
	for (let i = 0; i < args.length; i++) {
		const a = args[i];
		if (a === "--") return i + 1 < args.length;
		if (a.startsWith("-")) {
			if (valueFlags.has(a)) i++;
			continue;
		}
		if (ignored?.has(a)) continue;
		return true;
	}
	return false;
}

function matchTestSegment(rawTokens: ShellToken[]): BashTestMatch | null {
	const tokens = stripCommandPrefixes(rawTokens);
	if (tokens.length === 0) return null;
	const texts = tokens.map((t) => t.text);
	const exe = baseCommandName(texts[0]);

	// mem hcd verify
	if (exe === "mem" && texts[1] === "hcd" && texts[2] === "verify") {
		const separator = texts.indexOf("--", 3);
		const flags = texts.slice(3, separator < 0 ? undefined : separator);
		// El motor acepta --task N[,M] y --task=N[,M]. No leer flags de la comprobación.
		let taskList: string | undefined;
		for (let i = 0; i < flags.length; i++) {
			if (flags[i] === "--manual" || flags[i] === "--client") i++;
			else if (flags[i] === "--task") taskList = flags[++i];
			else if (flags[i].startsWith("--task=")) taskList = flags[i].slice(7);
		}
		const hcdTasks = taskList && /^\d+(?:,\d+)*$/.test(taskList) ? [...new Set(taskList.split(","))] : [];
		return { suiteId: "data", parser: "hcd", name: "mem hcd verify", partial: taskList !== undefined, hcdTasks };
	}

	// pytest directo, uv run pytest, poetry run pytest o python -m pytest
	let pytestArgs: string[] | null = null;
	if (exe === "pytest") {
		pytestArgs = texts.slice(1);
	} else if ((exe === "uv" || exe === "poetry") && texts[1] === "run" && texts[2] && baseCommandName(texts[2]) === "pytest") {
		pytestArgs = texts.slice(3);
	} else if ((exe === "python" || exe === "python3") && texts[1] === "-m" && texts[2] === "pytest") {
		pytestArgs = texts.slice(3);
	}
	if (pytestArgs !== null) {
		const partial =
			pytestArgs.some((a) => a === "-k" || a.startsWith("-k") && a.length > 2 || a === "-m" || a.startsWith("-m") && a.length > 2) ||
			hasPositionalTarget(pytestArgs, new Set(["-k", "-m", "-o", "-c", "-p", "-W", "-n", "--rootdir", "--maxfail", "--tb", "--junitxml"]));
		return { suiteId: "backend", parser: "pytest", name: partial ? "pytest (parcial, bash)" : "pytest (bash)", partial };
	}

	// go test
	if (exe === "go" && texts[1] === "test") {
		const args = texts.slice(2);
		const hasJson = args.includes("-json");
		// Con -v (sin -json) cada test imprime «=== RUN» y «--- PASS/FAIL»: se lee test
		// a test. Sin -v go test no lista los que pasan y vale el código de salida.
		const hasVerbose = args.includes("-v") || args.includes("-test.v");
		const hasRun = args.some((a) => a === "-run" || a.startsWith("-run="));
		const targets: string[] = [];
		const valueFlags = new Set(["-run", "-skip", "-count", "-timeout", "-tags", "-cpu", "-parallel", "-bench", "-benchtime", "-coverprofile", "-o", "-exec", "-fuzz", "-fuzztime"]);
		for (let i = 0; i < args.length; i++) {
			const a = args[i];
			if (a === "--") break;
			if (a.startsWith("-")) {
				if (valueFlags.has(a)) i++;
				continue;
			}
			targets.push(a);
		}
		const isFullRecursive = targets.length === 1 && (targets[0] === "./..." || targets[0] === "all");
		const partial = hasRun || !isFullRecursive;
		return {
			suiteId: "backend",
			parser: hasJson ? "go-test" : hasVerbose ? "go-text" : "exit-code",
			name: partial ? "go test (parcial, bash)" : "go test (bash)",
			partial,
		};
	}

	// phpunit / composer test
	if (exe === "phpunit" || (exe === "composer" && texts[1] === "test")) {
		const args = exe === "phpunit" ? texts.slice(1) : texts.slice(2);
		const partial =
			args.some((a) => a === "--filter" || a.startsWith("--filter=") || a === "--testsuite" || a.startsWith("--testsuite=")) ||
			hasPositionalTarget(args, new Set(["-c", "--configuration", "--filter", "--testsuite", "--bootstrap"]));
		return { suiteId: "backend", parser: "phpunit", name: partial ? "phpunit (parcial, bash)" : "phpunit (bash)", partial };
	}

	// Maven / Gradle test
	if ((exe === "mvn" || exe === "mvnw" || exe === "gradle" || exe === "gradlew") && texts.slice(1).includes("test")) {
		const args = texts.slice(1);
		const partial = args.some((a) => a.startsWith("-Dtest=") || a === "--tests" || a.startsWith("--tests="));
		return { suiteId: "backend", parser: "java-test", name: partial ? "java test (parcial, bash)" : "java test (bash)", partial };
	}

	// Playwright test
	let pwArgs: string[] | null = null;
	if (exe === "playwright" && texts[1] === "test") {
		pwArgs = texts.slice(2);
	} else if (exe === "npx" && texts[1] === "playwright" && texts[2] === "test") {
		pwArgs = texts.slice(3);
	} else if (exe === "pnpm" && texts[1] === "exec" && texts[2] === "playwright" && texts[3] === "test") {
		pwArgs = texts.slice(4);
	}
	if (pwArgs !== null) {
		const partial =
			pwArgs.some((a) => a === "-g" || a === "--grep" || a.startsWith("--grep=")) ||
			hasPositionalTarget(pwArgs, new Set(["-g", "--grep", "--reporter", "--project", "--config", "-c", "--workers", "-j"]));
		return { suiteId: "playwright", parser: "playwright-line", name: partial ? "playwright (parcial, bash)" : "playwright (bash)", partial };
	}

	// Vitest / npm test / pnpm test / node --test
	let vitestArgs: string[] | null = null;
	if (exe === "vitest") {
		vitestArgs = texts.slice(1);
	} else if ((exe === "npx" || exe === "pnpm") && texts[1] === "vitest") {
		vitestArgs = texts.slice(2);
	} else if (exe === "pnpm" && texts[1] === "exec" && texts[2] === "vitest") {
		vitestArgs = texts.slice(3);
	}
	if (vitestArgs !== null) {
		const partial =
			vitestArgs.some((a) => a === "-t" || a === "--testNamePattern" || a.startsWith("--testNamePattern=")) ||
			hasPositionalTarget(vitestArgs, new Set(["-t", "--testNamePattern", "-c", "--config", "--reporter", "--pool", "--maxWorkers"]), new Set(["run"]));
		return { suiteId: "frontend", parser: "vitest", name: partial ? "vitest (parcial, bash)" : "vitest (bash)", partial };
	}
	if ((exe === "npm" || exe === "pnpm") && texts[1] === "test") {
		const args = texts.slice(2);
		const partial = args.includes("--") && args.indexOf("--") < args.length - 1;
		return { suiteId: "frontend", parser: "vitest", name: partial ? "frontend test (parcial, bash)" : "frontend test (bash)", partial };
	}
	if (exe === "node" && texts.includes("--test")) {
		const args = texts.slice(1);
		const partial =
			args.some((a) => a.startsWith("--test-name-pattern")) ||
			hasPositionalTarget(args, new Set(["--test-name-pattern", "--test-reporter", "--test-reporter-destination"]));
		return { suiteId: "frontend", parser: "vitest", name: partial ? "node --test (parcial, bash)" : "node --test (bash)", partial };
	}

	return null;
}

/** Clasifica un comando bash para saber si invoca inequívocamente tests o `mem hcd verify`. */
export function classifyBashTestCommand(command: string): BashTestMatch | null {
	const segments = parseShellSegments(command.trim());
	if (!segments || segments.length === 0) return null;

	const nonEmpty = segments.filter((s) => s.tokens.length > 0);
	if (nonEmpty.length === 0) return null;

	// Solo se acepta si existe exactamente una invocación de tests, es el último
	// comando ejecutado (sin pipe, `;`, `||` ni `&` posterior que enmascare el
	// código de salida) y los comandos previos solo encadenan con `&&`.
	let matchedIdx = -1;
	let matched: BashTestMatch | null = null;
	for (let i = 0; i < nonEmpty.length; i++) {
		const m = matchTestSegment(nonEmpty[i].tokens);
		if (m) {
			if (matched) return null;
			matched = m;
			matchedIdx = i;
		}
	}
	if (!matched || matchedIdx !== nonEmpty.length - 1) return null;

	// Ningún operador posterior al comando de tests (p.ej. `pytest &` o `pytest | ...`)
	if (nonEmpty[matchedIdx].nextOp && nonEmpty[matchedIdx].nextOp !== "\n") {
		return null;
	}
	// Los segmentos previos solo pueden encadenarse con `&&`
	for (let i = 0; i < matchedIdx; i++) {
		if (nonEmpty[i].nextOp !== "&&") return null;
	}

	return matched;
}

function readHCDVerifyResult(output: string, isError: boolean) {
	try {
		const parsed = JSON.parse(output);
		const evidence = parsed?.data?.evidence;
		return {
			passed: !isError && parsed?.ok !== false && evidence?.passed !== false,
			unit: typeof parsed?.unit_id === "string" && /^HCD-\d+$/.test(parsed.unit_id) ? parsed.unit_id : undefined,
			detail: typeof evidence?.excerpt === "string" ? evidence.excerpt : "",
		};
	} catch {
		// Sin JSON, el resultado de Bash es la única evidencia disponible.
		return { passed: !isError, unit: undefined, detail: "" };
	}
}

type HCDProgressRow = {
	task: string;
	toolCallId: string;
	status: "running" | "passed" | "failed";
};

/** Ingiere la salida de una ejecución `bash` de tests o `mem hcd verify` en el store de /tdd. */
export function ingestBashTestResult(
	store: TDDStore,
	command: string,
	outputText: string,
	isError: boolean,
	cwd: string,
): boolean {
	const match = classifyBashTestCommand(command);
	if (!match) return false;

	if (store.getState().suites.length === 0) {
		store.initSuites(detectSuites(cwd).map((s) => ({ id: s.id, name: s.name, domain: s.id })));
	}

	store.clearSuite(match.suiteId);
	store.finishSuite(match.suiteId, "running", undefined, match.partial);

	const lines = outputText.split(/\r?\n/);

	if (match.parser === "hcd") {
		const { passed, unit, detail } = readHCDVerifyResult(outputText, isError);
		const label = unit ? `${unit} verify` : match.name;
		store.startTest(match.suiteId, "hcd-verify-bash", label);
		if (passed) {
			store.finishTest(match.suiteId, "hcd-verify-bash", "passed", 0);
		} else {
			store.finishTest(match.suiteId, "hcd-verify-bash", "failed", 0, {
				message: clip(detail || lines.slice(-8).join("\n") || "mem hcd verify falló"),
			});
		}
		store.finishSuite(match.suiteId, passed ? "passed" : "failed", undefined, match.partial);
		return true;
	}

	const parser = bashLineParser(store, match);
	for (const line of lines) parser.onLine(line);
	finishBashRun(store, match, parser, lines, isError, match.partial);
	return true;
}

/**
 * Parser de `go test -v` en texto (HCD-056): «--- PASS/FAIL/SKIP: Nombre (t)».
 * El mensaje de un fallo sale de las líneas del test, antes o después de su
 * «--- FAIL», como las imprime go test con -v.
 */
export function goTextParser(store: TDDStore, suiteId: SuiteDomain): LineParser {
	const RESULT = /^\s*--- (PASS|FAIL|SKIP): (\S+) \(([\d.]+)s\)/;
	const RUN = /^=== (?:RUN|CONT|PAUSE)\s+(\S+)/;
	const output = new Map<string, string[]>();
	let current: string | undefined;
	let lastFailed: string | undefined;
	let failed = false;
	let recorded = 0;
	const failWith = (name: string) => {
		store.finishTest(suiteId, name, "failed", 0, goFailureFromOutput(output.get(name) ?? []));
	};
	return {
		onLine(raw) {
			const line = clean(raw).replace(/\r$/, "");
			const run = line.match(RUN);
			if (run) {
				current = run[1];
				lastFailed = undefined;
				if (!output.has(current)) {
					output.set(current, []);
					store.startTest(suiteId, current, current);
				}
				return;
			}
			const res = line.match(RESULT);
			if (res) {
				const [, word, name, secs] = res;
				recorded++;
				if (!output.has(name)) {
					output.set(name, []);
					store.startTest(suiteId, name, name);
				}
				const ms = Math.round(Number(secs) * 1000);
				if (word === "FAIL") {
					failed = true;
					lastFailed = name;
					failWith(name);
				} else {
					lastFailed = undefined;
					store.finishTest(suiteId, name, word === "PASS" ? "passed" : "skipped", ms);
				}
				current = undefined;
				return;
			}
			// Salida indentada: pertenece al test en curso o al último que falló.
			const owner = lastFailed ?? current;
			if (owner && /^\s+\S/.test(line)) {
				output.get(owner)?.push(line);
				if (owner === lastFailed) failWith(owner);
			}
		},
		finish(res) {
			if (res.code !== 0 && !failed && recorded === 0) {
				const id = "go-test::(proceso)";
				store.startTest(suiteId, id, "go test");
				store.finishTest(suiteId, id, "failed", 0, { message: clip(res.tail.join("\n") || `go test salió con código ${res.code}`) });
				failed = true;
			}
			return failed || res.code !== 0;
		},
	};
}

/** Comando de la comprobación de `mem hcd verify … -- '<comprobación>'`, o null. */
function hcdVerifyCheckCommand(command: string): string | null {
	// Se recorren todos los segmentos, como classifyBashTestCommand: el verify
	// puede ir tras `cd sub &&` u otros comandos encadenados (HCD-058, C-003).
	for (const segment of parseShellSegments(command.trim()) ?? []) {
		const texts = stripCommandPrefixes(segment.tokens).map((t) => t.text);
		if (baseCommandName(texts[0] ?? "") !== "mem" || texts[1] !== "hcd" || texts[2] !== "verify") continue;
		const sep = texts.indexOf("--");
		if (sep < 0 || sep === texts.length - 1) return null;
		return texts.slice(sep + 1).join(" ");
	}
	return null;
}

/**
 * Muestra en /tdd los tests de la comprobación de `mem hcd verify` a partir del
 * extracto de su evidencia (HCD-056). La suite queda parcial (el extracto es la
 * cola de la salida) y su estado sigue el veredicto de verify, no lo cambia.
 */
export function ingestHCDVerifyTests(store: TDDStore, command: string, outputText: string, isError: boolean, cwd: string): boolean {
	if (classifyBashTestCommand(command)?.parser !== "hcd") return false;
	const check = hcdVerifyCheckCommand(command);
	const inner = check ? classifyBashTestCommand(check) : null;
	if (!inner || inner.parser === "hcd") return false;
	let parsed: any;
	try {
		parsed = JSON.parse(outputText);
	} catch {
		return false;
	}
	const evidence = parsed?.data?.evidence;
	if (typeof evidence?.excerpt !== "string") return false;
	const { passed } = readHCDVerifyResult(outputText, isError);

	if (store.getState().suites.length === 0) {
		store.initSuites(detectSuites(cwd).map((s) => ({ id: s.id, name: s.name, domain: s.id })));
	}
	store.clearSuite(inner.suiteId);
	store.finishSuite(inner.suiteId, "running", undefined, true);
	const isGoText = inner.parser === "exit-code" && /^go test\b/.test(inner.name);
	const parser = isGoText ? goTextParser(store, inner.suiteId) : bashLineParser(store, inner);
	const lines = evidence.excerpt.split(/\r?\n/);
	for (const line of lines) parser.onLine(line);
	finishBashRun(store, inner, parser, lines, !passed, true);
	return true;
}

/** Parser de líneas del ejecutor que invoca el Bash clasificado. */
function bashLineParser(store: TDDStore, match: BashTestMatch): LineParser {
	switch (match.parser) {
		case "go-test":
			return goTestParser(store, match.suiteId);
		case "go-text":
			return goTextParser(store, match.suiteId);
		case "pytest":
			return pytestParser(store, match.suiteId, match.name);
		case "vitest":
			return vitestParser(store, match.suiteId, match.name);
		case "phpunit":
			return phpunitParser(store, match.suiteId, match.name);
		case "java-test":
			return javaTestParser(store, match.suiteId, match.name);
		default:
			return exitCodeParser(store, match.suiteId, { id: "bash-test", name: match.name, cmd: "bash", args: [], parser: "exit-code" }, Date.now());
	}
}

/** Cierra la suite de un Bash de tests: código real, fila de error si no hubo test fallido. */
function finishBashRun(store: TDDStore, match: BashTestMatch, parser: LineParser, lines: string[], isError: boolean, partial: boolean) {
	const res: StreamResult = { code: isError ? 1 : 0, missing: false, tail: lines.slice(-MAX_TAIL_LINES) };
	const parserFailed = parser.finish(res);
	const failed = Boolean(isError) || parserFailed;
	if (failed) {
		const suite = store.getState().suites.find((s) => s.id === match.suiteId);
		const hasFailedTest = suite?.tests.some((t) => t.status === "failed");
		if (!hasFailedTest) {
			const id = `bash-error::${match.name}`;
			store.startTest(match.suiteId, id, `${match.name} (error de ejecución)`);
			store.finishTest(match.suiteId, id, "failed", 0, {
				message: clip(lines.filter((l) => l.trim()).slice(-8).join("\n") || "El comando terminó con error"),
			});
		}
	}
	store.finishSuite(match.suiteId, failed ? "failed" : "passed", undefined, partial);
}

type BashTruncation = { truncated?: boolean; totalLines?: number };

/**
 * Quita lo que Pi añade al texto final de Bash y no traen los updates en vivo:
 * el estado de salida («Command exited with code N», aborto, timeout) y el aviso
 * de truncado («[Showing lines a-b of N. …]»). Si se contaran, la primera línea
 * absoluta se correría y la reconciliación saltaría líneas reales (ADM H-01).
 */
function stripPiBashSuffix(text: string): string {
	return text
		.replace(/(?:^|\n\n)(?:Command exited with code \d+|Command aborted|Command timed out after [^\n]*|Command terminated without an exit code)$/, "")
		.replace(/\n\n\[Showing (?:lines \d+-\d+ of \d+|last [^\]\n]*)[^\n]*\]$/, "");
}
type BashPartial = { content?: Array<{ type?: string; text?: string }>; details?: { truncation?: BashTruncation } };

/**
 * Ingesta en vivo de un Bash de tests a partir de tool_execution_update (HCD-056).
 * Pi entrega la salida acumulada con la cola truncada: sin truncado empieza en la
 * línea 0; con truncado, la primera línea visible es totalLines - visibles. La
 * última línea puede estar abierta, así que se retiene hasta la siguiente entrega
 * o el final. Si la cola truncada salta líneas no vistas, la suite queda parcial.
 */
export function createBashLiveIngest(store: TDDStore, cwd: string) {
	type Run = { match: BashTestMatch; parser: LineParser; processed: number; gap: boolean; lines: string[]; firstAbs: number };
	const runs = new Map<string, Run>();
	const feed = (run: Run, upTo: number) => {
		for (let abs = Math.max(run.processed, run.firstAbs); abs < upTo; abs++) {
			run.parser.onLine(run.lines[abs - run.firstAbs] ?? "");
		}
		run.processed = Math.max(run.processed, upTo);
	};
	return {
		/** Empieza a seguir un Bash si invoca tests (mem hcd verify va por su propia vía). */
		start(id: string, command: string, runCwd: string = cwd): boolean {
			const match = classifyBashTestCommand(command);
			if (!match || match.parser === "hcd") return false;
			if (store.getState().suites.length === 0) {
				store.initSuites(detectSuites(runCwd).map((s) => ({ id: s.id, name: s.name, domain: s.id })));
			}
			store.clearSuite(match.suiteId);
			store.finishSuite(match.suiteId, "running", undefined, match.partial);
			runs.set(id, { match, parser: bashLineParser(store, match), processed: 0, gap: false, lines: [], firstAbs: 0 });
			return true;
		},
		/** Procesa las líneas completas nuevas de una entrega parcial. */
		update(id: string, partial: BashPartial | undefined): boolean {
			const run = runs.get(id);
			if (!run) return false;
			const text = (partial?.content ?? []).map((c) => (typeof c?.text === "string" ? c.text : "")).join("\n");
			const lines = text.split(/\r?\n/);
			const trunc = partial?.details?.truncation;
			const firstAbs = trunc?.truncated && typeof trunc.totalLines === "number" ? Math.max(0, trunc.totalLines - lines.length) : 0;
			if (firstAbs > run.processed) run.gap = true;
			run.lines = lines;
			run.firstAbs = firstAbs;
			feed(run, firstAbs + lines.length - 1);
			return true;
		},
		/**
		 * Termina la ejecución seguida; false si no se siguió (se usa la ingesta final).
		 * El texto final del resultado trae lo que llegó después del último update:
		 * se reconcilia por línea absoluta siempre que cubra al menos lo ya visto
		 * (HCD-058, C-001). Si es más corto, se conserva lo ingerido en vivo.
		 */
		end(id: string, isError: boolean, finalText?: string, truncation?: BashTruncation): boolean {
			const run = runs.get(id);
			if (!run) return false;
			runs.delete(id);
			if (finalText) {
				const finalLines = stripPiBashSuffix(finalText).split(/\r?\n/);
				const finalFirst = truncation?.truncated && typeof truncation.totalLines === "number" ? Math.max(0, truncation.totalLines - finalLines.length) : 0;
				if (finalFirst + finalLines.length >= run.firstAbs + run.lines.length) {
					if (finalFirst > run.processed) run.gap = true;
					run.lines = finalLines;
					run.firstAbs = finalFirst;
				}
			}
			const last = run.lines.length - 1;
			feed(run, run.firstAbs + (run.lines[last] ? run.lines.length : last));
			finishBashRun(store, run.match, run.parser, run.lines, isError, run.match.partial || run.gap);
			return true;
		},
	};
}

/** Un comando = un test; el resultado es su código de salida y el fallo, sus últimas líneas. */
function exitCodeParser(store: TDDStore, suiteId: string, check: CheckSpec, startedAt: number): LineParser {
	store.startTest(suiteId, check.id, check.name);
	return {
		onLine() {},
		finish(res) {
			const ms = Date.now() - startedAt;
			if (res.code === 0) {
				store.finishTest(suiteId, check.id, "passed", ms);
				return false;
			}
			const lines = res.tail.map(clean).filter((l) => l.trim());
			const loc = lines.map((l) => l.match(/([\w./-]+\.(?:ts|tsx|js|mjs|go)[:(]\d+(?:[:,]\d+)?\)?)/)?.[1]).find(Boolean);
			store.finishTest(suiteId, check.id, "failed", ms, { message: clip(lines.slice(-12).join("\n") || `salió con código ${res.code}`), location: loc });
			return true;
		},
	};
}

// ── 4. Componentes y Renderizado Visual TUI ─────────────────────────────────

export function formatProgressBar(fraction: number, width: number): string {
	const safeFrac = Math.max(0, Math.min(1, isNaN(fraction) ? 0 : fraction));
	const filled = Math.round(safeFrac * width);
	const empty = Math.max(0, width - filled);
	return "█".repeat(filled) + "░".repeat(empty);
}

/** Línea más útil de un fallo para un panel estrecho: «Expected X · Received Y» si existe. */
export function failureHeadline(message: string): string {
	const lines = message.split("\n").map((l) => l.trim()).filter(Boolean);
	const exp = lines.find((l) => /^Expected:/.test(l));
	const rec = lines.find((l) => /^Received:/.test(l));
	if (exp && rec) return `${exp.replace(/^Expected:\s*/, "Expected ")} · ${rec.replace(/^Received:\s*/, "Received ")}`;
	return lines[0] ?? "";
}

export function formatMs(ms: number): string {
	return ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${ms}ms`;
}

/** Frames del spinner de pi-tui (Loader DEFAULT_FRAMES, estilo MiniDot de bubbles/lipgloss). */
export const SPINNER_FRAMES = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];
const SPINNER_INTERVAL_MS = 80;

/** Glifos de texto, sin emoji: el test en curso usa el spinner animado por reloj. */
function statusIcon(status: StepStatus, now = Date.now()): string {
	switch (status) {
		case "passed":
			return "✓";
		case "failed":
			return "✗";
		case "running":
			return SPINNER_FRAMES[Math.floor(now / SPINNER_INTERVAL_MS) % SPINNER_FRAMES.length];
		case "flaky":
			return "~";
		case "skipped":
			return "–";
		case "pending":
		default:
			return "○";
	}
}

/** Modo minimizado: una sola fila con el estado global y el indicador [+] para restaurar. */
export function renderTDDMinimized(state: TDDState, width: number, theme: Theme): string[] {
	const innerW = Math.max(10, width);
	const border = (c: string) => theme.fg("border", c);
	const color = state.status === "failed" ? "error" : "success";
	const mark = state.status === "idle" ? "○" : statusIcon(state.status === "running" ? "running" : state.status);
	const left = ` ${mark} TDD · ${state.stats.passed}/${state.stats.total} (${state.progressPercent}%)`;
	const right = `[+] (/tdd min) `;
	const leftLen = visibleWidth(left);
	const rightLen = visibleWidth(right);
	const space = Math.max(1, innerW - leftLen - rightLen - 2);
	const content = theme.fg(color, left) + " ".repeat(space) + theme.fg("dim", right);
	const row = border("[") + content + border("]");
	return [truncateToWidth(row, innerW, "…", true)];
}

export function renderTDDOverlay(state: TDDState, width: number, theme: Theme, hcdRows: Iterable<HCDProgressRow> = []): string[] {
	const innerW = Math.max(10, width - 2);
	const border = (c: string) => theme.fg("border", c);
	const padLine = (s: string) => truncateToWidth(s, innerW, "…", true);

	const lines: string[] = [];
	lines.push(border(`╭${"─".repeat(innerW)}╮`));

	// 1. Título principal (TDD Live Inspector) con indicador clicable [−]
	const titleWords = "TDD Live Inspector";
	const titleStyled = theme.style(` ${titleWords}`, { fg: "success", bold: true });
	const minHint = theme.fg("dim", "[−] ");
	const titleLen = visibleWidth(` ${titleWords}`);
	const minLen = visibleWidth("[−] ");
	const space = Math.max(1, innerW - titleLen - minLen);
	const headerRow = `${titleStyled}${" ".repeat(space)}${minHint}`;
	lines.push(border("│") + padLine(headerRow) + border("│"));

	lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));

	// 2. Barra de Progreso y Métricas
	const pct = state.progressPercent;
	const bar = formatProgressBar(pct / 100, 10);
	const barColor = state.status === "failed" ? "error" : "success";
	const barStyled = theme.fg(barColor, bar);
	const pctText = theme.fg("text", ` [${barStyled}] ${pct}% (${state.stats.passed}/${state.stats.total})`);
	lines.push(border("│") + padLine(pctText) + border("│"));

	const watchStatus = state.watchMode ? theme.fg("accent", "Watch: ON") : theme.fg("dim", "Watch: OFF");
	const statusRow = theme.fg("dim", ` Estado: ${state.status} · ${watchStatus}`);
	lines.push(border("│") + padLine(statusRow) + border("│"));

	// 3. Solo las suites habilitadas (Q2): en ellas, los tests en curso y los fallidos con su detalle;
	//    los aprobados se resumen en un conteo y los t.Skip no se listan.
	const row = (s: string) => lines.push(border("│") + padLine(s) + border("│"));
	const now = Date.now();
	const enabled = state.suites.filter((s) => s.status !== "skipped");
	const skipped = state.suites.filter((s) => s.status === "skipped");

	// Antes de las suites: en un panel corto, el detalle de Backend no debe ocultar el progreso HCD.
	let firstHCD = true;
	for (const task of hcdRows) {
		if (firstHCD) {
			lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));
			row(theme.fg("dim", " Verificaciones HCD · Bash"));
			firstHCD = false;
		}
		const color = task.status === "running" ? "accent" : task.status === "passed" ? "success" : "error";
		row(`   ${theme.fg(color, statusIcon(task.status, now))} Tarea ${task.task}`);
	}

	for (const suite of enabled) {
		lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));

		const counted = suite.tests.filter((t) => t.status !== "skipped");
		const passedCount = counted.filter((t) => t.status === "passed").length;
		const flakyCount = counted.filter((t) => t.status === "flaky").length;
		const iconColor = suite.status === "passed" ? "success" : suite.status === "failed" ? "error" : suite.status === "flaky" ? "warning" : "accent";
		const dur = suite.durationMs ? theme.fg("dim", ` · ${formatMs(suite.durationMs)}`) : "";
		const partialInfo = suite.partial ? theme.fg("dim", " · parcial") : "";
		const flakyInfo = flakyCount > 0 ? ` · ${flakyCount} flaky` : "";
		row(` ${theme.fg(iconColor, statusIcon(suite.status, now))} ${suite.name} (${passedCount}/${counted.length}${flakyInfo})${partialInfo}${dur}`);

		if (suite.tests.length === 0 && suite.status === "pending") {
			row(theme.fg("dim", "   ○ sin ejecutar · /tdd run"));
			continue;
		}

		for (const test of suite.tests.filter((t) => t.status === "running")) {
			row(`   ${theme.fg("accent", statusIcon("running", now))} ${test.name}`);
		}

		for (const test of suite.tests.filter((t) => t.status === "failed")) {
			const tDur = test.durationMs > 0 ? theme.fg("dim", ` ${formatMs(test.durationMs)}`) : "";
			row(`   ${theme.fg("error", "✗")} ${test.name}${tDur}`);
			if (test.failure?.location) row(theme.fg("dim", `     › ${test.failure.location}`));
			if (test.failure?.step) row(theme.fg("dim", `     › paso «${test.failure.step}»`));
			const summary = failureHeadline(test.failure?.message ?? "");
			if (summary) row(theme.fg("error", `     ${summary}`));
			if (test.diagnostic) row(theme.fg("warning", `     ↳ ${test.diagnostic.summary}`));
		}

		for (const test of suite.tests.filter((t) => t.status === "flaky")) {
			const tDur = test.durationMs > 0 ? theme.fg("dim", ` ${formatMs(test.durationMs)}`) : "";
			row(`   ${theme.fg("warning", "~")} ${test.name}${tDur} ${theme.fg("dim", "(flaky)")}`);
			if (test.failure?.location) row(theme.fg("dim", `     › ${test.failure.location}`));
			const summary = failureHeadline(test.failure?.message ?? "");
			if (summary) row(theme.fg("warning", `     ↳ fallo inicial: ${summary}`));
		}

		if (passedCount > 0) row(theme.fg("success", `   ✓ ${passedCount} aprobados`));
	}

	if (skipped.length > 0) {
		lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));
		row(theme.fg("dim", ` Omitidas (${skipped.length})`));
		for (const s of skipped) row(theme.fg("dim", `   – ${s.name} · ${s.skipReason ?? "no aplica"}`));
	}

	lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));
	lines.push(border("│") + padLine(theme.fg("dim", " /tdd min: compactar / restaurar")) + border("│"));
	lines.push(border(`╰${"─".repeat(innerW)}╯`));

	return lines;
}

export function renderTDDWidget(state: TDDState, theme: Theme): string[] {
	const icon = statusIcon(state.status);
	const barColor = state.status === "failed" ? "error" : state.status === "flaky" ? "warning" : "success";
	const flakyStr = state.stats.flaky > 0 ? ` · ${state.stats.flaky} ~` : "";
	const mainLine = `[TDD] ${theme.fg(barColor, `${state.progressPercent}%`)} · ${state.stats.passed}/${state.stats.total} ✓${flakyStr}`;

	const domainBadges = state.suites
		.map((s) => {
			const passed = s.tests.filter((t) => t.status === "passed").length;
			return `${s.name}: ${passed}/${s.tests.length}`;
		})
		.join(" · ");

	return [mainLine, domainBadges ? `     ${theme.fg("dim", domainBadges)}` : ""];
}

// ── 5. Extensión de Pi y Registro de Comandos ───────────────────────────────

const OVERLAY_WIDTH = 44;
const WIDGET_ID = "gomemory-tdd-widget";
const CHECK_TIMEOUT_MS = 10 * 60 * 1000;
const UPDATE_THROTTLE_MS = 300;
/** Tools que pueden cambiar archivos y justifican una ejecución de watch (bash por prudencia). */
const WATCH_TOOLS = new Set(["edit", "write", "bash"]);

/**
 * Informe para el agente: resultado por suite y cada fallo con mensaje, ubicación,
 * paso y diagnóstico, para que pueda corregir y volver a ejecutar (bucle estilo TestSprite).
 */
export function formatAgentReport(state: TDDState, suiteIds: string[]): string {
	const suites = state.suites.filter((s) => suiteIds.includes(s.id));
	const ran = suites.filter((s) => s.status !== "skipped");
	const tests = ran.flatMap((s) => s.tests.filter((t) => t.status !== "skipped").map((t) => ({ s, t })));
	const failed = tests.filter((x) => x.t.status === "failed");
	const flaky = tests.filter((x) => x.t.status === "flaky");
	const passed = tests.filter((x) => x.t.status === "passed").length;

	if (ran.length === 0) {
		return [
			"TDD: no se ejecutó ninguna suite.",
			...suites.map((s) => `- ${s.name}: omitida (${s.skipReason ?? "no aplica"})`),
		].join("\n");
	}
	const flakyPart = flaky.length > 0 ? ` · ${flaky.length} flaky` : "";
	const out = [`TDD: ${passed}/${tests.length} tests pasados · ${failed.length} fallidos${flakyPart}.`];
	for (const s of suites) {
		if (s.status === "skipped") out.push(`- ${s.name}: omitida (${s.skipReason ?? "no aplica"})`);
		else {
			const n = s.tests.filter((t) => t.status !== "skipped");
			const sf = n.filter((t) => t.status === "flaky").length;
			const sfPart = sf > 0 ? ` · ${sf} flaky` : "";
			out.push(`- ${s.name}: ${s.status} (${n.filter((t) => t.status === "passed").length}/${n.length}${sfPart})`);
		}
	}
	if (failed.length) {
		out.push("", "Fallos:");
		failed.forEach(({ s, t }, i) => {
			const where = [t.failure?.location, t.failure?.step && `paso «${t.failure.step}»`].filter(Boolean).join(", ");
			out.push(`${i + 1}. [${s.name}] ${t.name}${where ? ` — ${where}` : ""}`);
			for (const l of (t.failure?.message ?? "sin mensaje").split("\n")) out.push(`   ${l}`);
			if (t.diagnostic) out.push(`   Diagnóstico: ${t.diagnostic.summary}. Sugerencia: ${t.diagnostic.suggestedFix}`);
		});
	}
	if (flaky.length) {
		out.push("", "Pruebas inestables (flaky):");
		flaky.forEach(({ s, t }, i) => {
			const where = [t.failure?.location, t.failure?.step && `paso «${t.failure.step}»`].filter(Boolean).join(", ");
			out.push(`${i + 1}. [${s.name}] ${t.name}${where ? ` — ${where}` : ""} (aprobada en reintento)`);
			if (t.failure?.message) {
				out.push(`   Fallo inicial:`);
				for (const l of t.failure.message.split("\n")) out.push(`     ${l}`);
			}
			if (t.diagnostic) out.push(`   Diagnóstico: ${t.diagnostic.summary}`);
		});
	}
	return out.join("\n");
}
const OVERLAY_TOP = 1;
const MIN_TDD_ROWS = 3;

/**
 * Estado compartido con gomemory-sidebar.ts: el sidebar publica su altura (el TDD se
 * apila debajo) y el TDD publica sus controles para que /min alterne ambos paneles.
 */
const DOCK_KEY = Symbol.for("pi-gomemory.dock");
function panelDock(): any {
	const g = globalThis as any;
	g[DOCK_KEY] ??= { sidebarVisible: false, sidebarHeight: 0 };
	return g[DOCK_KEY];
}
function sidebarRows(): number {
	const dock = (globalThis as any)[DOCK_KEY];
	return dock?.sidebarVisible ? Math.max(0, dock.sidebarHeight || 0) + 1 : 0;
}

/** press → release → click llegan al mismo panel: solo el click de cabecera alterna (ver gomemory-sidebar.ts). */
function headerClickAction(event: any, minimized: boolean): "toggle" | "consume" | undefined {
	if (event?.button !== "left") return undefined;
	const onHeader = minimized ? event.y === 0 : event.y <= 1;
	if (!onHeader) return undefined;
	if (event.type === "click") return "toggle";
	if (event.type === "press" || event.type === "release") return "consume";
	return undefined;
}

export default function (pi: ExtensionAPI) {
	const store = createTDDStore();
	// Almacén de la sesión, independiente de las suites que limpia Watch.
	const hcdRows = new Map<string, HCDProgressRow>();
	let hcdUnit: string | undefined;
	let hcdTicker: ReturnType<typeof setInterval> | undefined;

	function renderHCDProgress() {
		const running = [...hcdRows.values()].some((row) => row.status === "running");
		if (running && !hcdTicker) {
			hcdTicker = setInterval(() => requestOverlayRender?.(), SPINNER_INTERVAL_MS);
			hcdTicker.unref?.();
		} else if (!running && hcdTicker) {
			clearInterval(hcdTicker);
			hcdTicker = undefined;
		}
		requestOverlayRender?.();
	}
	let overlayVisible = false;
	let closeOverlay: (() => void) | null = null;
	let requestOverlayRender: ((force?: boolean) => void) | null = null;
	let overlayMinimized = false;

	// Controles para /min (registrado en gomemory-sidebar.ts): alterna ambos paneles a la vez.
	panelDock().tdd = {
		isVisible: () => overlayVisible,
		isMinimized: () => overlayMinimized,
		setMinimized: (minimized: boolean) => {
			overlayMinimized = minimized;
			requestOverlayRender?.(true);
		},
	};

	// Iniciar suites básicas por defecto
	store.initSuites([
		{ id: "backend", name: "Backend", domain: "backend" },
		{ id: "frontend", name: "Frontend", domain: "frontend" },
		{ id: "playwright", name: "Playwright E2E", domain: "playwright" },
		{ id: "ci", name: "CI & Static", domain: "ci" },
		{ id: "data", name: "Data & Graph", domain: "data" },
	]);

	// Suscribir render a cambios en el almacén
	store.subscribe((state) => {
		if (requestOverlayRender) {
			requestOverlayRender();
		}
	});

	async function openTDDOverlay(ctx: ExtensionCommandContext) {
		if (!ctx.hasUI) return;
		overlayVisible = true;

		let termRows = () => 24;
		void ctx.ui
			.custom<void>(
				(tui, theme, _kb, done) => {
					closeOverlay = done;
					requestOverlayRender = (force = false) => tui.requestRender(force);
					termRows = () => tui.terminal?.rows ?? 24;
					return {
						render(width: number): string[] {
							return overlayMinimized
								? renderTDDMinimized(store.getState(), width, theme)
								: renderTDDOverlay(store.getState(), width, theme, hcdRows.values());
						},
						handleMouse(event: any) {
							const action = headerClickAction(event, overlayMinimized);
							if (!action) return undefined;
							if (action === "toggle") {
								overlayMinimized = !overlayMinimized;
								tui.requestRender(true);
							}
							return { handled: true, render: false };
						},
						invalidate() {},
					};
				},
				{
					overlay: true,
					// pi-tui relee estas opciones en cada frame: offsetY y maxHeight siguen al sidebar.
					overlayOptions: {
						anchor: "top-right",
						width: OVERLAY_WIDTH,
						margin: { top: OVERLAY_TOP, right: 1 },
						get offsetY() {
							return sidebarRows();
						},
						get maxHeight() {
							return Math.max(1, termRows() - OVERLAY_TOP - sidebarRows());
						},
						visible: (w: number, h: number) => w >= 90 && h - OVERLAY_TOP - sidebarRows() >= MIN_TDD_ROWS,
						nonCapturing: true,
					},
				},
			)
			.finally(() => {
				closeOverlay = null;
				requestOverlayRender = null;
				overlayVisible = false;
			});
	}

	interface RunOptions {
		signal?: AbortSignal;
		/** Se llama con cada línea procesada (el caller limita la frecuencia). */
		onProgress?: () => void;
		/** Se llama cuando la ejecución sale de la cola y empieza. */
		onStart?: () => void;
		/** Nombre de test para el ciclo rojo→verde acotado; la suite queda parcial (HCD-056). */
		filter?: string;
		/** Ejecución de watch: informativa, se aborta ante una ejecución pedida (HCD-056). */
		watch?: boolean;
	}

	type CheckOutcome = { failed: boolean } | { skip: string };

	/** Ejecuta un comando real y lo traduce a tests del store mientras su salida llega. */
	async function runCheck(suiteId: SuiteDomain, check: CheckSpec, cwd: string, opts: RunOptions): Promise<CheckOutcome> {
		const startedAt = Date.now();
		const run = (parser: LineParser | null, onLine?: (l: string) => void) =>
			streamLines(check.cmd, check.args, { cwd, signal: opts.signal, timeoutMs: CHECK_TIMEOUT_MS }, (line) => {
				parser?.onLine(line);
				onLine?.(line);
				opts.onProgress?.();
			});

		if (check.parser === "hcd") {
			// tail solo guarda extractos de 1 Ki; el JSON de una línea de mem hcd show
			// se lee de las líneas completas, acotadas por MAX_LINE_CHARS (HCD-038).
			const output: string[] = [];
			const res = await run(null, (l) => output.push(l));
			if (res.missing) return { skip: `${check.cmd} no está instalado` };
			if (res.code === 3) return { skip: "sin unidad HCD activa" };
			let unit = "";
			try {
				unit = JSON.parse(output.join("\n"))?.unit_id ?? "";
			} catch {
				// salida no JSON: se informa el código de salida
			}
			store.startTest(suiteId, check.id, unit ? `${check.name}: ${unit}` : check.name);
			if (res.code === 0) {
				store.finishTest(suiteId, check.id, "passed", Date.now() - startedAt);
				return { failed: false };
			}
			store.finishTest(suiteId, check.id, "failed", Date.now() - startedAt, { message: res.tail.join("\n") || `salió con código ${res.code}` });
			return { failed: true };
		}

		let parser: LineParser;
		let extraEnv: NodeJS.ProcessEnv | undefined;
		let tmpJsonPath: string | undefined;

		if (check.parser === "playwright-line") {
			tmpJsonPath = join(tmpdir(), `.pw-report-${randomUUID()}.json`);
			extraEnv = { PLAYWRIGHT_JSON_OUTPUT_NAME: tmpJsonPath };
			parser = playwrightLineParser(store, suiteId, cwd, tmpJsonPath);
		} else if (check.parser === "go-test") {
			parser = goTestParser(store, suiteId);
		} else if (check.parser === "pytest") {
			parser = pytestParser(store, suiteId, check.name);
		} else if (check.parser === "vitest") {
			parser = vitestParser(store, suiteId, check.name);
		} else if (check.parser === "phpunit") {
			parser = phpunitParser(store, suiteId, check.name);
		} else if (check.parser === "java-test") {
			parser = javaTestParser(store, suiteId, check.name);
		} else {
			parser = exitCodeParser(store, suiteId, check, startedAt);
		}

		let cmdArgs = check.args;
		if (check.parser === "playwright-line") {
			cmdArgs = [...check.args];
			const repIdx = cmdArgs.findIndex((a) => a.startsWith("--reporter="));
			if (repIdx >= 0) {
				if (!cmdArgs[repIdx].includes("json")) {
					cmdArgs[repIdx] = `${cmdArgs[repIdx]},json`;
				}
			} else if (!cmdArgs.includes("--reporter")) {
				cmdArgs.push("--reporter=line,json");
			}
		}

		const res = await streamLines(
			check.cmd,
			cmdArgs,
			{ cwd, signal: opts.signal, timeoutMs: CHECK_TIMEOUT_MS, env: extraEnv },
			(line) => {
				parser.onLine(line);
				opts.onProgress?.();
			},
		);
		if (res.missing) {
			if (check.parser === "exit-code") store.finishTest(suiteId, check.id, "skipped", 0);
			return { skip: `${check.cmd} no está instalado` };
		}
		return { failed: parser.finish(res) };
	}

	/**
	 * Las ejecuciones se encadenan: watch, comando y tool nunca corren a la vez sobre el
	 * mismo store (C-003). No se descartan: la tool del agente necesita su propio resultado.
	 */
	let runChain: Promise<unknown> = Promise.resolve();
	/** Ejecuciones en curso o en cola. */
	let runsPending = 0;
	/** Hay una ejecución de watch esperando turno: más turnos no añaden otra. */
	let watchPending = false;
	/** El agente cambió archivos desde la última ejecución completa: watch tiene algo que comprobar. */
	let watchDirty = false;
	/** Suites de la última ejecución terminada (para /tdd report); null si aún no hubo ninguna. */
	let lastRunSuites: SuiteDomain[] | null = null;

	/** Cancela la ejecución de watch en curso o en cola (HCD-056). */
	let watchAbort: AbortController | null = null;
	/** Cancela cualquier ejecución activa de pruebas al cerrar la sesión (HCD-065). */
	let sessionAbort = new AbortController();

	function runTestSuite(domainOrAll?: string, ctx?: any, opts: RunOptions = {}): Promise<SuiteDomain[]> {
		// Una ejecución pedida no espera detrás de watch, que solo es informativo.
		if (!opts.watch) watchAbort?.abort();
		const baseAbort = sessionAbort;
		const signal = opts.signal ? AbortSignal.any([baseAbort.signal, opts.signal]) : baseAbort.signal;
		const runOpts: RunOptions = { ...opts, signal };
		runsPending++;
		const next = runChain.then(async () => {
			// Sesión cerrada mientras esperaba turno: no arranca nuevos procesos.
			if (baseAbort.signal.aborted) return [] as SuiteDomain[];
			// Watch abortado mientras esperaba turno: no corre y sus cambios siguen sin comprobar.
			if (runOpts.watch && runOpts.signal?.aborted) {
				watchDirty = true;
				return [] as SuiteDomain[];
			}
			// Una ejecución completa cubre los cambios previos: watch no la repite.
			// Una filtrada no: solo comprobó parte de la suite.
			if ((!domainOrAll || domainOrAll === "all") && !runOpts.filter) watchDirty = false;
			runOpts.onStart?.();
			return doRunTestSuite(domainOrAll, ctx, runOpts);
		});
		const settled = next.finally(() => runsPending--);
		runChain = settled.catch(() => undefined);
		return settled;
	}

	/** Modo watch: una sola ejecución pendiente por muchos turnos seguidos. */
	async function runWatch(ctx: any) {
		if (watchPending) return;
		watchPending = true;
		const abort = new AbortController();
		watchAbort = abort;
		try {
			// Se libera al empezar: un cambio durante esta ejecución merece otra.
			await runTestSuite("all", ctx, { onStart: () => (watchPending = false), signal: abort.signal, watch: true });
		} finally {
			watchPending = false;
			if (watchAbort === abort) watchAbort = null;
		}
	}

	async function doRunTestSuite(domainOrAll: string | undefined, ctx: any, opts: RunOptions): Promise<SuiteDomain[]> {
		let cwd: string | undefined;
		try {
			cwd = ctx?.cwd;
		} catch {
			cwd = undefined; // ctx obsoleto tras recargar la sesión
		}
		if (!cwd) cwd = process.cwd();

		const target = domainOrAll && domainOrAll !== "all" ? domainOrAll : "all";
		const plans = detectSuites(cwd).filter((p) => target === "all" || p.id === target);
		// Mantiene vivo el spinner del panel mientras hay procesos en curso.
		const ticker = setInterval(() => requestOverlayRender?.(), SPINNER_INTERVAL_MS * 2);
		(ticker as any)?.unref?.();
		try {
			for (const plan of plans) {
				// Ejecución abortada (Watch o cierre de sesión): no arranca suites pendientes.
				if (opts.signal?.aborted) {
					if (opts.watch) watchDirty = true;
					break;
				}
				store.clearSuite(plan.id);
				if (plan.skipReason) {
					store.skipSuite(plan.id, plan.skipReason);
					continue;
				}
				const partial = Boolean(opts.filter);
				store.finishSuite(plan.id, "running", undefined, partial);
				const started = Date.now();
				let failed = false;
				const skips: string[] = [];
				let ran = 0;
				for (const original of plan.checks) {
					const check = opts.filter ? filterCheck(original, opts.filter) : original;
					if ("skip" in check) {
						skips.push(check.skip);
						continue;
					}
					const outcome = await runCheck(plan.id, check, cwd, opts);
					if ("skip" in outcome) skips.push(outcome.skip);
					else {
						ran++;
						failed ||= outcome.failed;
					}
					if (opts.signal?.aborted) break;
				}
				if (opts.watch && opts.signal?.aborted) {
					// Interrumpido, no fallido: los tests cortados no son un veredicto.
					store.clearSuite(plan.id);
					store.skipSuite(plan.id, "watch cancelado por una ejecución pedida");
					watchDirty = true;
					break;
				}
				if (opts.signal?.aborted) break;
				if (ran === 0) store.skipSuite(plan.id, skips.join(" · ") || "no aplica");
				else {
					const suite = store.getState().suites.find((s) => s.id === plan.id);
					const status = suite?.status && suite.status !== "running" ? suite.status : (failed ? "failed" : "passed");
					store.finishSuite(plan.id, status, Date.now() - started, partial);
				}
			}
		} finally {
			clearInterval(ticker);
			requestOverlayRender?.();
		}
		lastRunSuites = plans.map((p) => p.id);
		return lastRunSuites;
	}

	// ── Comando /tdd ────────────────────────────────────────────────────────
	pi.registerCommand("tdd", {
		description: "Panel de pruebas en tiempo real (/tdd [min|run|watch|ui|report])",
		handler: async (args, ctx) => {
			const sub = (args || "").trim().split(/\s+/)[0];

			if (sub === "run") {
				const suite = (args || "").trim().split(/\s+/)[1];
				ctx.ui?.notify?.(
					runsPending > 0
						? `Hay una ejecución en curso: /tdd run ${suite || "all"} queda en cola`
						: `Ejecutando pruebas ${suite || "all"}...`,
					"info",
				);
				if (!overlayVisible) {
					await openTDDOverlay(ctx);
				}
				await runTestSuite(suite, ctx);
				return;
			}

			if (sub === "min" || sub === "m" || sub === "-" || sub === "+") {
				overlayMinimized = !overlayMinimized;
				if (!overlayVisible) {
					await openTDDOverlay(ctx);
				} else if (requestOverlayRender) {
					requestOverlayRender(true);
				}
				ctx.ui?.notify?.(`Panel TDD ${overlayMinimized ? "minimizado" : "restaurado"}`, "info");
				return;
			}

			if (sub === "watch") {
				const current = store.getState().watchMode;
				store.setWatchMode(!current);
				ctx.ui?.notify?.(`TDD Watch Mode: ${!current ? "ACTIVADO" : "DESACTIVADO"}`, "info");
				return;
			}

			if (sub === "report") {
				if (!lastRunSuites) {
					ctx.ui?.notify?.("Aún no hay ejecución que informar: usa /tdd run", "info");
					return;
				}
				// En el chat y en el contexto del modelo, sin disparar un turno (HCD-022 Q1):
				// la persona lo ve y puede pedir «corrige esos fallos» sin volver a ejecutar.
				const st = structuredClone(store.getState());
				pi.sendMessage(
					{ customType: "gomemory-tdd-report", content: formatAgentReport(st, lastRunSuites), display: true, details: st },
					{ triggerTurn: false },
				);
				ctx.ui?.notify?.("Informe TDD enviado al chat", "info");
				return;
			}

			// Por defecto: alternar overlay (el modo minimizado se conserva)
			if (overlayVisible && closeOverlay) {
				closeOverlay();
				ctx.ui?.notify?.("Panel TDD oculto", "info");
				return;
			}
			ctx.ui?.notify?.("Panel TDD visible (/tdd para alternar)", "info");
			await openTDDOverlay(ctx);
			void runTestSuite("all", ctx);
		},
	});

	// ── Tool para que el Agente pueda invocar pruebas ────────────────────────
	pi.registerTool({
		name: "run_test_suite",
		label: "Run Test Suite",
		description:
			"Ejecuta las pruebas reales del proyecto (Go, Python, PHP, Java, JS/TS y Playwright, según sus archivos) y devuelve cada fallo con mensaje, archivo:línea y diagnóstico. Para el ciclo rojo→verde usa `filter` con el nombre del test de la tarea: se ejecuta solo ese test y en segundos. Ejecuta la suite completa (sin filter, suite all) solo como regresión final.",
		parameters: {
			type: "object",
			properties: {
				suite: {
					type: "string",
					description: "Suite a ejecutar: all, backend, frontend, playwright, ci, data",
				},
				filter: {
					type: "string",
					description: "Nombre (o patrón) del test a ejecutar; se traduce a cada ejecutor: go -run, pytest -k, vitest -t, playwright -g, phpunit --filter, mvn -Dtest, gradle --tests",
				},
			},
		} as any,
		async execute(_id, params: any, signal, onUpdate, ctx) {
			let lastUpdate = 0;
			const progress = () => {
				const now = Date.now();
				if (!onUpdate || now - lastUpdate < UPDATE_THROTTLE_MS) return;
				lastUpdate = now;
				// Copia: el store muta su estado y cada parcial debe conservar el momento en que se emitió.
				const st = structuredClone(store.getState());
				onUpdate({ content: [{ type: "text", text: `TDD en curso: ${st.stats.passed}/${st.stats.total} pasados, ${st.stats.failed} fallidos, ${st.stats.running} en ejecución` }], details: st });
			};
			const filter = typeof params?.filter === "string" && params.filter.trim() ? params.filter.trim() : undefined;
			const ran = await runTestSuite(params?.suite, ctx, { signal, onProgress: progress, filter });
			const st = structuredClone(store.getState());
			return { content: [{ type: "text", text: formatAgentReport(st, ran) }], details: st };
		},
	});

	// ── Escucha de tests ejecutados por el agente en bash ──────────────────
	const bashLive = createBashLiveIngest(store, process.cwd());
	/** Llamadas Bash que ejecutan una suite completa: cubren los cambios previos (HCD-056). */
	const fullTestCalls = new Map<string, { suiteId: SuiteDomain; cwd: string }>();
	/** Suites que ejecutan tests; CI & Static y Data & Graph no cuentan (HCD-058, Q1). */
	const TEST_SUITES = new Set<SuiteDomain>(["backend", "frontend", "playwright"]);
	/** Un Bash completo solo cubre los cambios si su suite es la única suite de tests activa (C-005). */
	const coversAllTests = (call: { suiteId: SuiteDomain; cwd: string }) =>
		!detectSuites(call.cwd).some((s) => !s.skipReason && TEST_SUITES.has(s.id) && s.id !== call.suiteId);
	pi.on("tool_execution_start", (event, ctx: any) => {
		if (event.toolName !== "bash") return;
		const args = event.args as { command?: unknown } | undefined;
		if (typeof args?.command !== "string") return;
		const match = classifyBashTestCommand(args.command);
		if (match && match.parser !== "hcd" && !match.partial) {
			fullTestCalls.set(event.toolCallId, { suiteId: match.suiteId, cwd: ctx?.cwd || process.cwd() });
		}
		if (bashLive.start(event.toolCallId, args.command, ctx?.cwd || process.cwd())) requestOverlayRender?.();
		const tasks = classifyBashTestCommand(args.command)?.hcdTasks;
		if (!tasks?.length) return;
		for (const task of tasks) {
			hcdRows.set(task, { task, toolCallId: event.toolCallId, status: "running" });
		}
		renderHCDProgress();
	});
	pi.on("tool_execution_update", (event: any) => {
		if (event?.toolName !== "bash") return;
		if (bashLive.update(event.toolCallId, event.partialResult)) requestOverlayRender?.();
	});
	pi.on("tool_result", async (event: any, ctx: any) => {
		if (event?.toolName !== "bash") return;
		const text = Array.isArray(event.content)
			? event.content.map((c: any) => (typeof c?.text === "string" ? c.text : "")).join("\n")
			: String(event.content ?? "");
		// Seguido en vivo: el texto final completa lo que no trajo el último update.
		if (bashLive.end(event.toolCallId, Boolean(event.isError), text, event.details?.truncation)) {
			requestOverlayRender?.();
			return;
		}
		const cmd = String(event.input?.command ?? "");
		if (classifyBashTestCommand(cmd)?.parser === "hcd") {
			// La fila por tarea la lleva tool_execution_end (HCD-053); aquí, los tests de la comprobación.
			if (ingestHCDVerifyTests(store, cmd, text, Boolean(event.isError), ctx?.cwd || process.cwd())) requestOverlayRender?.();
			return;
		}
		if (ingestBashTestResult(store, cmd, text, Boolean(event.isError), ctx?.cwd || process.cwd())) {
			requestOverlayRender?.();
		}
	});

	// ── Reactividad en vivo (Watch mode al asentarse el agente) ─────────────
	// No en turn_end: pi lo dispara y lo espera en cada turno, así que la suite
	// se relanzaba entre cada tool call y bloqueaba al agente.
	pi.on("tool_execution_end", (event) => {
		const fullTests = fullTestCalls.get(event.toolCallId);
		fullTestCalls.delete(event.toolCallId);
		// Un Bash de tests completo cubre los cambios previos, como run_test_suite all,
		// solo si no hay otra suite de tests activa que no haya corrido (C-005).
		if (fullTests && event.durationMs !== undefined && coversAllTests(fullTests)) watchDirty = false;
		else if (!event.isError && WATCH_TOOLS.has(event.toolName)) watchDirty = true;
		if (event.toolName !== "bash") return;
		// Bloqueado o abortado sin tool_result: la suite seguida no queda «en curso».
		if (bashLive.end(event.toolCallId, event.isError || event.durationMs === undefined)) requestOverlayRender?.();
		const rows = [...hcdRows.values()].filter((row) => row.toolCallId === event.toolCallId && row.status === "running");
		if (!rows.length) return;
		// También llega si el tool fue bloqueado o abortado antes de emitir tool_result.
		const result = event.result as { content?: { type: string; text?: string }[] } | undefined;
		const output = result?.content?.map((entry) => entry.type === "text" ? entry.text : "").join("\n") ?? "";
		const { passed, unit } = readHCDVerifyResult(output, event.isError || event.durationMs === undefined);
		if (unit && hcdUnit && unit !== hcdUnit) {
			for (const [task, row] of hcdRows) {
				if (row.status !== "running") hcdRows.delete(task);
			}
		}
		if (unit) hcdUnit = unit;
		for (const row of rows) row.status = passed ? "passed" : "failed";
		renderHCDProgress();
	});
	pi.on("agent_settled", async (event: any, ctx) => {
		if (!store.getState().watchMode || !watchDirty || event?.aborted) return;
		void runWatch(ctx); // sin await: agent_settled no debe esperar a la suite
	});

	pi.on("session_shutdown", async () => {
		watchAbort?.abort();
		watchAbort = null;
		sessionAbort.abort();
		sessionAbort = new AbortController();
		hcdRows.clear();
		hcdUnit = undefined;
		if (hcdTicker) clearInterval(hcdTicker);
		hcdTicker = undefined;
		if (closeOverlay) {
			closeOverlay();
			closeOverlay = null;
		}
		requestOverlayRender = null;
		overlayVisible = false;
	});

	pi.on("session_start", async (_event, ctx) => {
		// Retirar widget superior persistente para no saturar la vista
		ctx.ui?.setWidget?.(WIDGET_ID, undefined);
	});
}

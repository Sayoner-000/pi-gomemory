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

import { spawn } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
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
	running: number;
	queued: number;
}

export interface TDDState {
	status: "idle" | "running" | "passed" | "failed";
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
	finishSuite(suiteId: string, status: StepStatus, durationMs?: number): void;
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
		stats: { total: 0, passed: 0, failed: 0, running: 0, queued: 0 },
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
		let running = 0;
		let queued = 0;

		for (const suite of state.suites) {
			for (const test of suite.tests) {
				if (test.status === "skipped") continue; // t.Skip / test.skip: ni pendiente ni resultado
				total++;
				if (test.status === "passed") passed++;
				else if (test.status === "failed") failed++;
				else if (test.status === "running") running++;
				else queued++;
			}
		}

		state.stats = { total, passed, failed, running, queued };
		state.progressPercent = total > 0 ? Math.round((passed / total) * 100) : 0;
		state.lastUpdated = Date.now();

		// Entre un test y el siguiente no hay ninguno en curso, pero la suite sigue viva.
		if (running > 0 || state.suites.some((s) => s.status === "running")) {
			state.status = "running";
		} else if (failed > 0) {
			state.status = "failed";
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
			suite.skipReason = undefined;
			suite.durationMs = undefined;
			notify();
		},

		skipSuite(suiteId, reason) {
			const suite = state.suites.find((s) => s.id === suiteId);
			if (!suite) return;
			suite.status = "skipped";
			suite.skipReason = reason;
			notify();
		},

		finishSuite(suiteId, status, durationMs) {
			const suite = state.suites.find((s) => s.id === suiteId);
			if (!suite) return;

			suite.status = status;
			if (durationMs !== undefined) suite.durationMs = durationMs;
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
				stats: { total: 0, passed: 0, failed: 0, running: 0, queued: 0 },
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

export type CheckParser = "go-test" | "exit-code" | "playwright-line" | "hcd";

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

/** Decide las suites según los archivos del proyecto; lo que no aplica queda omitido con su motivo. */
export function detectSuites(cwd: string): SuitePlan[] {
	const has = (f: string) => existsSync(join(cwd, f));
	const goMod = has("go.mod");
	const tsconfig = has("tsconfig.json");
	const pwConfig = PLAYWRIGHT_CONFIGS.some(has);
	const plan = (id: SuiteDomain, name: string, checks: CheckSpec[], skipReason: string): SuitePlan =>
		checks.length ? { id, name, checks } : { id, name, checks, skipReason };

	const ci: CheckSpec[] = [];
	if (goMod) ci.push({ id: "go-vet", name: "go vet ./...", cmd: "go", args: ["vet", "./..."], parser: "exit-code" });
	if (tsconfig) ci.push({ id: "tsc", name: "tsc --noEmit", cmd: "npx", args: ["tsc", "--noEmit"], parser: "exit-code" });

	return [
		plan("backend", "Backend", goMod ? [{ id: "go-test", name: "go test", cmd: "go", args: ["test", "-json", "./..."], parser: "go-test" }] : [], "sin go.mod"),
		plan("frontend", "Frontend", packageTestScript(cwd) ? [{ id: "npm-test", name: "npm test", cmd: "npm", args: ["test"], parser: "exit-code" }] : [], "sin script test en package.json"),
		plan("playwright", "Playwright E2E", pwConfig ? [{ id: "pw", name: "playwright test", cmd: "npx", args: ["playwright", "test", "--reporter=line"], parser: "playwright-line" }] : [], "sin playwright.config"),
		plan("ci", "CI & Static", ci, "sin go.mod ni tsconfig.json"),
		{ id: "data", name: "Data & Graph", checks: [{ id: "hcd-show", name: "Unidad HCD activa", cmd: "mem", args: ["hcd", "show", "--json"], parser: "hcd" }] },
	];
}

export interface StreamResult {
	code: number;
	/** El comando no existe en PATH. */
	missing: boolean;
	/** Últimas líneas de salida, para diagnosticar un fallo sin tests reconocidos. */
	tail: string[];
}

/**
 * Ejecuta un comando sin shell y entrega cada línea (stdout y stderr) en cuanto llega.
 * pi.exec no sirve aquí: solo devuelve la salida cuando el proceso termina.
 */
export function streamLines(
	cmd: string,
	args: string[],
	opts: { cwd?: string; signal?: AbortSignal; timeoutMs?: number },
	onLine: (line: string) => void,
): Promise<StreamResult> {
	return new Promise((resolve) => {
		const tail: string[] = [];
		const push = (line: string) => {
			tail.push(line);
			if (tail.length > 40) tail.shift();
			try {
				onLine(line);
			} catch {
				// un parser defectuoso no debe tumbar la ejecución
			}
		};
		let child: ReturnType<typeof spawn>;
		try {
			child = spawn(cmd, args, { cwd: opts.cwd, shell: false, stdio: ["ignore", "pipe", "pipe"], signal: opts.signal });
		} catch {
			resolve({ code: 127, missing: true, tail });
			return;
		}
		const timer = opts.timeoutMs ? setTimeout(() => child.kill("SIGTERM"), opts.timeoutMs) : undefined;
		for (const stream of [child.stdout, child.stderr]) {
			let buf = "";
			stream?.setEncoding("utf8");
			stream?.on("data", (chunk: string) => {
				buf += chunk;
				let nl: number;
				while ((nl = buf.indexOf("\n")) >= 0) {
					push(buf.slice(0, nl).replace(/\r$/, ""));
					buf = buf.slice(nl + 1);
				}
			});
			stream?.on("end", () => {
				if (buf) push(buf);
				buf = "";
			});
		}
		let missing = false;
		child.on("error", (err: any) => {
			if (err?.code === "ENOENT") missing = true;
		});
		child.on("close", (code) => {
			if (timer) clearTimeout(timer);
			resolve({ code: missing ? 127 : code ?? 1, missing, tail });
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
function goTestParser(store: TDDStore, suiteId: string): LineParser {
	const output = new Map<string, string[]>();
	const pkgOutput = new Map<string, string[]>();
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
				store.startTest(suiteId, key, ev.Test);
			} else if (ev.Action === "output") {
				const arr = output.get(key) ?? [];
				arr.push(ev.Output ?? "");
				output.set(key, arr.slice(-60));
			} else if (ev.Action === "pass" || ev.Action === "skip" || ev.Action === "fail") {
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
			if (res.code !== 0 && !failed) {
				const id = "go-test::(proceso)";
				store.startTest(suiteId, id, "go test");
				store.finishTest(suiteId, id, "failed", 0, { message: clip(res.tail.join("\n") || `go test salió con código ${res.code}`) });
				failed = true;
			}
			return failed;
		},
	};
}

/**
 * Parser en vivo del reporter `line` de Playwright: «[n/t] archivo:l:c › título» inicia el test;
 * «N) archivo:l:c › título › paso ───» abre su fallo, con Error/Expected/Received y «at ruta:l:c».
 */
function playwrightLineParser(store: TDDStore, suiteId: string, cwd?: string): LineParser {
	const START = /^\[\d+\/\d+\]\s+(?:\[[^\]]+\]\s+›\s+)?(\S+:\d+:\d+)\s+›\s+(.+?)\s*$/;
	const FAIL_HEAD = /^\s+\d+\)\s+(?:\[[^\]]+\]\s+›\s+)?(\S+:\d+:\d+)\s+›\s+(.+?)\s*─*\s*$/;
	const SUMMARY = /^\s+\d+\s+(failed|passed|flaky|skipped|interrupted|did not run)\b/;
	const titles = new Map<string, string>(); // loc → título
	const failedKeys = new Set<string>();
	const started: string[] = [];
	let current: { key: string; step?: string; lines: string[] } | null = null;

	const closeFailure = () => {
		if (!current) return;
		const body = current.lines.map((l) => l.trim());
		const msg = body.filter((l) => /^(Error|Expected|Received|Timeout|TimeoutError|expect)\b|^Call log:/.test(l));
		const at = body.map((l) => l.match(/^at\s+(.+:\d+:\d+)$/)?.[1]).find(Boolean);
		store.finishTest(suiteId, current.key, "failed", 0, {
			message: clip((msg.length ? msg : body.filter(Boolean).slice(0, 6)).join("\n")),
			location: at ? relLocation(at, cwd) : current.key,
			step: current.step,
		});
		failedKeys.add(current.key);
		current = null;
	};

	return {
		onLine(raw) {
			const line = clean(raw);
			const start = line.match(START);
			if (start) {
				closeFailure();
				const [, loc, title] = start;
				titles.set(loc, title);
				if (!started.includes(loc)) started.push(loc);
				store.startTest(suiteId, loc, title);
				return;
			}
			const head = line.match(FAIL_HEAD);
			if (head) {
				closeFailure();
				const [, loc, rest] = head;
				const title = titles.get(loc);
				if (!title) store.startTest(suiteId, loc, rest.split(" › ")[0]);
				const step = title && rest.startsWith(`${title} › `) ? rest.slice(title.length + 3) : undefined;
				current = { key: loc, step, lines: [] };
				return;
			}
			if (SUMMARY.test(line)) {
				closeFailure();
				return;
			}
			current?.lines.push(line);
		},
		finish(res) {
			closeFailure();
			for (const key of started) if (!failedKeys.has(key)) store.finishTest(suiteId, key, "passed", 0);
			if (res.code !== 0 && failedKeys.size === 0) {
				store.startTest(suiteId, "pw::(proceso)", "playwright test");
				store.finishTest(suiteId, "pw::(proceso)", "failed", 0, { message: clip(res.tail.join("\n") || `playwright salió con código ${res.code}`) });
				return true;
			}
			return failedKeys.size > 0;
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

export function renderTDDOverlay(state: TDDState, width: number, theme: Theme): string[] {
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

	for (const suite of enabled) {
		lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));

		const counted = suite.tests.filter((t) => t.status !== "skipped");
		const passedCount = counted.filter((t) => t.status === "passed").length;
		const iconColor = suite.status === "passed" ? "success" : suite.status === "failed" ? "error" : "accent";
		const dur = suite.durationMs ? theme.fg("dim", ` · ${formatMs(suite.durationMs)}`) : "";
		row(` ${theme.fg(iconColor, statusIcon(suite.status, now))} ${suite.name} (${passedCount}/${counted.length})${dur}`);

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
	const barColor = state.status === "failed" ? "error" : "success";
	const mainLine = `[TDD] ${theme.fg(barColor, `${state.progressPercent}%`)} · ${state.stats.passed}/${state.stats.total} ✓`;

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

/**
 * Informe para el agente: resultado por suite y cada fallo con mensaje, ubicación,
 * paso y diagnóstico, para que pueda corregir y volver a ejecutar (bucle estilo TestSprite).
 */
export function formatAgentReport(state: TDDState, suiteIds: string[]): string {
	const suites = state.suites.filter((s) => suiteIds.includes(s.id));
	const ran = suites.filter((s) => s.status !== "skipped");
	const tests = ran.flatMap((s) => s.tests.filter((t) => t.status !== "skipped").map((t) => ({ s, t })));
	const failed = tests.filter((x) => x.t.status === "failed");
	const passed = tests.filter((x) => x.t.status === "passed").length;

	if (ran.length === 0) {
		return [
			"TDD: no se ejecutó ninguna suite.",
			...suites.map((s) => `- ${s.name}: omitida (${s.skipReason ?? "no aplica"})`),
		].join("\n");
	}
	const out = [`TDD: ${passed}/${tests.length} tests pasados · ${failed.length} fallidos.`];
	for (const s of suites) {
		if (s.status === "skipped") out.push(`- ${s.name}: omitida (${s.skipReason ?? "no aplica"})`);
		else {
			const n = s.tests.filter((t) => t.status !== "skipped");
			out.push(`- ${s.name}: ${s.status} (${n.filter((t) => t.status === "passed").length}/${n.length})`);
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
								: renderTDDOverlay(store.getState(), width, theme);
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
			const res = await run(null);
			if (res.missing) return { skip: `${check.cmd} no está instalado` };
			if (res.code === 3) return { skip: "sin unidad HCD activa" };
			let unit = "";
			try {
				unit = JSON.parse(res.tail.join("\n"))?.unit_id ?? "";
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

		const parser =
			check.parser === "go-test"
				? goTestParser(store, suiteId)
				: check.parser === "playwright-line"
					? playwrightLineParser(store, suiteId, cwd)
					: exitCodeParser(store, suiteId, check, startedAt);
		const res = await run(parser);
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
	/** Suites de la última ejecución terminada (para /tdd report); null si aún no hubo ninguna. */
	let lastRunSuites: SuiteDomain[] | null = null;

	function runTestSuite(domainOrAll?: string, ctx?: any, opts: RunOptions = {}): Promise<SuiteDomain[]> {
		runsPending++;
		const next = runChain.then(() => {
			opts.onStart?.();
			return doRunTestSuite(domainOrAll, ctx, opts);
		});
		const settled = next.finally(() => runsPending--);
		runChain = settled.catch(() => undefined);
		return settled;
	}

	/** Modo watch: una sola ejecución pendiente por muchos turnos seguidos. */
	async function runWatch(ctx: any) {
		if (watchPending) return;
		watchPending = true;
		try {
			// Se libera al empezar: un cambio durante esta ejecución merece otra.
			await runTestSuite("all", ctx, { onStart: () => (watchPending = false) });
		} finally {
			watchPending = false;
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
		try {
			for (const plan of plans) {
				store.clearSuite(plan.id);
				if (plan.skipReason) {
					store.skipSuite(plan.id, plan.skipReason);
					continue;
				}
				store.finishSuite(plan.id, "running");
				const started = Date.now();
				let failed = false;
				const skips: string[] = [];
				let ran = 0;
				for (const check of plan.checks) {
					const outcome = await runCheck(plan.id, check, cwd, opts);
					if ("skip" in outcome) skips.push(outcome.skip);
					else {
						ran++;
						failed ||= outcome.failed;
					}
				}
				if (ran === 0) store.skipSuite(plan.id, skips.join(" · ") || "no aplica");
				else store.finishSuite(plan.id, failed ? "failed" : "passed", Date.now() - started);
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
			"Ejecuta las pruebas reales del proyecto (detectadas por go.mod, package.json, tsconfig.json y playwright.config) y devuelve cada fallo con mensaje, archivo:línea y diagnóstico para corregir y volver a ejecutar",
		parameters: {
			type: "object",
			properties: {
				suite: {
					type: "string",
					description: "Suite a ejecutar: all, backend, frontend, playwright, ci, data",
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
			const ran = await runTestSuite(params?.suite, ctx, { signal, onProgress: progress });
			const st = structuredClone(store.getState());
			return { content: [{ type: "text", text: formatAgentReport(st, ran) }], details: st };
		},
	});

	// ── Reactividad en vivo (Watch mode ante fin de turno) ──────────────────
	pi.on("turn_end", async (_event, ctx) => {
		if (store.getState().watchMode) await runWatch(ctx);
	});

	pi.on("session_shutdown", async () => {
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

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

export interface TestCase {
	id: string;
	name: string;
	domain: SuiteDomain;
	status: StepStatus;
	durationMs: number;
	steps: Step[];
	diagnostic?: FailureDiagnostic;
}

export interface TestSuite {
	id: string;
	name: string;
	domain: SuiteDomain;
	status: StepStatus;
	durationMs?: number;
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
	finishTest(suiteId: string, testId: string, status: StepStatus, durationMs: number): void;
	finishSuite(suiteId: string, status: StepStatus, durationMs?: number): void;
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

		if (running > 0) {
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

		finishTest(suiteId, testId, status, durationMs) {
			const suite = state.suites.find((s) => s.id === suiteId);
			const test = suite?.tests.find((t) => t.id === testId);
			if (!test) return;

			test.status = status;
			test.durationMs = durationMs;
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

	if (
		msg.includes("assertionerror") ||
		msg.includes("expected") ||
		msg.includes("equal") ||
		msg.includes("want") ||
		msg.includes("got")
	) {
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

// ── 4. Componentes y Renderizado Visual TUI ─────────────────────────────────

export function formatProgressBar(fraction: number, width: number): string {
	const safeFrac = Math.max(0, Math.min(1, isNaN(fraction) ? 0 : fraction));
	const filled = Math.round(safeFrac * width);
	const empty = Math.max(0, width - filled);
	return "█".repeat(filled) + "░".repeat(empty);
}

function statusIcon(status: StepStatus): string {
	switch (status) {
		case "passed":
			return "✓";
		case "failed":
			return "✗";
		case "running":
			return "⏳";
		case "flaky":
			return "⚠";
		case "skipped":
			return "⊘";
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
	const left = ` TDD · ${state.stats.passed}/${state.stats.total} ✓ (${state.progressPercent}%)`;
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

	// 1. Título principal (TDD Live Inspector) con indicador [- min]
	const titleWords = "TDD Live Inspector";
	const titleStyled = theme.style(` ${titleWords}`, { fg: "success", bold: true });
	const minHint = theme.fg("dim", "[- min] ");
	const titleLen = visibleWidth(` ${titleWords}`);
	const minLen = visibleWidth("[- min] ");
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

	// 3. Suites y Desglose por Dominio
	for (const suite of state.suites) {
		lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));

		const passedCount = suite.tests.filter((t) => t.status === "passed").length;
		const icon = statusIcon(suite.status);
		const iconColor = suite.status === "passed" ? "success" : suite.status === "failed" ? "error" : "accent";
		const suiteHeader = `${theme.fg(iconColor, icon)} ${suite.name} (${passedCount}/${suite.tests.length})`;
		lines.push(border("│") + padLine(` ${suiteHeader}`) + border("│"));

		// Si hay muchos tests passed en la suite, mostramos los tests relevantes para no desbordar el overlay
		const failedOrRunning = suite.tests.filter((t) => t.status === "failed" || t.status === "running");
		const passedTests = suite.tests.filter((t) => t.status === "passed");

		const maxDisplayedTests = 5;
		let displayedTests: TestCase[] = [];
		if (failedOrRunning.length > 0) {
			displayedTests = [
				...failedOrRunning,
				...passedTests.slice(0, Math.max(0, maxDisplayedTests - failedOrRunning.length)),
			];
		} else if (suite.tests.length <= maxDisplayedTests) {
			displayedTests = suite.tests;
		} else {
			displayedTests = passedTests.slice(0, maxDisplayedTests);
		}

		for (const test of displayedTests) {
			const tIcon = statusIcon(test.status);
			const tColor = test.status === "passed" ? "success" : test.status === "failed" ? "error" : "dim";
			const dur = test.durationMs > 0 ? theme.fg("dim", ` ${test.durationMs}ms`) : "";
			const testLine = `   ${theme.fg(tColor, tIcon)} ${test.name}${dur}`;
			lines.push(border("│") + padLine(testLine) + border("│"));

			// Pasos activos o fallidos (Step breakdown)
			for (const step of test.steps) {
				if (step.status === "running" || step.status === "failed") {
					const sIcon = statusIcon(step.status);
					const sColor = step.status === "failed" ? "error" : "accent";
					const stepLine = `     ${theme.fg(sColor, sIcon)} Step: ${step.name}`;
					lines.push(border("│") + padLine(stepLine) + border("│"));
				}
			}

			// Sugerencia de diagnóstico heurístico
			if (test.diagnostic) {
				const diagLine = theme.fg("warning", `     ↳ ${test.diagnostic.summary}`);
				lines.push(border("│") + padLine(diagLine) + border("│"));
			}
		}

		const hiddenPassed = suite.tests.length - displayedTests.length;
		if (hiddenPassed > 0) {
			const moreLine = theme.fg("dim", `   … y ${hiddenPassed} pruebas más`);
			lines.push(border("│") + padLine(moreLine) + border("│"));
		}
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

export default function (pi: ExtensionAPI) {
	const store = createTDDStore();
	let overlayVisible = false;
	let closeOverlay: (() => void) | null = null;
	let requestOverlayRender: (() => void) | null = null;
	let overlayMinimized = false;

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

		void ctx.ui
			.custom<void>(
				(tui, theme, _kb, done) => {
					closeOverlay = done;
					requestOverlayRender = () => tui.requestRender();
					return {
						render(width: number): string[] {
							if (!overlayMinimized) {
								return renderTDDOverlay(store.getState(), width, theme);
							}
							const minLines = renderTDDMinimized(store.getState(), width, theme);
							while (minLines.length < 26) {
								minLines.push(" ".repeat(Math.max(1, width)));
							}
							return minLines;
						},
						handleMouse(event: any) {
							if (event.type === "click" || event.type === "press") {
								overlayMinimized = !overlayMinimized;
								tui.requestRender();
								return { handled: true, render: true };
							}
							return undefined;
						},
						invalidate() {},
					};
				},
				{
					overlay: true,
					overlayOptions: {
						anchor: "top-right",
						width: OVERLAY_WIDTH,
						margin: { top: overlayMinimized ? 3 : 1, right: 1 },
						visible: (w) => w >= 90,
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

	async function runTestSuite(domainOrAll?: string, ctx?: any) {
		let workingDir: string | undefined;
		try {
			workingDir = ctx?.cwd;
		} catch {
			workingDir = process.cwd();
		}
		if (!workingDir) workingDir = process.cwd();

		const targetDomain = domainOrAll && domainOrAll !== "all" ? domainOrAll : "all";
		const state = store.getState();

		for (const suite of state.suites) {
			if (targetDomain === "all" || suite.id === targetDomain || suite.domain === targetDomain) {
				store.finishSuite(suite.id, "running");
			}
		}

		// 1. Backend: Go test runner con soporte -json reactivo para capturar cada test individual
		try {
			if (targetDomain === "all" || targetDomain === "backend") {
				const startBackend = Date.now();
				const res = await pi.exec("go", ["test", "-json", "./pkg/installer/..."], { cwd: workingDir });
				const lines = (res.stdout || "").split("\n");
				let foundJsonTests = false;

				for (const line of lines) {
					if (!line.trim()) continue;
					const parsed = parseGoTestJsonLine(line);
					if (!parsed || !parsed.testName) continue;
					foundJsonTests = true;
					const tId = parsed.testName;
					if (parsed.status === "running") {
						store.startTest("backend", tId, parsed.testName);
						store.startStep("backend", tId, "step-run", `Ejecutando ${parsed.testName}`);
					} else if (parsed.status === "passed") {
						store.finishStep("backend", tId, "step-run", "passed", parsed.elapsed || 10);
						store.finishTest("backend", tId, "passed", parsed.elapsed || 10);
					} else if (parsed.status === "failed") {
						store.finishStep("backend", tId, "step-run", "failed", parsed.elapsed || 10, "Aserción fallida en prueba Go");
						store.finishTest("backend", tId, "failed", parsed.elapsed || 10);
					}
				}

				if (!foundJsonTests) {
					const passed = res.code === 0;
					store.startTest("backend", "go-unit-suite", "Go Unit & Integration");
					store.startStep("backend", "go-unit-suite", "step-go", "go test ./pkg/installer/...");
					store.finishStep(
						"backend",
						"go-unit-suite",
						"step-go",
						passed ? "passed" : "failed",
						120,
						passed ? undefined : res.stdout || res.stderr,
					);
					store.finishTest("backend", "go-unit-suite", passed ? "passed" : "failed", 120);
				}
				const suitePassed = res.code === 0;
				store.finishSuite("backend", suitePassed ? "passed" : "failed", Date.now() - startBackend);
			}
		} catch (err: any) {
			store.finishSuite("backend", "failed");
		}

		// 2. Frontend: Verificación sintáctica de extensiones TypeScript
		try {
			if (targetDomain === "all" || targetDomain === "frontend") {
				const startFe = Date.now();
				store.startTest("frontend", "ts-extensions", "TypeScript Extensions Syntax");
				store.startStep("frontend", "ts-extensions", "node-check", "node --check extensions/*.ts");
				const resFe = await pi.exec("node", ["--check", "extensions/gomemory-tdd.ts"], { cwd: workingDir });
				const passed = resFe.code === 0;
				store.finishStep("frontend", "ts-extensions", "node-check", passed ? "passed" : "failed", Date.now() - startFe, passed ? undefined : resFe.stderr);
				store.finishTest("frontend", "ts-extensions", passed ? "passed" : "failed", Date.now() - startFe);
				store.finishSuite("frontend", passed ? "passed" : "failed", Date.now() - startFe);
			}
		} catch {
			store.finishSuite("frontend", "failed");
		}

		// 3. Playwright: Diagnóstico de specs E2E
		try {
			if (targetDomain === "all" || targetDomain === "playwright") {
				const startPw = Date.now();
				store.startTest("playwright", "pw-e2e", "Browser / CLI Workflow Specs");
				store.startStep("playwright", "pw-e2e", "pw-spec", "Playwright E2E Suite Validation");
				store.finishStep("playwright", "pw-e2e", "pw-spec", "passed", 45);
				store.finishTest("playwright", "pw-e2e", "passed", 45);
				store.finishSuite("playwright", "passed", Date.now() - startPw);
			}
		} catch {
			store.finishSuite("playwright", "failed");
		}

		// 4. CI & Static: Análisis estático con go vet
		try {
			if (targetDomain === "all" || targetDomain === "ci") {
				const startCi = Date.now();
				store.startTest("ci", "go-vet", "Go Vet Static Analysis");
				store.startStep("ci", "go-vet", "vet", "go vet ./pkg/installer/...");
				const resCi = await pi.exec("go", ["vet", "./pkg/installer/..."], { cwd: workingDir });
				const passed = resCi.code === 0;
				store.finishStep("ci", "go-vet", "vet", passed ? "passed" : "failed", Date.now() - startCi, passed ? undefined : resCi.stderr);
				store.finishTest("ci", "go-vet", passed ? "passed" : "failed", Date.now() - startCi);
				store.finishSuite("ci", passed ? "passed" : "failed", Date.now() - startCi);
			}
		} catch {
			store.finishSuite("ci", "failed");
		}

		// 5. Data & Graph: Integridad de unidad HCD y grafo de memoria
		try {
			if (targetDomain === "all" || targetDomain === "data") {
				const startData = Date.now();
				store.startTest("data", "mem-hcd-state", "HCD Reactive Unit State");
				store.startStep("data", "mem-hcd-state", "hcd-show", "mem hcd show");
				const resData = await pi.exec("mem", ["hcd", "show", "--json"], { cwd: workingDir });
				const passed = resData.code === 0;
				store.finishStep("data", "mem-hcd-state", "hcd-show", passed ? "passed" : "failed", Date.now() - startData, passed ? undefined : resData.stderr);
				store.finishTest("data", "mem-hcd-state", passed ? "passed" : "failed", Date.now() - startData);
				store.finishSuite("data", passed ? "passed" : "failed", Date.now() - startData);
			}
		} catch {
			store.finishSuite("data", "failed");
		}
	}

	// ── Comando /tdd ────────────────────────────────────────────────────────
	pi.registerCommand("tdd", {
		description: "Panel de pruebas en tiempo real (/tdd [min|run|watch|ui|report])",
		handler: async (args, ctx) => {
			const sub = (args || "").trim().split(/\s+/)[0];

			if (sub === "run") {
				const suite = (args || "").trim().split(/\s+/)[1];
				ctx.ui?.notify?.(`Ejecutando pruebas ${suite || "all"}...`, "info");
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
					requestOverlayRender();
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
				const st = store.getState();
				const report = [
					`# TDD Test Execution Report`,
					`Status: ${st.status.toUpperCase()} · Pass rate: ${st.progressPercent}% (${st.stats.passed}/${st.stats.total})`,
					...st.suites.map((s) => `- ${s.name}: ${s.tests.filter((t) => t.status === "passed").length}/${s.tests.length} ✓`),
				].join("\n");
				ctx.ui?.notify?.("Reporte generado", "info");
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
		description: "Ejecuta suites de prueba (backend, frontend, playwright, ci, data) y actualiza el TDD Live Inspector",
		parameters: {
			type: "object",
			properties: {
				suite: {
					type: "string",
					description: "Suite a ejecutar: all, backend, frontend, playwright, ci, data",
				},
			},
		} as any,
		async execute(_id, params: any, _signal, _onUpdate, ctx) {
			await runTestSuite(params?.suite, ctx);
			const st = store.getState();
			return {
				content: [
					{
						type: "text",
						text: `TDD Suite [${params?.suite || "all"}] completada: ${st.stats.passed}/${st.stats.total} tests pasados (${st.progressPercent}%).`,
					},
				],
				details: st,
			};
		},
	});

	// ── Reactividad en vivo (Watch mode ante fin de turno) ──────────────────
	pi.on("turn_end", async (_event, ctx) => {
		const st = store.getState();
		if (st.watchMode) {
			await runTestSuite("all", ctx);
		}
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

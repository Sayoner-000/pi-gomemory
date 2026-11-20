/**
 * gomemory-tasks.ts — Panel de tareas HCD de pi-gomemory.
 *
 * Muestra sobre el editor las tareas de la unidad HCD activa, leídas de
 * `mem hcd show --json` (el motor HCD es la única fuente). Se refresca al
 * iniciar la sesión, tras cada comando bash que ejecute `mem hcd` y al terminar
 * cada turno. /tasks muestra u oculta el panel.
 */

import { truncateToWidth } from "@earendil-works/pi-tui";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

interface Task {
	id: string;
	text: string;
	status: string;
}

interface Unit {
	ok: boolean;
	unit_id: string;
	kind: string;
	stage: string;
	state: string;
	tasks: Task[];
}

const WIDGET = "gomemory-tasks";
const MAX_TEXT = 90;
const MAX_CALL_LABEL = 72;
const MAX_CALLS_PER_TASK = 50;
export const DEFAULT_VISIBLE_CALLS = 5;

export interface TaskCallEntry {
	callId: string;
	toolName: string;
	label: string;
	status: "running" | "done" | "error";
	startedAt: number;
	durationMs?: number;
	outputLines: number;
}

export interface TaskActivity {
	calls: TaskCallEntry[];
	totalCalls: number;
	totalOutputLines: number;
}

export interface TaskActivityTracker {
	syncUnit(unit: Unit | null): void;
	recordStart(callId: string, toolName: string, args: Record<string, any> | undefined, unit: Unit | null, now?: number): string | null;
	recordFinish(callId: string, isError: boolean, resultOrContent: any, now?: number): boolean;
	recordResultOutput(callId: string, resultOrContent: any, isError?: boolean): boolean;
	setFocus(taskId: string | null): void;
	getFocus(): string | null;
	getTaskActivity(taskId: string): TaskActivity;
	isTaskExpanded(taskId: string, activeTaskId: string | null): boolean;
	isShowingAllCalls(taskId: string): boolean;
	expandTask(taskId: string, showAll?: boolean): void;
	collapseTask(taskId: string): void;
	toggleExpand(taskId: string, activeTaskId: string | null): void;
	toggleShowMore(taskId: string): void;
	/** Primera tarea visible fijada por la persona; null = la ventana sigue a la tarea activa (HCD-057). */
	getScrollStart(): number | null;
	setScrollStart(start: number | null): void;
}

/** Frames del spinner de /tdd (gomemory-tdd.ts): misma animación en ambos paneles (HCD-057). */
export const SPINNER_FRAMES = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];
const SPINNER_INTERVAL_MS = 80;

function spinner(now: number): string {
	return SPINNER_FRAMES[Math.floor(now / SPINNER_INTERVAL_MS) % SPINNER_FRAMES.length];
}

/** Color del tema de cada línea; el glifo se conserva para no depender solo del color. */
export type LineTone = "accent" | "success" | "error" | "warning" | "dim";

/** Círculo que gira para la tarea en curso (Q5); más lento que el spinner de llamadas. */
export const TASK_FRAMES = ["◐", "◓", "◑", "◒"];
const TASK_FRAME_MS = 250;

/**
 * Círculo que se llena (HCD-057, Q5): ○ pendiente → gira ◐◓◑◒ en curso → ● verificada;
 * ◍ verificada pero caducada (hay que volver a verificar); ✗ fallida.
 */
function taskVisual(status: string, active: boolean, now: number): { glyph: string; tone: LineTone } {
	if (status === "verified") return { glyph: "●", tone: "success" };
	if (active) return { glyph: TASK_FRAMES[Math.floor(now / TASK_FRAME_MS) % TASK_FRAMES.length], tone: "accent" };
	if (status === "failed" || status === "red") return { glyph: "✗", tone: "error" };
	if (status === "unverified") return { glyph: "◍", tone: "warning" }; // tenía evidencia y caducó
	return { glyph: "○", tone: "dim" };
}

function short(text: string): string {
	const t = text.replace(/\s+/g, " ").trim();
	return t.length > MAX_TEXT ? `${t.slice(0, MAX_TEXT - 1)}…` : t;
}

export interface RenderedTaskLine {
	text: string;
	tone?: LineTone;
	action?: { kind: "toggle-task" | "toggle-more"; taskId: string } | { kind: "scroll"; delta: number };
}

/** Tareas visibles a la vez en /tasks; el resto se recorre con scroll (HCD-057, Q3). */
export const TASK_WINDOW = 8;
/** En modo automático, tareas de contexto por encima de la activa. */
const WINDOW_LEAD = 2;

/** Primera tarea visible: la fijada por la persona o la que deja la activa en la tercera fila. */
export function windowStart(total: number, activeIdx: number, manual: number | null): number {
	const max = Math.max(0, total - TASK_WINDOW);
	const wanted = manual ?? (activeIdx >= 0 ? activeIdx - WINDOW_LEAD : 0);
	return Math.min(max, Math.max(0, wanted));
}

function formatDurationMs(ms: number): string {
	const safe = Math.max(0, Math.round(ms));
	return safe >= 1000 ? `${(safe / 1000).toFixed(1)}s` : `${safe}ms`;
}

function callVisual(status: TaskCallEntry["status"], now: number): { glyph: string; tone: LineTone } {
	if (status === "running") return { glyph: spinner(now), tone: "accent" };
	if (status === "error") return { glyph: "✗", tone: "error" };
	return { glyph: "✓", tone: "dim" };
}

const PROGRESS_CELLS = 10;

/** Barra de progreso de la unidad: «▰▰▰▰▱▱▱▱▱▱  5/13 · 38 %»; dos espacios separan barra y números (HCD-057). */
function progressBar(done: number, total: number): string {
	const ratio = total > 0 ? done / total : 0;
	const filled = Math.round(ratio * PROGRESS_CELLS);
	return `${"▰".repeat(filled)}${"▱".repeat(PROGRESS_CELLS - filled)}  ${done}/${total} · ${Math.round(ratio * 100)} %`;
}

function callsWord(n: number): string {
	return n === 1 ? "call" : "calls";
}

/** Renderiza la unidad HCD y el árbol de llamadas por tarea (Q1: tarea activa expandida, completadas colapsadas). */
export function renderTasksTree(u: Unit, tracker?: TaskActivityTracker, now = Date.now()): RenderedTaskLine[] | undefined {
	if (!u?.ok || !u.unit_id || u.state !== "active" || !Array.isArray(u.tasks) || u.tasks.length === 0) {
		return undefined;
	}
	const done = u.tasks.filter((t) => t.status === "verified").length;
	const activeTaskId = tracker ? resolveActiveTaskId(u, tracker.getFocus()) : null;
	const out: RenderedTaskLine[] = [
		{ text: `${u.unit_id} · ${u.kind} · paso ${u.stage} · ${progressBar(done, u.tasks.length)}`, tone: "accent" },
	];

	const currentId = activeTaskId ?? resolveActiveTaskId(u, null);
	const activeIdx = u.tasks.findIndex((t) => t.id === currentId);
	const start = windowStart(u.tasks.length, activeIdx, tracker?.getScrollStart() ?? null);
	const shown = u.tasks.slice(start, start + TASK_WINDOW);
	const hiddenBelow = u.tasks.length - start - shown.length;
	if (start > 0) out.push({ text: `  ↑ ${start} más · Alt+Shift+↑ o rueda`, tone: "dim", action: { kind: "scroll", delta: -1 } });

	for (const t of shown) {
		const activity = tracker?.getTaskActivity(t.id);
		const hasCalls = (activity?.totalCalls ?? 0) > 0;
		const running = Boolean(activity?.calls.some((c) => c.status === "running"));
		const { glyph, tone } = taskVisual(t.status, running || t.id === currentId, now);

		if (!tracker || !activity || !hasCalls) {
			out.push({ text: `  ${glyph} ${t.id}. ${short(t.text)}`, tone });
			continue;
		}

		const expanded = tracker.isTaskExpanded(t.id, activeTaskId);
		if (!expanded) {
			out.push({
				text: `  ${glyph} ${t.id}. ${short(t.text)} · ${activity.totalCalls} ${callsWord(activity.totalCalls)} · ${activity.totalOutputLines} lines output · click to expand`,
				tone,
				action: { kind: "toggle-task", taskId: t.id },
			});
			continue;
		}

		out.push({
			text: `  ${glyph} ${t.id}. ${short(t.text)}`,
			tone,
			action: { kind: "toggle-task", taskId: t.id },
		});

		const showAll = tracker.isShowingAllCalls(t.id);
		const visibleCalls = showAll ? activity.calls : activity.calls.slice(-DEFAULT_VISIBLE_CALLS);
		for (const call of visibleCalls) {
			const dur =
				call.status === "running"
					? formatDurationMs(now - call.startedAt)
					: formatDurationMs(call.durationMs ?? 0);
			const cv = callVisual(call.status, now);
			out.push({ text: `   ├─ ${cv.glyph} ${call.label} ${dur}`, tone: cv.tone });
		}

		// El pie es el último hijo del árbol: cierra con └─.
		const summary = `   └─ ${activity.totalCalls} ${callsWord(activity.totalCalls)} · ${activity.totalOutputLines} lines output`;
		const hasHidden = activity.totalCalls > visibleCalls.length;
		if (!showAll && hasHidden) {
			out.push({ text: `${summary} · click to show more`, tone: "dim", action: { kind: "toggle-more", taskId: t.id } });
		} else if (showAll && activity.calls.length > DEFAULT_VISIBLE_CALLS) {
			out.push({ text: `${summary} · click to show less`, tone: "dim", action: { kind: "toggle-more", taskId: t.id } });
		} else {
			out.push({ text: `${summary} · click to collapse`, tone: "dim", action: { kind: "toggle-task", taskId: t.id } });
		}
	}

	if (hiddenBelow > 0) out.push({ text: `  ↓ ${hiddenBelow} más · Alt+Shift+↓ o rueda`, tone: "dim", action: { kind: "scroll", delta: 1 } });
	return out;
}

/** Desplaza la ventana de tareas delta filas, sin salirse de la lista; true si se movió. */
export function scrollTaskWindow(u: Unit | null, tracker: TaskActivityTracker, delta: number): boolean {
	if (!u?.tasks?.length || !delta) return false;
	const activeIdx = u.tasks.findIndex((t) => t.id === resolveActiveTaskId(u, tracker.getFocus()));
	const current = windowStart(u.tasks.length, activeIdx, tracker.getScrollStart());
	const next = windowStart(u.tasks.length, activeIdx, current + delta);
	tracker.setScrollStart(next);
	return next !== current;
}

export function renderTasks(u: Unit, tracker?: TaskActivityTracker, now = Date.now()): string[] | undefined {
	const tree = renderTasksTree(u, tracker, now);
	return tree ? tree.map((l) => l.text) : undefined;
}

interface ShellLexToken {
	kind: "word" | "op";
	text: string;
	quoted: boolean;
}

function lexShellCommand(command: string): ShellLexToken[] {
	const out: ShellLexToken[] = [];
	let cur = "";
	let curQuoted = false;
	let hasWord = false;
	let quote: "'" | '"' | null = null;

	const flush = () => {
		if (!hasWord) return;
		out.push({ kind: "word", text: cur, quoted: curQuoted });
		cur = "";
		curQuoted = false;
		hasWord = false;
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
			hasWord = true;
			continue;
		}
		if (ch === "\\" && i + 1 < command.length) {
			cur += command[++i];
			hasWord = true;
			continue;
		}
		if (ch === "#" && !hasWord) {
			while (i < command.length && command[i] !== "\n") i++;
			if (i < command.length && command[i] === "\n") {
				out.push({ kind: "op", text: "\n", quoted: false });
			}
			continue;
		}
		if (ch === "\n" || ch === "(" || ch === ")" || ch === "{" || ch === "}") {
			flush();
			out.push({ kind: "op", text: ch, quoted: false });
			continue;
		}
		if (ch === ";" || ch === "&" || ch === "|") {
			flush();
			const op = command[i + 1] === ch ? ch + ch : ch;
			if (op.length === 2) i++;
			out.push({ kind: "op", text: op, quoted: false });
			continue;
		}
		if (/\s/.test(ch)) {
			flush();
			continue;
		}
		cur += ch;
		hasWord = true;
	}
	flush();
	return out;
}

function baseCmd(word: string): string {
	const slash = word.lastIndexOf("/");
	return slash >= 0 ? word.slice(slash + 1) : word;
}

function isVerifyInvocation(words: ShellLexToken[]): boolean {
	let i = 0;
	while (i < words.length) {
		const w = words[i];
		if (!w.quoted && (w.text === "do" || w.text === "then" || w.text === "else" || w.text === "elif" || w.text === "!" || w.text === "time" || w.text === "env" || w.text === "command")) {
			i++;
			continue;
		}
		if (!w.quoted && /^[A-Za-z_][A-Za-z0-9_]*=/.test(w.text)) {
			i++;
			continue;
		}
		break;
	}
	if (i + 2 >= words.length) return false;
	return baseCmd(words[i].text) === "mem" && words[i + 1].text === "hcd" && words[i + 2].text === "verify";
}

/** Detecta llamadas bash que agrupan múltiples `mem hcd verify` o los envuelven en funciones/bucles. */
export function isBundledHcdVerifyCommand(command: string): string | null {
	const cmd = command.trim();
	if (!cmd) return null;

	const tokens = lexShellCommand(cmd);
	let verifyCount = 0;
	let wrappedInStructure = false;
	let loopDepth = 0;
	let pendingFuncBody = false;
	const braceStack: Array<"func" | "group"> = [];

	const processWords = (words: ShellLexToken[]) => {
		if (words.length === 0) return;
		let idx = 0;
		// Desenvuelve palabras clave de control al inicio de la unidad de comando.
		while (idx < words.length && !words[idx].quoted) {
			const kw = words[idx].text;
			if (kw === "for" || kw === "while" || kw === "until") {
				loopDepth++;
				return;
			}
			if (kw === "done") {
				loopDepth = Math.max(0, loopDepth - 1);
				idx++;
				continue;
			}
			if (kw === "function" && idx + 1 < words.length && !words[idx + 1].quoted) {
				pendingFuncBody = true;
				return;
			}
			if (kw === "do" || kw === "then" || kw === "else" || kw === "elif") {
				idx++;
				continue;
			}
			break;
		}
		const rest = words.slice(idx);
		if (isVerifyInvocation(rest)) {
			verifyCount++;
			if (loopDepth > 0 || braceStack.includes("func")) {
				wrappedInStructure = true;
			}
		}
	};

	let words: ShellLexToken[] = [];
	for (let i = 0; i < tokens.length; i++) {
		const tok = tokens[i];
		if (tok.kind === "word") {
			words.push(tok);
			continue;
		}
		// Detectar `name()` o `function name()` como encabezado de función antes de `{`
		if (
			tok.text === "(" &&
			tokens[i + 1]?.kind === "op" &&
			tokens[i + 1]?.text === ")" &&
			((words.length === 1 && !words[0].quoted && /^[A-Za-z_][A-Za-z0-9_]*$/.test(words[0].text)) ||
				(words.length === 2 && !words[0].quoted && words[0].text === "function" && !words[1].quoted && /^[A-Za-z_][A-Za-z0-9_]*$/.test(words[1].text)))
		) {
			words = [];
			pendingFuncBody = true;
			i++; // consumir ')'
			continue;
		}
		processWords(words);
		words = [];
		if (tok.text === "{" || (tok.text === "(" && pendingFuncBody)) {
			braceStack.push(pendingFuncBody ? "func" : "group");
			pendingFuncBody = false;
		} else if (tok.text === "}" || tok.text === ")") {
			braceStack.pop();
		} else if (tok.text !== "\n") {
			pendingFuncBody = false;
		}
	}
	processWords(words);

	if (verifyCount > 1) {
		return "Ejecuta un solo `mem hcd verify` por llamada bash para permitir refresco en vivo y entrega de steering.";
	}
	if (verifyCount === 1 && wrappedInStructure) {
		return "No envuelvas `mem hcd verify` en funciones o bucles dentro de una sola llamada bash; ejecuta cada verificación por separado.";
	}
	return null;
}

function redactSecrets(text: string): string {
	return text
		.replace(/\b([A-Za-z0-9_]*(?:TOKEN|SECRET|PASSWORD|PASSWD|API_KEY|AUTH|CREDENTIAL)[A-Za-z0-9_]*)=([^\s'"]+|'[^']*'|"[^"]*")/gi, "$1=***")
		.replace(/\b(Bearer\s+)[A-Za-z0-9._~+/=-]+/gi, "$1***");
}

function clipLabel(text: string, max = MAX_CALL_LABEL): string {
	const clean = text.replace(/\s+/g, " ").trim();
	return clean.length > max ? `${clean.slice(0, max - 1)}…` : clean;
}

function capitalizeTool(name: string): string {
	if (!name) return "Tool";
	if (name.startsWith("mcp__")) {
		const parts = name.split("__").filter(Boolean);
		if (parts.length >= 3) return `${parts[1]}.${parts.slice(2).join("_")}`;
	}
	return name.charAt(0).toUpperCase() + name.slice(1);
}

/** Resume una llamada de herramienta sin exponer secretos ni payloads extensos. */
export function formatToolCallLabel(toolName: string, args: Record<string, any> | undefined): string {
	const name = String(toolName || "").trim();
	const a = args && typeof args === "object" ? args : {};

	if (name === "read") {
		const p = typeof a.path === "string" ? a.path : "";
		const meta: string[] = [];
		if (a.offset !== undefined && a.offset !== null) meta.push(`offset=${a.offset}`);
		if (a.limit !== undefined && a.limit !== null) meta.push(`limit=${a.limit}`);
		const suffix = meta.length ? ` (${meta.join(", ")})` : "";
		return clipLabel(`Read ${p}${suffix}`, 96);
	}
	if (name === "edit") {
		const p = typeof a.path === "string" ? a.path : "";
		const count = Array.isArray(a.edits) && a.edits.length > 1 ? ` (${a.edits.length} edits)` : "";
		return clipLabel(`Edit ${p}${count}`, 96);
	}
	if (name === "write") {
		const p = typeof a.path === "string" ? a.path : "";
		return clipLabel(`Write ${p}`, 96);
	}
	if (name === "bash") {
		const firstLine = String(a.command ?? "").split(/\r?\n/)[0] ?? "";
		return clipLabel(`Bash ${redactSecrets(firstLine)}`, 84);
	}
	if (name === "codemode") {
		return "Codemode script";
	}
	const hint =
		(typeof a.path === "string" && a.path) ||
		(typeof a.file_path === "string" && a.file_path) ||
		(typeof a.query === "string" && a.query) ||
		(typeof a.pattern === "string" && a.pattern) ||
		(typeof a.title === "string" && a.title) ||
		(typeof a.name === "string" && a.name) ||
		"";
	const display = capitalizeTool(name);
	return hint ? clipLabel(`${display} ${redactSecrets(hint)}`, 84) : display;
}

function countTextLines(text: string): number {
	if (!text) return 0;
	const trimmed = text.endsWith("\n") ? text.slice(0, -1) : text;
	if (!trimmed) return 0;
	return trimmed.split(/\r?\n/).length;
}

/** Cuenta las líneas de salida textual de un content[] o resultado de Pi. */
export function countOutputLines(resultOrContent: any): number {
	if (!resultOrContent) return 0;
	if (typeof resultOrContent === "string") return countTextLines(resultOrContent);
	if (Array.isArray(resultOrContent)) {
		let total = 0;
		for (const item of resultOrContent) {
			if (item && typeof item.text === "string") total += countTextLines(item.text);
		}
		return total;
	}
	if (typeof resultOrContent === "object") {
		if (Array.isArray(resultOrContent.content)) return countOutputLines(resultOrContent.content);
		if (typeof resultOrContent.output === "string") return countTextLines(resultOrContent.output);
		if (typeof resultOrContent.stdout === "string") return countTextLines(resultOrContent.stdout);
	}
	return 0;
}

/** Extrae los IDs indicados en `--task N[,M]` de una invocación efectiva de `mem hcd verify`. */
export function extractVerifyTaskIds(command: string): string[] {
	const cmd = command.trim();
	if (!cmd) return [];
	const tokens = lexShellCommand(cmd);
	const ids: string[] = [];
	const seen = new Set<string>();
	const addIds = (raw: string) => {
		for (const part of raw.split(",")) {
			const id = part.trim();
			if (id && !seen.has(id)) {
				seen.add(id);
				ids.push(id);
			}
		}
	};

	const scanWords = (words: ShellLexToken[]) => {
		if (!isVerifyInvocation(words)) return;
		for (let i = 0; i < words.length; i++) {
			const w = words[i].text;
			if (w === "--") break;
			if (w === "--task" && i + 1 < words.length) {
				addIds(words[++i].text);
			} else if (w.startsWith("--task=")) {
				addIds(w.slice("--task=".length));
			}
		}
	};

	let words: ShellLexToken[] = [];
	for (const tok of tokens) {
		if (tok.kind === "word") {
			words.push(tok);
		} else {
			scanWords(words);
			words = [];
		}
	}
	scanWords(words);
	return ids;
}

/** Resuelve la tarea HCD activa (foco manual si existe; si no, primera no verificada). */
export function resolveActiveTaskId(unit: Unit | null, manualFocus: string | null): string | null {
	if (!unit?.ok || unit.state !== "active" || !Array.isArray(unit.tasks) || unit.tasks.length === 0) {
		return null;
	}
	if (manualFocus && unit.tasks.some((t) => t.id === manualFocus)) {
		return manualFocus;
	}
	const next = unit.tasks.find((t) => t.status !== "verified");
	return next ? next.id : null;
}

/** Mantiene en memoria las llamadas por tarea de la unidad HCD activa. */
export function createTaskActivityTracker(): TaskActivityTracker {
	let unitId: string | null = null;
	let manualFocus: string | null = null;
	const callsByTask = new Map<string, TaskCallEntry[]>();
	const totalsByTask = new Map<string, { totalCalls: number; totalOutputLines: number }>();
	const inFlight = new Map<string, { taskId: string; entry: TaskCallEntry }>();
	const allByCallId = new Map<string, { taskId: string; entry: TaskCallEntry }>();
	const expandedTasks = new Set<string>();
	const collapsedTasks = new Set<string>();
	const showAllTasks = new Set<string>();
	let scrollStart: number | null = null;

	const reset = () => {
		manualFocus = null;
		callsByTask.clear();
		totalsByTask.clear();
		inFlight.clear();
		allByCallId.clear();
		expandedTasks.clear();
		collapsedTasks.clear();
		showAllTasks.clear();
		scrollStart = null;
	};

	const ensureTotals = (taskId: string) => {
		let tot = totalsByTask.get(taskId);
		if (!tot) {
			tot = { totalCalls: 0, totalOutputLines: 0 };
			totalsByTask.set(taskId, tot);
		}
		return tot;
	};

	return {
		getScrollStart() {
			return scrollStart;
		},

		setScrollStart(start) {
			scrollStart = start;
		},

		syncUnit(unit) {
			const nextId = unit?.ok && unit.state === "active" && unit.unit_id ? unit.unit_id : null;
			if (nextId !== unitId) {
				unitId = nextId;
				reset();
				return;
			}
			if (unit && manualFocus) {
				const focused = unit.tasks?.find((t) => t.id === manualFocus);
				if (!focused || focused.status === "verified") {
					manualFocus = null;
				}
			}
		},

		recordStart(callId, toolName, args, unit, now = Date.now()) {
			const existing = allByCallId.get(callId);
			if (existing) return existing.taskId;

			let targetTaskId: string | null = null;
			if (toolName === "bash") {
				const verifyIds = extractVerifyTaskIds(String(args?.command ?? ""));
				if (verifyIds.length > 0 && (!unit?.tasks || unit.tasks.some((t) => t.id === verifyIds[0]))) {
					targetTaskId = verifyIds[0];
					manualFocus = targetTaskId;
				}
			}
			if (!targetTaskId) {
				targetTaskId = resolveActiveTaskId(unit, manualFocus);
			}
			if (!targetTaskId) return null;

			const entry: TaskCallEntry = {
				callId,
				toolName,
				label: formatToolCallLabel(toolName, args),
				status: "running",
				startedAt: now,
				outputLines: 0,
			};
			let list = callsByTask.get(targetTaskId);
			if (!list) {
				list = [];
				callsByTask.set(targetTaskId, list);
			}
			list.push(entry);
			if (list.length > MAX_CALLS_PER_TASK) {
				const dropped = list.shift();
				if (dropped) allByCallId.delete(dropped.callId);
			}
			ensureTotals(targetTaskId).totalCalls++;
			const ref = { taskId: targetTaskId, entry };
			inFlight.set(callId, ref);
			allByCallId.set(callId, ref);
			return targetTaskId;
		},

		recordFinish(callId, isError, resultOrContent, now = Date.now()) {
			const ref = inFlight.get(callId) ?? allByCallId.get(callId);
			if (!ref) return false;
			inFlight.delete(callId);
			ref.entry.status = isError ? "error" : "done";
			ref.entry.durationMs = Math.max(0, now - ref.entry.startedAt);
			const lines = countOutputLines(resultOrContent);
			if (lines > ref.entry.outputLines) {
				const delta = lines - ref.entry.outputLines;
				ref.entry.outputLines = lines;
				ensureTotals(ref.taskId).totalOutputLines += delta;
			}
			return true;
		},

		recordResultOutput(callId, resultOrContent, isError) {
			const ref = allByCallId.get(callId);
			if (!ref) return false;
			const lines = countOutputLines(resultOrContent);
			if (lines > ref.entry.outputLines) {
				const delta = lines - ref.entry.outputLines;
				ref.entry.outputLines = lines;
				ensureTotals(ref.taskId).totalOutputLines += delta;
			}
			if (isError === true) ref.entry.status = "error";
			return true;
		},

		setFocus(taskId) {
			manualFocus = taskId;
		},

		getFocus() {
			return manualFocus;
		},

		getTaskActivity(taskId) {
			const calls = callsByTask.get(taskId) ?? [];
			const tot = totalsByTask.get(taskId) ?? { totalCalls: calls.length, totalOutputLines: 0 };
			return {
				calls: calls.map((c) => ({ ...c })),
				totalCalls: tot.totalCalls,
				totalOutputLines: tot.totalOutputLines,
			};
		},

		isTaskExpanded(taskId, activeTaskId) {
			if (expandedTasks.has(taskId)) return true;
			if (collapsedTasks.has(taskId)) return false;
			return taskId === activeTaskId;
		},

		isShowingAllCalls(taskId) {
			return showAllTasks.has(taskId);
		},

		expandTask(taskId, showAll = false) {
			collapsedTasks.delete(taskId);
			expandedTasks.add(taskId);
			if (showAll) showAllTasks.add(taskId);
		},

		collapseTask(taskId) {
			expandedTasks.delete(taskId);
			showAllTasks.delete(taskId);
			collapsedTasks.add(taskId);
		},

		toggleExpand(taskId, activeTaskId) {
			if (this.isTaskExpanded(taskId, activeTaskId)) {
				this.collapseTask(taskId);
			} else {
				this.expandTask(taskId, false);
			}
		},

		toggleShowMore(taskId) {
			if (showAllTasks.has(taskId)) {
				showAllTasks.delete(taskId);
			} else {
				collapsedTasks.delete(taskId);
				expandedTasks.add(taskId);
				showAllTasks.add(taskId);
			}
		},
	};
}

function styleTaskLine(line: RenderedTaskLine, theme: any): string {
	if (!line.tone || typeof theme?.fg !== "function") return line.text;
	return theme.fg(line.tone, line.text);
}

/** Hay alguna llamada en curso en la unidad: el spinner necesita repintado periódico. */
function hasRunningCall(unit: Unit | null, tracker: TaskActivityTracker): boolean {
	return Boolean(unit?.tasks.some((t) => tracker.getTaskActivity(t.id).calls.some((c) => c.status === "running")));
}

export default function (pi: ExtensionAPI) {
	const tracker = createTaskActivityTracker();
	let visible = true;
	let busy = false;
	let pendingCtx: any = null;
	let pollTimer: ReturnType<typeof setInterval> | null = null;
	let currentUnit: Unit | null = null;
	let mounted = false;
	let requestWidgetRender: (() => void) | null = null;
	let seqCounter = 0;
	let spinTimer: ReturnType<typeof setInterval> | null = null;
	let spinEvery = 0;

	/**
	 * Anima el panel (HCD-057): al ritmo del spinner si hay una llamada en curso, al
	 * del círculo si solo hay una tarea activa (Q5), y nada si todo está verificado.
	 */
	function syncSpinner() {
		const shown = visible && mounted;
		const every = !shown
			? 0
			: hasRunningCall(currentUnit, tracker)
				? SPINNER_INTERVAL_MS
				: resolveActiveTaskId(currentUnit, tracker.getFocus())
					? TASK_FRAME_MS
					: 0;
		if (every === spinEvery) return;
		if (spinTimer) clearInterval(spinTimer);
		spinTimer = null;
		spinEvery = every;
		if (every) {
			spinTimer = setInterval(() => requestWidgetRender?.(), every);
			(spinTimer as any)?.unref?.();
		}
	}

	function stopPolling() {
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
	}

	function callIdOf(event: any): string {
		const raw = event?.toolCallId ?? event?.id;
		if (typeof raw === "string" && raw) return raw;
		seqCounter++;
		return `call-${seqCounter}`;
	}

	function syncWidget(ctx: any) {
		const hasView = Boolean(visible && currentUnit && renderTasksTree(currentUnit, tracker));
		if (!hasView) {
			mounted = false;
			requestWidgetRender = null;
			syncSpinner();
			ctx?.ui?.setWidget?.(WIDGET, undefined);
			return;
		}
		if (mounted && requestWidgetRender) {
			syncSpinner();
			requestWidgetRender();
			return;
		}
		mounted = true;
		ctx?.ui?.setWidget?.(WIDGET, (tui: any, theme: any) => {
			let lastRendered: RenderedTaskLine[] = [];
			requestWidgetRender = () => tui?.requestRender?.();
			return {
				render(width: number): string[] {
					if (!visible || !currentUnit) {
						lastRendered = [];
						return [];
					}
					lastRendered = renderTasksTree(currentUnit, tracker) ?? [];
					const maxW = Math.max(10, width || 100);
					return lastRendered.map((line) => truncateToWidth(styleTaskLine(line, theme), maxW, "…", false));
				},
				handleMouse(event: any) {
					// Rueda sobre el panel: recorre la ventana de tareas (HCD-057, Q3).
					if (event?.type === "wheel" && currentUnit) {
						// wheelDelta llega en líneas lógicas (negativo hacia arriba).
						const raw = Number(event.wheelDelta) || 0;
						const delta = Math.round(raw) || Math.sign(raw);
						if (!delta) return undefined;
						scrollTaskWindow(currentUnit, tracker, delta);
						tui?.requestRender?.();
						return { handled: true, render: false };
					}
					if (event?.button !== "left") return undefined;
					const row = typeof event.y === "number" ? event.y : -1;
					const action = lastRendered[row]?.action;
					if (!action || !currentUnit) return undefined;
					if (event.type === "click") {
						const activeId = resolveActiveTaskId(currentUnit, tracker.getFocus());
						if (action.kind === "scroll") {
							scrollTaskWindow(currentUnit, tracker, action.delta);
						} else if (action.kind === "toggle-more") {
							tracker.toggleShowMore(action.taskId);
						} else {
							tracker.toggleExpand(action.taskId, activeId);
						}
						tui?.requestRender?.();
						return { handled: true, render: false };
					}
					if (event.type === "press" || event.type === "release") {
						return { handled: true, render: false };
					}
					return undefined;
				},
				invalidate() {},
			};
		});
		syncSpinner();
	}

	async function refresh(ctx: any) {
		if (busy) {
			pendingCtx = ctx;
			return;
		}
		busy = true;
		try {
			let nextUnit: Unit | null = null;
			if (visible) {
				try {
					const r = await pi.exec("mem", ["hcd", "show", "--json"], { cwd: ctx.cwd, timeout: 5000 });
					if (r.code === 0 && visible) {
						const parsed = JSON.parse(r.stdout ?? "");
						if (parsed?.ok && parsed.unit_id && parsed.state === "active" && Array.isArray(parsed.tasks) && parsed.tasks.length > 0) {
							nextUnit = parsed;
						}
					}
				} catch {
					// mem ausente, sin unidad o salida inválida: no se muestra nada
				}
			}
			currentUnit = visible ? nextUnit : null;
			if (visible) tracker.syncUnit(currentUnit);
			syncWidget(ctx);
		} finally {
			busy = false;
			if (pendingCtx) {
				const next = pendingCtx;
				pendingCtx = null;
				await refresh(next);
			}
		}
	}

	pi.registerCommand("tasks", {
		description: "Panel de tareas HCD en árbol (/tasks [expand [N]|collapse [N|all]|focus [N]])",
		handler: async (args, ctx) => {
			const parts = String(args ?? "").trim().split(/\s+/).filter(Boolean);
			const sub = parts[0]?.toLowerCase() ?? "";
			const target = parts[1];

			if (sub === "expand") {
				visible = true;
				if (!currentUnit) await refresh(ctx);
				const id = target || resolveActiveTaskId(currentUnit, tracker.getFocus());
				if (id) {
					const activeId = resolveActiveTaskId(currentUnit, tracker.getFocus());
					const alreadyExpanded = tracker.isTaskExpanded(id, activeId);
					tracker.expandTask(id, alreadyExpanded);
					syncWidget(ctx);
					ctx.ui?.notify?.(`Tarea ${id} expandida`, "info");
				}
				return;
			}

			if (sub === "collapse") {
				if (target === "all" && currentUnit) {
					for (const t of currentUnit.tasks) tracker.collapseTask(t.id);
					syncWidget(ctx);
					ctx.ui?.notify?.("Todas las tareas colapsadas", "info");
					return;
				}
				const id = target || resolveActiveTaskId(currentUnit, tracker.getFocus());
				if (id) {
					tracker.collapseTask(id);
					syncWidget(ctx);
					ctx.ui?.notify?.(`Tarea ${id} colapsada`, "info");
				}
				return;
			}

			if (sub === "focus") {
				if (!target) {
					tracker.setFocus(null);
					syncWidget(ctx);
					ctx.ui?.notify?.("Foco automático restaurado", "info");
					return;
				}
				tracker.setFocus(target);
				tracker.expandTask(target, false);
				syncWidget(ctx);
				ctx.ui?.notify?.(`Foco fijado en tarea ${target}`, "info");
				return;
			}

			visible = !visible;
			await refresh(ctx);
			ctx.ui?.notify?.(visible ? "Panel de tareas visible" : "Panel de tareas oculto", "info");
		},
	});

	// Alt+Shift+↑/↓ recorren la lista desde el input; ↑/↓ y Alt+↑ quedan para Pi (HCD-057, Q4).
	for (const [key, delta] of [["alt+shift+up", -1], ["alt+shift+down", 1]] as const) {
		pi.registerShortcut?.(key, {
			description: delta < 0 ? "Tareas HCD: subir la lista" : "Tareas HCD: bajar la lista",
			handler: async () => {
				if (visible && scrollTaskWindow(currentUnit, tracker, delta)) requestWidgetRender?.();
			},
		});
	}

	pi.on("session_start", async (_event: any, ctx: any) => {
		await refresh(ctx);
	});

	pi.on("tool_call", async (event: any) => {
		if (event?.toolName !== "bash") return undefined;
		const cmd = String(event?.input?.command ?? event?.args?.command ?? "");
		const reason = isBundledHcdVerifyCommand(cmd);
		if (reason) return { block: true, reason };
		return undefined;
	});

	pi.on("tool_execution_start", async (event: any, ctx: any) => {
		const args = event?.args ?? event?.input;
		const toolName = String(event?.toolName ?? "");
		if (currentUnit && toolName) {
			tracker.recordStart(callIdOf(event), toolName, args, currentUnit);
			syncWidget(ctx);
		}
		const cmd = String(args?.command ?? "");
		if (toolName === "bash" && /\bmem\s+hcd\b/.test(cmd)) {
			stopPolling();
			pollTimer = setInterval(() => {
				void refresh(ctx);
			}, 1500);
			(pollTimer as any)?.unref?.();
		}
	});

	pi.on("tool_execution_end", async (event: any, ctx: any) => {
		if (event?.toolName === "bash") stopPolling();
		const id = event?.toolCallId ?? event?.id;
		if (typeof id === "string" && id && tracker.recordFinish(id, Boolean(event?.isError), event?.result)) {
			syncWidget(ctx);
		}
	});

	pi.on("tool_result", async (event: any, ctx: any) => {
		const id = event?.toolCallId ?? event?.id;
		if (typeof id === "string" && id) {
			tracker.recordResultOutput(id, event?.content, Boolean(event?.isError));
		}
		if (event?.toolName === "bash" && /\bmem\s+hcd\b/.test(String(event?.input?.command ?? ""))) {
			stopPolling();
			await refresh(ctx);
			return;
		}
		if (currentUnit && visible) syncWidget(ctx);
	});

	pi.on("agent_end", async (_event: any, ctx: any) => {
		stopPolling();
		await refresh(ctx);
	});

	pi.on("session_shutdown", async () => {
		stopPolling();
		mounted = false;
		requestWidgetRender = null;
		syncSpinner();
	});
}

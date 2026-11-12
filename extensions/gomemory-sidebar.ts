/**
 * gomemory-sidebar.ts — Panel lateral de estado inspirado en OpenCode para Pi Coding Agent.
 *
 * Muestra un overlay superior derecho con:
 *  1. Título de la sesión o unidad HCD activa (mem hcd show --json).
 *  2. Métricas de contexto (tokens usados, % de ventana y gasto acumulado en USD).
 *  3. Servidores MCP declarados en mcp.json (estado "Configured"; no se comprueba la conexión).
 *
 * Comando: /sidebar para alternar visibilidad.
 */

import { readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
import type { AssistantMessage } from "@earendil-works/pi-ai";
import type { ExtensionAPI, ExtensionCommandContext, Theme } from "@earendil-works/pi-coding-agent";
import { truncateToWidth, visibleWidth } from "@earendil-works/pi-tui";

const EXTENSION_NAME = "gomemory-sidebar";
const OVERLAY_WIDTH = 36;

/**
 * Estado compartido con el panel /tdd (otra extensión, otro módulo): el sidebar
 * publica si está visible y cuántas filas ocupa para que el TDD se apile debajo.
 */
interface PanelDock {
	sidebarVisible: boolean;
	sidebarHeight: number;
	/** Controles que publica gomemory-tdd.ts para que /min alterne ambos paneles. */
	tdd?: { isVisible(): boolean; isMinimized(): boolean; setMinimized(minimized: boolean): void };
}
const DOCK_KEY = Symbol.for("pi-gomemory.dock");
function panelDock(): PanelDock {
	const g = globalThis as any;
	g[DOCK_KEY] ??= { sidebarVisible: false, sidebarHeight: 0 };
	return g[DOCK_KEY];
}

/**
 * En fullscreen, pi-tui entrega un clic como press → release → click al mismo
 * componente. Solo el click alterna; press y release se marcan manejados para
 * que el click llegue (sin press manejado no hay destino) y no alternen dos veces.
 */
export function headerClickToggles(event: any, minimized: boolean): "toggle" | "consume" | undefined {
	if (event?.button !== "left") return undefined;
	const onHeader = minimized ? event.y === 0 : event.y <= 1;
	if (!onHeader) return undefined;
	if (event.type === "click") return "toggle";
	if (event.type === "press" || event.type === "release") return "consume";
	return undefined;
}

export interface MCPServerInfo {
	name: string;
	status: "Configured";
}

export interface QuotaBucketInfo {
	displayName: string;
	remainingFraction: number;
	remainingPercent: number;
	resetIn: string;
}

export interface QuotaGroupInfo {
	displayName: string;
	buckets: QuotaBucketInfo[];
}

export interface QuotaSummaryInfo {
	planLabel: string;
	email?: string;
	groups: QuotaGroupInfo[];
}

export interface ClaudeTokenBucketInfo {
	input: number;
	output: number;
	cache_create: number;
	cache_read: number;
	messages: number;
}

export interface ClaudeUsageInfo {
	sessions: number;
	today: ClaudeTokenBucketInfo;
	week: ClaudeTokenBucketInfo;
	total: ClaudeTokenBucketInfo;
}

/** Parte un texto plano en líneas de como mucho `width` columnas por palabras (máximo `maxLines`). */
export function wrapWords(text: string, width: number, maxLines: number): string[] {
	const out: string[] = [];
	let line = "";
	for (const word of text.split(/\s+/).filter(Boolean)) {
		const next = line ? `${line} ${word}` : word;
		if (visibleWidth(next) <= width) {
			line = next;
			continue;
		}
		if (line) out.push(line);
		line = word;
	}
	if (line) out.push(line);
	if (out.length > maxLines) {
		const kept = out.slice(0, maxLines);
		kept[maxLines - 1] = truncateToWidth(`${kept[maxLines - 1]} …`, width, "…");
		return kept;
	}
	return out;
}

/**
 * Mensaje para el sidebar cuando anthropic-usage falla. Un binario anterior a
 * HCD-023 no conoce el comando: se indica cómo actualizarlo en vez del error crudo.
 */
export function anthropicPlanErrorMessage(stderr: string): string {
	if (/Comando desconocido: anthropic-usage/.test(stderr || "")) {
		return "pi-gomemory-setup sin anthropic-usage: ejecuta pi-gomemory-setup update";
	}
	return parseCleanQuotaError(stderr);
}

/** Límite del plan de Anthropic (pi-gomemory-setup anthropic-usage). */
export interface AnthropicLimitInfo {
	name: string;
	percent: number;
	resetsAt: string;
	resetIn: string;
}

export interface AnthropicPlanInfo {
	limits: AnthropicLimitInfo[];
}

export interface SidebarState {
	title: string;
	tokens: number;
	contextLimit: number;
	cost: number;
	servers: MCPServerInfo[];
	providerName?: string;
	quota?: QuotaSummaryInfo | null;
	quotaError?: string;
	isAntigravityProvider?: boolean;
	claudeUsage?: ClaudeUsageInfo | null;
	isClaudeProvider?: boolean;
	/** Límites del plan de la conexión OAuth de Pi con Anthropic (sesión, semana…). */
	anthropicPlan?: AnthropicPlanInfo | null;
	anthropicPlanError?: string;
}

export function formatNumber(n: number): string {
	return n.toLocaleString("en-US");
}

export function formatPercent(used: number, total: number): string {
	if (total <= 0) return "0% used";
	const pct = Math.min(100, Math.max(0, Math.round((used / total) * 100)));
	return `${pct}% used`;
}

export function formatCost(cost: number): string {
	return `$${cost.toFixed(2)} spent`;
}

export function formatProgressBar(remFraction: number, width = 10): string {
	const filled = Math.max(0, Math.min(width, Math.round(remFraction * width)));
	return `${"█".repeat(filled)}${"░".repeat(width - filled)}`;
}

export function formatCompactNumber(n: number): string {
	if (n >= 1e9) return `${(n / 1e9).toFixed(1)}B`;
	if (n >= 1e6) return `${(n / 1e6).toFixed(1)}M`;
	if (n >= 1e3) return `${(n / 1e3).toFixed(1)}k`;
	return String(n);
}

export function compactBucketName(name: string): string {
	const lower = name.toLowerCase();
	if (lower.includes("five hour") || lower.includes("5h") || lower.includes("5 hour")) return "5 Horas";
	if (lower.includes("argon") && lower.includes("weekly")) return "Argon";
	if (lower.includes("weekly")) return "Semanal";
	return name.replace("Limit Remaining", "").trim();
}

/**
 * Lee los servidores MCP declarados en ~/.pi/agent/mcp.json. Solo afirma que
 * están configurados: no comprueba la conexión. Sin archivo válido, lista vacía.
 */
export function getRegisteredMCPServers(agentDir?: string): MCPServerInfo[] {
	const dir = agentDir || join(homedir(), ".pi", "agent");
	try {
		const data = JSON.parse(readFileSync(join(dir, "mcp.json"), "utf8"));
		return Object.keys(data?.mcpServers ?? {}).map((name) => ({ name, status: "Configured" }));
	} catch {
		return [];
	}
}

/**
 * Recolecta métricas de tokens y coste desde la sesión activa de Pi.
 */
export function collectContextMetrics(ctx: any): { tokens: number; contextLimit: number; cost: number } {
	let tokens = 0;
	let cost = 0;
	let lastTurnTokens = 0;
	let contextLimit = 200000;

	try {
		if (ctx?.sessionManager?.getBranch) {
			for (const entry of ctx.sessionManager.getBranch()) {
				if (entry.type === "message" && entry.message?.role === "assistant") {
					const msg = entry.message as AssistantMessage;
					if (msg.usage) {
						tokens += (msg.usage.input || 0) + (msg.usage.output || 0);
						cost += msg.usage.cost?.total || 0;
						lastTurnTokens = (msg.usage.input || 0) + (msg.usage.output || 0);
					}
				}
			}
		}
		if (ctx?.model?.contextWindow) {
			contextLimit = ctx.model.contextWindow;
		}
	} catch {
		// Silencioso ante stale ctx o sesión en transición
	}

	const activeTokens = lastTurnTokens > 0 ? lastTurnTokens : tokens;
	return { tokens: activeTokens, contextLimit, cost };
}

/**
 * Modo minimizado: una sola fila con el uso de contexto y el indicador [+] para restaurar.
 */
export function renderSidebarMinimized(state: SidebarState, width: number, theme: Theme): string[] {
	const innerW = Math.max(10, width);
	const border = (c: string) => theme.fg("border", c);
	const pct = state.contextLimit > 0 ? Math.round((state.tokens / state.contextLimit) * 100) : 0;
	const left = ` HCD · ${formatCompactNumber(state.tokens)} (${pct}%) $${state.cost.toFixed(2)}`;
	const right = `[+] (/sidebar min) `;
	const leftLen = visibleWidth(left);
	const rightLen = visibleWidth(right);
	const space = Math.max(1, innerW - leftLen - rightLen - 2);
	const content = theme.fg("accent", left) + " ".repeat(space) + theme.fg("dim", right);
	const row = border("[") + content + border("]");
	return [truncateToWidth(row, innerW, "…", true)];
}

/**
 * Renderiza el contenido del panel lateral respetando el ancho disponible.
 */
export function renderSidebar(state: SidebarState, width: number, theme: Theme): string[] {
	const innerW = Math.max(10, width - 2);
	const border = (c: string) => theme.fg("border", c);
	const padLine = (s: string) => truncateToWidth(s, innerW, "…", true);

	const lines: string[] = [];
	lines.push(border(`╭${"─".repeat(innerW)}╮`));

	// 1. Título principal con indicador clicable de minimizado [−]. El título se recorta
	//    ANTES de unirlo: el presupuesto de [−] es fijo y el control nunca debe desaparecer (C-004).
	const MIN_HINT = "[−] ";
	const minLen = visibleWidth(MIN_HINT);
	const fullTitle = state.title.replace(/\s+/g, " ").trim() || "pi-gomemory";
	const titleWords = truncateToWidth(fullTitle, Math.max(1, innerW - minLen - 2), "…");
	const titleStyled = theme.style(titleWords, { fg: "success", bold: true });
	const minHint = theme.fg("dim", MIN_HINT);
	const titleLen = visibleWidth(titleWords);
	const space = Math.max(1, innerW - titleLen - minLen - 1);
	const headerRow = ` ${titleStyled}${" ".repeat(space)}${minHint}`;
	lines.push(border("│") + padLine(headerRow) + border("│"));

	lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));

	// 2. Sección Context
	const contextHeader = theme.style("Context", { fg: "success", bold: true });
	lines.push(border("│") + padLine(` ${contextHeader}`) + border("│"));

	const tokenLine = theme.fg("text", ` ${formatNumber(state.tokens)} tokens`);
	lines.push(border("│") + padLine(tokenLine) + border("│"));

	const pctLine = theme.fg("text", ` ${formatPercent(state.tokens, state.contextLimit)}`);
	lines.push(border("│") + padLine(pctLine) + border("│"));

	const costLine = theme.fg("text", ` ${formatCost(state.cost)}`);
	lines.push(border("│") + padLine(costLine) + border("│"));

	// 3. Sección Antigravity Quotas (si el modelo o proveedor es antigravity / agy)
	if (state.isAntigravityProvider) {
		lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));
		const quotaHeader = theme.style("▼ Google AI Pro Quota", { fg: "success", bold: true });
		lines.push(border("│") + padLine(` ${quotaHeader}`) + border("│"));

		if (state.quotaError) {
			const errLine = theme.fg("error", ` ${state.quotaError}`);
			lines.push(border("│") + padLine(errLine) + border("│"));
		} else if (!state.quota || !state.quota.groups || state.quota.groups.length === 0) {
			const loadLine = theme.fg("dim", " Cargando cuota...");
			lines.push(border("│") + padLine(loadLine) + border("│"));
		} else {
			for (const group of state.quota.groups) {
				const gTitle = theme.fg("accent", ` ${group.displayName}`);
				lines.push(border("│") + padLine(gTitle) + border("│"));

				for (const bucket of group.buckets) {
					const pct = bucket.remainingPercent;
					const bar = formatProgressBar(bucket.remainingFraction, 8);
					let barColor: "success" | "warning" | "error" = "success";
					if (pct < 20) {
						barColor = "error";
					} else if (pct < 50) {
						barColor = "warning";
					}

					const label = compactBucketName(bucket.displayName);
					const labelStyled = theme.fg("text", label.padEnd(8, " "));
					const barStyled = theme.fg(barColor, bar);
					const pctStyled = theme.fg("text", `${pct.toFixed(0).padStart(3, " ")}%`);
					const resetStyled = theme.fg("dim", bucket.resetIn ? `· ${bucket.resetIn}` : "");

					const rowText = `  ${labelStyled} ${barStyled} ${pctStyled} ${resetStyled}`;
					lines.push(border("│") + padLine(rowText) + border("│"));
				}
			}
		}
	}

	// 4a. Límites del plan de Anthropic (conexión OAuth de Pi): % usado y reinicio.
	if (state.isClaudeProvider && (state.anthropicPlan || state.anthropicPlanError)) {
		lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));
		lines.push(border("│") + padLine(` ${theme.style("▼ Anthropic · plan", { fg: "success", bold: true })}`) + border("│"));
		if (state.anthropicPlanError) {
			// El error se parte en líneas (no se recorta): la parte accionable suele ir al final.
			for (const part of wrapWords(state.anthropicPlanError, innerW - 1, 4)) {
				lines.push(border("│") + padLine(theme.fg("error", ` ${part}`)) + border("│"));
			}
		} else {
			for (const limit of state.anthropicPlan!.limits) {
				const used = Math.max(0, Math.min(100, limit.percent));
				const color = used >= 80 ? "error" : used >= 50 ? "warning" : "success";
				const label = theme.fg("text", limit.name.padEnd(6, " "));
				const bar = theme.fg(color, formatProgressBar(used / 100, 8));
				const pct = theme.fg("text", `${used.toFixed(0).padStart(3, " ")}%`);
				const reset = limit.resetIn ? theme.fg("dim", ` · ${limit.resetIn}`) : "";
				lines.push(border("│") + padLine(`  ${label} ${bar} ${pct}${reset}`) + border("│"));
			}
		}
	}

	// 4b. Sección Claude Code Usage (si el modelo o proveedor es Claude / Anthropic)
	if (state.isClaudeProvider && state.claudeUsage) {
		lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));
		const claudeHeader = theme.style("▼ Claude Code Usage", { fg: "success", bold: true });
		lines.push(border("│") + padLine(` ${claudeHeader}`) + border("│"));

		const cu = state.claudeUsage;
		const sumBucket = (b: ClaudeTokenBucketInfo) => (b.input || 0) + (b.output || 0) + (b.cache_create || 0) + (b.cache_read || 0);

		const todayTot = sumBucket(cu.today);
		const weekTot = sumBucket(cu.week);
		const allTot = sumBucket(cu.total);

		const rowToday = `  Hoy:      ${theme.fg("accent", formatCompactNumber(todayTot).padStart(6, " "))} tokens (${cu.today.messages || 0} msgs)`;
		const rowWeek = `  Semana:   ${theme.fg("accent", formatCompactNumber(weekTot).padStart(6, " "))} tokens (${cu.week.messages || 0} msgs)`;
		const rowTotal = `  Total:    ${theme.fg("accent", formatCompactNumber(allTot).padStart(6, " "))} tokens`;

		lines.push(border("│") + padLine(rowToday) + border("│"));
		lines.push(border("│") + padLine(rowWeek) + border("│"));
		lines.push(border("│") + padLine(rowTotal) + border("│"));

		if (cu.total.cache_read > 0) {
			const cacheRatio = Math.round((cu.total.cache_read / Math.max(1, allTot)) * 100);
			const cacheLine = theme.fg("dim", `  Ahorro caché: ${cacheRatio}% lectura`);
			lines.push(border("│") + padLine(cacheLine) + border("│"));
		}
	}

	// 5. Sección Proveedor genérico (si no es Antigravity ni Claude, ej. OpenCode, Codex, etc.)
	if (!state.isAntigravityProvider && !state.isClaudeProvider && state.providerName) {
		lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));
		const provHeader = theme.style(`▼ ${state.providerName}`, { fg: "success", bold: true });
		lines.push(border("│") + padLine(` ${provHeader}`) + border("│"));
		const statusLine = theme.fg("text", "  Estado: Conectado (live)");
		lines.push(border("│") + padLine(statusLine) + border("│"));
	}

	lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));

	// 6. Sección MCP
	const mcpHeader = theme.style("▼ MCP", { fg: "success", bold: true });
	lines.push(border("│") + padLine(` ${mcpHeader}`) + border("│"));

	for (const srv of state.servers) {
		const bullet = theme.fg("success", "•");
		const nameStyled = theme.fg("accent", srv.name);
		const statusStyled = theme.fg("dim", srv.status);

		const leftPart = ` ${bullet} ${nameStyled}`;
		const rightPart = `${statusStyled} `;
		const leftLen = visibleWidth(leftPart);
		const rightLen = visibleWidth(rightPart);
		const space = Math.max(1, innerW - leftLen - rightLen);
		const row = leftPart + " ".repeat(space) + rightPart;

		lines.push(border("│") + truncateToWidth(row, innerW, "…", true) + border("│"));
	}

	lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));
	lines.push(border("│") + padLine(theme.fg("dim", " /sidebar min: compactar / restaurar")) + border("│"));
	lines.push(border(`╰${"─".repeat(innerW)}╯`));
	return lines;
}

export function shouldRefreshQuota(
	now: number,
	lastFetchTime: number,
	force = false,
	hasError = false,
	ttlMs = 30000,
	backoffMs = 15000,
): boolean {
	if (force) return true;
	const minInterval = hasError ? backoffMs : ttlMs;
	return now - lastFetchTime >= minInterval;
}

export function detectProvider(ctx: any): "antigravity" | "claude" | "opencode" | "codex" | "openai" | "other" {
	let provider = "";
	let modelId = "";
	try {
		provider = ((ctx?.model?.provider || process.env.PI_PROVIDER || "") as string).toLowerCase();
		modelId = ((ctx?.model?.id || process.env.PI_MODEL || "") as string).toLowerCase();
	} catch {
		provider = (process.env.PI_PROVIDER || "").toLowerCase();
		modelId = (process.env.PI_MODEL || "").toLowerCase();
	}
	if (
		provider.includes("antigravity") ||
		provider.includes("agy") ||
		modelId.includes("antigravity") ||
		modelId.includes("agy")
	) {
		return "antigravity";
	}
	if (
		provider.includes("anthropic") ||
		provider.includes("claude") ||
		modelId.includes("claude") ||
		provider.includes("claude-code")
	) {
		return "claude";
	}
	if (provider.includes("opencode") || modelId.includes("opencode")) {
		return "opencode";
	}
	if (provider.includes("codex") || modelId.includes("codex")) {
		return "codex";
	}
	if (provider.includes("openai") || modelId.includes("openai")) {
		return "openai";
	}
	return "other";
}

export function formatProviderDisplayName(ctx: any): string {
	const detected = detectProvider(ctx);
	if (detected === "antigravity") return "Google AI Pro";
	if (detected === "claude") return "Claude Code";
	if (detected === "opencode") return "OpenCode";
	if (detected === "codex") return "Codex";
	if (detected === "openai") return "OpenAI";
	let raw = "";
	try {
		raw = ((ctx?.model?.provider || process.env.PI_PROVIDER || "") as string).trim();
	} catch {
		raw = (process.env.PI_PROVIDER || "").trim();
	}
	return raw ? raw.toUpperCase() : "Provider";
}

export function parseCleanQuotaError(rawStderr: string): string {
	const trimmed = (rawStderr || "").trim();
	if (!trimmed) return "Error consultando cuota";
	try {
		const parsed = JSON.parse(trimmed);
		if (typeof parsed?.error === "string") return parsed.error;
	} catch {
		// No es JSON
	}
	return trimmed.split("\n")[0];
}

export function validateQuotaSummary(data: any): QuotaSummaryInfo | null {
	if (!data || typeof data !== "object") return null;
	if (!Array.isArray(data.groups)) return null;
	for (const g of data.groups) {
		if (!g || typeof g.displayName !== "string" || !Array.isArray(g.buckets)) return null;
		for (const b of g.buckets) {
			if (!b || typeof b.displayName !== "string" || typeof b.remainingFraction !== "number" || typeof b.remainingPercent !== "number") {
				return null;
			}
		}
	}
	return data as QuotaSummaryInfo;
}

export default function (pi: ExtensionAPI) {
	let visible = false;
	let overlayMinimized = false;
	let closeSidebar: (() => void) | null = null;
	let requestRender: ((force?: boolean) => void) | null = null;
	let autoRefreshTimer: ReturnType<typeof setInterval> | null = null;
	let cachedTitle = "Instalación de binario con componentes";
	let activeCtx: any = null;

	let cachedQuota: QuotaSummaryInfo | null = null;
	let cachedQuotaError: string | undefined = undefined;
	let lastQuotaFetchTime = 0;
	let quotaInflight = false;
	const QUOTA_TTL_MS = 30 * 1000;
	const QUOTA_ERROR_BACKOFF_MS = 15 * 1000;
	const HEARTBEAT_INTERVAL_MS = 30 * 1000;

	let cachedClaudeUsage: ClaudeUsageInfo | null = null;
	let lastClaudeFetchTime = 0;
	let claudeInflight = false;
	const CLAUDE_TTL_MS = 30 * 1000;

	function isAntigravity(ctx: any): boolean {
		return detectProvider(ctx) === "antigravity";
	}

	function isClaude(ctx: any): boolean {
		return detectProvider(ctx) === "claude";
	}

	let cachedAnthropicPlan: AnthropicPlanInfo | null = null;
	let cachedAnthropicPlanError: string | undefined = undefined;
	let lastAnthropicPlanFetch = 0;
	let anthropicPlanInflight = false;

	/** Límites del plan de la conexión OAuth de Pi con Anthropic (sesión y semana). */
	async function refreshAnthropicPlanIfNeeded(ctx: any, force = false) {
		if (!isClaude(ctx) || anthropicPlanInflight) return;
		if (!shouldRefreshQuota(Date.now(), lastAnthropicPlanFetch, force, !!cachedAnthropicPlanError, QUOTA_TTL_MS, QUOTA_ERROR_BACKOFF_MS)) {
			return;
		}
		anthropicPlanInflight = true;
		try {
			const r = await pi.exec("pi-gomemory-setup", ["anthropic-usage"], { timeout: 12000 });
			lastAnthropicPlanFetch = Date.now();
			const parsed = r.code === 0 && r.stdout ? JSON.parse(r.stdout) : null;
			if (parsed && Array.isArray(parsed.limits) && parsed.limits.length > 0) {
				cachedAnthropicPlan = parsed as AnthropicPlanInfo;
				cachedAnthropicPlanError = undefined;
			} else {
				// Nunca se conservan cifras viejas como si fueran actuales.
				cachedAnthropicPlan = null;
				cachedAnthropicPlanError = r.code === 0 ? "Respuesta de uso sin límites" : anthropicPlanErrorMessage(r.stderr);
			}
		} catch (err: any) {
			lastAnthropicPlanFetch = Date.now();
			cachedAnthropicPlan = null;
			cachedAnthropicPlanError = err?.message || "No disponible";
		} finally {
			anthropicPlanInflight = false;
		}
		requestRender?.();
	}

	async function refreshClaudeUsageIfNeeded(ctx: any, force = false) {
		// Los límites del plan se refrescan en los mismos momentos que el uso de Claude Code.
		void refreshAnthropicPlanIfNeeded(ctx, force);
		if (!isClaude(ctx) || claudeInflight) {
			return;
		}
		const now = Date.now();
		if (!shouldRefreshQuota(now, lastClaudeFetchTime, force, false, CLAUDE_TTL_MS, 15000)) {
			return;
		}

		claudeInflight = true;
		try {
			const r = await pi.exec("pi-gomemory-setup", ["claude-usage"], { timeout: 5000 });
			if (r.code === 0 && r.stdout) {
				cachedClaudeUsage = JSON.parse(r.stdout) as ClaudeUsageInfo;
				lastClaudeFetchTime = Date.now();
			}
		} catch {
			// Silencioso
		} finally {
			claudeInflight = false;
		}
		requestRender?.();
	}

	async function refreshQuotaIfNeeded(ctx: any, force = false) {
		if (!isAntigravity(ctx) || quotaInflight) {
			return;
		}
		const now = Date.now();
		if (!shouldRefreshQuota(now, lastQuotaFetchTime, force, !!cachedQuotaError, QUOTA_TTL_MS, QUOTA_ERROR_BACKOFF_MS)) {
			return;
		}

		quotaInflight = true;
		try {
			const r = await pi.exec("pi-gomemory-setup", ["usage", "--json"], { timeout: 10000 });
			lastQuotaFetchTime = Date.now();
			if (r.code === 0 && r.stdout) {
				const validated = validateQuotaSummary(JSON.parse(r.stdout));
				if (validated) {
					cachedQuota = validated;
					cachedQuotaError = undefined;
				} else {
					cachedQuotaError = "Formato de cuota inválido";
				}
			} else {
				cachedQuotaError = parseCleanQuotaError(r.stderr);
			}
		} catch (err: any) {
			lastQuotaFetchTime = Date.now();
			cachedQuotaError = err?.message || "No disponible";
		} finally {
			quotaInflight = false;
		}
		requestRender?.();
	}

	async function fetchTitle(ctx: any): Promise<string> {
		let version = (process.env.PI_GOMEMORY_RUNNING_VERSION || "").trim();
		if (!version) {
			try {
				const r = await pi.exec("pi-gomemory-setup", ["version"], { timeout: 2000 });
				if (r.code === 0) version = r.stdout.trim();
			} catch {
				// pi-gomemory-setup no disponible
			}
		}
		const suffix = version && version !== "dev" ? ` · ${version}` : "";
		return "HCD · " + (await fetchUnitTitle(ctx)) + suffix;
	}

	async function fetchUnitTitle(ctx: any): Promise<string> {
		try {
			let workingDir: string | undefined;
			try {
				workingDir = ctx?.cwd;
			} catch {
				workingDir = process.cwd();
			}
			const r = await pi.exec("mem", ["hcd", "show", "--json"], { cwd: workingDir || process.cwd(), timeout: 3000 });
			if (r.code === 0 && r.stdout) {
				const u = JSON.parse(r.stdout);
				if (u.unit_id) {
					return `${u.unit_id} · ${u.kind} · paso ${u.stage}`;
				}
			}
		} catch {
			// mem hcd no disponible
		}
		return "pi-gomemory";
	}

	async function openSidebar(ctx: ExtensionCommandContext) {
		if (!ctx.hasUI) return;
		activeCtx = ctx;
		cachedTitle = await fetchTitle(ctx);
		void refreshQuotaIfNeeded(ctx, true);
		void refreshClaudeUsageIfNeeded(ctx, true);

		visible = true;

		// Heartbeat periódico cada 30s mientras el panel está visible
		if (autoRefreshTimer) {
			clearInterval(autoRefreshTimer);
		}
		autoRefreshTimer = setInterval(async () => {
			if (!visible) return;
			try {
				// Heartbeat forzado para mantener actualizados los % y tiempos de reinicio sin bloqueo de TTL
				await refreshQuotaIfNeeded(activeCtx || ctx, true);
				await refreshClaudeUsageIfNeeded(activeCtx || ctx, true);
			} catch {
				if (autoRefreshTimer) {
					clearInterval(autoRefreshTimer);
					autoRefreshTimer = null;
				}
			}
		}, HEARTBEAT_INTERVAL_MS);

		// nonCapturing: el panel flota sin quitar el teclado al editor. No se espera
		// la promesa para no bloquear el comando; se cierra con /sidebar.
		const dock = panelDock();
		void ctx.ui.custom<void>(
			(tui, theme, _kb, done) => {
				// El dock se libera antes de done(): pi-tui redibuja en nextTick, antes que el finally.
				closeSidebar = () => {
					dock.sidebarVisible = false;
					done();
				};
				requestRender = (force = false) => tui.requestRender(force);
				const publishHeight = (lines: string[]) => {
					if (dock.sidebarHeight !== lines.length) {
						dock.sidebarHeight = lines.length;
						tui.requestRender(); // el TDD lee la altura nueva en el siguiente frame
					}
					return lines;
				};
				return {
					render(width: number): string[] {
						const metrics = collectContextMetrics(ctx);
						const servers = getRegisteredMCPServers();
						const antigravityActive = isAntigravity(ctx);
						const claudeActive = isClaude(ctx);
						const providerName = formatProviderDisplayName(ctx);
						const state: SidebarState = {
							title: cachedTitle,
							tokens: metrics.tokens,
							contextLimit: metrics.contextLimit,
							cost: metrics.cost,
							servers,
							providerName,
							isAntigravityProvider: antigravityActive,
							quota: cachedQuota,
							quotaError: cachedQuotaError,
							isClaudeProvider: claudeActive,
							claudeUsage: cachedClaudeUsage,
							anthropicPlan: cachedAnthropicPlan,
							anthropicPlanError: cachedAnthropicPlanError,
						};
						return publishHeight(
							overlayMinimized
								? renderSidebarMinimized(state, width, theme)
								: renderSidebar(state, width, theme),
						);
					},
					handleMouse(event: any) {
						const action = headerClickToggles(event, overlayMinimized);
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
				overlayOptions: {
					anchor: "top-right",
					width: OVERLAY_WIDTH,
					margin: { top: 1, right: 1 },
					visible: (w) => {
						const shown = w >= 100;
						if (dock.sidebarVisible !== shown) {
							dock.sidebarVisible = shown;
							requestRender?.(); // el TDD se recoloca en el siguiente frame
						}
						return shown;
					},
					nonCapturing: true,
				},
			},
		).finally(() => {
			dock.sidebarVisible = false;
			if (autoRefreshTimer) {
				clearInterval(autoRefreshTimer);
				autoRefreshTimer = null;
			}
			closeSidebar = null;
			requestRender = null;
			visible = false;
		});
	}

	pi.registerCommand("sidebar", {
		description: "Muestra u oculta el panel lateral (/sidebar min o /sidebar m compacta a una fila)",
		handler: async (args, ctx) => {
			const sub = (args || "").trim().split(/\s+/)[0];
			if (sub === "min" || sub === "m" || sub === "-" || sub === "+") {
				overlayMinimized = !overlayMinimized;
				if (!visible) {
					await openSidebar(ctx);
				} else if (requestRender) {
					requestRender(true);
				}
				ctx.ui?.notify?.(`Panel lateral ${overlayMinimized ? "minimizado" : "restaurado"}`, "info");
				return;
			}
			if (visible && closeSidebar) {
				closeSidebar();
				ctx.ui?.notify?.("Panel lateral oculto", "info");
				return;
			}
			ctx.ui?.notify?.("Panel lateral visible (/sidebar para ocultarlo)", "info");
			await openSidebar(ctx);
		},
	});

	pi.registerCommand("min", {
		description: "Minimiza o restaura a la vez /sidebar y /tdd (/min)",
		handler: async (_args, ctx) => {
			// El TDD publica sus controles en el dock (otra extensión, otro módulo).
			const tdd = panelDock().tdd;
			const tddShown = Boolean(tdd?.isVisible());
			// Si alguno está expandido se minimizan todos; si todos están minimizados se restauran.
			const anyExpanded = !overlayMinimized || (tddShown && !tdd.isMinimized());
			const minimize = anyExpanded;

			overlayMinimized = minimize;
			if (!visible) {
				await openSidebar(ctx);
			} else if (requestRender) {
				requestRender(true);
			}
			if (tddShown) tdd.setMinimized(minimize);
			ctx.ui?.notify?.(`Paneles ${minimize ? "minimizados" : "restaurados"}`, "info");
		},
	});

	pi.on("session_start", async (_event: any, ctx: any) => {
		cachedTitle = await fetchTitle(ctx);
		void refreshQuotaIfNeeded(ctx);
		void refreshClaudeUsageIfNeeded(ctx);
	});

	pi.on("session_shutdown", async (_event: any, _ctx: any) => {
		if (autoRefreshTimer) {
			clearInterval(autoRefreshTimer);
			autoRefreshTimer = null;
		}
		if (closeSidebar) {
			closeSidebar();
			closeSidebar = null;
		}
		activeCtx = null;
		requestRender = null;
		visible = false;
	});

	pi.on("turn_end", async (_event: any, ctx: any) => {
		activeCtx = ctx;
		// Turno individual consumido: refresca con fuerza las cuotas y métricas de contexto
		cachedTitle = await fetchTitle(ctx);
		void refreshQuotaIfNeeded(ctx, true);
		void refreshClaudeUsageIfNeeded(ctx, true);
		requestRender?.();
	});

	pi.on("agent_end", async (_event: any, ctx: any) => {
		// Fin de corrida de agente: refresca con fuerza el balance final
		cachedTitle = await fetchTitle(ctx);
		void refreshQuotaIfNeeded(ctx, true);
		void refreshClaudeUsageIfNeeded(ctx, true);
		requestRender?.();
	});

	pi.on("model_select", async (_event: any, ctx: any) => {
		void refreshQuotaIfNeeded(ctx, true);
		void refreshClaudeUsageIfNeeded(ctx, true);
		requestRender?.();
	});
}

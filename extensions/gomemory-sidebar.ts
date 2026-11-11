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

export interface SidebarState {
	title: string;
	tokens: number;
	contextLimit: number;
	cost: number;
	servers: MCPServerInfo[];
	quota?: QuotaSummaryInfo | null;
	quotaError?: string;
	isAntigravityProvider?: boolean;
	claudeUsage?: ClaudeUsageInfo | null;
	isClaudeProvider?: boolean;
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

	try {
		if (ctx.sessionManager?.getBranch) {
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
	} catch {
		// Silencioso
	}

	const contextLimit = ctx.model?.contextWindow || 200000;
	const activeTokens = lastTurnTokens > 0 ? lastTurnTokens : tokens;
	return { tokens: activeTokens, contextLimit, cost };
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

	// 1. Título principal
	const titleWords = state.title.replace(/\s+/g, " ").trim();
	const titleStyled = theme.style(titleWords || "pi-gomemory", { fg: "success", bold: true });
	lines.push(border("│") + padLine(` ${titleStyled}`) + border("│"));

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

	// 4. Sección Claude Code Usage (si el modelo o proveedor es Claude / Anthropic)
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

	lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));

	// 5. Sección MCP
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

	lines.push(border(`╰${"─".repeat(innerW)}╯`));
	return lines;
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
	let closeSidebar: (() => void) | null = null;
	let requestRender: (() => void) | null = null;
	let autoRefreshTimer: ReturnType<typeof setInterval> | null = null;
	let cachedTitle = "Instalación de binario con componentes";

	let cachedQuota: QuotaSummaryInfo | null = null;
	let cachedQuotaError: string | undefined = undefined;
	let lastQuotaFetchTime = 0;
	let quotaInflight = false;
	const QUOTA_TTL_MS = 60 * 1000;
	const QUOTA_ERROR_BACKOFF_MS = 30 * 1000;

	let cachedClaudeUsage: ClaudeUsageInfo | null = null;
	let lastClaudeFetchTime = 0;
	let claudeInflight = false;
	const CLAUDE_TTL_MS = 60 * 1000;

	function isAntigravity(ctx: any): boolean {
		const provider = (ctx?.model?.provider || "").toLowerCase();
		const modelId = (ctx?.model?.id || "").toLowerCase();
		return (
			provider.includes("antigravity") ||
			provider.includes("agy") ||
			modelId.includes("antigravity") ||
			modelId.includes("agy")
		);
	}

	function isClaude(ctx: any): boolean {
		const provider = (ctx?.model?.provider || "").toLowerCase();
		const modelId = (ctx?.model?.id || "").toLowerCase();
		return (
			provider.includes("anthropic") ||
			provider.includes("claude") ||
			modelId.includes("claude") ||
			provider.includes("claude-code")
		);
	}

	async function refreshClaudeUsageIfNeeded(ctx: any, force = false) {
		if (!isClaude(ctx) || claudeInflight) {
			return;
		}
		const now = Date.now();
		if (!force && cachedClaudeUsage && now - lastClaudeFetchTime < CLAUDE_TTL_MS) {
			return;
		}

		claudeInflight = true;
		try {
			const r = await pi.exec("pi-gomemory-setup", ["claude-usage"], { timeout: 5000 });
			if (r.code === 0 && r.stdout) {
				cachedClaudeUsage = JSON.parse(r.stdout) as ClaudeUsageInfo;
				lastClaudeFetchTime = now;
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
		const minInterval = cachedQuotaError ? QUOTA_ERROR_BACKOFF_MS : QUOTA_TTL_MS;
		if (!force && now - lastQuotaFetchTime < minInterval) {
			return;
		}

		quotaInflight = true;
		try {
			const r = await pi.exec("pi-gomemory-setup", ["usage", "--json"], { timeout: 10000 });
			lastQuotaFetchTime = now;
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
			lastQuotaFetchTime = now;
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
			const r = await pi.exec("mem", ["hcd", "show", "--json"], { cwd: ctx.cwd, timeout: 3000 });
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
		cachedTitle = await fetchTitle(ctx);
		void refreshQuotaIfNeeded(ctx, true);
		void refreshClaudeUsageIfNeeded(ctx, true);

		visible = true;

		// Timer de refresco periódico cuando el sidebar está abierto
		if (autoRefreshTimer) {
			clearInterval(autoRefreshTimer);
		}
		autoRefreshTimer = setInterval(() => {
			if (!visible) return;
			void refreshQuotaIfNeeded(ctx);
			void refreshClaudeUsageIfNeeded(ctx);
		}, 30 * 1000);

		// nonCapturing: el panel flota sin quitar el teclado al editor. No se espera
		// la promesa para no bloquear el comando; se cierra con /sidebar.
		void ctx.ui.custom<void>(
			(tui, theme, _kb, done) => {
				closeSidebar = done;
				requestRender = () => tui.requestRender();
				return {
					render(width: number): string[] {
						const metrics = collectContextMetrics(ctx);
						const servers = getRegisteredMCPServers();
						const antigravityActive = isAntigravity(ctx);
						const claudeActive = isClaude(ctx);
						const state: SidebarState = {
							title: cachedTitle,
							tokens: metrics.tokens,
							contextLimit: metrics.contextLimit,
							cost: metrics.cost,
							servers,
							isAntigravityProvider: antigravityActive,
							quota: cachedQuota,
							quotaError: cachedQuotaError,
							isClaudeProvider: claudeActive,
							claudeUsage: cachedClaudeUsage,
						};
						return renderSidebar(state, width, theme);
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
					visible: (w) => w >= 100,
					nonCapturing: true,
				},
			},
		).finally(() => {
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
		description: "Muestra u oculta el panel lateral de estado gomemory/OpenCode en la esquina superior derecha",
		handler: async (_args, ctx) => {
			if (visible && closeSidebar) {
				closeSidebar();
				ctx.ui?.notify?.("Panel lateral oculto", "info");
				return;
			}
			ctx.ui?.notify?.("Panel lateral visible (/sidebar para ocultarlo)", "info");
			await openSidebar(ctx);
		},
	});

	pi.on("session_start", async (_event: any, ctx: any) => {
		cachedTitle = await fetchTitle(ctx);
		void refreshQuotaIfNeeded(ctx);
		void refreshClaudeUsageIfNeeded(ctx);
	});

	pi.on("agent_end", async (_event: any, ctx: any) => {
		cachedTitle = await fetchTitle(ctx);
		void refreshQuotaIfNeeded(ctx);
		void refreshClaudeUsageIfNeeded(ctx);
		requestRender?.();
	});

	pi.on("model_select", async (_event: any, ctx: any) => {
		void refreshQuotaIfNeeded(ctx, true);
		void refreshClaudeUsageIfNeeded(ctx, true);
		requestRender?.();
	});
}

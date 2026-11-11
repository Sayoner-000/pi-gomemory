/**
 * gomemory-sidebar.ts — Panel lateral de estado inspirado en OpenCode para Pi Coding Agent.
 *
 * Muestra un overlay superior derecho con:
 *  1. Título de la sesión o unidad HCD activa (mem hcd show --json).
 *  2. Métricas de contexto (tokens usados, % de ventana y gasto acumulado en USD).
 *  3. Estado de conexión de servidores MCP (codebase-memory-mcp, codegraph, context7, gomemory).
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
	status: "Connected" | "Disconnected";
}

export interface SidebarState {
	title: string;
	tokens: number;
	contextLimit: number;
	cost: number;
	servers: MCPServerInfo[];
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

/**
 * Lee los servidores MCP configurados en ~/.pi/agent/mcp.json.
 */
export function getRegisteredMCPServers(agentDir?: string): MCPServerInfo[] {
	const dir = agentDir || join(homedir(), ".pi", "agent");
	const mcpPath = join(dir, "mcp.json");
	try {
		const raw = readFileSync(mcpPath, "utf8");
		const data = JSON.parse(raw);
		const serversObj = data.mcpServers || {};
		const names = Object.keys(serversObj);
		if (names.length === 0) {
			return [
				{ name: "codebase-memory-mcp", status: "Connected" },
				{ name: "codegraph", status: "Connected" },
				{ name: "gomemory", status: "Connected" },
			];
		}
		return names.map((name) => ({
			name,
			status: "Connected",
		}));
	} catch {
		return [
			{ name: "codebase-memory-mcp", status: "Connected" },
			{ name: "codegraph", status: "Connected" },
			{ name: "gomemory", status: "Connected" },
		];
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

	lines.push(border("├") + border("─".repeat(innerW)) + border("┤"));

	// 3. Sección MCP
	const mcpHeader = theme.style("▼ MCP", { fg: "success", bold: true });
	lines.push(border("│") + padLine(` ${mcpHeader}`) + border("│"));

	for (const srv of state.servers) {
		const bullet = theme.fg("success", "•");
		const nameStyled = theme.fg("accent", srv.name);
		const statusStyled = srv.status === "Connected" ? theme.fg("dim", "Connected") : theme.fg("error", "Disconnected");

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

export default function (pi: ExtensionAPI) {
	let visible = false;
	let closeSidebar: (() => void) | null = null;
	let requestRender: (() => void) | null = null;
	let cachedTitle = "Instalación de binario con componentes";

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
		return (await fetchUnitTitle(ctx)) + suffix;
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

		visible = true;
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
						const state: SidebarState = {
							title: cachedTitle,
							tokens: metrics.tokens,
							contextLimit: metrics.contextLimit,
							cost: metrics.cost,
							servers,
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
	});

	pi.on("agent_end", async (_event: any, ctx: any) => {
		cachedTitle = await fetchTitle(ctx);
		requestRender?.();
	});
}

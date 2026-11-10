/**
 * gomemory-status.ts — TUI Status Bar & Token Usage Monitor para Pi Coding Agent
 *
 * Emula la estética y métricas de la TUI de gomemory y la barra de consumo de tokens
 * (estilo claude-token-usage de Sayoner-000) adaptada al ecosistema de Pi.
 */

import type { AssistantMessage } from "@earendil-works/pi-ai";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { truncateToWidth, visibleWidth } from "@earendil-works/pi-tui";

export default function (pi: ExtensionAPI) {
	let customFooterEnabled = true;

	function setupFooter(ctx: any) {
		if (!ctx.ui || !customFooterEnabled) return;

		ctx.ui.setFooter((tui: any, theme: any, footerData: any) => {
			const unsubBranch = footerData.onBranchChange ? footerData.onBranchChange(() => tui.requestRender()) : () => {};

			return {
				dispose: unsubBranch,
				invalidate() {},
				render(width: number): string[] {
					// 1. Recopilar métricas de tokens y coste del árbol de sesión actual
					let input = 0;
					let output = 0;
					let cacheRead = 0;
					let cost = 0;
					let lastTurnTokens = 0;

					try {
						for (const entry of ctx.sessionManager.getBranch()) {
							if (entry.type === "message" && entry.message.role === "assistant") {
								const msg = entry.message as AssistantMessage;
								if (msg.usage) {
									input += msg.usage.input || 0;
									output += msg.usage.output || 0;
									cacheRead += (msg.usage as any).cacheRead || (msg.usage as any).cache_read_input_tokens || 0;
									cost += msg.usage.cost?.total || 0;
									lastTurnTokens = (msg.usage.input || 0) + (msg.usage.output || 0);
								}
							}
						}
					} catch (_e) {
						// Fallback silencioso ante sesion en transición
					}

					// 2. Cálculo de ventana de contexto
					const contextLimit = ctx.model?.contextWindow || 200000;
					// El context usage aproximado activo corresponde al último input total + output
					const currentContextTokens = lastTurnTokens > 0 ? lastTurnTokens : input;
					const usageRatio = Math.min(1.0, Math.max(0.0, currentContextTokens / contextLimit));
					const pct = Math.round(usageRatio * 100);

					// Formateo de números compactos
					const fmt = (n: number) => {
						if (n < 1000) return `${n}`;
						if (n < 1000000) return `${(n / 1000).toFixed(1)}k`;
						return `${(n / 1000000).toFixed(2)}M`;
					};

					// 3. Barra visual gráfica tipo claude-token-usage
					const barLength = width > 100 ? 12 : 8;
					const filled = Math.min(barLength, Math.max(0, Math.round(usageRatio * barLength)));
					const empty = barLength - filled;
					const barVisual = "█".repeat(filled) + "░".repeat(empty);

					// Colores temáticos gomemory
					let barColorRole: "success" | "warning" | "error" = "success";
					if (usageRatio >= 0.8) {
						barColorRole = "error";
					} else if (usageRatio >= 0.5) {
						barColorRole = "warning";
					}

					// 4. Componentes de la barra de estado
					// Logo / Badge gomemory en Brillo/Cian
					const logo = theme.style("⟦mem⟧", { fg: "accent", bold: true });

					// Barra coloreada con porcentaje
					const barStyled = theme.style(`[${barVisual}] ${pct}%`, { fg: barColorRole, bold: true });
					const ctxText = theme.fg("dim", `(${fmt(currentContextTokens)}/${fmt(contextLimit)})`);

					// Desglose de tokens
					const inText = theme.fg("muted", `↑${fmt(input)}`);
					const outText = theme.fg("muted", `↓${fmt(output)}`);
					const cacheText = cacheRead > 0 ? theme.fg("dim", `⚡${fmt(cacheRead)} `) : "";
					const costText = cost > 0 ? theme.style(`$${cost.toFixed(3)}`, { fg: "warning" }) : theme.fg("dim", "$0.000");

					const left = `${logo} ${barStyled} ${ctxText} │ ${inText} ${outText} ${cacheText}│ ${costText}`;

					// Lado derecho: modelo, rama git y thinking
					const branch = footerData.getGitBranch ? footerData.getGitBranch() : "";
					const branchStr = branch ? ` (${branch})` : "";
					const modelStr = ctx.model?.id || "default";
					const thinkingStr = ctx.thinkingLevel && ctx.thinkingLevel !== "off" ? ` [🧠 ${ctx.thinkingLevel}]` : "";
					const right = theme.fg("dim", `${modelStr}${thinkingStr}${branchStr}`);

					// Padding y truncado seguro
					const leftLen = visibleWidth(left);
					const rightLen = visibleWidth(right);
					const gap = Math.max(1, width - leftLen - rightLen);
					const pad = " ".repeat(gap);

					return [truncateToWidth(left + pad + right, width)];
				},
			};
		});
	}

	// Activar footer en arranque de sesión
	pi.on("session_start", async (_event, ctx) => {
		setupFooter(ctx);
	});

	// Comando para alternar la barra de tokens y TUI gomemory
	pi.registerCommand("tokens", {
		description: "Alternar la barra de consumo de tokens y estado gomemory en el footer",
		handler: async (_args, ctx) => {
			customFooterEnabled = !customFooterEnabled;
			if (customFooterEnabled) {
				setupFooter(ctx);
				ctx.ui.notify("Barra de estado y tokens gomemory activada", "info");
			} else {
				ctx.ui.setFooter(undefined);
				ctx.ui.notify("Barra por defecto de Pi restaurada", "info");
			}
		},
	});
}

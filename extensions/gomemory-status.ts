/**
 * gomemory-status.ts — TUI Status Bar & Token Usage Monitor para Pi Coding Agent
 *
 * Emula la estética y métricas de la TUI de gomemory y la barra de consumo de tokens
 * (estilo claude-token-usage de Sayoner-000) adaptada al ecosistema de Pi.
 */

import type { AssistantMessage } from "@earendil-works/pi-ai";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { truncateToWidth, visibleWidth } from "@earendil-works/pi-tui";

/** Aviso de actualizaciones del meta-harness (HCD-029); la instalación sigue en el launcher. */
export const UPDATES_LABEL = "↑ actualizaciones disponibles";
const UPDATES_STATUS_KEY = "gomemory-updates";
/** `pi-gomemory-setup deps-status` sale con 10 cuando hay algo desactualizado. */
const DEPS_OUTDATED_EXIT = 10;

/** `pi-gomemory-setup skills-sync --check` sale con 10 si hay skills desfasadas (HCD-080). */
const SKILLS_DRIFT_EXIT = 10;
/** Un bash que actualizó gomemory: sus skills pueden haber cambiado (HCD-080). */
const MEM_UPDATE_RE = /(^|[\s;&|(])mem\s+update\b/;

/**
 * Lee la salida de `skills-sync --check`: las desfasadas que la sincronización
 * repone y las editadas localmente que no toca (Q1), cada grupo en su línea.
 */
export function skillsCheckReport(stderr: string): { drifted: string[]; edited: string[] } {
	const names = (re: RegExp) => {
		const m = re.exec(stderr ?? "");
		return m ? m[1].split(",").map((s) => s.trim()).filter(Boolean) : [];
	};
	return {
		drifted: names(/desfasadas en [^:\n]*:\s*([^\n]*?)\.\s*Ejecuta/),
		edited: names(/editadas localmente[^:\n]*:\s*([^\n]*?)\.\s*Ejecuta/),
	};
}

/** Plazo de `/componentes` para `sync`: hasta cinco instaladores de 10 min (HCD-076, N4). */
export const COMPONENTS_SYNC_TIMEOUT_MS = 60 * 60 * 1000;

type ComponentRow = {
	name: string;
	kind: "self" | "tool" | "package";
	required: boolean;
	installed: boolean;
	current: string;
	latest: string;
	outdated: boolean;
	error: string;
};
type SyncRow = { name: string; action: string; from: string; to: string; error?: string };

function componentVersion(c: ComponentRow): string {
	if (!c.installed) return c.latest ? `no instalado → ${c.latest}` : "no instalado";
	if (c.outdated) return `${c.current} → ${c.latest}`;
	return c.current || "versión desconocida";
}

const SYNC_ACTIONS: Record<string, string> = {
	installed: "instalado",
	updated: "actualizado",
	current: "al día",
	failed: "falló",
	skipped: "omitido",
};

/** Etiqueta para bloques de razonamiento colapsados (HCD-066, 2A). */
export const HIDDEN_THINKING_LABEL = "(thinking...)";
/** Ola de gotitas en glifos de texto puro (sin emojis, HCD-020 Q3 / HCD-066 2A). */
export const THINKING_DROPLET_FRAMES = ["·   ", "•·  ", "●•· ", " •●•", "  ·●", "   ·"] as const;
export const THINKING_INDICATOR_INTERVAL_MS = 110;

export function buildThinkingIndicator(theme?: { fg?: (role: string, text: string) => string }) {
	const color = (text: string) => (theme?.fg ? theme.fg("accent", text) : text);
	return {
		frames: THINKING_DROPLET_FRAMES.map(color),
		intervalMs: THINKING_INDICATOR_INTERVAL_MS,
	};
}

/**
 * Misma regla que `pi-gomemory-setup` y el launcher (HCD-037): vacía, "0" o
 * "false" (sin distinguir mayúsculas ni espacios) dejan la comprobación activa;
 * cualquier otro valor la desactiva.
 */
export function updateCheckDisabled(env: NodeJS.ProcessEnv = process.env): boolean {
	const v = (env.PI_GOMEMORY_NO_UPDATE_CHECK ?? "").trim().toLowerCase();
	return v !== "" && v !== "0" && v !== "false";
}

export default function (pi: ExtensionAPI) {
	let customFooterEnabled = true;
	let updatesAvailable = false;
	let requestRender: (() => void) | null = null;
	let activeUi: any = null;

	function setupThinkingIndicator(ctx: any) {
		if (!ctx?.ui) return;
		activeUi = ctx.ui;
		ctx.ui.setHiddenThinkingLabel?.(HIDDEN_THINKING_LABEL);
		ctx.ui.setWorkingIndicator?.(buildThinkingIndicator(ctx.ui.theme));
	}

	/** Publica el aviso: en el pie propio si está activo; si no, en el pie por defecto con setStatus. */
	function showUpdates(ctx: any, available: boolean) {
		updatesAvailable = available;
		ctx.ui?.setStatus?.(UPDATES_STATUS_KEY, !customFooterEnabled && available ? UPDATES_LABEL : undefined);
		requestRender?.();
	}

	/**
	 * Consulta el estado de la última comprobación (rápido, local), refresca en
	 * segundo plano (red, respeta la caché de 24 h) y vuelve a consultar: una
	 * release nueva aparece en la misma apertura sin retrasar el arranque.
	 */
	async function checkUpdates(ctx: any) {
		if (updateCheckDisabled()) return;
		const outdated = async () => {
			try {
				const r = await pi.exec("pi-gomemory-setup", ["deps-status"], { timeout: 10_000 });
				return r.code === DEPS_OUTDATED_EXIT;
			} catch {
				return false; // setup ausente o con error: no se afirma nada
			}
		};
		showUpdates(ctx, await outdated());
		try {
			await pi.exec("pi-gomemory-setup", ["deps-refresh"], { timeout: 60_000 });
		} catch {
			// sin red: se mantiene el resultado anterior
		}
		showUpdates(ctx, await outdated());
	}

	/**
	 * HCD-080: si `mem update` cambió las skills de gomemory, el repositorio que
	 * carga Pi (~/.pi-gomemory/skills) queda atrás hasta el siguiente install.
	 * Comprueba sin escribir y, solo si hay deriva, sincroniza y avisa: Pi carga
	 * las skills al arrancar, así que la versión nueva llega con /reload.
	 */
	let skillsCheck: Promise<void> | null = null;
	function syncSkills(ctx: any): Promise<void> {
		if (skillsCheck) return skillsCheck;
		skillsCheck = (async () => {
			try {
				const r = await pi.exec("pi-gomemory-setup", ["skills-sync", "--check"], { timeout: 10_000 });
				if (r.code !== SKILLS_DRIFT_EXIT) return; // al día, o setup antiguo sin --check
				const { drifted, edited } = skillsCheckReport(r.stderr);
				// Sin nombres reconocibles (setup de otra versión) se sincroniza igual.
				if (drifted.length || !edited.length) {
					const s = await pi.exec("pi-gomemory-setup", ["skills-sync"], { timeout: 30_000 });
					const list = drifted.length ? ` (${drifted.join(", ")})` : "";
					if (s.code === 0) {
						ctx.ui?.notify?.(`Skills de gomemory actualizadas desde mem${list}. Usa /reload para cargarlas.`, "info");
					} else {
						ctx.ui?.notify?.(`Skills de gomemory desfasadas${list}; no se pudieron sincronizar. Ejecuta pi-gomemory-setup skills-sync.`, "warning");
					}
				}
				// Las editadas a mano no se pisan: se avisa cómo reponerlas (Q1).
				if (edited.length) {
					ctx.ui?.notify?.(
						`Skills de gomemory editadas localmente y desfasadas respecto a mem: ${edited.join(", ")}. Para recibir la versión nueva: pi-gomemory-setup skills-sync --restore ${edited.join(" ")}`,
						"warning",
					);
				}
			} catch {
				// setup ausente: no se afirma nada
			} finally {
				skillsCheck = null;
			}
		})();
		return skillsCheck;
	}

	/**
	 * HCD-084: instala o actualiza impeccable y frontend-design desde skills.sh
	 * (`skills-update --if-due`, una vez cada 24 h). Si algo cambió, el setup lo
	 * dice en stdout y se avisa: Pi carga las skills al arrancar (/reload).
	 */
	async function updateExternalSkills(ctx: any): Promise<void> {
		try {
			const r = await pi.exec("pi-gomemory-setup", ["skills-update", "--if-due"], { timeout: 600_000 });
			const msg = String(r.stdout ?? "").trim();
			if (r.code === 0 && msg) ctx.ui?.notify?.(msg, "info");
		} catch {
			// setup ausente, antiguo o sin red: no se afirma nada
		}
	}

	function setupFooter(ctx: any) {
		if (!ctx.ui || !customFooterEnabled) return;

		ctx.ui.setFooter((tui: any, theme: any, footerData: any) => {
			requestRender = () => tui.requestRender();
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
									const u = msg.usage as any;
									const cr = u.cacheRead || u.cache_read_input_tokens || 0;
									const cw = u.cacheWrite || u.cache_creation_input_tokens || 0;
									input += msg.usage.input || 0;
									output += msg.usage.output || 0;
									cacheRead += cr;
									cost += msg.usage.cost?.total || 0;
									const parts = (msg.usage.input || 0) + (msg.usage.output || 0) + cr + cw;
									const turn = parts > 0 ? parts : (u.totalTokens || 0);
									if (turn > 0) lastTurnTokens = turn;
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
					// Glifos de texto, sin emoji (criterio de HCD-020 Q3): ↺ = lecturas de caché.
					const cacheText = cacheRead > 0 ? theme.fg("dim", `↺${fmt(cacheRead)} `) : "";
					const costText = cost > 0 ? theme.style(`$${cost.toFixed(3)}`, { fg: "warning" }) : theme.fg("dim", "$0.000");

					const left = `${logo} ${barStyled} ${ctxText} │ ${inText} ${outText} ${cacheText}│ ${costText}`;

					// Lado derecho: modelo, rama git y thinking
					const branch = footerData.getGitBranch ? footerData.getGitBranch() : "";
					const branchStr = branch ? ` (${branch})` : "";
					const modelStr = ctx.model?.id || "default";
					const thinkingStr = ctx.thinkingLevel && ctx.thinkingLevel !== "off" ? ` [thinking ${ctx.thinkingLevel}]` : "";
					const updates = updatesAvailable ? theme.style(UPDATES_LABEL, { fg: "warning", bold: true }) + "  " : "";
					const right = updates + theme.fg("dim", `${modelStr}${thinkingStr}${branchStr}`);

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

	// Activar footer e indicador de thinking en arranque de sesión
	pi.on("session_start", async (_event, ctx) => {
		setupFooter(ctx);
		setupThinkingIndicator(ctx);
		// Sin esperar: la consulta en red nunca retrasa el arranque de la sesión.
		if (ctx.ui) void checkUpdates(ctx);
		// Local y rápido, pero tampoco retrasa el arranque (HCD-080). Con
		// PI_GOMEMORY_NO_UPDATE_CHECK no se consulta nada (HCD-037, C-02).
		// HCD-084: después, las skills externas (skills.sh) en segundo plano,
		// como mucho una vez cada 24 h; no se solapa con la sincronización local.
		if (!updateCheckDisabled()) void syncSkills(ctx).then(() => updateExternalSkills(ctx));
	});

	// HCD-080: tras un `mem update` desde Pi, las skills de gomemory pueden haber cambiado.
	pi.on("tool_result", async (event: any, ctx: any) => {
		if (event?.toolName !== "bash" || event?.isError || updateCheckDisabled()) return;
		if (MEM_UPDATE_RE.test(String(event?.input?.command ?? ""))) void syncSkills(ctx);
	});

	pi.on("session_shutdown", async (_event, ctx) => {
		const ui = ctx?.ui ?? activeUi;
		ui?.setWorkingIndicator?.();
		ui?.setHiddenThinkingLabel?.();
		activeUi = null;
		requestRender = null;
	});

	// HCD-076: ver y reconciliar los componentes del meta-harness desde Pi.
	pi.registerCommand("componentes", {
		description: "Ver y actualizar los componentes del meta-harness (pi, mem, codegraph, herdr)",
		handler: async (_args, ctx) => {
			let rows: ComponentRow[];
			try {
				const r = await pi.exec("pi-gomemory-setup", ["components", "--json"], { timeout: 120_000 });
				if (r.code !== 0) throw new Error(r.stderr.trim() || `código ${r.code}`);
				rows = JSON.parse(r.stdout);
			} catch (e) {
				ctx.ui.notify(`No se pudo leer el estado de los componentes: ${(e as Error).message}`, "error");
				return;
			}
			const tools = rows.filter((c) => c.kind === "tool");
			const others = rows.filter((c) => c.kind !== "tool");
			const selected = new Map(tools.map((c) => [c.name, true]));
			const APPLY = "✓ Aplicar (instala lo que falta y actualiza lo marcado)";
			const CANCEL = "✗ Cancelar";
			const toolLabel = (c: ComponentRow) =>
				`${selected.get(c.name) ? "[x]" : "[ ]"} ${c.name} ${componentVersion(c)} · ${c.required ? "obligatorio" : "opcional"}`;
			const otherLabel = (c: ComponentRow) =>
				`· ${c.name} ${componentVersion(c)} · ${c.kind === "self" ? "se actualiza con pi-gomemory-setup update" : "paquete fijado (install)"}`;
			for (;;) {
				const toolOpts = tools.map(toolLabel);
				const options = [...toolOpts, ...others.map(otherLabel), APPLY, CANCEL];
				const choice = await ctx.ui.select("Componentes del meta-harness: elige para marcar o desmarcar", options);
				if (!choice || choice === CANCEL) return;
				if (choice === APPLY) break;
				const i = toolOpts.indexOf(choice);
				if (i < 0) continue;
				const c = tools[i];
				if (c.required) {
					ctx.ui.notify(`${c.name} es obligatorio: no se puede desmarcar`, "warning");
					continue;
				}
				selected.set(c.name, !selected.get(c.name));
			}
			const only = tools.filter((c) => selected.get(c.name)).map((c) => c.name);
			ctx.ui.notify(`Reconciliando ${only.join(", ")}…`, "info");
			const r = await pi.exec("pi-gomemory-setup", ["sync", "--json", `--only=${only.join(",")}`], {
				timeout: COMPONENTS_SYNC_TIMEOUT_MS,
			});
			if (r.killed) {
				ctx.ui.notify("sync no terminó en el plazo; ejecuta pi-gomemory-setup sync en la terminal", "error");
				return;
			}
			let results: SyncRow[] = [];
			try {
				results = JSON.parse(r.stdout || "[]");
			} catch {
				// sin JSON: se informa el código y stderr abajo
			}
			if (results.length > 0) {
				const summary = results
					.map((s) => `${s.name} ${SYNC_ACTIONS[s.action] ?? s.action}${s.action === "updated" ? ` (${s.from} → ${s.to})` : ""}`)
					.join("; ");
				ctx.ui.notify(summary, results.some((s) => s.action === "failed") ? "warning" : "info");
			}
			if (r.code !== 0) {
				const tail = r.stderr.trim().split("\n").slice(-3).join(" ");
				ctx.ui.notify(`sync terminó con código ${r.code}: ${tail}`, "error");
			}
			// pi se reconcilia siempre (obligatorio): el aviso depende de lo que pasó.
			if (results.some((s) => s.name === "pi" && (s.action === "updated" || s.action === "installed"))) {
				ctx.ui.notify("Reinicia Pi para usar la versión nueva de pi.", "warning");
			}
		},
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
				requestRender = null;
				ctx.ui.notify("Barra por defecto de Pi restaurada", "info");
			}
			showUpdates(ctx, updatesAvailable);
		},
	});
}

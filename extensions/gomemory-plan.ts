/**
 * gomemory-plan.ts — Modo /plan de pi-gomemory.
 *
 * /plan alterna un modo que bloquea las modificaciones comunes (edit, write y
 * comandos bash de gomemory-plan-patterns.json) e inyecta como sección del system
 * prompt (`gomemory-plan`) el método de descomposición atómica de `mem plan-context`.
 * /plan on y /plan off fijan el estado; /plan status lo consulta. /plan run
 * solo desactiva. /plan <texto> activa el modo y envía el texto como solicitud.
 */

import { readFileSync } from "node:fs";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

interface Patterns {
	tools: string[];
	strip: string[];
	block: string[];
}

const patterns: Patterns = JSON.parse(
	readFileSync(new URL("./gomemory-plan-patterns.json", import.meta.url), "utf8"),
);
const strip = patterns.strip.map((s) => new RegExp(s, "g"));
const block = patterns.block.map((s) => new RegExp(s));

export function bashBlocked(command: string): boolean {
	let cmd = command;
	for (const re of strip) cmd = cmd.replace(re, " ");
	return block.some((re) => re.test(cmd));
}

export const IMPACT_PROTOCOL = `## Validación obligatoria de impacto antes de HCD (solo lectura)

Antes de proponer el árbol de tareas o abrir una unidad HCD, valida el impacto completo del proyecto sin modificar archivos del repositorio:
1. **codegraph**: si el proyecto tiene \`.codegraph/\`, consulta \`codegraph_explore\` sobre los símbolos y archivos implicados para inspeccionar fuentes y llamadas dinámicas. Si no existe \`.codegraph/\`, decláralo sin ejecutar \`codegraph index\`.
2. **codebase-memory**: verifica el proyecto y su frescura con \`list_projects\` e \`index_status\`. Si el índice falta o está desactualizado, puedes refrescar el índice externo con \`index_repository(mode="moderate", persistence=false)\` (no escribe archivos en el repositorio; nunca uses \`persistence=true\` en modo plan).
3. **Radio de impacto**: ejecuta \`search_graph\`, \`trace_path\` (\`direction="both"\` para llamadores inbound y dependencias outbound) y \`check_index_coverage\` sobre todas las rutas candidatas antes de afirmar qué archivos o contratos se ven afectados.
4. **Entregable**: incluye una sección explícita **«Mapa de impacto»** (símbolos, llamadores, archivos afectados y estado/huecos de cobertura) antes del árbol de tareas atómicas. No modifiques código ni abras/mutes unidades HCD mientras el modo plan esté activo.`;

const FALLBACK = `[gomemory-plan:fallback]
Estás en MODO PLAN (solo planificación). No modifiques archivos ni ejecutes comandos que cambien estado.
Descompón la solicitud en un árbol de tareas atómicas: cada hoja con un verbo, un objeto, un único
resultado inspeccionable y la comprobación que la demuestra. Explora con herramientas de lectura,
entrega el árbol y pregunta si se ejecuta. La persona sale del modo con /plan off.

${IMPACT_PROTOCOL}`;

export const PLAN_SECTION = "gomemory-plan";

const REASON = "Modo plan activo: no se permiten modificaciones. Usa /plan off para salir.";
const PERSISTENCE_REASON =
	"Modo plan activo: index_repository con persistence=true escribe en el repositorio. Usa persistence=false para refrescar solo el índice externo.";
const INACTIVE = "El modo plan no está activo";

const BLOCKED_MCP_TOOLS = new Set(["delete_project", "mcp__codebase_memory__delete_project"]);

export function toolBlocked(toolName: string, input: any): string | undefined {
	if (patterns.tools.includes(toolName) || BLOCKED_MCP_TOOLS.has(toolName)) {
		return REASON;
	}
	if (toolName === "bash" && bashBlocked(String(input?.command ?? ""))) {
		return REASON;
	}
	if ((toolName === "index_repository" || toolName === "mcp__codebase_memory__index_repository") && input?.persistence === true) {
		return PERSISTENCE_REASON;
	}
	if (
		(toolName === "manage_adr" || toolName === "mcp__codebase_memory__manage_adr") &&
		(input?.mode === "update" || input?.mode === "set_sections")
	) {
		return REASON;
	}
	return undefined;
}

export default function (pi: ExtensionAPI) {
	// La promesa existe solo mientras el modo está activo. Su resolución no cambia el estado.
	let method: Promise<string> | undefined;

	async function loadMethod(cwd: string): Promise<string> {
		try {
			const r = await pi.exec("mem", ["plan-context"], { cwd, timeout: 5000 });
			const out = (r.stdout ?? "").trim();
			if (r.code === 0 && out !== "") {
				return `Estás en MODO PLAN: no modifiques archivos; valida el impacto completo del proyecto en solo lectura, entrega el árbol de tareas y pregunta si se ejecuta.\n\n${IMPACT_PROTOCOL}\n\n${out}`;
			}
		} catch {
			// mem ausente o con error: se usa el respaldo
		}
		return FALLBACK;
	}

	pi.registerCommand("plan", {
		description: "Modo plan de gomemory: on | off | status | run | <petición>",
		handler: async (args, ctx) => {
			const arg = String(args ?? "").trim();
			if (arg === "status") {
				ctx.ui.notify(method ? "Modo plan activo" : INACTIVE, "info");
				return;
			}
			if (arg === "off" || arg === "run" || (arg === "" && method)) {
				if (!method) {
					ctx.ui.notify(INACTIVE, "info");
					return;
				}
				method = undefined;
				ctx.ui.setStatus("gomemory-plan", undefined);
				ctx.ui.notify(arg === "run"
					? "Modo plan desactivado: ya se permiten modificaciones"
					: "Modo plan desactivado", "info");
				return;
			}
			if (!method) {
				method = loadMethod(ctx.cwd);
				ctx.ui.setStatus("gomemory-plan", "PLAN");
				ctx.ui.notify("Modo plan activo: edit, write y comandos que modifican están bloqueados", "info");
			} else if (arg === "on") {
				ctx.ui.notify("Modo plan activo", "info");
			}
			if (arg === "" || arg === "on") return;
			// /plan <texto>: activa el modo y envía la solicitud.
			if (ctx.isIdle?.() === false) {
				pi.sendUserMessage(arg, { deliverAs: "followUp" });
			} else {
				pi.sendUserMessage(arg);
			}
		},
	});

	pi.on("tool_call", async (event: any) => {
		if (!method) return;
		const reason = toolBlocked(String(event?.toolName ?? ""), event?.input ?? event?.args);
		if (reason) return { block: true, reason };
	});

	// El método va como sección del prompt (HCD-045, Q5): reemplazar systemPrompt
	// tapaba la sección de gomemory y rompía la caché del prompt en cada cambio.
	pi.on("before_agent_start", async (event: any) => {
		const opts = event?.systemPromptOptions;
		if (!opts) return;
		opts.sections ??= {};
		while (method) {
			const pending = method;
			const text = await pending;
			// El modo pudo cambiar mientras mem cargaba el método.
			if (method !== pending) continue;
			opts.sections[PLAN_SECTION] = text;
			return;
		}
		delete opts.sections[PLAN_SECTION];
	});
}

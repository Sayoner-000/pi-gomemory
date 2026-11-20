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

const FALLBACK = `[gomemory-plan:fallback]
Estás en MODO PLAN (solo planificación). No modifiques archivos ni ejecutes comandos que cambien estado.
Descompón la solicitud en un árbol de tareas atómicas: cada hoja con un verbo, un objeto, un único
resultado inspeccionable y la comprobación que la demuestra. Explora con herramientas de lectura,
entrega el árbol y pregunta si se ejecuta. La persona sale del modo con /plan off.`;

export const PLAN_SECTION = "gomemory-plan";

const REASON = "Modo plan activo: no se permiten modificaciones. Usa /plan off para salir.";
const INACTIVE = "El modo plan no está activo";

export default function (pi: ExtensionAPI) {
	// La promesa existe solo mientras el modo está activo. Su resolución no cambia el estado.
	let method: Promise<string> | undefined;

	async function loadMethod(cwd: string): Promise<string> {
		try {
			const r = await pi.exec("mem", ["plan-context"], { cwd, timeout: 5000 });
			const out = (r.stdout ?? "").trim();
			if (r.code === 0 && out !== "") {
				return `Estás en MODO PLAN: no modifiques archivos; entrega el árbol de tareas y pregunta si se ejecuta.\n\n${out}`;
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
		if (patterns.tools.includes(event.toolName)) {
			return { block: true, reason: REASON };
		}
		if (event.toolName === "bash" && bashBlocked(String(event.input?.command ?? ""))) {
			return { block: true, reason: REASON };
		}
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

/**
 * gomemory-plan.ts — Modo /plan de pi-gomemory.
 *
 * /plan alterna un modo que bloquea las modificaciones comunes (edit, write y
 * comandos bash de gomemory-plan-patterns.json) e inyecta en el system prompt el
 * método de descomposición atómica de `mem plan-context`. /plan run solo desactiva.
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
entrega el árbol y pregunta si se ejecuta. La persona sale del modo con /plan o /plan run.`;

const REASON = "Modo plan activo: no se permiten modificaciones. Usa /plan o /plan run para salir.";

export default function (pi: ExtensionAPI) {
	let active = false;
	let method = "";

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

	function setActive(ctx: any, on: boolean) {
		active = on;
		ctx.ui?.setStatus?.("gomemory-plan", on ? "PLAN" : undefined);
	}

	pi.registerCommand("plan", {
		description: "Modo plan de gomemory: bloquea modificaciones hasta /plan o /plan run",
		handler: async (args, ctx) => {
			const arg = String(args ?? "").trim();
			if (arg === "run") {
				if (!active) {
					ctx.ui.notify("El modo plan no está activo", "info");
					return;
				}
				setActive(ctx, false);
				ctx.ui.notify("Modo plan desactivado: ya se permiten modificaciones", "info");
				return;
			}
			if (arg !== "") {
				ctx.ui.notify(`Argumento desconocido: ${arg}. Usa /plan o /plan run`, "warning");
				return;
			}
			if (active) {
				setActive(ctx, false);
				ctx.ui.notify("Modo plan desactivado", "info");
				return;
			}
			method = await loadMethod(ctx.cwd);
			setActive(ctx, true);
			ctx.ui.notify("Modo plan activo: edit, write y comandos que modifican están bloqueados", "info");
		},
	});

	pi.on("tool_call", async (event: any) => {
		if (!active) return;
		if (patterns.tools.includes(event.toolName)) {
			return { block: true, reason: REASON };
		}
		if (event.toolName === "bash" && bashBlocked(String(event.input?.command ?? ""))) {
			return { block: true, reason: REASON };
		}
	});

	pi.on("before_agent_start", async (event: any) => {
		if (!active) return;
		return { systemPrompt: `${event.systemPrompt}\n\n${method || FALLBACK}` };
	});
}

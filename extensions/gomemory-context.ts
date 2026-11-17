/**
 * gomemory-context.ts — contexto vivo de gomemory para Pi (HCD-045).
 *
 * Pi solo lleva al modelo una línea de las instructions MCP de gomemory, así que
 * esta extensión hace lo que el hook SessionStart hace en Claude Code:
 *
 * - En session_start (startup, reload, new, resume, fork) lanza `mem context`
 *   con el cwd de la sesión, sin bloquear el arranque.
 * - En before_agent_start espera esa carga y añade la sección `gomemory` a
 *   systemPromptOptions.sections: la salida de `mem context` (como mucho 24 KB)
 *   más el bloque propio de Pi, derivado de las tools efectivas de la ejecución.
 *
 * La sección no cambia dentro de una sesión mientras no cambien las tools, para
 * conservar la caché del prompt. Si `mem` falla, solo va el bloque Pi y se avisa
 * una vez por sesión: nunca se inventa el contexto de gomemory.
 */

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export const SECTION = "gomemory";
export const MAX_CONTEXT_BYTES = 24576;
export const MEM_TIMEOUT_MS = 3000;
export const TRUNCATION_LINE = "… contexto recortado: llama get_context() para el resto";

type MemContext = { text: string } | { error: string };

/** Recorta text a maxBytes de UTF-8 por el final, sin partir un carácter. */
export function truncateUtf8(text: string, maxBytes: number): { text: string; truncated: boolean } {
	const bytes = Buffer.from(text, "utf8");
	if (bytes.length <= maxBytes) return { text, truncated: false };
	let end = maxBytes;
	// Retrocede si el corte cae en un byte de continuación (10xxxxxx).
	while (end > 0 && (bytes[end] & 0xc0) === 0x80) end--;
	return { text: bytes.subarray(0, end).toString("utf8"), truncated: true };
}

function toolName(tool: unknown): string {
	if (typeof tool === "string") return tool;
	return String((tool as { name?: unknown })?.name ?? "");
}

/**
 * Bloque propio de Pi. No nombra tools: el modelo ya ve las que el prompt
 * declara, y en before_agent_start las tools MCP aún no están en selectedTools
 * porque los servidores conectan en segundo plano (HCD-045, Q8). Solo la tool
 * builtin `codemode` decide la línea de cómo llegar al resto.
 */
export function piBlock(selectedTools: unknown[] = []): string {
	const codemode = selectedTools.map(toolName).includes("codemode");
	const lines = [
		"## Protocolo de pi-gomemory en Pi",
		"",
		"- Las tools de gomemory (mcp__gomemory__*) y codebase-memory (mcp__codebase_memory__*) declaradas en este prompt se llaman directamente.",
	];
	if (codemode) {
		lines.push(
			"- Codemode activo: el resto de tools de gomemory y codebase-memory (review_*, octopus_*, pack_*, hcd_history, socratica_flow, query_graph…) no se pierden; se llaman desde la tool 'codemode' con 'tools.<nombre>(args)'. No recurras a grep/find por no verlas como directas.",
		);
	}
	lines.push(
		"- HCD: comprueba la unidad activa con 'mem hcd show'; ejecuta las comprobaciones en bash para ver el rojo o el verde y registra la evidencia con 'mem hcd verify'.",
		"- Modo plan: antes de redactar un plan llama a get_plan_context() o 'mem plan-context'; entrega el árbol de tareas atómicas y detente.",
		"- Splash: ejecuta la tool splash_review con la ruta del documento y una lente opcional. No le envíes historial ni memoria, no la sustituyas por una autoevaluación y no tomes un fallo de ejecución por una revisión aprobada.",
		"- Si gomemory no respondió al iniciar la sesión, usa la CLI 'mem' (mem context, mem search) o get_context().",
	);
	return lines.join("\n");
}

/** Sección completa: contexto de gomemory (recortado) y bloque Pi. */
export function buildSection(mem: MemContext | undefined, selectedTools: unknown[] = []): string {
	const parts: string[] = [];
	if (mem && "text" in mem) {
		const { text, truncated } = truncateUtf8(mem.text, MAX_CONTEXT_BYTES);
		parts.push(truncated ? `${text}\n${TRUNCATION_LINE}` : text);
	}
	parts.push(piBlock(selectedTools));
	return parts.join("\n\n");
}

export default function (pi: ExtensionAPI) {
	let pending: Promise<MemContext> | undefined;
	let warned = false;

	async function load(cwd: string | undefined): Promise<MemContext> {
		try {
			const r: any = await pi.exec("mem", ["context"], { cwd, timeout: MEM_TIMEOUT_MS });
			const text = String(r?.stdout ?? "").trim();
			if (r?.killed) return { error: `mem context superó ${MEM_TIMEOUT_MS} ms` };
			if (r?.code !== 0) return { error: `mem context terminó con código ${r?.code}: ${String(r?.stderr ?? "").trim().split("\n")[0]}` };
			if (text === "") return { error: "mem context no devolvió contexto" };
			return { text };
		} catch (err: any) {
			return { error: `no se pudo ejecutar mem: ${err?.message ?? err}` };
		}
	}

	function cwdOf(ctx: any): string | undefined {
		try {
			return ctx?.cwd;
		} catch {
			return undefined; // ctx obsoleto tras recargar la sesión
		}
	}

	pi.on("session_start", async (_event: any, ctx: any) => {
		warned = false;
		pending = load(cwdOf(ctx)); // no se espera: el arranque no se bloquea
	});

	pi.on("before_agent_start", async (event: any, ctx: any) => {
		pending ??= load(cwdOf(ctx));
		const mem = await pending;
		if ("error" in mem && !warned) {
			warned = true;
			ctx?.ui?.notify?.(`gomemory: sin contexto en esta sesión (${mem.error})`, "warning");
		}
		const opts = event?.systemPromptOptions;
		if (!opts) return;
		opts.sections ??= {};
		opts.sections[SECTION] = buildSection(mem, opts.selectedTools ?? []);
	});
}

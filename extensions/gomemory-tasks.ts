/**
 * gomemory-tasks.ts — Panel de tareas HCD de pi-gomemory.
 *
 * Muestra sobre el editor las tareas de la unidad HCD activa, leídas de
 * `mem hcd show --json` (el motor HCD es la única fuente). Se refresca al
 * iniciar la sesión, tras cada comando bash que ejecute `mem hcd` y al terminar
 * cada turno. /tasks muestra u oculta el panel.
 */

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

function mark(status: string): string {
	if (status === "verified") return "✓";
	if (status === "failed" || status === "red") return "✗";
	if (status === "unverified") return "◌"; // tenía evidencia y caducó
	return "○";
}

function short(text: string): string {
	const t = text.replace(/\s+/g, " ").trim();
	return t.length > MAX_TEXT ? `${t.slice(0, MAX_TEXT - 1)}…` : t;
}

export function renderTasks(u: Unit): string[] | undefined {
	if (!u?.ok || !u.unit_id || u.state !== "active" || !Array.isArray(u.tasks) || u.tasks.length === 0) {
		return undefined;
	}
	const done = u.tasks.filter((t) => t.status === "verified").length;
	return [
		`${u.unit_id} · ${u.kind} · paso ${u.stage} · ${done}/${u.tasks.length}`,
		...u.tasks.map((t) => `  ${mark(t.status)} ${t.id}. ${short(t.text)}`),
	];
}

/** Detecta llamadas bash que agrupan múltiples `mem hcd verify` o los envuelven en funciones/bucles. */
export function isBundledHcdVerifyCommand(command: string): string | null {
	const cmd = command.trim();
	if (!cmd) return null;
	const verifyCount = (cmd.match(/\bmem\s+hcd\s+verify\b/g) ?? []).length;
	if (verifyCount > 1) {
		return "Ejecuta un solo `mem hcd verify` por llamada bash para permitir refresco en vivo y entrega de steering.";
	}
	if (
		verifyCount === 1 &&
		(/\b(?:for|while|until)\b[\s\S]*\bmem\s+hcd\s+verify\b/.test(cmd) ||
			/\b[A-Za-z_][A-Za-z0-9_]*\s*\(\s*\)\s*\{[\s\S]*\bmem\s+hcd\s+verify\b/.test(cmd))
	) {
		return "No envuelvas `mem hcd verify` en funciones o bucles dentro de una sola llamada bash; ejecuta cada verificación por separado.";
	}
	return null;
}

export default function (pi: ExtensionAPI) {
	let visible = true;
	let busy = false;
	let pendingCtx: any = null;
	let pollTimer: ReturnType<typeof setInterval> | null = null;

	function stopPolling() {
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
	}

	async function refresh(ctx: any) {
		if (busy) {
			pendingCtx = ctx;
			return;
		}
		busy = true;
		try {
			let lines: string[] | undefined;
			if (visible) {
				try {
					const r = await pi.exec("mem", ["hcd", "show", "--json"], { cwd: ctx.cwd, timeout: 5000 });
					if (r.code === 0 && visible) lines = renderTasks(JSON.parse(r.stdout ?? ""));
				} catch {
					// mem ausente, sin unidad o salida inválida: no se muestra nada
				}
			}
			if (!visible) lines = undefined;
			ctx.ui?.setWidget?.(WIDGET, lines);
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
		description: "Muestra u oculta el panel de tareas de la unidad HCD activa",
		handler: async (_args, ctx) => {
			visible = !visible;
			await refresh(ctx);
			ctx.ui.notify(visible ? "Panel de tareas visible" : "Panel de tareas oculto", "info");
		},
	});

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
		const cmd = String(event?.args?.command ?? event?.input?.command ?? "");
		if (event?.toolName === "bash" && /\bmem\s+hcd\b/.test(cmd)) {
			stopPolling();
			pollTimer = setInterval(() => {
				void refresh(ctx);
			}, 1500);
			(pollTimer as any)?.unref?.();
		}
	});

	pi.on("tool_execution_end", async () => {
		stopPolling();
	});

	pi.on("tool_result", async (event: any, ctx: any) => {
		if (event.toolName === "bash" && /\bmem\s+hcd\b/.test(String(event.input?.command ?? ""))) {
			stopPolling();
			await refresh(ctx);
		}
	});

	pi.on("agent_end", async (_event: any, ctx: any) => {
		stopPolling();
		await refresh(ctx);
	});

	pi.on("session_shutdown", async () => {
		stopPolling();
	});
}

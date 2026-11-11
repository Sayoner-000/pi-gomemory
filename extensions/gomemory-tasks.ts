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

export default function (pi: ExtensionAPI) {
	let visible = true;
	let busy = false;

	async function refresh(ctx: any) {
		if (busy) return;
		busy = true;
		try {
			let lines: string[] | undefined;
			if (visible) {
				try {
					const r = await pi.exec("mem", ["hcd", "show", "--json"], { cwd: ctx.cwd, timeout: 5000 });
					if (r.code === 0) lines = renderTasks(JSON.parse(r.stdout ?? ""));
				} catch {
					// mem ausente, sin unidad o salida inválida: no se muestra nada
				}
			}
			ctx.ui?.setWidget?.(WIDGET, lines);
		} finally {
			busy = false;
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

	pi.on("tool_result", async (event: any, ctx: any) => {
		if (event.toolName === "bash" && /\bmem\s+hcd\b/.test(String(event.input?.command ?? ""))) {
			await refresh(ctx);
		}
	});

	pi.on("agent_end", async (_event: any, ctx: any) => {
		await refresh(ctx);
	});
}

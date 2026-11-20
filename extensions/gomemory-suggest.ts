/**
 * gomemory-suggest — sugerencia del siguiente comando en el input de Pi (HCD-057).
 *
 * Con el input vacío muestra atenuado el siguiente paso; Tab lo copia al input
 * sin enviarlo (Q2). Origen (Q1): con una unidad HCD activa, el next_action del
 * motor (`mem hcd show --json`), sin tokens; sin unidad, una sugerencia breve
 * del modelo activo al terminar el turno, acotada y cancelable.
 *
 * El editor previo (de otra extensión) se envuelve, no se sustituye: se decoran
 * su render y su handleInput sobre la misma instancia.
 */

import { CustomEditor, type ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { matchesKey, truncateToWidth, visibleWidth } from "@earendil-works/pi-tui";

/**
 * Tope de tokens de la sugerencia del modelo. En los modelos con razonamiento el
 * tope incluye lo que piensan: con 40, Opus 5.5 no llegaba a escribir; con el
 * razonamiento mínimo usa 42-103 tokens (≈ 0,002-0,003 USD por turno, Pi real).
 */
export const SUGGEST_MAX_TOKENS = 300;
const MAX_CONTEXT_CHARS = 1500;
const MAX_SUGGESTION_CHARS = 120;
const HINT = " · Tab";
const OWN_FACTORY = Symbol.for("gomemory-suggest.factory");

const SYSTEM_PROMPT = `Eres un asistente que anticipa el siguiente mensaje de la persona a un agente de programación.
Con el último intercambio, propone el siguiente mensaje breve que la persona probablemente enviará.
Reglas: una sola línea, como máximo 12 palabras, en el idioma de la conversación, sin comillas ni explicaciones.
Si no hay un siguiente paso claro, responde con una línea vacía.`;

/** Comando sugerido por el motor HCD: «/<skill>» del next_action de una unidad activa. */
export function hcdSuggestion(show: any): string {
	if (!show?.ok || show.state !== "active") return "";
	const skill = show?.next_action?.skill;
	return typeof skill === "string" && /^[a-z][a-z0-9-]{0,40}$/.test(skill) ? `/${skill}` : "";
}

/** Primera línea útil de la respuesta del modelo, sin comillas y acotada. */
function cleanSuggestion(text: string): string {
	const line = text
		.split(/\r?\n/)
		.map((l) => l.trim())
		.find(Boolean);
	if (!line) return "";
	const unquoted = line.replace(/^["'`«“]+|["'`»”]+$/g, "").trim();
	return unquoted.slice(0, MAX_SUGGESTION_CHARS);
}

/** Texto de la última intervención de cada rol en la rama actual. */
function lastExchange(ctx: any): { user: string; assistant: string } {
	const texts = (msg: any) =>
		(Array.isArray(msg?.content) ? msg.content : [])
			.filter((c: any) => c?.type === "text" && typeof c.text === "string")
			.map((c: any) => c.text)
			.join("\n");
	let user = "";
	let assistant = "";
	let branch: any[] = [];
	try {
		branch = ctx?.sessionManager?.getBranch?.() ?? [];
	} catch {
		branch = [];
	}
	for (let i = branch.length - 1; i >= 0 && (!user || !assistant); i--) {
		const msg = branch[i]?.type === "message" ? branch[i].message : null;
		if (!user && msg?.role === "user") user = texts(msg);
		if (!assistant && msg?.role === "assistant") assistant = texts(msg);
	}
	return { user: user.slice(-MAX_CONTEXT_CHARS), assistant: assistant.slice(-MAX_CONTEXT_CHARS) };
}

/** Escribe la sugerencia atenuada en los espacios finales de la línea del cursor, sin cambiar su ancho. */
function ghostLine(line: string, text: string, dim: (s: string) => string): string {
	const trailing = line.length - line.replace(/ +$/, "").length;
	const room = trailing - 1; // un espacio de aire tras el cursor
	if (room < 4) return line;
	const ghost = truncateToWidth(`${text}${HINT}`, room, "…", false);
	return `${line.slice(0, line.length - trailing)} ${dim(ghost)}${" ".repeat(Math.max(0, room - visibleWidth(ghost)))}`;
}

export default function (pi: ExtensionAPI) {
	let suggestion = "";
	let pending: AbortController | null = null;
	let dim = (s: string) => `\x1b[2m${s}\x1b[22m`;
	let requestRender: () => void = () => {};

	function abortPending() {
		pending?.abort();
		pending = null;
	}

	function setSuggestion(next: string) {
		suggestion = next;
		requestRender();
	}

	/** Decora la instancia del editor: sugerencia en render y Tab en handleInput. */
	function decorate(editor: any) {
		const render = editor.render.bind(editor);
		const handleInput = editor.handleInput.bind(editor);
		editor.render = (width: number) => {
			const lines: string[] = render(width);
			if (!suggestion || editor.getText() !== "" || lines.length < 2) return lines;
			const out = [...lines];
			out[1] = ghostLine(out[1], suggestion, dim);
			return out;
		};
		editor.handleInput = (data: string) => {
			if (suggestion && editor.getText() === "" && matchesKey(data, "tab")) {
				editor.setText(suggestion);
				suggestion = "";
				requestRender();
				return;
			}
			handleInput(data);
			// La persona escribe: la sugerencia del modelo en curso ya no sirve.
			if (editor.getText() !== "") abortPending();
		};
		return editor;
	}

	/**
	 * Deja nuestro editor por fuera de la ranura. Otras extensiones (pi-cc-extensions)
	 * la reemplazan sin envolver en su propio session_start, después del nuestro: si
	 * la ranura ya no es nuestra, se envuelve la que haya. Si ya lo es, no se toca,
	 * porque reinstalar recrea el editor y perdería su estado.
	 */
	function ensureEditor(ctx: any) {
		try {
			const ui = ctx?.ui;
			if (!ctx?.hasUI || typeof ui?.setEditorComponent !== "function") return;
			const current = ui.getEditorComponent?.();
			if (current && (current as any)[OWN_FACTORY] !== undefined) return;
			installEditor(ctx, current);
		} catch {
			// ctx obsoleto tras /reload o /new: lo reinstala el siguiente session_start
		}
	}

	function installEditor(ctx: any, current: any) {
		const ui = ctx.ui;
		if (typeof ui.theme?.fg === "function") dim = (s: string) => ui.theme.fg("dim", s);
		const previous = current;
		const factory = (tui: any, theme: any, keybindings: any) => {
			requestRender = () => tui?.requestRender?.();
			const editor = previous ? previous(tui, theme, keybindings) : new CustomEditor(tui, theme, keybindings);
			return decorate(editor);
		};
		(factory as any)[OWN_FACTORY] = previous ?? null;
		ui.setEditorComponent(factory);
	}

	async function hcdNext(ctx: any): Promise<string> {
		try {
			const r = await pi.exec("mem", ["hcd", "show", "--json"], { cwd: ctx?.cwd, timeout: 5000 });
			return r.code === 0 ? hcdSuggestion(JSON.parse(r.stdout ?? "")) : "";
		} catch {
			return ""; // mem ausente o salida inválida
		}
	}

	async function modelNext(ctx: any): Promise<void> {
		const registry = ctx?.modelRegistry;
		if (!ctx?.model || typeof registry?.streamSimple !== "function") return;
		const { user, assistant } = lastExchange(ctx);
		if (!user && !assistant) return;
		const ac = new AbortController();
		pending = ac;
		try {
			// Razonamiento mínimo: omitirlo no lo apaga en Opus 5.5 y con el nivel de la
			// sesión el modelo gasta el tope pensando y no deja texto (visto en Pi real).
			const stream = registry.streamSimple(
				ctx.model,
				{
					systemPrompt: SYSTEM_PROMPT,
					messages: [
						{
							role: "user",
							content: [{ type: "text", text: `Mensaje de la persona:\n${user}\n\nRespuesta del agente:\n${assistant}` }],
							timestamp: Date.now(),
						},
					],
				},
				// Petición lateral: no debe reemplazar el estado de continuación de la conversación real.
				{ signal: ac.signal, maxTokens: SUGGEST_MAX_TOKENS, reasoning: "minimal", disableContinuation: true },
			);
			const res = await stream.result();
			if (ac.signal.aborted || pending !== ac || res?.stopReason === "aborted" || res?.stopReason === "error") return;
			const text = cleanSuggestion(
				(Array.isArray(res?.content) ? res.content : [])
					.filter((c: any) => c?.type === "text")
					.map((c: any) => c.text)
					.join("\n"),
			);
			if (text) setSuggestion(text);
		} catch {
			// Sin red, sin cuota o abortado: simplemente no hay sugerencia.
		} finally {
			if (pending === ac) pending = null;
		}
	}

	pi.on("session_start", async (_event: any, ctx: any) => {
		ensureEditor(ctx);
		if (!ctx?.hasUI) return;
		// Las demás extensiones terminan su session_start después: se comprueba de nuevo.
		for (const ms of [0, 100, 1000]) (setTimeout(() => ensureEditor(ctx), ms) as any)?.unref?.();
		// Al arrancar solo el motor: sin turno previo no hay nada que pedir al modelo.
		setSuggestion(await hcdNext(ctx));
		ensureEditor(ctx);
	});

	pi.on("before_agent_start", async (_event: any, ctx: any) => {
		abortPending();
		setSuggestion("");
		ensureEditor(ctx);
	});

	pi.on("agent_end", async (_event: any, ctx: any) => {
		ensureEditor(ctx);
		abortPending();
		suggestion = "";
		const fromHCD = await hcdNext(ctx);
		if (fromHCD) {
			setSuggestion(fromHCD);
			return;
		}
		requestRender();
		await modelNext(ctx);
	});

	pi.on("session_shutdown", async () => {
		abortPending();
		suggestion = "";
	});
}

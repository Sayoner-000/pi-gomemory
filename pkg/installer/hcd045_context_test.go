package installer

import (
	"strings"
	"testing"
)

// hcd045Harness monta un fakePi para gomemory-context.ts: registra los
// handlers, simula pi.exec con execImpl y deja helpers de sesión y ejecución.
const hcd045Harness = `
import ext, { SECTION, MAX_CONTEXT_BYTES, TRUNCATION_LINE, MEM_TIMEOUT_MS } from "./gomemory-context.ts";

const handlers = {};
const execCalls = [];
const notices = [];
let execImpl = async () => ({ code: 0, stdout: "# Memoria del Proyecto\nregla", stderr: "" });
const fakePi = {
	on: (event, handler) => { (handlers[event] ??= []).push(handler); },
	registerCommand: () => {},
	exec: (cmd, args, opts) => { execCalls.push({ cmd, args, opts }); return execImpl(cmd, args, opts); },
};
ext(fakePi);
const ctx = (cwd = "/proyecto") => ({ cwd, ui: { notify: (msg, level) => notices.push({ msg, level }) } });
const startSession = async (reason = "startup", cwd) => {
	for (const h of handlers.session_start ?? []) await h({ reason }, ctx(cwd));
};
const run = async (selectedTools = [], hiddenTools = [], cwd) => {
	const opts = { sections: {}, selectedTools, hiddenTools };
	const event = { messageType: "user", prompt: "hola", systemPrompt: "base", systemPromptOptions: opts };
	let result;
	for (const h of handlers.before_agent_start ?? []) result = await h(event, ctx(cwd));
	return { section: opts.sections[SECTION], result, opts };
};
const fails = [];
`

func runHCD045(t *testing.T, body string) {
	t.Helper()
	out := runNodeExt(t, []string{"extensions/gomemory-context.ts"}, hcd045Harness+body+`
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("%s", out)
	}
}

// C-01: session_start lanza mem context con el cwd de la sesión y el timeout,
// sin bloquear el arranque; before_agent_start espera la carga y añade la sección.
func TestHCD045ContextLoad(t *testing.T) {
	t.Parallel()

	runHCD045(t, `
let release;
execImpl = () => new Promise((r) => { release = r; });
let started = false;
const startPromise = startSession("startup", "/repo").then(() => { started = true; });
await new Promise((r) => setTimeout(r, 10));
if (!started) fails.push("session_start bloquea el arranque esperando a mem");
const c = execCalls[0];
if (!c || c.cmd !== "mem" || c.args.join(" ") !== "context") fails.push("no lanza mem context: " + JSON.stringify(c));
if (c && c.opts?.cwd !== "/repo") fails.push("cwd incorrecto: " + c?.opts?.cwd);
if (c && c.opts?.timeout !== MEM_TIMEOUT_MS) fails.push("timeout incorrecto: " + c?.opts?.timeout);
if (MEM_TIMEOUT_MS !== 3000) fails.push("MEM_TIMEOUT_MS debe ser 3000");
const pending = run([], [], "/repo");
release({ code: 0, stdout: "# Memoria del Proyecto\nregla viva\n", stderr: "" });
const { section, result } = await pending;
await startPromise;
if (!section || !section.includes("regla viva")) fails.push("la sección no contiene mem context: " + section);
if (result && result.systemPrompt !== undefined) fails.push("no debe reemplazar systemPrompt");
`)
}

// C-01 (Q2): la sección es idéntica entre ejecuciones de la misma sesión y
// mem context se ejecuta una sola vez por sesión.
func TestHCD045ContextStable(t *testing.T) {
	t.Parallel()

	runHCD045(t, `
let n = 0;
execImpl = async () => ({ code: 0, stdout: "contexto " + (++n), stderr: "" });
await startSession();
const a = (await run(["read"])).section;
const b = (await run(["read"])).section;
if (a !== b) fails.push("la sección cambia entre ejecuciones");
if (execCalls.length !== 1) fails.push("mem context se ejecutó " + execCalls.length + " veces");
`)
}

// C-03: cada session_start (cualquier reason) recarga el contexto.
func TestHCD045ContextNewSession(t *testing.T) {
	t.Parallel()

	runHCD045(t, `
let text = "regla antigua";
execImpl = async () => ({ code: 0, stdout: text, stderr: "" });
await startSession("startup");
const a = (await run()).section;
for (const reason of ["reload", "new", "resume", "fork"]) {
	text = "regla " + reason;
	await startSession(reason);
	const s = (await run()).section;
	if (!s.includes("regla " + reason)) fails.push(reason + " no recarga el contexto");
}
if (!a.includes("regla antigua")) fails.push("primera sesión sin contexto");
// Sin session_start previo (extensión recargada a mitad), carga bajo demanda.
`)
}

// C-08: más de 24 KB se recorta por el final sin partir UTF-8 y con la línea de recorte.
func TestHCD045ContextTruncate(t *testing.T) {
	t.Parallel()

	runHCD045(t, `
if (MAX_CONTEXT_BYTES !== 24576) fails.push("MAX_CONTEXT_BYTES debe ser 24576");
if (!TRUNCATION_LINE.includes("get_context()")) fails.push("la línea de recorte no remite a get_context()");
const big = "ñ".repeat(20000) + "FINAL";
execImpl = async () => ({ code: 0, stdout: big, stderr: "" });
await startSession();
const { section } = await run();
const memPart = section.slice(0, section.indexOf(TRUNCATION_LINE)).replace(/\n$/, "");
if (!section.includes(TRUNCATION_LINE)) fails.push("falta la línea de recorte");
if (section.includes("FINAL")) fails.push("no recortó por el final");
if (Buffer.byteLength(memPart, "utf8") > MAX_CONTEXT_BYTES) fails.push("supera el tope: " + Buffer.byteLength(memPart, "utf8"));
if (memPart.includes("�")) fails.push("partió un carácter UTF-8");
execImpl = async () => ({ code: 0, stdout: "corto", stderr: "" });
await startSession("new");
if ((await run()).section.includes(TRUNCATION_LINE)) fails.push("recorta un contexto corto");
`)
}

// C-04 (Q8): el bloque Pi no lee mcp.json ni nombra tools concretas (en
// before_agent_start las tools MCP aún no están en selectedTools: conectan en
// segundo plano); remite a las declaradas y dice si codemode está activo.
func TestHCD045PiBlock(t *testing.T) {
	t.Parallel()

	runHCD045(t, `
const src = (await import("node:fs")).readFileSync(new URL("./gomemory-context.ts", import.meta.url), "utf8");
if (/from\s+["']node:fs|require\(["']node:fs|readFile/.test(src)) fails.push("la extensión no debe leer archivos (mcp.json)");
await startSession();
const hybrid = (await run(["read", "codemode", "mcp__gomemory__search_code"])).section;
if (/mcp__\w+__\w+/.test(hybrid)) fails.push("el bloque Pi nombra tools concretas: " + hybrid.match(/mcp__\w+__\w+/)[0]);
if (!hybrid.includes("tools.<nombre>(args)")) fails.push("con codemode activo no explica cómo llamar al resto");
// Sin tools MCP todavía (como en Pi real), el bloque no dice «ninguna».
const early = (await run(["read", "codemode"])).section;
if (/ninguna/i.test(early)) fails.push("afirma que no hay tools directas cuando aún no conectaron");
if (early !== hybrid) fails.push("el bloque cambia según las tools MCP conectadas");
await startSession("new");
const direct = (await run(["read", "mcp__gomemory__search_code"])).section;
if (direct.includes("tools.<nombre>(args)")) fails.push("sin codemode no debe hablar de tools.<nombre>(args)");
`)
}

// C-09 (Q5): con /plan activo, sections contiene la sección de gomemory y la del
// método de plan; gomemory-plan.ts ya no reemplaza systemPrompt.
func TestHCD045PlanSections(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-context.ts", "extensions/gomemory-plan.ts", "extensions/gomemory-plan-patterns.json"}, `
import contextExt, { SECTION } from "./gomemory-context.ts";
import planExt from "./gomemory-plan.ts";

const handlers = {};
const commands = {};
const fakePi = {
	on: (event, handler) => { (handlers[event] ??= []).push(handler); },
	registerCommand: (name, def) => { commands[name] = def; },
	exec: async (cmd, args) => ({ code: 0, stdout: args[0] === "plan-context" ? "MÉTODO ATÓMICO" : "# Memoria del Proyecto", stderr: "" }),
	sendUserMessage: () => {},
};
contextExt(fakePi);
planExt(fakePi);
const ctx = { cwd: "/repo", ui: { notify: () => {}, setStatus: () => {} }, isIdle: () => true };
for (const h of handlers.session_start ?? []) await h({ reason: "startup" }, ctx);
await commands.plan.handler("", ctx);

const opts = { sections: {}, selectedTools: ["read"], hiddenTools: [] };
const event = { messageType: "user", prompt: "x", systemPrompt: "base", systemPromptOptions: opts };
const fails = [];
for (const h of handlers.before_agent_start ?? []) {
	const r = await h(event, ctx);
	if (r && r.systemPrompt !== undefined) fails.push("un handler reemplaza systemPrompt");
}
if (!opts.sections[SECTION]?.includes("Memoria del Proyecto")) fails.push("falta la sección gomemory con /plan activo");
const planSection = Object.entries(opts.sections).find(([k, v]) => k !== SECTION && String(v).includes("MÉTODO ATÓMICO"));
if (!planSection) fails.push("falta la sección del método de plan: " + Object.keys(opts.sections));

await commands.plan.handler("", ctx); // desactiva
const opts2 = { sections: {}, selectedTools: [], hiddenTools: [] };
for (const h of handlers.before_agent_start ?? []) await h({ ...event, systemPromptOptions: opts2 }, ctx);
if (Object.values(opts2.sections).some((v) => String(v).includes("MÉTODO ATÓMICO"))) fails.push("el método sigue tras salir de /plan");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("%s", out)
	}
}

// C-02 (Q6): mem ausente, con error, vacío o con timeout → solo bloque Pi y un
// único aviso warning por sesión; nunca texto que imite mem context.
func TestHCD045ContextMemFailure(t *testing.T) {
	t.Parallel()

	runHCD045(t, `
const cases = {
	ausente: async () => { throw new Error("spawn mem ENOENT"); },
	error: async () => ({ code: 1, stdout: "", stderr: "boom" }),
	vacio: async () => ({ code: 0, stdout: "  \n", stderr: "" }),
	timeout: async () => ({ code: null, killed: true, stdout: "", stderr: "" }),
};
for (const [name, impl] of Object.entries(cases)) {
	execImpl = impl;
	notices.length = 0;
	await startSession("new");
	const a = (await run()).section;
	const b = (await run()).section;
	if (!a || !a.includes("pi-gomemory")) fails.push(name + ": falta el bloque Pi");
	if (a.includes("Memoria del Proyecto")) fails.push(name + ": inventa el contexto de gomemory");
	if (a !== b) fails.push(name + ": la sección cambia entre ejecuciones");
	const warns = notices.filter((n) => n.level === "warning");
	if (warns.length !== 1) fails.push(name + ": avisos warning = " + warns.length);
}
`)
}

package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hcd057Harness monta gomemory-suggest.ts con un Pi falso. show es la salida de
// `mem hcd show --json`; complete simula ctx.modelRegistry.complete.
const hcd057Harness = `
import registerSuggest from "./gomemory-suggest.ts";
import { CustomEditor } from "@earendil-works/pi-coding-agent";
const fails = [];
const h = {};
let show = { ok: false };
let factory = null, prevFactory = null;
const completeCalls = [];
let completeImpl = async () => ({ stopReason: "stop", content: [{ type: "text", text: "Ejecuta los tests" }] });
const fakePi = {
	registerCommand() {}, registerShortcut() {},
	on(n, fn) { h[n] = fn; },
	async exec(cmd, args) {
		if (cmd === "mem" && args.join(" ") === "hcd show --json") return { code: show.ok ? 0 : 3, stdout: JSON.stringify(show) };
		return { code: 1, stdout: "" };
	},
};
const tui = { renders: 0, requestRender() { this.renders++; }, terminal: { rows: 40 } };
const ctx = {
	cwd: process.cwd(), hasUI: true, mode: "tui",
	model: { provider: "anthropic", id: "m" },
	// streamSimple es la vía neutral de Pi y admite «reasoning» (en Pi real, Opus
	// con el razonamiento de la sesión gastaba el tope pensando y la respuesta
	// llegaba vacía por «length»).
	modelRegistry: {
		streamSimple: (m, c, o) => { completeCalls.push({ m, c, o }); const p = Promise.resolve().then(() => completeImpl(m, c, o)); return { result: () => p }; },
		complete: () => { throw new Error("debe usarse streamSimple, no complete"); },
	},
	sessionManager: { getBranch: () => [
		{ type: "message", message: { role: "user", content: [{ type: "text", text: "arregla el login" }] } },
		{ type: "message", message: { role: "assistant", stopReason: "stop", content: [{ type: "text", text: "Hecho: corregí el token." }] } },
	] },
	ui: {
		theme: { fg: (c, s) => "<" + c + ">" + s + "</" + c + ">" },
		notify() {},
		getEditorComponent: () => prevFactory,
		setEditorComponent(f) { factory = f; },
	},
};
registerSuggest(fakePi);
const plain = (s) => s.replace(/<\/?[a-z]+>/g, "").replace(/\x1b\[[0-9;]*m/g, "");
const ghostOf = (ed) => { const l = ed.render(60)[1] ?? ""; return l.match(/<dim>(.*?)<\/dim>/)?.[1] ?? ""; };
`

// HCD-057 tarea 9: gomemory-suggest.ts se instala con las demás extensiones y
// figura en el manifiesto, que es lo que uninstall retira.
func TestHCD057Install(t *testing.T) {
	opts := sandbox(t)
	if err := InstallTUIExtension(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(opts.PiAgentDir, "extensions", "gomemory-suggest.ts")
	if _, err := os.Stat(dst); err != nil || readFile(t, dst) != readAsset(t, "extensions/gomemory-suggest.ts") {
		t.Fatalf("gomemory-suggest.ts no se instala o difiere del embebido: %v", err)
	}
	found := false
	for _, f := range installedFiles(opts) {
		found = found || f == dst
	}
	if !found {
		t.Fatal("gomemory-suggest.ts no figura en el manifiesto: uninstall no lo retiraría")
	}
}

// HCD-057 tarea 5: el editor de sugerencias envuelve el editor previo o, si no
// lo hay, crea un CustomEditor; conserva su comportamiento y su ancho.
func TestHCD057EditorWrap(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-suggest.ts"}, hcd057Harness+`
await h.session_start({}, ctx);
if (typeof factory !== "function") fails.push("debe instalar un editor con setEditorComponent");
const ed = factory?.(tui, {}, {});
if (!(ed instanceof CustomEditor)) fails.push("sin editor previo debe crear un CustomEditor");

// Con un editor previo, se envuelve la misma instancia y conserva su render.
class Prev extends CustomEditor { render(w) { return super.render(w).map((l, i) => (i === 0 ? "PREV" + l.slice(4) : l)); } }
prevFactory = (t2, th, kb) => new Prev(t2, th, kb);
factory = null;
await h.session_start({}, ctx);
const wrapped = factory?.(tui, {}, {});
if (!(wrapped instanceof Prev)) fails.push("debe envolver el editor previo, no sustituirlo");
if (!wrapped?.render(60)[0]?.startsWith("PREV")) fails.push("el render del editor previo debe conservarse");
wrapped?.handleInput("a");
if (wrapped?.getText() !== "a") fails.push("la escritura debe llegar al editor previo");

// Sin UI (modo print/json) no toca el editor.
factory = null;
await h.session_start({}, { ...ctx, hasUI: false });
if (factory) fails.push("sin UI no debe instalar editor");

// Caso real en Pi (pi-cc-extensions): otra extensión instala su editor DESPUÉS
// en session_start y reemplaza la ranura sin envolver. El nuestro debe volver a
// quedar por fuera envolviéndolo, a más tardar al empezar el siguiente turno.
{
	let slot;
	const slotCtx = { ...ctx, ui: { ...ctx.ui, getEditorComponent: () => slot, setEditorComponent(f) { slot = f; } } };
	await h.session_start({}, slotCtx);
	class Late extends CustomEditor { render(w) { return super.render(w).map((l, i) => (i === 0 ? "LATE" + l.slice(4) : l)); } }
	const lateFactory = (t2, th, kb) => new Late(t2, th, kb);
	slot = lateFactory; // pi-cc-extensions hace setEditorComponent duro tras el nuestro
	await new Promise((r) => setTimeout(r, 300));
	await h.before_agent_start?.({}, slotCtx);
	if (slot === lateFactory) fails.push("el editor que llega después debe quedar envuelto por el nuestro");
	const ed = slot?.(tui, {}, {});
	if (!(ed instanceof Late)) fails.push("debe envolver el editor que llegó después, no sustituirlo");
	if (!ed?.render(60)[0]?.startsWith("LATE")) fails.push("el render del editor tardío debe conservarse");
	const again = slot;
	await h.agent_end?.({}, slotCtx);
	if (slot !== again) fails.push("si la ranura ya es nuestra no se reinstala (perdería el estado del editor)");
}
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD057EditorWrap: %s", out)
	}
}

// HCD-057 tarea 6: con unidad HCD activa, el input vacío muestra atenuado el
// comando del next_action del motor, sin cambiar el ancho de la línea.
func TestHCD057HCDSuggestion(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-suggest.ts"}, hcd057Harness+`
import { hcdSuggestion } from "./gomemory-suggest.ts";
if (hcdSuggestion({ ok: true, state: "active", unit_id: "HCD-1", next_action: { skill: "hcd-implement" } }) !== "/hcd-implement") fails.push("hcdSuggestion debe dar /hcd-implement");
if (hcdSuggestion({ ok: false }) !== "") fails.push("sin unidad no hay sugerencia del motor");
if (hcdSuggestion({ ok: true, state: "closed", next_action: { skill: "hcd" } }) !== "") fails.push("unidad cerrada no sugiere");
if (hcdSuggestion({ ok: true, state: "active", next_action: { skill: "rm -rf /" } }) !== "") fails.push("solo nombres de skill válidos");

show = { ok: true, state: "active", unit_id: "HCD-057", next_action: { skill: "hcd-implement", why: "x" } };
await h.session_start({}, ctx);
const ed = factory(tui, {}, {});
if (!ghostOf(ed).startsWith("/hcd-implement")) fails.push("input vacío debe mostrar /hcd-implement atenuado: " + JSON.stringify(ed.render(60)[1]));
if (plain(ed.render(60)[1]).length !== plain(new CustomEditor(tui, {}, {}).render(60)[1]).length) fails.push("la sugerencia no debe cambiar el ancho de la línea");
if (completeCalls.length) fails.push("con unidad activa no se llama al modelo");

ed.handleInput("h");
if (ghostOf(ed)) fails.push("con texto en el input no se muestra sugerencia");
ed.handleInput("\x7f");

// Al terminar el turno se relee el motor.
show = { ok: true, state: "active", unit_id: "HCD-057", next_action: { skill: "hcd-close" } };
await h.agent_end({}, ctx);
if (!ghostOf(ed).startsWith("/hcd-close")) fails.push("tras el turno debe sugerir /hcd-close: " + ghostOf(ed));
if (completeCalls.length) fails.push("con unidad activa no se llama al modelo tras el turno");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD057HCDSuggestion: %s", out)
	}
}

// HCD-057 tarea 7: Tab con el input vacío copia la sugerencia sin enviarla; con
// texto, Tab sigue siendo el autocompletado del editor.
func TestHCD057TabAccept(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-suggest.ts"}, hcd057Harness+`
show = { ok: true, state: "active", unit_id: "HCD-057", next_action: { skill: "hcd-implement" } };
await h.session_start({}, ctx);
const ed = factory(tui, {}, {});
let submitted = null;
ed.onSubmit = (t) => (submitted = t);

ed.handleInput("\t");
if (ed.getText() !== "/hcd-implement") fails.push("Tab debe copiar la sugerencia: " + JSON.stringify(ed.getText()));
if (submitted !== null) fails.push("Tab no debe enviar");
if (ed.autocompleted) fails.push("con sugerencia aceptada, Tab no debe llegar al autocompletado");

// Con texto, Tab es del editor.
const ed2 = factory(tui, {}, {});
ed2.handleInput("a"); ed2.handleInput("b"); ed2.handleInput("\t");
if (ed2.getText() !== "ab" || ed2.autocompleted !== 1) fails.push("con texto Tab debe ir al autocompletado: " + ed2.getText() + " " + ed2.autocompleted);

// Aceptada, la sugerencia se consume: vaciar el input no la vuelve a mostrar.
for (let i = 0; i < 20; i++) ed.handleInput("\x7f");
if (ghostOf(ed)) fails.push("la sugerencia aceptada no debe reaparecer: " + ghostOf(ed));
// Sin sugerencia, Tab con input vacío es del editor.
ed.handleInput("\t");
if (ed.autocompleted !== 1) fails.push("sin sugerencia Tab debe ir al editor");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD057TabAccept: %s", out)
	}
}

// HCD-057 tarea 8: sin unidad activa, al terminar el turno se pide al modelo una
// sugerencia breve (tope de tokens); se cancela si la persona escribe o si
// empieza otro turno, y nunca se pide sin modelo ni al arrancar la sesión.
func TestHCD057LLMSuggestion(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-suggest.ts"}, hcd057Harness+`
import { SUGGEST_MAX_TOKENS } from "./gomemory-suggest.ts";
// En modelos con razonamiento el tope incluye lo que piensan: con 40 tokens Opus
// 5.5 no llegaba a escribir (Pi real); con razonamiento mínimo usa 42-103.
if (!(SUGGEST_MAX_TOKENS >= 200 && SUGGEST_MAX_TOKENS <= 400)) fails.push("tope de tokens fuera de rango: " + SUGGEST_MAX_TOKENS);
await h.session_start({}, ctx);
const ed = factory(tui, {}, {});
if (completeCalls.length) fails.push("al arrancar la sesión no se llama al modelo");

completeImpl = async () => ({ stopReason: "stop", content: [{ type: "text", text: "\"Ejecuta los tests del login\"\notra línea" }] });
await h.agent_end({}, ctx);
// Desde HCD-058 (C-002) agent_end no espera a la petición: se espera a que llegue.
await new Promise((r) => setTimeout(r, 20));
const call = completeCalls[0];
if (!call) fails.push("sin unidad, al terminar el turno debe pedirse una sugerencia");
if (call && call.m !== ctx.model) fails.push("debe usar el modelo activo de la sesión");
if (call && !(call.o?.maxTokens > 0 && call.o.maxTokens <= SUGGEST_MAX_TOKENS)) fails.push("debe acotar maxTokens: " + JSON.stringify(call.o));
if (call && !(call.o?.signal instanceof AbortSignal)) fails.push("debe pasar un signal cancelable");
// Omitir «reasoning» no lo apaga en Opus 5.5 (Pi real): se pide el mínimo.
if (call && call.o?.reasoning !== "minimal") fails.push("la petición lateral debe pedir el razonamiento mínimo: " + call.o?.reasoning);
const sent = JSON.stringify(call?.c ?? {});
if (!sent.includes("arregla el login") || !sent.includes("corregí el token")) fails.push("el contexto debe incluir el último intercambio");
if (ghostOf(ed) !== "Ejecuta los tests del login" && !ghostOf(ed).startsWith("Ejecuta los tests del login")) fails.push("debe mostrar la primera línea sin comillas: " + ghostOf(ed));

// Otro turno empieza: la sugerencia anterior desaparece.
await h.before_agent_start?.({}, ctx);
if (ghostOf(ed)) fails.push("al empezar otro turno la sugerencia se retira");

// Escribir durante la petición la cancela y descarta el resultado.
let seenSignal;
completeImpl = (m, c, o) => { seenSignal = o.signal; return new Promise((r) => setTimeout(() => r({ stopReason: "stop", content: [{ type: "text", text: "tarde" }] }), 80)); };
const pending = h.agent_end({}, ctx);
await new Promise((r) => setTimeout(r, 10));
ed.handleInput("x");
await pending;
if (!seenSignal?.aborted) fails.push("escribir debe abortar la petición en curso");
ed.handleInput("\x7f");
if (ghostOf(ed)) fails.push("el resultado tardío no se muestra: " + ghostOf(ed));

// Respuesta cortada por el tope con solo razonamiento: no hay sugerencia.
completeImpl = async () => ({ stopReason: "length", content: [{ type: "thinking", thinking: "pensando…" }] });
await h.agent_end({}, ctx);
await new Promise((r) => setTimeout(r, 20));
if (ghostOf(ed)) fails.push("solo razonamiento no deja sugerencia: " + ghostOf(ed));

// Un fallo del modelo no rompe nada ni muestra sugerencia.
completeImpl = async () => { throw new Error("sin red"); };
await h.agent_end({}, ctx);
await new Promise((r) => setTimeout(r, 20));
if (ghostOf(ed)) fails.push("un error del modelo no deja sugerencia");

// Sin modelo no se pide nada.
const before = completeCalls.length;
await h.agent_end({}, { ...ctx, model: undefined });
if (completeCalls.length !== before) fails.push("sin modelo no se llama");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD057LLMSuggestion: %s", out)
	}
}

package installer

import (
	"strings"
	"testing"
)

// Las tareas 1, 3 y 4 de HCD-058 (reconciliación del texto final, verify
// encadenado y alcance de watch en /tdd) se retiraron con /tdd en HCD-085.

// HCD-058 tarea 2 (C-002): el trabajo de agent_end no debe escribir la
// sugerencia ni pedirla al modelo si mientras tanto empezó otro turno o se
// cerró la sesión; y agent_end no espera a la petición al modelo.
func TestHCD058SuggestEpoch(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-suggest.ts"}, hcd057Harness+`
await h.session_start({}, ctx);
const ed = factory(tui, {}, {});

// a) Motor HCD: otro turno empieza mientras mem hcd show está en vuelo.
show = { ok: true, state: "active", unit_id: "HCD-058", next_action: { skill: "hcd-implement" } };
const a = h.agent_end({}, ctx);
await h.before_agent_start({}, ctx);
await a;
if (ghostOf(ed)) fails.push("a) la sugerencia del turno anterior no debe aparecer en el nuevo: " + ghostOf(ed));

// b) Modelo: otro turno empieza antes de lanzar la petición.
show = { ok: false };
const before = completeCalls.length;
const b = h.agent_end({}, ctx);
await h.before_agent_start({}, ctx);
await b;
await new Promise((r) => setTimeout(r, 20));
if (completeCalls.length !== before) fails.push("b) no debe pedirse al modelo en el turno siguiente");
if (ghostOf(ed)) fails.push("b) sin sugerencia en el turno nuevo");

// c) Sesión cerrada mientras se consultaba el motor.
show = { ok: true, state: "active", unit_id: "HCD-058", next_action: { skill: "hcd-close" } };
const c = h.agent_end({}, ctx);
await h.session_shutdown({}, ctx);
await c;
if (ghostOf(ed)) fails.push("c) tras cerrar la sesión no se escribe sugerencia");

// d) agent_end no espera a la petición al modelo.
show = { ok: false };
completeImpl = () => new Promise(() => {}); // nunca responde
const t0 = Date.now();
const done = await Promise.race([h.agent_end({}, ctx).then(() => "done"), new Promise((r) => setTimeout(() => r("timeout"), 300))]);
if (done !== "done") fails.push("d) agent_end no debe esperar a modelNext (" + (Date.now() - t0) + " ms)");
await h.before_agent_start({}, ctx);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD058SuggestEpoch: %s", out)
	}
}

// HCD-058 tarea 5 (C-004): los temporizadores que session_start programa para
// reenvolver el editor se cancelan al cerrar la sesión.
func TestHCD058SuggestTimers(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-suggest.ts"}, hcd057Harness+`
await h.session_start({}, ctx);
await h.session_shutdown({}, ctx);
factory = null; // cualquier reinstalación posterior vendría de un temporizador vivo
await new Promise((r) => setTimeout(r, 1200));
console.log(factory ? "FAIL un temporizador de session_start reinstaló el editor tras session_shutdown" : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD058SuggestTimers: %s", out)
	}
}

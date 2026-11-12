package installer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// hcd024Sidebar carga el sidebar con un pi simulado que cuenta las consultas a
// anthropic-usage (que responden con el 429 real) y expone los handlers.
const hcd024Sidebar = `
import sidebarExt from "./gomemory-sidebar.ts";
const commands = {}, handlers = {};
let usageCalls = 0;
const pi = {
	registerCommand: (n, d) => (commands[n] = d), registerTool() {},
	on: (e, h) => { handlers[e] = h; return () => {}; },
	exec: async (cmd, args) => {
		if (args && args[0] === "anthropic-usage") {
			usageCalls++;
			return { code: 1, stdout: "", stderr: '{"error": "uso de Anthropic: HTTP 429"}' };
		}
		return { code: 1, stdout: "", stderr: "" };
	},
};
sidebarExt(pi);
const theme = { fg: (_c, s) => s, style: (s) => s };
const tui = { terminal: { rows: 60, columns: 160 }, requestRender() {} };
let comp;
const ctx = {
	hasUI: true, cwd: process.cwd(),
	model: { provider: "anthropic", id: "claude-haiku", contextWindow: 200000 },
	sessionManager: { getBranch: () => [] },
	ui: { notify() {}, custom: (f) => new Promise((r) => { comp = f(tui, theme, {}, r); }) },
};
const settle = () => new Promise((r) => setTimeout(r, 30));
const fails = [];
// Cabeceras REALES de una respuesta de Anthropic vista en after_provider_response (Pi 1.5.2),
// con los reinicios desplazados a ahora + 2h14m y + 6d21h.
const now = Math.floor(Date.now() / 1000);
const realHeaders = {
	"anthropic-ratelimit-unified-5h-reset": String(now + 2 * 3600 + 14 * 60 + 30),
	"anthropic-ratelimit-unified-5h-status": "allowed",
	"anthropic-ratelimit-unified-5h-utilization": "0.72",
	"anthropic-ratelimit-unified-7d-reset": String(now + 6 * 86400 + 21 * 3600 + 30 * 60),
	"anthropic-ratelimit-unified-7d-status": "allowed",
	"anthropic-ratelimit-unified-7d-utilization": "0.1",
	"anthropic-ratelimit-unified-representative-claim": "five_hour",
	"anthropic-ratelimit-unified-status": "allowed",
	"content-type": "text/event-stream",
};
`

func TestHCD024PlanFromHeaders(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, hcd024Sidebar+`
if (!handlers.after_provider_response) { console.log("FAIL el sidebar no escucha after_provider_response"); process.exit(0); }
await handlers.after_provider_response({ status: 200, headers: realHeaders }, ctx);
await commands.sidebar.handler("", ctx);
await settle();
const lines = comp.render(36);
const ses = lines.find((l) => l.includes("Sesión")) || "";
const sem = lines.find((l) => l.includes("Semana")) || "";
if (!ses.includes("72%") || !ses.includes("2h 14m")) fails.push("sesión desde cabeceras: " + JSON.stringify(ses));
if (!sem.includes("10%") || !sem.includes("6d 21h")) fails.push("semana desde cabeceras: " + JSON.stringify(sem));
if (usageCalls !== 0) fails.push("con lectura de cabeceras no debe consultar anthropic-usage (" + usageCalls + " llamadas)");

// Una respuesta sin cabeceras de Anthropic (otro proveedor) no borra la lectura.
await handlers.after_provider_response({ status: 200, headers: { "content-type": "application/json" } }, ctx);
if (!comp.render(36).some((l) => l.includes("72%"))) fails.push("una respuesta sin cabeceras no debe borrar la lectura");

// Una respuesta nueva actualiza los valores.
await handlers.after_provider_response({ status: 200, headers: { ...realHeaders, "anthropic-ratelimit-unified-5h-utilization": "0.8" } }, ctx);
if (!comp.render(36).some((l) => l.includes("Sesión") && l.includes("80%"))) fails.push("la siguiente respuesta debe actualizar la sesión al 80%");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD024PlanFromHeaders: %s", out)
	}
}

func TestHCD024NoPolling(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, hcd024Sidebar+`
await commands.sidebar.handler("", ctx);
await settle();
for (let i = 0; i < 5; i++) {
	await handlers.turn_end?.({}, ctx);
	await handlers.agent_end?.({}, ctx);
	await handlers.model_select?.({}, ctx);
}
await settle();
if (usageCalls > 1) fails.push("abrir y 15 eventos de turno no deben sondear: " + usageCalls + " consultas a anthropic-usage");
// Texto del panel sin bordes: los avisos se reparten en varias líneas.
const text = comp.render(36).map((l) => l.slice(1, -1).trim()).join(" ");
if (/caducad/.test(text)) fails.push("un 429 no debe describirse como sesión caducada");
if (!/próxima respuesta/.test(text)) fails.push("sin lectura, debe indicar que se actualiza con la próxima respuesta");

// Cerrar y reabrir tras el error: no reintenta antes de 5 minutos.
await commands.sidebar.handler("", ctx);
await commands.sidebar.handler("", ctx);
await settle();
if (usageCalls > 1) fails.push("reabrir tras un error no debe reintentar antes de 5 min: " + usageCalls);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD024NoPolling: %s", out)
	}
}

func TestHCD024RateLimitMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("retry-after", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	withAnthropicURL(t, srv.URL)
	_, err := FetchAnthropicUsage(piAuth(t, oauthEntry(time.Now().Add(time.Hour))))
	if err == nil {
		t.Fatal("un 429 debe dar error")
	}
	if strings.Contains(err.Error(), "caducad") {
		t.Fatalf("un 429 no es una sesión caducada: %v", err)
	}
	if !strings.Contains(err.Error(), "429") || !strings.Contains(err.Error(), "frecuencia") {
		t.Fatalf("el error debe explicar el límite de frecuencia: %v", err)
	}
}

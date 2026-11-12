package installer

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// anthropicUsageFixture reproduce la respuesta REAL de /api/oauth/usage
// (capturada el 2026-11-11, campos irrelevantes omitidos) con un límite
// seven_day_opus añadido para cubrir los límites por modelo.
func anthropicUsageFixture(now time.Time) string {
	return fmt.Sprintf(`{"five_hour":{"utilization":52.0,"resets_at":%q,"limit_dollars":null},
"seven_day":{"utilization":7.0,"resets_at":%q,"limit_dollars":null},
"seven_day_oauth_apps":null,"seven_day_opus":{"utilization":9.0,"resets_at":%q},"seven_day_sonnet":null,
"extra_usage":{"is_enabled":false}}`,
		now.Add(2*time.Hour+14*time.Minute).UTC().Format("2006-01-02T15:04:05.000000+00:00"),
		now.Add(4*24*time.Hour+6*time.Hour+30*time.Minute).UTC().Format("2006-01-02T15:04:05.000000+00:00"),
		now.Add(4*24*time.Hour+6*time.Hour+30*time.Minute).UTC().Format("2006-01-02T15:04:05.000000+00:00"))
}

// piAuth escribe un auth.json de Pi con la entrada anthropic indicada.
func piAuth(t *testing.T, anthropic string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "auth.json"), `{"anthropic":`+anthropic+`}`)
	return dir
}

func oauthEntry(expires time.Time) string {
	return fmt.Sprintf(`{"type":"oauth","access":"tok-de-prueba","refresh":"r","expires":%d}`, expires.UnixMilli())
}

func withAnthropicURL(t *testing.T, url string) {
	t.Helper()
	prev := anthropicUsageURL
	anthropicUsageURL = url
	t.Cleanup(func() { anthropicUsageURL = prev })
}

func TestHCD023FetchAnthropicUsage(t *testing.T) {
	var gotAuth, gotBeta string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotBeta = r.Header.Get("Authorization"), r.Header.Get("anthropic-beta")
		_, _ = w.Write([]byte(anthropicUsageFixture(time.Now())))
	}))
	defer srv.Close()
	withAnthropicURL(t, srv.URL)

	u, err := FetchAnthropicUsage(piAuth(t, oauthEntry(time.Now().Add(time.Hour))))
	if err != nil {
		t.Fatalf("FetchAnthropicUsage: %v", err)
	}
	if gotAuth != "Bearer tok-de-prueba" || gotBeta != "oauth-2025-04-20" {
		t.Fatalf("cabeceras: Authorization=%q anthropic-beta=%q", gotAuth, gotBeta)
	}
	want := []struct {
		name    string
		percent float64
		resetIn string
	}{{"Sesión", 52, "2h"}, {"Semana", 7, "4d 6h"}, {"Opus", 9, "4d 6h"}}
	if len(u.Limits) != len(want) {
		t.Fatalf("límites: %+v (Sonnet viene null y no debe aparecer)", u.Limits)
	}
	for i, w := range want {
		l := u.Limits[i]
		if l.Name != w.name || l.Percent != w.percent || !strings.HasPrefix(l.ResetIn, w.resetIn) || l.ResetsAt == "" {
			t.Fatalf("límite %d = %+v; se esperaba %s %.0f%% reinicio %s…", i, l, w.name, w.percent, w.resetIn)
		}
	}
}

func TestHCD023AnthropicUsageErrors(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(anthropicUsageFixture(time.Now())))
	}))
	defer ok.Close()

	cases := []struct {
		name, auth, want string
		handler          http.HandlerFunc
	}{
		{"sin auth.json", "", "OAuth de Anthropic", nil},
		{"clave de API, no OAuth", `{"type":"api_key","key":"x"}`, "OAuth de Anthropic", nil},
		{"token caducado", oauthEntry(time.Now().Add(-time.Minute)), "caducado", nil},
		{"HTTP 401", oauthEntry(time.Now().Add(time.Hour)), "401", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}},
		{"más de 1 MiB", oauthEntry(time.Now().Add(time.Hour)), "1 MiB", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", 2<<20)))
		}},
		{"sin límites en la respuesta", oauthEntry(time.Now().Add(time.Hour)), "sin límites", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"five_hour":null,"seven_day":null}`))
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			url := ok.URL
			if c.handler != nil {
				s := httptest.NewServer(c.handler)
				defer s.Close()
				url = s.URL
			}
			withAnthropicURL(t, url)
			dir := t.TempDir()
			if c.auth != "" {
				dir = piAuth(t, c.auth)
			}
			u, err := FetchAnthropicUsage(dir)
			if err == nil {
				t.Fatalf("se esperaba error y se obtuvo %+v", u)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("el error debe mencionar %q: %v", c.want, err)
			}
			if strings.Contains(err.Error(), "tok-de-prueba") {
				t.Fatalf("el error no debe filtrar el token: %v", err)
			}
		})
	}
}

func TestHCD023CommandRegistered(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	if !strings.Contains(src, `case "anthropic-usage"`) || !strings.Contains(src, "FetchAnthropicUsage(") {
		t.Fatal("main.go debe despachar el comando anthropic-usage a FetchAnthropicUsage")
	}
	if !strings.Contains(src, "pi-gomemory-setup anthropic-usage") {
		t.Fatal("la ayuda de main.go debe listar anthropic-usage")
	}
}

func TestHCD023SidebarPlanLimits(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import { renderSidebar, anthropicPlanErrorMessage } from "./gomemory-sidebar.ts";
const theme = { fg: (_c, s) => s, style: (s) => s };
const base = { title: "t", tokens: 1, contextLimit: 10, cost: 0, servers: [], providerName: "Anthropic", isClaudeProvider: true, claudeUsage: null };
const fails = [];

const plan = { limits: [
	{ name: "Sesión", percent: 52, resetsAt: "x", resetIn: "2h 14m" },
	{ name: "Semana", percent: 7, resetsAt: "x", resetIn: "4d 6h" },
] };
const lines = renderSidebar({ ...base, anthropicPlan: plan }, 36, theme);
const text = lines.join("\n");
if (!text.includes("Anthropic · plan")) fails.push("falta la cabecera «Anthropic · plan»");
const ses = lines.find((l) => l.includes("Sesión")) || "";
const sem = lines.find((l) => l.includes("Semana")) || "";
if (!ses.includes("52%") || !ses.includes("2h 14m") || !ses.includes("█")) fails.push("fila de sesión incompleta: " + ses);
if (!sem.includes("7%") || !sem.includes("4d 6h")) fails.push("fila de semana incompleta: " + sem);
for (const l of lines) if ([...l].length !== 36) fails.push("fila de ancho " + [...l].length + ": " + l);

const errText = renderSidebar({ ...base, anthropicPlanError: "token caducado; abre pi para renovarlo" }, 36, theme).join("\n");
if (!errText.includes("Anthropic · plan") || !errText.includes("caducado")) fails.push("con error debe mostrar la cabecera y el error");
if (!errText.includes("renovarlo")) fails.push("el error largo no debe recortarse: falta la parte accionable");
for (const l of errText.split("\n")) if ([...l].length !== 36) fails.push("fila de error de ancho " + [...l].length + ": " + l);
const upd = renderSidebar({ ...base, anthropicPlanError: anthropicPlanErrorMessage("Comando desconocido: anthropic-usage") }, 36, theme).join("\n");
if (!upd.includes("pi-gomemory-setup update")) fails.push("el aviso de binario antiguo debe verse completo en 36 columnas");
if (/\d+%/.test(errText.split("Anthropic · plan")[1].split("MCP")[0])) fails.push("con error no debe mostrar porcentajes");

// Binario anterior sin el comando (salida real de v1.9.0): aviso accionable.
if (!/pi-gomemory-setup update/.test(anthropicPlanErrorMessage("Comando desconocido: anthropic-usage\n"))) fails.push("un binario sin anthropic-usage debe sugerir pi-gomemory-setup update");
if (anthropicPlanErrorMessage('{"error": "el token OAuth de Anthropic está caducado: abre pi para renovarlo"}') !== "el token OAuth de Anthropic está caducado: abre pi para renovarlo") fails.push("el error JSON del binario debe mostrarse limpio");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD023SidebarPlanLimits: %s", out)
	}
}

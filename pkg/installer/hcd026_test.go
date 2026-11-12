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

func codexUsageFixture(now time.Time) string {
	resetPrimary := now.Add(2*time.Hour + 15*time.Minute).Unix()
	resetSecondary := now.Add(4*24*time.Hour + 8*time.Hour).Unix()
	return fmt.Sprintf(`{
  "plan_type": "pro",
  "rate_limit": {
    "allowed": true,
    "limit_reached": false,
    "primary_window": {
      "used_percent": 24.5,
      "limit_window_seconds": 18000,
      "reset_at": %d
    },
    "secondary_window": {
      "used_percent": 65.0,
      "limit_window_seconds": 604800,
      "reset_at": %d
    }
  }
}`, resetPrimary, resetSecondary)
}

func piAuthCodex(t *testing.T, codexContent string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "auth.json"), `{"openai-codex":`+codexContent+`}`)
	return dir
}

func codexOAuthEntry(expires time.Time, accountID string) string {
	accLine := ""
	if accountID != "" {
		accLine = fmt.Sprintf(`,"accountId":%q`, accountID)
	}
	return fmt.Sprintf(`{"type":"oauth","access":"tok-codex-test","refresh":"r","expires":%d%s}`, expires.UnixMilli(), accLine)
}

func withCodexURL(t *testing.T, url string) {
	t.Helper()
	prev := codexUsageURL
	codexUsageURL = url
	t.Cleanup(func() { codexUsageURL = prev })
}

func TestHCD026FetchCodexUsage(t *testing.T) {
	var gotAuth, gotAccount string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccount = r.Header.Get("ChatGPT-Account-Id")
		_, _ = w.Write([]byte(codexUsageFixture(time.Now())))
	}))
	defer srv.Close()
	withCodexURL(t, srv.URL)

	u, err := FetchCodexUsage(piAuthCodex(t, codexOAuthEntry(time.Now().Add(time.Hour), "acc-12345")))
	if err != nil {
		t.Fatalf("FetchCodexUsage: %v", err)
	}
	if gotAuth != "Bearer tok-codex-test" {
		t.Fatalf("Authorization esperado %q, obtenido %q", "Bearer tok-codex-test", gotAuth)
	}
	if gotAccount != "acc-12345" {
		t.Fatalf("ChatGPT-Account-Id esperado %q, obtenido %q", "acc-12345", gotAccount)
	}
	if u.PlanType != "pro" {
		t.Fatalf("PlanType esperado 'pro', obtenido %q", u.PlanType)
	}
	want := []struct {
		name    string
		percent float64
		resetIn string
	}{
		{"5 Horas", 24.5, "2h 15m"},
		{"Semana", 65.0, "4d 8h"},
	}
	if len(u.Limits) != len(want) {
		t.Fatalf("límites: %+v, esperados %d", u.Limits, len(want))
	}
	for i, w := range want {
		got := u.Limits[i]
		if got.Name != w.name || got.Percent != w.percent {
			t.Errorf("límite %d: %+v, esperado %+v", i, got, w)
		}
		if !strings.HasPrefix(got.ResetIn, strings.Split(w.resetIn, " ")[0]) {
			t.Errorf("límite %d: resetIn=%q no parece coincidir con %q", i, got.ResetIn, w.resetIn)
		}
	}
}

func TestHCD026CodexUsageErrors(t *testing.T) {
	// 1. Sin archivo auth.json
	emptyDir := t.TempDir()
	if _, err := FetchCodexUsage(emptyDir); err == nil || !strings.Contains(err.Error(), "auth.json") {
		t.Fatalf("esperaba error de auth.json no encontrado, obtuve: %v", err)
	}

	// 2. Sin entrada openai-codex
	noCodexDir := t.TempDir()
	writeFile(t, filepath.Join(noCodexDir, "auth.json"), `{"antigravity":{}}`)
	if _, err := FetchCodexUsage(noCodexDir); err == nil || !strings.Contains(err.Error(), "openai-codex") {
		t.Fatalf("esperaba error de falta de openai-codex, obtuve: %v", err)
	}

	// 3. Token caducado
	expiredDir := piAuthCodex(t, codexOAuthEntry(time.Now().Add(-time.Hour), ""))
	if _, err := FetchCodexUsage(expiredDir); err == nil || !strings.Contains(err.Error(), "caducado") {
		t.Fatalf("esperaba error de token caducado, obtuve: %v", err)
	}

	// 4. Error HTTP 401
	srv401 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv401.Close()
	withCodexURL(t, srv401.URL)
	validDir := piAuthCodex(t, codexOAuthEntry(time.Now().Add(time.Hour), ""))
	if _, err := FetchCodexUsage(validDir); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("esperaba error con 401, obtuve: %v", err)
	}

	// 5. Error HTTP 429
	srv429 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv429.Close()
	withCodexURL(t, srv429.URL)
	if _, err := FetchCodexUsage(validDir); err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("esperaba error con 429, obtuve: %v", err)
	}
}

func TestHCD026CommandRegistered(t *testing.T) {
	// Verifica que main.go acepte "codex-usage"
	mainSrc, err := os.ReadFile(filepath.Join("..", "..", "main.go"))
	if err != nil {
		t.Fatalf("leer main.go: %v", err)
	}
	src := string(mainSrc)
	if !strings.Contains(src, `case "codex-usage":`) {
		t.Errorf("main.go no tiene 'case \"codex-usage\":'")
	}
	if !strings.Contains(src, "codex-usage") {
		t.Errorf("main.go no menciona 'codex-usage' en la ayuda")
	}
}

func TestHCD026SidebarCodexQuota(t *testing.T) {
	sidebarSrc, err := os.ReadFile(filepath.Join("..", "..", "extensions", "gomemory-sidebar.ts"))
	if err != nil {
		t.Fatalf("leer extensions/gomemory-sidebar.ts: %v", err)
	}
	src := string(sidebarSrc)
	if !strings.Contains(src, "codexQuota") && !strings.Contains(src, "isCodexProvider") {
		t.Errorf("gomemory-sidebar.ts no tiene soporte de estado o cuotas para Codex")
	}
	if !strings.Contains(src, "Codex Quota") {
		t.Errorf("gomemory-sidebar.ts no renderiza el título 'Codex Quota'")
	}
}

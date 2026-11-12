package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var codexUsageURL = "https://chatgpt.com/backend-api/wham/usage"

const maxCodexUsageBody = 1 << 20

// CodexLimit representa una ventana de límite de uso de Codex.
type CodexLimit struct {
	Name     string  `json:"name"`
	Percent  float64 `json:"percent"`
	ResetsAt string  `json:"resetsAt"`
	ResetIn  string  `json:"resetIn"`
}

// CodexUsageSummary es la respuesta estructurada devuelta por codex-usage.
type CodexUsageSummary struct {
	PlanType string       `json:"planType"`
	Limits   []CodexLimit `json:"limits"`
}

type codexWindow struct {
	UsedPercent        *float64 `json:"used_percent"`
	LimitWindowSeconds *int64   `json:"limit_window_seconds"`
	ResetAt            *int64   `json:"reset_at"`
}

type codexRateLimit struct {
	Allowed         *bool        `json:"allowed"`
	LimitReached    *bool        `json:"limit_reached"`
	PrimaryWindow   *codexWindow `json:"primary_window"`
	SecondaryWindow *codexWindow `json:"secondary_window"`
}

type codexUsageResponse struct {
	PlanType  string          `json:"plan_type"`
	RateLimit *codexRateLimit `json:"rate_limit"`
}

// FetchCodexUsage lee el token y accountId OAuth de OpenAI Codex desde auth.json
// y consulta https://chatgpt.com/backend-api/wham/usage para devolver los límites limpios.
func FetchCodexUsage(piAgentDir string) (*CodexUsageSummary, error) {
	token, accountID, err := codexOAuthToken(piAgentDir, time.Now())
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, codexUsageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "pi-gomemory-setup")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("consultar el uso de Codex: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCodexUsageBody+1))
	if err != nil {
		return nil, fmt.Errorf("leer el uso de Codex: %w", err)
	}
	if len(body) > maxCodexUsageBody {
		return nil, errors.New("respuesta de uso de Codex mayor que 1 MiB; se descarta")
	}

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, errors.New("uso de Codex: HTTP 429, límite de frecuencia alcanzado")
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("uso de Codex: HTTP %d, sesión no válida: inicia sesión en Pi con /login", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("uso de Codex: HTTP %d", resp.StatusCode)
	}

	var r codexUsageResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("respuesta de uso de Codex ilegible: %w", err)
	}

	summary := &CodexUsageSummary{
		PlanType: r.PlanType,
		Limits:   make([]CodexLimit, 0, 2),
	}

	if r.RateLimit != nil {
		now := time.Now()
		// Ventana primaria (ej. 5 horas)
		if w := r.RateLimit.PrimaryWindow; w != nil && w.UsedPercent != nil {
			name := "5 Horas"
			if w.LimitWindowSeconds != nil && *w.LimitWindowSeconds > 86400 {
				name = "Primaria"
			}
			summary.Limits = append(summary.Limits, buildCodexLimit(name, *w.UsedPercent, w.ResetAt, now))
		}
		// Ventana secundaria (ej. Semanal)
		if w := r.RateLimit.SecondaryWindow; w != nil && w.UsedPercent != nil {
			name := "Semana"
			if w.LimitWindowSeconds != nil && *w.LimitWindowSeconds <= 86400 {
				name = "Secundaria"
			}
			summary.Limits = append(summary.Limits, buildCodexLimit(name, *w.UsedPercent, w.ResetAt, now))
		}
	}

	if len(summary.Limits) == 0 {
		return nil, errors.New("la respuesta de Codex llegó sin ventanas de uso")
	}

	return summary, nil
}

func buildCodexLimit(name string, usedPercent float64, resetAt *int64, now time.Time) CodexLimit {
	resetsAtStr := ""
	resetInStr := ""
	if resetAt != nil && *resetAt > 0 {
		t := time.Unix(*resetAt, 0).UTC()
		resetsAtStr = t.Format(time.RFC3339)
		diff := t.Sub(now)
		if diff <= 0 {
			resetInStr = "now"
		} else {
			mins := int(math.Floor(diff.Minutes()))
			days := mins / 1440
			hours := (mins % 1440) / 60
			remainMins := mins % 60
			if days > 0 {
				resetInStr = fmt.Sprintf("%dd %dh", days, hours)
			} else if hours > 0 {
				resetInStr = fmt.Sprintf("%dh %dm", hours, remainMins)
			} else {
				resetInStr = fmt.Sprintf("%dm", mins)
			}
		}
	}
	return CodexLimit{
		Name:     name,
		Percent:  usedPercent,
		ResetsAt: resetsAtStr,
		ResetIn:  resetInStr,
	}
}

func codexOAuthToken(piAgentDir string, now time.Time) (token string, accountID string, err error) {
	if piAgentDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		piAgentDir = filepath.Join(home, ".pi", "agent")
	}

	path := filepath.Join(piAgentDir, "auth.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("Pi no tiene una sesión OAuth de Codex (%s): inicia sesión con /login en pi", path)
	}

	var auth struct {
		OpenAICodex *struct {
			Type      string `json:"type"`
			Access    string `json:"access"`
			Expires   int64  `json:"expires"`
			AccountID string `json:"accountId"`
		} `json:"openai-codex"`
	}
	if err := json.Unmarshal(data, &auth); err != nil {
		return "", "", fmt.Errorf("%s ilegible: %w", path, err)
	}

	c := auth.OpenAICodex
	if c == nil || c.Type != "oauth" || c.Access == "" {
		return "", "", errors.New("Pi no tiene una sesión OAuth de openai-codex: inicia sesión con /login en pi")
	}
	if c.Expires > 0 && now.UnixMilli() >= c.Expires {
		return "", "", errors.New("el token OAuth de openai-codex está caducado: abre pi para renovarlo")
	}

	return c.Access, c.AccountID, nil
}

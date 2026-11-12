package installer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// anthropicUsageURL es el endpoint de uso del plan que consultan claude.ai y el
// /usage de Claude Code (variable para los tests). No está documentado
// oficialmente: si cambia, el comando devuelve error y nunca cifras inventadas.
var anthropicUsageURL = "https://api.anthropic.com/api/oauth/usage"

const (
	anthropicOAuthBeta    = "oauth-2025-04-20"
	maxAnthropicUsageBody = 1 << 20
)

// AnthropicLimit es un límite del plan: porcentaje usado y cuándo se reinicia.
type AnthropicLimit struct {
	Name     string  `json:"name"`
	Percent  float64 `json:"percent"`
	ResetsAt string  `json:"resetsAt"`
	ResetIn  string  `json:"resetIn"`
}

// AnthropicUsage es el JSON que consume el sidebar.
type AnthropicUsage struct {
	Limits []AnthropicLimit `json:"limits"`
}

type anthropicWindow struct {
	Utilization *float64 `json:"utilization"`
	ResetsAt    string   `json:"resets_at"`
}

type anthropicUsageResponse struct {
	FiveHour       *anthropicWindow `json:"five_hour"`
	SevenDay       *anthropicWindow `json:"seven_day"`
	SevenDayOpus   *anthropicWindow `json:"seven_day_opus"`
	SevenDaySonnet *anthropicWindow `json:"seven_day_sonnet"`
}

// FetchAnthropicUsage lee el token OAuth de Anthropic que guarda Pi en
// auth.json y consulta los límites del plan: la sesión (ventana de 5 h), la
// semana y, si Anthropic los envía, los límites semanales por modelo.
// No refresca el token: eso lo hace Pi, y rotar el refresh token podría
// cerrarle la sesión. Cualquier fallo es un error explícito.
func FetchAnthropicUsage(piAgentDir string) (*AnthropicUsage, error) {
	token, err := anthropicOAuthToken(piAgentDir, time.Now())
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, anthropicUsageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", anthropicOAuthBeta)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "pi-gomemory-setup")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("consultar el uso de Anthropic: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAnthropicUsageBody+1))
	if err != nil {
		return nil, fmt.Errorf("leer el uso de Anthropic: %w", err)
	}
	if len(body) > maxAnthropicUsageBody {
		return nil, errors.New("respuesta de uso de Anthropic mayor que 1 MiB; se descarta")
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		// Límite de frecuencia del endpoint, no un problema de sesión (HCD-024).
		return nil, errors.New("uso de Anthropic: HTTP 429, límite de frecuencia de consulta; se actualizará con la próxima respuesta del modelo")
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("uso de Anthropic: HTTP %d, sesión no válida: abre pi para renovarla", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("uso de Anthropic: HTTP %d", resp.StatusCode)
	}

	var r anthropicUsageResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("respuesta de uso de Anthropic ilegible: %w", err)
	}

	usage := &AnthropicUsage{}
	for _, w := range []struct {
		name string
		win  *anthropicWindow
	}{{"Sesión", r.FiveHour}, {"Semana", r.SevenDay}, {"Opus", r.SevenDayOpus}, {"Sonnet", r.SevenDaySonnet}} {
		if w.win == nil || w.win.Utilization == nil {
			continue // Anthropic no envía este límite en el plan actual
		}
		usage.Limits = append(usage.Limits, AnthropicLimit{
			Name:     w.name,
			Percent:  *w.win.Utilization,
			ResetsAt: w.win.ResetsAt,
			ResetIn:  formatResetTime(w.win.ResetsAt),
		})
	}
	if len(usage.Limits) == 0 {
		return nil, errors.New("la respuesta de Anthropic llegó sin límites de sesión ni de semana")
	}
	return usage, nil
}

// anthropicOAuthToken devuelve el access token OAuth vigente de auth.json.
func anthropicOAuthToken(piAgentDir string, now time.Time) (string, error) {
	path := filepath.Join(piAgentDir, "auth.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("Pi no tiene una sesión OAuth de Anthropic (%s): inicia sesión con /login en pi", path)
	}
	var auth struct {
		Anthropic *struct {
			Type    string `json:"type"`
			Access  string `json:"access"`
			Expires int64  `json:"expires"`
		} `json:"anthropic"`
	}
	if err := json.Unmarshal(data, &auth); err != nil {
		return "", fmt.Errorf("%s ilegible: %w", path, err)
	}
	a := auth.Anthropic
	if a == nil || a.Type != "oauth" || a.Access == "" {
		return "", errors.New("Pi no tiene una sesión OAuth de Anthropic (los límites del plan solo existen con Claude Pro/Max): inicia sesión con /login en pi")
	}
	if a.Expires > 0 && now.UnixMilli() >= a.Expires {
		return "", errors.New("el token OAuth de Anthropic está caducado: abre pi para renovarlo")
	}
	return a.Access, nil
}

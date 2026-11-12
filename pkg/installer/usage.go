package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// AntigravityAccounts representa el archivo ~/.pi/agent/antigravity-accounts.json
type AntigravityAccounts struct {
	ActiveAccountId string `json:"activeAccountId"`
	Accounts        map[string]struct {
		Access    string `json:"access"`
		Refresh   string `json:"refresh"`
		ProjectId string `json:"projectId"`
		Email     string `json:"email"`
		Expires   int64  `json:"expires"`
	} `json:"accounts"`
}

type QuotaSummaryResponse struct {
	PlanLabel   string `json:"planLabel"`
	Description string `json:"description"`
	Groups      []struct {
		DisplayName string `json:"displayName"`
		Buckets     []struct {
			DisplayName       string  `json:"displayName"`
			RemainingFraction float64 `json:"remainingFraction"`
			ResetTime         string  `json:"resetTime"`
		} `json:"buckets"`
	} `json:"groups"`
}

var endpoints = []string{
	"https://daily-cloudcode-pa.googleapis.com",
	"https://daily-cloudcode-pa.sandbox.googleapis.com",
	"https://cloudcode-pa.googleapis.com",
}

// QuotaCleanGroup representa un grupo de cuotas simplificado para JSON
type QuotaCleanGroup struct {
	DisplayName string             `json:"displayName"`
	Buckets     []QuotaCleanBucket `json:"buckets"`
}

// QuotaCleanBucket representa un bucket de cuota procesado
type QuotaCleanBucket struct {
	DisplayName       string  `json:"displayName"`
	RemainingFraction float64 `json:"remainingFraction"`
	RemainingPercent  float64 `json:"remainingPercent"`
	ResetTime         string  `json:"resetTime"`
	ResetIn           string  `json:"resetIn"`
}

// QuotaCleanData es el payload JSON que consume el sidebar u otras herramientas
type QuotaCleanData struct {
	PlanLabel string            `json:"planLabel"`
	Email     string            `json:"email,omitempty"`
	Groups    []QuotaCleanGroup `json:"groups"`
}

// FetchQuotaSummary consulta las cuotas reales de Antigravity estructuradas
func FetchQuotaSummary(piAgentDir string) (*QuotaCleanData, error) {
	accPath := filepath.Join(piAgentDir, "antigravity-accounts.json")
	data, err := os.ReadFile(accPath)
	if err != nil {
		return nil, fmt.Errorf("no se encontró cuenta de Antigravity en %s", accPath)
	}

	var accs AntigravityAccounts
	if err := json.Unmarshal(data, &accs); err != nil {
		return nil, fmt.Errorf("parsear accounts: %w", err)
	}

	account, ok := accs.Accounts[accs.ActiveAccountId]
	if !ok || account.Access == "" {
		return nil, fmt.Errorf("cuenta activa %q no tiene token de acceso", accs.ActiveAccountId)
	}

	var lastErr error
	for _, ep := range endpoints {
		body, err := fetchQuota(ep, account.Access)
		if err != nil {
			lastErr = err
			continue
		}
		var quota QuotaSummaryResponse
		if err := json.Unmarshal(body, &quota); err != nil {
			lastErr = fmt.Errorf("respuesta de cuota ilegible en %s: %w", ep, err)
			continue
		}

		plan := quota.PlanLabel
		if plan == "" {
			plan = "Google AI Pro (g1-pro-tier)"
		}

		clean := &QuotaCleanData{
			PlanLabel: plan,
			Email:     account.Email,
			Groups:    make([]QuotaCleanGroup, 0, len(quota.Groups)),
		}

		for _, g := range quota.Groups {
			cg := QuotaCleanGroup{
				DisplayName: g.DisplayName,
				Buckets:     make([]QuotaCleanBucket, 0, len(g.Buckets)),
			}
			for _, b := range g.Buckets {
				remPct := math.Round(b.RemainingFraction*1000) / 10
				cg.Buckets = append(cg.Buckets, QuotaCleanBucket{
					DisplayName:       b.DisplayName,
					RemainingFraction: b.RemainingFraction,
					RemainingPercent:  remPct,
					ResetTime:         b.ResetTime,
					ResetIn:           formatResetTime(b.ResetTime),
				})
			}
			clean.Groups = append(clean.Groups, cg)
		}

		return clean, nil
	}

	return nil, fmt.Errorf("no se pudo obtener la cuota: %w", lastErr)
}

// FetchAndFormatUsage consulta las cuotas reales de Antigravity (Google AI Pro) y las formatea.
func FetchAndFormatUsage(piAgentDir string) (string, error) {
	accPath := filepath.Join(piAgentDir, "antigravity-accounts.json")
	data, err := os.ReadFile(accPath)
	if err != nil {
		return "", fmt.Errorf("no se encontró cuenta de Antigravity en %s", accPath)
	}

	var accs AntigravityAccounts
	if err := json.Unmarshal(data, &accs); err != nil {
		return "", fmt.Errorf("parsear accounts: %w", err)
	}

	account, ok := accs.Accounts[accs.ActiveAccountId]
	if !ok || account.Access == "" {
		return "", fmt.Errorf("cuenta activa %q no tiene token de acceso", accs.ActiveAccountId)
	}

	// Consultar retrieveUserQuotaSummary; sin respuesta real se devuelve error,
	// nunca cifras de relleno.
	var lastErr error
	for _, ep := range endpoints {
		body, err := fetchQuota(ep, account.Access)
		if err != nil {
			lastErr = err
			continue
		}
		var quota QuotaSummaryResponse
		if err := json.Unmarshal(body, &quota); err != nil {
			lastErr = fmt.Errorf("respuesta de cuota ilegible en %s: %w", ep, err)
			continue
		}
		return renderQuotaView(&quota, account.Email), nil
	}

	return "", fmt.Errorf("no se pudo obtener la cuota (¿token caducado? abre pi para renovarlo): %w", lastErr)
}

// maxQuotaBody acota la respuesta de cuota (C-005), como update.go y deps.go.
const maxQuotaBody = 1 << 20

// fetchQuota consulta un endpoint de cuota y devuelve el cuerpo si responde 200.
func fetchQuota(endpoint, accessToken string) ([]byte, error) {
	req, err := http.NewRequest("POST", endpoint+"/v1internal:retrieveUserQuotaSummary", bytes.NewBufferString("{}"))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "antigravity/cli/1.4.0 (aidev_client; os_type=linux; arch=amd64; cl=982146307; auth_method=consumer)")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Se lee un byte de más para distinguir «cabe justo» de «excede»: truncar en
	// silencio daría un JSON ilegible en vez de un error que explique la causa.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxQuotaBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxQuotaBody {
		return nil, fmt.Errorf("%s: respuesta de cuota mayor que 1 MiB; se descarta", endpoint)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: status %d", endpoint, resp.StatusCode)
	}
	return body, nil
}

func renderQuotaView(q *QuotaSummaryResponse, email string) string {
	var buf bytes.Buffer

	plan := q.PlanLabel
	if plan == "" {
		plan = "Google AI Pro (g1-pro-tier)"
	}

	buf.WriteString(fmt.Sprintf("%s%s%s\n", ColorBrillo, plan, ColorReset))
	if email != "" {
		buf.WriteString(fmt.Sprintf("%sCuenta: %s%s\n", ColorMuted, email, ColorReset))
	}
	buf.WriteString("\n")

	for _, group := range q.Groups {
		buf.WriteString(fmt.Sprintf(" %s%s%s\n", ColorCyan, group.DisplayName, ColorReset))
		for _, b := range group.Buckets {
			remPct := math.Round(b.RemainingFraction*1000) / 10
			bar := progressBar(b.RemainingFraction, 20)
			reset := formatResetTime(b.ResetTime)

			barColor := ColorBrillo
			if remPct < 20 {
				barColor = ColorError
			} else if remPct < 50 {
				barColor = ColorWarning
			}

			buf.WriteString(fmt.Sprintf("   %s%s%s %s: %.1f%% left · resets %s\n",
				barColor, bar, ColorReset, b.DisplayName, remPct, reset))
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

func progressBar(rem float64, width int) string {
	filled := int(math.Round(rem * float64(width)))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	var b bytes.Buffer
	b.WriteString("[")
	for i := 0; i < filled; i++ {
		b.WriteString("#")
	}
	for i := 0; i < empty; i++ {
		b.WriteString("-")
	}
	b.WriteString("]")
	return b.String()
}

func formatResetTime(resetStr string) string {
	if resetStr == "" {
		return "n/a"
	}
	t, err := time.Parse(time.RFC3339, resetStr)
	if err != nil {
		return resetStr
	}
	diff := time.Until(t)
	if diff <= 0 {
		return "now"
	}
	days := int(diff.Hours()) / 24
	hours := int(diff.Hours()) % 24
	mins := int(diff.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

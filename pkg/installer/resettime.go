package installer

import (
	"fmt"
	"time"
)

// formatResetTime convierte una marca RFC3339 en el tiempo que falta para el
// reinicio de un límite («4d 6h», «2h 14m», «9m»). Vivía en usage.go hasta que
// HCD-085 retiró la cuota de Antigravity; la usa el uso de Anthropic.
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

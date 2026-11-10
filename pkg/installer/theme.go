package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PiTheme representa el esquema del tema para Pi Coding Agent.
type PiTheme struct {
	Schema     string            `json:"$schema,omitempty"`
	Name       string            `json:"name"`
	Appearance string            `json:"appearance"`
	Vars       map[string]string `json:"vars,omitempty"`
	Colors     map[string]string `json:"colors"`
	Export     map[string]string `json:"export,omitempty"`
}

// BuildGomemoryTheme construye la estructura del tema gomemory para Pi.
func BuildGomemoryTheme() *PiTheme {
	return &PiTheme{
		Schema:     "https://raw.githubusercontent.com/earendil-works/pi/main/packages/coding-agent/src/modes/interactive/theme/theme-schema.json",
		Name:       "gomemory",
		Appearance: "dark",
		Vars: map[string]string{
			"abismo":    "#061328",
			"brillo":    "#22e3f7",
			"cian":      "#06c1ee",
			"azur":      "#2c8ff0",
			"indigo":    "#3926e3",
			"violeta":   "#6b56d4",
			"text":      "#e5f4ff",
			"muted":     "#91a9c5",
			"dim":       "#53647e",
			"selection": "#132e4c",
			"warning":   "#f4c66b",
			"error":     "#ff777f",
			"cardBg":    "#0a1c38",
			"panelBg":   "#0c1f3d",
		},
		Colors: map[string]string{
			"accent":             "brillo",
			"border":             "azur",
			"borderAccent":       "brillo",
			"borderMuted":        "selection",
			"success":            "brillo",
			"error":              "error",
			"warning":            "warning",
			"muted":              "muted",
			"dim":                "dim",
			"text":               "text",
			"thinkingText":       "muted",
			"selectedBg":         "selection",
			"scrollbarTrack":     "#0a1a30",
			"scrollbarThumb":     "dim",
			"searchMatchBg":      "#183b63",
			"searchMatchText":    "brillo",
			"userMessageBg":      "panelBg",
			"userMessageText":    "text",
			"customMessageBg":    "#161540",
			"customMessageText":  "muted",
			"customMessageLabel": "violeta",
			"toolPendingBg":      "panelBg",
			"toolSuccessBg":      "#062533",
			"toolErrorBg":        "#2b0e14",
			"toolTitle":          "text",
			"toolOutput":         "muted",
			"mdHeading":          "brillo",
			"mdLink":             "azur",
			"mdLinkUrl":          "muted",
			"mdCode":             "violeta",
			"mdCodeBlock":        "brillo",
			"mdCodeBlockBorder":  "selection",
			"mdQuote":            "muted",
			"mdQuoteBorder":      "azur",
			"mdHr":               "selection",
			"mdListBullet":       "cian",
			"toolDiffAdded":      "brillo",
			"toolDiffRemoved":    "error",
			"toolDiffContext":    "muted",
			"syntaxComment":      "dim",
			"syntaxKeyword":      "azur",
			"syntaxFunction":     "brillo",
			"syntaxVariable":     "cian",
			"syntaxString":       "warning",
			"syntaxNumber":       "brillo",
			"syntaxType":         "violeta",
			"syntaxOperator":     "cian",
			"syntaxPunctuation":  "muted",
			"thinkingOff":        "dim",
			"thinkingMinimal":    "indigo",
			"thinkingLow":        "azur",
			"thinkingMedium":     "violeta",
			"thinkingHigh":       "cian",
			"thinkingXhigh":      "brillo",
			"thinkingMax":        "warning",
			"bashMode":           "cian",
		},
		Export: map[string]string{
			"pageBg": "#061328",
			"cardBg": "#0c1f3d",
			"infoBg": "#132e4c",
		},
	}
}

// InstallTheme escribe el tema gomemory en la carpeta de temas de Pi y en la local.
func InstallTheme(piAgentDir string) error {
	theme := BuildGomemoryTheme()
	data, err := json.MarshalIndent(theme, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar tema: %w", err)
	}

	// 1. Guardar en ~/.pi/agent/themes/gomemory.json
	themesDir := filepath.Join(piAgentDir, "themes")
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return fmt.Errorf("crear dir themes en pi: %w", err)
	}

	destGlobal := filepath.Join(themesDir, "gomemory.json")
	if err := os.WriteFile(destGlobal, data, 0644); err != nil {
		return fmt.Errorf("escribir tema global: %w", err)
	}

	// 2. Activar tema en ~/.pi/agent/settings.json
	settingsPath := filepath.Join(piAgentDir, "settings.json")
	var settings map[string]any
	if raw, err := os.ReadFile(settingsPath); err == nil {
		_ = json.Unmarshal(raw, &settings)
	}
	if settings == nil {
		settings = make(map[string]any)
	}
	settings["theme"] = "gomemory"

	settingsData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar settings: %w", err)
	}
	if err := os.WriteFile(settingsPath, settingsData, 0644); err != nil {
		return fmt.Errorf("actualizar settings con tema: %w", err)
	}

	return nil
}

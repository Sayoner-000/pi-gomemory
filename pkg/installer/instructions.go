package installer

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Versiones del protocolo que pi-gomemory llegó a publicar en
// ~/.pi/agent/AGENTS.md, extraídas tal cual de git (HCD-045). Ya no se
// instalan: el contexto llega en vivo con la extensión gomemory-context.ts.
// Solo sirven para reconocer y retirar el formato antiguo sin marcadores, que
// exige coincidencia exacta para no tocar nunca texto ajeno.
var (
	//go:embed legacy/protocol-1.txt
	legacyProtocolV1 string // b47f46f/fb00282, 2026-11-09
	//go:embed legacy/protocol-2.txt
	legacyProtocolV2 string // f4a50ae…49cbbe9 (v1.14.0), con Splash
)

// legacyPiProtocols, de la más reciente a la más antigua.
var legacyPiProtocols = []string{legacyProtocolV2, legacyProtocolV1}

// legacyPiInstructions es la última versión publicada.
var legacyPiInstructions = legacyProtocolV2

// findLegacyProtocol localiza en content una versión publicada del protocolo
// sin marcadores y devuelve su posición y su texto.
func findLegacyProtocol(content string) (int, string, bool) {
	for _, p := range legacyPiProtocols {
		if i := strings.Index(content, p); i >= 0 {
			return i, p, true
		}
	}
	return -1, "", false
}

const (
	instructionsStart = "<!-- pi-gomemory:start -->"
	instructionsEnd   = "<!-- pi-gomemory:end -->"
)

// errAmbiguousAgents indica que AGENTS.md tiene marcadores ambiguos y se
// conserva entero (HCD-047, Q2).
var errAmbiguousAgents = errors.New(reasonAmbiguousAgents)

// RetireLegacyInstructions quita de ~/.pi/agent/AGENTS.md los bloques que
// escribían las instalaciones anteriores a HCD-045 y nunca escribe uno nuevo.
// Devuelve true si había bloque. Sin AGENTS.md o sin bloque no toca nada; con
// marcadores ambiguos tampoco, y devuelve errAmbiguousAgents.
func RetireLegacyInstructions(piAgentDir string) (bool, error) {
	path := filepath.Join(piAgentDir, "AGENTS.md")
	target, err := configTarget(path)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(target)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("leer %s: %w", target, err)
	}
	if ambiguousMarkers(string(data)) {
		return false, errAmbiguousAgents
	}
	out, ok := stripInstructions(string(data))
	if !ok {
		return false, nil
	}
	if err := writeStrippedAgents(path, target, out); err != nil {
		return false, fmt.Errorf("retirar el protocolo heredado de %s: %w", path, err)
	}
	return true, nil
}

// writeStrippedAgents escribe AGENTS.md ya sin el bloque de pi-gomemory. Lo
// comparten install y uninstall (HCD-045, Q4): si no queda texto, un archivo
// normal se borra y un enlace se conserva con su destino vacío (HCD-036); en
// cualquier otro caso se conservan los permisos (HCD-032).
func writeStrippedAgents(path, target, out string) error {
	if strings.TrimSpace(out) == "" {
		if target == path {
			return os.Remove(path)
		}
		out = ""
	}
	mode, err := modeOrDefault(target, 0644)
	if err != nil {
		return err
	}
	return writeFileAtomic(target, []byte(out), mode)
}

// retireLegacyStep es el paso de install. Un AGENTS.md enlazado que no lleva a
// un archivo aborta la instalación, como el resto de configuraciones de Pi
// (HCD-036, Q6). Cualquier otro fallo solo se avisa y la instalación sigue
// (HCD-045, C-07); doctor lo seguirá avisando.
func retireLegacyStep(piAgentDir string, w io.Writer) error {
	retired, err := RetireLegacyInstructions(piAgentDir)
	var link *linkNotFileError
	switch {
	case errors.As(err, &link):
		return err
	case errors.Is(err, errAmbiguousAgents):
		fmt.Fprintf(w, "%s    ⚠ AGENTS.md: %s%s\n", ColorWarning, reasonAmbiguousAgents, ColorReset)
	case err != nil:
		fmt.Fprintf(w, "%s    ⚠ No se pudo retirar el protocolo heredado de AGENTS.md: %v. Corrige el permiso y ejecuta pi-gomemory-setup install.%s\n", ColorWarning, err, ColorReset)
	case retired:
		fmt.Fprintf(w, "%s    ✓ Protocolo heredado retirado de AGENTS.md: el contexto llega en vivo con la extensión gomemory-context%s\n", ColorMuted, ColorReset)
	}
	return nil
}

// legacyAgentsItem es la comprobación de doctor: avisa si AGENTS.md conserva
// el bloque heredado y devuelve nil si no hay nada que avisar.
func legacyAgentsItem(piAgentDir string) *CheckItem {
	target, err := configTarget(filepath.Join(piAgentDir, "AGENTS.md"))
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return nil
	}
	if ambiguousMarkers(string(data)) {
		return &CheckItem{Name: "AGENTS.md de Pi", Status: "WARN", Details: "AGENTS.md tiene " + reasonAmbiguousAgents}
	}
	if _, ok := instructionsBlock(string(data)); !ok {
		return nil
	}
	return &CheckItem{
		Name:    "AGENTS.md de Pi",
		Status:  "WARN",
		Details: "AGENTS.md conserva el protocolo heredado de pi-gomemory; ejecuta pi-gomemory-setup install para retirarlo",
	}
}

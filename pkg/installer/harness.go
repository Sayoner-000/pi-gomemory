package installer

import (
	"fmt"
	"os"
	"path/filepath"
)

// GenerateHarnessScript genera el contenido del script lanzador pi-gomemory para Herdr.
func GenerateHarnessScript() string {
	return `#!/usr/bin/env bash
# ==============================================================================
# pi-gomemory — Harness para Pi Coding Agent en Herdr con gomemory
# ==============================================================================
set -euo pipefail

# 1. Asegurar entorno de terminal e integración con Herdr
export HERDR_ENV="${HERDR_ENV:-1}"
export COLORTERM="${COLORTERM:-truecolor}"
export TERM="${TERM:-xterm-256color}"
export GOMEMORY_THEME="${GOMEMORY_THEME:-dark}"
export GOMEMORY_CLIENT="pi"

# 2. Asegurar acceso prioritario a binarios locales
export PATH="${HOME}/.local/bin:${HOME}/.gomemory/bin:${PATH}"

# 3. Subcomando rápido para ver cuota y consumo de Google AI Pro / Antigravity
if [ "${1:-}" = "usage" ] || [ "${1:-}" = "quota" ] || [ "${1:-}" = "/antigravity.usage" ]; then
  if command -v pi-gomemory-setup &>/dev/null; then
    exec pi-gomemory-setup usage
  fi
fi

# 3. Comprobar que pi y mem estén presentes
if ! command -v pi &>/dev/null; then
  echo -e "\033[38;2;255;119;127m[ERROR]\033[0m No se encontró 'pi' en PATH." >&2
  exit 1
fi

# 4. Cargar contexto de gomemory si estamos en un proyecto inicializado
GOMEMORY_ARGS=()
if [ -f ".gomemory/memories.db" ] && command -v mem &>/dev/null; then
  CTX="$(mem context 2>/dev/null || true)"
  if [ -n "${CTX}" ]; then
    GOMEMORY_ARGS+=(--append-system-prompt "${CTX}")
  fi
fi

# 5. Ejecutar Pi con el tema gomemory
if [ ${#GOMEMORY_ARGS[@]} -gt 0 ]; then
  exec pi --use-theme "gomemory" "${GOMEMORY_ARGS[@]}" "$@"
else
  exec pi --use-theme "gomemory" "$@"
fi
`
}

// InstallHarness instala el ejecutable pi-gomemory en localBinDir (~/.local/bin).
func InstallHarness(localBinDir string) error {
	if err := os.MkdirAll(localBinDir, 0755); err != nil {
		return fmt.Errorf("crear dir local bin %s: %w", localBinDir, err)
	}
	targetScript := filepath.Join(localBinDir, "pi-gomemory")
	if err := os.WriteFile(targetScript, []byte(GenerateHarnessScript()), 0755); err != nil {
		return fmt.Errorf("instalar script en %s: %w", targetScript, err)
	}
	return nil
}

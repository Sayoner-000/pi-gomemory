#!/usr/bin/env bash
# ==============================================================================
# HCD-062 · Validación real del instalador en una máquina "limpia" simulada.
#
#   bash scripts/clean-install-check.sh            (C-11)
#     HOME temporal y PATH con solo los directorios del sistema más Go y Node.js
#     (enlazados en un directorio propio, para no heredar herramientas del
#     anfitrión). Ejecuta install.sh desde el repositorio (compilación local),
#     comprueba que no se usó la release fijada y que pi, mem, codegraph y
#     codebase-memory-mcp responden con el PATH del launcher.
#
#   bash scripts/clean-install-check.sh --no-node  (C-15)
#     Igual pero sin Node.js y sin terminal: el paso debe fallar en segundos
#     (no a los 10 minutos) y el error debe pedir Node.js 22.19+.
#
# Requiere red. No toca tu HOME real: todo queda en un temporal que se borra.
# ==============================================================================
set -euo pipefail

MODE="${1:-full}"
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SANDBOX="$(mktemp -d)"
trap 'chmod -R u+w "${SANDBOX}" 2>/dev/null || true; rm -rf "${SANDBOX}"' EXIT
export HOME="${SANDBOX}/home"
mkdir -p "${HOME}" "${SANDBOX}/toolbin"

link_tool() { # link_tool <nombre>: enlaza la herramienta del anfitrión al PATH aislado
  local src
  src="$(command -v "$1" 2>/dev/null || true)"
  if [ -z "${src}" ]; then
    echo "✗ Falta $1 en el anfitrión; se necesita para esta validación." >&2
    exit 2
  fi
  ln -s "${src}" "${SANDBOX}/toolbin/$1"
}

link_tool go
if [ "${MODE}" != "--no-node" ]; then
  link_tool node
  link_tool npm
  link_tool npx
fi
export PATH="${SANDBOX}/toolbin:/usr/bin:/bin:/usr/sbin:/sbin"
export GOCACHE="${GOCACHE:-$(cd "${REPO}" && "${SANDBOX}/toolbin/go" env GOCACHE)}"
export GOMODCACHE="${GOMODCACHE:-$(cd "${REPO}" && "${SANDBOX}/toolbin/go" env GOMODCACHE)}"
export PI_GOMEMORY_NO_UPDATE_CHECK=1
unset PI_GOMEMORY_SKIP_DEPS PI_GOMEMORY_BIN_DIR

for tool in pi mem codegraph codebase-memory-mcp; do
  if command -v "${tool}" >/dev/null 2>&1; then
    echo "✗ ${tool} ya está en el PATH aislado: la validación no sería limpia." >&2
    exit 2
  fi
done

LOG="${SANDBOX}/install.log"
cd "${REPO}"

# Sin terminal controladora: setsid (Linux) o una sesión nueva con perl (macOS).
run_detached() {
  if command -v setsid >/dev/null 2>&1; then
    setsid "$@"
  else
    perl -MPOSIX -e 'POSIX::setsid(); exec @ARGV or die "exec: $!"' "$@"
  fi
}

if [ "${MODE}" = "--no-node" ]; then
  START=$(date +%s)
  STATUS=0
  run_detached bash install.sh </dev/null >"${LOG}" 2>&1 || STATUS=$?
  ELAPSED=$(( $(date +%s) - START ))
  if [ "${STATUS}" -eq 0 ]; then
    echo "✗ Sin Node.js la instalación no debía terminar bien." >&2
    tail -n 30 "${LOG}" >&2
    exit 1
  fi
  if [ "${ELAPSED}" -ge 300 ]; then
    echo "✗ El fallo tardó ${ELAPSED} s (debe ser en segundos, no al vencer el plazo)." >&2
    exit 1
  fi
  if ! grep -q "Node.js 22.19" "${LOG}"; then
    echo "✗ El error no pide Node.js 22.19+:" >&2
    tail -n 30 "${LOG}" >&2
    exit 1
  fi
  echo "✓ Sin Node.js ni terminal: falla en ${ELAPSED} s y pide Node.js 22.19+ (código ${STATUS})."
  exit 0
fi

STATUS=0
run_detached bash install.sh </dev/null >"${LOG}" 2>&1 || STATUS=$?
if [ "${STATUS}" -ne 0 ]; then
  echo "✗ install.sh terminó con código ${STATUS}:" >&2
  tail -n 60 "${LOG}" >&2
  exit 1
fi
if ! grep -q "compilado en" "${LOG}"; then
  echo "✗ install.sh no compiló el binario local (¿usó la release fijada?)." >&2
  exit 1
fi
if grep -q "Binario v1.20.0 compilado\|Descargando pi-gomemory-setup_v1.20.0" "${LOG}"; then
  echo "✗ Se usó v1.20.0 en lugar del binario local." >&2
  exit 1
fi

# PATH del launcher (harness.go) más el del sistema y Node.js.
LAUNCHER_PATH="${HOME}/.local/bin:${HOME}/.pi/agent/bin:${HOME}/.gomemory/bin:${PATH}"
FAILED=0
for check in "pi --version" "mem version" "codegraph --version" "codebase-memory-mcp --version"; do
  if OUT="$(PATH="${LAUNCHER_PATH}" ${check} </dev/null 2>&1)"; then
    echo "✓ ${check}: $(printf '%s' "${OUT}" | head -n 1)"
  else
    echo "✗ ${check} falló: ${OUT}" >&2
    FAILED=1
  fi
done
if [ "${FAILED}" -ne 0 ]; then
  tail -n 60 "${LOG}" >&2
  exit 1
fi
echo "✓ Instalación limpia completa (log: ${LOG} se borra al salir)."

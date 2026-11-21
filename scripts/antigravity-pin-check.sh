#!/usr/bin/env bash
# ==============================================================================
# Prueba real de pi-antigravity con la cuenta Google del usuario (HCD-063).
#
#   bash scripts/antigravity-pin-check.sh 0.15.0 [0.14.0 ...]
#     Instala cada versión del plugin en un temporal (npm --ignore-scripts) y la
#     prueba cargando solo el plugin: sin MCP, skills ni AGENTS.md y sin sesión.
#     Casos: respuesta sin herramientas en tres modelos, respuesta con el prompt
#     por defecto de Pi y sus herramientas, una llamada real a bash, Claude vía
#     Antigravity y google_search del plugin. Sale con 1 si algún caso falla.
#
#   bash scripts/antigravity-pin-check.sh --installed 0.15.0
#     Comprueba la instalación real: versión del paquete en ~/.pi/agent,
#     entrada fijada en settings.json, cuota (pi-gomemory-setup usage --json) y
#     una respuesta de Pi con todas las extensiones y MCP cargados.
#
# Usa las credenciales que Pi ya tiene; no imprime tokens. Requiere red.
# ==============================================================================
set -uo pipefail

PROMPT='Responde exactamente con la palabra: LISTO'
AGENT_DIR="${PI_CODING_AGENT_DIR:-$HOME/.pi/agent}"
FAILED=0

with_deadline() { # with_deadline <segundos> <comando...>
  local secs="$1"; shift
  perl -e 'alarm shift @ARGV; exec @ARGV or die "exec: $!\n"' "$secs" "$@"
}

# check <etiqueta> <esperado> <comando...>: ejecuta y exige que la última línea
# de la salida contenga el texto esperado.
check() {
  local label="$1" want="$2"; shift 2
  local start out rc secs last
  start=$(date +%s)
  out="$(cd "${WORKDIR}" && with_deadline 240 "$@" </dev/null 2>&1)"
  rc=$?
  secs=$(( $(date +%s) - start ))
  last="$(printf '%s\n' "${out}" | grep -v '^[[:space:]]*$' | tail -n 1 | cut -c1-150)"
  if [ "${rc}" -eq 0 ] && printf '%s' "${last}" | grep -q "${want}"; then
    printf '  ✓ %-46s %3ss  %s\n' "${label}" "${secs}" "${last}"
  else
    printf '  ✗ %-46s %3ss  rc=%s %s\n' "${label}" "${secs}" "${rc}" "${last}"
    FAILED=1
  fi
}

WORKDIR="$(mktemp -d)"
trap 'rm -rf "${WORKDIR}"' EXIT

if [ "${1:-}" = "--installed" ]; then
  VERSION="${2:?uso: --installed <versión>}"
  echo "== Instalación real (${AGENT_DIR})"
  got="$(sed -n 's/.*"version": *"\([^"]*\)".*/\1/p' "${AGENT_DIR}/npm/node_modules/pi-antigravity/package.json" 2>/dev/null | head -n 1)"
  if [ "${got}" = "${VERSION}" ]; then echo "  ✓ paquete instalado ${got}"; else echo "  ✗ paquete instalado '${got}', se espera ${VERSION}"; FAILED=1; fi
  if grep -q "\"npm:pi-antigravity@${VERSION}\"" "${AGENT_DIR}/settings.json" 2>/dev/null; then
    echo "  ✓ settings.json fija npm:pi-antigravity@${VERSION}"
  else
    echo "  ✗ settings.json no fija npm:pi-antigravity@${VERSION}"; FAILED=1
  fi
  check "cuota (pi-gomemory-setup usage --json)" '}' pi-gomemory-setup usage --json
  check "Pi completo, gemini-4-argon:low" LISTO pi --no-session -p --model antigravity/gemini-4-argon:low "${PROMPT}"
  check "Pi completo, bash real" 'pin-42' pi --no-session -p --model antigravity/gemini-4-argon:low 'Ejecuta con la herramienta bash el comando: echo pin-$((6*7)) y responde solo con su salida exacta.'
  exit "${FAILED}"
fi

if [ "$#" -eq 0 ]; then
  echo "uso: $0 <versión> [<versión>...] | --installed <versión>" >&2
  exit 2
fi

for VERSION in "$@"; do
  DIR="${WORKDIR}/v${VERSION}"
  mkdir -p "${DIR}"
  if ! (cd "${DIR}" && npm install --ignore-scripts --no-audit --no-fund "pi-antigravity@${VERSION}" >/dev/null 2>&1); then
    echo "✗ no se pudo instalar pi-antigravity@${VERSION}" >&2
    FAILED=1
    continue
  fi
  SRC="${DIR}/node_modules/pi-antigravity/src"
  ONLY=(pi --no-extensions -e "${SRC}/index.ts" --no-mcp --no-skills --no-context-files --no-session -p)
  echo "== pi-antigravity ${VERSION} (solo el plugin)"
  for m in gemini-4-argon:low gemini-3.8-flash:low gemini-4-neon:low; do
    check "sin herramientas, ${m}" LISTO "${ONLY[@]}" --no-tools --model "antigravity/${m}" "${PROMPT}"
  done
  check "prompt de Pi con herramientas, gemini-4-argon:low" LISTO "${ONLY[@]}" --model antigravity/gemini-4-argon:low "${PROMPT}"
  check "bash real, gemini-4-argon:low" 'agy-42' "${ONLY[@]}" --model antigravity/gemini-4-argon:low 'Ejecuta con la herramienta bash el comando: echo agy-$((6*7)) y responde solo con su salida exacta.'
  check "Claude, claude-sonnet-5-5:low" LISTO "${ONLY[@]}" --no-tools --model antigravity/claude-sonnet-5-5:low "${PROMPT}"
  check "google_search, gemini-3.8-flash:low" '\.' "${ONLY[@]}" -e "${SRC}/features/google-search.ts" --model antigravity/gemini-3.8-flash:low 'Usa la herramienta google_search para buscar "pi.dev coding agent" y responde solo con el dominio del primer resultado.'
done
exit "${FAILED}"

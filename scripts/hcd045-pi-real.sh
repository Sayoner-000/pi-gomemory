#!/usr/bin/env bash
# HCD-045 · tarea 9: comprueba contra Pi real (no un fake) que la sesión recibe
# la sección `gomemory` con el contexto vivo de `mem context` y que
# ~/.pi/agent/AGENTS.md ya no contiene el protocolo heredado de pi-gomemory.
#
# Requisitos: pi instalado y con un proveedor autenticado, jq, y haber
# ejecutado antes `pi-gomemory-setup install` (o `go run . install`).
# Lanza una ejecución real del modelo con un prompt mínimo.
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd)"
agents="${PI_CODING_AGENT_DIR:-$HOME/.pi/agent}/AGENTS.md"
sessions="$(mktemp -d)"
trap 'rm -rf "$sessions"' EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }

[ -f "${agents%/AGENTS.md}/extensions/gomemory-context.ts" ] || fail "gomemory-context.ts no está instalada en Pi"

if [ -e "$agents" ] && grep -q "pi-gomemory:start" "$agents"; then
	fail "$agents conserva el bloque heredado pi-gomemory:start"
fi

(cd "$repo" && pi --print "Responde solo: ok" --session-dir "$sessions" >/dev/null) || fail "pi --print falló"

session="$(find "$sessions" -name '*.jsonl' -type f | head -n 1)"
[ -n "$session" ] || fail "pi no registró ninguna sesión en $sessions"

# Reproduce las secciones del prompt: el primer mensaje de sistema y sus parches.
section="$(jq -rs '[.[] | select(.type == "message" and .message.role == "system") | .message.sections // {}]
	| reduce .[] as $s ({}; . + $s) | .gomemory // empty' "$session")"
[ -n "$section" ] || fail "la sesión no tiene la sección gomemory (secciones: $(jq -rs '[.[] | select(.message.role? == "system") | .message.sections // {} | keys[]] | unique | join(",")' "$session"))"
grep -q "Memoria del Proyecto" <<<"$section" || fail "la sección gomemory no contiene la salida de mem context"
grep -q "Protocolo de pi-gomemory en Pi" <<<"$section" || fail "la sección gomemory no contiene el bloque Pi"

echo "OK: sección gomemory con mem context ($(wc -c <<<"$section" | tr -d ' ') bytes) y AGENTS.md sin bloque heredado"

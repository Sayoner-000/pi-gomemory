#!/usr/bin/env bash
# HCD-084: instalación real en un HOME temporal. Comprueba que install deja
# impeccable, frontend-design y playwright-cli (skills.sh, npx real) y
# gitlab-release (embebida) en ~/.pi-gomemory/skills, y que skills-update
# --if-due no repite dentro de las 24 h. Requiere red, Node.js y npx.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NPX="$(command -v npx)" || { echo "npx no está en el PATH" >&2; exit 1; }
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

go build -o "$WORK/pi-gomemory-setup" "$ROOT"
mkdir -p "$WORK/home" "$WORK/bin"
ln -s "$NPX" "$WORK/bin/npx"
ln -s "$(command -v node)" "$WORK/bin/node"
# Herramientas falsas y siempre al día: sync no instala ni actualiza nada y
# el paso de skills externas sí se ejecuta (PI_GOMEMORY_SKIP_DEPS lo omitiría).
for t in pi mem codegraph herdr; do
  printf '#!/bin/sh\necho 99.0.0\n' >"$WORK/bin/$t"
  chmod +x "$WORK/bin/$t"
done

run() {
  env -i HOME="$WORK/home" PATH="$WORK/bin:/usr/bin:/bin" \
    PI_GOMEMORY_SYNC_YES=1 PI_GOMEMORY_NO_UPDATE_CHECK= "$@"
}

run "$WORK/pi-gomemory-setup" install >"$WORK/install.log" 2>&1 || { tail -40 "$WORK/install.log"; exit 1; }

REPO="$WORK/home/.pi-gomemory/skills"
fail=0
for s in impeccable frontend-design playwright-cli; do
  target="$(readlink "$REPO/$s" || true)"
  if [ "$target" != "$WORK/home/.agents/skills/$s" ] || [ ! -f "$target/SKILL.md" ]; then
    echo "✗ $s no enlazada desde ~/.agents/skills (enlace: '$target')"; fail=1
  elif ! grep -q "\"$s\"" "$WORK/home/.agents/.skill-lock.json"; then
    echo "✗ $s no registrada en el lockfile de skills"; fail=1
  else
    echo "✓ $s instalada desde skills.sh y enlazada"
  fi
done
if [ -f "$REPO/gitlab-release/SKILL.md" ] && [ ! -L "$REPO/gitlab-release" ]; then
  echo "✓ gitlab-release embebida"
else
  echo "✗ gitlab-release no está embebida"; fail=1
fi
# Sin -a, skills enlazaría en decenas de agentes: solo debe existir ~/.agents.
extra="$(cd "$WORK/home" && ls -A | grep -v -E '^\.(agents|pi|pi-gomemory|local|npm|gomemory|config|cache)$' || true)"
if [ -n "$extra" ]; then echo "✗ directorios ajenos creados en HOME: $extra"; fail=1; else echo "✓ sin directorios de otros agentes"; fi

run "$WORK/pi-gomemory-setup" doctor 2>&1 | grep -q "Skills externas.*desde skills.sh" && echo "✓ doctor: Skills externas OK" || { echo "✗ doctor no informa Skills externas OK"; fail=1; }

# --if-due no repite dentro de las 24 h (la marca la dejó install).
start=$(date +%s)
run "$WORK/pi-gomemory-setup" skills-update --if-due
[ $(( $(date +%s) - start )) -lt 3 ] && echo "✓ --if-due no repite dentro de 24 h" || { echo "✗ --if-due repitió"; fail=1; }

exit $fail

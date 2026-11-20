#!/usr/bin/env bash
# Mide el reloj de la suite Go completa sin caché (HCD-056).
# Uso: scripts/hcd056-suite-time.sh <segundos_máximos>
# Falla si algún test falla o si la suite tarda el máximo o más.
set -uo pipefail

budget="${1:?uso: hcd056-suite-time.sh <segundos_máximos>}"
cd "$(dirname "$0")/.."

start=$(date +%s)
go test -count=1 ./...
code=$?
elapsed=$(( $(date +%s) - start ))

echo "suite: ${elapsed}s de reloj (máximo ${budget}s), salida go test ${code}"
if [ "$code" -ne 0 ]; then
	exit "$code"
fi
if [ "$elapsed" -ge "$budget" ]; then
	echo "FAIL: la suite supera el máximo"
	exit 1
fi

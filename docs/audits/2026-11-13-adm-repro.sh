#!/usr/bin/env bash
# HCD-034 · Reproducción de H-01, H-02 y H-03 de la ADM del 2026-11-13.
# Copia el árbol de trabajo (sin .git) a una carpeta temporal, añade allí las
# cuatro pruebas de la auditoría y las ejecuta. No modifica el repositorio.
set -euo pipefail
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")" && git rev-parse --show-toplevel)"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/hcd034.XXXX")"
trap 'rm -rf "${WORK}"' EXIT
(cd "${REPO}" && git ls-files -co --exclude-standard -z | xargs -0 tar cf -) | (cd "${WORK}" && tar xf -)
cp "${REPO}/docs/audits/2026-11-13-adm-repro_test.go.txt" "${WORK}/pkg/installer/adm_repro_test.go"
cd "${WORK}"
go test ./pkg/installer -run '^TestADM' -count=1 -v 2>&1 | grep -E "^(--- |ok|FAIL|\s+adm_repro)"

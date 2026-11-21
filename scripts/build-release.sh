#!/usr/bin/env bash
# ==============================================================================
# scripts/build-release.sh — Compilación cruzada y empaquetado para distribución
# ==============================================================================
set -euo pipefail

VERSION="${1:-v1.21.1}"
DIST_DIR="dist"

echo "==> Ejecutando go vet y go test -race antes de empaquetar..."
go vet ./...
go vet -tags pins ./pkg/installer
go test -race ./...

# Los paquetes de Pi fijados no se publican atrasados frente a npm sin saberlo
# (HCD-039). Sin caché (-count=1), porque la prueba depende de la red, y con -v
# para ver lo aceptado. PI_GOMEMORY_ALLOW_STALE_PINS=1 permite seguir.
echo "==> Comprobando los paquetes de Pi fijados frente a npm (pins-check)..."
if ! go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsCheck$'; then
  echo "✗ pins-check detuvo la release: hay paquetes fijados atrasados frente a npm o sin comprobar (detalle arriba)." >&2
  echo "  Valida el candidato con pins-validate, regístralo en RejectedVersions (pkg/installer/pins.go)" >&2
  echo "  o publica igualmente con PI_GOMEMORY_ALLOW_STALE_PINS=1." >&2
  exit 1
fi

if ! command -v shasum &>/dev/null && ! command -v sha256sum &>/dev/null; then
  echo "✗ Se requiere shasum o sha256sum para generar SHA256SUMS" >&2
  exit 1
fi

echo "==> Generando pack de distribución para pi-gomemory (${VERSION})..."
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

PLATFORMS=(
  "darwin/arm64"
  "darwin/amd64"
  "linux/amd64"
  "linux/arm64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
  GOOS="${PLATFORM%/*}"
  GOARCH="${PLATFORM#*/}"
  TARGET_NAME="pi-gomemory-setup_${VERSION}_${GOOS}_${GOARCH}"
  TARGET_DIR="${DIST_DIR}/${TARGET_NAME}"

  echo "  -> Compilando ${GOOS}/${GOARCH}..."
  mkdir -p "${TARGET_DIR}"

  CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o "${TARGET_DIR}/pi-gomemory-setup" .
  cp README.md LICENSE install.sh "${TARGET_DIR}/"

  echo "  -> Empaquetando ${TARGET_NAME}.tar.gz..."
  tar -czf "${DIST_DIR}/${TARGET_NAME}.tar.gz" -C "${TARGET_DIR}" pi-gomemory-setup README.md LICENSE install.sh
  rm -rf "${TARGET_DIR}"
done

echo "==> Calculando sumas de verificación SHA256..."
cd "${DIST_DIR}"
if command -v shasum &>/dev/null; then
  shasum -a 256 *.tar.gz > SHA256SUMS
elif command -v sha256sum &>/dev/null; then
  sha256sum *.tar.gz > SHA256SUMS
fi
cd - >/dev/null

echo "✓ Pack de distribución generado exitosamente en ${DIST_DIR}/:"
ls -lh "${DIST_DIR}"

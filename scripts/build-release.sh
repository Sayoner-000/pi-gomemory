#!/usr/bin/env bash
# ==============================================================================
# scripts/build-release.sh — Compilación cruzada y empaquetado para distribución
# ==============================================================================
set -euo pipefail

VERSION="${1:-v1.4.0}"
DIST_DIR="dist"

echo "==> Ejecutando go vet y go test antes de empaquetar..."
go vet ./...
go test ./...

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

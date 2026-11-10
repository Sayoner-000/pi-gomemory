#!/usr/bin/env bash
# ==============================================================================
# pi-gomemory — Script de instalación del harness todo-en-uno
# Repositorio: https://github.com/Sayoner-000/pi-gomemory
# ==============================================================================
set -euo pipefail

# Colores y estilo
C_RESET="\033[0m"
C_CYAN="\033[38;2;6;193;238m"
C_BRILLO="\033[38;2;34;227;247m"
C_MUTED="\033[38;2;145;169;197m"
C_ERROR="\033[38;2;255;119;127m"
C_SUCCESS="\033[38;2;46;204;113m"

REPO="Sayoner-000/pi-gomemory"
INSTALL_DIR="${PI_GOMEMORY_BIN_DIR:-${HOME}/.local/bin}"
VERSION="${PI_GOMEMORY_VERSION:-v1.0.0}"

echo -e "${C_CYAN}╔════════════════════════════════════════════════════════════════════╗${C_RESET}"
echo -e "${C_BRILLO}║         INSTALADOR DE PI-GOMEMORY PARA PI CODING AGENT             ║${C_RESET}"
echo -e "${C_CYAN}╚════════════════════════════════════════════════════════════════════╝${C_RESET}"
echo ""

# 1. Detección de Sistema Operativo y Arquitectura
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "${ARCH}" in
  x86_64|amd64)
    ARCH="amd64"
    ;;
  arm64|aarch64)
    ARCH="arm64"
    ;;
  *)
    echo -e "${C_ERROR}✗ Arquitectura no soportada: ${ARCH}${C_RESET}" >&2
    exit 1
    ;;
esac

case "${OS}" in
  darwin|linux)
    ;;
  *)
    echo -e "${C_ERROR}✗ Sistema operativo no soportado: ${OS}. Se requiere macOS o Linux.${C_RESET}" >&2
    exit 1
    ;;
esac

mkdir -p "${INSTALL_DIR}"

# 2. Obtener o compilar el binario pi-gomemory-setup
SETUP_BIN="${INSTALL_DIR}/pi-gomemory-setup"

# Opción A: Si estamos dentro del código fuente con go instalado
if [ -f "./go.mod" ] && [ -f "./main.go" ] && command -v go &>/dev/null; then
  echo -e "${C_MUTED}==> Compilando binario autocontenido desde código fuente local con Go...${C_RESET}"
  go build -ldflags="-s -w" -o "${SETUP_BIN}" .
  echo -e "${C_SUCCESS}✓ Binario compilado en: ${SETUP_BIN}${C_RESET}"
else
  # Opción B: Descargar release desde GitHub
  TARBALL="pi-gomemory-setup_${VERSION}_${OS}_${ARCH}.tar.gz"
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"
  TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'pi-gomemory')"

  echo -e "${C_MUTED}==> Descargando paquete de instalación (${OS}/${ARCH}) desde GitHub Releases...${C_RESET}"
  if command -v curl &>/dev/null; then
    if ! curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${TARBALL}"; then
      # Fallback a release latest
      LATEST_URL="https://github.com/${REPO}/releases/latest/download/${TARBALL}"
      echo -e "${C_MUTED}    Intentando desde última versión (latest)...${C_RESET}"
      curl -fsSL "${LATEST_URL}" -o "${TMP_DIR}/${TARBALL}" || {
        echo -e "${C_ERROR}✗ No se pudo descargar el paquete desde ${DOWNLOAD_URL}${C_RESET}" >&2
        rm -rf "${TMP_DIR}"
        exit 1
      }
    fi
  elif command -v wget &>/dev/null; then
    wget -qO "${TMP_DIR}/${TARBALL}" "${DOWNLOAD_URL}" || {
      echo -e "${C_ERROR}✗ No se pudo descargar con wget desde ${DOWNLOAD_URL}${C_RESET}" >&2
      rm -rf "${TMP_DIR}"
      exit 1
    }
  else
    echo -e "${C_ERROR}✗ Se requiere 'curl' o 'wget' para descargar el paquete.${C_RESET}" >&2
    rm -rf "${TMP_DIR}"
    exit 1
  fi

  echo -e "${C_MUTED}==> Extrayendo binario...${C_RESET}"
  tar -xzf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}"
  cp "${TMP_DIR}/pi-gomemory-setup" "${SETUP_BIN}"
  rm -rf "${TMP_DIR}"
  echo -e "${C_SUCCESS}✓ Binario instalado en: ${SETUP_BIN}${C_RESET}"
fi

chmod +x "${SETUP_BIN}"

# 3. Ejecutar la orquestación de instalación completa
echo ""
"${SETUP_BIN}" install

# 4. Verificación de PATH
echo ""
if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
  echo -e "${C_BRILLO}Nota importante sobre PATH:${C_RESET}"
  echo -e "${C_MUTED}Asegúrate de agregar ${INSTALL_DIR} a tu PATH en tu shell profile (~/.zshrc o ~/.bashrc):${C_RESET}"
  echo -e "  export PATH=\"${INSTALL_DIR}:\$PATH\""
  echo ""
fi

echo -e "${C_SUCCESS}✓ Instalación finalizada. Ahora puedes ejecutar '${C_BRILLO}pi-gomemory${C_SUCCESS}' para iniciar Pi en Herdr.${C_RESET}"

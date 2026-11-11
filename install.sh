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
VERSION="${PI_GOMEMORY_VERSION:-v1.6.3}"

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

# Opción A: compilar solo si el directorio actual es el código fuente de pi-gomemory
if [ -f "./go.mod" ] && grep -q '^module pi-gomemory$' ./go.mod && command -v go &>/dev/null; then
  echo -e "${C_MUTED}==> Compilando binario autocontenido desde código fuente local con Go...${C_RESET}"
  LOCAL_VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
  BUILD_TMP="$(mktemp "${INSTALL_DIR}/.pi-gomemory-setup.XXXXXX")"
  trap 'rm -f "${BUILD_TMP}"' EXIT
  go build -trimpath -ldflags="-s -w -X main.version=${LOCAL_VERSION}" -o "${BUILD_TMP}" .
  chmod +x "${BUILD_TMP}"
  mv -f "${BUILD_TMP}" "${SETUP_BIN}"
  echo -e "${C_SUCCESS}✓ Binario ${LOCAL_VERSION} compilado en: ${SETUP_BIN}${C_RESET}"
else
  # Opción B: Descargar release desde GitHub (o desde PI_GOMEMORY_BASE_URL)
  TARBALL="pi-gomemory-setup_${VERSION}_${OS}_${ARCH}.tar.gz"
  BASE_URL="${PI_GOMEMORY_BASE_URL:-https://github.com/${REPO}/releases/download/${VERSION}}"
  TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'pi-gomemory')"
  trap 'rm -rf "${TMP_DIR}"' EXIT

  download() { # download <url> <destino>
    if command -v curl &>/dev/null; then
      curl -fsSL "$1" -o "$2"
    elif command -v wget &>/dev/null; then
      wget -qO "$2" "$1"
    else
      echo -e "${C_ERROR}✗ Se requiere 'curl' o 'wget' para descargar el paquete.${C_RESET}" >&2
      exit 1
    fi
  }

  echo -e "${C_MUTED}==> Descargando ${TARBALL} desde ${BASE_URL}...${C_RESET}"
  if ! download "${BASE_URL}/${TARBALL}" "${TMP_DIR}/${TARBALL}" \
     || ! download "${BASE_URL}/SHA256SUMS" "${TMP_DIR}/SHA256SUMS"; then
    echo -e "${C_ERROR}✗ No se pudo descargar el paquete o SHA256SUMS desde ${BASE_URL}${C_RESET}" >&2
    exit 1
  fi

  echo -e "${C_MUTED}==> Verificando suma SHA256...${C_RESET}"
  EXPECTED="$(awk -v f="${TARBALL}" '$2 == f || $2 == "*"f {print $1}' "${TMP_DIR}/SHA256SUMS")"
  if command -v sha256sum &>/dev/null; then
    ACTUAL="$(sha256sum "${TMP_DIR}/${TARBALL}" | awk '{print $1}')"
  else
    ACTUAL="$(shasum -a 256 "${TMP_DIR}/${TARBALL}" | awk '{print $1}')"
  fi
  if [ -z "${EXPECTED}" ] || [ "${EXPECTED}" != "${ACTUAL}" ]; then
    echo -e "${C_ERROR}✗ La suma SHA256 de ${TARBALL} no coincide con SHA256SUMS; se aborta.${C_RESET}" >&2
    exit 1
  fi

  echo -e "${C_MUTED}==> Extrayendo binario...${C_RESET}"
  tar -xzf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}"
  # Temporal + rename en el mismo directorio: sobrescribir en su sitio un binario
  # firmado y ya ejecutado hace que macOS lo mate (Killed: 9).
  cp "${TMP_DIR}/pi-gomemory-setup" "${SETUP_BIN}.tmp.$$"
  chmod +x "${SETUP_BIN}.tmp.$$"
  mv -f "${SETUP_BIN}.tmp.$$" "${SETUP_BIN}"
  echo -e "${C_SUCCESS}✓ Binario instalado en: ${SETUP_BIN}${C_RESET}"
fi

chmod +x "${SETUP_BIN}"

# 3. Ejecutar la orquestación de instalación completa
echo ""
PI_GOMEMORY_BIN_DIR="${INSTALL_DIR}" "${SETUP_BIN}" install

# 4. Verificación de PATH
echo ""
if [[ ":${PATH}:" != *":${INSTALL_DIR}:"* ]]; then
  echo -e "${C_BRILLO}Nota importante sobre PATH:${C_RESET}"
  echo -e "${C_MUTED}Asegúrate de agregar ${INSTALL_DIR} a tu PATH en tu shell profile (~/.zshrc o ~/.bashrc):${C_RESET}"
  echo -e "  export PATH=\"${INSTALL_DIR}:\$PATH\""
  echo ""
fi

echo -e "${C_SUCCESS}✓ Instalación finalizada. Ahora puedes ejecutar '${C_BRILLO}pi-gomemory${C_SUCCESS}' para iniciar Pi en Herdr.${C_RESET}"

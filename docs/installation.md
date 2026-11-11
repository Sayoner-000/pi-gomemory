# Guía de instalación

Esta guía describe cómo instalar pi-gomemory en macOS y Linux.

## Requisitos

| Herramienta | Estado | Nota |
|---|---|---|
| Pi (`pi`) | Obligatorio | Debe estar en el PATH (`npm install -g @pi-agent/cli` o tu método habitual) |
| macOS o Linux | Obligatorio | Compatible con arquitecturas arm64 y amd64 |
| [gomemory](https://github.com/Sayoner-000/gomemory) (`mem`) | Recomendado | Memoria persistente del proyecto |
| [CodeGraph](https://github.com/colbymchenry/codegraph) (`codegraph`) | Opcional | Servidor MCP de grafo AST SQLite |
| [Codebase Memory](https://github.com/DeusData/codebase-memory-mcp) (`codebase-memory-mcp`) | Opcional | Servidor MCP de grafo estructural |
| Go 1.27+ | Solo para compilar | Requerido si instalas desde el código fuente |

Si algún servidor MCP no está instalado aún, el instalador registrará la configuración para que quede disponible tan pronto lo instales.

## Instalación rápida (recomendada)

Descarga el binario precompilado para tu arquitectura, lo ubica en `~/.local/bin/pi-gomemory-setup` y ejecuta la configuración inicial:

```bash
curl -fsSL https://raw.githubusercontent.com/Sayoner-000/pi-gomemory/main/install.sh | bash
```

## Compilación desde código fuente

Si tienes Go 1.27 o superior instalado:

```bash
git clone https://github.com/Sayoner-000/pi-gomemory.git
cd pi-gomemory
make install
```

El comando compila `bin/pi-gomemory-setup` sin símbolos de depuración (`-ldflags="-s -w"`), lo copia a `~/.local/bin/` y ejecuta el instalador.

## Descarga manual de binarios

Los binarios de cada versión se publican en [GitHub Releases](https://github.com/Sayoner-000/pi-gomemory/releases):

1. Descarga el archivo `.tar.gz` para tu sistema y `SHA256SUMS`:
   * `pi-gomemory-setup_v1.1.0_darwin_arm64.tar.gz` (Apple Silicon)
   * `pi-gomemory-setup_v1.1.0_darwin_amd64.tar.gz` (Intel Mac)
   * `pi-gomemory-setup_v1.1.0_linux_amd64.tar.gz` (Linux x86_64)
   * `pi-gomemory-setup_v1.1.0_linux_arm64.tar.gz` (Linux ARM64)

2. Verifica la suma, extrae el binario y muévelo a tu carpeta de ejecutables:

```bash
shasum -a 256 -c SHA256SUMS --ignore-missing
tar -xzf pi-gomemory-setup_v1.1.0_darwin_arm64.tar.gz
mkdir -p ~/.local/bin
mv pi-gomemory-setup ~/.local/bin/
pi-gomemory-setup install
```

## Skills de otros agentes

`pi-gomemory-setup install` añade al array `skills` de `~/.pi/agent/settings.json` las carpetas `~/.config/opencode/skills` y `~/.claude/skills` que existan, sin duplicarlas y sin tocar el resto de tu configuración. Pi las lee en su sitio: no se copia ni se modifica ninguna skill. `pi-gomemory-setup doctor` muestra cuántas skills ve en cada carpeta.

## Actualización

El launcher `pi-gomemory` comprueba al arrancar si hay una release nueva (como mucho una vez cada 24 h, con un tiempo de espera de 2 s) y, si la hay, te avisa. Para actualizar:

```bash
pi-gomemory-setup version   # versión instalada
pi-gomemory-setup update    # descarga la última release, verifica SHA256SUMS y reinstala
```

Si el checksum no coincide, la actualización se rechaza y el binario actual no cambia. Para desactivar el aviso, exporta `PI_GOMEMORY_NO_UPDATE_CHECK=1`.

## Verificación

Ejecuta el diagnóstico integrado para confirmar que cada componente quedó en su lugar:

```bash
pi-gomemory-setup doctor
```

Deberías ver una salida como esta:

```text
╔════════════════════════════════════════════════════════════════════╗
║           DIAGNÓSTICO DEL HARNESS PI-GOMEMORY                      ║
╚════════════════════════════════════════════════════════════════════╝

  ✓ Binario pi                       /Users/usuario/.local/bin/pi
  ✓ Binario mem                      /Users/usuario/.local/bin/mem
  ✓ Binario codegraph                /Users/usuario/.local/bin/codegraph
  ✓ Binario codebase-memory-mcp      /Users/usuario/.local/bin/codebase-memory-mcp
  ✓ Tema gomemory en Pi              /Users/usuario/.pi/agent/themes/gomemory.json
  ✓ MCP gomemory                     conectado
  ✓ MCP codegraph                    conectado
  ✓ MCP codebase-memory              conectado
  ✓ Plugin pi-antigravity (Google)   registrado en settings.json
  ✓ Claude Code TUI (pi-cc-extensions) activo (diffs, summaries, /context)
  ✓ Skills gomemory en Pi            21 skills cargadas
  ✓ TUI Status Bar & Tokens          extensión gomemory-status activa
  ✓ Launcher Herdr (pi-gomemory)     /Users/usuario/.local/bin/pi-gomemory
```

## Ajuste de PATH

Si tu terminal no encuentra los comandos, agrega `~/.local/bin` a tu archivo de configuración de shell (`~/.zshrc` o `~/.bashrc`):

```bash
export PATH="${HOME}/.local/bin:${PATH}"
```

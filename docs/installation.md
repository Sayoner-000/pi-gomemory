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
   * `pi-gomemory-setup_v1.6.3_darwin_arm64.tar.gz` (Apple Silicon)
   * `pi-gomemory-setup_v1.6.3_darwin_amd64.tar.gz` (Intel Mac)
   * `pi-gomemory-setup_v1.6.3_linux_amd64.tar.gz` (Linux x86_64)
   * `pi-gomemory-setup_v1.6.3_linux_arm64.tar.gz` (Linux ARM64)

2. Verifica la suma, extrae el binario y muévelo a tu carpeta de ejecutables:

```bash
shasum -a 256 -c SHA256SUMS --ignore-missing
tar -xzf pi-gomemory-setup_v1.6.3_darwin_arm64.tar.gz
mkdir -p ~/.local/bin
mv pi-gomemory-setup ~/.local/bin/
pi-gomemory-setup install
```

## Repositorio de skills

`pi-gomemory-setup install` reúne las skills en `~/.pi-gomemory/skills` y registra solo esa carpeta en el array `skills` de `~/.pi/agent/settings.json`, sin tocar el resto de tu configuración. Las 21 skills de gomemory se copian adaptadas para Pi y siempre ganan. Las de los demás agentes se enlazan, en este orden de prioridad: `~/.agents/skills`, `~/.config/opencode/skills`, `~/.claude/skills`, `~/.codex/skills` y `~/.gemini/skills`. No se copia ni se modifica ninguna skill de otro agente. El launcher vuelve a sincronizar en cada arranque (`pi-gomemory-setup skills-sync`) y `pi-gomemory-setup doctor` muestra el total y las colisiones resueltas.

## Actualización

En cada arranque, el launcher `pi-gomemory` comprueba pi-gomemory, `pi`, `mem`, `codegraph`, `codebase-memory-mcp`, `pi-antigravity` y `pi-cc-extensions`. La consulta corre en segundo plano y el aviso usa el resultado de la anterior. En una terminal interactiva pregunta si quieres actualizar (10 s; sin respuesta, no actualiza).

```bash
pi-gomemory-setup version       # versión instalada
pi-gomemory-setup deps-status   # lo desactualizado según la última consulta
pi-gomemory-setup deps-update   # actualiza cada componente con su comando oficial
pi-gomemory-setup update        # solo pi-gomemory: descarga la release, verifica SHA256SUMS y reinstala
```

Si el checksum no coincide, la actualización se rechaza y el binario actual no cambia. Para desactivar la comprobación, exporta `PI_GOMEMORY_NO_UPDATE_CHECK=1`.

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

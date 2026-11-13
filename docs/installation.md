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
   * `pi-gomemory-setup_v1.13.1_darwin_arm64.tar.gz` (Apple Silicon)
   * `pi-gomemory-setup_v1.13.1_darwin_amd64.tar.gz` (Intel Mac)
   * `pi-gomemory-setup_v1.13.1_linux_amd64.tar.gz` (Linux x86_64)
   * `pi-gomemory-setup_v1.13.1_linux_arm64.tar.gz` (Linux ARM64)

2. Verifica la suma, extrae el binario y muévelo a tu carpeta de ejecutables:

```bash
shasum -a 256 -c SHA256SUMS --ignore-missing
tar -xzf pi-gomemory-setup_v1.13.1_darwin_arm64.tar.gz
mkdir -p ~/.local/bin
mv pi-gomemory-setup ~/.local/bin/
pi-gomemory-setup install
```

Después de instalar v1.13.1, comprueba que `pi-gomemory-setup version` devuelve `v1.13.1` y ejecuta `/reload` en Pi para cargar las extensiones actualizadas (entre ellas, el aviso de actualizaciones del pie). La cuota Codex requiere una sesión OAuth `openai-codex` en Pi, no solo tener instalada la aplicación Codex.

**Si vienes de v1.12.0 o anterior:** esas versiones podían ampliar los permisos de los JSON de configuración y borrar archivos añadidos a skills legacy. v1.12.1 lo corrige; aun así, revisa [problemas conocidos](known-issues.md) y comprueba los permisos de `~/.pi/agent` después de actualizar. Ten en cuenta que `install` sigue dejando `~/.pi/agent/AGENTS.md` en `0644`.

## Repositorio de skills

`pi-gomemory-setup install` reúne las skills en `~/.pi-gomemory/skills` y registra solo esa carpeta en el array `skills` de `~/.pi/agent/settings.json`, sin tocar el resto de tu configuración. Las 21 skills de gomemory se copian adaptadas para Pi y siempre ganan. Las de los demás agentes se enlazan, en este orden de prioridad: `~/.agents/skills`, `~/.config/opencode/skills`, `~/.claude/skills`, `~/.codex/skills` y `~/.gemini/skills`. No se copia ni se modifica ninguna skill de otro agente. El launcher vuelve a sincronizar en cada arranque (`pi-gomemory-setup skills-sync`) y `pi-gomemory-setup doctor` muestra el total y las colisiones resueltas.

## Actualización

En cada arranque, el launcher `pi-gomemory` comprueba pi-gomemory, `pi`, `mem`, `codegraph`, `codebase-memory-mcp`, `pi-antigravity` y `pi-cc-extensions`. La consulta corre en segundo plano, vuelve a la red como mucho una vez por hora, y el aviso usa la última versión publicada según la consulta anterior, comparada con la que tienes instalada en ese momento. Una release nueva se anuncia, como mucho, una hora después de publicarse. En una terminal interactiva pregunta si quieres actualizar (10 s; sin respuesta, no actualiza).

Dentro de Pi, el pie muestra `↑ actualizaciones disponibles` abajo a la derecha mientras haya algo desactualizado; con `/tokens` apagado, el aviso pasa al pie por defecto de Pi. Solo informa: la instalación sigue en la pregunta del launcher o en `deps-update`.

```bash
pi-gomemory-setup version       # versión instalada
pi-gomemory-setup deps-status   # lo desactualizado: última publicada según la consulta frente a la versión instalada real (sale con 10)
pi-gomemory-setup deps-update   # actualiza cada componente con su comando oficial
pi-gomemory-setup update        # solo pi-gomemory: descarga la release, verifica SHA256SUMS y reinstala
```

Si el checksum no coincide, la actualización se rechaza y el binario actual no cambia. Para desactivar la comprobación, exporta `PI_GOMEMORY_NO_UPDATE_CHECK=1`.

## Desinstalar o reinstalar limpio

Para depurar un fallo o descartar restos de una instalación anterior:

```bash
pi-gomemory-setup uninstall --dry-run     # muestra qué se retiraría, sin tocar nada
pi-gomemory-setup uninstall               # muestra el plan y pide confirmación
pi-gomemory-setup uninstall --yes --verbose   # sin preguntar; detalla cada paso en ~/.pi-gomemory/uninstall.log
pi-gomemory-setup install                 # reinstalación limpia
```

Qué retira: extensiones `gomemory-*`, tema, launcher, skills embebidas intactas y enlaces de `~/.pi-gomemory/skills`, las entradas MCP `gomemory`, `codegraph` y `codebase-memory`, el bloque de pi-gomemory en `~/.pi/agent/AGENTS.md`, el registro de skills y el tema en `settings.json`, las cachés y el manifiesto.

Solo borra lo que puede demostrar que instaló: compara la huella SHA256 de cada archivo con `~/.pi-gomemory/manifest.json`. Lo que editaste, tus skills, otros servidores MCP y cualquier clave ajena se conservan y aparecen como omitidos.

El launcher se busca en el `BIN_DIR` del manifiesto, en `PI_GOMEMORY_BIN_DIR`, en `~/.local/bin` y junto a `pi-gomemory-setup`. Desde v1.13.0, si esa última carpeta queda fuera de `HOME` y nada la respalda (por ejemplo, al ejecutar `HOME=/tmp/prueba pi-gomemory-setup uninstall` con el binario instalado), el launcher no se borra: aparece como `[omitir]` con el motivo «fuera de HOME y sin manifiesto». Retíralo a mano o repite con `PI_GOMEMORY_BIN_DIR`.

Nunca toca la memoria de gomemory (`mem`, su base de datos, `.gomemory/` ni `hcd/`) ni desinstala `pi`, `mem`, `codegraph` o `codebase-memory-mcp`.

| Opción | Efecto |
|---|---|
| `--dry-run` | Lista el plan sin escribir nada. |
| `--yes` | Omite la confirmación. Sin terminal interactiva es obligatoria. |
| `--verbose` | Imprime cada acción y la guarda en `~/.pi-gomemory/uninstall.log`. |
| `--packages` | Quita también `pi-antigravity` y `pi-cc-extensions` en su versión fijada, y `pi-cc-extensions.json` si no lo personalizaste. |
| `--self` | Borra también el binario `pi-gomemory-setup` si está en la carpeta de instalación. |

Tras desinstalar queda la marca `~/.pi-gomemory/uninstalled`: un launcher que siga instalado ya no reinstala skills al arrancar, y `pi-gomemory-setup doctor` muestra «pi-gomemory no instalado». `install` retira la marca.

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

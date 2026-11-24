# Guía de instalación

Esta guía describe cómo instalar pi-gomemory en macOS y Linux.

## Requisitos

| Herramienta | Estado | Nota |
|---|---|---|
| macOS o Linux | Obligatorio | arm64 o amd64 |
| Node.js 22.19+ y npm | Obligatorio para Pi | Si faltan cuando hay que instalar Pi, la instalación se detiene y explica cómo instalarlos |
| Pi (`pi`) | Obligatorio | Si falta, se instala desde `https://pi.dev/install.sh` |
| [gomemory](https://github.com/Sayoner-000/gomemory) (`mem`) | Obligatorio | Si falta, se instala con su `scripts/install.sh` |
| [CodeGraph](https://github.com/colbymchenry/codegraph) (`codegraph`) | Opcional | Si falta, se instala con su `install.sh` |
| [Herdr](https://github.com/herdrdev/herdr) (`herdr`) | Opcional | Si falta, se instala con `https://herdr.dev/install.sh` |
| Go 1.27+ | Solo para compilar | Requerido si instalas desde el código fuente |

No hace falta instalar nada de esto antes. Desde v1.26.0 el primer paso de la instalación trae las herramientas que falten y actualiza las que estén desactualizadas.

## Instalación rápida (recomendada)

Descarga el binario precompilado de la última release para tu arquitectura, lo ubica en `~/.local/bin/pi-gomemory-setup` y ejecuta la configuración inicial:

```bash
curl -fsSL https://raw.githubusercontent.com/Sayoner-000/pi-gomemory/main/install.sh | bash
```

Desde v1.26.0 el script no lleva una versión fija. Lee la etiqueta de la redirección de `https://github.com/Sayoner-000/pi-gomemory/releases/latest` y verifica el binario con `SHA256SUMS`, como antes. Para instalar una versión concreta:

```bash
curl -fsSL https://raw.githubusercontent.com/Sayoner-000/pi-gomemory/main/install.sh | PI_GOMEMORY_VERSION=v1.29.2 bash
```

Si la etiqueta no se puede resolver, el script se detiene y sugiere `PI_GOMEMORY_VERSION`. Al compilar desde el código fuente no consulta la red.

### Herramientas: se instalan si faltan y se actualizan si están viejas

El paso `[1/9]` ejecuta `sync`. Busca `pi`, `mem`, `codegraph` y `herdr` en el `PATH` y en `~/.local/bin`, `~/.pi/agent/bin`, `~/.gomemory/bin` y `/usr/local/bin` (además de `PI_GOMEMORY_BIN_DIR`), consulta su última versión publicada y, para cada una, hace lo que haga falta:

| Herramienta | Si falta | Si está desactualizada | Si falla |
|---|---|---|---|
| `pi` | `https://pi.dev/install.sh` | `pi update --self` | no poder instalarla detiene la instalación; un fallo al actualizar solo avisa |
| `mem` | `scripts/install.sh` de gomemory, con `GOMEMORY_BIN_DIR` apuntando a tu directorio de binarios | `mem update -y` | igual que `pi` |
| `codegraph` | `install.sh` de colbymchenry/codegraph, con `CODEGRAPH_BIN_DIR` igual | `codegraph upgrade` | avisa y sigue |
| `herdr` | `https://herdr.dev/install.sh`, con `HERDR_INSTALL_DIR` igual | `herdr update` | avisa y sigue |

Con terminal, antes de empezar se muestra la lista numerada con la versión instalada, la publicada y si es obligatoria u opcional. Escribe los números que quieras marcar o desmarcar y pulsa Enter. `pi` y `mem` no se desmarcan; en modo lite las opcionales aparecen desmarcadas. Sin terminal, o si pulsas Enter sin cambios, se usa la selección por defecto. La lista se lee de `/dev/tty`, así que funciona con `curl | bash`.

Antes de instalar `pi` se comprueba que haya Node.js 22.19 o superior y npm. Si faltan, la instalación se detiene sin lanzar el instalador de Pi y explica cómo instalarlos.

Cada script se descarga solo por HTTPS a un directorio temporal junto a tu directorio de binarios y se ejecuta con la entrada en `/dev/null`, así no consume el resto de `install.sh` cuando lo lanzas con `curl | bash`. Las actualizaciones también se ejecutan sin entrada. Cada instalador tiene 10 minutos; si se pasa, cuenta como fallo.

`sync` consulta la red aunque `PI_GOMEMORY_NO_UPDATE_CHECK` esté puesta, porque lo pides tú, y al terminar reescribe la caché de versiones (`~/.pi/agent/pi-gomemory-deps.json`).

codebase-memory-mcp se retiró en v1.28.0: la instalación ya no lo trae ni lo actualiza, y CodeGraph es el único grafo externo. Si lo tienes instalado, pi-gomemory no lo desinstala.

Algunos instaladores oficiales añaden su directorio al `PATH` en `~/.zshrc` o `~/.bashrc`, y crean sus propias carpetas (`~/.codegraph`, `~/.pi/agent/install`).

Para saltarte este paso, exporta `PI_GOMEMORY_SKIP_DEPS=1` (vacía, `0` o `false` lo dejan activo). Con el modo lite (`pi-gomemory-setup lite on`) solo se tratan `pi` y `mem`, salvo que marques otra en la lista o la pidas con `sync --only`.

Los scripts de Pi, gomemory, codegraph y herdr se toman de su canal oficial sin fijar versión, igual que sus propias actualizaciones. El script de herdr verifica el SHA-256 del binario contra el manifiesto de su release. HTTPS protege el transporte; el contenido depende de cada proyecto.

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
   * `pi-gomemory-setup_v1.26.1_darwin_arm64.tar.gz` (Apple Silicon)
   * `pi-gomemory-setup_v1.26.1_darwin_amd64.tar.gz` (Intel Mac)
   * `pi-gomemory-setup_v1.26.1_linux_amd64.tar.gz` (Linux x86_64)
   * `pi-gomemory-setup_v1.26.1_linux_arm64.tar.gz` (Linux ARM64)

2. Verifica la suma, extrae el binario y muévelo a tu carpeta de ejecutables:

```bash
shasum -a 256 -c SHA256SUMS --ignore-missing
tar -xzf pi-gomemory-setup_v1.26.1_darwin_arm64.tar.gz
mkdir -p ~/.local/bin
mv pi-gomemory-setup ~/.local/bin/
pi-gomemory-setup install
```

Después de instalar, comprueba que `pi-gomemory-setup version` devuelve `v1.26.1` y ejecuta `/reload` en Pi para cargar las extensiones actualizadas. Desde `v1.22.0`, `install` también activa `hideThinkingBlock: true` (`Ctrl+T`) en `~/.pi/agent/settings.json` cuando esa clave no existe, respetando cualquier preferencia previa del usuario.

Si vienes de v1.12.0 o anterior, revisa los permisos de `~/.pi/agent` y los [problemas conocidos](known-issues.md). Desde v1.15.0, `install` no crea un protocolo en `AGENTS.md`: retira el bloque heredado sin modificar el texto propio. Conserva los permisos y los enlaces; si solo había protocolo, borra el archivo normal o vacía el destino enlazado. Desde v1.15.1, lo retira solo después de instalar `gomemory-context.ts`, quita a la vez el bloque marcado y las copias históricas sin marcas, y si los marcadores de pi-gomemory son ambiguos (huérfanos, repetidos o invertidos) conserva el archivo completo y avisa.

**Configuración enlazada (dotfiles):** desde v1.13.6, si `~/.pi/agent/mcp.json`, `settings.json`, `AGENTS.md` o `pi-cc-extensions.json` son enlaces simbólicos, `install`, `update`, `uninstall` y los subcomandos escriben en el archivo de destino y conservan el enlace y sus permisos. Un enlace roto, un bucle o un enlace a un directorio hacen fallar el paso con «el enlace no apunta a un archivo», sin tocar nada. Si una versión anterior ya sustituyó alguno de tus enlaces, vuelve a crearlo a mano.

## Mitigación temporal de Antigravity

Desde v1.16.0, el instalador incluye el guard de Antigravity. Este cambia la prioridad de las instrucciones y se puede desactivar con `ANTIGRAVITY_PROMPT_GUARD=off` al iniciar Pi. Consulta el [diagnóstico y los límites](antigravity.md).

## Codemode y contexto

Desde v1.24.0, la instalación y `update` dejan Pi en modo **codemode híbrido**: añaden `+codemode` a `defaultTools` (y anotan que lo añadió pi-gomemory, para retirarlo al desinstalar) y escriben el perfil híbrido de `gomemory`. Si ejecutaste `pi-gomemory-setup codemode off`, esa preferencia persiste en `~/.pi-gomemory/codemode.json` y la instalación mantiene el modo directo sin tocar `defaultTools`. Consulta el estado con `pi-gomemory-setup codemode status`. Si `codemode on`, `install` o `update` fallan a medias, `settings.json` y `mcp.json` vuelven a su contenido anterior.

Desde v1.23.2, `install` y `update` avisan si el modo queda `parcial` (con `codemode off`, un `+codemode` que no añadió pi-gomemory). El aviso de la extensión duplicada de codebase-memory-mcp se retiró en v1.28.0 junto con esa herramienta. No cambian nada por su cuenta: el aviso dice qué comando ejecutar. Los detalles están en la [guía de uso](usage.md#codemode-y-contexto-vivo).

Después de instalar, usa `/reload` o abre una sesión nueva. `gomemory-context.ts` consulta `mem context` con el directorio del proyecto, espera hasta tres segundos y añade el contexto al prompt. No necesitas iniciar Pi desde el launcher para que se ejecute esta extensión. Si falla, se muestra una advertencia; usa `get_context()` o `mem context` para recuperar la memoria.

## Repositorio de skills

`pi-gomemory-setup install` reúne las skills en `~/.pi-gomemory/skills` y registra solo esa carpeta en el array `skills` de `~/.pi/agent/settings.json`, sin tocar el resto de tu configuración. Las 20 skills de gomemory se copian adaptadas para Pi y siempre ganan; si `mem` instaló su propia copia de una skill en otro agente, se usa esa. Las de los demás agentes se enlazan, en este orden de prioridad: `~/.agents/skills`, `~/.config/opencode/skills`, `~/.claude/skills`, `~/.codex/skills` y `~/.gemini/skills`. No se copia ni se modifica ninguna skill de otro agente. El launcher vuelve a sincronizar en cada arranque (`pi-gomemory-setup skills-sync`) y `pi-gomemory-setup doctor` muestra el total y las colisiones resueltas.

Desde v1.29.0, Pi comprueba las skills al iniciar cada sesión y después de un `mem update` lanzado desde Pi, así que no depende de arrancar con el launcher. Si `mem` trajo versiones nuevas, sincroniza y te avisa de que `/reload` las carga. Las copias que editaste a mano no se sobrescriben; si se quedaron atrás, Pi te avisa en amarillo y puedes reponerlas:

```bash
pi-gomemory-setup skills-sync --check                 # sale con 10 si hay skills desfasadas o editadas que difieren
pi-gomemory-setup skills-sync --restore hcd-verify    # repone una skill de gomemory editada
```

`--restore` se niega si la carpeta tiene otros archivos tuyos aparte de `SKILL.md`.

## Actualización

En cada arranque, el launcher `pi-gomemory` comprueba pi-gomemory, `pi`, `mem`, `codegraph`, `herdr`, `pi-antigravity` y `pi-cc-extensions`. La consulta corre en segundo plano, vuelve a la red como mucho una vez por hora, y el aviso usa la última versión publicada según la consulta anterior, comparada con la que tienes instalada en ese momento. Una release nueva se anuncia, como mucho, una hora después de publicarse. En una terminal interactiva pregunta si quieres actualizar (10 s; sin respuesta, no actualiza).

Dentro de Pi, el pie muestra `↑ actualizaciones disponibles` abajo a la derecha mientras haya algo desactualizado; con `/tokens` apagado, el aviso pasa al pie por defecto de Pi. Solo informa: para instalar, responde a la pregunta del launcher, ejecuta `deps-update` o usa `/componentes`. Este comando muestra cada componente con su versión instalada y la publicada, deja marcar o desmarcar las herramientas opcionales y ejecuta `sync` con lo marcado (plazo de 60 minutos). Si se actualizó `pi`, te pide reiniciar Pi.

```bash
pi-gomemory-setup version       # versión instalada
pi-gomemory-setup components --json   # estado de cada componente (consulta la red)
pi-gomemory-setup sync          # instala lo que falta y actualiza lo desactualizado (con lista si hay terminal)
pi-gomemory-setup sync --yes --skip=herdr   # sin lista y sin herdr; también --only=a,b y --json
pi-gomemory-setup deps-status   # lo desactualizado: última publicada según la consulta frente a la versión instalada real (sale con 10)
pi-gomemory-setup deps-update   # pi-gomemory con update; las herramientas, en un solo sync; los paquetes fijados, con pi install
pi-gomemory-setup update        # descarga la release, verifica SHA256SUMS, reinstala y ejecuta sync
```

`sync --only` y `--skip` no pueden dejar fuera `pi` ni `mem`. Combinar las dos opciones, o nombrar algo que no es una herramienta (por ejemplo `pi-gomemory` o un paquete fijado), termina con código 2.

Si el checksum no coincide, la actualización se rechaza y el binario actual no cambia. Para desactivar las comprobaciones automáticas (launcher, pie de Pi, `check-update` y `deps-refresh`), exporta `PI_GOMEMORY_NO_UPDATE_CHECK=1`. Desde v1.13.7 todos aplican la misma regla: sin definir, vacía, `0` o `false` (sin distinguir mayúsculas) dejan la comprobación activa, y cualquier otro valor la desactiva. Desde v1.13.5, `update` consulta siempre, aunque esa variable esté puesta, porque lo pides tú.

`deps-install` ya no instala nada desde v1.28.0, cuando se retiró codebase-memory-mcp: indica cómo actualizar cada herramienta con su propio comando y sale con 2. Con una release nueva, `update` vuelve a ejecutar `install` sin mostrar la lista, y ese `install` ejecuta `sync` una vez. Si pi-gomemory ya está al día, `update` ejecuta `sync` directamente. Si no puede consultar la última release, termina con error y sugiere `pi-gomemory-setup sync`.

Desde v1.13.5, `update` reinstala en el `BIN_DIR` registrado en `~/.pi-gomemory/manifest.json`, aunque la shell no tenga `PI_GOMEMORY_BIN_DIR`. Si la variable tiene valor, manda ella. Si el manifiesto existe pero no se puede leer, `update` falla antes de descargar nada y el binario no cambia.

## Desinstalar o reinstalar limpio

Para depurar un fallo o descartar restos de una instalación anterior:

```bash
pi-gomemory-setup uninstall --dry-run     # muestra qué se retiraría, sin tocar nada
pi-gomemory-setup uninstall               # muestra el plan y pide confirmación
pi-gomemory-setup uninstall --yes --verbose   # sin preguntar; detalla cada paso en ~/.pi-gomemory/uninstall.log
pi-gomemory-setup install                 # reinstalación limpia
```

Qué retira: extensiones `gomemory-*`, tema, launcher, skills embebidas intactas y enlaces de `~/.pi-gomemory/skills`, las entradas MCP `gomemory` y `codegraph` (y la heredada `codebase-memory` si la escribió pi-gomemory), el bloque de pi-gomemory en `~/.pi/agent/AGENTS.md`, el registro de skills y el tema en `settings.json`, las cachés y el manifiesto. Desde v1.15.0 reconoce también el perfil híbrido intacto y retira `+codemode` si el registro de propiedad indica que lo añadió pi-gomemory; conserva una activación previa del usuario. Desde v1.15.1 reconoce además el perfil híbrido publicado en v1.14.0, y el registro de propiedad guarda el archivo real de `settings.json`: si el enlace apunta ahora a otro destino, o el registro de v1.15.0 no lo identifica, `+codemode` se conserva con aviso. Un `codemode.json` ilegible o inválido aparece como `[fallido]` y se conserva junto con `+codemode`.

Solo borra lo que puede demostrar que instaló: compara la huella SHA256 de cada archivo con `~/.pi-gomemory/manifest.json`. Lo que editaste, tus skills, otros servidores MCP y cualquier clave ajena se conservan y aparecen como omitidos.

Desde v1.13.4, cada retirada se vuelve a comprobar justo antes de escribir. Si editaste algo mientras `uninstall` esperaba confirmación, se conserva y aparece como `[omitir] … — cambió después del plan`; si ya no existe, como `[omitir] … — ya no existe`. Si un archivo gestionado existe pero no se puede leer, aparece como `[fallido]` y el comando termina con error en lugar de decir que no queda nada.

Desde v1.13.5, el manifiesto es lo último que se retira y solo si nada falló. Si algo falla, se conserva y la salida lo indica con `[conservar] … — conservado por fallos previos`, para que el reintento todavía encuentre tu `BIN_DIR`. `--dry-run` lo muestra como «se retira si no hay fallos».

El launcher se busca en el `BIN_DIR` del manifiesto, en `PI_GOMEMORY_BIN_DIR`, en `~/.local/bin` y junto a `pi-gomemory-setup`. Desde v1.13.0, si esa última carpeta queda fuera de `HOME` y nada la respalda (por ejemplo, al ejecutar `HOME=/tmp/prueba pi-gomemory-setup uninstall` con el binario instalado), el launcher no se borra: aparece como `[omitir]` con el motivo «fuera de HOME y sin manifiesto». Retíralo a mano o repite con `PI_GOMEMORY_BIN_DIR`.

Nunca toca la memoria de gomemory (`mem`, su base de datos, `.gomemory/` ni `hcd/`) ni desinstala `pi`, `mem`, `codegraph` o `herdr`. Al terminar te recuerda cómo quitar cada una con su propio comando. También borra los temporales `.pi-gomemory-deps-*` que hubieran quedado en el directorio de binarios.

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
  ✓ Tema gomemory en Pi              /Users/usuario/.pi/agent/themes/gomemory.json
  ✓ MCP gomemory                     conectado
  ✓ MCP codegraph                    conectado
  ✓ Plugin pi-antigravity (Google)   registrado en settings.json
  ✓ Claude Code TUI (pi-cc-extensions) activo (diffs, summaries, /context)
  ✓ Skills gomemory en Pi            20 skills cargadas
  ✓ TUI Status Bar & Tokens          extensión gomemory-status activa
  ✓ Launcher Herdr (pi-gomemory)     /Users/usuario/.local/bin/pi-gomemory
```

## Ajuste de PATH

Si tu terminal no encuentra los comandos, agrega `~/.local/bin` a tu archivo de configuración de shell (`~/.zshrc` o `~/.bashrc`):

```bash
export PATH="${HOME}/.local/bin:${PATH}"
```

El instalador de Pi puede dejar `pi` en `~/.pi/agent/bin` cuando ningún otro directorio candidato está en tu `PATH`. El launcher `pi-gomemory` ya busca ahí, y al terminar `install.sh` lista los directorios con herramientas instaladas que tu shell todavía no tiene, aunque la instalación haya fallado.

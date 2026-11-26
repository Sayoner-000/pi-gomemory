# Guía de uso

Esta guía cubre los comandos cotidianos y opciones del lanzador pi-gomemory.

## Inicio rápido

Para iniciar Pi con la paleta de color gomemory y contexto de memoria activo:

```bash
pi-gomemory
```

Si estás dentro de un proyecto donde ya ejecutaste `mem init` o existe `.gomemory/memories.db`, el lanzador inyecta las memorias más recientes con el parámetro `--append-system-prompt` de Pi.

## Antigravity (Google AI Pro): retirado

HCD-085 retiró toda la integración con Antigravity: el plugin `pi-antigravity`, la cuota de `/sidebar`, los comandos `pi-gomemory usage` y `pi-gomemory-setup usage`, y el workaround `antigravity-prompt-guard.js`. Consultar la API interna de Google con el token OAuth de la cuenta expone esa cuenta a bloqueos y observaciones.

`install` y `update` retiran de `settings.json` las entradas de `pi-antigravity` que escribió pi-gomemory, y borran las copias intactas del guard. Si añadiste `pi-antigravity` por tu cuenta, con otra versión, se conserva con un aviso. `~/.pi/agent/antigravity-accounts.json` guarda tu sesión OAuth y no se toca: bórralo o revoca el acceso desde tu cuenta de Google si ya no lo usas. El [diagnóstico de Antigravity](antigravity.md) queda como referencia histórica.

## Referencia de comandos CLI

El binario `pi-gomemory-setup` acepta los siguientes argumentos:

```text
Uso: pi-gomemory-setup [comando]

Comandos:
  install       Ejecuta la instalación completa (opción por defecto); su primer paso es sync
  claude-usage  Muestra telemetría agregada de tokens de Claude Code (hoy, semana, total)
  anthropic-usage  Devuelve los límites del plan de Anthropic en JSON
  codex-usage   Devuelve los límites del plan de OpenAI Codex en JSON
  doctor        Verifica binarios, tema, MCP, plugins, launcher y otros agentes detectados
  mcp           Registra gomemory y codegraph en mcp.json
  codemode      Activa, desactiva o consulta el modo codemode híbrido (on | off | status)
  theme         Instala la paleta gomemory en ~/.pi/agent/themes/
  plugin        Configura pi-cc-extensions en settings.json
  skills        Reúne las skills de gomemory y del anfitrión en ~/.pi-gomemory/skills/
  skills-sync   Sincroniza el repositorio central de skills y purga copias legadas
                --check: solo comprueba; sale con 10 si hay skills desfasadas o editadas que difieren de mem
                --restore <skill>…: repone skills de gomemory editadas localmente
  skills-update Instala o actualiza impeccable, frontend-design y playwright-cli desde skills.sh (npx skills)
                --if-due: solo actúa si pasaron 24 h desde la última revisión (respeta PI_GOMEMORY_NO_UPDATE_CHECK)
  harness       Genera el launcher pi-gomemory en ~/.local/bin (o PI_GOMEMORY_BIN_DIR)
  version       Muestra la versión instalada
  deps-status   Lista los componentes desactualizados frente a la versión instalada real (sale con 10)
  deps-update   Actualiza los componentes desactualizados (las herramientas, en un solo sync)
  deps-install  Ya no instala nada: codebase-memory-mcp se retiró en v1.28.0 (sale con 2)
  sync          Instala lo que falta y actualiza lo desactualizado de pi, mem, codegraph
                y herdr ([--yes] [--json] [--only=a,b | --skip=c])
  components    Lista los componentes del meta-harness (--names sin red; --json con su estado)
  update        Actualiza pi-gomemory desde la última release verificada y después ejecuta sync
  uninstall     Retira lo instalado por pi-gomemory (ver «Desinstalar» en installation.md)
  help          Muestra este mensaje de ayuda
```

## Codemode y contexto vivo

`install` y `update` dejan Pi en modo **codemode híbrido** por defecto: `gomemory` declara directas solo sus tools de protocolo y consulta diaria, `codegraph` sigue directo y el resto se llama desde `codemode`. Con `codemode off` vuelves al modo directo, y esa elección se respeta en las siguientes instalaciones:

```bash
pi-gomemory-setup codemode on
pi-gomemory-setup codemode off
pi-gomemory-setup codemode status
```

La preferencia y la propiedad de `+codemode` se guardan en `~/.pi-gomemory/codemode.json`. Las herramientas MCP declaradas se llaman directamente; el resto se consulta desde `codemode` con `tools.<nombre>(args)`. `mcp` regenera los servidores y conserva un perfil híbrido ya activo.

Si una skill nombra una tool que el modelo no ve declarada (por ejemplo `socratica_flow`), el protocolo que inyecta `gomemory-context.ts` le pide buscarla con `searchTools` desde `codemode` antes de darla por inexistente.

Cuando `settings.json` tiene `+codemode` y `mcp.json` sigue en directo, o al revés, `codemode status` responde `parcial`. Con el híbrido por defecto solo pasa si elegiste `codemode off` y otra herramienta añadió `+codemode`. En ese estado Pi carga todas las tools directas y además `codemode`, así que pagas las dos cosas en cada turno. `install`, `update` y `doctor` lo avisan. Para salir, elige un modo con `codemode on` u `codemode off`.

pi-gomemory ya no instala codebase-memory-mcp desde v1.28.0, y al instalar retira la entrada heredada `codebase-memory` de `mcp.json` si la había escrito él. Si la registraste tú y además `codebase-memory-mcp install` dejó una extensión para Pi (normalmente `~/.pi/agent/extensions/cbmem.ts`), Pi ve esas tools dos veces. pi-gomemory ya no lo avisa desde v1.28.0 ni borra la extensión, porque no la instaló. Bórrala tú y, si reinstalas codebase-memory-mcp, usa `--clients` sin `pi`; `codebase-memory-mcp uninstall` quita la integración de todos los agentes.

`pi-gomemory-setup lite on` retira los servidores de grafo gestionados por pi-gomemory y conserva los personalizados. El modo se mantiene al ejecutar `install`, `update`, `mcp` o cambiar codemode. Usa `lite status` para consultarlo y `lite off` para restaurar los servidores gestionados que falten.

Al abrir o cambiar una sesión, `gomemory-context.ts` carga `mem context` una vez con el directorio del proyecto y un timeout de 3000 ms. Inyecta hasta 24 576 bytes de texto UTF-8 más el aviso de recorte, si hace falta, y el bloque de protocolo de Pi. Ese bloque pide validar el impacto del proyecto en solo lectura con `codegraph` (`codegraph_explore`) antes de editar código o abrir una unidad HCD, con o sin `/plan`. Después de escrituras de código (`edit`, `write`), guardados de memoria (`save_memory`) y al cierre de turno (`agent_end`), la extensión dispara `mem code-refresh` en segundo plano con un cooldown de 15 segundos para mantener al día tanto CodeGraph como el índice interno de gomemory. Si `mem context` falla o no devuelve contenido, muestra una advertencia por sesión y mantiene el bloque de protocolo. `/plan` añade su método y el protocolo de Mapa de impacto en otra sección del prompt.

`AGENTS.md` queda para tus instrucciones. `install` retira el protocolo antiguo de pi-gomemory, conservando el texto propio. Abre una sesión nueva o ejecuta `/reload` después de actualizar las extensiones. Consulta las [limitaciones](known-issues.md) de cambios parciales de configuración y del bloqueo de `/plan`.

## Otros agentes (Claude Code, Codex, agy, OpenCode)

pi-gomemory configura solo Pi. La integración de los demás agentes la hace gomemory:

```bash
mem setup-mcp --scope global --agents claude,codex,agy,opencode
```

`pi-gomemory-setup doctor` lista los agentes que encuentra en el `PATH` junto con este comando, sin modificar su configuración.

## Comandos dentro de Pi

Durante una sesión interactiva de Pi:

### Banner de inicio
Al abrir Pi con `pi-gomemory`, la cabecera muestra la versión de pi-gomemory y la de gomemory instalada. El launcher las exporta en `PI_GOMEMORY_RUNNING_VERSION` (de `pi-gomemory-setup version`) y `GOMEMORY_INSTALLED_VERSION` (de `mem --version`). Las lee después de la pregunta de actualización, así que si aceptas actualizar al arrancar ya ves las versiones nuevas. Si falta alguna, se omite. Debajo del lema va el estado del modo plan (ver más abajo). Con `NO_COLOR` se dibuja sin colores.

Pi solo admite una cabecera. Si pi-cc-extensions tiene `showStartupHeader: true`, su cabecera sustituye al banner; por eso `pi-gomemory-setup install` (y `update`) fija esa clave en `false` en cada instalación, y `pi-gomemory-setup doctor` avisa en «Banner gomemory» si vuelve a activarse (por ejemplo desde `/ccstyle`).

### Thinking colapsado e indicador animado
Desde `v1.22.0`, la instalación activa `hideThinkingBlock: true` en `~/.pi/agent/settings.json` si el usuario no había fijado otro valor, de modo que Pi inicia con los bloques de razonamiento colapsados (`Ctrl+T` los expande o colapsa en cualquier momento). `gomemory-status.ts` muestra la etiqueta `(thinking...)` y una secuencia animada de puntos (`· • ●`) en el indicador de carga, sin interferir con el texto de `pi-cc-extensions`.

### Modo plan (`/plan`)
Mientras el modo plan está activo, Pi bloquea las modificaciones comunes del agente (`edit`, `write`, `powershell`, comandos destructivos en `bash`), las mutaciones de estado HCD (`mem hcd start`, `clarify`, `approve`, `verify`, `close`, `cancel`, `promote`, `rename`, también la tool `hcd_rename`) y `codegraph index`. Las consultas de solo lectura de `codegraph` y de gomemory siguen permitidas. Antes de proponer tareas o abrir un HCD, el protocolo exige entregar un Mapa de impacto con símbolos, llamadores, archivos afectados y cobertura.

- `/plan on` activa el modo sin enviar una petición.
- `/plan off` lo desactiva sin iniciar un turno. El siguiente turno ya no recibe las instrucciones de modo plan.
- `/plan status` muestra si está activo, sin cambiar el estado.
- `/plan run` conserva el comportamiento anterior: desactiva el modo, pero no ejecuta el plan automáticamente.
- `/plan` sin argumentos alterna el modo; `/plan <petición>` lo activa y envía la petición.
- `Alt+P` alterna el modo desde el teclado, sin iniciar un turno. `Ctrl+Alt+P` hace lo mismo y sirve en terminales de macOS que no envían Option como Alt (en ese caso Option+P escribe `π`). En Windows y WSL solo existe `Ctrl+Alt+P`, porque Pi usa allí `Alt+P` para volver al modelo anterior.

La cabecera indica el estado en todo momento: `◆ MODO PLAN ON · Alt+P` en ámbar (el color `warning` del tema gomemory) o `○ plan off · Alt+P` atenuado. Cambia al instante con el atajo o con `/plan`. La etiqueta `PLAN` del pie sigue apareciendo, aunque con pi-cc-extensions el pie la pinta con su propio color.

### Panel de tareas (`/tasks`)
`/tasks` muestra u oculta las tareas de la unidad HCD activa encima del editor. La cabecera lleva una barra de progreso (`━━━━──────  5/13 · 38 %`): tramo grueso para lo hecho y fino para lo que falta. El estado de cada tarea viene del motor y se dibuja como un círculo que se llena:

| Glifo | Estado |
|---|---|
| `○` | pendiente |
| `●` (parpadea) | en curso: la tarea activa, aunque no haya ninguna llamada en marcha; alterna `●` y `○` cada medio segundo |
| `●` | verificada |
| `●` (atenuado) | caducada mientras la unidad sigue en `implement`: con gomemory ≥ 2.59.0 cuenta como progreso y no hace falta reverificarla todavía |
| `↻` | caducada en `verify`/`close`: el código cambió y hay que revalidarla con `mem hcd verify --stale` |
| `✗` | fallida |

Cada estado tiene también su color; el glifo se mantiene para quien no distingue colores o usa un tema monocromo. Debajo de cada tarea, el árbol (`├─`, `└─`) muestra las llamadas de la sesión con su estado y duración; la que está en marcha usa el spinner braille de Pi.

Mientras está visible, el panel vuelve a consultar el motor cada 10 segundos. Si la unidad se cierra o se cancela, también desde otra sesión de Pi, el panel desaparece en ese plazo. Con todas las tareas verificadas pero la unidad aún abierta, sigue visible hasta el cierre.

La tarea en curso muestra por defecto sus últimas cinco llamadas y el sufijo `· en curso` mientras el agente trabaja. El foco automático prioriza tareas `pending`, luego `failed` y por último `stale`/`unverified`, así editar una tarea nueva no hace saltar el foco atrás a una tarea anterior caducada. Las llamadas terminadas se marcan con `✓` en verde en lugar de gris atenuado (`dim`), que queda reservado para tareas pendientes sin iniciar. Las tareas completadas quedan resumidas; usa `/tasks expand N` o haz clic en el resumen para abrirlas y `/tasks collapse N` para cerrarlas. Con `/tasks focus N` puedes decidir a qué tarea se asignan las próximas llamadas.

Si la unidad tiene más de 8 tareas, se ven 8 alrededor de la que está en curso, con `↑ N más` y `↓ N más` en los bordes. Para ver el resto usa la rueda del ratón sobre el panel, `Alt+Shift+↑` / `Alt+Shift+↓` desde el input o un clic en esas líneas. `↑` y `↓` solos siguen recorriendo el historial, y `Alt+↑` sigue recuperando los mensajes en cola.

### Sugerencia del siguiente comando
Con el input vacío, Pi muestra en gris lo que probablemente quieras escribir a continuación. `Tab` lo copia al input sin enviarlo; si ya escribiste algo, `Tab` vuelve a ser el autocompletado de siempre.

- Con una unidad HCD activa, la sugerencia es el siguiente paso que indica el motor (por ejemplo, `/hcd-implement`). Sale de `mem hcd show` y no consume tokens.
- Sin unidad activa, al terminar cada turno se pide al modelo de la sesión una propuesta de una línea. La petición lleva el último mensaje tuyo y la última respuesta del agente, usa el razonamiento mínimo y un tope de 300 tokens; con Claude Opus cuesta unos 0,002-0,003 USD. Si empiezas a escribir o lanzas otro turno, se cancela.

La extensión `gomemory-suggest.ts` envuelve el editor que haya instalado otra extensión (por ejemplo, el `❯` de pi-cc-extensions) en lugar de sustituirlo.

### Panel lateral estilo OpenCode (`/sidebar`)
Alterna la visibilidad de la barra lateral en la esquina superior derecha (`anchor: "top-right"`). Muestra la sesión/HCD activo, el consumo de ventana de contexto en tiempo real (tokens, porcentaje y coste), límites de Anthropic u OpenAI Codex y telemetría de Claude Code, y los servidores MCP declarados en `mcp.json` con el estado `Configured` (no se comprueba la conexión en vivo; sin `mcp.json` válido la lista queda vacía). El panel flota sin capturar el teclado, así que puedes seguir escribiendo; el título incluye la versión instalada. Se oculta con `/sidebar` o automáticamente si el terminal mide menos de 100 columnas.

**Límites del plan de Anthropic.** Si Pi está conectado a Anthropic con OAuth (suscripción Claude Pro/Max, `/login` en pi), el bloque «▼ Anthropic · plan» muestra el porcentaje usado de la **sesión** (ventana de 5 h) y de la **semana**, con el tiempo hasta cada reinicio. Si Anthropic los envía, añade también los límites semanales de Opus o Sonnet. Los límites se obtienen de forma pasiva a partir de las cabeceras `anthropic-ratelimit-unified-*` que llegan en cada respuesta del modelo (`after_provider_response`), recalculando la cuenta atrás en cada renderizado. Como respaldo inicial al abrir el panel sin respuesta previa, realiza una única consulta con `pi-gomemory-setup anthropic-usage` evitando saturar la cuota de la API (HTTP 429).

Si la consulta falla, el bloque muestra el error completo y nunca cifras viejas o inventadas; por ejemplo, token caducado (abre pi para renovarlo) o binario sin `anthropic-usage` (ejecuta `pi-gomemory-setup update`).

- Clic en `[−]` de la cabecera: minimiza el panel a una sola fila (`[ HCD · ... [+] ]`); clic en `[+]`: lo restaura. El ratón solo funciona en el modo `fullscreen` de Pi (el predeterminado); en `--tui-mode regular` usa los comandos.
- `/sidebar min` (o alias `/sidebar m`, `/sidebar -`): alterna el modo minimizado desde el teclado.
- `/min`: atajo de `/sidebar min`.

Con un título largo, el título se recorta con `…` y el control `[−]` sigue visible.

### Cuota de OpenAI Codex

Desde v1.12.0, cuando el proveedor activo es Codex (`openai-codex`), `/sidebar` muestra «▼ Codex Quota» con las ventanas primaria y secundaria del plan, habitualmente cinco horas y una semana. Cada fila indica el porcentaje **restante**, una barra y el tiempo hasta el reinicio. La barra cambia a amarillo por debajo del 50 % restante y a rojo por debajo del 20 %.

```bash
pi-gomemory-setup codex-usage
```

El JSON devuelve `planType` y `limits`; cada límite contiene `name`, `percent` (porcentaje **usado**), `resetsAt` y `resetIn`. El panel calcula lo restante como `100 - percent`. La consulta usa el token de `openai-codex` y, si existe, `accountId` en `~/.pi/agent/auth.json`, contra `https://chatgpt.com/backend-api/wham/usage`. Es una API externa sin contrato estable garantizado por este proyecto.

El comando no renueva tokens. Sin sesión OAuth, con token caducado, respuesta HTTP distinta de 200, JSON inválido, respuesta superior a 1 MiB o ventanas ausentes, termina con código 1. No imprime el token ni el cuerpo de errores HTTP. Renueva la sesión en Pi con `/login` cuando corresponda.

El panel consulta al abrirse, al seleccionar modelo, al iniciar sesión, al terminar respuestas y mediante un heartbeat de 30 segundos mientras está abierto. Estas rutas fuerzan la consulta; el TTL no es un límite global de frecuencia. Una guarda evita consultas simultáneas. Si falla, se muestra el error en lugar de conservar cifras anteriores. Después de actualizar, usa `/reload` en Pi.

### Inspector de pruebas: retirado

HCD-085 retiró el TDD Live Inspector y su tool `run_test_suite`. El progreso de las tareas HCD y sus comprobaciones sigue en `/tasks`. Si una versión anterior dejó `~/.pi/agent/extensions/gomemory-tdd.ts` sin editar, `install` y `update` lo borran; si lo editaste, se conserva con un aviso y puedes borrarlo y ejecutar `/reload`.

### Barra de tokens (`/tokens`)
Alterna la visibilidad de la barra inferior con el consumo acumulado de tokens de entrada, salida, lecturas de caché y coste estimado.

### Aviso de actualizaciones
Desde v1.13.0, el pie muestra **`↑ actualizaciones disponibles`** en su extremo derecho, delante del modelo y la rama, cuando `pi-gomemory-setup deps-status` sale con 10. Al iniciar la sesión consulta el estado, ejecuta `deps-refresh` en segundo plano (solo vuelve a la red si la última consulta tiene más de una hora) y vuelve a consultar, así que una release se anuncia como mucho una hora después de publicarse, sin retrasar el arranque. Con `/tokens` apagado, el aviso se publica en el pie por defecto de Pi. Solo informa: para instalar, responde a la pregunta del launcher, ejecuta `pi-gomemory-setup deps-update` o usa `/componentes`. `PI_GOMEMORY_NO_UPDATE_CHECK=1` lo desactiva. Vacía, `0` o `false` lo dejan activo, igual que en el launcher y en `check-update`.

### Componentes del meta-harness (`/componentes`)
Lista pi-gomemory, `pi`, `mem`, `codegraph`, `herdr` y los paquetes fijados, con la versión instalada y la publicada (lo lee de `pi-gomemory-setup components --json`). Elige una herramienta para marcarla o desmarcarla y luego **Aplicar**: se ejecuta `pi-gomemory-setup sync --only=…` con lo marcado y un plazo de 60 minutos, y al terminar ves qué se instaló, qué se actualizó y qué falló. `pi` y `mem` no se pueden desmarcar. pi-gomemory y los paquetes fijados aparecen solo como información: se actualizan con `update` e `install`. Si `pi` quedó actualizado, reinicia Pi para usar la versión nueva. Funciona aunque `PI_GOMEMORY_NO_UPDATE_CHECK` esté puesta.

### Ventana de contexto (`/context`)
Muestra el desglose de espacio ocupado en la ventana de contexto del modelo para identificar qué archivos o mensajes consumen más memoria.

## Catálogo de skills instaladas

El instalador copia y sanitiza 20 skills en `~/.pi-gomemory/skills/`, junto con enlaces a las skills de los demás agentes. Si `mem` instaló su propia copia de una skill en otro agente, esa copia gana a la embebida.

**Skills al día tras `mem update`.** Al iniciar cada sesión y después de un `mem update` ejecutado desde Pi, `gomemory-status` lanza `pi-gomemory-setup skills-sync --check`. Si hay copias desfasadas, sincroniza y avisa de que `/reload` carga la versión nueva. Las copias de gomemory editadas a mano no se sobrescriben nunca: si difieren de la versión de `mem`, Pi avisa en amarillo y `pi-gomemory-setup skills-sync --restore <skill>` las repone, salvo que la carpeta tenga otros archivos tuyos. En el repositorio de pi-gomemory, `TestHCD080EmbeddedSkillsMatchMem` falla si `skills/` se queda por detrás de las copias de `mem` de tu máquina (se omite si no hay ninguna o con `PI_GOMEMORY_SKIP_SKILL_DRIFT=1`); `make skills-embed` las copia.

Harness Continual Development (HCD):
`hcd`, `hcd-define`, `hcd-clarify`, `hcd-approve`, `hcd-slice`, `hcd-tasks`, `hcd-implement`, `hcd-verify`, `hcd-analyze`, `hcd-close`.

Rigor y expansión:
`verify-loop`, `search-first`, `iterative-retrieval`, `strategic-compact`, `instincts`, `atomic-decomposition`.

Auditoría y control de calidad:
`adversarial-consensus-review`, `defensive-qa-audit`.

Memoria y asistencia técnica:
`constitution`, `socratica`.

## Supresión y prevención de advertencias (warnings)

El instalador `pi-gomemory-setup` previene y elimina en origen las advertencias habituales de Pi Coding Agent:

1. **Colisiones de nombres de skills (`name "<skill>" collision`):**
   - Pi emite warnings al inicio si descubre skills con el mismo nombre en múltiples rutas del anfitrión (`~/.agents/skills`, `~/.claude/skills`, `~/.config/opencode/skills`, etc.).
   - `pi-gomemory-setup` consolida todas las skills en un repositorio central unificado (`~/.pi-gomemory/skills`), resolviendo colisiones mediante prioridad determinista (las 21 de gomemory primero, seguidas por el orden de precedencia del anfitrión) y registrando solo esta ruta única en `~/.pi/agent/settings.json`.

2. **Formato YAML frontmatter en skills (`failed to parse skill file`):**
   - Si un archivo `SKILL.md` contiene caracteres de dos puntos (`:`) sin comillas en el campo `description:`, el analizador de YAML de Pi emite warnings o descarta la skill.
   - La función `SanitizeFrontmatter` del instalador detecta descripciones con `:` no entrecomilladas y las asegura automáticamente con comillas dobles.

3. **Superposición de cabecera visual:**
   - Para impedir que `pi-cc-extensions` tape el banner de gomemory con su propia cabecera de inicio, `pi-gomemory-setup` impone `showStartupHeader: false` en `~/.pi/agent/pi-cc-extensions.json`.


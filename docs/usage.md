# Guía de uso

Esta guía cubre los comandos cotidianos y opciones del lanzador pi-gomemory.

## Inicio rápido

Para iniciar Pi con la paleta de color gomemory y contexto de memoria activo:

```bash
pi-gomemory
```

Si estás dentro de un proyecto donde ya ejecutaste `mem init` o existe `.gomemory/memories.db`, el lanzador inyecta las memorias más recientes con el parámetro `--append-system-prompt` de Pi.

## Consulta de cuotas de Google AI Pro

Para revisar el porcentaje de cuota restante y el tiempo hasta el reinicio de límites:

```bash
pi-gomemory usage
```

También funciona llamando al binario de configuración:

```bash
pi-gomemory-setup usage
```

Ejemplo de salida en terminal (los grupos y cifras los entrega la API de tu cuenta):

```text
Google AI Pro (g1-pro-tier)
Cuenta: tu-cuenta@ejemplo.com

 Gemini Models
   [################----] Weekly Limit Remaining: 80.0% left · resets 3d 21h
   [################----] Five Hour Limit Remaining: 80.0% left · resets 4h 17m
```

Si la API no responde o el token caducó, el comando termina con código 1 y un mensaje de error. Nunca muestra cifras aproximadas. Abre `pi` una vez para renovar el token y vuelve a intentarlo.

Desde v1.13.8, una respuesta 200 sin ningún límite (`null`, `{}` o grupos vacíos) o con un porcentaje fuera de rango cuenta como fallo de ese endpoint: el comando prueba el siguiente y, si ninguno trae cuotas, termina con código 1, y el panel lateral muestra el error en vez de «Cargando cuota...». Si Google envía cuotas pero no el nombre del plan, la primera línea dice «Plan no informado» en lugar de suponer «Google AI Pro (g1-pro-tier)».

## Antigravity devuelve HTTP 429 con cuota disponible

Si `/sidebar` o `pi-gomemory usage` muestran cuota, pero la generación termina con `Resource has been exhausted`, no des por probado un problema de login ni borres las credenciales. En HCD-040 se reprodujo un patrón asociado al contenido o ubicación de las instrucciones de Pi con la misma credencial.

Consulta el [diagnóstico de Antigravity](antigravity.md) antes de aplicar la referencia local opt-in. El guard conserva el texto, pero baja su prioridad al trasladarlo a un turno de usuario; no se instala con `install` ni `update`. Para desactivar una copia instalada manualmente, inicia un proceso nuevo con `ANTIGRAVITY_PROMPT_GUARD=off pi`. La validación de `/sidebar` incluye la confirmación de la persona, no una inspección visual del agente.

## Referencia de comandos CLI

El binario `pi-gomemory-setup` acepta los siguientes argumentos:

```text
Uso: pi-gomemory-setup [comando]

Comandos:
  install       Ejecuta la instalación completa (opción por defecto)
  usage         Muestra cuotas y límites de consumo de Google AI Pro (--json disponible)
  claude-usage  Muestra telemetría agregada de tokens de Claude Code (hoy, semana, total)
  anthropic-usage  Devuelve los límites del plan de Anthropic en JSON
  codex-usage   Devuelve los límites del plan de OpenAI Codex en JSON
  doctor        Verifica binarios, tema, MCP, plugins, launcher y otros agentes detectados
  mcp           Registra gomemory, codegraph y codebase-memory en mcp.json
  codemode      Activa, desactiva o consulta el modo codemode híbrido (on | off | status)
  theme         Instala la paleta gomemory en ~/.pi/agent/themes/
  plugin        Configura pi-antigravity y pi-cc-extensions en settings.json
  skills        Reúne las skills de gomemory y del anfitrión en ~/.pi-gomemory/skills/
  skills-sync   Sincroniza el repositorio central de skills y purga copias legadas
  harness       Genera el launcher pi-gomemory en ~/.local/bin (o PI_GOMEMORY_BIN_DIR)
  version       Muestra la versión instalada
  deps-status   Lista los componentes desactualizados frente a la versión instalada real (sale con 10)
  deps-update   Actualiza los componentes desactualizados
  update        Actualiza pi-gomemory desde la última release verificada
  uninstall     Retira lo instalado por pi-gomemory (ver «Desinstalar» en installation.md)
  help          Muestra este mensaje de ayuda
```

## Codemode y contexto vivo

`install` y `update` usan modo MCP **directo** (`codemode off`) por defecto para `gomemory`, `codegraph` y `codebase-memory`. Estas órdenes permiten activar el modo híbrido o volver a directo, guardando una elección explícita para las siguientes instalaciones:

```bash
pi-gomemory-setup codemode on
pi-gomemory-setup codemode off
pi-gomemory-setup codemode status
```

La preferencia y la propiedad de `+codemode` se guardan en `~/.pi-gomemory/codemode.json`. Las herramientas MCP declaradas se llaman directamente; el resto se consulta desde `codemode` con `tools.<nombre>(args)`. `mcp` regenera los servidores y conserva un perfil híbrido ya activo.

`pi-gomemory-setup lite on` retira los servidores de grafo gestionados por pi-gomemory y conserva los personalizados. El modo se mantiene al ejecutar `install`, `update`, `mcp` o cambiar codemode. Usa `lite status` para consultarlo y `lite off` para restaurar los servidores gestionados que falten.

Al abrir o cambiar una sesión, `gomemory-context.ts` carga `mem context` una vez con el directorio del proyecto y un timeout de 3000 ms. Inyecta hasta 24 576 bytes de texto UTF-8 más el aviso de recorte, si hace falta, y el bloque de protocolo de Pi. Si falla o no devuelve contexto, muestra una advertencia por sesión y mantiene solo ese bloque. `/plan` añade su método en otra sección del prompt.

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
Al abrir Pi con `pi-gomemory`, la cabecera muestra la versión de pi-gomemory y la de gomemory instalada. El launcher las exporta en `PI_GOMEMORY_RUNNING_VERSION` (de `pi-gomemory-setup version`) y `GOMEMORY_INSTALLED_VERSION` (de `mem --version`). Si falta alguna, se omite. Con `NO_COLOR` se dibuja sin colores.

Pi solo admite una cabecera. Si pi-cc-extensions tiene `showStartupHeader: true`, su cabecera sustituye al banner; por eso `pi-gomemory-setup install` (y `update`) fija esa clave en `false` en cada instalación, y `pi-gomemory-setup doctor` avisa en «Banner gomemory» si vuelve a activarse (por ejemplo desde `/ccstyle`).

### Modo plan (`/plan`)
Mientras el modo plan está activo, Pi bloquea las modificaciones comunes del agente.

- `/plan on` activa el modo sin enviar una petición.
- `/plan off` lo desactiva sin iniciar un turno. El siguiente turno ya no recibe las instrucciones de modo plan.
- `/plan status` muestra si está activo, sin cambiar el estado.
- `/plan run` conserva el comportamiento anterior: desactiva el modo, pero no ejecuta el plan automáticamente.
- `/plan` sin argumentos alterna el modo; `/plan <petición>` lo activa y envía la petición.

### Panel de tareas (`/tasks`)
`/tasks` muestra u oculta las tareas de la unidad HCD activa encima del editor. La cabecera lleva una barra de progreso (`▰▰▰▰▱▱▱▱▱▱  5/13 · 38 %`). El estado de cada tarea viene del motor y se dibuja como un círculo que se llena:

| Glifo | Estado |
|---|---|
| `○` | pendiente |
| `◐` (gira) | en curso: la tarea activa, aunque no haya ninguna llamada en marcha |
| `●` | verificada |
| `◍` | estuvo verificada, pero el código cambió y hay que volver a verificarla |
| `✗` | fallida |

Cada estado tiene también su color; el glifo se mantiene para quien no distingue colores o usa un tema monocromo. Debajo de cada tarea, el árbol (`├─`, `└─`) muestra las llamadas de la sesión con su estado y duración; la que está en marcha usa el mismo spinner que `/tdd`.

La tarea en curso muestra por defecto sus últimas cinco llamadas. Las tareas completadas quedan resumidas; usa `/tasks expand N` o haz clic en el resumen para abrirlas y `/tasks collapse N` para cerrarlas. Con `/tasks focus N` puedes decidir a qué tarea se asignan las próximas llamadas.

Si la unidad tiene más de 8 tareas, se ven 8 alrededor de la que está en curso, con `↑ N más` y `↓ N más` en los bordes. Para ver el resto usa la rueda del ratón sobre el panel, `Alt+Shift+↑` / `Alt+Shift+↓` desde el input o un clic en esas líneas. `↑` y `↓` solos siguen recorriendo el historial, y `Alt+↑` sigue recuperando los mensajes en cola.

### Sugerencia del siguiente comando
Con el input vacío, Pi muestra en gris lo que probablemente quieras escribir a continuación. `Tab` lo copia al input sin enviarlo; si ya escribiste algo, `Tab` vuelve a ser el autocompletado de siempre.

- Con una unidad HCD activa, la sugerencia es el siguiente paso que indica el motor (por ejemplo, `/hcd-implement`). Sale de `mem hcd show` y no consume tokens.
- Sin unidad activa, al terminar cada turno se pide al modelo de la sesión una propuesta de una línea. La petición lleva el último mensaje tuyo y la última respuesta del agente, usa el razonamiento mínimo y un tope de 300 tokens; con Claude Opus cuesta unos 0,002-0,003 USD. Si empiezas a escribir o lanzas otro turno, se cancela.

La extensión `gomemory-suggest.ts` envuelve el editor que haya instalado otra extensión (por ejemplo, el `❯` de pi-cc-extensions) en lugar de sustituirlo.

### Panel lateral estilo OpenCode (`/sidebar`)
Alterna la visibilidad de la barra lateral en la esquina superior derecha (`anchor: "top-right"`). Muestra la sesión/HCD activo, el consumo de ventana de contexto en tiempo real (tokens, porcentaje y coste), cuotas de Google AI Pro, límites de Anthropic u OpenAI Codex y telemetría de Claude Code, y los servidores MCP declarados en `mcp.json` con el estado `Configured` (no se comprueba la conexión en vivo; sin `mcp.json` válido la lista queda vacía). El panel flota sin capturar el teclado, así que puedes seguir escribiendo; el título incluye la versión instalada. Se oculta con `/sidebar` o automáticamente si el terminal mide menos de 100 columnas.

**Límites del plan de Anthropic.** Si Pi está conectado a Anthropic con OAuth (suscripción Claude Pro/Max, `/login` en pi), el bloque «▼ Anthropic · plan» muestra el porcentaje usado de la **sesión** (ventana de 5 h) y de la **semana**, con el tiempo hasta cada reinicio. Si Anthropic los envía, añade también los límites semanales de Opus o Sonnet. Los límites se obtienen de forma pasiva a partir de las cabeceras `anthropic-ratelimit-unified-*` que llegan en cada respuesta del modelo (`after_provider_response`), recalculando la cuenta atrás en cada renderizado. Como respaldo inicial al abrir el panel sin respuesta previa, realiza una única consulta con `pi-gomemory-setup anthropic-usage` evitando saturar la cuota de la API (HTTP 429).

Si la consulta falla, el bloque muestra el error completo y nunca cifras viejas o inventadas; por ejemplo, token caducado (abre pi para renovarlo) o binario sin `anthropic-usage` (ejecuta `pi-gomemory-setup update`).

- Clic en `[−]` de la cabecera: minimiza el panel a una sola fila (`[ HCD · ... [+] ]`); clic en `[+]`: lo restaura. El ratón solo funciona en el modo `fullscreen` de Pi (el predeterminado); en `--tui-mode regular` usa los comandos.
- `/sidebar min` (o alias `/sidebar m`, `/sidebar -`): alterna el modo minimizado desde el teclado.
- `/min`: minimiza o restaura a la vez `/sidebar` y `/tdd`. Si alguno está expandido, minimiza los dos; si los dos están minimizados, los restaura.

Con un título largo, el título se recorta con `…` y el control `[−]` sigue visible.

### Cuota de OpenAI Codex

Desde v1.12.0, cuando el proveedor activo es Codex (`openai-codex`), `/sidebar` muestra «▼ Codex Quota» con las ventanas primaria y secundaria del plan, habitualmente cinco horas y una semana. Cada fila indica el porcentaje **restante**, una barra y el tiempo hasta el reinicio. La barra cambia a amarillo por debajo del 50 % restante y a rojo por debajo del 20 %.

```bash
pi-gomemory-setup codex-usage
```

El JSON devuelve `planType` y `limits`; cada límite contiene `name`, `percent` (porcentaje **usado**), `resetsAt` y `resetIn`. El panel calcula lo restante como `100 - percent`. La consulta usa el token de `openai-codex` y, si existe, `accountId` en `~/.pi/agent/auth.json`, contra `https://chatgpt.com/backend-api/wham/usage`. Es una API externa sin contrato estable garantizado por este proyecto.

El comando no renueva tokens. Sin sesión OAuth, con token caducado, respuesta HTTP distinta de 200, JSON inválido, respuesta superior a 1 MiB o ventanas ausentes, termina con código 1. No imprime el token ni el cuerpo de errores HTTP. Renueva la sesión en Pi con `/login` cuando corresponda.

El panel consulta al abrirse, al seleccionar modelo, al iniciar sesión, al terminar respuestas y mediante un heartbeat de 30 segundos mientras está abierto. Estas rutas fuerzan la consulta; el TTL no es un límite global de frecuencia. Una guarda evita consultas simultáneas. Si falla, se muestra el error en lugar de conservar cifras anteriores. Después de actualizar, usa `/reload` en Pi.

### Inspector de pruebas en tiempo real (`/tdd`)

Desde v1.12.1, las interrupciones y los timeouts se informan como fallos con su causa, cada prueba Playwright se identifica por proyecto y las recuperadas en un reintento se cuentan como `flaky` (véase [problemas conocidos](known-issues.md)).
La sección «Verificaciones HCD · Bash» muestra una fila por tarea desde que empieza `mem hcd verify`. El glifo animado cambia a `✓` o `✗` al terminar. Watch conserva esas filas separadas de los resultados de las suites.

Abre el overlay **TDD Live Inspector**. Se coloca debajo del panel lateral, en la misma columna derecha, sin solaparse, y ejecuta las pruebas **reales** del proyecto abierto.

**Suites según el proyecto.** Cada suite se activa solo si el proyecto tiene lo que necesita; si no, aparece como «omitida» con su motivo, nunca como éxito ni como fallo:

| Suite | Se activa con | Comando |
|---|---|---|
| Backend | `go.mod` | `go test -json ./...` |
| Frontend | `package.json` con script `test` | `npm test` |
| Playwright E2E | `playwright.config.*` | `npx playwright test --reporter=line` |
| CI & Static | `go.mod` y/o `tsconfig.json` | `go vet ./...` y/o `npx tsc --noEmit` |
| Data & Graph | unidad HCD activa | `mem hcd show --json` |

**Recorrido en vivo.** Los procesos se ejecutan sin shell y su salida se lee línea a línea, así que el panel avanza test a test mientras corren. Muestra solo las suites habilitadas y, dentro de cada una:
- los tests en curso, con el spinner braille de Pi (`⠋⠙⠹…`);
- los fallidos (`✗`), con archivo:línea, el paso que falló (Playwright), el mensaje real (por ejemplo, `Expected 13 · Received 7`) y un diagnóstico;
- los aprobados, resumidos en un conteo (`✓ 108 aprobados`).

Las suites omitidas aparecen agrupadas al final, bajo «Omitidas (n)».

**Límite por línea.** Cada línea de salida admite hasta 64 Ki caracteres. Una más larga (un JSON o un bundle minificado sin saltos de línea) termina el proceso y la ejecución falla con «Salida rechazada: una línea excede el límite de 64 Ki caracteres». Para el diagnóstico, el panel guarda las últimas 40 líneas, cada una recortada a 1 Ki caracteres y 64 KiB en total; el parser de cada suite sigue recibiendo las líneas completas.

**Subcomandos:**
- `/tdd`: muestra u oculta el panel y lanza todas las suites.
- `/tdd run [suite]`: ejecuta `all` o una suite (`backend`, `frontend`, `playwright`, `ci`, `data`). Si ya hay una ejecución en curso, avisa y la nueva queda en cola; si lo que corre es watch, lo cancela en lugar de esperarlo.
- `/tdd min` (o alias `/tdd m`, `/tdd -`), o clic en `[−]` / `[+]`: minimiza o restaura el panel.
- `/tdd watch`: ejecuta las pruebas una vez cuando el agente termina el prompt (`agent_settled`), solo si cambió archivos con `edit`, `write` o `bash`. No bloquea al agente, no se dispara en prompts de solo lectura y no repite si el agente ya ejecutó `run_test_suite all` tras su último cambio. Los avisos que llegan durante una ejecución se agrupan en una sola pendiente.
- `/tdd report`: publica en el chat el informe de la última ejecución (resultado por suite y cada fallo con mensaje, archivo:línea, paso, diagnóstico y sugerencia). Queda también en el contexto del modelo sin iniciar un turno, así que puedes pedir «corrige esos fallos» sin volver a ejecutar. Si aún no hubo ejecución, lo indica.

**Herramienta del agente (`run_test_suite`).** Devuelve al modelo cada fallo con su suite, test, mensaje, ubicación, paso, diagnóstico y sugerencia, para que corrija y vuelva a ejecutar. Mientras corre, envía avances. Con `filter` ejecuta solo los tests que coinciden, para ver el rojo y luego el verde de un cambio sin esperar la suite entera; la suite queda marcada como «parcial». El filtro se traduce al ejecutor: `go test -run`, `pytest -k`, `vitest -t`, `playwright -g`, `phpunit --filter`, `mvn -Dtest` y `gradle --tests`. La suite completa queda para la regresión final.

**Paralelismo.** Go, Vitest y Playwright ya reparten los tests entre núcleos. Si el proyecto tiene `pytest-xdist` (declarado o instalado en `.venv`), `/tdd` añade `-n auto`; si existe `vendor/bin/paratest`, lo usa en lugar de `phpunit`. No instala nada ni fuerza `mvn -T` o `gradle --parallel`.

**Tests lanzados por el agente con Bash.** `/tdd` también muestra los tests que el agente ejecuta por su cuenta (`go test`, `pytest`, `vitest`, `phpunit`, `mvn`/`gradle test`, `playwright`), según van terminando y no solo al final. `go test` se lee test a test con `-json` o `-v`; sin ellos solo se conoce el resultado global. En `mem hcd verify`, también detrás de `cd sub &&`, el panel muestra los tests de la comprobación a partir de su evidencia, sin cambiar el veredicto del motor. Un Bash que ejecuta la suite completa deja a watch sin cambios pendientes solo si es la única suite de tests del proyecto: con Go y un frontend con tests, un `go test ./...` no cubre el frontend y watch sigue comprobándolo.

### Barra de tokens (`/tokens`)
Alterna la visibilidad de la barra inferior con el consumo acumulado de tokens de entrada, salida, lecturas de caché y coste estimado.

### Aviso de actualizaciones
Desde v1.13.0, el pie muestra **`↑ actualizaciones disponibles`** en su extremo derecho, delante del modelo y la rama, cuando `pi-gomemory-setup deps-status` sale con 10. Al iniciar la sesión consulta el estado, ejecuta `deps-refresh` en segundo plano (solo vuelve a la red si la última consulta tiene más de una hora) y vuelve a consultar, así que una release se anuncia como mucho una hora después de publicarse, sin retrasar el arranque. Con `/tokens` apagado, el aviso se publica en el pie por defecto de Pi. Solo informa: para instalar, responde a la pregunta del launcher o ejecuta `pi-gomemory-setup deps-update`. `PI_GOMEMORY_NO_UPDATE_CHECK=1` lo desactiva. Vacía, `0` o `false` lo dejan activo, igual que en el launcher y en `check-update`.

### Ventana de contexto (`/context`)
Muestra el desglose de espacio ocupado en la ventana de contexto del modelo para identificar qué archivos o mensajes consumen más memoria.

## Catálogo de skills instaladas

El instalador copia y sanitiza 21 skills en `~/.pi-gomemory/skills/`, junto con enlaces a las skills de los demás agentes:

Harness Continual Development (HCD):
`hcd`, `hcd-define`, `hcd-clarify`, `hcd-approve`, `hcd-slice`, `hcd-tasks`, `hcd-implement`, `hcd-verify`, `hcd-analyze`, `hcd-close`.

Rigor y expansión:
`verify-loop`, `search-first`, `iterative-retrieval`, `strategic-compact`, `instincts`, `atomic-decomposition`.

Auditoría y control de calidad:
`adversarial-consensus-review`, `defensive-qa-audit`.

Memoria y asistencia técnica:
`codebase-memory`, `constitution`, `socratica`.

## Supresión y prevención de advertencias (warnings)

El instalador `pi-gomemory-setup` previene y elimina en origen las advertencias habituales de Pi Coding Agent:

1. **Colisiones de nombres de skills (`name "<skill>" collision`):**
   - Pi emite warnings al inicio si descubre skills con el mismo nombre en múltiples rutas del anfitrión (`~/.agents/skills`, `~/.claude/skills`, `~/.config/opencode/skills`, etc.).
   - `pi-gomemory-setup` consolida todas las skills en un repositorio central unificado (`~/.pi-gomemory/skills`), resolviendo colisiones mediante prioridad determinista (las 21 de gomemory primero, seguidas por el orden de precedencia del anfitrión) y registrando solo esta ruta única en `~/.pi/agent/settings.json`.

2. **Formato YAML frontmatter en skills (`failed to parse skill file`):**
   - Si un archivo `SKILL.md` contiene caracteres de dos puntos (`:`) sin comillas en el campo `description:`, el analizador de YAML de Pi emite warnings o descarta la skill.
   - La función `SanitizeFrontmatter` del instalador detecta descripciones con `:` no entrecomilladas y las asegura automáticamente con comillas dobles.

3. **Dependencias provistas por el host (`host-provided extension packages`):**
   - Pi advierte si un paquete instalado (como `pi-antigravity`) lista librerías que el runtime de Pi ya provee (`typebox`) bajo `dependencies` en lugar de `peerDependencies`.
   - El instalador parchea el `package.json` de `pi-antigravity` asegurando `"peerDependencies": { "typebox": "*" }` y retirándolo de `dependencies`.

4. **Superposición de cabecera visual:**
   - Para impedir que `pi-cc-extensions` tape el banner de gomemory con su propia cabecera de inicio, `pi-gomemory-setup` impone `showStartupHeader: false` en `~/.pi/agent/pi-cc-extensions.json`.


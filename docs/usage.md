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
`/plan` activa o desactiva el modo plan, que bloquea las modificaciones; `/plan run` lo desactiva. `/plan <petición>` activa el modo y envía la petición en el mismo paso.

### Panel de tareas (`/tasks`)
Muestra u oculta, encima del editor, las tareas de la unidad HCD activa (`✓` verificada, `○` pendiente, `◌` verificación caducada, `✗` con fallo). Se actualiza solo tras cada `mem hcd` y al terminar cada respuesta.

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

**Subcomandos:**
- `/tdd`: muestra u oculta el panel y lanza todas las suites.
- `/tdd run [suite]`: ejecuta `all` o una suite (`backend`, `frontend`, `playwright`, `ci`, `data`). Si ya hay una ejecución en curso, avisa y la nueva queda en cola.
- `/tdd min` (o alias `/tdd m`, `/tdd -`), o clic en `[−]` / `[+]`: minimiza o restaura el panel.
- `/tdd watch`: ejecuta las pruebas una vez cuando el agente termina el prompt (`agent_settled`), solo si cambió archivos con `edit`, `write` o `bash`. No bloquea al agente, no se dispara en prompts de solo lectura y no repite si el agente ya ejecutó `run_test_suite all` tras su último cambio. Los avisos que llegan durante una ejecución se agrupan en una sola pendiente.
- `/tdd report`: publica en el chat el informe de la última ejecución (resultado por suite y cada fallo con mensaje, archivo:línea, paso, diagnóstico y sugerencia). Queda también en el contexto del modelo sin iniciar un turno, así que puedes pedir «corrige esos fallos» sin volver a ejecutar. Si aún no hubo ejecución, lo indica.

**Herramienta del agente (`run_test_suite`).** Devuelve al modelo cada fallo con su suite, test, mensaje, ubicación, paso, diagnóstico y sugerencia, para que corrija y vuelva a ejecutar. Mientras corre, envía avances.

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


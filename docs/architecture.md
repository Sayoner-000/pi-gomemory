# Arquitectura de pi-gomemory

Este documento describe la estructura interna, los componentes y las decisiones de diseño de pi-gomemory.

## Flujo general

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                           pi-gomemory-setup                             │
│                  Binario Go con recursos en embed.FS                    │
└────────────────────────────────────┬────────────────────────────────────┘
                                     │
         ┌───────────────────────────┼───────────────────────────┐
         ▼                           ▼                           ▼
┌──────────────────┐       ┌──────────────────┐       ┌──────────────────┐
│  Servidores MCP  │       │  Tema gomemory   │       │  Plugins y TUI   │
│  en mcp.json     │       │  themes/         │       │  settings.json   │
├──────────────────┤       ├──────────────────┤       ├──────────────────┤
│ mem (gomemory)   │       │ Paleta 24-bit    │       │ pi-antigravity   │
│ codegraph        │       │ Truecolor        │       │ pi-cc-extensions │
│ codebase-memory  │       │                  │       │ gomemory-status  │
└──────────────────┘       └──────────────────┘       └──────────────────┘
         │                           │                           │
         └───────────────────────────┼───────────────────────────┘
                                     │
         ┌───────────────────────────┴──────────────────────────┐
         ▼                                                      ▼
┌──────────────────────────────────┐  ┌──────────────────────────────────┐
│        21 Skills gomemory        │  │   Lanzador Herdr (pi-gomemory)   │
├──────────────────────────────────┤  ├──────────────────────────────────┤
│ Sanitización YAML frontmatter    │  │ Variables de terminal y PATH     │
│ HCD, Expand, Review, QA, etc.    │  │ Inyección automática de mem ctx  │
└──────────────────────────────────┘  └──────────────────────────────────┘
```

## Recursos embebidos en el binario

El instalador compila todos sus activos dentro del propio binario mediante `embed.FS` en `embed.go`:

```go
//go:embed skills/* extensions/* themes/* config/*
var EmbeddedAssets embed.FS
```

Esto evita depender de carpetas clonadas o rutas relativas en disco. El ejecutable se puede mover a cualquier equipo y conserva intactas las 21 skills, el tema (`themes/gomemory.json`), la configuración por defecto de `pi-cc-extensions` (`config/pi-cc-extensions.json`) y la extensión de estado. El embebido es la única fuente de lo que se instala: el instalador no lee carpetas locales y no escribe nada en el directorio desde el que se ejecuta. Con las skills de otros agentes no copia ni modifica nada. Reúne todo en el repositorio central `~/.pi-gomemory/skills`: las skills embebidas como copias (con una marca `.pi-gomemory` que guarda el SHA256 del `SKILL.md`, para no pisar nunca una copia editada) y las del anfitrión como enlaces simbólicos, con prioridad `~/.agents/skills` → `~/.config/opencode/skills` → `~/.claude/skills` → `~/.codex/skills` → `~/.gemini/skills`. Pi registra solo ese repositorio en `settings.json`, así que nunca ve dos veces la misma skill. `doctor` muestra el total y las colisiones resueltas.

En `mcp.json` solo se actualizan las tres entradas propias. En `pi-cc-extensions.json` se conservan las claves del usuario, salvo `showStartupHeader`, que se desactiva para no sustituir el banner. `AGENTS.md` deja de recibir instrucciones nuevas: la instalación retira únicamente el protocolo heredado. Un JSON ilegible detiene su paso; las escrituras de varios archivos no forman una transacción conjunta.

## Configuración de servidores MCP

El instalador actualiza `~/.pi/agent/mcp.json` para conectar tres herramientas complementarias:

[gomemory](https://github.com/Sayoner-000/gomemory) (`mem mcp`):
Gestiona la memoria del proyecto, los checkpoints, las sinapsis entre sesiones, el protocolo de revisión adversarial y el flujo de HCD (Harness Continual Development).

[codegraph](https://github.com/colbymchenry/codegraph) (`codegraph serve --mcp`):
Proporciona un grafo de código en SQLite generado a partir de AST para más de 30 lenguajes. Permite buscar símbolos, rutas de llamadas y blast radius en una sola consulta.

[codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp) (`codebase-memory-mcp`):
Ofrece consultas de grafo estructural profundo y trazado de llamadas para bases de código grandes.

La función `InstallMCPServers` lee el archivo `mcp.json` existente y solo agrega o actualiza estas tres entradas. No borra ni altera otros servidores que el usuario ya tenga configurados.

## Contexto por sesión y codemode

`gomemory-context.ts` inicia `mem context` en `session_start`, con el directorio de la sesión y un timeout de 3000 ms. `before_agent_start` espera esa carga y añade la sección `gomemory` a `systemPromptOptions.sections`. La salida se reutiliza dentro de la sesión y conserva hasta 24 576 bytes de texto UTF-8; el aviso de recorte y el bloque Pi se añaden después. Si la consulta falla o llega vacía, la sesión conserva solo el bloque Pi y muestra una advertencia.

`gomemory-plan.ts` usa la sección `gomemory-plan`, por lo que no reemplaza el prompt completo. El bloque Pi no enumera servidores según `selectedTools`: en Pi 1.5.2 los MCP pueden seguir conectándose cuando ocurre ese evento. Las declaraciones del prompt determinan qué herramientas se ofrecen directamente.

`InstallAll` deja los servidores MCP en modo directo (`codemode off`) por defecto y respeta un `codemode on` explícito. `SetCodemodePreference` guarda las órdenes explícitas on/off y la propiedad de `+codemode` en `~/.pi-gomemory/codemode.json`. La desinstalación reconoce las definiciones MCP directas e híbridas exactas y conserva las personalizadas.

`RetireLegacyInstructions` reconoce marcadores y variantes históricas publicadas, almacenadas en `pkg/installer/legacy/`. La migración conserva texto ajeno, permisos y enlaces. Si solo queda el bloque propio, borra el archivo normal o vacía el destino enlazado. `doctor` avisa si el protocolo sigue presente.

## Google AI Pro y monitor de consumo

### Plugin pi-antigravity

Para conectar Pi con los modelos de Google AI Pro mediante Antigravity, el instalador registra el paquete `npm:pi-antigravity@0.14.0` en `settings.json` (junto con `npm:pi-cc-extensions@1.0.21`). Ambos van fijados a una versión exacta, porque Pi no reinstala un paquete fijado así. `uninstall --packages` retira también las entradas de pins anteriores que escribió pi-gomemory (`historicPins` en `plugin.go`). Además, inyecta la dependencia `typebox: "*"` en la sección `peerDependencies` de la configuración para evitar avisos molestos de npm al resolver tipos.

### Guard opcional de generación

La [referencia del guard](../tests/antigravity-prompt-guard/guard.mjs) vive en `tests/`, fuera de los patrones de `embed.go` y de los recursos que instala `InstallAll`. Solo se carga si la persona la copia manualmente a las extensiones de Pi. Su hook `before_provider_request` actúa sobre peticiones `agent` de Antigravity con instrucciones textuales compatibles: clona el payload, conserva herramientas y metadatos, sustituye `systemInstruction` por un texto breve y antepone las instrucciones originales al primer turno de usuario. No cambia otros proveedores ni generación de imágenes y permite desactivación por entorno.

Conservar el texto no conserva la jerarquía: las instrucciones trasladadas tienen menor prioridad. El [diagnóstico y procedimiento opt-in](antigravity.md) describen ese riesgo; la mitigación no sustituye las protecciones locales ni una corrección upstream. Las 22 pruebas offline se ejecutan también desde la suite de Go, que requiere Node.js.

### Monitor de cuotas

El archivo `pkg/installer/usage.go` lee la cuenta activa guardada en `~/.pi/agent/antigravity-accounts.json` y hace una llamada POST al endpoint `retrieveUserQuotaSummary`. Con la respuesta construye una barra de texto que muestra el porcentaje restante y el tiempo que falta para que se reinicien las cuotas semanal y de cinco horas.

Las dos salidas, texto (`FetchAndFormatUsage`) y JSON para el panel lateral (`FetchQuotaSummary`), validan la respuesta con `parseQuotaResponse`. `json.Unmarshal` acepta `null` y `{}` sin error, así que la función exige al menos un bucket y fracciones entre 0 y 1. Si la respuesta no cumple, el bucle pasa al endpoint siguiente, y si ninguno responde con cuotas devuelve error. Cuando falta `planLabel`, la etiqueta es «Plan no informado» (`planLabelUnknown`).

## Límites del plan de OpenAI Codex

`pkg/installer/codex_usage.go` implementa `FetchCodexUsage`, invocado por el comando `codex-usage` de `main.go`. Lee la entrada OAuth `openai-codex` de `auth.json`, comprueba su vencimiento y envía una petición GET a `https://chatgpt.com/backend-api/wham/usage`, con `Authorization` y, si está disponible, `ChatGPT-Account-Id`.

El cliente tiene un timeout de 10 segundos y un máximo de respuesta de 1 MiB. Solo acepta HTTP 200 y devuelve errores explícitos para autenticación, frecuencia, formato o ventanas ausentes. No refresca ni escribe credenciales. El DTO expone el plan y las ventanas primaria/secundaria con porcentaje usado y reinicio.

`gomemory-sidebar.ts` detecta el proveedor Codex y ejecuta el subcomando mediante `pi.exec`. Conserva una caché de cuota o error y una guarda de consulta en curso. Los eventos y el heartbeat de 30 segundos fuerzan el refresco; no debe interpretarse el TTL como un máximo de una petición cada 30 segundos. La presentación transforma porcentaje usado en restante, sin cambiar la respuesta JSON del CLI.

## Interfaz visual y extensiones

La interfaz combina dos componentes:

pi-cc-extensions:
Configura el formato visual de los diffs de edición y escritura en Pi con barras laterales y prefijos limpios. También habilita el comando `/context` para revisar el peso de los mensajes en la ventana de contexto.

gomemory-status.ts:
Extensión en TypeScript que añade al pie de la terminal el conteo de tokens de entrada, salida, lecturas de caché y coste acumulado. Se puede activar o desactivar con el comando `/tokens`. Desde v1.13.0 también anuncia `↑ actualizaciones disponibles` en el extremo derecho: en `session_start` lanza, sin esperarla, la secuencia `deps-status` → `deps-refresh` → `deps-status` mediante `pi.exec` y trata la salida 10 como «hay algo desactualizado». Con el pie propio apagado, publica el aviso con `ctx.ui.setStatus("gomemory-updates", …)`. La lógica de versiones sigue en Go: `DepsStatusFor` combina la última publicada de la caché con la versión instalada que relee cada probe (`liveVersion`, límite de 3 s), y la caché solo decide qué es lo último publicado.

gomemory-tasks.ts:
Lee las tareas y su estado de verificación desde `mem hcd show --json`. Los eventos de herramientas alimentan un árbol de actividad por tarea en memoria. El árbol resume las llamadas de la sesión; no cambia el estado que registra el motor HCD.

gomemory-sidebar.ts:
Extensión en TypeScript inspirada en OpenCode que renderiza un overlay lateral derecho (`anchor: "top-right"`, 36 cols, responsive) con la sesión/HCD activo, métricas de contexto en vivo (tokens, porcentaje y coste) y conectividad de servidores MCP. Se activa o desactiva con `/sidebar` y no captura el teclado (`nonCapturing`). Publica su altura en un estado compartido (`globalThis[Symbol.for("pi-gomemory.dock")]`) para que `/tdd` se apile debajo. En `fullscreen`, pi-tui entrega un clic como `press → release → click`; el panel alterna solo con el `click` de su cabecera.

gomemory-tdd.ts:
Extensión en TypeScript del TDD Live Inspector (`/tdd`, tool `run_test_suite`). `detectSuites` decide qué ejecutar según los archivos del proyecto. `streamLines` lanza cada comando con `spawn` (sin shell) y entrega la salida línea a línea, porque `pi.exec` solo la devuelve al terminar. Desde v1.13.8 acota la memoria: una línea pendiente de más de `MAX_LINE_CHARS` (64 Ki caracteres) termina el grupo de procesos y fuerza el código 1, y `tail`, que solo sirve de diagnóstico, guarda como mucho 40 extractos de 1 Ki y 64 KiB contados en UTF-8. El check `hcd` lee el JSON de `mem hcd show --json` de las líneas completas que recibe por `onLine`, no de `tail`. Los parsers de `go test -json` y del reporter `line` de Playwright llevan cada test al `TDDStore` en vivo, con su mensaje, archivo:línea y duración reales. Las ejecuciones se encadenan para no pisar el mismo store. El modo watch marca un cambio en `tool_execution_end` (`edit`, `write`, `bash`) y lanza una ejecución, sin esperarla, en `agent_settled`. No usa `turn_end` porque Pi espera ese evento en cada turno y bloquearía al agente entre tool calls. El estado final de Playwright sale de su informe JSON oficial; el flujo de líneas solo alimenta el progreso.

Las verificaciones HCD lanzadas desde Bash se siguen por identificador de llamada: `tool_execution_start` abre las filas y `tool_execution_end` registra el resultado. Sus filas viven aparte de las suites para que Watch no las borre ni cambie sus totales.

## Riesgos y límites

Las escrituras JSON conservan permisos y crean los archivos nuevos con modo `0600`. La retirada de archivos revalida su identidad antes de ejecutar la acción; no promete atomicidad frente a sustituciones durante esa acción. La migración de `AGENTS.md` conserva su modo y no crea uno nuevo si no existe. Los fallos y límites que siguen vigentes están en [problemas conocidos](known-issues.md).

## Sanitización de skills

Pi utiliza un analizador estricto de YAML para leer el bloque inicial de metadatos (`---`) de cada archivo `SKILL.md`. Si el campo `description:` contiene dos puntos sin comillas, el analizador falla y la skill no se carga.

La función `SanitizeFrontmatter` procesa el contenido antes de copiarlo a `~/.pi-gomemory/skills/`. Si detecta dos puntos dentro del texto de la descripción, escapa las comillas existentes y envuelve todo el valor entre comillas dobles.

## Lanzador para Herdr

El script `pi-gomemory` en `~/.local/bin/` prepara el entorno antes de invocar el binario de Pi:

* Establece `HERDR_ENV=1` y `COLORTERM=truecolor`.
* Prioriza `~/.local/bin` y `~/.gomemory/bin` en el `PATH`.
* Si el primer argumento es `usage`, delega en `pi-gomemory-setup usage`.
* Comprueba si el directorio actual tiene una base de datos de gomemory (`.gomemory/memories.db`). Si existe, ejecuta `mem context` y pasa el resultado mediante `--append-system-prompt` para que el agente arranque con el historial reciente del proyecto.

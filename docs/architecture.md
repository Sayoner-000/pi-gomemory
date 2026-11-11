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

Los archivos de la persona se fusionan, nunca se reemplazan: en `mcp.json` solo se tocan las tres entradas propias, en `pi-cc-extensions.json` solo se añaden las claves que falten, y en `AGENTS.md` el protocolo vive entre `<!-- pi-gomemory:start -->` y `<!-- pi-gomemory:end -->`. Si un JSON existente es inválido, el instalador se detiene sin tocarlo.

## Configuración de servidores MCP

El instalador actualiza `~/.pi/agent/mcp.json` para conectar tres herramientas complementarias:

[gomemory](https://github.com/Sayoner-000/gomemory) (`mem mcp`):
Gestiona la memoria del proyecto, los checkpoints, las sinapsis entre sesiones, el protocolo de revisión adversarial y el flujo de HCD (Harness Continual Development).

[codegraph](https://github.com/colbymchenry/codegraph) (`codegraph serve --mcp`):
Proporciona un grafo de código en SQLite generado a partir de AST para más de 30 lenguajes. Permite buscar símbolos, rutas de llamadas y blast radius en una sola consulta.

[codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp) (`codebase-memory-mcp`):
Ofrece consultas de grafo estructural profundo y trazado de llamadas para bases de código grandes.

La función `InstallMCPServers` lee el archivo `mcp.json` existente y solo agrega o actualiza estas tres entradas. No borra ni altera otros servidores que el usuario ya tenga configurados.

## Google AI Pro y monitor de consumo

### Plugin pi-antigravity

Para conectar Pi con los modelos de Google AI Pro mediante Antigravity, el instalador registra el paquete `npm:pi-antigravity@0.12.0` en `settings.json` (junto con `npm:pi-cc-extensions@1.0.16`). Ambos van fijados a la versión del catálogo oficial de Pi: Pi no reinstala un paquete con versión exacta. Además, inyecta la dependencia `typebox: "*"` en la sección `peerDependencies` de la configuración para evitar avisos molestos de npm al resolver tipos.

### Monitor de cuotas

El archivo `pkg/installer/usage.go` lee la cuenta activa guardada en `~/.pi/agent/antigravity-accounts.json` y hace una llamada POST al endpoint `retrieveUserQuotaSummary`. Con la respuesta construye una barra de texto que muestra el porcentaje restante y el tiempo que falta para que se reinicien las cuotas semanal y de cinco horas.

## Interfaz visual y extensiones

La interfaz combina dos componentes:

pi-cc-extensions:
Configura el formato visual de los diffs de edición y escritura en Pi con barras laterales y prefijos limpios. También habilita el comando `/context` para revisar el peso de los mensajes en la ventana de contexto.

gomemory-status.ts:
Extensión en TypeScript que añade al pie de la terminal el conteo de tokens de entrada, salida, lecturas de caché y coste acumulado. Se puede activar o desactivar con el comando `/tokens`.

gomemory-tasks.ts:
Extensión en TypeScript que muestra sobre el editor de Pi las tareas de la unidad HCD activa con su estado de verificación (`✓`, `○`, `◌`, `✗`), actualizándose reactivamente tras cada comando `mem hcd`. Se activa o desactiva con `/tasks`.

gomemory-sidebar.ts:
Extensión en TypeScript inspirada en OpenCode que renderiza un overlay lateral derecho (`anchor: "top-right"`, 36 cols, responsive) con la sesión/HCD activo, métricas de contexto en vivo (tokens, porcentaje y coste) y conectividad de servidores MCP. Se activa o desactiva con `/sidebar`.

## Sanitización de skills

Pi utiliza un analizador estricto de YAML para leer el bloque inicial de metadatos (`---`) de cada archivo `SKILL.md`. Si el campo `description:` contiene dos puntos sin comillas, el analizador falla y la skill no se carga.

La función `SanitizeFrontmatter` procesa el contenido antes de copiarlo a `~/.pi-gomemory/skills/`. Si detecta dos puntos dentro del texto de la descripción, escapa las comillas existentes y envuelve todo el valor entre comillas dobles.

## Lanzador para Herdr

El script `pi-gomemory` en `~/.local/bin/` prepara el entorno antes de invocar el binario de Pi:

* Establece `HERDR_ENV=1` y `COLORTERM=truecolor`.
* Prioriza `~/.local/bin` y `~/.gomemory/bin` en el `PATH`.
* Si el primer argumento es `usage`, delega en `pi-gomemory-setup usage`.
* Comprueba si el directorio actual tiene una base de datos de gomemory (`.gomemory/memories.db`). Si existe, ejecuta `mem context` y pasa el resultado mediante `--append-system-prompt` para que el agente arranque con el historial reciente del proyecto.

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
  usage         Muestra cuotas y límites de consumo de Google AI Pro
  doctor        Verifica binarios, tema, MCP, plugins, launcher y otros agentes detectados
  mcp           Registra gomemory, codegraph y codebase-memory en mcp.json
  theme         Instala la paleta gomemory en ~/.pi/agent/themes/
  plugin        Configura pi-antigravity y pi-cc-extensions en settings.json
  skills        Reúne las skills de gomemory y del anfitrión en ~/.pi-gomemory/skills/
  skills-sync   Sincroniza el repositorio central de skills y purga copias legadas
  harness       Genera el launcher pi-gomemory en ~/.local/bin (o PI_GOMEMORY_BIN_DIR)
  version       Muestra la versión instalada
  deps-status   Lista los componentes desactualizados
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
Alterna la visibilidad de la barra lateral en la esquina superior derecha (`anchor: "top-right"`). Muestra la sesión/HCD activo, el consumo de ventana de contexto en tiempo real (tokens, porcentaje y coste) y los servidores MCP declarados en `mcp.json` con el estado `Configured` (no se comprueba la conexión en vivo; sin `mcp.json` válido la lista queda vacía). El panel flota sin capturar el teclado, así que puedes seguir escribiendo; el título incluye la versión instalada. Se oculta con `/sidebar` o automáticamente si el terminal mide menos de 100 columnas.

### Barra de tokens (`/tokens`)
Alterna la visibilidad de la barra inferior con el consumo acumulado de tokens de entrada, salida, lecturas de caché y coste estimado.

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


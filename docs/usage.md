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
  skills        Copia y sanitiza las 21 skills en ~/.pi/agent/skills/
  harness       Genera el script ejecutable ~/.local/bin/pi-gomemory
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

### Barra de tokens (`/tokens`)
Alterna la visibilidad de la barra inferior con el consumo acumulado de tokens de entrada, salida, lecturas de caché y coste estimado.

### Ventana de contexto (`/context`)
Muestra el desglose de espacio ocupado en la ventana de contexto del modelo para identificar qué archivos o mensajes consumen más memoria.

## Catálogo de skills instaladas

El instalador copia y sanitiza 21 skills en `~/.pi/agent/skills/`:

Harness Continual Development (HCD):
`hcd`, `hcd-define`, `hcd-clarify`, `hcd-approve`, `hcd-slice`, `hcd-tasks`, `hcd-implement`, `hcd-verify`, `hcd-analyze`, `hcd-close`.

Rigor y expansión:
`verify-loop`, `search-first`, `iterative-retrieval`, `strategic-compact`, `instincts`, `atomic-decomposition`.

Auditoría y control de calidad:
`adversarial-consensus-review`, `defensive-qa-audit`.

Memoria y asistencia técnica:
`codebase-memory`, `constitution`, `socratica`.

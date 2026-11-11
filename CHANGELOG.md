# Changelog

Todos los cambios notables de este proyecto se documentan en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/).

## [v1.6.0] - 2026-11-11

### Added
- Panel lateral de estado inspirado en OpenCode para Pi (`gomemory-sidebar.ts`): overlay interactivo en la esquina superior derecha (`anchor: "top-right"`, 36 columnas, responsive $\ge 100$ columnas) que consolida la sesión/HCD activo, métricas de contexto en tiempo real (tokens usados, porcentaje de ventana y gasto acumulado en USD) y lista de servidores MCP (`codebase-memory-mcp`, `codegraph`, `context7`, `gomemory`) con estado de conexión. Se activa y desactiva con el comando `/sidebar`.

## [v1.5.1] - 2026-11-11

### Added
- Registro de decisiones de arquitectura en `docs/adr/` (repositorio central de skills, extensiones TUI de HCD y orquestación de dependencias).

### Fixed
- `pi-gomemory-setup update`, `check-update` y la comprobación de dependencias ya no fallan con HTTP 403 cuando se agota el límite de la API de GitHub (60 consultas por hora y por IP). La última versión de pi-gomemory y de `codebase-memory-mcp` se lee de la redirección de `github.com/<repo>/releases/latest`, sin token.

## [v1.5.0] - 2026-11-11

### Added
- El banner de inicio muestra, junto a la versión de pi-gomemory, la de gomemory instalada (`pi - gomemory  v1.5.0 · gomemory v2.54.1`). El launcher la lee de `mem --version` y la exporta en `GOMEMORY_INSTALLED_VERSION`; si `mem` no está, se omite.

## [v1.4.1] - 2026-11-11

### Added
- Panel de tareas HCD en Pi (`gomemory-tasks.ts`): muestra sobre el editor las tareas de la unidad activa con su estado, leídas de `mem hcd show --json`, y se actualiza tras cada `mem hcd` y al terminar cada respuesta. `/tasks` lo muestra u oculta.

### Fixed
- `/plan <petición>` ya no se rechaza como argumento desconocido: activa el modo plan y envía la petición.

## [v1.4.0] - 2026-11-11

### Added
- Repositorio central de skills `~/.pi-gomemory/skills`: reúne las 21 skills de gomemory y enlaces a las de `~/.agents`, OpenCode, Claude Code, Codex y Gemini. Pi carga solo ese repositorio y se sincroniza en cada arranque (`pi-gomemory-setup skills-sync`).
- Comprobación de actualizaciones de pi-gomemory, `pi`, `mem`, `codegraph`, `codebase-memory-mcp`, `pi-antigravity` y `pi-cc-extensions` en cada arranque, en segundo plano, con pregunta de 10 s en terminal interactiva. Comandos `deps-refresh`, `deps-status` y `deps-update`.

### Changed
- `pi-antigravity` y `pi-cc-extensions` se instalan fijados a la versión del catálogo oficial de Pi (0.12.0 y 1.0.16).
- Las skills de gomemory adaptadas para Pi ganan a las copias de otros agentes; antes ganaba la de OpenCode.
- `check-update` deja de llamarse desde el launcher; lo sustituye `deps-status`.

### Fixed
- La instalación ya no borra skills propias de `~/.pi/agent/skills` con el mismo nombre que una del anfitrión.
- `PI_GOMEMORY_BIN_DIR` se aplica también al launcher, que encuentra `pi-gomemory-setup` en su propio directorio.
- `install.sh` reemplaza el binario con un renombrado atómico (evita `Killed: 9` en macOS) y la compilación local lleva la versión de git en lugar de `dev`.
- Las escrituras de `settings.json`, `mcp.json` y `pi-cc-extensions.json` son atómicas.
- Los fallos de `pi install` dejan de ignorarse en silencio.

### Security
- Los binarios de la release se compilan con Go 1.27.4, que incluye correcciones de seguridad para cgo, el comando go, el compilador, el enlazador y `net/http`.
- `build-release.sh` ejecuta `go vet` y `go test`, compila con `-trimpath` y se detiene si no puede generar `SHA256SUMS`.

## [v1.3.0] - 2026-11-10

### Added
- Banner de inicio con el logo de gomemory y la versión en ejecución (`gomemory-banner.ts`).
- Modo de planificación `/plan` (`gomemory-plan.ts`): bloquea las herramientas y comandos que modifican archivos mientras se planifica.
- Aviso de nuevas releases en el launcher (consulta como máximo cada 24 h, sin bloquear) y comandos `pi-gomemory-setup version`, `check-update` y `update`, este último con verificación de checksum.
- Pi carga las skills de OpenCode y Claude Code instaladas en el equipo.

### Changed
- El binario lleva su versión, inyectada al compilar con `-X main.version`.
- `doctor` cuenta todas las skills distintas que Pi tiene disponibles.

### Fixed
- Pi ya no avisa de colisiones de skills: se registra solo `~/.config/opencode/skills`, las skills que solo existen en `~/.claude/skills` se enlazan en `~/.pi/agent/skills` y se retiran las copias que ya ofrece el anfitrión.
- La cabecera de `pi-cc-extensions` ya no tapa el banner de gomemory (`showStartupHeader: false` por defecto).

## [v1.2.0] - 2026-11-10

### Added
- Herramienta `splash_review` en Pi: ejecuta al revisor en una sesión independiente, sin historial ni memoria del proyecto, con herramientas de lectura, cancelación y límite de cinco minutos.
- Definición neutral de Splash incluida en el paquete; no requiere archivos de Claude Code. El adaptador incluido corresponde a Pi.
- Diagnóstico de herramientas del portapapeles para macOS, X11/WSLg y Wayland, con indicaciones para instalar las dependencias ausentes.
- Pruebas de instalación de los recursos de Splash y del diagnóstico multiplataforma del portapapeles.

### Changed
- El instalador incluye la extensión y la definición de Splash junto con las instrucciones para ejecutarlo y registrar el resultado real.
- Documentación de Splash y de los requisitos del portapapeles. `xclip` continúa siendo una dependencia del sistema; no se incluye en los binarios.

### Fixed
- Pi dispone de una herramienta para ejecutar Splash y ya no necesita buscar su definición en la configuración de otro agente.
- Se declara Go 1.27.0 como mínimo y Go 1.27.4 como toolchain para evitar la descarga fallida de `go1.27` desde instalaciones anteriores de Go.

## [v1.1.0] - 2026-11-10

### Fixed
- `usage` ya no inventa cuotas cuando la API falla: devuelve un error y termina con código 1.
- `mcp.json`: se conservan todos los campos de los servidores ajenos (`url`, `headers`, `type`…). Un JSON inválido detiene la instalación sin sobrescribirlo. Mismo criterio para `settings.json` y `pi-cc-extensions.json`.
- `AGENTS.md`: el protocolo se escribe entre marcadores `<!-- pi-gomemory:start/end -->` y conserva el contenido propio. Las instalaciones v1.0.0 se migran sin duplicar.
- `install` y `harness` ya no crean `./bin/pi-gomemory` en el directorio actual.
- Se elimina el respaldo que leía y reescribía skills en `~/.config/opencode/skills`.
- El tema y la configuración de `pi-cc-extensions` se leen de `themes/` y `config/` embebidos, que antes no se usaban. En `pi-cc-extensions.json` solo se añaden las claves que falten.
- `install.sh` compila desde fuente solo si el directorio es este módulo, y verifica el tarball contra `SHA256SUMS`.

### Added
- `doctor` lista los otros agentes detectados (Claude Code, Codex, agy, OpenCode) y el comando de gomemory que los integra.
- Variable `PI_GOMEMORY_BASE_URL` en `install.sh` para probar contra un `dist/` local.
- Pruebas del instalador en un HOME aislado (`go test ./...`).

## [v1.0.0] - 2026-11-09

### Added
- Binario Go (`pi-gomemory-setup`) con recursos de configuración, temas y skills embebidos en el binario mediante `embed.FS`.
- Comando de instalación en un solo paso (`install`) que configura MCPs, tema, plugins, skills y lanzador.
- Integración de tres servidores MCP en `mcp.json`: `gomemory`, `codegraph` y `codebase-memory`.
- Configuración del plugin `npm:pi-antigravity` para modelos de Google AI Pro con parche de dependencias `typebox`.
- Monitor de cuotas `pi-gomemory usage` para consultar límites y tiempos de reinicio de Google AI Pro.
- Tema gomemory con paleta oscura en Truecolor de 24 bits.
- Extensión Claude Code TUI (`pi-cc-extensions`) con diffs visuales e inspección de ventana con `/context`.
- Extensión `gomemory-status.ts` para Pi con contadores de tokens y coste en el pie de terminal.
- 21 skills oficiales de gomemory con sanitización de frontmatter YAML para HCD (Harness Continual Development), revisión y auditoría.
- Lanzador `pi-gomemory` para Herdr con inyección de contexto de memoria en el arranque.
- Script instalador `install.sh` y script de empaquetado multiplataforma `scripts/build-release.sh`.
- Documentación técnica en `/docs` sobre arquitectura, instalación y comandos de uso.

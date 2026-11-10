# Changelog

Todos los cambios notables de este proyecto se documentan en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/).

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

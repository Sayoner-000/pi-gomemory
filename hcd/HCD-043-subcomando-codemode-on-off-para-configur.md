# Subcomando codemode on/off para configurar Pi y MCP en modo híbrido

ID: HCD-043
Tipo: feature
Carril: fast
Motivo: Feature acotada sobre CLI y configuración JSON de Pi sin decisiones de alto riesgo

## Intención
Permitir activar y desactivar el modo codemode híbrido en Pi desde pi-gomemory (settings.json + mcp.json), preservando el estado al reinstalar y documentando qué hace.


## Alcance
Incluido: Subcomando `pi-gomemory codemode <on|off|status>` en `main.go` y `pkg/installer`, soporte del campo `toolExposure` en `MCPServerDef` (`pkg/installer/mcp.go`), actualización segura e idempotente de `~/.pi/agent/settings.json` (`defaultTools: ["+codemode"]` sin duplicados ni pérdida de otras claves) y `~/.pi/agent/mcp.json` (`exposure: "codemode"` + `toolExposure` híbrido en `on`; restauración a `exposure: "direct"` sin `toolExposure` en `off`), preservación del modo codemode activo al ejecutar `pi-gomemory install` o `pi-gomemory mcp`, pruebas unitarias en `pkg/installer`, y nota explicativa en CLI (`codemode --help` / `status`) y `README.md`.
Excluido: Activar `codemode.mode: "only"` (ocultaría `read`, `edit`, `write` y `bash`), modificar `.pi/mcp.json` por proyecto, alterar el binario de `pi` o cambiar las extensiones TypeScript (`gomemory-plan.ts`, `gomemory-tasks.ts`, `gomemory-tdd.ts`).

## ¿Qué pasa si no lo hacemos?
Cada sesión de Pi sigue cargando las 54 definiciones MCP con `exposure: "direct"` (36 de `gomemory`, 17 de `codebase-memory`, 1 de `codegraph`) consumiendo tokens de esquema en cada turno, y cualquier usuario que active `codemode` manualmente en `~/.pi/agent/mcp.json` perderá su configuración en cuanto ejecute `pi-gomemory install` o `pi-gomemory mcp` porque `InstallMCPServers` sobrescribe las tres entradas con `exposure: "direct"`.

## Alternativa más barata
Crear `.pi/mcp.json` manualmente por proyecto y editar `~/.pi/agent/settings.json` con `jq`. Es válido para pruebas puntuales en un solo repositorio, pero obliga a repetir la configuración en cada proyecto, no resuelve el borrado en `~/.pi/agent/mcp.json` tras actualizar `pi-gomemory`, y no ofrece un interruptor reversible (`on`/`off`) ni documentación integrada.

## Normas aplicables
- Constitución: §1.2 (Go 1.27, biblioteca estándar primero, errores con `%w`, `gofmt`), §10 (pruebas y cobertura), §18 (artefacto HCD único), §20 (documentación técnica en español), §21 (simplicidad e idempotencia).
- Playbook HCD: `HCD-DEF-01`, `HCD-TDD-01`, `HCD-EVIDENCE-01`.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Añadir campo ToolExposure a MCPServerDef y construir perfiles direct e híbrido en mcp.go → go test ./pkg/installer -run TestCodemodeMCP
- [x] 2. Implementar activación y desactivación de +codemode en settings.json sin alterar claves existentes → go test ./pkg/installer -run TestCodemodeSettings
- [x] 3. Preservar el modo codemode activo cuando InstallMCPServers regenera mcp.json → go test ./pkg/installer -run TestInstallMCPServersPreservesCodemode
- [x] 4. Exponer el subcomando codemode on/off/status con nota explicativa en main.go y printHelp → go test . -run TestCodemodeCLI
- [x] 5. Documentar el modo codemode híbrido y qué hace en README.md → grep -q "codemode" README.md && go test ./...

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-17 10:27 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestCodemodeMCP` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/codemode_test.go:13:28: direct["gomemory"].ToolExposure undefined (type MCPServerDef has no field or method ToolExposure)
      pkg/installer/codemode_test.go:14:98: direct["gomemory"].ToolExposure undefined (type MCPServerDef has no field or method ToolExposure)
      pkg/installer/codemode_test.go:17:12: undefined: CodemodeMCPServers
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 10:27 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestCodemodeMCP` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.626s
- 2026-11-17 10:27 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestCodemodeSettings` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/codemode_test.go:77:12: undefined: SetCodemode
      pkg/installer/codemode_test.go:81:12: undefined: SetCodemode
      pkg/installer/codemode_test.go:101:13: undefined: GetCodemodeStatus
      pkg/installer/codemode_test.go:109:12: undefined: SetCodemode
      pkg/installer/codemode_test.go:127:12: undefined: SetCodemode
      pkg/installer/codemode_test.go:130:12: undefined: SetCodemode
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 10:28 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestCodemodeSettings` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.239s
- 2026-11-17 10:28 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestInstallMCPServersPreservesCodemode` (salida 1)
      --- FAIL: TestInstallMCPServersPreservesCodemode (0.02s)
          codemode_test.go:171: InstallMCPServers reverted codemode: {Enabled:false SettingsEnabled:true MCPEnabled:false Partial:true}
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.218s
      FAIL
- 2026-11-17 10:28 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestInstallMCPServersPreservesCodemode` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.217s
- 2026-11-17 10:28 -05:00 · tarea 4 · rojo observado · caducada · `go test . -run TestCodemodeCLI` (salida 1)
      # pi-gomemory [pi-gomemory.test]
      ./main_codemode_test.go:15:12: undefined: runCodemode
      ./main_codemode_test.go:31:12: undefined: runCodemode
      ./main_codemode_test.go:39:12: undefined: runCodemode
      ./main_codemode_test.go:50:12: undefined: runCodemode
      FAIL	pi-gomemory [build failed]
      FAIL
- 2026-11-17 10:28 -05:00 · tarea 4 · verde · caducada · `go test . -run TestCodemodeCLI` (salida 0)
      ok  	pi-gomemory	0.220s
- 2026-11-17 10:29 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestCodemodeMCP` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.550s
- 2026-11-17 10:29 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestCodemodeSettings` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.198s
- 2026-11-17 10:29 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestInstallMCPServersPreservesCodemode` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-17 10:29 -05:00 · tarea 4 · verde · vigente · `go test . -run TestCodemodeCLI` (salida 0)
      ok  	pi-gomemory	(cached)
- 2026-11-17 10:29 -05:00 · tarea 5 · verde · vigente · `grep -q "codemode" README.md && go test ./...` (salida 0)
      ok  	pi-gomemory	0.180s
      ok  	pi-gomemory/pkg/installer	70.336s

## Siguiente
Cerrada con aprobación sobre la revisión 27ad84750823.

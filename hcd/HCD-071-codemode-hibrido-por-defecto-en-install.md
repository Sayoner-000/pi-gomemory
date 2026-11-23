# Codemode híbrido por defecto en install y update, respetando codemode off

ID: HCD-071
Tipo: evolution
Carril: fast
Motivo: cambio acotado de valor por defecto con decisión tomada por la persona; revierte HCD-050 sin decisiones abiertas

## Intención
Que install y update dejen Pi en codemode híbrido por defecto (tools de protocolo directas, el resto por codemode), respeten un codemode off explícito y que el protocolo de Pi indique buscar por codemode una tool antes de darla por inexistente


## Contexto y evidencia
- Decisión de la persona (2026-11-22): «abre una unidad y dejamos el modo hibrido por defecto», tras la recomendación con tres condiciones (respetar `codemode off`, que el protocolo lleve a codemode antes de dar una tool por inexistente y valorar `hcd_history` directa).
- Medición real en Pi 1.6.0 (v1.23.2, HCD-070): con el perfil híbrido el modelo recibe 27 tools declaradas frente a unas 80 en directo con `cbmem.ts`, sin duplicados.
- Antecedentes: HCD-046 (v1.15.0) puso el híbrido por defecto; HCD-050 (v1.15.4) volvió a directo por preferencia de la persona. Esta unidad revierte HCD-050 y conserva sus garantías: `codemode off` persiste en `~/.pi-gomemory/codemode.json`, rollback de `settings.json` y `mcp.json` (HCD-047 C-01), propiedad de `+codemode` ligada al destino (C-08) y composición con lite (HCD-052).
- Impacto (codegraph + codebase-memory): `applyCodemodeDefault` → `applyCodemodeMode` (único llamador `InstallAll`, que usan `install` y `update`); `enableCodemodeWithLite`, `reconcileMCPServers`, `resolveMCPProfile`; `codemodeNote`/`runCodemode` en `main.go`; mensaje del paso [2/9] en `installer.go`; pruebas `hcd046_test.go`, `hcd047_regression_test.go`, `hcd052_test.go`, `hcd070_test.go`, `main_codemode_test.go`. `CodemodeMCPServers` tiene 7 llamadores (uninstall y perfiles gestionados) y no se modifica.
- `hcd_history` no pasa a directa: añadirla cambiaría el perfil híbrido publicado, `isManagedMCPServer` dejaría de reconocer los `mcp.json` ya escritos (se tratarían como personalizados) y habría que embeber otro perfil histórico. Las skills HCD usan `mem hcd history` por CLI, así que el beneficio no compensa.
- Las skills (`skills/socratica`, `skills/adversarial-consensus-review`) son copias de gomemory: no se editan aquí. La indicación va en el bloque de protocolo de `extensions/gomemory-context.ts`, que se inyecta en cada sesión con codemode activo.

## Alcance
Incluido: `install`/`update` aplican el perfil híbrido cuando no hay preferencia o es `on`, y el directo solo con `codemode off`; registran la propiedad de `+codemode` si lo añaden; mensajes de CLI (`installer.go`, `codemodeNote`, `runCodemode`); línea de protocolo en `extensions/gomemory-context.ts`; pruebas y documentación (README, `docs/usage.md`, `docs/installation.md`, `docs/architecture.md` y `docs/known-issues.md` si citan el valor por defecto).
- `pkg/installer/codemode.go`: `applyCodemodeMode`.
- `pkg/installer/installer.go` y `main.go`: textos.
- `extensions/gomemory-context.ts`: buscar por codemode antes de dar una tool por inexistente.
Excluido: cambiar `CodemodeMCPServers` o la lista de tools directas (incluida `hcd_history`); editar skills copiadas de gomemory; cambiar el subcomando `mcp` (`InstallMCPServersForOptions` sigue conservando el modo activo, porque sin `+codemode` el híbrido dejaría tools inalcanzables); retirar `+codemode` o tocar `cbmem.ts` ajenos.

## ¿Qué pasa si no lo hacemos?
Cada instalación nueva arranca con todas las tools MCP directas: unas 50 más por turno de las necesarias. Quien quiere el híbrido tiene que conocer `codemode on`.

## Alternativa más barata
Documentar `codemode on` como paso recomendado tras instalar. No cuesta código, pero la mayoría se queda con el valor por defecto y el ahorro no llega. Se descarta.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. install sin preferencia deja codemode híbrido y registra la propiedad de +codemode en pkg/installer/codemode.go → go test ./pkg/installer -run TestHCD071DefaultHybrid -count=1
- [x] 2. install respeta codemode off (directo, sin +codemode) y avisa del parcial si hay +codemode ajeno en pkg/installer/codemode.go → go test ./pkg/installer -run TestHCD071RespectsOff -count=1
- [x] 3. install sin preferencia con modo lite deja gomemory híbrido y sin servidores de grafo en pkg/installer/codemode.go → go test ./pkg/installer -run TestHCD071Lite -count=1
- [x] 4. el protocolo de extensions/gomemory-context.ts pide buscar por codemode antes de dar una tool por inexistente → go test . -run TestHCD071ContextProtocol -count=1
- [x] 5. los mensajes de main.go y pkg/installer/installer.go dicen que el híbrido es el valor por defecto → go test . -run TestHCD071CLIText -count=1
- [x] 6. regresión completa de pkg/installer y main → go test ./... -count=1

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-22 20:17 -05:00 · tarea 1,2,3 · rojo observado · caducada · `go test ./pkg/installer -run "TestHCD071" -count=1` (salida 1)
      --- FAIL: TestHCD071DefaultHybrid (0.03s)
          --- FAIL: TestHCD071DefaultHybrid/instalación_nueva (0.01s)
              hcd071_test.go:22: install debe dejar codemode híbrido por defecto: {Enabled:false SettingsEnabled:false MCPEnabled:false Partial:false}
          --- FAIL: TestHCD071DefaultHybrid/instalación_previa_en_directo_sin_preferencia_pasa_a_híbrido (0.01s)
              hcd071_test.go:41: update no pasó a híbrido: {Enabled:false SettingsEnabled:false MCPEnabled:false Partial:false}
          --- FAIL: TestHCD071DefaultHybrid/+codemode_ajeno:_híbrido_sin_atribuirse_la_propiedad (0.01s)
              hcd071_test.go:52: debía quedar híbrido completo: {Enabled:false SettingsEnabled:true MCPEnabled:false Partial:true}
          --- FAIL: TestHCD071DefaultHybrid/es_idempotente (0.01s)
              hcd071_test.go:68: open /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD071DefaultHybrides_idempotente2132567043/001/.pi/agent/settings.json: no such file or directory
      --- FAIL: TestHCD071Lite (0.00s)
          hcd071_test.go:129: gomemory no quedó híbrido con lite: {Command:mem Args:[mcp] Env:map[] Exposure:direct ToolExposure:map[] Description:Gomemory persistent project memory, semantic graphs, checkpoints, HCD and review protocol}
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.337s
      FAIL
- 2026-11-22 20:18 -05:00 · tarea 4,5 · rojo observado · caducada · `go test . -run "TestHCD071" -count=1` (salida 1)
      --- FAIL: TestHCD071ContextProtocol (0.00s)
          main_hcd071_test.go:20: falta "antes de concluir que una tool no existe" en el protocolo de gomemory-context.ts
      --- FAIL: TestHCD071CLIText (0.00s)
          main_hcd071_test.go:29: codemodeNote no describe el nuevo valor por defecto:
              Nota: el modo por defecto de install y update es directo (codemode off, con gomemory, codegraph y codebase-memory en exposure "direct"); `codemode on` activa el modo codemode híbrido de forma persistente. En modo codemode híbrido, Pi habilita +codemode (sandbox QuickJS para paralelizar llamadas y filtrar salidas >20 KB antes de entrar al contexto) y pasa gomemory y codebase-memory a exposure "codemode", manteniendo direct las herramientas de protocolo y consulta diaria (end_session, get_context, get_memory, get_plan_context, get_symbol, judge_memories, octopus_route_task, save_memory, save_session_summary, search_code, search_memories, start_session, check_index_coverage, get_code_snippet, search_graph, trace_path). El resto sigue disponible a través de la tool codemode.
      FAIL
      FAIL	pi-gomemory	0.173s
      FAIL
- 2026-11-22 20:21 -05:00 · tarea 6 · verde · vigente · `go test ./... -count=1` (salida 0)
      ok  	pi-gomemory	0.510s
      ok  	pi-gomemory/pkg/installer	44.045s

## Siguiente
Cerrada con aprobación sobre la revisión 1e5db5390851.

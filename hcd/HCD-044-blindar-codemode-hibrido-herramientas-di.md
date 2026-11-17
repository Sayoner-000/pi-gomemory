# Blindar codemode híbrido: herramientas directas coherentes con el protocolo

ID: HCD-044
Tipo: fix
Carril: fast
Motivo: fix con reproducción clara (memoria 435), sin decisiones abiertas

## Intención
Con codemode on, el protocolo instalado en `~/.pi/agent/AGENTS.md` promete search_code/get_symbol como directas pero no lo son, falta start_session y no se explica cómo llegar al resto por codemode; alinear lista directa, protocolo y tests para que no se pierdan las ventajas de gomemory

## Reproducción
En un HOME aislado, `codemode on` no deja directas las herramientas que el protocolo de Pi promete:

```
H=$(mktemp -d); HOME=$H go run . codemode on
jq -e '.mcpServers.gomemory.toolExposure | (.search_code=="direct" and .get_symbol=="direct" and .start_session=="direct")' $H/.pi/agent/mcp.json
→ false, exit=1
```

Origen: memoria 435. `GomemoryPiInstructions` (`pkg/installer/instructions.go:16`) declara `search_code` y `get_symbol` como «tools MCP directas», pero `CodemodeMCPServers` (`pkg/installer/mcp.go:52`) no las incluye. Falta también `start_session` (inicio del protocolo), y el protocolo no explica que el resto de herramientas se llama a través de `codemode`. Antecedente: HCD-043 (memoria 423) fijó la lista original y no contrastó el protocolo instalado.

## Alcance
Incluido: añadir `search_code`, `get_symbol`, `start_session`, `octopus_route_task` y `judge_memories` como `direct` en el perfil híbrido de gomemory; añadir al protocolo (`GomemoryPiInstructions`, instalado en `~/.pi/agent/AGENTS.md`) una línea sobre cómo llamar al resto de herramientas con `codemode`; un test de coherencia que falle si el protocolo nombra como directa una herramienta que el perfil híbrido no expone en directo; actualizar la nota del CLI `codemode on` (`main.go`) y el README con la lista nueva.
Excluido: cambiar la exposición de codebase-memory o codegraph; tocar `codemode off`, `uninstall` o `status`; guías HTML de versiones anteriores y CHANGELOG (se hacen en la release).

## ¿Qué pasa si no lo hacemos?
Con codemode activado, el agente de Pi intenta llamar en directo a `search_code` y `get_symbol` porque el protocolo se lo indica, falla y acaba usando `grep`/`find` por consola, justo lo que las reglas 7–9 prohíben. No llama a `start_session` y no encuentra `octopus_route_task` (obligatoria antes de delegar) ni `judge_memories`. Se pierden las ventajas de gomemory que el modo híbrido debía preservar, y nada impide que el desajuste vuelva a aparecer.

## Alternativa más barata
Solo reescribir el protocolo para que diga que `search_code` y `get_symbol` van por `codemode`. Es más barato, pero deja la búsqueda de código del grafo propio (de uso diario) detrás de un sandbox y no protege contra futuros desajustes. Se descarta: añadir cinco herramientas directas cuesta poco contexto (17 directas frente a 54 sin codemode).


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla: en pkg/installer/mcp.go, codemode on deja directas search_code, get_symbol, start_session, octopus_route_task y judge_memories → `bash -c 'H=$(mktemp -d); HOME=$H go run . codemode on >/dev/null && jq -e ".mcpServers.gomemory.toolExposure | (.search_code==\"direct\" and .get_symbol==\"direct\" and .start_session==\"direct\" and .octopus_route_task==\"direct\" and .judge_memories==\"direct\")" $H/.pi/agent/mcp.json'`
- [x] 2. Test de coherencia: toda tool que el protocolo declara directa lo es en el perfil híbrido, y el protocolo explica codemode → `go test ./pkg/installer -run 'TestCodemodeMCP|TestCodemodeProtocolCoherence' -count=1`
- [x] 3. La nota del CLI codemode on y el README listan las herramientas directas nuevas → `bash -c 'grep -q "octopus_route_task" README.md && H=$(mktemp -d); HOME=$H go run . codemode on | grep -q "octopus_route_task"'`
- [x] 4. Sin regresiones: vet y suite completa en verde → `bash -c 'go vet ./... && go test ./... -count=1'`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-17 13:52 -05:00 · tarea 1 · rojo observado · caducada · `bash -c 'H=$(mktemp -d); HOME=$H go run . codemode on >/dev/null && jq -e ".mcpServers.gomemory.toolExposure | (.search_code==\"direct\" and .get_symbol==\"direct\" and .start_session==\"direct\" and .octopus_route_task==\"direct\" and .judge_memories==\"direct\")" $H/.pi/agent/mcp.json'` (salida 1)
      false
- 2026-11-17 13:52 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run 'TestCodemodeMCP|TestCodemodeProtocolCoherence' -count=1` (salida 1)
      --- FAIL: TestCodemodeProtocolCoherence (0.00s)
          codemode_test.go:46: gomemory: el protocolo declara directas [get_context get_symbol save_memory search_code search_memories], el perfil híbrido expone [end_session get_context get_memory get_plan_context save_memory save_session_summary search_memories]
          codemode_test.go:51: gomemory ToolExposure["search_code"] = "", want direct
          codemode_test.go:51: gomemory ToolExposure["get_symbol"] = "", want direct
          codemode_test.go:51: gomemory ToolExposure["start_session"] = "", want direct
          codemode_test.go:51: gomemory ToolExposure["octopus_route_task"] = "", want direct
          codemode_test.go:51: gomemory ToolExposure["judge_memories"] = "", want direct
          codemode_test.go:55: el protocolo no explica cómo llamar por codemode al resto de tools
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.217s
      FAIL
- 2026-11-17 13:52 -05:00 · tarea 1 · verde · caducada · `bash -c 'H=$(mktemp -d); HOME=$H go run . codemode on >/dev/null && jq -e ".mcpServers.gomemory.toolExposure | (.search_code==\"direct\" and .get_symbol==\"direct\" and .start_session==\"direct\" and .octopus_route_task==\"direct\" and .judge_memories==\"direct\")" $H/.pi/agent/mcp.json'` (salida 0)
      true
- 2026-11-17 13:52 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run 'TestCodemodeMCP|TestCodemodeProtocolCoherence' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.186s
- 2026-11-17 13:53 -05:00 · tarea 3 · rojo observado · caducada · `bash -c 'grep -q "octopus_route_task" README.md && H=$(mktemp -d); HOME=$H go run . codemode on | grep -q "octopus_route_task"'` (salida 1)
      build cache is required, but could not be located: GOCACHE is not defined and $HOME is not defined
- 2026-11-17 13:53 -05:00 · tarea 3 · verde · caducada · `bash -c 'grep -q "octopus_route_task" README.md && H=$(mktemp -d); HOME=$H go run . codemode on | grep -q "octopus_route_task"'` (salida 0)
- 2026-11-17 13:53 -05:00 · tarea 4 · verde · caducada · `bash -c 'go vet ./... && go test ./... -count=1'` (salida 0)
      ok  	pi-gomemory	0.226s
      ok  	pi-gomemory/pkg/installer	69.861s
- 2026-11-17 13:54 -05:00 · tarea 1 · verde · caducada · `bash -c 'H=$(mktemp -d); HOME=$H go run . codemode on >/dev/null && jq -e ".mcpServers.gomemory.toolExposure | (.search_code==\"direct\" and .get_symbol==\"direct\" and .start_session==\"direct\" and .octopus_route_task==\"direct\" and .judge_memories==\"direct\")" $H/.pi/agent/mcp.json'` (salida 0)
      true
- 2026-11-17 13:54 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run 'TestCodemodeMCP|TestCodemodeProtocolCoherence' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.218s
- 2026-11-17 13:55 -05:00 · tarea 1 · verde · vigente · `bash -c 'H=$(mktemp -d); HOME=$H go run . codemode on >/dev/null && jq -e ".mcpServers.gomemory.toolExposure | (.search_code==\"direct\" and .get_symbol==\"direct\" and .start_session==\"direct\" and .octopus_route_task==\"direct\" and .judge_memories==\"direct\")" $H/.pi/agent/mcp.json'` (salida 0)
      true
- 2026-11-17 13:55 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run 'TestCodemodeMCP|TestCodemodeProtocolCoherence' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.170s
- 2026-11-17 13:55 -05:00 · tarea 3 · verde · vigente · `bash -c 'grep -q "octopus_route_task" README.md && H=$(mktemp -d); HOME=$H go run . codemode on | grep -q "octopus_route_task"'` (salida 0)
- 2026-11-17 13:55 -05:00 · tarea 4 · verde · vigente · `bash -c 'go vet ./... && go test ./... -count=1'` (salida 0)
      ok  	pi-gomemory	0.204s
      ok  	pi-gomemory/pkg/installer	68.939s

## Siguiente
Cerrada con aprobación sobre la revisión 394dbcd8e916.

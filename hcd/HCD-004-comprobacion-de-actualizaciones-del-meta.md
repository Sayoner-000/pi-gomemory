# Comprobación de actualizaciones del meta-harness y dependencias al iniciar

ID: HCD-004
Tipo: feature
Carril: fast
Motivo: decisiones resueltas por la persona: avisar y preguntar, consulta en segundo plano en cada arranque, todas las dependencias

## Intención
Al iniciar pi-gomemory, avisar de versiones nuevas de pi-gomemory, pi, mem, codegraph, codebase-memory-mcp, pi-antigravity y pi-cc-extensions y ofrecer actualizarlas

## Reproducción
Hoy el launcher solo ejecuta `pi-gomemory-setup check-update` (solo pi-gomemory, caché 24 h). codebase-memory-mcp 0.12.0 está instalado y existe v0.12.1 sin que nadie avise; lo mismo pasaría con pi, mem, codegraph y los paquetes npm.

## Alcance
Incluido: subcomandos `deps-refresh` (consulta en red y guarda caché `~/.pi/agent/pi-gomemory-deps.json`), `deps-status` (lee la caché y lista lo desactualizado, salida 10 si hay algo) y `deps-update` (ejecuta el comando oficial de actualización de cada componente desactualizado); el launcher lanza `deps-refresh` en segundo plano en cada arranque, muestra el aviso con el resultado anterior y, si hay TTY, pregunta [s/N] con timeout de 10 s; `PI_GOMEMORY_NO_UPDATE_CHECK` lo desactiva.
Excluido: actualizar sin preguntar; fijar versiones npm; firmas de releases.

## ¿Qué pasa si no lo hacemos?
Cada usuario queda con dependencias distintas y desactualizadas sin saberlo; los fallos de compatibilidad entre pi, mem y los MCP aparecen sin pista.

## Alternativa más barata
Documentar los comandos de actualización de cada herramienta en el README; depende de que cada persona lo recuerde.

## Aclaraciones
- [x] Q1. ¿Qué hacer al detectar una versión nueva? → Avisar y preguntar
- [x] Q2. ¿Frecuencia? → Cada arranque, en segundo plano
- [x] Q3. ¿Qué dependencias? → pi, mem, codegraph, codebase-memory-mcp, pi-antigravity, pi-cc-extensions (además de pi-gomemory)

## Checklist
- [x] 1. RefreshDeps consulta versión actual y última de cada componente y guarda la caché, tolerando fallos de red por componente → `go test ./pkg/installer -run TestRefreshDeps`
- [x] 2. DepsStatus lista solo los desactualizados con su comando de actualización → `go test ./pkg/installer -run TestDepsStatus`
- [x] 3. El launcher refresca en segundo plano, avisa, pregunta con timeout y respeta PI_GOMEMORY_NO_UPDATE_CHECK → `go test ./pkg/installer -run TestHarnessDepsCheck`
- [x] 4. Suite completa → `go test ./... && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 04:37 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestHarnessDepsCheck` (salida 1)
      --- FAIL: TestHarnessDepsCheck (0.00s)
          hcd004_test.go:67: el launcher no contiene "pi-gomemory-setup deps-refresh"
          hcd004_test.go:67: el launcher no contiene "pi-gomemory-setup deps-status"
          hcd004_test.go:67: el launcher no contiene "pi-gomemory-setup deps-update"
          hcd004_test.go:67: el launcher no contiene "read -r -t 10"
          hcd004_test.go:67: el launcher no contiene "[ -t 0 ]"
          hcd004_test.go:71: el launcher conserva la comprobación antigua de solo pi-gomemory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.436s
      FAIL
- 2026-11-11 04:39 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestRefreshDeps` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.189s
- 2026-11-11 04:39 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestDepsStatus` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.166s
- 2026-11-11 04:39 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHarnessDepsCheck` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.153s
- 2026-11-11 04:39 -05:00 · tarea 4 · verde · vigente · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	0.618s
- 2026-11-11 04:39 -05:00 · tarea 4 · verde · caducada · `GOTOOLCHAIN=local go test ./... && GOTOOLCHAIN=local go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 04:39 -05:00 · tarea 1,2,3 · verde · caducada · `GOTOOLCHAIN=local go test ./pkg/installer -run "TestRefreshDeps|TestDepsStatus|TestHarnessDepsCheck"` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.178s
- 2026-11-11 04:49 -05:00 · tarea 4 · verde · vigente · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	0.600s

## Siguiente
Cerrada con aprobación sobre la revisión 9f3704530e8f.

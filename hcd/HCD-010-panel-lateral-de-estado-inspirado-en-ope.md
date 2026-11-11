# Panel lateral de estado inspirado en OpenCode para Pi

ID: HCD-010
Tipo: feature
Carril: fast
Motivo: feature visual acotada sobre extensiones TUI de Pi sin decisiones abiertas interdependientes

## Intención
crear la extensión gomemory-sidebar con overlay superior derecho que muestre sesión, métricas de contexto y estado de servidores MCP, e integrarla en el instalador

## Alcance
Incluido: extensión `extensions/gomemory-sidebar.ts` con componente TUI en overlay superior derecho (`anchor: "top-right"`, `width: 36`, auto-hide si el ancho de la terminal es < 100 columnas). Muestra título de sesión o unidad HCD activa, sección `Context` (tokens usados, % de ventana y gasto acumulado) y sección `MCP` con estado de servidores (`Connected` / `Disconnected`). Comando `/sidebar` para mostrar/ocultar. Inclusión en `embed.go`, en `pkg/installer/extension.go` (`InstallTUIExtension`) y tests unitarios en `pkg/installer/hcd010_test.go`.
Excluido: modificar librerías core de Pi; alterar las extensiones existentes (`extensions/gomemory-tasks.ts`, `extensions/gomemory-plan.ts`, `extensions/gomemory-status.ts`); reemplazar el multiplexor Herdr.

## ¿Qué pasa si no lo hacemos?
Pi carece de un panel lateral consolidado que informe de manera simultánea el consumo de la ventana de contexto y la conectividad de los servidores MCP sin tener que ejecutar comandos interactivos continuos en el chat o depender solo de la barra de estado inferior.

## Alternativa más barata
Utilizar únicamente la extensión en la línea de estado inferior (`extensions/gomemory-status.ts`) y el widget sobre el editor (`extensions/gomemory-tasks.ts`), renunciando a la visualización lateral persistente y al panel unificado de servidores MCP de OpenCode.

## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->

## Checklist
- [x] 1. La extensión gomemory-sidebar.ts implementa el componente overlay lateral derecho con título/HCD, métricas de Context y estado de servidores MCP → `go test ./pkg/installer -run TestSidebarExtension$`
- [x] 2. El instalador copia gomemory-sidebar.ts desde el embed hacia extensions/ en Pi → `go test ./pkg/installer -run TestSidebarExtensionInstalled$`
- [x] 3. Suite completa y verificación de compilación → `go test ./... && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 05:34 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestSidebarExtension$` (salida 1)
      --- FAIL: TestSidebarExtension (0.00s)
          hcd010_test.go:11: open ../../extensions/gomemory-sidebar.ts: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.172s
      FAIL
- 2026-11-11 05:34 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestSidebarExtensionInstalled$` (salida 1)
      --- FAIL: TestSidebarExtensionInstalled (0.00s)
          hcd010_test.go:30: gomemory-sidebar.ts no se instala
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.166s
      FAIL
- 2026-11-11 05:36 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestSidebarExtension$` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 05:36 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestSidebarExtensionInstalled$` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 05:36 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestSidebarExtension$` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.164s
- 2026-11-11 05:36 -05:00 · tarea 3 · verde · caducada · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 05:37 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestSidebarExtension$` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 05:37 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestSidebarExtensionInstalled$` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 05:37 -05:00 · tarea 3 · verde · vigente · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	(cached)

## Siguiente
Cerrada con aprobación sobre la revisión 8668b456d559.

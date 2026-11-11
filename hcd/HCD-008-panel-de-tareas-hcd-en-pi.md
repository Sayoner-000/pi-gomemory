# Panel de tareas HCD en Pi

ID: HCD-008
Tipo: feature
Carril: fast
Motivo: propuesta aceptada por la persona; sin decisiones abiertas

## Intención
Mostrar en Pi las tareas de la unidad HCD activa con su estado, actualizado al verificar, inspirado en pi-goal-x pero usando el modelo de tareas de gomemory

## Reproducción
Hoy, para ver qué tareas de la unidad HCD están hechas hay que ejecutar `mem hcd show` a mano. Pi no muestra el progreso mientras el agente trabaja. pi-goal-x (pi.dev/packages/pi-goal-x, v0.34.1) muestra un árbol de tareas, pero mantiene su propia lista, que no se sincronizaría con HCD.

## Alcance
Incluido: extensión `extensions/gomemory-tasks.ts`, instalada desde el embed. Lee `mem hcd show --json` (timeout de 5 s, en el cwd de la sesión) y pinta un widget sobre el editor con la cabecera `HCD-NNN · tipo · paso` y una línea por tarea: `✓` verificada, `○` pendiente, `✗` con fallo. Se refresca al iniciar la sesión, después de cada comando `bash` que ejecute `mem hcd` y al terminar cada turno del agente. El comando `/tasks` muestra u oculta el panel. Si no hay unidad activa, `mem` falta o la salida no es válida, no se muestra nada.
Excluido: instalar pi-goal-x; crear o modificar tareas desde el panel (el motor HCD sigue siendo la única fuente); un auditor propio.

## ¿Qué pasa si no lo hacemos?
El avance de cada tarea solo se ve si se consulta a mano, y la persona no sabe qué falta mientras el agente trabaja.

## Alternativa más barata
Instalar pi-goal-x: muestra el progreso, pero con una segunda lista de tareas que no conoce HCD.

## Aclaraciones
- [x] Q1. ¿Adoptar pi-goal-x o usar el modelo de gomemory? → modelo de tareas de gomemory, con la presentación inspirada en pi-goal-x

## Checklist
- [x] 1. La extensión lee mem hcd show --json, pinta el widget con ✓/○/✗ y se refresca en sesión, tras mem hcd y al terminar el turno → `go test ./pkg/installer -run TestTasksExtension`
- [x] 2. El instalador copia la extensión desde el embed → `go test ./pkg/installer -run TestTasksExtensionInstalled`
- [x] 3. Suite completa → `go test ./... && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 05:13 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestTasksExtension$` (salida 1)
      --- FAIL: TestTasksExtension (0.00s)
          hcd008_test.go:11: open ../../extensions/gomemory-tasks.ts: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.347s
      FAIL
- 2026-11-11 05:13 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestTasksExtensionInstalled` (salida 1)
      --- FAIL: TestTasksExtensionInstalled (0.00s)
          hcd008_test.go:29: gomemory-tasks.ts no se instala
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.168s
      FAIL
- 2026-11-11 05:14 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestTasksExtension$` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.170s
- 2026-11-11 05:14 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestTasksExtensionInstalled` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.192s
- 2026-11-11 05:14 -05:00 · tarea 3 · verde · vigente · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	0.632s
- 2026-11-11 05:14 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestTasksExtension$` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.158s

## Siguiente
Cerrada con aprobación sobre la revisión b713d8883b2e.

# Cierre de hallazgos C-003..C-006 de la auditoría ACR/ADM v1.9.0

ID: HCD-021
Tipo: fix
Carril: fast
Motivo: hallazgos con reproducción clara y propuesta de cierre; sin decisiones abiertas

## Intención
El modo watch de /tdd no encola ejecuciones sin límite, el sidebar siempre muestra el control [−] aunque el título sea largo, fetchQuota rechaza respuestas de más de 1 MiB y queda demostrado que /tdd funciona en repositorios Go externos

## Reproducción
Fuente: informe docs/audits/2026-11-11-acr-adm-v1.9.0-audit.md (auditado sobre a97409b, antes de HCD-020), contrastado con el código actual:
- C-004: `renderSidebar` con un título de unidad largo (`HCD · HCD-020 · evolution · paso close · v1.9.0`, más de 34 columnas) recorta la fila por la derecha y desaparece el `[−]` clicable.
- C-005: `fetchQuota` (pkg/installer/usage.go) lee el cuerpo con `io.ReadAll` sin límite; una respuesta enorme se carga entera en memoria.
- C-003: HCD-020 ya encadena las ejecuciones (`runChain`), así que no corren a la vez. Pero cada `turn_end` en modo watch encola una ejecución completa: con N turnos durante una ejecución larga quedan N ejecuciones en cola.
- C-006: HCD-020 ya sustituyó las rutas fijas por `detectSuites` (`go test -json ./...`); falta demostrarlo en un repositorio Go externo.
Comprobación automatizada: `go test ./pkg/installer -run HCD021 -count=1`.

## Alcance
Incluido: pre-recorte del título en `renderSidebar` reservando el ancho de `[−]` (C-004); `fetchQuota` con `io.LimitReader` de 1 MiB y error explícito si se excede, en lugar de truncar en silencio (C-005); agrupar las peticiones del modo watch de /tdd en una sola ejecución pendiente y avisar con notify cuando un `/tdd run` queda en cola (C-003); test de /tdd sobre un módulo Go externo sin `pkg/installer` (C-006); test nuevo `pkg/installer/hcd021_audit_test.go`.
Excluido: descartar sin más las ejecuciones concurrentes, como proponía el informe para C-003 (la tool del agente necesita su resultado, así que se encola); preferir `./pkg/installer` cuando existe, como proponía el informe para C-006 (sería volver a rutas fijas); C-001 y C-002, cerrados en HCD-020; `docs/audits/`, que se conserva tal cual como registro.

## ¿Qué pasa si no lo hacemos?
Con un título de unidad largo, el panel lateral pierde su control de minimizar. Una respuesta de red anómala puede agotar la memoria de `pi-gomemory-setup usage`. En modo watch, una conversación activa acumula ejecuciones completas de tests en cola, con un coste creciente de CPU y tiempo.

## Alternativa más barata
Aplicar literalmente los parches del informe: una bandera inflight que descarta y un `LimitReader` que trunca. Es menos código, pero la bandera haría que la tool del agente devolviera resultados de otra ejecución, y el truncado convertiría una respuesta grande en un error de JSON confuso.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla en `pkg/installer/hcd021_audit_test.go` → `go test ./pkg/installer -run HCD021 -count=1`
- [x] 2. Mostrar siempre `[−]` en `renderSidebar` aunque el título exceda el ancho (C-004) → `go test ./pkg/installer -run TestHCD021SidebarLongTitle -count=1`
- [x] 3. Rechazar en `fetchQuota` las respuestas de más de 1 MiB con un error explícito (C-005) → `go test ./pkg/installer -run TestHCD021QuotaBodyLimit -count=1`
- [x] 4. Agrupar en una sola ejecución pendiente las peticiones del modo watch de /tdd (C-003) → `go test ./pkg/installer -run TestHCD021WatchCoalesces -count=1`
- [x] 5. Ejecutar /tdd en un módulo Go externo sin `pkg/installer` con `go test -json ./...` (C-006) → `go test ./pkg/installer -run TestHCD021ExternalGoRepo -count=1`
- [x] 6. Mantener en verde toda la suite y `go vet` → `go test ./... -count=1 && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 19:24 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run HCD021 -count=1` (salida 1)
      --- FAIL: TestHCD021SidebarLongTitle (0.13s)
          hcd021_audit_test.go:32: TestHCD021SidebarLongTitle: FAIL ancho 36: la cabecera perdió [−]: "│ HCD · HCD-020 · evolution · paso…│" | ancho 24: la cabecera perdió [−]: "│ HCD · HCD-020 · evol…│"
      --- FAIL: TestHCD021QuotaBodyLimit (0.00s)
          hcd021_audit_test.go:48: una respuesta de 2.5 MiB debe dar error; se leyeron 2621440 bytes
      --- FAIL: TestHCD021WatchCoalesces (3.57s)
          hcd021_audit_test.go:104: TestHCD021WatchCoalesces: FAIL go test se ejecutó 7 veces; se esperaban 3 (tool, watch agrupado, /tdd run) | un /tdd run en cola debe avisarlo: ["TDD Watch Mode: ACTIVADO","Ejecutando pruebas all..."]
      FAIL
      FAIL	pi-gomemory/pkg/installer	4.321s
      FAIL
- 2026-11-11 19:24 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD021SidebarLongTitle -count=1` (salida 1)
      --- FAIL: TestHCD021SidebarLongTitle (0.12s)
          hcd021_audit_test.go:32: TestHCD021SidebarLongTitle: FAIL ancho 36: la cabecera perdió [−]: "│ HCD · HCD-020 · evolution · paso…│" | ancho 24: la cabecera perdió [−]: "│ HCD · HCD-020 · evol…│"
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.270s
      FAIL
- 2026-11-11 19:24 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD021QuotaBodyLimit -count=1` (salida 1)
      --- FAIL: TestHCD021QuotaBodyLimit (0.00s)
          hcd021_audit_test.go:48: una respuesta de 2.5 MiB debe dar error; se leyeron 2621440 bytes
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.150s
      FAIL
- 2026-11-11 19:24 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD021WatchCoalesces -count=1` (salida 1)
      --- FAIL: TestHCD021WatchCoalesces (3.34s)
          hcd021_audit_test.go:104: TestHCD021WatchCoalesces: FAIL go test se ejecutó 7 veces; se esperaban 3 (tool, watch agrupado, /tdd run) | un /tdd run en cola debe avisarlo: ["TDD Watch Mode: ACTIVADO","Ejecutando pruebas all..."]
      FAIL
      FAIL	pi-gomemory/pkg/installer	3.471s
      FAIL
- 2026-11-11 19:25 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run HCD021 -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.345s
- 2026-11-11 19:25 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD021SidebarLongTitle -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.269s
- 2026-11-11 19:25 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD021QuotaBodyLimit -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.155s
- 2026-11-11 19:25 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestHCD021WatchCoalesces -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.751s
- 2026-11-11 19:25 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run TestHCD021ExternalGoRepo -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.567s
- 2026-11-11 19:25 -05:00 · tarea 6 · verde · vigente · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	14.271s

## Siguiente
Cerrada con aprobación sobre la revisión 47696170ab79.

# /plan con texto activa el modo plan y envía la solicitud

ID: HCD-007
Tipo: fix
Carril: fast
Motivo: fix con reproducción clara, sin decisiones abiertas

## Intención
/plan <texto> activa el modo plan y envía el texto como mensaje, en lugar de rechazarlo como argumento desconocido

## Reproducción
En pi-gomemory v1.4.0, `/plan deseo instalar ...` muestra «Warning: Argumento desconocido: deseo instalar ... Usa /plan o /plan run» y no activa el modo plan ni envía la solicitud (`extensions/gomemory-plan.ts`, rama `arg !== ""`).

## Alcance
Incluido: `/plan <texto>` activa el modo (si no lo estaba) y envía `<texto>` con `pi.sendUserMessage` (con `deliverAs: "followUp"` si el agente está ocupado). `/plan` sin texto y `/plan run` mantienen su comportamiento.
Excluido: la integración del seguimiento de tareas al estilo pi-goal-x (se trata en una unidad aparte).

## ¿Qué pasa si no lo hacemos?
La forma natural de usar /plan (con la petición en la misma línea) falla y se pierde lo escrito.

## Alternativa más barata
Documentar que hay que escribir /plan y luego la petición en otro mensaje; sigue perdiendo el texto.

## Aclaraciones

## Checklist
- [x] 1. /plan con texto activa el modo y envía el texto → `go test ./pkg/installer -run TestPlanWithPrompt`
- [x] 2. Suite completa → `go test ./... && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 05:11 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestPlanWithPrompt` (salida 1)
      --- FAIL: TestPlanWithPrompt (0.00s)
          hcd002_test.go:174: gomemory-plan.ts no contiene "sendUserMessage(arg"
          hcd002_test.go:174: gomemory-plan.ts no contiene "deliverAs: \"followUp\""
          hcd002_test.go:174: gomemory-plan.ts no contiene "isIdle"
          hcd002_test.go:178: /plan <texto> no debe rechazarse como argumento desconocido
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.314s
      FAIL
- 2026-11-11 05:11 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestPlanWithPrompt` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.160s
- 2026-11-11 05:11 -05:00 · tarea 2 · verde · caducada · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 05:16 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestPlanWithPrompt` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.315s
- 2026-11-11 05:16 -05:00 · tarea 2 · verde · vigente · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	(cached)

## Siguiente
Cerrada con aprobación sobre la revisión 17d9ad5f1bf4.

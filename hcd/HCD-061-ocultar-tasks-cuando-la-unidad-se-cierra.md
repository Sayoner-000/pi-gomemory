# Ocultar /tasks cuando la unidad se cierra desde otra sesión

ID: HCD-061
Tipo: fix
Carril: fast
Motivo: Defecto observado por la persona con causa clara y solución decidida; sin decisiones abiertas

## Intención
Con el panel visible, cada sesión de Pi consulta mem hcd show cada 10 s y oculta /tasks en cuanto la unidad deja de estar activa, aunque se haya cerrado o cancelado desde otra sesión.

## Reproducción
La persona trabaja con varias sesiones de Pi en el mismo proyecto. Al cerrar una unidad HCD en una, las demás siguen mostrando su panel `/tasks` «confundiendo al usuario». Causa: `extensions/gomemory-tasks.ts` solo llama a `refresh` (que ya oculta el panel cuando `mem hcd show` no devuelve una unidad activa) en `session_start`, `agent_end` y cuando la propia sesión ejecuta `mem hcd` por Bash. Una sesión inactiva nunca vuelve a consultar al motor.

## Alcance
Incluido: en `extensions/gomemory-tasks.ts`, un latido de 10 s que llama a `refresh` mientras el panel está montado, se detiene al ocultarse y en `session_shutdown`, y no se solapa con un `refresh` en curso (el existente ya encola). Documentarlo en `docs/usage.md` y `docs/architecture.md`.
Excluido: ocultar el panel con todas las tareas verificadas pero la unidad sin cerrar (la persona eligió esperar al cierre); mostrar el panel en una sesión sin panel cuando otra abre una unidad; vigilar la carpeta `hcd/` en lugar de consultar.

## ¿Qué pasa si no lo hacemos?
Las sesiones que no ejecutan `mem hcd` siguen mostrando tareas de una unidad cerrada hasta su siguiente turno, y la persona puede creer que hay trabajo pendiente.

## Alternativa más barata
Pedir a la persona que ejecute `/tasks` dos veces en cada sesión tras cerrar una unidad. No resuelve la confusión que describe; se descarta.

## Normas aplicables
- Constitución §10 Pruebas: rojo antes del cambio en todo `fix`; la prueba no depende de esperas reales de 10 s.
- Constitución §8 Manejo de errores y resiliencia: sin `mem` o con salida inválida, el panel se oculta como hoy, sin errores.

## Aclaraciones

## Checklist
- [x] 1. Con el panel visible, consultar mem hcd show cada 10 s y ocultar /tasks cuando la unidad deja de estar activa; detener el latido al ocultar y al cerrar la sesión → `go test ./pkg/installer -run '^TestHCD061Heartbeat$' -count=1`
- [x] 2. Mantener en verde las pruebas existentes del panel /tasks → `go test ./pkg/installer -run '^(TestHCD06.*|TestHCD057.*|TestHCD054.*|TestHCD051TasksLivePolling|TestTasksExtension)$' -count=1`
- [x] 3. Comprobar en Pi real con dos sesiones que cerrar la unidad en una oculta /tasks en la otra en ≤ 10 s → observación manual registrada
- [x] 4. Ejecutar la regresión completa → `go vet ./... && go test -race ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-20 04:41 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD061Heartbeat$' -count=1` (salida 1)
      --- FAIL: TestHCD061Heartbeat (0.38s)
          hcd061_test.go:56: TestHCD061Heartbeat: FAIL con el panel visible debe haber un latido de 10 s activo: 0 | el latido debe consultar mem hcd show | con la unidad cerrada el panel debe ocultarse | una unidad activa debe volver a montar el panel con un latido: 0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.641s
      FAIL
- 2026-11-20 04:41 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD061Heartbeat$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.484s
- 2026-11-20 04:41 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^(TestHCD06.*|TestHCD057.*|TestHCD054.*|TestHCD051TasksLivePolling|TestTasksExtension)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.311s
- 2026-11-20 04:43 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD061Heartbeat$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.469s
- 2026-11-20 04:43 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^(TestHCD06.*|TestHCD057.*|TestHCD054.*|TestHCD051TasksLivePolling|TestTasksExtension)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.644s
- 2026-11-20 04:43 -05:00 · tarea 3 · manual · vigente
      Pi 1.5.2 real vía Herdr (pestaña w3X:t, cwd en el scratchpad) con gomemory-tasks.ts instalado (SHA256 b9d0eb2a igual al repo). Para simular otra sesión sin cerrar una unidad real (el cierre exige la cita de la persona), PATH de ese panel con un envoltorio de mem que sirve 'mem hcd show --json' desde un archivo y delega el resto en el mem real. Con la unidad de prueba HCD-999 activa, /tasks mostraba '● 1. Tarea de prueba de HCD-061'. Se cambió el archivo a state=closed sin interactuar con Pi: el panel desapareció a los 8 s (dentro del latido de 10 s) con el agente idle, sin turnos de por medio.
- 2026-11-20 04:43 -05:00 · tarea 4 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	50.494s

## Siguiente
Cerrada con aprobación sobre la revisión 3fb4d5cb6a24.

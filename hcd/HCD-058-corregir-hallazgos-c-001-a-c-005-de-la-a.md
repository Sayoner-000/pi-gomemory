# Corregir hallazgos C-001 a C-005 de la auditoría ACR/ADM de HCD-056 y HCD-057

ID: HCD-058
Tipo: fix
Carril: fast
Motivo: Defectos confirmados con evidencia reproducida y cierre propuesto por la auditoría; una única decisión abierta (C-005) ya resuelta con la persona

## Intención
Cerrar C-001 (pérdida de la salida final de un Bash seguido en vivo), C-002 (sugerencia escrita en el turno siguiente), C-003 (verify compuesto sin ingesta en /tdd), C-005 (watchDirty limpiado con otras suites de tests activas) y C-004 (temporizadores sin limpiar), cada uno con su prueba en rojo antes del cambio.

## Reproducción
Auditoría ACR + ADM `acr_74ee3e86-2025-4bf2-ad3d-6e0dfbb1e864` (veredicto ESCALATED) sobre v1.19.0:
- C-001 (alto): en `createBashLiveIngest.end`, si llegó al menos una línea por `tool_execution_update`, `finalText` se descarta. La salida posterior al último update se pierde: los tests quedan `running` y un fallo real se sustituye por `bash-error` genérico.
- C-002 (medio): `agent_end` de `extensions/gomemory-suggest.ts` espera `mem hcd show` sin guarda de turno; si empieza otro turno o se cierra la sesión mientras tanto, escribe la sugerencia o lanza la petición al modelo en el turno siguiente, y `await modelNext` retrasa el final de `agent_end`.
- C-003 (medio): `hcdVerifyCheckCommand` solo mira el primer segmento shell; `cd sub && mem hcd verify --task 1 -- 'go test ./...'` no se ingiere en /tdd.
- C-005 (bajo): un Bash de tests completo limpia `watchDirty` aunque haya otras suites de tests activas (frontend, playwright).
- C-004 (bajo, sospechoso): los `setTimeout` de `session_start` en `extensions/gomemory-suggest.ts` no se cancelan en `session_shutdown`.

## Alcance
Incluido: en `extensions/gomemory-tdd.ts`, reconciliar `finalText` (con su truncado) en `end` cuando cubre al menos lo ya visto; recorrer todos los segmentos en `hcdVerifyCheckCommand`; limpiar `watchDirty` desde Bash solo si la suite del comando es la única suite de tests activa (backend, frontend, playwright), sin contar CI & Static ni Data & Graph (Q1). En `extensions/gomemory-suggest.ts`, un contador `epoch` que invalida el trabajo de `agent_end` al empezar otro turno o cerrar la sesión, `modelNext` sin `await` y cancelación de los temporizadores de `session_start`. Una prueba por hallazgo con rojo observado.
Excluido: cambiar el formato de evidencia del motor HCD; suciedad por suite en watch (Q1, opción descartada); publicar una versión nueva (lo decide la persona).

## ¿Qué pasa si no lo hacemos?
/tdd puede mostrar como «error de ejecución» un test que falló de verdad y dejar tests en curso para siempre (C-001); la sugerencia del modelo puede aparecer o gastar tokens durante el turno siguiente (C-002); las verificaciones lanzadas tras `cd` no aparecen en /tdd (C-003); y watch puede no comprobar el frontend tras un `go test` (C-005).

## Alternativa más barata
Desactivar la ingesta en vivo y la sugerencia del modelo hasta la siguiente versión. Elimina los síntomas pero retira dos funciones recién publicadas en v1.19.0; se descarta porque los cierres son locales y pequeños.

## Normas aplicables
- Constitución §10 Pruebas: deterministas; rojo antes del cambio en todo `fix`.
- Constitución §8 Manejo de errores y resiliencia: el trabajo asíncrono obsoleto no debe mutar estado.
- Playbook HCD-TDD-01 y HCD-EVIDENCE-01.

## Aclaraciones
- [x] Q1. C-005: el cierre literal (limpiar watchDirty solo con ≤ 1 suite activa) anula HCD-056 tarea 9 en repos Go (Backend + CI): (1) contar solo suites de tests (backend, frontend, playwright); (2) literal; (3) suciedad por suite? → Solo suites de tests (Recommended)

## Checklist
- [x] 1. C-001: reconciliar en createBashLiveIngest.end el texto final del resultado (con su truncado) cuando cubre lo ya visto → `go test ./pkg/installer -run '^TestHCD058FinalTextReconcile$' -count=1`
- [x] 2. C-002: invalidar con un epoch el trabajo de agent_end al empezar otro turno o cerrar la sesión, y lanzar modelNext sin esperar → `go test ./pkg/installer -run '^TestHCD058SuggestEpoch$' -count=1`
- [x] 3. C-003: recorrer todos los segmentos en hcdVerifyCheckCommand para ingerir verify compuestos → `go test ./pkg/installer -run '^TestHCD058VerifyCompound$' -count=1`
- [x] 4. C-005: limpiar watchDirty desde Bash solo si su suite es la única suite de tests activa (Q1) → `go test ./pkg/installer -run '^TestHCD058WatchDirtyScope$' -count=1`
- [x] 5. C-004: cancelar en session_shutdown los temporizadores de session_start de gomemory-suggest.ts → `go test ./pkg/installer -run '^TestHCD058SuggestTimers$' -count=1`
- [x] 6. Ejecutar la regresión completa → `go vet ./... && go test -race ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-20 04:11 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD058FinalTextReconcile$' -count=1` (salida 1)
      --- FAIL: TestHCD058FinalTextReconcile (0.26s)
          hcd058_test.go:66: TestHCD058FinalTextReconcile: FAIL 1: TestB debió fallar con su salida real: running | 1: no debe sustituirse por bash-error | 2: con cola truncada TestB debió fallar: running
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.486s
      FAIL
- 2026-11-20 04:11 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD058SuggestEpoch$' -count=1` (salida 1)
      --- FAIL: TestHCD058SuggestEpoch (0.55s)
          hcd058_test.go:115: TestHCD058SuggestEpoch: FAIL a) la sugerencia del turno anterior no debe aparecer en el nuevo: /hcd-implement · Tab | b) no debe pedirse al modelo en el turno siguiente | b) sin sugerencia en el turno nuevo | c) tras cerrar la sesión no se escribe sugerencia | d) agent_end no debe esperar a modelNext (301 ms)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.758s
      FAIL
- 2026-11-20 04:11 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD058VerifyCompound$' -count=1` (salida 1)
      --- FAIL: TestHCD058VerifyCompound (0.25s)
          hcd058_test.go:143: TestHCD058VerifyCompound: FAIL no se ingirió: cd sub && mem hcd verify --task 1 --json -- 'go test -v ./...' | no se ingirió: export X=1 && cd sub && mem hcd verify --task 1 -- 'go test -v ./...'
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.439s
      FAIL
- 2026-11-20 04:11 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD058WatchDirtyScope$' -count=1` (salida 1)
      --- FAIL: TestHCD058WatchDirtyScope (1.45s)
          hcd058_test.go:182: TestHCD058WatchDirtyScope: FAIL watch debió comprobar el frontend tras un go test: []
      FAIL
      FAIL	pi-gomemory/pkg/installer	1.666s
      FAIL
- 2026-11-20 04:11 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD058SuggestTimers$' -count=1` (salida 1)
      --- FAIL: TestHCD058SuggestTimers (1.44s)
          hcd058_test.go:199: TestHCD058SuggestTimers: FAIL un temporizador de session_start reinstaló el editor tras session_shutdown
      FAIL
      FAIL	pi-gomemory/pkg/installer	1.638s
      FAIL
- 2026-11-20 04:11 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD058FinalTextReconcile$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.458s
- 2026-11-20 04:12 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD058VerifyCompound$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.432s
- 2026-11-20 04:12 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD058WatchDirtyScope$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.654s
- 2026-11-20 04:13 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD058SuggestEpoch$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.491s
- 2026-11-20 04:13 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD058SuggestTimers$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.639s
- 2026-11-20 04:13 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD058FinalTextReconcile$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.461s
- 2026-11-20 04:13 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD058VerifyCompound$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.453s
- 2026-11-20 04:13 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD058WatchDirtyScope$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.682s
- 2026-11-20 04:13 -05:00 · tarea 6 · verde · caducada · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	41.780s
- 2026-11-20 04:15 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD058FinalTextReconcile$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.486s
- 2026-11-20 04:15 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD058SuggestEpoch$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.457s
- 2026-11-20 04:15 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD058VerifyCompound$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.460s
- 2026-11-20 04:15 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD058WatchDirtyScope$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.656s
- 2026-11-20 04:15 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD058SuggestTimers$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.633s
- 2026-11-20 04:15 -05:00 · tarea 6 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	(cached)

## Siguiente
Cerrada con aprobación sobre la revisión a4fbf13f7797.

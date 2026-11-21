# Corregir hallazgos H-01 a H-06 de la auditoría ACR/ADM v1.23.0

ID: HCD-069
Tipo: fix
Carril: fast
Motivo: Vía rápida: 6 correcciones acotadas con reproducción clara y sin decisiones interdependientes

## Intención
Cerrar los 6 hallazgos confirmados (4 Medios, 2 Bajos) de la auditoría ACR/ADM sobre v1.23.0 (acr_06563425-907d-49bf-b17c-c74fd1772a7e) con pruebas rojo->verde

## Reproducción
Auditoría ACR/ADM sobre `v1.23.0` (`1df7b84`, ACR `acr_06563425-907d-49bf-b17c-c74fd1772a7e`, memoria `#604`), continuando `HCD-065`:
1. **H-01 (`extensions/gomemory-tdd.ts:2553-2585`)**: abortar `doRunTestSuite` en una ejecución no-watch sale por `break` en la línea 2580 sin llamar a `finishSuite` ni `skipSuite`, dejando la suite y `store.getState().status` en `"running"`.
2. **H-02 (`pkg/installer/tools_signal.go:125-136`)**: crear `.pi-gomemory-deps-orphan/.pid` con el PID de un proceso ajeno vivo (reutilización de PID tras `SIGKILL`); `RemoveStaleDepsTemps` lo omite al validar solo `syscall.Kill(pid, 0)`.
3. **H-03 (`extensions/gomemory-plan.ts:56-74`)**: `toolBlocked("hcd_rename", { title: "x" })` y `toolBlocked("mcp__gomemory__hcd_rename", { title: "x" })` devuelven `undefined` mientras `toolBlocked("bash", { command: "mem hcd rename --title x" })` bloquea.
4. **H-04 (`extensions/gomemory-tdd.ts:1555-1568`)**: `classifyBashTestCommand("go test ./... -skip TestSlow")` devuelve `partial: false` porque `valueFlags` consume `-skip` y `hasRun` solo detecta `-run`.
5. **H-05 (`extensions/gomemory-tasks.ts:913-950`)**: secuencia `session_start(ctx1)` → `session_shutdown(ctx1)` → `session_start(ctx2)` antes de resolver el primer `pi.exec`; el `finally` borra `pendingCtx = null` por `epoch !== sessionEpoch` y `ctx2` nunca ejecuta su `refresh` (`execCalls=1, mounts=0`).
6. **H-06 (`pkg/installer/tools_exec.go:70-90`, `pkg/installer/tools_signal.go:112`)**: un directorio `.pi-gomemory-staging-*` huérfano antes de `os.Rename` no se registra en `trackTemp` ni es retirado por `RemoveStaleDepsTemps`.

## Alcance
Incluido: corrección con pruebas rojo→verde de H-01 y H-04 en `extensions/gomemory-tdd.ts`, H-02 y H-06 en `pkg/installer/tools_exec.go` y `pkg/installer/tools_signal.go`, H-03 en `extensions/gomemory-plan.ts`, H-05 en `extensions/gomemory-tasks.ts`, y tests en `pkg/installer/hcd069_fixes_test.go` y `tests/gomemory-plan.test.mjs`. Normas aplicables: Constitución §1.2 (Go, biblioteca estándar primero, `%w`), §8 (manejo de errores), §10 (pruebas deterministas), §21 (causa raíz, simplicidad); Playbook `HCD-DEF-01`, `HCD-TASK-01`, `HCD-TDD-01`, `HCD-EVIDENCE-01`. Antecedente: `HCD-065` (cierre parcial de `H-07`/`consensus_id 86`).
Excluido: cambios en `install.sh`, `scripts/*.sh`, `.gitignore`, publicación de release, refactorizaciones fuera de las 5 superficies afectadas o instalación de linters externos en el host.

## ¿Qué pasa si no lo hacemos?
- En `/tdd`, cancelar una ejecución deja el panel atascado en `running` hasta reiniciar Pi (`H-01`) y un `go test ./... -skip ...` suprime el Watch de regresión ocultando fallos (`H-04`).
- En `/plan`, el agente puede renombrar archivos HCD y mutar memorias vía `hcd_rename` en modo de solo lectura (`H-03`).
- En `/tasks`, cambiar de sesión con un `refresh` en vuelo deja el panel vacío hasta un evento posterior (`H-05`).
- En el instalador, temporales `.pi-gomemory-deps-*` (~335 MB) o `.pi-gomemory-staging-*` pueden quedar huérfanos de forma permanente en `~/.local/bin` (`H-02`, `H-06`).

## Alternativa más barata
Dejarlos abiertos como deuda documentada en memoria `#604`; se descarta porque las 6 correcciones son acotadas, locales, sin nuevas dependencias y cierran regresiones directas de `HCD-065` y `HCD-066`.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. H-01 (`/tdd`): cerrar y marcar `skipped` la suite al abortar una ejecución no-watch en `doRunTestSuite` → `go test ./pkg/installer -run '^TestHCD069TDDAbortClosesRunningSuite$'`
- [x] 2. H-02 (`tools_signal.go`/`tools_exec.go`): proteger temporales con `flock` para limpiar huérfanos aunque un proceso ajeno reutilice el PID → `go test ./pkg/installer -run '^TestHCD069RemoveStaleReusedPIDOrphan$'`
- [x] 3. H-03 (`/plan`): bloquear `hcd_rename` y `mcp__gomemory__hcd_rename` en `toolBlocked` → `node --test tests/gomemory-plan.test.mjs`
- [x] 4. H-04 (`/tdd`): clasificar `go test ./... -skip ...` y `-short` como `partial=true` en `matchTestSegment` → `go test ./pkg/installer -run '^TestHCD069TDDGoTestSkipIsPartial$'`
- [x] 5. H-05 (`/tasks`): preservar y drenar `pendingCtx` de la nueva sesión en el `finally` de `refresh` → `go test ./pkg/installer -run '^TestHCD069TasksSessionSwitchRunsPendingRefresh$'`
- [x] 6. H-06 (`tools_exec.go`/`tools_signal.go`): rastrear `.pi-gomemory-staging-*` en `trackTemp` y barrer huérfanos en `RemoveStaleDepsTemps` → `go test ./pkg/installer -run '^TestHCD069RemoveStaleStagingTemps$'`
- [x] 7. Regresión completa Go (`-race`), Node y sintaxis shell → `go vet ./... && go test -race -count=1 ./... && node --test tests/*.test.mjs tests/antigravity-prompt-guard/*.test.* && bash -n install.sh scripts/*.sh`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-21 08:50 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD069TDDAbortClosesRunningSuite$'` (salida 1)
      --- FAIL: TestHCD069TDDAbortClosesRunningSuite (0.39s)
          hcd069_fixes_test.go:88: TestHCD069TDDAbortClosesRunningSuite: FAIL estado global TDD quedó running tras abortar | suite ci quedó running tras abortar | suite ci=running, esperado skipped
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.794s
      FAIL
- 2026-11-21 08:50 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD069RemoveStaleReusedPIDOrphan$'` (salida 1)
      --- FAIL: TestHCD069RemoveStaleReusedPIDOrphan (0.02s)
          hcd069_fixes_test.go:105: depsTempDir debe crear /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD069RemoveStaleReusedPIDOrphan2591991065/001/.local/bin/.pi-gomemory-deps-406118337/.lock: stat /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD069RemoveStaleReusedPIDOrphan2591991065/001/.local/bin/.pi-gomemory-deps-406118337/.lock: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.231s
      FAIL
- 2026-11-21 08:50 -05:00 · tarea 3 · rojo observado · caducada · `node --test tests/gomemory-plan.test.mjs` (salida 1)
      ✖ failing tests:
      test at tests/gomemory-plan.test.mjs:113:1
      ✖ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (1.619334ms)
        AssertionError [ERR_ASSERTION]: debía bloquear hcd_rename {"title":"nuevo"}
        + actual - expected
        + undefined
        - true
            at TestContext.<anonymous> (file:///Users/josegomezj/home/rcw/pi-gomemory/tests/gomemory-plan.test.mjs:156:10)
            at async Test.run (node:internal/test_runner/test:1409:7)
            at async Test.processPendingSubtests (node:internal/test_runner/test:974:7) {
          generatedMessage: false,
          code: 'ERR_ASSERTION',
          actual: undefined,
          expected: true,
          operator: 'strictEqual',
          diff: 'simple'
        }
- 2026-11-21 08:50 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD069TDDGoTestSkipIsPartial$'` (salida 1)
      --- FAIL: TestHCD069TDDGoTestSkipIsPartial (0.25s)
          hcd069_fixes_test.go:162: TestHCD069TDDGoTestSkipIsPartial: FAIL go test ./... -skip TestSlow debió ser partial=true: {"suiteId":"backend","parser":"exit-code","name":"go test (bash)","partial":false} | go test -skip=TestSlow ./... debió ser partial=true: {"suiteId":"backend","parser":"exit-code","name":"go test (bash)","partial":false} | go test -short ./... debió ser partial=true: {"suiteId":"backend","parser":"exit-code","name":"go test (bash)","partial":false} | go test -race -short ./... debió ser partial=true: {"suiteId":"backend","parser":"exit-code","name":"go test (bash)","partial":false}
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.469s
      FAIL
- 2026-11-21 08:50 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD069TasksSessionSwitchRunsPendingRefresh$'` (salida 1)
      --- FAIL: TestHCD069TasksSessionSwitchRunsPendingRefresh (0.26s)
          hcd069_fixes_test.go:219: TestHCD069TasksSessionSwitchRunsPendingRefresh: FAIL execCalls=1, esperado >=2 (pendingCtx descartado) | mounts=0, esperado >=1 para la nueva sesión
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.476s
      FAIL
- 2026-11-21 08:50 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD069RemoveStaleStagingTemps$'` (salida 1)
      --- FAIL: TestHCD069RemoveStaleStagingTemps (0.00s)
          hcd069_fixes_test.go:243: el staging huérfano /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD069RemoveStaleStagingTemps2673214588/001/.local/bin/.pi-gomemory-staging-dead debía eliminarse: <nil>
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.217s
      FAIL
- 2026-11-21 08:51 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD069TDDAbortClosesRunningSuite$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.619s
- 2026-11-21 08:51 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD069RemoveStaleReusedPIDOrphan$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.225s
- 2026-11-21 08:51 -05:00 · tarea 3 · verde · vigente · `node --test tests/gomemory-plan.test.mjs` (salida 0)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (3.260125ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (0.261958ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.172209ms)
      ✔ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (0.95825ms)
      ✔ planShortcutKeys: Alt+P y Ctrl+Alt+P; en Windows y WSL solo Ctrl+Alt+P (0.124125ms)
      ✔ Alt+P alterna el modo plan, bloquea y publica el estado para el header (0.237917ms)
      ℹ tests 6
      ℹ suites 0
      ℹ pass 6
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 245.320917
- 2026-11-21 08:51 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD069TDDGoTestSkipIsPartial$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.454s
- 2026-11-21 08:51 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD069TasksSessionSwitchRunsPendingRefresh$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.473s
- 2026-11-21 08:51 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run '^TestHCD069RemoveStaleStagingTemps$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.214s
- 2026-11-21 08:51 -05:00 · tarea 7 · verde · vigente · `go vet ./... && go test -race -count=1 ./... && node --test tests/*.test.mjs tests/antigravity-prompt-guard/*.test.* && bash -n install.sh scripts/*.sh` (salida 0)
      ✔ passes through blank system text (0.083375ms)
      ✔ passes through malformed conversation turns (0.052792ms)
      ✔ does not discard non-text system metadata (0.054291ms)
      ✔ registers a working Pi hook scoped to Antigravity (0.165541ms)
      ✔ planHeaderLine: ON en ámbar, OFF atenuado, sin escapes con NO_COLOR, centrada (0.706667ms)
      ✔ el header muestra el modo actual y se repinta al cambiarlo (1.538584ms)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (3.726208ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (0.283125ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.179833ms)
      ✔ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (0.959542ms)
      ✔ planShortcutKeys: Alt+P y Ctrl+Alt+P; en Windows y WSL solo Ctrl+Alt+P (0.122833ms)
      ✔ Alt+P alterna el modo plan, bloquea y publica el estado para el header (0.254167ms)
      ℹ tests 30
      ℹ suites 0
      ℹ pass 30
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 251.180167

## Siguiente
Cerrada con aprobación sobre la revisión e5af6fb620ee.

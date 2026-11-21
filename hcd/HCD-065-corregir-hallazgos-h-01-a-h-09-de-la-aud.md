# Corregir hallazgos H-01 a H-09 de la auditoría ACR/ADM de v1.21.1

ID: HCD-065
Tipo: fix
Carril: fast
Motivo: Fix acotado sobre los 9 hallazgos confirmados (6 Medios, 3 Bajos) de la auditoría ACR/ADM sobre v1.21.1 (bb5f342), con reproducción determinista y sin decisiones interdependientes de alto riesgo

## Intención
Cerrar los 9 hallazgos confirmados de la auditoría ACR/ADM sobre v1.21.1 (bb5f342, ACR acr_3edbeeef-cb54-4a7c-a08f-0b6514cc1c04): H-01 guarda de scripts/clean-install-check.sh en commits etiquetados; H-02 finalización del parser en finishBashRun cuando isError=true; H-03 evitar remontaje/fuga de timers tras session_shutdown en gomemory-tasks.ts; H-04 incluir cacheRead/cacheWrite en métricas de contexto de sidebar y status; H-05 clasificar go test sin paquete como partial=true en gomemory-tdd.ts; H-06 cancelar subprocesos detached y watchAbort en session_shutdown de gomemory-tdd.ts; H-07 descartar pid==os.Getpid() en depsTempOwnedByLiveProcess; H-08 crear temporal en staging con .pid validado antes de renombrar en depsTempDir; H-09 limpiar temporales huérfanos en opts.LocalBinDir y manifest.BinDir en Uninstall.

## Reproducción
Reproducibles de forma determinista sobre v1.21.1 (`bb5f342c9c7b`, auditoría ACR `acr_3edbeeef-cb54-4a7c-a08f-0b6514cc1c04` / ADM memoria `#589`):
1. **H-01 (C-001):** en un checkout limpio del commit etiquetado `v1.21.1`, `install.sh:59` calcula `LOCAL_VERSION=v1.21.1` e imprime `✓ Binario v1.21.1 compilado en: ...`; `scripts/clean-install-check.sh:103` busca `Binario ${PINNED_VER} compilado` y sale con código 1 rechazando la compilación local.
2. **H-02 (C-005):** iniciar `createBashLiveIngest` con `go test -json ./...`, alimentar `{"Action":"run","Package":"example","Test":"TestSlow"}` y cerrar con `end(id, true, ...)` — `finishBashRun` hace cortocircuito en `Boolean(isError) || parser.finish(res)` y deja `TestSlow` y el estado global en `running` (`stats.running=1`).
3. **H-03 (C-006):** en `extensions/gomemory-tasks.ts`, iniciar un `refresh` con `pi.exec` diferido, emitir `session_shutdown` y resolver `pi.exec` después — `refresh` remonta el widget (`widgetMounts=1`) y vuelve a arrancar los intervalos de `500 ms` y `10000 ms`.
4. **H-04 (C-007):** pasar a `collectContextMetrics` (`extensions/gomemory-sidebar.ts`) y al cálculo de `extensions/gomemory-status.ts` un turno con `input=100, output=200, cacheRead=100000, cacheWrite=5000, totalTokens=105300` en ventana de `200000` — devuelven `300` tokens (`0%`) en lugar de `105300` (`53%`).
5. **H-05 (C-008):** llamar a `classifyBashTestCommand("go test")` o `classifyBashTestCommand("go test -v")` — devuelve `partial: false` pese a ejecutar solo el paquete actual y no `./...`, limpiando `watchDirty` y omitiendo otros paquetes (como `pkg/installer`).
6. **H-06 (C-009):** lanzar `/tdd run backend` (o Watch) y emitir `session_shutdown` mientras corre — el grupo de procesos detached en `streamLines` no recibe cancelación y los timers activos sobreviven al cierre.
7. **H-07 (C-002):** crear `<binDir>/.pi-gomemory-deps-orphan` no registrado en `liveTemps` con `.pid` igual a `os.Getpid()` y llamar `RemoveStaleDepsTemps` — `depsTempOwnedByLiveProcess` devuelve `true` y deja el huérfano sin borrar.
8. **H-08 (C-003):** `depsTempDir` (`pkg/installer/tools_exec.go:68-80`) crea el directorio directamente con `depsTempPrefix` antes de escribir `.pid` y descarta el error de `writeFileAtomic` con `_ = ...`, abriendo una ventana TOCTOU y ocultando fallos de escritura.
9. **H-09 (C-004):** con un manifiesto apuntando a `BinDir=B` y `PI_GOMEMORY_BIN_DIR=A` (donde existe un temporal huérfano `.pi-gomemory-deps-*`), ejecutar `Uninstall --yes` — solo escanea `B` y deja el temporal de `A`.

## Alcance
Incluido: corregir H-01 a H-09 en `scripts/clean-install-check.sh`, `extensions/gomemory-tdd.ts`, `extensions/gomemory-tasks.ts`, `extensions/gomemory-sidebar.ts`, `extensions/gomemory-status.ts`, `pkg/installer/tools_signal.go`, `pkg/installer/tools_exec.go` y `pkg/installer/uninstall.go`, con pruebas de regresión rojo→verde en `hcd062_shell_test.go` y `pkg/installer/hcd065_fixes_test.go`.
Excluido: nuevas funcionalidades ajenas a los 9 hallazgos, cambios de pins/dependencias, publicación de release/tag o modificación de documentación de release.

## ¿Qué pasa si no lo hacemos?
`scripts/clean-install-check.sh` falla en cada commit de release etiquetado; `/tdd` deja tests en estado `running` tras interrupciones, puede suprimir regresiones Watch con `go test` local y deja subprocesos detached corriendo al cerrar sesión; `/tasks` fuga timers y remonta widgets tras `session_shutdown`; `/sidebar` y la barra de estado ocultan el consumo real de contexto con prompt caching; y la limpieza de temporales de `pkg/installer` retiene huérfanos ante PID reciclado, fallos silenciosos de `.pid` o `PI_GOMEMORY_BIN_DIR` distinto del manifiesto.

## Alternativa más barata
Corregir solo H-01 (consenso ACR) y dejar H-02..H-09 para después. Sin embargo, los 9 hallazgos están confirmados y reproducidos con cambios mínimos y acotados sobre archivos ya conocidos (decisiones previas: HCD-058, HCD-061 y HCD-064), y cerrarlos en una sola unidad `fix` evita dejar fugas activas de procesos/timers y métricas erróneas de contexto. Normas aplicables: Constitución §0 y Playbook HCD-FLOW-01 / HCD-DEF-01.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. H-01 (C-001): ajustar la guarda de `scripts/clean-install-check.sh` para aceptar `Binario ${PINNED_VER} compilado` en commits etiquetados y rechazar solo la descarga de la release fijada (`scripts/clean-install-check.sh`) → `go test . -run TestHCD065CleanInstallCheckTaggedCommit -count=1`
- [x] 2. H-02 (C-005): ejecutar siempre `parser.finish(res)` en `finishBashRun` aunque `isError` sea `true` para cerrar tests interrumpidos (`extensions/gomemory-tdd.ts`) → `go test -race ./pkg/installer -run TestHCD065TDDBashInterruptedClosesRunning -count=1`
- [x] 3. H-03 (C-006): invalidar refrescos en vuelo y limpiar `pendingCtx` en `session_shutdown` de `/tasks` para evitar remontaje y fuga de timers (`extensions/gomemory-tasks.ts`) → `go test -race ./pkg/installer -run TestHCD065TasksShutdownStopsLateRefresh -count=1`
- [x] 4. H-04 (C-007): incluir `cacheRead` y `cacheWrite` (con respaldo `totalTokens`) en el cálculo de contexto de sidebar y status (`extensions/gomemory-sidebar.ts`, `extensions/gomemory-status.ts`) → `go test -race ./pkg/installer -run TestHCD065ContextMetricsIncludesCache -count=1`
- [x] 5. H-05 (C-008): clasificar `go test` sin paquete como `partial: true` para no suprimir la regresión Watch `./...` (`extensions/gomemory-tdd.ts`) → `go test -race ./pkg/installer -run TestHCD065TDDBareGoTestIsPartial -count=1`
- [x] 6. H-06 (C-009): cancelar `watchAbort`, ejecuciones activas y subprocesos detached en `session_shutdown` de `/tdd` (`extensions/gomemory-tdd.ts`) → `go test -race ./pkg/installer -run TestHCD065TDDShutdownAbortsActiveRun -count=1`
- [x] 7. H-07 (C-002): descartar `pid == os.Getpid()` en `depsTempOwnedByLiveProcess` para retirar huérfanos no registrados con PID reciclado (`pkg/installer/tools_signal.go`) → `go test -race ./pkg/installer -run TestHCD065RemoveStaleOwnPIDOrphan -count=1`
- [x] 8. H-08 (C-003): crear el temporal en staging, propagar error de `writeFileAtomic(.pid)` y renombrar a `depsTempPrefix` en `depsTempDir` (`pkg/installer/tools_exec.go`) → `go test -race ./pkg/installer -run TestHCD065DepsTempDirAtomicPID -count=1`
- [x] 9. H-09 (C-004): limpiar temporales huérfanos tanto en `opts.LocalBinDir` como en `manifest.BinDir` (deduplicados) en `Uninstall` (`pkg/installer/uninstall.go`) → `go test -race ./pkg/installer -run TestHCD065UninstallCleansEnvAndManifestBinDir -count=1`
- [x] 10. Regresión completa (sin rojo previo: comprueba toda la suite Go, sintaxis shell y pruebas Node tras los 9 cierres) → `go vet ./... && go test -race ./... && bash -n install.sh scripts/*.sh && node --check extensions/antigravity-prompt-guard.js && node --test tests/gomemory-plan.test.mjs tests/antigravity-prompt-guard/guard.test.mjs`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-21 07:21 -05:00 · tarea 1 · rojo observado · caducada · `go test . -run TestHCD065CleanInstallCheckTaggedCommit -count=1` (salida 1)
      --- FAIL: TestHCD065CleanInstallCheckTaggedCommit (0.00s)
          hcd062_shell_test.go:125: scripts/clean-install-check.sh no debe rechazar 'Binario ${PINNED_VER} compilado' porque install.sh imprime ese mensaje al compilar localmente en un commit etiquetado
      FAIL
      FAIL	pi-gomemory	0.369s
      FAIL
- 2026-11-21 07:21 -05:00 · tarea 1 · verde · caducada · `go test . -run TestHCD065CleanInstallCheckTaggedCommit -count=1` (salida 0)
      ok  	pi-gomemory	0.178s
- 2026-11-21 07:22 -05:00 · tarea 2 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD065TDDBashInterruptedClosesRunning -count=1` (salida 1)
      --- FAIL: TestHCD065TDDBashInterruptedClosesRunning (0.26s)
          hcd065_fixes_test.go:34: TestHCD065TDDBashInterruptedClosesRunning: FAIL estado global quedó running tras error | stats.running=1, esperado 0 | TestSlow=running, esperado failed
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.545s
      FAIL
- 2026-11-21 07:22 -05:00 · tarea 2 · verde · caducada · `go test -race ./pkg/installer -run TestHCD065TDDBashInterruptedClosesRunning -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.515s
- 2026-11-21 07:23 -05:00 · tarea 3 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD065TasksShutdownStopsLateRefresh -count=1` (salida 1)
      --- FAIL: TestHCD065TasksShutdownStopsLateRefresh (0.25s)
          hcd065_fixes_test.go:103: TestHCD065TasksShutdownStopsLateRefresh: FAIL remontó widget tras shutdown: 1 | fugó timers tras shutdown: 2
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.531s
      FAIL
- 2026-11-21 07:23 -05:00 · tarea 3 · verde · caducada · `go test -race ./pkg/installer -run TestHCD065TasksShutdownStopsLateRefresh -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.520s
- 2026-11-21 07:24 -05:00 · tarea 4 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD065ContextMetricsIncludesCache -count=1` (salida 1)
      --- FAIL: TestHCD065ContextMetricsIncludesCache (0.25s)
          hcd065_fixes_test.go:182: TestHCD065ContextMetricsIncludesCache: FAIL sidebar tokens=300, esperado 105300 | sidebar fallback totalTokens=0, esperado 60000 | footer no muestra 53%: ⟦mem⟧ [░░░░░░░░░░░░] 0% (300/200.0k) │ ↑100 ↓200 ↺100.0k │ $0.150                                                              claude-sonnet | footer no muestra 105.3k/200.0k: ⟦mem⟧ [░░░░░░░░░░░░] 0% (300/200.0k) │ ↑100 ↓200 ↺100.0k │ $0.150                                                              claude-sonnet
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.524s
      FAIL
- 2026-11-21 07:24 -05:00 · tarea 4 · verde · caducada · `go test -race ./pkg/installer -run TestHCD065ContextMetricsIncludesCache -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.516s
- 2026-11-21 07:25 -05:00 · tarea 5 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD065TDDBareGoTestIsPartial -count=1` (salida 1)
      --- FAIL: TestHCD065TDDBareGoTestIsPartial (0.26s)
          hcd065_fixes_test.go:206: TestHCD065TDDBareGoTestIsPartial: FAIL go test debió ser partial=true: {"suiteId":"backend","parser":"exit-code","name":"go test (bash)","partial":false} | go test -v debió ser partial=true: {"suiteId":"backend","parser":"go-text","name":"go test (bash)","partial":false} | go test -json debió ser partial=true: {"suiteId":"backend","parser":"go-test","name":"go test (bash)","partial":false}
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.543s
      FAIL
- 2026-11-21 07:25 -05:00 · tarea 5 · verde · caducada · `go test -race ./pkg/installer -run TestHCD065TDDBareGoTestIsPartial -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.513s
- 2026-11-21 07:25 -05:00 · tarea 6 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD065TDDShutdownAbortsActiveRun -count=1` (salida 1)
      --- FAIL: TestHCD065TDDShutdownAbortsActiveRun (1.88s)
          hcd065_fixes_test.go:292: TestHCD065TDDShutdownAbortsActiveRun: FAIL /tdd run siguió vivo tras session_shutdown | subproceso detached pid=64678 siguió vivo tras session_shutdown | timers vivos tras session_shutdown: 1
      FAIL
      FAIL	pi-gomemory/pkg/installer	2.152s
      FAIL
- 2026-11-21 07:26 -05:00 · tarea 6 · verde · caducada · `go test -race ./pkg/installer -run TestHCD065TDDShutdownAbortsActiveRun -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.633s
- 2026-11-21 07:26 -05:00 · tarea 7 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD065RemoveStaleOwnPIDOrphan -count=1` (salida 1)
      --- FAIL: TestHCD065RemoveStaleOwnPIDOrphan (0.00s)
          hcd065_fixes_test.go:328: el huérfano /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD065RemoveStaleOwnPIDOrphan2037826879/001/.local/bin/.pi-gomemory-deps-own-pid-orphan con pid==os.Getpid() no registrado en liveTemps debía eliminarse: <nil>
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.272s
      FAIL
- 2026-11-21 07:27 -05:00 · tarea 7 · verde · caducada · `go test -race ./pkg/installer -run TestHCD065RemoveStaleOwnPIDOrphan -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.284s
- 2026-11-21 07:27 -05:00 · tarea 8 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD065DepsTempDirAtomicPID -count=1` (salida 1)
      --- FAIL: TestHCD065DepsTempDirAtomicPID (0.00s)
          hcd065_fixes_test.go:349: tools_exec.go no debe ignorar el error de writeFileAtomic al escribir .pid
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.278s
      FAIL
- 2026-11-21 07:28 -05:00 · tarea 8 · verde · caducada · `go test -race ./pkg/installer -run TestHCD065DepsTempDirAtomicPID -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.469s
- 2026-11-21 07:28 -05:00 · tarea 9 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD065UninstallCleansEnvAndManifestBinDir -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd065_fixes_test.go:422:17: assignment mismatch: 1 variable but DefaultOptions returns 2 values
      pkg/installer/hcd065_fixes_test.go:432:51: too many arguments in call to InstallAll
      	have (fs.FS, unknown type, *bytes.Buffer)
      	want (fs.FS, *Options)
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-21 07:29 -05:00 · tarea 9 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD065UninstallCleansEnvAndManifestBinDir -count=1` (salida 1)
      rio codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD065UninstallCleansEnvAndManifestBinDir1744301292/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD065UninstallCleansEnvAndManifestBinDir1744301292/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- FAIL: TestHCD065UninstallCleansEnvAndManifestBinDir (0.37s)
          hcd065_fixes_test.go:441: el huérfano en PI_GOMEMORY_BIN_DIR (/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD065UninstallCleansEnvAndManifestBinDir1744301292/001/bin-env/.pi-gomemory-deps-orphan-env) debía eliminarse: <nil>
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.798s
      FAIL
- 2026-11-21 07:29 -05:00 · tarea 9 · verde · vigente · `go test -race ./pkg/installer -run TestHCD065UninstallCleansEnvAndManifestBinDir -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.605s
- 2026-11-21 07:29 -05:00 · tarea 10 · verde · vigente · `go vet ./... && go test -race ./... && bash -n install.sh scripts/*.sh && node --check extensions/antigravity-prompt-guard.js && node --test tests/gomemory-plan.test.mjs tests/antigravity-prompt-guard/guard.test.mjs` (salida 0)
      ✔ passes through malformed or nontext input null (0.093ms)
      ✔ passes through malformed or nontext input [] (0.048292ms)
      ✔ passes through malformed or nontext input {} (0.093167ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":null} (0.039875ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":{"systemInstruction":{"parts":[{"text":"text"},{"inlineData":{}}]},"contents":[]}} (0.045083ms)
      ✔ passes through blank system text (0.071042ms)
      ✔ passes through malformed conversation turns (0.060209ms)
      ✔ does not discard non-text system metadata (0.048958ms)
      ✔ registers a working Pi hook scoped to Antigravity (0.154667ms)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (2.951459ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (0.235375ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.171ms)
      ℹ tests 25
      ℹ suites 0
      ℹ pass 25
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 242.2655
- 2026-11-21 07:30 -05:00 · tarea 1 · verde · vigente · `go test . -run TestHCD065CleanInstallCheckTaggedCommit -count=1` (salida 0)
      ok  	pi-gomemory	0.210s
- 2026-11-21 07:31 -05:00 · tarea 2,3,4,5,6,7,8 · verde · vigente · `go test -race -v ./pkg/installer -run TestHCD065 -count=1` (salida 0)
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD065UninstallCleansEnvAndManifestBinDir1124219248/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- PASS: TestHCD065UninstallCleansEnvAndManifestBinDir (0.31s)
      === CONT  TestHCD065TDDBashInterruptedClosesRunning
      === CONT  TestHCD065ContextMetricsIncludesCache
      === CONT  TestHCD065TDDShutdownAbortsActiveRun
      === CONT  TestHCD065TasksShutdownStopsLateRefresh
      === CONT  TestHCD065TDDBareGoTestIsPartial
      --- PASS: TestHCD065TasksShutdownStopsLateRefresh (0.28s)
      --- PASS: TestHCD065ContextMetricsIncludesCache (0.29s)
      --- PASS: TestHCD065TDDBashInterruptedClosesRunning (0.29s)
      --- PASS: TestHCD065TDDBareGoTestIsPartial (0.30s)
      --- PASS: TestHCD065TDDShutdownAbortsActiveRun (0.41s)
      PASS
      ok  	pi-gomemory/pkg/installer	2.185s

## Siguiente
Cerrada con aprobación sobre la revisión 986210dca647.

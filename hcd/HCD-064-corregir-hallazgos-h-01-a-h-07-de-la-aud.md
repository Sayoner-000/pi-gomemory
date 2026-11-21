# Corregir hallazgos H-01 a H-07 de la auditoría ACR/ADM de v1.21.0 (HCD-062)

ID: HCD-064
Tipo: fix
Carril: fast
Motivo: Fix acotado sobre los 7 hallazgos confirmados (4 Medios, 3 Bajos) de la auditoría ACR/ADM sobre v1.21.0 (2c10005), sin decisiones interdependientes de alto riesgo

## Intención
Cerrar los 7 hallazgos confirmados de la auditoría ACR/ADM sobre v1.21.0 (2c10005, ACR acr_4f4d6293-1286-4642-bd68-8260ac29384c): H-01 matar grupo hijo en SIGINT/SIGTERM y salir con 143 en SIGTERM; H-02 mover limpieza de temporales huérfanos en uninstall tras la confirmación; H-03 proteger temporales vivos entre procesos con .pid en RemoveStaleDepsTemps; H-04 normalizar PI_GOMEMORY_BIN_DIR con filepath.Abs; H-05 limpiar descendientes en timeout con TTY; H-06 aceptar entradas de directorio ./ y dir/ en extractExact; H-07 leer la versión fijada dinámicamente en scripts/clean-install-check.sh.

## Reproducción
Reproducibles de forma determinista sobre v1.21.0 (`2c10005`, HCD-062, auditoría ACR `acr_4f4d6293-1286-4642-bd68-8260ac29384c` / ADM memoria 583):
1. **H-01 (C-002):** sin TTY, enviar `SIGTERM` al manejador de `trackTemp` mientras corre un instalador hijo en su propio `pgid` — el manejador borra temporales y llama a `signalExit(130)` sin matar al grupo hijo (que sigue vivo) y devuelve `130` en lugar de `143`.
2. **H-02 (C-003):** crear `<binDir>/.pi-gomemory-deps-orphan` y ejecutar `Uninstall` con `--dry-run=false` respondiendo `n` (o sin TTY ni `--yes`) — el directorio se borra antes de la confirmación aunque la salida diga «desinstalación cancelada: no se modificó nada».
3. **H-03 (C-001):** crear `<binDir>/.pi-gomemory-deps-other` representando un temporal activo de otro proceso (no presente en el mapa local `liveTemps`) y llamar a `RemoveStaleDepsTemps` — borra el directorio activo de la otra instancia.
4. **H-04 (C-004):** fijar `PI_GOMEMORY_BIN_DIR=relbin` (ruta relativa) y ejecutar `installByScript` o `installCodebaseMemory` — falla por doble resolución de la ruta relativa contra `cmd.Dir`.
5. **H-05 (C-005):** con `hooks.TTY = true`, ejecutar un script que lanza un subproceso hijo y vence `ScriptTimeout` — el nieto queda vivo tras agotarse el plazo.
6. **H-06 (C-006):** pasar a `extractExact` un `.tar.gz` válido con entrada de directorio raíz `./` o `dir/` y el binario `codebase-memory-mcp` — falla con «tarball con ruta no permitida».
7. **H-07 (C-007):** en `scripts/clean-install-check.sh:102` la guarda de release fijada sigue buscando `v1.20.0` aunque `install.sh:18` fija `v1.21.0`.

## Alcance
Incluido: corregir H-01 a H-07 en `pkg/installer/tools_signal.go`, `pkg/installer/tools_exec.go`, `pkg/installer/tools_cbm.go`, `pkg/installer/tools.go`, `pkg/installer/installer.go`, `pkg/installer/uninstall.go` y `scripts/clean-install-check.sh`, con pruebas de regresión rojo→verde en `pkg/installer/hcd064_*_test.go` y `hcd062_shell_test.go`.
Excluido: cambios ajenos en el árbol de trabajo (HCD-063 / pin de `pi-antigravity`), firma criptográfica/attestations upstream de `codebase-memory-mcp` (C-008 informativo), y publicación/tag de nueva release.

## ¿Qué pasa si no lo hacemos?
Un `SIGTERM` en CI o un timeout interactivo deja instaladores oficiales huérfanos mutando el equipo tras reportar cancelación; `uninstall` cancelado o sin TTY borra directorios contradiciendo su propio contrato; dos ejecuciones simultáneas de `install`/`uninstall` se rompen mutuamente; cualquier usuario que pase un `PI_GOMEMORY_BIN_DIR` relativo sufre un fallo determinista al instalar dependencias; un reempaquetado estándar con `./` de `codebase-memory-mcp` rompería su instalación; y `scripts/clean-install-check.sh` deja de detectar la caída a la release fijada `v1.21.0`.

## Alternativa más barata
Corregir solo los 4 Medios (H-01..H-04) y posponer los 3 Bajos (H-05..H-07). Sin embargo, H-06 y H-07 son cambios de pocas líneas en los mismos archivos de HCD-062 (decisión previa: HCD-062, memoria 577) y resolver los 7 juntos evita abrir una segunda unidad de mantenimiento. Normas aplicables: Constitución §0 y Playbook HCD-FLOW-01 / HCD-DEF-01.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. H-01 (C-002): matar el grupo de procesos hijo al recibir SIGINT/SIGTERM y devolver código 143 en SIGTERM (`pkg/installer/tools_signal.go`, `pkg/installer/tools_exec.go`) → `go test -race ./pkg/installer -run TestHCD064SignalKillsChildGroup -count=1`
- [x] 2. H-02 (C-003): mover la limpieza de temporales huérfanos en `Uninstall` después de la confirmación y el check de TTY (`pkg/installer/uninstall.go`) → `go test -race ./pkg/installer -run TestHCD064UninstallCancelPreservesTemps -count=1`
- [x] 3. H-03 (C-001): escribir marca `.pid` en `depsTempDir` y proteger temporales de otros procesos vivos en `RemoveStaleDepsTemps` (`pkg/installer/tools_exec.go`, `pkg/installer/tools_signal.go`) → `go test -race ./pkg/installer -run TestHCD064RemoveStaleRespectsLivePID -count=1`
- [x] 4. H-04 (C-004): normalizar `PI_GOMEMORY_BIN_DIR` a ruta absoluta en `DefaultOptions` y `depsTempDir` (`pkg/installer/installer.go`, `pkg/installer/tools_exec.go`) → `go test -race ./pkg/installer -run TestHCD064RelativeBinDir -count=1`
- [x] 5. H-05 (C-005): terminar descendientes del hijo cuando vence `ScriptTimeout` con TTY activo (`pkg/installer/tools_exec.go`) → `go test -race ./pkg/installer -run TestHCD064TTYTimeoutKillsDescendants -count=1`
- [x] 6. H-06 (C-006): aceptar entradas de directorio `./` y `dir/` y nombres benignos con `..` en `extractExact` manteniendo bloqueo de traversal y symlinks (`pkg/installer/tools_cbm.go`) → `go test -race ./pkg/installer -run TestHCD064ExtractExactDirEntries -count=1`
- [x] 7. H-07 (C-007): leer dinámicamente la versión fijada desde `install.sh` en `scripts/clean-install-check.sh` → `go test . -run TestHCD064CleanInstallCheckDynamicVersion -count=1`
- [x] 8. Regresión completa (`go vet`, `go test -race` y sintaxis shell) → `go vet ./... && go test -race ./... && bash -n install.sh && bash -n scripts/clean-install-check.sh`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-20 19:51 -05:00 · tarea 1 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD064SignalKillsChildGroup -count=1` (salida 1)
            /Users/josegomezj/home/rcw/pi-gomemory/pkg/installer/main_test.go:33 +0x19c
        main.main()
            _testmain.go:662 +0x104
      Goroutine 12 (running) created at:
        pi-gomemory/pkg/installer.TestHCD064SignalKillsChildGroup()
            /Users/josegomezj/home/rcw/pi-gomemory/pkg/installer/hcd064_fixes_test.go:43 +0x4ec
        testing.tRunner()
            /opt/homebrew/Cellar/go/1.27.4/libexec/src/testing/testing.go:2195 +0x164
        testing.(*T).Run.gowrap1()
            /opt/homebrew/Cellar/go/1.27.4/libexec/src/testing/testing.go:2260 +0x34
      ==================
      --- FAIL: TestHCD064SignalKillsChildGroup (2.03s)
          hcd064_fixes_test.go:73: código de salida en SIGTERM = 130, se esperaba 143
          hcd064_fixes_test.go:83: el instalador hijo (pid 86766) siguió vivo tras SIGTERM
          hcd064_fixes_test.go:86: el grupo de procesos -86766 sigue vivo tras SIGTERM: <nil>
          testing.go:1867: race detected during execution of test
      FAIL
      FAIL	pi-gomemory/pkg/installer	2.738s
      FAIL
- 2026-11-20 19:51 -05:00 · tarea 1 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064SignalKillsChildGroup -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.308s
- 2026-11-20 19:52 -05:00 · tarea 2 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD064UninstallCancelPreservesTemps -count=1` (salida 1)
      2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD064UninstallCancelPreservesTempsno_tty1643835211/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- FAIL: TestHCD064UninstallCancelPreservesTemps (0.58s)
          --- FAIL: TestHCD064UninstallCancelPreservesTemps/cancel (0.30s)
              hcd064_fixes_test.go:117: el temporal /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD064UninstallCancelPreservesTempscancel1355692411/001/.local/bin/.pi-gomemory-deps-orphan-cancel se borró pese a cancelar: stat /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD064UninstallCancelPreservesTempscancel1355692411/001/.local/bin/.pi-gomemory-deps-orphan-cancel: no such file or directory
          --- FAIL: TestHCD064UninstallCancelPreservesTemps/no_tty (0.28s)
              hcd064_fixes_test.go:137: el temporal /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD064UninstallCancelPreservesTempsno_tty1643835211/001/.local/bin/.pi-gomemory-deps-orphan-notty se borró sin confirmación ni TTY: stat /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD064UninstallCancelPreservesTempsno_tty1643835211/001/.local/bin/.pi-gomemory-deps-orphan-notty: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.868s
      FAIL
- 2026-11-20 19:52 -05:00 · tarea 2 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064UninstallCancelPreservesTemps -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.851s
- 2026-11-20 19:53 -05:00 · tarea 3 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD064RemoveStaleRespectsLivePID -count=1` (salida 1)
      --- FAIL: TestHCD064RemoveStaleRespectsLivePID (0.00s)
          hcd064_fixes_test.go:154: depsTempDir debe escribir .pid: open /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD064RemoveStaleRespectsLivePID1315908632/001/.local/bin/.pi-gomemory-deps-2050954644/.pid: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.284s
      FAIL
- 2026-11-20 19:53 -05:00 · tarea 3 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064RemoveStaleRespectsLivePID -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.268s
- 2026-11-20 19:54 -05:00 · tarea 4 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD064RelativeBinDir -count=1` (salida 1)
      --- FAIL: TestHCD064RelativeBinDir (0.00s)
          hcd064_fixes_test.go:199: DefaultOptions().LocalBinDir debe ser absoluta, obtenido: "relbin"
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.282s
      FAIL
- 2026-11-20 19:54 -05:00 · tarea 4 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064RelativeBinDir -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.350s
- 2026-11-20 19:55 -05:00 · tarea 5 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD064TTYTimeoutKillsDescendants -count=1` (salida 1)
      --- FAIL: TestHCD064TTYTimeoutKillsDescendants (6.91s)
          hcd064_fixes_test.go:255: el proceso nieto (pid 93282) siguió vivo tras vencer el timeout con TTY
      FAIL
      FAIL	pi-gomemory/pkg/installer	7.194s
      FAIL
- 2026-11-20 19:55 -05:00 · tarea 5 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064TTYTimeoutKillsDescendants -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.711s
- 2026-11-20 19:56 -05:00 · tarea 6 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD064ExtractExactDirEntries -count=1` (salida 1)
      --- FAIL: TestHCD064ExtractExactDirEntries (0.00s)
          hcd064_fixes_test.go:303: extractExact rechazó un tarball válido con ./, dir/ y NOTES..md: tarball con ruta no permitida: "./"
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.286s
      FAIL
- 2026-11-20 19:56 -05:00 · tarea 6 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064ExtractExactDirEntries -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.280s
- 2026-11-20 19:56 -05:00 · tarea 7 · rojo observado · caducada · `go test . -run TestHCD064CleanInstallCheckDynamicVersion -count=1` (salida 1)
      --- FAIL: TestHCD064CleanInstallCheckDynamicVersion (0.00s)
          hcd062_shell_test.go:100: scripts/clean-install-check.sh no debe fijar literalmente v1.20.0
      FAIL
      FAIL	pi-gomemory	0.203s
      FAIL
- 2026-11-20 19:57 -05:00 · tarea 1 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064SignalKillsChildGroup -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.286s
- 2026-11-20 19:57 -05:00 · tarea 2 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064UninstallCancelPreservesTemps -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.889s
- 2026-11-20 19:57 -05:00 · tarea 3 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064RemoveStaleRespectsLivePID -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.271s
- 2026-11-20 19:57 -05:00 · tarea 4 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064RelativeBinDir -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.292s
- 2026-11-20 19:57 -05:00 · tarea 5 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064TTYTimeoutKillsDescendants -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.695s
- 2026-11-20 19:57 -05:00 · tarea 6 · verde · caducada · `go test -race ./pkg/installer -run TestHCD064ExtractExactDirEntries -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.253s
- 2026-11-20 19:57 -05:00 · tarea 7 · verde · caducada · `go test . -run TestHCD064CleanInstallCheckDynamicVersion -count=1` (salida 0)
      ok  	pi-gomemory	0.191s
- 2026-11-20 19:57 -05:00 · tarea 8 · falló · caducada · `go vet ./... && go test -race ./... && bash -n install.sh && bash -n scripts/clean-install-check.sh` (salida 1)
        [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean3070052563/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean3070052563/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- FAIL: TestHCD012AtomicNoWriteFile (0.12s)
          hcd012_atomic_test.go:30: tools_exec.go usa os.WriteFile
      FAIL
      FAIL	pi-gomemory/pkg/installer	48.873s
      FAIL
- 2026-11-20 19:58 -05:00 · tarea 1 · verde · vigente · `go test -race ./pkg/installer -run TestHCD064SignalKillsChildGroup -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.311s
- 2026-11-20 19:58 -05:00 · tarea 2 · verde · vigente · `go test -race ./pkg/installer -run TestHCD064UninstallCancelPreservesTemps -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.849s
- 2026-11-20 19:58 -05:00 · tarea 3 · verde · vigente · `go test -race ./pkg/installer -run TestHCD064RemoveStaleRespectsLivePID -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.261s
- 2026-11-20 19:58 -05:00 · tarea 4 · verde · vigente · `go test -race ./pkg/installer -run TestHCD064RelativeBinDir -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.311s
- 2026-11-20 19:58 -05:00 · tarea 5 · verde · vigente · `go test -race ./pkg/installer -run TestHCD064TTYTimeoutKillsDescendants -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.755s
- 2026-11-20 19:58 -05:00 · tarea 6 · verde · vigente · `go test -race ./pkg/installer -run TestHCD064ExtractExactDirEntries -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.264s
- 2026-11-20 19:58 -05:00 · tarea 7 · verde · vigente · `go test . -run TestHCD064CleanInstallCheckDynamicVersion -count=1` (salida 0)
      ok  	pi-gomemory	0.220s
- 2026-11-20 19:58 -05:00 · tarea 8 · verde · vigente · `go vet ./... && go test -race ./... && bash -n install.sh && bash -n scripts/clean-install-check.sh` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	47.134s

## Siguiente
Cerrada con aprobación sobre la revisión 0e7bb6da5ef0.

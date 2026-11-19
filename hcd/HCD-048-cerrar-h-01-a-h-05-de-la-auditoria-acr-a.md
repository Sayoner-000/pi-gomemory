# Cerrar H-01 a H-05 de la auditoría ACR/ADM v1.15.1

ID: HCD-048
Tipo: fix
Carril: fast
Motivo: Fixes con reproducción clara y sin decisiones abiertas interdependientes

## Intención
Corregir los 5 hallazgos (2 medios, 3 bajos) detectados en la auditoría ACR/ADM sobre v1.15.1 (4ca6835): re-entrancia en openSidebar, éxito falso en playwrightLineParser.finish, marca .uninstalled tras fallo parcial en Uninstall, truncamiento >64 KiB en SanitizeFrontmatter y carrera en /tasks refresh.

## Reproducción
1. **H-01 (`extensions/gomemory-sidebar.ts:815-855`):** Invocar `/sidebar` dos veces mientras `fetchTitle(ctx)` está pendiente (o disparar `session_shutdown` durante el `await`) monta dos overlays `ctx.ui.custom()` y deja el primero huérfano, o abre el panel tras el cierre.
2. **H-02 (`extensions/gomemory-tdd.ts:1029-1037`):** Ejecutar Playwright con un reporte JSON donde los tests registrados están `passed` pero el proceso sale con `res.code !== 0` (p. ej. fallo en `globalTeardown`); `playwrightLineParser.finish` marca la suite `passed` y retorna `false`.
3. **H-03 (`pkg/installer/uninstall.go:1134-1136,1191-1193`):** Ejecutar `Uninstall` cuando una acción falla (`counts[ActionFail] > 0`); la marca `UninstalledMarkerPath` se escribe antes del bucle y permanece en disco aunque `Uninstall` retorne error.
4. **H-04 (`pkg/installer/skills.go:39-72`):** Pasar un contenido con una línea superior a 64 KiB (`bufio.MaxScanTokenSize`) a `SanitizeFrontmatter`; `scanner.Scan()` se detiene sin comprobar `scanner.Err()` y devuelve el contenido truncado.
5. **H-05 (`extensions/gomemory-tasks.ts:57-83`):** Invocar `/tasks` mientras `tool_result` o `agent_end` tiene `refresh(ctx)` en `busy === true`; el toggle se ignora o el refresh en vuelo sobrescribe el widget tras `await pi.exec`.

## Alcance
Incluido: Correcciones mínimas y pruebas de regresión para H-01 (`extensions/gomemory-sidebar.ts`), H-02 (`extensions/gomemory-tdd.ts`), H-03 (`pkg/installer/uninstall.go`), H-04 (`pkg/installer/skills.go`) y H-05 (`extensions/gomemory-tasks.ts`), conforme a Constitución (§1.2 Go, §10 Pruebas, §18 HCD) y Playbook (HCD-EVIDENCE-01, HCD-TDD-01, HCD-TASK-01, HCD-DEF-01).
Excluido: Cambios en CLI/subcomandos públicos, refactors fuera de los cinco puntos indicados y cambios de dependencias.

## ¿Qué pasa si no lo hacemos?
Permanecen dos defectos medios (overlays huérfanos en `/sidebar` y éxito falso en `/tdd` con Playwright cuando `globalTeardown` falla) y tres bajos (estado `.uninstalled` inconsistente tras fallo parcial, truncamiento silencioso >64 KiB en `SanitizeFrontmatter` y carrera en `/tasks`).

## Alternativa más barata
Dejar los hallazgos documentados en la auditoría sin corregir. Es más barato a corto plazo, pero mantiene un éxito falso en `/tdd` y fugas/inconsistencias de estado en UI e instalador.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Blindar re-entrancia y apagado de sesión en `openSidebar` (`extensions/gomemory-sidebar.ts`) → `go test ./pkg/installer -run TestHCD048H01SidebarReentrancy`
- [x] 2. Propagar código de salida distinto de cero en `playwrightLineParser.finish` (`extensions/gomemory-tdd.ts`) → `go test ./pkg/installer -run TestHCD048H02PlaywrightNonZeroExit`
- [x] 3. Evitar persistir `.uninstalled` cuando `Uninstall` falla (`pkg/installer/uninstall.go`) → `go test ./pkg/installer -run TestHCD048H03UninstallMarkerOnFail`
- [x] 4. Manejar líneas mayores a 64 KiB y errores de escaneo en `SanitizeFrontmatter` (`pkg/installer/skills.go`) → `go test ./pkg/installer -run TestHCD048H04SanitizeFrontmatterLongLine`
- [x] 5. Encolar refresh pendiente y re-verificar `visible` en `gomemory-tasks.ts` → `go test ./pkg/installer -run TestHCD048H05TasksConcurrentToggle`
- [x] 6. Verificar suite completa con detector de carreras y análisis estático → `go vet ./... && go test -race ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-18 07:12 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD048H01SidebarReentrancy` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd048_test.go:154:2: undefined: writeManifestFixture
      pkg/installer/hcd048_test.go:154:34: undefined: ManagedItem
      pkg/installer/hcd048_test.go:163:70: not enough arguments in call to Uninstall
      	have (Options, UninstallFlags, *bytes.Buffer)
      	want (fs.FS, *Options, UninstallFlags, io.Writer)
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-18 07:18 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD048H01SidebarReentrancy` (salida -1)
      (tiempo límite agotado)
- 2026-11-18 09:48 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD048H01SidebarReentrancy` (salida 1)
      --- FAIL: TestHCD048H01SidebarReentrancy (0.23s)
          hcd048_test.go:76: TestHCD048H01SidebarReentrancy: FAIL concurrent open mounted 2 overlays (want 1) | open after session_shutdown mounted 1 overlays (want 0)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.449s
      FAIL
- 2026-11-18 13:46 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD048H01SidebarReentrancy` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.958s
- 2026-11-18 15:11 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD048H02PlaywrightNonZeroExit` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.065s
- 2026-11-18 15:41 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD048H03UninstallMarkerOnFail` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.522s
- 2026-11-18 15:44 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD048H04SanitizeFrontmatterLongLine` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.386s
- 2026-11-18 17:01 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run TestHCD048H05TasksConcurrentToggle` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.646s
- 2026-11-18 19:06 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD048H01SidebarReentrancy` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.508s
- 2026-11-18 19:06 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD048H02PlaywrightNonZeroExit` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.385s
- 2026-11-18 19:06 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD048H03UninstallMarkerOnFail` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.155s
- 2026-11-18 19:06 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD048H04SanitizeFrontmatterLongLine` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-18 19:06 -05:00 · tarea 6 · falló · caducada · `go vet ./... && go test -race ./...` (salida 1)
        [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2475080079/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2475080079/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	97.578s
      FAIL
- 2026-11-18 19:10 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestHCD048H01SidebarReentrancy` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.614s
- 2026-11-18 19:10 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD048H02PlaywrightNonZeroExit` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.549s
- 2026-11-18 19:10 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD048H03UninstallMarkerOnFail` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.224s
- 2026-11-18 19:10 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestHCD048H04SanitizeFrontmatterLongLine` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.206s
- 2026-11-18 19:10 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run TestHCD048H05TasksConcurrentToggle` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.430s
- 2026-11-18 19:10 -05:00 · tarea 6 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	95.469s

## Siguiente
Cerrada con aprobación sobre la revisión 6b0a87cc7700.

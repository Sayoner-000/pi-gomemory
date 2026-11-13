# Una release nueva tarda hasta 24 h en anunciarse

ID: HCD-031
Tipo: fix
Carril: fast
Motivo: Vía rápida: fix con reproducción real y una sola decisión ya tomada por la persona (consultar cada hora)

## Intención
Tras publicar v1.13.0, la persona abrió pi-gomemory y no vio el aviso: la caché de versiones (~/.pi/agent/pi-gomemory-deps.json) se había consultado 23 minutos antes y RefreshDeps no vuelve a la red hasta pasadas 24 h. El aviso debe reflejar una release como mucho una hora después de publicarse.

Antecedente: HCD-029 afirmó en las notas de v1.13.0 que «una release publicada desde la última consulta aparece en la misma apertura». Solo es cierto si esa consulta tiene más de 24 h; la prueba de HCD-029 usó una caché fabricada y no cubrió este caso. Decisión de la persona antes de abrir la unidad: «Cada hora (Recomendado)».

## Reproducción
`bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd031-repro.sh`: binario v1.12.1 en un HOME aislado con una caché consultada hace 90 min (más que el TTL decidido de 1 h, menos que el anterior de 24 h) que registra `pi-gomemory latest v1.12.1`. Tras `deps-refresh`, `deps-status` sale con 0 aunque v1.13.0 está publicada: `RefreshDeps` (`pkg/installer/deps.go:251`) no vuelve a la red hasta pasadas 24 h. `PGS_BIN=<binario>` repite la prueba con un binario ya compilado; con el v1.12.1 publicado falla. Caso real del 2026-11-13: caché de las 06:05, release a las 06:28, aviso ausente a las 06:30. Con el TTL de 1 h ese caso se habría anunciado a partir de las 07:05: la decisión acota el retraso a una hora, no lo elimina. La primera versión de la reproducción usaba 30 min, dentro del TTL decidido, y no representaba la decisión; el rojo registrado de la tarea 1 corresponde a esa versión.

## Alcance
Incluido: constante `depsCacheTTL = time.Hour` en `pkg/installer/deps.go`, usada por `RefreshDeps` en lugar de `24*time.Hour`; test de regresión con una caché de más y de menos de una hora; corrección de las frases que prometen 24 h o «la misma apertura» en README, `docs/usage.md`, `docs/installation.md`, `docs/architecture.md`, guía HTML v1.13, `docs/releases/v1.13.0.md` y el comentario de `RefreshDeps`.
Excluido: la caché de `check-update` (`updateCheckTTL`, 24 h), que no alimenta el aviso; consultar en cada apertura; cambiar los probes o el formato de la caché; publicar una release nueva (se decide aparte).

## ¿Qué pasa si no lo hacemos?
Cada release puede tardar hasta un día en anunciarse, justo lo contrario de lo que pidió la persona («cada vez que se abra pi-gomemory mencione que hay una version nueva»), y la documentación de v1.13.0 sigue prometiendo algo que no ocurre.

## Alternativa más barata
Borrar la caché a mano tras cada release (`rm ~/.pi/agent/pi-gomemory-deps.json`). Solo sirve a quien publica, no a quien instala. Se descarta.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla → re-ejecutar la reproducción: `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd031-repro.sh`
- [x] 2. Usar `depsCacheTTL = time.Hour` en `RefreshDeps` (`pkg/installer/deps.go`) → `go test ./pkg/installer -run '^TestHCD031DepsCacheTTL$' -count=1 -v`; observar primero el rojo con `--expect-fail`
- [x] 3. Corregir la promesa de 24 h y de «la misma apertura» en README, `docs/usage.md`, `docs/installation.md`, `docs/architecture.md`, `docs/Guia-PI-GOMEMORY-v1.13.html` y `docs/releases/v1.13.0.md` → `! grep -rnE "misma apertura|respeta la caché de 24 h" README.md docs/usage.md docs/installation.md docs/architecture.md docs/Guia-PI-GOMEMORY-v1.13.html docs/releases/v1.13.0.md` (las entradas ya publicadas de CHANGELOG no se reescriben: la corrección va en la entrada de la versión siguiente)
- [x] 4. Mantener verde la suite completa → `go vet ./... && go test ./... -count=1`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-13 06:32 -05:00 · tarea 1 · rojo observado · caducada · `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd031-repro.sh` (salida 1)
      ✗ caché de hace 30 min: deps-status salió con 0 y no anuncia la release publicada
- 2026-11-13 06:32 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD031DepsCacheTTL$' -count=1 -v` (salida 1)
      === RUN   TestHCD031DepsCacheTTL
          hcd031_deps_ttl_test.go:31: pasada una hora debe volver a la red: llamadas=0, latest=v1.12.1
      --- FAIL: TestHCD031DepsCacheTTL (0.00s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.189s
      FAIL
- 2026-11-13 06:32 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD031DepsCacheTTL$' -count=1 -v` (salida 0)
      === RUN   TestHCD031DepsCacheTTL
      --- PASS: TestHCD031DepsCacheTTL (0.01s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.191s
- 2026-11-13 06:32 -05:00 · tarea 1 · falló · caducada · `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd031-repro.sh` (salida 1)
      ✗ caché de hace 30 min: deps-status salió con 0 y no anuncia la release publicada
- 2026-11-13 06:33 -05:00 · tarea 1 · verde · caducada · `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd031-repro.sh` (salida 0)
        pi-gomemory          v1.12.1 → v1.13.0   [38;2;145;169;197m(pi-gomemory-setup update)[0m
      ✓ la release publicada tras la última consulta se anuncia
- 2026-11-13 06:33 -05:00 · tarea 3 · verde · caducada · `! grep -rnE "misma apertura|respeta la caché de 24 h" README.md docs/usage.md docs/installation.md docs/architecture.md docs/Guia-PI-GOMEMORY-v1.13.html docs/releases/v1.13.0.md` (salida 0)
- 2026-11-13 06:33 -05:00 · tarea 4 · falló · caducada · `go vet ./... && go test ./... -count=1` (salida 1)
      ══════════════════════════════╝[0m
        [38;2;244;198;107m⚠[0m Portapapeles del sistema         [38;2;145;169;197mse requieren pbcopy y pbpaste del sistema[0m
        [38;2;255;119;127m✗[0m Binario pi                       [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean561767430/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean561767430/001/.local/bin/pi-gomemory[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	32.220s
      FAIL
- 2026-11-13 06:35 -05:00 · tarea 1 · verde · vigente · `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd031-repro.sh` (salida 0)
        pi-gomemory          v1.12.1 → v1.13.0   [38;2;145;169;197m(pi-gomemory-setup update)[0m
      ✓ la release publicada tras la última consulta se anuncia
- 2026-11-13 06:35 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD031DepsCacheTTL$' -count=1 -v` (salida 0)
      === RUN   TestHCD031DepsCacheTTL
      --- PASS: TestHCD031DepsCacheTTL (0.01s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.198s
- 2026-11-13 06:35 -05:00 · tarea 4 · verde · vigente · `go vet ./... && go test ./... -count=1` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	31.374s
- 2026-11-13 06:35 -05:00 · tarea 3 · verde · vigente · `! grep -rnE "misma apertura|respeta la caché de 24 h" README.md docs/usage.md docs/installation.md docs/architecture.md docs/Guia-PI-GOMEMORY-v1.13.html docs/releases/v1.13.0.md` (salida 0)

## Siguiente
Cerrada con aprobación sobre la revisión b690eb37dc69.

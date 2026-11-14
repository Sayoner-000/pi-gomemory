# PI_GOMEMORY_NO_UPDATE_CHECK significa lo mismo en Go, el launcher y el pie de Pi

ID: HCD-037
Tipo: fix
Carril: fast
Motivo: Vía rápida: inconsistencia acotada a cuatro consumidores de una variable, con dos decisiones independientes y sin riesgo sobre datos

## Intención
El binario, el launcher y el pie de Pi interpretan PI_GOMEMORY_NO_UPDATE_CHECK con la misma regla, y deps-refresh se comporta de forma coherente con ella; update explícito sigue consultando siempre (HCD-035)

Origen: pendiente de HCD-035 (Q10), anotado en `docs/known-issues.md` de v1.13.5 y v1.13.6. Antecedentes: HCD-001 (la variable y `check-update`), HCD-004 (el launcher la respeta), HCD-029 (el pie la respeta) y HCD-035 (`update` pedido explícitamente consulta siempre).

## Reproducción
Sobre v1.13.6 (`466a7d5`). Interpretación de cada valor en el launcher (`pkg/installer/harness.go:45`, `[ -z "${PI_GOMEMORY_NO_UPDATE_CHECK:-}" ]`) y en el pie (`extensions/gomemory-status.ts:36`, `if (process.env.PI_GOMEMORY_NO_UPDATE_CHECK) return`), evaluada el 2026-11-14 con bash y node:

```bash
for v in "" 0 false FALSE 1 yes; do
  PI_GOMEMORY_NO_UPDATE_CHECK="$v" bash -c '[ -z "${PI_GOMEMORY_NO_UPDATE_CHECK:-}" ] && echo comprueba || echo desactivada'
  PI_GOMEMORY_NO_UPDATE_CHECK="$v" node -e 'console.log(process.env.PI_GOMEMORY_NO_UPDATE_CHECK ? "desactivada" : "comprueba")'
done
```

Resultado: con `0`, `false` y `FALSE`, el launcher y el pie dicen «desactivada». En cambio, Go (`UpdateCheckDisabled`, `pkg/installer/update.go:111-114`) solo desactiva si el valor, sin espacios, no está vacío y no es `0` ni `false` (sin distinguir mayúsculas), así que `check-update` con esos valores **comprueba**. `deps-refresh` (`RefreshDeps`, `pkg/installer/deps.go:254`) no consulta la variable y va a la red con cualquier valor. Hoy solo lo lanzan el launcher y el pie, que filtran antes con su propia regla.

Verificación del cierre: sobre el árbol de trabajo. Las pruebas de regresión se añaden primero y su rojo, también el de la tarea 1, se registra con `mem hcd verify --expect-fail` antes de tocar producción.

## Alcance
Incluido: la regla de Go (Q1) aplicada en el launcher y en el pie, y `deps-refresh` sin red cuando la variable desactiva la comprobación (Q2). Detalle:
- El launcher generado por `GenerateHarnessScript` (`harness.go`) usa una función de shell `update_check_disabled`, compatible con el bash 3.2 de macOS:
  - quita los espacios de los extremos y pasa el valor a minúsculas;
  - desactiva si el resultado no está vacío y no es `0` ni `false`.
- `gomemory-status.ts` exporta `updateCheckDisabled(env)` con la misma regla, y `checkUpdates` la usa.
- `RefreshDeps` (`deps.go`): si `UpdateCheckDisabled()` es verdadero, no consulta ninguna sonda ni escribe la caché, y devuelve el contenido de la caché tal como esté (vacío si no existe).
- Una prueba que comprueba la regla en Go con la misma tabla de valores.

Excluido: `update` y `deps-update`, otras variables y opciones del launcher, avisos ya en caché, y release/commit/push. Detalle:
- `update` y `deps-update` siguen consultando siempre (HCD-035). `deps-update` no llama a `RefreshDeps`.
- Cambiar el nombre de la variable o añadir otras (`=true`, `=on`…): cualquier valor que no sea vacío, `0` o `false` desactiva, como en Go.
- El aviso que ya está en la caché de `deps-status`: con la variable activa, el launcher y el pie no lo consultan, igual que hoy.
- Release, commit y push, que requieren autorización aparte.

## ¿Qué pasa si no lo hacemos?
- Quien escribe `PI_GOMEMORY_NO_UPDATE_CHECK=0` o `=false` esperando dejar la comprobación activa la desactiva en el launcher y en el pie sin darse cuenta, y deja de ver releases nuevas. `check-update`, en cambio, sí comprueba con esos valores: el mismo ajuste hace cosas distintas según el componente.
- `deps-refresh` ejecutado por otra vía (un alias o un cron) consulta la red aunque la persona haya desactivado las comprobaciones.

## Alternativa más barata
Documentar que basta con que la variable tenga cualquier valor (cambiando solo Go) o dejar la diferencia anotada en problemas conocidos. Se descarta: la persona eligió la regla de Go (Q1), que sigue la convención habitual de `=0` para «no desactivar», y documentarlo no evita el error. El cambio es pequeño: una función en el launcher, otra en el pie y una condición en `RefreshDeps`.

## Historias
### US1. Una sola regla
Como persona que configura PI_GOMEMORY_NO_UPDATE_CHECK, quiero que el mismo valor tenga el mismo efecto en el binario, el launcher, el pie y `deps-refresh`, para controlar las comprobaciones automáticas sin sorpresas.
- C-01: Dados los valores vacío, ` `, `0`, ` 0 `, `false`, `FALSE` y `False`, cuando se evalúa la regla en Go, el launcher y el pie, entonces los tres dejan la comprobación **activa**. Dados `1`, `yes`, `true` y `no`, los tres la **desactivan**.
- C-02: Dado `PI_GOMEMORY_NO_UPDATE_CHECK=0`, cuando se abre una sesión de Pi, entonces el pie ejecuta `pi-gomemory-setup deps-status`. Dado `=1`, no lo ejecuta.
- C-03: Dada la variable con un valor que desactiva la comprobación, cuando se ejecuta `RefreshDeps`, entonces no se llama a ninguna sonda, la caché no se modifica y se devuelve su contenido. Con un valor que la deja activa, consulta como hoy.
- C-04: Dado el launcher generado, cuando se inspecciona, entonces la condición del bloque de actualizaciones usa `update_check_disabled` y no `[ -z "${PI_GOMEMORY_NO_UPDATE_CHECK:-}" ]` (sigue pasando `TestHarnessDepsCheck`).
- C-05: Dado `PI_GOMEMORY_NO_UPDATE_CHECK=1`, cuando se ejecuta `update`, entonces sigue consultando (`TestHCD035C13`, `TestHCD035C14`; guarda sin rojo).

## Aclaraciones
- [x] Q1. ¿Qué valores de PI_GOMEMORY_NO_UPDATE_CHECK desactivan la comprobación en Go, el launcher y el pie? → La de Go (Recomendado): vacía, «0» o «false» (sin distinguir mayúsculas, ignorando espacios) dejan la comprobación activa; cualquier otro valor la desactiva. Cambian el launcher y el pie.
- [x] Q2. ¿Debe deps-refresh respetar PI_GOMEMORY_NO_UPDATE_CHECK? → Sí, sin red (Recomendado): con la variable activa, deps-refresh no consulta la red y deja la caché como está; update y deps-update siguen consultando.

## Checklist
- [x] 1. La reproducción ya no falla → go test ./... -count=1 && go test -race ./... -count=1
- [x] 2. Aplicar la regla de Go en el launcher con update_check_disabled (pkg/installer/harness.go) (C-01, C-04) → go test ./pkg/installer -run '^(TestHCD037Launcher|TestHarnessDepsCheck)$' -count=1
- [x] 3. Aplicar la regla de Go en el pie con updateCheckDisabled (extensions/gomemory-status.ts) (C-01, C-02) → go test ./pkg/installer -run '^TestHCD037Footer$' -count=1
- [x] 4. Omitir la red y la escritura de caché en RefreshDeps con la comprobación desactivada (pkg/installer/deps.go) (C-03) → go test ./pkg/installer -run '^TestHCD037DepsRefresh$' -count=1
- [x] 5. Conservar la regla de Go como referencia (pkg/installer/update.go, guarda sin rojo) (C-01) → go test ./pkg/installer -run '^(TestHCD037GoRule|TestCheckUpdateDisabled)$' -count=1
- [x] 6. Conservar que update consulta siempre (main.go, guarda sin rojo) (C-05) → go test . -run '^TestHCD035C1[34]$' -count=1

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-14 07:12 -05:00 · tarea 1 · rojo observado · caducada · `go test ./... -count=1 && go test -race ./... -count=1` (salida 1)
      ═════════════════════════════╝[0m
        [38;2;244;198;107m⚠[0m Portapapeles del sistema         [38;2;145;169;197mse requieren pbcopy y pbpaste del sistema[0m
        [38;2;255;119;127m✗[0m Binario pi                       [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean1336254361/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean1336254361/001/.local/bin/pi-gomemory[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	52.799s
      FAIL
- 2026-11-14 07:13 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^(TestHCD037Launcher|TestHarnessDepsCheck)$' -count=1` (salida 1)
      --- FAIL: TestHCD037Launcher (0.00s)
          hcd037_update_check_test.go:42: C-04: el launcher conserva la condición [ -z ], que desactiva con =0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.172s
      FAIL
- 2026-11-14 07:13 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD037Footer$' -count=1` (salida 1)
      --- FAIL: TestHCD037Footer (0.12s)
          hcd037_update_check_test.go:74: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD037Footer672845154/001/test.mjs:3
              import status, { updateCheckDisabled } from "./gomemory-status.ts";
                               ^^^^^^^^^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-status.ts' does not provide an export named 'updateCheckDisabled'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.264s
      FAIL
- 2026-11-14 07:13 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD037DepsRefresh$' -count=1` (salida 1)
      --- FAIL: TestHCD037DepsRefresh (0.01s)
          hcd037_update_check_test.go:121: C-03: con la comprobación desactivada se consultó la red 1 veces
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.146s
      FAIL
- 2026-11-14 07:13 -05:00 · tarea 1 · falló · caducada · `go test ./... -count=1 && go test -race ./... -count=1` (salida 1)
      ═════════════════════════════╝[0m
        [38;2;244;198;107m⚠[0m Portapapeles del sistema         [38;2;145;169;197mse requieren pbcopy y pbpaste del sistema[0m
        [38;2;255;119;127m✗[0m Binario pi                       [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2247756226/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2247756226/001/.local/bin/pi-gomemory[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	52.872s
      FAIL
- 2026-11-14 07:14 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^(TestHCD037Launcher|TestHarnessDepsCheck)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.210s
- 2026-11-14 07:14 -05:00 · tarea 3 · falló · caducada · `go test ./pkg/installer -run '^TestHCD037Footer$' -count=1` (salida 1)
              + false
              - undefined
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD037Footer3804744782/001/test.mjs:7:9
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5) {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: false,
                expected: undefined,
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.262s
      FAIL
- 2026-11-14 07:14 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD037DepsRefresh$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.147s
- 2026-11-14 07:14 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^(TestHCD037GoRule|TestCheckUpdateDisabled)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.141s
- 2026-11-14 07:14 -05:00 · tarea 6 · verde · caducada · `go test . -run '^TestHCD035C1[34]$' -count=1` (salida 0)
      ok  	pi-gomemory	0.132s
- 2026-11-14 07:15 -05:00 · tarea 1 · verde · vigente · `go test ./... -count=1 && go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	0.170s
      ok  	pi-gomemory/pkg/installer	54.828s
      ok  	pi-gomemory	1.228s
      ok  	pi-gomemory/pkg/installer	56.063s
- 2026-11-14 07:17 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^(TestHCD037Launcher|TestHarnessDepsCheck)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.228s
- 2026-11-14 07:17 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD037Footer$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.378s
- 2026-11-14 07:17 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD037DepsRefresh$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.147s
- 2026-11-14 07:17 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^(TestHCD037GoRule|TestCheckUpdateDisabled)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.142s
- 2026-11-14 07:17 -05:00 · tarea 6 · verde · vigente · `go test . -run '^TestHCD035C1[34]$' -count=1` (salida 0)
      ok  	pi-gomemory	0.136s

## Siguiente
Cerrada con aprobación sobre la revisión 6dae78262cae.

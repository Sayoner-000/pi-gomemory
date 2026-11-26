# Retirar /tdd y la integración Antigravity

ID: HCD-085
Tipo: evolution
Carril: fast
Motivo: Alcance ya decidido por la persona en el plan; sin decisiones abiertas interdependientes

## Intención
Retirar el comando /tdd (no da la funcionalidad necesaria) y toda la integración Antigravity/Agy (cuota en sidebar, comando usage/quota, plugin pi-antigravity, antigravity-prompt-guard.js) por riesgo de bloqueo de cuentas Google; install/update retiran las copias intactas ya instaladas y conservan con aviso las editadas.


## Alcance
Incluido: borrar `extensions/gomemory-tdd.ts` y su acoplamiento en el sidebar (`PanelDock.tdd`); quitar del sidebar la sección «Google AI Pro Quota» y su sondeo de `pi-gomemory-setup usage --json`; borrar `pkg/installer/usage.go` y el comando `usage`/`quota`/`antigravity.usage`; dejar de instalar y fijar `pi-antigravity` (plugin.go, pins.go, contratos de pins_candidate.go) y retirar su entrada de `settings.json`; borrar `extensions/antigravity-prompt-guard.js` y sus tests; añadir `retiredResources` para que install borre las copias intactas de `gomemory-tdd.ts` y `antigravity-prompt-guard.js` (según el hash del manifiesto) y conserve con aviso las editadas; README y CHANGELOG.
Excluido: borrar `~/.pi/agent/antigravity-accounts.json` o revocar tokens (son credenciales de la persona: solo se avisa); desinstalar `node_modules/pi-antigravity` por npm; los usos de cuota de Anthropic y Codex; publicar una release.

Contradice a propósito HCD-014, HCD-015, HCD-018, HCD-038, HCD-040 y el pin de HCD-005: el riesgo de bloqueo de cuentas pesa más que la función.

## ¿Qué pasa si no lo hacemos?
El sidebar sigue haciendo POST a `v1internal:retrieveUserQuotaSummary` con un User-Agent que imita `antigravity/cli` y, junto con el proveedor pi-antigravity, expone a la persona a que Google bloquee u observe su cuenta. /tdd sigue cargándose en cada sesión de Pi (~2.700 líneas y un listener de Bash) sin aportar lo que se necesita.

## Alternativa más barata
Apagar solo la sección de cuota con una variable de entorno. Se descarta porque el plugin y el prompt-guard siguen usando la API no oficial, y porque las instalaciones existentes conservarían los archivos. Desactivar /tdd sin borrarlo deja código muerto embebido.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Retirar gomemory-tdd.ts del repo y de tuiResources → `test ! -e extensions/gomemory-tdd.ts && go test ./pkg/installer -run '^TestHCD085TDDRetired$'`
- [x] 2. Quitar del sidebar la cuota Antigravity y el acoplamiento tdd → `go test ./pkg/installer -run '^TestHCD085SidebarNoAntigravity$'`
- [x] 3. Borrar usage.go y el comando usage/quota del CLI → `go test . -run '^TestHCD085UsageCommandRemoved$'`
- [x] 4. Dejar de instalar y fijar pi-antigravity y retirar su entrada de settings.json → `go test ./pkg/installer -run '^TestHCD085RetireAntigravityPackage$'`
- [x] 5. Retirar antigravity-prompt-guard.js y sus tests → `go test ./pkg/installer -run '^TestHCD085PromptGuardRetired$'`
- [x] 6. Borrar en install las copias retiradas intactas y conservar las editadas → `go test ./pkg/installer -run '^TestHCD085RetiredResources$'`
- [x] 7. Actualizar README y CHANGELOG → `go test ./pkg/installer -run '^TestHCD085DocsUpdated$'`
- [x] 8. Regresión completa → `go vet ./... && go test ./... && node --test tests/*.test.mjs`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-26 17:12 -05:00 · tarea 1 · verde · vigente · `test ! -e extensions/gomemory-tdd.ts && go test -count=1 ./pkg/installer -run '^TestHCD085TDDRetired$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.022s
- 2026-11-26 17:12 -05:00 · tarea 2 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD085SidebarNoAntigravity$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.095s
- 2026-11-26 17:12 -05:00 · tarea 3 · verde · vigente · `test ! -e pkg/installer/usage.go && go test -count=1 . -run '^TestHCD085UsageCommandRemoved$' && go test -count=1 ./pkg/installer -run '^TestHCD085LauncherNoUsage$'` (salida 0)
      ok  	pi-gomemory	0.003s
      ok  	pi-gomemory/pkg/installer	0.003s
- 2026-11-26 17:13 -05:00 · tarea 4 · verde · vigente · `go test -count=1 ./pkg/installer -run '^(TestHCD085RetireAntigravityPackage|TestHCD040HistoricPins|TestHCD039Rejected)$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.342s
- 2026-11-26 17:13 -05:00 · tarea 5 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD085PromptGuardRetired$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.003s
- 2026-11-26 17:13 -05:00 · tarea 6 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD085RetiredResources$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.013s
- 2026-11-26 17:13 -05:00 · tarea 7 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD085DocsUpdated$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.003s
- 2026-11-26 17:13 -05:00 · tarea 8 · verde · vigente · `go vet ./... && go test -count=1 ./... && node --test tests/*.test.mjs` (salida 0)
      ok 17 - renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)
        ---
        duration_ms: 0.615108
        type: 'test'
        ...
      # Subtest: HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
      ok 18 - HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
        ---
        duration_ms: 216.019974
        type: 'test'
        ...
      1..18
      # tests 18
      # suites 0
      # pass 18
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 354.555999

## Siguiente
Cerrada con aprobación sobre la revisión 0495f329ec42.

# Corregir hallazgos H-01 a H-04 y H-06 a H-08 del ACR/ADM de v1.29.2

ID: HCD-083
Tipo: fix
Carril: fast
Motivo: Siete fixes acotados con causa raíz y reproducción determinista del ACR/ADM sobre v1.29.2 (acr_0eb943a5), sin decisiones abiertas interdependientes

## Intención
Cerrar el hallazgo confirmado H-01 y los seis hallazgos verificados accionables (H-02, H-03, H-04, H-06, H-07 y H-08) de la auditoría ACR/ADM de v1.29.2 con pruebas de regresión, respetando la política de HCD-076 C-06 para H-05.

## Reproducción
Los siete hallazgos accionables detectados en la revisión ACR `acr_0eb943a5-6de2-487f-aded-70d456b0a85e` y ADM sobre `v1.29.2` (`b113063`) se reproducen de forma determinista con pruebas unitarias en `pkg/installer/hcd083_test.go` y `tests/gomemory-context.test.mjs`:
1. **H-01 (`scripts/clean-install-check.sh:10,49,111`):** el script sigue exigiendo `codebase-memory-mcp` aunque fue retirado del catálogo en `9e7a60d`, por lo que siempre falla tras una instalación limpia válida.
2. **H-02 (`extensions/gomemory-context.ts:138-140`):** `agent_end` invoca `mem code-refresh` incluso cuando `globalThis[Symbol.for("pi-gomemory.plan")].on === true`, violando el contrato de solo lectura de `/plan`.
3. **H-03 (`pkg/installer/sync.go:228-233`):** `Sync` evalúa `states[byState[t.Name]]` antes de comprobar `_, ok := byState[t.Name]`, provocando `panic: index out of range [0]` si `states` está vacío y lectura de `states[0]` si falta la clave.
4. **H-04 (`pkg/installer/plugin.go:192,232`):** `InstallAntigravityPlugin` e `InstallCCExtensions` invocan `exec.Command(piBin, "install", ...).CombinedOutput()` sin `CommandContext`/timeout ni stdin aislado, bloqueándose indefinidamente si `pi install` se cuelga.
5. **H-06 (`pkg/installer/sync.go:183`, `pkg/installer/tools.go:162`):** `Sync` y `EnsureTools` usan `IsLiteMode` (que descarta errores devolviendo `false`) en lugar de `GetLiteStatus`, ignorando silenciosamente un `lite.json` corrupto.
6. **H-07 (`extensions/gomemory-context.ts:84,129-152`):** un fallo transitorio o timeout de `mem context` en `session_start` queda cacheado como `{ error }` en `pending` durante toda la sesión sin reintento en turnos posteriores.
7. **H-08 (`scripts/build-release.sh:8`):** `VERSION="${1:-v1.26.1}"` empaqueta con versión obsoleta `v1.26.1` si se ejecuta directamente sin argumento y no valida el formato de versión.

Comprobación de reproducción (debe fallar antes del cambio):
`go test ./pkg/installer -run 'TestHCD083' && node --test tests/gomemory-context.test.mjs`

## Alcance
Incluido: corrección de H-01, H-02, H-03, H-04, H-06, H-07 y H-08 del ACR/ADM de v1.29.2, más pruebas unitarias dedicadas y regresión completa:
- **H-01 (C-001 CONFIRMED MEDIUM):** retirar `codebase-memory-mcp` de `scripts/clean-install-check.sh` (comentario línea 10, bucle limpio línea 49 y verificación línea 111) y añadir prueba que verifique paridad con `Catalog()`.
- **H-02 (C-008 MEDIUM):** omitir `triggerAutoRefresh` en `agent_end` de `extensions/gomemory-context.ts` cuando `globalThis[Symbol.for("pi-gomemory.plan")]?.on === true`.
- **H-03 (C-004 LOW):** en `pkg/installer/sync.go:228-233`, comprobar `if i, ok := byState[t.Name]; ok` antes de indexar `states[i]`.
- **H-04 (C-005 LOW):** en `pkg/installer/plugin.go`, ejecutar `pi install` mediante helper con `exec.CommandContext` + timeout configurable (`pluginInstallTimeout = 2 * time.Minute`) y `cmd.Stdin = nil`.
- **H-06 (C-006 LOW):** en `pkg/installer/sync.go` y `pkg/installer/tools.go`, leer el modo liviano con `GetLiteStatus(opts.HomeDir)` y propagar el error si `lite.json` es inválido o ilegible.
- **H-07 (C-007 LOW):** en `extensions/gomemory-context.ts`, no retener permanentemente `{ error }` en `pending` (limpiar `pending = undefined` tras un fallo en `before_agent_start`) para permitir recuperación en el siguiente turno.
- **H-08 (C-002 LOW):** en `scripts/build-release.sh`, sustituir el fallback hardcodeado `v1.26.1` por `git describe --tags --exact-match 2>/dev/null || true` y validar `^v[0-9]+\.[0-9]+\.[0-9]+([.+-][0-9A-Za-z.+-]*)?$` antes de compilar.
- Pruebas dedicadas en `pkg/installer/hcd083_test.go` y `tests/gomemory-context.test.mjs`, más regresión completa (`go vet ./...`, `go test -race ./...`, `node --test tests/*.test.mjs`, `bash -n install.sh scripts/*.sh`).

Excluido: H-05 (política explícita de HCD-076 C-06), llamadores informativos de `IsLiteMode` y publicación:
- **H-05 (C-003 LOW — `pkg/installer/sync.go` exit 0 al fallar `updateTool`):** comportamiento explícitamente decidido y fijado por contrato en HCD-076 C-06 (`TestHCD076SyncFailurePolicy`: solo es fatal no poder instalar un obligatorio que falta). No se cambia para no romper esa decisión ni la suite de HCD-076.
- Cambios en `IsLiteMode` para llamadores informativos (`doctor`, banner) que no deben abortar por `lite.json`.
- Commit, push o publicación de release.

Antecedentes: ACR `acr_0eb943a5-6de2-487f-aded-70d456b0a85e` / ADM v1.29.2 (memoria #640), HCD-076 (catálogo único y política de fallos C-06), HCD-067 (estado compartido `PLAN_STATE_KEY` en modo plan), commit `9e7a60d` (retiro de `codebase-memory-mcp`).

## ¿Qué pasa si no lo hacemos?
- `scripts/clean-install-check.sh` falla de forma determinista sobre cualquier instalación limpia válida, dejando rota la verificación real del instalador.
- El modo `/plan` puede crear `.codegraph/` o escribir índices en segundo plano al terminar un turno, rompiendo su garantía de solo lectura.
- `Sync` mantiene un pánico latente al indexar `states[0]` cuando falta una sonda, `pi install` puede colgar `install`/`plugin` indefinidamente ante fallos de red, un `lite.json` corrupto activa silenciosamente instalaciones opcionales, un timeout transitorio de `mem context` deja toda la sesión sin memoria y `build-release.sh` directo empaqueta artefactos con etiqueta obsoleta `v1.26.1`.

## Alternativa más barata
Corregir únicamente el hallazgo confirmado por consenso ACR (`H-01`) y diferir los otros seis. Se descarta porque los seis hallazgos adicionales fueron confirmados directamente en el código fuente, son cambios mínimos y localizados, y se prueban de forma determinista sin ampliar el alcance funcional.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla (H-01..H-04 y H-06..H-08 reproducidos en rojo y corregidos en verde) → `go test ./pkg/installer -run 'TestHCD083' && node --test tests/gomemory-context.test.mjs`
- [x] 2. H-01: `scripts/clean-install-check.sh` no exige `codebase-memory-mcp` y coincide con los tools de `Catalog()` → `go test ./pkg/installer -run 'TestHCD083CleanInstallCheckMatchesCatalog'`
- [x] 3. H-02: `gomemory-context.ts` no ejecuta `mem code-refresh` en `agent_end` mientras `/plan` está activo → `node --test tests/gomemory-context.test.mjs --test-name-pattern 'H-02'`
- [x] 4. H-03: `Sync` no entra en pánico ni lee `states[0]` si falta una entrada en `byState` → `go test ./pkg/installer -run 'TestHCD083SyncMissingProbeNoPanic'`
- [x] 5. H-04: `InstallAntigravityPlugin` e `InstallCCExtensions` cancelan `pi install` al vencer el timeout → `go test ./pkg/installer -run 'TestHCD083PluginInstallTimeout'`
- [x] 6. H-06: `Sync` y `EnsureTools` devuelven error ante un `lite.json` corrupto en lugar de asumir lite=false → `go test ./pkg/installer -run 'TestHCD083CorruptLiteJSONFails'`
- [x] 7. H-07: `gomemory-context.ts` reintenta `mem context` en el siguiente turno tras un fallo transitorio → `node --test tests/gomemory-context.test.mjs --test-name-pattern 'H-07'`
- [x] 8. H-08: `scripts/build-release.sh` no tiene versión obsoleta hardcodeada y rechaza versiones inválidas → `go test ./pkg/installer -run 'TestHCD083BuildReleaseVersionGuard'`
- [x] 9. Regresión completa (Go vet/test -race, Node tests y sintaxis bash) en verde → `go vet ./... && go test -race ./... && node --test tests/*.test.mjs && bash -n install.sh scripts/*.sh`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-24 20:03 -05:00 · tarea 1 · rojo observado · caducada · `sh -c go test ./pkg/installer -run 'TestHCD083' && node --test tests/gomemory-context.test.mjs` (salida 2)
      	buildjson       build -json encoding
      	buildmode       build modes
      	c               calling between Go and C
      	cache           build and test caching
      	environment     environment variables
      	filetype        file types
      	goauth          GOAUTH environment variable
      	go.mod          the go.mod file
      	gopath          GOPATH environment variable
      	goproxy         module proxy protocol
      	importpath      import path syntax
      	modules         modules, module versions, and more
      	module-auth     module authentication using go.sum
      	packages        package lists and patterns
      	private         configuration for downloading non-public code
      	testflag        testing flags
      	testfunc        testing functions
      	vcs             controlling version control with GOVCS
      Use "go help <topic>" for more information about that topic.
- 2026-11-24 20:04 -05:00 · tarea 1 · falló · caducada · `sh -c go test ./pkg/installer -run 'TestHCD083' && node --test tests/gomemory-context.test.mjs` (salida 2)
      	buildjson       build -json encoding
      	buildmode       build modes
      	c               calling between Go and C
      	cache           build and test caching
      	environment     environment variables
      	filetype        file types
      	goauth          GOAUTH environment variable
      	go.mod          the go.mod file
      	gopath          GOPATH environment variable
      	goproxy         module proxy protocol
      	importpath      import path syntax
      	modules         modules, module versions, and more
      	module-auth     module authentication using go.sum
      	packages        package lists and patterns
      	private         configuration for downloading non-public code
      	testflag        testing flags
      	testfunc        testing functions
      	vcs             controlling version control with GOVCS
      Use "go help <topic>" for more information about that topic.
- 2026-11-24 20:04 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run 'TestHCD083' && node --test tests/gomemory-context.test.mjs` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.547s
      ✔ H-02: agent_end no ejecuta mem code-refresh mientras /plan está activo (0.560708ms)
      ✔ H-07: before_agent_start reintenta mem context tras un fallo transitorio (0.189666ms)
      ℹ tests 2
      ℹ suites 0
      ℹ pass 2
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 125.976917
- 2026-11-24 20:04 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run 'TestHCD083CleanInstallCheckMatchesCatalog'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.176s
- 2026-11-24 20:04 -05:00 · tarea 3 · verde · vigente · `node --test tests/gomemory-context.test.mjs --test-name-pattern 'H-02'` (salida 0)
      ✔ H-02: agent_end no ejecuta mem code-refresh mientras /plan está activo (0.669583ms)
      ✔ H-07: before_agent_start reintenta mem context tras un fallo transitorio (0.201708ms)
      ℹ tests 2
      ℹ suites 0
      ℹ pass 2
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 123.136833
- 2026-11-24 20:04 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run 'TestHCD083SyncMissingProbeNoPanic'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.174s
- 2026-11-24 20:04 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run 'TestHCD083PluginInstallTimeout'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.501s
- 2026-11-24 20:05 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run 'TestHCD083CorruptLiteJSONFails'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.181s
- 2026-11-24 20:05 -05:00 · tarea 7 · verde · vigente · `node --test tests/gomemory-context.test.mjs --test-name-pattern 'H-07'` (salida 0)
      ✔ H-02: agent_end no ejecuta mem code-refresh mientras /plan está activo (0.613459ms)
      ✔ H-07: before_agent_start reintenta mem context tras un fallo transitorio (0.1895ms)
      ℹ tests 2
      ℹ suites 0
      ℹ pass 2
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 127.68875
- 2026-11-24 20:05 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run 'TestHCD083BuildReleaseVersionGuard'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.183s
- 2026-11-24 20:05 -05:00 · tarea 9 · verde · vigente · `go vet ./... && go test -race ./... && node --test tests/*.test.mjs && bash -n install.sh scripts/*.sh` (salida 0)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.205625ms)
      ✔ modo plan permite lecturas de codegraph y bloquea mutaciones HCD o de índice (0.931333ms)
      ✔ planShortcutKeys: Alt+P y Ctrl+Alt+P; en Windows y WSL solo Ctrl+Alt+P (0.168167ms)
      ✔ Alt+P alterna el modo plan, bloquea y publica el estado para el header (0.315416ms)
      ✔ resolveActiveTaskId: no salta atrás a tareas stale/unverified mientras haya pending (0.37075ms)
      ✔ resolveActiveTaskId: sin pending, prioriza failed antes que stale (0.057333ms)
      ✔ resolveActiveTaskId: cuando no quedan pending ni failed, enfoca la primera stale (0.045625ms)
      ✔ HCD-079 stale en implement: se pinta como progreso, sin warning ni ↻ (0.321083ms)
      ✔ HCD-079 stale en verify: mantiene ↻ warning (0.07025ms)
      ✔ HCD-079 stale en close: mantiene ↻ warning (0.047125ms)
      ✔ renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim) (0.28775ms)
      ✔ HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida (558.765417ms)
      ℹ tests 18
      ℹ suites 0
      ℹ pass 18
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 724.253166

## Siguiente
Cerrada con aprobación sobre la revisión 5f3c4d167c8f.

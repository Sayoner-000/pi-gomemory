# timeout propio en httpJSON y -race al empaquetar

ID: HCD-033
Tipo: technical
Carril: fast
Motivo: Vía rápida: dos endurecimientos acotados (C-002 y C-003 de la auditoría v1.12.1) sin decisiones abiertas; la persona confirmó la unidad

## Intención
Cerrar C-002 y C-003 de la auditoría ACR/ADM de v1.12.1. C-002: httpJSON (pkg/installer/deps.go) usa http.DefaultClient y solo el contexto limita la espera; un llamador sin plazo podría colgarse. Debe usar un cliente con timeout propio de 20 s (depsProbeTimeout). C-003: scripts/build-release.sh ejecuta go test sin -race; el empaquetado debe ejecutar go test -race ./... como el Makefile y las verificaciones.


Antecedentes: la [auditoría ACR/ADM de v1.12.1](../docs/audits/2026-11-13-acr-adm-v1.12.1.md) clasificó C-002 como sospecha baja y C-003 como observación informativa. Hoy C-002 no puede ocurrir porque el único llamador (`npmLatest` dentro de `RefreshDeps`) pasa un contexto con plazo de 20 s, así que es una brecha latente. Otros clientes HTTP del proyecto ya tienen timeout propio (`pkg/installer/anthropic_usage.go`, `pkg/installer/codex_usage.go` y `fetchLatestTag` en `pkg/installer/update.go`).

## Alcance
Incluido: `httpJSON` (`pkg/installer/deps.go`) usa un cliente de paquete `depsHTTPClient` con `Timeout: depsProbeTimeout` (20 s), y el contexto sigue acotando la espera; test de regresión que comprueba el timeout configurado y que `httpJSON` usa ese cliente; `scripts/build-release.sh` ejecuta `go test -race ./...` antes de compilar; `docs/known-issues.md` marca C-002 y C-003 como corregidos.
Excluido: cambiar el plazo de 20 s; los demás clientes HTTP del proyecto; añadir `-race` a otros scripts; publicar una release (se decide aparte).

## ¿Qué pasa si no lo hacemos?
Un llamador futuro de `httpJSON` sin plazo en el contexto podría dejar colgada una consulta de versiones ante una conexión que no responde. Y una release podría empaquetarse con una carrera de datos que `go test` sin `-race` no detecta.

## Alternativa más barata
Documentar que todo llamador de `httpJSON` debe pasar un contexto con plazo y ejecutar `-race` a mano antes de cada release. Depende de que se recuerde siempre, y el cambio de código cuesta lo mismo. Se descarta.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Dar a `httpJSON` (`pkg/installer/deps.go`) un cliente con timeout propio `depsHTTPClient` (`Timeout: depsProbeTimeout`) → `go test ./pkg/installer -run '^TestHCD033HTTPJSONClientTimeout$' -count=1 -v`; observar primero el rojo con `--expect-fail` tras introducir el punto de inyección sin cambiar el comportamiento
- [x] 2. Ejecutar `go test -race ./...` en `scripts/build-release.sh` → `bash -n scripts/build-release.sh && grep -qE "^go test -race \./\.\.\.$" scripts/build-release.sh && ! grep -qE "^go test \./\.\.\.$" scripts/build-release.sh`; observar primero el rojo con `--expect-fail`
- [x] 3. Marcar C-002 y C-003 como corregidos en `docs/known-issues.md` → `grep -E "^\| C-002 " docs/known-issues.md | grep -q Corregido && grep -E "^\| C-003 " docs/known-issues.md | grep -q Corregido`
- [x] 4. Mantener verde la suite completa → `go vet ./... && go test -race ./... -count=1`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-13 07:05 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD033HTTPJSONClientTimeout$' -count=1 -v` (salida 1)
      === RUN   TestHCD033HTTPJSONClientTimeout
          hcd033_http_timeout_test.go:16: timeout del cliente de httpJSON = 0s; quiero 20s
      --- FAIL: TestHCD033HTTPJSONClientTimeout (0.00s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.391s
      FAIL
- 2026-11-13 07:05 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD033HTTPJSONClientTimeout$' -count=1 -v` (salida 0)
      === RUN   TestHCD033HTTPJSONClientTimeout
      --- PASS: TestHCD033HTTPJSONClientTimeout (0.20s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.396s
- 2026-11-13 07:05 -05:00 · tarea 2 · rojo observado · caducada · `bash -n scripts/build-release.sh && grep -qE "^go test -race \./\.\.\.$" scripts/build-release.sh && ! grep -qE "^go test \./\.\.\.$" scripts/build-release.sh` (salida 1)
- 2026-11-13 07:06 -05:00 · tarea 2 · verde · caducada · `bash -n scripts/build-release.sh && grep -qE "^go test -race \./\.\.\.$" scripts/build-release.sh && ! grep -qE "^go test \./\.\.\.$" scripts/build-release.sh` (salida 0)
- 2026-11-13 07:06 -05:00 · tarea 3 · verde · caducada · `grep -E "^\| C-002 " docs/known-issues.md | grep -q Corregido && grep -E "^\| C-003 " docs/known-issues.md | grep -q Corregido` (salida 0)
- 2026-11-13 07:06 -05:00 · tarea 4 · verde · caducada · `go vet ./... && go test -race ./... -count=1` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	33.516s
- 2026-11-13 07:06 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD033HTTPJSONClientTimeout$' -count=1 -v` (salida 0)
      === RUN   TestHCD033HTTPJSONClientTimeout
      --- PASS: TestHCD033HTTPJSONClientTimeout (0.20s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.366s
- 2026-11-13 07:06 -05:00 · tarea 2 · verde · caducada · `bash -n scripts/build-release.sh && grep -qE "^go test -race \./\.\.\.$" scripts/build-release.sh && ! grep -qE "^go test \./\.\.\.$" scripts/build-release.sh` (salida 0)
- 2026-11-13 07:07 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD033HTTPJSONClientTimeout$' -count=1 -v` (salida 0)
      === RUN   TestHCD033HTTPJSONClientTimeout
      --- PASS: TestHCD033HTTPJSONClientTimeout (0.20s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.373s
- 2026-11-13 07:07 -05:00 · tarea 2 · verde · vigente · `bash -n scripts/build-release.sh && grep -qE "^go test -race \./\.\.\.$" scripts/build-release.sh && ! grep -qE "^go test \./\.\.\.$" scripts/build-release.sh` (salida 0)
- 2026-11-13 07:07 -05:00 · tarea 3 · verde · vigente · `grep -E "^\| C-002 " docs/known-issues.md | grep -q Corregido && grep -E "^\| C-003 " docs/known-issues.md | grep -q Corregido` (salida 0)
- 2026-11-13 07:07 -05:00 · tarea 4 · verde · vigente · `go vet ./... && go test -race ./... -count=1` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	33.318s

## Siguiente
Cerrada con aprobación sobre la revisión f3d4f9760074.

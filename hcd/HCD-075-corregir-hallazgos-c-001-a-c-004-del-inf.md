# Corregir hallazgos C-001 a C-004 del informe ACR v1.25.0

ID: HCD-075
Tipo: fix
Carril: fast
Motivo: Hallazgos acotados del informe ACR acr_ee2904a0-6cd7-41db-9c0f-94d547587a98 sin decisiones interdependientes

## Intención
Corregir truncado UTF-8 de CTX en launcher (C-001), error de ReadDir en skillDriftItem (C-002), JSON corrupto en cachedDeps (C-003) y carga defensiva de patrones en gomemory-plan.ts (C-004)

## Reproducción
Informe ACR `acr_ee2904a0-6cd7-41db-9c0f-94d547587a98` sobre `v1.25.0` (`bffc7f151949389774bb1e9eef3958dff56e616f`):
1. `C-001` (`pkg/installer/harness.go:88-93`): `${#CTX}` cuenta caracteres en locales UTF-8 y no bytes, y `${CTX:0:65536}` corta sin respetar límites UTF-8, pudiendo superar `MAX_ARG_STRLEN` (128 KiB en Linux) con contextos ricos en glifos Unicode.
2. `C-002` (`pkg/installer/hostskills.go:322-347`): `skillDriftItem` ignora el error de `os.ReadDir(repo)` y reporta `"mem no instaló skills en ningún agente"` cuando `~/.pi-gomemory/skills` no existe o no es legible.
3. `C-003` (`pkg/installer/deps.go:355-365`): `cachedDeps` descarta errores de `json.Unmarshal` y oculta que `pi-gomemory-deps.json` está corrupto.
4. `C-004` (`extensions/gomemory-plan.ts:21-23`): `JSON.parse(readFileSync(...))` en top-level lanza una excepción no capturada si falta `gomemory-plan-patterns.json`.

## Alcance
Incluido: corregir `C-001` en `pkg/installer/harness.go`, `C-002` en `pkg/installer/hostskills.go`, `C-003` en `pkg/installer/deps.go`, `C-004` en `extensions/gomemory-plan.ts`, y añadir pruebas en `pkg/installer/hcd075_acr_test.go`.
Excluido: refactorizaciones ajenas a estos cuatro puntos, cambios en otros comandos o publicación de release.

## ¿Qué pasa si no lo hacemos?
El launcher puede superar `MAX_ARG_STRLEN` o partir UTF-8 con contextos largos, `doctor` oculta errores de lectura en el repositorio de skills, `deps-status` ignora cachés corruptas y `/plan` falla al cargar si falta su JSON de patrones.

## Alternativa más barata
Corregir únicamente `C-001` (`CONFIRMED MEDIUM`); incluir los tres `LOW` en la misma unidad cierra todo el informe ACR con cambios mínimos y pruebas unitarias acotadas. Constitución §8, §10, §18; Playbook HCD-DEF-01, HCD-TDD-01, HCD-EVIDENCE-01.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla en pkg/installer/harness.go (C-001: truncado de CTX por bytes UTF-8 sin partir multibyte) → go test ./pkg/installer -run TestHCD075LauncherCtxBytes -count=1
- [x] 2. Corregir skillDriftItem en pkg/installer/hostskills.go para distinguir error de os.ReadDir(repo) con WARN (C-002) → go test ./pkg/installer -run TestHCD075SkillDriftReadError -count=1
- [x] 3. Corregir cachedDeps en pkg/installer/deps.go para informar cuando pi-gomemory-deps.json tiene JSON corrupto (C-003) → go test ./pkg/installer -run TestHCD075CorruptDepsCache -count=1
- [x] 4. Corregir carga de patrones en extensions/gomemory-plan.ts con try/catch y respaldo mínimo (C-004) → go test ./pkg/installer -run TestHCD075PlanMissingPatterns -count=1
- [x] 5. Ejecutar regresión completa del repositorio → go vet ./... && go test -race ./...

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-23 19:47 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD075LauncherCtxBytes -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd075_acr_test.go:71:12: undefined: DepsCacheError
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-23 19:47 -05:00 · tarea 1 · falló · caducada · `go test ./pkg/installer -run TestHCD075LauncherCtxBytes -count=1` (salida 1)
      --- FAIL: TestHCD075LauncherCtxBytes (0.06s)
          hcd075_acr_test.go:43: CTX truncado partió un carácter UTF-8 al final
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.506s
      FAIL
- 2026-11-23 19:48 -05:00 · tarea 1 · falló · caducada · `go test ./pkg/installer -run TestHCD075LauncherCtxBytes -count=1` (salida 1)
      --- FAIL: TestHCD075LauncherCtxBytes (0.04s)
          hcd075_acr_test.go:43: CTX truncado partió un carácter UTF-8 al final
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.270s
      FAIL
- 2026-11-23 19:48 -05:00 · tarea 1 · falló · caducada · `go test ./pkg/installer -run TestHCD075LauncherCtxBytes -count=1` (salida 1)
      --- FAIL: TestHCD075LauncherCtxBytes (0.06s)
          hcd075_acr_test.go:43: CTX truncado partió un carácter UTF-8 al final
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.285s
      FAIL
- 2026-11-23 19:48 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestHCD075LauncherCtxBytes -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.271s
- 2026-11-23 19:48 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD075SkillDriftReadError -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.185s
- 2026-11-23 19:48 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD075CorruptDepsCache -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.223s
- 2026-11-23 19:48 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestHCD075PlanMissingPatterns -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.418s
- 2026-11-23 19:48 -05:00 · tarea 5 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	2.175s
      ok  	pi-gomemory/pkg/installer	47.717s

## Siguiente
Cerrada con aprobación sobre la revisión fccbbb91c4b6.

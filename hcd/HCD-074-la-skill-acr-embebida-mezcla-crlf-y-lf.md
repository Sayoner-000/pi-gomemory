# La skill ACR embebida mezcla CRLF y LF

ID: HCD-074
Tipo: fix
Carril: fast
Motivo: fix con reproducción clara (file la marca con CRLF y LF) y sin decisiones abiertas

## Intención
Que la copia embebida de adversarial-consensus-review use solo LF como la de gomemory v2.56.0 y que un test impida embeber skills con CR

## Reproducción
`file skills/adversarial-consensus-review/SKILL.md` informa «with CRLF, LF line terminators». La copia se tomó de `mem` antes de que gomemory v2.56.0 (HCD-062) la normalizara a LF.

## Alcance
Incluido: reemplazar la copia embebida de la skill ACR por la versión LF de gomemory v2.56.0, sin cambios de texto; un test que falle si una skill embebida en `skills/` contiene CR.
Excluido: la normalización en tiempo de instalación (`SanitizeFrontmatter` ya quita los CR) y las skills del anfitrión.

## ¿Qué pasa si no lo hacemos?
La copia de respaldo difiere en bytes de la de `mem`, sus diffs muestran el archivo entero como cambiado y la comprobación de la tarea 9 de HCD-073 falla en cuanto el anfitrión instala gomemory v2.56.0.

## Alternativa más barata
Dejarla así, porque el instalador ya quita los CR al escribirla; pero el repositorio seguiría guardando una copia distinta de la fuente.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Escribir en pkg/installer un test que falle si una skill embebida de skills/ contiene CR → go test ./pkg/installer -run TestHCD074SkillsSinCR -count=1
- [x] 2. Reemplazar skills/adversarial-consensus-review/SKILL.md por la versión LF de gomemory v2.56.0 → go test ./pkg/installer -run TestHCD074SkillsSinCR -count=1 && git diff --ignore-cr-at-eol --quiet -- skills/adversarial-consensus-review/SKILL.md
- [x] 3. Ejecutar la regresión completa → go vet ./... && go test ./... -count=1

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-23 19:41 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD074SkillsSinCR -count=1` (salida 1)
      --- FAIL: TestHCD074SkillsSinCR (0.00s)
          hcd074_test.go:29: skills embebidas con CR: adversarial-consensus-review
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.399s
      FAIL
- 2026-11-23 19:41 -05:00 · tarea 1,2 · verde · caducada · `go test ./pkg/installer -run TestHCD074SkillsSinCR -count=1 && git diff --ignore-cr-at-eol --quiet -- skills/adversarial-consensus-review/SKILL.md` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.217s
- 2026-11-23 19:41 -05:00 · tarea 3 · verde · caducada · `go vet ./... && go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL|--- FAIL)"; test ${PIPESTATUS[0]} -eq 0` (salida 0)
      ok  	pi-gomemory	1.242s
      ok  	pi-gomemory/pkg/installer	44.570s
- 2026-11-24 18:52 -05:00 · tarea 1,2 · verde · vigente · `go test ./pkg/installer -run TestHCD074SkillsSinCR -count=1 && git diff --ignore-cr-at-eol --quiet -- skills/adversarial-consensus-review/SKILL.md` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.183s
- 2026-11-24 18:52 -05:00 · tarea 3 · verde · vigente · `go vet ./... && go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL|--- FAIL)"; test ${PIPESTATUS[0]} -eq 0` (salida 0)
      ok  	pi-gomemory/pkg/installer	51.342s

## Siguiente
Cerrada con aprobación sobre la revisión bd285530e615.

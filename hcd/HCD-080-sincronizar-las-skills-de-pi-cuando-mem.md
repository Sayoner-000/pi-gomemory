# Sincronizar las skills de Pi cuando mem las actualiza

ID: HCD-080
Tipo: feature
Carril: fast
Motivo: Diseño acordado con la persona tras HCD-079 (detección por eventos, sin timer); sin decisiones abiertas ni contratos de datos

## Intención
Cuando mem actualiza sus skills (mem update), Pi carga la versión nueva sin esperar a un install/update de pi-gomemory, y el repo avisa si sus skills embebidas se quedan por detrás de las de mem


## Alcance
Incluido: (1) `SkillRepoDrift` en pkg/installer/skills.go: lista las skills de gomemory cuya copia intacta en ~/.pi-gomemory/skills difiere de lo que escribiría una sincronización (copia de `mem` o embebida, saneada) o falta; ignora copias editadas por la persona y no escribe nada. (2) `pi-gomemory-setup skills-sync --check`: sale con 0 si está al día y con 10 (como `deps-status`) listando las desfasadas. (3) extensions/gomemory-status.ts: en `session_start` y tras un bash que ejecute `mem update`, comprueba; si hay deriva, ejecuta `skills-sync` y avisa de que `/reload` las carga. (4) Test de deriva del repo entre skills/ embebidas y las copias que instaló `mem` (se omite si no hay copias) y `make skills-embed` para copiarlas. (5) Docs y CHANGELOG.
Excluido: temporizador periódico (las skills solo cambian al actualizar `mem`); recargar Pi automáticamente (lo decide la persona con `/reload`); tocar copias editadas por la persona; detectar un `mem update` ejecutado fuera de Pi en una sesión ya abierta (lo cubre el siguiente `session_start`).

## ¿Qué pasa si no lo hacemos?
Comprobado en HCD-079: tras `mem update` a 2.59.0, las copias de los agentes estaban al día pero ~/.pi-gomemory/skills (la que carga Pi) seguía con hcd-verify de 2.58.0, y las skills embebidas del repo también se habían quedado atrás sin que ningún test lo detectara. Pi trabaja con instrucciones HCD que contradicen al motor hasta el siguiente install/update.

## Alternativa más barata
Ejecutar `skills-sync` sin comprobar en cada `session_start`: más simple, pero reescribe 20 archivos en cada arranque y no permite avisar de qué cambió. Se descarta. Un heartbeat periódico también se descarta: gasta consultas sin que nada cambie.


## Aclaraciones
- [x] Q1. En esta máquina ~/.pi-gomemory/skills/hcd-verify y hcd-implement están editadas a mano (09:35, adaptación provisional a --stale anterior a 2.59.0); la sincronización protege las copias editadas, así que --check sale 0 y Pi seguiría con instrucciones obsoletas. ¿Cómo se tratan las copias editadas que difieren de la versión de mem? → A — --check informa aparte de las copias editadas que difieren de mem sin sobrescribirlas, la extensión avisa en amarillo y se añade skills-sync --restore <nombre>; después se restauran las dos copias de esta máquina

## Checklist
- [x] 1. Añadir `SkillRepoDrift` en pkg/installer/skills.go (copia intacta distinta o ausente → deriva; editada → se ignora; sin escrituras) → `go test -count=1 ./pkg/installer -run '^TestHCD080SkillRepoDrift$'`
- [x] 2. Añadir `skills-sync --check` en main.go con salida 10 y la lista de desfasadas → `go test -count=1 -run '^TestHCD080SkillsSyncCheck$' .`
- [x] 3. Sincronizar y avisar desde extensions/gomemory-status.ts en `session_start` cuando `--check` sale con 10 → `go test -count=1 ./pkg/installer -run '^TestHCD080StatusSessionStart$'`
- [x] 4. Repetir la comprobación desde extensions/gomemory-status.ts tras un bash con `mem update` → `go test -count=1 ./pkg/installer -run '^TestHCD080StatusAfterMemUpdate$'`
- [x] 5. Añadir `EmbeddedSkillDrift` en pkg/installer/skills.go y el test de deriva del repo con `make skills-embed` en Makefile → `go test -count=1 ./pkg/installer -run '^TestHCD080EmbeddedSkillDrift$' && go test -count=1 -run '^TestHCD080EmbeddedSkillsMatchMem$' .`
- [x] 9. Añadir `SkillRepoEdited` en pkg/installer/skills.go: copias con marca de pi-gomemory editadas que difieren de la vigente; ignora carpetas propias sin marca (Q1) → `go test -count=1 ./pkg/installer -run '^TestHCD080SkillRepoEdited$'`
- [x] 10. Informar en `skills-sync --check` (main.go) de las editadas en línea aparte con salida 10, sin escribir (Q1) → `go test -count=1 -run '^TestHCD080SkillsSyncCheckEdited$' .`
- [x] 11. Añadir `skills-sync --restore <nombre>` en main.go: repone una skill de gomemory editada y rechaza nombres ajenos o carpetas con más archivos (Q1) → `go test -count=1 -run '^TestHCD080SkillsSyncRestore$' .`
- [x] 12. Avisar en amarillo desde extensions/gomemory-status.ts de las editadas con el comando `--restore`, sin sincronizar si solo hay editadas (Q1) → `go test -count=1 ./pkg/installer -run '^TestHCD080StatusEditedWarning$'`
- [x] 6. Comprobar en esta máquina que ~/.pi-gomemory/skills/hcd-verify/SKILL.md queda al día con el binario nuevo (rojo: `--check` 10 por las dos editadas → `--restore` → 0) → `go build -o /tmp/pgs-080 . && /tmp/pgs-080 skills-sync --check`
- [x] 7. Documentar en README.md, docs/usage.md y CHANGELOG.md (documentación: sin rojo) → `rg -n "skills-sync --check" README.md docs/usage.md CHANGELOG.md`
- [x] 8. Regresión completa (regresión final: sin rojo) → `go vet ./... && go test -count=1 ./... && node --test tests/*.mjs tests/antigravity-prompt-guard/*.test.mjs`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-24 15:43 -05:00 · tarea 1 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080SkillRepoDrift$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd080_test.go:25:16: undefined: SkillRepoDrift
      pkg/installer/hcd080_test.go:38:15: undefined: SkillRepoDrift
      pkg/installer/hcd080_test.go:50:16: undefined: SkillRepoDrift
      pkg/installer/hcd080_test.go:58:16: undefined: SkillRepoDrift
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 15:43 -05:00 · tarea 1 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080SkillRepoDrift$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.023s
- 2026-11-24 15:43 -05:00 · tarea 2 · rojo observado · caducada · `go test -count=1 -run '^TestHCD080SkillsSyncCheck$' .` (salida 1)
      # pi-gomemory [pi-gomemory.test]
      ./main_hcd080_test.go:21:13: undefined: runSkillsSync
      ./main_hcd080_test.go:32:13: undefined: runSkillsSync
      ./main_hcd080_test.go:35:13: undefined: runSkillsSync
      ./main_hcd080_test.go:38:13: undefined: runSkillsSync
      FAIL	pi-gomemory [build failed]
      FAIL
- 2026-11-24 15:43 -05:00 · tarea 2 · verde · caducada · `go test -count=1 -run '^TestHCD080SkillsSyncCheck$' .` (salida 0)
      ok  	pi-gomemory	0.111s
- 2026-11-24 15:44 -05:00 · tarea 3 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080StatusSessionStart$'` (salida 1)
      --- FAIL: TestHCD080StatusSessionStart (0.18s)
          hcd080_test.go:65: con deriva debe comprobar y sincronizar: []
              FIRST=[]
              NOTES=[]
              SECOND=[]
              NOTES2=[]
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.181s
      FAIL
- 2026-11-24 15:44 -05:00 · tarea 3 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080StatusSessionStart$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.169s
- 2026-11-24 15:44 -05:00 · tarea 4 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080StatusAfterMemUpdate$'` (salida 1)
      --- FAIL: TestHCD080StatusAfterMemUpdate (0.21s)
          hcd080_test.go:93: tras mem update debe comprobar, sincronizar y avisar: [] []
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.214s
      FAIL
- 2026-11-24 15:44 -05:00 · tarea 4 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080StatusAfterMemUpdate$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.215s
- 2026-11-24 15:45 -05:00 · tarea 5 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080EmbeddedSkillDrift$' && go test -count=1 -run '^TestHCD080EmbeddedSkillsMatchMem$' .` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd080_test.go:102:17: undefined: EmbeddedSkillDrift
      pkg/installer/hcd080_test.go:110:14: undefined: EmbeddedSkillDrift
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 15:45 -05:00 · tarea 5 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080EmbeddedSkillDrift$' && go test -count=1 -run '^TestHCD080EmbeddedSkillsMatchMem$' .` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.005s
      ok  	pi-gomemory	0.006s
- 2026-11-24 15:45 -05:00 · tarea 6 · rojo NO observado (el comando pasó) · caducada · `go build -o /tmp/pgs-080 . && /tmp/pgs-080 skills-sync --check` (salida 0)
- 2026-11-24 15:47 -05:00 · tarea 9 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080SkillRepoEdited$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd080_test.go:110:16: undefined: SkillRepoEdited
      pkg/installer/hcd080_test.go:127:13: undefined: SkillRepoEdited
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 15:47 -05:00 · tarea 9 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080SkillRepoEdited$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.028s
- 2026-11-24 15:48 -05:00 · tarea 10 · rojo observado · caducada · `go test -count=1 -run '^TestHCD080SkillsSyncCheckEdited$' .` (salida 1)
      --- FAIL: TestHCD080SkillsSyncCheckEdited (0.10s)
          main_hcd080_test.go:36: con una editada: código 0, quiero 10
      FAIL
      FAIL	pi-gomemory	0.106s
      FAIL
- 2026-11-24 15:48 -05:00 · tarea 10 · verde · caducada · `go test -count=1 -run '^TestHCD080SkillsSyncCheckEdited$' .` (salida 0)
      ok  	pi-gomemory	0.120s
- 2026-11-24 15:48 -05:00 · tarea 11 · rojo observado · caducada · `go test -count=1 -run '^TestHCD080SkillsSyncRestore$' .` (salida 1)
      --- FAIL: TestHCD080SkillsSyncRestore (0.09s)
          main_hcd080_test.go:63: con archivos propios debe negarse: código 2
      FAIL
      FAIL	pi-gomemory	0.098s
      FAIL
- 2026-11-24 15:48 -05:00 · tarea 11 · verde · caducada · `go test -count=1 -run '^TestHCD080SkillsSyncRestore$' .` (salida 0)
      ok  	pi-gomemory	0.181s
- 2026-11-24 15:49 -05:00 · tarea 12 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080StatusEditedWarning$'` (salida 1)
      --- FAIL: TestHCD080StatusEditedWarning (0.16s)
          hcd080_test.go:131: solo editadas: no debe sincronizar: ["pi-gomemory-setup skills-sync --check","pi-gomemory-setup skills-sync"]
              ONLY_EXECS=["pi-gomemory-setup skills-sync --check","pi-gomemory-setup skills-sync"]
              ONLY_NOTES=[["info","Skills de gomemory actualizadas desde mem. Usa /reload para cargarlas."]]
              MIX_EXECS=["pi-gomemory-setup skills-sync --check","pi-gomemory-setup skills-sync"]
              MIX_NOTES=[["info","Skills de gomemory actualizadas desde mem (hcd-close). Usa /reload para cargarlas."]]
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.164s
      FAIL
- 2026-11-24 15:49 -05:00 · tarea 12 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD080StatusEditedWarning$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.186s
- 2026-11-24 15:49 -05:00 · tarea 6 · rojo observado · caducada · `go build -o /tmp/pgs-080 . && /tmp/pgs-080 skills-sync --check` (salida 10)
      skills de gomemory editadas localmente y desfasadas respecto a mem: hcd-implement, hcd-verify. Ejecuta pi-gomemory-setup skills-sync --restore hcd-implement hcd-verify
- 2026-11-24 15:49 -05:00 · tarea 6 · verde · caducada · `go build -o /tmp/pgs-080 . && /tmp/pgs-080 skills-sync --check` (salida 0)
- 2026-11-24 15:50 -05:00 · tarea 7 · verde · caducada · `rg -n "skills-sync --check" README.md docs/usage.md CHANGELOG.md` (salida 0)
      de/skills`, `~/.claude/skills`, `~/.codex/skills` y `~/.gemini/skills`. Pi carga solo ese repositorio, así que no hay avisos de colisión. Se sincroniza al instalar y en cada arranque de `pi-gomemory`. Nunca modifica las carpetas de otros agentes ni las skills tuyas. Si `mem update` trae skills nuevas, Pi lo detecta al iniciar la sesión o justo después de un `mem update` lanzado desde Pi (`pi-gomemory-setup skills-sync --check`), sincroniza y avisa de que `/reload` las carga. Las copias de gomemory que se editaron a mano no se sobrescriben: Pi avisa en amarillo de las que quedaron desfasadas y `pi-gomemory-setup skills-sync --restore <skill>` las repone.
      docs/usage.md:253:**Skills al día tras `mem update`.** Al iniciar cada sesión y después de un `mem update` ejecutado desde Pi, `gomemory-status` lanza `pi-gomemory-setup skills-sync --check`. Si hay copias desfasadas, sincroniza y avisa de que `/reload` carga la versión nueva. Las copias de gomemory editadas a mano no se sobrescriben nunca: si difieren de la versión de `mem`, Pi avisa en amarillo y `pi-gomemory-setup skills-sync --restore <skill>` las repone, salvo que la carpeta tenga otros archivos tuyos. En el repositorio de pi-gomemory, `TestHCD080EmbeddedSkillsMatchMem` falla si `skills/` se queda por detrás de las copias de `mem` de tu máquina (se omite si no hay ninguna o con `PI_GOMEMORY_SKIP_SKILL_DRIFT=1`); `make skills-embed` las copia.
      CHANGELOG.md:17:- Skills de Pi al día tras `mem update` (HCD-080): `pi-gomemory-setup skills-sync --check` compara `~/.pi-gomemory/skills` con la versión vigente (la copia de `mem` o la embebida) sin escribir y sale con 10 si hay skills desfasadas o editadas que difieren. `gomemory-status.ts` lo ejecuta en `session_start` y tras un bash con `mem update`: sincroniza las desfasadas y avisa de `/reload`. Las copias editadas a mano no se sobrescriben; se avisa en amarillo y `skills-sync --restore <skill>…` las repone (se niega si la carpeta tiene otros archivos).
- 2026-11-24 15:50 -05:00 · tarea 8 · falló · caducada · `go vet ./... && go test -count=1 ./... && node --test tests/*.mjs tests/antigravity-prompt-guard/*.test.mjs` (salida 1)
              AssertionError [ERR_ASSERTION]: C-02: con =1 el pie no debe consultar nada
              1 !== 0
                  at file:///tmp/TestHCD037Footer1963190639/001/test.mjs:23:8 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: 1,
                expected: 0,
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v22.23.2
      --- FAIL: TestHCD029StatusExtensionUpdateIndicator (0.24s)
          hcd029_status_test.go:67: secuencia de consultas = ["pi-gomemory-setup deps-status","pi-gomemory-setup skills-sync --check","pi-gomemory-setup deps-refresh","pi-gomemory-setup deps-status"]; quiero ["pi-gomemory-setup deps-status","pi-gomemory-setup deps-refresh","pi-gomemory-setup deps-status"]
      FAIL
      FAIL	pi-gomemory/pkg/installer	33.357s
      FAIL
- 2026-11-24 15:52 -05:00 · tarea 8 · verde · vigente · `go vet ./... && go test -count=1 ./... && node --test tests/*.mjs tests/antigravity-prompt-guard/*.test.mjs` (salida 0)
      ok 37 - renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)
        ---
        duration_ms: 1.614544
        type: 'test'
        ...
      # Subtest: HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
      ok 38 - HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
        ---
        duration_ms: 249.192899
        type: 'test'
        ...
      1..38
      # tests 38
      # suites 0
      # pass 38
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 392.341215
- 2026-11-24 15:53 -05:00 · tarea 1 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD080SkillRepoDrift$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.022s
- 2026-11-24 15:53 -05:00 · tarea 2 · verde · vigente · `go test -count=1 -run '^TestHCD080SkillsSyncCheck$' .` (salida 0)
      ok  	pi-gomemory	0.126s
- 2026-11-24 15:53 -05:00 · tarea 3 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD080StatusSessionStart$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.239s
- 2026-11-24 15:53 -05:00 · tarea 4 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD080StatusAfterMemUpdate$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.239s
- 2026-11-24 15:53 -05:00 · tarea 5 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD080EmbeddedSkillDrift$' && go test -count=1 -run '^TestHCD080EmbeddedSkillsMatchMem$' .` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.008s
      ok  	pi-gomemory	0.003s
- 2026-11-24 15:53 -05:00 · tarea 9 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD080SkillRepoEdited$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.026s
- 2026-11-24 15:53 -05:00 · tarea 10 · verde · vigente · `go test -count=1 -run '^TestHCD080SkillsSyncCheckEdited$' .` (salida 0)
      ok  	pi-gomemory	0.111s
- 2026-11-24 15:53 -05:00 · tarea 11 · verde · vigente · `go test -count=1 -run '^TestHCD080SkillsSyncRestore$' .` (salida 0)
      ok  	pi-gomemory	0.206s
- 2026-11-24 15:53 -05:00 · tarea 12 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD080StatusEditedWarning$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.168s
- 2026-11-24 15:53 -05:00 · tarea 6 · verde · vigente · `go build -o /tmp/pgs-080 . && /tmp/pgs-080 skills-sync --check` (salida 0)
- 2026-11-24 15:53 -05:00 · tarea 7 · verde · vigente · `rg -n "skills-sync --check" README.md docs/usage.md CHANGELOG.md` (salida 0)
      de/skills`, `~/.claude/skills`, `~/.codex/skills` y `~/.gemini/skills`. Pi carga solo ese repositorio, así que no hay avisos de colisión. Se sincroniza al instalar y en cada arranque de `pi-gomemory`. Nunca modifica las carpetas de otros agentes ni las skills tuyas. Si `mem update` trae skills nuevas, Pi lo detecta al iniciar la sesión o justo después de un `mem update` lanzado desde Pi (`pi-gomemory-setup skills-sync --check`), sincroniza y avisa de que `/reload` las carga. Las copias de gomemory que se editaron a mano no se sobrescriben: Pi avisa en amarillo de las que quedaron desfasadas y `pi-gomemory-setup skills-sync --restore <skill>` las repone.
      docs/usage.md:253:**Skills al día tras `mem update`.** Al iniciar cada sesión y después de un `mem update` ejecutado desde Pi, `gomemory-status` lanza `pi-gomemory-setup skills-sync --check`. Si hay copias desfasadas, sincroniza y avisa de que `/reload` carga la versión nueva. Las copias de gomemory editadas a mano no se sobrescriben nunca: si difieren de la versión de `mem`, Pi avisa en amarillo y `pi-gomemory-setup skills-sync --restore <skill>` las repone, salvo que la carpeta tenga otros archivos tuyos. En el repositorio de pi-gomemory, `TestHCD080EmbeddedSkillsMatchMem` falla si `skills/` se queda por detrás de las copias de `mem` de tu máquina (se omite si no hay ninguna o con `PI_GOMEMORY_SKIP_SKILL_DRIFT=1`); `make skills-embed` las copia.
      CHANGELOG.md:17:- Skills de Pi al día tras `mem update` (HCD-080): `pi-gomemory-setup skills-sync --check` compara `~/.pi-gomemory/skills` con la versión vigente (la copia de `mem` o la embebida) sin escribir y sale con 10 si hay skills desfasadas o editadas que difieren. `gomemory-status.ts` lo ejecuta en `session_start` y tras un bash con `mem update`: sincroniza las desfasadas y avisa de `/reload`. Las copias editadas a mano no se sobrescriben; se avisa en amarillo y `skills-sync --restore <skill>…` las repone (se niega si la carpeta tiene otros archivos).

## Siguiente
Cerrada con aprobación sobre la revisión 10f388c394b1.

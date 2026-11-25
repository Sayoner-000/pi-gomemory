# Skills externas impeccable y frontend-design desde skills.sh y gitlab-release embebida

ID: HCD-084
Tipo: feature
Carril: fast
Motivo: Vía rápida: decisiones cerradas por la persona (npx skills, actualizar siempre, revisar al iniciar y en update)

## Intención
install, update e inicio de Pi instalan o actualizan impeccable (pbakaus/impeccable), frontend-design (anthropics/skills) y playwright-cli (microsoft/playwright-cli) con npx skills -g -a pi; gitlab-release viaja embebida en el meta-harness; doctor las informa y uninstall las conserva


## Alcance
Incluido: catálogo ExternalSkills (impeccable ← pbakaus/impeccable, frontend-design ← anthropics/skills, playwright-cli ← microsoft/playwright-cli); UpdateExternalSkills instala con `npx -y skills add <src> -g -a pi -s <skill> -y` las que faltan o no están en el lockfile de skills (copias manuales) y actualiza siempre las gestionadas con `npx -y skills update <skills> -g -y`; se llama en el paso [6/9] de install (antes de reunir el repositorio central), en `update` cuando pi-gomemory ya está al día, y en el arranque de Pi desde gomemory-status.ts mediante `pi-gomemory-setup skills-update --if-due` (como mucho una vez cada 24 h, en segundo plano); comando `skills-update [--if-due]`; comprobación «Skills externas» en doctor; uninstall las conserva y muestra `npx skills remove`; gitlab-release embebida en skills/; README, CHANGELOG y ayuda.
Excluido: añadirlas al catálogo de componentes versionados (no tienen semver; el catálogo y sus pruebas siguen igual); modo lite y PI_GOMEMORY_SKIP_DEPS solo omiten la red de skills con aviso; no se instalan en otros agentes (`-a pi` evita los ~56 enlaces que crea skills sin -a).

## ¿Qué pasa si no lo hacemos?
En una máquina nueva Pi no tiene impeccable, frontend-design ni gitlab-release: hoy solo llegan por enlaces a instalaciones manuales en ~/.agents/skills y ~/.claude/skills, y nunca se actualizan.

## Alternativa más barata
Documentar `npx skills add …` en el README: no garantiza el combo completo ni la actualización, que es lo pedido.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Definir ExternalSkills con fuente oficial → go test ./pkg/installer -run '^TestHCD084Catalog$'
- [x] 2. Instalar las ausentes con `skills add -g -a pi` y actualizar siempre las presentes → go test ./pkg/installer -run '^TestHCD084InstallAndUpdate$'
- [x] 3. Avisar y seguir sin npx, con skip-deps o desinstalado → go test ./pkg/installer -run '^TestHCD084Degrade$'
- [x] 4. Limitar --if-due a una ejecución cada 24 h → go test ./pkg/installer -run '^TestHCD084IfDue$'
- [x] 5. Llamar UpdateExternalSkills en el paso [6/9] de install antes de InstallSkills → go test ./pkg/installer -run '^TestHCD084InstallStep$'
- [x] 6. Exponer `skills-update [--if-due]` y llamarlo en update ya al día → go test . -run '^TestHCD084'
- [x] 7. Ejecutar skills-update --if-due al arrancar Pi desde gomemory-status.ts → go test ./pkg/installer -run '^TestHCD084StartupHook$'
- [x] 8. Informar «Skills externas» en doctor → go test ./pkg/installer -run '^TestHCD084Doctor$'
- [x] 9. Conservar las skills externas en uninstall con su comando de retirada → go test ./pkg/installer -run '^TestHCD084Uninstall$'
- [x] 10. Embeber gitlab-release en skills/ → go test ./pkg/installer -run '^TestHCD084GitlabRelease$'
- [x] 11. Documentar en README, CHANGELOG y ayuda del CLI → go test . -run '^TestHCD084Help$'
- [x] 12. Regresión completa → go vet ./... && go test ./...
- [x] 13. Instalación real en HOME temporal deja impeccable, frontend-design, playwright-cli y gitlab-release en ~/.pi-gomemory/skills → bash scripts/e2e-hcd084.sh

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-25 08:24 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD084Catalog$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd084_test.go:65:9: undefined: ExternalSkills
      pkg/installer/hcd084_test.go:66:12: undefined: ExternalSkill
      pkg/installer/hcd084_test.go:76:23: undefined: externalSkillAddArgs
      pkg/installer/hcd084_test.go:79:23: undefined: externalSkillUpdateArgs
      pkg/installer/hcd084_test.go:97:9: undefined: UpdateExternalSkills
      pkg/installer/hcd084_test.go:109:6: undefined: ExternalSkillPresent
      pkg/installer/hcd084_test.go:116:8: undefined: UpdateExternalSkills
      pkg/installer/hcd084_test.go:132:10: undefined: UpdateExternalSkills
      pkg/installer/hcd084_test.go:141:10: undefined: UpdateExternalSkills
      pkg/installer/hcd084_test.go:150:10: undefined: UpdateExternalSkills
      pkg/installer/hcd084_test.go:150:10: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-25 08:25 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD084Catalog$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.004s
- 2026-11-25 08:25 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD084InstallAndUpdate$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.008s
- 2026-11-25 08:25 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD084Degrade$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.010s
- 2026-11-25 08:25 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD084IfDue$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.185s
- 2026-11-25 08:25 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD084InstallStep$'` (salida 1)
      --- FAIL: TestHCD084InstallStep (0.15s)
          hcd084_test.go:240: InstallAll no usa installSkillsStep en el paso [6/9]
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.158s
      FAIL
- 2026-11-25 08:25 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD084InstallStep$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.107s
- 2026-11-25 08:26 -05:00 · tarea 6 · rojo observado · caducada · `go test . -run '^TestHCD084(UpdateRunsSkills|SkillsUpdateCommand)$'` (salida 1)
      # pi-gomemory [pi-gomemory.test]
      ./main_test.go:26:3: unknown field skills in struct literal of type updateDeps
      FAIL	pi-gomemory [build failed]
      FAIL
- 2026-11-25 08:26 -05:00 · tarea 6 · verde · caducada · `go test . -run '^TestHCD084(UpdateRunsSkills|SkillsUpdateCommand)$'` (salida 0)
      ok  	pi-gomemory	0.008s
- 2026-11-25 08:26 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD084StartupHook$'` (salida 1)
      --- FAIL: TestHCD084StartupHook (0.00s)
          hcd084_test.go:252: gomemory-status.ts no ejecuta skills-update --if-due al arrancar
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.003s
      FAIL
- 2026-11-25 08:27 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD084StartupHook$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.006s
- 2026-11-25 08:27 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD084Doctor$'` (salida 1)
      --- FAIL: TestHCD084Doctor (0.00s)
          hcd084_test.go:265: falta «Skills externas» en doctor
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.006s
      FAIL
- 2026-11-25 08:27 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD084Doctor$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.004s
- 2026-11-25 08:27 -05:00 · tarea 9 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD084Uninstall$'` (salida 1)
      --- FAIL: TestHCD084Uninstall (0.00s)
          hcd084_test.go:284: falta "npx skills remove impeccable -g -y":
              Se conservan pi, mem, codegraph y herdr: son herramientas independientes que otros agentes pueden usar. Para quitarlas:
                pi:                  rm -rf ~/.pi/agent/install ~/.pi/agent/bin/pi (y su enlace en ~/.local/bin)
                mem:                 curl -fsSL https://raw.githubusercontent.com/Sayoner-000/gomemory/main/scripts/install.sh | bash -s -- --uninstall
                codegraph:           curl -fsSL https://raw.githubusercontent.com/colbymchenry/codegraph/main/install.sh | sh -s -- --uninstall
                herdr:               rm "$(command -v herdr)" (herdr no tiene subcomando de desinstalación)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.003s
      FAIL
- 2026-11-25 08:27 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run '^TestHCD084Uninstall$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.005s
- 2026-11-25 08:27 -05:00 · tarea 10 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD084GitlabRelease$'` (salida 1)
      --- FAIL: TestHCD084GitlabRelease (0.00s)
          hcd084_test.go:293: open ../../skills/gitlab-release/SKILL.md: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.003s
      FAIL
- 2026-11-25 08:28 -05:00 · tarea 10 · verde · caducada · `go test ./pkg/installer -run '^(TestHCD084GitlabRelease|TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean)$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.434s
- 2026-11-25 08:28 -05:00 · tarea 11 · verde · caducada · `go test . -run '^TestHCD084Help$'` (salida 0)
      ok  	pi-gomemory	0.004s
- 2026-11-25 08:29 -05:00 · tarea 11 · verde · caducada · `go test . -run '^TestHCD084Help$'` (salida 0)
      ok  	pi-gomemory	(cached)
- 2026-11-25 08:34 -05:00 · tarea 12 · verde · caducada · `go vet ./... && go test ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	169.727s
- 2026-11-25 08:38 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD084InstallAndUpdate$'` (salida 1)
      --- FAIL: TestHCD084InstallAndUpdate (0.01s)
          hcd084_test.go:124: npx llamado con:
              [-y skills add pbakaus/impeccable -g -a pi -s impeccable -y -y skills update frontend-design -g -y]
              quiero:
              [-y skills add pbakaus/impeccable -g -a pi -s impeccable -y -y skills add microsoft/playwright-cli -g -a pi -s playwright-cli -y -y skills update frontend-design -g -y]
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.019s
      FAIL
- 2026-11-25 08:38 -05:00 · tarea 2 · falló · caducada · `go test ./pkg/installer -run '^TestHCD084InstallAndUpdate$'` (salida 1)
      --- FAIL: TestHCD084InstallAndUpdate (0.03s)
          hcd084_test.go:143: cambio del lockfile no detectado: {Installed:[] Updated:[impeccable frontend-design playwright-cli] Changed:[impeccable frontend-design playwright-cli] Failed:[] Skipped:}
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.040s
      FAIL
- 2026-11-25 08:38 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD084InstallAndUpdate$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.021s
- 2026-11-25 08:39 -05:00 · tarea 13 · falló · caducada · `bash scripts/e2e-hcd084.sh` (salida 1)
      ✗ impeccable no enlazada desde ~/.agents/skills (enlace: '')
      ✗ frontend-design no enlazada desde ~/.agents/skills (enlace: '')
      ✗ playwright-cli no enlazada desde ~/.agents/skills (enlace: '')
      ✓ gitlab-release embebida
      ✓ sin directorios de otros agentes
      ✗ doctor no informa Skills externas OK
      ✓ --if-due no repite dentro de 24 h
- 2026-11-25 08:39 -05:00 · tarea 13 · verde · caducada · `bash scripts/e2e-hcd084.sh` (salida 0)
      ✓ impeccable instalada desde skills.sh y enlazada
      ✓ frontend-design instalada desde skills.sh y enlazada
      ✓ playwright-cli instalada desde skills.sh y enlazada
      ✓ gitlab-release embebida
      ✓ sin directorios de otros agentes
      ✓ doctor: Skills externas OK
      ✓ --if-due no repite dentro de 24 h
- 2026-11-25 08:40 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD084Catalog$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.003s
- 2026-11-25 08:40 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD084InstallAndUpdate$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-25 08:40 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD084Degrade$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.008s
- 2026-11-25 08:40 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD084IfDue$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.294s
- 2026-11-25 08:40 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD084InstallStep$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.173s
- 2026-11-25 08:40 -05:00 · tarea 6 · verde · vigente · `go test . -run '^TestHCD084(UpdateRunsSkills|SkillsUpdateCommand)$'` (salida 0)
      ok  	pi-gomemory	0.023s
- 2026-11-25 08:40 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run '^TestHCD084StartupHook$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.006s
- 2026-11-25 08:40 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run '^TestHCD084Doctor$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.007s
- 2026-11-25 08:40 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run '^TestHCD084Uninstall$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.008s
- 2026-11-25 08:40 -05:00 · tarea 10 · verde · vigente · `go test ./pkg/installer -run '^(TestHCD084GitlabRelease|TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean)$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.432s
- 2026-11-25 08:40 -05:00 · tarea 11 · verde · vigente · `go test . -run '^TestHCD084Help$'` (salida 0)
      ok  	pi-gomemory	0.005s
- 2026-11-25 08:40 -05:00 · tarea 12 · verde · vigente · `go vet ./... && go test ./...` (salida 0)
      ok  	pi-gomemory	0.023s
      ok  	pi-gomemory/pkg/installer	192.855s
- 2026-11-25 08:44 -05:00 · tarea 13 · verde · vigente · `bash scripts/e2e-hcd084.sh` (salida 0)
      ✓ impeccable instalada desde skills.sh y enlazada
      ✓ frontend-design instalada desde skills.sh y enlazada
      ✓ playwright-cli instalada desde skills.sh y enlazada
      ✓ gitlab-release embebida
      ✓ sin directorios de otros agentes
      ✓ doctor: Skills externas OK
      ✓ --if-due no repite dentro de 24 h

## Siguiente
Cerrada con aprobación sobre la revisión 888233a92744.

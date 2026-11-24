# Instalar en Pi las skills de gomemory desde mem y usar codegraph como grafo principal

ID: HCD-073
Tipo: technical
Carril: fast
Motivo: mejora técnica sin decisiones interdependientes; codegraph elegido como principal (memoria 2217)

## Intención
Que Pi use las skills HCD que instala mem en lugar de copias embebidas desfasadas, que doctor avise del desfase, que la fuente de Gemini apunte a ~/.gemini/config/skills y que el protocolo pida codegraph como grafo principal y codebase-memory solo ante huecos


## Alcance
Incluido: que las 21 skills embebidas que `mem` también instala en los agentes del anfitrión lleguen a Pi con el contenido vigente de `mem` (hoy 6 difieren: hcd, hcd-approve, hcd-define, hcd-verify, adversarial-consensus-review y defensive-qa-audit; hcd y hcd-approve no conocen el examen socrático, la decisión splash ni el renombre); que `doctor` avise cuando una skill de Pi no coincide con la que instaló `mem`; que la fuente de Gemini sea `~/.gemini/config/skills`, donde instala `mem`; que el protocolo de impacto (gomemory-context.ts y gomemory-plan.ts) pida codegraph como grafo principal y codebase-memory solo ante huecos de cobertura o trazas que codegraph no da.
Excluido: cambiar las plantillas de skills de gomemory; retirar el servidor codebase-memory de mcp.json; las skills de terceros de `~/.agents/skills`; detección remota de versiones nuevas.

## ¿Qué pasa si no lo hacemos?
Cada versión de `mem` que cambia una skill deja a Pi con instrucciones viejas hasta que alguien actualiza a mano las copias embebidas (memorias 2214 y 2215), y Pi recorre la ceremonia distinto que los demás agentes. El protocolo sigue obligando a consultar dos grafos para cada análisis de impacto, duplicando llamadas y tokens.

## Alternativa más barata
Copiar a mano las skills de `mem` a `skills/` en cada versión; es lo que se hizo en HCD-072 y es justamente lo que se desfasa. Un test que compare las copias embebidas con las de `mem` avisaría pero no corregiría la instalación de la persona.


## Aclaraciones
- [x] Q1. Algunas copias embebidas tienen ajustes propios de Pi (hcd-verify pide una verificación por llamada Bash para el panel de tareas). ¿Cómo se resuelve? (a) Pi usa la copia de mem tal cual y los ajustes de Pi pasan al protocolo de gomemory-context.ts; (b) copia de mem más un apéndice «En Pi» que añade el instalador; (c) se mantienen las embebidas y solo se avisa del desfase → a: Pi usa la copia de mem tal cual y los ajustes propios de Pi pasan al protocolo de gomemory-context.ts (respuesta de la persona: «a»)

## Checklist
- [x] 1. Escribir un test en el que SyncSkillRepo tome el contenido de la copia de mem en el anfitrión, use la embebida si no hay copia de mem y no pise una copia personalizada → go test ./pkg/installer -run TestHCD073SkillsDesdeMem -count=1
- [x] 2. Hacer que installEmbeddedSkills tome el SKILL.md de la primera copia de mem en ~/.claude, ~/.config/opencode, ~/.codex o ~/.gemini/config antes que el embebido → go test ./pkg/installer -run TestHCD073SkillsDesdeMem -count=1
- [x] 3. Escribir un test que enlace una skill que solo existe en ~/.gemini/config/skills → go test ./pkg/installer -run TestHCD073GeminiConfig -count=1
- [x] 4. Añadir ~/.gemini/config/skills a hostSkillSources → go test ./pkg/installer -run TestHCD073GeminiConfig -count=1
- [x] 5. Escribir un test en el que doctor avise con WARN y el nombre de la skill cuando la copia de Pi difiere de la de mem → go test ./pkg/installer -run TestHCD073DoctorDesfase -count=1
- [x] 6. Añadir a RunDoctor la comprobación de desfase entre las skills de Pi y las de mem → go test ./pkg/installer -run TestHCD073DoctorDesfase -count=1
- [x] 7. Escribir un test en el que el protocolo de impacto (gomemory-context.ts e IMPACT_PROTOCOL) pida codegraph como principal y codebase-memory solo ante huecos, y la viñeta HCD pida una verificación por llamada Bash → go test ./pkg/installer -run TestHCD073Protocolo -count=1
- [x] 8. Reescribir las viñetas de impacto, HCD y modo plan de gomemory-context.ts y el IMPACT_PROTOCOL de gomemory-plan.ts → go test ./pkg/installer -run 'TestHCD073Protocolo|TestHCD066' -count=1
- [x] 9. Actualizar en skills/ las 6 skills embebidas que difieren con el contenido de mem → go test . -run TestHCD072EmbeddedScheduling -count=1 && for n in skills/*/; do cmp -s "$n/SKILL.md" "$HOME/.claude/skills/$(basename $n)/SKILL.md" || exit 1; done
- [x] 10. Ejecutar la regresión completa → go vet ./... && go test ./... -count=1

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-23 19:18 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD073SkillsDesdeMem -count=1` (salida 1)
      --- FAIL: TestHCD073SkillsDesdeMem (0.02s)
          hcd073_test.go:50: hcd-verify no viene de mem:
              ---
              name: hcd-verify
              description: embebida
              ---
              embebida
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.437s
      FAIL
- 2026-11-23 19:18 -05:00 · tarea 1,2 · verde · caducada · `go test ./pkg/installer -run TestHCD073SkillsDesdeMem -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.266s
- 2026-11-23 19:19 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD073DoctorDesfase -count=1` (salida 1)
      --- FAIL: TestHCD073DoctorDesfase (0.04s)
          hcd073_test.go:62: recién sincronizado se espera OK: {Name: Status: Details:}
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.256s
      FAIL
- 2026-11-23 19:19 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD073GeminiConfig -count=1` (salida 1)
      --- FAIL: TestHCD073GeminiConfig (0.03s)
          hcd073_test.go:47: solo-gemini no se enlazó desde ~/.gemini/config/skills: "" readlink /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD073GeminiConfig3374687234/001/.pi-gomemory/skills/solo-gemini: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.255s
      FAIL
- 2026-11-23 19:19 -05:00 · tarea 5,6 · verde · caducada · `go test ./pkg/installer -run TestHCD073DoctorDesfase -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.283s
- 2026-11-23 19:19 -05:00 · tarea 3,4 · verde · caducada · `go test ./pkg/installer -run TestHCD073GeminiConfig -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.220s
- 2026-11-23 19:19 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD073Protocolo -count=1` (salida 1)
              node:fs:624
                  return binding.readFileUtf8(path, stringToFlags(options.flag));
                                 ^
              Error: ENOENT: no such file or directory, open '/private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD073Protocolo3601202945/001/gomemory-plan-patterns.json'
                  at readFileSync (node:fs:624:20)
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD073Protocolo3601202945/001/gomemory-plan.ts:22:2
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5) {
                errno: -2,
                code: 'ENOENT',
                syscall: 'open',
                path: '/private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD073Protocolo3601202945/001/gomemory-plan-patterns.json'
              }
              Node.js v26.13.1
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.376s
      FAIL
- 2026-11-23 19:20 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD073Protocolo -count=1` (salida 1)
      --- FAIL: TestHCD073Protocolo (0.12s)
          hcd073_test.go:62: TestHCD073Protocolo: FAIL impacto: codegraph no figura como principal | impacto: codebase-memory no queda como respaldo | plan: codegraph no figura como principal | plan: codebase-memory no queda como respaldo | IMPACT_PROTOCOL: codegraph no figura como principal | IMPACT_PROTOCOL: codebase-memory no queda como respaldo | HCD: falta una verificación por llamada Bash para el panel de tareas | HCD: sigue exigiendo los dos grafos
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.352s
      FAIL
- 2026-11-23 19:20 -05:00 · tarea 7,8 · verde · caducada · `go test ./pkg/installer -run 'TestHCD073Protocolo|TestHCD066' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.965s
- 2026-11-23 19:20 -05:00 · tarea 9 · rojo observado · caducada · `go test . -run TestHCD072EmbeddedScheduling -count=1 && for n in skills/*/; do cmp -s "$n/SKILL.md" "$HOME/.claude/skills/$(basename $n)/SKILL.md" || { echo "difiere $n"; exit 1; }; done` (salida 1)
      ok  	pi-gomemory	0.522s
      difiere skills/adversarial-consensus-review/
- 2026-11-23 19:21 -05:00 · tarea 9 · verde · caducada · `go test . -run TestHCD072EmbeddedScheduling -count=1 && for n in skills/*/; do cmp -s "$n/SKILL.md" "$HOME/.claude/skills/$(basename $n)/SKILL.md" || { echo "difiere $n"; exit 1; }; done` (salida 0)
      ok  	pi-gomemory	0.691s
- 2026-11-23 19:21 -05:00 · tarea 10 · falló · caducada · `go vet ./... && go test ./... -count=1` (salida 1)
      sesEmbeddedAssetsAndLeavesCwdClean2873314024/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Modo codemode                    [38;2;145;169;197mhíbrido (codemode on)[0m
        [38;2;34;227;247m✓[0m Tools de codebase-memory         [38;2;145;169;197msin extensiones que dupliquen el servidor MCP[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Thinking colapsado (Ctrl+T)      [38;2;145;169;197mhideThinkingBlock activo en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Skills de gomemory al día        [38;2;145;169;197mmem no instaló skills en ningún agente; Pi usa las embebidas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2873314024/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	45.202s
      FAIL
- 2026-11-23 19:23 -05:00 · tarea 1,2,3,4,5,6,7,8,10 · verde · caducada · `go vet ./... && go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL|--- FAIL)"; test ${PIPESTATUS[0]} -eq 0` (salida 0)
      ok  	pi-gomemory	1.070s
      ok  	pi-gomemory/pkg/installer	45.123s
- 2026-11-23 19:23 -05:00 · tarea 9 · verde · caducada · `go test . -run TestHCD072EmbeddedScheduling -count=1 && for n in skills/*/; do cmp -s "$n/SKILL.md" "$HOME/.claude/skills/$(basename $n)/SKILL.md" || { echo "difiere $n"; exit 1; }; done` (salida 0)
      ok  	pi-gomemory	0.505s
- 2026-11-23 19:25 -05:00 · tarea 1,2,3,4,5,6,7,8,10 · verde · caducada · `go vet ./... && go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL|--- FAIL)"; test ${PIPESTATUS[0]} -eq 0` (salida 0)
      ok  	pi-gomemory	1.156s
      ok  	pi-gomemory/pkg/installer	43.670s
- 2026-11-23 19:26 -05:00 · tarea 9 · verde · vigente · `go test . -run TestHCD072EmbeddedScheduling -count=1 && for n in skills/*/; do cmp -s "$n/SKILL.md" "$HOME/.claude/skills/$(basename $n)/SKILL.md" || { echo "difiere $n"; exit 1; }; done` (salida 0)
      ok  	pi-gomemory	0.641s
- 2026-11-23 19:26 -05:00 · tarea 1,2,3,4,5,6,7,8,10 · verde · vigente · `go vet ./... && go test ./... -count=1 2>&1 | grep -E "^(ok|FAIL|--- FAIL)"; test ${PIPESTATUS[0]} -eq 0` (salida 0)
      ok  	pi-gomemory	1.157s
      ok  	pi-gomemory/pkg/installer	47.150s

## Siguiente
Cerrada con aprobación sobre la revisión 582dc5492cf5.

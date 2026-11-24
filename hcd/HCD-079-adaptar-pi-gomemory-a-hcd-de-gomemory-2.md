# Adaptar pi-gomemory a HCD de gomemory 2.59.0 (stale como progreso)

ID: HCD-079
Tipo: evolution
Carril: fast
Motivo: Evolución acotada con decisiones ya tomadas por la persona (Regla A de HCD-064); sin decisiones abiertas ni contratos de datos

## Intención
El panel /tasks no alerta por tareas stale mientras la unidad está en implement, las skills hcd-verify/hcd-implement y el protocolo inyectado reflejan mem hcd verify --stale sin comando, alineados con gomemory 2.59.0


## Alcance
Incluido: `taskVisual`/`renderTasksTree` en extensions/gomemory-tasks.ts reciben el stage de la unidad: stale en `implement` se pinta como progreso (● atenuado, sin warning); en `verify`/`close` se mantiene ↻ warning. `resolveActiveTaskId` sin cambios (ya prioriza pending). Sincronizar skills/hcd-verify y skills/hcd-implement con las plantillas de gomemory c93d427 y ampliar hcd072_skills_test.go con las reglas nuevas. Actualizar el protocolo de extensions/gomemory-context.ts (`mem hcd verify --stale` sin comando; stale con pendientes no exige reverificar). Validación contra el binario real mem 2.59.0, CHANGELOG y preparación de la release v1.29.0 sin publicarla.
Excluido: pin de versión mínima de mem (no existe ninguno en código, tests ni docs vigentes; las guías HTML históricas se dejan como están), el resto de skills HCD (idénticas a las plantillas), publicar la release sin aprobación de la persona.

## ¿Qué pasa si no lo hacemos?
Con gomemory 2.59.0 el panel /tasks marca en amarillo ↻ tareas que el motor considera progreso durante `implement`, y las skills/protocolo siguen pidiendo revalidar con una regresión tras `--`, contradiciendo al motor y empujando al agente a reverificaciones innecesarias.

## Alternativa más barata
Solo sincronizar las skills y dejar el panel como está: más barato, pero el panel seguiría alertando de algo que el motor ya no considera alerta (incoherencia visible para la persona). Se descarta.


## Aclaraciones
- [x] Q1. La regresión (tarea 7) falla por 2 tests ya rotos en v1.28.0 ajenos a HCD-079: installer_test.go:187 espera 21 skills (hay 20 tras retirar codebase-memory) y hcd048_test.go:160 asume que chmod 0500 bloquea la escritura, falso como root. ¿Cómo se tratan? → A — arreglar ambos tests dentro de HCD-079 como tareas 8 y 9 (21→20 skills; TestHCD048H03 se omite como root)
- [x] Q2. La regresión revela 3 tests más ya rotos en v1.28.0 (comprobado en worktree de HEAD) que aún esperan codebase-memory, retirado a propósito en v1.28.0: TestHCD036C01 (hcd036_symlink_test.go:112), TestHCD046Uninstall/propio_tras_codemode_on (hcd046_test.go:152) y TestHCD051LiteModeAndCodegraphGate (hcd051_test.go:350). ¿Se arreglan en HCD-079 quitando codebase-memory de lo esperado? → A — arreglar los 3 tests en HCD-079 como tareas 10 a 12, quitando codebase-memory de lo esperado

## Checklist
- [x] 1. Pintar como progreso (● atenuado, sin warning) una tarea stale cuando la unidad está en stage implement en extensions/gomemory-tasks.ts → `node --test --test-name-pattern "HCD-079 stale en implement" tests/gomemory-tasks.test.mjs`
- [x] 2. Mantener ↻ warning en extensions/gomemory-tasks.ts para una tarea stale cuando la unidad está en verify/close (guarda de comportamiento existente: sin rojo posible) → `node --test --test-name-pattern "HCD-079 stale en verify" tests/gomemory-tasks.test.mjs`
- [x] 3. Sincronizar skills/hcd-verify/SKILL.md y skills/hcd-implement/SKILL.md con las plantillas de gomemory 2.59.0 → `go test -run '^TestHCD072EmbeddedScheduling$' .`
- [x] 4. Actualizar el protocolo HCD inyectado por gomemory-context.ts a `mem hcd verify --stale` sin comando y stale con pendientes como progreso → `go test -run '^TestHCD079ContextProtocol$' .`
- [x] 5. Validar el panel /tasks contra el binario real mem 2.59.0 en un repo temporal desde ./tests/gomemory-tasks.test.mjs (stale sin alerta en implement; `verify --stale` sin comando la deja verificada) → `node --test --test-name-pattern "HCD-079 mem real" tests/gomemory-tasks.test.mjs`
- [x] 6. Documentar el cambio en CHANGELOG y docs/releases/v1.29.0.md (documentación: sin rojo automatizable) → `rg -n "1\.29\.0" CHANGELOG.md docs/releases/v1.29.0.md`
- [x] 7. Regresión completa (regresión final: sin rojo) → `go vet ./... && go test ./... && node --test tests/*.mjs tests/antigravity-prompt-guard/*.test.mjs`
- [x] 8. Ajustar a 20 las skills esperadas en installer_test.go tras retirar codebase-memory en v1.28.0 (Q1) → `go test -count=1 ./pkg/installer -run '^TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean$'`
- [x] 9. Omitir TestHCD048H03UninstallMarkerOnFail como root, donde chmod 0500 no impide escribir (Q1) → `go test -count=1 -v ./pkg/installer -run '^TestHCD048H03UninstallMarkerOnFail$'`
- [x] 10. Quitar codebase-memory de los servidores esperados en el mcp.json enlazado (Q2) → `go test -count=1 ./pkg/installer -run '^TestHCD036C01$'`
- [x] 11. Quitar mcpServers.codebase-memory de lo que uninstall debe retirar tras codemode on (Q2) → `go test -count=1 ./pkg/installer -run '^TestHCD046Uninstall$'`
- [x] 12. Esperar 2 servidores (gomemory, codegraph) al salir del modo lite (Q2) → `go test -count=1 ./pkg/installer -run '^TestHCD051LiteModeAndCodegraphGate$'`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-24 15:20 -05:00 · tarea 1 · rojo observado · caducada · `node --test --test-name-pattern "HCD-079 stale en implement" tests/gomemory-tasks.test.mjs` (salida 1)
        name: 'AssertionError'
        expected: 'warning'
        actual: 'warning'
        operator: 'notStrictEqual'
        stack: |-
          TestContext.<anonymous> (file:///home/admindocker/data/pi-gomemory/tests/gomemory-tasks.test.mjs:72:9)
          Test.runInAsyncScope (node:async_hooks:214:14)
          Test.run (node:internal/test_runner/test:1047:25)
          Test.start (node:internal/test_runner/test:944:17)
          startSubtestAfterBootstrap (node:internal/test_runner/harness:296:17)
        ...
      1..1
      # tests 1
      # suites 0
      # pass 0
      # fail 1
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 93.483075
- 2026-11-24 15:21 -05:00 · tarea 1 · verde · caducada · `node --test --test-name-pattern "HCD-079 stale en implement" tests/gomemory-tasks.test.mjs` (salida 0)
      TAP version 13
      # Subtest: HCD-079 stale en implement: se pinta como progreso, sin warning ni ↻
      ok 1 - HCD-079 stale en implement: se pinta como progreso, sin warning ni ↻
        ---
        duration_ms: 2.01085
        type: 'test'
        ...
      1..1
      # tests 1
      # suites 0
      # pass 1
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 122.424659
- 2026-11-24 15:21 -05:00 · tarea 2 · verde · caducada · `node --test --test-name-pattern "HCD-079 stale en verify|HCD-079 stale en close|stale inactiva en verify" tests/gomemory-tasks.test.mjs` (salida 0)
      ok 2 - HCD-079 stale en close: mantiene ↻ warning
        ---
        duration_ms: 0.132709
        type: 'test'
        ...
      # Subtest: renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)
      ok 3 - renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)
        ---
        duration_ms: 0.550453
        type: 'test'
        ...
      1..3
      # tests 3
      # suites 0
      # pass 3
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 97.116179
- 2026-11-24 15:22 -05:00 · tarea 3 · rojo observado · caducada · `go test -count=1 -run '^TestHCD072EmbeddedScheduling$' .` (salida 1)
      --- FAIL: TestHCD072EmbeddedScheduling (0.18s)
          hcd072_skills_test.go:25: hcd-implement missing cuentan como progreso
          hcd072_skills_test.go:25: hcd-implement missing en una sola llamada
          hcd072_skills_test.go:25: hcd-implement missing mem hcd verify --stale --json --client <agente>
          hcd072_skills_test.go:25: hcd-verify missing cuentan como progreso
          hcd072_skills_test.go:25: hcd-verify missing en una sola llamada
          hcd072_skills_test.go:25: hcd-verify missing mem hcd verify --stale --json --client <agente>
      FAIL
      FAIL	pi-gomemory	0.188s
      FAIL
- 2026-11-24 15:22 -05:00 · tarea 3 · verde · caducada · `go test -count=1 -run '^TestHCD072EmbeddedScheduling$' .` (salida 0)
      ok  	pi-gomemory	0.180s
- 2026-11-24 15:22 -05:00 · tarea 4 · rojo observado · caducada · `go test -count=1 -run '^TestHCD079ContextProtocol$' .` (salida 1)
      --- FAIL: TestHCD079ContextProtocol (0.00s)
          main_hcd071_test.go:40: falta "una tarea caducada (stale) cuenta como progreso" en el protocolo de gomemory-context.ts
      FAIL
      FAIL	pi-gomemory	0.007s
      FAIL
- 2026-11-24 15:22 -05:00 · tarea 4 · verde · caducada · `go test -count=1 -run '^TestHCD079ContextProtocol$' .` (salida 0)
      ok  	pi-gomemory	0.003s
- 2026-11-24 15:23 -05:00 · tarea 5 · rojo observado · caducada · `node --test --test-name-pattern "HCD-079 mem real" tests/gomemory-tasks.test.mjs` (salida 1)
        code: 'ERR_TEST_FAILURE'
        name: 'SyntaxError'
        stack: |-
          JSON.parse (<anonymous>)
          mem (file:///home/admindocker/data/pi-gomemory/tests/gomemory-tasks.test.mjs:145:33)
          TestContext.<anonymous> (file:///home/admindocker/data/pi-gomemory/tests/gomemory-tasks.test.mjs:170:3)
          Test.runInAsyncScope (node:async_hooks:214:14)
          Test.run (node:internal/test_runner/test:1047:25)
          Test.start (node:internal/test_runner/test:944:17)
          startSubtestAfterBootstrap (node:internal/test_runner/harness:296:17)
        ...
      1..1
      # tests 1
      # suites 0
      # pass 0
      # fail 1
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 252.143002
- 2026-11-24 15:23 -05:00 · tarea 5 · rojo observado · caducada · `node --test --test-name-pattern "HCD-079 mem real" tests/gomemory-tasks.test.mjs` (salida 1)
        name: 'AssertionError'
        expected: 'warning'
        actual: 'warning'
        operator: 'notStrictEqual'
        stack: |-
          TestContext.<anonymous> (file:///home/admindocker/data/pi-gomemory/tests/gomemory-tasks.test.mjs:184:10)
          Test.runInAsyncScope (node:async_hooks:214:14)
          Test.run (node:internal/test_runner/test:1047:25)
          Test.start (node:internal/test_runner/test:944:17)
          startSubtestAfterBootstrap (node:internal/test_runner/harness:296:17)
        ...
      1..1
      # tests 1
      # suites 0
      # pass 0
      # fail 1
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 338.495222
- 2026-11-24 15:23 -05:00 · tarea 5 · falló · caducada · `node --test --test-name-pattern "HCD-079 mem real" tests/gomemory-tasks.test.mjs` (salida 1)
        name: 'AssertionError'
        expected: 'warning'
        actual: 'accent'
        operator: 'strictEqual'
        stack: |-
          TestContext.<anonymous> (file:///home/admindocker/data/pi-gomemory/tests/gomemory-tasks.test.mjs:193:10)
          Test.runInAsyncScope (node:async_hooks:214:14)
          Test.run (node:internal/test_runner/test:1047:25)
          Test.start (node:internal/test_runner/test:944:17)
          startSubtestAfterBootstrap (node:internal/test_runner/harness:296:17)
        ...
      1..1
      # tests 1
      # suites 0
      # pass 0
      # fail 1
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 331.760415
- 2026-11-24 15:23 -05:00 · tarea 5 · verde · caducada · `node --test --test-name-pattern "HCD-079 mem real" tests/gomemory-tasks.test.mjs` (salida 0)
      TAP version 13
      # Subtest: HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
      ok 1 - HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
        ---
        duration_ms: 347.140911
        type: 'test'
        ...
      1..1
      # tests 1
      # suites 0
      # pass 1
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 474.444588
- 2026-11-24 15:24 -05:00 · tarea 6 · verde · caducada · `rg -n "1\.29\.0" CHANGELOG.md docs/releases/v1.29.0.md` (salida 0)
      CHANGELOG.md:7:## [v1.29.0] - 2026-11-24
      docs/releases/v1.29.0.md:1:# Notas de v1.29.0
- 2026-11-24 15:24 -05:00 · tarea 7 · falló · caducada · `go vet ./... && go test ./... && node --test tests/*.mjs tests/antigravity-prompt-guard/*.test.mjs` (salida 1)
      --- FAIL: TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean (0.12s)
          installer_test.go:187: skills instaladas = 20, se esperan 21
      --- FAIL: TestHCD048H03UninstallMarkerOnFail (0.02s)
          hcd048_test.go:160: expected Uninstall to fail when themesDir is read-only; output:
              Plan de desinstalación:
                [retirar] /tmp/TestHCD048H03UninstallMarkerOnFail1490101704/001/.pi/agent/themes/gomemory.json — instalado por pi-gomemory
                [retirar] /tmp/TestHCD048H03UninstallMarkerOnFail1490101704/001/.pi-gomemory/manifest.json — manifiesto de instalación (se retira si no hay fallos)
              Resumen: 2 retirados, 0 conservados, 0 omitidos, 0 fallidos.
              Se conservan pi, mem, codegraph y herdr: son herramientas independientes que otros agentes pueden usar. Para quitarlas:
                pi:                  rm -rf ~/.pi/agent/install ~/.pi/agent/bin/pi (y su enlace en ~/.local/bin)
                mem:                 curl -fsSL https://raw.githubusercontent.com/Sayoner-000/gomemory/main/scripts/install.sh | bash -s -- --uninstall
                codegraph:           curl -fsSL https://raw.githubusercontent.com/colbymchenry/codegraph/main/install.sh | sh -s -- --uninstall
                herdr:               rm "$(command -v herdr)" (herdr no tiene subcomando de desinstalación)
              ✓ pi-gomemory desinstalado. Para reinstalar: pi-gomemory-setup install
      --- FAIL: TestHCD057Glyphs (1.58s)
          hcd057_test.go:95: TestHCD057Glyphs: FAIL tarea 3 tono dim, se esperaba warning | tarea 3 sin glifo ↻
      FAIL
      FAIL	pi-gomemory/pkg/installer	36.483s
      FAIL
- 2026-11-24 15:31 -05:00 · tarea 8 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run '^TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean$'` (salida 1)
           [38;2;145;169;197m/tmp/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean649102418/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Modo codemode                    [38;2;145;169;197mhíbrido (codemode on)[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Thinking colapsado (Ctrl+T)      [38;2;145;169;197mhideThinkingBlock activo en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m20 skills cargadas[0m
        [38;2;145;169;197m→[0m Skills de gomemory al día        [38;2;145;169;197mmem no instaló skills en ningún agente; Pi usa las embebidas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 20 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/tmp/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean649102418/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- FAIL: TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean (0.17s)
          installer_test.go:187: skills instaladas = 20, se esperan 21
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.169s
      FAIL
- 2026-11-24 15:31 -05:00 · tarea 8 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.174s
- 2026-11-24 15:32 -05:00 · tarea 8 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.194s
- 2026-11-24 15:32 -05:00 · tarea 9 · rojo observado · caducada · `go test -count=1 -v ./pkg/installer -run '^TestHCD048H03UninstallMarkerOnFail$'` (salida 1)
      === RUN   TestHCD048H03UninstallMarkerOnFail
      === PAUSE TestHCD048H03UninstallMarkerOnFail
      === CONT  TestHCD048H03UninstallMarkerOnFail
          hcd048_test.go:160: expected Uninstall to fail when themesDir is read-only; output:
              Plan de desinstalación:
                [retirar] /tmp/TestHCD048H03UninstallMarkerOnFail4120142142/001/.pi/agent/themes/gomemory.json — instalado por pi-gomemory
                [retirar] /tmp/TestHCD048H03UninstallMarkerOnFail4120142142/001/.pi-gomemory/manifest.json — manifiesto de instalación (se retira si no hay fallos)
              Resumen: 2 retirados, 0 conservados, 0 omitidos, 0 fallidos.
              Se conservan pi, mem, codegraph y herdr: son herramientas independientes que otros agentes pueden usar. Para quitarlas:
                pi:                  rm -rf ~/.pi/agent/install ~/.pi/agent/bin/pi (y su enlace en ~/.local/bin)
                mem:                 curl -fsSL https://raw.githubusercontent.com/Sayoner-000/gomemory/main/scripts/install.sh | bash -s -- --uninstall
                codegraph:           curl -fsSL https://raw.githubusercontent.com/colbymchenry/codegraph/main/install.sh | sh -s -- --uninstall
                herdr:               rm "$(command -v herdr)" (herdr no tiene subcomando de desinstalación)
              ✓ pi-gomemory desinstalado. Para reinstalar: pi-gomemory-setup install
      --- FAIL: TestHCD048H03UninstallMarkerOnFail (0.01s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.011s
      FAIL
- 2026-11-24 15:32 -05:00 · tarea 9 · verde · caducada · `go test -count=1 -v ./pkg/installer -run '^TestHCD048H03UninstallMarkerOnFail$'` (salida 0)
      === RUN   TestHCD048H03UninstallMarkerOnFail
      === PAUSE TestHCD048H03UninstallMarkerOnFail
      === CONT  TestHCD048H03UninstallMarkerOnFail
          hcd048_test.go:140: como root los permisos de solo lectura no bloquean la escritura
      --- SKIP: TestHCD048H03UninstallMarkerOnFail (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.003s
- 2026-11-24 15:32 -05:00 · tarea 7 · falló · caducada · `go vet ./... && go test ./... && node --test tests/*.mjs tests/antigravity-prompt-guard/*.test.mjs` (salida 1)
      opcional, no instalado (curl -fsSL https://herdr.dev/install.sh | sh)[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/tmp/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2809209937/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Modo codemode                    [38;2;145;169;197mhíbrido (codemode on)[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Thinking colapsado (Ctrl+T)      [38;2;145;169;197mhideThinkingBlock activo en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m20 skills cargadas[0m
        [38;2;145;169;197m→[0m Skills de gomemory al día        [38;2;145;169;197mmem no instaló skills en ningún agente; Pi usa las embebidas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 20 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/tmp/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2809209937/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	32.753s
      FAIL
- 2026-11-24 15:35 -05:00 · tarea 10 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD036C01$'` (salida 1)
                    "args": [
                      "serve",
                      "--mcp"
                    ],
                    "exposure": "direct",
                    "description": "CodeGraph SQLite AST code intelligence and structural exploration"
                  },
                  "gomemory": {
                    "command": "mem",
                    "args": [
                      "mcp"
                    ],
                    "exposure": "direct",
                    "description": "Gomemory persistent project memory, semantic graphs, checkpoints, HCD and review protocol"
                  }
                }
              }
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.017s
      FAIL
- 2026-11-24 15:35 -05:00 · tarea 10 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD036C01$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.008s
- 2026-11-24 15:35 -05:00 · tarea 11 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD046Uninstall$'` (salida 1)
      4;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Modo codemode                    [38;2;145;169;197mhíbrido (codemode on)[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Thinking colapsado (Ctrl+T)      [38;2;145;169;197mhideThinkingBlock activo en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m20 skills cargadas[0m
        [38;2;145;169;197m→[0m Skills de gomemory al día        [38;2;145;169;197mmem no instaló skills en ningún agente; Pi usa las embebidas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 20 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/tmp/TestHCD046Uninstall+codemode_ajeno_se_conserva84238304/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- FAIL: TestHCD046Uninstall (0.35s)
          --- FAIL: TestHCD046Uninstall/propio_tras_codemode_on (0.20s)
              hcd046_test.go:161: /tmp/TestHCD046Uninstallpropio_tras_codemode_on3367649247/001/.pi/agent/mcp.json › mcpServers.codebase-memory: quiero retirar, obtengo <nil>
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.354s
      FAIL
- 2026-11-24 15:35 -05:00 · tarea 11 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD046Uninstall$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.340s
- 2026-11-24 15:38 -05:00 · tarea 12 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD051LiteModeAndCodegraphGate$'` (salida 1)
                    "command": "codegraph",
                    "args": [
                      "serve",
                      "--mcp"
                    ],
                    "exposure": "direct",
                    "description": "CodeGraph SQLite AST code intelligence and structural exploration"
                  },
                  "gomemory": {
                    "command": "mem",
                    "args": [
                      "mcp"
                    ],
                    "exposure": "direct",
                    "description": "Gomemory persistent project memory, semantic graphs, checkpoints, HCD and review protocol"
                  }
                }
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.026s
      FAIL
- 2026-11-24 15:38 -05:00 · tarea 12 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD051LiteModeAndCodegraphGate$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.015s
- 2026-11-24 15:38 -05:00 · tarea 7 · verde · caducada · `go vet ./... && go test -count=1 ./... && node --test tests/*.mjs tests/antigravity-prompt-guard/*.test.mjs` (salida 0)
      ok 37 - renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)
        ---
        duration_ms: 0.728793
        type: 'test'
        ...
      # Subtest: HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
      ok 38 - HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
        ---
        duration_ms: 244.857122
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
      # duration_ms 390.84609
- 2026-11-24 15:39 -05:00 · tarea 1 · verde · caducada · `node --test --test-name-pattern "HCD-079 stale en implement" tests/gomemory-tasks.test.mjs` (salida 0)
      TAP version 13
      # Subtest: HCD-079 stale en implement: se pinta como progreso, sin warning ni ↻
      ok 1 - HCD-079 stale en implement: se pinta como progreso, sin warning ni ↻
        ---
        duration_ms: 1.360537
        type: 'test'
        ...
      1..1
      # tests 1
      # suites 0
      # pass 1
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 108.521944
- 2026-11-24 15:39 -05:00 · tarea 2 · verde · caducada · `node --test --test-name-pattern "HCD-079 stale en verify|HCD-079 stale en close|stale inactiva en verify" tests/gomemory-tasks.test.mjs` (salida 0)
      ok 2 - HCD-079 stale en close: mantiene ↻ warning
        ---
        duration_ms: 0.124725
        type: 'test'
        ...
      # Subtest: renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)
      ok 3 - renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)
        ---
        duration_ms: 0.678421
        type: 'test'
        ...
      1..3
      # tests 3
      # suites 0
      # pass 3
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 107.148181
- 2026-11-24 15:39 -05:00 · tarea 3 · verde · caducada · `go test -count=1 -run '^TestHCD072EmbeddedScheduling$' .` (salida 0)
      ok  	pi-gomemory	0.227s
- 2026-11-24 15:39 -05:00 · tarea 4 · verde · caducada · `go test -count=1 -run '^TestHCD079ContextProtocol$' .` (salida 0)
      ok  	pi-gomemory	0.004s
- 2026-11-24 15:39 -05:00 · tarea 5 · verde · caducada · `node --test --test-name-pattern "HCD-079 mem real" tests/gomemory-tasks.test.mjs` (salida 0)
      TAP version 13
      # Subtest: HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
      ok 1 - HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
        ---
        duration_ms: 284.085676
        type: 'test'
        ...
      1..1
      # tests 1
      # suites 0
      # pass 1
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 407.570658
- 2026-11-24 15:39 -05:00 · tarea 6 · verde · caducada · `rg -n "1\.29\.0" CHANGELOG.md docs/releases/v1.29.0.md` (salida 0)
      docs/releases/v1.29.0.md:1:# Notas de v1.29.0
      CHANGELOG.md:7:## [v1.29.0] - 2026-11-24
- 2026-11-24 15:39 -05:00 · tarea 8 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.181s
- 2026-11-24 15:39 -05:00 · tarea 9 · verde · caducada · `go test -count=1 -v ./pkg/installer -run '^TestHCD048H03UninstallMarkerOnFail$'` (salida 0)
      === RUN   TestHCD048H03UninstallMarkerOnFail
      === PAUSE TestHCD048H03UninstallMarkerOnFail
      === CONT  TestHCD048H03UninstallMarkerOnFail
          hcd048_test.go:140: como root los permisos de solo lectura no bloquean la escritura
      --- SKIP: TestHCD048H03UninstallMarkerOnFail (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.004s
- 2026-11-24 15:39 -05:00 · tarea 10 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD036C01$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.011s
- 2026-11-24 15:39 -05:00 · tarea 11 · verde · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD046Uninstall$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.395s
- 2026-11-24 15:40 -05:00 · tarea 1 · verde · vigente · `node --test --test-name-pattern "HCD-079 stale en implement" tests/gomemory-tasks.test.mjs` (salida 0)
      TAP version 13
      # Subtest: HCD-079 stale en implement: se pinta como progreso, sin warning ni ↻
      ok 1 - HCD-079 stale en implement: se pinta como progreso, sin warning ni ↻
        ---
        duration_ms: 1.223935
        type: 'test'
        ...
      1..1
      # tests 1
      # suites 0
      # pass 1
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 105.912185
- 2026-11-24 15:40 -05:00 · tarea 2 · verde · vigente · `node --test --test-name-pattern "HCD-079 stale en verify|HCD-079 stale en close|stale inactiva en verify" tests/gomemory-tasks.test.mjs` (salida 0)
      ok 2 - HCD-079 stale en close: mantiene ↻ warning
        ---
        duration_ms: 0.143175
        type: 'test'
        ...
      # Subtest: renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)
      ok 3 - renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)
        ---
        duration_ms: 0.608416
        type: 'test'
        ...
      1..3
      # tests 3
      # suites 0
      # pass 3
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 108.252597
- 2026-11-24 15:40 -05:00 · tarea 3 · verde · vigente · `go test -count=1 -run '^TestHCD072EmbeddedScheduling$' .` (salida 0)
      ok  	pi-gomemory	0.213s
- 2026-11-24 15:40 -05:00 · tarea 4 · verde · vigente · `go test -count=1 -run '^TestHCD079ContextProtocol$' .` (salida 0)
      ok  	pi-gomemory	0.004s
- 2026-11-24 15:40 -05:00 · tarea 5 · verde · vigente · `node --test --test-name-pattern "HCD-079 mem real" tests/gomemory-tasks.test.mjs` (salida 0)
      TAP version 13
      # Subtest: HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
      ok 1 - HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
        ---
        duration_ms: 245.214733
        type: 'test'
        ...
      1..1
      # tests 1
      # suites 0
      # pass 1
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 353.19618
- 2026-11-24 15:40 -05:00 · tarea 6 · verde · vigente · `rg -n "1\.29\.0" CHANGELOG.md docs/releases/v1.29.0.md` (salida 0)
      CHANGELOG.md:7:## [v1.29.0] - 2026-11-24
      docs/releases/v1.29.0.md:1:# Notas de v1.29.0
- 2026-11-24 15:40 -05:00 · tarea 7 · verde · vigente · `go vet ./... && go test -count=1 ./... && node --test tests/*.mjs tests/antigravity-prompt-guard/*.test.mjs` (salida 0)
      ok 37 - renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)
        ---
        duration_ms: 0.671962
        type: 'test'
        ...
      # Subtest: HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
      ok 38 - HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
        ---
        duration_ms: 249.448966
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
      # duration_ms 441.401787
- 2026-11-24 15:40 -05:00 · tarea 8 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.172s
- 2026-11-24 15:40 -05:00 · tarea 9 · verde · vigente · `go test -count=1 -v ./pkg/installer -run '^TestHCD048H03UninstallMarkerOnFail$'` (salida 0)
      === RUN   TestHCD048H03UninstallMarkerOnFail
      === PAUSE TestHCD048H03UninstallMarkerOnFail
      === CONT  TestHCD048H03UninstallMarkerOnFail
          hcd048_test.go:140: como root los permisos de solo lectura no bloquean la escritura
      --- SKIP: TestHCD048H03UninstallMarkerOnFail (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.004s
- 2026-11-24 15:40 -05:00 · tarea 10 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD036C01$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.012s
- 2026-11-24 15:40 -05:00 · tarea 11 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD046Uninstall$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.403s
- 2026-11-24 15:40 -05:00 · tarea 12 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD051LiteModeAndCodegraphGate$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.020s

## Siguiente
Cerrada con aprobación sobre la revisión 1ef2b70b2d6a.

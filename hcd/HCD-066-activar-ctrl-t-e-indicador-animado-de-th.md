# Activar Ctrl+T e indicador animado de thinking por defecto y exigir impacto con grafos en /plan

ID: HCD-066
Tipo: feature
Carril: fast
Motivo: Feature acotada con las 3 decisiones ya cerradas por la persona (1A: hideThinkingBlock=true sin tocar defaultThinkingLevel; 2A: gotitas en glifos de texto + etiqueta (thinking...); 3B: impacto con codegraph y codebase-memory en /plan permitiendo index_repository externo sin tocar archivos del repo).

## Intención
1) Configurar hideThinkingBlock=true por defecto al instalar pi-gomemory (respetando preferencia explícita previa en settings.json), reportarlo en doctor y personalizar en Pi el indicador animado de thinking con gotitas en glifos de texto y etiqueta (thinking...) compatible con pi-cc-extensions. 2) Reforzar /plan (extensions/gomemory-plan.ts, gomemory-plan-patterns.json y gomemory-context.ts) para exigir análisis de impacto previo a HCD con codegraph y codebase-memory, bloqueando mutaciones de código y estado HCD mientras permite consultas de grafos e index_repository externo sin persistencia en el repo.


## Alcance
Incluido: fijar `hideThinkingBlock: true` por defecto en `~/.pi/agent/settings.json` desde `InstallTheme` (sin pisar preferencia previa del usuario) y reportarlo en `RunDoctor` (1A); configurar en `extensions/gomemory-status.ts` el indicador animado de gotitas corriendo (`setWorkingIndicator`, glifos de texto puro) y etiqueta `(thinking...)` (`setHiddenThinkingLabel`) en `session_start` con limpieza en `session_shutdown` (2A); exigir validación de impacto previo a HCD con `codegraph` y `codebase-memory` en `extensions/gomemory-plan.ts` y `extensions/gomemory-context.ts`, bloqueando mutaciones de código, HCD y grafos persistentes mientras permite lecturas e `index_repository` externo (3B); pruebas rojo→verde y verificación en Pi real.
Excluido: modificar `defaultThinkingLevel` (opción 1A), usar emojis en el indicador (opción 2A), desactivar features de `pi-cc-extensions` (verificado sin conflicto), o publicar nueva release/tag.

## ¿Qué pasa si no lo hacemos?
Al instalar `pi-gomemory`, Pi muestra los bloques completos de razonamiento saturando la pantalla hasta que la persona pulsa `Ctrl+T` manualmente, sin la identidad visual animada de `gomemory`; además, entrar en `/plan` antes de un HCD no exige mapear el impacto real con `codegraph` y `codebase-memory` ni impide mutaciones HCD o escrituras accidentales como `index_repository(persistence: true)` o `codegraph index`.

## Alternativa más barata
Documentar ambos comportamientos solo en el README o en `AGENTS.md`. Esto no resuelve el problema porque `settings.json` seguiría sin `hideThinkingBlock: true` al instalar, no habría indicador animado en Pi, y el modo `/plan` no tendría ni el protocolo inyectado en el prompt ni la barrera técnica de solo lectura para mutaciones HCD/grafo. Decisiones previas respetadas: HCD-020 Q3 (glifos de texto sin emojis), HCD-045 (sección `gomemory` en `systemPromptOptions.sections`), HCD-057 (convivencia con `pi-cc-extensions`). Normas aplicables: Constitución §0 y Playbook HCD-FLOW-01 / HCD-DEF-01.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Configurar `hideThinkingBlock: true` por defecto en `InstallTheme` (sin pisar un valor previo explícito del usuario en `settings.json`) y reportarlo en `RunDoctor` (`pkg/installer/theme.go`, `pkg/installer/doctor.go`) → `go test -race ./pkg/installer -run TestHCD066ThinkingDefaultsAndDoctor -count=1`
- [x] 2. Configurar en `extensions/gomemory-status.ts` el indicador animado de gotitas corriendo (`setWorkingIndicator`, glifos de texto puro) y la etiqueta `(thinking...)` (`setHiddenThinkingLabel`) en `session_start`, restaurándolos en `session_shutdown` → `go test -race ./pkg/installer -run TestHCD066ThinkingIndicator -count=1`
- [x] 3. Añadir en `extensions/gomemory-plan.ts` (en `loadMethod` y `FALLBACK`) y en `extensions/gomemory-context.ts` el protocolo obligatorio de validación de impacto previo a HCD con `codegraph` y `codebase-memory` → `go test -race ./pkg/installer -run TestHCD066PlanImpactProtocol -count=1`
- [x] 4. Blindar `extensions/gomemory-plan.ts` y `extensions/gomemory-plan-patterns.json` para bloquear mutaciones de código, HCD y grafos persistentes (`delete_project`, `manage_adr` escritura, `index_repository(persistence:true)`, `mem hcd start|verify|close|...`, `codegraph index`) manteniendo permitidas las lecturas e `index_repository` externo → `node --test tests/gomemory-plan.test.mjs && go test -race ./pkg/installer -run TestHCD066PlanReadOnlyGuards -count=1`
- [x] 5. Regresión completa (sin rojo previo: `go vet`, `go test -race`, `bash -n`, `node --test`) → `go vet ./... && go test -race ./... && bash -n install.sh scripts/*.sh && node --check extensions/antigravity-prompt-guard.js && node --test tests/gomemory-plan.test.mjs tests/antigravity-prompt-guard/guard.test.mjs`
- [x] 6. Verificación en Pi real instalado (sin rojo previo: comprueba `hideThinkingBlock: true`, carga limpia de `gomemory-status.ts` y `gomemory-plan.ts` en Pi 1.6.0 real) → observación directa sobre Pi real
- [x] 7. Elevar la validación previa de impacto con `codegraph` y `codebase-memory` a regla general siempre activa (independiente de `/plan`) en `extensions/gomemory-context.ts` para cualquier unidad HCD o solicitud de cambio → `go test -race ./pkg/installer -run TestHCD066AlwaysOnImpactProtocol -count=1`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-21 07:41 -05:00 · tarea 1 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD066ThinkingDefaultsAndDoctor -count=1` (salida 1)
      --- FAIL: TestHCD066ThinkingDefaultsAndDoctor (0.00s)
          --- FAIL: TestHCD066ThinkingDefaultsAndDoctor/default_true_on_fresh_settings (0.03s)
              hcd066_test.go:33: hideThinkingBlock = <nil>, esperado true
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.498s
      FAIL
- 2026-11-21 07:42 -05:00 · tarea 1 · verde · caducada · `go test -race ./pkg/installer -run TestHCD066ThinkingDefaultsAndDoctor -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.460s
- 2026-11-21 07:42 -05:00 · tarea 2 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD066ThinkingIndicator -count=1` (salida 1)
      --- FAIL: TestHCD066ThinkingIndicator (0.26s)
          hcd066_test.go:146: TestHCD066ThinkingIndicator: FAIL etiqueta inicial inesperada: [] | frames insuficientes: undefined | falta manejador session_shutdown
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.547s
      FAIL
- 2026-11-21 07:43 -05:00 · tarea 2 · verde · caducada · `go test -race ./pkg/installer -run TestHCD066ThinkingIndicator -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.526s
- 2026-11-21 07:43 -05:00 · tarea 3 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD066PlanImpactProtocol -count=1` (salida 1)
      --- FAIL: TestHCD066PlanImpactProtocol (0.25s)
          hcd066_test.go:207: TestHCD066PlanImpactProtocol: FAIL loadMethod omite codegraph_explore | loadMethod omite codebase-memory | loadMethod omite search_graph | loadMethod omite trace_path | loadMethod omite check_index_coverage | loadMethod omite index_repository | loadMethod omite Mapa de impacto | FALLBACK omite codegraph_explore | FALLBACK omite codebase-memory | FALLBACK omite search_graph | FALLBACK omite trace_path | FALLBACK omite check_index_coverage | FALLBACK omite index_repository | FALLBACK omite Mapa de impacto | piBlock no menciona validación de impacto con codegraph y codebase-memory en Modo plan
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.539s
      FAIL
- 2026-11-21 07:44 -05:00 · tarea 3 · verde · caducada · `go test -race ./pkg/installer -run TestHCD066PlanImpactProtocol -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.508s
- 2026-11-21 07:44 -05:00 · tarea 4 · rojo observado · caducada · `node --test tests/gomemory-plan.test.mjs && go test -race ./pkg/installer -run TestHCD066PlanReadOnlyGuards -count=1` (salida 1)
      ✖ failing tests:
      test at tests/gomemory-plan.test.mjs:110:1
      ✖ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (1.64825ms)
        AssertionError [ERR_ASSERTION]: debía bloquear delete_project {"project":"p"}
        + actual - expected
        + undefined
        - true
            at TestContext.<anonymous> (file:///Users/josegomezj/home/rcw/pi-gomemory/tests/gomemory-plan.test.mjs:151:10)
            at async Test.run (node:internal/test_runner/test:1409:7)
            at async Test.processPendingSubtests (node:internal/test_runner/test:974:7) {
          generatedMessage: false,
          code: 'ERR_ASSERTION',
          actual: undefined,
          expected: true,
          operator: 'strictEqual',
          diff: 'simple'
        }
- 2026-11-21 07:45 -05:00 · tarea 4 · verde · caducada · `node --test tests/gomemory-plan.test.mjs && go test -race ./pkg/installer -run TestHCD066PlanReadOnlyGuards -count=1` (salida 0)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (3.414708ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (0.933ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.175167ms)
      ✔ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (0.961208ms)
      ℹ tests 4
      ℹ suites 0
      ℹ pass 4
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 260.36725
      ok  	pi-gomemory/pkg/installer	1.666s
- 2026-11-21 07:45 -05:00 · tarea 5 · falló · caducada · `go vet ./... && go test -race ./... && bash -n install.sh scripts/*.sh && node --check extensions/antigravity-prompt-guard.js && node --test tests/gomemory-plan.test.mjs tests/antigravity-prompt-guard/guard.test.mjs` (salida 1)
      ;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean756402752/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Thinking colapsado (Ctrl+T)      [38;2;145;169;197mhideThinkingBlock activo en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean756402752/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- FAIL: TestPlanPatterns (0.07s)
          hcd002_test.go:143: tools = [edit write powershell delete_project mcp__codebase_memory__delete_project]
      FAIL
      FAIL	pi-gomemory/pkg/installer	49.640s
      FAIL
- 2026-11-21 07:46 -05:00 · tarea 5 · verde · caducada · `go vet ./... && go test -race ./... && bash -n install.sh scripts/*.sh && node --check extensions/antigravity-prompt-guard.js && node --test tests/gomemory-plan.test.mjs tests/antigravity-prompt-guard/guard.test.mjs` (salida 0)
      ✔ passes through malformed or nontext input [] (0.060917ms)
      ✔ passes through malformed or nontext input {} (0.112708ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":null} (0.045625ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":{"systemInstruction":{"parts":[{"text":"text"},{"inlineData":{}}]},"contents":[]}} (0.06275ms)
      ✔ passes through blank system text (0.133417ms)
      ✔ passes through malformed conversation turns (0.066083ms)
      ✔ does not discard non-text system metadata (0.055292ms)
      ✔ registers a working Pi hook scoped to Antigravity (0.213625ms)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (4.0155ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (0.910667ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.219833ms)
      ✔ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (0.985208ms)
      ℹ tests 26
      ℹ suites 0
      ℹ pass 26
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 259.220333
- 2026-11-21 07:47 -05:00 · tarea 1,2,3,4 · verde · caducada · `node --test tests/gomemory-plan.test.mjs && go test -race -v ./pkg/installer -run TestHCD066 -count=1` (salida 0)
      === RUN   TestHCD066PlanReadOnlyGuards
      === PAUSE TestHCD066PlanReadOnlyGuards
      === CONT  TestHCD066ThinkingIndicator
      === CONT  TestHCD066PlanReadOnlyGuards
      === CONT  TestHCD066PlanImpactProtocol
      === CONT  TestHCD066ThinkingDefaultsAndDoctor
      === RUN   TestHCD066ThinkingDefaultsAndDoctor/default_true_on_fresh_settings
      === PAUSE TestHCD066ThinkingDefaultsAndDoctor/default_true_on_fresh_settings
      === RUN   TestHCD066ThinkingDefaultsAndDoctor/preserves_explicit_user_false
      === PAUSE TestHCD066ThinkingDefaultsAndDoctor/preserves_explicit_user_false
      === CONT  TestHCD066ThinkingDefaultsAndDoctor/default_true_on_fresh_settings
      === CONT  TestHCD066ThinkingDefaultsAndDoctor/preserves_explicit_user_false
      --- PASS: TestHCD066PlanImpactProtocol (0.28s)
      --- PASS: TestHCD066ThinkingIndicator (0.29s)
      --- PASS: TestHCD066PlanReadOnlyGuards (0.29s)
      --- PASS: TestHCD066ThinkingDefaultsAndDoctor (0.00s)
          --- PASS: TestHCD066ThinkingDefaultsAndDoctor/preserves_explicit_user_false (0.04s)
          --- PASS: TestHCD066ThinkingDefaultsAndDoctor/default_true_on_fresh_settings (1.28s)
      PASS
      ok  	pi-gomemory/pkg/installer	2.592s
- 2026-11-21 07:48 -05:00 · tarea 6 · manual · caducada
      Binario pi-gomemory-setup compilado desde HEAD+HCD-066 ejecutado en HOME limpio temporal (/tmp/hcd066-pi-real.*): settings.json quedó con theme='gomemory', hideThinkingBlock=true y defaultThinkingLevel=undefined; pi-gomemory-setup doctor mostró '✓ Thinking colapsado (Ctrl+T) hideThinkingBlock activo en settings.json'; Pi 1.6.0 real (--mode rpc --no-session) cargó gomemory-status.ts, gomemory-plan.ts y gomemory-context.ts instalados sin errores y devolvió get_state success=true.
- 2026-11-21 07:51 -05:00 · tarea 7 · rojo observado · caducada · `go test -race ./pkg/installer -run TestHCD066AlwaysOnImpactProtocol -count=1` (salida 1)
      --- FAIL: TestHCD066AlwaysOnImpactProtocol (0.25s)
          hcd066_test.go:295: TestHCD066AlwaysOnImpactProtocol: FAIL falta viñeta dedicada de Impacto obligatorio independiente de /plan
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.695s
      FAIL
- 2026-11-21 07:52 -05:00 · tarea 7 · verde · vigente · `go test -race ./pkg/installer -run TestHCD066AlwaysOnImpactProtocol -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.482s
- 2026-11-21 07:52 -05:00 · tarea 1,2,3,4,5 · falló · vigente · `go vet ./... && go test -race ./... && bash -n install.sh scripts/*.sh && node --check extensions/antigravity-prompt-guard.js && node --test tests/gomemory-plan.test.mjs tests/antigravity-prompt-guard/guard.test.mjs` (salida 1)
      ntrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2877852225/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Thinking colapsado (Ctrl+T)      [38;2;145;169;197mhideThinkingBlock activo en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2877852225/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- FAIL: TestHCD019TDDClickToggleAndDock (0.39s)
          testing.go:1619: TempDir RemoveAll cleanup: unlinkat /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD019TDDClickToggleAndDock3156040745/001: directory not empty
      FAIL
      FAIL	pi-gomemory/pkg/installer	54.058s
      FAIL
- 2026-11-21 07:53 -05:00 · tarea 1,2,3,4,5 · verde · vigente · `go vet ./... && go test -race ./... && bash -n install.sh scripts/*.sh && node --check extensions/antigravity-prompt-guard.js && node --test tests/gomemory-plan.test.mjs tests/antigravity-prompt-guard/guard.test.mjs` (salida 0)
      ✔ passes through malformed or nontext input [] (0.036792ms)
      ✔ passes through malformed or nontext input {} (0.088333ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":null} (0.066958ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":{"systemInstruction":{"parts":[{"text":"text"},{"inlineData":{}}]},"contents":[]}} (0.057667ms)
      ✔ passes through blank system text (0.064917ms)
      ✔ passes through malformed conversation turns (0.051167ms)
      ✔ does not discard non-text system metadata (0.054458ms)
      ✔ registers a working Pi hook scoped to Antigravity (0.135708ms)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (3.064625ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (0.769417ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.210333ms)
      ✔ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (0.974917ms)
      ℹ tests 26
      ℹ suites 0
      ℹ pass 26
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 247.203625
- 2026-11-21 07:55 -05:00 · tarea 6 · manual · vigente
      Re-verificado sobre binario pi-gomemory-setup final (HCD-066 incl. tarea 7) en HOME temporal limpio (/tmp/hcd066-pi-real2.*): gomemory-context.ts instalado incluye 'Impacto obligatorio (siempre, con o sin /plan)', settings.json tiene hideThinkingBlock=true sin defaultThinkingLevel, doctor marca OK y Pi 1.6.0 real (--mode rpc --no-session) cargó gomemory-status.ts, gomemory-plan.ts y gomemory-context.ts sin errores devolviendo get_state success=true.

## Siguiente
Cerrada con aprobación sobre la revisión 17024f71992f.

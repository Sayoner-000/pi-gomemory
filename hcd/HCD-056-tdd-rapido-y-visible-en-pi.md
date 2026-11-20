# TDD rápido y visible en Pi

ID: HCD-056
Tipo: feature
Carril: fast
Motivo: Plan aprobado con causas medidas y sin decisiones interdependientes abiertas; HCD-055 (concurrencia en el motor) se canceló y este enfoque se limita a pi-gomemory

## Intención
Bajar la suite Go de 98 s a menos de 30 s con t.Parallel, permitir el ciclo rojo→verde acotado con un filtro en run_test_suite y en las skills HCD, mostrar en vivo en /tdd los tests lanzados por Bash y evitar que watch bloquee al agente, sin cambiar veredictos.


Evidencia medida el 2026-11-19 (memoria 531): `go test -count=1 ./...` tarda 1 min 38 s de reloj con 42 % de CPU en 10 núcleos. Los 263 tests de nivel superior de `pkg/installer` corren en serie: ninguno usa `t.Parallel()`, y unos 90 s se pueden paralelizar. Además, `run_test_suite` no admite filtro, cada tarea HCD se verifica con la suite completa y el rojo solo es obligatorio en `fix`. El panel ignora la salida de `mem hcd verify` y la de Bash solo llega al terminar. Watch comparte la cola con la tool, así que el agente puede esperar unos 100 s detrás de una ejecución que solo es informativa.

## Alcance
Incluido: añadir `t.Parallel()` a los tests de `pkg/installer` que no modifican el entorno del proceso; añadir el parámetro opcional `filter` a `run_test_suite` y traducirlo por ejecutor (go, pytest, vitest, playwright, phpunit, maven y gradle); exigir en `skills/hcd-tasks` y `skills/hcd-implement` una comprobación acotada por tarea, una única tarea final de regresión completa y el rojo en toda tarea con un test automatizable; ingerir en vivo en /tdd los tests lanzados por Bash a partir de `tool_execution_update`, según la forma que se observe en Pi 1.5.2; hacer que `run_test_suite` y `/tdd run` aborten la ejecución de watch, y que un Bash de tests completo limpie `watchDirty`; activar en /tdd y `run_test_suite` el paralelismo nativo de cada ejecutor cuando el proyecto ya lo tenga disponible, sin instalar nada (Q1): pytest con `-n auto` si pytest-xdist está declarado o instalado en `.venv`, y paratest en lugar de phpunit si existe `vendor/bin/paratest`; go, vitest y playwright ya paralelizan por defecto; verificar desde Pi real con el artefacto instalado.
Excluido: concurrencia entre suites dentro de `mem hcd verify` (HCD-055, repositorio gomemory); cambiar los veredictos, las reglas de evidencia o el motor HCD; reescribir o acortar los tests lentos; cambiar el playbook de gomemory (las skills de este repositorio pueden ser más estrictas que HCD-TDD-01); forzar `mvn -T` o `gradle --parallel`, porque pueden cambiar veredictos con plugins o tareas no seguros entre hilos. Si el proyecto ya los configura (`.mvn/maven.config` o `org.gradle.parallel`), el ejecutor los aplica solo.

## ¿Qué pasa si no lo hacemos?
Cada tarea HCD seguirá esperando entre 1,5 y 2 minutos su verificación, aunque el cambio afecte a un solo test. El agente seguirá sin una forma acotada de observar el rojo con la tool, el panel no mostrará lo que corre por Bash y watch podrá retrasar unos 100 s una ejecución pedida por el agente.

## Alternativa más barata
Hacer solo la rama 1 (`t.Parallel()`), que es la mayor ganancia medida y no toca el comportamiento de Pi. Deja sin resolver tres de los cuatro síntomas que confirmó la persona (panel, TDD del agente y watch), por eso se descarta como alcance completo, aunque es el primer incremento.

## Normas aplicables
- Constitución §10: las pruebas deben ser deterministas, sin depender del orden ni de datos compartidos. `-shuffle=on` y `-count=3` con `-race` lo comprueban al paralelizar. No se modifican pruebas existentes para hacer pasar una implementación.
- Playbook HCD-TDD-01 y HCD-EVIDENCE-01: rojo antes del verde y evidencia vigente.


## Aclaraciones
- [x] Q1. ¿Cómo se aplica el paralelismo para que valga en cualquier lenguaje del harness: (1) mantener t.Parallel en la suite Go del repo y añadir paralelismo nativo por ejecutor en /tdd y run_test_suite solo si ya está disponible, sin instalar nada ni cambiar veredictos; (2) solo el paralelismo por ejecutor; (3) mantener el plan sin paralelismo por ejecutor? → Ambas (Recommended)
- [x] Q2. La suite Go paralela tarda 49 s y los 114 tests en serie suman 29,7 s: ¿qué máximo se fija para la tarea 1: (1) 60 s, aceptando 49 s y siguiendo con las tareas multilenguaje; (2) refactorizar sandbox para paralelizar los 114; (3) 60 s ahora y el refactor en otra unidad? → Aceptar 49 s, umbral 60 s (Recommended)
- [x] Q3. ¿Dónde se aplica el cambio de las skills hcd-tasks y hcd-implement, que son copias de las plantillas de gomemory: (1) en la plantilla de gomemory y en la copia de pi-gomemory; (2) solo en la copia de Pi; (3) sin tocar skills? → Plantilla gomemory + copia (Recommended)

## Checklist
- [x] 1. Añadir t.Parallel() a los tests de pkg/installer que no modifican el entorno del proceso → `bash scripts/hcd056-suite-time.sh 60` (la suite completa pasa en menos de 60 s de reloj; antes tardaba 100 s; umbral fijado en Q2)
- [x] 2. Comprobar que la suite paralela es determinista → `go test -race -shuffle=on -count=3 ./...`
- [x] 3. Añadir el parámetro filter a run_test_suite traducido por ejecutor y marcar la suite como parcial → `go test ./pkg/installer -run '^TestHCD056Filter$' -count=1`
- [x] 4. Indicar en la descripción de run_test_suite que filter sirve para rojo→verde y all para la regresión final → `go test ./pkg/installer -run '^TestHCD056ToolDescription$' -count=1`
- [x] 5. Exigir en hcd-tasks y hcd-implement una comprobación acotada por tarea, una regresión final y el rojo en toda tarea con un test automatizable → `go test ./pkg/installer -run '^TestHCD056SkillsTDD$' -count=1`
- [x] 6. Ingerir en /tdd las líneas nuevas de tool_execution_update de un Bash de tests por número de línea absoluto, marcando la suite como parcial si la cola truncada deja huecos → `go test ./pkg/installer -run '^TestHCD056BashLiveIngest$' -count=1`
- [x] 7. Ingerir en /tdd los tests de la comprobación que trae la salida de mem hcd verify, sin cambiar su veredicto → `go test ./pkg/installer -run '^TestHCD056HCDVerifyIngest$' -count=1`
- [x] 8. Hacer que run_test_suite y /tdd run aborten la ejecución de watch en curso o en cola → `go test ./pkg/installer -run '^TestHCD056WatchAbort$' -count=1`
- [x] 9. Limpiar watchDirty tras un Bash de tests completo (no parcial) → `go test ./pkg/installer -run '^TestHCD056WatchDirtyBash$' -count=1`
- [x] 10. Comprobar en Pi real con la extensión instalada (SHA256 igual al repositorio) el rojo→verde con filter, el panel en vivo de un Bash y el aborto de watch → observación manual registrada con tiempos
- [x] 11. Ejecutar la regresión completa → `go vet ./... && go test -race ./...`
- [x] 12. Activar en detectSuites el paralelismo nativo disponible por ejecutor (pytest -n auto con xdist, paratest si existe) sin instalar nada → `go test ./pkg/installer -run '^TestHCD056RunnerParallel$' -count=1`
- [x] 13. Ingerir en vivo por test un Bash de go test -v (texto, sin -json), hallado en Pi real en la tarea 10 → `go test ./pkg/installer -run '^TestHCD056BashGoVerbose$' -count=1`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-19 21:40 -05:00 · tarea 1 · rojo observado · caducada · `bash scripts/hcd056-suite-time.sh 30` (salida 1)
      ok  	pi-gomemory	0.726s
      ok  	pi-gomemory/pkg/installer	98.225s
      suite: 100s de reloj (máximo 30s), salida go test 0
      FAIL: la suite supera el máximo
- 2026-11-19 21:44 -05:00 · tarea 1 · falló · caducada · `bash scripts/hcd056-suite-time.sh 30` (salida 1)
      ok  	pi-gomemory	0.247s
      ok  	pi-gomemory/pkg/installer	47.656s
      suite: 49s de reloj (máximo 30s), salida go test 0
      FAIL: la suite supera el máximo
- 2026-11-19 21:47 -05:00 · tarea 1 · falló · caducada · `bash scripts/hcd056-suite-time.sh 60` (salida 1)
      odebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean844109942/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean844109942/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- FAIL: TestHCD038TailBudget (0.75s)
          hcd038_test.go:124: TestHCD038TailBudget: FAIL el parser debe recibir cada línea completa | tail debe conservar la última línea, termina en "€€€€€€€€€€€€€€€€€€€€"
      FAIL
      FAIL	pi-gomemory/pkg/installer	44.074s
      FAIL
      suite: 45s de reloj (máximo 60s), salida go test 1
- 2026-11-19 21:48 -05:00 · tarea 1 · verde · caducada · `bash scripts/hcd056-suite-time.sh 60` (salida 0)
      ok  	pi-gomemory	0.305s
      ok  	pi-gomemory/pkg/installer	43.773s
      suite: 45s de reloj (máximo 60s), salida go test 0
- 2026-11-19 21:49 -05:00 · tarea 2 · falló · caducada · `go test -race -shuffle=on -count=3 ./...` (salida 1)
             [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD046Uninstall+codemode_ajeno_se_conserva1277675344/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD046Uninstall+codemode_ajeno_se_conserva1277675344/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	136.942s
      FAIL
- 2026-11-19 21:54 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD056Filter$' -count=1` (salida 1)
      --- FAIL: TestHCD056Filter (0.27s)
          hcd056_test.go:26: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD056Filter2090338515/003/test.mjs:3
              import factory, { filterCheck } from "./gomemory-tdd.ts";
                                ^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tdd.ts' does not provide an export named 'filterCheck'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.492s
      FAIL
- 2026-11-19 21:55 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056Filter$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.596s
- 2026-11-19 21:55 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD056ToolDescription$' -count=1` (salida 1)
      --- FAIL: TestHCD056ToolDescription (0.38s)
          hcd056_test.go:88: TestHCD056ToolDescription: FAIL no menciona filter | no menciona el ciclo rojo→verde | no reserva la suite completa para la regresión final | la lista de detección omite Python, PHP y Java
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.590s
      FAIL
- 2026-11-19 21:55 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056ToolDescription$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.451s
- 2026-11-19 21:57 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD056SkillsTDD$' -count=1` (salida 1)
      --- FAIL: TestHCD056SkillsTDD (0.00s)
          hcd056_test.go:101: hcd-tasks/SKILL.md no contiene "comprobación acotada"
          hcd056_test.go:101: hcd-tasks/SKILL.md no contiene "regresión completa"
          hcd056_test.go:101: hcd-tasks/SKILL.md no contiene "go test ./pkg/x -run"
          hcd056_test.go:101: hcd-tasks/SKILL.md no contiene "pytest"
          hcd056_test.go:101: hcd-tasks/SKILL.md no contiene "vitest"
          hcd056_test.go:107: hcd-implement/SKILL.md no contiene "Rojo antes del cambio en toda tarea con un test automatizable"
          hcd056_test.go:107: hcd-implement/SKILL.md no contiene "filter"
          hcd056_test.go:107: hcd-implement/SKILL.md no contiene "no automatizable"
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.236s
      FAIL
- 2026-11-19 21:57 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056SkillsTDD$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.208s
- 2026-11-19 21:58 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD056BashLiveIngest$' -count=1` (salida 1)
      --- FAIL: TestHCD056BashLiveIngest (0.24s)
          hcd056_test.go:99: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD056BashLiveIngest3797853407/001/test.mjs:2
              import factory, { createTDDStore, createBashLiveIngest } from "./gomemory-tdd.ts";
                                                ^^^^^^^^^^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tdd.ts' does not provide an export named 'createBashLiveIngest'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.468s
      FAIL
- 2026-11-19 21:59 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056BashLiveIngest$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.447s
- 2026-11-19 22:00 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056BashLiveIngest$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.491s
- 2026-11-19 22:00 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056ToolDescription$' -count=1 -v` (salida 0)
      === RUN   TestHCD056ToolDescription
      === PAUSE TestHCD056ToolDescription
      === CONT  TestHCD056ToolDescription
      --- PASS: TestHCD056ToolDescription (0.25s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.455s
- 2026-11-19 22:02 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD056HCDVerifyIngest$' -count=1` (salida 1)
      --- FAIL: TestHCD056HCDVerifyIngest (0.26s)
          hcd056_test.go:175: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD056HCDVerifyIngest3924408092/001/test.mjs:2
              import { createTDDStore, ingestHCDVerifyTests } from "./gomemory-tdd.ts";
                                       ^^^^^^^^^^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tdd.ts' does not provide an export named 'ingestHCDVerifyTests'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.481s
      FAIL
- 2026-11-19 22:03 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056HCDVerifyIngest$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.831s
- 2026-11-19 22:03 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD056WatchAbort$' -count=1` (salida 1)
      --- FAIL: TestHCD056WatchAbort (8.58s)
          hcd056_test.go:264: TestHCD056WatchAbort: FAIL run_test_suite esperó a watch: 7925 ms
      FAIL
      FAIL	pi-gomemory/pkg/installer	8.850s
      FAIL
- 2026-11-19 22:04 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056WatchAbort$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.933s
- 2026-11-19 22:04 -05:00 · tarea 9 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD056WatchDirtyBash$' -count=1` (salida 1)
      --- FAIL: TestHCD056WatchDirtyBash (3.47s)
          hcd056_test.go:328: TestHCD056WatchDirtyBash: FAIL a) tras un Bash completo watch no debe ejecutar; ejecutó 1 | b) tras un Bash parcial watch debe ejecutar una vez; ejecutó 2 | c) un edit posterior al Bash debe disparar watch; ejecutó 3 | d) un Bash que no es de tests debe disparar watch; ejecutó 4
      FAIL
      FAIL	pi-gomemory/pkg/installer	4.018s
      FAIL
- 2026-11-19 22:04 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056WatchDirtyBash$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	3.980s
- 2026-11-19 22:05 -05:00 · tarea 12 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD056RunnerParallel$' -count=1` (salida 1)
      --- FAIL: TestHCD056RunnerParallel (0.26s)
          hcd056_test.go:376: TestHCD056RunnerParallel: FAIL venv con xdist: ["-q"] | uv.lock con xdist: ["run","pytest","-q"] | paratest: {"id":"phpunit","name":"phpunit","cmd":"/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD056RunnerParallel2370044083/004/vendor/bin/phpunit","args":[],"parser":"phpunit"} | filtro con xdist: ["-q","-k","test_y"]
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.516s
      FAIL
- 2026-11-19 22:05 -05:00 · tarea 12 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056RunnerParallel$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.523s
- 2026-11-19 22:06 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056Filter$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.656s
- 2026-11-19 22:06 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056ToolDescription$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.478s
- 2026-11-19 22:06 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056SkillsTDD$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.243s
- 2026-11-19 22:06 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056BashLiveIngest$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.487s
- 2026-11-19 22:06 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056HCDVerifyIngest$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.497s
- 2026-11-19 22:06 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056WatchAbort$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.930s
- 2026-11-19 22:06 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056WatchDirtyBash$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	3.731s
- 2026-11-19 22:06 -05:00 · tarea 12 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056RunnerParallel$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.517s
- 2026-11-19 22:06 -05:00 · tarea 1 · falló · caducada · `bash scripts/hcd056-suite-time.sh 60` (salida 1)
              AssertionError [ERR_ASSERTION]: Watch y la ejecución encolada deben correr
              'x' !== 'xx'
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD053LiveHCDRows2476807045/001/test.mjs:66:8 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: 'x',
                expected: 'xx',
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      --- FAIL: TestHCD021WatchCoalesces (1.50s)
          hcd021_audit_test.go:115: TestHCD021WatchCoalesces: FAIL go test se ejecutó 2 veces; se esperaban 3 (tool, watch agrupado, /tdd run)
      FAIL
      FAIL	pi-gomemory/pkg/installer	45.317s
      FAIL
      suite: 46s de reloj (máximo 60s), salida go test 1
- 2026-11-19 22:07 -05:00 · tarea 2 · falló · caducada · `go test -race -shuffle=on -count=3 ./...` (salida 1)
              AssertionError [ERR_ASSERTION]: Watch y la ejecución encolada deben correr
              'x' !== 'xx'
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD053LiveHCDRows1487369332/001/test.mjs:66:8 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: 'x',
                expected: 'xx',
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      --- FAIL: TestHCD021WatchCoalesces (1.61s)
          hcd021_audit_test.go:115: TestHCD021WatchCoalesces: FAIL go test se ejecutó 2 veces; se esperaban 3 (tool, watch agrupado, /tdd run)
      FAIL
      FAIL	pi-gomemory/pkg/installer	132.930s
      FAIL
- 2026-11-20 03:12 -05:00 · tarea 2 · verde · caducada · `go test -race -shuffle=on -count=3 ./...` (salida 0)
      ok  	pi-gomemory	1.471s
      ok  	pi-gomemory/pkg/installer	132.455s
- 2026-11-20 03:14 -05:00 · tarea 1 · verde · caducada · `bash scripts/hcd056-suite-time.sh 60` (salida 0)
      ok  	pi-gomemory	0.338s
      ok  	pi-gomemory/pkg/installer	38.823s
      suite: 39s de reloj (máximo 60s), salida go test 0
- 2026-11-20 03:15 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056Filter$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.784s
- 2026-11-20 03:15 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056ToolDescription$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.458s
- 2026-11-20 03:15 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056SkillsTDD$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.434s
- 2026-11-20 03:15 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056BashLiveIngest$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.473s
- 2026-11-20 03:15 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056HCDVerifyIngest$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.453s
- 2026-11-20 03:15 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056WatchAbort$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.868s
- 2026-11-20 03:15 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056WatchDirtyBash$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	3.689s
- 2026-11-20 03:15 -05:00 · tarea 12 · verde · caducada · `go test ./pkg/installer -run '^TestHCD056RunnerParallel$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.457s
- 2026-11-20 03:15 -05:00 · tarea 11 · verde · caducada · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	46.269s
- 2026-11-20 03:21 -05:00 · tarea 13 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD056BashGoVerbose$' -count=1` (salida 1)
      --- FAIL: TestHCD056BashGoVerbose (0.27s)
          hcd056_gotext_test.go:48: TestHCD056BashGoVerbose: FAIL go test -v debe usar el parser de texto | TestSuma debió pasar en vivo: undefined | TestLento1 debió pasar en vivo | TestLento2 debió fallar: undefined | debe haber 3 filas, hay 2
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.656s
      FAIL
- 2026-11-20 03:21 -05:00 · tarea 13 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD056BashGoVerbose$' -count=1` (salida 1)
      --- FAIL: TestHCD056BashGoVerbose (0.25s)
          hcd056_gotext_test.go:48: TestHCD056BashGoVerbose: FAIL go test -v debe usar el parser de texto | TestSuma debió pasar en vivo: undefined | TestLento1 debió pasar en vivo | TestLento2 debió fallar: undefined | debe haber 3 filas, hay 2
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.462s
      FAIL
- 2026-11-20 03:21 -05:00 · tarea 13 · verde · vigente · `go test ./pkg/installer -run '^TestHCD056BashGoVerbose$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.458s
- 2026-11-20 03:22 -05:00 · tarea 10 · manual · vigente
      Pi real vía Herdr (pane w3X:p3, módulo Go desechable con TestSuma y 4 tests de 3 s; suite completa 12 s en serie), extensión instalada con SHA256 b554f2b4… igual al repo. (a) filter: run_test_suite filter=TestSuma dio rojo 0/1 «Backend · parcial · 899ms» y, tras corregir Suma, verde 1/1 «parcial · 801ms»; cada ronda agente+tool tardó 5 s frente a 12 s de la suite. (b) Bash en vivo: la primera prueba con go test -v NO mostró filas hasta el final y contó 1/1 (clasificado exit-code); corregido en la tarea 13 y reinstalado, el panel mostró los tests uno a uno (1/2 a t=4 s, 2/3 a 8 s, 3/4 a 10 s, 4/5 a 14 s, 5/5 a 16 s). (c) Watch: con /tdd watch y un edit del agente, watch arrancó la suite; /tdd run lanzado a mitad avisó «queda en cola» y terminó a los 13 s (12,3 s propios), sin esperar los ~12 s restantes de watch.
- 2026-11-20 03:22 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD056Filter$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.578s
- 2026-11-20 03:23 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD056ToolDescription$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.454s
- 2026-11-20 03:23 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD056SkillsTDD$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.201s
- 2026-11-20 03:23 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run '^TestHCD056BashLiveIngest$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.456s
- 2026-11-20 03:23 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run '^TestHCD056HCDVerifyIngest$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.455s
- 2026-11-20 03:23 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run '^TestHCD056WatchAbort$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.866s
- 2026-11-20 03:23 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run '^TestHCD056WatchDirtyBash$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	3.674s
- 2026-11-20 03:23 -05:00 · tarea 12 · verde · vigente · `go test ./pkg/installer -run '^TestHCD056RunnerParallel$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.448s
- 2026-11-20 03:23 -05:00 · tarea 1 · verde · vigente · `bash scripts/hcd056-suite-time.sh 60` (salida 0)
      ok  	pi-gomemory	0.369s
      ok  	pi-gomemory/pkg/installer	42.695s
      suite: 44s de reloj (máximo 60s), salida go test 0
- 2026-11-20 03:23 -05:00 · tarea 2 · verde · vigente · `go test -race -shuffle=on -count=3 ./...` (salida 0)
      ok  	pi-gomemory	1.469s
      ok  	pi-gomemory/pkg/installer	132.828s
- 2026-11-20 03:26 -05:00 · tarea 11 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	48.233s

## Siguiente
Cerrada con aprobación sobre la revisión ca22abbba930.

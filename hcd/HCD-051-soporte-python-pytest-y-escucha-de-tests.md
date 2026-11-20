# Soporte Python/pytest, escucha en bash, refresco vivo de tareas y contención de RAM

ID: HCD-051
Tipo: feature
Carril: fast
Motivo: Mejora acotada en extensiones de Pi (gomemory-tdd.ts, gomemory-tasks.ts), skills/hooks y configuración de MCPs de pi-gomemory

## Intención
Resolver cuatro problemas observados en uso real con Pi en proyectos Python/TypeScript y entornos con RAM acotada (7.6 GiB en WSL2):
1. Que `gomemory-tdd.ts` soporte proyectos **Python (`pytest`), PHP, Java, JavaScript, HTML y CSS** en `detectSuites()` y en la ejecución/parseo de pruebas y checks estáticos.
2. Que `gomemory-tdd.ts` refleje en el panel `/tdd` los tests ejecutados por el agente vía `bash` (`pytest`, `vitest`, `go test`, `npm test`, `mem hcd verify`), no solo `/tdd run` o `run_test_suite`.
3. Que `gomemory-tasks.ts` refresque el checklist durante llamadas `bash` largas con `mem hcd` (no solo al llegar `tool_result`), y evitar que el agente agrupe múltiples `mem hcd verify` en una sola llamada `bash` que bloquea la entrega de *steering* en Pi.
4. Reducir la contención de memoria RAM cuando corren varias sesiones `pi` en paralelo junto a Docker y forks de pruebas (procesos `codebase-memory`, `codegraph` + watchdog y `gomemory` duplicados por sesión).

## Alcance
Incluido: detección y ejecución/parseo de suites y checks estáticos para **Python (`pytest`, `ruff`/`mypy`), PHP (`phpunit`/`composer`/`php -l`), Java (`mvn`/`gradle`), JavaScript (`npm test`/`vitest`/`node --test`), HTML y CSS** en `extensions/gomemory-tdd.ts`; escucha de ejecuciones de tests vía `bash` (`tool_result` de `pytest`, `vitest`, `phpunit`, `mvn`/`gradle`, `go test`, `npm test`, `playwright`, `mem hcd verify`) para actualizar el panel `/tdd`; refresco periódico en `extensions/gomemory-tasks.ts` mientras hay un `bash` de `mem hcd` en curso (`tool_execution_start` → `tool_execution_end`); prevención de llamadas `bash` monolíticas que agrupan múltiples `mem hcd verify`; y mitigación de consumo de RAM de los MCPs por sesión en `pi-gomemory`.
Excluido: modificar el código fuente del binario `pi-coding-agent` (la entrega de *steering* entre tools es comportamiento del harness); gestionar contenedores Docker o la configuración interna de Vitest en repositorios externos.

## ¿Qué pasa si no lo hacemos?
1. En proyectos Python (`kolmena-core-ia`), el panel `/tdd` muestra todas las suites omitidas salvo `Data & Graph`.
2. Las pruebas corridas por `bash` o `mem hcd verify` nunca aparecen en `/tdd`.
3. Durante verificaciones largas en un solo `bash`, el widget de tareas queda congelado (ej. `1/21`) y el usuario no puede enviar *steering* hasta que termina todo el bloque.
4. En equipos con ~7.6 GiB de RAM (WSL2), abrir 2+ sesiones `pi` junto a Docker/Vitest empuja el sistema a swap (~559 MiB) y degrada gravemente la fluidez.

## Alternativa más barata
Seguir ejecutando `pytest` y `mem hcd verify` manualmente sin reflejo en `/tdd` ni refresco en vivo, pedir manualmente al agente que no agrupe `mem hcd verify`, y cerrar sesiones/MCPs a mano cuando falte RAM. No resuelve la pérdida de visibilidad ni la saturación en uso normal.

## Aclaraciones
- [x] Q1. Para evitar que el agente agrupe múltiples `mem hcd verify` en una sola llamada `bash` (que bloquea tanto el *steering* de Pi como `tool_result`): ¿basta con reforzar la instrucción en `hcd-verify` / `gomemory-context.ts`, o quieres también un bloqueo/aviso en `tool_call` cuando una llamada `bash` contenga más de un `mem hcd verify` (o funciones/bucles alrededor)? → 1
- [x] Q2. Para la contención de RAM con múltiples sesiones `pi` + MCPs (`codebase-memory`, `codegraph` + watchdog, `gomemory`): ¿qué estrategia prefieres priorizar? (a) no lanzar `codegraph` / `codebase-memory` en sesiones donde no exista `.codegraph/` o mediante un modo liviano (`--lite` / `PI_GOMEMORY_LITE=1`), (b) conectar los MCPs a un daemon compartido por HTTP/socket en vez de un proceso stdio por sesión cuando el servidor lo soporte, o (c) ambas cosas combinadas. → 1

## Checklist
- [x] 1. Ampliar `detectSuites()` en `extensions/gomemory-tdd.ts` para detectar proyectos Python (`pytest`), PHP (`phpunit`/`composer`) y Java (`mvn`/`gradle`) → `go test ./pkg/installer/... -run TestHCD051DetectBackendSuites -v`
- [x] 2. Ampliar `detectSuites()` en `extensions/gomemory-tdd.ts` para detectar JavaScript (`package.json`/`vitest`/`node --test`) y checks estáticos de HTML/CSS → `go test ./pkg/installer/... -run TestHCD051DetectFrontendAndStaticSuites -v`
- [x] 3. Añadir parsers de salida para `pytest`, `vitest`, `phpunit` y `mvn`/`gradle` en `extensions/gomemory-tdd.ts` → `go test ./pkg/installer/... -run TestHCD051OutputParsers -v`
- [x] 4. Escuchar `tool_result` de llamadas `bash` con comandos de test o `mem hcd verify` en `extensions/gomemory-tdd.ts` para actualizar el panel `/tdd` → `go test ./pkg/installer/... -run TestHCD051BashTestListener -v`
- [x] 5. Añadir refresco periódico en `extensions/gomemory-tasks.ts` mientras corre un `bash` con `mem hcd` entre `tool_execution_start` y `tool_execution_end` → `go test ./pkg/installer/... -run TestHCD051TasksLivePolling -v`
- [x] 6. Bloquear en `tool_call` las llamadas `bash` que agrupen múltiples `mem hcd verify` y reforzar la instrucción en `skills/hcd-verify/SKILL.md` → `go test ./pkg/installer/... -run TestHCD051VerifySingleCallGuard -v`
- [x] 7. Implementar modo liviano (`lite on|off` / `PI_GOMEMORY_LITE=1`) y omitir `codegraph` sin `.codegraph/` para reducir consumo de RAM entre sesiones `pi` → `go test ./pkg/installer/... -run TestHCD051LiteModeAndCodegraphGate -v`
- [x] 8. Validar `go vet ./...` y la suite completa `go test ./...` sin regresiones → `go vet ./... && go test ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-19 19:26 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer/... -run TestHCD051DetectBackendSuites -v` (salida 1)
                  ^
              AssertionError [ERR_ASSERTION]: backend Python debe detectarse
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD051DetectBackendSuites1613582211/001/test.mjs:18:10
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5) {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: false,
                expected: true,
                operator: '==',
                diff: 'simple'
              }
              Node.js v26.13.0
      --- FAIL: TestHCD051DetectBackendSuites (0.16s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.349s
      FAIL
- 2026-11-19 19:26 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer/... -run TestHCD051DetectBackendSuites -v` (salida 0)
      === RUN   TestHCD051DetectBackendSuites
          hcd051_test.go:81: 
      --- PASS: TestHCD051DetectBackendSuites (0.14s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.306s
- 2026-11-19 19:26 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer/... -run TestHCD051DetectFrontendAndStaticSuites -v` (salida 1)
                  ^
              AssertionError [ERR_ASSERTION]: frontend con vitest.config debe detectarse
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD051DetectFrontendAndStaticSuites419903025/001/test.mjs:18:10
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5) {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: false,
                expected: true,
                operator: '==',
                diff: 'simple'
              }
              Node.js v26.13.0
      --- FAIL: TestHCD051DetectFrontendAndStaticSuites (0.13s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.326s
      FAIL
- 2026-11-19 19:26 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer/... -run TestHCD051DetectFrontendAndStaticSuites -v` (salida 0)
      === RUN   TestHCD051DetectFrontendAndStaticSuites
          hcd051_test.go:122: 
      --- PASS: TestHCD051DetectFrontendAndStaticSuites (0.14s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.303s
- 2026-11-19 19:27 -05:00 · tarea 3 · rojo NO observado (el comando pasó) · caducada · `go test ./pkg/installer/... -run TestHCD051OutputParsers -v` (salida 0)
      testing: warning: no tests to run
      PASS
      ok  	pi-gomemory/pkg/installer	0.167s [no tests to run]
- 2026-11-19 19:27 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer/... -run TestHCD051OutputParsers -v` (salida 1)
      === RUN   TestHCD051OutputParsers
          hcd051_test.go:126: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD051OutputParsers1464108833/001/test.mjs:3
              import { createTDDStore, pytestParser, vitestParser, phpunitParser, javaTestParser } from "./gomemory-tdd.ts";
                                                                                  ^^^^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tdd.ts' does not provide an export named 'javaTestParser'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      --- FAIL: TestHCD051OutputParsers (0.15s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.366s
      FAIL
- 2026-11-19 19:27 -05:00 · tarea 3 · falló · caducada · `go test ./pkg/installer/... -run TestHCD051OutputParsers -v` (salida 1)
      === RUN   TestHCD051OutputParsers
          hcd051_test.go:126: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD051OutputParsers1095775609/001/test.mjs:16
              	assert.ok(suite.tests.some((t) => t.name.includes("test_ok") && t.status === "passed"));
              	                ^
              TypeError: Cannot read properties of undefined (reading 'tests')
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD051OutputParsers1095775609/001/test.mjs:16:18
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      --- FAIL: TestHCD051OutputParsers (0.13s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.287s
      FAIL
- 2026-11-19 19:27 -05:00 · tarea 3 · falló · caducada · `go test ./pkg/installer/... -run TestHCD051OutputParsers -v` (salida 1)
      === RUN   TestHCD051OutputParsers
          hcd051_test.go:126: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD051OutputParsers2694979129/001/test.mjs:17
              	assert.ok(suite.tests.some((t) => t.name.includes("test_ok") && t.status === "passed"));
              	                ^
              TypeError: Cannot read properties of undefined (reading 'tests')
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD051OutputParsers2694979129/001/test.mjs:17:18
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      --- FAIL: TestHCD051OutputParsers (0.13s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.324s
      FAIL
- 2026-11-19 19:28 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer/... -run TestHCD051OutputParsers -v` (salida 0)
      === RUN   TestHCD051OutputParsers
          hcd051_test.go:193: 
      --- PASS: TestHCD051OutputParsers (0.14s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.325s
- 2026-11-19 19:28 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer/... -run TestHCD051BashTestListener -v` (salida 1)
      === RUN   TestHCD051BashTestListener
          hcd051_test.go:197: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD051BashTestListener2289525863/001/test.mjs:3
              import registerTDD, { classifyBashTestCommand } from "./gomemory-tdd.ts";
                                    ^^^^^^^^^^^^^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tdd.ts' does not provide an export named 'classifyBashTestCommand'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      --- FAIL: TestHCD051BashTestListener (0.14s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.324s
      FAIL
- 2026-11-19 19:28 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer/... -run TestHCD051BashTestListener -v` (salida 0)
      === RUN   TestHCD051BashTestListener
          hcd051_test.go:250: 
      --- PASS: TestHCD051BashTestListener (0.14s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.343s
- 2026-11-19 19:28 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer/... -run TestHCD051TasksLivePolling -v` (salida 1)
                  ^
              AssertionError [ERR_ASSERTION]: debe registrar tool_execution_start
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD051TasksLivePolling1118432228/001/test.mjs:19:8
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5) {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: undefined,
                expected: true,
                operator: '==',
                diff: 'simple'
              }
              Node.js v26.13.0
      --- FAIL: TestHCD051TasksLivePolling (0.13s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.338s
      FAIL
- 2026-11-19 19:28 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer/... -run TestHCD051TasksLivePolling -v` (salida 0)
      === RUN   TestHCD051TasksLivePolling
          hcd051_test.go:282: 
      --- PASS: TestHCD051TasksLivePolling (1.87s)
      PASS
      ok  	pi-gomemory/pkg/installer	2.043s
- 2026-11-19 19:29 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer/... -run TestHCD051VerifySingleCallGuard -v` (salida 1)
      === RUN   TestHCD051VerifySingleCallGuard
          hcd051_test.go:286: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD051VerifySingleCallGuard1646142549/001/test.mjs:3
              import registerTasks, { isBundledHcdVerifyCommand } from "./gomemory-tasks.ts";
                                      ^^^^^^^^^^^^^^^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tasks.ts' does not provide an export named 'isBundledHcdVerifyCommand'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      --- FAIL: TestHCD051VerifySingleCallGuard (0.18s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.375s
      FAIL
- 2026-11-19 19:29 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer/... -run TestHCD051VerifySingleCallGuard -v` (salida 0)
      === RUN   TestHCD051VerifySingleCallGuard
          hcd051_test.go:305: 
      --- PASS: TestHCD051VerifySingleCallGuard (0.13s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.284s
- 2026-11-19 19:29 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer/... -run TestHCD051LiteModeAndCodegraphGate -v` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd051_test.go:313:12: undefined: SetLiteMode
      pkg/installer/hcd051_test.go:316:6: undefined: IsLiteMode
      pkg/installer/hcd051_test.go:328:12: undefined: SetLiteMode
      pkg/installer/hcd051_test.go:331:5: undefined: IsLiteMode
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-19 19:29 -05:00 · tarea 7 · falló · caducada · `go test ./pkg/installer/... -run TestHCD051LiteModeAndCodegraphGate -v` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/lite.go:51:10: undefined: InstallMCPWithServers
      pkg/installer/lite.go:53:9: undefined: InstallMCPWithServers
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-19 19:30 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer/... -run TestHCD051LiteModeAndCodegraphGate -v` (salida 0)
      === RUN   TestHCD051LiteModeAndCodegraphGate
      --- PASS: TestHCD051LiteModeAndCodegraphGate (0.02s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.216s
- 2026-11-19 19:30 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer/... -run TestHCD051DetectBackendSuites -v` (salida 0)
      === RUN   TestHCD051DetectBackendSuites
          hcd051_test.go:84: 
      --- PASS: TestHCD051DetectBackendSuites (0.18s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.348s
- 2026-11-19 19:30 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer/... -run TestHCD051DetectFrontendAndStaticSuites -v` (salida 0)
      === RUN   TestHCD051DetectFrontendAndStaticSuites
          hcd051_test.go:125: 
      --- PASS: TestHCD051DetectFrontendAndStaticSuites (0.19s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.345s
- 2026-11-19 19:30 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer/... -run TestHCD051TasksLivePolling -v` (salida 0)
      === RUN   TestHCD051TasksLivePolling
          hcd051_test.go:285: 
      --- PASS: TestHCD051TasksLivePolling (1.86s)
      PASS
      ok  	pi-gomemory/pkg/installer	2.018s
- 2026-11-19 19:30 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer/... -run TestHCD051OutputParsers -v` (salida 0)
      === RUN   TestHCD051OutputParsers
          hcd051_test.go:196: 
      --- PASS: TestHCD051OutputParsers (0.16s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.326s
- 2026-11-19 19:30 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer/... -run TestHCD051BashTestListener -v` (salida 0)
      === RUN   TestHCD051BashTestListener
          hcd051_test.go:253: 
      --- PASS: TestHCD051BashTestListener (0.17s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.327s
- 2026-11-19 19:30 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer/... -run TestHCD051VerifySingleCallGuard -v` (salida 0)
      === RUN   TestHCD051VerifySingleCallGuard
          hcd051_test.go:308: 
      --- PASS: TestHCD051VerifySingleCallGuard (0.15s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.299s
- 2026-11-19 19:30 -05:00 · tarea 8 · verde · vigente · `go vet ./... && go test ./...` (salida 0)
      ok  	pi-gomemory	0.209s
      ok  	pi-gomemory/pkg/installer	76.708s

## Siguiente
Cerrada con aprobación sobre la revisión cdbf63cfb883.

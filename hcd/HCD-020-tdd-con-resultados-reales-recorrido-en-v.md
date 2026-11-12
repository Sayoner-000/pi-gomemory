# /tdd con resultados reales, recorrido en vivo y fallos devueltos al agente

ID: HCD-020
Tipo: evolution
Carril: fast
Motivo: intención clara con evidencia real del estado actual; las decisiones abiertas (Playwright, streaming) son independientes

## Intención
Que /tdd solo muestre resultados reales detectados según el proyecto, que el recorrido avance en vivo test a test con el mensaje, la ubicación y la duración reales de cada fallo, y que run_test_suite devuelva al agente los fallos con su diagnóstico para corregir y volver a ejecutar


## Estado actual (evidencia, 2026-11-11)
Se ejecutó `/tdd run` con pi-tui 1.5.2 real y procesos reales (memoria 122), sobre este repo y sobre un proyecto Go con un test roto:
- Backend (`go test -json`) y CI (`go vet`) se ejecutan de verdad.
- Frontend falla siempre en falso: `node --check` no elimina tipos de TypeScript.
- Playwright se marca «passed» en 45 ms sin ejecutar nada.
- Data falla en falso si no hay unidad HCD activa (`mem hcd show` sale con código 3).
- No hay recorrido en vivo: `pi.exec` espera al final del proceso; el panel queda en 0/0 durante 10 s y luego salta al 99%.
- Un test que falla muestra «Fallo durante la ejecución»: se pierden el mensaje (`Suma(2,3) = -1, want 5`) y el archivo:línea.
- Las duraciones de 10 ms son un valor de respaldo, no la duración real.
- Las rutas están fijas (`./pkg/installer`, `extensions/gomemory-tdd.ts`).
- `run_test_suite` devuelve al agente solo el conteo, sin los fallos.

## Alcance
Incluido: (1) **Resultados reales**: las suites se detectan según el proyecto (`go.mod` → `go test -json ./...` y `go vet ./...`; un package.json del proyecto analizado con script `test` → `npm test`; un tsconfig.json del proyecto analizado → `tsc --noEmit`); una suite que no aplica o cuya herramienta falta se muestra «omitida» con su motivo, nunca como fallo ni como éxito inventado; Data se muestra «omitida» si no hay unidad HCD activa; Playwright deja de ser un resultado fijo: con `playwright.config.*` se ejecuta `npx playwright test --reporter=line` (Q1) y, sin config, la suite se muestra «omitida». (2) **Recorrido en vivo**: los procesos se lanzan con `spawn` de `node:child_process`, sin shell y con argumentos en array, y la salida se procesa línea a línea, de modo que cada test aparece al empezar y al terminar; en los fallos se guarda el mensaje real y el `archivo:línea` (del Output de `go test -json`) y se muestran en el panel; la duración es la real (`Elapsed`), sin valor de respaldo. (3) **Bucle con el agente**: `run_test_suite` devuelve en `content` cada fallo con su suite, test, mensaje, ubicación y diagnóstico, y avanza con `onUpdate` mientras corre; `classifyFailure` deja de confundir «unexpected» con «expected». (4) **Qué se muestra (Q2)**: el panel lista solo las suites habilitadas y, dentro, los tests en ejecución y los fallidos (los aprobados se resumen en un conteo); las suites omitidas se resumen en una línea y los tests `t.Skip` no se listan. (5) **Iconos (Q3)**: sin emoji; el test en curso usa el spinner braille de pi-tui y el resto, glifos de texto (✓ ✗ ○ – ›). Pruebas nuevas en `pkg/installer/hcd020_tdd_real_test.go`. Se reescribe `TestHCD018TDDLiveExecutionMultiTest`, que fijaba un `pi.exec` simulado sobre `/dummy` (el contrato falso que HCD-020 corrige), para validar lo mismo contra un módulo Go real.
Excluido: generar planes o tests con un modelo (pasos 1-2 de TestSprite); ejecución en la nube; `/tdd ui` y `/tdd report`; capturas o vídeo; el layout del panel y el clic (HCD-019); recursos y ritmo del modo watch.

## ¿Qué pasa si no lo hacemos?
`/tdd` sigue mostrando un Playwright verde inventado y fallos falsos en Frontend y Data, así que el panel no es fiable. Cuando un test falla de verdad, ni la persona ni el agente ven el porqué, y el panel no avanza mientras corre: no hay recorrido, que era el objetivo de HCD-018.

## Alternativa más barata
Retirar Frontend, Playwright y Data, y dejar solo Backend y CI con `pi.exec` tal como están. Quita los resultados falsos con muy poco código, pero no da recorrido en vivo ni el detalle de los fallos, que es lo que se pidió; solo sirve como paso intermedio.

## Aclaraciones
- [x] Q1. Playwright: ¿se ejecuta de verdad (`npx playwright test --reporter=line`) cuando existe `playwright.config.*`, o se retira la suite? [NEEDS CLARIFICATION: Playwright real o retirada] → Ejecutar real (Recomendado): si existe playwright.config.*, ejecutar npx playwright test --reporter=line y mostrar cada test y paso en vivo con su fallo real; sin config, la suite se muestra omitida
- [x] Q2. Q2. ¿Qué tests y suites muestra el panel? → Persona: «los test se deben mostrar los que están habilitados en la prueba ... o los que se esten ejecutando». Se muestran solo las suites habilitadas para el proyecto y, dentro, los tests en ejecución y los fallidos; las suites omitidas se resumen en una línea y los tests t.Skip no se listan
- [x] Q3. Q3. ¿Qué iconos usa el panel? → Persona: «quita los emojis y trae los iconos de lispgloss o los que sea para npx de pi». Sin emoji: spinner braille de pi-tui (DEFAULT_FRAMES de Loader, estilo MiniDot de bubbles/lipgloss) para el test en curso y glifos de texto ✓ ✗ ○ – ›

## Checklist
- [x] 1. Detectar las suites según el proyecto (en el proyecto analizado: go.mod, package.json con script test, tsconfig.json, playwright.config) y marcar «omitida» con motivo la que no aplica → `go test ./pkg/installer -run TestHCD020SuiteDetection -count=1`
- [x] 2. Ejecutar los procesos con `spawn` sin shell y entregar la salida línea a línea mientras el proceso sigue vivo → `go test ./pkg/installer -run TestHCD020LineStreaming -count=1`
- [x] 3. Mostrar cada test Go en vivo con su duración real y, si falla, su mensaje y `archivo:línea` del Output de `go test -json` (proyecto Go real con un test roto) → `go test ./pkg/installer -run TestHCD020GoFailureDetail -count=1`
- [x] 4. Ejecutar Playwright de verdad con `--reporter=line` y registrar cada test y su fallo real → `go test ./pkg/installer -run TestHCD020PlaywrightLine -count=1`
- [x] 5. Eliminar los resultados falsos: sin Playwright fijo ni `node --check`, Data «omitida» sin unidad HCD activa, y `classifyFailure` sin confundir «unexpected» con «expected» → `go test ./pkg/installer -run TestHCD020NoFakeResults -count=1`
- [x] 6. Devolver al agente desde `run_test_suite` cada fallo con su suite, test, mensaje, ubicación y diagnóstico, avanzando con `onUpdate` → `go test ./pkg/installer -run TestHCD020AgentLoop -count=1`
- [x] 7. Mantener en verde la suite previa (HCD-018/019) y `go vet` → `go test ./... -count=1 && go vet ./...`
- [x] 8. Mostrar en el panel solo las suites habilitadas y, dentro, los tests en ejecución y los fallidos, con las omitidas resumidas en una línea (Q2) → `go test ./pkg/installer -run TestHCD020PanelShowsEnabled -count=1`
- [x] 9. Sustituir los emojis del panel por el spinner braille de pi-tui y glifos de texto (Q3) → `go test ./pkg/installer -run TestHCD020PanelIcons -count=1`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 19:13 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD020SuiteDetection -count=1` (salida 1)
      --- FAIL: TestHCD020SuiteDetection (0.12s)
          hcd020_tdd_real_test.go:110: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD020SuiteDetection2095797036/003/test.mjs:2
              import { detectSuites } from "./gomemory-tdd.ts";
                       ^^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tdd.ts' does not provide an export named 'detectSuites'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.292s
      FAIL
- 2026-11-11 19:13 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD020LineStreaming -count=1` (salida 1)
      --- FAIL: TestHCD020LineStreaming (0.11s)
          hcd020_tdd_real_test.go:138: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD020LineStreaming1286830873/001/test.mjs:2
              import { streamLines } from "./gomemory-tdd.ts";
                       ^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tdd.ts' does not provide an export named 'streamLines'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.259s
      FAIL
- 2026-11-11 19:13 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD020GoFailureDetail -count=1` (salida 1)
      --- FAIL: TestHCD020GoFailureDetail (0.12s)
          hcd020_tdd_real_test.go:175: TestHCD020GoFailureDetail: FAIL TestSuma debe fallar | TestOK debe pasar | TestLento debe tener su duración real (≥100ms), tiene undefined
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.256s
      FAIL
- 2026-11-11 19:13 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD020PlaywrightLine -count=1` (salida 1)
      --- FAIL: TestHCD020PlaywrightLine (0.11s)
          hcd020_tdd_real_test.go:204: TestHCD020PlaywrightLine: FAIL deben registrarse 3 tests, hay 1 | 'total del carrito' debe fallar | ubicación: undefined | paso fallido: undefined | mensaje: undefined | los otros dos deben pasar | debe verse el avance en vivo mientras corre
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.255s
      FAIL
- 2026-11-11 19:13 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD020NoFakeResults -count=1` (salida 1)
      --- FAIL: TestHCD020NoFakeResults (0.11s)
          hcd020_tdd_real_test.go:225: TestHCD020NoFakeResults: FAIL en un proyecto vacío nada debe pasar ni contarse: {"total":4,"passed":1,"failed":0,"running":3,"queued":0} | backend debe quedar omitida con motivo, está failed | frontend debe quedar omitida con motivo, está failed | playwright debe quedar omitida con motivo, está passed | ci debe quedar omitida con motivo, está failed | data debe quedar omitida con motivo, está failed | data sin unidad HCD activa: undefined | SyntaxError mal diagnosticado: Discrepancia en contrato o aserción lógica de la prueba
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.262s
      FAIL
- 2026-11-11 19:13 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD020AgentLoop -count=1` (salida 1)
      --- FAIL: TestHCD020AgentLoop (0.12s)
          hcd020_tdd_real_test.go:243: TestHCD020AgentLoop: FAIL el informe al agente no incluye "TestSuma" | el informe al agente no incluye "calc_test.go:" | el informe al agente no incluye "Suma(2,3) = -1, want 5" | el informe al agente no incluye "Backend" | el informe debe incluir la sugerencia del diagnóstico | onUpdate no se llamó durante la ejecución
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.276s
      FAIL
- 2026-11-11 19:15 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD020PanelShowsEnabled -count=1` (salida 1)
      --- FAIL: TestHCD020PanelShowsEnabled (0.13s)
          hcd020_tdd_real_test.go:272: TestHCD020PanelShowsEnabled: FAIL debe listar el fallido con ubicación y mensaje | los aprobados se resumen, no se listan uno a uno | falta el resumen de aprobados | Frontend debe aparecer solo en la línea resumen de omitidas | Playwright E2E debe aparecer solo en la línea resumen de omitidas | Data & Graph debe aparecer solo en la línea resumen de omitidas | las omitidas van en una sola línea
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.321s
      FAIL
- 2026-11-11 19:15 -05:00 · tarea 9 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD020PanelIcons -count=1` (salida 1)
      --- FAIL: TestHCD020PanelIcons (0.12s)
          hcd020_tdd_real_test.go:292: TestHCD020PanelIcons: FAIL quedan emojis: ⏳ | el test en curso debe usar el spinner braille de pi-tui: │   ⏳ TestEnCurso│
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.285s
      FAIL
- 2026-11-11 19:18 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD020SuiteDetection -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.332s
- 2026-11-11 19:18 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD020LineStreaming -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.775s
- 2026-11-11 19:18 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD020GoFailureDetail -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.791s
- 2026-11-11 19:18 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD020PlaywrightLine -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.796s
- 2026-11-11 19:18 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run TestHCD020NoFakeResults -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.336s
- 2026-11-11 19:18 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run TestHCD020AgentLoop -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.704s
- 2026-11-11 19:18 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run TestHCD020PanelShowsEnabled -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.280s
- 2026-11-11 19:18 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run TestHCD020PanelIcons -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.267s
- 2026-11-11 19:18 -05:00 · tarea 7 · verde · caducada · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	12.718s
- 2026-11-11 19:19 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD020SuiteDetection -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.285s
- 2026-11-11 19:19 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD020LineStreaming -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.756s
- 2026-11-11 19:19 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD020GoFailureDetail -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.701s
- 2026-11-11 19:19 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD020PlaywrightLine -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.794s
- 2026-11-11 19:19 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run TestHCD020NoFakeResults -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.363s
- 2026-11-11 19:19 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run TestHCD020AgentLoop -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.748s
- 2026-11-11 19:19 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run TestHCD020PanelShowsEnabled -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.278s
- 2026-11-11 19:19 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run TestHCD020PanelIcons -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.273s
- 2026-11-11 19:19 -05:00 · tarea 7 · verde · caducada · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	12.267s
- 2026-11-11 19:20 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestHCD020SuiteDetection -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.277s
- 2026-11-11 19:20 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD020LineStreaming -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.777s
- 2026-11-11 19:20 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD020GoFailureDetail -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.696s
- 2026-11-11 19:20 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestHCD020PlaywrightLine -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.795s
- 2026-11-11 19:20 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run TestHCD020NoFakeResults -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.344s
- 2026-11-11 19:20 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run TestHCD020AgentLoop -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.701s
- 2026-11-11 19:20 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run TestHCD020PanelShowsEnabled -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.263s
- 2026-11-11 19:20 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run TestHCD020PanelIcons -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.254s
- 2026-11-11 19:20 -05:00 · tarea 7 · verde · vigente · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	12.019s

## Siguiente
Cerrada con aprobación sobre la revisión d3119f5e9229.

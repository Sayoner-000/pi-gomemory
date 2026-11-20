# Vista en árbol del progreso y llamadas por tarea en /tasks

ID: HCD-054
Tipo: feature
Carril: fast
Motivo: Mejora visual autocontenida en extensions/gomemory-tasks.ts; el motor HCD sigue siendo la única fuente de verdad de las tareas y no altera contratos externos ni datos persistentes

## Intención
Mostrar en el panel `/tasks` (`extensions/gomemory-tasks.ts`) un árbol visual de progreso bajo cada tarea HCD (en progreso o completada), listando las llamadas de herramientas ejecutadas (`Read`, `Edit`, `Write`, `Bash`, MCP, etc.) con glifo de estado (`⟳` en curso, `✓` completada, `✗` error), argumentos clave resumidos, duración en `ms` y un pie resumen (`└ N calls · M lines output · click to show more` / `/tasks expand N`), manteniendo el historial de la sesión actual en memoria y sin alterar la fuente de verdad de `mem hcd show --json`.

## Antecedentes y normas aplicables
- Antecedentes: `HCD-008` creó el panel `/tasks`, `HCD-048` blindó el toggle concurrente de `refresh`, `HCD-051` añadió polling periódico en `bash` y bloqueo de `mem hcd verify` agrupados, y `HCD-052` corrigió el análisis sintáctico de `isBundledHcdVerifyCommand`. Esta unidad extiende la presentación visual de `extensions/gomemory-tasks.ts` preservando todos esos contratos.
- Normas aplicables: Constitución §1.4 (TypeScript), §10 (pruebas deterministas), §16 (calidad), §18 (HCD) y §21 (simplicidad); Playbook `HCD-FLOW-01`, `HCD-FLOW-02`, `HCD-DEF-01`, `HCD-TASK-01`, `HCD-TDD-01` y `HCD-EVIDENCE-01`.

## Alcance
Incluido: seguimiento en memoria por sesión de llamadas de herramientas en `extensions/gomemory-tasks.ts` (`tool_execution_start`, `tool_execution_end`, `tool_result`), cálculo de duración en `ms` y líneas de salida por llamada, formato en árbol (`├`, `└`) con resumen de argumentos (sin exponer secretos ni payloads completos), soporte de expandir/colapsar por tarea vía subcomando `/tasks` y clic en componente de widget TUI (con compatibilidad hacia atrás para `setWidget` y los tests de `HCD-008`/`HCD-048`/`HCD-051`/`HCD-052`), y pruebas unitarias deterministas en `pkg/installer/hcd054_test.go`.
Excluido: persistir trazas de herramientas en disco o en la base de `gomemory`, modificar el motor `mem hcd` o el formato de `mem hcd show --json`, alterar `extensions/gomemory-tdd.ts` o `extensions/gomemory-sidebar.ts`, o modificar la configuración de instalación/release.

## ¿Qué pasa si no lo hacemos?
`/tasks` solo muestra una lista plana de tareas (`○`/`✓`/`✗`) sin visibilidad de qué herramientas está ejecutando el agente en una tarea en curso ni cuántas llamadas/líneas produjo una tarea ya completada; la persona debe desplazarse por todo el transcript para inspeccionar el progreso detallado.

## Alternativa más barata
Seguir revisando cada bloque de herramienta directamente en el transcript del chat. No requiere cambios, pero mezcla el trabajo de múltiples tareas y no permite consultar el árbol colapsado/expandido por tarea desde `/tasks`.

## Aclaraciones
- [x] Q1. ¿Qué tareas deben mostrar su árbol de llamadas expandido por defecto y con qué tope visible? (1) Solo la tarea en curso expandida por defecto (últimas 5 llamadas + pie `└ N calls · M lines output · click to show more`) y las completadas colapsadas con resumen en una línea expandible por clic o `/tasks expand N` (recomendada); (2) tarea en curso y última tarea completada expandidas (tope 5 cada una); (3) todas las tareas con llamadas expandidas por defecto. → 1
- [x] Q2. ¿Cómo asociamos las llamadas de herramientas (`Read`, `Edit`, `Bash`, etc.) a una tarea HCD cuando aún no ha corrido `mem hcd verify --task N`? (1) Atribuirlas automáticamente a la primera tarea no verificada (`pending`/`unverified`/`failed`), re-fijar el foco a `N` cuando corra `mem hcd verify --task N` y permitir `/tasks focus N` (recomendada); (2) agruparlas en una rama genérica de actividad y anidar bajo la tarea `N` solo a partir de `mem hcd verify --task N`. → 1

## Checklist
- [x] 1. Implementar en `extensions/gomemory-tasks.ts` el registro en memoria de llamadas de herramientas por tarea (`tool_execution_start`/`tool_execution_end`/`tool_result`), cálculo de duración y líneas de salida, resumen seguro de argumentos y atribución automática/manual según Q2 → `go test ./pkg/installer -run TestHCD054ActivityTracking -v`
- [x] 2. Implementar en `extensions/gomemory-tasks.ts` el renderizado en árbol (`├`, `└`) por tarea en progreso o completada con glifo (`⟳`/`✓`/`✗`), duración en `ms`, tope de 5 llamadas visibles y pie resumen (`N calls · M lines output · click to show more`) según Q1 → `go test ./pkg/installer -run TestHCD054TreeRendering -v`
- [x] 3. Integrar en `extensions/gomemory-tasks.ts` el widget interactivo con clic (`handleMouse` en `setWidget`), los subcomandos `/tasks expand [N]`, `/tasks collapse [N]` y `/tasks focus N`, y refresco en vivo sin romper los contratos previos (`HCD-008`, `HCD-048`, `HCD-051`, `HCD-052`) → `go test ./pkg/installer -run TestHCD054WidgetInteraction -v`
- [x] 4. Validar análisis estático y suite completa con detector de carreras sin regresiones → `go vet ./... && go test -race ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-19 20:10 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD054ActivityTracking -v` (salida 1)
      === RUN   TestHCD054ActivityTracking
          hcd054_test.go:6: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD054ActivityTracking2218657427/001/test.mjs:5
              	countOutputLines,
              	^^^^^^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tasks.ts' does not provide an export named 'countOutputLines'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      --- FAIL: TestHCD054ActivityTracking (0.14s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.501s
      FAIL
- 2026-11-19 20:11 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD054ActivityTracking -v` (salida 0)
      === RUN   TestHCD054ActivityTracking
          hcd054_test.go:75: 
      --- PASS: TestHCD054ActivityTracking (0.14s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.315s
- 2026-11-19 20:11 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD054TreeRendering -v` (salida 1)
      === RUN   TestHCD054TreeRendering
          hcd054_test.go:79: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD054TreeRendering3734580743/001/test.mjs:3
              import { createTaskActivityTracker, renderTasks, renderTasksTree } from "./gomemory-tasks.ts";
                                                               ^^^^^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tasks.ts' does not provide an export named 'renderTasksTree'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      --- FAIL: TestHCD054TreeRendering (0.15s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.371s
      FAIL
- 2026-11-19 20:12 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD054TreeRendering -v` (salida 0)
      === RUN   TestHCD054TreeRendering
          hcd054_test.go:138: 
      --- PASS: TestHCD054TreeRendering (0.15s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.306s
- 2026-11-19 20:12 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD054WidgetInteraction -v` (salida 1)
          hcd054_test.go:142: node falló: exit status 1
              node:internal/modules/run_main:111
                  triggerUncaughtException(
                  ^
              AssertionError [ERR_ASSERTION]: session_start debe montar el componente interactivo del widget
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD054WidgetInteraction4250108045/001/test.mjs:48:8 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: null,
                expected: true,
                operator: '==',
                diff: 'simple'
              }
              Node.js v26.13.0
      --- FAIL: TestHCD054WidgetInteraction (0.14s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.348s
      FAIL
- 2026-11-19 20:13 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD054WidgetInteraction -v` (salida 0)
      === RUN   TestHCD054WidgetInteraction
          hcd054_test.go:232: 
      --- PASS: TestHCD054WidgetInteraction (0.14s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.302s
- 2026-11-19 20:13 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestHCD054ActivityTracking -v` (salida 0)
      === RUN   TestHCD054ActivityTracking
          hcd054_test.go:75: 
      --- PASS: TestHCD054ActivityTracking (0.13s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.292s
- 2026-11-19 20:14 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD054TreeRendering -v` (salida 0)
      === RUN   TestHCD054TreeRendering
          hcd054_test.go:138: 
      --- PASS: TestHCD054TreeRendering (0.13s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.299s
- 2026-11-19 20:14 -05:00 · tarea 4 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	85.064s

## Siguiente
Cerrada con aprobación sobre la revisión 9fc6a64ed60c.

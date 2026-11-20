# Tareas HCD más legibles y sugerencia del siguiente comando en Pi

ID: HCD-057
Tipo: feature
Carril: fast
Motivo: Mejora visual y de interacción acotada a extensiones de pi-gomemory; sin datos persistentes ni contratos públicos; las decisiones abiertas son independientes y se resuelven como aclaraciones

## Intención
En /tasks: glifos de estado más claros, progreso visible del árbol de trabajo y show more para la lista de tareas cuando son muchas. En el input de Pi: sugerir el siguiente comando como texto fantasma atenuado que la persona acepta con una tecla, inspirado en Claude Code.

Estado de partida (revisado en `extensions/gomemory-tasks.ts`): HCD-054 entregó el árbol de llamadas por tarea con «click to show more» de llamadas (5 visibles). La lista de tareas se pinta entera sin límite, la cabecera solo muestra `hechas/total`, la tarea y la llamada en curso usan `⟳` fijo y los glifos no llevan color. Ninguna extensión de pi-gomemory reemplaza el editor de entrada; Pi 1.5.2 lo permite con `ctx.ui.setEditorComponent` y `CustomEditor` (docs/extensions.md de pi-coding-agent).

## Alcance
Incluido: en `extensions/gomemory-tasks.ts`, spinner animado (los mismos `SPINNER_FRAMES` de /tdd) para la tarea y la llamada en curso, color por estado con el tema (`success`, `error`, `warning`, `dim`), conectores de árbol `├─`/`└─` con el último hijo cerrado en `└─`, y barra de progreso en la cabecera (`▰▰▰▱▱ 5/13 · 38 %`); ventana de 8 filas de tareas que sigue a la tarea activa, con indicadores «↑ N más» y «↓ N más», desplazable con la rueda del ratón sobre el panel y con Alt+Shift+↑/↓ desde el input (Q3, Q4: Alt+↑ ya es app.message.dequeue en Pi); ↑/↓ solos siguen recuperando el historial. Nueva extensión `extensions/gomemory-suggest.ts`: editor que extiende `CustomEditor` y envuelve el editor que ya hubiera (`getEditorComponent`), muestra la sugerencia atenuada solo con el input vacío y la copia al input con Tab sin enviarla (Q2); con texto en el input, Tab conserva su autocompletado. Origen de la sugerencia (Q1): con unidad HCD activa, el comando de `next_action` del motor (`mem hcd show --json`), sin tokens; sin unidad, una sugerencia breve del modelo al terminar el turno del agente, con tope de tokens y cancelada si la persona empieza a escribir. Registrar la extensión en el instalador como las demás.
Excluido: enviar la sugerencia con Enter (solo se completa con Tab); cambiar el motor HCD o el formato de `mem hcd show`; persistir sugerencias entre sesiones; reemplazar el historial de ↑/↓; cambiar el árbol de llamadas de HCD-054 más allá de glifos y conectores; sugerencias del modelo cuando la sesión no tiene un modelo disponible.

## ¿Qué pasa si no lo hacemos?
Con unidades de 10 o más tareas, el panel /tasks empuja el chat fuera de pantalla y no permite ver la tarea activa sin perder el resto; el estado de cada tarea se distingue mal porque los glifos no tienen color ni animación. La persona tiene que recordar o copiar a mano el siguiente comando HCD (`/hcd-implement`, `/hcd-close`…) aunque el motor ya lo conoce.

## Alternativa más barata
Limitar la lista a las N primeras tareas y escribir el siguiente comando como línea de estado en el pie de Pi (`setStatus`). Es casi gratis, pero no permite recorrer las tareas ocultas ni completar el comando con Tab, que son las dos cosas que pidió la persona; se descarta.

## Normas aplicables
- Constitución §10 Pruebas: deterministas, sin depender del orden; cada tarea con test automatizable observa el rojo antes del cambio.
- Constitución §14 Frontend: legibilidad y accesibilidad; el color nunca es el único portador del estado (cada estado conserva su glifo).
- Constitución §9 Configuración y secretos: la sugerencia del modelo no envía secretos ni contenido fuera del proveedor que la sesión ya usa.
- Playbook HCD-TDD-01 y HCD-EVIDENCE-01: rojo antes del verde y evidencia vigente.

## Aclaraciones
- [x] Q1. ¿De dónde sale el comando que se sugiere atenuado en el input de Pi: (1) motor HCD con unidad activa y LLM sin unidad; (2) solo motor HCD; (3) solo LLM? → Motor HCD + LLM (Recommended)
- [x] Q2. ¿Con qué tecla se acepta la sugerencia atenuada: (1) Tab y flecha derecha; (2) solo Tab; (3) solo flecha derecha? → Solo Tab — «la atenuación o sugerencia se complete con la tecla tab»
- [x] Q3. ¿Cómo se recorren muchas tareas en /tasks, sabiendo que ↑/↓ en el input de Pi recuperan el historial: (1) rueda del ratón sobre el panel y Alt+↑/↓ con 8 filas visibles e indicadores ↑/↓ N más; (2) ↑/↓ con el input vacío; (3) modo foco /tasks focus? → Rueda + Alt+↑/↓ (Recommended) — la persona pidió «moverse con flecha hacia arriba y abajo para ver las filas y haga scroll»
- [x] Q4. Alt+↑ ya está asignado en Pi 1.5.2 a app.message.dequeue: ¿qué teclas desplazan /tasks: (1) Alt+Shift+↑/↓; (2) Alt+↑/↓ quitando dequeue; (3) solo rueda y clic? → Alt+Shift+↑/↓ (Recommended)
- [x] Q5. ¿Qué glifos de estado usa /tasks, si ◌ (caducada) frente a ○ (pendiente) confunde y la tarea en curso debería animarse: (1) círculo que se llena ○ pendiente, ◐◓◑◒ girando en curso (la activa siempre), ● verificada, ◍ caducada, ✗ fallida; (2) parpadeo ○/●; (3) mantener ✓ y spinner con ↻ para caducada? → Círculo que se llena (Recommended) — la persona: «◌ entonces debería estar rellena cuando se verifique» y «○ si está en progreso debería parpadear»

## Checklist
- [x] 1. Pintar en /tasks las tareas con el círculo que se llena (Q5: ○ pendiente, ◐◓◑◒ girando para la tarea activa aunque no haya llamada, ● verificada, ◍ caducada, ✗ fallida), la llamada en curso con el spinner de /tdd, el color del tema por estado sin quitar el glifo, y los conectores ├─/└─ con el último hijo en └─ → `go test ./pkg/installer -run '^TestHCD057Glyphs$' -count=1`
- [x] 2. Mostrar en la cabecera de /tasks una barra de progreso con hechas/total y porcentaje → `go test ./pkg/installer -run '^TestHCD057Progress$' -count=1`
- [x] 3. Limitar la lista de /tasks a una ventana de 8 filas que sigue a la tarea activa, con indicadores «↑ N más» y «↓ N más» → `go test ./pkg/installer -run '^TestHCD057TaskWindow$' -count=1`
- [x] 4. Desplazar la ventana de /tasks con la rueda del ratón sobre el panel y con Alt+Shift+↑/↓ desde el input, sin tocar ↑/↓ solos ni Alt+↑ → `go test ./pkg/installer -run '^TestHCD057Scroll$' -count=1`
- [x] 5. Crear gomemory-suggest.ts con un editor que extiende CustomEditor y envuelve el editor previo (getEditorComponent) → `go test ./pkg/installer -run '^TestHCD057EditorWrap$' -count=1`
- [x] 6. Sugerir atenuado, con el input vacío y unidad HCD activa, el comando del next_action del motor → `go test ./pkg/installer -run '^TestHCD057HCDSuggestion$' -count=1`
- [x] 7. Completar la sugerencia con Tab sin enviarla y dejar Tab con su autocompletado cuando el input tiene texto → `go test ./pkg/installer -run '^TestHCD057TabAccept$' -count=1`
- [x] 8. Sin unidad activa, pedir al modelo una sugerencia breve al terminar el turno, con tope de tokens y cancelada si la persona escribe → `go test ./pkg/installer -run '^TestHCD057LLMSuggestion$' -count=1`
- [x] 9. Instalar gomemory-suggest.ts con el resto de extensiones y retirarla al desinstalar → `go test ./pkg/installer -run '^TestHCD057Install$' -count=1`
- [x] 10. Comprobar en Pi real con las extensiones instaladas (SHA256 igual al repositorio) la ventana y el scroll de /tasks con 13 tareas, la sugerencia HCD y la del modelo, y Tab → observación manual registrada con capturas de pantalla de Herdr
- [x] 11. Ejecutar la regresión completa → `go vet ./... && go test -race ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-20 03:37 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057Glyphs$' -count=1` (salida 1)
      --- FAIL: TestHCD057Glyphs (0.26s)
          hcd057_test.go:27: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD057Glyphs396246958/001/test.mjs:2
              import registerTasks, { createTaskActivityTracker, renderTasksTree, SPINNER_FRAMES } from "./gomemory-tasks.ts";
                                                                                  ^^^^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tasks.ts' does not provide an export named 'SPINNER_FRAMES'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.915s
      FAIL
- 2026-11-20 03:38 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057Glyphs$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.750s
- 2026-11-20 03:38 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057Progress$' -count=1` (salida 1)
      --- FAIL: TestHCD057Progress (0.25s)
          hcd057_test.go:110: TestHCD057Progress: FAIL esperaba «▰▰▱▱▱▱▱▱▱▱ 1/5 · 20 %» en «HCD-057 · feature · paso implement · 1/5» | esperaba «▱▱▱▱▱▱▱▱▱▱ 0/3 · 0 %» en «HCD-057 · feature · paso implement · 0/3» | esperaba «▰▰▰▰▰▰▰▰▰▰ 2/2 · 100 %» en «HCD-057 · feature · paso implement · 2/2» | esperaba «▰▰▰▰▱▱▱▱▱▱ 5/13 · 38 %» en «HCD-057 · feature · paso implement · 5/13»
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.474s
      FAIL
- 2026-11-20 03:38 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057Progress$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.446s
- 2026-11-20 03:39 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057TaskWindow$' -count=1` (salida 1)
      --- FAIL: TestHCD057TaskWindow (0.25s)
          hcd057_test.go:128: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD057TaskWindow2314510289/001/test.mjs:2
              import { renderTasksTree, createTaskActivityTracker, TASK_WINDOW } from "./gomemory-tasks.ts";
                                                                   ^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tasks.ts' does not provide an export named 'TASK_WINDOW'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.486s
      FAIL
- 2026-11-20 03:39 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057TaskWindow$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.449s
- 2026-11-20 03:41 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057Scroll$' -count=1` (salida 1)
      --- FAIL: TestHCD057Scroll (0.25s)
          hcd057_test.go:211: TestHCD057Scroll: FAIL la rueda sobre el panel debe manejarse | rueda abajo 1: 4,5,6,7,8,9,10,11 | rueda arriba hasta el tope: 4,5,6,7,8,9,10,11 | rueda abajo hasta el tope: 4,5,6,7,8,9,10,11 | falta el atajo alt+shift+up | falta el atajo alt+shift+down | Alt+Shift+↑: 4,5,6,7,8,9,10,11 | Alt+Shift+↓: 4,5,6,7,8,9,10,11 | clic en ↑: 4,5,6,7,8,9,10,11 | el indicador debe nombrar Alt+Shift+↑:   ↑ 3 más · Alt+↑ o rueda
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.626s
      FAIL
- 2026-11-20 03:41 -05:00 · tarea 4 · falló · caducada · `go test ./pkg/installer -run '^TestHCD057Scroll$' -count=1` (salida 1)
      --- FAIL: TestHCD057Scroll (0.25s)
          hcd057_test.go:211: TestHCD057Scroll: FAIL rueda arriba hasta el tope: 4,5,6,7,8,9,10,11 | rueda abajo hasta el tope: 5,6,7,8,9,10,11,12 | Alt+Shift+↑: 4,5,6,7,8,9,10,11 | Alt+Shift+↓: 5,6,7,8,9,10,11,12 | clic en ↑: 4,5,6,7,8,9,10,11
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.453s
      FAIL
- 2026-11-20 03:42 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057Scroll$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.443s
- 2026-11-20 03:44 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057EditorWrap$' -count=1` (salida 1)
      --- FAIL: TestHCD057EditorWrap (0.01s)
          hcd057_suggest_test.go:53: open ../../extensions/gomemory-suggest.ts: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.387s
      FAIL
- 2026-11-20 03:45 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057HCDSuggestion$' -count=1` (salida 1)
      --- FAIL: TestHCD057HCDSuggestion (0.00s)
          hcd057_suggest_test.go:86: open ../../extensions/gomemory-suggest.ts: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.213s
      FAIL
- 2026-11-20 03:45 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057TabAccept$' -count=1` (salida 1)
      --- FAIL: TestHCD057TabAccept (0.01s)
          hcd057_suggest_test.go:121: open ../../extensions/gomemory-suggest.ts: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.279s
      FAIL
- 2026-11-20 03:45 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057LLMSuggestion$' -count=1` (salida 1)
      --- FAIL: TestHCD057LLMSuggestion (0.00s)
          hcd057_suggest_test.go:157: open ../../extensions/gomemory-suggest.ts: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.201s
      FAIL
- 2026-11-20 03:46 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057EditorWrap$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.450s
- 2026-11-20 03:46 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057HCDSuggestion$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.482s
- 2026-11-20 03:46 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057TabAccept$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.428s
- 2026-11-20 03:46 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057LLMSuggestion$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.523s
- 2026-11-20 03:46 -05:00 · tarea 9 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057Install$' -count=1` (salida 1)
      --- FAIL: TestHCD057Install (0.06s)
          hcd057_suggest_test.go:59: gomemory-suggest.ts no se instala o difiere del embebido: stat /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD057Install3276233145/001/.pi/agent/extensions/gomemory-suggest.ts: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.279s
      FAIL
- 2026-11-20 03:47 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057Install$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.282s
- 2026-11-20 03:49 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057EditorWrap$' -count=1` (salida 1)
      --- FAIL: TestHCD057EditorWrap (0.55s)
          hcd057_suggest_test.go:120: TestHCD057EditorWrap: FAIL el editor que llega después debe quedar envuelto por el nuestro
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.944s
      FAIL
- 2026-11-20 03:50 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057Progress$' -count=1` (salida 1)
      --- FAIL: TestHCD057Progress (0.25s)
          hcd057_test.go:111: TestHCD057Progress: FAIL esperaba «▰▰▱▱▱▱▱▱▱▱  1/5 · 20 %» en «HCD-057 · feature · paso implement · ▰▰▱▱▱▱▱▱▱▱ 1/5 · 20 %» | esperaba «▱▱▱▱▱▱▱▱▱▱  0/3 · 0 %» en «HCD-057 · feature · paso implement · ▱▱▱▱▱▱▱▱▱▱ 0/3 · 0 %» | esperaba «▰▰▰▰▰▰▰▰▰▰  2/2 · 100 %» en «HCD-057 · feature · paso implement · ▰▰▰▰▰▰▰▰▰▰ 2/2 · 100 %» | esperaba «▰▰▰▰▱▱▱▱▱▱  5/13 · 38 %» en «HCD-057 · feature · paso implement · ▰▰▰▰▱▱▱▱▱▱ 5/13 · 38 %»
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.489s
      FAIL
- 2026-11-20 03:50 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057Glyphs$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.744s
- 2026-11-20 03:50 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057Progress$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.449s
- 2026-11-20 03:50 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057TaskWindow$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.458s
- 2026-11-20 03:50 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057Scroll$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.453s
- 2026-11-20 03:50 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057EditorWrap$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.731s
- 2026-11-20 03:50 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057HCDSuggestion$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.445s
- 2026-11-20 03:50 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057TabAccept$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.448s
- 2026-11-20 03:50 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057LLMSuggestion$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.518s
- 2026-11-20 03:50 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057Install$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.268s
- 2026-11-20 03:53 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057LLMSuggestion$' -count=1` (salida 1)
      --- FAIL: TestHCD057LLMSuggestion (0.26s)
          hcd057_suggest_test.go:257: TestHCD057LLMSuggestion: FAIL sin unidad, al terminar el turno debe pedirse una sugerencia | el contexto debe incluir el último intercambio | debe mostrar la primera línea sin comillas:  | escribir debe abortar la petición en curso
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.641s
      FAIL
- 2026-11-20 03:53 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057LLMSuggestion$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.528s
- 2026-11-20 03:55 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057LLMSuggestion$' -count=1` (salida 1)
      --- FAIL: TestHCD057LLMSuggestion (0.33s)
          hcd057_suggest_test.go:260: TestHCD057LLMSuggestion: FAIL tope de tokens fuera de rango: 40 | la petición lateral debe pedir el razonamiento mínimo: undefined
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.562s
      FAIL
- 2026-11-20 03:55 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD057LLMSuggestion$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.530s
- 2026-11-20 03:57 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD057Glyphs$' -count=1` (salida 1)
      --- FAIL: TestHCD057Glyphs (0.26s)
          hcd057_test.go:27: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD057Glyphs1598014619/001/test.mjs:2
              import registerTasks, { createTaskActivityTracker, renderTasksTree, SPINNER_FRAMES, TASK_FRAMES } from "./gomemory-tasks.ts";
                                                                                                  ^^^^^^^^^^^
              SyntaxError: The requested module './gomemory-tasks.ts' does not provide an export named 'TASK_FRAMES'
                  at #asyncInstantiate (node:internal/modules/esm/module_job:463:21)
                  at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.639s
      FAIL
- 2026-11-20 03:58 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD057Glyphs$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.346s
- 2026-11-20 03:58 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD057Progress$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.443s
- 2026-11-20 03:58 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD057TaskWindow$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.436s
- 2026-11-20 03:58 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD057Scroll$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.460s
- 2026-11-20 03:58 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD057EditorWrap$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.744s
- 2026-11-20 03:58 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run '^TestHCD057HCDSuggestion$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.437s
- 2026-11-20 03:58 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run '^TestHCD057TabAccept$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.446s
- 2026-11-20 03:58 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run '^TestHCD057LLMSuggestion$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.626s
- 2026-11-20 03:58 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run '^TestHCD057Install$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.259s
- 2026-11-20 03:58 -05:00 · tarea 10 · manual · vigente
      Pi 1.5.2 real vía Herdr (pestaña w3X:t4, 176 columnas), extensiones instaladas con SHA256 igual al repo (gomemory-tasks.ts 831eac52, gomemory-suggest.ts 8239ba1b). /tasks en el repo con HCD-057 (11 tareas): cabecera «▰▰▰▰▰▰▰▰▱▱  9/11 · 82 %» (barra separada por dos espacios tras el comentario de la persona); ventana de 8 tareas que sigue a la activa con «↑ 3 más · Alt+Shift+↑ o rueda»; Alt+Shift+↑ pasó a «↑ 2 más / ↓ 1 más»; rueda SGR abajo volvió al final y dos ruedas arriba dejaron «↑ 1 más / ↓ 2 más». Glifos Q5: ● verificadas, ○ pendiente, la activa giró ◐→◓ entre dos lecturas a 0,3 s. Sugerencia HCD: «❯   /hcd-implement · Tab» en color dim del tema, conservando el ❯ de pi-cc-extensions; Tab dejó «❯ /hcd-implement» sin enviar. Hallazgos corregidos en esta tarea: (1) pi-cc-extensions reemplazaba nuestro editor tras session_start → ensureEditor lo vuelve a envolver; (2) Opus 5.5 gastaba el tope de 40 tokens razonando (stopReason length, sin texto) incluso sin reasoning → streamSimple con reasoning minimal y tope 300. Sugerencia del modelo sin unidad (módulo de prueba): «❯   Ejecuta los tests de suma_test.go · Tab» ~2 s tras el turno, 42-103 tokens (≈0,002-0,003 USD); Tab la copió sin enviar (agente idle).
- 2026-11-20 03:58 -05:00 · tarea 11 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	44.132s

## Siguiente
Cerrada con aprobación sobre la revisión d446cc5712f9.

# Pi recibe el protocolo y el contexto en vivo desde gomemory, no desde AGENTS.md

ID: HCD-045
Tipo: evolution
Carril: full
Motivo: decisiones interdependientes (fuente del protocolo, momento de inyección, migración de AGENTS.md) y toca el arranque de toda sesión de Pi

## Intención
Que el agente de Pi reciba reglas, baseline, memoria y protocolo desde gomemory en cada sesión mediante una extensión (before_agent_start / systemPromptOptions.sections), y que pi-gomemory deje de escribir una copia estática del protocolo en ~/.pi/agent/AGENTS.md, retirando el bloque heredado sin tocar el resto del archivo


## Contexto y evidencia
- Hoy `InstallInstructions` (`pkg/installer/instructions.go:60`) escribe la constante `GomemoryPiInstructions` en `~/.pi/agent/AGENTS.md`. Es una copia estática: se desfasó del código (causa raíz de HCD-044, memoria 439) y solo se renueva con `install`/`update`. El `AGENTS.md` real de la persona sigue hoy con el protocolo anterior a HCD-044.
- Contradice la regla 11 del proyecto: gomemory es la única fuente de instrucciones.
- Pi 1.5.2 no lleva el protocolo de gomemory al modelo por MCP. Según `docs/mcp.md:204`, de los servidores con tools `codemode` o `deferred` solo entra al system prompt **una línea** (la `description` o la primera línea de las instructions); de un servidor todo `direct` no entra ninguna. Según `mcp.md:236`, las instructions completas solo se leen desde codemode con `describeNamespace()`.
- Pi sí ofrece el mecanismo: `before_agent_start` expone `event.systemPromptOptions.sections` y `.selectedTools` (`docs/extensions.md:615-660`).
  - Al cambiar `sections`, Pi envía un parche solo de lo que cambió. Los modelos que aceptan mensajes de sistema a mitad de conversación conservan la caché; los demás pierden la caché una vez por cambio (`extensions.md:657`). Una sección estable durante la sesión no provoca ninguna de las dos cosas: eso justifica Q2.
  - Devolver `systemPrompt`, como hace hoy `extensions/gomemory-plan.ts:111`, reemplaza el prompt entero y pierde la caché cada vez que cambia (motivo de Q5).
- `session_start` se dispara con `reason` = `startup | reload | new | resume | fork` (`extensions.md:473-481`). El directorio de trabajo de cada sesión llega en `ctx.cwd`.
- `mem context`, medido con `/usr/bin/time -p mem context | wc -c` en este repositorio el 2026-11-17: 0,04 s y 11 501 bytes. Contiene reglas fijadas, baseline, memorias y sesión, incluidas las reglas de memoria (save_memory, search_memories, plan-context), pero no el bloque específico de Pi (codemode, `splash_review`, HCD). Se usa solo `mem context`: `mem hook session-start` es un punto de entrada interno de hooks y no se usa.
- Antecedentes:
  - HCD-032 (AGENTS.md conserva sus permisos) y HCD-036 (enlaces simbólicos) regulan cómo se edita `AGENTS.md`.
  - `uninstall` retira el bloque con `stripInstructions` (`pkg/installer/uninstall.go:496`). Si el resultado queda vacío, borra el archivo normal y vacía el destino si es un enlace (`uninstall.go:565-569`); lo exige el test `hcd011_apply_test.go:50`.
  - `update` relanza `install` con el binario nuevo (`main.go`, `runUpdateWith`), así que hereda el comportamiento de `install`.
- Código y tests que hoy dependen del bloque escrito, que esta unidad debe adaptar:
  - `codemode_test.go` (`protocolDirectTools`/`TestCodemodeProtocolCoherence`).
  - `hcd012_atomic_test.go:72`, `hcd032_agents_mode_test.go`, `hcd036_symlink_test.go:143` e `installer_test.go:105-124`.
  - La detección del formato heredado sin marcadores en `stripInstructions`/`instructionsBlock`, que compara con la constante `GomemoryPiInstructions`.
  - La lista de extensiones instaladas (`pkg/installer/extension.go:13-20`).

## Alcance
Incluido: extensión de contexto, migración de /plan a sections, retirada del bloque heredado en install/update, aviso en doctor y tests.
- Una extensión nueva, `extensions/gomemory-context.ts`, instalada y desinstalada como las demás. En `before_agent_start` añade a `systemPromptOptions.sections` una sección con `mem context` y el bloque específico de Pi.
- Migrar `gomemory-plan.ts` para que inyecte su método como sección y no reemplace `systemPrompt` (Q5).
- `install`/`update` dejan de escribir el protocolo en `AGENTS.md` y retiran el bloque heredado con la misma lógica que `uninstall` (Q4).
- `doctor` avisa si el bloque heredado sigue presente.
- Se conserva una copia congelada del texto heredado, solo para detectar el formato sin marcadores.
- Adaptar los tests listados y hacer una prueba con Pi real.

Excluido: gomemory y su MCP, el AGENTS.md de proyecto, el perfil de HCD-044 y el refresco dentro de una sesión.
- Cambiar gomemory (`mem`) o su servidor MCP.
- Tocar el `AGENTS.md` de proyecto (`./AGENTS.md`) o detectar si duplica el protocolo: Pi lo carga como `contextFiles` y es de la persona.
- El perfil codemode de HCD-044.
- Refrescar la sección dentro de una sesión (Q2).

## ¿Qué pasa si no lo hacemos?
El agente de Pi sigue trabajando con una copia del protocolo que se desfasa con cada cambio del código (como en HCD-044) y que no recibe reglas, baseline ni memoria fijada de gomemory al arrancar: en Pi no existe el equivalente del hook SessionStart de Claude Code. Cada cambio de reglas en gomemory exige reinstalar pi-gomemory para que Pi lo vea, y la regla 11 queda incumplida.

## Alternativa más barata
Mantener `AGENTS.md` y regenerarlo en cada `pi-gomemory` (el launcher) antes de arrancar Pi. Es más barato, pero sigue escribiendo instrucciones en un archivo de la persona, mantiene la copia estática del bloque de Pi y no inyecta la memoria del proyecto según el directorio de trabajo (el contexto depende del proyecto). Se descarta.

## Historias
### US1. El agente de Pi arranca con el contexto vivo de gomemory
Como persona que usa Pi con pi-gomemory, quiero que cada sesión reciba las reglas, el baseline y la memoria del proyecto desde gomemory, para no depender de una copia instalada que se queda vieja.
- C-01: Dado un proyecto con memorias en gomemory, cuando se dispara `session_start` (cualquier `reason`), la extensión lanza `mem context` con `cwd = ctx.cwd` y un timeout de 3000 ms, sin bloquear el arranque. En cada `before_agent_start` de esa sesión espera la carga pendiente (con el mismo límite) y añade a `systemPromptOptions.sections` la sección `gomemory`, con texto idéntico en todas las ejecuciones de la sesión (Q2).
- C-02: Dado que `mem` no existe, devuelve un código distinto de 0, una salida vacía o supera el timeout, cuando empieza la sesión, entonces Pi arranca igual, la sección solo lleva el bloque Pi (Q6) y aparece un único aviso por sesión (`ctx.ui.notify`, nivel warning) con la causa. Nunca se inyecta texto que imite la salida de `mem context`.
- C-03: Dado un cambio en las reglas fijadas de gomemory, cuando empieza una sesión nueva (`startup`, `new`, `resume`, `fork` o `reload`), entonces la sección refleja el cambio sin reinstalar pi-gomemory.
- C-08: Dada una salida de `mem context` de más de 24 KB (24 576 bytes), cuando se construye la sección, entonces se recorta por el final a ese tamaño, sin partir un carácter UTF-8, y termina con la línea «… contexto recortado: llama get_context() para el resto» (Q7).

### US2. El protocolo específico de Pi viaja con la extensión, no en AGENTS.md
Como persona, quiero que las instrucciones propias de Pi (codemode, splash_review, HCD) lleguen con la extensión instalada, para que coincidan siempre con la versión de pi-gomemory.
- C-04: Dado el estado efectivo de la sesión, cuando se construye el bloque Pi, este no lee `mcp.json`, no nombra tools concretas y no usa listas copiadas a mano (Q8). Remite a las tools de gomemory y codebase-memory que el prompt ya declara, y dice que codemode está activo si y solo si `codemode` está en `systemPromptOptions.selectedTools`; en ese caso, indica que el resto se llama con `tools.<nombre>(args)`. La verificación con Pi real mostró que en `before_agent_start` las tools MCP aún no están en `selectedTools`, porque conectan en segundo plano.
- C-09: Dado `/plan` activo, cuando empieza una ejecución, entonces `systemPromptOptions.sections` contiene a la vez la sección `gomemory` y la sección del método de plan, y `gomemory-plan.ts` ya no devuelve `systemPrompt` (Q5).

### US3. AGENTS.md vuelve a ser solo de la persona
Como persona, quiero que `install` y `update` dejen de escribir en `~/.pi/agent/AGENTS.md` y retiren el bloque heredado, para que ese archivo contenga solo lo mío.
- C-05: Dado un `AGENTS.md` con texto propio y el bloque heredado (con marcadores o en el formato antiguo sin ellos), cuando ejecuto `install`, entonces:
  - el contenido resultante es exactamente `stripInstructions(original)`: el texto anterior y el posterior al bloque, sin el salto de línea que sigue al marcador final y, si no queda texto detrás, sin los saltos de línea finales del texto anterior, más uno;
  - se cubren el bloque en medio y el bloque al final del archivo;
  - se conservan los permisos y el enlace simbólico (la edición se hace en el destino).
- C-06: Dado un `AGENTS.md` que solo contenía el bloque heredado, cuando ejecuto `install`, entonces se aplica lo mismo que `uninstall` (Q4): un archivo normal se borra y un enlace se conserva con su destino vacío.
- C-07: Dado el bloque heredado todavía presente (por ejemplo, porque `install` no pudo escribir en un `AGENTS.md` de solo lectura), cuando ejecuto `doctor`, entonces aparece un WARN «AGENTS.md conserva el protocolo heredado de pi-gomemory» que nombra la orden literal `pi-gomemory-setup install`. Si `install` falla al escribir, informa del error y no aborta el resto de la instalación.
- C-10: Dado `update` con una release nueva, cuando relanza `install`, entonces se cumplen C-05 y C-06, y `install` nunca vuelve a escribir el protocolo en `AGENTS.md`.
- C-11: La extensión `gomemory-context.ts` figura en la lista de `extension.go`, se instala con `install` y se retira con `uninstall`. El texto heredado se conserva congelado solo para la detección, y `TestCodemodeProtocolCoherence` pasa a verificar el bloque generado (C-04).

## Aclaraciones
- [x] Q1. ¿Qué parte del protocolo inyecta la extensión? Opciones: (A, recomendada) `mem context` completo más un bloque Pi mínimo generado en la extensión; (B) solo el bloque Pi y una llamada a `get_context` en el primer turno; (C) `mem context` sin bloque Pi. → mem context + bloque Pi (Recommended)
- [x] Q2. ¿Cuándo se refresca la sección? Opciones: (A, recomendada) una vez por sesión (`session_start` y cambio de sesión), reutilizada en cada ejecución para no romper la caché; (B) en cada ejecución; (C) por sesión y además al terminar un `save_memory`. → Una vez por sesión (Recommended)
- [x] Q3. ¿Qué hace `install` con un `AGENTS.md` que solo contenía el bloque heredado? Opciones: (A, recomendada) dejarlo vacío, sin borrarlo, para no romper un enlace o una expectativa de la persona; (B) borrarlo; (C) no tocarlo y solo avisar en `doctor`. → Dejarlo vacío (Recommended)
- [x] Q4. (S-02, sustituye a Q3) ¿Qué hacen install/update con un AGENTS.md que solo tenía el bloque? Opciones: igual que uninstall (archivo normal vacío se borra; enlace se vacía en su destino) [recomendada] | vaciar siempre y cambiar uninstall → Igual que uninstall (Recommended)
- [x] Q5. (S-03) ¿Qué se hace con /plan, que hoy reemplaza el systemPrompt entero? Opciones: migrar gomemory-plan.ts a systemPromptOptions.sections [recomendada] | dejarlo y solo comprobar que la sección de gomemory sigue presente → Migrar plan a sections (Recommended)
- [x] Q6. (S-09) Si mem falta o falla al iniciar sesión, ¿qué se inyecta? Opciones: solo el bloque Pi + aviso [recomendada] | nada + aviso → Solo el bloque Pi + aviso (Recommended)
- [x] Q7. (S-06) ¿Tope de tamaño para la sección de mem context? Opciones: 24 KB con recorte final y línea «llama get_context() para el resto» [recomendada] | sin tope → Tope 24 KB con recorte (Recommended)
- [x] Q8. (verificación real) En before_agent_start las tools MCP aún no están en selectedTools: el bloque Pi decía «ninguna» con 12 declaradas. ¿Cómo se corrige C-04? Opciones: no listar tools y solo indicar el modo codemode [recomendada] | listar y refrescar en cada ejecución → No listar tools (Recommended)

## Slices
### VS-01. Cada sesión de Pi recibe el contexto vivo de gomemory
Tareas: 1, 2
Criterios: C-01, C-02, C-03, C-08

### VS-02. El bloque Pi refleja las tools efectivas y /plan convive con la sección
Tareas: 3, 4
Criterios: C-04, C-09

### VS-03. AGENTS.md deja de recibir el protocolo y pierde el bloque heredado
Tareas: 5, 6, 7
Criterios: C-05, C-06, C-07, C-10

### VS-04. Instalación completa y comprobación con Pi real
Tareas: 8, 9
Criterios: C-11, C-01

## Checklist
- [x] 1. extensions/gomemory-context.ts: en session_start lanza `mem context` (cwd = ctx.cwd, timeout 3000 ms) sin bloquear, y en before_agent_start espera esa carga y añade la sección `gomemory` a systemPromptOptions.sections, idéntica en todas las ejecuciones de la sesión y recortada a 24 576 bytes con la línea de recorte → `go test ./pkg/installer -run 'TestHCD045Context(Load|Stable|NewSession|Truncate)' -count=1`
- [x] 2. extensions/gomemory-context.ts: si `mem` falta, devuelve un código distinto de 0, sale vacío o supera el timeout, la sección solo lleva el bloque Pi y hay un único aviso warning por sesión → `go test ./pkg/installer -run 'TestHCD045ContextMemFailure' -count=1`
- [x] 3. extensions/gomemory-context.ts: el bloque Pi se deriva de systemPromptOptions.selectedTools (tools de gomemory/codebase-memory y si codemode está activo), sin leer mcp.json; pkg/installer/codemode_test.go deja de leer el protocolo heredado → `go test ./pkg/installer -run 'TestHCD045PiBlock|TestCodemode' -count=1`
- [x] 4. extensions/gomemory-plan.ts: con /plan activo inyecta su método como sección y ya no devuelve systemPrompt; con ambas extensiones, sections contiene `gomemory` y la sección del plan → `go test ./pkg/installer -run 'TestHCD045PlanSections|TestPlan' -count=1`
- [x] 5. pkg/installer/instructions.go: install deja de escribir el protocolo y deja AGENTS.md = stripInstructions(original) (bloque en medio, al final, formato antiguo sin marcadores), conservando permisos y enlace → `go test ./pkg/installer -run 'TestHCD045InstallStrips' -count=1`
- [x] 6. pkg/installer/instructions.go: si AGENTS.md solo tenía el bloque, el archivo normal se borra y el enlace se vacía en su destino; un AGENTS.md de solo lectura se informa sin abortar la instalación → `go test ./pkg/installer -run 'TestHCD045InstallOnlyBlock|TestHCD045InstallReadOnly' -count=1`
- [x] 7. pkg/installer/doctor.go: WARN «AGENTS.md conserva el protocolo heredado de pi-gomemory» con la orden `pi-gomemory-setup install` cuando el bloque sigue presente, y ningún aviso cuando no está → `go test ./pkg/installer -run 'TestHCD045DoctorLegacyBlock' -count=1`
- [x] 8. pkg/installer/extension.go: gomemory-context.ts se instala y se retira con uninstall; los tests heredados (hcd012, hcd032, hcd036, installer_test) se adaptan; vet y suite completa en verde → `bash -c 'go vet ./... && go test ./... -count=1'`
- [x] 9. Pi real (scripts/hcd045-pi-real.sh): tras `go run . install`, una sesión `pi --print` en este repo registra en su jsonl una sección `gomemory` que contiene «Memoria del Proyecto», y ~/.pi/agent/AGENTS.md ya no contiene el bloque `pi-gomemory:start` → `bash scripts/hcd045-pi-real.sh`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-17 14:14 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD045Context(Load|Stable|NewSession|Truncate)' -count=1` (salida 1)
      --- FAIL: TestHCD045ContextLoad (0.00s)
          hcd045_context_test.go:50: open ../../extensions/gomemory-context.ts: no such file or directory
      --- FAIL: TestHCD045ContextStable (0.00s)
          hcd045_context_test.go:74: open ../../extensions/gomemory-context.ts: no such file or directory
      --- FAIL: TestHCD045ContextNewSession (0.00s)
          hcd045_context_test.go:87: open ../../extensions/gomemory-context.ts: no such file or directory
      --- FAIL: TestHCD045ContextTruncate (0.00s)
          hcd045_context_test.go:105: open ../../extensions/gomemory-context.ts: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.374s
      FAIL
- 2026-11-17 14:14 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD045ContextMemFailure' -count=1` (salida 1)
      --- FAIL: TestHCD045ContextMemFailure (0.00s)
          hcd045_context_test.go:126: open ../../extensions/gomemory-context.ts: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.157s
      FAIL
- 2026-11-17 14:14 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD045PiBlock|TestCodemode' -count=1` (salida 1)
      --- FAIL: TestHCD045PiBlock (0.00s)
          hcd045_context_test.go:126: open ../../extensions/gomemory-context.ts: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.233s
      FAIL
- 2026-11-17 14:15 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run 'TestHCD045Context(Load|Stable|NewSession|Truncate)' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.697s
- 2026-11-17 14:15 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run 'TestHCD045ContextMemFailure' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.322s
- 2026-11-17 14:15 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run 'TestHCD045PiBlock|TestCodemode' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.399s
- 2026-11-17 14:15 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD045PlanSections|TestPlan' -count=1` (salida 1)
      --- FAIL: TestHCD045PlanSections (0.13s)
          hcd045_context_test.go:184: FAIL un handler reemplaza systemPrompt | falta la sección del método de plan: gomemory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.379s
      FAIL
- 2026-11-17 14:16 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run 'TestHCD045PlanSections|TestPlan' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.309s
- 2026-11-17 14:17 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD045InstallStrips' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd045_install_test.go:13:36: undefined: legacyPiInstructions
      pkg/installer/hcd045_install_test.go:23:46: undefined: legacyPiInstructions
      pkg/installer/hcd045_install_test.go:43:21: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:55:20: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:70:22: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:80:22: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:96:16: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:107:16: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:128:15: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:132:2: undefined: retireLegacyStep
      pkg/installer/hcd045_install_test.go:132:2: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 14:17 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD045InstallOnlyBlock|TestHCD045InstallReadOnly' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd045_install_test.go:13:36: undefined: legacyPiInstructions
      pkg/installer/hcd045_install_test.go:23:46: undefined: legacyPiInstructions
      pkg/installer/hcd045_install_test.go:43:21: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:55:20: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:70:22: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:80:22: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:96:16: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:107:16: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:128:15: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:132:2: undefined: retireLegacyStep
      pkg/installer/hcd045_install_test.go:132:2: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 14:17 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD045DoctorLegacyBlock' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd045_install_test.go:13:36: undefined: legacyPiInstructions
      pkg/installer/hcd045_install_test.go:23:46: undefined: legacyPiInstructions
      pkg/installer/hcd045_install_test.go:43:21: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:55:20: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:70:22: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:80:22: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:96:16: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:107:16: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:128:15: undefined: RetireLegacyInstructions
      pkg/installer/hcd045_install_test.go:132:2: undefined: retireLegacyStep
      pkg/installer/hcd045_install_test.go:132:2: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 14:18 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run 'TestHCD045InstallStrips' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.411s
- 2026-11-17 14:18 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run 'TestHCD045InstallOnlyBlock|TestHCD045InstallReadOnly' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.177s
- 2026-11-17 14:18 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run 'TestHCD045DoctorLegacyBlock' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.167s
- 2026-11-17 14:18 -05:00 · tarea 8 · falló · caducada · `bash -c 'go vet ./... && go test ./... -count=1'` (salida 1)
        [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean1735905121/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean1735905121/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	70.501s
      FAIL
- 2026-11-17 14:23 -05:00 · tarea 8 · verde · caducada · `bash -c 'go vet ./... && go test ./... -count=1'` (salida 0)
      ok  	pi-gomemory	0.197s
      ok  	pi-gomemory/pkg/installer	70.763s
- 2026-11-17 14:26 -05:00 · tarea 9 · rojo observado · caducada · `bash scripts/hcd045-pi-real.sh` (salida 1)
      FAIL: gomemory-context.ts no está instalada en Pi
- 2026-11-17 14:28 -05:00 · tarea 9 · verde · caducada · `bash scripts/hcd045-pi-real.sh` (salida 0)
      OK: sección gomemory con mem context (8479 bytes) y AGENTS.md sin bloque heredado
- 2026-11-17 14:28 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run 'TestHCD045Context(Load|Stable|NewSession|Truncate)' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.701s
- 2026-11-17 14:31 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run 'TestHCD045Context(Load|Stable|NewSession|Truncate)' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.675s
- 2026-11-17 14:31 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run 'TestHCD045ContextMemFailure' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.278s
- 2026-11-17 14:31 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run 'TestHCD045PiBlock|TestCodemode' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.306s
- 2026-11-17 14:31 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run 'TestHCD045PlanSections|TestPlan' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.322s
- 2026-11-17 14:31 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run 'TestHCD045InstallStrips' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.207s
- 2026-11-17 14:31 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run 'TestHCD045InstallOnlyBlock|TestHCD045InstallReadOnly' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.177s
- 2026-11-17 14:31 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run 'TestHCD045DoctorLegacyBlock' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.160s
- 2026-11-17 14:31 -05:00 · tarea 8 · verde · vigente · `bash -c 'go vet ./... && go test ./... -count=1'` (salida 0)
      ok  	pi-gomemory	0.251s
      ok  	pi-gomemory/pkg/installer	70.984s
- 2026-11-17 14:32 -05:00 · tarea 9 · verde · vigente · `bash scripts/hcd045-pi-real.sh` (salida 0)
      OK: sección gomemory con mem context (8559 bytes) y AGENTS.md sin bloque heredado

## Siguiente
Cerrada con aprobación sobre la revisión ea9aceca3b0b.

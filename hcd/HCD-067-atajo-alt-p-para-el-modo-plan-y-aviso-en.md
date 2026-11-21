# Atajo Alt+P para el modo plan y aviso en el header

ID: HCD-067
Tipo: feature
Carril: fast
Motivo: En Pi solo se entra al modo plan con /plan on|off y el estado solo aparece como PLAN en el pie; la persona pide un atajo (Alt+P, respaldo Ctrl+Alt+P) y que el header muestre MODO PLAN ON en ámbar del theme y plan off atenuado

## Intención
Entrar y salir del modo plan de Pi con una tecla (Alt+P; Ctrl+Alt+P de respaldo para terminales macOS que no envían Option como Alt) y ver siempre en el header si el modo está activo: «◆ MODO PLAN ON · Alt+P» en ámbar `#f4c66b` (color `warning` del theme gomemory) o «○ plan off · Alt+P» atenuado, repintado al instante sin reiniciar Pi.


## Alcance
Incluido: en `extensions/gomemory-plan.ts`, una única función de encendido/apagado compartida por `/plan on|off|run` y los atajos `alt+p` y `ctrl+alt+p` (en Windows/WSL solo `ctrl+alt+p`, porque Pi usa allí `alt+p` para el modelo anterior); publicación del estado en `globalThis[Symbol.for("pi-gomemory.plan")]` con suscriptores; en `extensions/gomemory-banner.ts`, la línea de estado del modo plan en el header (ámbar ON, atenuada OFF, sin escapes con `NO_COLOR`) y su repintado al cambiar el modo; la etiqueta «PLAN» del pie en ámbar cuando el theme está disponible; README y CHANGELOG; pruebas rojo→verde y comprobación en Pi real.
Excluido: cambiar `shift+tab` (thinking de Pi) o cualquier atajo nativo de Pi; usar Ctrl+Tab (las terminales no lo distinguen de Tab); cambiar qué bloquea el modo plan; configurar la terminal de la persona (Option como Alt); publicar release.

## ¿Qué pasa si no lo hacemos?
Para entrar o salir del modo plan hay que escribir `/plan on` o `/plan off`, y el único aviso es una etiqueta «PLAN» pequeña en el pie; la persona puede creer que está en modo plan cuando no lo está (o al revés) y pedir cambios que quedan bloqueados o, peor, que se ejecutan.

## Alternativa más barata
Dejar solo los comandos `/plan on|off` y documentarlos. No da el atajo pedido ni hace visible el estado en el header; tampoco se puede resolver con `keybindings.json` porque el modo plan es un comando de extensión, no una acción de Pi. Decisiones previas respetadas: HCD-045 (método como sección del prompt), memoria 527 (`/plan on|off|status` no envían mensajes), HCD-066 (protocolo de impacto en modo plan) y HCD-057 (convivencia con pi-cc-extensions).


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Añadir en `extensions/gomemory-plan.ts` los atajos `alt+p` y `ctrl+alt+p` (solo `ctrl+alt+p` en Windows/WSL) que alternan el modo con la misma lógica que `/plan on|off`, y publicar el estado en `Symbol.for("pi-gomemory.plan")` → `node --test tests/gomemory-plan.test.mjs`
- [x] 2. Añadir en `extensions/gomemory-banner.ts` `planHeaderLine` (ON ámbar 244;198;107 «MODO PLAN ON», OFF atenuado «plan off», sin escapes con NO_COLOR) en el header y repintarlo al cambiar el modo → `node --test tests/gomemory-banner.test.mjs`
- [x] 3. Regresión completa (sin rojo previo) → `go vet ./... && go test ./... && node --test tests/gomemory-plan.test.mjs tests/gomemory-banner.test.mjs tests/antigravity-prompt-guard/guard.test.mjs`
- [x] 4. Comprobar en Pi real que Alt+P alterna el header entre ámbar ON y OFF atenuado y que con ON `edit` queda bloqueado (sin rojo previo) → observación directa en Pi real

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-21 08:14 -05:00 · tarea 1 · rojo observado · caducada · `node --test tests/gomemory-plan.test.mjs` (salida 1)
          at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
          at async node:internal/modules/esm/loader:650:26
          at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
      Node.js v26.13.0
      ✖ tests/gomemory-plan.test.mjs (229.831041ms)
      ℹ tests 1
      ℹ suites 0
      ℹ pass 0
      ℹ fail 1
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 235.898958
      ✖ failing tests:
      test at tests/gomemory-plan.test.mjs:1:1
      ✖ tests/gomemory-plan.test.mjs (229.831041ms)
        'test failed'
- 2026-11-21 08:14 -05:00 · tarea 1 · verde · caducada · `node --test tests/gomemory-plan.test.mjs` (salida 0)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (3.475875ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (0.376375ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.199208ms)
      ✔ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (0.99575ms)
      ✔ planShortcutKeys: Alt+P y Ctrl+Alt+P; en Windows y WSL solo Ctrl+Alt+P (0.125875ms)
      ✔ Alt+P alterna el modo plan, bloquea y publica el estado para el header (0.249458ms)
      ℹ tests 6
      ℹ suites 0
      ℹ pass 6
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 238.252458
- 2026-11-21 08:15 -05:00 · tarea 2 · rojo observado · caducada · `node --test tests/gomemory-banner.test.mjs` (salida 1)
          at async ModuleJob.run (node:internal/modules/esm/module_job:561:5)
          at async node:internal/modules/esm/loader:650:26
          at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
      Node.js v26.13.0
      ✖ tests/gomemory-banner.test.mjs (245.134583ms)
      ℹ tests 1
      ℹ suites 0
      ℹ pass 0
      ℹ fail 1
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 251.236875
      ✖ failing tests:
      test at tests/gomemory-banner.test.mjs:1:1
      ✖ tests/gomemory-banner.test.mjs (245.134583ms)
        'test failed'
- 2026-11-21 08:15 -05:00 · tarea 2 · verde · caducada · `node --test tests/gomemory-banner.test.mjs` (salida 0)
      ✔ planHeaderLine: ON en ámbar, OFF atenuado, sin escapes con NO_COLOR, centrada (0.921208ms)
      ✔ el header muestra el modo actual y se repinta al cambiarlo (1.9245ms)
      ℹ tests 2
      ℹ suites 0
      ℹ pass 2
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 253.988791
- 2026-11-21 08:15 -05:00 · tarea 3 · verde · caducada · `go vet ./... && go test ./... && node --test tests/gomemory-plan.test.mjs tests/gomemory-banner.test.mjs tests/antigravity-prompt-guard/guard.test.mjs` (salida 0)
      ✔ passes through blank system text (0.069083ms)
      ✔ passes through malformed conversation turns (0.048834ms)
      ✔ does not discard non-text system metadata (0.041042ms)
      ✔ registers a working Pi hook scoped to Antigravity (0.141ms)
      ✔ planHeaderLine: ON en ámbar, OFF atenuado, sin escapes con NO_COLOR, centrada (0.935208ms)
      ✔ el header muestra el modo actual y se repinta al cambiarlo (1.609625ms)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (3.547041ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (0.924291ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.216208ms)
      ✔ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (1.012667ms)
      ✔ planShortcutKeys: Alt+P y Ctrl+Alt+P; en Windows y WSL solo Ctrl+Alt+P (0.130417ms)
      ✔ Alt+P alterna el modo plan, bloquea y publica el estado para el header (0.257041ms)
      ℹ tests 30
      ℹ suites 0
      ℹ pass 30
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 256.385917
- 2026-11-21 08:17 -05:00 · tarea 4 · manual · caducada
      Pi 1.6.0 real en tmux (140x50) con las extensiones copiadas a ~/.pi/agent/extensions: al iniciar el header muestra '○ plan off · Alt+P'; Alt+P (M-p) -> '◆ MODO PLAN ON · Alt+P' con escape 1;38;2;244;198;107 (ámbar) y aviso 'edit, write y comandos que modifican están bloqueados'; Alt+P otra vez -> 'plan off' + 'Modo plan desactivado'; Ctrl+Alt+P (C-M-p) -> ON y '/plan status' responde 'Modo plan activo' (mismo estado que usa el bloqueo de tool_call); Ctrl+Alt+P -> off. Limitación observada: la etiqueta PLAN del pie la recolorea el footer de pi-cc-extensions (colorUsageChip), así que allí no sale ámbar; el header sí.
- 2026-11-21 08:17 -05:00 · tarea 1 · verde · caducada · `node --test tests/gomemory-plan.test.mjs` (salida 0)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (3.531208ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (0.831667ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.206375ms)
      ✔ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (0.973417ms)
      ✔ planShortcutKeys: Alt+P y Ctrl+Alt+P; en Windows y WSL solo Ctrl+Alt+P (0.131417ms)
      ✔ Alt+P alterna el modo plan, bloquea y publica el estado para el header (0.2495ms)
      ℹ tests 6
      ℹ suites 0
      ℹ pass 6
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 249.944
- 2026-11-21 08:17 -05:00 · tarea 2 · verde · caducada · `node --test tests/gomemory-banner.test.mjs` (salida 0)
      ✔ planHeaderLine: ON en ámbar, OFF atenuado, sin escapes con NO_COLOR, centrada (0.747708ms)
      ✔ el header muestra el modo actual y se repinta al cambiarlo (1.650708ms)
      ℹ tests 2
      ℹ suites 0
      ℹ pass 2
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 250.895958
- 2026-11-21 08:22 -05:00 · tarea 3 · verde · vigente · `go vet ./... && go test ./... && node --test tests/gomemory-plan.test.mjs tests/gomemory-banner.test.mjs tests/antigravity-prompt-guard/guard.test.mjs` (salida 0)
      ✔ passes through blank system text (0.062375ms)
      ✔ passes through malformed conversation turns (0.047042ms)
      ✔ does not discard non-text system metadata (0.046625ms)
      ✔ registers a working Pi hook scoped to Antigravity (0.126709ms)
      ✔ planHeaderLine: ON en ámbar, OFF atenuado, sin escapes con NO_COLOR, centrada (0.720708ms)
      ✔ el header muestra el modo actual y se repinta al cambiarlo (1.568334ms)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (3.456917ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (0.785917ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.181792ms)
      ✔ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (0.99175ms)
      ✔ planShortcutKeys: Alt+P y Ctrl+Alt+P; en Windows y WSL solo Ctrl+Alt+P (0.126542ms)
      ✔ Alt+P alterna el modo plan, bloquea y publica el estado para el header (0.24425ms)
      ℹ tests 30
      ℹ suites 0
      ℹ pass 30
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 249.120459
- 2026-11-21 08:22 -05:00 · tarea 1 · verde · vigente · `node --test tests/gomemory-plan.test.mjs` (salida 0)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (4.065292ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (0.947041ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.281125ms)
      ✔ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (1.028625ms)
      ✔ planShortcutKeys: Alt+P y Ctrl+Alt+P; en Windows y WSL solo Ctrl+Alt+P (0.14575ms)
      ✔ Alt+P alterna el modo plan, bloquea y publica el estado para el header (0.269208ms)
      ℹ tests 6
      ℹ suites 0
      ℹ pass 6
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 268.105625
- 2026-11-21 08:22 -05:00 · tarea 2 · verde · vigente · `node --test tests/gomemory-banner.test.mjs` (salida 0)
      ✔ planHeaderLine: ON en ámbar, OFF atenuado, sin escapes con NO_COLOR, centrada (0.787959ms)
      ✔ el header muestra el modo actual y se repinta al cambiarlo (1.5595ms)
      ℹ tests 2
      ℹ suites 0
      ℹ pass 2
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 269.229541
- 2026-11-21 08:23 -05:00 · tarea 4 · manual · vigente
      Reobservado tras HCD-068 en Pi 1.6.0 real (tmux 140x50), extensiones instaladas idénticas al repo (cmp): inicio '○ plan off · Alt+P'; Alt+P -> '◆ MODO PLAN ON' con escape 38;2;244;198;107 (ámbar); Alt+P -> '○ plan off'; Ctrl+Alt+P -> '◆ MODO PLAN ON · Alt+P'. El bloqueo de edit con el modo activo lo cubre tests/gomemory-plan.test.mjs (mismo estado method que usa tool_call). Limitación: el chip PLAN del pie lo recolorea pi-cc-extensions.

## Siguiente
Cerrada con aprobación sobre la revisión cf6131a27d42.

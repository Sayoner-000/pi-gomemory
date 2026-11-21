# Glifos de /tasks legibles: barra ━─ y tarea caducada ↻

ID: HCD-068
Tipo: evolution
Carril: fast
Motivo: La barra ▰▱ se ve entrecortada (fuente de respaldo) y ◍ no comunica que la tarea caducó; la persona eligió barra A (━ lleno acento, ─ vacío atenuado) y ↻ ámbar para caducada

## Intención
Que la cabecera y la lista de /tasks se lean limpias en cualquier fuente monoespaciada: la barra de progreso pasa de `▰▱` (paralelogramos que la terminal dibuja con fuente de respaldo y salen entrecortados) a una línea continua `━━━━──────` (tramo lleno grueso, vacío fino) y la tarea caducada pasa de `◍` a `↻` en ámbar, que se lee como «volver a verificar».


## Alcance
Incluido: `progressBar` y `taskVisual` en `extensions/gomemory-tasks.ts` (y sus comentarios de leyenda); expectativas de glifos en `pkg/installer/hcd057_test.go` (`TestHCD057Glyphs`, `TestHCD057Progress`) actualizadas primero para observar el rojo; comprobación en Pi real.
Excluido: los demás glifos (○ pendiente, ●/○ en curso, ● verificada, ✗ fallida, spinner de llamadas); la barra de cuota de `pkg/installer/usage.go`; colorear por separado el tramo lleno y el vacío (la línea conserva un solo tono; la diferencia la marca el grosor); publicar release.

## ¿Qué pasa si no lo hacemos?
La barra sigue viéndose entrecortada (captura de la persona del 2026-11-21 8:16) y la tarea caducada sigue con un círculo rayado que no explica que hay que volver a verificar.

## Alternativa más barata
Cambiar la fuente de la terminal por una que traiga `▰▱◍`. No es razonable exigirlo a cada persona ni a Windows Terminal (Cascadia Mono); `━─` y `↻` están en todas las fuentes monoespaciadas habituales. Decisiones previas respetadas: HCD-057 Q5 (familia de círculos y dos espacios entre barra y números), HCD-060 (parpadeo ●/○), HCD-020 Q3 (glifos de texto, sin emojis: por eso no ⚠).


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Cambiar en `extensions/gomemory-tasks.ts` la barra de `progressBar` a `━` (lleno) y `─` (vacío) con 10 celdas y dos espacios antes de los números → `go test ./pkg/installer -run TestHCD057Progress -count=1`
- [x] 2. Cambiar en `extensions/gomemory-tasks.ts` el glifo de tarea caducada de `taskVisual` de `◍` a `↻` con tono warning → `go test ./pkg/installer -run TestHCD057Glyphs -count=1`
- [x] 3. Regresión completa (sin rojo previo) → `go vet ./... && go test ./... && node --test tests/gomemory-plan.test.mjs tests/gomemory-banner.test.mjs`
- [x] 4. Comprobar en Pi real que `/tasks` muestra la barra `━─` continua y `↻` en la tarea caducada (`extensions/gomemory-tasks.ts` instalada) (sin rojo previo) → observación directa en Pi real

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-21 08:19 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD057Progress -count=1` (salida 1)
      --- FAIL: TestHCD057Progress (0.25s)
          hcd057_test.go:127: TestHCD057Progress: FAIL esperaba «━━────────  1/5 · 20 %» en «HCD-057 · feature · paso implement · ▰▰▱▱▱▱▱▱▱▱  1/5 · 20 %» | esperaba «──────────  0/3 · 0 %» en «HCD-057 · feature · paso implement · ▱▱▱▱▱▱▱▱▱▱  0/3 · 0 %» | esperaba «━━━━━━━━━━  2/2 · 100 %» en «HCD-057 · feature · paso implement · ▰▰▰▰▰▰▰▰▰▰  2/2 · 100 %» | esperaba «━━━━──────  5/13 · 38 %» en «HCD-057 · feature · paso implement · ▰▰▰▰▱▱▱▱▱▱  5/13 · 38 %»
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.640s
      FAIL
- 2026-11-21 08:19 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD057Glyphs -count=1` (salida 1)
      --- FAIL: TestHCD057Glyphs (1.64s)
          hcd057_test.go:95: TestHCD057Glyphs: FAIL tarea 3 sin glifo ↻
      FAIL
      FAIL	pi-gomemory/pkg/installer	1.861s
      FAIL
- 2026-11-21 08:19 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD057Progress -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.448s
- 2026-11-21 08:20 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD057Glyphs -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.836s
- 2026-11-21 08:20 -05:00 · tarea 3 · verde · vigente · `go vet ./... && go test ./... && node --test tests/gomemory-plan.test.mjs tests/gomemory-banner.test.mjs` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	49.412s
      ✔ planHeaderLine: ON en ámbar, OFF atenuado, sin escapes con NO_COLOR, centrada (0.810458ms)
      ✔ el header muestra el modo actual y se repinta al cambiarlo (1.573417ms)
      ✔ on → off → siguiente turno mantiene el estado, el prompt y el bloqueo sincronizados (3.824875ms)
      ✔ run solo desactiva; /plan alterna y /plan <petición> sigue enviando texto (1.0155ms)
      ✔ off no deja que una carga tardía vuelva a activar el modo (0.205209ms)
      ✔ modo plan permite lecturas de codegraph/codebase-memory e index_repository externo, y bloquea mutaciones HCD o persistentes (1.000917ms)
      ✔ planShortcutKeys: Alt+P y Ctrl+Alt+P; en Windows y WSL solo Ctrl+Alt+P (0.128458ms)
      ✔ Alt+P alterna el modo plan, bloquea y publica el estado para el header (0.25525ms)
      ℹ tests 8
      ℹ suites 0
      ℹ pass 8
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 259.2335
- 2026-11-21 08:21 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD057Glyphs -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.955s

## Siguiente
Cerrada con aprobación sobre la revisión 7eb889189109.

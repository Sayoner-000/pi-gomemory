# Clic de minimizar/expandir y paneles /sidebar y /tdd sin solaparse

ID: HCD-019
Tipo: fix
Carril: fast
Motivo: fix con reproducción clara y causa raíz confirmada en pi-tui, sin decisiones abiertas

## Intención
Un clic en la cabecera alterna una sola vez cada panel (/sidebar y /tdd), los paneles se apilan en la columna derecha sin solaparse y el modo minimizado ocupa una sola fila sin tapar el transcript

## Reproducción
Pi 1.5.2 en modo `fullscreen` (por defecto). Pasos: `/sidebar` y luego `/tdd`.
1. Los dos paneles se dibujan en `top-right` con `margin.top = 1` y quedan solapados.
2. Un clic sobre cualquiera de los dos no cambia nada. Causa: pi-tui (módulo alt-screen del paquete de Pi, `handleMouseEvent`) entrega un clic como `press`, `release` y `click` sintético. Los dos `handleMouse` alternan tanto en `press` como en `click`, así que el panel se minimiza y se restaura en el mismo gesto.
3. Con `/sidebar min`, el panel ocupa 26 filas, porque el render minimizado se rellena con espacios. Esas filas tapan el transcript y capturan los clics de toda esa zona.

Reproducción automatizada: `go test ./pkg/installer -run HCD019`. El test simula `ctx.ui.custom` y la secuencia `press → release → click` en la cabecera, y espera un único cambio de estado, un minimizado de 1 fila y un TDD desplazado bajo el sidebar.

## Alcance
Incluido: `handleMouse` de `extensions/gomemory-sidebar.ts` y `extensions/gomemory-tdd.ts`, que solo alternan con `click` izquierdo en la fila de cabecera (`press` y `release` se marcan como manejados sin alternar); render minimizado de 1 fila, sin relleno, con redibujado forzado al alternar; apilado en la columna derecha, con el sidebar publicando su altura en `globalThis[Symbol.for("pi-gomemory.dock")]` y el TDD usando `offsetY`/`maxHeight` dinámicos (se quita `margin.top: overlayMinimized ? 3 : 1`); indicadores de cabecera `[−]` y `[+]`; test nuevo `pkg/installer/hcd019_panels_test.go`.
Excluido: cambiar `/min` (hoy solo alterna el sidebar; queda como hallazgo para otra unidad); el modo `regular`, que no tiene ratón por diseño de Pi (se conserva el camino por teclado `/sidebar min` y `/tdd min`); el contenido, las métricas y las cuotas de los paneles.

## ¿Qué pasa si no lo hacemos?
El clic de minimizar y expandir anunciado en v1.9.0 no funciona. Con los dos paneles abiertos, uno tapa al otro, y el minimizado sigue ocupando 26 filas encima de la conversación, que es justo lo que pretendía evitar.

## Alternativa más barata
Quitar `handleMouse` y dejar solo `/sidebar min` y `/tdd min`, y anclar el TDD en otra esquina. Arregla el solape, pero renuncia al clic que pidió la persona, y el TDD en `top-left` o `bottom-*` tapa el inicio del transcript o el editor. Se descarta: el arreglo real del clic son unas pocas líneas y apilar los paneles evita esos problemas.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla en `pkg/installer/hcd019_panels_test.go` (toca `extensions/gomemory-sidebar.ts` y `extensions/gomemory-tdd.ts`) → `go test ./pkg/installer -run HCD019 -count=1`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 18:57 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run HCD019 -count=1` (salida 1)
      --- FAIL: TestHCD019SidebarClickToggle (0.14s)
          hcd019_panels_test.go:85: TestHCD019SidebarClickToggle: FAIL dock.sidebarHeight debe ser 18, es undefined | tras un clic en la cabecera debe quedar minimizado en 1 fila, tiene 18 | minimizado sin indicador [+] | dock no refleja la altura minimizada | un clic fuera de la cabecera no debe alternar | sidebar visible debe publicarse en el dock
      --- FAIL: TestHCD019TDDClickToggleAndDock (0.13s)
          hcd019_panels_test.go:122: TestHCD019TDDClickToggleAndDock: FAIL tras un clic en la cabecera debe quedar minimizado en 1 fila, tiene 27 | con sidebar de 10 filas offsetY debe ser 11, es undefined | maxHeight debe dejar fuera las filas del sidebar, es undefined | sin hueco bajo el sidebar el TDD debe ocultarse, no solaparse
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.444s
      FAIL
- 2026-11-11 19:00 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run HCD019 -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.567s
- 2026-11-11 19:01 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run HCD019 -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.425s

## Siguiente
Cerrada con aprobación sobre la revisión f5b6b2654028.

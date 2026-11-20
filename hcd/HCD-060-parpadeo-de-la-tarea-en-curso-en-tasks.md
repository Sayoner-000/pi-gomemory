# Parpadeo de la tarea en curso en /tasks

ID: HCD-060
Tipo: evolution
Carril: fast
Motivo: Ajuste visual acotado a extensions/gomemory-tasks.ts pedido por la persona; sin decisiones abiertas

## Intención
La tarea en curso de /tasks deja de girar ◐◓◑◒ y parpadea alternando ● y ○ cada 500 ms en color accent; la verificada sigue siendo ● fijo en success.

La persona lo pidió tras ver `◑` en Pi: «cambia esto por un parpadeo ●». HCD-057 (Q5) eligió el círculo que gira; este ajuste cambia solo esa animación y conserva el resto del esquema (`○` pendiente, `●` verificada, `◍` caducada, `✗` fallida).

## Alcance
Incluido: en `extensions/gomemory-tasks.ts`, `TASK_FRAMES` pasa a `●` y `○` con 500 ms por fotograma, y el temporizador del panel repinta a ese ritmo cuando solo hay una tarea activa; actualizar la tabla de glifos de `docs/usage.md` y la nota de `docs/architecture.md`; adaptar la prueba de HCD-057 que fija los fotogramas.
Excluido: el spinner de las llamadas en curso (sigue siendo el de /tdd); publicar una versión nueva.

## ¿Qué pasa si no lo hacemos?
La tarea en curso sigue mostrándose con un círculo que gira, que la persona no reconoce como «en progreso».

## Alternativa más barata
Dejar `●` fijo en color accent para la tarea en curso. Es más simple, pero no parpadea, que es lo que pidió la persona; se descarta.

## Normas aplicables
- Constitución §10 Pruebas: rojo antes del cambio.
- Constitución §14 Frontend: el estado no depende solo del color; el parpadeo distingue la tarea en curso del `●` fijo de una verificada.

## Aclaraciones

## Checklist
- [x] 1. Hacer parpadear la tarea en curso alternando ● y ○ cada 500 ms y repintar el panel a ese ritmo → `go test ./pkg/installer -run '^TestHCD060Blink$' -count=1`
- [x] 2. Mantener en verde las pruebas existentes del panel /tasks → `go test ./pkg/installer -run '^(TestHCD057.*|TestHCD054.*)$' -count=1`
- [x] 3. Comprobar en Pi real con la extensión instalada (SHA256 igual al repositorio) que la tarea en curso parpadea ●/○ → observación manual registrada
- [x] 4. Ejecutar la regresión completa → `go vet ./... && go test -race ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-20 04:37 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD060Blink$' -count=1` (salida 1)
      --- FAIL: TestHCD060Blink (1.45s)
          hcd060_test.go:46: TestHCD060Blink: FAIL TASK_FRAMES debe ser ●○: ◐,◓,◑,◒ | de 0 a 499 ms debe verse ●: ◐◓ | de 500 a 999 ms debe verse ○: ◑◒ | a los 1000 ms vuelve ●: ◐ | en 1,1 s debe repintar 2 veces (cada 500 ms): 4
      FAIL
      FAIL	pi-gomemory/pkg/installer	1.855s
      FAIL
- 2026-11-20 04:37 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD060Blink$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.558s
- 2026-11-20 04:37 -05:00 · tarea 2 · falló · caducada · `go test ./pkg/installer -run '^(TestHCD057.*|TestHCD054.*)$' -count=1` (salida 1)
      --- FAIL: TestHCD057Glyphs (1.22s)
          hcd057_test.go:94: TestHCD057Glyphs: FAIL TASK_FRAMES debe ser ◐◓◑◒: ●,○ | el giro de la tarea activa debe avanzar con el reloj | el giro de la tarea debe avanzar con el reloj | con una tarea activa el widget debe seguir animando sin llamadas: 1
      FAIL
      FAIL	pi-gomemory/pkg/installer	1.490s
      FAIL
- 2026-11-20 04:37 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^(TestHCD057.*|TestHCD054.*)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.014s
- 2026-11-20 04:37 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD060Blink$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.541s
- 2026-11-20 04:38 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD060Blink$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.570s
- 2026-11-20 04:38 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^(TestHCD057.*|TestHCD054.*)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.036s
- 2026-11-20 04:38 -05:00 · tarea 3 · manual · vigente
      Pi 1.5.2 real vía Herdr (pestaña w3X:t, 176 columnas) en el repositorio con HCD-060 activa; gomemory-tasks.ts instalado con SHA256 db8a5ee0 igual al repo. Seis lecturas de la fila de la tarea activa (la 1) cada 0,3 s: ○ ○ ● ○ ○ ●, es decir, alterna ●/○ a ~500 ms; la secuencia ANSI muestra el glifo en color accent (38;2;34;227;247). La cabecera y el resto de glifos (○ pendientes) sin cambios.
- 2026-11-20 04:38 -05:00 · tarea 4 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	44.622s

## Siguiente
Cerrada con aprobación sobre la revisión 91b76c6c5b29.

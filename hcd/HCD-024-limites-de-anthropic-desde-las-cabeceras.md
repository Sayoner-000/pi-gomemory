# Límites de Anthropic desde las cabeceras de cada respuesta, sin sondeo (HTTP 429)

ID: HCD-024
Tipo: fix
Carril: fast
Motivo: fallo observado en Pi real (HTTP 429) con causa raíz y solución verificadas contra la API; sin decisiones abiertas

## Intención
El sidebar obtiene la sesión y la semana del plan de Anthropic de las cabeceras anthropic-ratelimit-unified-* que llegan en cada respuesta del modelo (after_provider_response), como claude-token-usage; deja de sondear /api/oauth/usage cada 30 s y en cada turno, y un HTTP 429 deja de presentarse como sesión caducada

## Reproducción
En Pi v1.11.0 con Anthropic, el bloque «▼ Anthropic · plan» muestra «uso de Anthropic: HTTP 429 (¿sesión caducada? abre pi para renovarla)». Causa raíz: HCD-023 consulta `/api/oauth/usage` al abrir, cada 30 s (heartbeat) y en cada `turn_end`, `agent_end` y `model_select`, siempre forzado; Anthropic limita esa frecuencia con un 429. Además, el texto atribuye el 429 a una sesión caducada, lo cual es falso.
Referencia: claude-token-usage (Sayoner-000) no consulta nada; lee los límites que llegan en cada respuesta de la API. Verificado en Pi 1.5.2: `after_provider_response` entrega, en una respuesta real de Anthropic (HTTP 200), `anthropic-ratelimit-unified-5h-utilization: 0.72`, `-5h-reset: 1794458400` (epoch en segundos), `-7d-utilization: 0.1`, `-7d-reset: 1795039200` y los estados `-5h-status` y `-7d-status`.
Comprobación automatizada: `go test ./pkg/installer -run HCD024 -count=1`.

## Alcance
Incluido: el sidebar se suscribe a `after_provider_response` y, si la respuesta trae `anthropic-ratelimit-unified-5h-*` o `-7d-*`, guarda la lectura (porcentaje y reinicio) y redibuja; el tiempo hasta el reinicio se calcula al pintar, así que la cuenta atrás no se congela. Se elimina el sondeo: `anthropic-usage` se consulta como máximo una vez al abrir el panel, solo si aún no llegó ninguna respuesta, y tras un error no se reintenta antes de 5 minutos. Un HTTP 429 de `anthropic-usage` se describe como límite de frecuencia («se actualizará con la próxima respuesta del modelo»), no como sesión caducada. Tests nuevos en `pkg/installer/hcd024_test.go`.
Excluido: la proyección de agotamiento de claude-token-usage (otra unidad si se quiere); los límites por modelo, que no llegan en las cabeceras (siguen disponibles por `anthropic-usage`); persistir la lectura entre sesiones de Pi.

## ¿Qué pasa si no lo hacemos?
El bloque de Anthropic queda inservible: en cuanto Anthropic limita la frecuencia, muestra un 429 con un diagnóstico falso en lugar de los porcentajes, y las consultas repetidas gastan cuota de la API de uso.

## Alternativa más barata
Bajar la frecuencia del sondeo (por ejemplo, a 5 min). Reduce los 429, pero no los evita, los datos quedan desfasados entre consultas y sigue dependiendo de un endpoint no documentado, cuando el dato ya llega gratis en cada respuesta.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla en `pkg/installer/hcd024_test.go` → `go test ./pkg/installer -run HCD024 -count=1`
- [x] 2. Leer sesión y semana de las cabeceras `anthropic-ratelimit-unified-*` de `after_provider_response` y mostrarlas en el sidebar con el reinicio calculado al pintar → `go test ./pkg/installer -run TestHCD024PlanFromHeaders -count=1`
- [x] 3. Eliminar el sondeo: como mucho una consulta a `anthropic-usage` al abrir y solo sin lectura previa; espera de 5 min tras un error → `go test ./pkg/installer -run TestHCD024NoPolling -count=1`
- [x] 4. Describir un HTTP 429 de `anthropic-usage` como límite de frecuencia, no como sesión caducada → `go test ./pkg/installer -run TestHCD024RateLimitMessage -count=1`
- [x] 5. Mantener en verde toda la suite y `go vet` → `go test ./... -count=1 && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 19:59 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run HCD024 -count=1` (salida 1)
      --- FAIL: TestHCD024PlanFromHeaders (0.15s)
          hcd024_test.go:80: TestHCD024PlanFromHeaders: FAIL el sidebar no escucha after_provider_response
      --- FAIL: TestHCD024NoPolling (0.23s)
          hcd024_test.go:108: TestHCD024NoPolling: FAIL abrir y 15 eventos de turno no deben sondear: 16 consultas a anthropic-usage | sin lectura, debe indicar que se actualiza con la próxima respuesta | reabrir tras un error no debe reintentar antes de 5 min: 17
      --- FAIL: TestHCD024RateLimitMessage (0.00s)
          hcd024_test.go:124: un 429 no es una sesión caducada: uso de Anthropic: HTTP 429 (¿sesión caducada? abre pi para renovarla)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.741s
      FAIL
- 2026-11-11 19:59 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD024PlanFromHeaders -count=1` (salida 1)
      --- FAIL: TestHCD024PlanFromHeaders (0.13s)
          hcd024_test.go:80: TestHCD024PlanFromHeaders: FAIL el sidebar no escucha after_provider_response
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.274s
      FAIL
- 2026-11-11 19:59 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD024NoPolling -count=1` (salida 1)
      --- FAIL: TestHCD024NoPolling (0.23s)
          hcd024_test.go:108: TestHCD024NoPolling: FAIL abrir y 15 eventos de turno no deben sondear: 16 consultas a anthropic-usage | sin lectura, debe indicar que se actualiza con la próxima respuesta | reabrir tras un error no debe reintentar antes de 5 min: 17
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.385s
      FAIL
- 2026-11-11 19:59 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD024RateLimitMessage -count=1` (salida 1)
      --- FAIL: TestHCD024RateLimitMessage (0.00s)
          hcd024_test.go:124: un 429 no es una sesión caducada: uso de Anthropic: HTTP 429 (¿sesión caducada? abre pi para renovarla)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.151s
      FAIL
- 2026-11-11 20:01 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run HCD024 -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.592s
- 2026-11-11 20:01 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD024PlanFromHeaders -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.328s
- 2026-11-11 20:01 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD024NoPolling -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.368s
- 2026-11-11 20:01 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD024RateLimitMessage -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.151s
- 2026-11-11 20:01 -05:00 · tarea 5 · verde · caducada · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	15.800s
- 2026-11-11 20:23 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run HCD024 -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.742s
- 2026-11-11 20:23 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD024PlanFromHeaders -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.339s
- 2026-11-11 20:23 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD024NoPolling -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.419s
- 2026-11-11 20:23 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestHCD024RateLimitMessage -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.184s
- 2026-11-11 20:23 -05:00 · tarea 5 · verde · vigente · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	17.945s

## Siguiente
Cerrada con aprobación sobre la revisión dff88446d134.

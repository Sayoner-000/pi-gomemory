# Refresco por turnos y heartbeat de proveedores en sidebar

ID: HCD-017
Tipo: fix
Carril: fast
Motivo: Las cuotas y porcentajes del sidebar no se actualizan al avanzar turnos ni por timer debido a TTL bloqueante

## Intención
Garantizar que el panel lateral gomemory-sidebar refresque las cuotas y métricas de los proveedores conectados (Antigravity/Agy, Claude Code, OpenAI/Codex, etc.) en tiempo real conforme se avanza en la sesión (por consumo de turnos) y mediante un heartbeat periódico confiable que mantenga actualizados los porcentajes y tiempos de reinicio sin ser bloqueado por TTL.

## Reproducción
1. Con el sidebar visible y proveedor Antigravity/Agy activo, realizar un turno de conversación (prompt + respuesta).
2. Observar que no existe handler para `turn_end`, y en `agent_end` se invoca `refreshQuotaIfNeeded(ctx)` con `force = false`, quedando descartado por el TTL de 60s si el turno dura menos de 1 minuto.
3. Observar que `autoRefreshTimer` ejecuta cada 30s llamando con `force = false`, siendo rechazado sistemáticamente por el TTL de 60s en los ciclos intermedios.

## Alcance
Incluido: suscripción del sidebar a turn_end y actualización forzada tras cada turno de consumo (force = true en fin de turno); sincronización del heartbeat periódico del timer para que refresque cuotas y porcentajes sin descarte por TTL; detección de proveedores activos vía ctx.model.provider y variables de entorno PI_PROVIDER/PI_MODEL para cubrir Antigravity/Agy, Claude Code y proveedores genéricos; y tests unitarios de regresión en Go.
Excluido: cambios en endpoints remotos de Google Cloud Code o CLI externa de Claude.

## ¿Qué pasa si no lo hacemos?
El usuario ve porcentajes y cuotas estáticas y desactualizadas en el sidebar durante toda la sesión interactiva a menos que cierre y reabra el panel manualmente.

## Alternativa más barata
Forzar al usuario a ejecutar `/sidebar` para cerrar y volver a abrir el panel cada vez que desee conocer el consumo real de su cuota.

## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->

## Checklist
- [x] 1. Suscripción a turn_end y refresco forzado tras consumo de turnos → go test -v ./pkg/installer -run TestHCD017SidebarTurnEndRefresh
- [x] 2. Heartbeat periódico del timer refresca cuotas y porcentajes sin bloqueo por TTL → go test -v ./pkg/installer -run TestHCD017SidebarHeartbeatTimer
- [x] 3. Detección robusta de proveedor activo con fallback a entorno PI_PROVIDER → go test -v ./pkg/installer -run TestHCD017SidebarProviderDetection

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 13:17 -05:00 · tarea 1 · rojo observado · caducada · `go test -v ./pkg/installer -run TestHCD017SidebarTurnEndRefresh` (salida 1)
      === RUN   TestHCD017SidebarTurnEndRefresh
          hcd017_sidebar_test.go:34: TestHCD017SidebarTurnEndRefresh fallo: FAIL no se registro handler para turn_end
      --- FAIL: TestHCD017SidebarTurnEndRefresh (0.06s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.066s
      FAIL
- 2026-11-11 13:18 -05:00 · tarea 1 · verde · vigente · `go test -v ./pkg/installer -run TestHCD017SidebarTurnEndRefresh` (salida 0)
      === RUN   TestHCD017SidebarTurnEndRefresh
      --- PASS: TestHCD017SidebarTurnEndRefresh (0.06s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.061s
- 2026-11-11 13:18 -05:00 · tarea 2 · verde · vigente · `go test -v ./pkg/installer -run TestHCD017SidebarHeartbeatTimer` (salida 0)
      === RUN   TestHCD017SidebarHeartbeatTimer
      --- PASS: TestHCD017SidebarHeartbeatTimer (0.08s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.085s
- 2026-11-11 13:19 -05:00 · tarea 3 · verde · vigente · `go test -v ./pkg/installer -run TestHCD017SidebarProviderDetection` (salida 0)
      === RUN   TestHCD017SidebarProviderDetection
      --- PASS: TestHCD017SidebarProviderDetection (0.07s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.078s

## Siguiente
Cerrada con aprobación sobre la revisión 899983201863.

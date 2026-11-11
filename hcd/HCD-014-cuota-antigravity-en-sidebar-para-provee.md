# Cuota Antigravity en sidebar para proveedores antigravity y agy

ID: HCD-014
Tipo: feature
Carril: fast
Motivo: Permitir visualizar cuotas de Google AI Pro directo en el sidebar cuando se usa antigravity o agy

## Intención
Integrar en el panel lateral (`/sidebar`) de pi-gomemory la visualización de cuotas reales de Google AI Pro (Gemini Models, Claude and GPT models y Gemini 4 Argon) consumiendo el endpoint oficial de Antigravity cuando el modelo o proveedor activo sea antigravity o agy.

## Alcance
Incluido: endpoints actualizados y User-Agent en usage.go alineados con la CLI de Antigravity; soporte para flag --json en pi-gomemory-setup usage y función FetchQuotaSummary; tarjeta compacta Google AI Pro Quota en gomemory-sidebar.ts con barras de progreso, porcentajes y reset time; detección de proveedores antigravity y agy con TTL de 120s; tests unitarios en Go y Node.
Excluido: cambios en backend de Google o soporte sin credenciales; visualización de cuotas para modelos no Antigravity.

## ¿Qué pasa si no lo hacemos?
El usuario tiene que ejecutar comandos manuales (`/antigravity.usage` o salir a terminal) para conocer el estado de sus límites semanales y de 5 horas mientras trabaja en Pi.

## Alternativa más barata
Ejecutar manualmente `pi-gomemory-setup usage` por consola fuera de la interfaz de Pi.

## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->

## Checklist
- [x] 1. Endpoints actualizados y soporte JSON de cuotas en installer → go test -v ./pkg/installer -run TestFetchAndFormatUsageNeverFabricates
- [x] 2. Renderizado de tarjeta de cuotas en sidebar con ancho 36 y sin falsos positivos → go test -v ./pkg/installer -run TestHCD014SidebarQuotaRendering

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 11:25 -05:00 · tarea 1 · verde · vigente · `go test -v ./pkg/installer -run TestFetchAndFormatUsageNeverFabricates` (salida 0)
      === RUN   TestFetchAndFormatUsageNeverFabricates
      --- PASS: TestFetchAndFormatUsageNeverFabricates (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.008s
- 2026-11-11 11:25 -05:00 · tarea 2 · verde · vigente · `go test -v ./pkg/installer -run TestHCD014SidebarQuotaRendering` (salida 0)
      === RUN   TestHCD014SidebarQuotaRendering
      --- PASS: TestHCD014SidebarQuotaRendering (0.07s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)

## Siguiente
Cerrada con aprobación sobre la revisión 61ca901101f1.

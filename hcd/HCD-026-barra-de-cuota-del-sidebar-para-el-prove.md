# Barra de cuota del sidebar para el proveedor Codex

ID: HCD-026
Tipo: feature
Carril: fast
Motivo: Feature acotada a una sección del sidebar, sin decisiones interdependientes ni riesgo de datos o contratos

## Intención
Que el sidebar muestre, cuando el proveedor activo es Codex (openai-codex), barras de progreso de uso del plan (ventana de 5 h y semanal, % y reinicio) con el mismo formato visual que la sección 'Google AI Pro Quota', en lugar de la línea genérica 'Estado: Conectado (live)'


## Alcance
Incluido: comando `pi-gomemory-setup codex-usage` (siempre JSON) en `pkg/installer/codex_usage.go`, que lee el token OAuth y el accountId de Pi (`~/.pi/agent/auth.json`, clave `openai-codex`) y consulta el endpoint `https://chatgpt.com/backend-api/wham/usage`; devuelve los límites de las ventanas primaria (5 horas) y secundaria (semanal) con porcentaje usado, reset_at y tiempo hasta el reinicio (`resetIn`). Sin sesión OAuth, con token caducado, HTTP distinto de 200 o respuesta no válida devuelve error explícito y nunca cifras inventadas. El token no se refresca en segundo plano. En `extensions/gomemory-sidebar.ts`, cuando el proveedor activo es Codex (`detectProvider(ctx) === "codex"`), se añade la sección visual «▼ Codex Quota» mostrando barras de progreso formateadas de forma idéntica a Google AI Pro, porcentaje y cuenta regresiva de reinicio, sustituyendo la línea estática «Estado: Conectado (live)». Pruebas en Go en `pkg/installer/hcd026_test.go` y suite general verde.
Excluido: renovar o refrescar el token OAuth de OpenAI; otros endpoints de ChatGPT o Codex ajenos a cuotas; modificar secciones de Antigravity o Claude.

## ¿Qué pasa si no lo hacemos?
Los usuarios que utilizan el modelo o proveedor Codex en Pi no tienen visibilidad de su consumo de cuota ni del tiempo de reinicio de sus ventanas de uso (5 horas y semanal). Al agotarse el cupo se enfrentan a errores HTTP 429 inesperados sin previo aviso en la interfaz del sidebar, degradando la experiencia respecto a los usuarios de Google AI Pro o Claude.

## Alternativa más barata
Mantener únicamente la línea estática actual «Estado: Conectado (live)». No tiene coste de implementación pero resulta inútil para la supervisión de recursos y no satisface la necesidad de observar el progreso de la cuota en el panel lateral.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Leer en Go los límites de uso de OpenAI Codex (ventanas primaria y secundaria) desde auth.json y la API wham/usage y devolverlos como JSON limpio → `go test ./pkg/installer -run TestHCD026FetchCodexUsage -count=1`
- [x] 2. Devolver un error explícito sin sesión OAuth, con token caducado o HTTP distinto de 200 en Codex → `go test ./pkg/installer -run TestHCD026CodexUsageErrors -count=1`
- [x] 3. Registrar el comando `codex-usage` en `main.go` y en su ayuda CLI → `go test ./pkg/installer -run TestHCD026CommandRegistered -count=1`
- [x] 4. Mostrar en el sidebar la sección de cuota para Codex con barras de progreso, porcentajes y tiempos de reinicio → `go test ./pkg/installer -run TestHCD026SidebarCodexQuota -count=1`
- [x] 5. Mantener en verde toda la suite del proyecto y `go vet` → `go test ./... -count=1 && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-12 05:42 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD026FetchCodexUsage -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.193s
- 2026-11-12 05:42 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD026CodexUsageErrors -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.177s
- 2026-11-12 05:42 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD026CommandRegistered -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.186s
- 2026-11-12 05:43 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestHCD026SidebarCodexQuota -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.177s
- 2026-11-12 05:44 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestHCD026FetchCodexUsage -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.167s
- 2026-11-12 05:44 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD026CodexUsageErrors -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.174s
- 2026-11-12 05:44 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD026CommandRegistered -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.180s
- 2026-11-12 05:44 -05:00 · tarea 5 · verde · vigente · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	15.375s

## Siguiente
Cerrada con aprobación sobre la revisión 72f6def1e6f0.

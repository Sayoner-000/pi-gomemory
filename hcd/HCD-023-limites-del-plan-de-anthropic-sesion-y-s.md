# Límites del plan de Anthropic (sesión y semana) en el sidebar

ID: HCD-023
Tipo: feature
Carril: fast
Motivo: comportamiento nuevo acotado con la fuente de datos verificada contra la API real; decisión de la persona ya tomada (límites del plan)

## Intención
Cuando Pi está conectado a Anthropic con OAuth, el sidebar muestra el porcentaje usado de la sesión de 5 h y de la semana (y por modelo si Anthropic lo envía), con su hora de reinicio, leídos de la API real; si no se puede consultar, muestra el error y nunca cifras inventadas


## Evidencia previa (2026-11-11)
- Pi guarda una sesión OAuth de Anthropic en `~/.pi/agent/auth.json` (`anthropic`: `type: "oauth"`, `access`, `refresh`, `expires`).
- `GET https://api.anthropic.com/api/oauth/usage` con `Authorization: Bearer <access>` y `anthropic-beta: oauth-2025-04-20` respondió HTTP 200 con datos reales: `five_hour` (`utilization` 52.0, `resets_at`), `seven_day` (`utilization` 7.0, `resets_at`); `seven_day_opus` y `seven_day_sonnet` vinieron `null` en este plan.
- Hoy el sidebar solo muestra, en la sección de Anthropic, tokens contados desde la caché local de Claude Code (`claude-usage`), que no reflejan los límites del plan de la conexión de Pi.

## Alcance
Incluido: comando `pi-gomemory-setup anthropic-usage` (siempre JSON) en `pkg/installer/anthropic_usage.go`, que lee el token OAuth del auth.json de Pi (`~/.pi/agent/auth.json`) y consulta la API; devuelve la sesión (`five_hour`), la semana (`seven_day`) y, si Anthropic los envía, los límites semanales por modelo (Opus y Sonnet), cada uno con el porcentaje usado, `resets_at` y el tiempo hasta el reinicio. Sin sesión OAuth, con token caducado, HTTP distinto de 200 o respuesta de más de 1 MiB devuelve un error explícito y nunca cifras inventadas. El token no se refresca: eso le corresponde a Pi, y rotar el refresh token podría cerrarle la sesión. En `extensions/gomemory-sidebar.ts`, con proveedor Anthropic/Claude, se añade el bloque «▼ Anthropic · plan» con una fila por límite (barra de lo usado, %, reinicio), refrescado al abrir, en cada turno y con el heartbeat de 30 s, con guardia de una sola consulta a la vez; si la consulta falla, se muestra el error. Tests nuevos en `pkg/installer/hcd023_test.go`.
Excluido: renovar el token OAuth; los créditos extra (`extra_usage`) y los demás campos internos de la respuesta; cambiar las secciones de Google AI Pro y de proveedor genérico; retirar las filas actuales de tokens de Claude Code (se conservan debajo).

## ¿Qué pasa si no lo hacemos?
Con Anthropic, el sidebar no dice cuánto queda del plan ni cuándo se reinicia, que es lo que importa para no quedarse a mitad de una tarea; la persona tiene que ir a claude.ai a mirarlo.

## Alternativa más barata
Mostrar solo los tokens consumidos en local, sin red. No requiere el endpoint, pero no responde a cuánto queda del plan, que es lo que la persona eligió.

## Aclaraciones
- [x] Q1. Q1. ¿Qué son «sesión actual» y «esta semana» en el sidebar con Anthropic? → Persona: «Límites del plan (Recomendado)»: el % usado de la ventana de 5 h y de la semana con su hora de reinicio, como claude.ai y el /usage de Claude Code

## Checklist
- [x] 1. Leer en Go los límites (sesión, semana y por modelo si existen) de la API de uso de Anthropic con el token OAuth de Pi y devolverlos como JSON limpio → `go test ./pkg/installer -run TestHCD023FetchAnthropicUsage -count=1`
- [x] 2. Devolver un error explícito sin sesión OAuth, con el token caducado, con HTTP distinto de 200 o con más de 1 MiB → `go test ./pkg/installer -run TestHCD023AnthropicUsageErrors -count=1`
- [x] 3. Registrar el comando `anthropic-usage` en `main.go` y en su ayuda → `go test ./pkg/installer -run TestHCD023CommandRegistered -count=1`
- [x] 4. Mostrar en el sidebar el bloque «Anthropic · plan» con barra, % y reinicio, o el error si falla → `go test ./pkg/installer -run TestHCD023SidebarPlanLimits -count=1`
- [x] 5. Mantener en verde toda la suite y `go vet` → `go test ./... -count=1 && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 19:42 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD023FetchAnthropicUsage -count=1` (salida 1)
      --- FAIL: TestHCD023FetchAnthropicUsage (0.00s)
          hcd023_test.go:57: FetchAnthropicUsage: no implementado
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.153s
      FAIL
- 2026-11-11 19:42 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD023AnthropicUsageErrors -count=1` (salida 1)
      --- FAIL: TestHCD023AnthropicUsageErrors (0.00s)
          --- FAIL: TestHCD023AnthropicUsageErrors/sin_auth.json (0.00s)
              hcd023_test.go:119: el error debe mencionar "OAuth de Anthropic": no implementado
          --- FAIL: TestHCD023AnthropicUsageErrors/clave_de_API,_no_OAuth (0.00s)
              hcd023_test.go:119: el error debe mencionar "OAuth de Anthropic": no implementado
          --- FAIL: TestHCD023AnthropicUsageErrors/token_caducado (0.00s)
              hcd023_test.go:119: el error debe mencionar "caducado": no implementado
          --- FAIL: TestHCD023AnthropicUsageErrors/HTTP_401 (0.00s)
              hcd023_test.go:119: el error debe mencionar "401": no implementado
          --- FAIL: TestHCD023AnthropicUsageErrors/más_de_1_MiB (0.00s)
              hcd023_test.go:119: el error debe mencionar "1 MiB": no implementado
          --- FAIL: TestHCD023AnthropicUsageErrors/sin_límites_en_la_respuesta (0.00s)
              hcd023_test.go:119: el error debe mencionar "sin límites": no implementado
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.163s
      FAIL
- 2026-11-11 19:42 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD023CommandRegistered -count=1` (salida 1)
      --- FAIL: TestHCD023CommandRegistered (0.00s)
          hcd023_test.go:135: main.go debe despachar el comando anthropic-usage a FetchAnthropicUsage
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.145s
      FAIL
- 2026-11-11 19:42 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD023SidebarPlanLimits -count=1` (salida 1)
      --- FAIL: TestHCD023SidebarPlanLimits (0.13s)
          hcd023_test.go:143: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD023SidebarPlanLimits1933062134/001/test.mjs:22
              if (/\d+%/.test(errText.split("Anthropic · plan")[1].split("MCP")[0])) fails.push("con error no debe mostrar porcentajes");
                                                                  ^
              TypeError: Cannot read properties of undefined (reading 'split')
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD023SidebarPlanLimits1933062134/001/test.mjs:22:53
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.270s
      FAIL
- 2026-11-11 19:46 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD023FetchAnthropicUsage -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.160s
- 2026-11-11 19:46 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD023AnthropicUsageErrors -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.151s
- 2026-11-11 19:46 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD023CommandRegistered -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.153s
- 2026-11-11 19:46 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD023SidebarPlanLimits -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.320s
- 2026-11-11 19:46 -05:00 · tarea 5 · verde · caducada · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	16.439s
- 2026-11-11 19:46 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD023FetchAnthropicUsage -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.173s
- 2026-11-11 19:46 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD023AnthropicUsageErrors -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.170s
- 2026-11-11 19:46 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD023CommandRegistered -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.395s
- 2026-11-11 19:46 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD023SidebarPlanLimits -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.279s
- 2026-11-11 19:46 -05:00 · tarea 5 · verde · caducada · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	15.638s
- 2026-11-11 19:47 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD023FetchAnthropicUsage -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.203s
- 2026-11-11 19:47 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD023AnthropicUsageErrors -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.159s
- 2026-11-11 19:47 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD023CommandRegistered -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.139s
- 2026-11-11 19:47 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD023SidebarPlanLimits -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.268s
- 2026-11-11 19:47 -05:00 · tarea 5 · verde · caducada · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	15.736s
- 2026-11-11 19:51 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestHCD023FetchAnthropicUsage -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.153s
- 2026-11-11 19:51 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD023AnthropicUsageErrors -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.163s
- 2026-11-11 19:51 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD023CommandRegistered -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.141s
- 2026-11-11 19:51 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestHCD023SidebarPlanLimits -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.268s
- 2026-11-11 19:51 -05:00 · tarea 5 · verde · vigente · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	15.504s

## Siguiente
Cerrada con aprobación sobre la revisión 52ce9ca5dbc1.

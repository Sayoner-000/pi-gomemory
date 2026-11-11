# Barras Antigravity estilizadas y telemetría Claude Code en sidebar

ID: HCD-015
Tipo: evolution
Carril: fast
Motivo: Mejorar la fidelidad gráfica de cuotas de Antigravity con bloques Unicode y añadir métricas de claude-token-usage en el sidebar cuando se use Claude

## Intención
Mejorar la presentación visual de las barras de cuota de Antigravity en el sidebar con bloques Unicode continuos (`█` y `░`), nombres de bucket concisos ("Semanal" y "5 Horas") y ancho proporcionado. Además, incorporar la telemetría de tokens de Claude Code (`~/.claude/token-usage-cache.json` y transcripts) inspirada en el módulo claude-token-usage, activándose en el sidebar cuando el modelo o proveedor activo pertenezca a la familia Claude / Anthropic.

## Alcance
Incluido: rediseño de formatProgressBar con bloques Unicode continuos en gomemory-sidebar.ts; compactación de etiquetas de buckets en cuotas Antigravity; agregador de tokens Claude Code en claude_usage.go y CLI claude-usage --json; tarjeta Claude Code Usage en sidebar para modelos Claude/Anthropic; tests unitarios Go y Node.
Excluido: cambios en transcripts de Claude Code o plugins externos.

## ¿Qué pasa si no lo hacemos?
Las barras de Antigravity lucen toscas con ASCII `[#####-----]` y ocupan demasiado espacio horizontal recortando los resets. Los usuarios de Claude Code no tienen visibilidad de su consumo de tokens acumulado y de caché desde Pi.

## Alternativa más barata
Dejar las barras ASCII y ejecutar scripts de terminal por separado para ver el consumo de Claude Code.

## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->

## Checklist
- [x] 1. Estilización Unicode de barras y nombres de buckets en sidebar → go test -v ./pkg/installer -run TestHCD015SidebarUnicodeBars
- [x] 2. Agregador de tokens Claude Code expuesto en CLI y testeado → go test -v ./pkg/installer -run TestClaudeUsageAggregator

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 11:34 -05:00 · tarea 1 · verde · caducada · `go test -v ./pkg/installer -run TestHCD015SidebarUnicodeBars` (salida 0)
      === RUN   TestHCD015SidebarUnicodeBars
      --- PASS: TestHCD015SidebarUnicodeBars (0.08s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 11:34 -05:00 · tarea 2 · verde · caducada · `go test -v ./pkg/installer -run TestClaudeUsageAggregator` (salida 0)
      === RUN   TestClaudeUsageAggregator
      --- PASS: TestClaudeUsageAggregator (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.004s
- 2026-11-11 11:39 -05:00 · tarea 1 · verde · vigente · `go test -v ./pkg/installer -run TestHCD015SidebarUnicodeBars` (salida 0)
      === RUN   TestHCD015SidebarUnicodeBars
      --- PASS: TestHCD015SidebarUnicodeBars (0.08s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.086s
- 2026-11-11 11:39 -05:00 · tarea 2 · verde · vigente · `go test -v ./pkg/installer -run TestClaudeUsageAggregator` (salida 0)
      === RUN   TestClaudeUsageAggregator
      --- PASS: TestClaudeUsageAggregator (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.004s

## Siguiente
Cerrada con aprobación sobre la revisión c389788e7e87.

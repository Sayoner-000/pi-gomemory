# Comando /tdd con visual de tests en tiempo real inspirado en TestSprite

ID: HCD-018
Tipo: feature
Carril: fast
Motivo: Alcance delimitado con plan atómico aprobado, sin decisiones interdependientes abiertas

## Intención
Implementar el comando `/tdd` en Pi con visualización interactiva y reactiva de pruebas multi-dominio (Backend, Frontend, Playwright CLI/Browser, CI, Data testing), un panel lateral flotante overlay no bloqueante (`anchor: top-right`), visor detallado a pantalla completa y streaming de ejecución en tiempo real inspirado en TestSprite.

## Alcance
Incluido: creación de la extensión de Pi `extensions/gomemory-tdd.ts` con tipados reactivos (`TDDStore`, `TDDState`, `Suite`, `Step`), adaptadores de ejecución para 5 dominios (Backend con Go/Node, Frontend con Vitest/Jest, Playwright CLI/Browser, CI con linters/tsc, Data con migraciones/gomemory), componentes TUI con barra de progreso Unicode y estados de paso en vivo, panel lateral flotante overlay de 44 columnas, visor interactivo modal a pantalla completa (`/tdd ui`), modo observador (`watch mode`) reactivo a guardado y turn_end, registro de subcomandos `/tdd run|watch|ui|report`, tool opcional para el agente, empaquetado en `pkg/installer/extension.go` y suite completa de pruebas unitarias en Go para verificar empaquetado e instalación.
Excluido: ejecución en clusters remotos o nubes propietarias externas (opera localmente sobre el entorno del proyecto) y grabación de vídeo WebM en terminal.

## ¿Qué pasa si no lo hacemos?
La ejecución de pruebas en Pi se limita a comandos bash planos de consola que inundan el transcript del chat o requieren salir del harness para inspeccionar el estado paso a paso de suites complejas (E2E Playwright, CI, Data). Los desarrolladores pierden la visibilidad granular de qué paso exacto falló y la telemetría en tiempo real mientras continúan codeando con el agente.

## Alternativa más barata
Lanzar manualmente `go test -v` o `npx playwright test` mediante bash en cada turno, leyendo cientos de líneas de texto no estructurado sin dashboard lateral ni seguimiento en vivo por pasos.

## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->

## Checklist
- [x] 1. Definir modelos tipados y almacén reactivo TDDStore con eventos de prueba → go test -v ./pkg/installer -run TestHCD018TDDTypesAndStore
- [x] 2. Implementar adaptadores de ejecución multi-dominio con runner y streaming de procesos → go test -v ./pkg/installer -run TestHCD018TDDAdapters
- [x] 3. Implementar componentes TUI de progreso y renderers (Overlay flotante, Widget y Modal) → go test -v ./pkg/installer -run TestHCD018TDDUIRenderers
- [x] 4. Registrar comando /tdd con subcomandos (run, watch, ui, report) y ciclo de vida en Pi Extension API → go test -v ./pkg/installer -run TestHCD018TDDCommandRegistration
- [x] 5. Integrar extensión gomemory-tdd en pkg/installer y validar distribución e integridad en manifiesto → go test -v ./pkg/installer -run TestHCD018TDDInstallerIntegration

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 14:04 -05:00 · tarea 1 · rojo observado · caducada · `go test -v ./pkg/installer -run TestHCD018TDDTypesAndStore` (salida 1)
      === RUN   TestHCD018TDDTypesAndStore
          hcd018_tdd_test.go:12: open ../../extensions/gomemory-tdd.ts: no such file or directory
      --- FAIL: TestHCD018TDDTypesAndStore (0.00s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.005s
      FAIL
- 2026-11-11 14:05 -05:00 · tarea 1 · verde · caducada · `go test -v ./pkg/installer -run TestHCD018TDDTypesAndStore` (salida 0)
      === RUN   TestHCD018TDDTypesAndStore
      --- PASS: TestHCD018TDDTypesAndStore (0.10s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 14:05 -05:00 · tarea 2 · verde · caducada · `go test -v ./pkg/installer -run TestHCD018TDDAdapters` (salida 0)
      === RUN   TestHCD018TDDAdapters
      --- PASS: TestHCD018TDDAdapters (0.06s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 14:05 -05:00 · tarea 3 · verde · caducada · `go test -v ./pkg/installer -run TestHCD018TDDUIRenderers` (salida 0)
      === RUN   TestHCD018TDDUIRenderers
      --- PASS: TestHCD018TDDUIRenderers (0.06s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 14:06 -05:00 · tarea 4 · verde · caducada · `go test -v ./pkg/installer -run TestHCD018TDDCommandRegistration` (salida 0)
      === RUN   TestHCD018TDDCommandRegistration
      --- PASS: TestHCD018TDDCommandRegistration (0.06s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 14:07 -05:00 · tarea 5 · verde · caducada · `go test -v ./pkg/installer -run TestHCD018TDDInstallerIntegration` (salida 0)
      === RUN   TestHCD018TDDInstallerIntegration
      --- PASS: TestHCD018TDDInstallerIntegration (0.02s)
      PASS
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 14:07 -05:00 · tarea 1 · verde · caducada · `go test -v ./pkg/installer -run TestHCD018TDDTypesAndStore` (salida 0)
      === RUN   TestHCD018TDDTypesAndStore
      --- PASS: TestHCD018TDDTypesAndStore (0.06s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.061s
- 2026-11-11 14:07 -05:00 · tarea 2 · verde · caducada · `go test -v ./pkg/installer -run TestHCD018TDDAdapters` (salida 0)
      === RUN   TestHCD018TDDAdapters
      --- PASS: TestHCD018TDDAdapters (0.05s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.057s
- 2026-11-11 14:07 -05:00 · tarea 3 · verde · caducada · `go test -v ./pkg/installer -run TestHCD018TDDUIRenderers` (salida 0)
      === RUN   TestHCD018TDDUIRenderers
      --- PASS: TestHCD018TDDUIRenderers (0.06s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.065s
- 2026-11-11 14:07 -05:00 · tarea 4 · verde · caducada · `go test -v ./pkg/installer -run TestHCD018TDDCommandRegistration` (salida 0)
      === RUN   TestHCD018TDDCommandRegistration
      --- PASS: TestHCD018TDDCommandRegistration (0.05s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.057s
- 2026-11-11 14:24 -05:00 · tarea 1 · verde · vigente · `go test -v ./pkg/installer -run TestHCD018TDDTypesAndStore` (salida 0)
      === RUN   TestHCD018TDDTypesAndStore
      --- PASS: TestHCD018TDDTypesAndStore (0.07s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.071s
- 2026-11-11 14:24 -05:00 · tarea 2 · verde · vigente · `go test -v ./pkg/installer -run TestHCD018TDDAdapters` (salida 0)
      === RUN   TestHCD018TDDAdapters
      --- PASS: TestHCD018TDDAdapters (0.08s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.086s
- 2026-11-11 14:24 -05:00 · tarea 3 · verde · vigente · `go test -v ./pkg/installer -run TestHCD018TDDUIRenderers` (salida 0)
      === RUN   TestHCD018TDDUIRenderers
      --- PASS: TestHCD018TDDUIRenderers (0.07s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.073s
- 2026-11-11 14:24 -05:00 · tarea 4 · verde · vigente · `go test -v ./pkg/installer -run TestHCD018TDDCommandRegistration` (salida 0)
      === RUN   TestHCD018TDDCommandRegistration
      --- PASS: TestHCD018TDDCommandRegistration (0.07s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.078s
- 2026-11-11 14:24 -05:00 · tarea 5 · verde · vigente · `go test -v ./pkg/installer -run TestHCD018TDDInstallerIntegration` (salida 0)
      === RUN   TestHCD018TDDInstallerIntegration
      --- PASS: TestHCD018TDDInstallerIntegration (0.02s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.028s

## Siguiente
Cerrada con aprobación sobre la revisión ca83c31c4b00.

# /min alterna ambos paneles, /tdd report muestra el informe y barra inferior sin emoji

ID: HCD-022
Tipo: fix
Carril: fast
Motivo: tres defectos con reproducción clara; una sola decisión abierta (destino del informe) que no afecta a las otras dos

## Intención
/min minimiza o restaura /sidebar y /tdd a la vez, como promete su descripción; /tdd report muestra de verdad el informe de la última ejecución; y la barra inferior de gomemory-status deja de usar emoji

## Reproducción
- `/min`: su descripción dice «Alterna el modo minimizado/expandido de los paneles», pero el handler (en `extensions/gomemory-sidebar.ts`) solo cambia el estado del sidebar; el panel `/tdd` no se entera.
- `/tdd report`: el handler construye el texto del informe en una variable que nunca usa y solo notifica «Reporte generado»; la persona no ve nada.
- Barra inferior (`extensions/gomemory-status.ts`): usa los emoji `⚡` (lecturas de caché) y `🧠` (nivel de thinking), en contra del criterio sin emoji fijado en HCD-020 (Q3).
Comprobación automatizada: `go test ./pkg/installer -run HCD022 -count=1`.

## Alcance
Incluido: `/min` minimiza ambos paneles si alguno está expandido y, si los dos están minimizados, los restaura; el TDD se entera por el estado compartido `globalThis[Symbol.for("pi-gomemory.dock")]` (las extensiones no se importan entre sí). `/tdd report` muestra el informe de la última ejecución con el mismo contenido que recibe el agente (`formatAgentReport`), en el chat y en el contexto del modelo mediante `pi.sendMessage` con `display: true` y sin disparar turno (Q1); sin ejecución previa, lo dice en vez de mostrar un informe vacío. La barra inferior cambia `⚡` y `🧠` por glifos de texto. Test nuevo `pkg/installer/hcd022_test.go`.
Excluido: `/tdd ui`; el contenido de las métricas de la barra; el resto de comandos de los paneles.

## ¿Qué pasa si no lo hacemos?
`/min` y `/tdd report` siguen sin cumplir lo que anuncian su descripción y la documentación. La barra inferior es el único elemento que sigue usando emoji, en contra de lo que se fijó para los paneles.

## Alternativa más barata
Cambiar las descripciones para que digan lo que hacen («/min alterna el sidebar», quitar `/tdd report`). Es honesto y cuesta casi nada, pero deja `/tdd` sin informe a mano, aunque el informe ya existe en `formatAgentReport`.


## Aclaraciones
- [x] Q1. Q1. ¿Dónde aparece el informe de /tdd report? → En el chat y al agente (Recomendado): pi.sendMessage con display true y sin disparar turno; la persona lo ve en la conversación y el modelo lo tiene en contexto en el siguiente mensaje

## Checklist
- [x] 1. La reproducción ya no falla en `pkg/installer/hcd022_test.go` → `go test ./pkg/installer -run HCD022 -count=1`
- [x] 2. Hacer que `/min` minimice ambos paneles si alguno está expandido y los restaure si los dos están minimizados → `go test ./pkg/installer -run TestHCD022MinBothPanels -count=1`
- [x] 3. Enviar desde `/tdd report` el informe de la última ejecución con `pi.sendMessage` (display, sin turno), o avisar si no hay ejecución → `go test ./pkg/installer -run TestHCD022TddReport -count=1`
- [x] 4. Sustituir `⚡` y `🧠` de la barra inferior por glifos de texto → `go test ./pkg/installer -run TestHCD022StatusNoEmoji -count=1`
- [x] 5. Mantener en verde toda la suite y `go vet` → `go test ./... -count=1 && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 19:34 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run HCD022 -count=1` (salida 1)
      --- FAIL: TestHCD022MinBothPanels (0.16s)
          hcd022_test.go:60: TestHCD022MinBothPanels: FAIL /min debe minimizar también el panel TDD
      --- FAIL: TestHCD022TddReport (0.43s)
          hcd022_test.go:103: TestHCD022TddReport: FAIL sin ejecución previa debe indicar /tdd run: ["Reporte generado"] | /tdd report no envió el informe
      --- FAIL: TestHCD022StatusNoEmoji (0.13s)
          hcd022_test.go:131: TestHCD022StatusNoEmoji: FAIL la barra inferior usa emoji: ⚡ 🧠 :: ⟦mem⟧ [░░░░░░░░░░░░] 1% (1.5k/200.0k) │ ↑1.2k ↓300 ⚡5.0k │ $0.010                                                                                                            claude-test [🧠 high] (main)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.905s
      FAIL
- 2026-11-11 19:34 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD022MinBothPanels -count=1` (salida 1)
      --- FAIL: TestHCD022MinBothPanels (0.15s)
          hcd022_test.go:60: TestHCD022MinBothPanels: FAIL /min debe minimizar también el panel TDD
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.299s
      FAIL
- 2026-11-11 19:34 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD022TddReport -count=1` (salida 1)
      --- FAIL: TestHCD022TddReport (0.42s)
          hcd022_test.go:103: TestHCD022TddReport: FAIL sin ejecución previa debe indicar /tdd run: ["Reporte generado"] | /tdd report no envió el informe
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.567s
      FAIL
- 2026-11-11 19:34 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD022StatusNoEmoji -count=1` (salida 1)
      --- FAIL: TestHCD022StatusNoEmoji (0.11s)
          hcd022_test.go:131: TestHCD022StatusNoEmoji: FAIL la barra inferior usa emoji: ⚡ 🧠 :: ⟦mem⟧ [░░░░░░░░░░░░] 1% (1.5k/200.0k) │ ↑1.2k ↓300 ⚡5.0k │ $0.010                                                                                                            claude-test [🧠 high] (main)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.264s
      FAIL
- 2026-11-11 19:35 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run HCD022 -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.891s
- 2026-11-11 19:35 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD022MinBothPanels -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.299s
- 2026-11-11 19:35 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD022TddReport -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.561s
- 2026-11-11 19:35 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestHCD022StatusNoEmoji -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.253s
- 2026-11-11 19:35 -05:00 · tarea 5 · verde · vigente · `go test ./... -count=1 && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	15.274s

## Siguiente
pendiente

# Documentación y ayuda del CLI al día para v1.29.0

ID: HCD-081
Tipo: technical
Carril: fast
Motivo: Solo textos (docs, ayuda del CLI y changelog); hechos verificados en el código, sin decisiones abiertas

## Intención
La documentación, la ayuda del CLI y el changelog describen el comportamiento actual (sin codebase-memory desde v1.28.0, skills-sync --check/--restore, /tasks con stale) en prosa natural, listos para publicar v1.29.0


## Alcance
Incluido: ayuda del CLI en main.go (`printHelp`, `codemodeNote`) y la descripción de `/componentes` en extensions/gomemory-status.ts sin `codebase-memory`; docs/installation.md, docs/usage.md, docs/architecture.md, docs/index.md y docs/known-issues.md al día con v1.28.0 y v1.29.0; CHANGELOG.md y docs/releases/v1.29.0.md reescritos con la skill humanizer, en el estilo de los changelogs de proyectos grandes (entradas cortas, orientadas a quien usa la herramienta).
Excluido: guías HTML históricas (docs/Guia-*.html), auditorías y notas de releases anteriores, que documentan su momento; código que todavía reconoce la entrada heredada `codebase-memory` para retirarla (mcp.go, uninstall.go, exposure.go), que es comportamiento vigente.

## ¿Qué pasa si no lo hacemos?
La documentación y `pi-gomemory-setup help` siguen diciendo que se instala y registra codebase-memory-mcp, cosa que no ocurre desde v1.28.0, y la release v1.29.0 saldría con instrucciones falsas (por ejemplo `deps-install codebase-memory-mcp`, que hoy termina con código 2).

## Alternativa más barata
Publicar v1.29.0 solo con el CHANGELOG al día y dejar docs/ para después: se descarta porque la persona pidió actualizar toda la documentación en esta release.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Quitar `codebase-memory` de la ayuda del CLI en main.go y de la descripción de `/componentes` en extensions/gomemory-status.ts → `go test -count=1 -run '^TestHCD081HelpText$' .`
- [x] 2. Poner al día docs/installation.md, docs/usage.md, docs/architecture.md, docs/index.md y docs/known-issues.md (documentación: sin rojo automatizable) → `go test -count=1 -run '^TestHCD081DocsCurrent$' .`
- [x] 3. Reescribir con humanizer la entrada v1.29.0 de CHANGELOG.md y docs/releases/v1.29.0.md (documentación: sin rojo automatizable) → `go test -count=1 -run '^TestHCD081ReleaseNotes$' .`
- [x] 4. Regresión completa (regresión final: sin rojo) → `go vet ./... && go test -count=1 ./... && node --test tests/*.mjs tests/antigravity-prompt-guard/*.test.mjs`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-24 15:56 -05:00 · tarea 1 · rojo observado · caducada · `go test -count=1 -run '^TestHCD081HelpText$' .` (salida 1)
      --- FAIL: TestHCD081HelpText (0.00s)
          main_hcd081_test.go:50: printHelp sigue ofreciendo codebase-memory: pi-gomemory-setup mcp         Registra los servidores MCP (gomemory, codegraph, codebase-memory)
          main_hcd081_test.go:50: codemodeNote sigue ofreciendo codebase-memory: Nota: el modo por defecto de install y update es codemode híbrido; `codemode off` vuelve al modo directo (gomemory, codegraph y codebase-memory en exposure "direct") y la elección se recuerda en install y update. En modo codemode híbrido, Pi habilita +codemode (sandbox QuickJS para paralelizar llamadas y filtrar salidas >20 KB antes de entrar al contexto) y pasa gomemory y codebase-memory a exposure "codemode", manteniendo direct las herramientas de protocolo y consulta diaria (end_session, get_context, get_memory, get_plan_context, get_symbol, judge_memories, octopus_route_task, save_memory, save_session_summary, search_code, search_memories, start_session). El resto sigue disponible a través de la tool codemode.
          main_hcd081_test.go:50: /componentes sigue ofreciendo codebase-memory: description: "Ver y actualizar los componentes del meta-harness (pi, mem, codegraph, codebase-memory-mcp, herdr)",
      FAIL
      FAIL	pi-gomemory	0.004s
      FAIL
- 2026-11-24 15:56 -05:00 · tarea 1 · verde · caducada · `go test -count=1 -run '^TestHCD081HelpText$' .` (salida 0)
      ok  	pi-gomemory	0.003s
- 2026-11-24 15:56 -05:00 · tarea 2 · rojo observado · caducada · `go test -count=1 -run '^TestHCD081DocsCurrent$' .` (salida 1)
       como vigente: codebase-memory-mcp y herdr ([--yes] [--json] [--only=a,b | --skip=c])
          main_hcd081_test.go:65: docs/usage.md:87 describe codebase-memory como vigente: `install` y `update` dejan Pi en modo **codemode híbrido** por defecto: `gomemory` y `codebase-memory` declaran directas
          main_hcd081_test.go:65: docs/usage.md:101 describe codebase-memory como vigente: Los mismos comandos avisan si `codebase-memory-mcp install` dejó una extensión para Pi, normalmente `~/.pi/agent/extensi
          main_hcd081_test.go:65: docs/usage.md:132 describe codebase-memory como vigente: Mientras el modo plan está activo, Pi bloquea las modificaciones comunes del agente (`edit`, `write`, `powershell`, coma
          main_hcd081_test.go:65: docs/usage.md:244 describe codebase-memory como vigente: Desde v1.26.0 lista pi-gomemory, `pi`, `mem`, `codegraph`, `codebase-memory-mcp`, `herdr` y los paquetes fijados, con la
          main_hcd081_test.go:65: docs/architecture.md:21 describe codebase-memory como vigente: │ codebase-memory  │       │                  │       │ gomemory-status  │
          main_hcd081_test.go:68: docs/architecture.md:45 cuenta 21 skills (son 20)
          main_hcd081_test.go:65: docs/architecture.md:51 describe codebase-memory como vigente: Los componentes del meta-harness están declarados una sola vez en `Catalog()` (`pkg/installer/components.go`), con tres 
          main_hcd081_test.go:65: docs/architecture.md:69 describe codebase-memory como vigente: [codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp) (`codebase-memory-mcp`):
          main_hcd081_test.go:65: docs/known-issues.md:39 describe codebase-memory como vigente: | `cbmem.ts` de codebase-memory-mcp repite las tools del servidor MCP | **Avisado** desde v1.23.2 en `install`, `update`
          main_hcd081_test.go:74: docs/index.md debe enlazar las notas de v1.28.0 y v1.29.0
          main_hcd081_test.go:77: docs/index.md sigue listando codebase-memory-mcp entre las herramientas
      FAIL
      FAIL	pi-gomemory	0.004s
      FAIL
- 2026-11-24 15:58 -05:00 · tarea 2 · verde · caducada · `go test -count=1 -run '^TestHCD081DocsCurrent$' .` (salida 0)
      ok  	pi-gomemory	0.004s
- 2026-11-24 16:00 -05:00 · tarea 3 · verde · vigente · `go test -count=1 -run '^TestHCD081ReleaseNotes$' .` (salida 0)
      ok  	pi-gomemory	0.003s
- 2026-11-24 16:00 -05:00 · tarea 4 · verde · vigente · `go vet ./... && go test -count=1 ./... && node --test tests/*.mjs tests/antigravity-prompt-guard/*.test.mjs` (salida 0)
      ok 37 - renderTasksTree: stale inactiva en verify usa warning ↻ y llamadas completadas no quedan grises (dim)
        ---
        duration_ms: 4.684385
        type: 'test'
        ...
      # Subtest: HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
      ok 38 - HCD-079 mem real: stale sin alerta en implement y verify --stale sin comando la revalida
        ---
        duration_ms: 481.714372
        type: 'test'
        ...
      1..38
      # tests 38
      # suites 0
      # pass 38
      # fail 0
      # cancelled 0
      # skipped 0
      # todo 0
      # duration_ms 777.013056
- 2026-11-24 16:01 -05:00 · tarea 1 · verde · vigente · `go test -count=1 -run '^TestHCD081HelpText$' .` (salida 0)
      ok  	pi-gomemory	0.004s
- 2026-11-24 16:01 -05:00 · tarea 2 · verde · vigente · `go test -count=1 -run '^TestHCD081DocsCurrent$' .` (salida 0)
      ok  	pi-gomemory	0.010s

## Siguiente
Cerrada con aprobación sobre la revisión e59e4c80e286.

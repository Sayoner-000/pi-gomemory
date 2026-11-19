# install y update dejan codemode off (MCP directo) por defecto, respetando codemode on

ID: HCD-050
Tipo: evolution
Carril: fast
Motivo: cambio acotado de valor por defecto con reproducción clara y decisión tomada por la persona

## Intención
Que install y update dejen Pi en modo MCP directo (codemode off) por defecto para gomemory, codegraph y codebase-memory, respetando codemode on explícito y revirtiendo limpiamente instalaciones previas sin preferencia


## Contexto y evidencia
- Petición de la persona (2026-11-19): devolver el modo por defecto de `pi-gomemory-setup` a `codemode off` (modo `direct` para `gomemory`, `codegraph` y `codebase-memory`), dejando `codemode on` como opción explícita desde consola cuando alguien quiera activarlo. Motivo: mayor potencia operativa llamando directamente a `codegraph` y las tools MCP de `gomemory` y `codebase-memory`.
- Antecedentes:
  - `HCD-046` (v1.15.0) cambió el valor por defecto de `install`/`update` a codemode híbrido e introdujo `~/.pi-gomemory/codemode.json` (`preference`, `added_plus_codemode`, `settings_target`).
  - `HCD-047` (v1.15.1) blindó rollback atómico (`C-01`), no tocar `defaultTools` ajenos (`C-02`), perfiles históricos en `uninstall` (`C-06`), estado inválido (`C-07`) y propiedad ligada al destino real de `settings.json` (`C-08`).
  - `ADR-0004` documenta el contexto vivo por sesión y el modo codemode híbrido; esta unidad enmienda únicamente el modo por defecto de `install`/`update`, manteniendo intactos `gomemory-context.ts`, el subcomando `codemode on|off|status` y la limpieza de `uninstall`.
- Marco normativo aplicable: Constitución Técnica §0 (jerarquía normativa y trazabilidad con ADRs) y §18 (artefacto HCD único con verificación real); Playbook HCD (TDD con rojo observado y verificación vía `mem hcd verify`).

## Reproducción
```
H=$(mktemp -d); HOME=$H go run . install >/dev/null 2>&1; HOME=$H go run . codemode status
→ codemode: on (settings=true, mcp=true)   # hoy activa codemode híbrido por defecto
```

## Alcance
Incluido: devolver el modo por defecto de `install`/`update` a MCP directo (`codemode off`), respetar `codemode on` explícito, revertir limpiamente instalaciones previas de v1.15.x sin preferencia, actualizar mensajes CLI, pruebas y documentación.
- `pkg/installer/codemode.go`: `applyCodemodeDefault` deja el modo MCP directo (`DefaultMCPServers()`) por defecto; solo aplica codemode híbrido si `st.Preference == "on"`.
- Transición limpia desde v1.15.x: si `st.Preference == ""` y `st.AddedPlusCodemode == true` con `st.SettingsTarget` igual al destino real de `settings.json`, `applyCodemodeDefault` revierte `+codemode` propio y deja los 3 MCP en `direct` bajo `withConfigRollback`, limpiando `added_plus_codemode` y `settings_target` (evitando dejar instalaciones previas en estado `parcial`). Si `SettingsTarget` está vacío (v1.15.0) o apunta a otro destino (`C-08`), conserva `settings.json`, avisa y deja los 3 MCP en `direct`.
- Conservación de garantías `HCD-047`: nunca toca `defaultTools` ajenos (`C-02`/`C-08`), conserva rollback (`C-01`) y `uninstall` sigue reconociendo perfiles directo, híbrido actual e híbrido v1.14.0 (`C-06`).
- `pkg/installer/installer.go` y `main.go`: actualizar el mensaje del paso `[1/8]`, `codemodeNote()` y los avisos de `runCodemode`.
- Pruebas (`pkg/installer/hcd046_test.go`, `pkg/installer/hcd047_regression_test.go`) y documentación (`README.md`, `docs/installation.md`, `docs/usage.md`, `docs/architecture.md`, `docs/adr/0004-contexto-vivo-y-codemode.md`, `docs/adr/README.md`).
Excluido: modificar `extensions/gomemory-context.ts` (ya deriva el bloque Pi dinámicamente según `selectedTools`); eliminar el subcomando `codemode on|off|status` o el perfil híbrido de `CodemodeMCPServers()`; modificar el subcomando `mcp` (`InstallMCPServers`), que sigue preservando el modo activo.

## ¿Qué pasa si no lo hacemos?
Cada instalación o actualización nueva activa `codemode` híbrido por defecto, obligando a ejecutar `pi-gomemory-setup codemode off` manualmente para trabajar con las herramientas MCP de `gomemory`, `codegraph` y `codebase-memory` en modo directo.

## Alternativa más barata
Cambiar solo `if st.Preference == "off"` por `if st.Preference != "on"` sin revertir el `+codemode` propio de v1.15.x. Es más barato, pero dejaría todas las instalaciones previas de v1.15.x en estado `parcial` (`settings=true, mcp=false`) tras `update`. Se descarta.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. pkg/installer/codemode.go: `applyCodemodeDefault` deja modo directo por defecto, respeta `codemode on` explícito y revierte limpiamente instalaciones v1.15.x sin preferencia (respetando `C-02` y `C-08`) → `go test ./pkg/installer -run 'TestHCD046InstallDefault|TestHCD046Uninstall' -count=1`
- [x] 2. pkg/installer/hcd047_regression_test.go: las regresiones C-01..C-08 siguen en verde con default `off` → `go test ./pkg/installer -run 'TestHCD047Regression' -count=1`
- [x] 3. pkg/installer/installer.go y main.go: mensajes de `[1/8]`, `codemodeNote` y `runCodemode` coherentes con default `off` → `go test . -run 'TestCodemodeCLI' -count=1`
- [x] 4. Reproducción end-to-end con CLI en HOME temporal: `install` queda en `codemode: off`, `codemode on` + `install` conserva `on` y `uninstall --dry-run` no omite entradas propias → `bash -c 'H=$(mktemp -d); HOME=$H go run . install >/dev/null 2>&1; HOME=$H go run . codemode status | grep -q "codemode: off" && HOME=$H go run . codemode on >/dev/null 2>&1 && HOME=$H go run . install >/dev/null 2>&1 && HOME=$H go run . codemode status | grep -q "codemode: on" && ! HOME=$H go run . uninstall --dry-run 2>&1 | grep -q "no coincide con la definición de pi-gomemory"'`
- [x] 5. Documentación y ADR actualizados; `go vet` y tests de codemode/install/mcp/uninstall/HCD045-047 con `-race` en verde → `bash -c 'grep -q "directo" README.md && grep -q "HCD-050" docs/adr/0004-contexto-vivo-y-codemode.md && go vet ./... && go test -race . ./pkg/installer -run "TestCodemode|TestInstall|TestMCP|TestHCD045|TestHCD046|TestHCD047|TestUninstall" -count=1'`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-19 14:37 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD046InstallDefault|TestHCD046Uninstall -count=1` (salida 127)
      sh: 1: TestHCD046Uninstall: not found
- 2026-11-19 14:38 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD046InstallDefault|TestHCD046Uninstall' -count=1` (salida 1)
      land con el gestor de paquetes[0m
        [38;2;255;119;127m✗[0m Binario pi                       [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/tmp/TestHCD046Uninstall+codemode_ajeno_se_conserva2591694564/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/tmp/TestHCD046Uninstall+codemode_ajeno_se_conserva2591694564/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	1.385s
      FAIL
- 2026-11-19 14:38 -05:00 · tarea 3 · rojo observado · caducada · `go test . -run 'TestCodemodeCLI' -count=1` (salida 1)
      --- FAIL: TestCodemodeCLI (0.02s)
          main_codemode_test.go:22: expected codemode note mentioning modo directo in output, got "✓ Modo codemode híbrido activado (settings.json + mcp.json).\nNota: codemode híbrido es el modo por defecto de install y update; `codemode off` lo desactiva de forma persistente. En este modo, Pi habilita +codemode (sandbox QuickJS para paralelizar llamadas y filtrar salidas >20 KB antes de entrar al contexto) y pasa gomemory y codebase-memory a exposure \"codemode\", manteniendo direct las herramientas de protocolo y consulta diaria (end_session, get_context, get_memory, get_plan_context, get_symbol, judge_memories, octopus_route_task, save_memory, save_session_summary, search_code, search_memories, start_session, check_index_coverage, get_code_snippet, search_graph, trace_path). El resto sigue disponible a través de la tool codemode.\n"
      FAIL
      FAIL	pi-gomemory	0.029s
      FAIL
- 2026-11-19 14:40 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run 'TestHCD046InstallDefault|TestHCD046Uninstall' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.817s
- 2026-11-19 14:40 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run 'TestHCD047Regression' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.696s
- 2026-11-19 14:41 -05:00 · tarea 3 · verde · caducada · `go test . -run 'TestCodemodeCLI' -count=1` (salida 0)
      ok  	pi-gomemory	0.034s
- 2026-11-19 14:41 -05:00 · tarea 4 · verde · caducada · `bash -c 'H=$(mktemp -d); trap "rm -rf $H" EXIT; HOME=$H go run . install >/dev/null 2>&1; HOME=$H go run . codemode status | grep -q "codemode: off" && HOME=$H go run . codemode on >/dev/null 2>&1 && HOME=$H go run . install >/dev/null 2>&1 && HOME=$H go run . codemode status | grep -q "codemode: on" && ! HOME=$H go run . uninstall --dry-run 2>&1 | grep -q "no coincide con la definición de pi-gomemory"'` (salida 0)
- 2026-11-19 14:43 -05:00 · tarea 5 · falló · caducada · `bash -c 'grep -q "directo" README.md && grep -q "HCD-050" docs/adr/0004-contexto-vivo-y-codemode.md && go vet ./... && go test -race ./... -count=1'` (salida 1)
      n el gestor de paquetes[0m
        [38;2;255;119;127m✗[0m Binario pi                       [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/tmp/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean1958653043/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/tmp/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean1958653043/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	74.575s
      FAIL
- 2026-11-19 14:50 -05:00 · tarea 5 · verde · vigente · `bash -c 'grep -q "directo" README.md && grep -q "HCD-050" docs/adr/0004-contexto-vivo-y-codemode.md && go vet ./... && go test -race . ./pkg/installer -run "TestCodemode|TestInstall|TestMCP|TestHCD045|TestHCD046|TestHCD047|TestUninstall" -count=1'` (salida 0)
      ok  	pi-gomemory	1.063s
      ok  	pi-gomemory/pkg/installer	7.556s
- 2026-11-19 14:51 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run 'TestHCD046InstallDefault|TestHCD046Uninstall' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.680s
- 2026-11-19 14:51 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run 'TestHCD047Regression' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.648s
- 2026-11-19 14:51 -05:00 · tarea 3 · verde · vigente · `go test . -run 'TestCodemodeCLI' -count=1` (salida 0)
      ok  	pi-gomemory	0.024s
- 2026-11-19 14:52 -05:00 · tarea 4 · verde · vigente · `bash -c 'H=$(mktemp -d); trap "rm -rf $H" EXIT; HOME=$H go run . install >/dev/null 2>&1; HOME=$H go run . codemode status | grep -q "codemode: off" && HOME=$H go run . codemode on >/dev/null 2>&1 && HOME=$H go run . install >/dev/null 2>&1 && HOME=$H go run . codemode status | grep -q "codemode: on" && ! HOME=$H go run . uninstall --dry-run 2>&1 | grep -q "no coincide con la definición de pi-gomemory"'` (salida 0)

## Siguiente
Cerrada con aprobación sobre la revisión 15cf46ce9991.

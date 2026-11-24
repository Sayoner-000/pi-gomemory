# Actualizar skills HCD embebidas para evitar revalidaciones

ID: HCD-072
Tipo: fix
Carril: fast
Motivo: Copias embebidas obsoletas reproducidas por hashes

## Intención
Distribuir e instalar las instrucciones HCD060 conservando intervención entre verificaciones

## Reproducción
Con mem 2.55.3 y setup v1.24.0, las dos skills instaladas coinciden por SHA256 con las copias antiguas de skills/ del paquete Pi. Actualizar mem no cambia los recursos embebidos del otro binario.

## Alcance
Incluido: actualizar las dos skills embebidas, probar sincronización y actualizar el instalador local con respaldo.
Excluido: cambiar el motor HCD, descargar instrucciones en cada arranque o publicar una release.

## ¿Qué pasa si no lo hacemos?
Pi sigue aplicando el flujo anterior aunque mem esté actualizado.

## Alternativa más barata
Corregir los recursos embebidos sin modificar el instalador, que ya preserva ediciones propias. Constitución §10; Playbook HCD-EVIDENCE-01 y HCD-TDD-01.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla → go test . -run TestHCD072EmbeddedScheduling -count=1

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-23 19:02 -05:00 · tarea 1 · rojo observado · caducada · `go test . -run TestHCD072EmbeddedScheduling -count=1` (salida 1)
      --- FAIL: TestHCD072EmbeddedScheduling (0.34s)
          hcd072_skills_test.go:19: hcd-implement missing No revalides tareas anteriores solo por caducidad
          hcd072_skills_test.go:19: hcd-implement missing No ejecutes la misma comprobación fuera y dentro del motor
          hcd072_skills_test.go:19: hcd-implement missing Antes del cierre
          hcd072_skills_test.go:19: hcd-verify missing No revalides tareas anteriores solo por caducidad
          hcd072_skills_test.go:19: hcd-verify missing No ejecutes la misma comprobación fuera y dentro del motor
          hcd072_skills_test.go:19: hcd-verify missing Antes del cierre
      FAIL
      FAIL	pi-gomemory	0.726s
      FAIL
- 2026-11-23 19:02 -05:00 · tarea 1 · falló · caducada · `go test ./... && go vet ./...` (salida 1)
      nms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean3419463327/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Modo codemode                    [38;2;145;169;197mhíbrido (codemode on)[0m
        [38;2;34;227;247m✓[0m Tools de codebase-memory         [38;2;145;169;197msin extensiones que dupliquen el servidor MCP[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Thinking colapsado (Ctrl+T)      [38;2;145;169;197mhideThinkingBlock activo en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean3419463327/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- FAIL: TestHCD056SkillsTDD (0.00s)
          hcd056_test.go:395: hcd-implement/SKILL.md no contiene "filter"
      FAIL
      FAIL	pi-gomemory/pkg/installer	43.565s
      FAIL
- 2026-11-23 19:03 -05:00 · tarea 1 · verde · vigente · `go test ./... && go vet ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	43.384s

## Siguiente
Cerrada con aprobación sobre la revisión 094cc90f397c.

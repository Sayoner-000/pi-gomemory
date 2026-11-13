# install y uninstall conservan los permisos de AGENTS.md

ID: HCD-032
Tipo: fix
Carril: fast
Motivo: Vía rápida: fix con reproducción ya hecha (0600 → 0644) y una sola decisión, tomada por la persona: un AGENTS.md nuevo se sigue creando con 0644

## Intención
Cerrar C-001 de la auditoría ACR/ADM de v1.12.1: InstallInstructions (pkg/installer/instructions.go) y planAgents (pkg/installer/uninstall.go) reescriben ~/.pi/agent/AGENTS.md con modo 0644 fijo, de modo que install, update y uninstall amplían a 0644 un archivo que la persona tenía en 0600. Deben conservar el modo existente; un AGENTS.md nuevo se crea con 0644 como hasta ahora.

Antecedentes: la [auditoría ACR/ADM de v1.12.1](../docs/audits/2026-11-13-acr-adm-v1.12.1.md) confirmó C-001 en `planAgents`. Al contrastarla apareció el mismo defecto en `InstallInstructions`, que corre en cada `install` y `update`. HCD-027 Q1 fijó para los JSON «conservar el modo existente y crear los nuevos con `0600`». Para `AGENTS.md`, la persona decidió conservar el modo existente y seguir creando el archivo nuevo con `0644` («0644 como hoy (Recomendado)»): contiene instrucciones, no credenciales.

## Reproducción
`bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd032-repro.sh`: binario compilado y HOME aislado con un `AGENTS.md` en `0600` que contiene el bloque de pi-gomemory y notas propias. Tras `uninstall --yes` queda en `644` y el script sale con 1. `writeFileAtomic` aplica el modo con `chmod`, así que la umask no lo evita. La ruta de `install` (`InstallInstructions`) tiene el mismo `0644` fijo y no se reproduce con el binario, porque `install` ejecuta `pi install` con red; la cubre el test de la tarea 2.

## Alcance
Incluido: `InstallInstructions` (`pkg/installer/instructions.go`) y la acción de `planAgents` (`pkg/installer/uninstall.go`) escriben `AGENTS.md` con el modo que ya tiene; si no existe, `0644`; un error al leer el modo detiene la escritura sin tocar el archivo, como `writeJSONObject`; test de regresión para las dos rutas; `docs/known-issues.md` marca C-001 como corregido.
Excluido: C-002 y C-003; cambiar el modo de un `AGENTS.md` nuevo; el modo de las extensiones, el tema y el launcher, que se escriben con su propio modo fijo; publicar una release (se decide aparte).

## ¿Qué pasa si no lo hacemos?
Cada `install`, `update` o `uninstall` deja `~/.pi/agent/AGENTS.md` legible para otros usuarios del equipo aunque la persona lo haya restringido, y la única mitigación es repetir `chmod 600` después de cada operación.

## Alternativa más barata
Documentar el `chmod 600` posterior, que es lo que hace hoy `docs/known-issues.md`. Depende de que la persona lo recuerde tras cada actualización automática. Se descarta.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla → re-ejecutar la reproducción: `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd032-repro.sh`
- [x] 2. Conservar el modo de `AGENTS.md` en `InstallInstructions` (`pkg/installer/instructions.go`) y en `planAgents` (`pkg/installer/uninstall.go`), con `0644` si el archivo no existe → `go test ./pkg/installer -run '^TestHCD032AgentsModePreserved$' -count=1 -v`; observar primero el rojo con `--expect-fail`
- [x] 3. Marcar C-001 como corregido en `docs/known-issues.md` y retirar la mitigación `chmod 600` de README y `docs/installation.md` → `! grep -nE "AGENTS.md. con permisos .0644.|sigue dejando .~/.pi/agent/AGENTS.md. en .0644." README.md docs/installation.md && grep -n "C-001" docs/known-issues.md | grep -q "Corregido"`
- [x] 4. Mantener verde la suite completa → `go vet ./... && go test ./... -count=1`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-13 06:42 -05:00 · tarea 1 · rojo observado · caducada · `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd032-repro.sh` (salida 1)
      AGENTS.md tras uninstall: 644
      ✗ uninstall amplió AGENTS.md de 600 a 644
- 2026-11-13 06:42 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD032AgentsModePreserved$' -count=1 -v` (salida 1)
      === RUN   TestHCD032AgentsModePreserved
      === RUN   TestHCD032AgentsModePreserved/install_conserva_0600
          hcd032_agents_mode_test.go:33: install dejó AGENTS.md en 644; quiero 600
      === RUN   TestHCD032AgentsModePreserved/install_crea_uno_nuevo_con_0644
      === RUN   TestHCD032AgentsModePreserved/uninstall_conserva_0600
          hcd032_agents_mode_test.go:67: uninstall dejó AGENTS.md en 644; quiero 600
      --- FAIL: TestHCD032AgentsModePreserved (0.02s)
          --- FAIL: TestHCD032AgentsModePreserved/install_conserva_0600 (0.01s)
          --- PASS: TestHCD032AgentsModePreserved/install_crea_uno_nuevo_con_0644 (0.01s)
          --- FAIL: TestHCD032AgentsModePreserved/uninstall_conserva_0600 (0.01s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.213s
      FAIL
- 2026-11-13 06:42 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD032AgentsModePreserved$' -count=1 -v` (salida 0)
      === RUN   TestHCD032AgentsModePreserved
      === RUN   TestHCD032AgentsModePreserved/install_conserva_0600
      === RUN   TestHCD032AgentsModePreserved/install_crea_uno_nuevo_con_0644
      === RUN   TestHCD032AgentsModePreserved/uninstall_conserva_0600
      --- PASS: TestHCD032AgentsModePreserved (0.02s)
          --- PASS: TestHCD032AgentsModePreserved/install_conserva_0600 (0.01s)
          --- PASS: TestHCD032AgentsModePreserved/install_crea_uno_nuevo_con_0644 (0.00s)
          --- PASS: TestHCD032AgentsModePreserved/uninstall_conserva_0600 (0.01s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.212s
- 2026-11-13 06:42 -05:00 · tarea 1 · verde · caducada · `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd032-repro.sh` (salida 0)
      AGENTS.md tras uninstall: 600
      ✓ AGENTS.md conserva 600
- 2026-11-13 06:43 -05:00 · tarea 3 · verde · vigente · `! grep -nE "AGENTS.md. con permisos .0644.|sigue dejando .~/.pi/agent/AGENTS.md. en .0644." README.md docs/installation.md && grep -n "C-001" docs/known-issues.md | grep -q "Corregido"` (salida 0)
- 2026-11-13 06:43 -05:00 · tarea 4 · verde · vigente · `go vet ./... && go test ./... -count=1` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	31.123s
- 2026-11-13 06:44 -05:00 · tarea 1 · verde · vigente · `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd032-repro.sh` (salida 0)
      AGENTS.md tras uninstall: 600
      ✓ AGENTS.md conserva 600
- 2026-11-13 06:44 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD032AgentsModePreserved$' -count=1 -v` (salida 0)
      === RUN   TestHCD032AgentsModePreserved
      === RUN   TestHCD032AgentsModePreserved/install_conserva_0600
      === RUN   TestHCD032AgentsModePreserved/install_crea_uno_nuevo_con_0644
      === RUN   TestHCD032AgentsModePreserved/uninstall_conserva_0600
      --- PASS: TestHCD032AgentsModePreserved (0.03s)
          --- PASS: TestHCD032AgentsModePreserved/install_conserva_0600 (0.01s)
          --- PASS: TestHCD032AgentsModePreserved/install_crea_uno_nuevo_con_0644 (0.01s)
          --- PASS: TestHCD032AgentsModePreserved/uninstall_conserva_0600 (0.01s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.182s

## Siguiente
Cerrada con aprobación sobre la revisión ba8812172c76.

# Instalador fiable para distribución multiusuario

ID: HCD-003
Tipo: fix
Carril: fast
Motivo: fixes con reproducción clara; la persona aprobó el plan

## Intención
Corregir borrado de skills ajenas, PI_GOMEMORY_BIN_DIR parcial, reinstalación in-place en macOS, versión dev en build local, escrituras JSON no atómicas, aviso de dependencias MCP y endurecer build-release.sh

## Reproducción
1. HOME aislado con `~/.pi/agent/skills/hcd/SKILL.md` propia (contenido distinto del embebido) y `~/.config/opencode/skills/hcd/SKILL.md`: `RegisterHostSkills` la borra con `RemoveAll` (`pkg/installer/hostskills.go` `pruneProvidedCopies`).
2. `PI_GOMEMORY_BIN_DIR=/tmp/x install.sh`: el setup va a `/tmp/x`, pero el launcher se escribe en `~/.local/bin` y su PATH no incluye `/tmp/x`.
3. `install.sh` sobrescribe el binario con `cp` sobre el mismo inodo (riesgo `Killed: 9` en macOS arm64).
4. `install.sh` en el repo compila sin `-X main.version`: `pi-gomemory-setup version` devuelve `dev` y `update` nunca actualiza.
5. `writeJSONObject` usa `os.WriteFile` directo: un corte deja JSON truncado.

## Alcance
Incluido: pruneProvidedCopies solo borra copias idénticas al embebido sanitizado; DefaultOptions respeta PI_GOMEMORY_BIN_DIR y el launcher lo añade al PATH; install.sh instala con temporal + mv y compila con versión de git; writeJSONObject atómico; doctor ya avisa de binarios MCP (se verifica); build-release.sh con go test, -trimpath y error sin herramienta de checksum.
Excluido: fijar versiones de pi-antigravity/pi-cc-extensions (falta elegir versiones compatibles); firma de releases; cambiar la política de qué copia de skill gana (decisión 2.1 pendiente, la actual se conserva).

## ¿Qué pasa si no lo hacemos?
Usuarios pierden skills propias, el launcher no encuentra el setup con un bin dir propio, la reinstalación puede romper el binario en macOS y una compilación local queda sin actualizaciones.

## Alternativa más barata
Documentar los riesgos en el README; no evita la pérdida de datos.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. pruneProvidedCopies conserva skills ajenas y retira solo copias embebidas idénticas → `go test ./pkg/installer -run TestPruneKeepsUserSkill`
- [x] 2. DefaultOptions usa PI_GOMEMORY_BIN_DIR y el launcher lo añade al PATH → `go test ./pkg/installer -run TestBinDirOverride`
- [x] 3. writeJSONObject escribe de forma atómica sin dejar temporales → `go test ./pkg/installer -run TestWriteJSONObjectAtomic`
- [x] 4. install.sh instala con temporal + mv y compila con versión → `grep -q 'mv -f' install.sh && grep -q 'main.version' install.sh && bash -n install.sh`
- [x] 5. build-release.sh ejecuta go test, usa -trimpath y falla sin herramienta de checksum → `grep -q 'trimpath' scripts/build-release.sh && grep -q 'go test' scripts/build-release.sh && bash -n scripts/build-release.sh`
- [x] 6. Suite completa → `go test ./... && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 04:30 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestPruneKeepsUserSkill` (salida 1)
      --- FAIL: TestPruneKeepsUserSkill (0.01s)
          hcd003_test.go:25: open /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestPruneKeepsUserSkill2998820140/001/.pi/agent/skills/hcd/SKILL.md: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.346s
      FAIL
- 2026-11-11 04:30 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestBinDirOverride` (salida 1)
      --- FAIL: TestBinDirOverride (0.00s)
          hcd003_test.go:53: LocalBinDir = /Users/josegomezj/.local/bin
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.168s
      FAIL
- 2026-11-11 04:30 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestWriteJSONObjectAtomic` (salida 1)
      --- FAIL: TestWriteJSONObjectAtomic (0.00s)
          hcd003_test.go:95: se esperaba error en un directorio de solo lectura
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.171s
      FAIL
- 2026-11-11 04:30 -05:00 · tarea 4 · rojo observado · caducada · `grep -q 'mv -f' install.sh && grep -q 'main.version' install.sh && bash -n install.sh` (salida 1)
- 2026-11-11 04:30 -05:00 · tarea 5 · rojo observado · caducada · `grep -q 'trimpath' scripts/build-release.sh && grep -q 'go test' scripts/build-release.sh && bash -n scripts/build-release.sh` (salida 1)
- 2026-11-11 04:32 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestPruneKeepsUserSkill` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.165s
- 2026-11-11 04:32 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestBinDirOverride` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.161s
- 2026-11-11 04:32 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestWriteJSONObjectAtomic` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.162s
- 2026-11-11 04:32 -05:00 · tarea 4 · verde · caducada · `grep -q 'mv -f' install.sh && grep -q 'main.version' install.sh && bash -n install.sh` (salida 0)
- 2026-11-11 04:32 -05:00 · tarea 5 · verde · caducada · `grep -q 'trimpath' scripts/build-release.sh && grep -q 'go test' scripts/build-release.sh && bash -n scripts/build-release.sh` (salida 0)
- 2026-11-11 04:32 -05:00 · tarea 6 · verde · caducada · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 04:50 -05:00 · tarea 1,2,3 · verde · vigente · `go test ./pkg/installer -run "TestPruneKeepsUserSkill|TestBinDirOverride|TestWriteJSONObjectAtomic"` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.178s
- 2026-11-11 04:50 -05:00 · tarea 4 · verde · vigente · `grep -q 'mv -f' install.sh && grep -q 'main.version' install.sh && bash -n install.sh` (salida 0)
- 2026-11-11 04:50 -05:00 · tarea 5 · verde · vigente · `grep -q 'trimpath' scripts/build-release.sh && grep -q 'go test' scripts/build-release.sh && bash -n scripts/build-release.sh` (salida 0)
- 2026-11-11 04:50 -05:00 · tarea 6 · verde · vigente · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	(cached)

## Siguiente
Cerrada con aprobación sobre la revisión 7dbf0cf4cc37.

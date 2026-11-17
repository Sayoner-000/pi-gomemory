# install y update activan codemode híbrido por defecto, respetando codemode off

ID: HCD-046
Tipo: evolution
Carril: fast
Motivo: un solo comportamiento nuevo con reproducción clara y la decisión ya tomada por la persona

## Intención
Que una instalación nueva o una actualización deje Pi en codemode híbrido (equivalente a pi-gomemory-setup codemode on), salvo que la persona lo haya desactivado expresamente con codemode off, elección que se recuerda y se respeta


## Contexto y evidencia
- Petición de la persona (2026-11-17): «deja pi gomemory en modo default "pi-gomemory-setup codemode on"». Aclarada: install/update lo activan por defecto y se respeta un `codemode off` explícito.
- Antecedente: HCD-043 (memoria 423) creó `codemode on/off` e hizo que `install`/`mcp` conserven el modo activo (`InstallMCPServers`, `pkg/installer/mcp.go`). Hoy un estado «off» no distingue «nunca activado» de «desactivado a propósito».
- Defecto latente que este cambio activaría (regla 7): con codemode activo, `uninstall --dry-run` marca `[omitir] mcpServers.gomemory` y `mcpServers.codebase-memory` («no coincide con la definición de pi-gomemory»), porque `planMCP` (`pkg/installer/uninstall.go`) solo compara con `DefaultMCPServers()`. Tampoco retira `+codemode` de `defaultTools`. Si codemode pasa a ser el modo por defecto, toda instalación queda con restos tras `uninstall`.

## Reproducción
```
H=$(mktemp -d); HOME=$H go run . install; HOME=$H go run . codemode status
→ codemode: off (settings=false, mcp=false)
HOME=$H go run . codemode on; HOME=$H go run . uninstall --dry-run
→ [omitir] …mcp.json › mcpServers.gomemory — no coincide con la definición de pi-gomemory
→ [omitir] …mcp.json › mcpServers.codebase-memory — no coincide con la definición de pi-gomemory
```

## Alcance
Incluido: preferencia persistente de codemode en pi-gomemory; install/update activan codemode salvo un off explícito; uninstall retira el perfil híbrido y el `+codemode` que añadió pi-gomemory; README y tests.
- `codemode on` y `codemode off` guardan la preferencia en `~/.pi-gomemory/codemode.json`. `codemode off` la deja en `off`; `codemode on` la deja en `on`.
- `install` (y por tanto `update`) aplica el perfil híbrido completo (`+codemode` en `settings.json` y exposición híbrida en `mcp.json`) salvo que la preferencia sea `off`; en ese caso deja el modo directo. Una instalación que ya estaba en codemode sigue en codemode.
- Propiedad de `+codemode` (como el manifiesto de HCD-011): el estado registra si fue pi-gomemory quien añadió `+codemode` a `defaultTools`. Si la persona ya lo tenía, no se registra como propio.
- `uninstall` retira las entradas MCP de pi-gomemory tanto en perfil directo como híbrido, retira `+codemode` solo si lo añadió pi-gomemory y retira `~/.pi-gomemory/codemode.json`.
- README y nota del CLI: el modo por defecto es codemode híbrido; `codemode off` lo desactiva de forma persistente.
Excluido: cambiar la lista de tools directas del perfil híbrido (HCD-044); cambiar `codemode.mode` de Pi (`only`); el subcomando `mcp`, que sigue conservando el modo vigente.

## ¿Qué pasa si no lo hacemos?
Cada instalación nueva arranca en modo directo, con las 54 tools MCP declaradas en todos los prompts, y la persona tiene que acordarse de ejecutar `pi-gomemory-setup codemode on`. Si además lo activa, `uninstall` deja restos de pi-gomemory en `mcp.json` y `settings.json`.

## Alternativa más barata
Llamar a `SetCodemode(true)` en `install` sin preferencia persistente. Es más barato, pero pisa un `codemode off` explícito en cada `update` y deja intacto el defecto de `uninstall`. Se descarta.

## Aclaraciones
- [x] Q2. ¿Qué significa «modo default codemode on»? Opciones: install lo activa por defecto respetando codemode off [recomendada] | solo el entorno de la persona | forzar siempre on → Install lo activa por defecto (Recommended)

## Checklist
- [x] 1. pkg/installer/codemode.go: `codemode on/off` guardan la preferencia en ~/.pi-gomemory/codemode.json y registran si pi-gomemory añadió `+codemode` → `go test ./pkg/installer -run 'TestHCD046Preference' -count=1`
- [x] 2. pkg/installer/installer.go: install aplica codemode híbrido salvo preferencia off, y conserva un off explícito → `go test ./pkg/installer -run 'TestHCD046InstallDefault' -count=1`
- [x] 3. pkg/installer/uninstall.go: uninstall retira las entradas MCP en perfil híbrido, el `+codemode` propio (no el ajeno) y el estado de codemode → `go test ./pkg/installer -run 'TestHCD046Uninstall' -count=1`
- [x] 4. La reproducción ya no falla: instalación nueva en codemode on y uninstall sin omitir entradas de pi-gomemory → `bash -c 'H=$(mktemp -d); HOME=$H go run . install >/dev/null 2>&1; HOME=$H go run . codemode status | grep -q "codemode: on" && ! HOME=$H go run . uninstall --dry-run 2>&1 | grep -q "no coincide con la definición de pi-gomemory"'`
- [x] 5. README.md y la nota del CLI documentan el modo por defecto; vet y suite completa en verde → `bash -c 'grep -q "por defecto" README.md && go vet ./... && go test ./... -count=1'`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-17 14:38 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD046Preference' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd046_test.go:12:45: undefined: codemodeState
      pkg/installer/hcd046_test.go:14:13: undefined: readCodemodeState
      pkg/installer/hcd046_test.go:37:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:43:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:56:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:82:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:86:13: undefined: applyCodemodeDefault
      pkg/installer/hcd046_test.go:113:4: undefined: CodemodeStatePath
      pkg/installer/hcd046_test.go:126:24: undefined: CodemodeStatePath
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 14:38 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD046InstallDefault' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd046_test.go:12:45: undefined: codemodeState
      pkg/installer/hcd046_test.go:14:13: undefined: readCodemodeState
      pkg/installer/hcd046_test.go:37:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:43:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:56:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:82:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:86:13: undefined: applyCodemodeDefault
      pkg/installer/hcd046_test.go:113:4: undefined: CodemodeStatePath
      pkg/installer/hcd046_test.go:126:24: undefined: CodemodeStatePath
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 14:38 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD046Uninstall' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd046_test.go:12:45: undefined: codemodeState
      pkg/installer/hcd046_test.go:14:13: undefined: readCodemodeState
      pkg/installer/hcd046_test.go:37:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:43:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:56:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:82:13: undefined: SetCodemodePreference
      pkg/installer/hcd046_test.go:86:13: undefined: applyCodemodeDefault
      pkg/installer/hcd046_test.go:113:4: undefined: CodemodeStatePath
      pkg/installer/hcd046_test.go:126:24: undefined: CodemodeStatePath
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 14:38 -05:00 · tarea 4 · rojo observado · caducada · `bash -c 'H=$(mktemp -d); HOME=$H go run . install >/dev/null 2>&1; HOME=$H go run . codemode status | grep -q "codemode: on" && ! HOME=$H go run . uninstall --dry-run 2>&1 | grep -q "no coincide con la definición de pi-gomemory"'` (salida 1)
- 2026-11-17 14:40 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run 'TestHCD046Preference' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.201s
- 2026-11-17 14:40 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run 'TestHCD046InstallDefault' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.528s
- 2026-11-17 14:40 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run 'TestHCD046Uninstall' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.730s
- 2026-11-17 14:40 -05:00 · tarea 4 · verde · caducada · `bash -c 'H=$(mktemp -d); HOME=$H go run . install >/dev/null 2>&1; HOME=$H go run . codemode status | grep -q "codemode: on" && ! HOME=$H go run . uninstall --dry-run 2>&1 | grep -q "no coincide con la definición de pi-gomemory"'` (salida 0)
- 2026-11-17 14:40 -05:00 · tarea 5 · verde · caducada · `bash -c 'grep -q "por defecto" README.md && go vet ./... && go test ./... -count=1'` (salida 0)
      ok  	pi-gomemory	0.221s
      ok  	pi-gomemory/pkg/installer	73.249s
- 2026-11-17 14:41 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run 'TestHCD046Preference' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.395s
- 2026-11-17 14:41 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run 'TestHCD046InstallDefault' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.420s
- 2026-11-17 14:42 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run 'TestHCD046Uninstall' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.757s
- 2026-11-17 14:42 -05:00 · tarea 4 · verde · vigente · `bash -c 'H=$(mktemp -d); HOME=$H go run . install >/dev/null 2>&1; HOME=$H go run . codemode status | grep -q "codemode: on" && ! HOME=$H go run . uninstall --dry-run 2>&1 | grep -q "no coincide con la definición de pi-gomemory"'` (salida 0)
- 2026-11-17 14:42 -05:00 · tarea 5 · verde · vigente · `bash -c 'grep -q "por defecto" README.md && go vet ./... && go test ./... -count=1'` (salida 0)
      ok  	pi-gomemory	0.215s
      ok  	pi-gomemory/pkg/installer	71.473s

## Siguiente
Cerrada con aprobación sobre la revisión de3b31f105f0.

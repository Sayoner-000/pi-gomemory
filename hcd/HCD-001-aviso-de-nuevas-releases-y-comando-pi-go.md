# Aviso de nuevas releases y comando pi-gomemory-setup update

ID: HCD-001
Tipo: feature
Carril: fast
Motivo: Decisiones cerradas con la persona (canal launcher + notificar), sin datos sensibles ni contratos rotos

## Intención
El launcher pi-gomemory consulta GitHub Releases (máx. 1 vez/24h, caché, timeout corto, sin bloquear) y avisa si hay versión nueva; nuevo subcomando 'pi-gomemory-setup update' descarga el release, verifica checksum y reinstala


## Alcance
Incluido: versión inyectada, `check-update` con caché de 24 h, aviso desde el launcher, `update` con verificación SHA-256, pruebas y documentación.
- Versión inyectada en el binario con `-ldflags -X main.version=…` (Makefile y `scripts/build-release.sh`); `pi-gomemory-setup version` la imprime.
- `pi-gomemory-setup check-update`: consulta `GET https://api.github.com/repos/Sayoner-000/pi-gomemory/releases/latest` con un tiempo de espera de 2 s, guarda `latest_tag` y `checked_at` en `~/.pi/agent/pi-gomemory-update.json` y compara las versiones con semver. Si han pasado menos de 24 h desde la última consulta, usa la caché. Si hay una versión más nueva, imprime el aviso con el comando `pi-gomemory-setup update`. Ante un fallo de red, no imprime nada y sale con código 0.
- El launcher `pi-gomemory` (harness.go) ejecuta `check-update` al arrancar sin bloquear el inicio de Pi.
- `pi-gomemory-setup update`: descarga el tarball de la última release para el sistema y la arquitectura, verifica el SHA-256 contra el `SHA256SUMS` de la release (lo genera `scripts/build-release.sh`), reemplaza el binario y vuelve a ejecutar `install`. Si la versión instalada ya es la última, lo indica y termina.
- Variable `PI_GOMEMORY_NO_UPDATE_CHECK=1` para desactivar la comprobación.
- Pruebas Go con un servidor HTTP falso para la comparación de versiones, la caché de 24 h, el fallo de red silencioso y un checksum incorrecto (rechazo).
- Documentación de los comandos nuevos en el README y en `docs/installation.md`.

Excluido: autoinstalación, avisos dentro de la interfaz de Pi, actualización de otras herramientas y firma de releases.
- Instalación automática sin intervención del usuario.
- Notificaciones dentro de la interfaz de Pi (extensión TS), demonios, cron o launchd.
- Actualizar gomemory (`mem`), Pi u otros agentes.
- Firma criptográfica de las releases más allá del SHA-256, y la vuelta atrás a versiones anteriores.

## ¿Qué pasa si no lo hacemos?
Los usuarios se quedan en la versión que instalaron. Solo se enteran de una release nueva si revisan GitHub a mano, y para actualizar tienen que volver a ejecutar `install.sh` con `PI_GOMEMORY_VERSION`. Las correcciones (por ejemplo, D1–D8 del instalador) no llegan a quien ya instaló.

## Alternativa más barata
Dejar que `install.sh` resuelva la última versión con `releases/latest` en lugar de usar `v1.2.0` fijo, y documentar «vuelve a ejecutar install.sh». Eso es más barato, pero no avisa al usuario, que es el objetivo pedido. Se descarta como solución completa, aunque se incorpora el mismo criterio (resolver `latest`) dentro del comando `update`.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Inyectar la versión con `-ldflags -X main.version` en Makefile y build-release.sh y añadir el subcomando `version` → `go build -ldflags "-X main.version=v9.9.9" -o /tmp/pgs . && /tmp/pgs version | grep -q v9.9.9`
- [x] 2. Implementar la comparación semver de tags (`IsNewer`) en `pkg/installer/update.go` → `go test ./pkg/installer -run TestIsNewer`
- [x] 3. Implementar `CheckUpdate` con caché de 24 h en `pi-gomemory-update.json`, timeout de 2 s y fallo de red silencioso → `go test ./pkg/installer -run TestCheckUpdate`
- [x] 4. Añadir el subcomando `check-update` en main.go, respetando `PI_GOMEMORY_NO_UPDATE_CHECK=1` → `go test ./pkg/installer -run TestCheckUpdateDisabled`
- [x] 5. Implementar `Update`: descarga del tarball, verificación contra `SHA256SUMS` (rechaza checksum incorrecto) y reemplazo del binario → `go test ./pkg/installer -run TestUpdate`
- [x] 6. Añadir el subcomando `update` en main.go que reejecuta `install` tras reemplazar el binario → `go build ./... && go vet ./...`
- [x] 7. Llamar a `pi-gomemory-setup check-update` desde el launcher sin bloquear el arranque → `go test ./pkg/installer -run TestHarnessCallsCheckUpdate`
- [x] 8. Documentar `version`, `check-update`, `update` y la variable de desactivación en README y docs/installation.md → `grep -q "check-update" README.md && grep -q "PI_GOMEMORY_NO_UPDATE_CHECK" docs/installation.md`
- [x] 9. Ejecutar la suite completa → `go test ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-10 21:47 -05:00 · tarea 1 · rojo observado · caducada · `go build -ldflags "-X main.version=v9.9.9" -o /tmp/pgs . && /tmp/pgs version | grep -q v9.9.9` (salida 1)
      Comando desconocido: version
- 2026-11-10 21:47 -05:00 · tarea 1 · verde · caducada · `go build -ldflags "-X main.version=v9.9.9" -o /tmp/pgs . && /tmp/pgs version | grep -q v9.9.9` (salida 0)
- 2026-11-10 21:48 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestIsNewer` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/update_test.go:37:13: undefined: IsNewer
      pkg/installer/update_test.go:54:11: undefined: updateCheckEnv
      pkg/installer/update_test.go:57:9: undefined: UpdateCheckConfig
      pkg/installer/update_test.go:59:41: undefined: updateCacheFile
      pkg/installer/update_test.go:63:9: undefined: CheckUpdate
      pkg/installer/update_test.go:72:12: undefined: CheckUpdate
      pkg/installer/update_test.go:80:2: undefined: CheckUpdate
      pkg/installer/update_test.go:90:11: undefined: updateCheckEnv
      pkg/installer/update_test.go:91:9: undefined: CheckUpdate
      pkg/installer/update_test.go:91:31: undefined: UpdateCheckConfig
      pkg/installer/update_test.go:91:31: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 21:48 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestCheckUpdate` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/update_test.go:37:13: undefined: IsNewer
      pkg/installer/update_test.go:54:11: undefined: updateCheckEnv
      pkg/installer/update_test.go:57:9: undefined: UpdateCheckConfig
      pkg/installer/update_test.go:59:41: undefined: updateCacheFile
      pkg/installer/update_test.go:63:9: undefined: CheckUpdate
      pkg/installer/update_test.go:72:12: undefined: CheckUpdate
      pkg/installer/update_test.go:80:2: undefined: CheckUpdate
      pkg/installer/update_test.go:90:11: undefined: updateCheckEnv
      pkg/installer/update_test.go:91:9: undefined: CheckUpdate
      pkg/installer/update_test.go:91:31: undefined: UpdateCheckConfig
      pkg/installer/update_test.go:91:31: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 21:48 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run TestCheckUpdateDisabled` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/update_test.go:37:13: undefined: IsNewer
      pkg/installer/update_test.go:54:11: undefined: updateCheckEnv
      pkg/installer/update_test.go:57:9: undefined: UpdateCheckConfig
      pkg/installer/update_test.go:59:41: undefined: updateCacheFile
      pkg/installer/update_test.go:63:9: undefined: CheckUpdate
      pkg/installer/update_test.go:72:12: undefined: CheckUpdate
      pkg/installer/update_test.go:80:2: undefined: CheckUpdate
      pkg/installer/update_test.go:90:11: undefined: updateCheckEnv
      pkg/installer/update_test.go:91:9: undefined: CheckUpdate
      pkg/installer/update_test.go:91:31: undefined: UpdateCheckConfig
      pkg/installer/update_test.go:91:31: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 21:48 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run TestUpdate` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/update_test.go:37:13: undefined: IsNewer
      pkg/installer/update_test.go:54:11: undefined: updateCheckEnv
      pkg/installer/update_test.go:57:9: undefined: UpdateCheckConfig
      pkg/installer/update_test.go:59:41: undefined: updateCacheFile
      pkg/installer/update_test.go:63:9: undefined: CheckUpdate
      pkg/installer/update_test.go:72:12: undefined: CheckUpdate
      pkg/installer/update_test.go:80:2: undefined: CheckUpdate
      pkg/installer/update_test.go:90:11: undefined: updateCheckEnv
      pkg/installer/update_test.go:91:9: undefined: CheckUpdate
      pkg/installer/update_test.go:91:31: undefined: UpdateCheckConfig
      pkg/installer/update_test.go:91:31: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 21:48 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run TestHarnessCallsCheckUpdate` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/update_test.go:37:13: undefined: IsNewer
      pkg/installer/update_test.go:54:11: undefined: updateCheckEnv
      pkg/installer/update_test.go:57:9: undefined: UpdateCheckConfig
      pkg/installer/update_test.go:59:41: undefined: updateCacheFile
      pkg/installer/update_test.go:63:9: undefined: CheckUpdate
      pkg/installer/update_test.go:72:12: undefined: CheckUpdate
      pkg/installer/update_test.go:80:2: undefined: CheckUpdate
      pkg/installer/update_test.go:90:11: undefined: updateCheckEnv
      pkg/installer/update_test.go:91:9: undefined: CheckUpdate
      pkg/installer/update_test.go:91:31: undefined: UpdateCheckConfig
      pkg/installer/update_test.go:91:31: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 21:49 -05:00 · tarea 1 · verde · vigente · `go build -ldflags "-X main.version=v9.9.9" -o /tmp/pgs . && /tmp/pgs version | grep -q v9.9.9` (salida 0)
- 2026-11-10 21:49 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestIsNewer` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.202s
- 2026-11-10 21:49 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestCheckUpdate` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.508s
- 2026-11-10 21:49 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestCheckUpdateDisabled` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.584s
- 2026-11-10 21:49 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run TestUpdate` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.240s
- 2026-11-10 21:49 -05:00 · tarea 6 · verde · vigente · `go build ./... && go vet ./...` (salida 0)
- 2026-11-10 21:49 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run TestHarnessCallsCheckUpdate` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.229s
- 2026-11-10 21:49 -05:00 · tarea 8 · verde · vigente · `grep -q "check-update" README.md && grep -q "PI_GOMEMORY_NO_UPDATE_CHECK" docs/installation.md` (salida 0)
- 2026-11-10 21:49 -05:00 · tarea 9 · verde · vigente · `go test ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	0.565s

## Siguiente
Cerrada con aprobación sobre la revisión b2a6444e78f5.

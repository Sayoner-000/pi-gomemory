# Publicar v1.12.0 con cuota Codex y documentación actualizada

ID: HCD-028
Tipo: technical
Carril: fast
Motivo: Entrega de cambios existentes HCD-026: referencias, documentación y artefactos reproducibles, sin nuevas decisiones de comportamiento

## Intención
Validar y publicar los cambios realizados de cuota Codex como v1.12.0: commits, push a origin/main, tag anotado, release GitHub con cuatro binarios y SHA256SUMS, referencias y documentación vigente actualizadas. Mantener HCD-027 pendiente sin presentar sus hallazgos como corregidos. No añadir correcciones de comportamiento ni leer credenciales reales.


## Alcance
Incluido: validar el cambio HCD-026 existente; documentar codex-usage y la cuota Codex; actualizar README, docs vigentes, guía HTML, changelog, versiones del instalador y empaquetador; compilar macOS/Linux en amd64/arm64 con SHA256SUMS; crear commits separados, publicar origin/main, tag v1.12.0 y release GitHub; actualizar el binario local de desarrollo y verificar los artefactos descargados.
Excluido: implementar o cerrar HCD-027, corregir comportamiento nuevo, cambiar dependencias, modificar informes históricos y ADR sin relación con esta versión, publicar secretos, reescribir historial y ejecutar instalación completa sobre HOME real.

## ¿Qué pasa si no lo hacemos?
Los cambios de cuota Codex quedan solo en el árbol local; la documentación y los binarios descargables continúan describiendo v1.11.1 y las personas no pueden instalar esta funcionalidad desde la release.

## Alternativa más barata
Publicar solo el código requiere menos trabajo, pero no satisface la petición de binarios y documentación sincronizados. Se reutiliza el empaquetador existente sin modificar su lógica.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Base normativa
Constitución §10 (validación de comportamiento), §15 (integridad de artefactos y mínimo privilegio), §16 (commits), §18 (documentación) y §21 (simplicidad y evidencia); Playbook HCD-TASK-01, HCD-EVIDENCE-01 y HCD-CLOSE-01. No se considera que publicar una versión cierre defectos pendientes ni que autorice automáticamente el cierre HCD.

## Checklist
- [x] 1. Validar tests, detector de carreras, vet y sintaxis de scripts → `go test -race ./... -count=1 && go vet ./... && bash -n install.sh scripts/build-release.sh && git diff --check`
- [x] 2. Comprobar referencias y documentación de v1.12.0 → inspección de cambios y referencias vigentes, preservando historial y estado pendiente de HCD-027
- [x] 3. Construir y comprobar los cuatro paquetes de distribución y el binario local → `bash scripts/build-release.sh v1.12.0 && (cd dist && shasum -a 256 -c SHA256SUMS) && make build VERSION=v1.12.0 && bin/pi-gomemory-setup version`
- [x] 4. Publicar commits, tag y release y verificar descargas → `gh release view v1.12.0 --json tagName,isDraft,isPrerelease,url,assets` más verificación de hashes de los archivos descargados y tag remoto

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-12 06:03 -05:00 · tarea 1 · verde · caducada · `go test -race ./... -count=1 && go vet ./... && bash -n install.sh scripts/build-release.sh && git diff --check` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	17.164s
- 2026-11-12 06:04 -05:00 · tarea 2 · manual · caducada
      Inspeccionados README, changelog y documentación vigente: codex-usage devuelve porcentaje usado y sidebar restante; eventos forzados y heartbeat distinguidos del TTL; referencias actuales v1.12.0; guía HTML v1.12 nueva conserva v1.11 histórica. Notas de release y known-issues declaran HCD-027 no implementada y flaky pendiente. ADR y auditorías históricas sin reescritura. Diff y archivos nuevos revisados sin credenciales reales observadas; pruebas usan tokens sintéticos. No hubo escáner automático de secretos disponible.
- 2026-11-12 06:04 -05:00 · tarea 3 · verde · caducada · `bash scripts/build-release.sh v1.12.0 && (cd dist && shasum -a 256 -c SHA256SUMS) && make build VERSION=v1.12.0 && bin/pi-gomemory-setup version` (salida 0)
        -> Compilando darwin/amd64...
        -> Empaquetando pi-gomemory-setup_v1.12.0_darwin_amd64.tar.gz...
        -> Compilando linux/amd64...
        -> Empaquetando pi-gomemory-setup_v1.12.0_linux_amd64.tar.gz...
        -> Compilando linux/arm64...
        -> Empaquetando pi-gomemory-setup_v1.12.0_linux_arm64.tar.gz...
      ==> Calculando sumas de verificación SHA256...
      ✓ Pack de distribución generado exitosamente en dist/:
      total 24304
      -rw-r--r--@ 1 josegomezj  staff   3.1M Nov 12 06:04 pi-gomemory-setup_v1.12.0_darwin_amd64.tar.gz
      -rw-r--r--@ 1 josegomezj  staff   2.9M Nov 12 06:04 pi-gomemory-setup_v1.12.0_darwin_arm64.tar.gz
      -rw-r--r--@ 1 josegomezj  staff   3.1M Nov 12 06:04 pi-gomemory-setup_v1.12.0_linux_amd64.tar.gz
      -rw-r--r--@ 1 josegomezj  staff   2.8M Nov 12 06:04 pi-gomemory-setup_v1.12.0_linux_arm64.tar.gz
      -rw-r--r--@ 1 josegomezj  staff   446B Nov 12 06:04 SHA256SUMS
      pi-gomemory-setup_v1.12.0_darwin_amd64.tar.gz: OK
      pi-gomemory-setup_v1.12.0_darwin_arm64.tar.gz: OK
      pi-gomemory-setup_v1.12.0_linux_amd64.tar.gz: OK
      pi-gomemory-setup_v1.12.0_linux_arm64.tar.gz: OK
      go build -ldflags="-s -w -X main.version=v1.12.0" -o bin/pi-gomemory-setup .
      v1.12.0
- 2026-11-12 06:08 -05:00 · tarea 1,3,4 · verde · vigente · `go test -race ./... -count=1 && go vet ./... && bash -n install.sh scripts/build-release.sh && git diff --check && (cd dist && shasum -a 256 -c SHA256SUMS) && (cd /tmp/pi-release-download-uGldYd && shasum -a 256 -c SHA256SUMS) && cmp dist/SHA256SUMS /tmp/pi-release-download-uGldYd/SHA256SUMS && test "$(bin/pi-gomemory-setup version)" = v1.12.0 && test "$(/tmp/pi-release-download-uGldYd/host/pi-gomemory-setup version)" = v1.12.0 && go version -m /tmp/pi-release-download-uGldYd/host/pi-gomemory-setup && gh release view v1.12.0 --json tagName,isDraft,isPrerelease,url,assets && git ls-remote origin refs/tags/v1.12.0 "refs/tags/v1.12.0^{}"` (salida 0)
      state":"uploaded","updatedAt":"2026-11-12T11:06:06Z","url":"https://github.com/Sayoner-000/pi-gomemory/releases/download/v1.12.0/pi-gomemory-setup_v1.12.0_darwin_arm64.tar.gz"},{"apiUrl":"https://api.github.com/repos/Sayoner-000/pi-gomemory/releases/assets/739158304","contentType":"application/x-gtar","createdAt":"2026-11-12T11:05:59Z","digest":"sha256:386b0877775c120d0f8298bf170faeeb6ae6117e073bb05123afe51edd32093c","downloadCount":0,"id":"RA_kwDOVo76P84sDqkg","label":"","name":"pi-gomemory-setup_v1.12.0_linux_amd64.tar.gz","size":3236466,"state":"uploaded","updatedAt":"2026-11-12T11:06:06Z","url":"https://github.com/Sayoner-000/pi-gomemory/releases/download/v1.12.0/pi-gomemory-setup_v1.12.0_linux_amd64.tar.gz"},{"apiUrl":"https://api.github.com/repos/Sayoner-000/pi-gomemory/releases/assets/739158300","contentType":"application/x-gtar","createdAt":"2026-11-12T11:05:59Z","digest":"sha256:c16d9a5c822805fda75dbf74929c44912e7486c7506d818dce4f7fe65364ddc0","downloadCount":0,"id":"RA_kwDOVo76P84sDqkc","label":"","name":"pi-gomemory-setup_v1.12.0_linux_arm64.tar.gz","size":2911059,"state":"uploaded","updatedAt":"2026-11-12T11:06:06Z","url":"https://github.com/Sayoner-000/pi-gomemory/releases/download/v1.12.0/pi-gomemory-setup_v1.12.0_linux_arm64.tar.gz"},{"apiUrl":"https://api.github.com/repos/Sayoner-000/pi-gomemory/releases/assets/739158287","contentType":"application/octet-stream","createdAt":"2026-11-12T11:05:59Z","digest":"sha256:df3feacc1287d0bdbc8623a8a3a4f4c3c8f6f32d52ecf4cfc1fea900bcfefea6","downloadCount":0,"id":"RA_kwDOVo76P84sDqkP","label":"","name":"SHA256SUMS","size":446,"state":"uploaded","updatedAt":"2026-11-12T11:05:59Z","url":"https://github.com/Sayoner-000/pi-gomemory/releases/download/v1.12.0/SHA256SUMS"}],"isDraft":false,"isPrerelease":false,"tagName":"v1.12.0","url":"https://github.com/Sayoner-000/pi-gomemory/releases/tag/v1.12.0"}
      9c1e7ac5c61a50ccac42abaed204d851ecfb6b71	refs/tags/v1.12.0
      fae830c655f5189682a5af761b2007716da16110	refs/tags/v1.12.0^{}
- 2026-11-12 06:08 -05:00 · tarea 2 · manual · vigente
      Revisión final en tag v1.12.0/fae830c: README, changelog, docs vigentes, guía HTML v1.12 y referencias de descarga sincronizados. Notas GitHub enlazan a archivos del tag con URL absoluta. HCD-027 y flaky señalados explícitamente como pendientes. ADR, auditorías y guía v1.11 conservados como históricos. Datos de pruebas sintéticos y ningún secreto real observado en el cambio; sin escáner automático. Binario instalado actualizado atómicamente desde descarga validada, v1.12.0; sidebar instalado coincide con el código, sin reinstalación completa.

## Siguiente
Consultar el motor; ejecutar las comprobaciones y registrar evidencia sin editarla manualmente. Al terminar, solicitar aprobación de cierre de la revisión de trabajo exacta; la autorización de publicar no se convierte en una cita de cierre.

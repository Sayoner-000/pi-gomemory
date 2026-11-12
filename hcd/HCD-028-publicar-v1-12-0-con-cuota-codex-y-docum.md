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
- [ ] 4. Publicar commits, tag y release y verificar descargas → `gh release view v1.12.0 --json tagName,isDraft,isPrerelease,url,assets` más verificación de hashes de los archivos descargados y tag remoto

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-12 06:03 -05:00 · tarea 1 · verde · vigente · `go test -race ./... -count=1 && go vet ./... && bash -n install.sh scripts/build-release.sh && git diff --check` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	17.164s
- 2026-11-12 06:04 -05:00 · tarea 2 · manual · vigente
      Inspeccionados README, changelog y documentación vigente: codex-usage devuelve porcentaje usado y sidebar restante; eventos forzados y heartbeat distinguidos del TTL; referencias actuales v1.12.0; guía HTML v1.12 nueva conserva v1.11 histórica. Notas de release y known-issues declaran HCD-027 no implementada y flaky pendiente. ADR y auditorías históricas sin reescritura. Diff y archivos nuevos revisados sin credenciales reales observadas; pruebas usan tokens sintéticos. No hubo escáner automático de secretos disponible.
- 2026-11-12 06:04 -05:00 · tarea 3 · verde · vigente · `bash scripts/build-release.sh v1.12.0 && (cd dist && shasum -a 256 -c SHA256SUMS) && make build VERSION=v1.12.0 && bin/pi-gomemory-setup version` (salida 0)
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

## Siguiente
Consultar el motor; ejecutar las comprobaciones y registrar evidencia sin editarla manualmente. Al terminar, solicitar aprobación de cierre de la revisión de trabajo exacta; la autorización de publicar no se convierte en una cita de cierre.

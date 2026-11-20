# Publicar v1.15.3 con regresión de Antigravity y documentación de la mitigación local

ID: HCD-049
Tipo: technical
Carril: fast
Motivo: Publicación solicitada explícitamente: cambios de documentación, metadatos y artefactos sin modificar el comportamiento del instalador

## Intención
Publicar los ajustes aprobados de HCD-040 en v1.15.3 con documentación coherente, commits y tag en origin, cuatro binarios y SHA256 verificadas; el guard continúa siendo una mitigación local opt-in, no un recurso instalado por defecto.


## Alcance
Incluido: publicar la regresión y referencia opt-in aprobadas de HCD-040; actualizar README, documentación vigente, badge, CHANGELOG y versiones predeterminadas de los scripts a v1.15.3; validar, crear commits, subir main y tag, empaquetar darwin/linux × amd64/arm64 y publicar/verificar los cuatro archivos y SHA256SUMS en GitHub.
Excluido: instalar automáticamente el guard, embeberlo en el instalador, modificar credenciales o el hardening, actualizar dependencias o pins, reescribir notas históricas y afirmar que la actualización del binario por sí sola corrige todos los HTTP 429.

## ¿Qué pasa si no lo hacemos?
La instalación local recuperada seguirá funcionando, pero la regresión no quedará publicada y la documentación vigente no explicará el fallo, el riesgo del guard ni su reversión. Los usuarios no tendrán los binarios y referencias solicitados.

## Alternativa más barata
Crear solo un commit con la regresión y una nota breve cuesta menos, pero no entrega el tag, los paquetes descargables y la documentación actualizada que la persona pidió. Se mantiene la publicación completa sin ampliar el comportamiento del instalador.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Actualizar las referencias vigentes de README.md, CHANGELOG.md, docs/, install.sh y scripts/build-release.sh para v1.15.3 sin presentar el guard como instalado por defecto → node /tmp/pi-gomemory-v1.15.3-checks/check-docs.mjs
- [x] 2. Validar el código fuente de v1.15.3 y la regresión pkg/installer/hcd040_prompt_guard_test.go → go build ./... && go vet ./... && go vet -tags pins ./... && go test -race ./... -count=1
- [x] 3. Validar los cuatro paquetes dist/ de v1.15.3 generados por scripts/build-release.sh, sus SHA256, plataformas y revisión Git → node /tmp/pi-gomemory-v1.15.3-checks/check-artifacts.mjs
- [x] 4. Verificar los commits, tag y cinco assets publicados de v1.15.3 en origin → node /tmp/pi-gomemory-v1.15.3-checks/check-remote.mjs

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-19 05:50 -05:00 · tarea 1 · verde · caducada · `node /tmp/pi-gomemory-v1.15.3-checks/check-docs.mjs` (salida 0)
      OK: 12 current documentation files; 143 local references; versions, scope and shell syntax checked
- 2026-11-19 05:50 -05:00 · tarea 2 · verde · caducada · `go build ./... && go vet ./... && go vet -tags pins ./... && go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.856s
      ok  	pi-gomemory/pkg/installer	104.327s
- 2026-11-19 06:03 -05:00 · tarea 3,4 · verde · caducada · `node /tmp/pi-gomemory-v1.15.3-checks/check-artifacts.mjs && node /tmp/pi-gomemory-v1.15.3-checks/check-remote.mjs` (salida 0)
      OK darwin_amd64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK native version: v1.15.3
      OK darwin_arm64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK linux_amd64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK linux_arm64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK: four release archives match tag v1.15.3 (092a857361d3109ba0e27ae76d796e3d3b915c0d)
      OK downloaded pi-gomemory-setup_v1.15.3_darwin_amd64.tar.gz: 0634df41493cfffa1f0da8e983f6436bfd8732705e09ed8821767cbd31489fef
      OK downloaded pi-gomemory-setup_v1.15.3_darwin_arm64.tar.gz: 982c78c40c2056063832564b78ed0e508426c3c2f68290d10b419690946eb4ad
      OK downloaded pi-gomemory-setup_v1.15.3_linux_amd64.tar.gz: d4e9bbaa898b69ed5da39082fd6c408f6f94a213f0cbbc182aa5edcce94f4654
      OK downloaded pi-gomemory-setup_v1.15.3_linux_arm64.tar.gz: ecb6d51a2bda5cc527a3e5f672292a2571c6b62e2bc38ddacb6fa027eb835beb
      OK: origin main 092a857361d3109ba0e27ae76d796e3d3b915c0d, annotated tag 092a857361d3109ba0e27ae76d796e3d3b915c0d, latest public release and all five downloaded assets
      https://github.com/Sayoner-000/pi-gomemory/releases/tag/v1.15.3
- 2026-11-19 06:05 -05:00 · tarea 1 · verde · vigente · `node /tmp/pi-gomemory-v1.15.3-checks/check-docs.mjs` (salida 0)
      OK: 12 current documentation files; 143 local references; versions, scope and shell syntax checked
- 2026-11-19 06:05 -05:00 · tarea 2 · verde · vigente · `go build ./... && go vet ./... && go vet -tags pins ./... && go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.336s
      ok  	pi-gomemory/pkg/installer	91.747s
- 2026-11-19 06:07 -05:00 · tarea 3 · verde · vigente · `node /tmp/pi-gomemory-v1.15.3-checks/check-artifacts.mjs` (salida 0)
      OK darwin_amd64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK native version: v1.15.3
      OK darwin_arm64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK linux_amd64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK linux_arm64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK: four release archives match tag v1.15.3 (092a857361d3109ba0e27ae76d796e3d3b915c0d)
- 2026-11-19 06:07 -05:00 · tarea 4 · verde · vigente · `node /tmp/pi-gomemory-v1.15.3-checks/check-remote.mjs` (salida 0)
      OK downloaded pi-gomemory-setup_v1.15.3_darwin_amd64.tar.gz: 0634df41493cfffa1f0da8e983f6436bfd8732705e09ed8821767cbd31489fef
      OK downloaded pi-gomemory-setup_v1.15.3_darwin_arm64.tar.gz: 982c78c40c2056063832564b78ed0e508426c3c2f68290d10b419690946eb4ad
      OK downloaded pi-gomemory-setup_v1.15.3_linux_amd64.tar.gz: d4e9bbaa898b69ed5da39082fd6c408f6f94a213f0cbbc182aa5edcce94f4654
      OK downloaded pi-gomemory-setup_v1.15.3_linux_arm64.tar.gz: ecb6d51a2bda5cc527a3e5f672292a2571c6b62e2bc38ddacb6fa027eb835beb
      OK: origin main 2a02bde2c5c83183d6e49f235fcb57509687b97b, annotated tag 092a857361d3109ba0e27ae76d796e3d3b915c0d, latest public release and all five downloaded assets
      https://github.com/Sayoner-000/pi-gomemory/releases/tag/v1.15.3
- 2026-11-19 06:09 -05:00 · tarea 1 · verde · vigente · `node /tmp/pi-gomemory-v1.15.3-checks/check-docs.mjs` (salida 0)
      OK: 12 current documentation files; 143 local references; versions, scope and shell syntax checked
- 2026-11-19 06:09 -05:00 · tarea 2 · verde · vigente · `go build ./... && go vet ./... && go vet -tags pins ./... && go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.454s
      ok  	pi-gomemory/pkg/installer	91.687s
- 2026-11-19 06:10 -05:00 · tarea 3 · verde · vigente · `node /tmp/pi-gomemory-v1.15.3-checks/check-artifacts.mjs` (salida 0)
      OK darwin_amd64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK native version: v1.15.3
      OK darwin_arm64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK linux_amd64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK linux_arm64: SHA256, four regular entries, tagged documents, static Go metadata and clean source 092a857361d3
      OK: four release archives match tag v1.15.3 (092a857361d3109ba0e27ae76d796e3d3b915c0d)
- 2026-11-19 06:10 -05:00 · tarea 4 · verde · vigente · `node /tmp/pi-gomemory-v1.15.3-checks/check-remote.mjs` (salida 0)
      OK downloaded pi-gomemory-setup_v1.15.3_darwin_amd64.tar.gz: 0634df41493cfffa1f0da8e983f6436bfd8732705e09ed8821767cbd31489fef
      OK downloaded pi-gomemory-setup_v1.15.3_darwin_arm64.tar.gz: 982c78c40c2056063832564b78ed0e508426c3c2f68290d10b419690946eb4ad
      OK downloaded pi-gomemory-setup_v1.15.3_linux_amd64.tar.gz: d4e9bbaa898b69ed5da39082fd6c408f6f94a213f0cbbc182aa5edcce94f4654
      OK downloaded pi-gomemory-setup_v1.15.3_linux_arm64.tar.gz: ecb6d51a2bda5cc527a3e5f672292a2571c6b62e2bc38ddacb6fa027eb835beb
      OK: origin main 2a02bde2c5c83183d6e49f235fcb57509687b97b, annotated tag 092a857361d3109ba0e27ae76d796e3d3b915c0d, latest public release and all five downloaded assets
      https://github.com/Sayoner-000/pi-gomemory/releases/tag/v1.15.3

## Siguiente
Cerrada con aprobación sobre la revisión 427f1d9e7b0b.

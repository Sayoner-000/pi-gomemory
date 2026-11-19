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
- [ ] 3. Validar los cuatro paquetes dist/ de v1.15.3 generados por scripts/build-release.sh, sus SHA256, plataformas y revisión Git → node /tmp/pi-gomemory-v1.15.3-checks/check-artifacts.mjs
- [ ] 4. Verificar los commits, tag y cinco assets publicados de v1.15.3 en origin → node /tmp/pi-gomemory-v1.15.3-checks/check-remote.mjs

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-19 05:50 -05:00 · tarea 1 · verde · vigente · `node /tmp/pi-gomemory-v1.15.3-checks/check-docs.mjs` (salida 0)
      OK: 12 current documentation files; 143 local references; versions, scope and shell syntax checked
- 2026-11-19 05:50 -05:00 · tarea 2 · verde · vigente · `go build ./... && go vet ./... && go vet -tags pins ./... && go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.856s
      ok  	pi-gomemory/pkg/installer	104.327s

## Siguiente
pendiente

# update y deps sin depender de la API de GitHub

ID: HCD-009
Tipo: fix
Carril: fast
Motivo: fix con causa observada (HTTP 403 por límite de consultas), sin decisiones abiertas

## Intención
Obtener la última release desde la redirección de github.com/<repo>/releases/latest, sin la API limitada a 60 consultas por hora

## Reproducción
El 2026-11-11, tras varias consultas, `pi-gomemory-setup update` falló con «consultar la última release: releases API: HTTP 403». `api.github.com` limita a 60 consultas por hora y por IP sin autenticar; detrás de una IP compartida, todos los usuarios agotan el mismo cupo. `deps-refresh` usa la misma API para pi-gomemory y codebase-memory-mcp. En cambio, `https://github.com/<repo>/releases/latest` responde con una redirección (`location: .../releases/tag/v1.5.0`) que no usa ese cupo.

## Alcance
Incluido: `fetchLatestTag` pide la URL sin seguir redirecciones y toma la etiqueta de la cabecera `Location` (`/releases/tag/<tag>`); si recibe un 200 con JSON, sigue leyendo `tag_name`. La URL por defecto de `CheckUpdate`, `update` y `deps-refresh` pasa a `https://github.com/<repo>/releases/latest`, para pi-gomemory y para codebase-memory-mcp.
Excluido: autenticarse con token de GitHub; cambiar la descarga de los tarballs (ya usa github.com).

## ¿Qué pasa si no lo hacemos?
`update` y la comprobación de dependencias fallan durante una hora cada vez que se agota el cupo, y más a menudo en oficinas con una IP compartida.

## Alternativa más barata
Admitir un `GITHUB_TOKEN` opcional para subir el límite; obliga a cada persona a crear y guardar un token.

## Aclaraciones

## Checklist
- [x] 1. fetchLatestTag lee la etiqueta de la redirección y mantiene el JSON → `go test ./pkg/installer -run TestLatestTagFromRedirect`
- [x] 2. Las URL por defecto no usan api.github.com → `go test ./pkg/installer -run TestNoGitHubAPI`
- [x] 3. Suite completa → `go test ./... && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 05:27 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestLatestTagFromRedirect` (salida 1)
      --- FAIL: TestLatestTagFromRedirect (0.00s)
          hcd009_test.go:20: no debe seguir la redirección: /owner/repo/releases/tag/v9.8.7
          hcd009_test.go:26: tag = "", releases API: EOF
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.338s
      FAIL
- 2026-11-11 05:27 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestNoGitHubAPI` (salida 1)
      --- FAIL: TestNoGitHubAPI (0.00s)
          hcd009_test.go:41: DefaultReleasesAPI = https://api.github.com/repos/Sayoner-000/pi-gomemory/releases/latest
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.166s
      FAIL
- 2026-11-11 05:27 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestLatestTagFromRedirect` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.168s
- 2026-11-11 05:27 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestNoGitHubAPI` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.159s
- 2026-11-11 05:27 -05:00 · tarea 3 · verde · vigente · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	0.663s

## Siguiente
Cerrada con aprobación sobre la revisión 27b0f4750573.

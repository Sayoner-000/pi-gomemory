# Fijar versiones de pi-antigravity y pi-cc-extensions

ID: HCD-005
Tipo: technical
Carril: fast
Motivo: decisión de la persona: versiones oficiales; sin decisiones abiertas

## Intención
Instalar npm:pi-antigravity@0.12.0 y npm:pi-cc-extensions@1.0.16 para que todos los usuarios tengan las mismas dependencias

## Reproducción
`plugin.go` instala `npm:pi-antigravity` y `npm:pi-cc-extensions` sin versión: Pi reinstala la última publicada en cada arranque y dos usuarios con el mismo binario pueden tener versiones distintas. Context7 (/earendil-works/pi, docs/packages.md): «A version pinned to an exact number, such as `@1.2.3`, is not checked». Versiones oficiales actuales en npm (dist-tag latest, sin deprecación): pi-antigravity 0.12.0, pi-cc-extensions 1.0.16.

## Alcance
Incluido: constantes con versión exacta; `settings.json.packages` sustituye la entrada sin versión (o con otra versión) por la fijada, conservando las demás; el fallo de `pi install` de pi-cc-extensions deja de ignorarse en silencio (cae al registro declarativo); deps-status compara lo instalado con la versión fijada y actualiza con `pi install` de esa versión; doctor reconoce la entrada fijada.
Excluido: cambiar de paquete; firmar releases.

## ¿Qué pasa si no lo hacemos?
Cada arranque de Pi puede traer una versión nueva sin probar con pi-gomemory y romper la instalación de forma distinta para cada usuario.

## Alternativa más barata
Documentar las versiones probadas; no impide que Pi las cambie.

## Aclaraciones
- [x] Q1. ¿Qué versiones? → las oficiales actuales (npm latest): pi-antigravity 0.12.0, pi-cc-extensions 1.0.16

## Checklist
- [x] 1. settings.json queda con las entradas fijadas, sin duplicados y conservando otras → `go test ./pkg/installer -run TestPinnedPackages`
- [x] 2. deps compara con la versión fijada y la reinstala con pi install → `go test ./pkg/installer -run TestPinnedDepProbes`
- [x] 3. Suite completa → `go test ./... && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 04:51 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestPinnedPackages` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd005_test.go:16:13: undefined: EnsurePinnedPackages
      pkg/installer/hcd005_test.go:21:57: undefined: AntigravityVersion
      pkg/installer/hcd005_test.go:21:110: undefined: CCExtensionsVersion
      pkg/installer/hcd005_test.go:29:53: undefined: AntigravityVersion
      pkg/installer/hcd005_test.go:29:118: undefined: CCExtensionsVersion
      pkg/installer/hcd005_test.go:42:11: undefined: AntigravityVersion
      pkg/installer/hcd005_test.go:44:11: undefined: CCExtensionsVersion
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 04:51 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestPinnedDepProbes` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd005_test.go:16:13: undefined: EnsurePinnedPackages
      pkg/installer/hcd005_test.go:21:57: undefined: AntigravityVersion
      pkg/installer/hcd005_test.go:21:110: undefined: CCExtensionsVersion
      pkg/installer/hcd005_test.go:29:53: undefined: AntigravityVersion
      pkg/installer/hcd005_test.go:29:118: undefined: CCExtensionsVersion
      pkg/installer/hcd005_test.go:42:11: undefined: AntigravityVersion
      pkg/installer/hcd005_test.go:44:11: undefined: CCExtensionsVersion
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 04:52 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestPinnedPackages` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.175s
- 2026-11-11 04:52 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestPinnedDepProbes` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.155s
- 2026-11-11 04:52 -05:00 · tarea 3 · verde · vigente · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	(cached)

## Siguiente
Cerrada con aprobación sobre la revisión 4318094e68c6.

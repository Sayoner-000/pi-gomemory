# Cerrar H-02 a H-05 de la ACR/ADM 2026-11-17

ID: HCD-041
Tipo: fix
Carril: fast
Motivo: Fix con cuatro defectos reproducidos y correcciones locales; las decisiones abiertas (estado desconocido en deps-update, umbrales de extracción) son independientes y se resuelven con clarify

## Intención
uninstall conserva un enlace sobrevenido tras el plan, los probes no aceptan la salida de un comando fallido como versión, deps-update no declara al día un estado desconocido y la extracción de candidatos limita entradas y bytes descomprimidos

## Reproducción
Origen: [ACR/ADM del 2026-11-17](../docs/audits/2026-11-17-acr-adm.md), sobre `5183867` más HCD-039 y HCD-040 (digest `c607cab1488c…`). H-01 se cerró en [HCD-040](HCD-040-subir-los-pins-a-pi-antigravity-0-14-0-y.md) (Q2); esta unidad cierra el resto.

Los reproductores de la auditoría (`docs/audits/2026-11-17-acr-adm-repro_test.go.txt`) pasan a `pkg/installer/hcd041_test.go`, salvo `TestACRADM01`, que ya está en la suite:

```bash
go test ./pkg/installer -run '^TestACRADM0[2-5]' -count=1
```

Hoy fallan:
- `TestACRADM02PreserveSymlinkAfterPlan` (H-02): un `pi-cc-extensions.json` por defecto sustituido por un enlace después del plan se borra (`lstat … no such file or directory`). La acción de `planCCExtensions` (`pkg/installer/uninstall.go:796-805`) sigue el enlace con `ReadFile`, compara y hace `os.Remove` de la ruta.
- `TestACRADM03FailedCommandIsNotVersion` (H-03): `sh -c "printf '9.0.0\n'; exit 7"` se acepta como versión `9.0.0` sin error. `runOutput` (`pkg/installer/deps.go`) solo devuelve el error si la salida está vacía.
- `TestACRADM04MissingCacheIsNotHealthy` (H-04): con la caché ausente, `RunDepsUpdate` sale con 0 y dice «✓ El meta-harness y sus dependencias están al día.».
- `TestACRADM05EmptyEntriesAreBounded` (H-05): hoy es un testigo que pasa (2.000 entradas vacías se extraen). Se invierte: un tarball con más entradas que el límite debe rechazarse.

## Alcance
Incluido: cerrar H-02, H-03, H-04 y H-05 con sus reproductores como pruebas permanentes. En detalle:
- H-02: la acción de retirada de `pi-cc-extensions.json` comprueba con `Lstat`, antes de leer, que la ruta sigue siendo un archivo normal; un enlace, un directorio o cualquier otro tipo aparecido después del plan se conserva como «cambió después del plan». Cubre el cambio previo al inicio de la acción, que es el reproducido; no se presenta como garantía frente a un actor que sustituya la entrada en paralelo durante la propia acción.
- H-03: `runOutput` deja de aceptar la salida de un comando que termina con error o que se cancela. Las comprobaciones reales salen hoy con 0 cuando todo está al día (medido el 2026-11-17: `pi --version`, `mem version`, `mem update -check`, `codegraph --version`, `codegraph upgrade --check` y `codebase-memory-mcp --version`), y el código de `mem` confirma que `mem update -check` también sale con 0 cuando hay una versión nueva y solo falla si no puede consultar (`gomemory/adapters/primary/cli/cmd_update.go:63-83`). De `codegraph upgrade --check` solo consta la salida con 0 al día. Error estricto en todos los probes (Q1): un código no cero o una cancelación nunca dan versión y el componente queda con `error` en la caché; si `codegraph` saliera con código no cero al haber actualización, el síntoma es un error visible, no un falso «al día», y se corrige para ese probe cuando se observe.
- H-04: `deps-update` no declara «al día» un estado que no conoce (Q2). Sin caché o con la caché ilegible, no actualiza nada, pide `pi-gomemory-setup deps-refresh` y sale con 1. Con una caché válida, actualiza lo que sabe atrasado, lista cada componente con `error` como «<nombre>: estado desconocido (<error>)» y sale con 1 si hubo alguno; solo dice «al día» si la caché es válida, ningún componente tiene error y nada va atrás. Un componente opcional no instalado no bloquea el resto de actualizaciones.
- H-05: `extractCandidate` limita el número de entradas del tarball y los bytes del flujo tar descomprimido (cabeceras incluidas), además del límite de contenido que ya tiene, y comprueba la cancelación del contexto entre entradas. Los tarballs reales miden 56 entradas y 0,5 MiB (`pi-antigravity@0.14.0`) y 85 entradas y 1 MiB (`pi-cc-extensions@1.0.20`). Límites (Q3): 10.000 entradas y 256 MiB de flujo tar descomprimido, cabeceras incluidas, además de los 200 MiB de contenido actuales.
- Pruebas: los reproductores 02-05 en `pkg/installer/hcd041_test.go`, más los casos que pide la auditoría para cada hallazgo (enlace original y sobrevenido, directorio y enlace roto; fallo con stderr semver, cancelación y éxito; JSON inválido, caché parcial y comprobación desactivada; tarball justo por debajo y por encima de cada límite, directorios vacíos, cabeceras voluminosas y limpieza tras el error).

Excluido: H-01 (cerrado en HCD-040), un diseño de retirada atómica frente a un actor local concurrente en `uninstall`, instalar `govulncheck`, `staticcheck` o `shellcheck`, y la release.

## ¿Qué pasa si no lo hacemos?
- H-02: `uninstall --packages` puede borrar un enlace a una configuración propia que se creó mientras esperaba la confirmación.
- H-03: un componente cuyo comando de versión falla, pero imprime una versión en su error, aparece como instalado o al día.
- H-04: `deps-update` dice «al día» sin caché, o con la caché rota, y sale con 0: un script o la persona creen que todo está actualizado.
- H-05: un tarball de candidato con millones de entradas vacías agota inodos o CPU durante `pins-validate`; el riesgo es acotado porque se ejecuta a mano y el tarball viene del registro con integridad verificada.

## Alternativa más barata
- Cerrar solo los tres CONFIRMED por ACR (H-02, H-03, H-04 queda SUSPECT): H-04 y H-05 tienen reproducción ADM propia (H-04 funcional, H-05 estructural) y su corrección es local; dejarlos abiertos no ahorra trabajo relevante.
- Documentar H-05 como riesgo aceptado: posible, porque `pins-validate` lo ejecuta solo quien mantiene el proyecto, pero el límite cuesta unas líneas y una prueba.
Se propone cerrar los cuatro.


## Aclaraciones
- [x] Q1. H-03: ¿error estricto en todos los probes (un comando con código no cero nunca da versión), o los dos de última versión (latestFromCmd, con su patrón concreto) aceptan su salida aunque el código no sea cero? → Estricto en todos: un comando con código no cero o cancelado nunca da versión y el componente queda con error en la caché; si codegraph resultara salir con código no cero al haber actualización, se corrige para ese probe cuando se observe.
- [x] Q2. H-04: con la caché ausente, ilegible o con algún componente en error, ¿deps-update se detiene con código no cero pidiendo deps-refresh, o actualiza lo que sí conoce y avisa de lo desconocido con código no cero? → Actualizar lo conocido: sin caché o con caché ilegible no actualiza nada, pide deps-refresh y sale con 1; con caché válida actualiza lo que sabe atrasado, lista los componentes con error como estado desconocido y sale con 1 si hubo alguno, sin decir nunca «al día».
- [x] Q3. H-05: ¿límites de 10.000 entradas y 256 MiB de flujo tar descomprimido (cabeceras incluidas), como propone la auditoría, u otros? → 10.000 entradas y 256 MiB de flujo tar descomprimido, cabeceras incluidas, manteniendo el límite de 200 MiB de contenido.

## Checklist
- [x] 1. La reproducción ya no falla (reproductores 02-05 de la auditoría en `pkg/installer/hcd041_test.go`, el 05 invertido) → go test ./pkg/installer -run '^TestACRADM0[2-5]' -count=1
- [x] 2. Conservar en `uninstall --packages` un `pi-cc-extensions.json` que deja de ser archivo normal después del plan (enlace, enlace roto o directorio) y seguir retirando el archivo normal por defecto (H-02) → go test ./pkg/installer -run '^TestHCD041CCExtensionsAfterPlan$' -count=1
- [x] 3. Rechazar en `runOutput` la salida de un comando que termina con error o se cancela, conservando el éxito (H-03, Q1) → go test ./pkg/installer -run '^TestHCD041FailedCommand$' -count=1
- [x] 4. Hacer que `deps-update` no declare «al día» un estado desconocido: sin caché o ilegible no actualiza y sale con 1; con componentes en error actualiza lo conocido y sale con 1 (H-04, Q2) → go test ./pkg/installer -run '^TestHCD041DepsUpdateUnknown$' -count=1
- [x] 5. Limitar en `extractCandidate` las entradas (10.000) y el flujo tar descomprimido (256 MiB) y comprobar la cancelación entre entradas (H-05, Q3) → go test ./pkg/installer -run '^TestHCD041ExtractLimits$' -count=1
- [x] 6. Mantener verde la suite completa con el detector de carreras → go test -race ./... -count=1
- [x] 7. Mantener limpio el análisis estático con y sin la etiqueta `pins` → go vet ./... && go vet -tags pins ./...

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-17 08:03 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestACRADM0[2-5]' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd041_test.go:86:9: undefined: candidateEntryLimit
      pkg/installer/hcd041_test.go:87:2: undefined: candidateEntryLimit
      pkg/installer/hcd041_test.go:88:21: undefined: candidateEntryLimit
      pkg/installer/hcd041_test.go:89:79: too many arguments in call to extractCandidate
      	have (context.Context, []byte, string)
      	want ([]byte, string)
      pkg/installer/hcd041_test.go:92:79: too many arguments in call to extractCandidate
      	have (context.Context, []byte, string)
      	want ([]byte, string)
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 08:03 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD041ExtractLimits$' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd041_test.go:87:5: undefined: candidateEntryLimit
      pkg/installer/hcd041_test.go:87:38: undefined: candidateTarStreamLimit
      pkg/installer/hcd041_test.go:88:74: undefined: candidateEntryLimit
      pkg/installer/hcd041_test.go:88:95: undefined: candidateTarStreamLimit
      pkg/installer/hcd041_test.go:90:25: undefined: candidateEntryLimit
      pkg/installer/hcd041_test.go:90:46: undefined: candidateTarStreamLimit
      pkg/installer/hcd041_test.go:91:21: undefined: candidateEntryLimit
      pkg/installer/hcd041_test.go:91:42: undefined: candidateTarStreamLimit
      pkg/installer/hcd041_test.go:94:2: undefined: candidateEntryLimit
      pkg/installer/hcd041_test.go:102:44: too many arguments in call to extractCandidate
      	have (context.Context, []byte, string)
      	want ([]byte, string)
      pkg/installer/hcd041_test.go:102:44: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 08:04 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD041ExtractLimits$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.353s
- 2026-11-17 08:04 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD041CCExtensionsAfterPlan$' -count=1` (salida 1)
      --- FAIL: TestHCD041CCExtensionsAfterPlan (0.00s)
          --- FAIL: TestHCD041CCExtensionsAfterPlan/enlace_a_un_archivo_propio (0.00s)
              hcd041_test.go:128: debe devolver errChangedAfterPlan, devolvió <nil>
              hcd041_test.go:131: la entrada sobrevenida debe conservarse: lstat /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD041CCExtensionsAfterPlanenlace_a_un_archivo_propio3393475317/001/pi-cc-extensions.json: no such file or directory
          --- FAIL: TestHCD041CCExtensionsAfterPlan/enlace_roto (0.00s)
              hcd041_test.go:128: debe devolver errChangedAfterPlan, devolvió ya no existe
          --- FAIL: TestHCD041CCExtensionsAfterPlan/directorio (0.00s)
              hcd041_test.go:128: debe devolver errChangedAfterPlan, devolvió read /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD041CCExtensionsAfterPlandirectorio1782652694/001/pi-cc-extensions.json: is a directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.215s
      FAIL
- 2026-11-17 08:04 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD041CCExtensionsAfterPlan$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.199s
- 2026-11-17 08:05 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD041FailedCommand$' -count=1` (salida 1)
      --- FAIL: TestHCD041FailedCommand (5.05s)
          hcd041_test.go:93: semver en stdout: un fallo no debe dar versión, dio "9.0.0"
          hcd041_test.go:93: semver en stderr: un fallo no debe dar versión, dio "9.0.0"
          hcd041_test.go:98: latestFromCmd tampoco acepta un fallo con su patrón: "9.9.9"
          hcd041_test.go:104: una cancelación con salida previa no debe dar versión: "1.2.3"
      FAIL
      FAIL	pi-gomemory/pkg/installer	5.297s
      FAIL
- 2026-11-17 08:05 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD041FailedCommand$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.446s
- 2026-11-17 08:07 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD041DepsUpdateUnknown$' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd041_test.go:124:17: undefined: strings
      pkg/installer/hcd041_test.go:130:83: undefined: strings
      pkg/installer/hcd041_test.go:141:16: undefined: strings
      pkg/installer/hcd041_test.go:142:4: undefined: strings
      pkg/installer/hcd041_test.go:148:55: undefined: strings
      pkg/installer/hcd041_test.go:154:52: undefined: strings
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 08:07 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD041DepsUpdateUnknown$' -count=1` (salida 1)
      --- FAIL: TestHCD041DepsUpdateUnknown (0.14s)
          hcd041_test.go:126: ausente: sin estado conocido no actualiza, pide deps-refresh y sale con 1: rc=0 out="✓ El meta-harness y sus dependencias están al día.\n" err="" calls=""
          hcd041_test.go:126: JSON inválido: sin estado conocido no actualiza, pide deps-refresh y sale con 1: rc=0 out="✓ El meta-harness y sus dependencias están al día.\n" err="" calls=""
          hcd041_test.go:126: sin checked_at: sin estado conocido no actualiza, pide deps-refresh y sale con 1: rc=0 out="==> Actualizando pi-antigravity (0.0.1 → 0.14.0)...\n✓ Dependencias actualizadas.\n" err="" calls="install npm:pi-antigravity@0.14.0\n"
          hcd041_test.go:132: con la comprobación desactivada explica por qué no hay datos: rc=0 err=""
          hcd041_test.go:144: actualiza lo conocido y lista lo desconocido con código 1: rc=0 out="==> Actualizando pi-antigravity (0.0.1 → 0.14.0)...\n✓ Dependencias actualizadas.\n" err="" calls="install npm:pi-antigravity@0.14.0\n"
          hcd041_test.go:150: caché parcial: rc=0 out="==> Actualizando pi-antigravity (0.0.1 → 0.14.0)...\n✓ Dependencias actualizadas.\n" err=""
          hcd041_test.go:156: con todo conocido y al día dice «al día» y sale con 0: rc=0 out="==> Actualizando pi-antigravity (0.0.1 → 0.14.0)...\n✓ Dependencias actualizadas.\n" calls="install npm:pi-antigravity@0.14.0\n"
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.475s
      FAIL
- 2026-11-17 09:05 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestACRADM0[2-5]' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.196s
- 2026-11-17 09:05 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD041CCExtensionsAfterPlan$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.153s
- 2026-11-17 09:05 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD041FailedCommand$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.386s
- 2026-11-17 09:05 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD041DepsUpdateUnknown$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.263s
- 2026-11-17 09:05 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD041ExtractLimits$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.188s
- 2026-11-17 09:05 -05:00 · tarea 6 · verde · vigente · `go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.192s
      ok  	pi-gomemory/pkg/installer	58.139s
- 2026-11-17 09:06 -05:00 · tarea 7 · verde · vigente · `go vet ./... && go vet -tags pins ./...` (salida 0)

## Siguiente
Cerrada con aprobación sobre la revisión c3472feedfd3.

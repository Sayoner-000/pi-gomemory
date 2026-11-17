# Cerrar F-01 y F-02 del seguimiento ACR y ADM

ID: HCD-042
Tipo: fix
Carril: fast
Motivo: Dos defectos medios corroborados y reproducidos, con correcciones locales independientes; aclarar la preservación de entradas en cachés y manifiesto antes de implementar

## Intención
Evitar que deps-update declare al día versiones no comparables y preservar entradas sustituidas después del plan al retirar cachés y manifiesto, según el contrato que se acuerde en la definición

## Reproducción
Origen: [seguimiento ACR/ADM del 2026-11-17](../docs/audits/2026-11-17-acr-adm-followup.md), hallazgos F-01 y F-02, ambos CONFIRMED MEDIUM; revisión ACR `acr_cf822591-fe3c-4086-a087-42a73823a7d6`. Reproductores en [adjunto sintético](../docs/audits/2026-11-17-acr-adm-followup-repro_test.go.txt).

Reproducción observada durante la auditoría, en una copia temporal del producto: copiar el adjunto como `pkg/installer/acradm_followup_test.go` y ejecutar:

```sh
go test ./pkg/installer -run '^TestFollowup' -count=1 -v
```

- F-01: `TestFollowupInvalidVersionNotHealthy` falla con `current=1.0.0/latest=desconocida` y `current=dev/latest=1.0.0`; `RunDepsUpdate` devuelve 0 y anuncia «están al día». Las cadenas no están vacías y no hay Error, pero `IsNewer` no puede compararlas y devuelve false.
- F-02: `TestFollowupLateEntryPreserved` falla en cuatro casos: caché/manifiesto × enlace/directorio vacío. Tras sustituir el archivo regular después del plan, `apply` devuelve nil y elimina la entrada nueva. El destino del enlace permanece intacto. `planPlainFile` y `planManifest` llaman a Remove sin revalidación al ejecutar.

Los reproductores se incorporarán como pruebas permanentes al implementar, antes de corregir. El rojo oficial de la tarea 1 deberá registrarse con el motor sobre el código sin corrección; el resultado de auditoría no lo sustituye.

## Alcance
Incluido: corregir F-01 en el diagnóstico de dependencias y F-02 en la retirada de las dos cachés y del manifiesto de instalación, con pruebas permanentes.

- F-01: clasificar como estado desconocido las versiones efectivas no comparables según el comparador estable existente, incluido `dev`, sin anunciarlas como «al día». Resolver primero el objetivo del pin compilado y la versión local conforme a HCD-040; no rechazar una latest antigua de caché que el pin sustituye válidamente. Conservar la política HCD-041: actualizar los componentes conocidos, informar los desconocidos, pedir `deps-refresh` y salir con 1 si existe alguno. No agregar consultas de red implícitas ni tomar comandos de la caché. Versiones estables comparables realmente al día mantienen salida 0.
- F-02: revalidar antes de ejecutar la retirada de las cachés de dependencias/actualizaciones y del manifiesto, preservando enlaces y directorios aparecidos después del plan. Según Q1 (opción 1), conservar enlaces y directorios aunque ya existieran al planificar, y archivos regulares sustituidos después del plan, revalidando su identidad. Según Q2 (opción 1), retener el archivo original hasta ejecutar o descartar el plan para impedir la reutilización de su inode durante ese intervalo, con cierre explícito de recursos en todas las rutas de salida, incluidas simulación, cancelación y error. Mantener la retirada del manifiesto al final y solo sin fallos previos (`ifClean`, HCD-035), así como el contrato de reintentos.
- Verificación: reproductores originales de ambos hallazgos, casos sanos, actualización parcial conocida y precedencia de pins; preservación del destino del enlace; archivo regular original retirado; condiciones del manifiesto y reintentos; suite con carreras y análisis estático con y sin `pins`.

Excluido: retirada atómica frente a sustituciones concurrentes durante la propia acción; rediseñar el comparador global de versiones o admitir nuevos formatos; ampliar la retirada a otros recursos; cambiar pins; instalar en HOME real; completar o cerrar HCD-040; commit y release.

Antecedentes: HCD-041 cerró los escenarios originales H-02/H-04, no todas las variantes de retirada y comparabilidad. Esta unidad amplía la preservación más allá de `pi-cc-extensions.json`, sin declarar regresión del cierre anterior. HCD-040 conserva prioridad del pin compilado; HCD-035 conserva manifiesto para reintentos tras fallos.

## ¿Qué pasa si no lo hacemos?
- F-01: una compilación local `dev` o una caché con versiones no comparables sigue produciendo un éxito falso; la persona o un script interpretan un estado desconocido como actualizado.
- F-02: al confirmar uninstall se puede borrar un enlace o directorio vacío creado durante la espera en una ruta de caché o manifiesto. No se borra el archivo destino, pero se pierde la entrada nueva.
El impacto es acotado y no se demostró compromiso de credenciales ni pérdida del contenido del destino; sigue siendo un defecto de diagnóstico y preservación.

## Alternativa más barata
Documentar que `dev` no se compara y exigir no modificar cachés/manifiesto mientras se confirma uninstall evita cambios de código, pero deja ambos comportamientos reproducidos sin corregir. No constituye el cierre técnico solicitado. Corregir solo F-01 sería más barato, pero dejaría F-02 abierto. Se mantienen ambas correcciones locales y se acuerda la política de preservación antes de implementarlas.

## Normas aplicables
Constitución §8 (errores explícitos), §10 (pruebas de comportamiento), §18 (documento único), §21 (causa raíz y seguridad por defecto). Playbook HCD-DEF-01, HCD-FLOW-02, HCD-TDD-01 y HCD-EVIDENCE-01: definición verificable, aclaraciones resueltas antes de implementar y evidencia roja/verde registrada por el motor.


## Aclaraciones
- [x] Q1. F-02: al retirar cachés y manifiesto, ¿conservar toda entrada no regular incluso si ya existía al planificar y todo archivo regular sustituido por otro (revalidando identidad), o mantener la retirada de enlaces preexistentes y conservar solo cambios posteriores al plan? → Q1: 1 — Conservar enlaces y directorios aunque ya existieran al planificar, y archivos sustituidos después del plan, revalidando su identidad.
- [x] Q2. Q1 exige preservar archivos sustituidos: ¿reforzamos la identidad para cubrir también reutilización de inode entre plan y ejecución (requiere diseñar retención del archivo original y cierre explícito de recursos del plan), o aceptamos la identidad estándar dispositivo/inode de os.SameFile y documentamos esa limitación residual? → Q2: 1 — Retener el archivo original hasta ejecutar o descartar el plan, con cierre explícito de recursos, para cubrir la reutilización de inode y mantener la garantía de Q1.

## Checklist
- [x] 1. La reproducción ya no falla (F-01/F-02 en `pkg/installer/hcd042_test.go`) → go test ./pkg/installer -run '^TestFollowup' -count=1
- [x] 2. Clasificar las versiones efectivas no comparables como desconocidas en `pkg/installer/deps.go`, conservando actualización parcial y precedencia de pins → go test ./pkg/installer -run '^TestHCD042DepsComparable$' -count=1
- [x] 3. Preservar entradas no regulares y archivos sustituidos en las retiradas de cachés/manifiesto de `pkg/installer/uninstall.go` (Q1) → go test ./pkg/installer -run '^TestHCD042Removal$' -count=1
- [x] 4. Conservar los cierres anteriores de manifiesto, pins y extracción en `pkg/installer` → go test ./pkg/installer -run '^(TestHCD035|TestHCD040|TestHCD041|TestACRADM0[1-5])' -count=1
- [x] 5. Mantener verde la suite completa con detector de carreras → go test -race ./... -count=1
- [x] 6. Mantener limpio el análisis estático con y sin `pins` → go vet ./... && go vet -tags pins ./...
- [x] 7. Retener el archivo original y cerrar explícitamente los recursos del plan de `pkg/installer/uninstall.go` al ejecutar o descartar, incluidas simulación, cancelación y error (Q2) → go test ./pkg/installer -run '^TestHCD042PlanLifecycle$' -count=1

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-17 09:23 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestFollowup' -count=1` (salida 1)
      --- FAIL: TestFollowupInvalidVersionNotHealthy (0.00s)
          hcd042_test.go:23: uncomparable [1.0.0 desconocida]: rc=0 stdout="✓ El meta-harness y sus dependencias están al día.\n" stderr=""
          hcd042_test.go:23: uncomparable [dev 1.0.0]: rc=0 stdout="✓ El meta-harness y sus dependencias están al día.\n" stderr=""
      --- FAIL: TestFollowupLateEntryPreserved (0.00s)
          --- FAIL: TestFollowupLateEntryPreserved/cache/symlink (0.00s)
              hcd042_test.go:52: replacement removed: apply=<nil> lstat=lstat /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestFollowupLateEntryPreservedcachesymlink969501272/001/owned.json: no such file or directory
          --- FAIL: TestFollowupLateEntryPreserved/cache/directory (0.00s)
              hcd042_test.go:52: replacement removed: apply=<nil> lstat=lstat /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestFollowupLateEntryPreservedcachedirectory1204303203/001/owned.json: no such file or directory
          --- FAIL: TestFollowupLateEntryPreserved/manifest/symlink (0.00s)
              hcd042_test.go:52: replacement removed: apply=<nil> lstat=lstat /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestFollowupLateEntryPreservedmanifestsymlink3165120689/001/owned.json: no such file or directory
          --- FAIL: TestFollowupLateEntryPreserved/manifest/directory (0.00s)
              hcd042_test.go:52: replacement removed: apply=<nil> lstat=lstat /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestFollowupLateEntryPreservedmanifestdirectory3322198647/001/owned.json: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.369s
      FAIL
- 2026-11-17 09:23 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD042DepsComparable$' -count=1` (salida 1)
      --- FAIL: TestHCD042DepsComparable (0.01s)
          --- FAIL: TestHCD042DepsComparable/latest_inválida (0.00s)
              hcd042_test.go:102: rc=0 stdout="✓ El meta-harness y sus dependencias están al día.\n" stderr=""
          --- FAIL: TestHCD042DepsComparable/current_dev (0.00s)
              hcd042_test.go:102: rc=0 stdout="✓ El meta-harness y sus dependencias están al día.\n" stderr=""
          --- FAIL: TestHCD042DepsComparable/local_dev_con_caché_sana (0.00s)
              hcd042_test.go:102: rc=0 stdout="✓ El meta-harness y sus dependencias están al día.\n" stderr=""
          --- FAIL: TestHCD042DepsComparable/prerelease_no_comparable (0.00s)
              hcd042_test.go:102: rc=0 stdout="✓ El meta-harness y sus dependencias están al día.\n" stderr=""
          --- FAIL: TestHCD042DepsComparable/actualización_parcial (0.01s)
              hcd042_test.go:125: rc=0 stdout="==> Actualizando known (1.0.0 → 2.0.0)...\n✓ Dependencias actualizadas.\n" stderr=""
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.164s
      FAIL
- 2026-11-17 09:24 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD042DepsComparable$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.173s
- 2026-11-17 09:24 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD042Removal$' -count=1` (salida 1)
       TestHCD042Removal/cache/after/broken_symlink (0.00s)
              hcd042_test.go:222: esperado changed: <nil>
          --- FAIL: TestHCD042Removal/cache/after/gone (0.00s)
              hcd042_test.go:218: esperado gone: remove /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042Removalcacheaftergone511680624/001/owned: no such file or directory
          --- FAIL: TestHCD042Removal/manifest/preexisting/symlink (0.00s)
              hcd042_test.go:164: no debe planificar retirada: [{Kind:retirar Target:/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042Removalmanifestpreexistingsymlink337772989/001/owned Reason:manifiesto de instalación (se retira si no hay fallos) apply:0x1026850e0 notice:false ifClean:true}]
          --- FAIL: TestHCD042Removal/manifest/preexisting/broken_symlink (0.00s)
              hcd042_test.go:164: no debe planificar retirada: [{Kind:retirar Target:/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042Removalmanifestpreexistingbroken_symlink3145408485/001/owned Reason:manifiesto de instalación (se retira si no hay fallos) apply:0x1026850e0 notice:false ifClean:true}]
          --- FAIL: TestHCD042Removal/manifest/preexisting/directory (0.00s)
              hcd042_test.go:164: no debe planificar retirada: [{Kind:retirar Target:/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042Removalmanifestpreexistingdirectory3469524861/001/owned Reason:manifiesto de instalación (se retira si no hay fallos) apply:0x1026850e0 notice:false ifClean:true}]
          --- FAIL: TestHCD042Removal/manifest/after/replaced_regular (0.00s)
              hcd042_test.go:222: esperado changed: <nil>
          --- FAIL: TestHCD042Removal/manifest/after/broken_symlink (0.00s)
              hcd042_test.go:222: esperado changed: <nil>
          --- FAIL: TestHCD042Removal/manifest/after/gone (0.00s)
              hcd042_test.go:218: esperado gone: remove /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042Removalmanifestaftergone315300506/001/owned: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.167s
      FAIL
- 2026-11-17 09:24 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD042Removal$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.183s
- 2026-11-17 09:25 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestFollowup' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.180s
- 2026-11-17 09:25 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD042DepsComparable$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.228s
- 2026-11-17 09:25 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^(TestHCD035|TestHCD040|TestHCD041|TestACRADM0[1-5])' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	8.996s
- 2026-11-17 09:25 -05:00 · tarea 5 · verde · caducada · `go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.227s
      ok  	pi-gomemory/pkg/installer	59.679s
- 2026-11-17 09:26 -05:00 · tarea 6 · verde · caducada · `go vet ./... && go vet -tags pins ./...` (salida 0)
- 2026-11-17 09:34 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD042PlanLifecycle$' -count=1` (salida 1)
      hqxk80000gn/T/TestHCD042PlanLifecycleUninstallexecution_error1077990018/001/.pi/agent/pi-gomemory-deps.json: referencias=0 esperado=1
              hcd042_lifecycle_test.go:196: plan no retiene /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042PlanLifecycleUninstallexecution_error1077990018/001/.pi/agent/pi-gomemory-update.json: referencias=0 esperado=1
              hcd042_lifecycle_test.go:196: plan no retiene /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042PlanLifecycleUninstallexecution_error1077990018/001/.pi-gomemory/manifest.json: referencias=0 esperado=1
          --- FAIL: TestHCD042PlanLifecycle/Uninstall/inspection_error (0.00s)
              hcd042_lifecycle_test.go:196: plan no retiene /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042PlanLifecycleUninstallinspection_error2151683892/001/.pi/agent/pi-gomemory-update.json: referencias=0 esperado=1
              hcd042_lifecycle_test.go:196: plan no retiene /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042PlanLifecycleUninstallinspection_error2151683892/001/.pi-gomemory/manifest.json: referencias=0 esperado=1
              hcd042_lifecycle_test.go:206: error=<nil> salida=Plan de desinstalación:
                    [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042PlanLifecycleUninstallinspection_error2151683892/001/.pi/agent/pi-gomemory-deps.json — caché de dependencias
                    [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042PlanLifecycleUninstallinspection_error2151683892/001/.pi/agent/pi-gomemory-update.json — caché de actualizaciones
                    [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD042PlanLifecycleUninstallinspection_error2151683892/001/.pi-gomemory/manifest.json — manifiesto de instalación (se retira si no hay fallos)
                  Resumen: 3 retirados, 0 conservados, 0 omitidos, 0 fallidos.
                  ✓ pi-gomemory desinstalado. Para reinstalar: pi-gomemory-setup install
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.378s
      FAIL
- 2026-11-17 09:37 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD042PlanLifecycle$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.180s
- 2026-11-17 09:37 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestFollowup' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.172s
- 2026-11-17 09:37 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD042DepsComparable$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.168s
- 2026-11-17 09:37 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD042Removal$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.190s
- 2026-11-17 09:37 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^(TestHCD035|TestHCD040|TestHCD041|TestACRADM0[1-5])' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	9.980s
- 2026-11-17 09:38 -05:00 · tarea 5 · verde · caducada · `go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.241s
      ok  	pi-gomemory/pkg/installer	58.336s
- 2026-11-17 09:39 -05:00 · tarea 6 · verde · caducada · `go vet ./... && go vet -tags pins ./...` (salida 0)
- 2026-11-17 09:59 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestFollowup' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.494s
- 2026-11-17 09:59 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD042DepsComparable$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.161s
- 2026-11-17 09:59 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD042Removal$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.157s
- 2026-11-17 09:59 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^(TestHCD035|TestHCD040|TestHCD041|TestACRADM0[1-5])' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	9.195s
- 2026-11-17 09:59 -05:00 · tarea 6 · verde · vigente · `go vet ./... && go vet -tags pins ./...` (salida 0)
- 2026-11-17 09:59 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run '^TestHCD042PlanLifecycle$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.197s
- 2026-11-17 09:59 -05:00 · tarea 5 · verde · vigente · `go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.227s
      ok  	pi-gomemory/pkg/installer	74.630s

## Siguiente
Cerrada con aprobación sobre la revisión d94ce297d2d0.

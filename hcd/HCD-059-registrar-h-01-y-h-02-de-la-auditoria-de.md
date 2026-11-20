# Registrar H-01 y H-02 de la auditoría defensiva del 2026-11-20 publicados en v1.19.2

ID: HCD-059
Tipo: fix
Carril: fast
Motivo: Unidad de registro de dos arreglos ya publicados en v1.19.2, con causa raíz verificada en el código de Pi 1.5.2; sin decisiones abiertas

## Intención
Dejar en el historial HCD la causa raíz, el cierre y la evidencia ejecutada de H-01 (aviso de truncado y estado de salida de Pi desplazaban la reconciliación del texto final de Bash en /tdd) y H-02 (install.sh dejaba el binario temporal si fallaba al colocarlo), publicados en v1.19.2.

Los arreglos llegaron al repositorio fuera de una unidad HCD, después de publicar v1.19.1, y se publicaron en v1.19.2 (commits `b569e62` y `528105c`). Esta unidad no cambia código: registra con el motor la evidencia de que el defecto existía en v1.19.1 y de que v1.19.2 lo corrige.

## Reproducción
- H-01: Pi 1.5.2 (`core/tools/bash.js`, `formatOutput` y `appendStatus`) añade al texto final de Bash `\n\n[Showing lines a-b of N. Full output: …]` (o «Showing last …», «… limit») cuando trunca la salida, y `\n\nCommand exited with code N`, `Command aborted`, `Command timed out after …` o `Command terminated without an exit code` como estado. Las actualizaciones en vivo no los traen. La reconciliación de HCD-058 (C-001) contaba esas líneas al calcular la primera línea absoluta del texto truncado y saltaba las últimas líneas reales: un `--- FAIL` final quedaba sin resultado. `TestADM20261120BashFinalSuffix` lo reproduce sobre el código de v1.19.1.
- H-02: `install.sh` copiaba el binario a `${SETUP_BIN}.tmp.$$` y, con `set -e`, un fallo de `cp`, `chmod` o `mv` dejaba ese temporal en `INSTALL_DIR`. `TestADM20261120InstallShStagingCleanup` lo reproduce con un `chmod` que falla.

## Alcance
Incluido: registrar con `mem hcd verify` el rojo de ambas pruebas sobre los fuentes de v1.19.1 (tag `v1.19.1`) y el verde sobre el árbol actual, y la regresión completa.
Excluido: cambiar `extensions/gomemory-tdd.ts` o `install.sh` (ya publicados); publicar otra versión.

## ¿Qué pasa si no lo hacemos?
El historial HCD (`mem hcd history`) no reflejaría por qué cambió la reconciliación del texto final de Bash en v1.19.2 ni la limpieza del temporal de `install.sh`, y una revisión futura de esos archivos no encontraría la causa raíz ni su evidencia.

## Alternativa más barata
Dejarlo solo en el CHANGELOG y en la memoria del proyecto. No registra evidencia ejecutada por el motor ni aparece en `mem hcd history --file`; se descarta porque la persona pidió la unidad de registro.

## Normas aplicables
- Constitución §10 Pruebas: rojo observado antes del verde.
- Playbook HCD-TDD-01 y HCD-EVIDENCE-01: la evidencia la escribe el motor.

## Aclaraciones

## Checklist
- [x] 1. H-01: demostrar que en v1.19.1 el aviso de truncado y el estado de salida desplazaban la reconciliación y que v1.19.2 la corrige → `go test ./pkg/installer -run '^TestADM20261120BashFinalSuffix$' -count=1`
- [x] 2. H-02: demostrar que en v1.19.1 install.sh dejaba el temporal y que v1.19.2 lo retira → `go test ./pkg/installer -run '^TestADM20261120InstallShStagingCleanup$' -count=1`
- [x] 3. Ejecutar la regresión completa → `go vet ./... && go test -race ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-20 04:33 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestADM20261120BashFinalSuffix$' -count=1` (salida 1)
      --- FAIL: TestADM20261120BashFinalSuffix (0.28s)
          adm20261120_test.go:77: TestADM20261120BashFinalSuffix: FAIL 1: TestB debió fallar con su salida real: undefined | 1: no debe sustituirse por bash-error | 2: TestC debió pasar: undefined
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.599s
      FAIL
- 2026-11-20 04:33 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestADM20261120InstallShStagingCleanup$' -count=1` (salida 1)
      --- FAIL: TestADM20261120InstallShStagingCleanup (0.21s)
          adm20261120_test.go:146: install.sh dejó el temporal pi-gomemory-setup.tmp.64531 en INSTALL_DIR
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.420s
      FAIL
- 2026-11-20 04:34 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestADM20261120BashFinalSuffix$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.449s
- 2026-11-20 04:34 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestADM20261120InstallShStagingCleanup$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.341s
- 2026-11-20 04:34 -05:00 · tarea 3 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	48.759s

## Siguiente
Cerrada con aprobación sobre la revisión 28bdeebdbd44.

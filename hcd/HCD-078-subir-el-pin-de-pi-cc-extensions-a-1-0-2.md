# Subir el pin de pi-cc-extensions a 1.0.22

ID: HCD-078
Tipo: technical
Carril: fast
Motivo: Cambio acotado de una constante con precedente (HCD-063) y sin decisiones abiertas

## Intención
pi-gomemory instala pi-cc-extensions 1.0.22, la versión que npm publica, tras validarla contra el contrato C1, y pins-check vuelve a estar al día para publicar sin PI_GOMEMORY_ALLOW_STALE_PINS


## Alcance
Incluido: `CCExtensionsVersion` pasa a 1.0.22 en `pkg/installer/plugin.go`, 1.0.21 entra en `historicPins` para que `uninstall --packages` la retire, `TestHCD040PinsBumped` espera 1.0.22, y README y `docs/architecture.md` nombran el pin nuevo.
Excluido: el pin de pi-antigravity (sigue en 0.15.0, al día con npm), cambios en la configuración embebida `config/pi-cc-extensions.json` y publicar la release (va aparte).

## ¿Qué pasa si no lo hacemos?
`scripts/build-release.sh` detiene cada release en pins-check (npm publica 1.0.22 frente al pin 1.0.21) y hay que saltarlo con `PI_GOMEMORY_ALLOW_STALE_PINS=1`, como en v1.26.0. `doctor` sigue mostrando `⚠` en «Paquete fijado pi-cc-extensions» y los usuarios se quedan en una versión anterior.

## Alternativa más barata
Registrar 1.0.22 en `RejectedVersions` para que pins-check deje de contarla. Cuesta lo mismo y solo tiene sentido si la versión rompe algo; el 2026-11-24 `pins-validate` dio C1 cumplido (`package/extensions/config/config.ts`), así que no hay motivo para rechazarla.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Validar el candidato `pi-cc-extensions@1.0.22` contra los contratos de pi-gomemory → `PINS_CANDIDATE=pi-cc-extensions@1.0.22 go test -tags pins -count=1 ./pkg/installer -run '^TestPinsValidate$'`
- [x] 2. Subir el pin en `pkg/installer/plugin.go` a 1.0.22 con 1.0.21 en `historicPins` → `go test -count=1 ./pkg/installer -run '^TestHCD040PinsBumped$'`
- [x] 3. Comprobar con npm que el pin queda al día en `pkg/installer/pins.go` (pins-check sin escape) → `PI_GOMEMORY_ALLOW_STALE_PINS= go test -tags pins -count=1 ./pkg/installer -run '^TestPinsCheck$'`
- [x] 4. Instalar 1.0.22 con Pi real en un directorio de agente temporal → `D=$(mktemp -d) && PI_CODING_AGENT_DIR=$D pi install npm:pi-cc-extensions@1.0.22 && grep -q '"version": "1.0.22"' $D/npm/node_modules/pi-cc-extensions/package.json && test -f $D/npm/node_modules/pi-cc-extensions/extensions/config/config.ts; S=$?; rm -rf $D; exit $S`
- [x] 5. Nombrar el pin nuevo en `README.md` y `docs/architecture.md` → `grep -q 'pi-cc-extensions@1.0.22' README.md && grep -q 'pi-cc-extensions@1.0.22' docs/architecture.md`
- [x] 6. Ejecutar la regresión completa → `go vet ./... && go test -race ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-24 05:54 -05:00 · tarea 1 · verde · caducada · `PINS_CANDIDATE=pi-cc-extensions@1.0.22 go test -tags pins -count=1 ./pkg/installer -run '^TestPinsValidate$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.620s
- 2026-11-24 05:54 -05:00 · tarea 2 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run '^TestHCD040PinsBumped$'` (salida 1)
      --- FAIL: TestHCD040PinsBumped (0.00s)
          hcd040_test.go:149: pi-cc-extensions: 1.0.21 / npm:pi-cc-extensions@1.0.21, se espera 1.0.22
          hcd040_test.go:152: 1.0.21 debe quedar como pin histórico para que uninstall --packages lo retire
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.201s
      FAIL
- 2026-11-24 05:54 -05:00 · tarea 5 · rojo observado · caducada · `grep -q 'pi-cc-extensions@1.0.22' README.md && grep -q 'pi-cc-extensions@1.0.22' docs/architecture.md` (salida 1)
- 2026-11-24 05:54 -05:00 · tarea 3 · rojo observado · caducada · `PI_GOMEMORY_ALLOW_STALE_PINS= go test -tags pins -count=1 ./pkg/installer -run '^TestPinsCheck$'` (salida 1)
      --- FAIL: TestPinsCheck (0.17s)
          pins_check_test.go:22: pi-antigravity: al día (0.15.0)
          pins_check_test.go:22: pi-cc-extensions: fijado 1.0.21 · npm publica 1.0.22 sin validar
          pins_check_test.go:22: Para seguir: valida el candidato con PINS_CANDIDATE=<paquete>@<versión> go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$', regístralo como rechazado en RejectedVersions (pkg/installer/pins.go) o publica igualmente con PI_GOMEMORY_ALLOW_STALE_PINS=1.
          pins_check_test.go:25: pins-check: hay paquetes fijados atrasados frente a npm o sin comprobar
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.368s
      FAIL
- 2026-11-24 05:54 -05:00 · tarea 2 · verde · vigente · `go test -count=1 ./pkg/installer -run '^TestHCD040PinsBumped$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.178s
- 2026-11-24 05:54 -05:00 · tarea 3 · verde · vigente · `PI_GOMEMORY_ALLOW_STALE_PINS= go test -tags pins -count=1 ./pkg/installer -run '^TestPinsCheck$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.782s
- 2026-11-24 05:54 -05:00 · tarea 5 · verde · vigente · `grep -q 'pi-cc-extensions@1.0.22' README.md && grep -q 'pi-cc-extensions@1.0.22' docs/architecture.md` (salida 0)
- 2026-11-24 05:54 -05:00 · tarea 4 · verde · vigente · `D=$(mktemp -d) && PI_CODING_AGENT_DIR=$D pi install npm:pi-cc-extensions@1.0.22 && grep -q '"version": "1.0.22"' $D/npm/node_modules/pi-cc-extensions/package.json && test -f $D/npm/node_modules/pi-cc-extensions/extensions/config/config.ts; S=$?; rm -rf $D; exit $S` (salida 0)
      Installing npm:pi-cc-extensions@1.0.22...
      added 51 packages in 4s
      Installed npm:pi-cc-extensions@1.0.22
- 2026-11-24 05:54 -05:00 · tarea 6 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	2.520s
      ok  	pi-gomemory/pkg/installer	49.479s
- 2026-11-24 05:55 -05:00 · tarea 1 · verde · vigente · `PINS_CANDIDATE=pi-cc-extensions@1.0.22 go test -tags pins -count=1 ./pkg/installer -run '^TestPinsValidate$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.592s
- 2026-11-24 05:55 -05:00 · tarea 1 · verde · vigente · `PINS_CANDIDATE=pi-cc-extensions@1.0.22 go test -tags pins -count=1 ./pkg/installer -run '^TestPinsValidate$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.434s

## Siguiente
Cerrada con aprobación sobre la revisión 18908c22aea3.

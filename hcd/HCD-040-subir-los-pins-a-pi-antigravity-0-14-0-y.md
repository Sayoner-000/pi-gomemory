# Subir los pins a pi-antigravity 0.14.0 y pi-cc-extensions 1.0.20

ID: HCD-040
Tipo: technical
Carril: fast
Motivo: Cambio de dependencias acotado y sin decisiones abiertas: pins-validate ya demuestra los contratos de ambos candidatos; la Constitución §0 pide hacerlo en un cambio separado con pruebas y vuelta atrás

## Intención
pi-gomemory instala pi-antigravity 0.14.0 y pi-cc-extensions 1.0.20, validados con pins-validate y probados en Pi real (cuota, sidebar y banner), de modo que pins-check queda al día y la release v1.14.0 pasa su propio control sin forzarlo


## Contexto
[HCD-039](HCD-039-los-paquetes-de-pi-fijados-por-pi-gomemo.md) añadió `pins-check`, `pins-validate` y el aviso de `doctor`. Contra npm real, el 2026-11-17, `pins-check` detecta `pi-antigravity` 0.12.0 → 0.14.0 y `pi-cc-extensions` 1.0.16 → 1.0.20, y `pins-validate` da A1, A3 y C1 como cumplidos en los dos candidatos. La Constitución §0 exige que una actualización de dependencias vaya en un cambio separado, con pruebas y plan de vuelta atrás; esta unidad es ese cambio.

## Alcance
Incluido: subir `AntigravityVersion` a `0.14.0` y `CCExtensionsVersion` a `1.0.20`, con la evidencia de `pins-validate` y `pins-check`, la prueba en Pi real y las referencias de versión de la documentación. En detalle:
- `pkg/installer/plugin.go:16-17`: las dos constantes. `EnsurePinnedPackages` reescribe la entrada de `settings.json` en el siguiente `install` o `update`, y `pi install` instala la versión nueva.
- Evidencia de los candidatos: `PINS_CANDIDATE=pi-antigravity@0.14.0` y `PINS_CANDIDATE=pi-cc-extensions@1.0.20` con `TestPinsValidate`, y `TestPinsCheck` en verde contra npm real después del cambio.
- Prueba en Pi real, en tu máquina, después de `pi-gomemory-setup install` con el binario nuevo:
  - `pi list` (o `settings.json`) muestra `npm:pi-antigravity@0.14.0` y `npm:pi-cc-extensions@1.0.20`, y `~/.pi/agent/npm/node_modules/*/package.json` las mismas versiones;
  - `pi-gomemory-setup usage` muestra las cuotas de Antigravity;
  - en Pi, el sidebar muestra el bloque de cuota de Antigravity con un modelo `antigravity/*`;
  - el banner de gomemory sigue visible (`showStartupHeader` en `false`) y `doctor` no avisa del banner;
  - `doctor` muestra `✓` en las dos líneas de paquetes fijados.
- Referencias de versión de la documentación que describen los pins como actuales: `README.md:179`, `docs/architecture.md:68` y `docs/Guia-PI-GOMEMORY-v1.13.html:488,661`. Las notas de releases y el CHANGELOG son historia y no cambian.
- `uninstall --packages` con una entrada fijada por una versión anterior de pi-gomemory: hoy `planPinnedPackages` (`pkg/installer/uninstall.go:741-744`) solo retira la entrada que coincide exactamente con el pin actual, así que tras subirlo una instalación que conserve `npm:pi-antigravity@0.12.0` dejaría esa entrada y el paquete sin avisar. Se corrige en este cambio (Q1): los pins anteriores de pi-gomemory (`pi-antigravity@0.12.0` y `pi-cc-extensions@1.0.16`, y en adelante cada pin que se sustituya) se registran en el código, y `uninstall --packages` los trata como el pin actual: los revalida justo antes de escribir con `pinnedEntryUnchanged`, ejecuta `pi remove` con esa fuente y retira la entrada. Una entrada con otra versión que pi-gomemory nunca fijó se sigue conservando, como hasta ahora.

- H-01 de la [ACR/ADM del 2026-11-17](../docs/audits/2026-11-17-acr-adm.md) (Q2): una caché de dependencias anterior al cambio oculta el pin nuevo, porque `DepsStatus` (`pkg/installer/deps.go`) filtra con la `latest` guardada antes de consultar los probes y `DepsStatusFor` no sustituye ese objetivo. Con `current=latest=0.12.0` en caché y el pin compilado en 0.14.0, `deps-update`, `deps-status` y el pie no ofrecen la actualización hasta que la caché se refresque. Se corrige en este cambio: `DepsStatusFor` recorre la caché completa, toma `Latest` del pin compilado para cada paquete fijado y solo después relee `Current` y decide con `Outdated()`. Se ofrece exactamente la fuente compilada; los demás componentes no cambian. El reproductor de la auditoría (`TestACRADM01NewPinOverridesOldCache`) pasa a la suite.

Excluido: H-02 a H-05 de la misma auditoría (irán en otra unidad, Q2), cambiar el mecanismo de HCD-039, adoptar modelos nuevos del catálogo de 0.14.0 como predeterminados, la release v1.14.0 y cualquier otro paquete. En detalle:
- El catálogo de 0.14.0 trae modelos nuevos (por ejemplo, `gemini-4-hydrogen`). pi-gomemory no fija ningún modelo (`settings.json` solo tiene `modelThinkingLevels` de la persona), así que esta unidad no elige ninguno; la prueba en Pi real solo confirma que el modelo que usas sigue respondiendo.
- La release v1.14.0, con HCD-039 y esta unidad, va después del cierre por `/gitlab-release`.

## ¿Qué pasa si no lo hacemos?
La release v1.14.0 se detiene en su propio `pins-check` y habría que publicarla con `PI_GOMEMORY_ALLOW_STALE_PINS=1`, estrenando el control saltándolo. `doctor` avisaría del desfase en cada máquina con la versión nueva, y la persona seguiría sin las correcciones y el catálogo de modelos de `pi-antigravity` 0.14.0.

## Alternativa más barata
- Registrar 0.14.0 y 1.0.20 en `RejectedVersions` para callar `pins-check`: sería falso, porque `pins-validate` los da por buenos y no hay motivo de rechazo.
- Subir solo `pi-antigravity`, que es el paquete con más desfase: deja a `pi-cc-extensions` atrás y la release seguiría deteniéndose por él.
Subir los dos en el mismo cambio cuesta lo mismo y deja el control en verde.

## Vuelta atrás
Volver a poner `0.12.0` y `1.0.16` en `plugin.go:16-17` y ejecutar `pi-gomemory-setup install`: `EnsurePinnedPackages` reescribe la entrada y `pi install` reinstala la versión anterior.


## Aclaraciones
- [x] Q1. Tras subir los pins, ¿uninstall --packages retira también las entradas de pins anteriores escritas por pi-gomemory (npm:pi-antigravity@0.12.0, npm:pi-cc-extensions@1.0.16), o las conserva y avisa de que quedan? → Retirar pins históricos: los pins anteriores de pi-gomemory (0.12.0 y 1.0.16, y los que se sustituyan en adelante) se registran en el código y uninstall --packages los trata como el pin actual: revalidación antes de escribir, pi remove y retirada de la entrada.
- [x] Q2. ¿Se cierra en HCD-040 el hallazgo H-01 de la ACR/ADM 2026-11-17 (una caché antigua oculta el pin nuevo en deps-update) y los demás (H-02 a H-05) en otra unidad? → cierra H-01 dentro de HCD-040 y abre otra HCD para el resto

## Checklist
- [x] 1. Subir `AntigravityVersion` a 0.14.0 y `CCExtensionsVersion` a 1.0.20 en `pkg/installer/plugin.go` → go test ./pkg/installer -run '^TestHCD040PinsBumped$' -count=1
- [x] 2. Registrar los pins históricos (`pi-antigravity@0.12.0`, `pi-cc-extensions@1.0.16`) y retirarlos en `uninstall --packages` con revalidación y `pi remove` (Q1) → go test ./pkg/installer -run '^TestHCD040HistoricPins$' -count=1
- [x] 3. Validar los dos candidatos contra el registro npm real con `pins-validate` → PINS_CANDIDATE=pi-antigravity@0.14.0 go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$' && PINS_CANDIDATE=pi-cc-extensions@1.0.20 go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$'
- [x] 4. Dejar `pins-check` en verde contra el registro npm real → go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsCheck$'
- [x] 5. Actualizar las referencias de versión de `README.md:179`, `docs/architecture.md:68` y `docs/Guia-PI-GOMEMORY-v1.13.html:488,661` → ! grep -nE 'pi-antigravity@0\.12\.0|pi-cc-extensions@1\.0\.16|pi-antigravity 0\.12\.0|pi-cc-extensions 1\.0\.16' README.md docs/architecture.md docs/Guia-PI-GOMEMORY-v1.13.html
- [ ] 6. Instalar en `~/.pi/agent` con el binario nuevo y comprobar las versiones instaladas, la cuota y las líneas de `doctor` → go build -o /tmp/hcd040-setup . && /tmp/hcd040-setup install >/dev/null && grep -q '"npm:pi-antigravity@0.14.0"' ~/.pi/agent/settings.json && grep -q '"npm:pi-cc-extensions@1.0.20"' ~/.pi/agent/settings.json && grep -q '"version": "0.14.0"' ~/.pi/agent/npm/node_modules/pi-antigravity/package.json && grep -q '"version": "1.0.20"' ~/.pi/agent/npm/node_modules/pi-cc-extensions/package.json && /tmp/hcd040-setup usage >/dev/null && /tmp/hcd040-setup deps-refresh && /tmp/hcd040-setup doctor | grep -c 'al día con npm' | grep -qx 2
- [ ] 7. Observar en Pi real (persona) el bloque de cuota de Antigravity en el sidebar con un modelo `antigravity/*`, el banner de gomemory visible y una respuesta del modelo que usas → observación manual de la persona en Pi tras `/reload`
- [x] 8. Mantener verde la suite completa con el detector de carreras → go test -race ./... -count=1
- [x] 9. Mantener limpio el análisis estático con y sin la etiqueta `pins` → go vet ./... && go vet -tags pins ./...
- [x] 10. Ofrecer en `DepsStatusFor` el pin compilado aunque la caché guarde una `latest` anterior, sin cambiar los demás componentes (H-01, Q2) → go test ./pkg/installer -run '^(TestHCD040StaleCacheOffersNewPin|TestACRADM01NewPinOverridesOldCache)$' -count=1

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-17 07:35 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD040PinsBumped$' -count=1` (salida 1)
      --- FAIL: TestHCD040PinsBumped (0.00s)
          hcd040_test.go:10: pi-antigravity: 0.12.0 / npm:pi-antigravity@0.12.0, se espera 0.14.0
          hcd040_test.go:13: pi-cc-extensions: 1.0.16 / npm:pi-cc-extensions@1.0.16, se espera 1.0.20
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.563s
      FAIL
- 2026-11-17 07:35 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD040PinsBumped$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.214s
- 2026-11-17 07:36 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD040HistoricPins$' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd040_test.go:45:22: undefined: historicPinnedSources
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 07:37 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD040HistoricPins$' -count=1` (salida 1)
      /TestHCD040HistoricPins1149283905/001/.pi-gomemory/skills/verify-loop — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD040HistoricPins1149283905/001/.pi/agent/mcp.json › mcpServers.gomemory — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD040HistoricPins1149283905/001/.pi/agent/mcp.json › mcpServers.codegraph — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD040HistoricPins1149283905/001/.pi/agent/mcp.json › mcpServers.codebase-memory — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD040HistoricPins1149283905/001/.pi/agent/AGENTS.md › bloque pi-gomemory — protocolo escrito por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD040HistoricPins1149283905/001/.pi/agent/settings.json › theme — tema gomemory activado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD040HistoricPins1149283905/001/.pi/agent/settings.json › skills (~/.pi-gomemory/skills) — registro del repositorio de skills
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD040HistoricPins1149283905/001/.pi/agent/pi-cc-extensions.json — configuración por defecto de pi-cc-extensions
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD040HistoricPins1149283905/001/.pi-gomemory/manifest.json — manifiesto de instalación (se retira si no hay fallos)
              Resumen: 40 retirados, 0 conservados, 0 omitidos, 0 fallidos.
              ✓ pi-gomemory desinstalado. Para reinstalar: pi-gomemory-setup install
          hcd040_test.go:37: falta «pi remove npm:pi-antigravity@0.12.0»; llamadas:
          hcd040_test.go:37: falta «pi remove npm:pi-cc-extensions@1.0.16»; llamadas:
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.828s
      FAIL
- 2026-11-17 07:37 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD040HistoricPins$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.956s
- 2026-11-17 07:37 -05:00 · tarea 3 · verde · caducada · `PINS_CANDIDATE=pi-antigravity@0.14.0 go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$' && PINS_CANDIDATE=pi-cc-extensions@1.0.20 go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$'` (salida 0)
      === RUN   TestPinsValidate
          pins_validate_test.go:28: pi-antigravity@0.14.0 A1: cumplido (package/src/auth/accounts.ts)
          pins_validate_test.go:28: pi-antigravity@0.14.0 A3: cumplido (package/src/models/models.ts)
      --- PASS: TestPinsValidate (0.57s)
      PASS
      ok  	pi-gomemory/pkg/installer	1.157s
      === RUN   TestPinsValidate
          pins_validate_test.go:28: pi-cc-extensions@1.0.20 C1: cumplido (package/extensions/config/config.ts)
      --- PASS: TestPinsValidate (0.44s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.605s
- 2026-11-17 07:37 -05:00 · tarea 4 · verde · caducada · `go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsCheck$'` (salida 0)
      === RUN   TestPinsCheck
          pins_check_test.go:20: pi-antigravity: al día (0.14.0)
          pins_check_test.go:20: pi-cc-extensions: al día (1.0.20)
      --- PASS: TestPinsCheck (0.22s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.378s
- 2026-11-17 07:37 -05:00 · tarea 5 · rojo observado · caducada · `! grep -nE 'pi-antigravity@0\.12\.0|pi-cc-extensions@1\.0\.16|pi-antigravity 0\.12\.0|pi-cc-extensions 1\.0\.16' README.md docs/architecture.md docs/Guia-PI-GOMEMORY-v1.13.html` (salida 1)
      README.md:179:`pi-antigravity` y `pi-cc-extensions` se instalan fijados a la versión del catálogo oficial de Pi (`npm:pi-antigravity@0.12.0`, `npm:pi-cc-extensions@1.0.16`); cambian solo con una release nueva de pi-gomemory.
      docs/architecture.md:68:Para conectar Pi con los modelos de Google AI Pro mediante Antigravity, el instalador registra el paquete `npm:pi-antigravity@0.12.0` en `settings.json` (junto con `npm:pi-cc-extensions@1.0.16`). Ambos van fijados a la versión del catálogo oficial de Pi: Pi no reinstala un paquete con versión exacta. Además, inyecta la dependencia `typebox: "*"` en la sección `peerDependencies` de la configuración para evitar avisos molestos de npm al resolver tipos.
      docs/Guia-PI-GOMEMORY-v1.13.html:488:      <p><strong>Versiones comprobadas</strong>pi-gomemory 1.13.8 · Pi 1.5.2 · gomemory 2.54.1 · pi-antigravity 0.12.0 · pi-cc-extensions 1.0.16</p>
      docs/Guia-PI-GOMEMORY-v1.13.html:661:      <tr><td><strong>Plugins</strong></td><td><code>pi-antigravity@0.12.0</code> y <code>pi-cc-extensions@1.0.16</code>, registro de skills y tema</td><td><code>~/.pi/agent/settings.json</code></td></tr>
- 2026-11-17 07:38 -05:00 · tarea 5 · verde · caducada · `! grep -nE 'pi-antigravity@0\.12\.0|pi-cc-extensions@1\.0\.16|pi-antigravity 0\.12\.0|pi-cc-extensions 1\.0\.16' README.md docs/architecture.md docs/Guia-PI-GOMEMORY-v1.13.html` (salida 0)
- 2026-11-17 07:38 -05:00 · tarea 8 · verde · caducada · `go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.223s
      ok  	pi-gomemory/pkg/installer	59.388s
- 2026-11-17 07:39 -05:00 · tarea 9 · verde · caducada · `go vet ./... && go vet -tags pins ./...` (salida 0)
- 2026-11-17 07:40 -05:00 · tarea 6 · falló · caducada · `go build -o /tmp/hcd040-setup . && /tmp/hcd040-setup install >/dev/null && grep -q '"npm:pi-antigravity@0.14.0"' ~/.pi/agent/settings.json && grep -q '"npm:pi-cc-extensions@1.0.20"' ~/.pi/agent/settings.json && grep -q '"version": "0.14.0"' ~/.pi/agent/npm/node_modules/pi-antigravity/package.json && grep -q '"version": "1.0.20"' ~/.pi/agent/npm/node_modules/pi-cc-extensions/package.json && /tmp/hcd040-setup usage >/dev/null && /tmp/hcd040-setup deps-refresh && /tmp/hcd040-setup doctor | grep -c 'al día con npm' | grep -qx 2` (salida 1)
      [38;2;244;198;107mError consultando cuota: no se pudo obtener la cuota (¿token caducado? abre pi para renovarlo): https://cloudcode-pa.googleapis.com: status 401[0m
- 2026-11-17 07:54 -05:00 · tarea 10 · rojo observado · caducada · `go test ./pkg/installer -run '^(TestHCD040StaleCacheOffersNewPin|TestACRADM01NewPinOverridesOldCache)$' -count=1` (salida 1)
      --- FAIL: TestACRADM01NewPinOverridesOldCache (0.00s)
          hcd040_test.go:22: old cache hides new approved pin 0.14.0: got []
      --- FAIL: TestHCD040StaleCacheOffersNewPin (0.00s)
          --- FAIL: TestHCD040StaleCacheOffersNewPin/vigente (0.00s)
              hcd040_test.go:63: se ofrece el pin compilado y el resto no cambia:
                   got pi-gomemory→v1.1.0,mem→2.0.0
                  want pi-gomemory→v1.1.0,mem→2.0.0,pi-antigravity→0.14.0,pi-cc-extensions→1.0.20
          --- FAIL: TestHCD040StaleCacheOffersNewPin/caducada (0.00s)
              hcd040_test.go:63: se ofrece el pin compilado y el resto no cambia:
                   got pi-gomemory→v1.1.0,mem→2.0.0
                  want pi-gomemory→v1.1.0,mem→2.0.0,pi-antigravity→0.14.0,pi-cc-extensions→1.0.20
          hcd040_test.go:84: con el pin instalado no se ofrece nada, aunque la caché diga 0.99.0: pi-antigravity→0.99.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.309s
      FAIL
- 2026-11-17 07:54 -05:00 · tarea 10 · verde · vigente · `go test ./pkg/installer -run '^(TestHCD040StaleCacheOffersNewPin|TestACRADM01NewPinOverridesOldCache)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.212s

## Siguiente
pendiente

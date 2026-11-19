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

Validación temporal autorizada en Q3: preparar el workaround de PR #153 únicamente en `/tmp/pi-agy-validation.ZgzNjn/prompt-guard/`, con pruebas offline y cargas explícitas mediante `pi -e` en HOME aislado. Comprobar la conservación íntegra del texto, metadatos y partes de herramientas; idempotencia; opt-out; exclusión de otros proveedores e imágenes; respuesta de Pi; bucle real de lectura y bloqueo por hook. Durante la fase temporal Q3 no se copió la extensión a rutas auto-descubiertas. No cambiar instrucciones o credenciales originales y no contar estas pruebas como observación de UI de la tarea 7. Rigen Constitución §10, §18 y §21 y Playbook HCD-TASK-01, HCD-TDD-01 y HCD-EVIDENCE-01.

Instalación local autorizada en Q4: instalar únicamente `~/.pi/agent/extensions/antigravity-prompt-guard.js`, sin modificar el plugin npm, settings ni credenciales. Mantener la transformación ya probada, sin instrumentación ni registro de prompts, y validar de nuevo ese archivo exacto con las pruebas offline. Comprobar la auto-discovery mediante peticiones con la configuración real de Pi, sin cargar explícitamente la extensión; Hydrogen y Sonnet deben responder y el control `ANTIGRAVITY_PROMPT_GUARD=off` debe reproducir el rechazo. Las pruebas de humo deshabilitan herramientas y conexión MCP para evitar efectos laterales, pero conservan contexto, skills y extensiones globales. La reducción de prioridad se acepta en Q4 como mitigación provisional, no como equivalencia de seguridad. Desactivar mediante esa variable en un nuevo proceso, o retirar el archivo y ejecutar `/reload`. Q4 no autoriza una release ni sustituye la observación de UI de la tarea 7.

Criterio vigente de instalación acordado en Q5: la tarea 6 verifica, sin reinstalar ni degradar, los paquetes ya operativos `pi-antigravity@0.14.0` y `pi-cc-extensions@1.0.21`, su registro en settings, una respuesta real de cuota con buckets válidos y las filas de `doctor` para ambos pins. No ejecuta `install`, `deps-update` ni modifica credenciales, configuración o el workaround. Las referencias a `1.0.20` en el contexto y las pruebas históricas de la subida original se conservan como historia; no se cambia ningún pin del código. El script de comprobación omite cuentas, tokens y cuerpos completos de las respuestas.

Reconstrucción reproducible autorizada en Q6: conservar el arnés offline y una copia de referencia en `tests/antigravity-prompt-guard/`, integrados en la suite mediante `pkg/installer/hcd040_prompt_guard_test.go`. La tarea 11 registra un ciclo nuevo sobre dos versiones de esa referencia: primero una implementación válida sin transformación, después la transformación que coincide con la extensión local ya comprobada. El rojo nuevo no se presenta como recuperación de evidencia histórica. La copia y las pruebas no son recursos de `embed.go`, no se añaden a `tuiResources`, no instalan nada y no acceden a credenciales ni a la red. Las comprobaciones locales de la tarea 13 podrán seleccionar explícitamente el archivo instalado usando el mismo arnés archivado. Las demás pruebas reales siguen siendo comprobaciones locales opt-in, no pruebas de CI. Q6 no autoriza commit, push ni release y no cambia la extensión operativa, las reglas del motor ni su base de datos.

Excluido: instalar el workaround en otras máquinas o publicarlo sin autorización posterior; H-02 a H-05 de la misma auditoría (irán en otra unidad, Q2), cambiar el mecanismo de HCD-039, adoptar modelos nuevos del catálogo de 0.14.0 como predeterminados, la release v1.14.0 y cualquier otro paquete. En detalle:
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
- [x] Q3. Para desbloquear la prueba real de Antigravity (tarea 7), ¿autorizas probar e instalar una extensión reversible basada en el workaround upstream PR #153, que conserva todos los bytes de las instrucciones pero traslada su contenido al primer mensaje de usuario (menor prioridad que systemInstruction), o prefieres mantener la prioridad actual y esperar una corrección del proveedor? La prueba HTTP aislada A/B/A del traslado ya dio 429/200/429; aún faltan pruebas de herramientas y UI antes de cerrar. → 1 — Validar primero el workaround en un entorno temporal, sin instalarlo todavía. Se reconoce que conserva el texto completo pero traslada las instrucciones al mensaje de usuario, con menor prioridad. La instalación queda fuera de esta autorización.
- [x] Q4. Q3 autorizó únicamente la validación temporal y excluyó la instalación. Tras pasar las pruebas, ¿autorizas instalar localmente la extensión reversible en ~/.pi/agent/extensions/, aceptando que las instrucciones trasladadas al primer mensaje de usuario tienen menor prioridad que systemInstruction, sin quitar su texto ni modificar credenciales, o prefieres mantener solo la prueba temporal y no instalar? La instalación local no implica publicar una release ni cerrar HCD-040 sin verificar UI y demás pendientes. → 1
- [x] Q5. La tarea 6 exige pi-cc-extensions 1.0.20 y reinstalar, pero el código vigente y la instalación v1.15.2 ya usan pi-antigravity 0.14.0 y pi-cc-extensions 1.0.21. ¿Actualizamos el criterio de la tarea 6 para verificar la instalación vigente 0.14.0/1.0.21, cuota y doctor sin reinstalar ni degradar (recomendado), o conservamos el criterio histórico 1.0.20 y lo validamos únicamente en un HOME temporal sin tocar la instalación operativa? No se cambiará ningún pin ni se cerrará la unidad automáticamente. → 1
- [x] Q6. Las 14 tareas están verificadas, pero el motor bloquea el cierre por RED-MISSING de la tarea 11: el código temporal de /tmp no forma parte de la huella Git. ¿Autorizas conservar en el repositorio únicamente el arnés reproducible de pruebas y una copia de referencia del guard, sin integrarlo en el instalador ni cambiar la extensión operativa, y recrear explícitamente un ciclo rojo/verde aislado con esas versiones identificadas (recomendado), o prefieres mantener todos los artefactos fuera del repositorio y dejar HCD-040 bloqueada? La recreación se registrará como nueva reproducción, no como evidencia histórica recuperada; no se modificarán las reglas del motor, la base de datos ni archivos arbitrarios para alterar la huella, y no autoriza commit, push o release. → 1

## Checklist
- [x] 1. Subir `AntigravityVersion` a 0.14.0 y `CCExtensionsVersion` a 1.0.20 en `pkg/installer/plugin.go` → go test ./pkg/installer -run '^TestHCD040PinsBumped$' -count=1
- [x] 2. Registrar los pins históricos (`pi-antigravity@0.12.0`, `pi-cc-extensions@1.0.16`) y retirarlos en `uninstall --packages` con revalidación y `pi remove` (Q1) → go test ./pkg/installer -run '^TestHCD040HistoricPins$' -count=1
- [x] 3. Validar los dos candidatos contra el registro npm real con `pins-validate` → PINS_CANDIDATE=pi-antigravity@0.14.0 go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$' && PINS_CANDIDATE=pi-cc-extensions@1.0.20 go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$'
- [x] 4. Dejar `pins-check` en verde contra el registro npm real → go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsCheck$'
- [x] 5. Actualizar las referencias de versión de `README.md:179`, `docs/architecture.md:68` y `docs/Guia-PI-GOMEMORY-v1.13.html:488,661` → ! grep -nE 'pi-antigravity@0\.12\.0|pi-cc-extensions@1\.0\.16|pi-antigravity 0\.12\.0|pi-cc-extensions 1\.0\.16' README.md docs/architecture.md docs/Guia-PI-GOMEMORY-v1.13.html
- [x] 6. Verificar en `~/.pi/agent` la instalación vigente de Antigravity 0.14.0 y CC Extensions 1.0.21, su registro, la cuota real y las filas de `doctor`, sin reinstalar ni cambiar archivos originales (Q5) → node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/verify-current-installation.mjs
- [x] 7. Observar en Pi real (persona) el bloque de cuota de Antigravity en el sidebar con un modelo `antigravity/*`, el banner de gomemory visible y una respuesta del modelo que usas → observación manual de la persona en Pi tras `/reload`
- [x] 8. Mantener verde la suite completa con el detector de carreras → go test -race ./... -count=1
- [x] 9. Mantener limpio el análisis estático con y sin la etiqueta `pins` → go vet ./... && go vet -tags pins ./...
- [x] 10. Ofrecer en `DepsStatusFor` el pin compilado aunque la caché guarde una `latest` anterior, sin cambiar los demás componentes (H-01, Q2) → go test ./pkg/installer -run '^(TestHCD040StaleCacheOffersNewPin|TestACRADM01NewPinOverridesOldCache)$' -count=1
- [x] 11. Validar la copia de referencia de `tests/antigravity-prompt-guard/guard.mjs` con un nuevo rojo/verde trazable (Q6), sin pérdida de instrucciones ni partes de herramientas, con idempotencia, opt-out y límites por proveedor, mediante `pkg/installer/hcd040_prompt_guard_test.go` → go test ./pkg/installer -run '^TestHCD040PromptGuardRegression$' -count=1 -v
- [x] 12. Validar el workaround de `/tmp/pi-agy-validation.ZgzNjn/prompt-guard/guard.mjs` en Pi real temporal con controles sin guard, opt-out, respuesta de dos familias de modelos, bucle de lectura y bloqueo de ruta, sin alterar la instalación original → node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/validate-pi.mjs
- [x] 13. Validar el archivo local instalado `~/.pi/agent/extensions/antigravity-prompt-guard.js` con las pruebas offline, sin instrumentación ni dependencias nuevas → env AGY_GUARD_MODULE=file:///Users/josegomezj/.pi/agent/extensions/antigravity-prompt-guard.js node --test tests/antigravity-prompt-guard/guard.test.mjs
- [x] 14. Verificar auto-discovery y respuestas de Hydrogen y Sonnet con la configuración real de Pi, y reproducir el rechazo con opt-out sin cambiar credenciales ni el plugin → node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/validate-installed.mjs

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
- 2026-11-17 07:54 -05:00 · tarea 10 · verde · caducada · `go test ./pkg/installer -run '^(TestHCD040StaleCacheOffersNewPin|TestACRADM01NewPinOverridesOldCache)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.212s
- 2026-11-19 04:46 -05:00 · tarea 7 · falló · caducada · `node /tmp/pi-agy-validation.ZgzNjn/capture-http.mjs` (salida 1)
      {"exitCode":1,"stdout":"","stderr":"Antigravity API error (request-failed, endpoint=https://cloudcode-pa.googleapis.com, project=aicode-consumers, runtimeModel=gemini-4-hydrogen-xhigh, matched=static:gemini-4-hydrogen-low, available=gemini-4-neon,claude-sonnet-5-5,gemini-3.8-flash,gemini-4-hydrogen,gemini-4-argon,gemini-4-hydrogen-sonar,claude-opus-5-5,gemini-4-hydrogen-tiered,gpt-oss-120b): This model is unavailable right now (Resource has been exhausted (e.g. check quota).). Next: switch to gemini-3.8-flash or another Flash model. This is not necessarily an account quota wall.\n"}
      RAW_GOOGLE_RESPONSE {"status":429,"runtimeModel":"gemini-4-hydrogen-xhigh","lastText":"{\n  \"error\": {\n    \"code\": 429,\n    \"message\": \"Resource has been exhausted (e.g. check quota).\",\n    \"status\": \"RESOURCE_EXHAUSTED\"\n  }\n}\n","envelope":{"keys":["project","model","request","requestType","userAgent","requestId"],"model":"gemini-4-hydrogen-xhigh","project":"aicode-consumers","requestType":"agent","userAgent":"antigravity","requestKeys":["contents","systemInstruction","generationConfig","sessionId","labels"],"generationConfig":{"thinkingConfig":{"includeThoughts":true,"thinkingBudget":1000},"maxOutputTokens":262144}}}
      {"originalAuthUnchanged":true,"temporaryCredentialsRemoved":true}
- 2026-11-19 04:52 -05:00 · tarea 11 · rojo observado · caducada · `node --test /tmp/pi-agy-validation.ZgzNjn/prompt-guard/guard.test.mjs` (salida 1)
            at Test.run (node:internal/test_runner/test:1467:12)
            at async Test.processPendingSubtests (node:internal/test_runner/test:974:7) {
          generatedMessage: true,
          code: 'ERR_ASSERTION',
          actual: { project: 'aicode-consumers', model: 'gemini-4-hydrogen-low', requestType: 'agent', requestId: 'test-id', request: { systemInstruction: [Object], contents: [Array], tools: [Array], labels: [Object], generationConfig: [Object] } },
          expected: { project: 'aicode-consumers', model: 'gemini-4-hydrogen-low', requestType: 'agent', requestId: 'test-id', request: { systemInstruction: [Object], contents: [Array], tools: [Array], labels: [Object], generationConfig: [Object] } },
          operator: 'notStrictEqual',
          diff: 'simple'
        }
      test at ../../../../../private/tmp/pi-agy-validation.ZgzNjn/prompt-guard/guard.test.mjs:41:1
      ✖ registers a working Pi hook scoped to Antigravity (0.158041ms)
        TypeError [Error]: Cannot read properties of undefined (reading 'request')
            at TestContext.<anonymous> (file:///private/tmp/pi-agy-validation.ZgzNjn/prompt-guard/guard.test.mjs:43:88)
            at Test.runInAsyncScope (node:async_hooks:226:14)
            at Test.run (node:internal/test_runner/test:1402:25)
            at Test.processPendingSubtests (node:internal/test_runner/test:974:18)
            at Test.postRun (node:internal/test_runner/test:1542:19)
            at Test.run (node:internal/test_runner/test:1467:12)
            at async Test.processPendingSubtests (node:internal/test_runner/test:974:7)
- 2026-11-19 04:52 -05:00 · tarea 11 · verde · caducada · `node --test /tmp/pi-agy-validation.ZgzNjn/prompt-guard/guard.test.mjs` (salida 0)
      ✔ passes through foreign provider undefined (0.074125ms)
      ✔ passes through image generation (0.063083ms)
      ✔ supports explicit opt-out (0.048625ms)
      ✔ passes through malformed or nontext input null (0.048583ms)
      ✔ passes through malformed or nontext input [] (0.031292ms)
      ✔ passes through malformed or nontext input {} (0.043542ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":null} (0.032708ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":{"systemInstruction":{"parts":[{"text":"text"},{"inlineData":{}}]},"contents":[]}} (0.059917ms)
      ✔ passes through blank system text (0.072083ms)
      ✔ passes through malformed conversation turns (0.059375ms)
      ✔ does not discard non-text system metadata (0.042833ms)
      ✔ registers a working Pi hook scoped to Antigravity (0.177333ms)
      ℹ tests 22
      ℹ suites 0
      ℹ pass 22
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 197.514292
- 2026-11-19 04:55 -05:00 · tarea 12 · verde · caducada · `node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/validate-pi.mjs` (salida 0)
      {"name":"baseline","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":0,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"timeout":false}
      {"name":"hydrogen","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":1,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"expectedReply":true,"timeout":false}
      {"name":"sonnet","model":"claude-sonnet-5-5","passed":true,"exitCode":0,"providerRequests":1,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"expectedReply":true,"timeout":false}
      {"name":"read-loop","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":2,"instructionsPreserved":true,"toolCalls":[{"tool":"read","allowed":true}],"functionResponseObserved":true,"expectedReply":true,"timeout":false}
      {"name":"blocked-read","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":1,"instructionsPreserved":true,"toolCalls":[{"tool":"read","allowed":false}],"functionResponseObserved":false,"timeout":false}
      {"name":"opt-out","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":0,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"timeout":false}
      {"overall":true,"originalFilesUnchanged":true,"temporaryCredentialsRemoved":true}
- 2026-11-19 05:00 -05:00 · tarea 13 · verde · caducada · `env AGY_GUARD_MODULE=file:///Users/josegomezj/.pi/agent/extensions/antigravity-prompt-guard.js node --test /tmp/pi-agy-validation.ZgzNjn/prompt-guard/guard.test.mjs` (salida 0)
      ✔ passes through foreign provider undefined (0.085417ms)
      ✔ passes through image generation (0.070291ms)
      ✔ supports explicit opt-out (0.052416ms)
      ✔ passes through malformed or nontext input null (0.789084ms)
      ✔ passes through malformed or nontext input [] (0.072334ms)
      ✔ passes through malformed or nontext input {} (0.060625ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":null} (0.068166ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":{"systemInstruction":{"parts":[{"text":"text"},{"inlineData":{}}]},"contents":[]}} (0.067083ms)
      ✔ passes through blank system text (0.080834ms)
      ✔ passes through malformed conversation turns (0.056833ms)
      ✔ does not discard non-text system metadata (0.044333ms)
      ✔ registers a working Pi hook scoped to Antigravity (0.156958ms)
      ℹ tests 22
      ℹ suites 0
      ℹ pass 22
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 196.916375
- 2026-11-19 05:00 -05:00 · tarea 14 · verde · caducada · `node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/validate-installed.mjs` (salida 0)
      {"model":"gemini-4-hydrogen","guardDisabled":false,"passed":true,"exitCode":0,"stopReason":"stop","expectedReply":true,"guardLoadedExplicitly":false,"timeout":false}
      {"model":"claude-sonnet-5-5","guardDisabled":false,"passed":true,"exitCode":0,"stopReason":"stop","expectedReply":true,"guardLoadedExplicitly":false,"timeout":false}
      {"model":"gemini-4-hydrogen","guardDisabled":true,"passed":true,"exitCode":0,"stopReason":"error","expectedReply":false,"guardLoadedExplicitly":false,"timeout":false}
      {"overall":true,"unchanged":[{"path":"auth.json","unchanged":true},{"path":"settings.json","unchanged":true},{"path":"antigravity-accounts.json","unchanged":true},{"path":"npm/node_modules/pi-antigravity/src/stream/stream.ts","unchanged":true}]}
- 2026-11-19 05:03 -05:00 · tarea 7 · manual · caducada
      Validación de la persona en Pi real: tras enviar «prueba de hydrogen con agy» y solicitar confirmación del sidebar con cuota y del banner, responde literalmente «veo todo en el sidebar con cuota y todo OK». Evidencia manual de funcionamiento y UI aportada por la persona; no es una observación visual realizada por el agente.
- 2026-11-19 05:09 -05:00 · tarea 6 · verde · caducada · `node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/verify-current-installation.mjs` (salida 0)
      {"test":"installed_package","passed":true,"name":"pi-antigravity","expected":"0.14.0","installed":"0.14.0","registered":true}
      {"test":"installed_package","passed":true,"name":"pi-cc-extensions","expected":"1.0.21","installed":"1.0.21","registered":true}
      {"test":"live_quota","passed":true,"groups":3,"buckets":5}
      {"test":"doctor_exit","passed":true,"exitCode":0}
      {"test":"doctor_package","passed":true,"name":"pi-antigravity","expected":"0.14.0","rowFound":true}
      {"test":"doctor_package","passed":true,"name":"pi-cc-extensions","expected":"1.0.21","rowFound":true}
      {"test":"original_files_unchanged","passed":true,"files":[{"path":"auth.json","unchanged":true},{"path":"settings.json","unchanged":true},{"path":"antigravity-accounts.json","unchanged":true},{"path":"extensions/antigravity-prompt-guard.js","unchanged":true},{"path":"npm/node_modules/pi-antigravity/src/stream/stream.ts","unchanged":true}]}
      {"overall":true,"reinstallPerformed":false,"packagesChanged":false}
- 2026-11-19 05:14 -05:00 · tarea 1,2,8,9,10 · verde · caducada · `go vet ./... && go vet -tags pins ./... && go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.791s
      ok  	pi-gomemory/pkg/installer	93.652s
- 2026-11-19 05:16 -05:00 · tarea 3 · verde · caducada · `PINS_CANDIDATE=pi-antigravity@0.14.0 go test -tags pins -count=1 -v ./pkg/installer -run "^TestPinsValidate$" && PINS_CANDIDATE=pi-cc-extensions@1.0.20 go test -tags pins -count=1 -v ./pkg/installer -run "^TestPinsValidate$"` (salida 0)
      === RUN   TestPinsValidate
          pins_validate_test.go:28: pi-antigravity@0.14.0 A1: cumplido (package/src/auth/accounts.ts)
          pins_validate_test.go:28: pi-antigravity@0.14.0 A3: cumplido (package/src/models/models.ts)
      --- PASS: TestPinsValidate (0.54s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.771s
      === RUN   TestPinsValidate
          pins_validate_test.go:28: pi-cc-extensions@1.0.20 C1: cumplido (package/extensions/config/config.ts)
      --- PASS: TestPinsValidate (0.52s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.718s
- 2026-11-19 05:16 -05:00 · tarea 4 · verde · caducada · `go test -tags pins -count=1 -v ./pkg/installer -run "^TestPinsCheck$"` (salida 0)
      === RUN   TestPinsCheck
          pins_check_test.go:20: pi-antigravity: al día (0.14.0)
          pins_check_test.go:20: pi-cc-extensions: al día (1.0.21)
      --- PASS: TestPinsCheck (0.16s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.363s
- 2026-11-19 05:16 -05:00 · tarea 5 · verde · caducada · `node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/verify-doc-pins.mjs` (salida 0)
      {"path":"README.md","exists":true,"obsoletePinsAbsent":true}
      {"path":"docs/architecture.md","exists":true,"obsoletePinsAbsent":true}
      {"path":"docs/Guia-PI-GOMEMORY-v1.13.html","exists":true,"obsoletePinsAbsent":true}
      {"overall":true}
- 2026-11-19 05:16 -05:00 · tarea 11,13 · verde · caducada · `node --test /tmp/pi-agy-validation.ZgzNjn/prompt-guard/guard.test.mjs && AGY_GUARD_MODULE=file:///Users/josegomezj/.pi/agent/extensions/antigravity-prompt-guard.js node --test /tmp/pi-agy-validation.ZgzNjn/prompt-guard/guard.test.mjs` (salida 0)
      ✔ passes through foreign provider undefined (0.076542ms)
      ✔ passes through image generation (0.067959ms)
      ✔ supports explicit opt-out (0.049833ms)
      ✔ passes through malformed or nontext input null (0.529709ms)
      ✔ passes through malformed or nontext input [] (0.038333ms)
      ✔ passes through malformed or nontext input {} (0.058083ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":null} (0.030458ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":{"systemInstruction":{"parts":[{"text":"text"},{"inlineData":{}}]},"contents":[]}} (0.039416ms)
      ✔ passes through blank system text (0.069542ms)
      ✔ passes through malformed conversation turns (0.058917ms)
      ✔ does not discard non-text system metadata (0.047209ms)
      ✔ registers a working Pi hook scoped to Antigravity (0.134042ms)
      ℹ tests 22
      ℹ suites 0
      ℹ pass 22
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 190.143
- 2026-11-19 05:16 -05:00 · tarea 12 · verde · caducada · `node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/validate-pi.mjs` (salida 0)
      {"name":"baseline","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":0,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"timeout":false}
      {"name":"hydrogen","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":1,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"expectedReply":true,"timeout":false}
      {"name":"sonnet","model":"claude-sonnet-5-5","passed":true,"exitCode":0,"providerRequests":1,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"expectedReply":true,"timeout":false}
      {"name":"read-loop","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":2,"instructionsPreserved":true,"toolCalls":[{"tool":"read","allowed":true}],"functionResponseObserved":true,"expectedReply":true,"timeout":false}
      {"name":"blocked-read","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":1,"instructionsPreserved":true,"toolCalls":[{"tool":"read","allowed":false}],"functionResponseObserved":false,"timeout":false}
      {"name":"opt-out","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":0,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"timeout":false}
      {"overall":true,"originalFilesUnchanged":true,"temporaryCredentialsRemoved":true}
- 2026-11-19 05:17 -05:00 · tarea 14 · falló · caducada · `node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/validate-installed.mjs` (salida 1)
      {"model":"gemini-4-hydrogen","guardDisabled":false,"passed":true,"exitCode":0,"stopReason":"stop","expectedReply":true,"guardLoadedExplicitly":false,"timeout":false}
      {"model":"claude-sonnet-5-5","guardDisabled":false,"passed":true,"exitCode":0,"stopReason":"stop","expectedReply":true,"guardLoadedExplicitly":false,"timeout":false}
      {"model":"gemini-4-hydrogen","guardDisabled":true,"passed":true,"exitCode":0,"stopReason":"error","expectedReply":false,"guardLoadedExplicitly":false,"timeout":false}
      {"overall":false,"unchanged":[{"path":"auth.json","unchanged":false},{"path":"settings.json","unchanged":true},{"path":"antigravity-accounts.json","unchanged":false},{"path":"npm/node_modules/pi-antigravity/src/stream/stream.ts","unchanged":true}]}
- 2026-11-19 05:17 -05:00 · tarea 7 · manual · caducada
      Se conserva como evidencia manual la declaración previa de la persona: «veo todo en el sidebar con cuota y todo OK», tras su prueba «prueba de hydrogen con agy» y la solicitud de confirmar sidebar con cuota y banner. No se afirma una nueva inspección visual por el agente. El criterio UI de la tarea 7 no cambió; Q5 ajustó solo la comprobación de instalación. La confirmación original consta en la evidencia 01a1f343-09ee-7586-9ecb-acc5ca91df2e y memoria 501.
- 2026-11-19 05:18 -05:00 · tarea 14 · verde · caducada · `node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/validate-installed.mjs` (salida 0)
      {"model":"gemini-4-hydrogen","guardDisabled":false,"passed":true,"exitCode":0,"stopReason":"stop","expectedReply":true,"guardLoadedExplicitly":false,"timeout":false}
      {"model":"claude-sonnet-5-5","guardDisabled":false,"passed":true,"exitCode":0,"stopReason":"stop","expectedReply":true,"guardLoadedExplicitly":false,"timeout":false}
      {"model":"gemini-4-hydrogen","guardDisabled":true,"passed":true,"exitCode":0,"stopReason":"error","expectedReply":false,"guardLoadedExplicitly":false,"timeout":false}
      {"overall":true,"unchanged":[{"path":"auth.json","unchanged":true},{"path":"settings.json","unchanged":true},{"path":"antigravity-accounts.json","unchanged":true},{"path":"npm/node_modules/pi-antigravity/src/stream/stream.ts","unchanged":true}]}
- 2026-11-19 05:30 -05:00 · tarea 11 · rojo observado · caducada · `go test ./pkg/installer -run "^TestHCD040PromptGuardRegression$" -count=1 -v` (salida 1)
                  actual: { project: 'aicode-consumers', model: 'gemini-4-hydrogen-low', requestType: 'agent', requestId: 'test-id', request: { systemInstruction: [Object], contents: [Array], tools: [Array], labels: [Object], generationConfig: [Object] } },
                  expected: { project: 'aicode-consumers', model: 'gemini-4-hydrogen-low', requestType: 'agent', requestId: 'test-id', request: { systemInstruction: [Object], contents: [Array], tools: [Array], labels: [Object], generationConfig: [Object] } },
                  operator: 'notStrictEqual',
                  diff: 'simple'
                }
              test at ../../tests/antigravity-prompt-guard/guard.test.mjs:41:1
              ✖ registers a working Pi hook scoped to Antigravity (0.188125ms)
                TypeError [Error]: Cannot read properties of undefined (reading 'request')
                    at TestContext.<anonymous> (file:///Users/josegomezj/home/rcw/pi-gomemory/tests/antigravity-prompt-guard/guard.test.mjs:43:88)
                    at Test.runInAsyncScope (node:async_hooks:226:14)
                    at Test.run (node:internal/test_runner/test:1402:25)
                    at Test.processPendingSubtests (node:internal/test_runner/test:974:18)
                    at Test.postRun (node:internal/test_runner/test:1542:19)
                    at Test.run (node:internal/test_runner/test:1467:12)
                    at async Test.processPendingSubtests (node:internal/test_runner/test:974:7)
      --- FAIL: TestHCD040PromptGuardRegression (0.44s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	1.140s
      FAIL
- 2026-11-19 05:31 -05:00 · tarea 11 · verde · vigente · `go test ./pkg/installer -run "^TestHCD040PromptGuardRegression$" -count=1 -v` (salida 0)
              ✔ passes through malformed or nontext input null (0.540583ms)
              ✔ passes through malformed or nontext input [] (0.042458ms)
              ✔ passes through malformed or nontext input {} (0.050375ms)
              ✔ passes through malformed or nontext input {"requestType":"agent","request":null} (0.033583ms)
              ✔ passes through malformed or nontext input {"requestType":"agent","request":{"systemInstruction":{"parts":[{"text":"text"},{"inlineData":{}}]},"contents":[]}} (0.044291ms)
              ✔ passes through blank system text (0.231208ms)
              ✔ passes through malformed conversation turns (0.061167ms)
              ✔ does not discard non-text system metadata (0.040125ms)
              ✔ registers a working Pi hook scoped to Antigravity (0.126875ms)
              ℹ tests 22
              ℹ suites 0
              ℹ pass 22
              ℹ fail 0
              ℹ cancelled 0
              ℹ skipped 0
              ℹ todo 0
              ℹ duration_ms 191.17775
      --- PASS: TestHCD040PromptGuardRegression (0.37s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.959s
- 2026-11-19 05:31 -05:00 · tarea 1,2,8,9,10 · verde · vigente · `go vet ./... && go vet -tags pins ./... && go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.335s
      ok  	pi-gomemory/pkg/installer	91.271s
- 2026-11-19 05:33 -05:00 · tarea 3 · verde · vigente · `PINS_CANDIDATE=pi-antigravity@0.14.0 go test -tags pins -count=1 -v ./pkg/installer -run "^TestPinsValidate$" && PINS_CANDIDATE=pi-cc-extensions@1.0.20 go test -tags pins -count=1 -v ./pkg/installer -run "^TestPinsValidate$"` (salida 0)
      === RUN   TestPinsValidate
          pins_validate_test.go:28: pi-antigravity@0.14.0 A1: cumplido (package/src/auth/accounts.ts)
          pins_validate_test.go:28: pi-antigravity@0.14.0 A3: cumplido (package/src/models/models.ts)
      --- PASS: TestPinsValidate (0.49s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.753s
      === RUN   TestPinsValidate
          pins_validate_test.go:28: pi-cc-extensions@1.0.20 C1: cumplido (package/extensions/config/config.ts)
      --- PASS: TestPinsValidate (1.09s)
      PASS
      ok  	pi-gomemory/pkg/installer	1.288s
- 2026-11-19 05:33 -05:00 · tarea 4 · verde · vigente · `go test -tags pins -count=1 -v ./pkg/installer -run "^TestPinsCheck$"` (salida 0)
      === RUN   TestPinsCheck
          pins_check_test.go:20: pi-antigravity: al día (0.14.0)
          pins_check_test.go:20: pi-cc-extensions: al día (1.0.21)
      --- PASS: TestPinsCheck (0.54s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.749s
- 2026-11-19 05:34 -05:00 · tarea 5 · verde · vigente · `node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/verify-doc-pins.mjs` (salida 0)
      {"path":"README.md","exists":true,"obsoletePinsAbsent":true}
      {"path":"docs/architecture.md","exists":true,"obsoletePinsAbsent":true}
      {"path":"docs/Guia-PI-GOMEMORY-v1.13.html","exists":true,"obsoletePinsAbsent":true}
      {"overall":true}
- 2026-11-19 05:34 -05:00 · tarea 6 · verde · vigente · `node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/verify-current-installation.mjs` (salida 0)
      {"test":"installed_package","passed":true,"name":"pi-antigravity","expected":"0.14.0","installed":"0.14.0","registered":true}
      {"test":"installed_package","passed":true,"name":"pi-cc-extensions","expected":"1.0.21","installed":"1.0.21","registered":true}
      {"test":"live_quota","passed":true,"groups":3,"buckets":5}
      {"test":"doctor_exit","passed":true,"exitCode":0}
      {"test":"doctor_package","passed":true,"name":"pi-antigravity","expected":"0.14.0","rowFound":true}
      {"test":"doctor_package","passed":true,"name":"pi-cc-extensions","expected":"1.0.21","rowFound":true}
      {"test":"original_files_unchanged","passed":true,"files":[{"path":"auth.json","unchanged":true},{"path":"settings.json","unchanged":true},{"path":"antigravity-accounts.json","unchanged":true},{"path":"extensions/antigravity-prompt-guard.js","unchanged":true},{"path":"npm/node_modules/pi-antigravity/src/stream/stream.ts","unchanged":true}]}
      {"overall":true,"reinstallPerformed":false,"packagesChanged":false}
- 2026-11-19 05:34 -05:00 · tarea 13 · verde · vigente · `AGY_GUARD_MODULE=file:///Users/josegomezj/.pi/agent/extensions/antigravity-prompt-guard.js node --test tests/antigravity-prompt-guard/guard.test.mjs` (salida 0)
      ✔ passes through foreign provider undefined (0.082375ms)
      ✔ passes through image generation (0.067458ms)
      ✔ supports explicit opt-out (0.051875ms)
      ✔ passes through malformed or nontext input null (0.061375ms)
      ✔ passes through malformed or nontext input [] (0.807458ms)
      ✔ passes through malformed or nontext input {} (0.087208ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":null} (0.0695ms)
      ✔ passes through malformed or nontext input {"requestType":"agent","request":{"systemInstruction":{"parts":[{"text":"text"},{"inlineData":{}}]},"contents":[]}} (0.056167ms)
      ✔ passes through blank system text (0.097875ms)
      ✔ passes through malformed conversation turns (0.087625ms)
      ✔ does not discard non-text system metadata (0.07525ms)
      ✔ registers a working Pi hook scoped to Antigravity (0.170167ms)
      ℹ tests 22
      ℹ suites 0
      ℹ pass 22
      ℹ fail 0
      ℹ cancelled 0
      ℹ skipped 0
      ℹ todo 0
      ℹ duration_ms 203.760542
- 2026-11-19 05:34 -05:00 · tarea 12 · verde · vigente · `node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/validate-pi.mjs` (salida 0)
      {"name":"baseline","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":0,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"timeout":false}
      {"name":"hydrogen","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":1,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"expectedReply":true,"timeout":false}
      {"name":"sonnet","model":"claude-sonnet-5-5","passed":true,"exitCode":0,"providerRequests":1,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"expectedReply":true,"timeout":false}
      {"name":"read-loop","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":2,"instructionsPreserved":true,"toolCalls":[{"tool":"read","allowed":true}],"functionResponseObserved":true,"expectedReply":true,"timeout":false}
      {"name":"blocked-read","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":1,"instructionsPreserved":true,"toolCalls":[{"tool":"read","allowed":false}],"functionResponseObserved":false,"timeout":false}
      {"name":"opt-out","model":"gemini-4-hydrogen","passed":true,"exitCode":0,"providerRequests":0,"instructionsPreserved":true,"toolCalls":[],"functionResponseObserved":false,"timeout":false}
      {"overall":true,"originalFilesUnchanged":true,"temporaryCredentialsRemoved":true}
- 2026-11-19 05:34 -05:00 · tarea 14 · verde · vigente · `node /tmp/pi-agy-validation.ZgzNjn/prompt-guard/validate-installed.mjs` (salida 0)
      {"model":"gemini-4-hydrogen","guardDisabled":false,"passed":true,"exitCode":0,"stopReason":"stop","expectedReply":true,"guardLoadedExplicitly":false,"timeout":false}
      {"model":"claude-sonnet-5-5","guardDisabled":false,"passed":true,"exitCode":0,"stopReason":"stop","expectedReply":true,"guardLoadedExplicitly":false,"timeout":false}
      {"model":"gemini-4-hydrogen","guardDisabled":true,"passed":true,"exitCode":0,"stopReason":"error","expectedReply":false,"guardLoadedExplicitly":false,"timeout":false}
      {"overall":true,"unchanged":[{"path":"auth.json","unchanged":true},{"path":"settings.json","unchanged":true},{"path":"antigravity-accounts.json","unchanged":true},{"path":"npm/node_modules/pi-antigravity/src/stream/stream.ts","unchanged":true}]}
- 2026-11-19 05:35 -05:00 · tarea 7 · manual · vigente
      Se conserva la observación ya aportada por la persona: «veo todo en el sidebar con cuota y todo OK», tras su prueba «prueba de hydrogen con agy» y la solicitud de confirmar sidebar con cuota y banner. La observación original consta en evidencia 01a1f343-09ee-7586-9ecb-acc5ca91df2e y memoria 501. Q6 añadió únicamente una referencia y pruebas offline, sin cambiar la extensión operativa ni la UI; la comparación de referencia e instalada y las verificaciones locales confirman que el código operativo sigue igual. No se afirma una nueva inspección visual por el agente.

## Siguiente
Cerrada con aprobación sobre la revisión 8b6e5f600761.

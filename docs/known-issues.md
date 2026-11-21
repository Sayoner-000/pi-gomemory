# Problemas conocidos

## Antigravity: HTTP 429 con OAuth y cuota disponibles

**Mitigación temporal desde v1.16.0.** En la instalación investigada, la comparación A/B/A con la misma credencial produjo 429 con las instrucciones originales de Pi, 200 al trasladarlas al primer mensaje de usuario y 429 al restaurarlas. También se reprodujo con versiones anteriores de Pi y del plugin; no se atribuye el fallo exclusivamente al pin 0.14.0 ni al hardening.

La [guía de Antigravity](antigravity.md) explica la referencia publicada, su desactivación y el riesgo: conservar el texto no conserva su prioridad ni garantiza protección equivalente frente a inyección de instrucciones. La prueba real recuperó Hydrogen, Sonnet y un ciclo de herramientas; la persona confirmó el panel y las cuotas. Esto no demuestra que todos los HTTP 429 tengan la misma causa. Los pins ya publicados en v1.14.0 no equivalían a ese smoke real; HCD-040 se cerró después de completarlo y obtener aprobación.

Desde v1.21.1 el pin es pi-antigravity 0.15.0, que corrige el problema upstream (pi-antigravity#152). En la prueba real sin el guard, 0.14.0 devolvió 429 en todos los casos y 0.15.0 respondió en todos. El guard local sigue instalado y es compatible con 0.15.0; retirarlo queda pendiente de decisión.

## ACR y ADM de v1.21.0

La auditoría ACR/ADM sobre `v1.21.0` (`2c10005`, veredicto `APPROVED` sin críticos ni altos) identificó cuatro defectos medios y tres bajos en el paso de instalación de herramientas (señales `SIGINT`/`SIGTERM`, borrado prematuro de temporales en `uninstall`, colisión entre procesos concurrentes, `PI_GOMEMORY_BIN_DIR` relativo, timeout con TTY, entradas de directorio en `extractExact` y versión literal en `scripts/clean-install-check.sh`). [HCD-064](../hcd/HCD-064-corregir-hallazgos-h-01-a-h-07-de-la-aud.md) corrige los siete en [v1.21.1](releases/v1.21.1.md).

## ACR y ADM de v1.16.0

La [auditoría de v1.16.0](audits/2026-11-19-acr-adm-v1.16.0.md) confirmó cinco defectos, entre ellos la pérdida de configuraciones MCP personalizadas al alternar `lite`. [HCD-052](../hcd/HCD-052-corregir-hallazgos-c-001-a-c-005-de-la-a.md) corrige los cinco en [v1.17.0](releases/v1.17.0.md). La política de confianza de los ejecutores locales de `/tdd` quedó como observación no confirmada, fuera de esa corrección.

## Revisiones anteriores

La [auditoría ACR/ADM de v1.15.1](audits/2026-11-18-acr-adm-v1.15.1.md) encontró cinco hallazgos (dos medios y tres bajos), corregidos en v1.15.2 ([HCD-048](../hcd/HCD-048-cerrar-h-01-a-h-05-de-la-auditoria-acr-a.md)). La suite completa con detector de carreras (`go test -race ./...`) y `go vet ./...` pasa sobre v1.15.2. El [ACR de v1.15.0](audits/2026-11-17-acr-v1.15.0.md) encontró ocho hallazgos medios, corregidos en v1.15.1 (ver abajo). El [ACR de HCD-045](audits/2026-11-17-acr-adm-hcd045.md) terminó `INCOMPLETE` sobre una copia anterior mientras la implementación seguía en curso; sus dos fallos de tests heredados ya no aparecen en la validación.

| Caso | Estado documentado | Medida recomendada |
|---|---|---|
| Codemode modifica settings antes de validar o escribir MCP | **Corregido** en v1.15.1 (C-001): se restauran `settings.json` y `mcp.json` si la conmutación falla | En v1.15.0, respalda los archivos y comprueba `codemode status` si la orden falla; corrige el JSON o los permisos y repite la orden |
| deps-update reutiliza la versión de caché si falla el sondeo local | Abierto: puede anunciar «al día» sin confirmar la versión actual | Ejecuta `deps-refresh` y verifica por separado el binario instalado si hubo errores de lectura |
| /plan activa el bloqueo después de cargar el método | Preexistente y abierto; la carga puede dejar una ventana sin bloqueo | Activa `/plan` cuando el agente esté inactivo; no lo trates como un aislamiento de seguridad |
| /plan conserva su estado al cambiar de sesión | Política pendiente de aclarar | Comprueba el indicador PLAN y desactívalo explícitamente con `/plan run` cuando corresponda |
| Selección explícita defaultTools vacía, absoluta o con modificadores | No se garantiza preservar toda la semántica de selecciones personalizadas al conmutar codemode | Revisa defaultTools antes y después si lo personalizaste; status comprueba marcadores, no todas las reglas efectivas de Pi |
| El aviso de recorte se suma al límite de contexto | Comportamiento documentado: 24 576 bytes del texto original más aviso y bloque Pi | No usar ese valor como límite del tamaño total de la sección |
| MCP híbridos propios conservados tras uninstall | Corregido por HCD-046 para definiciones directas e híbridas exactas | Usa uninstall --dry-run; las definiciones personalizadas se conservan |
| Protocolo publicado sin marcadores no reconocido por la migración | Corregido por HCD-045 con fixtures históricos y pruebas de regresión | Ejecuta install, abre una sesión nueva y comprueba doctor |

Los [hallazgos sobre v1.14.0](audits/2026-11-17-acr-adm-v1.14.md) y las observaciones del snapshot HCD-045 conservan su evidencia original. No se han modificado esos informes para aparentar una aprobación posterior.

### ACR de v1.15.0: corregido en v1.15.1 (HCD-047)

El [ACR de v1.15.0](audits/2026-11-17-acr-v1.15.0.md) reprodujo ocho hallazgos medios; el informe se conserva tal cual. [HCD-047](../hcd/HCD-047-corregir-los-ocho-hallazgos-del-acr-de-v.md) los corrige en v1.15.1, con `TestHCD047Regression` (rojo observado antes del cambio) y `TestHCD047Binary` (binario compilado en HOME temporales). Siguen presentes en v1.15.0: actualiza con `pi-gomemory-setup update`.

| Hallazgo | Comportamiento corregido |
|---|---|
| C-001 | Si falla MCP o el registro de codemode, `settings.json` y `mcp.json` vuelven a sus bytes anteriores (o se informa del error de restauración); el reintento registra la propiedad de `+codemode` |
| C-002 | Con `codemode off`, `install` y `update` solo escriben el perfil MCP directo y no tocan `defaultTools` |
| C-003 | El protocolo heredado se retira de `AGENTS.md` después de instalar `gomemory-context.ts`; si esa instalación falla, `AGENTS.md` no cambia |
| C-004 | Con marcadores de pi-gomemory ambiguos (huérfanos, repetidos o invertidos), `install`, `uninstall` y `doctor` conservan `AGENTS.md` completo y avisan |
| C-005 | Se retiran juntos el bloque marcado y todas las copias exactas del protocolo histórico sin marcas |
| C-006 | `uninstall` reconoce el perfil híbrido exacto de v1.14.0; las definiciones editadas se conservan |
| C-007 | Un `codemode.json` ilegible o inválido hace fallar `uninstall`, que lo conserva junto con `+codemode` |
| C-008 | La propiedad de `+codemode` queda ligada al destino real de `settings.json`; si el destino cambia, o el registro de v1.15.0 no lo identifica, `+codemode` se conserva con aviso |


## ACR/ADM del 2026-11-17 y seguimiento sobre v1.13.8

La [auditoría ACR/ADM](audits/2026-11-17-acr-adm.md) y su [seguimiento](audits/2026-11-17-acr-adm-followup.md) encontraron hallazgos de severidad media/baja reproducidos en `docs/audits/2026-11-17-acr-adm-*.txt`. Quedan cerrados en v1.14.0 mediante [HCD-040](../hcd/HCD-040-subir-los-pins-a-pi-antigravity-0-14-0-y.md) (H-01), [HCD-041](../hcd/HCD-041-cerrar-h-02-a-h-05-de-la-acr-adm-2026-11.md) (H-02..H-05) y [HCD-042](../hcd/HCD-042-cerrar-f-01-y-f-02-del-seguimiento-acr-y.md) (F-01, F-02):

| Hallazgo | Riesgo en versiones anteriores | Estado |
|---|---|---|
| H-01 | Los pins de `pi-antigravity` y `pi-cc-extensions` quedaban atrás frente a npm | **Corregido** en v1.14.0 (HCD-039 + HCD-040): `pins-check`, `pins-validate` y subida a `0.14.0` / `1.0.21` |
| H-02 | `uninstall` eliminaba un enlace simbólico en `pi-cc-extensions.json` sobrevenido tras el plan | **Corregido** en v1.14.0 (HCD-041): revalida `Lstat` antes de borrar |
| H-03 | `runOutput` aceptaba la salida de un comando fallido como versión válida | **Corregido** en v1.14.0 (HCD-041): devuelve error si el proceso falla |
| H-04 | `RunDepsUpdate` declaraba al día un estado desconocido con caché ausente | **Corregido** en v1.14.0 (HCD-041): no oculta caché ausente como saludable |
| H-05 | La extracción de tarballs candidatos no acotaba entradas ni bytes descomprimidos | **Corregido** en v1.14.0 (HCD-041): límite de 10.000 entradas y 256 MiB |
| F-01 | `RunDepsUpdate` declaraba al día versiones no comparables (`desconocida`, `dev`) | **Corregido** en v1.14.0 (HCD-042): trata versiones no comparables como fallo/desconocido |
| F-02 | `uninstall` borraba entradas no regulares o sustituidas tras el plan en cachés y manifiesto | **Corregido** en v1.14.0 (HCD-042): revalidación previa antes de retirar |

## ADM del 2026-11-17 sobre v1.13.7

La [auditoría defensiva](audits/2026-11-17-defensive-qa.md) encontró dos hallazgos de severidad media, reproducidos sobre v1.13.7 (`b22a5c4`) con [`2026-11-17-adm-repro_test.go.txt`](audits/2026-11-17-adm-repro_test.go.txt). Los cierra [HCD-038](../hcd/HCD-038-cerrar-h-01-y-h-02-de-la-adm-2026-11-17.md), publicada en v1.13.8.

| Hallazgo | Riesgo en v1.13.7 y anteriores | Estado | Mitigación si sigues en v1.13.7 o anterior |
|---|---|---|---|
| H-01 | Una suite que escribe una línea enorme sin salto (un JSON o un bundle minificado) hace crecer sin límite la memoria de Pi mientras dura la ejecución, y la línea entera queda en el diagnóstico | **Corregido** en v1.13.8 (`TestADM20261117StreamOutputBound`, `TestHCD038LongLineFails`, `TestHCD038TailBudget`, `TestHCD038HcdCheckLongJSON`): una línea de más de 64 Ki caracteres termina el proceso y la ejecución falla; el diagnóstico queda acotado a 64 KiB | Evita ejecutar con `/tdd` suites que vuelcan salida sin saltos de línea; redirígela a un archivo |
| H-02 | Una respuesta 200 de Antigravity con `null` o `{}` se muestra como éxito sin cuotas, con la etiqueta «Google AI Pro (g1-pro-tier)», y el endpoint de respaldo no se prueba | **Corregido** en v1.13.8 (`TestADM20261117QuotaRejectsMissingData`, `TestHCD038QuotaParse`, `TestHCD038QuotaFallback`, `TestHCD038PlanLabelMissing`): la respuesta sin límites cuenta como fallo, se prueba el siguiente endpoint y, sin plan informado, se muestra «Plan no informado» | Si `usage` no muestra ninguna barra, trátalo como un error y vuelve a consultar más tarde |

## ACR/ADM del 2026-11-14 sobre v1.13.4

La [auditoría ACR/ADM](audits/2026-11-14-acr-adm.md) encontró cuatro hallazgos de severidad media y uno bajo, reproducidos sobre v1.13.4 (`8fdfa19`) con [`2026-11-14-adm-repro_test.go.txt`](audits/2026-11-14-adm-repro_test.go.txt). [HCD-035](../hcd/HCD-035-cerrar-h-01-a-h-05-de-la-auditoria-acr-a.md) corrige H-01, H-03, H-04 y H-05 en v1.13.5, y [HCD-036](../hcd/HCD-036-las-configuraciones-de-pi-enlazadas-con.md) corrige H-02 en v1.13.6. Los cinco están cerrados.

| Hallazgo | Riesgo en versiones anteriores a la corrección | Estado | Mitigación si sigues en una versión anterior |
|---|---|---|---|
| H-01 | Un `uninstall` que falla a medias borra `~/.pi-gomemory/manifest.json`, así que el reintento pierde el `BIN_DIR` personalizado y las huellas de la instalación | **Corregido** en v1.13.5 (`TestHCD035C01`, `C02`, `C03`, `C16`, `C22`): el manifiesto se retira el último y solo si nada falló | Antes de desinstalar, copia `~/.pi-gomemory/manifest.json`; si el uninstall falla, restáurala antes de reintentar |
| H-02 | `install` y `uninstall` sustituyen por un archivo normal el enlace simbólico de `mcp.json`, `settings.json`, `AGENTS.md` o `pi-cc-extensions.json`, y el archivo enlazado deja de recibir los cambios | **Corregido** en v1.13.6 (`TestHCD036C01` a `TestHCD036C16`): se escribe en el archivo de destino sin sustituir el enlace, uninstall nunca lo borra y un enlace que no apunta a un archivo se rechaza sin tocar nada | En v1.13.5 o anterior, tras `install`, `update` o `uninstall`, comprueba con `ls -l ~/.pi/agent` que tus enlaces siguen siéndolo y vuelve a crearlos si hace falta |
| H-03 | Cancelar `/tdd`, o que agote su tiempo, deja vivo un proceso de prueba descendiente que ignora SIGTERM | **Corregido** en v1.13.5 (`TestHCD035C07`, `C08`): el SIGKILL al grupo se mantiene tras el cierre del proceso principal | Tras cancelar, busca procesos huérfanos de la suite (por ejemplo, `pgrep -fl node`) y termínalos |
| H-04 | `update` desde un `BIN_DIR` personalizado, en una shell sin `PI_GOMEMORY_BIN_DIR`, reinstala el launcher en `~/.local/bin` y el manifiesto pasa a apuntar allí | **Corregido** en v1.13.5 (`TestHCD035C10`, `C11`, `C12`, `C24`): el `install` relanzado recibe el `bin_dir` del manifiesto | Exporta `PI_GOMEMORY_BIN_DIR` con tu carpeta antes de ejecutar `update` |
| H-05 | Con `PI_GOMEMORY_NO_UPDATE_CHECK=1`, `update` dice «ya está al día» sin haber consultado | **Corregido** en v1.13.5 (`TestHCD035C13`, `C14`): un `update` pedido explícitamente consulta siempre | Ejecuta `update` sin esa variable |

Desde v1.13.6, si uno de esos cuatro archivos es un enlace roto (por ejemplo, con el repositorio de dotfiles sin montar), `install` falla con «el enlace no apunta a un archivo» en lugar de sustituirlo. El launcher ignora a propósito los errores de `skills-sync` para no impedir que Pi arranque, así que en ese caso el error solo aparece al ejecutar `pi-gomemory-setup skills-sync` o `install` a mano.

Hasta v1.13.6, `PI_GOMEMORY_NO_UPDATE_CHECK` no tenía el mismo efecto en todos los componentes. Con `=0` o `=false`, el launcher y el pie desactivaban el aviso mientras `check-update` seguía comprobando, y `deps-refresh` consultaba la red con cualquier valor. Desde v1.13.7 todos siguen la misma regla ([HCD-037](../hcd/HCD-037-pi-gomemory-no-update-check-significa-lo.md)): vacía, `0` o `false` dejan la comprobación activa; cualquier otro valor la desactiva, también en `deps-refresh`. En versiones anteriores, usa `=1` para desactivarla y no definas la variable para dejarla activa.

## Retraso del aviso de releases en v1.13.0 y anteriores

La comprobación de versiones solo volvía a la red si su última consulta tenía más de 24 h, así que una release podía tardar hasta un día en anunciarse en el launcher y en el pie de Pi. Desde v1.13.1 vuelve a la red como mucho una vez por hora ([HCD-031](../hcd/HCD-031-una-release-nueva-tarda-hasta-24-h-en-an.md)): el retraso queda acotado a una hora, no desaparece. En versiones anteriores, borra `~/.pi/agent/pi-gomemory-deps.json` y ejecuta `pi-gomemory-setup deps-refresh` para forzar la consulta.

## ADM del 2026-11-13 sobre v1.13.3

La [auditoría defensiva](audits/2026-11-13-defensive-qa.md) encontró tres hallazgos de severidad media, reproducidos sobre v1.13.3 con `bash docs/audits/2026-11-13-adm-repro.sh`. Los cierra [HCD-034](../hcd/HCD-034-uninstall-y-update-no-pierden-datos-ni-a.md), publicada en v1.13.4.

| Hallazgo | Riesgo en v1.13.3 y anteriores | Estado | Mitigación si sigues en v1.13.3 o anterior |
|---|---|---|---|
| H-01 | `uninstall` borra un archivo, una skill o una entrada de configuración que editaste mientras esperaba confirmación, porque solo comprobaba la propiedad al construir el plan | **Corregido** en v1.13.4 (`TestHCD034C01` a `TestHCD034C05`): cada retirada se revalida justo antes de escribir; lo que cambió se conserva como `[omitir] … — cambió después del plan` | No edites nada en `~/.pi/agent` ni en `~/.pi-gomemory` mientras `uninstall` espera confirmación; usa `--dry-run` y confirma enseguida |
| H-02 | Un archivo gestionado que no se puede leer se trata como inexistente y `uninstall` anuncia «No queda nada que retirar» | **Corregido** en v1.13.4 (`TestHCD034C06` a `TestHCD034C08`, `TestHCD034C11`): solo la ausencia es inocua; cualquier otro error da `[fallido]` y el comando termina con error | Tras desinstalar, revisa `~/.pi/agent` y `~/.pi-gomemory` y retira a mano lo que quede |
| H-03 | `update` aceptaba un ejecutable cortado a 200 MiB si el miembro del tarball era mayor | **Corregido** en v1.13.4 (`TestHCD034C09`, `TestHCD034C10`): `extractBinary` rechaza un ejecutable vacío, mayor que el límite o truncado, y `update` conserva el binario anterior | Ninguna necesaria hoy: el ejecutable publicado ocupa mucho menos de 200 MiB |

## Auditoría ACR/ADM de v1.12.1

La [revisión combinada ACR y ADM](audits/2026-11-13-acr-adm-v1.12.1.md) de v1.12.1 (`2be2613`) terminó en `APPROVED`, sin defectos críticos ni altos. Al contrastarla con el código se reprodujo C-001, se encontró un segundo punto con el mismo defecto y apareció un hallazgo nuevo (C-004).

| Hallazgo | Severidad | Riesgo | Estado | Mitigación |
|---|---|---|---|---|
| C-001 | Baja | `install`, `update` y `uninstall` reescribían `~/.pi/agent/AGENTS.md` con modo `0644` fijo (`pkg/installer/instructions.go`, `pkg/installer/uninstall.go`). Si lo tenías en `0600`, quedaba legible para otros usuarios del equipo | **Corregido** en v1.13.2 ([HCD-032](../hcd/HCD-032-install-y-uninstall-conservan-los-permis.md), `TestHCD032AgentsModePreserved`): se conserva el modo existente y uno nuevo se crea con `0644` | En v1.13.1 y anteriores, vuelve a ejecutar `chmod 600 ~/.pi/agent/AGENTS.md` después de cada instalación o actualización |
| C-002 | Baja | `httpJSON` (`pkg/installer/deps.go`) usaba `http.DefaultClient`. No podía bloquearse porque su único llamador pasaba un plazo de 20 s, pero un llamador futuro sin plazo sí podría | **Corregido** en v1.13.3 ([HCD-033](../hcd/HCD-033-timeout-propio-en-httpjson-y-race-al-emp.md), `TestHCD033HTTPJSONClientTimeout`): cliente propio con timeout de 20 s | Ninguna necesaria: en versiones anteriores el plazo del llamador ya lo evitaba |
| C-003 | Info | `scripts/build-release.sh` ejecutaba `go test` sin `-race` | **Corregido** en v1.13.3 ([HCD-033](../hcd/HCD-033-timeout-propio-en-httpjson-y-race-al-emp.md)): el script ejecuta `go test -race ./...` | Las releases desde v1.12.1 se verificaron además con `go test -race ./...` antes de empaquetar |
| C-004 | Baja | `uninstall` con otro `HOME` borraba el launcher que está junto a `pi-gomemory-setup`, aunque perteneciera a la instalación real | **Corregido** en v1.13.0 ([HCD-030](../hcd/HCD-030-uninstall-no-debe-tocar-un-launcher-fuer.md), `TestHCD030LauncherOutsideHome`) | En v1.12.1 y anteriores, no ejecutes `uninstall` con otro `HOME` usando el binario instalado; si ya ocurrió, restaura con `pi-gomemory-setup harness` |

## Aviso de actualizaciones en v1.12.1 y anteriores

`deps-status` tomaba la versión instalada de la caché `~/.pi/agent/pi-gomemory-deps.json`. Tras actualizar por otra vía (`install`, `mem update`, `pi update`), el launcher seguía anunciando y preguntando por versiones ya instaladas hasta el siguiente refresco. Corregido en v1.13.0 ([HCD-029](../hcd/HCD-029-aviso-de-version-nueva-al-abrir-pi-gomem.md)). En versiones anteriores, `pi-gomemory-setup deps-refresh` corrige la caché.

## Hallazgos de HCD-027 (cerrados en v1.12.1)

La release **v1.12.0** publicó solo HCD-026 (cuota Codex). **v1.12.1** incluye la unidad [HCD-027](../hcd/HCD-027-cierre-de-los-seis-defectos-reproducidos.md), verificada con la suite de pruebas y dos revisiones independientes ([matriz de verificación](audits/2026-11-12-hcd027-verification.md)).

| Hallazgo | Riesgo en v1.12.0 y anteriores | Estado en v1.12.1 | Mitigación si sigues en v1.12.0 |
|---|---|---|---|
| C-001 | Sincronizar una skill legacy marcada con SKILL.md intacto puede borrar archivos añadidos a su directorio | **Corregido** (`TestHCD027SkillExtras`, `TestHCD027SkillCleanupCompatibility`) | Respaldar archivos propios fuera de las carpetas gestionadas antes de instalar o sincronizar |
| C-002 | Reescribir mcp.json puede cambiar 0600 a 0644 y ampliar lectura local si los directorios padre son atravesables | **Corregido** (`TestHCD027MCPModes`, `TestHCD027JSONAtomicErrors`) | Mantener `~/.pi` y `~/.pi/agent` privados; comprobar el modo después de la operación |
| C-003 | El timeout TDD puede aceptar una salida 0 tras SIGTERM o no recuperar la cola ante procesos resistentes | **Corregido** (`TestHCD027ProcessTerminationCauses`, `TestHCD027ProcessGroupCleanup`) | Ejecutar directamente las pruebas con supervisión de procesos |
| C-004 | Pruebas Playwright de varios proyectos con la misma ubicación pueden mezclarse | **Corregido** (`TestHCD027PlaywrightIdentity`) | Consultar el informe original de Playwright y ejecutar proyectos por separado |
| C-005 | Una prueba Go interrumpida puede quedarse running en el inspector | **Corregido** (`TestHCD027GoPendingResults`, `TestHCD027GoPreservesResults`) | Confirmar el resultado con `go test` |
| C-006 | Una prueba Playwright sin resultado terminal puede aparecer passed | **Corregido** (`TestHCD027PlaywrightTerminalResults`, `TestHCD027FlakyContract`) | No confiar en aprobaciones de una salida parcial o interrumpida |

C-002 tuvo corroboración independiente inicial de dos revisores ACR en `9807703`. Los seis defectos se revalidaron de forma independiente sobre el código corregido ([Revisor A](audits/2026-11-12-hcd027-reviewer-a.md), [Revisor B](audits/2026-11-12-hcd027-reviewer-b.md)).

La política `flaky` de Q2 está implementada en `extensions/gomemory-tdd.ts` con contador y estado propios, y el fallo definitivo prevalece. El ejecutor no añade reintentos automáticos.

## `/tdd watch` en v1.12.0 y anteriores

El modo watch se ejecutaba en cada `turn_end`. Pi espera ese evento en cada turno, así que la suite completa corría entre tool calls, bloqueaba al agente y se relanzaba en el turno siguiente. Corregido en v1.12.1. En versiones anteriores, desactívalo con `/tdd watch` y ejecuta `/tdd run` cuando lo necesites.

## Integración real con Playwright

`TestHCD027PlaywrightIntegration` solo se ejecuta si `PLAYWRIGHT_TEST_CLI` apunta a un ejecutor instalado (por ejemplo, el `playwright/cli.js` de una instalación global). Sin la variable se omite, con su motivo. Se verificó con `1.61.0-alpha`; la compatibilidad con una versión estable no está comprobada.

## Cuota Codex

`codex-usage` consulta una API externa de ChatGPT, que puede cambiar. Requiere el acceso OAuth de Pi vigente; no refresca tokens. El sidebar muestra porcentaje restante, mientras que el campo `percent` del JSON indica porcentaje usado.

El heartbeat del panel es de 30 segundos, pero abrirlo y otros eventos fuerzan consultas adicionales: ese intervalo no es una garantía de frecuencia máxima. Un HTTP 429 se informa como error. Cierra el panel si necesitas evitar su sondeo automático.

## Límites de validación

Las pruebas usan cuentas y respuestas sintéticas. La API autenticada de Codex no se ha comprobado con una cuenta real. El modo watch se verificó en Pi 1.5.2 real mediante su modo RPC y el panel TDD en una sesión interactiva; el resto de la TUI se valida con pruebas de renderizado. No se garantiza la ausencia de otros defectos.

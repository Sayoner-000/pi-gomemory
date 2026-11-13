# Problemas conocidos — v1.13.2

## Retraso del aviso de releases en v1.13.0 y anteriores

La comprobación de versiones solo volvía a la red si su última consulta tenía más de 24 h, así que una release podía tardar hasta un día en anunciarse en el launcher y en el pie de Pi. Desde v1.13.1 vuelve a la red como mucho una vez por hora ([HCD-031](../hcd/HCD-031-una-release-nueva-tarda-hasta-24-h-en-an.md)): el retraso queda acotado a una hora, no desaparece. En versiones anteriores, borra `~/.pi/agent/pi-gomemory-deps.json` y ejecuta `pi-gomemory-setup deps-refresh` para forzar la consulta.

## Auditoría ACR/ADM de v1.12.1

La [revisión combinada ACR y ADM](audits/2026-11-13-acr-adm-v1.12.1.md) de v1.12.1 (`2be2613`) terminó en `APPROVED`, sin defectos críticos ni altos. Al contrastarla con el código se reprodujo C-001, se encontró un segundo punto con el mismo defecto y apareció un hallazgo nuevo (C-004).

| Hallazgo | Severidad | Riesgo | Estado | Mitigación |
|---|---|---|---|---|
| C-001 | Baja | `install`, `update` y `uninstall` reescribían `~/.pi/agent/AGENTS.md` con modo `0644` fijo (`pkg/installer/instructions.go`, `pkg/installer/uninstall.go`). Si lo tenías en `0600`, quedaba legible para otros usuarios del equipo | **Corregido** en v1.13.2 ([HCD-032](../hcd/HCD-032-install-y-uninstall-conservan-los-permis.md), `TestHCD032AgentsModePreserved`): se conserva el modo existente y uno nuevo se crea con `0644` | En v1.13.1 y anteriores, vuelve a ejecutar `chmod 600 ~/.pi/agent/AGENTS.md` después de cada instalación o actualización |
| C-002 | Baja | `httpJSON` (`pkg/installer/deps.go`) usa `http.DefaultClient`. Hoy no puede bloquearse porque su único llamador pasa un plazo de 20 s, pero un llamador futuro sin plazo sí podría | **Abierto** (latente) | Ninguna necesaria en esta versión |
| C-003 | Info | `scripts/build-release.sh` ejecuta `go test` sin `-race` | **Abierto** | Ejecuta `go test -race ./...` antes de empaquetar |
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

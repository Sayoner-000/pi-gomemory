# Problemas conocidos — v1.12.1

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

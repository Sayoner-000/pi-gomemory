# Problemas conocidos — v1.12.0

La funcionalidad publicada en esta versión corresponde a HCD-026 (cuota Codex). La unidad [HCD-027](../hcd/HCD-027-cierre-de-los-seis-defectos-reproducidos.md) sigue pendiente de aprobación e implementación. Publicar esta versión no cierra los hallazgos ACR/ADM de `9807703`.

| Hallazgo | Riesgo pendiente | Mitigación temporal |
|---|---|---|
| C-001 | Sincronizar una skill legacy marcada con SKILL.md intacto puede borrar archivos añadidos a su directorio | Respaldar archivos propios fuera de las carpetas gestionadas antes de instalar o sincronizar; el launcher sincroniza al arrancar |
| C-002 | Reescribir mcp.json puede cambiar 0600 a 0644 y ampliar lectura local si los directorios padre son atravesables | Mantener `~/.pi` y `~/.pi/agent` privados; comprobar el modo después de la operación y restaurar 0600 cuando corresponda |
| C-003 | El timeout TDD puede aceptar una salida 0 tras SIGTERM o no recuperar la cola ante procesos resistentes | Ejecutar directamente las pruebas con supervisión de procesos; no interpretar una ejecución vencida como éxito |
| C-004 | Pruebas Playwright de varios proyectos con la misma ubicación pueden mezclarse | Consultar el informe original de Playwright y, si hace falta, ejecutar proyectos por separado |
| C-005 | Una prueba Go interrumpida puede quedarse running en el inspector | Confirmar el resultado con `go test`; un proceso terminado no implica que el panel haya cerrado todos sus estados |
| C-006 | Una prueba Playwright sin resultado terminal puede aparecer passed | No confiar en aprobaciones de una salida parcial o interrumpida; consultar el ejecutor original |

C-002 tiene corroboración independiente de dos revisores ACR. Los otros cinco están reproducidos en ADM, pero mantienen clasificación unilateral en el registro ACR original. No se presentan como consenso previo.

La política de reintentos `flaky` de HCD-027 es una definición acordada, no comportamiento nuevo de v1.12.0. No se añaden reintentos automáticos ni se debe asumir un contador `flaky` operativo en esta versión.

## Cuota Codex

`codex-usage` consulta una API externa de ChatGPT, que puede cambiar. Requiere el acceso OAuth de Pi vigente; no refresca tokens. El sidebar muestra porcentaje restante, mientras que el campo `percent` del JSON indica porcentaje usado.

El heartbeat del panel es de 30 segundos, pero abrirlo y otros eventos fuerzan consultas adicionales: ese intervalo no es una garantía de frecuencia máxima. Un HTTP 429 se informa como error. Cierra el panel si necesitas evitar su sondeo automático.

## Límites de validación

Las pruebas usan cuentas y respuestas sintéticas. No se ha comprobado en esta entrega la API autenticada con una cuenta real ni la TUI mediante un navegador o terminal interactiva. Se validan el código, las pruebas existentes y los artefactos compilados; no se garantiza ausencia de otros defectos.

## Hallazgos

### A-01 — Major · Preservación de selección de herramientas
- **Fuente:** `/tmp/pi-gomemory-hcd045-audit.LqKdyy/pkg/installer/codemode.go:71–87`.
- **Problema:** activar codemode sobre `"defaultTools": []` escribe `["+codemode"]`. En Pi real esto activa también `read`, `bash`, `edit` y `write`, aunque la persona había deshabilitado explícitamente las herramientas incorporadas. Al desactivarlo, borrar la clave tampoco recupera la selección vacía.
- **Evidencia:** Pi **1.5.2**, `dist/core/settings-manager.js:90–92`, resuelve una lista vacía como ninguna herramienta incorporada, pero una lista exclusivamente de modificadores parte de `DEFAULT_TOOL_NAMES`. El test `codemode_test.go:129–146` verifica eliminar la clave, no preservar esta diferencia semántica.
- **Clase:** análisis estático contrastado con implementación real. **Confianza:** alta.
- **Pregunta / cierre mínimo:** ¿Puede conservarse la distinción entre clave ausente y selección explícita vacía, añadiendo un test de activación/desactivación desde `[]` que compruebe las herramientas efectivas?

## Comprobado y límites

- Documento objetivo y contrato HCD045 del snapshot.
- Revisión limitada a `codemode.go`, `deps.go` y fuentes auxiliares necesarias; sin repetir hallazgos de otros módulos.
- Compatibilidad de `defaultTools` y del alias `pi update --self` contrastada con Pi instalado **1.5.2**.
- Sin ejecución de tests o instalaciones, sin modificaciones y sin consultar historial ni configuraciones de otros agentes.
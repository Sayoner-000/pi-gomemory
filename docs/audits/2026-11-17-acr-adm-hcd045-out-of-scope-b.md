## Hallazgos

Rutas relativas al snapshot `/tmp/pi-gomemory-hcd045-audit.LqKdyy`.

### B-01 — Major · Compatibilidad Pi: se pierde una selección explícitamente vacía
- **Fuente:** `pkg/installer/codemode.go:71-88`.
- **Problema:** activar codemode sobre `"defaultTools": []` produce `["+codemode"]`. En Pi 1.5.2 esto habilita también `read`, `bash`, `edit` y `write`, aunque la persona los había deshabilitado. Asimismo, desactivar sobre `["codemode"]` elimina la clave y restaura esos defaults.
- **Evidencia:** Pi instalado, `dist/core/settings-manager.js:87-93`: una lista vacía resuelve a ninguna herramienta; una lista únicamente de modificadores parte de `DEFAULT_TOOL_NAMES`. El borrado de la clave ocurre cuando `filtered` queda vacío.
- **Clase / confianza:** análisis estático contrastado con implementación real / alta.
- **Pregunta de cierre:** ¿Cómo se conservará la diferencia entre selección explícita vacía y selección heredada al activar y desactivar codemode?

### B-02 — Major · Estado declarado distinto del efectivo
- **Fuente:** `pkg/installer/codemode.go:71-74,114`.
- **Problema:** comprobar solo la presencia de `+codemode` o `codemode` ignora los modificadores posteriores. Con `["read","+codemode","-codemode"]`, activar no cambia nada y `hasCodemodeInSettings` devuelve `true`, aunque Pi resuelve codemode como deshabilitado.
- **Evidencia:** Pi instalado, `dist/core/settings-manager.js:60-72,90-93`: los modificadores se aplican en orden. Con autoactivación MCP deshabilitada, el resultado tampoco queda compensado por MCP.
- **Clase / confianza:** contraejemplo estático contrastado con implementación real / alta.
- **Pregunta de cierre:** ¿Se normalizarán los modificadores al activar y se calculará el estado usando la semántica efectiva de Pi?

## Inspeccionado y límites
- Documento indicado y criterios HCD045 del snapshot.
- `codemode.go`, `deps.go` y tests relacionados; sin ampliar hallazgos a otros módulos.
- Pi instalado **1.5.2**: resolución real de herramientas y documentación de `pi update --self`, que confirma ese alias usado en `deps.go`.
- Sin ejecución de tests ni instalaciones: las herramientas disponibles solo permiten lectura. No se modificaron archivos. No se verificó la integridad SHA256 del snapshot ni se atribuyeron estos defectos a un cambio respecto de HEAD.
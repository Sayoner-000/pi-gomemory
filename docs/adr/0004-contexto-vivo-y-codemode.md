# ADR-0004: contexto de gomemory por sesión y codemode híbrido

- Estado: aceptado en HCD-044, HCD-045 y HCD-046.
- Fecha: 2026-11-17.

## Contexto

El protocolo de Pi se copiaba en AGENTS.md y podía quedar desactualizado respecto de las reglas de gomemory. La lista fija de herramientas directas tampoco reflejaba todos los cambios de exposición MCP. Activar codemode manualmente obligaba a mantener dos archivos y se perdía el objetivo de una instalación lista para usar.

## Decisión

`gomemory-context.ts` carga `mem context` una vez por sesión con el directorio del proyecto. La consulta tiene un timeout de tres segundos; se conservan hasta 24 576 bytes de texto UTF-8, más aviso y bloque Pi. Si falla, se muestra una advertencia y no se inventa contexto. El resultado se añade a `systemPromptOptions.sections.gomemory`.

El método de `/plan` ocupa otra sección. El bloque Pi no enumera MCP desde `selectedTools`: en Pi 1.5.2 sus conexiones pueden no haber terminado cuando se ejecuta `before_agent_start`.

La instalación retira el protocolo heredado de AGENTS.md mediante marcadores y fixtures históricos exactos. El texto del usuario, los permisos y los enlaces se conservan.

`install` y `update` usan codemode híbrido por defecto. Las órdenes explícitas on/off y la propiedad de `+codemode` quedan en `~/.pi-gomemory/codemode.json`. `uninstall` reconoce perfiles MCP directos e híbridos intactos y solo retira una activación de codemode atribuible al instalador.

## Consecuencias

Los cambios de reglas llegan en la siguiente sesión sin reinstalar el protocolo. La carga de memoria depende de `mem`; el fallback mantiene únicamente las instrucciones específicas de Pi. El recorte puede omitir memoria y remite a `get_context()`.

Las escrituras de settings, MCP y preferencia siguen siendo operaciones separadas. Los límites de conmutación, selección de herramientas y bloqueo de plan están en [problemas conocidos](../known-issues.md).

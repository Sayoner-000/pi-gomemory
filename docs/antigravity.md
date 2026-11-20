# Antigravity: HTTP 429 y mitigación local

## Alcance vigente (v1.16.0+)

Desde `v1.16.0`, `extensions/antigravity-prompt-guard.js` se embebe e instala por defecto como **workaround temporal** hasta que upstream publique la corrección de [pi-antigravity#152](https://github.com/Rahularya01/pi-antigravity/issues/152). Puede desactivarse por proceso con `ANTIGRAVITY_PROMPT_GUARD=off` y se retira limpiamente con `uninstall`. En `v1.15.3`–`v1.15.4` permanecía únicamente como referencia opt-in.

## Qué se reprodujo

En la instalación investigada, OAuth renovaba la sesión, la consulta de cuota respondía y había cuota disponible, pero la generación devolvía HTTP 429 `RESOURCE_EXHAUSTED` con `Resource has been exhausted (e.g. check quota).`. El fallo también apareció con Pi 0.86.1/1.4.0/1.5.2, Antigravity 0.12.0/0.14.0 y el binario publicado de pi-gomemory v1.13.3 en entornos aislados. Esa evidencia no atribuye el fallo exclusivamente al último pin ni al hardening.

La comparación controlada A/B/A conservó la credencial y los demás parámetros de la petición:

| Variante | Resultado observado |
|---|---|
| Instrucciones originales de Pi en `systemInstruction` | HTTP 429 |
| Mismo texto trasladado al primer turno de usuario y `systemInstruction` breve | HTTP 200 |
| Restauración de la petición original | HTTP 429 |

Un sistema breve también funcionó como control. No se usa como solución de producción: omitir las reglas no equivale a conservarlas. El patrón aislado depende del contenido o ubicación de las instrucciones; no demuestra que todos los HTTP 429 tengan esa causa.

La evidencia y sus límites se conservan en [HCD-040](../hcd/HCD-040-subir-los-pins-a-pi-antigravity-0-14-0-y.md). El ciclo rojo/verde del archivo de regresión fue reconstruido y registrado como evidencia nueva, no como una prueba histórica anterior al workaround.

## Riesgo y límites del guard

La [referencia](../tests/antigravity-prompt-guard/guard.mjs) se registra en `before_provider_request`. Solo transforma peticiones `agent` del proveedor `antigravity` con instrucciones textuales compatibles. Clona el payload, conserva el texto completo de las instrucciones y las antepone al primer mensaje de usuario; deja en `systemInstruction` una instrucción breve. Conserva las declaraciones de herramientas, los resultados y los metadatos. Otros proveedores, generación de imágenes y formatos incompatibles pasan sin transformación.

**Conservar el texto no conserva su prioridad.** Las reglas trasladadas al turno de usuario tienen menor jerarquía. No se garantiza protección equivalente frente a inyección de instrucciones ni el mismo comportamiento de seguimiento de reglas. No lo actives si necesitas mantener esa garantía. La decisión de usarlo es explícita y local; la referencia no es una corrección upstream.

El guard no registra prompts, payloads ni credenciales. No borra sesiones, no cambia los hooks de protección de rutas y no desactiva el hardening. Que el hook local siga bloqueando una ruta protegida no prueba que todas las protecciones basadas en instrucciones mantengan su eficacia.

## Activación manual y reversible

Requiere un checkout del tag `v1.15.3`, Node.js en el PATH y una versión de Pi compatible con `before_provider_request` (validada: Pi 1.5.2). La referencia está en el repositorio; no se incluye en los cuatro paquetes binarios.

1. Revisa `tests/antigravity-prompt-guard/guard.mjs` y acepta el riesgo anterior antes de cargarlo.
2. Ejecuta la regresión offline desde el checkout:

   ```bash
   node --test tests/antigravity-prompt-guard/guard.test.mjs
   ```

3. Si aceptas el cambio de prioridad, copia la referencia a las extensiones locales. `cp -n` no reemplaza una copia existente; si ya existe, revísala antes de decidir actualizarla:

   ```bash
   mkdir -p "$HOME/.pi/agent/extensions"
   cp -n tests/antigravity-prompt-guard/guard.mjs \
     "$HOME/.pi/agent/extensions/antigravity-prompt-guard.js"
   ```

4. Abre una sesión nueva de Pi o ejecuta `/reload`. Selecciona el proveedor Antigravity y prueba una generación breve antes de retomar trabajo sensible.

No copies la referencia a `extensions/` del repositorio de pi-gomemory: esa carpeta se embebe y cambiaría el alcance de la instalación predeterminada.

### Desactivación

Para un proceso nuevo, sin mover archivos:

```bash
ANTIGRAVITY_PROMPT_GUARD=off pi
```

También puedes retirar únicamente `antigravity-prompt-guard.js` de la carpeta local de extensiones y ejecutar `/reload`. No borres `auth.json`, `antigravity-accounts.json` ni las demás extensiones. Quitar el guard puede hacer reaparecer el fallo original; no implica que OAuth se haya roto.

## Verificación y diagnóstico

- Se validaron 22 pruebas offline, Hydrogen y Sonnet con Pi real, un ciclo de lectura/retorno de herramienta, exclusión de otros proveedores, bloqueo local de una ruta protegida y desactivación explícita.
- La persona confirmó el panel lateral y las cuotas en su interfaz. Esa confirmación es evidencia manual, no una inspección visual del agente.
- `pi-gomemory usage` y `/sidebar` verifican cuota, no que la generación funcione. Un login correcto tampoco basta para demostrarlo.
- En modo JSON de Pi, inspecciona `stopReason` y `errorMessage` del mensaje del asistente: un proceso con código 0 puede contener un error del modelo.
- Antes de comparar dos clientes, confirma que usan la misma cuenta. Un cliente distinto que responde no demuestra por sí solo que la credencial sea equivalente.

La suite de Go ejecuta la regresión mediante `TestHCD040PromptGuardRegression`. Requiere Node.js; si falta, el test falla en lugar de omitirse. No usa la extensión instalada ni hace llamadas a Google.

## Referencias upstream

El estado de estos enlaces puede cambiar; no se da por integrada una corrección solo por existir una propuesta:

- [Issue #152](https://github.com/Rahularya01/pi-antigravity/issues/152)
- [Issue #137](https://github.com/Rahularya01/pi-antigravity/issues/137)
- [Issue #148](https://github.com/Rahularya01/pi-antigravity/issues/148)
- [PR #153](https://github.com/Rahularya01/pi-antigravity/pull/153)

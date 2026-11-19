# HCD-040: regresión offline del prompt guard

Este directorio conserva la referencia local autorizada de la mitigación de
Antigravity y sus pruebas sintéticas. Se publica como referencia opt-in en
v1.15.3: no es un paquete de Pi, una carpeta de descubrimiento de extensiones ni
un recurso del instalador. `embed.go` no embebe este directorio. Consulta el
[diagnóstico y la activación manual](../../docs/antigravity.md).

Desde la raíz del repositorio, con Node.js 22.19 o posterior:

```sh
node --test tests/antigravity-prompt-guard/guard.test.mjs
go test ./pkg/installer -run '^TestHCD040PromptGuardRegression$' -count=1 -v
```

La regresión de Go también se ejecuta con `go test ./...`. El entorno del proceso
hijo excluye `AGY_GUARD_MODULE` y `ANTIGRAVITY_PROMPT_GUARD`, por lo que prueba la
referencia del repositorio, no la instalación del anfitrión. Si falta Node, falla
en lugar de omitir la prueba. Los tests predeterminados usan datos sintéticos;
no consultan la red ni leen credenciales.

Para comprobar explícitamente una copia local ya instalada:

```sh
AGY_GUARD_MODULE="file://$HOME/.pi/agent/extensions/antigravity-prompt-guard.js" \
  node --test tests/antigravity-prompt-guard/guard.test.mjs
```

## Límites de la evidencia

HCD-040 Q6 autoriza una reconstrucción nueva y aislada del caso sin transformación,
seguida de la referencia correspondiente al guard local que ya se había probado.
El rojo y el verde nuevos se registran con `mem hcd verify` contra las dos huellas
del repositorio. No recuperan ni sustituyen la evidencia histórica de `/tmp`.
Esta repetición no cambia archivos instalados, credenciales ni reglas del motor.

Conservar todo el texto de las instrucciones no conserva su prioridad: la
mitigación lo mueve del sistema a un turno de usuario. Es una medida provisional
de compatibilidad, no una garantía de resistencia equivalente frente a inyección
de instrucciones. No se instala automáticamente.

# Subir el pin de pi-antigravity a 0.15.0 tras probarlo con cuenta real

ID: HCD-063
Tipo: technical
Carril: fast
Motivo: Cambio de dependencia acotado: pins-validate ya aprueba los contratos A1 y A3 de 0.15.0; falta la prueba con cuenta real y subir el pin con su vuelta atrás

## Intención
pi-gomemory instala pi-antigravity 0.15.0, probado con la cuenta real (OAuth, cuota e inferencia) frente a 0.14.0, de modo que pins-check queda al día y la próxima release pasa su control sin forzarlo


## Alcance
Incluido: subir `AntigravityVersion` a 0.15.0 y añadir 0.14.0 a `historicPins` para que `uninstall --packages` siga retirando el pin anterior; un script reutilizable (`scripts/antigravity-pin-check.sh`) que instala versiones del plugin en un temporal y las prueba con la cuenta real cargando solo el plugin; la prueba de la instalación real con el binario nuevo (plugin, cuota y una respuesta de Pi con todas las extensiones); notas en `docs/antigravity.md`.
Excluido: retirar el guard local `antigravity-prompt-guard.js` (0.15.0 ya incluye la corrección upstream de pi-antigravity#152, pero quitarlo es una decisión aparte); cambiar `pi-cc-extensions`; publicar la release.

### Evidencia previa (2026-11-20, cuenta real, Pi 1.6.0)
- `PINS_CANDIDATE=pi-antigravity@0.15.0 go test -tags pins ./pkg/installer -run '^TestPinsValidate$'`: A1 y A3 cumplidos.
- Diferencias 0.14.0 → 0.15.0: `auth/` y `accounts` sin cambios; cambian `client.ts` (sufijos de esfuerzo, modelos deshabilitados, sin el endpoint sandbox), `models.ts` (Gemini 4 Neon y Argon, Claude 5.5), `grouping.ts`, `types.ts` y `stream.ts` (neutraliza el arranque del prompt de Pi y lo pliega al turno de usuario por encima de 126 caracteres, reintentos por capacidad, orden de `functionResponse` en Claude). Cierra pi-antigravity#152 (cerrado el 2026-11-20).
- A/B cargando solo el plugin (`--no-extensions -e`, sin MCP, skills ni AGENTS.md, sin sesión), con `gemini-4-argon:low`, `gemini-3.8-flash:low` y `gemini-4-neon:low`, sin herramientas y con el prompt por defecto de Pi: 0.14.0 falla en los 6 casos con 429 `Resource has been exhausted` (y resuelve `:low` a `-medium`); 0.15.0 responde `LISTO` en los 6.
- Con 0.15.0 también funcionan una llamada real a la herramienta bash (`agy-42`), `claude-sonnet-5-5:low` y `google_search` del plugin; 0.14.0 falla en los tres.
- Con el guard local cargado, 0.14.0 y 0.15.0 responden (`guard-45`) con `gemini-4-argon:low` y `claude-sonnet-5-5:low`: subir el pin no rompe la combinación que ya tienen los usuarios.

## ¿Qué pasa si no lo hacemos?
`make dist` sigue detenido por pins-check y cada release hay que forzarla con `PI_GOMEMORY_ALLOW_STALE_PINS=1`. Quien desactive el guard (`ANTIGRAVITY_PROMPT_GUARD=off`) sigue sin poder generar, y no llegan los modelos nuevos ni las correcciones de Claude con herramientas.

## Alternativa más barata
Registrar 0.15.0 en `RejectedVersions` para silenciar pins-check. No tiene sentido: la prueba real muestra que 0.15.0 funciona donde 0.14.0 falla.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. `pkg/installer/plugin.go`: pin 0.15.0 y 0.14.0 en `historicPins`; pins-check al día → `PI_GOMEMORY_ALLOW_STALE_PINS= go test -tags pins -count=1 ./pkg/installer -run '^TestPinsCheck$'`
- [x] 2. `scripts/antigravity-pin-check.sh`: prueba real de una versión del plugin con la cuenta del usuario (sin herramientas, con herramientas, bash real, Claude y google_search) → `bash scripts/antigravity-pin-check.sh 0.15.0`
- [x] 3. Instalación real con el binario nuevo: `pi-gomemory-setup plugin` deja 0.15.0, la cuota responde y Pi con todas las extensiones genera → `bash scripts/antigravity-pin-check.sh --installed 0.15.0`
- [x] 4. `docs/antigravity.md` documenta 0.15.0 y la relación con el guard → `grep -q '0.15.0' docs/antigravity.md && grep -q '#152' docs/antigravity.md`
- [x] 5. Suite completa en verde → `go test ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-20 19:34 -05:00 · tarea 1 · rojo observado · caducada · `PI_GOMEMORY_ALLOW_STALE_PINS= go test -tags pins -count=1 ./pkg/installer -run '^TestPinsCheck$'` (salida 1)
      --- FAIL: TestPinsCheck (1.58s)
          pins_check_test.go:22: pi-antigravity: fijado 0.14.0 · npm publica 0.15.0 sin validar
          pins_check_test.go:22: pi-cc-extensions: al día (1.0.21)
          pins_check_test.go:22: Para seguir: valida el candidato con PINS_CANDIDATE=<paquete>@<versión> go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$', regístralo como rechazado en RejectedVersions (pkg/installer/pins.go) o publica igualmente con PI_GOMEMORY_ALLOW_STALE_PINS=1.
          pins_check_test.go:25: pins-check: hay paquetes fijados atrasados frente a npm o sin comprobar
      FAIL
      FAIL	pi-gomemory/pkg/installer	2.000s
      FAIL
- 2026-11-20 19:35 -05:00 · tarea 1 · verde · caducada · `PI_GOMEMORY_ALLOW_STALE_PINS= go test -tags pins -count=1 ./pkg/installer -run '^TestPinsCheck$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.395s
- 2026-11-20 19:38 -05:00 · tarea 2 · verde · caducada · `bash scripts/antigravity-pin-check.sh 0.15.0` (salida 0)
      == pi-antigravity 0.15.0 (solo el plugin)
        ✓ sin herramientas, gemini-4-argon:low             9s  LISTO
        ✓ sin herramientas, gemini-3.8-flash:low           2s  LISTO
        ✓ sin herramientas, gemini-4-neon:low              3s  LISTO
        ✓ prompt de Pi con herramientas, gemini-4-argon:low  10s  LISTO
        ✓ bash real, gemini-4-argon:low                   14s  agy-42
        ✓ Claude, claude-sonnet-5-5:low                    2s  LISTO
        ✓ google_search, gemini-3.8-flash:low             11s  agentic-ai.readthedocs.io
- 2026-11-20 19:39 -05:00 · tarea 3 · verde · caducada · `bash scripts/antigravity-pin-check.sh --installed 0.15.0` (salida 0)
      == Instalación real (/Users/josegomezj/.pi/agent)
        ✓ paquete instalado 0.15.0
        ✓ settings.json fija npm:pi-antigravity@0.15.0
        ✓ cuota (pi-gomemory-setup usage --json)           1s  }
        ✓ Pi completo, gemini-4-argon:low                  7s  LISTO
        ✓ Pi completo, bash real                          16s  pin-42
- 2026-11-20 19:40 -05:00 · tarea 4 · verde · vigente · `grep -q '0.15.0' docs/antigravity.md && grep -q '#152' docs/antigravity.md` (salida 0)
- 2026-11-20 19:40 -05:00 · tarea 1 · verde · vigente · `PI_GOMEMORY_ALLOW_STALE_PINS= go test -tags pins -count=1 ./pkg/installer -run '^TestPinsCheck$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.789s
- 2026-11-20 19:40 -05:00 · tarea 2 · verde · vigente · `bash scripts/antigravity-pin-check.sh 0.15.0` (salida 0)
      == pi-antigravity 0.15.0 (solo el plugin)
        ✓ sin herramientas, gemini-4-argon:low             8s  LISTO
        ✓ sin herramientas, gemini-3.8-flash:low           3s  LISTO
        ✓ sin herramientas, gemini-4-neon:low              2s  LISTO
        ✓ prompt de Pi con herramientas, gemini-4-argon:low   6s  LISTO
        ✓ bash real, gemini-4-argon:low                   10s  agy-42
        ✓ Claude, claude-sonnet-5-5:low                    2s  LISTO
        ✓ google_search, gemini-3.8-flash:low             10s  pi.dev
- 2026-11-20 19:41 -05:00 · tarea 3 · verde · vigente · `bash scripts/antigravity-pin-check.sh --installed 0.15.0` (salida 0)
      == Instalación real (/Users/josegomezj/.pi/agent)
        ✓ paquete instalado 0.15.0
        ✓ settings.json fija npm:pi-antigravity@0.15.0
        ✓ cuota (pi-gomemory-setup usage --json)           0s  }
        ✓ Pi completo, gemini-4-argon:low                  6s  LISTO
        ✓ Pi completo, bash real                          12s  pin-42
- 2026-11-20 19:41 -05:00 · tarea 4 · verde · vigente · `grep -q '0.15.0' docs/antigravity.md && grep -q '#152' docs/antigravity.md` (salida 0)
- 2026-11-20 19:41 -05:00 · tarea 5 · verde · vigente · `go test ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	43.663s

## Siguiente
Cerrada con aprobación sobre la revisión 52ebdcf210c8.

# Changelog

Todos los cambios notables de este proyecto se documentan en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/).

## [v1.13.6] - 2026-11-14

### Fixed
- **Las configuraciones de Pi enlazadas con symlinks siguen enlazadas (HCD-036, H-02):** `install`, `update`, `uninstall` y los subcomandos `mcp`, `theme`, `plugin` y `skills-sync` escriben en el archivo de destino de `mcp.json`, `settings.json`, `AGENTS.md` y `pi-cc-extensions.json` cuando son enlaces simbólicos, también relativos o en cadena, y conservan sus bits de permiso. Antes sustituían el enlace por un archivo normal, y el repositorio de dotfiles dejaba de recibir los cambios, incluso al abrir Pi por el `skills-sync` del launcher.
- **Un enlace que no apunta a un archivo se rechaza (HCD-036):** si uno de esos cuatro archivos es un enlace roto, un bucle o un enlace a un directorio, `install` y `plugin` fallan con «el enlace no apunta a un archivo» sin tocar nada. `plugin` lo comprueba antes de ejecutar `pi install`. Antes, un enlace roto se sustituía en silencio por un archivo nuevo casi vacío.
- **`uninstall` nunca borra un enlace de configuración (HCD-036):** retira lo de pi-gomemory en el archivo de destino. Un `AGENTS.md` enlazado que queda vacío se vacía en el destino, y un `pi-cc-extensions.json` enlazado se omite con aviso. Un enlace roto aparece como `[fallido]` en el plan o al aplicar, y conserva el manifiesto. `pi-cc-extensions.json` solo se comprueba con `--packages`.

### Documentation
- Problemas conocidos: H-02 de la auditoría ACR/ADM del 2026-11-14 pasa a corregido. Los cinco hallazgos quedan cerrados.
- Badge, referencias de descarga, guía HTML e instaladores sincronizados con `v1.13.6`.

## [v1.13.5] - 2026-11-14

### Fixed
- **`uninstall` conserva el manifiesto si algo falla (HCD-035, H-01):** la retirada de `~/.pi-gomemory/manifest.json` pasa a ser la última del plan y solo se aplica si ninguna otra acción falló. Si se conserva, la salida lo indica siempre (`[conservar] … — conservado por fallos previos`) y el comando termina con error, así que un reintento todavía encuentra el `BIN_DIR` y las huellas de la instalación. Un archivo que cambió después del plan no cuenta como fallo, y un manifiesto ilegible se retira igualmente. `--dry-run` lo muestra como «se retira si no hay fallos».
- **Cancelar `/tdd` no deja procesos de prueba vivos (HCD-035, H-03):** si la suite se cancela o agota su tiempo, el SIGKILL al grupo de procesos se mantiene aunque el proceso principal ya haya terminado. Antes, un descendiente que ignoraba SIGTERM sobrevivía.
- **`update` reinstala en el `BIN_DIR` de la instalación (HCD-035, H-04):** el `install` que relanza `update` recibe el `bin_dir` del manifiesto, salvo que `PI_GOMEMORY_BIN_DIR` esté definida y no vacía. Antes, desde una shell sin esa variable, se creaba un segundo launcher en `~/.local/bin` y el manifiesto pasaba a apuntar allí. Si el manifiesto existe pero no se puede leer, `update` falla antes de descargar nada.
- **`update` consulta siempre (HCD-035, H-05):** `PI_GOMEMORY_NO_UPDATE_CHECK` ya no hace que `update` diga «ya está al día» sin haber consultado. La variable sigue desactivando las comprobaciones automáticas del launcher, del pie de Pi y de `check-update`.

### Documentation
- Auditoría ACR/ADM del 2026-11-14 con sus reproducciones (`docs/audits/2026-11-14-*`) y problemas conocidos: H-01, H-03, H-04 y H-05 corregidos; H-02 (configuración gestionada con enlaces simbólicos) sigue abierto.
- Badge, referencias de descarga, guía HTML e instaladores sincronizados con `v1.13.5`.

## [v1.13.4] - 2026-11-13

### Fixed
- **`uninstall` no borra lo que cambió después del plan (HCD-034, H-01):** cada retirada comprueba, justo antes de escribir, que el archivo, la skill, el enlace, la entrada de `mcp.json` o `settings.json`, el bloque de `AGENTS.md`, la entrada `packages` (antes de `pi remove`) o `pi-cc-extensions.json` siguen como los examinó el plan. Lo que cambió se conserva y aparece como `[omitir] … — cambió después del plan`; lo que desapareció, como `[omitir] … — ya no existe`. El mensaje final indica cuántos elementos se conservaron por cambio.
- **`uninstall` no anuncia éxito si no pudo leer algo (HCD-034, H-02):** solo la ausencia de un recurso es inocua. Cualquier otro error al inspeccionar o al releer da `[fallido]` y el comando termina con error, en lugar de mostrar «No queda nada que retirar». Un launcher ilegible en una carpeta que no respalda la instalación se omite e informa, aunque no quede nada que retirar.
- **`update` no instala un ejecutable incompleto (HCD-034, H-03):** `extractBinary` rechaza un ejecutable vacío, mayor de 200 MiB o truncado; el binario instalado no cambia.

### Documentation
- Auditoría defensiva del 2026-11-13 con su reproducción (`docs/audits/2026-11-13-adm-repro.sh`) y problemas conocidos con H-01, H-02 y H-03 corregidos.
- Badge, referencias de descarga, guía HTML e instaladores sincronizados con `v1.13.4`.

## [v1.13.3] - 2026-11-13

### Fixed
- **Timeout propio en `httpJSON` (HCD-033, C-002):** la consulta de versiones en npm usa un cliente HTTP con límite de 20 s en lugar de `http.DefaultClient`, de modo que no depende de que quien llama pase un plazo.

### Changed
- **`-race` al empaquetar (HCD-033, C-003):** `scripts/build-release.sh` ejecuta `go test -race ./...` antes de compilar los binarios.

### Documentation
- Problemas conocidos marca C-002 y C-003 como corregidos: los cuatro hallazgos de la auditoría ACR/ADM de v1.12.1 quedan cerrados.
- Badge, referencias de descarga e instaladores sincronizados con `v1.13.3`.

## [v1.13.2] - 2026-11-13

### Security
- **Permisos de `AGENTS.md` (HCD-032, C-001):** `install`, `update` y `uninstall` reescribían `~/.pi/agent/AGENTS.md` con modo `0644` fijo y ampliaban el acceso de un archivo restringido a `0600`. Ahora conservan el modo existente; uno nuevo se crea con `0644`. Si no se puede leer el modo, el archivo no se modifica.

### Documentation
- Problemas conocidos marca C-001 como corregido; README, instalación y arquitectura retiran la mitigación `chmod 600`.
- Badge, referencias de descarga e instaladores sincronizados con `v1.13.2`.

## [v1.13.1] - 2026-11-13

### Fixed
- **Releases nuevas anunciadas con hasta un día de retraso (HCD-031):** la comprobación de versiones del launcher y del pie de Pi reutilizaba su caché durante 24 h, así que una release publicada poco después de la última consulta no se anunciaba hasta el día siguiente. Ahora vuelve a la red como mucho una vez por hora (`depsCacheTTL`).

### Documentation
- La entrada de v1.13.0 afirmaba que una release nueva se anunciaba «en la misma apertura». Solo era cierto si la última consulta tenía más de 24 h; desde esta versión el retraso máximo es de una hora. README, uso, instalación, guía HTML v1.13, notas de v1.13.0 y problemas conocidos describen el comportamiento real.
- Badge, referencias de descarga e instaladores sincronizados con `v1.13.1`.

## [v1.13.0] - 2026-11-13

### Added
- **Aviso de actualizaciones en el pie de Pi (HCD-029):** `↑ actualizaciones disponibles` aparece en el extremo derecho del pie de gomemory cuando `pi-gomemory-setup deps-status` sale con 10. Al iniciar la sesión consulta, refresca en segundo plano y vuelve a consultar, así que una release nueva se anuncia en la misma apertura sin retrasar el arranque. Con `/tokens` apagado, el aviso se publica en el pie por defecto de Pi. Respeta `PI_GOMEMORY_NO_UPDATE_CHECK`.

### Fixed
- **Avisos de versiones ya instaladas (HCD-029):** `deps-status` y la pregunta del launcher comparaban con la versión guardada en la caché. Tras actualizar por otra vía seguían anunciando versiones ya instaladas. Ahora la versión instalada se relee de cada componente (límite de 3 s); si no se puede leer, se conserva la de la caché.
- **`uninstall` con otro `HOME` borraba el launcher real (HCD-030, C-004):** la búsqueda junto a `pi-gomemory-setup` ya no retira un launcher fuera de `HOME` sin respaldo del manifiesto ni de `PI_GOMEMORY_BIN_DIR`: se lista como `[omitir]` con su motivo.

### Documentation
- README, uso, instalación, arquitectura, índice, guía HTML v1.13 y notas de versión describen el aviso en el pie y el nuevo alcance de `uninstall`.
- Problemas conocidos recoge la auditoría ACR/ADM de v1.12.1 (`docs/audits/2026-11-13-acr-adm-v1.12.1.md`): C-004 corregido; C-001 (`AGENTS.md` reescrito con `0644`, también en `install`), C-002 y C-003 siguen abiertos.
- Badge, referencias de descarga e instaladores sincronizados con `v1.13.0`.

## [v1.12.1] - 2026-11-13

### Fixed
- **Permisos de configuración JSON (HCD-027, C-002):** reescribir `mcp.json` y otros JSON conserva el modo existente; los archivos nuevos se crean con `0600` y, ante un error de escritura, el original queda intacto y sin temporales residuales.
- **Limpieza de skills legacy (HCD-027, C-001):** solo se retiran copias gestionadas intactas que contienen únicamente `SKILL.md` y su marca; scripts, notas y subdirectorios añadidos conservan sus bytes.
- **Terminación del ejecutor TDD (HCD-027, C-003):** timeout y aborto son causas explícitas no exitosas, aunque el proceso salga con 0 tras SIGTERM. Se termina el grupo de procesos propio (SIGTERM y luego SIGKILL) y la siguiente ejecución en cola arranca a tiempo.
- **Resultados Go interrumpidos (HCD-027, C-005):** las pruebas Go pendientes al terminar el proceso quedan como fallidas con diagnóstico; ninguna queda en `running`.
- **Resultados Playwright (HCD-027, C-004 y C-006):** cada prueba se identifica por proyecto y ubicación, y su estado final sale del informe JSON oficial. Una salida parcial o interrumpida ya no aparece como aprobada. Las pruebas recuperadas en un reintento se cuentan como `flaky`, aparte, y un fallo definitivo prevalece.
- **Bucle de `/tdd watch`:** el modo watch se ejecutaba en cada `turn_end`, que Pi espera en cada turno, así que la suite completa corría entre tool calls y bloqueaba al agente. Ahora corre una sola vez cuando el agente termina (`agent_settled`), solo si hubo `edit`, `write` o `bash`, y sin bloquear. Si el agente ya ejecutó `run_test_suite all` tras su último cambio, no se repite.

### Changed
- La prueba de integración con Playwright real es opcional: sin `PLAYWRIGHT_TEST_CLI` se omite con su motivo, en lugar de dejar `/tdd`, `go test ./...` y CI en rojo. Con la variable apuntando a una ruta inexistente sigue fallando.

### Documentation
- Problemas conocidos, guía HTML, uso, instalación y arquitectura reflejan el cierre de HCD-027 y el nuevo disparador del modo watch. Matriz de verificación e informes de dos revisores independientes en `docs/audits/`.
- Badge, referencias de descarga e instaladores sincronizados con `v1.12.1`.

## [v1.12.0] - 2026-11-12

### Added
- **Cuota de OpenAI Codex (HCD-026):** `/sidebar` muestra «▼ Codex Quota» cuando Codex es el proveedor activo, con porcentaje restante, barras y tiempo hasta el reinicio de las ventanas primaria y secundaria.
- **`pi-gomemory-setup codex-usage`:** devuelve el plan y los límites en JSON con porcentaje usado. Usa la sesión OAuth `openai-codex` de Pi y el endpoint `wham/usage`; no renueva tokens. Rechaza errores HTTP, respuestas ilegibles, ventanas ausentes y cuerpos mayores que 1 MiB.

### Documentation
- Referencia CLI y guías de uso, instalación y arquitectura actualizadas; guía HTML v1.12, notas de versión y documento de problemas conocidos.
- Badge, referencias de descarga e instaladores sincronizados con `v1.12.0`; distribución para macOS y Linux en amd64 y arm64 con `SHA256SUMS`.

### Known Issues
- **HCD-027 permanece pendiente de aprobación e implementación:** esta versión no corrige sus seis hallazgos sobre permisos MCP, archivos añadidos a skills legacy y resultados/terminación TDD. La política `flaky` acordada para esa unidad tampoco está implementada. Véase `docs/known-issues.md`.

## [v1.11.1] - 2026-11-11

### Fixed
- **Límites de Anthropic sin sondeo ni HTTP 429 (HCD-024):** el bloque «▼ Anthropic · plan» del sidebar ahora obtiene la utilización y tiempos de reinicio de la sesión (5 h) y de la semana directamente a partir de las cabeceras `anthropic-ratelimit-unified-*` que Anthropic envía en cada respuesta del modelo (`after_provider_response`), de forma pasiva e instantánea. Se eliminó el sondeo recurrente cada 30 s y en cada turno que saturaba el endpoint `/api/oauth/usage` con errores 429. El comando `anthropic-usage` actúa ahora solo como respaldo de una única consulta al abrir el panel si aún no hay lecturas previas, con tiempo de reintento acotado.
- **Banner ASCII predeterminado:** restaurado el banner ASCII en color de alta resolución como predeterminado en Pi, eliminando la dependencia de secuencias gráficas en el arranque.

### Documentation
- **Guía del harness pi-gomemory (`docs/Guia-PI-GOMEMORY-v1.11.html`):** guía integral en formato HTML interactivo que abarca arquitectura, instalación, herramientas MCP, extensiones TUI, flujo HCD y casos de uso de punta a punta.
- Actualizados badges, referencias de versión e instaladores a `v1.11.1`.

## [v1.11.0] - 2026-11-11

### Added
- **Límites del plan de Anthropic en `/sidebar` (HCD-023):** con Pi conectado a Anthropic por OAuth (Claude Pro/Max), el bloque «▼ Anthropic · plan» muestra el porcentaje usado de la sesión (ventana de 5 h) y de la semana, con el tiempo hasta cada reinicio y una barra que cambia a amarillo al 50 % y a rojo al 80 %. Si Anthropic envía límites semanales por modelo (Opus, Sonnet), también aparecen. Se actualiza al abrir el panel, en cada turno y cada 30 s.
- **Comando `pi-gomemory-setup anthropic-usage`:** devuelve esos límites en JSON. Usa el token OAuth que Pi guarda en `~/.pi/agent/auth.json` y consulta la API de uso de Anthropic, la misma que usan claude.ai y el `/usage` de Claude Code. No renueva el token, porque eso lo hace Pi.

### Changed
- Los errores del bloque de Anthropic se muestran completos en varias líneas en lugar de recortarse, para no perder la parte accionable («abre pi para renovarlo», «ejecuta pi-gomemory-setup update»).

### Security
- `anthropic-usage` lee como máximo 1 MiB de respuesta y sus errores nunca incluyen el token. Sin sesión OAuth, con el token caducado, ante una respuesta distinta de HTTP 200 o sin límites, devuelve un error explícito y nunca cifras viejas o inventadas.

### Documentation
- Bloque de Anthropic en `docs/usage.md`, comando `anthropic-usage` en el README y referencias de versión, badge, instalador y script de build actualizados a `v1.11.0`.

## [v1.10.1] - 2026-11-11

### Fixed
- **`/min` alterna ambos paneles (HCD-022):** antes solo minimizaba el sidebar aunque su descripción hablaba de «los paneles». Ahora, si alguno de `/sidebar` o `/tdd` está expandido, minimiza los dos; si los dos están minimizados, los restaura.
- **`/tdd report` muestra el informe:** antes construía el texto pero solo notificaba «Reporte generado». Ahora publica en el chat el informe de la última ejecución, con el resultado por suite y cada fallo con mensaje, archivo:línea, paso, diagnóstico y sugerencia. Queda también en el contexto del modelo, sin iniciar un turno, para pedir la corrección sin volver a ejecutar. Sin ejecución previa, indica que se use `/tdd run`.
- **Barra inferior sin emoji:** `⚡` (lecturas de caché) pasa a `↺` y `[🧠 nivel]` a `[thinking nivel]`, con el mismo criterio de glifos de texto que los paneles.

### Documentation
- `/min` y `/tdd report` descritos en `docs/usage.md` y en la tabla de comandos del README; referencias de versión, badge, instalador y script de build actualizados a `v1.10.1`.

## [v1.10.0] - 2026-11-11

### Added
- **Ejecución real en `/tdd` (HCD-020):** las suites se detectan según el proyecto abierto (`go.mod` → `go test -json ./...` y `go vet ./...`; `package.json` con script `test` → `npm test`; `tsconfig.json` → `npx tsc --noEmit`; `playwright.config.*` → `npx playwright test --reporter=line`). Lo que no aplica, o cuya herramienta falta, aparece como «omitida» con su motivo.
- **Recorrido en vivo:** los procesos se lanzan con `spawn` (sin shell) y la salida se procesa línea a línea, así que el panel avanza test a test mientras corren, con la duración real de cada uno.
- **Detalle de cada fallo:** mensaje real, `archivo:línea`, paso fallido de Playwright (`› título › paso`) y un resumen `Expected · Received`, además del diagnóstico.
- **Bucle con el agente:** `run_test_suite` devuelve cada fallo con su suite, test, mensaje, ubicación, paso, diagnóstico y sugerencia, y envía avances mientras corre.
- **Clic para minimizar y expandir (HCD-019):** clic en `[−]` o `[+]` en la cabecera de `/sidebar` y `/tdd` (modo `fullscreen` de Pi).
- **Referencia de comandos** en el README: comandos del CLI que faltaban (`check-update`, `deps-refresh`, `claude-usage`, `uninstall`) y una tabla de los comandos dentro de Pi.

### Changed
- **Panel `/tdd`:** muestra solo las suites habilitadas y, dentro de cada una, los tests en curso y los fallidos; los aprobados se resumen en un conteo y las omitidas se agrupan al final. Sin emoji: el test en curso usa el spinner braille de pi-tui y el resto, `✓ ✗ ○ –`.
- **Paneles apilados:** `/tdd` se coloca debajo de `/sidebar`, en la misma columna, y se reubica cuando el sidebar cambia de altura.
- **Modo minimizado:** ocupa una sola fila, sin el relleno de 26 filas en blanco que tapaba la conversación.
- **Modo watch:** varios turnos durante una ejecución se agrupan en una sola ejecución pendiente, y `/tdd run` avisa cuando queda en cola.

### Fixed
- **Clic sin efecto:** pi-tui entrega un clic como `press → release → click` y los paneles alternaban en `press` y en `click`, así que se minimizaban y restauraban en el mismo gesto. Ahora solo alterna el `click`.
- **Resultados falsos en `/tdd`:** Playwright se marcaba como aprobado sin ejecutar nada; Frontend fallaba siempre porque `node --check` no admite TypeScript; Data fallaba sin unidad HCD activa. Hallazgos C-001 y C-002 de la auditoría v1.9.0.
- **Rutas fijas:** `/tdd` ya no asume `./pkg/installer` y funciona en cualquier repositorio Go (C-006).
- **`[−]` oculto con títulos largos:** el título del sidebar se recorta reservando el espacio del control (C-004).
- **Diagnóstico de errores de sintaxis:** «unexpected token» ya no se clasifica como discrepancia de contrato.

### Security
- **`fetchQuota` limitado a 1 MiB:** una respuesta mayor se rechaza con un error explícito, en lugar de cargarse entera en memoria (C-005).

### Documentation
- Guía de `/tdd` y `/sidebar` en `docs/usage.md`, arquitectura de `gomemory-tdd.ts` en `docs/architecture.md`, informe de la auditoría ACR/ADM v1.9.0 en `docs/audits/`, y referencias de versión, badge, instalador y script de build actualizados a `v1.10.0`.

## [v1.9.0] - 2026-11-11

### Added
- **Inspector de pruebas en tiempo real (`/tdd`):** Nueva extensión `gomemory-tdd.ts` con visualizador reactivo TDD Live Inspector: soporte multi-suite (Backend con `go test -json`, Frontend con chequeo sintáctico TypeScript, Playwright E2E, CI estático con `go vet`, y Data & Graph con estado HCD y memoria), seguimiento de latencia y porcentaje de aprobación, y diagnóstico heurístico de fallos (`code_defect`, `selector_or_timeout`, `env_or_service`).
- **Tool `run_test_suite`:** Herramienta directa registrada en Pi para que el modelo o subagentes invoquen suites de pruebas de forma programática y reciban resultados estructurados.
- **Modo minimizado y comando `/min`:** Subcomando `/sidebar min` (y alias `/sidebar m`, `/sidebar -`) y `/tdd min` (alias `/tdd m`, `/tdd -`), además del comando global `/min` para compactar o restaurar ambos paneles a una sola fila enmarcada (`[ HCD · ... [+] ]` y `[ TDD · ... [+] ]`).
- **Apilamiento vertical sin colisión:** El overlay de TDD minimizado se posiciona con margen vertical coordinado debajo del sidebar para convivir simultáneamente en la esquina superior derecha sin solapamiento visual.
- **Registro de distribución:** Inclusión de `extensions/gomemory-tdd.ts` en `pkg/installer/extension.go` para empaquetado embebido y despliegue automático.

### Fixed
- **Crash en `/reload` por stale `ctx`:** Limpieza estricta de temporizadores activos (`setInterval`) y cierre de overlays en el evento `session_shutdown`, evitando excepciones no capturadas al acceder a propiedades de `ctx` invalidado tras recargas o cambios de sesión.
- **Acceso defensivo a contexto de Pi:** Protección con `try/catch` y fallbacks garantizados (`process.env.PI_PROVIDER`, `process.env.PI_MODEL`, `process.cwd()`) en todas las lecturas de propiedades dinámicas de `ctx` (`ctx.model`, `ctx.cwd`, `ctx.sessionManager`).
- **Limpieza de buffer residual al minimizar:** Sobreescritura completa con espacios en blanco en las filas inferiores del overlay para que el compositor diferencial de la terminal borre de inmediato los restos del panel expandido.
- **Eliminado widget superior fijo:** Retirado el widget estático en la cabecera para no ocupar espacio permanente sobre el editor.

### Documentation
- Actualización de badges de versión, instalador, documentación de instalación (`docs/installation.md`), guía de uso (`docs/usage.md`) y scripts de compilación a `v1.9.0`.

## [v1.8.1] - 2026-11-11

### Added
- Detección integral de proveedores activos (`detectProvider`) cubriendo Antigravity/Agy (`Google AI Pro`), Claude Code, OpenCode, Codex y OpenAI, combinando `ctx.model` con fallback a variables de entorno del harness (`PI_PROVIDER`, `PI_MODEL`).
- Tarjeta de conexión y estado (heartbeat en vivo) para proveedores activos no-Antigravity en el panel lateral `/sidebar`.

### Changed
- Sincronización del heartbeat periódico del panel (`autoRefreshTimer` a 30 s) con refresco forzado en cada tick para mantener actualizados los porcentajes y tiempos de reinicio sin descarte por TTL.
- Ajuste del TTL de cuota (`QUOTA_TTL_MS`) y telemetría a 30 s con backoff de 15 s ante fallos.

### Fixed
- **Refresco por turnos en sidebar (HCD-017):** Suscripción al ciclo de vida `turn_end` y actualización forzada en `agent_end` para que cada turno consumido refresque inmediatamente las cuotas y métricas sin quedar bloqueado por TTL.
- **Cierre formal HCD-016:** Verificación y registro formal de cierre de las 8 tareas de auditoría ACR/ADM followup en el motor HCD y en memoria persistente.

### Documentation
- Actualización de badges de versión, instalador y referencias documentales de instalación a `v1.8.1`.

## [v1.8.0] - 2026-11-11

### Added
- Integración en el panel lateral (`/sidebar`) de cuotas de Antigravity (Google AI Pro) con barras continuas Unicode (`████░░░░`), porcentajes restantes y tiempos de reinicio.
- Telemetría agregada de Claude Code en el panel lateral (`/sidebar`) y comando CLI `claude-usage` (`--json`), calculando consumo de hoy, semana y total acumulado con porcentaje de ahorro de lectura en caché a partir de `~/.claude/token-usage-cache.json`.
- Comando CLI `usage --json` para consumir la cuota de Antigravity estructurada.
- Enmienda en `constitution.md` formalizando los 5 principios de diseño SOLID (SRP, OCP, LSP, ISP, DIP).
- Timer periódico en background (30 s) dentro de `gomemory-sidebar.ts` para refrescar métricas automáticamente mientras el sidebar está visible.

### Changed
- Endpoints de consulta de cuota de Antigravity actualizados al servicio oficial de Google Cloud Code PA (`daily-cloudcode-pa.googleapis.com`) con User-Agent de Antigravity CLI 1.4.0.
- `RefreshDeps` en `deps.go` respeta un TTL de 24 horas sobre la caché de dependencias para evitar consultas de red innecesarias en cada arranque del launcher.
- El lanzador `pi-gomemory` (`harness.go`) trunca el contexto inyectado de gomemory a 64 KiB para no desbordar los límites del kernel de Linux.

### Fixed
- **Integridad de datos (H-01):** `pruneLegacyPiSkills` en `hostskills.go` no elimina carpetas de skills si contienen archivos o scripts adicionales del usuario además de `SKILL.md`.
- **Desinstalador (H-02):** `pi-gomemory-setup uninstall` y `--dry-run` devuelven código de error si existen acciones fallidas (`ActionFail`), aun cuando `Pending()` sea 0.
- **Diagnóstico (H-03):** `pi-gomemory-setup doctor` invoca `pi mcp list` con timeout de 10 segundos y emite advertencia `[WARN]` si la consulta falla o el JSON es ilegible.
- **Guardarraíl Modo Plan (H-04):** `gomemory-plan-patterns.json` bloquea evasiones de comandos mediante intérpretes (`bash -c`, `sh -c`, `zsh -c`, `python -c`, `node -e`, `perl -e`, `ruby -e`) y herramientas mutables (`make`, `xargs`, `env`, `find -delete/-exec`, `dd of=`, `git apply/clean/pull`, `patch`, `curl -o`, `wget -O`, `install`, `rsync`, `unlink`, `go build -o`).
- **Telemetría Claude Code (H-05, H-06):** Fechas comparadas en zona horaria local mediante parseo RFC3339 y ordenamiento determinista de claves de caché para evitar variaciones en el ranking de proyectos.
- **Sidebar (H-08):** Saneamiento de mensajes de error de stderr en JSON, guarda contra llamadas concurrentes (`inflight`), backoff ante fallos y validación defensiva de la estructura de cuota.
- **Sincronización y paquetes (H-09):** `SyncSkillRepo` solo elimina symlinks huérfanos cuyo destino apunte a fuentes del host, conservando enlaces personalizados del usuario; `planPackages` invoca `pi remove` con timeout y propaga errores.

### Documentation
- Actualiza badges de versión y referencias documentales de instalación y uso a `v1.8.0`.

## [v1.7.4] - 2026-11-11

### Documentation
- Agrega el comando `skills-sync` a la lista de comandos CLI en `README.md`, `docs/usage.md` y a la ayuda interactiva del binario instalador (`main.go`).
- Actualiza badges de versión y referencias documentales de instalación a `v1.7.4`.

## [v1.7.3] - 2026-11-11

### Fixed
- `pruneLegacyPiSkills` retira de `~/.pi/agent/skills` las copias heredadas de versiones previas que no tenían la marca `.pi-gomemory` cuando su contenido coincide con el del repositorio central. Esto erradica por completo los avisos `[Skill warnings] collision` al iniciar Pi.

### Documentation
- Actualiza badges de versión y referencias documentales de instalación a `v1.7.3`.

## [v1.7.2] - 2026-11-11

### Documentation
- Documenta en `docs/usage.md` el mecanismo de supresión y saneamiento de advertencias (warnings) de Pi Coding Agent implementado en el instalador: resolución previa de colisiones de skills vía repositorio centralizado `~/.pi-gomemory/skills`, saneamiento de YAML frontmatter en `SKILL.md` (`SanitizeFrontmatter`), y normalización de dependencias de host (`peerDependencies` en `pi-antigravity`).
- Actualiza badges de versión y referencias documentales de instalación a `v1.7.2`.

## [v1.7.1] - 2026-11-11

### Fixed
- El banner de gomemory vuelve a verse tras `install` o `update` en cualquier sistema. El instalador ahora impone `showStartupHeader: false` en `~/.pi/agent/pi-cc-extensions.json`; antes solo añadía claves ausentes y una configuración previa con `true` dejaba la cabecera de pi-cc-extensions encima del banner. El resto de preferencias de pi-cc-extensions se conserva.

### Added
- `pi-gomemory-setup doctor` incluye la comprobación «Banner gomemory», que avisa si la cabecera de pi-cc-extensions está tapando el banner.

## [v1.7.0] - 2026-11-11

### Added
- `pi-gomemory-setup uninstall` retira solo lo que instaló pi-gomemory, comprobado con un manifiesto SHA256 que escribe cada `install`. Opciones `--dry-run`, `--verbose` (con log en `~/.pi-gomemory/uninstall.log`), `--yes`, `--packages` y `--self`. Conserva la memoria de gomemory y `doctor` informa «pi-gomemory no instalado».

### Changed
- El panel lateral (`/sidebar`) muestra los servidores MCP declarados en `mcp.json` como `Configured` y ya no afirma una conexión que no comprueba; sin `mcp.json` válido la lista queda vacía.
- codebase-memory-mcp ya no se actualiza automáticamente: `deps-status` y `deps-update` indican la URL del release para actualizarlo a mano.

### Fixed
- `deps-update` ya no dice «Dependencias actualizadas» cuando quedan componentes manuales pendientes.
- `extractBinary` y `lookupChecksum` leen los datos sin copiarlos a texto.

### Security
- Se elimina la ejecución de `curl | bash` sin verificar al actualizar codebase-memory-mcp.
- `deps-update` ejecuta los comandos definidos en el binario y no los guardados en la caché `pi-gomemory-deps.json`.
- `splash_review` rechaza documentos fuera del repositorio, incluidos `..`, rutas absolutas y enlaces simbólicos.
- El modo plan bloquea la herramienta `powershell`.
- El instalador escribe extensiones, tema, launcher, `AGENTS.md`, skills y cachés de forma atómica.
- Informe completo en `docs/audits/2026-11-11-acr-adm-review.md`; queda como riesgo pendiente que las herramientas de la sesión Splash aceptan rutas absolutas.

## [v1.6.3] - 2026-11-11

### Changed
- Nueva identidad visual: logotipo de pi-gomemory en SVG y WebP con variantes clara y oscura, más icono en SVG y PNG (`assets/`). El README usa los SVG según el tema del sistema.

## [v1.6.2] - 2026-11-11

### Changed
- El banner identifica el enfoque HCD: la línea de versiones pasa a `HCD · pi - gomemory  vX.Y.Z · gomemory vX.Y.Z` y debajo aparece `HCD · Harness Continual Development`.
- El título del panel lateral (`/sidebar`) empieza por `HCD ·`.

## [v1.6.1] - 2026-11-11

### Fixed
- El panel lateral (`/sidebar`) ya no captura el teclado: flota sin quitar el foco al editor, se oculta con `/sidebar` y refresca tokens y coste al terminar cada respuesta.
- El banner y el panel lateral muestran la versión instalada aunque Pi se lance con `pi` en lugar del launcher `pi-gomemory`: si faltan las variables de entorno, la consultan con `pi-gomemory-setup version` y `mem --version`.

## [v1.6.0] - 2026-11-11

### Added
- Panel lateral de estado inspirado en OpenCode para Pi (`gomemory-sidebar.ts`): overlay interactivo en la esquina superior derecha (`anchor: "top-right"`, 36 columnas, responsive $\ge 100$ columnas) que consolida la sesión/HCD activo, métricas de contexto en tiempo real (tokens usados, porcentaje de ventana y gasto acumulado en USD) y lista de servidores MCP (`codebase-memory-mcp`, `codegraph`, `context7`, `gomemory`) con estado de conexión. Se activa y desactiva con el comando `/sidebar`.

## [v1.5.1] - 2026-11-11

### Added
- Registro de decisiones de arquitectura en `docs/adr/` (repositorio central de skills, extensiones TUI de HCD y orquestación de dependencias).

### Fixed
- `pi-gomemory-setup update`, `check-update` y la comprobación de dependencias ya no fallan con HTTP 403 cuando se agota el límite de la API de GitHub (60 consultas por hora y por IP). La última versión de pi-gomemory y de `codebase-memory-mcp` se lee de la redirección de `github.com/<repo>/releases/latest`, sin token.

## [v1.5.0] - 2026-11-11

### Added
- El banner de inicio muestra, junto a la versión de pi-gomemory, la de gomemory instalada (`pi - gomemory  v1.5.0 · gomemory v2.54.1`). El launcher la lee de `mem --version` y la exporta en `GOMEMORY_INSTALLED_VERSION`; si `mem` no está, se omite.

## [v1.4.1] - 2026-11-11

### Added
- Panel de tareas HCD en Pi (`gomemory-tasks.ts`): muestra sobre el editor las tareas de la unidad activa con su estado, leídas de `mem hcd show --json`, y se actualiza tras cada `mem hcd` y al terminar cada respuesta. `/tasks` lo muestra u oculta.

### Fixed
- `/plan <petición>` ya no se rechaza como argumento desconocido: activa el modo plan y envía la petición.

## [v1.4.0] - 2026-11-11

### Added
- Repositorio central de skills `~/.pi-gomemory/skills`: reúne las 21 skills de gomemory y enlaces a las de `~/.agents`, OpenCode, Claude Code, Codex y Gemini. Pi carga solo ese repositorio y se sincroniza en cada arranque (`pi-gomemory-setup skills-sync`).
- Comprobación de actualizaciones de pi-gomemory, `pi`, `mem`, `codegraph`, `codebase-memory-mcp`, `pi-antigravity` y `pi-cc-extensions` en cada arranque, en segundo plano, con pregunta de 10 s en terminal interactiva. Comandos `deps-refresh`, `deps-status` y `deps-update`.

### Changed
- `pi-antigravity` y `pi-cc-extensions` se instalan fijados a la versión del catálogo oficial de Pi (0.12.0 y 1.0.16).
- Las skills de gomemory adaptadas para Pi ganan a las copias de otros agentes; antes ganaba la de OpenCode.
- `check-update` deja de llamarse desde el launcher; lo sustituye `deps-status`.

### Fixed
- La instalación ya no borra skills propias de `~/.pi/agent/skills` con el mismo nombre que una del anfitrión.
- `PI_GOMEMORY_BIN_DIR` se aplica también al launcher, que encuentra `pi-gomemory-setup` en su propio directorio.
- `install.sh` reemplaza el binario con un renombrado atómico (evita `Killed: 9` en macOS) y la compilación local lleva la versión de git en lugar de `dev`.
- Las escrituras de `settings.json`, `mcp.json` y `pi-cc-extensions.json` son atómicas.
- Los fallos de `pi install` dejan de ignorarse en silencio.

### Security
- Los binarios de la release se compilan con Go 1.27.4, que incluye correcciones de seguridad para cgo, el comando go, el compilador, el enlazador y `net/http`.
- `build-release.sh` ejecuta `go vet` y `go test`, compila con `-trimpath` y se detiene si no puede generar `SHA256SUMS`.

## [v1.3.0] - 2026-11-10

### Added
- Banner de inicio con el logo de gomemory y la versión en ejecución (`gomemory-banner.ts`).
- Modo de planificación `/plan` (`gomemory-plan.ts`): bloquea las herramientas y comandos que modifican archivos mientras se planifica.
- Aviso de nuevas releases en el launcher (consulta como máximo cada 24 h, sin bloquear) y comandos `pi-gomemory-setup version`, `check-update` y `update`, este último con verificación de checksum.
- Pi carga las skills de OpenCode y Claude Code instaladas en el equipo.

### Changed
- El binario lleva su versión, inyectada al compilar con `-X main.version`.
- `doctor` cuenta todas las skills distintas que Pi tiene disponibles.

### Fixed
- Pi ya no avisa de colisiones de skills: se registra solo `~/.config/opencode/skills`, las skills que solo existen en `~/.claude/skills` se enlazan en `~/.pi/agent/skills` y se retiran las copias que ya ofrece el anfitrión.
- La cabecera de `pi-cc-extensions` ya no tapa el banner de gomemory (`showStartupHeader: false` por defecto).

## [v1.2.0] - 2026-11-10

### Added
- Herramienta `splash_review` en Pi: ejecuta al revisor en una sesión independiente, sin historial ni memoria del proyecto, con herramientas de lectura, cancelación y límite de cinco minutos.
- Definición neutral de Splash incluida en el paquete; no requiere archivos de Claude Code. El adaptador incluido corresponde a Pi.
- Diagnóstico de herramientas del portapapeles para macOS, X11/WSLg y Wayland, con indicaciones para instalar las dependencias ausentes.
- Pruebas de instalación de los recursos de Splash y del diagnóstico multiplataforma del portapapeles.

### Changed
- El instalador incluye la extensión y la definición de Splash junto con las instrucciones para ejecutarlo y registrar el resultado real.
- Documentación de Splash y de los requisitos del portapapeles. `xclip` continúa siendo una dependencia del sistema; no se incluye en los binarios.

### Fixed
- Pi dispone de una herramienta para ejecutar Splash y ya no necesita buscar su definición en la configuración de otro agente.
- Se declara Go 1.27.0 como mínimo y Go 1.27.4 como toolchain para evitar la descarga fallida de `go1.27` desde instalaciones anteriores de Go.

## [v1.1.0] - 2026-11-10

### Fixed
- `usage` ya no inventa cuotas cuando la API falla: devuelve un error y termina con código 1.
- `mcp.json`: se conservan todos los campos de los servidores ajenos (`url`, `headers`, `type`…). Un JSON inválido detiene la instalación sin sobrescribirlo. Mismo criterio para `settings.json` y `pi-cc-extensions.json`.
- `AGENTS.md`: el protocolo se escribe entre marcadores `<!-- pi-gomemory:start/end -->` y conserva el contenido propio. Las instalaciones v1.0.0 se migran sin duplicar.
- `install` y `harness` ya no crean `./bin/pi-gomemory` en el directorio actual.
- Se elimina el respaldo que leía y reescribía skills en `~/.config/opencode/skills`.
- El tema y la configuración de `pi-cc-extensions` se leen de `themes/` y `config/` embebidos, que antes no se usaban. En `pi-cc-extensions.json` solo se añaden las claves que falten.
- `install.sh` compila desde fuente solo si el directorio es este módulo, y verifica el tarball contra `SHA256SUMS`.

### Added
- `doctor` lista los otros agentes detectados (Claude Code, Codex, agy, OpenCode) y el comando de gomemory que los integra.
- Variable `PI_GOMEMORY_BASE_URL` en `install.sh` para probar contra un `dist/` local.
- Pruebas del instalador en un HOME aislado (`go test ./...`).

## [v1.0.0] - 2026-11-09

### Added
- Binario Go (`pi-gomemory-setup`) con recursos de configuración, temas y skills embebidos en el binario mediante `embed.FS`.
- Comando de instalación en un solo paso (`install`) que configura MCPs, tema, plugins, skills y lanzador.
- Integración de tres servidores MCP en `mcp.json`: `gomemory`, `codegraph` y `codebase-memory`.
- Configuración del plugin `npm:pi-antigravity` para modelos de Google AI Pro con parche de dependencias `typebox`.
- Monitor de cuotas `pi-gomemory usage` para consultar límites y tiempos de reinicio de Google AI Pro.
- Tema gomemory con paleta oscura en Truecolor de 24 bits.
- Extensión Claude Code TUI (`pi-cc-extensions`) con diffs visuales e inspección de ventana con `/context`.
- Extensión `gomemory-status.ts` para Pi con contadores de tokens y coste en el pie de terminal.
- 21 skills oficiales de gomemory con sanitización de frontmatter YAML para HCD (Harness Continual Development), revisión y auditoría.
- Lanzador `pi-gomemory` para Herdr con inyección de contexto de memoria en el arranque.
- Script instalador `install.sh` y script de empaquetado multiplataforma `scripts/build-release.sh`.
- Documentación técnica en `/docs` sobre arquitectura, instalación y comandos de uso.

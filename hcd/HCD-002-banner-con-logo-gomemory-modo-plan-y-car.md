# Banner con logo gomemory, modo /plan y carga de skills del anfitrión

ID: HCD-002
Tipo: feature
Carril: full
Motivo: Varias decisiones interdependientes (directorios de skills, duplicados, comportamiento de /plan, formato del logo); elegido por la persona

## Intención
Al iniciar pi-gomemory se muestra un banner con el logo de gomemory (gota) y el texto «pi - gomemory»; una extensión TypeScript añade el modo de planificación /plan; Pi carga también las skills disponibles en el anfitrión (p. ej. impeccable, gitlab-release) además de las 21 embebidas


## Alcance
Incluido: extensión de banner, extensión `/plan` de solo lectura, registro de carpetas de skills del anfitrión en `settings.json`, ajuste de `doctor`, pruebas y documentación.
- Banner: extensión nueva `extensions/gomemory-banner.ts` que en `session_start` llama a `ctx.ui.setHeader(...)` (API de extensiones de Pi 1.5.1, docs/extensions.md) con una gota dibujada en caracteres Unicode y colores truecolor del tema gomemory, el texto «pi - gomemory» y la versión. La versión la exporta el launcher `pi-gomemory` en `PI_GOMEMORY_VERSION` (salida de `pi-gomemory-setup version`); si falta, se muestra sin versión. Con `NO_COLOR` definida se dibuja sin secuencias de color. Al lanzar `pi` sin el launcher también se ve el banner, sin versión.
- `/plan`: extensión nueva `extensions/gomemory-plan.ts` que registra el comando con `pi.registerCommand("plan", …)`.
  - Con el modo activo, un manejador `pi.on("tool_call", …)` devuelve `{ block: true, reason }` para las herramientas `edit` y `write`, y para `bash` cuando el comando coincide con la lista de patrones modificadores exportada por la extensión: `rm`, `mv`, `cp`, `mkdir`, `touch`, `chmod`, `ln`, `tee`, redirección `>`/`>>`, `sed -i`, `git commit|push|reset|checkout|merge|rebase|add`, `npm|pnpm|yarn install`, `go mod tidy`.
  - Al activarse obtiene el método ejecutando `mem plan-context`; si `mem` no existe o falla, usa un texto de respaldo incluido en la extensión. El texto se inyecta en el system prompt con `before_agent_start` mientras el modo está activo.
  - Un indicador de estado («PLAN») se muestra mientras el modo está activo.
  - `/plan` otra vez, o `/plan run`, desactiva el modo.
- Skills del anfitrión: `pi-gomemory-setup install` añade al array `skills` del `~/.pi/agent/settings.json` (docs/settings.md, sección Resources) las rutas `~/.config/opencode/skills` y `~/.claude/skills` que existan, fusionando con el contenido del usuario y sin duplicar entradas. `~/.agents/skills` no se registra porque Pi ya la descubre por sí mismo (docs/skills.md). Pi lee esas carpetas en su sitio.
- Nombres repetidos: Pi conserva la primera skill que descubre, y las carpetas del array `skills` se descubren antes que `~/.pi/agent/skills`, así que sin intervención la copia de otro agente tapa la adaptada para Pi (observado por la persona el 2026-11-10: 21 colisiones). Por eso, al registrar cada carpeta del anfitrión, `pi-gomemory-setup` añade una exclusión exacta `-<carpeta>/<nombre>` (settings.md: «exact exclusion with `-path`»; Pi expande `~` y compara con la carpeta de la skill) por cada skill embebida en pi-gomemory que exista en esa carpeta. Así ganan las de `~/.pi/agent/skills` y las demás skills del anfitrión siguen visibles.
- `doctor`: mantiene la comprobación de las 21 skills embebidas y añade una línea informativa por carpeta del anfitrión registrada, con el número de subcarpetas que contienen `SKILL.md`.
- Actualizar el principio «el instalador no lee carpetas de otros agentes» en `docs/architecture.md`, el comentario de `InstallSkills` en `pkg/installer/skills.go` y la prueba D5 de `pkg/installer/installer_test.go`, que pasa a comprobar que no se copia ni modifica nada.
- Instalar las dos extensiones nuevas desde `pkg/installer/extension.go`.
- Documentación en README y `docs/installation.md`.

Excluido: copiar o modificar skills de otros agentes, logo como imagen, `/plan` en otros agentes y planificación que ejecute sola.
- Copiar, sincronizar o editar archivos de skills de otros agentes.
- Cambiar el orden de descubrimiento de Pi u ocultar skills del anfitrión distintas de las 21 embebidas.
- Logo como imagen (protocolos kitty/iTerm).
- Carpetas de skills distintas de las listadas, o de proyecto.
- Persistir el plan en un archivo o en gomemory automáticamente.
- Bloqueo exhaustivo de cualquier comando de consola: solo la lista de patrones anterior.

## ¿Qué pasa si no lo hacemos?
Pi arranca sin identidad visual de gomemory, no hay forma de planificar sin riesgo de que el agente edite archivos, y skills que la persona ya usa en OpenCode o Claude Code (impeccable, gitlab-release…) no existen dentro de Pi, así que el mismo flujo funciona en un agente y no en otro.

## Alternativa más barata
Para las skills: copiarlas a mano a `~/.pi/agent/skills`; es inmediato pero la copia se desactualiza y hay que repetirlo. Para `/plan`: escribir «planifica sin editar» en cada prompt; no hay garantía de que el agente lo respete. Para el banner: un `echo` en el launcher; posible, pero queda fuera de la interfaz de Pi. Se descartan porque no cumplen la intención de forma fiable.

Decisión previa revisada: la memoria 11, `pkg/installer/skills.go` y `docs/architecture.md` fijan que pi-gomemory no lee skills de otros agentes. Por petición de la persona (2026-11-10) se cambia solo la parte de «leer»: Pi las referencia en su sitio y pi-gomemory sigue sin copiarlas ni modificarlas.

## Historias
### US1. Banner con el logo al iniciar
Como persona que usa pi-gomemory, quiero ver el logo de la gota y «pi - gomemory» al iniciar, para saber que estoy en el harness con gomemory y qué versión tengo.
- C-01: Dado que lanzo `pi-gomemory` en una terminal truecolor, cuando arranca la sesión, entonces la cabecera muestra la gota en color, el texto «pi - gomemory» y la versión de `PI_GOMEMORY_VERSION` (comprobación manual declarada; el launcher exportando la variable se comprueba con `go test`).
- C-02: Dado que `NO_COLOR` está definida, cuando se genera el banner, entonces el texto no contiene secuencias de escape ANSI y Pi arranca igual.

### US2. Modo /plan de solo lectura
Como persona que planifica un cambio, quiero un modo /plan que impida modificar archivos, para revisar el plan antes de que el agente toque nada.
- C-03: Dado que activé `/plan`, cuando el agente llama a `edit`, `write` o a `bash` con un comando de la lista de patrones, entonces la llamada se bloquea con un motivo que menciona el modo plan; y `bash` con `ls` o `git status` no se bloquea.
- C-04: Dado que activé `/plan`, cuando empieza el turno del agente, entonces el system prompt incluye el texto de `mem plan-context` o, si falla, el texto de respaldo; y el indicador «PLAN» está visible.
- C-05: Dado que el modo plan está activo, cuando escribo `/plan` otra vez o `/plan run`, entonces el indicador desaparece, el texto deja de inyectarse y `edit` y `write` vuelven a funcionar.

### US3. Skills del anfitrión disponibles en Pi
Como persona con skills instaladas en otros agentes, quiero que Pi las vea, para usar impeccable o gitlab-release sin copiarlas.
- C-06: Dado que existe la carpeta `~/.config/opencode/skills` (con `impeccable` y `gitlab-release` dentro), cuando ejecuto `pi-gomemory-setup install`, entonces el array `skills` de `~/.pi/agent/settings.json` contiene la ruta de esa carpeta padre y el resto de claves del usuario se conservan.
- C-07: Dado que `~/.claude/skills` no existe, cuando instalo, entonces esa ruta no se registra y la instalación no falla.
- C-08: Dado que instalo dos veces, cuando reviso `settings.json`, entonces no hay rutas duplicadas en `skills` y ningún archivo de las carpetas del anfitrión cambió (mismo contenido y fecha de modificación).
- C-09: Dado que hay carpetas del anfitrión registradas, cuando ejecuto `pi-gomemory-setup doctor`, entonces aparece una línea por carpeta con su ruta y el número de skills que contiene, y la comprobación de las 21 skills embebidas sigue igual.
- C-10: Dado que una skill embebida (p. ej. `hcd`) existe también en `~/.config/opencode/skills` o `~/.claude/skills`, cuando ejecuto `pi-gomemory-setup install`, entonces `skills` contiene `-~/.config/opencode/skills/hcd` (y la equivalente de Claude), no contiene exclusiones para skills no embebidas como `impeccable`, y al arrancar Pi ya no muestra avisos de colisión para las embebidas (esto último, comprobación manual declarada).

## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->

## Slices
### VS-01. Pi ve las skills del anfitrión y doctor las informa
Tareas: 1, 2, 3, 4
Criterios: C-06, C-07, C-08, C-09, C-10

### VS-02. Banner con la gota y «pi - gomemory» al iniciar
Tareas: 5, 6, 7
Criterios: C-01, C-02

### VS-03. Modo /plan que bloquea modificaciones comunes
Tareas: 8, 9, 10, 11
Criterios: C-03, C-04, C-05

### VS-04. Documentación y suite completa
Tareas: 12, 13
Criterios: C-06, C-01, C-03

## Checklist
- [x] 1. Implementar `RegisterHostSkills(piAgentDir, home, embedded)` en `pkg/installer/hostskills.go`, que añade al array `skills` de `settings.json` las rutas `~/.config/opencode/skills` y `~/.claude/skills` existentes (guardadas con `~`) y una exclusión `-<carpeta>/<nombre>` por cada skill embebida presente en ellas, sin duplicar y conservando las demás claves → `go test ./pkg/installer -run TestRegisterHostSkills`
- [x] 2. Llamar a `RegisterHostSkills` desde `InstallAll` y comprobar que dos instalaciones no duplican rutas ni cambian contenido ni mtime de las carpetas del anfitrión → `go test ./pkg/installer -run TestInstallAllHostSkillsIdempotent`
- [x] 3. Añadir a `doctor` una línea por carpeta del anfitrión registrada con su número de `SKILL.md`, sin tocar la comprobación de las 21 skills → `go test ./pkg/installer -run TestDoctorHostSkills`
- [x] 4. Actualizar el principio en `docs/architecture.md` y el comentario de `InstallSkills` («no copia ni modifica; solo comprueba existencia y cuenta skills del anfitrión»), conservando D5 → `grep -q "no copia ni modifica" docs/architecture.md && go test ./pkg/installer -run TestInstallSkillsFailsWithoutEmbeddedSkills`
- [x] 5. Exportar `PI_GOMEMORY_RUNNING_VERSION` en el launcher a partir de `pi-gomemory-setup version`, sin tocar `PI_GOMEMORY_VERSION` → `go test ./pkg/installer -run TestHarnessExportsRunningVersion`
- [x] 6. Crear `extensions/gomemory-banner.ts` con la gota Unicode, «pi - gomemory» y la versión (omitida si falta o es `dev`), sin ANSI con `NO_COLOR`, vía `ctx.ui.setHeader` en `session_start` → `go test ./pkg/installer -run TestBannerExtensionEmbedded`
- [ ] 7. Instalar la extensión del banner desde `pkg/installer/extension.go` y comprobarla en Pi con y sin launcher → manual declarada: `pi-gomemory` muestra gota, «pi - gomemory» y versión; `pi` solo, sin versión; `NO_COLOR=1 pi-gomemory`, sin color
- [x] 8. Definir los patrones de bloqueo en `extensions/gomemory-plan-patterns.json` (por token de comando, redirección `>`/`>>` salvo `/dev/null`) y probarlos desde Go contra casos que bloquean y que no (`ls`, `git status`, `scp`, `echo x > /dev/null`) → `go test ./pkg/installer -run TestPlanPatterns`
- [x] 9. Crear `extensions/gomemory-plan.ts`: `/plan` alterna el modo, `/plan run` solo desactiva, `tool_call` bloquea `edit`, `write` y `bash` según el JSON, indicador «PLAN» → `go test ./pkg/installer -run TestPlanExtensionEmbedded`
- [x] 10. Inyectar en `before_agent_start` la salida de `mem plan-context` (timeout 5 s, cwd de la sesión; salida vacía o error usa el respaldo marcado `[gomemory-plan:fallback]`) → `go test ./pkg/installer -run TestPlanExtensionFallbackMarker`
- [ ] 11. Instalar la extensión /plan desde `pkg/installer/extension.go` y comprobarla en Pi → manual declarada: con `/plan` se bloquean `edit` y `rm x`, pasa `ls`, aparece «PLAN»; con `/plan` otra vez o `/plan run` vuelve `edit` y desaparece el indicador
- [x] 12. Documentar el banner, `/plan` y las skills del anfitrión en README y `docs/installation.md` → `grep -q "/plan" README.md && grep -q "config/opencode/skills" docs/installation.md`
- [x] 13. Ejecutar la suite completa → `go test ./... && go vet ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-10 22:19 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestRegisterHostSkills` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd002_test.go:29:12: undefined: RegisterHostSkills
      pkg/installer/hcd002_test.go:69:12: undefined: RegisterHostSkills
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 22:19 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestInstallAllHostSkillsIdempotent` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd002_test.go:29:12: undefined: RegisterHostSkills
      pkg/installer/hcd002_test.go:69:12: undefined: RegisterHostSkills
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 22:19 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestDoctorHostSkills` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd002_test.go:29:12: undefined: RegisterHostSkills
      pkg/installer/hcd002_test.go:69:12: undefined: RegisterHostSkills
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 22:19 -05:00 · tarea 4 · rojo observado · caducada · `grep -q "no copia ni modifica" docs/architecture.md && go test ./pkg/installer -run TestInstallSkillsFailsWithoutEmbeddedSkills` (salida 1)
- 2026-11-10 22:19 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run TestHarnessExportsRunningVersion` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd002_test.go:29:12: undefined: RegisterHostSkills
      pkg/installer/hcd002_test.go:69:12: undefined: RegisterHostSkills
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 22:19 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run TestBannerExtensionEmbedded` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd002_test.go:29:12: undefined: RegisterHostSkills
      pkg/installer/hcd002_test.go:69:12: undefined: RegisterHostSkills
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 22:19 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run TestPlanPatterns` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd002_test.go:29:12: undefined: RegisterHostSkills
      pkg/installer/hcd002_test.go:69:12: undefined: RegisterHostSkills
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 22:19 -05:00 · tarea 9 · rojo observado · caducada · `go test ./pkg/installer -run TestPlanExtensionEmbedded` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd002_test.go:29:12: undefined: RegisterHostSkills
      pkg/installer/hcd002_test.go:69:12: undefined: RegisterHostSkills
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 22:19 -05:00 · tarea 10 · rojo observado · caducada · `go test ./pkg/installer -run TestPlanExtensionFallbackMarker` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd002_test.go:29:12: undefined: RegisterHostSkills
      pkg/installer/hcd002_test.go:69:12: undefined: RegisterHostSkills
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-10 22:19 -05:00 · tarea 12 · rojo observado · caducada · `grep -q "/plan" README.md && grep -q "config/opencode/skills" docs/installation.md` (salida 1)
- 2026-11-10 22:20 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestRegisterHostSkills` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.192s
- 2026-11-10 22:20 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestInstallAllHostSkillsIdempotent` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.224s
- 2026-11-10 22:20 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestDoctorHostSkills` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.208s
- 2026-11-10 22:20 -05:00 · tarea 4 · verde · vigente · `grep -q "no copia ni modifica" docs/architecture.md && go test ./pkg/installer -run TestInstallSkillsFailsWithoutEmbeddedSkills` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.208s
- 2026-11-10 22:20 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run TestHarnessExportsRunningVersion` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.180s
- 2026-11-10 22:20 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run TestBannerExtensionEmbedded` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.223s
- 2026-11-10 22:20 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run TestPlanPatterns` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.192s
- 2026-11-10 22:21 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run TestPlanExtensionEmbedded` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.206s
- 2026-11-10 22:21 -05:00 · tarea 10 · verde · vigente · `go test ./pkg/installer -run TestPlanExtensionFallbackMarker` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.204s
- 2026-11-10 22:21 -05:00 · tarea 12 · verde · vigente · `grep -q "/plan" README.md && grep -q "config/opencode/skills" docs/installation.md` (salida 0)
- 2026-11-10 22:21 -05:00 · tarea 13 · verde · vigente · `go test ./... && go vet ./...` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	(cached)

## Siguiente
pendiente

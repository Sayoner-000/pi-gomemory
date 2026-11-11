# Desinstalador de pi-gomemory con modo depuración

ID: HCD-011
Tipo: feature
Carril: full
Motivo: Borra archivos del usuario (riesgo de datos) y hay varias decisiones abiertas: qué se elimina, conservar memoria, dry-run y log

## Intención
Añadir `pi-gomemory-setup uninstall` para retirar lo que instala `pi-gomemory-setup install`, de modo que la persona pueda depurar un fallo o reinstalar desde cero. Debe ofrecer `--dry-run` (listar sin borrar), `--verbose` (detallar cada paso y guardarlo en un log) y conservar por defecto la memoria de gomemory.

Inventario de lo que hoy escribe `InstallAll` (pkg/installer/installer.go):
1. `~/.pi/agent/mcp.json`: entradas `gomemory`, `codegraph` y `codebase-memory` en `mcpServers`.
2. `~/.pi/agent/themes/gomemory.json` y `"theme": "gomemory"` en `settings.json`.
3. y 4. Paquetes `npm:pi-antigravity@0.12.0` y `npm:pi-cc-extensions@1.0.16` en `settings.json` (`packages`), su instalación con `pi install`, `~/.pi/agent/pi-cc-extensions.json` (solo se añaden las claves que faltaban) y el parche de `peerDependencies` en `~/.pi/agent/npm/node_modules/pi-antigravity/package.json` (plugin.go:100).
5. Bloque `<!-- pi-gomemory:start -->…<!-- pi-gomemory:end -->` en `~/.pi/agent/AGENTS.md` (o el protocolo sin marcadores de instalaciones ≤ v1.0.0).
6. `~/.pi-gomemory/skills` (copias embebidas con marca `.pi-gomemory` y enlaces simbólicos a skills del anfitrión) y su registro en `settings.json` (`skills`); copias embebidas antiguas en `~/.pi/agent/skills`.
7. Extensiones `gomemory-*.ts`, `gomemory-plan-patterns.json` y `gomemory-splash-prompt.md` en `~/.pi/agent/extensions/`.
8. Launcher `pi-gomemory` en `~/.local/bin` (o `PI_GOMEMORY_BIN_DIR`), generado desde texto en código (harness.go); en cada arranque ejecuta `skills-sync` y refresca cachés.
Además: cachés `~/.pi/agent/pi-gomemory-deps.json` y `~/.pi/agent/pi-gomemory-update.json`, y el propio binario `pi-gomemory-setup`.

## Alcance
Regla de propiedad (revisión Splash H1): `install` escribe `~/.pi-gomemory/manifest.json` con la ruta, el SHA256 de cada archivo que escribe y el `BIN_DIR` usado. `uninstall` borra un archivo solo si su SHA256 coincide con el del manifiesto, sea cual sea la versión del binario. Sin manifiesto (instalaciones anteriores), compara con el contenido que generaría esta versión. Lo que no coincide se omite y se informa. Las entradas MCP se retiran solo si coinciden con `DefaultMCPServers()` (H14).
Incluido: escritura del manifiesto en `install`; subcomando `uninstall` en `main.go` y su ayuda; retirada de cada elemento del inventario quitando solo las entradas propias en los JSON compartidos (`mcp.json`, `settings.json`) y solo el bloque delimitado (o el protocolo sin marcadores) en `AGENTS.md`, borrando `AGENTS.md` si queda vacío; borrado de skills embebidas intactas (carpeta con exactamente `SKILL.md` y `.pi-gomemory`; si contiene más archivos se omite, H10), incluidas las copias antiguas de `~/.pi/agent/skills`; borrado de enlaces simbólicos del repositorio solo si su destino está bajo una fuente de skills del anfitrión, sin seguirlos (H9); borrado de las dos cachés (H6); borrado de `~/.pi-gomemory/skills` y `~/.pi-gomemory` solo si quedan vacíos, nunca de `themes/` ni `extensions/` (H11); el launcher se busca en el `BIN_DIR` del manifiesto, en `PI_GOMEMORY_BIN_DIR`, en `~/.local/bin` y junto al ejecutable (H7); marca de desinstalado `~/.pi-gomemory/uninstalled` que hace que `skills-sync` no haga nada y que `install` retira (H2); `--dry-run` que lista cada acción sin escribir; `--verbose` que detalla cada acción y la guarda en un archivo de log; confirmación interactiva tras mostrar el plan, omitible con `--yes`, y parada sin borrar si no hay terminal interactiva ni `--yes` (Q4); la clave `theme` de `settings.json` se borra solo si vale `gomemory` y el archivo del tema se borra en la misma ejecución; si el archivo se conserva, la clave también (Q2, H3); opción `--packages` que retira solo las entradas con la fuente fijada exacta (con `pi remove`) y borra `pi-cc-extensions.json` solo si coincide con los valores por defecto embebidos; un paquete que la persona instaló antes no se distingue y se retira igual (Q1, H5); sin `--packages`, el parche de `pi-antigravity/package.json` se conserva y no se revierte (H4); opción `--self` que borra `os.Executable()` sin resolver enlaces y solo si está en el `BIN_DIR` efectivo; si no, se omite e informa (Q3, H8); en `doctor`, una línea `INFO` «pi-gomemory no instalado» cuando no queda ningún componente, sin exigir `pi` (H12); resumen final de lo retirado, lo conservado y lo omitido; que `doctor` informe cuando no queda ningún componente de pi-gomemory (C-12); tests en sandbox; documentación en `docs/usage.md` y `docs/installation.md`.
Excluido: un atajo `install --clean` (para reinstalar limpio basta `uninstall` y luego `install`); borrar la memoria de gomemory (`mem`, su base de datos, `.gomemory/` o `hcd/` de los proyectos); desinstalar `mem`, `codegraph`, `codebase-memory-mcp` o Pi; tocar skills del anfitrión a las que apuntan los enlaces (`~/.claude/skills`, `~/.config/opencode/skills`…); borrar skills propias de la persona o copias embebidas que haya editado; servidores MCP, paquetes o claves de JSON ajenos a pi-gomemory.

## ¿Qué pasa si no lo hacemos?
Para depurar o reinstalar limpio, la persona tiene que deshacer a mano ocho pasos repartidos por `~/.pi/agent`, `~/.pi-gomemory` y `~/.local/bin`, y editar JSON compartidos. Es fácil dejar restos (el tema activo apuntando a un archivo borrado, extensiones huérfanas que siguen cargándose) o borrar de más (skills propias, otros servidores MCP). Eso hace difícil distinguir un fallo real de uno causado por restos de una instalación anterior.

## Alternativa más barata
Documentar en `docs/installation.md` una lista de pasos manuales. Cuesta casi nada, pero no resuelve el riesgo: la edición de los JSON y del bloque de `AGENTS.md` a mano es justo lo que causa restos o borrados de más, y no ofrece `--dry-run`. Se descarta como solución, aunque la documentación del comando sí entra en el alcance.

## Historias
### US1. Ver qué se borraría antes de hacerlo
Como persona que depura pi-gomemory, quiero ejecutar `pi-gomemory-setup uninstall --dry-run`, para saber exactamente qué se retiraría sin arriesgar nada.
- C-01: Dado una instalación completa, cuando ejecuto `uninstall --dry-run`, entonces se lista cada archivo, entrada de JSON y bloque que se retiraría, y ningún archivo cambia (mismo contenido, comprobado en un sandbox sin el launcher en ejecución, H15).

### US2. Desinstalar para reinstalar limpio
Como persona que depura pi-gomemory, quiero que `pi-gomemory-setup uninstall` retire todo lo que instaló, para poder reinstalar desde cero y descartar restos.
- C-02: Dado una instalación completa, cuando ejecuto `uninstall`, entonces desaparecen las extensiones `gomemory-*`, el tema, el launcher, las copias embebidas intactas y los enlaces de `~/.pi-gomemory/skills`, las tres entradas MCP, el bloque delimitado de `AGENTS.md`, el registro del repositorio de skills en `settings.json`, las dos cachés y el manifiesto.
- C-12: Dado que acabo de desinstalar, cuando ejecuto `pi-gomemory-setup doctor`, entonces muestra la línea `INFO` «pi-gomemory no instalado», aunque `pi` no esté disponible.
- C-16: Dado que instalé con una versión anterior de pi-gomemory, cuando ejecuto `uninstall` con una versión más nueva, entonces se retiran los archivos que coinciden con el manifiesto y no se omiten como «editados».
- C-17: Dado que instalé con `PI_GOMEMORY_BIN_DIR=<dir>`, cuando ejecuto `uninstall` sin esa variable, entonces el launcher de `<dir>` se retira.
- C-18: Dado que acabo de desinstalar sin `--self`, cuando se ejecuta `pi-gomemory-setup skills-sync` (por ejemplo, desde un launcher conservado), entonces no se recrean `~/.pi-gomemory/skills` ni su registro; y tras `install` vuelve a funcionar.
- C-03: Dado que acabo de desinstalar, cuando ejecuto `pi-gomemory-setup install`, entonces la instalación termina sin errores y `doctor` queda igual que en una instalación nueva.
- C-13: Dado una instalación completa, cuando ejecuto `uninstall` sin `--packages` ni `--self`, entonces `pi-antigravity`, `pi-cc-extensions`, `pi-cc-extensions.json` y el binario `pi-gomemory-setup` se conservan; con `--packages` se retiran los paquetes y su config, y con `--self` se borra el binario.
- C-14: Dado que `settings.json` tiene `"theme"` con un valor distinto de `gomemory`, cuando ejecuto `uninstall`, entonces esa clave se conserva; si vale `gomemory` y el archivo del tema se retira, se elimina; si el archivo del tema se conserva por estar editado, la clave también.
- C-15: Dado que no hay terminal interactiva y no paso `--yes`, cuando ejecuto `uninstall`, entonces no se modifica ningún archivo y el comando sale con código distinto de cero indicando que use `--yes`; en terminal interactiva, si no confirmo, tampoco se modifica nada.
- C-04: Dado que desinstalo dos veces seguidas, cuando ejecuto el segundo `uninstall`, entonces termina sin error e informa de que no queda nada que retirar.

### US3. No perder lo que no es de pi-gomemory
Como persona que usa Pi con otras herramientas, quiero que el desinstalador conserve mi memoria y mis configuraciones, para no perder trabajo al depurar.
- C-05: Dado que `mcp.json` y `settings.json` tienen servidores, paquetes y claves ajenos, cuando ejecuto `uninstall`, entonces esos elementos siguen presentes con el mismo valor; y una entrada `gomemory`, `codegraph` o `codebase-memory` que no coincide con la definición de pi-gomemory se conserva y se informa como omitida.
- C-06: Dado que `AGENTS.md` tiene texto de la persona fuera del bloque delimitado, cuando ejecuto `uninstall`, entonces ese texto se conserva, salvo los saltos de línea que añadió el instalador alrededor del bloque.
- C-07: Dado una skill propia, una copia embebida editada o con archivos añadidos, o un enlace propio que apunta fuera de las fuentes del anfitrión en `~/.pi-gomemory/skills`, cuando ejecuto `uninstall`, entonces se conserva y el resumen lo menciona como omitido; y la carpeta del anfitrión a la que apunta un enlace no se toca.
- C-11: Dado que edité a mano una extensión `gomemory-*`, el tema o el launcher (su SHA256 ya no coincide con el del manifiesto), cuando ejecuto `uninstall`, entonces ese archivo se conserva y el resumen lo menciona como omitido.
- C-08: Dado que existe la memoria de gomemory, cuando ejecuto `uninstall` sin opciones, entonces la base de datos de `mem` y los directorios `.gomemory/` y `hcd/` siguen intactos.

### US4. Registrar la desinstalación
Como persona que depura pi-gomemory, quiero `uninstall --verbose`, para ver cada paso y conservar un log que pueda revisar o compartir.
- C-09: Dado una instalación completa, cuando ejecuto `uninstall --verbose`, entonces cada acción (retirado, conservado, omitido o fallido) se imprime y se escribe en un archivo de log cuya ruta se muestra al final.
- C-10: Dado que una acción falla (por ejemplo, un JSON ilegible), cuando ejecuto `uninstall`, entonces ese archivo no se modifica, el resto de acciones continúa, el error aparece en el resumen y el comando sale con código distinto de cero.

## Aclaraciones
- [x] Q1. ¿Debe uninstall quitar también los paquetes pi-antigravity y pi-cc-extensions (settings.json y pi remove) y ~/.pi/agent/pi-cc-extensions.json, o se conservan? → Conservar por defecto (Recommended): se conservan y solo se quitan con la opción explícita --packages
- [x] Q2. Al retirar el tema, ¿se borra la clave theme de settings.json o se intenta restaurar el tema anterior, que hoy no se guarda? → Borrar si es gomemory (Recommended): se borra la clave theme solo si vale gomemory y Pi vuelve a su tema por defecto
- [x] Q3. ¿Debe uninstall borrar el propio binario pi-gomemory-setup? → Conservar (Recommended): el binario se conserva para reinstalar y solo se borra con la opción explícita --self
- [x] Q4. ¿Pide confirmación interactiva antes de borrar (con --yes para omitirla) o actúa directamente porque ya existe --dry-run? → Confirmar con --yes (Recommended): muestra el plan y pide confirmación en terminal interactiva; --yes la omite; sin terminal ni --yes se detiene sin borrar
- [x] Q5. Revisión Splash H1: ¿cómo demuestra uninstall que algo lo instaló pi-gomemory? → Manifiesto al instalar (Recommended): install escribe ~/.pi-gomemory/manifest.json con ruta, SHA256 y BIN_DIR; sin manifiesto se compara con esta versión
- [x] Q6. Revisión Splash H2: el launcher ejecuta skills-sync en cada arranque y recrea skills; ¿qué hacemos? → Marca de desinstalado (Recommended): uninstall deja ~/.pi-gomemory/uninstalled, skills-sync no actúa mientras exista e install la retira

## Slices
### VS-01. install deja un manifiesto y skills-sync respeta la marca de desinstalado
Tareas: 1, 2
Criterios: C-16, C-17, C-18

### VS-02. uninstall --dry-run muestra el plan sin tocar nada
Tareas: 3, 4
Criterios: C-01, C-05, C-06, C-07, C-11, C-14

### VS-03. uninstall retira lo propio con confirmación, --verbose y log
Tareas: 5, 6, 7
Criterios: C-02, C-04, C-08, C-09, C-10, C-15

### VS-04. --packages, --self, doctor «no instalado» y documentación
Tareas: 8, 9, 10
Criterios: C-03, C-12, C-13

## Checklist
- [x] 1. Escribir `~/.pi-gomemory/manifest.json` (ruta, SHA256, BIN_DIR) al final de `InstallAll` y retirar la marca de desinstalado → `go test ./pkg/installer -run TestHCD011Manifest`
- [x] 2. Hacer que `skills-sync` no actúe si existe `~/.pi-gomemory/uninstalled` → `go test ./pkg/installer -run TestHCD011SkillsSyncRespectsMarker`
- [x] 3. Construir el plan de desinstalación (retirar, conservar, omitir) aplicando la regla de propiedad a archivos, skills, enlaces, tema, MCP y AGENTS.md → `go test ./pkg/installer -run TestHCD011Plan`
- [x] 4. Añadir `uninstall --dry-run` que imprime el plan sin escribir → `go test ./pkg/installer -run TestHCD011DryRun`
- [x] 5. Ejecutar el plan con edición parcial de JSON, bloque de AGENTS.md y marca de desinstalado, continuando ante fallos con salida distinta de cero → `go test ./pkg/installer -run TestHCD011Apply`
- [x] 6. Pedir confirmación interactiva, aceptar `--yes` y detenerse sin terminal ni `--yes` → `go test ./pkg/installer -run TestHCD011Confirm`
- [x] 7. Añadir `--verbose` que imprime cada acción y la escribe en un log cuya ruta se muestra → `go test ./pkg/installer -run TestHCD011Verbose`
- [x] 8. Añadir `--packages` (fuente fijada exacta, config por defecto) y `--self` (os.Executable en BIN_DIR, sin resolver enlaces) → `go test ./pkg/installer -run TestHCD011Options`
- [x] 9. Mostrar en `doctor` la línea INFO «pi-gomemory no instalado» sin exigir `pi`, y comprobar reinstalación limpia → `go test ./pkg/installer -run 'TestHCD011Doctor|TestHCD011Reinstall'`
- [x] 10. Documentar `uninstall` en `docs/usage.md`, `docs/installation.md` y la ayuda de `main.go` → `go build ./... && go run . help | grep -q uninstall && grep -q 'uninstall' docs/usage.md docs/installation.md`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 07:20 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD011Manifest` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd011_test.go:22:15: undefined: UninstalledMarkerPath
      pkg/installer/hcd011_test.go:27:12: undefined: ReadManifest
      pkg/installer/hcd011_test.go:47:23: undefined: UninstalledMarkerPath
      pkg/installer/hcd011_test.go:50:24: undefined: ManifestPath
      pkg/installer/hcd011_test.go:59:15: undefined: UninstalledMarkerPath
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 07:20 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD011SkillsSyncRespectsMarker` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd011_test.go:22:15: undefined: UninstalledMarkerPath
      pkg/installer/hcd011_test.go:27:12: undefined: ReadManifest
      pkg/installer/hcd011_test.go:47:23: undefined: UninstalledMarkerPath
      pkg/installer/hcd011_test.go:50:24: undefined: ManifestPath
      pkg/installer/hcd011_test.go:59:15: undefined: UninstalledMarkerPath
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 07:21 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD011Manifest` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.215s
- 2026-11-11 07:21 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD011SkillsSyncRespectsMarker` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.138s
- 2026-11-11 07:22 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD011Plan` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd011_plan_test.go:36:22: undefined: UninstallPlan
      pkg/installer/hcd011_plan_test.go:36:53: undefined: UninstallAction
      pkg/installer/hcd011_plan_test.go:71:15: undefined: BuildUninstallPlan
      pkg/installer/hcd011_plan_test.go:71:55: undefined: UninstallFlags
      pkg/installer/hcd011_plan_test.go:76:60: undefined: ActionRemove
      pkg/installer/hcd011_plan_test.go:77:60: undefined: ActionSkip
      pkg/installer/hcd011_plan_test.go:78:62: undefined: ActionRemove
      pkg/installer/hcd011_plan_test.go:79:60: undefined: ActionRemove
      pkg/installer/hcd011_plan_test.go:80:60: undefined: ActionRemove
      pkg/installer/hcd011_plan_test.go:81:60: undefined: ActionSkip
      pkg/installer/hcd011_plan_test.go:81:60: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 07:22 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD011DryRun` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd011_plan_test.go:36:22: undefined: UninstallPlan
      pkg/installer/hcd011_plan_test.go:36:53: undefined: UninstallAction
      pkg/installer/hcd011_plan_test.go:71:15: undefined: BuildUninstallPlan
      pkg/installer/hcd011_plan_test.go:71:55: undefined: UninstallFlags
      pkg/installer/hcd011_plan_test.go:76:60: undefined: ActionRemove
      pkg/installer/hcd011_plan_test.go:77:60: undefined: ActionSkip
      pkg/installer/hcd011_plan_test.go:78:62: undefined: ActionRemove
      pkg/installer/hcd011_plan_test.go:79:60: undefined: ActionRemove
      pkg/installer/hcd011_plan_test.go:80:60: undefined: ActionRemove
      pkg/installer/hcd011_plan_test.go:81:60: undefined: ActionSkip
      pkg/installer/hcd011_plan_test.go:81:60: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 07:23 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD011Plan` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.220s
- 2026-11-11 07:23 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD011DryRun` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.183s
- 2026-11-11 07:25 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD011Manifest` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.212s
- 2026-11-11 07:25 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD011SkillsSyncRespectsMarker` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.143s
- 2026-11-11 07:25 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD011Plan` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.195s
- 2026-11-11 07:25 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD011DryRun` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.186s
- 2026-11-11 07:25 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run TestHCD011Apply` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.254s
- 2026-11-11 07:25 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run TestHCD011Confirm` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.221s
- 2026-11-11 07:25 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run TestHCD011Verbose` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.224s
- 2026-11-11 07:27 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD011Options` (salida 1)
        [38;2;255;119;127m✗[0m Binario pi                       [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD011Options1624233210/005/.pi/agent/themes/gomemory.json[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD011Options1624233210/005/.local/bin/pi-gomemory[0m
      --- FAIL: TestHCD011Options (0.22s)
          hcd011_options_test.go:56: --packages: quiero solo el paquete ajeno, obtengo [npm:otro-paquete@1.0.0 npm:pi-antigravity@0.12.0 npm:pi-cc-extensions@1.0.16]
          hcd011_options_test.go:59: --packages no borró pi-cc-extensions.json con valores por defecto
          hcd011_options_test.go:62: --self no borró el binario del BIN_DIR
          hcd011_options_test.go:71: pi-cc-extensions.json editado: quiero omitir, obtengo <nil>
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.584s
      FAIL
- 2026-11-11 07:27 -05:00 · tarea 9 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD011Doctor|TestHCD011Reinstall'` (salida 1)
      ════════════════╗[0m
      [38;2;34;227;247m║           DIAGNÓSTICO DEL HARNESS PI-GOMEMORY                      ║[0m
      [38;2;6;193;238m╚════════════════════════════════════════════════════════════════════╝[0m
        [38;2;244;198;107m⚠[0m Portapapeles del sistema         [38;2;145;169;197mse requieren pbcopy y pbpaste del sistema[0m
        [38;2;255;119;127m✗[0m Binario pi                       [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD011Reinstall1347121746/003/.pi/agent/themes/gomemory.json[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD011Reinstall1347121746/003/.local/bin/pi-gomemory[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.453s
      FAIL
- 2026-11-11 07:28 -05:00 · tarea 10 · rojo observado · caducada · `go build ./... && go run . help | grep -q uninstall && grep -q 'uninstall' docs/usage.md docs/installation.md` (salida 1)
- 2026-11-11 07:28 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestHCD011Manifest` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.247s
- 2026-11-11 07:28 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD011SkillsSyncRespectsMarker` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.182s
- 2026-11-11 07:29 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD011Plan` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.266s
- 2026-11-11 07:29 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestHCD011DryRun` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.238s
- 2026-11-11 07:29 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run TestHCD011Apply` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.339s
- 2026-11-11 07:29 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run TestHCD011Confirm` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.283s
- 2026-11-11 07:29 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run TestHCD011Verbose` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.284s
- 2026-11-11 07:29 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run TestHCD011Options` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.404s
- 2026-11-11 07:29 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run 'TestHCD011Doctor|TestHCD011Reinstall'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.452s
- 2026-11-11 07:29 -05:00 · tarea 10 · verde · vigente · `go build ./... && go run . help | grep -q uninstall && grep -q 'uninstall' docs/usage.md docs/installation.md` (salida 0)

## Siguiente
Cerrada con aprobación sobre la revisión daa4b67abc56.

# Catálogo único de componentes con sync, selector y última versión

ID: HCD-076
Tipo: feature
Carril: full
Motivo: Cambia el contrato público de install/update e install.sh; varias decisiones interdependientes (catálogo, reconciliación, selector, latest)

## Intención
Que `curl -fsSL https://raw.githubusercontent.com/Sayoner-000/pi-gomemory/main/install.sh | bash` instale la última release de pi-gomemory y deje pi, mem, codegraph, codebase-memory-mcp y herdr (opcional) instalados y al día desde sus canales oficiales. Todo sale de un catálogo único de componentes, con un selector en la terminal y un comando `/componentes` dentro de Pi.

Decisiones de la persona (2026-11-24): herdr es opcional; install.sh usa siempre la última release; las herramientas presentes pero desactualizadas se actualizan sin preguntar; todo en un solo HCD; se aplican las correcciones H1–H15 de Splash.

Antecedentes que se conservan: HCD-062 (canales oficiales, SHA-256, `/dev/tty`, entrada a `/dev/null`, puntos de sustitución `toolHooks`), HCD-029 (el launcher sigue preguntando antes de actualizar al abrir), HCD-031/039/040 (deps-status/refresh/update, caché de 1 h, pins de pi-antigravity y pi-cc-extensions), HCD-035 (Force en las acciones explícitas), HCD-063 (pin de pi-antigravity 0.15.0).

## Alcance
Incluido: catálogo único, herdr opcional, sync con selector, `/componentes`, latest en install.sh, comprobación de Node.js y doctor con opcionales en WARN:
- **Catálogo único** `Component` en `pkg/installer`, en este orden y con tres tipos:
  - `self`: pi-gomemory (se actualiza con `pi-gomemory-setup update`).
  - `tool` (binarios externos): pi (obligatorio), mem (obligatorio), codegraph (opcional), codebase-memory-mcp (opcional) y herdr (opcional).
  - `package` (paquetes de Pi fijados): pi-antigravity y pi-cc-extensions.
- **Listas que se derivan del catálogo:**
  - de los `tool`: `productionTools()`, los binarios de `RunDoctor`, `printToolsKept` (con el comando de desinstalación de cada uno), `pi-gomemory-setup components --names` (que la lista de PATH de `install.sh` usa además de `pi-gomemory-setup` y `pi-gomemory`) y los textos con nombres de herramientas del paso [1/9] (`installer.go`), del aviso de `PI_GOMEMORY_SKIP_DEPS` y del uso de `deps-install` (`tools.go`);
  - de todos los tipos: `DefaultDepProbes()`.
- **Herdr como `tool` opcional.** Canales observados el 2026-11-24: `https://herdr.dev/install.sh` responde HTTP 200, verifica SHA-256 y acepta `HERDR_INSTALL_DIR`, que recibe el directorio de binarios de la instalación (ya está en `knownToolDirs`). La versión instalada se lee con `herdr --version` (`herdr 0.9.5`). La última sale de la redirección de `https://github.com/herdrdev/herdr/releases/latest` (a `v0.9.5`). Se actualiza con `herdr update`, que solo admite `--handoff` y no tiene opciones interactivas. Intérprete `sh` (el script empieza por `#!/bin/sh`). Instalación manual: `curl -fsSL https://herdr.dev/install.sh | sh`. Desinstalación: `rm <bindir>/herdr`, porque herdr no tiene subcomando para desinstalarse. La evidencia queda en la sección «Evidencia de los canales» y en un comentario fechado en `tools.go`, como en HCD-062.
- **Subcomando `pi-gomemory-setup sync [--yes] [--only=a,b | --skip=c]`.** Solo reconcilia los `tool`, así que no hay ciclo con `self`. Para cada uno:
  - si falta, se instala por su canal oficial;
  - si está desactualizado, se actualiza con su comando oficial sin preguntar, con la entrada en `/dev/null` y la salida por tubería;
  - si está al día, no se hace nada.

  Además:
  - Ignora la caché de 1 h y refresca a la fuerza los tres tipos aunque `PI_GOMEMORY_NO_UPDATE_CHECK` esté puesta, porque es una acción explícita (como `update`, HCD-035). Tras reconciliar, reescribe la caché con todos los componentes: los `tool` tocados con la versión instalada después de actualizarlos, y pi-gomemory y los pins con el estado refrescado.
  - `sync --json` informa, para cada `tool`, `name`, `action` (`installed`, `updated`, `current`, `failed` o `skipped`), `from` y `to`; sin `--json` imprime lo mismo en texto.
  - No poder instalar un obligatorio que falta es error fatal, con el comando manual. Que falle la actualización de un obligatorio ya presente, o cualquier fallo de un opcional, es aviso y se sigue.
  - `PI_GOMEMORY_SKIP_DEPS` lo omite entero.
- **Flags de `sync`.**
  - `--only` y `--skip` no pueden excluir obligatorios: estos se reconcilian siempre.
  - Combinar `--only` y `--skip`, usar un nombre que no está en el catálogo o un componente que no es `tool` (pi-gomemory o un pin) termina con código 2 y un mensaje claro.
  - En modo lite, la selección por defecto (sin terminal, con `--yes` o al pulsar Enter) no incluye los opcionales: no se instalan ni se actualizan, como `EnsureTools` hoy.
  - Cualquiera de `--yes`, `--only` o `--skip` evita el selector.
- **Lugar de `sync` en `install`:** sustituye a `EnsureTools` en el paso [1/9] de `InstallAll`, porque sin pi no se puede configurar nada después. Los pins siguen instalándose en los pasos [4/9] y [5/9], como hoy.
- **Lugar de `sync` en `update`:**
  - con release nueva, se reemplaza el binario y se relanza `install`, que hace un solo `sync --yes` sin selector;
  - con pi-gomemory al día, `update` ejecuta `sync --yes` directamente;
  - si falla la consulta de la release, `update` termina con error como hoy, no ejecuta `sync` y sugiere `pi-gomemory-setup sync`.
- **`deps-update` (lo usa el launcher, HCD-029):** si pi-gomemory está desactualizado, solo ejecuta `update` y para, porque su `sync` ya reconcilia los `tool`. Si no lo está, delega los `tool` desactualizados en `sync --yes --only=…` y actualiza los pins como hoy. Así ninguna herramienta se actualiza dos veces.
- **Selector de casillas en `/dev/tty`** cuando hay terminal y no se pasan flags: nombre, versión instalada, última versión y si es obligatorio u opcional. Los obligatorios están marcados y bloqueados. Los opcionales vienen marcados, salvo en modo lite, donde aparecen desmarcados. `install` (y por tanto `curl | bash`) lo muestra cuando hay terminal; la entrada estándar sigue en `/dev/null`.
- **Subcomando `pi-gomemory-setup components [--names | --json]`.**
  - `--json` refresca de forma explícita (como `sync`) y devuelve, para cada componente del catálogo y en su orden: `name`, `kind` (`self`, `tool` o `package`), `required`, `installed`, `current`, `latest`, `outdated` y `error`. Un componente ausente tiene `installed:false`, `current:""` y `outdated:false`. Ejemplo: `[{"name":"herdr","kind":"tool","required":false,"installed":true,"current":"0.9.4","latest":"v0.9.5","outdated":true,"error":""}]`.
  - `--names` imprime los nombres de los `tool`, uno por línea, sin red.
- **Comando `/componentes` en la extensión `gomemory-status.ts`:**
  - lista los componentes de `components --json`;
  - permite elegir entre los `tool` (no ofrece pi-gomemory ni los pins);
  - ejecuta `pi-gomemory-setup sync --json --only=…` con un plazo de 60 min; si vence, notifica «sync no terminó; ejecuta `pi-gomemory-setup sync` en la terminal»;
  - notifica el resultado y, si pi quedó `updated` o `installed` en la salida de `sync --json` (esté o no en la selección, porque los obligatorios se reconcilian siempre), avisa de que hay que reiniciar Pi.

  Funciona aunque `PI_GOMEMORY_NO_UPDATE_CHECK` esté puesta, porque es una acción explícita. El aviso pasivo del launcher (HCD-029) no cambia.
- **`install.sh` y la última versión.**
  - La etiqueta solo se resuelve en la opción B (descarga de la release); la compilación local (opción A, HCD-062 C-11) no consulta la red.
  - Sin `PI_GOMEMORY_VERSION`, también con `PI_GOMEMORY_BASE_URL` puesta (hace falta la etiqueta para el nombre del tarball), resuelve la etiqueta leyendo la cabecera `Location` de `PI_GOMEMORY_LATEST_URL` (por defecto `https://github.com/Sayoner-000/pi-gomemory/releases/latest`).
  - Con la variable, usa esa etiqueta.
  - Si la resolución falla, aborta con un mensaje que sugiere `PI_GOMEMORY_VERSION`.
  - La verificación SHA256SUMS y `PI_GOMEMORY_BASE_URL` no cambian.
- **Comprobación de Node.js ≥ 22.19 y npm** solo cuando hay que instalar pi porque falta, antes de ejecutar su instalador, con un mensaje de cómo instalarlos.
- **Doctor:** un `tool` opcional ausente se marca WARN («opcional, no instalado»); un obligatorio ausente sigue siendo FAIL.

Excluido: Windows, cambiar pins, configuración de herdr, desinstalar herramientas externas, cambiar el aviso del launcher e instalar Node.js:
- Windows y PowerShell.
- Cambiar los pins de pi-antigravity o pi-cc-extensions, o reconciliarlos en `sync`.
- Configuración de herdr (keybindings, `config.toml`) e integración del launcher con el binario herdr.
- Que `uninstall` desinstale las herramientas externas: sigue conservándolas e informándolas.
- Cambiar el aviso del launcher (HCD-029), que sigue preguntando antes de actualizar al abrir.
- Instalar Node.js automáticamente.
- Renombrar el subcomando existente `pi-gomemory-setup harness`.

## ¿Qué pasa si no lo hacemos?
Una instalación desde cero instala la versión fijada en install.sh (hoy v1.25.1) y no la última si nadie sube el pin. Las herramientas que ya existían en una versión vieja se quedan viejas tras `install` (`EnsureTools` solo dice «ya está instalado»). Herdr no se instala aunque el mensaje final invite a usar Pi en Herdr. La lista de componentes vive en cuatro sitios que se desincronizan al añadir una herramienta, y la persona no puede elegir qué instalar ni ver el estado del meta-harness desde Pi.

## Alternativa más barata
Tres cambios puntuales sin catálogo: resolver latest en install.sh, añadir herdr a `productionTools()` y `DefaultDepProbes()`, y llamar a `RunDepsUpdate` tras `EnsureTools`. Cubre la instalación desde cero y la actualización, pero mantiene las cuatro listas duplicadas (más una quinta con herdr) y no da selector ni `/componentes`. Se descarta porque la persona pidió una lista única con selección.

Desviación de la Constitución (fuente: `mem constitution` v2.2.0, revisión del motor `e4a52d5ce2e0`, §0 «Política Context7», texto literal: «No se permite citar `latest` como versión. La referencia reproducible es una versión estable concreta, un archivo de bloqueo y, para imágenes, preferiblemente un digest.»): install.sh no declara `latest` como dependencia. Resuelve en tiempo de ejecución una etiqueta estable concreta, la verifica con SHA256SUMS y la deja registrada en el binario instalado. La instalación reproducible sigue disponible con `PI_GOMEMORY_VERSION`. Se documenta en esta unidad, sin enmienda.

## Historias
### US1. Instalación desde cero completa y al día
Como desarrollador con una máquina nueva, quiero que una sola línea curl deje el meta-harness completo en su última versión, para empezar a trabajar sin instalar nada a mano.
- C-01: Dado un HOME sin pi, mem, codegraph, codebase-memory-mcp ni herdr, con Node.js ≥ 22.19 y npm, y sin terminal, cuando se ejecuta `install` con los instaladores sustituidos por dobles (`toolHooks`), entonces se ejecuta el canal oficial de los cinco en el orden del catálogo y doctor marca sus binarios como OK. Además, antes de publicar, `./install.sh` ejecutado desde el repositorio (opción A) con un HOME temporal vacío deja los cinco instalados desde sus canales reales. La prueba con `curl | bash` contra la release publicada es una comprobación posterior a la release, fuera de la verificación de esta unidad.
- C-02: Dado que no hay `PI_GOMEMORY_VERSION` y `PI_GOMEMORY_LATEST_URL` apunta a un servidor de prueba que redirige a `/releases/tag/vX.Y.Z`, cuando se ejecuta `install.sh`, entonces descarga `vX.Y.Z`. Dado `PI_GOMEMORY_VERSION=vA.B.C`, entonces descarga exactamente esa etiqueta. Dado que la resolución falla, entonces aborta con código distinto de 0 y menciona `PI_GOMEMORY_VERSION`. Dado que se ejecuta desde el código fuente (opción A) sin red, entonces compila sin consultar `PI_GOMEMORY_LATEST_URL`.
- C-03: Dado que falta pi y falta Node.js ≥ 22.19 o npm, cuando `sync` llega a pi, entonces termina con error antes de ejecutar el instalador de pi y muestra cómo instalar Node.js.
- C-14: Dado que herdr no está instalado porque se desmarcó o por modo lite, cuando se ejecuta doctor, entonces su binario aparece como WARN y no como FAIL.

### US2. Reconciliación sin preguntas
Como desarrollador con herramientas ya instaladas, quiero que install, update y sync las dejen en su última versión sin preguntarme cada una, para no tener un meta-harness a medias.
- C-04: Dado codebase-memory-mcp instalado en una versión anterior a la última y una caché de deps de menos de una hora que lo da por al día, cuando se ejecuta `pi-gomemory-setup sync --yes`, entonces consulta la red, lo actualiza sin ninguna pregunta y reescribe la caché con todos los componentes (`self`, `tool` y `package`), con codebase-memory-mcp en su versión nueva.
- C-05: Dado pi-antigravity instalado en una versión distinta del pin, cuando se ejecuta `install`, entonces se instala la versión fijada y no la publicada en npm, y `sync` no lo toca.
- C-06: Dado que falla la instalación de herdr (opcional) o la actualización de pi ya instalado, cuando se ejecuta sync, entonces avisa con el comando manual y continúa con código 0. Dado que falta pi o mem y su instalación falla, entonces termina con error y el comando manual.
- C-07: Dado `PI_GOMEMORY_SKIP_DEPS=1`, cuando se ejecuta install, entonces no se comprueba, instala ni actualiza ningún `tool`.
- C-15: Dado pi-gomemory al día, cuando se ejecuta `update`, entonces ejecuta `sync --yes` una vez. Dado una release nueva, entonces reemplaza el binario, relanza `install` y `sync` se ejecuta una sola vez y sin selector. Dado que falla la consulta de la release, entonces termina con error, no ejecuta `sync` y sugiere `pi-gomemory-setup sync`.
- C-17: Dado pi-gomemory y mem desactualizados, cuando se ejecuta `deps-update`, entonces solo se ejecuta `update` y mem se actualiza una sola vez (dentro de su `sync`). Dado solo mem desactualizado, entonces se ejecuta `sync --yes --only=mem`.

### US3. Catálogo único
Como mantenedor, quiero declarar cada componente en un solo sitio, para que añadir o cambiar una herramienta no exija tocar varias listas.
- C-08: Dado el catálogo, cuando se comparan las listas, entonces:
  - `productionTools()`, los binarios de doctor, `printToolsKept` y `components --names` contienen exactamente los `tool` del catálogo en su orden;
  - `DefaultDepProbes()` contiene exactamente todos los componentes del catálogo (`self`, `tool` y `package`) en su orden;
  - los textos del paso [1/9], del aviso de `PI_GOMEMORY_SKIP_DEPS` y del uso de `deps-install` nombran exactamente los `tool` del catálogo.

### US4. Selector en la terminal
Como desarrollador que instala desde una terminal, quiero marcar qué componentes opcionales instalar o actualizar, para controlar qué entra en mi máquina.
- C-09: Dado una terminal y ningún flag, cuando se ejecuta sync, entonces se muestra la lista con nombre, versión instalada, última versión y obligatorio u opcional. Los obligatorios no se pueden desmarcar, y desmarcar herdr hace que no se instale ni se actualice.
- C-13: Dado `curl | bash` desde una terminal, cuando install llega al sync, entonces muestra el selector leyendo de `/dev/tty` (la entrada estándar sigue en `/dev/null`) y pulsar Enter sin cambios reconcilia todos los `tool`. En modo lite los opcionales aparecen desmarcados.
- C-10: Dado que no hay terminal o se pasa `--yes`, cuando se ejecuta sync, entonces no se muestra el selector y se reconcilian todos los `tool` por defecto sin bloquearse.
- C-11: Dado `--only=codegraph`, cuando se ejecuta sync, entonces se reconcilian codegraph, pi y mem. Dado `--skip=herdr`, entonces se omite herdr. Dado `--skip=pi`, entonces pi se reconcilia igualmente. Combinar `--only` y `--skip`, usar un nombre desconocido o `--only=pi-gomemory` termina con código 2. En modo lite con `--yes`, herdr no se instala ni se actualiza.

### US5. Estado y actualización desde Pi
Como desarrollador dentro de Pi, quiero ver y actualizar los componentes con `/componentes`, para no salir a la terminal.
- C-12: Dado Pi con la extensión gomemory-status, cuando se ejecuta `/componentes`, entonces:
  - se listan los componentes de `pi-gomemory-setup components --json` con su versión instalada y la última;
  - al confirmar una selección de `tool` se ejecuta `pi-gomemory-setup sync --json --only=…` con un plazo de 60 min y se notifica el resultado, o el aviso de plazo vencido;
  - si pi quedó `updated` o `installed` en la salida, aunque no estuviera seleccionado, se avisa de reiniciar Pi.
- C-16: Dado `components --json`, cuando se ejecuta, entonces devuelve un JSON válido con `name`, `kind`, `required`, `installed`, `current`, `latest`, `outdated` y `error` para cada componente del catálogo, en su orden, y un componente ausente tiene `installed:false`, `current:""` y `outdated:false`.

## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->
- [x] Q1. ¿Herdr obligatorio u opcional? → opcional.
- [x] Q2. ¿install.sh con versión fijada o latest? → siempre latest.
- [x] Q3. ¿Herramienta presente y desactualizada en install? → se actualiza sin preguntar.
- [x] Q4. ¿Uno o varios HCD? → todo en un HCD.
- [x] Q5. ¿Sync en install/update respeta PI_GOMEMORY_NO_UPDATE_CHECK? → no: es una acción explícita (como update, HCD-035); para no tocar herramientas existe PI_GOMEMORY_SKIP_DEPS.
- [x] Q6. ¿El selector aparece en `curl | bash` con terminal? → sí, con todo marcado salvo los opcionales en modo lite; Enter reconcilia lo marcado, y `--yes` o la falta de terminal lo saltan.
- [x] Q7. ¿Correcciones de Splash H1–H15? → aplicar todas (sync en [1/9], catálogo con self/tool/package, evidencia de herdr, sync sin caché, fallo al actualizar es aviso, dobles y WARN en doctor, PI_GOMEMORY_LATEST_URL, components --json, `/componentes` con plazo de 15 min y aviso de reinicio, fuente de §0, lite desmarcado, obligatorios siempre incluidos, Node solo al instalar pi).

- [x] Q8. ¿Correcciones de Splash v2 (M1–M6, N1–N8)? → aplicar todas (sync --json con acción por tool, caché completa tras sync, update/deps-update sin dobles ni selector, evidencia de herdr, latest solo en opción B, validación real con ./install.sh antes de publicar, nombres no-tool con código 2, lite sin opcionales, plazo de 60 min, textos derivados, esquema de components --json, update con consulta fallida, cita literal de §0).

## Evidencia de los canales
Observada el 2026-11-24 en macOS arm64:
- `curl -fsSLI https://herdr.dev/install.sh` → `HTTP/2 200`, `content-type: text/plain`.
- El script empieza por `#!/bin/sh` y `set -eu`; línea 6: `INSTALL_DIR="${HERDR_INSTALL_DIR:-$HOME/.local/bin}"`; verifica SHA-256 contra el manifiesto de la release («downloaded Herdr checksum did not match») y aborta si no coincide.
- `curl -sI https://github.com/herdrdev/herdr/releases/latest` → `location: https://github.com/herdrdev/herdr/releases/tag/v0.9.5`.
- `herdr --version` → `herdr 0.9.5`.
- `herdr update --help` → «Download and install the latest version», única opción `--handoff`.
- `herdr --help` no tiene subcomando de desinstalación.

## Slices
<!-- ### VS-01. comportamiento entregable
Tareas: 1, 2
Criterios: C-01 -->
### VS-01. Un catálogo único alimenta install, deps, doctor, uninstall y components
Tareas: 1, 2, 3, 4, 5, 6, 7
Criterios: C-08, C-14, C-16

### VS-02. `sync` instala lo que falta y actualiza lo viejo sin preguntar, y install lo usa en [1/9]
Tareas: 8, 9, 10, 11, 12, 13
Criterios: C-03, C-04, C-05, C-06, C-07, C-11, C-01

### VS-03. `update` y `deps-update` reconcilian una sola vez
Tareas: 14, 15
Criterios: C-15, C-17

### VS-04. Selector de casillas en la terminal
Tareas: 16, 17
Criterios: C-09, C-10, C-13

### VS-05. `install.sh` instala la última release
Tareas: 18, 19
Criterios: C-02

### VS-06. `/componentes` en Pi
Tareas: 20
Criterios: C-12

### VS-07. Validación real y regresión
Tareas: 21, 22
Criterios: C-01

## Checklist
- [x] 1. Crear el catálogo `Component` (self/tool/package, herdr opcional con su canal) en `pkg/installer/components.go` → `go test ./pkg/installer -run '^TestHCD076CatalogOrder$'`
- [x] 2. Derivar `productionTools()` y los textos de [1/9], `PI_GOMEMORY_SKIP_DEPS` y `deps-install` del catálogo → `go test ./pkg/installer -run '^TestHCD076ToolsFromCatalog$'`
- [x] 3. Derivar `DefaultDepProbes()` del catálogo con la sonda de herdr → `go test ./pkg/installer -run '^TestHCD076ProbesFromCatalog$'`
- [x] 4. Derivar los binarios de `RunDoctor` del catálogo con WARN para un opcional ausente → `go test ./pkg/installer -run '^TestHCD076DoctorOptionalWarn$'`
- [x] 5. Derivar `printToolsKept` del catálogo con la desinstalación de herdr → `go test ./pkg/installer -run '^TestHCD076UninstallKeptFromCatalog$'`
- [x] 6. Añadir `pi-gomemory-setup components --names` → `go test . -run '^TestHCD076ComponentsNames$'`
- [x] 7. Añadir `pi-gomemory-setup components --json` con el esquema de la unidad → `go test ./pkg/installer -run '^TestHCD076ComponentsJSON$'`
- [x] 8. Implementar la reconciliación de `sync` (instala faltantes, actualiza desactualizados ignorando la caché y la reescribe completa) → `go test ./pkg/installer -run '^TestHCD076SyncUpdatesOutdated$'`
- [x] 9. Aplicar la política de fallos de `sync` (fatal solo al instalar un obligatorio que falta) → `go test ./pkg/installer -run '^TestHCD076SyncFailurePolicy$'`
- [x] 10. Comprobar Node.js ≥ 22.19 y npm antes de instalar pi cuando falta → `go test ./pkg/installer -run '^TestHCD076SyncNodeCheck$'`
- [x] 11. Implementar `--yes`, `--only`, `--skip` y el modo lite en `sync` → `go test ./pkg/installer -run '^TestHCD076SyncFlags$'`
- [x] 12. Añadir el subcomando `sync` con salida `--json` por herramienta → `go test ./pkg/installer -run '^TestHCD076SyncCommandJSON$'`
- [x] 13. Sustituir `EnsureTools` por `sync` en el paso [1/9] de `InstallAll` conservando pins y `PI_GOMEMORY_SKIP_DEPS` → `go test ./pkg/installer -run '^TestHCD076InstallAllUsesSync$'`
- [x] 14. Hacer que `update` ejecute `sync --yes` una sola vez (al día, release nueva, consulta fallida) → `go test . -run '^TestHCD076UpdateRunsSync$'`
- [x] 15. Hacer que `deps-update` delegue en `update` o en `sync --yes --only` sin duplicar → `go test ./pkg/installer -run '^TestHCD076DepsUpdateDelegates$'`
- [x] 16. Implementar el selector de casillas sobre `/dev/tty` (obligatorios bloqueados, lite desmarcado) → `go test ./pkg/installer -run '^TestHCD076Selector$'`
- [x] 17. Omitir el selector sin terminal o con flags y mostrarlo en `install` con terminal → `go test ./pkg/installer -run '^TestHCD076SelectorGating$'`
- [x] 18. Resolver la última etiqueta en la opción B de `install.sh` con `PI_GOMEMORY_LATEST_URL` → `go test . -run '^TestHCD076InstallShLatest$'`
- [x] 19. Usar `components --names` en la lista de PATH de `install.sh` → `go test . -run '^TestHCD076InstallShPathFromCatalog$'`
- [x] 20. Añadir el comando `/componentes` a `extensions/gomemory-status.ts` → `go test ./pkg/installer -run '^TestHCD076ComponentesCommand$'`
- [x] 21. Ejecutar `./install.sh` desde el repositorio con un HOME temporal vacío → observación: `doctor` del HOME temporal marca OK pi, mem, codegraph, codebase-memory-mcp y herdr
- [x] 22. Ejecutar la regresión completa → `go vet ./... && go test -race ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-24 05:13 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076CatalogOrder$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_catalog_test.go:11:28: undefined: ComponentKind
      pkg/installer/hcd076_catalog_test.go:13:20: undefined: Catalog
      pkg/installer/hcd076_catalog_test.go:29:12: undefined: ComponentKind
      pkg/installer/hcd076_catalog_test.go:32:19: undefined: KindSelf
      pkg/installer/hcd076_catalog_test.go:33:10: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:34:11: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:35:17: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:36:27: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:37:13: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:38:22: undefined: KindPackage
      pkg/installer/hcd076_catalog_test.go:38:22: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:13 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076ToolsFromCatalog$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_catalog_test.go:11:28: undefined: ComponentKind
      pkg/installer/hcd076_catalog_test.go:13:20: undefined: Catalog
      pkg/installer/hcd076_catalog_test.go:29:12: undefined: ComponentKind
      pkg/installer/hcd076_catalog_test.go:32:19: undefined: KindSelf
      pkg/installer/hcd076_catalog_test.go:33:10: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:34:11: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:35:17: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:36:27: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:37:13: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:38:22: undefined: KindPackage
      pkg/installer/hcd076_catalog_test.go:38:22: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:13 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076ProbesFromCatalog$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_catalog_test.go:11:28: undefined: ComponentKind
      pkg/installer/hcd076_catalog_test.go:13:20: undefined: Catalog
      pkg/installer/hcd076_catalog_test.go:29:12: undefined: ComponentKind
      pkg/installer/hcd076_catalog_test.go:32:19: undefined: KindSelf
      pkg/installer/hcd076_catalog_test.go:33:10: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:34:11: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:35:17: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:36:27: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:37:13: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:38:22: undefined: KindPackage
      pkg/installer/hcd076_catalog_test.go:38:22: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:13 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076DoctorOptionalWarn$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_catalog_test.go:11:28: undefined: ComponentKind
      pkg/installer/hcd076_catalog_test.go:13:20: undefined: Catalog
      pkg/installer/hcd076_catalog_test.go:29:12: undefined: ComponentKind
      pkg/installer/hcd076_catalog_test.go:32:19: undefined: KindSelf
      pkg/installer/hcd076_catalog_test.go:33:10: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:34:11: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:35:17: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:36:27: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:37:13: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:38:22: undefined: KindPackage
      pkg/installer/hcd076_catalog_test.go:38:22: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:13 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076UninstallKeptFromCatalog$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_catalog_test.go:11:28: undefined: ComponentKind
      pkg/installer/hcd076_catalog_test.go:13:20: undefined: Catalog
      pkg/installer/hcd076_catalog_test.go:29:12: undefined: ComponentKind
      pkg/installer/hcd076_catalog_test.go:32:19: undefined: KindSelf
      pkg/installer/hcd076_catalog_test.go:33:10: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:34:11: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:35:17: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:36:27: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:37:13: undefined: KindTool
      pkg/installer/hcd076_catalog_test.go:38:22: undefined: KindPackage
      pkg/installer/hcd076_catalog_test.go:38:22: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:14 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076CatalogOrder$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.490s
- 2026-11-24 05:14 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076ToolsFromCatalog$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.168s
- 2026-11-24 05:14 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076ProbesFromCatalog$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.177s
- 2026-11-24 05:14 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076DoctorOptionalWarn$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.178s
- 2026-11-24 05:14 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076UninstallKeptFromCatalog$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.181s
- 2026-11-24 05:15 -05:00 · tarea 6 · rojo observado · caducada · `go test . -run '^TestHCD076ComponentsNames$'` (salida 1)
      # pi-gomemory [pi-gomemory.test]
      ./main_hcd076_test.go:16:13: undefined: runComponents
      ./main_hcd076_test.go:23:13: undefined: runComponents
      FAIL	pi-gomemory [build failed]
      FAIL
- 2026-11-24 05:15 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076ComponentsJSON$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_components_test.go:58:12: undefined: WriteComponentsJSON
      pkg/installer/hcd076_components_test.go:61:12: undefined: ComponentStatus
      pkg/installer/hcd076_components_test.go:66:23: undefined: ComponentStatus
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:16 -05:00 · tarea 6 · verde · caducada · `go test . -run '^TestHCD076ComponentsNames$'` (salida 0)
      ok  	pi-gomemory	0.616s
- 2026-11-24 05:16 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076ComponentsJSON$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.221s
- 2026-11-24 05:17 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076SyncUpdatesOutdated$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_sync_test.go:20:9: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:21:2: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:22:21: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:28:9: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:29:2: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:30:21: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:55:20: undefined: SyncResult
      pkg/installer/hcd076_sync_test.go:89:14: undefined: Sync
      pkg/installer/hcd076_sync_test.go:89:25: undefined: SyncOptions
      pkg/installer/hcd076_sync_test.go:128:14: undefined: Sync
      pkg/installer/hcd076_sync_test.go:128:14: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:17 -05:00 · tarea 11 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076SyncFlags$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_sync_test.go:20:9: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:21:2: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:22:21: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:28:9: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:29:2: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:30:21: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:55:20: undefined: SyncResult
      pkg/installer/hcd076_sync_test.go:89:14: undefined: Sync
      pkg/installer/hcd076_sync_test.go:89:25: undefined: SyncOptions
      pkg/installer/hcd076_sync_test.go:128:14: undefined: Sync
      pkg/installer/hcd076_sync_test.go:128:14: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:17 -05:00 · tarea 10 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076SyncNodeCheck$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_sync_test.go:20:9: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:21:2: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:22:21: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:28:9: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:29:2: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:30:21: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:55:20: undefined: SyncResult
      pkg/installer/hcd076_sync_test.go:89:14: undefined: Sync
      pkg/installer/hcd076_sync_test.go:89:25: undefined: SyncOptions
      pkg/installer/hcd076_sync_test.go:128:14: undefined: Sync
      pkg/installer/hcd076_sync_test.go:128:14: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:17 -05:00 · tarea 9 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076SyncFailurePolicy$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_sync_test.go:20:9: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:21:2: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:22:21: undefined: syncProbes
      pkg/installer/hcd076_sync_test.go:28:9: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:29:2: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:30:21: undefined: nodeRuntime
      pkg/installer/hcd076_sync_test.go:55:20: undefined: SyncResult
      pkg/installer/hcd076_sync_test.go:89:14: undefined: Sync
      pkg/installer/hcd076_sync_test.go:89:25: undefined: SyncOptions
      pkg/installer/hcd076_sync_test.go:128:14: undefined: Sync
      pkg/installer/hcd076_sync_test.go:128:14: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:18 -05:00 · tarea 11 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076SyncFlags$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.174s
- 2026-11-24 05:18 -05:00 · tarea 10 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076SyncNodeCheck$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.290s
- 2026-11-24 05:18 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076SyncUpdatesOutdated$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.318s
- 2026-11-24 05:18 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076SyncFailurePolicy$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.321s
- 2026-11-24 05:19 -05:00 · tarea 12 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076SyncCommandJSON$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_sync_test.go:191:13: undefined: RunSync
      pkg/installer/hcd076_sync_test.go:204:13: undefined: RunSync
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:19 -05:00 · tarea 13 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076InstallAllUsesSync$'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd076_sync_test.go:191:13: undefined: RunSync
      pkg/installer/hcd076_sync_test.go:204:13: undefined: RunSync
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:19 -05:00 · tarea 12 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076SyncCommandJSON$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.393s
- 2026-11-24 05:19 -05:00 · tarea 13 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076InstallAllUsesSync$'` (salida 1)
      MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Modo codemode                    [38;2;145;169;197mhíbrido (codemode on)[0m
        [38;2;34;227;247m✓[0m Tools de codebase-memory         [38;2;145;169;197msin extensiones que dupliquen el servidor MCP[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Thinking colapsado (Ctrl+T)      [38;2;145;169;197mhideThinkingBlock activo en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Skills de gomemory al día        [38;2;145;169;197mmem no instaló skills en ningún agente; Pi usa las embebidas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD076InstallAllUsesSync2981883071/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- FAIL: TestHCD076InstallAllUsesSync (0.39s)
          hcd076_sync_test.go:231: install debe actualizar codegraph sin preguntar: "1.0.0\n"
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.655s
      FAIL
- 2026-11-24 05:19 -05:00 · tarea 13 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076InstallAllUsesSync$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.023s
- 2026-11-24 05:20 -05:00 · tarea 14 · rojo observado · caducada · `go test . -run '^TestHCD076UpdateRunsSync$'` (salida 1)
      # pi-gomemory [pi-gomemory.test]
      ./main_hcd035_test.go:103:3: unknown field sync in struct literal of type updateDeps
      FAIL	pi-gomemory [build failed]
      FAIL
- 2026-11-24 05:20 -05:00 · tarea 14 · verde · caducada · `go test . -run '^TestHCD076UpdateRunsSync$'` (salida 0)
      ok  	pi-gomemory	0.197s
- 2026-11-24 05:21 -05:00 · tarea 15 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076DepsUpdateDelegates$'` (salida 1)
      --- FAIL: TestHCD076DepsUpdateDelegates (0.19s)
          hcd076_sync_test.go:286: código 1:
              ==> Actualizando pi-gomemory (v1.0.0 → v1.1.0)...
              ==> Actualizando mem (1.0.0 → 1.1.0)...
              ==> Actualizando pi-cc-extensions (0.0.1 → 1.0.21)...
              [38;2;255;119;127m✗ mem: exec: "mem": executable file not found in $PATH[0m
              ✗ No se pudieron actualizar: [mem]
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.393s
      FAIL
- 2026-11-24 05:21 -05:00 · tarea 15 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076DepsUpdateDelegates$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.414s
- 2026-11-24 05:22 -05:00 · tarea 16 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076Selector$'` (salida 1)
      --- FAIL: TestHCD076Selector (0.00s)
          hcd076_selector_test.go:37: selección = map[codegraph:true herdr:true pi:true]
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.188s
      FAIL
- 2026-11-24 05:22 -05:00 · tarea 17 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076SelectorGating$'` (salida 1)
      --- FAIL: TestHCD076SelectorGating (0.11s)
          hcd076_selector_test.go:73: herdr desmarcado no debe instalarse:
              [38;2;145;169;197m    ✓ pi al día (1.0.0)[0m
              [38;2;145;169;197m    → instalando herdr desde su canal oficial…[0m
              [38;2;244;198;107m    ⚠ no se pudo instalar herdr (descargar https://invalid.example/h.sh: Get "https://invalid.example/h.sh": red bloqueada en pruebas (HCD-062)). Instálalo a mano con: m[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.292s
      FAIL
- 2026-11-24 05:22 -05:00 · tarea 17 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076SelectorGating$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.871s
- 2026-11-24 05:22 -05:00 · tarea 16 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076Selector$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.193s
- 2026-11-24 05:23 -05:00 · tarea 18 · rojo observado · caducada · `go test . -run '^TestHCD076InstallShLatest$'` (salida 1)
      --- FAIL: TestHCD076InstallShLatest (0.03s)
          main_hcd076_test.go:122: sin versión debe descargar la última (v9.8.7): código 1
              [38;2;6;193;238m╔════════════════════════════════════════════════════════════════════╗[0m
              [38;2;34;227;247m║         INSTALADOR DE PI-GOMEMORY PARA PI CODING AGENT             ║[0m
              [38;2;6;193;238m╚════════════════════════════════════════════════════════════════════╝[0m
              [38;2;145;169;197m==> Descargando pi-gomemory-setup_v1.25.1_darwin_arm64.tar.gz desde http://127.0.0.1:59344...[0m
              curl: (22) The requested URL returned error: 404
              [38;2;255;119;127m✗ No se pudo descargar el paquete o SHA256SUMS desde http://127.0.0.1:59344[0m
      FAIL
      FAIL	pi-gomemory	0.352s
      FAIL
- 2026-11-24 05:23 -05:00 · tarea 19 · rojo observado · caducada · `go test . -run '^TestHCD076InstallShPathFromCatalog$'` (salida 1)
      --- FAIL: TestHCD076InstallShPathFromCatalog (0.00s)
          main_hcd076_test.go:163: install.sh debe obtener los tool de `components --names`, no de una lista fija
      FAIL
      FAIL	pi-gomemory	0.312s
      FAIL
- 2026-11-24 05:24 -05:00 · tarea 18 · verde · caducada · `go test . -run '^TestHCD076InstallShLatest$'` (salida 0)
      ok  	pi-gomemory	0.733s
- 2026-11-24 05:24 -05:00 · tarea 19 · verde · caducada · `go test . -run '^TestHCD076InstallShPathFromCatalog$'` (salida 0)
      ok  	pi-gomemory	0.346s
- 2026-11-24 05:25 -05:00 · tarea 20 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD076ComponentesCommand$'` (salida 1)
      --- FAIL: TestHCD076ComponentesCommand (0.13s)
          hcd076_componentes_test.go:15: node falló: exit status 1
              file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD076ComponentesCommand418392465/001/test.mjs:39
              await commands.componentes.handler("", ctx);
                                         ^
              TypeError: Cannot read properties of undefined (reading 'handler')
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD076ComponentesCommand418392465/001/test.mjs:39:28
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5)
              Node.js v26.13.1
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.336s
      FAIL
- 2026-11-24 05:25 -05:00 · tarea 20 · verde · caducada · `go test ./pkg/installer -run '^TestHCD076ComponentesCommand$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.270s
- 2026-11-24 05:25 -05:00 · tarea 22 · falló · caducada · `go vet ./... && go test -race ./...` (salida 1)
      Goroutine 127 (running) created at:
        testing.(*T).Run()
            /opt/homebrew/Cellar/go/1.27.4/libexec/src/testing/testing.go:2260 +0x7bc
        testing.runTests.func1()
            /opt/homebrew/Cellar/go/1.27.4/libexec/src/testing/testing.go:2744 +0x74
        testing.tRunner()
            /opt/homebrew/Cellar/go/1.27.4/libexec/src/testing/testing.go:2195 +0x164
        testing.runTests()
            /opt/homebrew/Cellar/go/1.27.4/libexec/src/testing/testing.go:2742 +0x7a0
        testing.(*M).Run()
            /opt/homebrew/Cellar/go/1.27.4/libexec/src/testing/testing.go:2602 +0xb38
        main.main()
            _testmain.go:84 +0x100
      ==================
      --- FAIL: TestHCD076InstallShLatest (0.48s)
          testing.go:1867: race detected during execution of test
      FAIL
      FAIL	pi-gomemory	1.386s
      ok  	pi-gomemory/pkg/installer	51.369s
      FAIL
- 2026-11-24 05:27 -05:00 · tarea 22 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	2.340s
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-24 05:27 -05:00 · tarea 21 · verde · vigente · `bash scripts/clean-install-check.sh` (salida 0)
      ✓ pi --version: 1.6.0
      ✓ mem version: gomemory 2.56.0
      ✓ codegraph --version: 1.6.4
      ✓ codebase-memory-mcp --version: codebase-memory-mcp 0.12.1
      ✓ herdr --version: herdr 0.9.5
      ✓ Instalación limpia completa (log: /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/tmp.mimQgFhVml/install.log se borra al salir).
- 2026-11-24 05:28 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD076ProbesFromCatalog$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.177s
- 2026-11-24 05:28 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD076DoctorOptionalWarn$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.169s
- 2026-11-24 05:28 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD076UninstallKeptFromCatalog$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.164s
- 2026-11-24 05:28 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run '^TestHCD076SyncUpdatesOutdated$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.326s
- 2026-11-24 05:28 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD076ToolsFromCatalog$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.175s
- 2026-11-24 05:28 -05:00 · tarea 10 · verde · vigente · `go test ./pkg/installer -run '^TestHCD076SyncNodeCheck$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.347s
- 2026-11-24 05:28 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD076CatalogOrder$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.159s
- 2026-11-24 05:28 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run '^TestHCD076ComponentsJSON$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.187s
- 2026-11-24 05:28 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run '^TestHCD076SyncFailurePolicy$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.338s
- 2026-11-24 05:28 -05:00 · tarea 6 · verde · vigente · `go test . -run '^TestHCD076ComponentsNames$'` (salida 0)
      ok  	pi-gomemory	0.187s
- 2026-11-24 05:28 -05:00 · tarea 13 · verde · vigente · `go test ./pkg/installer -run '^TestHCD076InstallAllUsesSync$'` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.337s

## Siguiente
Cerrada con aprobación sobre la revisión e9d21338e871.

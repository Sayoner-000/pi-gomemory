# Instalar pi, mem, codegraph y codebase-memory desde sus canales oficiales

ID: HCD-062
Tipo: feature
Carril: full
Motivo: Toca la cadena de instalación, ejecuta instaladores remotos y descarga binarios: riesgo de seguridad y decisiones de canal interdependientes

## Intención
Que curl -fsSL …/pi-gomemory/main/install.sh | bash deje pi, mem, codegraph y codebase-memory-mcp instalados desde sus canales oficiales, sin que el desarrollador abra páginas


## Alcance
Incluido: un paso nuevo al inicio de `InstallAll` (`pkg/installer/installer.go`) que comprueba `pi`, `mem`, `codegraph` y `codebase-memory-mcp` y, si falta alguno, lo instala desde su canal oficial verificado el 2026-11-20:
- `pi`: `https://pi.dev/install.sh` (instalador gestionado de Earendil; requiere Node.js 22.19+ y lo ofrece él mismo).
- `mem`: `https://raw.githubusercontent.com/Sayoner-000/gomemory/main/scripts/install.sh` con `GOMEMORY_BIN_DIR` = directorio de binarios de pi-gomemory (verifica SHA-256 por su cuenta).
- `codegraph`: `https://raw.githubusercontent.com/colbymchenry/codegraph/main/install.sh` (instalador oficial del proyecto).
- `codebase-memory-mcp`: el `.tar.gz` del último release de `DeusData/codebase-memory-mcp` para el SO y la arquitectura (Linux usa la variante `-portable`), verificado contra el `checksums.txt` del mismo release; en macOS se retira la cuarentena y se firma ad hoc como hace su instalador; se instala con el subcomando oficial del binario `install -y --force --dir=<bin> --skip-config`, sin tocar la configuración de otros agentes.
**Ejecución de los instaladores (B-1, M-7).** Cada script se descarga por HTTPS a un temporal (límite 2 min) y se ejecuta con un intérprete fijo por herramienta (`pi`: `sh`; `mem`: `bash`; `codegraph`: `sh`) con la entrada estándar en `/dev/null` (nunca hereda la del `curl | bash` de pi-gomemory) y la salida estándar y de errores conectadas a una tubería que pi-gomemory copia a la suya (un `io.Writer` que no es `*os.File`, de modo que para el hijo nunca es una terminal y no ofrece arrancar `pi` ni configurar el directorio actual). Las preguntas que sí necesitan hacer (por ejemplo, el instalador de pi ofrece instalar Node.js) las leen ellos mismos de `/dev/tty`; sin terminal ese instalador termina con error y el mensaje pide instalar Node.js 22.19+ y repetir. Cada instalador tiene un límite de 10 minutos; superarlo cuenta como fallo. Grupo de procesos (B-1 de la tercera revisión): si `/dev/tty` se puede abrir (sesión interactiva), el hijo se queda en el grupo de procesos de pi-gomemory, que es el de primer plano de la terminal, para que pueda leer `/dev/tty` sin recibir `SIGTTIN` y para que Ctrl‑C llegue también a él; al vencer el plazo se mata el hijo directo y `WaitDelay` (5 s) cierra la tubería aunque un nieto la retenga, de modo que el paso nunca se queda colgado. Sin terminal no hay preguntas posibles: el hijo corre en su propio grupo (`Setpgid`) y al vencer el plazo se mata el grupo entero, incluidos los nietos. Descargas: solo HTTPS (se rechaza cualquier redirección a otro esquema), se exige HTTP 200, límite de 1 MiB por script y de 200 MiB por archivo de release; todo va a un directorio temporal privado creado junto al directorio de binarios de pi-gomemory (`os.MkdirTemp(<bin>, ".pi-gomemory-deps-*")`, 0700, como el `STAGE` de `install.sh`, para no depender de un `/tmp` montado con `noexec`) que se borra siempre, y el hijo se ejecuta con ese temporal como directorio de trabajo, de modo que `InstallAll` sigue sin escribir en el directorio actual. Además, `install.sh` lanza `pi-gomemory-setup install` con `</dev/null` y `update` deja de pasar su stdin al `install` hijo.

**PATH (M-1, N-1, N-2).** Directorios de binarios conocidos, en este orden: el directorio de binarios de pi-gomemory (`PI_GOMEMORY_BIN_DIR`, por defecto `~/.local/bin`), `~/.local/bin`, `~/.pi/agent/bin`, `~/.gomemory/bin` y `/usr/local/bin`. Al empezar el paso, y antes de ejecutar ningún instalador, se anteponen al `PATH` del proceso los que aún no estén, sin duplicar; así `pi.dev/install.sh` enlaza `pi` en uno de los directorios conocidos (con el valor por defecto, `~/.local/bin`) y los pasos siguientes (`pi install …`) lo encuentran. Node.js: si el instalador de pi lo instala por su cuenta (sin Homebrew ni apt) lo deja en `~/.local/share/pi-node/current/bin` sin tocar el `PATH`, y el lanzador gestionado de `pi` (`~/.pi/agent/bin/pi`) añade ese directorio antes de ejecutar `pi` y sus hijos (`npm` de `pi install`), así que no hace falta añadirlo a la lista. Una herramienta cuenta como presente si `exec.LookPath` la encuentra con ese `PATH`. El launcher `pi-gomemory` añade `~/.pi/agent/bin` a su `PATH` (hoy solo añade `~/.local/bin` y `~/.gomemory/bin`). `doctor`, tanto al final de `install` como lanzado a mano, amplía el `PATH` con la misma lista antes de buscar. El aviso final de `install.sh` nombra los directorios conocidos que el shell del usuario no tiene, y se muestra también cuando `install` falla (N-5): `install.sh` guarda el código de salida, imprime la nota y termina con ese código.

**Obligatorias y opcionales (C-06, N-3, N-4).** `pi` y `mem` son obligatorios: si fallan, la instalación se aborta con un error que nombra la herramienta y su comando manual. `codegraph` y `codebase-memory-mcp` son opcionales: si fallan, se avisa con el comando manual y se sigue; con el modo lite activo según `IsLiteMode(opts.HomeDir)` (reinstalación o `update`; en una máquina limpia nunca lo está) no se comprueban. `PI_GOMEMORY_SKIP_DEPS` sigue la regla de `PI_GOMEMORY_NO_UPDATE_CHECK`: vacía, `0` o `false` (sin distinguir mayúsculas ni espacios) dejan el paso activo; cualquier otro valor lo omite.

**Actualizaciones (M-2, M-5).** Como `update` vuelve a ejecutar `install`, el paso también corre en cada actualización e instala lo que falte. Es deliberado: `mcp.json` registra `codegraph` y `codebase-memory` y deben existir; quien no los quiera usa el modo lite o `PI_GOMEMORY_SKIP_DEPS`. Nunca se reinstala una herramienta presente. `codebase-memory-mcp` deja de ser manual: su `Update` pasa a ser `pi-gomemory-setup deps-install codebase-memory-mcp` (subcomando nuevo que ejecuta el mismo canal verificado y fuerza la instalación del último release); ese es el comando que muestra el aviso de dependencias, y el test de HCD-012 que exigía `Manual` se actualiza. `deps-install` admite solo `codebase-memory-mcp` (las demás se actualizan con su propio comando oficial); cualquier otro nombre o la falta de argumento responde con el uso y código 2. El destino es siempre el directorio de binarios de pi-gomemory, que el launcher antepone a su `PATH` (así `deps-status`, que corre desde el launcher, ve la versión nueva y el aviso desaparece). Si `exec.LookPath` con el `PATH` original del usuario resuelve otro `codebase-memory-mcp` (por ejemplo de Homebrew), no se toca y se avisa de que esa copia tiene precedencia fuera de pi-gomemory. `UpdateDeps` resuelve `pi-gomemory-setup` con `os.Executable()` en vez de buscarlo en el `PATH`.

**Pruebas (M-4).** El paso usa un punto de sustitución del paquete (búsqueda de binarios, descarga y ejecución) al estilo de `updateDeps` en `main.go`. `TestMain` lo sustituye por uno que declara todas las herramientas presentes, de modo que ningún test existente de `InstallAll` sale a la red; las pruebas nuevas usan un servidor HTTP local y scripts falsos. Los tests que compilan y ejecutan el binario como proceso aparte (`hcd047_binary_test.go` y cualquier otro que lance `install` o `update`) fijan `PI_GOMEMORY_SKIP_DEPS=1` en el entorno del hijo, la misma variable que puede usar una persona. No se añade ninguna variable de entorno solo para pruebas.

**Efectos fuera de pi-gomemory (M-4 de la segunda revisión).** Los instaladores oficiales pueden añadir una línea `PATH` al archivo de arranque del shell (`codebase-memory-mcp` lo hace en el rc del shell de la persona; el de pi puede hacerlo también) y crean sus propios directorios (`~/.codegraph`, `~/.pi/agent/install`). El paso lo anuncia en su salida. `uninstall` conserva explícitamente `pi`, `mem`, `codegraph` y `codebase-memory-mcp`, porque son herramientas independientes que otros agentes pueden usar, y su resumen dice cómo desinstalar cada una con su propio comando.

**Publicación (M-1 de la segunda revisión).** La vía `curl … | bash` usa el `VERSION` fijado en `install.sh`; el comportamiento llega a esa vía cuando se publique una release que lo incluya y se suba `VERSION`, lo que se hace en el paso de release habitual, fuera de esta unidad. C-11 verifica el mismo `install.sh` compilando el binario en local. También el cambio de `update` (no pasar su entrada estándar al `install` hijo) se nota a partir de la versión siguiente, porque `update` lo ejecuta el binario ya instalado; mientras tanto los instaladores reciben `/dev/null` igualmente.

`install.sh` documenta el paso; README y `doctor` muestran el canal de cada herramienta.

### Evidencia de los canales (M-6), observada el 2026-11-20
- `pi.dev/install.sh` (HTTP 200): instalador gestionado; instala en `~/.pi/agent/install` y enlaza `pi` en el primer directorio escribible del `PATH` entre `~/.pi/agent/bin`, `~/.local/bin`, `~/bin`; si ninguno está en el `PATH`, usa `~/.pi/agent/bin`. Pide Node.js 22.19+ por `/dev/tty`. Solo ofrece arrancar `pi` si la salida es una terminal (`[ -t 1 ]`).
- gomemory `scripts/install.sh` (HTTP 200): admite `GOMEMORY_BIN_DIR`; verifica SHA-256 contra `checksums.txt`; solo ofrece `mem install .` con salida en terminal.
- codegraph `install.sh` (HTTP 200): instala en `~/.codegraph` y enlaza en `CODEGRAPH_BIN_DIR` (por defecto `~/.local/bin`).
- codebase-memory-mcp v0.12.1: assets `codebase-memory-mcp-<darwin|linux>-<amd64|arm64>.tar.gz`, con sufijo `-portable` en Linux (`uname -m` `x86_64`→`amd64`, `aarch64`/`arm64`→`arm64`); `checksums.txt` con líneas `<sha256>  <archivo>`; el `.tar.gz` contiene exactamente `codebase-memory-mcp`, `LICENSE`, `install.sh` y `THIRD_PARTY_NOTICES.md`. `install -y --force --dir=<bin> --skip-config` probado en un HOME temporal: escribe solo el binario, una línea `export PATH=…` en `~/.zshrc` y logs en `~/.cache/codebase-memory-mcp`; no toca la configuración de ningún agente.

### Orden seguro para codebase-memory-mcp (N-1)
Resolver una sola vez la etiqueta del último release con `fetchLatestTag` sobre la redirección de `https://github.com/DeusData/codebase-memory-mcp/releases/latest` (sin `api.github.com`, por el cupo de 60 consultas por hora; `TestNoGitHubAPI` se amplía al archivo nuevo) y construir con ella las dos URL (N-4) → descargar `.tar.gz` y `checksums.txt` al temporal privado → verificar SHA-256 (si no coincide o falta, se aborta sin extraer) → extraer solo la entrada regular `codebase-memory-mcp`, rechazando rutas absolutas, `..` o enlaces → en macOS retirar `com.apple.quarantine` y firmar ad hoc; si `xattr` o `codesign` fallan se ignora, como hace su instalador oficial (N-3) → `--version` (si falla, es fallo de la herramienta opcional) → `install -y --force --dir=<bin> --skip-config` → borrar el temporal.

### Riesgo aceptado (M-3)
Los scripts de pi, gomemory y codegraph se toman de su canal oficial sin fijar versión ni suma, igual que hoy hacen sus actualizaciones (`pi update --self`, `mem update`, `codegraph upgrade`); el transporte es HTTPS y la integridad del contenido depende de cada proyecto. `checksums.txt` de codebase-memory-mcp detecta corrupción, no un release comprometido. La aprobación de esta definición acepta explícitamente ese riesgo.

Excluido: instalar Node.js por nuestra cuenta (lo ofrece el instalador de pi); Windows; fijar versiones o sumas de estas herramientas; configurar MCP de otros agentes; reinstalar o actualizar una herramienta presente desde `install` (eso sigue siendo `deps-update`); instalar Herdr.

## ¿Qué pasa si no lo hacemos?
Tras `curl … | bash`, la instalación asume que `pi` existe y el desarrollador termina con un harness a medias: sin `pi`, sin `mem` y con los MCP de `codegraph` y `codebase-memory` registrados en `mcp.json` pero apuntando a binarios inexistentes. Debe abrir cuatro páginas y repetir la instalación.

## Alternativa más barata
Imprimir al final los cuatro comandos de instalación. Es más barato pero no cumple el pedido («sin tener que ir a abrir páginas») y deja `mcp.json` apuntando a binarios ausentes hasta que la persona actúe.

## Historias
### US1. Instalación completa en una máquina limpia
Como desarrollador que ejecuta el instalador de una línea, quiero que se instalen pi, mem, codegraph y codebase-memory-mcp desde sus canales oficiales, para poder usar pi-gomemory sin abrir páginas ni repetir pasos.

- C-01: Dada una máquina sin `pi`, cuando se ejecuta `pi-gomemory-setup install`, entonces se descarga y ejecuta `https://pi.dev/install.sh` con la entrada estándar en `/dev/null` y la salida canalizada, antes de configurar plugins, y los pasos siguientes encuentran `pi` aunque el shell aún no tenga el directorio en su `PATH`; el launcher generado incluye `~/.pi/agent/bin` en su `PATH`.
- C-02: Dada una máquina sin `mem`, cuando se instala, entonces se ejecuta el instalador oficial de gomemory con `GOMEMORY_BIN_DIR` igual al directorio de binarios de pi-gomemory.
- C-03: Dada una máquina sin `codegraph`, cuando se instala sin modo lite, entonces se ejecuta el instalador oficial de codegraph.
- C-04: Dada una máquina sin `codebase-memory-mcp`, cuando se instala sin modo lite, entonces se descarga el `.tar.gz` del último release para el SO y la arquitectura, se verifica su SHA-256 contra `checksums.txt` y se instala con `install -y --force --dir=<bin> --skip-config`; si el SHA-256 no coincide o falta en `checksums.txt`, no se ejecuta nada del archivo.

### US2. Instalación segura e idempotente
Como desarrollador que reinstala o trabaja en un entorno restringido, quiero que el paso no repita lo que ya tengo y que falle de forma clara, para no perder tiempo ni quedar con un harness roto sin saberlo.

- C-05: Dada una herramienta ya presente en el `PATH` (o en los directorios de binarios conocidos), cuando se instala, entonces no se descarga ni se ejecuta su instalador.
- C-06: Dado un fallo al instalar `pi` o `mem`, cuando se instala, entonces la instalación se aborta con un error que nombra la herramienta y el comando manual; dado un fallo en `codegraph` o `codebase-memory-mcp`, entonces se avisa con el comando manual y la instalación continúa.
- C-07: Dado `PI_GOMEMORY_SKIP_DEPS` con un valor distinto de vacío, `0` o `false`, cuando se instala, entonces el paso no comprueba ni instala nada; dado el modo lite activo, entonces solo se comprueban `pi` y `mem`.
- C-08: Dado `codebase-memory-mcp` desactualizado, cuando se ejecuta `pi-gomemory-setup deps-update`, entonces se ejecuta `pi-gomemory-setup deps-install codebase-memory-mcp`, que instala el último release por el mismo canal verificado, y el aviso de dependencias muestra ese comando en lugar de una URL.
- C-09: Dado un instalador que no termina en 10 minutos (incluido uno que deja un proceso hijo vivo) o una descarga que no termina en 2, cuando se instala, entonces el grupo de procesos entero termina, el paso no se queda colgado y cuenta como fallo de esa herramienta con su comando manual (y se aplica C-06).
- C-12: Dada una descarga que redirige a HTTP, responde distinto de 200 o supera el límite de tamaño, cuando se instala, entonces no se ejecuta nada de lo descargado y cuenta como fallo; en todos los casos el directorio temporal se borra y no se escribe nada en el directorio actual.
- C-14: Dado un instalador ejecutado por el paso, cuando escribe en su salida estándar, entonces su salida es una tubería (`[ -p /dev/stdout ]`) aunque la salida de pi-gomemory sea un archivo, y con `/dev/tty` disponible el hijo comparte el grupo de procesos de pi-gomemory (comprobado con un script falso que informa su `pgid`).
- C-15: Dada una máquina sin Node.js 22.19+ y sin terminal, cuando se instala, entonces el instalador de pi termina con error en segundos (no a los 10 minutos) y el error de pi-gomemory pide instalar Node.js 22.19+ y repetir. Con terminal, la oferta de instalar Node.js la responde la persona (verificación manual).
- C-13: Dado `deps-install` con un nombre distinto de `codebase-memory-mcp` o sin nombre, cuando se ejecuta, entonces muestra el uso y termina con código 2 sin descargar nada.
- C-10: Dado `install.sh` ejecutado como `curl … | bash`, cuando lanza `pi-gomemory-setup install`, entonces lo hace con la entrada estándar en `/dev/null`, y `pi-gomemory-setup update` no pasa su entrada estándar al `install` hijo.
- C-11: Dada una máquina limpia real (HOME temporal y `PATH` con solo los directorios del sistema y el de Node.js), cuando se ejecuta `bash install.sh` desde la copia local del repositorio con Go en el `PATH` (opción de compilación local, y la salida debe mostrar la versión compilada en local, no `v1.20.0`), entonces, con el `PATH` del launcher (`~/.local/bin:~/.pi/agent/bin:~/.gomemory/bin` más el del sistema), `pi --version`, `mem version`, `codegraph --version` y `codebase-memory-mcp --version` responden. Requiere red.

## Aclaraciones
- [x] Q1. ¿Qué canal oficial se usa para pi? → `pi.dev/install.sh` (respuesta de la persona, 2026-11-20).
- [x] Q2. ¿Cómo se instala codebase-memory-mcp? → binario del release verificado con SHA-256, no su `install.sh` (respuesta de la persona, 2026-11-20).
- [x] Q3. ¿También se instala mem si falta? → sí (respuesta de la persona, 2026-11-20).

### Revisión Splash sobre 54e877a2f414
La persona aceptó Splash. Hallazgos (1 Blocker, 7 Major, 5 Nit) incorporados en esta revisión de la definición: B-1 y M-7 en «Ejecución de los instaladores» y C-09/C-10; M-1 y N-2 en «PATH» y C-01; M-2 y M-5 en «Actualizaciones» y C-08; M-3 en «Riesgo aceptado»; M-4 en «Pruebas»; M-6 en «Evidencia de los canales»; N-1 en «Orden seguro»; N-3 y N-4 en «Obligatorias y opcionales» y C-07; N-5 en C-11.

### Revisión Splash sobre 31d85284b8e4
La persona aceptó una segunda revisión Splash. Hallazgos (1 Blocker, 5 Major, 6 Nit) incorporados: B-1 en «Pruebas» (tests de binario con `PI_GOMEMORY_SKIP_DEPS=1`); M-1 en «Publicación» y C-11; M-2 en «Ejecución de los instaladores» y C-09; M-3 en «Ejecución de los instaladores» y C-12; M-4 en «Efectos fuera de pi-gomemory»; M-5 en «Actualizaciones» y C-13; N-1, N-2 y N-5 en «PATH»; N-3 y N-4 en «Orden seguro»; N-6 en «Obligatorias y opcionales».

### Revisión Splash sobre b56d099b2981
La persona aceptó una tercera revisión Splash. Hallazgos (1 Blocker, 4 Major, 4 Nit) incorporados: B-1 en «Ejecución de los instaladores» (mismo grupo de procesos con terminal; grupo propio sin ella) y C-14; M-1 en «Orden seguro» (`fetchLatestTag`, sin API de GitHub); M-2 en «PATH» (Node.js lo resuelve el lanzador de pi) y C-15; M-3 en «Actualizaciones» (destino fijo y aviso de copia con precedencia); M-4 en «Ejecución» y C-14; N-1 en «PATH»; N-2 y N-3 en «Ejecución»; N-4 en «Publicación».

## Slices
### VS-01. Paso de herramientas en InstallAll
Tareas: 1, 2, 3, 4, 5, 6
Criterios: C-01, C-02, C-03, C-04, C-05, C-06, C-07, C-09, C-12, C-14

### VS-02. Actualización verificada de codebase-memory-mcp
Tareas: 7
Criterios: C-08, C-13

### VS-03. Integración con install.sh, update, launcher, doctor y uninstall
Tareas: 8, 9, 10, 11, 12
Criterios: C-10

### VS-04. Validación real
Tareas: 13, 14, 15
Criterios: C-11, C-15

## Checklist
- [x] 1. `pkg/installer/tools.go`: lista de herramientas, directorios conocidos y `PATH` ampliado (orden, sin duplicar, `MkdirAll` del directorio de binarios); detección de presentes (C-05) → `go test ./pkg/installer -run TestHCD062Detect`
- [x] 2. `pkg/installer/tools_exec.go`: ejecución de scripts con `/dev/null`, tubería, temporal privado junto al directorio de binarios como cwd, plazo y grupo de procesos (C-09, C-14) → `go test ./pkg/installer -run TestHCD062Runner`
- [x] 3. `pkg/installer/tools_exec.go`: descarga endurecida (solo HTTPS, 200, límite, temporal borrado) (C-12) → `go test ./pkg/installer -run TestHCD062Download`
- [x] 4. `pkg/installer/tools.go`: `EnsureTools` instala `pi`, `mem` y `codegraph` por sus scripts con su entorno, obligatorias y opcionales, `PI_GOMEMORY_SKIP_DEPS` y lite (C-01, C-02, C-03, C-06, C-07) → `go test ./pkg/installer -run TestHCD062Ensure`
- [x] 5. `pkg/installer/tools_cbm.go`: `codebase-memory-mcp` desde el release (etiqueta validada, SHA-256, extracción exacta, `install --skip-config`) (C-04) → `go test ./pkg/installer -run TestHCD062Codebase`
- [x] 6. `pkg/installer/installer.go`: paso [1/9] antes de retirar la marca de desinstalado; `TestMain` con herramientas falsas; tests de binario con `PI_GOMEMORY_SKIP_DEPS=1` → `go test ./pkg/installer -run 'TestHCD062InstallAll|TestHCD047'`
- [x] 7. `main.go` y `pkg/installer/deps.go`: subcomando `deps-install`, `Update` de la sonda, destino según manifiesto y `os.Executable()` en `UpdateDeps` (C-08, C-13) → `go test ./... -run 'TestHCD062DepsInstall|TestHCD012'`
- [x] 8. `pkg/installer/tools_exec.go`: Ctrl‑C borra el temporal; `install` y `uninstall` retiran `.pi-gomemory-deps-*` huérfanos → `go test ./pkg/installer -run TestHCD062Stale`
- [x] 9. `pkg/installer/harness.go` y `pkg/installer/doctor.go`: launcher con `~/.pi/agent/bin`; doctor con el `PATH` ampliado → `go test ./pkg/installer -run TestHCD062LauncherDoctor`
- [x] 10. `install.sh` y `main.go`: `install` con `</dev/null`, nota de `PATH` también al fallar y solo con directorios usados; `update` sin `os.Stdin` (C-10) → `go test . ./pkg/installer -run TestHCD062Shell`
- [x] 11. `pkg/installer/uninstall.go`: el resumen conserva las cuatro herramientas y dice cómo quitarlas → `go test ./pkg/installer -run TestHCD062Uninstall`
- [x] 12. `README.md`: documenta el paso, los canales, `PI_GOMEMORY_SKIP_DEPS` y `deps-install` → `grep -q 'PI_GOMEMORY_SKIP_DEPS' README.md && grep -q 'deps-install' README.md && grep -q 'pi.dev/install.sh' README.md`
- [x] 13. Suite completa en verde → `go test ./...`
- [x] 14. `scripts/clean-install-check.sh`: instalación real en un HOME temporal con compilación local; las cuatro herramientas responden (C-11) → `bash scripts/clean-install-check.sh`
- [x] 15. Sin Node.js ni terminal, el instalador de pi falla en segundos con el mensaje de Node.js (C-15) → `bash scripts/clean-install-check.sh --no-node`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-20 18:55 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD062Detect` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd062_tools_test.go:30:11: undefined: skipDepsEnv
      pkg/installer/hcd062_tools_test.go:35:36: undefined: toolHooks
      pkg/installer/hcd062_tools_test.go:37:9: undefined: hooks
      pkg/installer/hcd062_tools_test.go:38:2: undefined: hooks
      pkg/installer/hcd062_tools_test.go:39:21: undefined: hooks
      pkg/installer/hcd062_tools_test.go:42:38: undefined: Tool
      pkg/installer/hcd062_tools_test.go:44:9: undefined: installTools
      pkg/installer/hcd062_tools_test.go:45:2: undefined: installTools
      pkg/installer/hcd062_tools_test.go:46:21: undefined: installTools
      pkg/installer/hcd062_tools_test.go:79:38: undefined: toolHooks
      pkg/installer/hcd062_tools_test.go:46:21: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-20 18:57 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD062Runner` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.950s
- 2026-11-20 18:57 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD062Detect` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.235s
- 2026-11-20 18:57 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run TestHCD062Codebase` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.376s
- 2026-11-20 18:57 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD062Ensure` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.277s
- 2026-11-20 18:57 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD062Download` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.240s
- 2026-11-20 18:57 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD062InstallAll|TestHCD047'` (salida 1)
                             [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD062InstallAllContinuesWhenPresent298198717/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD062InstallAllContinuesWhenPresent298198717/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	4.454s
      FAIL
- 2026-11-20 18:59 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run 'TestHCD062InstallAll|TestHCD047'` (salida 0)
      ok  	pi-gomemory/pkg/installer	3.919s
- 2026-11-20 19:00 -05:00 · tarea 7 · rojo observado · caducada · `go test ./... -run 'TestHCD062DepsInstall|TestHCD012'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd062_deps_test.go:59:14: undefined: RunDepsInstall
      pkg/installer/hcd062_deps_test.go:85:13: undefined: RunDepsInstall
      ok  	pi-gomemory	0.212s [no tests to run]
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-20 19:00 -05:00 · tarea 7 · verde · caducada · `go test ./... -run 'TestHCD062DepsInstall|TestHCD012'` (salida 0)
      ok  	pi-gomemory	0.186s [no tests to run]
      ok  	pi-gomemory/pkg/installer	1.018s
- 2026-11-20 19:01 -05:00 · tarea 8 · rojo NO observado (el comando pasó) · caducada · `go test ./pkg/installer -run TestHCD062Stale` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.623s
- 2026-11-20 19:02 -05:00 · tarea 9 · rojo NO observado (el comando pasó) · caducada · `go test ./pkg/installer -run TestHCD062LauncherDoctor` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.346s
- 2026-11-20 19:04 -05:00 · tarea 10 · rojo observado · caducada · `go test . ./pkg/installer -run TestHCD062Shell` (salida 1)
      # pi-gomemory [pi-gomemory.test]
      ./hcd062_shell_test.go:82:9: undefined: installChildCmd
      FAIL	pi-gomemory [build failed]
      ok  	pi-gomemory/pkg/installer	0.213s [no tests to run]
      FAIL
- 2026-11-20 19:04 -05:00 · tarea 10 · verde · caducada · `go test . ./pkg/installer -run TestHCD062Shell` (salida 0)
      ok  	pi-gomemory	0.343s
      ok  	pi-gomemory/pkg/installer	(cached) [no tests to run]
- 2026-11-20 19:05 -05:00 · tarea 12 · verde · caducada · `grep -q 'PI_GOMEMORY_SKIP_DEPS' README.md && grep -q 'deps-install' README.md && grep -q 'pi.dev/install.sh' README.md` (salida 0)
- 2026-11-20 19:12 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD062Detect` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.243s
- 2026-11-20 19:12 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD062Runner` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.940s
- 2026-11-20 19:12 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD062Download` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.218s
- 2026-11-20 19:12 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run TestHCD062Ensure` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.337s
- 2026-11-20 19:12 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run TestHCD062Codebase` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.364s
- 2026-11-20 19:12 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run 'TestHCD062InstallAll|TestHCD047'` (salida 0)
      ok  	pi-gomemory/pkg/installer	4.129s
- 2026-11-20 19:12 -05:00 · tarea 7 · verde · caducada · `go test ./... -run 'TestHCD062DepsInstall|TestHCD012'` (salida 0)
      ok  	pi-gomemory	0.180s [no tests to run]
      ok  	pi-gomemory/pkg/installer	0.998s
- 2026-11-20 19:12 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run TestHCD062Stale` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.637s
- 2026-11-20 19:12 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run TestHCD062LauncherDoctor` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.286s
- 2026-11-20 19:12 -05:00 · tarea 10 · verde · caducada · `go test . ./pkg/installer -run TestHCD062Shell` (salida 0)
      ok  	pi-gomemory	0.363s
      ok  	pi-gomemory/pkg/installer	0.204s [no tests to run]
- 2026-11-20 19:12 -05:00 · tarea 11 · verde · caducada · `go test ./pkg/installer -run TestHCD062Uninstall` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.527s
- 2026-11-20 19:12 -05:00 · tarea 12 · verde · caducada · `grep -q 'PI_GOMEMORY_SKIP_DEPS' README.md && grep -q 'deps-install' README.md && grep -q 'pi.dev/install.sh' README.md` (salida 0)
- 2026-11-20 19:12 -05:00 · tarea 15 · verde · caducada · `bash scripts/clean-install-check.sh --no-node` (salida 0)
      ✓ Sin Node.js ni terminal: falla en 16 s y pide Node.js 22.19+ (código 1).
- 2026-11-20 19:12 -05:00 · tarea 14 · verde · caducada · `bash scripts/clean-install-check.sh` (salida 0)
      ✓ pi --version: 1.6.0
      ✓ mem version: gomemory 2.55.0
      ✓ codegraph --version: 1.6.4
      ✓ codebase-memory-mcp --version: codebase-memory-mcp 0.12.1
      ✓ Instalación limpia completa (log: /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/tmp.lgWHmocw6W/install.log se borra al salir).
- 2026-11-20 19:12 -05:00 · tarea 13 · verde · caducada · `go test ./...` (salida 0)
      ok  	pi-gomemory	0.907s
      ok  	pi-gomemory/pkg/installer	51.310s
- 2026-11-20 19:14 -05:00 · tarea 8 · rojo observado · caducada · `cd /tmp/hcd062-red && go test -count=1 ./pkg/installer -run TestHCD062Stale` (salida 1)
      signal: interrupt
      FAIL	pi-gomemory/pkg/installer	0.426s
      FAIL
- 2026-11-20 19:14 -05:00 · tarea 9 · rojo observado · caducada · `cd /tmp/hcd062-red && go test -count=1 ./pkg/installer -run TestHCD062LauncherDoctor` (salida 1)
                  # Evitar desbordar el límite de tamaño de argumentos del kernel (≈128 KiB).
                  # Si excede 64 KiB, se trunca de forma segura para no romper 'exec pi'.
                  if [ "${#CTX}" -gt 65536 ]; then
                    CTX="${CTX:0:65536}"
                  fi
                  GOMEMORY_ARGS+=(--append-system-prompt "${CTX}")
                fi
              fi
              # 5. Ejecutar Pi con el tema gomemory
              if [ ${#GOMEMORY_ARGS[@]} -gt 0 ]; then
                exec pi --use-theme "gomemory" "${GOMEMORY_ARGS[@]}" "$@"
              else
                exec pi --use-theme "gomemory" "$@"
              fi
      --- FAIL: TestHCD062LauncherDoctorFindsKnownDirs (0.00s)
          hcd062_launcher_test.go:25: doctor no encuentra pi en ~/.pi/agent/bin: {Name:Binario pi Status:FAIL Details:no encontrado en PATH}
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.428s
      FAIL
- 2026-11-20 19:14 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run TestHCD062LauncherDoctor` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-20 19:15 -05:00 · tarea 9 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run TestHCD062LauncherDoctor` (salida 1)
                  # Evitar desbordar el límite de tamaño de argumentos del kernel (≈128 KiB).
                  # Si excede 64 KiB, se trunca de forma segura para no romper 'exec pi'.
                  if [ "${#CTX}" -gt 65536 ]; then
                    CTX="${CTX:0:65536}"
                  fi
                  GOMEMORY_ARGS+=(--append-system-prompt "${CTX}")
                fi
              fi
              # 5. Ejecutar Pi con el tema gomemory
              if [ ${#GOMEMORY_ARGS[@]} -gt 0 ]; then
                exec pi --use-theme "gomemory" "${GOMEMORY_ARGS[@]}" "$@"
              else
                exec pi --use-theme "gomemory" "$@"
              fi
      --- FAIL: TestHCD062LauncherDoctorFindsKnownDirs (0.00s)
          hcd062_launcher_test.go:25: doctor no encuentra pi en ~/.pi/agent/bin: {Name:Binario pi Status:FAIL Details:no encontrado en PATH}
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.255s
      FAIL
- 2026-11-20 19:15 -05:00 · tarea 8 · rojo observado · caducada · `go test -count=1 ./pkg/installer -run TestHCD062Stale` (salida 1)
        [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD062StaleTempsRemovedByInstall2632258808/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD062StaleTempsRemovedByInstall2632258808/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- FAIL: TestHCD062StaleTempsRemovedByInstall (0.43s)
          hcd062_cleanup_test.go:65: debe retirarse /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD062StaleTempsRemovedByInstall2632258808/001/.local/bin/.pi-gomemory-deps-abc: <nil>
      FAIL
      FAIL	pi-gomemory/pkg/installer	3.684s
      FAIL
- 2026-11-20 19:16 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestHCD062Detect` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.209s
- 2026-11-20 19:16 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD062Runner` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.952s
- 2026-11-20 19:16 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD062Download` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.219s
- 2026-11-20 19:16 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run TestHCD062Ensure` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.318s
- 2026-11-20 19:16 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run TestHCD062Codebase` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.367s
- 2026-11-20 19:16 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run 'TestHCD062InstallAll|TestHCD047'` (salida 0)
      ok  	pi-gomemory/pkg/installer	3.971s
- 2026-11-20 19:16 -05:00 · tarea 7 · verde · vigente · `go test ./... -run 'TestHCD062DepsInstall|TestHCD012'` (salida 0)
      ok  	pi-gomemory	(cached) [no tests to run]
      ok  	pi-gomemory/pkg/installer	0.991s
- 2026-11-20 19:16 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run TestHCD062Stale` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.556s
- 2026-11-20 19:16 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run TestHCD062LauncherDoctor` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.278s
- 2026-11-20 19:16 -05:00 · tarea 10 · verde · vigente · `go test . ./pkg/installer -run TestHCD062Shell` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	0.213s [no tests to run]
- 2026-11-20 19:16 -05:00 · tarea 11 · verde · vigente · `go test ./pkg/installer -run TestHCD062Uninstall` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.495s
- 2026-11-20 19:16 -05:00 · tarea 12 · verde · vigente · `grep -q 'PI_GOMEMORY_SKIP_DEPS' README.md && grep -q 'deps-install' README.md && grep -q 'pi.dev/install.sh' README.md` (salida 0)
- 2026-11-20 19:16 -05:00 · tarea 13 · verde · vigente · `go test ./...` (salida 0)
      ok  	pi-gomemory	(cached)
      ok  	pi-gomemory/pkg/installer	40.696s
- 2026-11-20 19:17 -05:00 · tarea 14 · verde · vigente · `bash scripts/clean-install-check.sh` (salida 0)
      ✓ pi --version: 1.6.0
      ✓ mem version: gomemory 2.55.0
      ✓ codegraph --version: 1.6.4
      ✓ codebase-memory-mcp --version: codebase-memory-mcp 0.12.1
      ✓ Instalación limpia completa (log: /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/tmp.oY0HqFesGZ/install.log se borra al salir).
- 2026-11-20 19:18 -05:00 · tarea 15 · verde · vigente · `bash scripts/clean-install-check.sh --no-node` (salida 0)
      ✓ Sin Node.js ni terminal: falla en 7 s y pide Node.js 22.19+ (código 1).

## Siguiente
Cerrada con aprobación sobre la revisión 94814dc0f678.

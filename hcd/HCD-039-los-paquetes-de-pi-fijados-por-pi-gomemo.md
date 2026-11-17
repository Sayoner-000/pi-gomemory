# Los paquetes de Pi fijados por pi-gomemory no se quedan atrás frente a npm sin que nadie lo sepa

ID: HCD-039
Tipo: feature
Carril: full
Motivo: Tres decisiones interdependientes (bloquear o avisar en la release, revisión programada, visibilidad del aviso al usuario) y modifica la política de versiones exactas de HCD-005

## Intención
Cada release de pi-gomemory sabe si pi-antigravity y pi-cc-extensions van atrás respecto de npm, valida el paquete candidato contra los contratos de los que depende pi-gomemory antes de subir el pin, y la persona sabe cuándo hay una versión publicada todavía no validada, sin que se instale nada sin probar


## Contexto
[HCD-005](HCD-005-fijar-versiones-de-pi-antigravity-y-pi-c.md) fijó `pi-antigravity@0.12.0` y `pi-cc-extensions@1.0.16` a versión exacta: Pi no vuelve a revisar un paquete con versión exacta y la instalación es reproducible. Esta unidad conserva esa decisión (la Constitución §0 prohíbe `latest` y exige que una actualización de dependencias vaya en un cambio separado, con pruebas y vuelta atrás). Lo que cambia es que hoy nadie sabe cuándo el pin se queda atrás:

- `DefaultDepProbes` (`pkg/installer/deps.go:236-247`) usa `fixedVersion(AntigravityVersion)` como «última publicada», así que el aviso de dependencias compara el pin consigo mismo y nunca consulta npm para estos dos paquetes.
- No hay CI (`.github/workflows` no existe). El 2026-11-17 el desfase se descubrió por casualidad, diagnosticando un error de Antigravity: npm publica `pi-antigravity@0.14.0` (2026-11-14) y `pi-cc-extensions@1.0.20`.

pi-gomemory depende del interior de esos paquetes en estos contratos, comprobados en el código de `pi-antigravity` 0.12.0 y 0.14.0 y de `pi-cc-extensions` 1.0.16. Los dos paquetes publican su fuente TypeScript (`src/` y `extensions/`), así que cada contrato se comprueba sobre los archivos del tarball:

| Id | Paquete | Archivo del tarball | Regla | De qué depende pi-gomemory |
|---|---|---|---|---|
| A1 | pi-antigravity | `package/src/auth/accounts.ts` | contiene `"antigravity-accounts.json"`, `activeAccountId` y `accounts: Record<` (la declaración del tipo, `accounts.ts:23` en 0.12.0; una subcadena `accounts` sola coincidiría con el nombre del archivo) | `usage.go` lee `activeAccountId` y `accounts.<id>` de ese archivo |
| A3 | pi-antigravity | `package/src/models/models.ts` | contiene `PROVIDER_ID = "antigravity"` | la detección del proveedor en el sidebar |
| C1 | pi-cc-extensions | algún `package/extensions/**/*.ts` | contiene `showStartupHeader` | `managedCCKeys` en `pkg/installer/plugin.go` |

Las rutas llevan el prefijo `package/` con el que npm empaqueta los tarballs. Las reglas son textuales: detectan un renombre o una retirada, no un cambio de significado con el mismo nombre. Quedan fuera de las reglas:
- La clave `version` del archivo de cuentas, que `usage.go` no lee.
- Los campos de cada cuenta. `usage.go:19-23` declara `access`, `refresh`, `projectId`, `email` y `expires`, pero `FetchQuotaSummary` y `FetchAndFormatUsage` solo usan `access` y `email`. Esos campos vienen del tipo `OAuthCredentials` de `@earendil-works/pi-ai`, fuera de estos tarballs.
- `typebox` en `package.json` (A2 se retiró tras la segunda revisión de Splash). `PatchAntigravityPackageJSON` (`plugin.go:107-148`) pone siempre `typebox: "*"` en `peerDependencies` y lo quita de `dependencies` si está, así que cualquiera de las dos formas funciona y no es un contrato. 0.12.0 y 0.14.0 ya lo declaran como `peerDependency`.

## Alcance
Incluido: detectar en la release el desfase de los dos pins frente a npm, validar un paquete candidato contra los contratos de su tabla sin instalarlo, y avisar en `doctor` de lo publicado en npm que pi-gomemory todavía no validó. En detalle:
- **Detección (`pins-check`).** Prueba Go con la etiqueta de compilación `pins`, no comando del binario distribuido: solo la usa quien mantiene el proyecto y el `go test` normal no sale a la red (examen socrático, P2). Se ejecuta siempre sin caché de resultados y con salida visible: `go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsCheck$'`, porque Go no sabe que la prueba depende de la red y repetiría un verde antiguo, y sin `-v` ocultaría lo aceptado. Consulta `NPMRegistryURL` para cada paquete fijado. Lee la lista de versiones del documento del paquete (`/<paquete>`) con `Accept: application/vnd.npm.install-v1+json`, el formato abreviado que incluye `versions.<v>.dist`, y con un límite propio de 16 MiB, no con `httpJSON`, que corta en 4 MiB (`deps.go:163`). Calcula `published`, la versión estable (sin prereleases) más alta que no esté rechazada; puede ser igual al pin. Hay desfase si `published` es mayor que el pin (Q4): una versión intermedia sin revisar nunca queda oculta por un rechazo. Son errores, y nunca «al día», que falte `versions`, que no haya ninguna versión estable o que el pin no aparezca en la lista. La misma función calcula el `published` que guarda la caché para `doctor`. La prueba lee `PI_GOMEMORY_ALLOW_STALE_PINS`: con la variable a `1`, informa cada desfase o error con `t.Log` y pasa; sin ella, falla.
- **Versiones rechazadas.** Una lista en Go (`pkg/installer/pins.go`, junto a las constantes de `plugin.go`), compilada en el binario, con paquete, versión y motivo de cada versión revisada y descartada (examen socrático, P1). La usan `pins-check` y `doctor`.
- **Validación de candidato (`pins-validate`).** Prueba Go con la etiqueta `pins` que lee `PINS_CANDIDATE=<paquete>@<versión>` y se ejecuta con `PINS_CANDIDATE=pi-antigravity@0.14.0 go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$'`, para que no arrastre `TestPinsCheck`. Toma `dist.tarball` y `dist.integrity` de la misma lista de versiones y descarga el tarball con un cliente propio a un directorio temporal, con un límite de 50 MiB, comprueba su `dist.integrity` antes de extraer (si falta o no es `sha512-…`, rechaza el candidato), extrae con un límite de 200 MiB descomprimidos, solo archivos normales y directorios (rechaza enlaces simbólicos o duros, dispositivos y FIFO), y rechazando rutas absolutas y componentes `..`, y aplica las reglas de la tabla de contratos de ese paquete. Nombra cada contrato con su id y su resultado. No escribe en `~/.pi/agent` y borra el directorio temporal al terminar.
- **Punto de control en la release.** `scripts/build-release.sh` añade `go vet -tags pins ./pkg/installer`, para revisar también los archivos de esa etiqueta, y ejecuta `pins-check` después de `go vet` y `go test -race` (líneas 11-12) y antes de `rm -rf "$DIST_DIR"`. La invocación va dentro de un `if ! …; then` para que `set -euo pipefail` (línea 5) no corte el script antes de imprimir cómo seguir (C-09 a C-11).
- **Caché de dependencias.** El probe de cada paquete fijado conserva `Latest = fixedVersion(...)`: `deps-status` y `deps-update` siguen ofreciendo solo el pin, y `TestPinnedDepProbes` (`hcd005_test.go:34-53`) no cambia. Se añaden a `DepProbe` una función opcional `Published` (nil en los demás probes, que `RefreshDeps` debe tolerar) y a `DepState` dos campos propios, `published` y `published_error` (con `omitempty`), más la fecha de la consulta, que ya guarda `checked_at`. El error de npm va en `published_error` y nunca en `error`, porque `Outdated()` (`deps.go:54-56`) devuelve falso con `Error != ""` y un corte de red dejaría de ofrecer el pin. `RefreshDeps` consulta `Published` aunque `Current` falle (hoy corta en `deps.go:280-285`), para saber la versión de npm también sin el paquete instalado. Además trata como caducada una caché en la que un probe con `Published` no tenga ni `published` ni `published_error` (por ejemplo, la escrita por un binario anterior), para que `deps-refresh` la rellene sin esperar la hora de `depsCacheTTL` (`deps.go:261`). Con `PI_GOMEMORY_NO_UPDATE_CHECK` activa manda la variable: `RefreshDeps` sigue devolviendo la caché tal como está, sin red (HCD-037).
- **Aviso en `doctor` (Q1).** `RunDoctor` (`doctor.go:38`), que hoy no lee la caché de dependencias, pasa a leer `pi-gomemory-deps.json` y añade una línea por paquete fijado al final de la lista, después de los agentes detectados (sección 8). Así no cambia la posición de las comprobaciones que ya existen, aunque las pruebas que comparan la salida completa de `doctor` (por ejemplo, `hcd011_options_test.go:109-122`) se revisan. Por cada paquete muestra una de estas líneas, sin salir a la red y comparando con la constante del pin compilada en el binario, no con lo guardado en la caché:
  - `⚠` (WARN) si `published` es mayor que el pin, con la versión fijada, la de npm y la fecha de la consulta.
  - `✓` (OK) si `published` es igual al pin, con la fecha de la consulta.
  - `→` (INFO) con la versión y el motivo si hay versiones por encima del pin y todas están rechazadas.
  - `→` (INFO) «sin datos de npm» en los casos de C-22.

  Con pi-gomemory desinstalado no se muestra, porque `RunDoctor` sale antes (`doctor.go:44-50`). El diagnóstico que imprime `update` sale del `install` que lanza el binario nuevo (`main.go:225` lo lanza; `installer.go:91` lo imprime), así que compara con el pin nuevo. El pie `↑ actualizaciones disponibles`, `deps-status` y `deps-update` no cambian por esta diferencia.
- **Pruebas de la suite normal** con un registro npm falso y tarballs sintéticos, sin salir a la red. Un `TestMain` de `pkg/installer` apunta `NPMRegistryURL` a una dirección inválida, salvo cuando el binario de pruebas se compila con la etiqueta `pins`: una constante definida en dos archivos con etiquetas opuestas (`//go:build pins` y `//go:build !pins`) se lo indica. Así cualquier prueba de la suite normal que llame a `Published` sin su propio servidor local falla en vez de salir a internet, y `TestPinsCheck` y `TestPinsValidate` consultan el registro real. Con la etiqueta `pins` solo se ejecutan esas dos pruebas, siempre con `-run`, desde `build-release.sh` o a petición explícita.

Excluido: subir los pins a 0.14.0 y 1.0.20, rangos o `latest` en `settings.json`, un manifiesto de compatibilidad remoto y cualquier instalación automática de una versión no validada. En detalle:
- Subir los pins va en una HCD posterior que use `pins-validate` como prueba.
- Rangos o `latest`: contradicen HCD-005 y la Constitución §0.
- Manifiesto remoto que el binario descargue para cambiar pins sin release: abre un riesgo de cadena de suministro y publicar una release ya es barato.
- Los demás componentes (`pi`, `mem`, `codegraph`, `codebase-memory-mcp`) ya comparan con su última publicada y no cambian.
- Un workflow programado de GitHub Actions (Q3): bastan la detención en cada release y el aviso de `doctor`; sería la primera CI del repositorio y se decide en su propia unidad.
- `/gitlab-release` es una skill externa al repositorio: el punto de control vive en `scripts/build-release.sh` y `make dist`, que sí están aquí.

## ¿Qué pasa si no lo hacemos?
Los pins siguen sin seguimiento. Hoy `pi-antigravity` va dos versiones menores atrás y `pi-cc-extensions` cuatro parches, y se supo por casualidad. La persona no recibe catálogos de modelos nuevos ni correcciones del plugin, y un cambio de contrato (por ejemplo, del formato de `antigravity-accounts.json`) se descubriría al romperse la cuota en el uso real, no antes de subir el pin.

## Alternativa más barata
- Un recordatorio en la lista de pasos de la release: no cuesta código, pero depende de que alguien lo recuerde, que es justo lo que falló.
- Solo la detección (`pins-check` en la release), sin validación ni aviso: es la primera capa y la más barata; se descartó al abrir la unidad porque dejaría la subida del pin sin pruebas de contrato.
Se propone el alcance completo, con cada capa en su propio slice para que la detección se pueda entregar primero.

## Historias
### US1. Saber en cada release si un pin va atrás
Como mantenedor de pi-gomemory, quiero que preparar una release me diga qué paquetes fijados van atrás respecto de npm, para no publicar sin saberlo.
- C-01: Dado un pin por debajo de la versión `published` de npm, cuando se ejecuta `pins-check`, entonces lista el paquete, la versión fijada y la publicada, y sale con un código distinto de cero.
- C-02: Dado `published` igual al pin en todos los paquetes fijados, cuando se ejecuta `pins-check`, entonces sale con 0.
- C-03: Dado un registro npm que no responde, o un documento sin `versions`, sin ninguna versión estable o en el que no aparece el pin, cuando se ejecuta `pins-check`, entonces informa el error por paquete, nunca lo presenta como «al día» y sale con un código distinto de cero.
- C-04: Dadas todas las versiones estables por encima del pin registradas como rechazadas, cuando se ejecuta `pins-check`, entonces las muestra como «rechazada: <motivo>» y sale con 0; si npm publica después una versión más nueva, vuelve a contar como desfase.
- C-05: Dado el pin en 0.12.0, con 0.13.0 y 0.14.0 publicadas y solo 0.14.0 rechazada, cuando se ejecutan `pins-check` y `doctor`, entonces informan del desfase con 0.13.0 (Q4).
- C-06: Dada una prerelease (`0.15.0-beta.1`) como única versión por encima del pin, cuando se ejecuta `pins-check`, entonces no la cuenta como desfase.
- C-07: Dado `go test ./...` sin la etiqueta `pins`, cuando se ejecuta la suite, entonces ninguna prueba consulta la red: con `NPMRegistryURL` inválido en `TestMain`, la suite completa pasa. Con la etiqueta `pins`, `TestPinsCheck` consulta el registro real.
- C-08: Dada una ejecución verde de `pins-check` seguida de una versión nueva en npm, cuando `scripts/build-release.sh` vuelve a ejecutarse, entonces la detecta: la invocación lleva `-count=1` y no reutiliza el resultado en caché.
- C-09: Dado un pin atrasado que no está registrado como rechazado, cuando se ejecuta `scripts/build-release.sh`, entonces se detiene después de `go test -race` y antes de borrar `dist/`, muestra el paquete, la versión fijada y la de npm, e indica cómo seguir: validarla con `pins-validate`, registrarla como rechazada o publicar igualmente con `PI_GOMEMORY_ALLOW_STALE_PINS=1` (Q2).
- C-10: Dado un registro npm que no responde, cuando se ejecuta `scripts/build-release.sh`, entonces también se detiene, y `PI_GOMEMORY_ALLOW_STALE_PINS=1` permite seguir (Q2).
- C-11: Dado `PI_GOMEMORY_ALLOW_STALE_PINS=1`, cuando se ejecuta `scripts/build-release.sh`, entonces empaqueta y deja en la salida lo que se aceptó: el desfase de cada paquete atrasado, o «no comprobado: error de red» si el registro no respondió (Q2).

### US2. Validar un candidato antes de subir el pin
Como mantenedor, quiero comprobar una versión candidata contra los contratos de los que depende pi-gomemory, para subir el pin con evidencia y no a ciegas.
- C-12: Dado un tarball candidato que cumple todas las reglas de su paquete en la tabla de contratos (A1 y A3 para `pi-antigravity`, C1 para `pi-cc-extensions`), cuando se ejecuta `pins-validate` con su comando canónico, entonces informa cada contrato por su id como cumplido y sale con 0.
- C-13: Dado un tarball candidato que rompe una regla de la tabla (por ejemplo, `src/models/models.ts` sin `PROVIDER_ID = "antigravity"`, o sin el archivo que la regla cita), cuando se ejecuta `pins-validate`, entonces sale con un código distinto de cero y nombra el id del contrato roto.
- C-14: Dado `PINS_CANDIDATE` vacío, cuando se ejecuta `pins-validate`, entonces se omite con `t.Skip`; dado un paquete que no está entre los fijados, un valor sin `@<versión>` o una versión que no aparece en el registro, entonces falla con un mensaje que lo explica.
- C-15: Dado un candidato sin `dist.integrity` o con un valor que no es `sha512-…`, un tarball cuyo sha512 no coincide, una descarga de más de 50 MiB, una extracción de más de 200 MiB, o un tarball con una ruta absoluta, un componente `..` o una entrada que no sea archivo normal ni directorio, cuando se ejecuta `pins-validate`, entonces lo rechaza antes de aplicar ninguna regla y sin escribir fuera del directorio temporal.
- C-16: Dada cualquier validación, cuando termina, entonces `~/.pi/agent` no cambió y el directorio temporal se eliminó.

### US3. Saber que hay una versión publicada sin validar
Como persona que usa pi-gomemory, quiero distinguir una versión publicada en npm de una validada por pi-gomemory, para no instalar algo sin probar y aun así saber que existe.
- C-17: Dado un pin atrasado frente a npm con el pin ya instalado, cuando se ejecuta `deps-update`, entonces no instala la versión de npm: solo ofrece la versión fijada.
- C-18: Dada una instalación por debajo del pin, cuando se ejecutan `deps-status` y `deps-update`, entonces siguen ofreciendo el pin, igual que hoy, también cuando la consulta a npm falla (el error queda en `published_error` y no en `error`).
- C-19: Dada una caché con `published` mayor que la constante del pin del binario, cuando se ejecuta `doctor`, entonces muestra `⚠` (WARN) con el paquete, la versión fijada, la de npm y la fecha de la consulta, e indica que no está validada por pi-gomemory, con el pin instalado, por debajo o por encima de él, y sin salir a la red (Q1).
- C-20: Dadas versiones por encima del pin, todas registradas como rechazadas, cuando se ejecuta `doctor`, entonces no muestra WARN: muestra `→` (INFO) con la versión y el motivo del rechazo.
- C-21: Dada una caché con `published` igual al pin, cuando se ejecuta `doctor`, entonces muestra `✓` (OK) con la fecha de la consulta.
- C-22: Dada una caché ausente, escrita por un binario anterior sin el campo `published`, o con solo `published_error`, cuando se ejecuta `doctor`, entonces no muestra WARN ni afirma que el pin esté al día: muestra `→` (INFO) «sin datos de npm; ejecuta pi-gomemory-setup deps-refresh». Con `PI_GOMEMORY_NO_UPDATE_CHECK` activa no recomienda `deps-refresh`, que no consultaría la red: dice «sin datos de npm (comprobación desactivada)».
- C-23: Dado el mismo caso que C-19, cuando se ejecutan `deps-status` y el pie de Pi, entonces no lo cuentan como actualización disponible (Q1). Hoy ya se cumple, porque el pie solo mira el código 10 de `deps-status` (`gomemory-status.ts:49-50`); este criterio protege que siga así.
- C-24: Dada una caché escrita por un binario anterior, sin `published` ni `published_error` en un paquete fijado, cuando se ejecuta `deps-refresh` antes de que pase una hora, entonces consulta npm igualmente y la caché queda con `published` o `published_error`; con `PI_GOMEMORY_NO_UPDATE_CHECK` activa no consulta y la caché no cambia.
- C-25: Dado un paquete fijado que no está instalado, cuando se ejecuta `deps-refresh`, entonces la caché guarda igualmente su `published`.

## Aclaraciones
- [x] Q1. Q3: con un pin atrasado frente a npm y el pin instalado, ¿dónde ve la persona que hay una versión publicada no validada por pi-gomemory? → En doctor, como aviso (WARN), para que se revise y evalúe con quien desarrolla pi-gomemory. Lee la última versión de npm de la caché pi-gomemory-deps.json, sin salir a la red; el pie ↑ actualizaciones y deps-update no cambian y nunca instalan la versión de npm.
- [x] Q2. Con un pin atrasado frente a npm, ¿scripts/build-release.sh detiene la release antes de empaquetar salvo que se pida saltarlo explícitamente, o solo avisa y sigue? → Detener, con escape: build-release.sh se detiene antes de compilar mostrando paquete, versión fijada y de npm, y cómo seguir (validar con pins-validate, registrar como rechazada o publicar igualmente con PI_GOMEMORY_ALLOW_STALE_PINS=1). Sin red también se detiene y la misma variable permite seguir.
- [x] Q3. ¿Se añade un workflow semanal de GitHub Actions que ejecute pins-check y abra un issue si hay desfase, o basta con la comprobación en cada release y el aviso de doctor? → No por ahora: bastan la detención en cada release (Q2) y el aviso de doctor (Q1). El workflow semanal sería la primera CI del repositorio y se decide en su propia unidad; queda en Excluido.
- [x] Q4. pins-check y doctor, ¿comparan el pin solo con la etiqueta latest de npm, o con todas las versiones publicadas mayores que el pin que no estén rechazadas? → Mayor no rechazada: se compara con la versión estable más alta publicada por encima del pin que no esté rechazada, leída de la lista de versiones del registro (sin prereleases). Un rechazo nunca oculta una versión intermedia sin revisar, y solo cuenta la más alta.

## Slices
### VS-01. Ninguna release sale sin saber si un pin va atrás frente a npm
Calcula `published` desde la lista de versiones del registro, registra las versiones rechazadas, aísla la suite normal de la red y detiene `scripts/build-release.sh` ante un desfase o un error de npm, salvo con `PI_GOMEMORY_ALLOW_STALE_PINS=1`. Se entrega solo: ya resuelve lo que motivó la unidad.
Tareas: 1, 2, 3, 4, 5
Criterios: C-01, C-02, C-03, C-04, C-05, C-06, C-07, C-08, C-09, C-10, C-11

### VS-02. Un paquete candidato se valida contra los contratos antes de subir el pin
Descarga el tarball del candidato con integridad sha512, lo extrae de forma segura en un directorio temporal y aplica las reglas A1, A3 y C1. Depende de la lectura de la lista de versiones de VS-01.
Tareas: 6, 7, 8
Criterios: C-12, C-13, C-14, C-15, C-16

### VS-03. `doctor` avisa de lo publicado en npm sin validar, sin tocar `deps-update` ni el pie
Guarda `published` y `published_error` en la caché de dependencias sin alterar lo que ofrece `deps-update`, y `doctor` muestra WARN, OK o INFO por cada paquete fijado. Depende del cálculo de `published` y de la lista de rechazadas de VS-01.
Tareas: 9, 10, 11, 12, 13, 14
Criterios: C-17, C-18, C-19, C-20, C-21, C-22, C-23, C-24, C-25

## Checklist
- [x] 1. Calcular `published` desde la lista de versiones abreviada del registro (estable más alta no rechazada; error sin `versions`, sin estable o sin el pin; límite de 16 MiB) en `pkg/installer/pins.go` (C-01, C-02, C-03, C-06) → go test ./pkg/installer -run '^TestHCD039Published$' -count=1
- [x] 2. Registrar en `pkg/installer/pins.go` la lista de versiones rechazadas con su motivo y aplicarla al cálculo de `published` (C-04, C-05) → go test ./pkg/installer -run '^TestHCD039Rejected$' -count=1
- [x] 3. Aislar de la red la suite normal con un `TestMain` y la constante por etiqueta `pins`/`!pins` (C-07) → go test ./pkg/installer -run '^TestHCD039NoNetworkByDefault$' -count=1
- [x] 4. Añadir `TestPinsCheck` con la etiqueta `pins`, sobre una función de informe probada contra un registro local, que respeta `PI_GOMEMORY_ALLOW_STALE_PINS` (C-01, C-02, C-03, C-11) → go test ./pkg/installer -run '^TestHCD039PinsCheckReport$' -count=1 && go vet -tags pins ./pkg/installer
- [x] 5. Ejecutar `pins-check` en `scripts/build-release.sh` con `-count=1 -v` dentro de un `if !`, después de `go test -race` y antes de borrar `dist/` (C-08, C-09, C-10, C-11) → go test ./pkg/installer -run '^TestHCD039BuildReleaseGate$' -count=1
- [x] 6. Descargar y extraer de forma segura el tarball del candidato (sha512 de `dist.integrity`, 50 MiB de descarga, 200 MiB extraídos, solo archivos normales y directorios, sin rutas absolutas ni `..`) en un directorio temporal que se borra (C-15, C-16) → go test ./pkg/installer -run '^TestHCD039SafeExtract$' -count=1
- [x] 7. Aplicar las reglas A1, A3 y C1 sobre el árbol extraído y nombrar el contrato roto por su id (C-12, C-13) → go test ./pkg/installer -run '^TestHCD039Contracts$' -count=1
- [x] 8. Añadir `TestPinsValidate` con la etiqueta `pins`, sobre una función de validación probada contra un registro local, que lee `PINS_CANDIDATE` y se omite si está vacío (C-12, C-14) → go test ./pkg/installer -run '^TestHCD039ValidateCandidate$' -count=1 && go vet -tags pins ./pkg/installer
- [x] 9. Guardar `published` y `published_error` en `DepState` con `DepProbe.Published` opcional, consultando aunque falle `Current` y sin tocar `Error` (C-18, C-25) → go test ./pkg/installer -run '^TestHCD039DepsPublished$' -count=1
- [x] 10. Tratar como caducada en `RefreshDeps` la caché sin `published` ni `published_error`, salvo con `PI_GOMEMORY_NO_UPDATE_CHECK` (C-24) → go test ./pkg/installer -run '^TestHCD039StaleCacheRefresh$' -count=1
- [x] 11. Mostrar en `doctor`, al final de la lista, una línea WARN, OK o INFO por paquete fijado leída de la caché (C-19, C-20, C-21, C-22) → go test ./pkg/installer -run '^TestHCD039Doctor$' -count=1
- [x] 12. Conservar `deps-update`, `deps-status` y el pie sin cambios ante el desfase (guarda sin rojo) (C-17, C-23) → go test ./pkg/installer -run '^(TestHCD039DepsUnchanged|TestPinnedDepProbes)$' -count=1
- [x] 13. Mantener verde la suite completa con el detector de carreras → go test -race ./... -count=1
- [x] 14. Mantener limpio el análisis estático con y sin la etiqueta `pins` → go vet ./... && go vet -tags pins ./...

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-17 07:13 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039Published$' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd039_pins_test.go:63:15: undefined: publishedVersion
      pkg/installer/hcd039_pins_test.go:73:19: undefined: registryVersions
      pkg/installer/hcd039_pins_test.go:80:15: undefined: registryVersions
      pkg/installer/hcd039_pins_test.go:83:15: undefined: registryVersions
      pkg/installer/hcd039_pins_test.go:90:45: undefined: maxRegistryDoc
      pkg/installer/hcd039_pins_test.go:95:15: undefined: registryVersions
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 07:14 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD039Published$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.374s
- 2026-11-17 07:14 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039Rejected$' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd039_pins_test.go:59:11: undefined: rejectedAbove
      pkg/installer/hcd039_pins_test.go:69:20: undefined: PinnedPackages
      pkg/installer/hcd039_pins_test.go:73:36: undefined: PinnedPackages
      pkg/installer/hcd039_pins_test.go:75:20: undefined: RejectedVersions
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 07:14 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD039Rejected$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.195s
- 2026-11-17 07:14 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039NoNetworkByDefault$' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd039_pins_test.go:47:5: undefined: pinsLive
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 07:14 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039NoNetworkByDefault$' -count=1` (salida 1)
      --- FAIL: TestHCD039NoNetworkByDefault (0.00s)
          hcd039_pins_test.go:51: la suite normal no debe apuntar al registro real: NPMRegistryURL = "https://registry.npmjs.org"
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.194s
      FAIL
- 2026-11-17 07:15 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD039NoNetworkByDefault$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.206s
- 2026-11-17 07:16 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039PinsCheckReport$' -count=1 && go vet -tags pins ./pkg/installer` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd039_pins_test.go:53:12: undefined: CheckPins
      pkg/installer/hcd039_pins_test.go:54:17: undefined: FormatPinChecks
      pkg/installer/hcd039_pins_test.go:65:16: undefined: FormatPinChecks
      pkg/installer/hcd039_pins_test.go:68:17: undefined: FormatPinChecks
      pkg/installer/hcd039_pins_test.go:77:20: undefined: FormatPinChecks
      pkg/installer/hcd039_pins_test.go:77:36: undefined: CheckPins
      pkg/installer/hcd039_pins_test.go:82:15: undefined: CheckPins
      pkg/installer/hcd039_pins_test.go:83:16: undefined: FormatPinChecks
      pkg/installer/hcd039_pins_test.go:87:16: undefined: FormatPinChecks
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 07:16 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD039PinsCheckReport$' -count=1 && go vet -tags pins ./pkg/installer` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.220s
- 2026-11-17 07:17 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039BuildReleaseGate$' -count=1` (salida 1)
              ==> Generando pack de distribución para pi-gomemory (v0.0.0-test)...
                -> Compilando darwin/arm64...
                -> Empaquetando pi-gomemory-setup_v0.0.0-test_darwin_arm64.tar.gz...
                -> Compilando darwin/amd64...
                -> Empaquetando pi-gomemory-setup_v0.0.0-test_darwin_amd64.tar.gz...
                -> Compilando linux/amd64...
                -> Empaquetando pi-gomemory-setup_v0.0.0-test_linux_amd64.tar.gz...
                -> Compilando linux/arm64...
                -> Empaquetando pi-gomemory-setup_v0.0.0-test_linux_arm64.tar.gz...
              ==> Calculando sumas de verificación SHA256...
              ✓ Pack de distribución generado exitosamente en dist/:
              total 40
              -rw-r--r--@ 1 josegomezj  staff   590B Nov 17 07:17 pi-gomemory-setup_v0.0.0-test_darwin_amd64.tar.gz
              -rw-r--r--@ 1 josegomezj  staff   589B Nov 17 07:17 pi-gomemory-setup_v0.0.0-test_darwin_arm64.tar.gz
              -rw-r--r--@ 1 josegomezj  staff   586B Nov 17 07:17 pi-gomemory-setup_v0.0.0-test_linux_amd64.tar.gz
              -rw-r--r--@ 1 josegomezj  staff   586B Nov 17 07:17 pi-gomemory-setup_v0.0.0-test_linux_arm64.tar.gz
              -rw-r--r--@ 1 josegomezj  staff   462B Nov 17 07:17 SHA256SUMS
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.415s
      FAIL
- 2026-11-17 07:17 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD039BuildReleaseGate$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.810s
- 2026-11-17 07:17 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039SafeExtract$' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd039_candidate_test.go:79:27: undefined: candidateTempRoot
      pkg/installer/hcd039_candidate_test.go:79:46: undefined: candidateDownloadLimit
      pkg/installer/hcd039_candidate_test.go:79:70: undefined: candidateExtractLimit
      pkg/installer/hcd039_candidate_test.go:80:2: undefined: candidateTempRoot
      pkg/installer/hcd039_candidate_test.go:81:21: undefined: candidateTempRoot
      pkg/installer/hcd039_candidate_test.go:81:40: undefined: candidateDownloadLimit
      pkg/installer/hcd039_candidate_test.go:81:64: undefined: candidateExtractLimit
      pkg/installer/hcd039_candidate_test.go:94:23: undefined: fetchCandidate
      pkg/installer/hcd039_candidate_test.go:106:22: undefined: fetchCandidate
      pkg/installer/hcd039_candidate_test.go:140:2: undefined: candidateDownloadLimit
      pkg/installer/hcd039_candidate_test.go:140:2: too many errors
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 07:18 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^TestHCD039SafeExtract$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.245s
- 2026-11-17 07:18 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039Contracts$' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd039_candidate_test.go:89:20: undefined: ContractResult
      pkg/installer/hcd039_candidate_test.go:90:19: undefined: ContractResult
      pkg/installer/hcd039_candidate_test.go:98:13: undefined: checkContracts
      pkg/installer/hcd039_candidate_test.go:105:10: undefined: checkContracts
      pkg/installer/hcd039_candidate_test.go:109:10: undefined: checkContracts
      pkg/installer/hcd039_candidate_test.go:113:10: undefined: checkContracts
      pkg/installer/hcd039_candidate_test.go:118:12: undefined: checkContracts
      pkg/installer/hcd039_candidate_test.go:122:10: undefined: checkContracts
      pkg/installer/hcd039_candidate_test.go:126:15: undefined: checkContracts
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 07:19 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD039Contracts$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.233s
- 2026-11-17 07:19 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039ValidateCandidate$' -count=1 && go vet -tags pins ./pkg/installer` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd039_candidate_test.go:97:13: undefined: validateCandidate
      pkg/installer/hcd039_candidate_test.go:98:17: undefined: FormatContracts
      pkg/installer/hcd039_candidate_test.go:106:12: undefined: validateCandidate
      pkg/installer/hcd039_candidate_test.go:107:16: undefined: FormatContracts
      pkg/installer/hcd039_candidate_test.go:117:16: undefined: validateCandidate
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 07:19 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD039ValidateCandidate$' -count=1 && go vet -tags pins ./pkg/installer` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.198s
- 2026-11-17 07:20 -05:00 · tarea 9 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039DepsPublished$' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd039_deps_test.go:55:8: ag.Published undefined (type DepState has no field or method Published)
      pkg/installer/hcd039_deps_test.go:59:8: cc.Published undefined (type DepState has no field or method Published)
      pkg/installer/hcd039_deps_test.go:62:46: o.Published undefined (type DepState has no field or method Published)
      pkg/installer/hcd039_deps_test.go:62:67: o.PublishedError undefined (type DepState has no field or method PublishedError)
      pkg/installer/hcd039_deps_test.go:67:104: hcd039StateOf(t, onDisk.Deps, "pi-antigravity").Published undefined (type DepState has no field or method Published)
      pkg/installer/hcd039_deps_test.go:76:8: ag.PublishedError undefined (type DepState has no field or method PublishedError)
      pkg/installer/hcd039_deps_test.go:76:35: ag.Published undefined (type DepState has no field or method Published)
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 07:20 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run '^TestHCD039DepsPublished$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.227s
- 2026-11-17 07:20 -05:00 · tarea 10 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039StaleCacheRefresh$' -count=1` (salida 1)
      --- FAIL: TestHCD039StaleCacheRefresh (0.00s)
          hcd039_deps_test.go:60: C-24: la caché antigua debe refrescarse antes de la hora: llamadas=0 [{Name:pi-antigravity Current:0.12.0 Latest:0.12.0 Update:[] Manual: Error: Published: PublishedError:}]
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.205s
      FAIL
- 2026-11-17 07:21 -05:00 · tarea 10 · verde · caducada · `go test ./pkg/installer -run '^TestHCD039StaleCacheRefresh$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.282s
- 2026-11-17 07:21 -05:00 · tarea 11 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD039Doctor$' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd039_deps_test.go:61:11: undefined: pinsDoctorItems
      pkg/installer/hcd039_deps_test.go:81:10: undefined: pinsDoctorItems
      pkg/installer/hcd039_deps_test.go:93:22: undefined: pinsDoctorItems
      pkg/installer/hcd039_deps_test.go:98:21: undefined: pinsDoctorItems
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 07:21 -05:00 · tarea 11 · verde · caducada · `go test ./pkg/installer -run '^TestHCD039Doctor$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.225s
- 2026-11-17 07:22 -05:00 · tarea 12 · verde · caducada · `go test ./pkg/installer -run '^(TestHCD039DepsUnchanged|TestPinnedDepProbes)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.357s
- 2026-11-17 07:22 -05:00 · tarea 13 · falló · caducada · `go test -race ./... -count=1` (salida 1)
        [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2219729140/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean2219729140/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	59.374s
      FAIL
- 2026-11-17 07:23 -05:00 · tarea 14 · verde · caducada · `go vet ./... && go vet -tags pins ./...` (salida 0)
- 2026-11-17 07:26 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD039Published$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.343s
- 2026-11-17 07:26 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD039Rejected$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.165s
- 2026-11-17 07:26 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD039NoNetworkByDefault$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.162s
- 2026-11-17 07:26 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD039BuildReleaseGate$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.485s
- 2026-11-17 07:26 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run '^TestHCD039SafeExtract$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.156s
- 2026-11-17 07:26 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run '^TestHCD039Contracts$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.158s
- 2026-11-17 07:26 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run '^TestHCD039DepsPublished$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.227s
- 2026-11-17 07:26 -05:00 · tarea 10 · verde · vigente · `go test ./pkg/installer -run '^TestHCD039StaleCacheRefresh$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.228s
- 2026-11-17 07:26 -05:00 · tarea 11 · verde · vigente · `go test ./pkg/installer -run '^TestHCD039Doctor$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.142s
- 2026-11-17 07:26 -05:00 · tarea 12 · verde · vigente · `go test ./pkg/installer -run '^(TestHCD039DepsUnchanged|TestPinnedDepProbes)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.276s
- 2026-11-17 07:26 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD039PinsCheckReport$' -count=1 && go vet -tags pins ./pkg/installer` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.140s
- 2026-11-17 07:26 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run '^TestHCD039ValidateCandidate$' -count=1 && go vet -tags pins ./pkg/installer` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.164s
- 2026-11-17 07:26 -05:00 · tarea 13 · verde · vigente · `go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.218s
      ok  	pi-gomemory/pkg/installer	57.586s
- 2026-11-17 07:27 -05:00 · tarea 14 · verde · vigente · `go vet ./... && go vet -tags pins ./...` (salida 0)

## Siguiente
Cerrada con aprobación sobre la revisión a45823058b11.

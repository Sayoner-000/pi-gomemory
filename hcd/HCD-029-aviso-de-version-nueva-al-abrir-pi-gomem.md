# Aviso de versión nueva al abrir pi-gomemory con pregunta de instalación

ID: HCD-029
Tipo: evolution
Carril: fast
Motivo: Vía rápida: cambio acotado al aviso existente del launcher, sin datos ni contratos públicos; la persona eligió mantenerlo solo en el launcher

## Intención
Cada vez que se abra pi-gomemory, el launcher menciona con datos correctos que hay una versión nueva (del meta-harness o sus dependencias) y pregunta si instalarla, para que todo quede al día. Corrige: (1) deps-status usa la versión instalada guardada en caché en vez de la real, dando avisos falsos tras actualizar; (2) el aviso se basa en la consulta anterior, así que la primera apertura tras una release no la menciona.


Ampliación pedida por la persona tras abrir la unidad: «también dentro de Pi aparezca hay actualizaciones disponibles.. en la parte inferior derecha.. con la interfaz de pi».

Reproducción (2026-11-13): `~/.pi/agent/pi-gomemory-deps.json` del 11-nov registraba `pi-gomemory v1.6.2 → v1.9.0`; `pi-gomemory-setup version` devolvía `v1.12.1`; `pi-gomemory-setup deps-status` imprimía el aviso falso y salía con 10.

## Alcance
Incluido: (1) `DepsStatusFor` (`pkg/installer/deps.go`) vuelve a leer la versión instalada real con el `Current` del probe de cada componente que la caché marca desactualizado y lo descarta si ya está al día, para que el aviso y la pregunta `¿Actualizar ahora? [s/N]` del launcher no mientan tras actualizar por otra vía; (2) indicador en la interfaz de Pi: el pie propio (`extensions/gomemory-status.ts`) muestra `↑ actualizaciones disponibles` al principio de su lado derecho cuando `pi-gomemory-setup deps-status` sale con 10; al iniciar sesión consulta el estado, ejecuta `deps-refresh` en segundo plano (respeta la caché de 24 h) y vuelve a consultar, de modo que una release nueva aparece en la misma apertura sin retrasar el arranque; (3) con el pie propio desactivado (`/tokens`), el indicador se publica con `ctx.ui.setStatus` para que el pie por defecto de Pi también lo muestre.
Excluido: la pregunta del launcher sigue usando la consulta anterior (sin espera de red antes de abrir Pi); no se pregunta ni se instala desde dentro de Pi, el indicador solo informa (la instalación sigue en la pregunta del launcher o en `pi-gomemory-setup deps-update`); sin cambios en `CheckUpdate`, en los probes ni en el formato de la caché.

## ¿Qué pasa si no lo hacemos?
El launcher sigue preguntando por actualizaciones que ya están instaladas (caso reproducido: v1.6.2 → v1.9.0 con v1.12.1 instalada), la persona pierde confianza en el aviso y lo ignora. Dentro de Pi no hay ninguna señal de que existan versiones nuevas, porque la pregunta del launcher desaparece cuando Pi pinta su interfaz.

## Alternativa más barata
Borrar la caché tras cada actualización: corrige solo el caso de `deps-update` (que ya lo hace) y no cubre `install`, el instalador por `curl` ni las actualizaciones hechas con `mem update` o `pi update`. Tampoco añade nada dentro de Pi. Se descarta.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Releer en `DepsStatusFor` (`pkg/installer/deps.go`) la versión instalada real con `probe.Current` y descartar lo que ya está al día → `go test ./pkg/installer -run '^TestHCD029DepsStatusUsesLiveVersion$' -count=1 -v`; observar primero el rojo con `--expect-fail` (caché `v1.6.2 → v1.9.0`, probe que devuelve `v1.12.1`)
- [x] 2. Mostrar `↑ actualizaciones disponibles` al principio del lado derecho del pie propio de `extensions/gomemory-status.ts`, con consulta `deps-status` → `deps-refresh` en segundo plano → `deps-status` al iniciar sesión, y publicarlo con `ctx.ui.setStatus` cuando `/tokens` desactiva el pie propio → `go test ./pkg/installer -run '^TestHCD029StatusExtensionUpdateIndicator$' -count=1 -v`; observar primero el rojo con `--expect-fail`
- [x] 3. Re-ejecutar la reproducción real con el binario del repositorio y la caché obsoleta del 11-nov en un HOME aislado → `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd029-repro.sh`; `deps-status` debe salir con 0 y sin aviso
- [x] 4. Comprobar el indicador en la interfaz real de Pi tras instalar la extensión → observar en Pi, con una caché que marque un componente desactualizado, `↑ actualizaciones disponibles` abajo a la derecha, y su ausencia con la caché al día
- [x] 5. Mantener verde la suite completa → `go vet ./... && go test ./... -count=1`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-13 06:12 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD029DepsStatusUsesLiveVersion$' -count=1 -v` (salida 1)
      === RUN   TestHCD029DepsStatusUsesLiveVersion
          hcd029_deps_test.go:33: pi-gomemory ya está en v1.12.1 y se anuncia v1.6.2 → v1.9.0
      --- FAIL: TestHCD029DepsStatusUsesLiveVersion (0.00s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.342s
      FAIL
- 2026-11-13 06:12 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD029DepsStatusUsesLiveVersion$' -count=1 -v` (salida 0)
      === RUN   TestHCD029DepsStatusUsesLiveVersion
      --- PASS: TestHCD029DepsStatusUsesLiveVersion (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.191s
- 2026-11-13 06:12 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD029StatusExtensionUpdateIndicator$' -count=1 -v` (salida 1)
      === RUN   TestHCD029StatusExtensionUpdateIndicator
          hcd029_status_test.go:65: secuencia de consultas = []; quiero ["pi-gomemory-setup deps-status","pi-gomemory-setup deps-refresh","pi-gomemory-setup deps-status"]
          hcd029_status_test.go:70: el pie no anuncia la actualización:
              ⟦mem⟧ [░░░░░░░░░░░░] 0% (0/200.0k) │ ↑0 ↓0 │ $0.000                                                                                  default
      --- FAIL: TestHCD029StatusExtensionUpdateIndicator (0.16s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.354s
      FAIL
- 2026-11-13 06:13 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD029StatusExtensionUpdateIndicator$' -count=1 -v` (salida 0)
      === RUN   TestHCD029StatusExtensionUpdateIndicator
      --- PASS: TestHCD029StatusExtensionUpdateIndicator (0.14s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.281s
- 2026-11-13 06:13 -05:00 · tarea 3 · verde · vigente · `bash /private/tmp/claude-501/-Users-josegomezj-home-rcw-pi-gomemory/d61fc791-a101-4e12-b1c6-8ea46c0786af/scratchpad/hcd029-repro.sh` (salida 0)
      caché: "name":"pi-gomemory","current":"v1.6.2","latest":"v1.9.0",
      deps-status salió con 0
      ✓ sin aviso falso
- 2026-11-13 06:13 -05:00 · tarea 5 · verde · vigente · `go vet ./... && go test ./... -count=1` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	31.965s
- 2026-11-13 06:16 -05:00 · tarea 4 · manual · vigente
      Observación en Pi real tras pi-gomemory-setup install (binario del repo v1.12.1). Con caché de prueba pi-gomemory v1.12.1 → v1.13.0 (deps-status exit 10), la persona abrió pi-gomemory y confirmó el indicador abajo a la derecha: «funciono». Restaurada la caché real: deps-status exit 0, sin aviso.
- 2026-11-13 06:16 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD029DepsStatusUsesLiveVersion$' -count=1 -v` (salida 0)
      === RUN   TestHCD029DepsStatusUsesLiveVersion
      --- PASS: TestHCD029DepsStatusUsesLiveVersion (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.175s

## Siguiente
Cerrada con aprobación sobre la revisión 2fa1928cdb15.

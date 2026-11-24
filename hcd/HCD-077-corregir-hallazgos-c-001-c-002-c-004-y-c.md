# Corregir hallazgos C-001, C-002, C-004 y C-005 del ACR sobre HCD-076

ID: HCD-077
Tipo: fix
Carril: fast
Motivo: Fix acotado sobre cuatro defectos concretos con reproducción determinista del ACR acr_cd2f4c27-b953-4749-bef3-7c6509862121, sin decisiones abiertas interdependientes

## Intención
Cerrar el hallazgo confirmado C-001 y los tres hallazgos sospechosos accionables (C-004, C-005 y C-002) del ACR sobre HCD-074/HCD-076: ampliar PATH en components --json, consultar Latest aunque Current falle para herramientas ausentes, incluir el tool obligatorio fallido en sync --json y rechazar --only=/--skip= vacíos con código 2.

## Reproducción
Los cuatro defectos detectados en la revisión ACR `acr_cd2f4c27-b953-4749-bef3-7c6509862121` sobre el working tree de HCD-074 + HCD-076 (`409d1b5+dirty`) se reproducen de forma determinista con pruebas unitarias en `pkg/installer` y el paquete raíz:
1. **C-001 (`main.go:244-261`):** invocar `runComponents` con `--json` teniendo un `tool` instalado en `opts.LocalBinDir` pero con `PATH` sin ese directorio devuelve `installed: false` y `current: ""`.
2. **C-004 (`pkg/installer/deps.go:279-291`, `pkg/installer/sync.go:165`):** ejecutar `ForceRefreshDeps` cuando `p.Current` falla (herramienta ausente) devuelve `Latest: ""` sin invocar `p.Latest`, dejando al selector sin última versión (HCD-076 C-09) y la caché sin `Latest` tras instalar.
3. **C-005 (`pkg/installer/sync.go:203-209`):** cuando falla la instalación de un `tool` obligatorio ausente, `Sync` retorna antes de añadir la entrada con `Action: "failed"` a `results`, de modo que `sync --json` omite el componente fallido (`[]` si falla el primero).
4. **C-002 (`pkg/installer/sync.go:59-69`):** `ParseSyncArgs([]string{"--only="})` y `ParseSyncArgs([]string{"--skip= , "})` devuelven `err == nil` y listas vacías en lugar de `SyncUsageError` (código 2).

Comprobación de reproducción (debe fallar antes del cambio):
`go test . ./pkg/installer -run 'TestHCD077'`

## Alcance
Incluido: corrección de C-001, C-004, C-005 y C-002 del ACR sobre HCD-076, más pruebas unitarias y regresión:
- **C-001 (CONFIRMED MEDIUM):** ampliar `PATH` con los directorios conocidos de herramientas (sin crear directorios, mediante `PrependToolDirs(opts)` como `doctor`) antes de invocar `WriteComponentsJSON` en `runComponents` (`main.go`) y restaurar/probar con `PATH` restringido.
- **C-004 (SUSPECT MEDIUM):** en `refreshDeps` (`pkg/installer/deps.go`), consultar `p.Latest(ctx)` aunque `p.Current(ctx)` falle (siguiendo el mismo patrón de `p.Published` en HCD-039 C-25), preservando `Current: ""`, `Outdated() == false` y el error original de `Current` cuando corresponda.
- **C-005 (SUSPECT MEDIUM):** en la rama fatal de `Sync` (`pkg/installer/sync.go`) al fallar la instalación de un `tool` obligatorio ausente, asignar `r.Action = "failed"`, `r.Error = err.Error()` y hacer `results = append(results, r)` antes de retornar el error.
- **C-002 (SUSPECT LOW):** en `ParseSyncArgs` (`pkg/installer/sync.go`), rechazar `--only=` y `--skip=` cuando no contengan ningún nombre válido tras recortar espacios y comas, devolviendo `SyncUsageError` (código de salida 2).
- Pruebas unitarias dedicadas en `main_hcd077_test.go` y `pkg/installer/hcd077_test.go`, y verificación de regresión completa con `go test ./...`.

Excluido: C-003 (decisión vigente de HCD-076), cambios en extensiones/scripts y publicación:
- **C-003 (`install.sh:18-21`):** consultar `PI_GOMEMORY_LATEST_URL` sin `PI_GOMEMORY_VERSION` incluso con `PI_GOMEMORY_BASE_URL` puesta es comportamiento explícitamente decidido en HCD-076 (se necesita la etiqueta para construir el nombre del tarball).
- Cambios en `extensions/gomemory-status.ts`, `install.sh`, `scripts/clean-install-check.sh` o el catálogo de componentes.
- Commit, push o publicación de release.

Referencias normativas y antecedentes:
- Constitución §1.2 (Go, errores con `%w`), §8 (manejo de errores y validación en el borde), §10 (pruebas deterministas), §18 (documento único HCD), §21 (causa raíz, fallar rápido, TDD).
- Playbook HCD: HCD-FLOW-01 (carril `fast`), HCD-DEF-01, HCD-TASK-01, HCD-TDD-01 (rojo antes del verde), HCD-EVIDENCE-01.
- Antecedentes: HCD-076 (catálogo único, `sync`, `components --json`, selector C-09, código 2 en C-11), HCD-039 (C-25: sondeo independiente de `Current`), HCD-062 (directorios conocidos de herramientas), ACR `acr_cd2f4c27-b953-4749-bef3-7c6509862121`.

## ¿Qué pasa si no lo hacemos?
- `components --json` y `/componentes` reportarán falsos negativos (`installed: false`) en shells cuyo `PATH` no incluya `~/.local/bin`, `~/.pi/agent/bin` o `~/.gomemory/bin`, contradiciendo a `sync` y `doctor`.
- El selector de `sync` omitirá la última versión disponible para herramientas ausentes como `herdr` (incumpliendo HCD-076 C-09) y dejará la caché incompleta tras instalarlas.
- `sync --json` devolverá una salida incompleta (o `[]`) cuando falle un `tool` obligatorio, dificultando el diagnóstico en `/componentes` y scripts.
- `--only=` o `--skip=` vacíos actuarán silenciosamente como «sin filtro», reconciliando opcionales no pedidos o abriendo el selector interactivo.

## Alternativa más barata
Corregir únicamente el hallazgo confirmado `C-001` y dejar `C-004`, `C-005` y `C-002` como deuda técnica. Se descarta porque los tres sospechosos fueron verificados directamente en el código fuente con evidencia determinista y sus correcciones son mínimas y localizadas en los mismos archivos de HCD-076.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla (C-001, C-002, C-004 y C-005 reproducidos en rojo y corregidos en verde) → `go test . ./pkg/installer -run 'TestHCD077'`
- [x] 2. C-001: `runComponents --json` amplía PATH con los directorios conocidos de herramientas sin crearlos → `go test . -run 'TestHCD077ComponentsExtendsPath'`
- [x] 3. C-004: `ForceRefreshDeps` consulta `Latest` aunque `Current` falle, preservando `Outdated()==false` y mostrando última versión en selector/caché → `go test ./pkg/installer -run 'TestHCD077RefreshQueriesLatestWhenMissing'`
- [x] 4. C-005: `Sync` incluye la entrada `action:"failed"` del `tool` obligatorio fallido antes de devolver el error fatal → `go test ./pkg/installer -run 'TestHCD077SyncJSONIncludesFatalRequiredFailure'`
- [x] 5. C-002: `ParseSyncArgs` rechaza `--only=` y `--skip=` vacíos (o solo comas/espacios) con `SyncUsageError` → `go test ./pkg/installer -run 'TestHCD077ParseSyncArgsRejectsEmptyFilter'`
- [x] 6. Regresión completa de toda la suite Go sin roturas → `go test ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-24 05:40 -05:00 · tarea 1 · rojo observado · caducada · `go test . ./pkg/installer -run 'TestHCD077'` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd077_test.go:47:19: cannot use func(h *toolHooks) {…} (value of type func(h *toolHooks)) as toolHooks value in argument to withToolHooks
      pkg/installer/hcd077_test.go:48:5: h.Run undefined (type *toolHooks has no field or method Run)
      --- FAIL: TestHCD077ComponentsExtendsPath (0.01s)
          main_hcd077_test.go:73: herdr en LocalBinDir fuera de PATH = {Name:herdr Kind:tool Required:false Installed:false Current: Latest: Outdated:false Error:}, quiero installed=true current=0.9.5
      FAIL
      FAIL	pi-gomemory	0.383s
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-24 05:41 -05:00 · tarea 1 · rojo observado · caducada · `go test . ./pkg/installer -run 'TestHCD077'` (salida 1)
      --- FAIL: TestHCD077ComponentsExtendsPath (0.01s)
          main_hcd077_test.go:73: herdr en LocalBinDir fuera de PATH = {Name:herdr Kind:tool Required:false Installed:false Current: Latest: Outdated:false Error:}, quiero installed=true current=0.9.5
      FAIL
      FAIL	pi-gomemory	0.151s
      --- FAIL: TestHCD077RefreshQueriesLatestWhenMissing (0.00s)
          hcd077_test.go:28: Latest = "", quiero v0.9.5 (state={Name:herdr Current: Latest: Update:[] Manual: Error:no está en PATH Published: PublishedError:})
      --- FAIL: TestHCD077SyncJSONIncludesFatalRequiredFailure (0.01s)
          hcd077_test.go:51: esperaba error fatal al fallar instalación de pi
      --- FAIL: TestHCD077ParseSyncArgsRejectsEmptyFilter (0.00s)
          hcd077_test.go:81: ParseSyncArgs([--only=]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:81: ParseSyncArgs([--only= , , ]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:81: ParseSyncArgs([--skip=]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:81: ParseSyncArgs([--skip= , ]): err=<nil>, quiero *SyncUsageError
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.198s
      FAIL
- 2026-11-24 05:41 -05:00 · tarea 1 · rojo observado · caducada · `go test . ./pkg/installer -run 'TestHCD077'` (salida 1)
      --- FAIL: TestHCD077ComponentsExtendsPath (0.01s)
          main_hcd077_test.go:73: herdr en LocalBinDir fuera de PATH = {Name:herdr Kind:tool Required:false Installed:false Current: Latest: Outdated:false Error:}, quiero installed=true current=0.9.5
      FAIL
      FAIL	pi-gomemory	0.158s
      --- FAIL: TestHCD077RefreshQueriesLatestWhenMissing (0.00s)
          hcd077_test.go:28: Latest = "", quiero v0.9.5 (state={Name:herdr Current: Latest: Update:[] Manual: Error:no está en PATH Published: PublishedError:})
      --- FAIL: TestHCD077SyncJSONIncludesFatalRequiredFailure (0.01s)
          hcd077_test.go:54: esperaba error fatal al fallar instalación de pi
      --- FAIL: TestHCD077ParseSyncArgsRejectsEmptyFilter (0.00s)
          hcd077_test.go:84: ParseSyncArgs([--only=]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:84: ParseSyncArgs([--only= , , ]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:84: ParseSyncArgs([--skip=]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:84: ParseSyncArgs([--skip= , ]): err=<nil>, quiero *SyncUsageError
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.211s
      FAIL
- 2026-11-24 05:42 -05:00 · tarea 1 · rojo observado · caducada · `go test . ./pkg/installer -run 'TestHCD077'` (salida 1)
      --- FAIL: TestHCD077ComponentsExtendsPath (0.01s)
          main_hcd077_test.go:73: herdr en LocalBinDir fuera de PATH = {Name:herdr Kind:tool Required:false Installed:false Current: Latest: Outdated:false Error:}, quiero installed=true current=0.9.5
      FAIL
      FAIL	pi-gomemory	0.164s
      --- FAIL: TestHCD077RefreshQueriesLatestWhenMissing (0.00s)
          hcd077_test.go:28: Latest = "", quiero v0.9.5 (state={Name:herdr Current: Latest: Update:[] Manual: Error:no está en PATH Published: PublishedError:})
      --- FAIL: TestHCD077SyncJSONIncludesFatalRequiredFailure (0.01s)
          hcd077_test.go:56: esperaba error fatal al fallar instalación de pi
      --- FAIL: TestHCD077ParseSyncArgsRejectsEmptyFilter (0.00s)
          hcd077_test.go:86: ParseSyncArgs([--only=]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:86: ParseSyncArgs([--only= , , ]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:86: ParseSyncArgs([--skip=]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:86: ParseSyncArgs([--skip= , ]): err=<nil>, quiero *SyncUsageError
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.209s
      FAIL
- 2026-11-24 05:42 -05:00 · tarea 1 · rojo observado · caducada · `go test . ./pkg/installer -run 'TestHCD077'` (salida 1)
      --- FAIL: TestHCD077ComponentsExtendsPath (0.01s)
          main_hcd077_test.go:73: herdr en LocalBinDir fuera de PATH = {Name:herdr Kind:tool Required:false Installed:false Current: Latest: Outdated:false Error:}, quiero installed=true current=0.9.5
      FAIL
      FAIL	pi-gomemory	0.160s
      --- FAIL: TestHCD077RefreshQueriesLatestWhenMissing (0.00s)
          hcd077_test.go:29: Latest = "", quiero v0.9.5 (state={Name:herdr Current: Latest: Update:[] Manual: Error:no está en PATH Published: PublishedError:})
      --- FAIL: TestHCD077SyncJSONIncludesFatalRequiredFailure (0.01s)
          hcd077_test.go:62: esperaba error fatal al fallar instalación de pi
      --- FAIL: TestHCD077ParseSyncArgsRejectsEmptyFilter (0.00s)
          hcd077_test.go:92: ParseSyncArgs([--only=]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:92: ParseSyncArgs([--only= , , ]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:92: ParseSyncArgs([--skip=]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:92: ParseSyncArgs([--skip= , ]): err=<nil>, quiero *SyncUsageError
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.212s
      FAIL
- 2026-11-24 05:42 -05:00 · tarea 1 · rojo observado · caducada · `go test . ./pkg/installer -run 'TestHCD077'` (salida 1)
      --- FAIL: TestHCD077ComponentsExtendsPath (0.01s)
          main_hcd077_test.go:73: herdr en LocalBinDir fuera de PATH = {Name:herdr Kind:tool Required:false Installed:false Current: Latest: Outdated:false Error:}, quiero installed=true current=0.9.5
      FAIL
      FAIL	pi-gomemory	0.189s
      --- FAIL: TestHCD077RefreshQueriesLatestWhenMissing (0.00s)
          hcd077_test.go:29: Latest = "", quiero v0.9.5 (state={Name:herdr Current: Latest: Update:[] Manual: Error:no está en PATH Published: PublishedError:})
      --- FAIL: TestHCD077SyncJSONIncludesFatalRequiredFailure (0.00s)
          hcd077_test.go:65: results vacío tras fallo fatal de pi; sync --json omitió el componente fallido
      --- FAIL: TestHCD077ParseSyncArgsRejectsEmptyFilter (0.00s)
          hcd077_test.go:92: ParseSyncArgs([--only=]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:92: ParseSyncArgs([--only= , , ]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:92: ParseSyncArgs([--skip=]): err=<nil>, quiero *SyncUsageError
          hcd077_test.go:92: ParseSyncArgs([--skip= , ]): err=<nil>, quiero *SyncUsageError
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.212s
      FAIL
- 2026-11-24 05:43 -05:00 · tarea 1 · verde · vigente · `go test . ./pkg/installer -run 'TestHCD077'` (salida 0)
      ok  	pi-gomemory	0.297s
      ok  	pi-gomemory/pkg/installer	0.206s
- 2026-11-24 05:43 -05:00 · tarea 2 · verde · vigente · `go test . -run 'TestHCD077ComponentsExtendsPath'` (salida 0)
      ok  	pi-gomemory	0.245s
- 2026-11-24 05:43 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run 'TestHCD077RefreshQueriesLatestWhenMissing'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.184s
- 2026-11-24 05:43 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run 'TestHCD077SyncJSONIncludesFatalRequiredFailure'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.191s
- 2026-11-24 05:43 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run 'TestHCD077ParseSyncArgsRejectsEmptyFilter'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.168s
- 2026-11-24 05:43 -05:00 · tarea 6 · verde · vigente · `go test ./...` (salida 0)
      ok  	pi-gomemory	2.113s
      ok  	pi-gomemory/pkg/installer	43.754s

## Siguiente
Cerrada con aprobación sobre la revisión 5fa8fb2343ae.

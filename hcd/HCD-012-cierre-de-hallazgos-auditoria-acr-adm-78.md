# Cierre de hallazgos auditoría ACR/ADM 781e39d

ID: HCD-012
Tipo: fix
Carril: full
Motivo: Riesgo de seguridad alto (ejecución remota curl|bash y path traversal) y decisión abierta sobre cómo actualizar codebase-memory-mcp

## Intención
Cerrar C-001..C-006 de docs/audits/2026-11-11-acr-adm-review.md (ACR acr_d173952c-f8d4-403e-8864-ca94074098b0) con pruebas que demuestren cada cierre, sin cambiar el comportamiento del resto del instalador ni de las extensiones.

## Reproducción
Sobre HEAD 781e39d:
- C-001: `pkg/installer/deps.go:219` define `Update: bash -c "curl … main/install.sh | bash"` para codebase-memory-mcp; `pi-gomemory-setup deps-update` lo ejecuta sin verificar integridad. Además, `deps-update` ejecuta el comando guardado en la caché `pi-gomemory-deps.json` (`deps.go:247,265`; `main.go:95-100`), así que una caché previa conserva el `curl | bash` y cualquier comando escrito en ese JSON se ejecuta; y `main.go:106` imprime «✓ Dependencias actualizadas» aunque un componente se haya saltado.
- C-002: `extensions/gomemory-splash.ts:20` hace `resolve(ctx.cwd, params.document)` y lee el archivo sin comprobar que quede dentro de `ctx.cwd`; `document: "../../../etc/passwd"` se lee y se pasa a Splash. Un enlace simbólico dentro del repo que apunte fuera también escapa.
- C-003: `os.WriteFile` directo en `pkg/installer/extension.go:36`, `pkg/installer/instructions.go:69`, `pkg/installer/theme.go:22`, `pkg/installer/harness.go:94`, `pkg/installer/skills.go:113/116` y la caché de `pkg/installer/update.go:148`, en lugar de `writeFileAtomic` (`pkg/installer/jsonfile.go:46`, temporal `.<base>.tmp-*` + `os.Chmod(perm)` en `:64` + rename).
- C-004: `extensions/gomemory-sidebar.ts:52-78` devuelve `status: "Connected"` para todo servidor declarado (y para una lista fija si falta `mcp.json`).
- C-005: `pkg/installer/update.go:312` `gzip.NewReader(strings.NewReader(string(tarball)))` copia el tarball (hasta 200 MiB).
- C-006: referencia Pi 1.5.2 instalado (`~/.pi/agent/install/current-version` = 1.5.2): `~/.pi/agent/install/releases/1.5.2/node_modules/@earendil-works/pi-coding-agent/dist/core/tools/` contiene bash, powershell, edit, write, read, grep, find y ls, y `powershell.js:29` declara `name: "powershell"`. `extensions/gomemory-plan.ts:101-106` bloquea entera cada herramienta de `gomemory-plan-patterns.json` (`tools: ["edit","write"]`) y filtra `bash` por patrón; `powershell` pasa sin filtro.
- Ninguna prueba ejecuta la lógica de las extensiones: las que las tocan (`hcd002_test.go`, `hcd010_test.go`, `hcd008_test.go`) solo buscan texto, y `hcd011_*_test.go` solo comprueban que se instalan o retiran. `hcd010_test.go:15` exige además los literales `Connected`/`Disconnected`.

## Alcance
Incluido: los seis hallazgos C-001..C-006 en los archivos citados; la lógica de `deps-status`/`deps-update` pasa de `main.go` a funciones de `pkg/installer` (`main.go` solo las llama) para poder probarla; pruebas de comportamiento de las extensiones ejecutadas con `node` ≥ 22.18 (stripping de tipos nativo, stubs de los paquetes `@earendil-works/*`), que fallan —no se omiten— si falta `node`; ajustar `hcd010_test.go` al nuevo estado del sidebar y `hcd002_test.go:138` (`TestPlanPatterns`) a `edit,write,powershell`; `docs/usage.md` (estado del sidebar) y la actualización del estado de cierre en el informe de auditoría, con el riesgo residual de Splash.
Excluido: un actualizador propio con checksum para codebase-memory-mcp; consultar el estado real de los MCP (`pi mcp list`) desde el sidebar; filtrar comandos PowerShell por patrón (se bloquea entera); confinar las herramientas read/grep/find/ls de la sesión Splash, que aceptan rutas absolutas (riesgo residual documentado en el informe, unidad aparte); las escrituras de `uninstall.go` (marca y log; no las reportó la auditoría). La autoactualización de pi-gomemory ya verifica el tarball con `SHA256SUMS` (`update.go`), no requiere cambio.

## ¿Qué pasa si no lo hacemos?
Un compromiso del repositorio o de la red de DeusData ejecuta código arbitrario con los privilegios del usuario al aceptar la actualización; un prompt malicioso puede exfiltrar claves o archivos privados al proveedor del modelo vía Splash; un corte durante la instalación deja AGENTS.md, el launcher o extensiones truncados; el sidebar afirma conexiones que no existen; el modo plan no protege en Windows.

## Alternativa más barata
Documentar los riesgos sin corregirlos. Se descarta para C-001 y C-002 (severidad Alta) y el resto son cambios de pocas líneas con prueba directa.

## Historias
### US1. Actualización sin ejecución remota ciega
Como usuario de pi-gomemory, quiero que la actualización de dependencias nunca ejecute un script descargado sin verificar, para no exponer mi equipo a un compromiso de la cadena de suministro.
- C-01: Dado que codebase-memory-mcp está desactualizado, cuando ejecuto `deps-update`, entonces no se ejecuta ningún comando para él, se imprime «actualízalo manualmente: https://github.com/DeusData/codebase-memory-mcp/releases/latest» y la salida final no dice «✓ Dependencias actualizadas» sino que lista los componentes manuales pendientes (código de salida 0).
- C-02: Dado `DefaultDepProbes`, cuando se inspeccionan sus comandos, entonces ninguno contiene `curl`, `| bash` ni `| sh`.
- C-09: Dada una caché `pi-gomemory-deps.json` cuyo `update` contiene un comando arbitrario (p. ej. `curl … | bash`), cuando se ejecutan `deps-status` o `deps-update`, entonces el comando mostrado o ejecutado es el del probe actual por nombre; un nombre sin probe no se muestra ni se ejecuta (si solo hay nombres sin probe, `deps-status` sale con 0 sin aviso y `deps-update` dice «✓ El meta-harness y sus dependencias están al día.»). Contrato: `installer.RunDepsUpdate(cachePath, probes, stdout, stderr) int` devuelve el código de salida y `installer.DepsStatusFor(cachePath, probes)` la lista filtrada; `main.go` solo los llama y hace `os.Exit`.

### US2. Splash confinado al repositorio
Como usuario, quiero que `splash_review` solo lea documentos del repositorio, para que un agente no pueda enviar archivos privados al modelo.
- C-03: Dado `document` que resuelve fuera de `ctx.cwd` (por `..`, ruta absoluta o enlace simbólico), cuando se invoca `splash_review`, entonces falla antes de leer el archivo y de contactar al proveedor.
- C-04: Dado un documento dentro del repositorio (incluido un nombre que empieza por `..`, p. ej. `..notas.md`), cuando se invoca, entonces se acepta.

### US3. Escrituras del instalador todo-o-nada
Como usuario, quiero que el instalador escriba sus archivos de forma atómica, para que una interrupción no deje archivos truncados.
- C-05: Dado el paquete `pkg/installer`, cuando se buscan llamadas a `os.WriteFile` en sus `.go` que no son `_test.go` ni `uninstall.go`, entonces no hay ninguna; y tras `InstallHarness`, `InstallTheme`, `InstallTUIExtension`, `InstallInstructions` e `InstallSkills` el contenido es el esperado, el launcher tiene modo 0755 y no queda ningún temporal `.*.tmp-*` en los directorios destino.

### US4. Sidebar honesto
Como usuario, quiero que el sidebar no afirme conexiones no comprobadas, para no confiar en un estado falso.
- C-06: Dado `mcp.json` con servidores, cuando se abre el sidebar, entonces cada servidor aparece como `Configured` y el código de la extensión ya no contiene `Connected`; dado que falta `mcp.json` o no es JSON válido, entonces la lista de servidores está vacía (no se inventa ninguno).

### US5. Correcciones menores
Como mantenedor, quiero cerrar los hallazgos bajos, para no arrastrarlos.
- C-07: Dado un tarball válido, cuando `extractBinary` lo descomprime, entonces devuelve el binario (las pruebas de update siguen verdes) y el código no contiene `string(tarball)`.
- C-08: Dado el modo plan activo, cuando el manejador `tool_call` recibe `toolName: "powershell"`, entonces devuelve `block: true`; dado el modo inactivo, no bloquea.

## Aclaraciones
- [x] Q1. ¿Cómo se actualiza codebase-memory-mcp? → Solo avisar: sin comando automático; se muestra la instrucción manual con la URL del release.
- [x] Q2. ¿Qué estado muestra el sidebar por servidor MCP? → "Configured".
- [x] Q3. Splash (B4): las herramientas de la sesión Splash aceptan rutas absolutas; ¿se confinan aquí? → No: excluir y documentar como riesgo residual en el informe.
- [x] Q4. Si falta `node`, ¿se omiten las pruebas de extensiones? → No: fallan, para que un verde nunca signifique «no ejecutado» (decisión derivada de la segunda revisión Splash; `node` 26 disponible).

## Slices
### VS-01. Hallazgos altos (C-001, C-002)
Tareas: 1, 2
Criterios: C-01, C-02, C-09, C-03, C-04

### VS-02. Medio y bajos (C-003..C-006) e informe
Tareas: 3, 4, 5, 6, 7
Criterios: C-05, C-06, C-07, C-08

## Checklist
- [x] 1. Sustituir el `curl | bash` de codebase-memory-mcp por un aviso manual; tomar los comandos del probe por nombre (no de la caché) en `deps-status`/`deps-update`; no declarar «actualizado» con manuales pendientes → `go test ./pkg/installer -run TestHCD012Deps`
- [x] 2. Confinar `splash_review` a `ctx.cwd` con `realpath` y comparación por separador, antes de leer; prueba ejecutada con node (`..`, absoluta, enlace simbólico, `..notas.md`) → `go test ./pkg/installer -run TestHCD012Splash`
- [x] 3. Usar `writeFileAtomic` en extension, instructions, theme, harness, skills y la caché de update → `go test ./pkg/installer -run TestHCD012Atomic`
- [x] 4. Mostrar `Configured` en el sidebar y lista vacía sin `mcp.json`; prueba ejecutada con node; actualizar `docs/usage.md` → `go test ./pkg/installer -run 'TestHCD012Sidebar|TestHCD010'`
- [x] 5. Usar `bytes.NewReader` en `extractBinary` → `go test ./pkg/installer -run 'TestHCD012Extract|TestUpdate'`
- [x] 6. Añadir `powershell` a `tools` de `gomemory-plan-patterns.json`; prueba ejecutada con node del manejador `tool_call` → `go test ./pkg/installer -run 'TestHCD012PlanPowershell|TestPlan'`
- [x] 7. Marcar C-001..C-006 como cerrados en el informe y pasar la suite → `go vet ./... && go test ./... && grep -q 'Cerrado en HCD-012' docs/audits/2026-11-11-acr-adm-review.md && [ "$(grep -c 'Cerrado en HCD-012' docs/audits/2026-11-11-acr-adm-review.md)" -ge 6 ] && grep -q 'Riesgo residual' docs/audits/2026-11-11-acr-adm-review.md`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-11 07:35 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD012Deps` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd012_deps_test.go:20:84: p.Manual undefined (type DepProbe has no field or method Manual)
      pkg/installer/hcd012_deps_test.go:42:8: undefined: DepsStatusFor
      pkg/installer/hcd012_deps_test.go:47:13: undefined: RunDepsUpdate
      pkg/installer/hcd012_deps_test.go:55:10: undefined: DepsStatusFor
      pkg/installer/hcd012_deps_test.go:59:13: undefined: RunDepsUpdate
      pkg/installer/hcd012_deps_test.go:67:35: unknown field Manual in struct literal of type DepProbe
      pkg/installer/hcd012_deps_test.go:69:10: undefined: RunDepsUpdate
      pkg/installer/hcd012_deps_test.go:80:27: undefined: DepsStatusFor
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-11 07:36 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD012Deps` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.172s
- 2026-11-11 07:36 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD012Splash` (salida 1)
      --- FAIL: TestHCD012Splash (0.12s)
          hcd012_splash_test.go:42: FAIL ../secret.txt => ENOENT: no such file or directory, open '/private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD012Splash2489583591/001/gomemory-splash-prompt.md' | /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/splash-G0GdqQ/secret.txt => ENOENT: no such file or directory, open '/private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD012Splash2489583591/001/gomemory-splash-prompt.md' | link.md => ENOENT: no such file or directory, open '/private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD012Splash2489583591/001/gomemory-splash-prompt.md' | ..notas.md => ENOENT: no such file or directory, open '/private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD012Splash2489583591/001/gomemory-splash-prompt.md' | doc.md => ENOENT: no such file or directory, open '/private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD012Splash2489583591/001/gomemory-splash-prompt.md'
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.292s
      FAIL
- 2026-11-11 07:37 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD012Splash` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.282s
- 2026-11-11 07:37 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD012Atomic` (salida 1)
      --- FAIL: TestHCD012AtomicNoWriteFile (0.00s)
          hcd012_atomic_test.go:28: extension.go usa os.WriteFile
          hcd012_atomic_test.go:28: harness.go usa os.WriteFile
          hcd012_atomic_test.go:28: instructions.go usa os.WriteFile
          hcd012_atomic_test.go:28: skills.go usa os.WriteFile
          hcd012_atomic_test.go:28: skills.go usa os.WriteFile
          hcd012_atomic_test.go:28: theme.go usa os.WriteFile
          hcd012_atomic_test.go:28: update.go usa os.WriteFile
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.188s
      FAIL
- 2026-11-11 07:38 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD012Sidebar|TestHCD010'` (salida 1)
      --- FAIL: TestHCD012Sidebar (0.14s)
          hcd012_sidebar_test.go:11: gomemory-sidebar.ts todavía contiene Connected
          hcd012_sidebar_test.go:30: FAIL sin mcp.json no está vacío | JSON inválido no está vacío | [{"name":"a","status":"Connected"},{"name":"b","status":"Connected"}]
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.324s
      FAIL
- 2026-11-11 07:38 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD012Extract|TestUpdate'` (salida 1)
      --- FAIL: TestHCD012Extract (0.00s)
          hcd012_extract_test.go:11: extractBinary todavía convierte el tarball a string
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.193s
      FAIL
- 2026-11-11 07:39 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run 'TestHCD012PlanPowershell|TestPlan'` (salida 1)
      --- FAIL: TestPlanPatterns (0.00s)
          hcd002_test.go:139: tools = [edit write]
      --- FAIL: TestHCD012PlanPowershell (0.13s)
          hcd012_plan_test.go:28: FAIL powershell no bloqueado: undefined
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.331s
      FAIL
- 2026-11-11 07:39 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestHCD012Deps` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.183s
- 2026-11-11 07:39 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD012Splash` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.259s
- 2026-11-11 07:39 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD012Atomic` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.381s
- 2026-11-11 07:39 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run 'TestHCD012Sidebar|TestHCD010'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.279s
- 2026-11-11 07:39 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run 'TestHCD012Extract|TestUpdate'` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.164s
- 2026-11-11 07:39 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run 'TestHCD012PlanPowershell|TestPlan'` (salida 0)
      ok  	pi-gomemory/pkg/installer	(cached)
- 2026-11-11 07:39 -05:00 · tarea 7 · verde · vigente · `go vet ./... && go test ./... && grep -q 'Cerrado en HCD-012' docs/audits/2026-11-11-acr-adm-review.md && [ "$(grep -c 'Cerrado en HCD-012' docs/audits/2026-11-11-acr-adm-review.md)" -ge 6 ] && grep -q 'Riesgo residual' docs/audits/2026-11-11-acr-adm-review.md` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	7.068s

## Siguiente
Cerrada con aprobación sobre la revisión dda444d964f1.

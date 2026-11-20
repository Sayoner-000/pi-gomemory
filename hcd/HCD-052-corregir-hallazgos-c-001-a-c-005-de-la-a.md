# Corregir hallazgos C-001 a C-005 de la auditoría ACR/ADM v1.16.0

ID: HCD-052
Tipo: fix
Carril: fast
Motivo: Fixes defensivos con reproducción verificada en docs/audits/2026-11-19-acr-adm-v1.16.0.md siguiendo el modelo de propiedad existente (managedMCPProfiles), sin decisiones abiertas interdependientes

## Intención
Corregir los cinco hallazgos confirmados en `docs/audits/2026-11-19-acr-adm-v1.16.0.md` (revisión `d8f2186`, ACR `acr_a4416dad-6086-4927-9b66-3f24cad4cd92`): preservar configuraciones MCP personalizadas en `lite on/off` (C-003), garantizar consistencia atómica del estado `lite` ante errores de lectura/escritura (C-001), componer `lite` con `codemode`, `install`, `update` y `mcp` desde un perfil centralizado (C-002), evitar falsos verdes en `/tdd` ante menciones en texto o errores externos (C-004) y evitar falsos bloqueos en `/tasks` ante menciones citadas o comentadas de `mem hcd verify` (C-005).

## Normas aplicables
Constitución §1.2 (Go), §10 (pruebas deterministas con `HOME` temporal y datos sintéticos), §16 (calidad), §18 (HCD) y §21 (simplicidad y causa raíz); Playbook `HCD-FLOW-01`, `HCD-DEF-01`, `HCD-TASK-01`, `HCD-TDD-01` y `HCD-EVIDENCE-01`.

## Reproducción
Origen: `docs/audits/2026-11-19-acr-adm-v1.16.0.md` sobre `d8f2186`, reproducido con datos sintéticos y `HOME` temporal:
1. **C-003 (`pkg/installer/lite.go`):** con una entrada `codegraph` personalizada (`command: "custom-graph"`), ejecutar `SetLiteMode(opts, true)` y luego `SetLiteMode(opts, false)` borra la entrada personalizada y la reemplaza por el preset estándar.
2. **C-001 (`pkg/installer/lite.go`):** con `mcpServers` inválido (lista en lugar de objeto) o error de escritura, `SetLiteMode(opts, true)` falla pero ya dejó `enabled: true` persistido en estado `lite`.
3. **C-002 (`pkg/installer/codemode.go`, `pkg/installer/lite.go`, `pkg/installer/mcp.go`, `main.go`):** tras `SetLiteMode(opts, true)`, ejecutar `SetCodemodePreference(opts, true)` o `applyCodemodeDefault` vuelve a escribir `codegraph` y `codebase-memory` contradiciendo la preferencia `lite` activa.
4. **C-004 (`extensions/gomemory-tdd.ts`):** `ingestBashTestResult` con `printf 'go test ./...\n'` marca `backend` como `passed`; además, con `isError: true` y un JSON HCD con `evidence.passed: true`, marca `data` como `passed`.
5. **C-005 (`extensions/gomemory-tasks.ts`):** `isBundledHcdVerifyCommand` sobre `printf 'mem hcd verify ... mem hcd verify\n'` devuelve motivo de bloqueo aunque no haya invocaciones efectivas.

## Alcance
Incluido: correcciones defensivas y pruebas de regresión en `pkg/installer/lite.go`, `pkg/installer/mcp.go`, `pkg/installer/codemode.go`, `main.go`, `extensions/gomemory-tdd.ts`, `extensions/gomemory-tasks.ts` y `pkg/installer/hcd052_test.go` para C-003, C-001, C-002, C-004 y C-005 usando datos sintéticos y `HOME` temporales.
Excluido: C-006 (política de confianza de runners locales en `/tdd`), C-007 (guard temporal de Antigravity), modificaciones al informe `docs/audits/2026-11-19-acr-adm-v1.16.0.md`, credenciales reales, pruebas de explotación, instalación real, commit, push, release y cierre HCD sin autorización separada.

## ¿Qué pasa si no lo hacemos?
`lite on/off` destruye definiciones MCP personalizadas del usuario (severidad alta), un fallo de escritura deja la preferencia `lite` desincronizada, `codemode`/`install`/`update` rompen el modo `lite`, `/tdd` puede reportar falsos verdes sobre menciones en texto o errores externos y `/tasks` bloquea comandos bash legítimos que solo citan `mem hcd verify`.

## Alternativa más barata
Desaconsejar el uso de `lite on/off` y no confiar en `/tdd` ni `/tasks` en bash; no resuelve la pérdida silenciosa de configuración ni los falsos resultados ya publicados en `v1.16.0`.

## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->

## Checklist
- [x] 1. La reproducción ya no falla: preservar configuraciones MCP personalizadas en `lite on/off` (C-003), garantizar atomicidad de `lite` ante fallos (C-001) y componer `lite` con `codemode`/`install`/`update`/`mcp` (C-002) → `go test ./pkg/installer -run TestHCD052LiteAndMCP -v`
- [x] 2. Corregir C-004 en `extensions/gomemory-tdd.ts`: aceptar solo invocaciones inequívocas, distinguir ejecuciones parciales, conservar la suite ante coincidencias ambiguas y hacer prevalecer `isError` sobre evidencia contradictoria → `go test ./pkg/installer -run TestHCD052TDDBashIngest -v`
- [x] 3. Corregir C-005 en `extensions/gomemory-tasks.ts`: contar invocaciones efectivas de `mem hcd verify` ignorando texto entre comillas y comentarios, manteniendo el bloqueo ante agrupaciones reales → `go test ./pkg/installer -run TestHCD052TasksVerifyGuard -v`
- [x] 4. Validar análisis estático y suite completa con detector de carreras sin regresiones → `go vet ./... && go test -race ./...`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-19 19:48 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD052LiteAndMCP -v` (salida 1)
                  }
      === RUN   TestHCD052LiteAndMCP/C-002/install/update_(applyCodemodeDefault)_respeta_lite_en_direct_y_codemode
          hcd052_test.go:317: C-002: applyCodemodeDefault reintrodujo codegraph con lite activo
      --- FAIL: TestHCD052LiteAndMCP (0.17s)
          --- FAIL: TestHCD052LiteAndMCP/C-003 (0.06s)
              --- PASS: TestHCD052LiteAndMCP/C-003/ida_y_vuelta_estandar_conserva_entrada_ajena (0.03s)
              --- FAIL: TestHCD052LiteAndMCP/C-003/preserva_definiciones_personalizadas_de_codegraph_y_gomemory (0.02s)
              --- FAIL: TestHCD052LiteAndMCP/C-003/no_pisa_ediciones_manuales_hechas_entre_lite_on_y_lite_off (0.02s)
          --- FAIL: TestHCD052LiteAndMCP/C-001 (0.04s)
              --- FAIL: TestHCD052LiteAndMCP/C-001/mcpServers_invalido_no_persiste_lite_on (0.00s)
              --- FAIL: TestHCD052LiteAndMCP/C-001/error_en_lite_off_conserva_lite_on_previo (0.02s)
              --- PASS: TestHCD052LiteAndMCP/C-001/fallo_al_escribir_estado_restaura_mcp.json (0.01s)
              --- FAIL: TestHCD052LiteAndMCP/C-001/fallo_al_escribir_mcp.json_conserva_estado_anterior (0.01s)
          --- FAIL: TestHCD052LiteAndMCP/C-002 (0.08s)
              --- FAIL: TestHCD052LiteAndMCP/C-002/lite_on_+_codemode_on_mantiene_solo_gomemory_en_codemode (0.03s)
              --- FAIL: TestHCD052LiteAndMCP/C-002/codemode_on_+_lite_on_+_codemode_off_mantiene_lite (0.04s)
              --- FAIL: TestHCD052LiteAndMCP/C-002/install/update_(applyCodemodeDefault)_respeta_lite_en_direct_y_codemode (0.01s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.543s
      FAIL
- 2026-11-19 19:48 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD052TasksVerifyGuard -v` (salida 1)
              + 'Ejecuta un solo `mem hcd verify` por llamada bash para permitir refresco en vivo y entrega de steering.'
              - null
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD052TasksVerifyGuard1857298586/001/test.mjs:14:9
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5) {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: 'Ejecuta un solo `mem hcd verify` por llamada bash para permitir refresco en vivo y entrega de steering.',
                expected: null,
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      --- FAIL: TestHCD052TasksVerifyGuard (0.26s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.423s
      FAIL
- 2026-11-19 19:48 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run TestHCD052TDDBashIngest -v` (salida 1)
              + }
              - null
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD052TDDBashIngest3855225924/001/test.mjs:15:9
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5) {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: { suiteId: 'backend', parser: 'exit-code', name: 'go test (bash)' },
                expected: null,
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      --- FAIL: TestHCD052TDDBashIngest (0.27s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.435s
      FAIL
- 2026-11-19 19:50 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run TestHCD052LiteAndMCP -v` (salida 0)
      === RUN   TestHCD052LiteAndMCP/C-002
      === RUN   TestHCD052LiteAndMCP/C-002/lite_on_+_codemode_on_mantiene_solo_gomemory_en_codemode
      === RUN   TestHCD052LiteAndMCP/C-002/codemode_on_+_lite_on_+_codemode_off_mantiene_lite
      === RUN   TestHCD052LiteAndMCP/C-002/install/update_(applyCodemodeDefault)_respeta_lite_en_direct_y_codemode
      --- PASS: TestHCD052LiteAndMCP (0.21s)
          --- PASS: TestHCD052LiteAndMCP/C-003 (0.07s)
              --- PASS: TestHCD052LiteAndMCP/C-003/ida_y_vuelta_estandar_conserva_entrada_ajena (0.03s)
              --- PASS: TestHCD052LiteAndMCP/C-003/preserva_definiciones_personalizadas_de_codegraph_y_gomemory (0.02s)
              --- PASS: TestHCD052LiteAndMCP/C-003/no_pisa_ediciones_manuales_hechas_entre_lite_on_y_lite_off (0.02s)
          --- PASS: TestHCD052LiteAndMCP/C-001 (0.03s)
              --- PASS: TestHCD052LiteAndMCP/C-001/mcpServers_invalido_no_persiste_lite_on (0.00s)
              --- PASS: TestHCD052LiteAndMCP/C-001/error_en_lite_off_conserva_lite_on_previo (0.01s)
              --- PASS: TestHCD052LiteAndMCP/C-001/fallo_al_escribir_estado_restaura_mcp.json (0.01s)
              --- PASS: TestHCD052LiteAndMCP/C-001/fallo_al_escribir_mcp.json_conserva_estado_anterior (0.01s)
          --- PASS: TestHCD052LiteAndMCP/C-002 (0.11s)
              --- PASS: TestHCD052LiteAndMCP/C-002/lite_on_+_codemode_on_mantiene_solo_gomemory_en_codemode (0.03s)
              --- PASS: TestHCD052LiteAndMCP/C-002/codemode_on_+_lite_on_+_codemode_off_mantiene_lite (0.03s)
              --- PASS: TestHCD052LiteAndMCP/C-002/install/update_(applyCodemodeDefault)_respeta_lite_en_direct_y_codemode (0.05s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.483s
- 2026-11-19 19:51 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run TestHCD052TDDBashIngest -v` (salida 0)
      === RUN   TestHCD052TDDBashIngest
          hcd052_test.go:436: 
      --- PASS: TestHCD052TDDBashIngest (0.15s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.398s
- 2026-11-19 19:52 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run TestHCD052TasksVerifyGuard -v` (salida 0)
      === RUN   TestHCD052TasksVerifyGuard
          hcd052_test.go:468: 
      --- PASS: TestHCD052TasksVerifyGuard (0.13s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.301s
- 2026-11-19 19:53 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run TestHCD052LiteAndMCP -v` (salida 0)
      === RUN   TestHCD052LiteAndMCP/C-002/lite_on_+_codemode_on_mantiene_solo_gomemory_en_codemode
      === RUN   TestHCD052LiteAndMCP/C-002/codemode_on_+_lite_on_+_codemode_off_mantiene_lite
      === RUN   TestHCD052LiteAndMCP/C-002/install/update_(applyCodemodeDefault)_respeta_lite_en_direct_y_codemode
      --- PASS: TestHCD052LiteAndMCP (0.22s)
          --- PASS: TestHCD052LiteAndMCP/C-003 (0.07s)
              --- PASS: TestHCD052LiteAndMCP/C-003/ida_y_vuelta_estandar_conserva_entrada_ajena (0.02s)
              --- PASS: TestHCD052LiteAndMCP/C-003/preserva_definiciones_personalizadas_de_codegraph_y_gomemory (0.02s)
              --- PASS: TestHCD052LiteAndMCP/C-003/no_pisa_ediciones_manuales_hechas_entre_lite_on_y_lite_off (0.02s)
          --- PASS: TestHCD052LiteAndMCP/C-001 (0.04s)
              --- PASS: TestHCD052LiteAndMCP/C-001/mcpServers_invalido_no_persiste_lite_on (0.00s)
              --- PASS: TestHCD052LiteAndMCP/C-001/error_en_lite_off_conserva_lite_on_previo (0.01s)
              --- PASS: TestHCD052LiteAndMCP/C-001/fallo_al_escribir_estado_restaura_mcp.json (0.01s)
              --- PASS: TestHCD052LiteAndMCP/C-001/fallo_al_escribir_mcp.json_conserva_estado_anterior (0.01s)
              --- PASS: TestHCD052LiteAndMCP/C-001/lite.json_corrupto_falla_sin_mutar_mcp.json (0.01s)
          --- PASS: TestHCD052LiteAndMCP/C-002 (0.12s)
              --- PASS: TestHCD052LiteAndMCP/C-002/lite_on_+_codemode_on_mantiene_solo_gomemory_en_codemode (0.03s)
              --- PASS: TestHCD052LiteAndMCP/C-002/codemode_on_+_lite_on_+_codemode_off_mantiene_lite (0.03s)
              --- PASS: TestHCD052LiteAndMCP/C-002/install/update_(applyCodemodeDefault)_respeta_lite_en_direct_y_codemode (0.05s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.407s
- 2026-11-19 19:53 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run TestHCD052TDDBashIngest -v` (salida 0)
      === RUN   TestHCD052TDDBashIngest
          hcd052_test.go:471: 
      --- PASS: TestHCD052TDDBashIngest (0.16s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.319s
- 2026-11-19 19:53 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run TestHCD052TasksVerifyGuard -v` (salida 0)
      === RUN   TestHCD052TasksVerifyGuard
          hcd052_test.go:505: 
      --- PASS: TestHCD052TasksVerifyGuard (0.13s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.286s
- 2026-11-19 19:53 -05:00 · tarea 4 · verde · vigente · `go vet ./... && go test -race ./...` (salida 0)
      ok  	pi-gomemory	1.263s
      ok  	pi-gomemory/pkg/installer	81.654s

## Siguiente
Cerrada con aprobación sobre la revisión 1616ba03dcd8.

# Corregir los ocho hallazgos del ACR de v1.15.0

ID: HCD-047
Tipo: fix
Carril: full
Motivo: La migración y la desinstalación pueden alterar datos del usuario; las decisiones de recuperación y de propiedad se afectan entre sí

## Intención
Corregir C-001 a C-008 preservando instrucciones, configuración ajena y la preferencia de codemode ante fallos y reintentos

## Reproducción
Origen: [ACR de v1.15.0](../docs/audits/2026-11-17-acr-v1.15.0.md), commit `ac5cd1b`. Reproducción diagnóstica observada con el código de `ac5cd1b` en HOME temporales, sin usar una instalación real. El comando compila el árbol presente; para repetir la observación histórica, ejecutarlo desde un árbol aislado en ese commit:

```sh
go build -o /tmp/pi-hcd047-baseline .
cp docs/audits/2026-11-17-acr-v1.15.0-repro.mjs.txt /tmp/pi-hcd047-baseline.mjs
node /tmp/pi-hcd047-baseline.mjs /tmp/pi-hcd047-baseline
```

El arnés del informe solo registra observaciones. La tarea 1 añadirá `pkg/installer/hcd047_regression_test.go`, con `TestHCD047Regression` y un subtest nombrado por cada criterio C-01 a C-08. El comando de la tarea ejecutará las aserciones con datos sintéticos y HOME temporales. Se ejecutará primero sobre el código de producción de `ac5cd1b` sin correcciones y luego sobre el código corregido. Un fallo de compilación, la ausencia de los ocho subtests en la salida o un error de preparación no cuenta como rojo válido. El verde requiere que se ejecuten y pasen los ocho subtests.

| Caso | Aserción de regresión |
|---|---|
| C-01 | Un error de MCP o estado no deja un cambio silencioso; al reintentar se registra la propiedad si el instalador añade el token. |
| C-02 | `off` persiste, los tres MCP propios quedan directos y `defaultTools` conserva la selección ajena. |
| C-03 | El fallo de instalación conserva los bytes de `AGENTS.md`; el reintento instala la extensión y retira solo el protocolo. |
| C-04 | Con marcadores ambiguos, incluso junto a bloques reconocibles, se conservan todos los bytes de `AGENTS.md` y se comprueba el aviso (Q2). |
| C-05 | La migración retira el protocolo marcado y el histórico sin marcas de un archivo sin ambigüedad, conservando texto y permisos o enlace. |
| C-06 | El perfil exacto de v1.14.0 se elimina usando un fixture obtenido del tag, independiente del perfil actual; una entrada propia del usuario bajo el mismo nombre se conserva. |
| C-07 | La salida informa del estado inválido y se conservan sus bytes y el token sin propiedad verificable. |
| C-08 | Un destino diferente conserva su token si cambia antes o después del plan, o antes de reinstalar; un registro heredado válido sin destino conserva el token y emite el aviso de Q1. |

## Alcance
Incluido: corregir C-001 a C-008, conservar la configuración ajena y validar los fallos y reintentos con el binario en HOME temporales. Mantener las garantías de HCD-032, HCD-036, HCD-045 y HCD-046: permisos y enlaces, contexto instalado antes de retirar el protocolo anterior, preferencia persistente y retiro solo con prueba de propiedad. Actualizar las referencias vigentes sobre el comportamiento una vez verificado, sin reescribir el ACR histórico.
Excluido: `/plan`, `deps-update`, nuevas reglas de selección de herramientas de Pi, una garantía frente a cortes abruptos del proceso o escrituras concurrentes arbitrarias, dependencias nuevas, cambios en HOME real, commit, push y release.

## ¿Qué pasa si no lo hacemos?
Una reinstalación puede contradecir una preferencia explícita. Un fallo de escritura puede dejar activaciones sin dueño, y una migración o desinstalación puede retirar instrucciones o herramientas ajenas. El usuario puede tener que corregir archivos a mano aunque el comando haya terminado con éxito.

## Alternativa más barata
Documentar los casos y pedir copias de respaldo antes de actualizar reduce el impacto, pero no impide que la operación borre texto propio, pierda el registro de propiedad o anuncie éxito parcial. Restringir la corrección a los cuatro hallazgos de consenso dejaría cuatro comportamientos ya reproducidos sin resolver.

## Normas aplicables
Constitución §10, §18 y §21; Playbook HCD-FLOW-02/03, HCD-DEF-01, HCD-TRACE-01, HCD-TDD-01 y HCD-EVIDENCE-01.

## Historias
### US1. Conmutación recuperable de codemode
Como persona que instala o actualiza Pi, quiero que codemode respete mi elección y que pueda reintentar tras un fallo, para no quedar con un perfil incoherente ni perder la prueba de propiedad.
- C-01: Dado un fallo al escribir MCP o el archivo de estado tras empezar a conmutar, cuando el comando devuelve error y se corrige la causa, entonces la configuración anterior queda restaurada, o el error de restauración se informa, y un reintento exitoso no pierde la propiedad del token que haya añadido el instalador. Cubre ACR C-001.
- C-02: Dado `off` persistido y un `+codemode` que el usuario habilitó para otros usos, cuando `install` o `update` reinstala, entonces los servidores propios quedan directos y la selección global ajena queda intacta. Cubre ACR C-002.

### US2. Migración que conserva las instrucciones del usuario
Como persona que usa `AGENTS.md`, quiero que el instalador retire solo el protocolo reconocido cuando su reemplazo esté disponible, para conservar mis instrucciones incluso si falla la actualización.
- C-03: Dado un protocolo heredado y un fallo al instalar la extensión de contexto, cuando `install` falla, entonces el protocolo y el texto propio permanecen intactos. Cubre ACR C-003.
- C-04: Dado un `AGENTS.md` con marcadores ambiguos, incluso si contiene otros bloques reconocibles, cuando se intenta retirar el protocolo, entonces se conserva el archivo completo sin modificaciones y se avisa de la ambigüedad (Q2). Cubre ACR C-004.
- C-05: Dado un archivo sin marcadores ambiguos que combina bloques marcados inequívocos y protocolos históricos exactos sin marcas, cuando la migración termina correctamente, entonces se retiran todos los bloques reconocidos conservando el texto propio y sus permisos o enlace. Cubre ACR C-008.

### US3. Desinstalación que no confunde propiedad con coincidencia
Como persona que usa configuraciones propias o de una versión anterior, quiero que `uninstall` retire solo los recursos gestionados, para mantener mis personalizaciones y poder reintentar cuando el estado no se pueda interpretar.
- C-06: Dado el perfil MCP híbrido exacto publicado en v1.14.0, cuando se desinstala con el binario corregido, entonces se retiran las entradas propias intactas y se conservan las definiciones personalizadas. Cubre ACR C-005.
- C-07: Dado un `codemode.json` ilegible o inválido, cuando `uninstall` construye y ejecuta su plan, entonces informa del fallo y conserva ese archivo y el token cuya propiedad no puede determinarse. Cubre ACR C-006.
- C-08: Dado que cambia el destino de `settings.json`, cuando `uninstall` prepara o aplica la retirada, entonces no usa la prueba de propiedad del destino anterior para retirar un token del destino nuevo, ni reclama propiedad de una activación que ya existía. Si un registro válido de v1.15.0 afirma propiedad pero no identifica el destino, conserva `+codemode` y avisa que no se pudo comprobar su propiedad (Q1). Cubre ACR C-007.

## Aclaraciones
- [x] Q1. ¿Qué debe hacer uninstall con un codemode.json válido de v1.15.0 que afirma propiedad de +codemode pero no identifica el destino de settings.json? → Q1:1
- [x] Q2. Si AGENTS.md contiene marcadores ambiguos junto con bloques reconocibles, ¿se conserva todo el archivo y se avisa, o se retiran solo los bloques inequívocos que queden fuera de la zona ambigua? → 1

## Slices
La tarea 1 es la regresión transversal de los ocho casos. Primero se observa el rojo; el verde de la suite completa se obtiene al terminar los tres incrementos. Las tareas específicas se incorporan y vinculan en `/hcd-tasks`.

### VS-01. Codemode se recupera de fallos y respeta la preferencia al reinstalar
Tareas: 1, 2, 3, 9, 10
Criterios: C-01, C-02

### VS-02. AGENTS.md conserva las instrucciones propias durante la migración
Tareas: 1, 4, 5, 9, 10
Criterios: C-03, C-04, C-05

### VS-03. Uninstall respeta los perfiles gestionados y conserva la configuración ajena
Tareas: 1, 6, 7, 8, 9, 10
Criterios: C-06, C-07, C-08

## Checklist
- [x] 1. Demostrar la corrección de los ocho casos del ACR con las aserciones de regresión C-01 a C-08 → `go test ./pkg/installer -run '^TestHCD047Regression$' -count=1 -v`
- [x] 2. Restaurar settings.json y mcp.json cuando falla la conmutación de codemode → `go test ./pkg/installer -run '^TestHCD047Regression$/^C-01$' -count=1 -v`
- [x] 3. Escribir el perfil MCP directo sin tocar defaultTools cuando la preferencia es off → `go test ./pkg/installer -run '^TestHCD047Regression$/^C-02$' -count=1 -v`
- [x] 4. Retirar el protocolo heredado después de instalar la extensión de contexto → `go test ./pkg/installer -run '^TestHCD047Regression$/^C-03$' -count=1 -v`
- [x] 5. Conservar AGENTS.md ambiguo con aviso y retirar todos los bloques inequívocos → `go test ./pkg/installer -run '^TestHCD047Regression$/^C-0[45]$' -count=1 -v`
- [x] 6. Reconocer en uninstall el perfil híbrido exacto de v1.14.0 → `go test ./pkg/installer -run '^TestHCD047Regression$/^C-06$' -count=1 -v`
- [x] 7. Informar y conservar un codemode.json inválido en uninstall → `go test ./pkg/installer -run '^TestHCD047Regression$/^C-07$' -count=1 -v`
- [x] 8. Asociar la propiedad de +codemode al destino de settings.json → `go test ./pkg/installer -run '^TestHCD047Regression$/^C-08$' -count=1 -v`
- [x] 9. Comprobar los ocho escenarios con el binario compilado en HOME temporales → `go test ./pkg/installer -run '^TestHCD047Binary$' -count=1 -v`
- [x] 10. Ejecutar la suite completa, vet y formato sin regresiones → `test -z "$(gofmt -l pkg)" && go vet ./... && go test ./... -count=1`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-17 15:32 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD047Regression$' -count=1 -v` (salida 1)
              --- FAIL: TestHCD047Regression/C-04/ambiguo_junto_a_formato_sin_marcas (0.01s)
              --- FAIL: TestHCD047Regression/C-04/cierre_antes_de_apertura (0.00s)
              --- FAIL: TestHCD047Regression/C-04/apertura_huérfana (0.00s)
          --- FAIL: TestHCD047Regression/C-05 (0.01s)
              --- FAIL: TestHCD047Regression/C-05/archivo (0.01s)
              --- FAIL: TestHCD047Regression/C-05/enlace (0.01s)
          --- FAIL: TestHCD047Regression/C-06 (0.01s)
              --- FAIL: TestHCD047Regression/C-06/perfil_exacto_se_retira (0.01s)
              --- PASS: TestHCD047Regression/C-06/entrada_propia_con_el_mismo_nombre_se_conserva (0.00s)
          --- FAIL: TestHCD047Regression/C-07 (0.01s)
              --- FAIL: TestHCD047Regression/C-07/JSON_truncado (0.00s)
              --- PASS: TestHCD047Regression/C-07/error_de_lectura (0.00s)
          --- FAIL: TestHCD047Regression/C-08 (0.11s)
              --- FAIL: TestHCD047Regression/C-08/cambio_antes_del_plan (0.03s)
              --- FAIL: TestHCD047Regression/C-08/cambio_después_del_plan (0.02s)
              --- FAIL: TestHCD047Regression/C-08/cambio_antes_de_reinstalar (0.05s)
              --- FAIL: TestHCD047Regression/C-08/registro_heredado_sin_destino_(Q1) (0.01s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	1.080s
      FAIL
- 2026-11-17 15:39 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD047Regression$' -count=1 -v` (salida 0)
              --- PASS: TestHCD047Regression/C-04/cierre_antes_de_apertura (0.00s)
              --- PASS: TestHCD047Regression/C-04/apertura_huérfana (0.00s)
              --- PASS: TestHCD047Regression/C-04/bloques_duplicados (0.00s)
              --- PASS: TestHCD047Regression/C-04/ambiguo_junto_a_formato_sin_marcas (0.00s)
          --- PASS: TestHCD047Regression/C-05 (0.01s)
              --- PASS: TestHCD047Regression/C-05/archivo (0.01s)
              --- PASS: TestHCD047Regression/C-05/enlace (0.01s)
          --- PASS: TestHCD047Regression/C-06 (0.02s)
              --- PASS: TestHCD047Regression/C-06/perfil_exacto_se_retira (0.02s)
              --- PASS: TestHCD047Regression/C-06/entrada_propia_con_el_mismo_nombre_se_conserva (0.00s)
          --- PASS: TestHCD047Regression/C-07 (0.00s)
              --- PASS: TestHCD047Regression/C-07/JSON_truncado (0.00s)
              --- PASS: TestHCD047Regression/C-07/error_de_lectura (0.00s)
          --- PASS: TestHCD047Regression/C-08 (0.08s)
              --- PASS: TestHCD047Regression/C-08/cambio_antes_del_plan (0.03s)
              --- PASS: TestHCD047Regression/C-08/cambio_después_del_plan (0.01s)
              --- PASS: TestHCD047Regression/C-08/cambio_antes_de_reinstalar (0.04s)
              --- PASS: TestHCD047Regression/C-08/registro_heredado_sin_destino_(Q1) (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	1.268s
- 2026-11-17 15:39 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD047Regression$/^C-01$' -count=1 -v` (salida 0)
      === RUN   TestHCD047Regression
      === RUN   TestHCD047Regression/C-01
      === RUN   TestHCD047Regression/C-01/fallo_de_MCP_restaura_y_el_reintento_registra_la_propiedad
      === RUN   TestHCD047Regression/C-01/fallo_del_estado_restaura_y_el_reintento_registra_la_propiedad
      --- PASS: TestHCD047Regression (0.10s)
          --- PASS: TestHCD047Regression/C-01 (0.10s)
              --- PASS: TestHCD047Regression/C-01/fallo_de_MCP_restaura_y_el_reintento_registra_la_propiedad (0.05s)
              --- PASS: TestHCD047Regression/C-01/fallo_del_estado_restaura_y_el_reintento_registra_la_propiedad (0.05s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.275s
- 2026-11-17 15:39 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD047Regression$/^C-02$' -count=1 -v` (salida 0)
      ado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD047RegressionC-021377688925/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD047RegressionC-021377688925/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- PASS: TestHCD047Regression (0.26s)
          --- PASS: TestHCD047Regression/C-02 (0.26s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.426s
- 2026-11-17 15:39 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD047Regression$/^C-03$' -count=1 -v` (salida 0)
      ado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD047RegressionC-031319807206/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD047RegressionC-031319807206/001/.local/bin/pi-gomemory[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-antigravity    [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
        [38;2;145;169;197m→[0m Paquete fijado pi-cc-extensions  [38;2;145;169;197msin datos de npm; ejecuta pi-gomemory-setup deps-refresh[0m
      --- PASS: TestHCD047Regression (0.53s)
          --- PASS: TestHCD047Regression/C-03 (0.53s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.822s
- 2026-11-17 15:39 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD047Regression$/^C-0[45]$' -count=1 -v` (salida 0)
      === RUN   TestHCD047Regression
      === RUN   TestHCD047Regression/C-04
      === RUN   TestHCD047Regression/C-04/bloques_duplicados
      === RUN   TestHCD047Regression/C-04/ambiguo_junto_a_formato_sin_marcas
      === RUN   TestHCD047Regression/C-04/cierre_antes_de_apertura
      === RUN   TestHCD047Regression/C-04/apertura_huérfana
      === RUN   TestHCD047Regression/C-05
      === RUN   TestHCD047Regression/C-05/archivo
      === RUN   TestHCD047Regression/C-05/enlace
      --- PASS: TestHCD047Regression (0.02s)
          --- PASS: TestHCD047Regression/C-04 (0.01s)
              --- PASS: TestHCD047Regression/C-04/bloques_duplicados (0.00s)
              --- PASS: TestHCD047Regression/C-04/ambiguo_junto_a_formato_sin_marcas (0.00s)
              --- PASS: TestHCD047Regression/C-04/cierre_antes_de_apertura (0.00s)
              --- PASS: TestHCD047Regression/C-04/apertura_huérfana (0.00s)
          --- PASS: TestHCD047Regression/C-05 (0.01s)
              --- PASS: TestHCD047Regression/C-05/archivo (0.01s)
              --- PASS: TestHCD047Regression/C-05/enlace (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.383s
- 2026-11-17 15:39 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run '^TestHCD047Regression$/^C-06$' -count=1 -v` (salida 0)
      === RUN   TestHCD047Regression
      === RUN   TestHCD047Regression/C-06
      === RUN   TestHCD047Regression/C-06/perfil_exacto_se_retira
      === RUN   TestHCD047Regression/C-06/entrada_propia_con_el_mismo_nombre_se_conserva
      --- PASS: TestHCD047Regression (0.02s)
          --- PASS: TestHCD047Regression/C-06 (0.02s)
              --- PASS: TestHCD047Regression/C-06/perfil_exacto_se_retira (0.02s)
              --- PASS: TestHCD047Regression/C-06/entrada_propia_con_el_mismo_nombre_se_conserva (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.435s
- 2026-11-17 15:39 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run '^TestHCD047Regression$/^C-07$' -count=1 -v` (salida 0)
      === RUN   TestHCD047Regression
      === RUN   TestHCD047Regression/C-07
      === RUN   TestHCD047Regression/C-07/JSON_truncado
      === RUN   TestHCD047Regression/C-07/error_de_lectura
      --- PASS: TestHCD047Regression (0.00s)
          --- PASS: TestHCD047Regression/C-07 (0.00s)
              --- PASS: TestHCD047Regression/C-07/JSON_truncado (0.00s)
              --- PASS: TestHCD047Regression/C-07/error_de_lectura (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.170s
- 2026-11-17 15:39 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run '^TestHCD047Regression$/^C-08$' -count=1 -v` (salida 0)
      === RUN   TestHCD047Regression
      === RUN   TestHCD047Regression/C-08
      === RUN   TestHCD047Regression/C-08/cambio_antes_del_plan
      === RUN   TestHCD047Regression/C-08/cambio_después_del_plan
      === RUN   TestHCD047Regression/C-08/cambio_antes_de_reinstalar
      === RUN   TestHCD047Regression/C-08/registro_heredado_sin_destino_(Q1)
      --- PASS: TestHCD047Regression (0.08s)
          --- PASS: TestHCD047Regression/C-08 (0.08s)
              --- PASS: TestHCD047Regression/C-08/cambio_antes_del_plan (0.03s)
              --- PASS: TestHCD047Regression/C-08/cambio_después_del_plan (0.01s)
              --- PASS: TestHCD047Regression/C-08/cambio_antes_de_reinstalar (0.04s)
              --- PASS: TestHCD047Regression/C-08/registro_heredado_sin_destino_(Q1) (0.00s)
      PASS
      ok  	pi-gomemory/pkg/installer	0.245s
- 2026-11-17 15:39 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run '^TestHCD047Binary$' -count=1 -v` (salida 0)
      === RUN   TestHCD047Binary
      === RUN   TestHCD047Binary/off_persistido_(C-02)
      === RUN   TestHCD047Binary/propiedad_tras_reintento_(C-01)
      === RUN   TestHCD047Binary/migración_después_de_la_extensión_(C-03)
      === RUN   TestHCD047Binary/perfil_histórico_(C-06)
      === RUN   TestHCD047Binary/marca_huérfana_(C-04)
      === RUN   TestHCD047Binary/estado_inválido_(C-07)
      === RUN   TestHCD047Binary/cambio_de_dotfiles_(C-08)
      === RUN   TestHCD047Binary/dos_formatos_(C-05)
      --- PASS: TestHCD047Binary (2.05s)
          --- PASS: TestHCD047Binary/off_persistido_(C-02) (0.51s)
          --- PASS: TestHCD047Binary/propiedad_tras_reintento_(C-01) (0.07s)
          --- PASS: TestHCD047Binary/migración_después_de_la_extensión_(C-03) (0.55s)
          --- PASS: TestHCD047Binary/perfil_histórico_(C-06) (0.02s)
          --- PASS: TestHCD047Binary/marca_huérfana_(C-04) (0.27s)
          --- PASS: TestHCD047Binary/estado_inválido_(C-07) (0.05s)
          --- PASS: TestHCD047Binary/cambio_de_dotfiles_(C-08) (0.04s)
          --- PASS: TestHCD047Binary/dos_formatos_(C-05) (0.27s)
      PASS
      ok  	pi-gomemory/pkg/installer	2.254s
- 2026-11-17 15:39 -05:00 · tarea 10 · verde · vigente · `test -z "$(gofmt -l pkg)" && go vet ./... && go test ./... -count=1` (salida 0)
      ok  	pi-gomemory	0.210s
      ok  	pi-gomemory/pkg/installer	76.463s

## Siguiente
Cerrada con aprobación sobre la revisión 08eaff8bc8e4.

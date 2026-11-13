# uninstall y update no pierden datos ni aceptan binarios truncados

ID: HCD-034
Tipo: fix
Carril: full
Motivo: Ceremonia completa: riesgo de pérdida de contenido del usuario (H-01) y de integridad del binario instalado (H-03), con una decisión de diseño abierta en H-01; la persona eligió este carril

## Intención
Cerrar los 3 hallazgos medios de la ADM del 2026-11-13 (docs/audits/2026-11-13-defensive-qa.md), reproducidos sobre v1.13.3: H-01 uninstall borra ediciones y notas añadidas tras construir el plan porque la propiedad no se revalida al aplicar; H-02 un archivo gestionado ilegible se trata como inexistente y uninstall anuncia éxito; H-03 extractBinary acepta un ejecutable truncado cuando el miembro tar supera 200 MiB.

Antecedentes: HCD-011 definió la regla de propiedad de `uninstall` (SHA256 del manifiesto o del contenido de esta versión) y HCD-027 protegió las skills con archivos añadidos **antes** de planificar. H-01 es el caso de cambios **posteriores** al plan, que ninguna de las dos cubre. HCD-030 acotó el launcher a su `HOME`; sus pruebas deben seguir verdes. La auditoría se contrastó sobre `33c5e25`; la reproducción de abajo confirma los tres defectos sobre v1.13.3 (`9879a13`).

## Reproducción
`bash docs/audits/2026-11-13-adm-repro.sh`: copia el árbol de trabajo a una carpeta temporal, añade las cuatro pruebas de `docs/audits/2026-11-13-adm-repro_test.go.txt` y las ejecuta. Hoy fallan las cuatro:
- `TestADMRevalidateOwnedFile`: «user edit deleted after plan; apply error=<nil>» (H-01, `planOwnedFile`).
- `TestADMRevalidateSkill`: «new user notes deleted after plan; apply error=<nil>» (H-01, `planSkillDir`).
- `TestADMUnreadableUninstall`: «unreadable managed path silently ignored: ✓ No queda nada que retirar de pi-gomemory.» (H-02).
- `TestADMExtractRejectsOversize`: «oversized member accepted and truncated: header=209715201 returned=209715200» (H-03, `extractBinary`).

Salvedad: `TestADMUnreadableUninstall` usa permisos `0000` y, ejecutando como root, se salta (`t.Skip`) en lugar de fallar; los criterios C-06 y C-07 no tienen esa limitación.

El script vive en el repositorio, junto a las pruebas de la auditoría, y no lo modifica. La verificación de las tareas no depende de él: cada criterio C-NN tiene su prueba `TestHCD034CNN` en `pkg/installer/hcd034_*_test.go`, y el conjunto se verifica con `go test ./pkg/installer -run '^TestHCD034' -count=1`. El script solo sirve para observar el rojo inicial y repetir la reproducción de la auditoría.

La revisión Splash de la revisión `a3358969cc76` encontró un Blocker y siete Major; esta versión los incorpora mediante Q3 a Q6. La revisión Splash de `aa4fe7c1d0a1` (0 Blocker, 4 Major, 5 Nit) se incorpora en el texto de Alcance e Historias. La de `726042896c30` (1 Blocker, 4 Major, 2 Nit) se incorpora mediante Q7 a Q10 y en el texto.

## Alcance
Incluido: (1) H-01, revalidar cada retirada al aplicar; (2) H-02, solo la ausencia de un recurso es inocua; (3) H-03, ejecutable completo y dentro del límite; (4) pruebas por criterio y `docs/known-issues.md`. Detalle:
(1) **H-01 · revalidar al aplicar.** Cada acción de retirada comprueba, justo antes de escribir, que el recurso sigue siendo el que examinó el plan: `planOwnedFile` (archivo regular con la misma huella SHA256 que la examinada; un symlink en su lugar cuenta como cambio), `planSkillDir` (skill embebida que sigue cumpliendo `isRemovableEmbeddedCopy`; enlace a skill del anfitrión que apunta al mismo destino), `planMCP` (entrada `mcpServers.<nombre>` que sigue igual a la de `DefaultMCPServers`), `planSettings` (`theme` que sigue valiendo `gomemory`; registro del repositorio de skills que sigue en `skills`), `planAgents` (bloque de pi-gomemory idéntico al planificado) y `planPackages` (entrada `packages` con la fuente fijada exacta, revalidada **antes** de invocar `pi remove`; `pi-cc-extensions.json` que sigue siendo igual a los valores por defecto) (Q1, Q3, Q4). Si cambió, se conserva y la acción termina como `omitir` con el motivo «cambió después del plan». Si el recurso ya no existe al aplicar, la acción termina como `omitir` con el motivo «ya no existe», en cualquier retirada (Q5); lo mismo si el archivo sigue pero la clave, la entrada o el bloque planificado ya no está, y no cuenta para N (Q8). Si la relectura al aplicar falla con un error distinto de la ausencia, la acción termina como `fallido` con el error y el comando termina con error (Q9). Ambas líneas «omitir» se imprimen siempre, también sin `--verbose`, una sola vez cada una, y no hacen fallar el comando. Si hubo conservados por cambio, el mensaje final pasa a «✓ pi-gomemory desinstalado; se conservaron N elementos que cambiaron después del plan», donde N es el número de acciones `omitir` con el motivo «cambió después del plan»; la línea «Resumen: …» no cambia y las sigue contando entre los omitidos. Se acepta la carrera residual de milisegundos entre revalidar y escribir.
(2) **H-02 · solo la ausencia es inocua.** «Ausencia» es `fs.ErrNotExist` o `ENOTDIR`. En el plan, cualquier otro error al inspeccionar un recurso añade una acción `fallido` con la ruta y el error, de modo que `uninstall` y `uninstall --dry-run` terminan con error y nunca muestran «No queda nada que retirar» (Q2). Con `--yes`, un `fallido` del plan no detiene las demás retiradas: se aplican todas y el comando termina con error, como hoy (`uninstall.go:546-558`). Si la inspección del tema termina en `fallido`, la clave `theme` de `settings.json` se conserva, igual que con el tema editado (Q10). Lista cerrada de inspecciones: `ReadFile` de `planOwnedFile` (extensiones, tema y launcher de las carpetas respaldadas: el `BIN_DIR` del manifiesto o, sin manifiesto, `LocalBinDir`, y `PI_GOMEMORY_BIN_DIR`); `ReadDir` de las dos carpetas de skills y, por entrada, `Readlink`, `ReadDir`/`ReadFile` de `isRemovableEmbeddedCopy` y `Stat` de la marca; `Stat` de `mcp.json` y `settings.json`; `ReadFile` de `AGENTS.md`; `Lstat` de `planPlainFile` (cachés y manifiesto); con `--packages`, la lectura de `settings.json` y de `pi-cc-extensions.json`; con `--self`, `os.Executable` y `Lstat`. Excepción (Q5): en las carpetas candidatas de launcher que no están respaldadas (`~/.local/bin`, `LocalBinDir` cuando el manifiesto registra otro `BIN_DIR`, la carpeta del ejecutable dentro de `HOME` y el `Lstat` de la carpeta del ejecutable fuera de `HOME`, `foreign`), un error al inspeccionar `<carpeta>/pi-gomemory` distinto de la ausencia se registra como `omitir` con el motivo «no se pudo inspeccionar: <error>», porque puede no ser de pi-gomemory. Esta definición de carpetas respaldadas y candidatas precisa la frase «sin respaldo del manifiesto» de Q5: un error en `PI_GOMEMORY_BIN_DIR` o, sin manifiesto, en `LocalBinDir` es `fallido`. Aunque no quede ninguna retirada pendiente, estas acciones `omitir` (y la de `foreign`) se imprimen antes de «✓ No queda nada que retirar» (Q7).
(3) **H-03 · ejecutable completo y dentro del límite.** `extractBinary` usa un límite propio del ejecutable, `maxBinaryBytes` (200 MiB, variable para que las pruebas lo reduzcan, Q6). Rechaza con error un miembro con tamaño declarado 0 o mayor que el límite, y uno cuyo contenido leído no coincida con el tamaño declarado. El caso del contenido truncado ya falla hoy con `io.ErrUnexpectedEOF`; aquí solo se blinda con una prueba. `DownloadAndReplace` conserva entonces el binario anterior byte a byte.
(4) Pruebas de regresión por criterio en `pkg/installer/hcd034_*_test.go` y `docs/known-issues.md` actualizado.
Excluido: revalidar al aplicar las cachés, el manifiesto y el binario de `--self`, que escribe solo pi-gomemory y no guardan contenido propio (Q4; sí les aplica «ya no existe» de Q5); cerrar por completo la carrera entre revalidar y escribir (cuarentena descartada en Q1); validar el formato del ejecutable o el contenido restante del gzip; cambiar el límite de 200 MiB del tarball descargado; incorporar `govulncheck`, `shellcheck` o escaneo de secretos (otras recomendaciones de la auditoría); publicar una release.

## ¿Qué pasa si no lo hacemos?
Una edición o unas notas añadidas mientras `uninstall` espera confirmación se borran sin aviso (H-01). Una desinstalación puede dejar archivos gestionados en disco y decir que no queda nada, así que la persona no sabe que debe revisarlos (H-02). Y si una release llegara a tener un ejecutable de más de 200 MiB descomprimidos, `update` instalaría un binario cortado que no arranca (H-03).

## Alternativa más barata
Documentar «no edites nada mientras uninstall espera confirmación», «revisa ~/.pi/agent tras desinstalar» y «comprueba el tamaño del binario antes de publicar». Deja los tres defectos activos y depende de la memoria de cada persona. Se descarta.

## Historias
### US1. uninstall no borra lo que cambió después del plan
Como persona que desinstala pi-gomemory, quiero que `uninstall` conserve lo que edité o añadí después de ver el plan, para no perder contenido propio.
- C-01: Dado un archivo gestionado intacto que el plan marca para retirar, cuando lo edito (o lo sustituyo por un symlink) antes de confirmar, entonces `uninstall --yes` no lo modifica, su salida sin `--verbose` contiene exactamente una vez «[omitir] <ruta> — cambió después del plan» (también una sola vez con `--verbose`) y el comando termina sin error.
- C-02: Dada una skill embebida intacta que el plan marca para retirar, cuando añado un archivo a su carpeta antes de confirmar, entonces la carpeta se conserva completa y la salida contiene la misma línea «[omitir] <ruta> — cambió después del plan».
- C-03: Dadas una entrada `mcpServers.gomemory`, la clave `theme`, el registro del repositorio de skills en `settings.json`, el bloque de `AGENTS.md`, una entrada `packages` fijada y `pi-cc-extensions.json` (ambas con `--packages`; para `packages`, sin que se invoque `pi remove`) y un enlace a skill del anfitrión marcados para retirar, cuando cada uno cambia antes de confirmar, entonces se conserva tal como lo dejé y la salida contiene su línea «[omitir] … — cambió después del plan».
- C-04: Dado un recurso marcado para retirar que desaparece antes de confirmar (un archivo, o una clave, entrada o bloque cuyo archivo sigue presente), cuando confirmo, entonces la salida contiene «[omitir] <ruta> — ya no existe» y el comando termina sin error.
- C-05: Dado que nada cambia entre el plan y la confirmación, cuando confirmo, entonces `uninstall` retira lo mismo que en v1.13.3 (`go test ./pkg/installer -run 'TestHCD011|TestHCD027|TestHCD030|TestHCD032|TestH0'` en verde) y el mensaje final es «✓ pi-gomemory desinstalado. Para reinstalar: pi-gomemory-setup install»; si hubo conservados por cambio, el mensaje es «✓ pi-gomemory desinstalado; se conservaron N elementos que cambiaron después del plan».

### US2. uninstall no anuncia éxito si no pudo inspeccionar algo
Como persona que desinstala pi-gomemory, quiero que `uninstall` falle si no pudo leer un recurso gestionado, para saber que algo quedó por revisar.
- C-06: Para cada inspección de la lista cerrada (una subprueba por inspección), dado un recurso que existe pero no se puede inspeccionar, cuando ejecuto `uninstall --yes` o `uninstall --dry-run`, entonces la salida contiene «[fallido] <ruta>», el comando termina con error y no aparece «No queda nada que retirar». El fallo se provoca de forma que ocurra también como root: para las lecturas `ReadFile`, una carpeta donde se espera un archivo (EISDIR); para `Lstat`, `Stat`, `ReadDir` y `Readlink`, un bucle de symlinks en la ruta (ELOOP). Un archivo donde se espera una carpeta da ENOTDIR y cuenta como ausencia (C-04/C-08).
- C-11: Dada una retirada cuya relectura al aplicar falla con un error distinto de la ausencia, cuando confirmo, entonces la salida contiene «[fallido] <ruta>» y el comando termina con error (Q9); y si la inspección de `themes/gomemory.json` termina en `fallido`, la clave `theme` de `settings.json` se conserva (Q10).
- C-07: Dada una carpeta candidata de launcher no respaldada cuyo `<carpeta>/pi-gomemory` no se puede inspeccionar (en la prueba, una carpeta en su lugar: EISDIR, también como root), cuando ejecuto `uninstall --yes` en un `HOME` donde no queda ninguna otra retirada pendiente, entonces la salida contiene «[omitir] <ruta> — no se pudo inspeccionar» antes de «✓ No queda nada que retirar de pi-gomemory.» y el comando termina sin error (Q7).
- C-08: Dado un `HOME` sin nada de pi-gomemory, cuando ejecuto `uninstall --yes`, entonces sigue mostrando «✓ No queda nada que retirar de pi-gomemory.» y termina sin error.

### US3. update no instala un binario incompleto
Numeración: Q6 se resolvió con la numeración de `a3358969cc76`; su «C-07 sobre extractBinary» es hoy C-09 y su «C-06 sobre DownloadAndReplace» es hoy C-10.
Como persona que actualiza pi-gomemory, quiero que `update` rechace un ejecutable vacío, truncado o mayor que el límite, para no quedarme con un binario que no arranca.
- C-09: Con `maxBinaryBytes` reducido a 1 MiB en la prueba, `extractBinary` acepta un miembro de exactamente 1 MiB y rechaza con error uno de 1 MiB + 1 byte, uno vacío y uno cuyo tar está truncado.
- C-10: Con `maxBinaryBytes` reducido y un servidor HTTP local que sirve un tarball con checksum correcto cuyo ejecutable supera el límite, `DownloadAndReplace` devuelve error y el binario destino queda idéntico byte a byte.

## Aclaraciones
- [x] Q1. Q1 (H-01): si un archivo o una skill cambia entre el plan de uninstall y la confirmación, ¿qué pasa? A) Conservar y omitir: se revalida justo antes de borrar; si cambió, se conserva, se lista como [omitir] «cambió después del plan» siempre visible y uninstall termina sin error; se acepta la carrera residual de milisegundos (recomendado). B) Conservar y fallar: igual, pero [fallido] y error. C) Cuarentena con recuperación: cierra la carrera, mucho más complejo. → Conservar y omitir (Recomendado)
- [x] Q2. Q2 (H-02): ¿qué lecturas del plan de uninstall registran [fallido] cuando fallan por algo distinto de «no existe»? A) Todas: archivos gestionados, carpetas de skills, AGENTS.md, mcp.json, settings.json, cachés y manifiesto (recomendado). B) Solo planOwnedFile. → Todas (Recomendado)
- [x] Q3. Q3 (Splash H1, Blocker): ¿las retiradas en mcp.json (mcpServers.<nombre>), settings.json › theme y el bloque de AGENTS.md revalidan la propiedad al aplicar? A) Incluir: misma entrada MCP, theme sigue en gomemory y mismo bloque que el planificado; si cambió, [omitir] como Q1 (recomendado). B) Riesgo aceptado y Excluido reescrito. → Incluir (Recomendado)
- [x] Q4. Q4 (Splash H2): ¿qué otras retiradas revalidan? A) Contenido editable: pi-cc-extensions.json y el enlace a skill del anfitrión; cachés, manifiesto y binario (--self) quedan fuera porque solo los escribe pi-gomemory (recomendado). B) Todas. → Contenido editable (Recomendado)
- [x] Q5. Q5 (Splash H5 y H6): casos límite de ausencia. A) Recurso desaparecido entre plan y confirmación = [omitir] «ya no existe» sin error; ENOTDIR cuenta como ausencia; carpeta candidata de launcher ilegible sin respaldo del manifiesto = [omitir] informado; sustituir por symlink cuenta como cambio; el resto ilegible = [fallido] (recomendado). B) Todo error distinto de ENOENT es [fallido]. → Desaparecido = omitir; carpeta candidata ajena ilegible = omitir (Recomendado)
- [x] Q6. Q6 (Splash H8): ¿cómo se prueban C-06 y C-07? A) Límite del ejecutable inyectable reducido a 1 MiB en pruebas: C-07 sobre extractBinary y C-06 sobre DownloadAndReplace con servidor local (recomendado). B) 200 MiB reales. → Límite inyectable (Recomendado)
- [x] Q7. Q7 (Splash H1, Blocker): con Pending()==0, ¿se imprimen las acciones [omitir] «no se pudo inspeccionar» (y la omitir de la carpeta del ejecutable fuera de HOME) antes de «✓ No queda nada que retirar»? A) Sí, se imprimen; con HOME vacío solo sale la línea ✓ (recomendado). B) C-07 exige montar una retirada pendiente. → Sí, se imprimen (Recomendado)
- [x] Q8. Q8 (Splash H3): si al aplicar el archivo existe pero la clave, la entrada o el bloque planificado (mcpServers.<nombre>, theme, registro de skills, bloque de AGENTS.md, packages.<nombre>) ya no está, ¿qué motivo se registra? A) [omitir] «ya no existe», sin contar para N (recomendado). B) [omitir] «cambió después del plan», contando para N. → Ya no existe, sin contar para N (Recomendado)
- [x] Q9. Q9 (Splash H4): si la relectura al aplicar falla con un error distinto de la ausencia (EACCES, EISDIR…), ¿qué pasa? A) [fallido] con el error y el comando termina con error, como cualquier error de aplicación hoy (recomendado). B) [omitir] «cambió después del plan», contando para N. → Fallido y error (Recomendado)
- [x] Q10. Q10 (Splash H5): si la inspección de themes/gomemory.json termina en [fallido], ¿la clave theme de settings.json se conserva o se retira? A) Se conserva, como cuando el tema está editado (recomendado). B) Se retira. → Se conserva (Recomendado)

## Slices
### VS-01. update rechaza un ejecutable vacío, truncado o mayor que el límite
Tareas: 1, 2
Criterios: C-09, C-10

### VS-02. uninstall conserva lo que cambió o desapareció después del plan
Tareas: 3, 4, 5
Criterios: C-01, C-02, C-03, C-04, C-05

### VS-03. uninstall no anuncia éxito si no pudo inspeccionar o releer algo
Tareas: 6, 7, 8
Criterios: C-06, C-07, C-08, C-11

## Checklist
- [x] 1. extractBinary aplica maxBinaryBytes y rechaza vacío, mayor que el límite y truncado → go test ./pkg/installer -run '^TestHCD034C09$' -count=1
- [x] 2. DownloadAndReplace conserva el binario anterior byte a byte ante un ejecutable mayor que el límite → go test ./pkg/installer -run '^TestHCD034C10$' -count=1
- [x] 3. planOwnedFile y planSkillDir revalidan al aplicar y registran «cambió después del plan» o «ya no existe» → go test ./pkg/installer -run '^TestHCD034C0[124]$' -count=1
- [x] 4. planMCP, planSettings, planAgents, planPackages y el enlace de skill revalidan al aplicar (packages antes de pi remove) → go test ./pkg/installer -run '^TestHCD034C03$' -count=1
- [x] 5. Sin cambios tras el plan, uninstall retira lo mismo que en v1.13.3 y el mensaje final refleja N → go test ./pkg/installer -run '^TestHCD034C05$|^TestHCD011|^TestHCD027|^TestHCD030|^TestHCD032|^TestH0' -count=1
- [x] 6. El plan registra [fallido] por cada inspección de la lista cerrada y [omitir] «no se pudo inspeccionar» en candidatas, impreso también sin pendientes → go test ./pkg/installer -run '^TestHCD034C0[678]$' -count=1
- [x] 7. La relectura al aplicar con error distinto de ausencia da [fallido] y theme se conserva con el tema fallido → go test ./pkg/installer -run '^TestHCD034C11$' -count=1
- [x] 8. docs/known-issues.md registra H-01, H-02 y H-03 como cerrados por HCD-034 y la suite completa pasa con -race → grep -q 'HCD-034' docs/known-issues.md && go vet ./... && go test -race ./... -count=1

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-13 16:27 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run ^TestHCD034C09$ -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd034_update_test.go:40:9: undefined: maxBinaryBytes
      pkg/installer/hcd034_update_test.go:41:2: undefined: maxBinaryBytes
      pkg/installer/hcd034_update_test.go:42:21: undefined: maxBinaryBytes
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-13 16:28 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run ^TestHCD034C09$ -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.296s
- 2026-11-13 16:28 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run ^TestHCD034C10$ -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.140s
- 2026-11-13 16:35 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run ^TestHCD034C0[124]$ -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.609s
- 2026-11-13 16:35 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run ^TestHCD034C03$ -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.382s
- 2026-11-13 16:35 -05:00 · tarea 5 · falló · caducada · `go test ./pkg/installer -run ^TestHCD034C05$|^TestHCD011|^TestHCD027|^TestHCD030|^TestHCD032|^TestH0 -count=1` (salida 127)
      sh: ^TestHCD011: command not found
      sh: ^TestHCD027: command not found
      sh: ^TestHCD030: command not found
      sh: ^TestHCD032: command not found
      sh: ^TestH0: command not found
- 2026-11-13 16:35 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run ^TestHCD034C0[678]$ -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.162s
- 2026-11-13 16:35 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run ^TestHCD034C11$ -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.741s
- 2026-11-13 16:35 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD034C05$|^TestHCD011|^TestHCD027|^TestHCD030|^TestHCD032|^TestH0' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	14.985s
- 2026-11-13 16:36 -05:00 · tarea 8 · verde · vigente · `grep -q 'HCD-034' docs/known-issues.md && go vet ./... && go test -race ./... -count=1` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	41.094s
- 2026-11-13 16:37 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD034C09$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.170s
- 2026-11-13 16:37 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD034C10$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.135s
- 2026-11-13 16:37 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD034C0[124]$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.420s
- 2026-11-13 16:37 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD034C03$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.482s
- 2026-11-13 16:37 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD034C05$|^TestHCD011|^TestHCD027|^TestHCD030|^TestHCD032|^TestH0' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	15.267s
- 2026-11-13 16:37 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run '^TestHCD034C0[678]$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.170s
- 2026-11-13 16:37 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run '^TestHCD034C11$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.744s

## Siguiente
Cerrada con aprobación sobre la revisión 4f05d2343927.

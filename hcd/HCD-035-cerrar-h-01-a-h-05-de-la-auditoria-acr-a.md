# Cerrar H-01 a H-05 de la auditoría ACR/ADM 2026-11-14

ID: HCD-035
Tipo: fix
Carril: full
Motivo: Carril completo: tres políticas decididas por la persona (symlink, BIN_DIR, update) que interactúan y riesgo sobre datos de instalación

## Intención
uninstall conserva el manifiesto ante fallos; cancelar pruebas mata descendientes resistentes; update relanza install con el BIN_DIR del manifiesto; update explícito ignora PI_GOMEMORY_NO_UPDATE_CHECK y nunca dice «al día» sin consultar. H-02 (configuraciones enlazadas) sale de esta unidad a HCD-036 (Q11).

Origen: `docs/audits/2026-11-14-acr-adm.md` (HEAD auditado 8fdfa1945e9f). Los cinco hallazgos se re-reprodujeron de forma independiente el 2026-11-14 en una copia `git archive` de esa revisión.

Decisiones de la persona: H-04 usa el BIN_DIR del manifiesto al relanzar, con la variable explícita por encima (Q1, Q2, Q8, Q13, Q14); H-05 hace que un `update` explícito ignore `PI_GOMEMORY_NO_UPDATE_CHECK` (Q10); H-01 conserva el manifiesto ante fallos ajenos a él (Q7 corregida por Q17, Q12, Q16, Q18). Q3–Q6 pertenecen a H-02 y pasan a HCD-036 (Q11).

Antecedentes que se respetan: HCD-030 (un launcher fuera de HOME sin manifiesto se omite y se informa), HCD-034 (cada retirada de uninstall se revalida al aplicar, y un archivo cambiado después del plan se conserva con éxito: salida 0 y «✓ desinstalado; se conservaron N…», `TestHCD034C01`/`TestHCD034C05` sin cambios, Q17) y HCD-027 (limpieza del grupo de procesos de las pruebas).

## Reproducción
Fallo en la revisión auditada, desde la raíz del repositorio y sin tocar producción:

```bash
WORK="$(mktemp -d)"
git archive 8fdfa1945e9ff4e6f0cb31296bbdb209a5e253b6 | tar -x -C "$WORK"
cp docs/audits/2026-11-14-adm-repro_test.go.txt "$WORK/pkg/installer/adm_review_test.go"
(cd "$WORK" && go test ./pkg/installer -run '^TestADM' -count=1 -v)
(cd "$WORK" && go build -ldflags='-X main.version=v0.0.1' -o "$WORK/setup" .)
PI_GOMEMORY_NO_UPDATE_CHECK=1 HOME="$WORK/home" "$WORK/setup" update; echo "exit=$?"
```

Resultado en 8fdfa1945e9f (relevantes para esta unidad):
- `TestADMManifestSurvivesFailure` FAIL: «H-01: failed uninstall removed recovery manifest».
- `TestADMResistantDetachedStdioDescendant` FAIL: «H-03: resistant descendant remains alive after parent closed and 800ms escalation elapsed».
- `TestADMCustomBinDirRetainedByChildInstall` FAIL: «H-04: update child chooses …/.local/bin instead of recorded …/custom-bin».
- `update` con la comprobación desactivada imprime «✓ pi-gomemory-setup ya está al día (v0.0.1).» y sale con 0 (H-05).

Causas verificadas en la fuente:
- H-01: `BuildUninstallPlan` planifica el manifiesto con `planPlainFile` (`pkg/installer/uninstall.go:656`), antes de `--packages` y `--self`, y lo retira aunque otra acción haya fallado.
- H-03: el manejador `close` de `streamLines` (`extensions/gomemory-tdd.ts:600-602`) cancela el SIGKILL programado tras el SIGTERM.
- H-04: `runUpdate` (`main.go:227`) relanza `install` sin BIN_DIR, y `DefaultOptions` vuelve a `~/.local/bin`.
- H-05: `CheckUpdate` devuelve `Newer=false` y `Err=nil` con la comprobación desactivada (`pkg/installer/update.go:122`), y `runUpdate` lo trata como «al día» (`main.go:211`).

Verificación del cierre: sobre el árbol de trabajo, no con el bloque anterior, que exporta un commit (S-01 de Splash 1).
- Cada tarea que cambia comportamiento registra su rojo con `mem hcd verify` antes del cambio y su verde después (Q15). C-09, C-15 y C-23 ya se cumplen en HEAD: son guardas de regresión sin rojo (Q19).
- Las reproducciones de H-01, H-03 y H-04 pasan a regresiones de producción. La de H-04 comprueba el entorno del hijo de `runUpdate`, no `DefaultOptions` (Q8, Q13).
- La tarea 1 se cierra con `go test ./... -count=1` y `go test -race ./... -count=1` en verde. Las pruebas de extensiones requieren node: si falta, fallan y no dan un verde falso (Q15).

## Alcance
Incluido: H-01, H-03, H-04 y H-05 con las políticas Q1, Q2, Q7, Q8, Q10 y Q12–Q16; el refactor de `runUpdate` para que sea inyectable en las pruebas; y las regresiones de producción. Detalle:
- H-01: la retirada del manifiesto pasa a ser la última del plan y solo se aplica si no hubo ningún `ActionFail` ajeno al manifiesto (Q12, Q17). Al conservarlo, uninstall lo avisa siempre, también sin `--verbose`; el código distinto de 0 es el que ya existe ante fallos (Q16, Q17). El cálculo del código de salida no cambia (Q18).
- H-03: la escalada a SIGKILL del grupo sobrevive al `close` del hijo directo cuando ya se pidió terminar.
- H-04: `runUpdate` pasa al hijo `PI_GOMEMORY_BIN_DIR` con el orden de Q14. Se hace testeable inyectando la ruta del ejecutable, las URL de releases y el lanzador del hijo (Q8). Se prueba en proceso con un `httptest.Server` y un ejecutable temporal, nunca el binario de la persona ni el de las pruebas.
- H-05: `update` consulta siempre; `check-update`, el launcher y el pie siguen respetando la variable (Q10).

Excluido: H-02 (pasa a HCD-036), E2E/CI, escáneres, semántica unificada de la variable, inyección por entorno en la release, y release/commit/push. Detalle:
- **H-02 y la política de configuraciones enlazadas, con Q3–Q6:** pasan a HCD-036, que se abrirá tras cerrar esta unidad y empezará por un inventario de todos los escritores de `mcp.json`, `settings.json`, `AGENTS.md` y `pi-cc-extensions.json`, incluidos `pi install`/`pi remove`, `InstallTheme`, `skills-sync` y los subcomandos `mcp`, `theme` y `plugin` (Q11).
- La ejecución E2E de Playwright y los jobs de CI en Linux.
- Los escáneres (govulncheck, staticcheck, shellcheck, gitleaks, tsc).
- Unificar qué valores de `PI_GOMEMORY_NO_UPDATE_CHECK` cuentan como «desactivada» entre Go, shell y JS, y hacer que `deps-refresh` la respete (Q10, para otra unidad).
- Que un `install` directo, sin variable, use el BIN_DIR del manifiesto: solo cambia el relanzamiento desde `update`.
- Cualquier punto de inyección de URL por variable de entorno o flag en el binario de release (Q8).
- Release, commit y push, que requieren autorización aparte.
- Los documentos de auditoría sin commit.

## ¿Qué pasa si no lo hacemos?
- Un uninstall que falla a medias borra el manifiesto: el reintento ya no encuentra un launcher en un BIN_DIR personalizado, se pierden las huellas de la instalación y aun así el comando dice «✓ desinstalado».
- Cancelar o agotar el tiempo de `/tdd` deja vivos procesos de prueba que ignoran SIGTERM, con puertos y recursos ocupados.
- `update` desde un BIN_DIR personalizado duplica el launcher en `~/.local/bin` y reescribe el manifiesto; un uninstall posterior puede omitir el original como ajeno.
- Con la comprobación desactivada, `update` afirma «al día» sin haber consultado nada y no actualiza.

## Alternativa más barata
Documentar las limitaciones: exportar siempre `PI_GOMEMORY_BIN_DIR`, reintentar el uninstall a mano y no desactivar la comprobación al actualizar. Se descarta porque H-01 y H-04 pierden datos de instalación en silencio y H-03 deja procesos vivos, y ningún aviso lo evita. Las correcciones son locales: una retirada condicional, una línea del temporizador, el BIN_DIR en el entorno del hijo y la consulta incondicional en `update`. El único cambio estructural es hacer `runUpdate` testeable.

## Historias
### US1. El manifiesto sobrevive a una desinstalación fallida
Como persona que desinstala pi-gomemory, quiero que un fallo parcial conserve el manifiesto y me lo diga, para poder reintentar y que se localicen los archivos de un BIN_DIR personalizado.
- C-01: Dado un manifiesto y un launcher en un directorio sin permiso de escritura, cuando `uninstall --yes` falla al retirar el launcher, entonces el manifiesto sigue en disco y la salida normal, sin `--verbose`, dice que se conservó por fallos previos (Q16).
- C-02: Dado un uninstall sin ningún `ActionFail` ajeno al manifiesto, ni en el plan ni al aplicar (incluidas `--packages` y `--self`), cuando se aplica, entonces el manifiesto se retira como última acción. Esto vale también si hubo archivos conservados por cambiar después del plan; en ese caso la salida sigue siendo 0 con «✓ pi-gomemory desinstalado; se conservaron N…», como fija HCD-034 (Q17).
- C-03: Dado `uninstall` en modo simulación, cuando se muestra el plan, entonces el manifiesto aparece como «se retira si no hay fallos» y no se borra nada (Q7). La acción del manifiesto sigue siendo `ActionRemove`; solo cambian su motivo y la condición al aplicarla, así que `Pending()` y las ramas de `uninstall.go:819-830` no cambian (Q21).
- C-16: Dado un plan con un `ActionFail` ajeno al manifiesto (por ejemplo `settings.json` ilegible), cuando se aplica `uninstall --yes`, entonces el manifiesto se conserva (Q17).
- C-21: Dado un uninstall que aplica acciones tras confirmar y conserva el manifiesto (no la simulación, ni la cancelación o falta de TTY, ni la salida temprana sin acciones pendientes; Q21), cuando termina, entonces la salida normal incluye el aviso de manifiesto conservado y el comando sale con el error de fallos que ya existe, sin «✓» (Q16, Q17).
- C-22: Dado un manifiesto ilegible o con JSON inválido y ningún otro fallo, cuando se aplica `uninstall --yes`, entonces sale con 1 por la inspección fallida pero retira el manifiesto, y un segundo uninstall sale con 0 (Q12, Q18). Las pruebas que provocan fallos con `chmod` (C-01, C-22) se omiten si se ejecutan como root (Q21).

### US3. Cancelar pruebas no deja procesos vivos
Como persona que ejecuta `/tdd`, quiero que cancelar o agotar el tiempo de una suite termine también a los descendientes que ignoran SIGTERM, para no quedarme con procesos huérfanos.
- C-07: Dado un descendiente que ignora SIGTERM, no comparte las tuberías del padre y permanece en el grupo de procesos de la suite (Q20), cuando se cancela la ejecución y el hijo directo termina, entonces el descendiente recibe SIGKILL y no sigue vivo 1500 ms después de la cancelación (medidos desde la cancelación, no desde el `close`), y el proceso anfitrión sigue vivo más de 800 ms después del `close` (Q21).
- C-08: Dado el mismo descendiente, cuando la suite agota su tiempo, entonces también recibe SIGKILL y no sigue vivo 1500 ms después del timeout, y el anfitrión sigue vivo más de 800 ms después del `close` (Q21).
- C-09: Dado cualquier ejecución, cuando se envía la escalada, entonces solo se señaliza el grupo de procesos de la suite y nunca al anfitrión (sigue pasando `TestHCD027ProcessGroupCleanup`; guarda sin rojo, Q19). Riesgo residual aceptado: si el grupo desaparece y su PGID se reutiliza en la ventana de 800 ms, el SIGKILL diferido podría alcanzar otro grupo.

### US4. update conserva el BIN_DIR de la instalación
Como persona que instaló pi-gomemory en un BIN_DIR personalizado, quiero que `update` reinstale en ese mismo directorio, para no acabar con launchers duplicados ni un manifiesto que apunte a otro sitio.
- C-10: Dado un manifiesto legible con `bin_dir` personalizado y `PI_GOMEMORY_BIN_DIR` vacía o sin definir, cuando `runUpdate` relanza `install` con el lanzador inyectado, entonces el hijo recibe `PI_GOMEMORY_BIN_DIR` con el `bin_dir` registrado (Q8, Q13).
- C-23: Dado `PI_GOMEMORY_BIN_DIR` apuntando a un directorio personalizado, cuando se recorre `DefaultOptions` → `InstallAll`, entonces el launcher y el manifiesto quedan en ese directorio y no se crea launcher en `~/.local/bin` (Q13, Q20; guarda sin rojo, Q19).
- C-11: Dado `PI_GOMEMORY_BIN_DIR` no vacía al ejecutar `update`, cuando se relanza `install`, entonces el hijo recibe esa variable sin cambios y el manifiesto no se lee: con un manifiesto inválido de fixture, `update` no falla (Q1, Q14, Q21).
- C-12: Dado `PI_GOMEMORY_BIN_DIR` vacía y un manifiesto inexistente o con `bin_dir` vacío, cuando `update` relanza `install`, entonces el hijo no recibe ningún BIN_DIR añadido (hereda la variable vacía tal cual, que `DefaultOptions` trata como ausente) y se usa `~/.local/bin` (Q20). La parte de `~/.local/bin` la demuestra la guarda existente `hcd003_test.go:46`. `ENOTDIR` (por ejemplo, `~/.pi-gomemory` es un archivo) cuenta como inexistente (Q21). La presencia de la marca de desinstalado no cambia nada: un manifiesto legible con `bin_dir` se usa igual (Q14).
- C-24: Dado `PI_GOMEMORY_BIN_DIR` vacía, una versión nueva disponible y un manifiesto que da cualquier error de lectura distinto de «inexistente» (permisos, JSON inválido, EISDIR, E/S; Q20), cuando se ejecuta `update`, entonces falla antes de descargar y el ejecutable queda intacto. Sin versión nueva, el manifiesto no se lee: con un manifiesto inválido de fixture, `update` dice «al día» tras consultar (Q2, Q14, Q21).

### US5. update explícito siempre consulta
Como persona que ejecuta `pi-gomemory-setup update`, quiero que consulte la última release aunque haya desactivado la comprobación automática, para no recibir un «al día» falso.
- C-13: Dado `PI_GOMEMORY_NO_UPDATE_CHECK=1` y un servidor local con una versión más nueva, cuando se ejecuta `update`, entonces se consulta el servidor y se actualiza el ejecutable temporal.
- C-14: Dado `PI_GOMEMORY_NO_UPDATE_CHECK=1` y un servidor local con la misma versión, cuando se ejecuta `update`, entonces dice «al día» solo después de haber consultado.
- C-15: Dado `PI_GOMEMORY_NO_UPDATE_CHECK=1`, cuando se ejecuta `check-update` o se comprueba la ruta pasiva de `CheckUpdate`, entonces no hay petición de red (Q10; guarda sin rojo, Q19). El launcher, el pie y la marca de desinstalado no cambian en esta unidad. `deps-update` queda cubierto por C-13 al heredar el entorno (Q20).

## Aclaraciones
- [x] Q1. C-11: si al ejecutar update existe PI_GOMEMORY_BIN_DIR apuntando a un directorio distinto del BIN_DIR del manifiesto, ¿cuál usa el install relanzado? → Variable explícita: si PI_GOMEMORY_BIN_DIR está puesta gana ella; el BIN_DIR del manifiesto solo se usa cuando la variable falta.
- [x] Q2. C-12: si update no encuentra manifiesto, o no puede leerlo, al relanzar install, ¿qué hace? → Ausente→defecto, ilegible→falla: sin manifiesto se usa PI_GOMEMORY_BIN_DIR o ~/.local/bin; si existe pero no se puede leer, update falla antes de descargar y el binario queda intacto.
- [x] Q3. C-06: si el archivo de configuración es un enlace simbólico colgante (su destino no existe), ¿qué hace la escritura? → Error explícito: ante un enlace colgante no se toca ni el enlace ni el destino y se informa de que el enlace está roto.
- [x] Q4. S-02: en uninstall, ¿qué pasa con una configuración enlazada que debería desaparecer (AGENTS.md vacío, pi-cc-extensions.json con --packages)? → «acepto todas las recomendaciones, sigue»: se conserva el enlace; AGENTS.md vacío se escribe vacío en el destino; pi-cc-extensions.json enlazado se omite con aviso; nunca se borra nada del repositorio de dotfiles.
- [x] Q5. S-03: ¿qué archivos siguen la política de enlaces? → «acepto todas las recomendaciones, sigue»: solo mcp.json, settings.json, AGENTS.md y pi-cc-extensions.json; launcher, tema, extensiones, skills, cachés y manifiesto mantienen el comportamiento actual.
- [x] Q6. S-05: ¿qué hacen install y uninstall ante un enlace colgante? → «acepto todas las recomendaciones, sigue»: install falla en ese paso y aborta, igual que con un JSON inválido; uninstall lo cuenta como fallo, lo que conserva el manifiesto.
- [x] Q7. S-06: ¿qué cuenta como fallo para conservar el manifiesto y cómo se muestra en el dry-run? → «acepto todas las recomendaciones, sigue»: cualquier ActionFail, del plan o de la aplicación, y también las omisiones por errChangedAfterPlan; el dry-run muestra «se retira si no hay fallos».
- [x] Q8. S-07/S-08: ¿dónde va el arreglo de H-04 y cómo se prueba update de extremo a extremo? → «acepto todas las recomendaciones, sigue»: en runUpdate, refactorizado para inyectar en las pruebas de Go la ruta del ejecutable, las URL y el lanzador del hijo; sin variables de entorno ni puertas en el binario de release; la prueba de H-04 comprueba el entorno del hijo, no DefaultOptions.
- [x] Q9. S-09: ¿cuándo es ilegible el manifiesto y cuándo se lee? → «acepto todas las recomendaciones, sigue»: error de permisos o JSON inválido = ilegible y update falla; bin_dir vacío o marca de desinstalado = ausente; se lee solo si hay versión nueva y antes de descargar.
- [x] Q10. S-10: ¿qué hacen los demás consumidores de PI_GOMEMORY_NO_UPDATE_CHECK? → «acepto todas las recomendaciones, sigue»: check-update, launcher y pie la siguen respetando por ser pasivos; deps-update, al llamar a update, siempre consulta; unificar los valores entre Go, shell y JS y que deps-refresh la respete quedan excluidos para otra unidad.
- [x] Q11. Splash 2: ¿se separa H-02 (política de enlaces) en su propia unidad? → «acepto todas las recomendaciones, sigue»: H-02 sale a HCD-036 (inventario de todos los escritores de los cuatro archivos, incluidos pi install/remove, InstallTheme, skills-sync y los subcomandos mcp/theme/plugin); Q3–Q6 pasan a esa unidad; HCD-035 queda con H-01, H-03, H-04 y H-05.
- [x] Q12. S-01 (Splash 2): ¿un fallo sobre el propio manifiesto impide retirarlo? → «acepto todas las recomendaciones, sigue»: no; un ActionFail cuyo objetivo es el propio manifiesto (ilegible o JSON inválido) no bloquea su retirada; cualquier otro fallo sí la bloquea.
- [x] Q13. S-02 (Splash 2): ¿cómo se demuestra C-10? → «acepto todas las recomendaciones, sigue»: con dos pruebas: runUpdate con lanzador inyectado comprueba que el hijo recibe PI_GOMEMORY_BIN_DIR con el BIN_DIR registrado; InstallAll con esa variable deja launcher y manifiesto en ese directorio.
- [x] Q14. S-11/S-06 (Splash 2): ¿la marca de desinstalado hace que update trate el manifiesto como ausente, y en qué orden se comprueba? → «acepto todas las recomendaciones, sigue»: la marca deja de contar (modifica Q9); solo cuentan como ausente el archivo inexistente o bin_dir vacío; orden: si PI_GOMEMORY_BIN_DIR no está vacía no se lee el manifiesto (Q1); si no, se lee y uno ilegible hace fallar update antes de descargar.
- [x] Q15. S-07 (Splash 2): ¿cómo se registra el rojo previo de cada tarea? → «acepto todas las recomendaciones, sigue»: cada tarea registra su rojo con mem hcd verify antes del cambio (red_missing del motor); si falta node las pruebas fallan, no dan verde falso, y se mantiene así.
- [x] Q16. S-09 (Splash 2): ¿qué informa uninstall cuando conserva el manifiesto? → «acepto todas las recomendaciones, sigue»: lo dice siempre, también sin --verbose, y termina con código distinto de 0 en lugar de «✓ desinstalado».
- [x] Q17. S-01 (Splash 3): ¿una omisión por errChangedAfterPlan conserva el manifiesto y hace fallar uninstall, contra HCD-034 C-01/C-05? → «continua» (acepta las recomendaciones de Splash 3): no; corrige Q7: solo un ActionFail (del plan o de la aplicación, salvo el del propio manifiesto) conserva el manifiesto; un archivo cambiado tras el plan no es fallo, el manifiesto se retira y uninstall sale con 0, igual que fija HCD-034; C-21 se limita a avisar de que se conservó el manifiesto, y el código distinto de 0 es el que ya existe ante fallos.
- [x] Q18. S-02 (Splash 3): con un manifiesto ilegible, ¿con qué código sale el primer uninstall? → «continua» (acepta las recomendaciones de Splash 3): sale con 1 porque la inspección falló, pero retira el manifiesto; el segundo uninstall sale con 0; no cambia el cálculo del código de salida.
- [x] Q19. S-03 (Splash 3): ¿cómo se tratan C-09, C-15 y C-23, que ya se cumplen en HEAD? → «continua» (acepta las recomendaciones de Splash 3): como guardas de regresión sin rojo, declaradas así; el rojo de Q15 se exige solo a las tareas que cambian comportamiento.
- [x] Q20. Nits de Splash 3 (S-04 a S-09): ¿cómo se precisan los criterios? → «continua» (acepta las recomendaciones de Splash 3): C-23 recorre DefaultOptions → InstallAll; C-07 exige que el descendiente permanezca en el grupo de la suite; con la variable vacía el hijo la hereda tal cual y DefaultOptions la trata como ausente; cualquier error de lectura distinto de inexistente hace fallar update; deps-update queda cubierto por C-13 al heredar el entorno; la marca de desinstalado no cambia.
- [x] Q21. Splash 4 (0 Blocker): ¿cómo se precisan los 3 Major y 5 Nit antes de aprobar? → «A, precisa y continúa»: C-21 solo en el camino que aplica acciones tras confirmar; el manifiesto sigue siendo ActionRemove y solo cambian motivo y condición al aplicar; C-07/C-08 añaden que el anfitrión sigue vivo más de 800 ms tras el close; pruebas de permisos se omiten como root; ENOTDIR cuenta como inexistente; hcd003_test.go:46 es guarda de ~/.local/bin en C-12; fixture de manifiesto inválido demuestra «no se lee»; los 1500 ms se miden desde la cancelación; la Splash de la revisión resultante se omite con esta cita.

## Slices
### VS-01. Cancelar o agotar el tiempo de /tdd mata a los descendientes resistentes sin tocar al anfitrión (H-03)
Tareas: 2, 3, 4
Criterios: C-07, C-08, C-09

### VS-02. Un uninstall con fallos conserva el manifiesto y lo avisa; sin fallos lo retira al final (H-01)
Tareas: 5, 6, 7, 8, 9
Criterios: C-01, C-02, C-03, C-16, C-21, C-22

### VS-03. update explícito consulta siempre la última release, con runUpdate testeable (H-05)
Tareas: 10, 11, 12, 13
Criterios: C-13, C-14, C-15

### VS-04. update relanza install en el BIN_DIR de la instalación (H-04)
Tareas: 14, 15, 16, 17, 18
Criterios: C-10, C-11, C-12, C-23, C-24

### VS-05. Suite completa y detector de carreras en verde sobre el árbol de trabajo
Tareas: 1
Criterios: C-09, C-15, C-23

## Checklist
- [x] 1. La reproducción ya no falla → go test ./... -count=1 && go test -race ./... -count=1
- [x] 2. Mantener el SIGKILL diferido del grupo tras el close cuando se canceló (C-07) → go test ./pkg/installer -run '^TestHCD035C07$' -count=1
- [x] 3. Matar al descendiente resistente cuando la suite agota su tiempo (C-08) → go test ./pkg/installer -run '^TestHCD035C08$' -count=1
- [x] 4. Conservar la guarda de limpieza del grupo sin señalizar al anfitrión (C-09, guarda sin rojo) → go test ./pkg/installer -run '^TestHCD027ProcessGroupCleanup$' -count=1
- [x] 5. Conservar y avisar del manifiesto cuando falla una retirada (C-01, C-21) → go test ./pkg/installer -run '^TestHCD035C01$' -count=1
- [x] 6. Retirar el manifiesto como última acción si no hubo ActionFail, también con cambios tras el plan (C-02) → go test ./pkg/installer -run '^(TestHCD035C02|TestHCD034C01|TestHCD034C05)$' -count=1
- [x] 7. Mostrar en simulación el manifiesto como «se retira si no hay fallos» (C-03) → go test ./pkg/installer -run '^TestHCD035C03$' -count=1
- [x] 8. Conservar el manifiesto ante un ActionFail del plan ajeno a él (C-16) → go test ./pkg/installer -run '^TestHCD035C16$' -count=1
- [x] 9. Retirar un manifiesto ilegible sin bloquear el reintento (C-22) → go test ./pkg/installer -run '^TestHCD035C22$' -count=1
- [x] 10. Inyectar en runUpdate (main.go) la ruta del ejecutable, las URL de releases y el lanzador del hijo (habilita C-10..C-14, C-24) → go test . -run '^TestHCD035RunUpdateInjected$' -count=1
- [x] 11. Consultar y actualizar en runUpdate (main.go) aunque PI_GOMEMORY_NO_UPDATE_CHECK=1 (C-13) → go test . -run '^TestHCD035C13$' -count=1
- [x] 12. Decir «al día» en runUpdate (main.go) solo después de consultar (C-14) → go test . -run '^TestHCD035C14$' -count=1
- [x] 13. Conservar la comprobación pasiva desactivada sin red (C-15, guarda sin rojo) → go test ./pkg/installer -run '^TestCheckUpdateDisabled$' -count=1
- [x] 14. Pasar al hijo de runUpdate (main.go) el bin_dir registrado en el manifiesto (C-10) → go test . -run '^TestHCD035C10$' -count=1
- [x] 15. Respetar en runUpdate (main.go) PI_GOMEMORY_BIN_DIR no vacía sin leer el manifiesto (C-11) → go test . -run '^TestHCD035C11$' -count=1
- [x] 16. Usar el directorio por defecto sin manifiesto, con bin_dir vacío o ENOTDIR (C-12) → go test . -run '^TestHCD035C12$' -count=1 && go test ./pkg/installer -run '^TestBinDirOverride$' -count=1
- [x] 17. Instalar launcher y manifiesto en PI_GOMEMORY_BIN_DIR vía DefaultOptions → InstallAll (C-23, guarda sin rojo) → go test ./pkg/installer -run '^TestHCD035C23$' -count=1
- [x] 18. Fallar en runUpdate (main.go) antes de descargar con un manifiesto ilegible y versión nueva (C-24) → go test . -run '^TestHCD035C24$' -count=1

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-14 06:27 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD035C07$' -count=1` (salida 1)
                  triggerUncaughtException(
                  ^
              AssertionError [ERR_ASSERTION]: C-07: el descendiente resistente sigue vivo 1500 ms después de la cancelación
              true !== false
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C073574986138/001/test.mjs:22:9 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: true,
                expected: false,
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	2.591s
      FAIL
- 2026-11-14 06:27 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD035C08$' -count=1` (salida 1)
                  triggerUncaughtException(
                  ^
              AssertionError [ERR_ASSERTION]: C-08: el descendiente resistente sigue vivo 1500 ms después del timeout
              true !== false
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C081938932884/001/test.mjs:23:9 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: true,
                expected: false,
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	3.258s
      FAIL
- 2026-11-14 06:27 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD035C07$' -count=1` (salida 1)
                  triggerUncaughtException(
                  ^
              AssertionError [ERR_ASSERTION]: C-07: el descendiente resistente sigue vivo 1500 ms después de la cancelación
              true !== false
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C074211147799/001/test.mjs:22:9 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: true,
                expected: false,
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	1.952s
      FAIL
- 2026-11-14 06:27 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD035C08$' -count=1` (salida 1)
                  triggerUncaughtException(
                  ^
              AssertionError [ERR_ASSERTION]: C-08: el descendiente resistente sigue vivo 1500 ms después del timeout
              true !== false
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C081920284425/001/test.mjs:23:9 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: true,
                expected: false,
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	3.297s
      FAIL
- 2026-11-14 06:27 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD035C07$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.938s
- 2026-11-14 06:27 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD035C08$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	3.272s
- 2026-11-14 06:27 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027ProcessGroupCleanup$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.874s
- 2026-11-14 06:28 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD035C01$' -count=1` (salida 1)
      gomemory/skills/socratica — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C011822006433/001/.pi-gomemory/skills/strategic-compact — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C011822006433/001/.pi-gomemory/skills/verify-loop — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C011822006433/001/.pi/agent/mcp.json › mcpServers.gomemory — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C011822006433/001/.pi/agent/mcp.json › mcpServers.codegraph — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C011822006433/001/.pi/agent/mcp.json › mcpServers.codebase-memory — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C011822006433/001/.pi/agent/AGENTS.md › bloque pi-gomemory — protocolo escrito por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C011822006433/001/.pi/agent/settings.json › theme — tema gomemory activado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C011822006433/001/.pi/agent/settings.json › skills (~/.pi-gomemory/skills) — registro del repositorio de skills
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C011822006433/001/.pi-gomemory/manifest.json — manifiesto de instalación
                [fallido] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C011822006433/001/.local/bin/pi-gomemory — remove /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C011822006433/001/.local/bin/pi-gomemory: permission denied
              Resumen: 38 retirados, 0 conservados, 0 omitidos, 1 fallidos.
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.637s
      FAIL
- 2026-11-14 06:28 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD035C02$' -count=1` (salida 1)
      035C021077402460/001/.pi-gomemory/skills/strategic-compact — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C021077402460/001/.pi-gomemory/skills/verify-loop — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C021077402460/001/.pi/agent/mcp.json › mcpServers.gomemory — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C021077402460/001/.pi/agent/mcp.json › mcpServers.codegraph — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C021077402460/001/.pi/agent/mcp.json › mcpServers.codebase-memory — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C021077402460/001/.pi/agent/AGENTS.md › bloque pi-gomemory — protocolo escrito por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C021077402460/001/.pi/agent/settings.json › theme — tema gomemory activado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C021077402460/001/.pi/agent/settings.json › skills (~/.pi-gomemory/skills) — registro del repositorio de skills
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C021077402460/001/.pi-gomemory/manifest.json — manifiesto de instalación
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C021077402460/001/.local/bin/pi-gomemory-setup — binario pi-gomemory-setup (--self)
              Log: /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C021077402460/001/.pi-gomemory/uninstall.log
              Resumen: 39 retirados, 0 conservados, 1 omitidos, 0 fallidos.
              ✓ pi-gomemory desinstalado; se conservaron 1 elementos que cambiaron después del plan
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.461s
      FAIL
- 2026-11-14 06:28 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD035C03$' -count=1` (salida 1)
       skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C031543301752/001/.pi-gomemory/skills/search-first — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C031543301752/001/.pi-gomemory/skills/socratica — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C031543301752/001/.pi-gomemory/skills/strategic-compact — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C031543301752/001/.pi-gomemory/skills/verify-loop — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C031543301752/001/.pi/agent/mcp.json › mcpServers.gomemory — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C031543301752/001/.pi/agent/mcp.json › mcpServers.codegraph — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C031543301752/001/.pi/agent/mcp.json › mcpServers.codebase-memory — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C031543301752/001/.pi/agent/AGENTS.md › bloque pi-gomemory — protocolo escrito por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C031543301752/001/.pi/agent/settings.json › theme — tema gomemory activado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C031543301752/001/.pi/agent/settings.json › skills (~/.pi-gomemory/skills) — registro del repositorio de skills
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C031543301752/001/.pi-gomemory/manifest.json — manifiesto de instalación
              39 acciones retirarían algo.
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.405s
      FAIL
- 2026-11-14 06:28 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD035C16$' -count=1` (salida 1)
       embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C163785141660/001/.pi-gomemory/skills/strategic-compact — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C163785141660/001/.pi-gomemory/skills/verify-loop — skill embebida intacta
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C163785141660/001/.pi/agent/mcp.json › mcpServers.gomemory — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C163785141660/001/.pi/agent/mcp.json › mcpServers.codegraph — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C163785141660/001/.pi/agent/mcp.json › mcpServers.codebase-memory — servidor MCP registrado por pi-gomemory
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C163785141660/001/.pi/agent/AGENTS.md › bloque pi-gomemory — protocolo escrito por pi-gomemory
                [conservar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C163785141660/001/.pi/agent/settings.json › theme — el archivo del tema no se pudo inspeccionar y se conserva
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C163785141660/001/.pi/agent/settings.json › skills (~/.pi-gomemory/skills) — registro del repositorio de skills
                [retirar] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C163785141660/001/.pi-gomemory/manifest.json — manifiesto de instalación
                [fallido] /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C163785141660/001/.pi/agent/themes/gomemory.json — open /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C163785141660/001/.pi/agent/themes/gomemory.json: permission denied
              Resumen: 37 retirados, 1 conservados, 0 omitidos, 1 fallidos.
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.452s
      FAIL
- 2026-11-14 06:29 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD035C01$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.538s
- 2026-11-14 06:29 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^(TestHCD035C02|TestHCD034C01|TestHCD034C05)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.356s
- 2026-11-14 06:29 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD035C03$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.397s
- 2026-11-14 06:29 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD035C16$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.463s
- 2026-11-14 06:29 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run '^TestHCD035C22$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.420s
- 2026-11-14 06:31 -05:00 · tarea 10 · rojo observado · caducada · `go test . -run '^TestHCD035RunUpdateInjected$' -count=1` (salida 1)
      # pi-gomemory [pi-gomemory.test]
      ./main_hcd035_test.go:93:32: undefined: updateDeps
      ./main_hcd035_test.go:94:9: undefined: updateDeps
      ./main_hcd035_test.go:130:12: undefined: runUpdateWith
      FAIL	pi-gomemory [build failed]
      FAIL
- 2026-11-14 06:31 -05:00 · tarea 10 · falló · caducada · `go test . -run '^TestHCD035RunUpdateInjected$' -count=1` (salida 1)
      --- FAIL: TestHCD035RunUpdateInjected (0.00s)
          main_hcd035_test.go:140: el hijo debe lanzarse con el ejecutable inyectado: ran=true exe="/private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035RunUpdateInjected2512553607/002/pi-gomemory-setup"
      FAIL
      FAIL	pi-gomemory	0.305s
      FAIL
- 2026-11-14 06:31 -05:00 · tarea 10 · verde · caducada · `go test . -run '^TestHCD035RunUpdateInjected$' -count=1` (salida 0)
      ok  	pi-gomemory	0.183s
- 2026-11-14 06:32 -05:00 · tarea 11 · rojo observado · caducada · `go test . -run '^TestHCD035C13$' -count=1` (salida 1)
      --- FAIL: TestHCD035C13 (0.00s)
          main_hcd035_test.go:156: C-13: update no consultó con la comprobación desactivada:
              ✓ pi-gomemory-setup ya está al día (v1.0.0).
      FAIL
      FAIL	pi-gomemory	0.167s
      FAIL
- 2026-11-14 06:32 -05:00 · tarea 12 · rojo observado · caducada · `go test . -run '^TestHCD035C14$' -count=1` (salida 1)
      --- FAIL: TestHCD035C14 (0.00s)
          main_hcd035_test.go:171: C-14: dijo «al día» sin consultar:
              ✓ pi-gomemory-setup ya está al día (v1.1.0).
      FAIL
      FAIL	pi-gomemory	0.138s
      FAIL
- 2026-11-14 06:32 -05:00 · tarea 14 · rojo observado · caducada · `go test . -run '^TestHCD035C10$' -count=1` (salida 1)
      --- FAIL: TestHCD035C10 (0.00s)
          main_hcd035_test.go:200: C-10: el hijo recibe PI_GOMEMORY_BIN_DIR="", quiero "/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C104283206190/003/custom-bin"
      FAIL
      FAIL	pi-gomemory	0.143s
      FAIL
- 2026-11-14 06:32 -05:00 · tarea 16 · rojo observado · caducada · `go test . -run '^TestHCD035C12$' -count=1` (salida 1)
      --- FAIL: TestHCD035C12 (0.01s)
          --- FAIL: TestHCD035C12/marca_de_desinstalado (0.00s)
              main_hcd035_test.go:254: C-12: el hijo recibe PI_GOMEMORY_BIN_DIR="", quiero "/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD035C121062209319/001/custom-bin"
      FAIL
      FAIL	pi-gomemory	0.137s
      FAIL
- 2026-11-14 06:32 -05:00 · tarea 18 · rojo observado · caducada · `go test . -run '^TestHCD035C24$' -count=1` (salida 1)
      --- FAIL: TestHCD035C24 (0.00s)
          main_hcd035_test.go:268: C-24: un manifiesto ilegible debe hacer fallar update:
              ==> Actualizando v1.0.0 → v1.1.0...
              ✓ pi-gomemory actualizado a v1.1.0.
      FAIL
      FAIL	pi-gomemory	0.127s
      FAIL
- 2026-11-14 06:32 -05:00 · tarea 10 · verde · caducada · `go test . -run '^TestHCD035RunUpdateInjected$' -count=1` (salida 0)
      ok  	pi-gomemory	0.228s
- 2026-11-14 06:32 -05:00 · tarea 11 · verde · caducada · `go test . -run '^TestHCD035C13$' -count=1` (salida 0)
      ok  	pi-gomemory	0.129s
- 2026-11-14 06:32 -05:00 · tarea 12 · verde · caducada · `go test . -run '^TestHCD035C14$' -count=1` (salida 0)
      ok  	pi-gomemory	0.125s
- 2026-11-14 06:32 -05:00 · tarea 13 · verde · caducada · `go test ./pkg/installer -run '^TestCheckUpdateDisabled$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.176s
- 2026-11-14 06:33 -05:00 · tarea 14 · verde · caducada · `go test . -run '^TestHCD035C10$' -count=1` (salida 0)
      ok  	pi-gomemory	0.123s
- 2026-11-14 06:33 -05:00 · tarea 15 · verde · caducada · `go test . -run '^TestHCD035C11$' -count=1` (salida 0)
      ok  	pi-gomemory	0.125s
- 2026-11-14 06:33 -05:00 · tarea 16 · verde · caducada · `go test . -run '^TestHCD035C12$' -count=1 && go test ./pkg/installer -run '^TestBinDirOverride$' -count=1` (salida 0)
      ok  	pi-gomemory	0.130s
      ok  	pi-gomemory/pkg/installer	0.160s
- 2026-11-14 06:33 -05:00 · tarea 18 · verde · caducada · `go test . -run '^TestHCD035C24$' -count=1` (salida 0)
      ok  	pi-gomemory	0.127s
- 2026-11-14 06:33 -05:00 · tarea 17 · verde · vigente · `go test ./pkg/installer -run '^TestHCD035C23$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.455s
- 2026-11-14 06:33 -05:00 · tarea 1 · rojo observado · caducada · `go test ./... -count=1 && go test -race ./... -count=1` (salida 1)
      ═════════════════════════════╝[0m
        [38;2;244;198;107m⚠[0m Portapapeles del sistema         [38;2;145;169;197mse requieren pbcopy y pbpaste del sistema[0m
        [38;2;255;119;127m✗[0m Binario pi                       [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario mem                      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codegraph                [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;255;119;127m✗[0m Binario codebase-memory-mcp      [38;2;145;169;197mno encontrado en PATH[0m
        [38;2;34;227;247m✓[0m Tema gomemory en Pi              [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean3074235516/001/.pi/agent/themes/gomemory.json[0m
        [38;2;244;198;107m⚠[0m MCP en Pi                        [38;2;145;169;197mno se pudo consultar `pi mcp list`[0m
        [38;2;34;227;247m✓[0m Plugin pi-antigravity (Google)   [38;2;145;169;197mregistrado en settings.json[0m
        [38;2;34;227;247m✓[0m Claude Code TUI (pi-cc-extensions) [38;2;145;169;197mactivo (diffs, summaries, /context)[0m
        [38;2;34;227;247m✓[0m Banner gomemory                  [38;2;145;169;197mcabecera gomemory activa[0m
        [38;2;34;227;247m✓[0m Skills gomemory en Pi            [38;2;145;169;197m21 skills cargadas[0m
        [38;2;145;169;197m→[0m Repositorio de skills            [38;2;145;169;197m~/.pi-gomemory/skills · 21 de gomemory, 0 del anfitrión, 0 colisiones resueltas[0m
        [38;2;34;227;247m✓[0m TUI Status Bar & Tokens          [38;2;145;169;197mextensión gomemory-status activa[0m
        [38;2;34;227;247m✓[0m Launcher Herdr (pi-gomemory)     [38;2;145;169;197m/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean3074235516/001/.local/bin/pi-gomemory[0m
      FAIL
      FAIL	pi-gomemory/pkg/installer	44.110s
      FAIL
- 2026-11-14 06:34 -05:00 · tarea 1 · verde · vigente · `go test ./... -count=1 && go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	0.200s
      ok  	pi-gomemory/pkg/installer	46.891s
      ok  	pi-gomemory	1.209s
      ok  	pi-gomemory/pkg/installer	49.523s
- 2026-11-14 06:36 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD035C07$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.105s
- 2026-11-14 06:36 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD035C08$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	3.263s
- 2026-11-14 06:36 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027ProcessGroupCleanup$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.869s
- 2026-11-14 06:37 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD035C01$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.409s
- 2026-11-14 06:37 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run '^(TestHCD035C02|TestHCD034C01|TestHCD034C05)$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.257s
- 2026-11-14 06:37 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run '^TestHCD035C03$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.381s
- 2026-11-14 06:37 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run '^TestHCD035C16$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.402s
- 2026-11-14 06:37 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run '^TestHCD035C22$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.422s
- 2026-11-14 06:37 -05:00 · tarea 10 · verde · vigente · `go test . -run '^TestHCD035RunUpdateInjected$' -count=1` (salida 0)
      ok  	pi-gomemory	0.161s
- 2026-11-14 06:37 -05:00 · tarea 11 · verde · vigente · `go test . -run '^TestHCD035C13$' -count=1` (salida 0)
      ok  	pi-gomemory	0.142s
- 2026-11-14 06:37 -05:00 · tarea 12 · verde · vigente · `go test . -run '^TestHCD035C14$' -count=1` (salida 0)
      ok  	pi-gomemory	0.126s
- 2026-11-14 06:37 -05:00 · tarea 13 · verde · vigente · `go test ./pkg/installer -run '^TestCheckUpdateDisabled$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.128s
- 2026-11-14 06:37 -05:00 · tarea 14 · verde · vigente · `go test . -run '^TestHCD035C10$' -count=1` (salida 0)
      ok  	pi-gomemory	0.125s
- 2026-11-14 06:37 -05:00 · tarea 15 · verde · vigente · `go test . -run '^TestHCD035C11$' -count=1` (salida 0)
      ok  	pi-gomemory	0.125s
- 2026-11-14 06:37 -05:00 · tarea 16 · verde · vigente · `go test . -run '^TestHCD035C12$' -count=1 && go test ./pkg/installer -run '^TestBinDirOverride$' -count=1` (salida 0)
      ok  	pi-gomemory	0.131s
      ok  	pi-gomemory/pkg/installer	0.156s
- 2026-11-14 06:37 -05:00 · tarea 18 · verde · vigente · `go test . -run '^TestHCD035C24$' -count=1` (salida 0)
      ok  	pi-gomemory	0.127s

## Siguiente
Cerrada con aprobación sobre la revisión 7ce59772e9b7.

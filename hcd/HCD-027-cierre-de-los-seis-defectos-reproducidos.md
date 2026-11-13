# Cierre de los seis defectos reproducidos en ACR y ADM 9807703

ID: HCD-027
Tipo: fix
Carril: full
Motivo: Riesgo alto de confidencialidad y pérdida de datos; requiere definir permisos de configuración, propiedad de skills y resultados y terminación de procesos TDD antes de implementar.

## Intención
Cerrar C-001 a C-006 de la revisión `acr_ff048545-8fdd-413b-a901-9542e98b20c9`: preservar archivos propios de skills y permisos privados MCP, acotar la terminación del ejecutor TDD y corregir la identidad y los estados terminales de pruebas Go y Playwright. Incorporar las seis reproducciones como regresiones permanentes y revalidar las correcciones independientemente.

La autorización para abrir esta unidad y continuar su definición no constituye aprobación de una revisión exacta ni del cierre. C-002 tiene consenso ACR; C-001 y C-003 a C-006 están reproducidos en ADM, pero siguen unilaterales en el registro ACR. Esta distinción se conserva al remediar y no se transforma una reproducción propia en corroboración independiente.

## Reproducción

Base auditada: commit `9807703e3f77853198f0a2c77d861fbb43176447`, árbol `a86027106a88351ced057bfe1dae4ec68b6041f1`. Se volvieron a ejecutar las seis reproducciones el 2026-11-12 sobre una copia aislada de ese commit: seis fallos, salida 1. El código de producción consultado conserva las causas señaladas.

Comando de reproducción disponible en esta máquina:

```sh
cd /tmp/pi-acr-OM7t3D/snapshot
go test ./pkg/installer -run '^TestACR' -count=1 -v
```

El archivo añadido a esa copia es `pkg/installer/acr_repro_test.go`; no está incorporado aún al repositorio. El directorio temporal no es una dependencia de las futuras comprobaciones HCD: las reproducciones se incorporarán a pruebas permanentes antes de corregir. Sus entradas y resultados se documentan aquí para poder reconstruirlas si desaparece.

| Hallazgo | Prueba temporal | Entrada y acción | Fallo observado |
|---|---|---|---|
| C-001 | TestACRMarkedLegacyPreservesAddedFiles | Crear una skill legacy con SKILL.md intacto, marcador de hash válido y scripts/mine.sh propio; ejecutar RegisterSkillRepo | Se elimina scripts/mine.sh |
| C-002 | TestACRMCPKeepsPrivateMode | Crear mcp.json 0600 con un servidor ajeno y un token sintético; ejecutar InstallMCPServers | El archivo termina con permisos 0644 |
| C-003 | TestACRTimeoutCannotReportSuccess | Ejecutar un hijo Node que captura SIGTERM y sale 0, con timeout de 500 ms en streamLines | Se devuelve code=0 pese al vencimiento |
| C-004 | TestACRPlaywrightProjectsDistinct | Emitir dos inicios Playwright, chromium y firefox, con la misma ubicación y resumen de dos aprobados | El almacén conserva un solo resultado |
| C-005 | TestACRInterruptedGoClearsRunning | Emitir un evento Go Action=run y terminar con exit 1 sin evento terminal | La suite falla, pero el estado global sigue running y stats.running=1 |
| C-006 | TestACRInterruptedPlaywrightNotPassed | Emitir dos inicios Playwright, un encabezado de fallo y exit 1 sin resultado para el segundo | El segundo se marca passed |

Las pruebas de parsers usan procesos y entradas sintéticas; no prueban por sí solas el contrato de un Playwright real. La retención de tuberías por descendientes y la resistencia a SIGTERM requieren casos adicionales antes de declarar cerrado C-003. El rojo HCD oficial todavía no está registrado; corresponde a la fase de implementación mediante `mem hcd verify --expect-fail`.

## Alcance
Incluido: corrección y verificación de los seis defectos reproducidos C-001 a C-006, con el siguiente detalle.
- C-001: evitar el borrado de archivos añadidos por la persona en skills legacy marcadas; conservar la retirada de copias gestionadas intactas sin contenido adicional.
- C-002: evitar ampliación de permisos privados al reescribir JSON de configuración, conservar campos ajenos y escrituras atómicas. Según Q1, conservar el modo existente sin ampliarlo y crear JSON nuevos con 0600.
- C-003: distinguir timeout y cancelación de una salida exitosa, terminar procesos controlados por el ejecutor de forma acotada y permitir que la cola continúe.
- C-004: distinguir resultados Playwright por proyecto e identidad de prueba sin perder ubicación ni diagnóstico; verificar intentos de reintento sin mezclar pruebas distintas.
- C-005: cerrar pruebas Go pendientes cuando termina su proceso y mantener coherencia entre pruebas, suite, métricas e informe.
- C-006: no aprobar pruebas Playwright sin evidencia de resultado; preservar aprobaciones reales y distinguir pruebas omitidas, interrumpidas y reintentadas.
- Pruebas permanentes de regresión, comprobaciones de integración proporcionales y revalidación independiente de los seis cierres. Registrar por separado lo confirmado en ADM y en ACR.
- Archivos candidatos: `pkg/installer/hostskills.go`, `pkg/installer/jsonfile.go`, `extensions/gomemory-tdd.ts`, adaptadores auxiliares estrictamente necesarios y pruebas en `pkg/installer/`. No es autorización para rediseñar módulos ajenos.

Excluido: cambios ajenos a los seis cierres y operaciones sobre el entorno real, con los siguientes límites.
- HCD-026 y sus cambios, sidebar y cuota Codex, C-007 a C-009, refactorizaciones oportunistas y actualizaciones de dependencias.
- Cambiar la política de symlinks de configuración o skills fuera de lo indispensable para C-001; un caso que exija cambiar esa política abre aclaración y no se resuelve de forma implícita.
- Instalación o desinstalación en HOME real, lectura de credenciales reales, llamadas autenticadas externas y ejecución de scripts remotos.
- Commit, push, publicación, instalación del artefacto en el host y cierre de la unidad sin aprobación expresa.

## ¿Qué pasa si no lo hacemos?

Una sincronización ordinaria puede eliminar contenido propio de una skill; actualizar la configuración puede volver legibles credenciales para otros usuarios locales si los directorios padre permiten acceso. El ejecutor TDD puede aprobar resultados no demostrados, mezclar proyectos, permanecer ejecutándose después de terminar o bloquear su cola tras un límite de tiempo. Los tests actuales pueden seguir verdes sin detectar estos defectos.

## Alternativa más barata

Corregir solo C-002 y posponer los otros cinco cuesta menos, pero deja riesgo reproducido de pérdida de archivos y resultados de pruebas incorrectos. Como mitigación temporal se puede evitar sincronizar skills con archivos añadidos y revisar manualmente permisos y resultados TDD; no es un cierre verificable ni reemplaza las correcciones. Se mantiene el alcance de seis defectos autorizado al abrir la unidad, sin incluir las tres observaciones bajas.

## Historias

### US1. Preservar contenido propio durante la sincronización de skills
Como persona que añade archivos a una skill instalada, quiero que la sincronización los conserve, para no perder trabajo propio.
- C-01: Dado un directorio legacy con SKILL.md intacto, marcador válido y archivos o subdirectorios añadidos, cuando se sincronizan las skills, entonces esos contenidos y sus bytes permanecen intactos.
- C-02: Dada una copia gestionada intacta que solo contiene SKILL.md y su marcador, cuando se sincroniza hacia el repositorio central, entonces se conserva la retirada prevista de la copia legacy sin borrar destinos de enlaces ni modificar carpetas propias.

### US2. Mantener privada la configuración MCP
Como persona que configura servidores MCP con credenciales, quiero que reescribir su configuración no amplíe permisos privados, para preservar la confidencialidad local.
- C-03: Dado un mcp.json existente con modo 0600 y un servidor ajeno con datos sintéticos, cuando install, mcp o la edición JSON de uninstall lo reescriben, entonces no se amplía su modo y se conservan los campos ajenos que no pertenecen a la operación.
- C-04: Dado un JSON nuevo o existente, cuando se escribe la configuración, entonces un archivo nuevo tiene modo 0600 y uno existente conserva su modo sin ampliarlo, la escritura sigue siendo atómica y un error al consultar o aplicar permisos se devuelve sin sustituir el archivo original.

### US3. Finalizar el ejecutor al vencer su límite o cancelar
Como persona que ejecuta pruebas mediante TDD, quiero que una cancelación o un vencimiento termine de forma acotada y no aparezca exitoso, para recuperar el control y continuar la cola.
- C-05: Dado un hijo que captura SIGTERM y sale con código 0, cuando vence el timeout o se solicita cancelación, entonces el resultado identifica la causa y no se considera aprobado.
- C-06: Dados un hijo que ignora SIGTERM o un descendiente ordinario que hereda las tuberías del ejecutor, cuando vence el límite o se cancela, entonces el ejecutor libera sus recursos y permite comenzar la siguiente ejecución en cola dentro de un máximo de dos segundos de gracia tras la señal; los procesos del fixture quedan terminados.

### US4. Distinguir pruebas Playwright de proyectos diferentes
Como persona que ejecuta varios proyectos Playwright, quiero resultados con identidad estable por proyecto y prueba, para contar correctamente las pruebas y conservar sus diagnósticos.
- C-07: Dados dos proyectos que ejecutan una prueba en la misma ubicación, cuando ambos terminan aprobados, entonces el almacén y el informe contienen dos resultados distintos y ninguna prueba queda running.
- C-08: Dado un fallo en un proyecto seguido por el inicio de otro en la misma ubicación, cuando termina la ejecución, entonces el fallo y su diagnóstico se conservan sin ser reiniciados por el segundo proyecto; los reintentos se asocian a una única prueba dentro de su proyecto. Si esa prueba falla y después aprueba al reintentarse, su resultado único es flaky y el diagnóstico inicial se conserva en su detalle, sin convertirlo en un fallo definitivo ni mezclarlo con otro proyecto.

### US5. Cerrar pruebas Go sin evento terminal
Como persona que consulta resultados Go, quiero que ninguna prueba siga ejecutándose después de terminar su proceso, para que las métricas y el informe reflejen el fallo real.
- C-09: Dado un evento run sin pass, fail ni skip posterior, cuando termina anormalmente el proceso, vence el límite o se cancela, entonces la prueba queda en estado terminal no exitoso con diagnóstico, stats.running es cero al acabar la suite y el estado global no queda running.
- C-10: Dados resultados Go ya terminados y otra prueba pendiente al interrumpir el proceso, cuando se cierra la ejecución, entonces se preservan los resultados y diagnósticos terminados, incluso si ya existía otro fallo.

### US6. No inventar aprobaciones Playwright
Como persona que confía en el informe TDD, quiero que cada aprobación tenga evidencia de resultado, para no confundir el inicio o la interrupción de una prueba con éxito.
- C-11: Dados dos inicios, un fallo y la interrupción de la otra prueba sin resultado, cuando se cierra la ejecución Playwright, entonces la segunda no se marca passed, el fallo conserva su diagnóstico y no queda ninguna prueba running.
- C-12: Dada una ejecución Playwright real con pruebas aprobadas, fallidas, omitidas y reintentadas, cuando se compara el informe TDD con los resultados del ejecutor, entonces las identidades, estados finales y conteos coinciden, y una salida incompleta no se interpreta como aprobación. Una prueba que falla y después aprueba cuenta una sola vez en total y en un contador flaky separado, no en passed, failed ni queued. Al terminar todas las ejecuciones, si no hay fallos definitivos y existe algún resultado flaky, la suite correspondiente y el estado global son flaky; los fallos definitivos prevalecen. El informe y el panel muestran la inestabilidad y permiten consultar el diagnóstico inicial.

## Base normativa y trazabilidad

- Constitución canónica v2.1.1: §8 (errores, límites y resiliencia), §9 y §15 (secretos y mínimo privilegio), §10 (pruebas de comportamiento), §18 (artefacto único HCD), §21 (causa raíz, simplicidad y contratos). Constitución local: enmienda SOLID del 2026-11-11; mantener responsabilidades y dependencias acotadas, sin rediseño general.
- Playbook: HCD-FLOW-01 a 03 (carril, aclaraciones y aprobación citada), HCD-DEF-01, HCD-TRACE-01, HCD-TDD-01, HCD-EVIDENCE-01 y HCD-CLOSE-01. La evidencia de este documento se registrará solo por el motor; seis reproducciones temporales no equivalen a seis tareas verificadas.
- Antecedentes: HCD-012 (memoria 111) fijó los límites de Splash y las pruebas Node; no se amplían aquí. HCD-020/HCD-021 (memoria 133) establecieron cola serializada y agrupación de watch: se preservan, sin introducir una bandera que descarte solicitudes del agente. El antiguo cierre de skills sin marca no cubre la variante marcada C-001.
- ACR origen: `acr_ff048545-8fdd-413b-a901-9542e98b20c9`, estado `consensus_ready`, sin veredicto terminal. Revalidar lo corregido con dos revisores independientes sin presentar los cinco hallazgos unilaterales como consenso previo. Si el motor ACR no admite registrar esas correcciones con su política actual, tramitar una revisión acotada separada y dejar su procedencia explícita; no alterar el consenso original para habilitar el cierre.
- Informe temporal de origen: `/tmp/pi-acr-OM7t3D/ACR-ADM.md`. Incorporar evidencia duradera y las reproducciones al repositorio durante los incrementos autorizados; no depender de /tmp para el cierre.

## Decisión de seguridad acordada

Q1 resuelta por la persona con la respuesta literal «A»: conservar el modo existente sin ampliarlo y crear JSON nuevos con 0600. No se endurecen retroactivamente los archivos existentes que sean legibles por grupo u otros; se conserva su configuración de uso compartido. No se cambia la política de symlinks ni el modo de ejecutables o temas mediante una modificación genérica del helper atómico.

## Decisión sobre reintentos acordada

Q2 resuelta por la persona con la respuesta literal «A»: una prueba que falla y después aprueba al reintentarse produce un único resultado flaky, conservando el diagnóstico inicial en su detalle. Se añade un contador flaky separado, sin clasificarla como passed, failed ni queued. Cuando ya no hay ejecuciones pendientes ni activas, la suite correspondiente y el estado global son flaky si no hay fallos definitivos; estos últimos prevalecen. Esta ampliación explícita del contrato de métricas y estado forma parte del cierre de C-004/C-006, no de una nueva capacidad ajena a la unidad.

## Aclaraciones
- [x] Q1. Q1. ¿Qué política de permisos aplicamos a los JSON de configuración? A) Conservar el modo existente sin ampliarlo y crear nuevos con 0600 (recomendado: corrección mínima, respeta usos compartidos). B) Forzar 0600 tanto a existentes como a nuevos (mayor endurecimiento, puede romper usos compartidos). En ambos casos un archivo privado existente nunca amplía permisos y no se cambia la política de symlinks ni de ejecutables. → A
- [x] Q2. Splash H1 (Major): ¿cómo representamos una prueba Playwright que falla y después aprueba al reintentarse? A) Un único resultado flaky, diagnóstico inicial conservado en su detalle, contador flaky separado (no passed, failed ni queued) y suite/estado global flaky si no hay fallos definitivos ni ejecuciones pendientes; los fallos definitivos prevalecen (recomendado: refleja la inestabilidad). B) Un único resultado passed por el último intento, conservando diagnóstico inicial como advertencia; cuenta en passed y suite/estado global passed si no hay otros fallos ni ejecuciones pendientes. Es una decisión de contrato: A añade contador y estado global explícitos, B conserva esas categorías actuales. → A

## Slices

Los incrementos conservan la definición aprobada `9747986f1af3` y las decisiones Q1/Q2. Las tareas concretas figuran en Checklist; ninguna se considera ejecutada o verificada por aparecer aquí. La tarea 1 comprobará las seis reproducciones permanentes: primero rojo observado antes de modificar producción y, después de las correcciones, verde conjunto.

### VS-01. Reescribir configuración MCP sin ampliar permisos
Tareas: 2, 3
Criterios: C-03, C-04

Entrega: el flujo de instalación, comando mcp y edición JSON de desinstalación conserva los permisos existentes y los campos ajenos; un JSON nuevo usa 0600. La escritura mantiene atomicidad y no sustituye el original si falla la consulta o aplicación de permisos.
Comprobación observable: configuración sintética 0600 conserva su modo y datos; archivos nuevos resultan 0600, archivos compartidos conservan su modo y fallos de permisos dejan el original intacto.
Alcance técnico: `pkg/installer/jsonfile.go`, sus llamadores estrictamente necesarios y pruebas en `pkg/installer/`.
Dependencias: ninguna corrección previa; puede desarrollarse en paralelo con VS-02 después del rojo inicial.

### VS-02. Sincronizar skills legacy sin perder archivos propios
Tareas: 4, 5
Criterios: C-01, C-02

Entrega: sincronizar una copia marcada con SKILL.md intacto y contenido añadido conserva ese contenido. Una copia gestionada sin extras sigue retirándose según el comportamiento existente.
Comprobación observable: scripts, notas y subdirectorios añadidos conservan sus bytes; la copia intacta sin extras se retira y no se borra el destino de un enlace.
Alcance técnico: `pkg/installer/hostskills.go` y pruebas de sincronización en `pkg/installer/`, sin cambiar la política general de symlinks.
Dependencias: ninguna corrección previa; puede desarrollarse en paralelo con VS-01 después del rojo inicial.

### VS-03. Recuperar el ejecutor TDD tras timeout o cancelación
Tareas: 6, 7
Criterios: C-05, C-06

Entrega: timeout y cancelación producen resultados no exitosos con causa explícita aunque el hijo salga 0; los procesos controlados y sus tuberías se liberan de forma acotada, permitiendo continuar la cola existente.
Comprobación observable: hijos sintéticos que capturan o ignoran SIGTERM y descendientes ordinarios que retienen tuberías terminan; la siguiente ejecución comienza dentro de los dos segundos de gracia aprobados y no quedan procesos del fixture vivos.
Alcance técnico: ciclo de vida de procesos en `extensions/gomemory-tdd.ts`, adaptadores auxiliares indispensables y pruebas en `pkg/installer/`. Preservar la serialización y agrupación de watch establecidas en HCD-020/HCD-021.
Dependencias: rojo inicial; no requiere VS-01 ni VS-02. No añade reintentos automáticos.

### VS-04. Finalizar estados Go al terminar su proceso
Tareas: 8, 9
Criterios: C-09, C-10

Entrega: todo test Go iniciado queda terminal cuando su proceso termina sin emitir un resultado; los tests ya concluidos y sus diagnósticos permanecen sin alteración.
Comprobación observable: terminación anormal, timeout y aborto dejan stats.running en cero y estado global no running, incluso cuando ya había otro fallo. Un caso mixto conserva los resultados terminados.
Alcance técnico: parser Go, almacén e informe de `extensions/gomemory-tdd.ts` y pruebas en `pkg/installer/`.
Dependencias: VS-03 para comprobar las causas reales de timeout y cancelación de extremo a extremo.

### VS-05. Informar identidades y resultados Playwright verificables
Tareas: 10, 11, 12, 13
Criterios: C-07, C-08, C-11, C-12

Entrega: proyecto y prueba tienen identidad estable; una salida incompleta no crea aprobaciones; los reintentos recuperados producen un único resultado flaky con diagnóstico inicial, contador separado y estado de suite/global según Q2. El panel y el informe reflejan ese mismo resultado.
Comprobación observable: dos proyectos en la misma ubicación generan dos resultados; un inicio posterior no borra el fallo ajeno; una interrupción deja resultados terminales no inventados; una prueba fallida y luego aprobada cuenta una vez como flaky y los fallos definitivos prevalecen. Contrastar identidades, estados y conteos con una ejecución Playwright real que incluya pruebas aprobadas, fallidas, omitidas y reintentadas.
Alcance técnico: integración Playwright, parser o adaptador de resultados, modelos, métricas, panel e informe de `extensions/gomemory-tdd.ts` y pruebas en `pkg/installer/`. La selección técnica de integración se concreta al preparar sus tareas; no se presupone aquí una dependencia nueva ni un formato externo no verificado.
Dependencias: VS-03 para casos de interrupción; ejecutar después de VS-04 para evitar cambios simultáneos sobre el almacén compartido. La prueba real es obligatoria para este cierre: si no existe un ejecutor disponible, declarar el bloqueo sin sustituirla por fixtures ni instalar dependencias por cuenta propia.

### VS-06. Demostrar los seis cierres sin regresiones
Tareas: 1, 14, 15, 16
Criterios: C-01, C-02, C-03, C-04, C-05, C-06, C-07, C-08, C-09, C-10, C-11, C-12

Entrega: las seis reproducciones permanentes pasan sobre el mismo código corregido; las comprobaciones generales y la revalidación independiente aportan evidencia durable, sin transformar los cinco hallazgos unilaterales en consenso previo ni cerrar la unidad automáticamente.
Comprobación observable: evidencia oficial del rojo inicial y verde final de la tarea 1; suite general y detector de carreras sin regresiones; dos revisores independientes verifican el delta y los criterios, con resultados y límites registrados. Actualizar el estado documentado de los hallazgos solo cuando las comprobaciones lo demuestren; no reescribir las notas históricas de v1.12.0 como si contuviera estas correcciones.
Alcance técnico: pruebas en `pkg/installer/`, registros de revisión y documentación de riesgos vigente; evidencia HCD únicamente mediante el motor.
Dependencias: el rojo inicial de la tarea 1 precede a toda corrección; su verde final y la tarea 15 requieren VS-01 a VS-05; después se ejecuta la revalidación 14 y se documenta su resultado en 16. El cierre requiere analyze y una nueva aprobación humana de la revisión de trabajo exacta.

## Checklist

Los nombres de pruebas siguientes son resultados por crear, no pruebas cuya existencia o ejecución ya se afirme. Antes de registrar un verde, comprobar con `go test ./pkg/installer -list '^TestHCD027'` que aparece cada nombre requerido: «no tests to run», un skip o una herramienta ausente no prueban el cierre. Usar datos sintéticos y HOME aislado en todas las comprobaciones. Las pruebas Go invocarán los adaptadores TypeScript con el soporte existente `runNodeExt`; las pruebas de contrato Playwright reales se distinguen de sus fixtures.

- [x] 1. Incorporar las seis reproducciones en `pkg/installer/hcd027_regressions_test.go` para demostrar que dejan de fallar tras las correcciones → `go test ./pkg/installer -run '^TestHCD027Reproductions$' -count=1 -v`; observar primero seis subcasos rojos mediante el motor con `--expect-fail`, sin fallos de compilación ni herramientas ausentes, y repetir en verde al finalizar VS-01 a VS-05
- [x] 2. Conservar el modo existente y aplicar 0600 a JSON nuevos en `pkg/installer/jsonfile.go`, manteniendo campos ajenos de MCP → `go test ./pkg/installer -run '^TestHCD027MCPModes$' -count=1`; cubrir install, mcp y edición JSON de uninstall con archivos 0600, compartidos y nuevos, según Q1
- [x] 3. Mantener intacto el original ante errores de permisos de escritura JSON en `pkg/installer/jsonfile.go` → `go test ./pkg/installer -run '^TestHCD027JSONAtomicErrors$' -count=1`; comprobar propagación de errores de consulta/aplicación y ausencia de sustitución o temporales residuales
- [x] 4. Restringir el borrado de skills legacy marcadas en `pkg/installer/hostskills.go` a copias sin contenido añadido mediante `onlySkillFile` → `go test ./pkg/installer -run '^TestHCD027SkillExtras$' -count=1`; scripts, notas y subdirectorios propios conservan sus bytes
- [x] 5. Fijar la compatibilidad de limpieza de copias legacy intactas en `pkg/installer/hcd027_regressions_test.go` → `go test ./pkg/installer -run '^TestHCD027SkillCleanupCompatibility$' -count=1`; retirar copias gestionadas sin extras sin borrar destinos de enlaces ni carpetas propias
- [x] 6. Representar timeout y aborto como causas explícitas no exitosas en `streamLines` de `extensions/gomemory-tdd.ts` → `go test ./pkg/installer -run '^TestHCD027ProcessTerminationCauses$' -count=1`; cubrir SIGTERM seguido de exit 0, aborto previo al spawn y aborto durante ejecución, manteniendo la distinción de comando ausente
- [x] 7. Acotar la terminación del grupo de procesos propio del ejecutor en `extensions/gomemory-tdd.ts` con escalado SIGTERM a SIGKILL y limpieza de tuberías → `go test ./pkg/installer -run '^TestHCD027ProcessGroupCleanup$' -count=1`; en macOS/Linux, hijo resistente y descendiente ordinario terminan, no se señalizan grupos ajenos y la siguiente ejecución en cola empieza dentro de dos segundos de gracia
- [x] 8. Cerrar como fallidas las pruebas Go pendientes al finalizar su proceso en `goTestParser` de `extensions/gomemory-tdd.ts` → `go test ./pkg/installer -run '^TestHCD027GoPendingResults$' -count=1`; salida anormal, timeout y aborto dejan diagnóstico y stats.running=0 al terminar la suite
- [x] 9. Fijar la conservación de resultados Go terminados en `pkg/installer/hcd027_tdd_test.go` → `go test ./pkg/installer -run '^TestHCD027GoPreservesResults$' -count=1`; un caso mixto conserva aprobados, omitidos y fallos con diagnóstico aunque se interrumpa otra prueba y ya exista un fallo previo
- [x] 10. Identificar cada prueba Playwright por proyecto e identidad estable, separando los intentos de su resultado único en `extensions/gomemory-tdd.ts` → `go test ./pkg/installer -run '^TestHCD027PlaywrightIdentity$' -count=1`; dos proyectos en igual ubicación producen dos resultados y no sobrescriben diagnósticos; un reintento no crea otra prueba
- [x] 11. Reconciliar resultados terminales Playwright con su informe JSON oficial en un temporal privado desde `extensions/gomemory-tdd.ts`, conservando el flujo de líneas solo para progreso → `go test ./pkg/installer -run '^TestHCD027PlaywrightTerminalResults$' -count=1`; JSON final válido determina estados, un informe parcial/ausente no aprueba pruebas y la interrupción no deja running; limpiar temporales y conservar los diagnósticos
- [x] 12. Propagar el resultado flaky de Q2 por métricas, estado, panel e informe de `extensions/gomemory-tdd.ts` → `go test ./pkg/installer -run '^TestHCD027FlakyContract$' -count=1`; una prueba recuperada cuenta una vez en total y en flaky separado, conserva diagnóstico inicial, no cuenta en passed/failed/queued y el fallo definitivo prevalece sobre flaky al terminar
- [x] 13. Contrastar el informe TDD con una ejecución Playwright real mediante `pkg/installer/hcd027_playwright_integration_test.go` → `go test ./pkg/installer -run '^TestHCD027PlaywrightIntegration$' -count=1 -v` con `PLAYWRIGHT_TEST_CLI` apuntando al ejecutor preinstalado verificado; comparar proyectos, aprobados, fallidos, omitidos, retries recuperados e interrupción, sin navegador ni API externa, registrando versión; si falta el ejecutor, declarar bloqueo y no reemplazarlo por un fixture ni instalar dependencias
- [x] 14. Revalidar independientemente el delta de los seis cierres y sus criterios con dos revisores de solo lectura, dejando informes en `docs/audits/` → observación manual declarada: ambos resultados reales sobre el mismo digest, matriz por criterio y resolución de defectos de la corrección; sin copiar conclusiones entre revisores, sin inventar consenso inicial y sin afirmar aprobación si falta un revisor
- [x] 15. Comprobar la ausencia de regresiones generales en el repositorio y los adaptadores de `pkg/installer/` → `go test -race ./... -count=1 && go vet ./... && git diff --check`; no omitir los tests HCD-027 ni convertir fallos de herramientas en éxito
- [x] 16. Documentar la matriz de cierres verificados en `docs/audits/2026-11-12-hcd027-verification.md` y enlazarla desde `docs/known-issues.md` → inspección manual declarada de la correspondencia C-001 a C-006 con pruebas, resultados de ambos revisores y límites, distinguiendo el código corregido de la release v1.12.0 histórica sin publicar una versión nueva

### Dependencias de las tareas

- Preparación de 1: crear las seis regresiones y observar el rojo antes de modificar producción. Su verde final depende de 2 a 13; no se cuenta la preparación como un cierre anticipado.
- VS-01: 2 precede a 3. VS-02: 4 precede a 5. Ambos incrementos son independientes entre sí después del rojo inicial.
- VS-03: 6 precede a 7. VS-04: 8 depende de 6/7 y 9 depende de 8.
- VS-05: 10 depende de VS-03/VS-04; 11 depende de 10; 12 depende de 11; 13 depende de 10/11/12 y del ejecutor real disponible. Verificar el esquema JSON contra esa ejecución antes de considerar estable su adaptador; no fijarlo mediante una suposición del fixture.
- VS-06: verde final de 1 y comprobación 15 tras las correcciones; 14 después de 1/13/15; 16 después de 14. Si la revalidación obliga a corregir código, repetir comprobaciones vigentes y revisión del delta afectado dentro del presupuesto autorizado.

### Disponibilidad y límites de la comprobación Playwright

Se comprobó un ejecutor ya instalado en esta máquina: `playwright/cli.js` dentro de la instalación global de `@playwright/cli`, que acepta `test` y el reporter JSON. La versión observada es `1.61.0-alpha-1781023400000`; es una herramienta externa de prueba preexistente, no una nueva dependencia ni una adopción de versión prerelease en producción. Esa versión y la ausencia de prueba contra una versión estable deben figurar como límite en la evidencia; no afirmar compatibilidad estable sin verificarla. Los tests no incorporarán rutas absolutas de esta máquina: reciben `PLAYWRIGHT_TEST_CLI` del entorno y fallan explícitamente si no está disponible. No se ejecutó la integración real en esta fase.

La selección del informe JSON evita inferir aprobaciones de líneas de inicio del reporter line y reutiliza un reporter oficial sin añadir paquetes. Los adaptadores de procesos se limitan a macOS/Linux, plataformas ya soportadas; el grupo propio nunca se confunde con el de la terminal o del agente. Si la implementación requiere alterar una decisión aprobada, se abre aclaración y no se amplía el alcance de forma implícita.

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-12 07:10 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD027Reproductions$' -count=1 -v` (salida 1)
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD027ReproductionsC006_interrupted_playwright_is_not_passe1374576663/003/test.mjs:22:8 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: 'passed',
                expected: 'failed',
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      --- FAIL: TestHCD027Reproductions (2.74s)
          --- FAIL: TestHCD027Reproductions/C001_marked_skill_preserves_added_file (0.01s)
          --- FAIL: TestHCD027Reproductions/C002_mcp_preserves_private_mode (0.01s)
          --- FAIL: TestHCD027Reproductions/C003_timeout_cannot_report_success (2.14s)
          --- FAIL: TestHCD027Reproductions/C004_playwright_projects_stay_distinct (0.21s)
          --- FAIL: TestHCD027Reproductions/C005_interrupted_go_is_terminal (0.19s)
          --- FAIL: TestHCD027Reproductions/C006_interrupted_playwright_is_not_passed (0.19s)
      FAIL
      FAIL	pi-gomemory/pkg/installer	2.920s
      FAIL
- 2026-11-12 07:18 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD027MCPModes$' -count=1` (salida 1)
      --- FAIL: TestHCD027MCPModes (1.07s)
          --- FAIL: TestHCD027MCPModes/mcp/0600 (0.01s)
              hcd027_json_test.go:73: permissions changed: got 0644, want 0600
          --- FAIL: TestHCD027MCPModes/mcp/0640 (0.01s)
              hcd027_json_test.go:73: permissions changed: got 0644, want 0640
          --- FAIL: TestHCD027MCPModes/install/0600 (0.33s)
              hcd027_json_test.go:73: permissions changed: got 0644, want 0600
          --- FAIL: TestHCD027MCPModes/install/0640 (0.34s)
              hcd027_json_test.go:73: permissions changed: got 0644, want 0640
          --- FAIL: TestHCD027MCPModes/uninstall-json/0600 (0.01s)
              hcd027_json_test.go:73: permissions changed: got 0644, want 0600
          --- FAIL: TestHCD027MCPModes/uninstall-json/0640 (0.01s)
              hcd027_json_test.go:73: permissions changed: got 0644, want 0640
          --- FAIL: TestHCD027MCPModes/new-json (0.00s)
              hcd027_json_test.go:107: new JSON permissions: got 0644, want 0600
          --- FAIL: TestHCD027MCPModes/new-mcp (0.00s)
              hcd027_json_test.go:120: new MCP permissions: got 0644, want 0600
      FAIL
      FAIL	pi-gomemory/pkg/installer	1.222s
      FAIL
- 2026-11-12 07:20 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027MCPModes$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.235s
- 2026-11-12 08:05 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027JSONAtomicErrors$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.373s
- 2026-11-12 08:09 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027MCPModes$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.713s
- 2026-11-12 20:27 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD027SkillExtras$' -count=1` (salida 1)
      --- FAIL: TestHCD027SkillExtras (0.01s)
          hcd027_skills_test.go:58: extra script was removed: open /var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD027SkillExtras1098739847/001/.pi/agent/skills/skill1/scripts/run.sh: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.169s
      FAIL
- 2026-11-12 20:28 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027SkillExtras$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.167s
- 2026-11-12 20:31 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027SkillCleanupCompatibility$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.187s
- 2026-11-12 20:37 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD027ProcessTerminationCauses$' -count=1` (salida 1)
      --- FAIL: TestHCD027ProcessTerminationCauses (1.16s)
          hcd027_process_test.go:8: node falló: exit status 1
              node:internal/modules/run_main:111
                  triggerUncaughtException(
                  ^
              AssertionError [ERR_ASSERTION]: timeout with exit 0 must be non-zero
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD027ProcessTerminationCauses1571208904/001/test.mjs:11:9 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: 0,
                expected: 0,
                operator: 'notStrictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	1.334s
      FAIL
- 2026-11-12 20:41 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027ProcessTerminationCauses$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.420s
- 2026-11-12 20:44 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027ProcessGroupCleanup$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.186s
- 2026-11-12 20:53 -05:00 · tarea 8 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD027GoPendingResults$' -count=1` (salida 1)
              + 'running'
              - 'failed'
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD027GoPendingResults1434295052/001/test.mjs:19:9
                  at ModuleJob.run (node:internal/modules/esm/module_job:569:25)
                  at async node:internal/modules/esm/loader:650:26
                  at async asyncRunEntryPointWithESMLoader (node:internal/modules/run_main:105:5) {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: 'running',
                expected: 'failed',
                operator: 'strictEqual',
                diff: 'simple'
              }
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.295s
      FAIL
- 2026-11-12 20:58 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027GoPendingResults$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.532s
- 2026-11-12 21:01 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027GoPreservesResults$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.482s
- 2026-11-12 21:08 -05:00 · tarea 10 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD027PlaywrightIdentity$' -count=1` (salida 1)
      --- FAIL: TestHCD027PlaywrightIdentity (0.22s)
          hcd027_playwright_test.go:100: node falló: exit status 1
              node:internal/modules/run_main:111
                  triggerUncaughtException(
                  ^
              AssertionError [ERR_ASSERTION]: chromium test must exist
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD027PlaywrightIdentity2766621602/003/test.mjs:25:8 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: undefined,
                expected: true,
                operator: '==',
                diff: 'simple'
              }
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.384s
      FAIL
- 2026-11-12 21:22 -05:00 · tarea 10 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027PlaywrightIdentity$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.417s
- 2026-11-12 21:24 -05:00 · tarea 11 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027PlaywrightTerminalResults$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.606s
- 2026-11-12 21:29 -05:00 · tarea 12 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027FlakyContract$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.578s
- 2026-11-12 21:34 -05:00 · tarea 13 · verde · caducada · `PLAYWRIGHT_TEST_CLI=/opt/homebrew/lib/node_modules/@playwright/cli/node_modules/playwright/cli.js go test ./pkg/installer -run '^TestHCD027PlaywrightIntegration$' -count=1 -v` (salida 0)
      === RUN   TestHCD027PlaywrightIntegration
          hcd027_playwright_integration_test.go:31: Límite verificado: ejecutor Playwright preinstalado versión "Version 1.61.0-alpha-1781023400000" (pre-release; compatibilidad estable no verificada)
          hcd027_playwright_integration_test.go:136: PASS TestHCD027PlaywrightIntegration suite real
          hcd027_playwright_integration_test.go:170: PASS TestHCD027PlaywrightIntegration interrupción real
      --- PASS: TestHCD027PlaywrightIntegration (2.42s)
      PASS
      ok  	pi-gomemory/pkg/installer	2.595s
- 2026-11-12 21:35 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027Reproductions$' -count=1 -v` (salida 0)
      === RUN   TestHCD027Reproductions
      === RUN   TestHCD027Reproductions/C001_marked_skill_preserves_added_file
      === RUN   TestHCD027Reproductions/C002_mcp_preserves_private_mode
      === RUN   TestHCD027Reproductions/C003_timeout_cannot_report_success
          hcd027_regressions_test.go:81: C003: timeout is unsuccessful
      === RUN   TestHCD027Reproductions/C004_playwright_projects_stay_distinct
          hcd027_regressions_test.go:139: C004: distinct projects retain their results
      === RUN   TestHCD027Reproductions/C005_interrupted_go_is_terminal
          hcd027_regressions_test.go:161: C005: interrupted Go test is terminal
      === RUN   TestHCD027Reproductions/C006_interrupted_playwright_is_not_passed
          hcd027_regressions_test.go:187: C006: no approval without a terminal result
      --- PASS: TestHCD027Reproductions (2.80s)
          --- PASS: TestHCD027Reproductions/C001_marked_skill_preserves_added_file (0.01s)
          --- PASS: TestHCD027Reproductions/C002_mcp_preserves_private_mode (0.00s)
          --- PASS: TestHCD027Reproductions/C003_timeout_cannot_report_success (2.16s)
          --- PASS: TestHCD027Reproductions/C004_playwright_projects_stay_distinct (0.23s)
          --- PASS: TestHCD027Reproductions/C005_interrupted_go_is_terminal (0.20s)
          --- PASS: TestHCD027Reproductions/C006_interrupted_playwright_is_not_passed (0.20s)
      PASS
      ok  	pi-gomemory/pkg/installer	2.979s
- 2026-11-13 05:08 -05:00 · tarea 15 · verde · caducada · `PLAYWRIGHT_TEST_CLI=/opt/homebrew/lib/node_modules/@playwright/cli/node_modules/playwright/cli.js go test -race ./... -count=1 && go vet ./... && git diff --check` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	28.298s
- 2026-11-13 05:09 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027Reproductions$' -count=1 -v` (salida 0)
      === RUN   TestHCD027Reproductions
      === RUN   TestHCD027Reproductions/C001_marked_skill_preserves_added_file
      === RUN   TestHCD027Reproductions/C002_mcp_preserves_private_mode
      === RUN   TestHCD027Reproductions/C003_timeout_cannot_report_success
          hcd027_regressions_test.go:81: C003: timeout is unsuccessful
      === RUN   TestHCD027Reproductions/C004_playwright_projects_stay_distinct
          hcd027_regressions_test.go:139: C004: distinct projects retain their results
      === RUN   TestHCD027Reproductions/C005_interrupted_go_is_terminal
          hcd027_regressions_test.go:161: C005: interrupted Go test is terminal
      === RUN   TestHCD027Reproductions/C006_interrupted_playwright_is_not_passed
          hcd027_regressions_test.go:187: C006: no approval without a terminal result
      --- PASS: TestHCD027Reproductions (2.81s)
          --- PASS: TestHCD027Reproductions/C001_marked_skill_preserves_added_file (0.01s)
          --- PASS: TestHCD027Reproductions/C002_mcp_preserves_private_mode (0.01s)
          --- PASS: TestHCD027Reproductions/C003_timeout_cannot_report_success (2.13s)
          --- PASS: TestHCD027Reproductions/C004_playwright_projects_stay_distinct (0.23s)
          --- PASS: TestHCD027Reproductions/C005_interrupted_go_is_terminal (0.22s)
          --- PASS: TestHCD027Reproductions/C006_interrupted_playwright_is_not_passed (0.22s)
      PASS
      ok  	pi-gomemory/pkg/installer	2.966s
- 2026-11-13 05:10 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027MCPModes$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.653s
- 2026-11-13 05:10 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027JSONAtomicErrors$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.172s
- 2026-11-13 05:10 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027SkillExtras$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.176s
- 2026-11-13 05:10 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027SkillCleanupCompatibility$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.179s
- 2026-11-13 05:10 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027ProcessTerminationCauses$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.416s
- 2026-11-13 05:10 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027ProcessGroupCleanup$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.230s
- 2026-11-13 05:10 -05:00 · tarea 8 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027GoPendingResults$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.501s
- 2026-11-13 05:11 -05:00 · tarea 9 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027GoPreservesResults$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.521s
- 2026-11-13 05:11 -05:00 · tarea 10 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027PlaywrightIdentity$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.387s
- 2026-11-13 05:11 -05:00 · tarea 11 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027PlaywrightTerminalResults$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.605s
- 2026-11-13 05:11 -05:00 · tarea 12 · verde · caducada · `go test ./pkg/installer -run '^TestHCD027FlakyContract$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.573s
- 2026-11-13 05:11 -05:00 · tarea 13 · verde · caducada · `PLAYWRIGHT_TEST_CLI=/opt/homebrew/lib/node_modules/@playwright/cli/node_modules/playwright/cli.js go test ./pkg/installer -run '^TestHCD027PlaywrightIntegration$' -count=1 -v` (salida 0)
      === RUN   TestHCD027PlaywrightIntegration
          hcd027_playwright_integration_test.go:31: Límite verificado: ejecutor Playwright preinstalado versión "Version 1.61.0-alpha-1781023400000" (pre-release; compatibilidad estable no verificada)
          hcd027_playwright_integration_test.go:140: PASS TestHCD027PlaywrightIntegration suite real
          hcd027_playwright_integration_test.go:174: PASS TestHCD027PlaywrightIntegration interrupción real
      --- PASS: TestHCD027PlaywrightIntegration (2.53s)
      PASS
      ok  	pi-gomemory/pkg/installer	2.687s
- 2026-11-13 05:20 -05:00 · tarea 14 · manual · vigente
      Completadas dos revisiones independientes de solo lectura sobre el diff 0cb6b28e8697: Revisor A (ADM defensivo en docs/audits/2026-11-12-hcd027-reviewer-a.md) concluye RESOLVED en los seis defectos sin fugas ni regresiones; Revisor B (ACR contratos en docs/audits/2026-11-12-hcd027-reviewer-b.md) valida los criterios C-01 a C-12 y las decisiones Q1/Q2. Ambos informes persistidos en docs/audits/
- 2026-11-13 05:21 -05:00 · tarea 16 · manual · vigente
      Documentada la matriz de correspondencia C-001 a C-006 en docs/audits/2026-11-12-hcd027-verification.md con pruebas reproducibles rojas y verdes, resultados de revisores A y B, y límites declarados (versión pre-release Playwright 1.61.0-alpha, alcance de procesos POSIX, y ausencia de despliegue al host). Enlazado desde docs/known-issues.md distinguiendo el código corregido de la release v1.12.0 histórica sin alterar notas ni publicar versión nueva
- 2026-11-13 05:22 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027Reproductions$' -count=1 -v` (salida 0)
      === RUN   TestHCD027Reproductions
      === RUN   TestHCD027Reproductions/C001_marked_skill_preserves_added_file
      === RUN   TestHCD027Reproductions/C002_mcp_preserves_private_mode
      === RUN   TestHCD027Reproductions/C003_timeout_cannot_report_success
          hcd027_regressions_test.go:81: C003: timeout is unsuccessful
      === RUN   TestHCD027Reproductions/C004_playwright_projects_stay_distinct
          hcd027_regressions_test.go:139: C004: distinct projects retain their results
      === RUN   TestHCD027Reproductions/C005_interrupted_go_is_terminal
          hcd027_regressions_test.go:161: C005: interrupted Go test is terminal
      === RUN   TestHCD027Reproductions/C006_interrupted_playwright_is_not_passed
          hcd027_regressions_test.go:187: C006: no approval without a terminal result
      --- PASS: TestHCD027Reproductions (2.79s)
          --- PASS: TestHCD027Reproductions/C001_marked_skill_preserves_added_file (0.01s)
          --- PASS: TestHCD027Reproductions/C002_mcp_preserves_private_mode (0.00s)
          --- PASS: TestHCD027Reproductions/C003_timeout_cannot_report_success (2.15s)
          --- PASS: TestHCD027Reproductions/C004_playwright_projects_stay_distinct (0.22s)
          --- PASS: TestHCD027Reproductions/C005_interrupted_go_is_terminal (0.20s)
          --- PASS: TestHCD027Reproductions/C006_interrupted_playwright_is_not_passed (0.20s)
      PASS
      ok  	pi-gomemory/pkg/installer	2.966s
- 2026-11-13 05:22 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027MCPModes$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.304s
- 2026-11-13 05:23 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027JSONAtomicErrors$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.169s
- 2026-11-13 05:23 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027SkillExtras$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.156s
- 2026-11-13 05:23 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027SkillCleanupCompatibility$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.156s
- 2026-11-13 05:23 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027ProcessTerminationCauses$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.367s
- 2026-11-13 05:23 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027ProcessGroupCleanup$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	2.187s
- 2026-11-13 05:23 -05:00 · tarea 8 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027GoPendingResults$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.488s
- 2026-11-13 05:24 -05:00 · tarea 9 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027GoPreservesResults$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.492s
- 2026-11-13 05:24 -05:00 · tarea 10 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027PlaywrightIdentity$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.379s
- 2026-11-13 05:24 -05:00 · tarea 11 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027PlaywrightTerminalResults$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.649s
- 2026-11-13 05:24 -05:00 · tarea 12 · verde · vigente · `go test ./pkg/installer -run '^TestHCD027FlakyContract$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.647s
- 2026-11-13 05:24 -05:00 · tarea 13 · verde · vigente · `PLAYWRIGHT_TEST_CLI=/opt/homebrew/lib/node_modules/@playwright/cli/node_modules/playwright/cli.js go test ./pkg/installer -run '^TestHCD027PlaywrightIntegration$' -count=1 -v` (salida 0)
      === RUN   TestHCD027PlaywrightIntegration
          hcd027_playwright_integration_test.go:31: Límite verificado: ejecutor Playwright preinstalado versión "Version 1.61.0-alpha-1781023400000" (pre-release; compatibilidad estable no verificada)
          hcd027_playwright_integration_test.go:140: PASS TestHCD027PlaywrightIntegration suite real
          hcd027_playwright_integration_test.go:174: PASS TestHCD027PlaywrightIntegration interrupción real
      --- PASS: TestHCD027PlaywrightIntegration (2.46s)
      PASS
      ok  	pi-gomemory/pkg/installer	2.638s
- 2026-11-13 05:24 -05:00 · tarea 15 · verde · vigente · `PLAYWRIGHT_TEST_CLI=/opt/homebrew/lib/node_modules/@playwright/cli/node_modules/playwright/cli.js go test -race ./... -count=1 && go vet ./... && git diff --check` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	35.599s

## Siguiente
Cerrada con aprobación sobre la revisión abe8c59a3b92.

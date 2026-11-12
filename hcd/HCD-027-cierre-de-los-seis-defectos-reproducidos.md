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
<!-- ### VS-01. comportamiento entregable
Tareas: 1, 2
Criterios: C-01 -->

## Checklist
- [ ] 1. La reproducción ya no falla → re-ejecutar la reproducción

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->

## Siguiente
Consultar `mem hcd show HCD-027 --json`: realizar el examen socrático y la revisión independiente Splash de la definición, y solicitar aprobación de la revisión exacta antes de preparar slices o implementar.

# Reducir el tiempo total cuando hay suites independientes, sin cambiar sus veredictos

ID: HCD-055
Tipo: evolution
Carril: full
Motivo: Decisiones interdependientes sobre cómo recuperar la política de GoMemory, declarar independencia y limitar la concurrencia sin alterar los veredictos.

## Intención
El problema reportado está en cómo Pi ejecuta las pruebas TDD mediante Bash. La unidad debe comprobar el comportamiento desde Pi; una ejecución directa en una terminal externa no basta. El caso reportado en Pi es `bash` → `mem hcd verify`: la vista de Bash muestra cuatro llamadas de HCD-054 marcadas con ✓. Las tres comprobaciones acotadas muestran 1.3 s, 1.3 s y 824 ms. La tarea 4, `mem hcd verify --task 4 --json -- 'go vet ./... && go test -race ./...'`, muestra 1 min 27 s. Estos tiempos son observaciones de la interfaz, no una medición comparativa: todavía no sabemos si las llamadas se solaparon ni cuánto tiempo se consumió en cada comando de la cadena.

Reducir el tiempo total de las verificaciones TDD lanzadas mediante una llamada Bash de Pi a `mem hcd verify`, sin cambiar sus veredictos. En esa llamada, `mem hcd verify` recibe la declaración de independencia, ejecuta las suites y registra la evidencia. Recuperar de GoMemory las decisiones aplicables al proyecto y la Constitución vigente, sin añadir un archivo de configuración por proyecto. Admitir comandos de prueba sin depender del lenguaje, informar sus resultados reales y permitir el solapamiento solo cuando la independencia esté declarada.

## Alcance
Incluido: recuperar las decisiones aplicables y la Constitución vigente desde GoMemory; ejecutar solo los comandos indicados en la llamada a `mem hcd verify`, incluidos los de ejecutores no autodetectados; GoMemory aporta la Constitución y las decisiones aplicables, no la lista de comandos; informar la salida y el estado reales por suite; limitar el paralelismo a suites declaradas independientes; verificar rojo → verde con Go, Python y un ejecutor no autodetectado; comparar tiempos y veredictos desde Pi con las mismas suites en serie y en paralelo. HCD-055 abarca ambos repositorios: `gomemory` implementa la ejecución multisuite y el registro de evidencia en `mem hcd verify`; `pi-gomemory` integra y verifica el flujo desde Bash en Pi, sin duplicar la orquestación. La evidencia de extremo a extremo identifica por versión o commit el binario `mem` que Pi ejecutó.
Excluido: añadir un archivo de configuración por proyecto, cambiar `/tdd` o `run_test_suite` como puntos de entrada, eludir las protecciones existentes de Bash en Pi, reescribir las pruebas de los proyectos, inferir independencia o cambiar las reglas de evidencia y aprobación de `mem hcd verify`.

## ¿Qué pasa si no lo hacemos?
Los proyectos pueden seguir ejecutando sus pruebas, pero las suites independientes seguirán esperando unas por otras. Un proyecto con un ejecutor no reconocido tendrá que lanzarlo fuera de este flujo. Sin verificar el contrato de salida y estado, no tendremos una garantía comprobada de que un fallo de arranque se distinga de una prueba aprobada.

## Alternativa más barata
Mantener `mem hcd verify` en serie y medir por separado los tramos de las verificaciones lentas desde Bash en Pi. No requiere añadir paralelismo, pero las suites independientes siguen sin poder solaparse.

## Antecedentes y normas aplicables
- HCD-020: se conserva el requisito de mostrar resultados reales, el progreso y el diagnóstico cuando el ejecutor los proporciona. Esta unidad amplía la selección de comandos y la orquestación; no inventa resultados cuando no hay análisis específico de la salida.
- HCD-051: se conserva la cobertura existente, incluida la de Python. Go y Python son ejemplos de validación, no requisitos para los proyectos.
- HCD-053: se mantiene separado el progreso de `mem hcd verify` de la ejecución de suites y de los refrescos de Watch.
- Constitución vigente (`mem constitution`, revisión `6a8f82da75b8`) §§8–10, 15, 18 y 21: preservar la causa del error, validar configuración, proteger secretos y entradas, probar comportamiento y mantener un único documento de unidad. Playbook vigente (`mem playbook`, revisión `fd576035ef3f`), reglas HCD-DEF-01, HCD-TRACE-01, HCD-FLOW-02, HCD-TDD-01 y HCD-EVIDENCE-01: definición verificable, aclaraciones antes de decidir y evidencia vigente.

## Historias
### US1. Ejecutar pruebas en Pi con la política vigente de GoMemory
Como responsable de un proyecto, quiero recuperar sus decisiones aplicables y la Constitución vigente desde GoMemory, para ejecutar verificaciones en Pi sin duplicar la política en otro archivo ni depender del lenguaje.
- C-01: Dado un proyecto sin archivo nuevo de configuración de pruebas y con decisiones aplicables en GoMemory, cuando se prepara una verificación TDD en Pi, entonces se recuperan esas decisiones y la Constitución vigente de GoMemory, sin leer una copia local de la política.
- C-02: Dado un comando indicado explícitamente, cuando Pi invoca `mem hcd verify` desde Bash, entonces `mem hcd verify` ejecuta el comando desde el contexto del proyecto aunque su ejecutor no sea autodetectado y registra la evidencia según las reglas vigentes.
- C-09: Dada una llamada a `mem hcd verify` sin comandos, cuando se solicita ejecutar las suites de esta unidad, entonces no se ejecuta ningún comando por autodetección ni a partir de GoMemory, y la solicitud no se registra como una verificación aprobada.
- C-08: Dado que no se puede recuperar de GoMemory la Constitución vigente o las decisiones aplicables, cuando se solicita la verificación desde Pi, entonces se emite un aviso y las suites se ejecutan en serie, incluso si la llamada solicita paralelismo. El aviso no sustituye ni altera los resultados de los comandos.
- C-10: Dada una consulta válida a GoMemory, cuando la llamada declara suites independientes, entonces una decisión aplicable que prohíba el solapamiento obliga a ejecutarlas en serie; una respuesta válida sin decisiones aplicables no se confunde con un fallo de consulta y no bloquea por sí sola el paralelismo autorizado por C-06.

### US2. Distinguir resultados de pruebas de errores del ejecutor
Como persona que ejecuta pruebas en Pi, quiero ver la salida y el estado que produjo cada comando, para distinguir una prueba fallida de un error al arrancar el ejecutor.
- C-03: Dado un comando iniciado que imprime información en la salida estándar o de error y termina con código distinto de cero, cuando se informa su resultado, entonces se muestra la salida disponible atribuida a la suite y el código real, y la suite no aparece aprobada.
- C-04: Dado que `mem hcd verify` no logra iniciar el proceso que debe observar, cuando informa el resultado de la suite, entonces distingue el error de arranque de una salida no cero y no le asigna un código de salida a un proceso que no arrancó. En el modo de varias suites, `mem hcd verify` inicia y observa directamente el ejecutable explícito de cada suite, sin envolverlo automáticamente en Bash. Si el proceso observado sí inicia y devuelve un código, se informa ese código real. El comando único conserva su ejecución y semántica vigentes: un `127` emitido por Bash es una salida de Bash, no un fallo al iniciar el proceso observado; `--expect-fail` mantiene el tratamiento vigente de las salidas del comando único, y no cuenta como rojo un fallo al iniciar el proceso observado.
- C-05: Dadas pruebas controladas de Go, Python y un ejecutor no autodetectado, cuando se ejecuta la validación rojo → verde, entonces se registra una aserción del nuevo contrato que falla antes del cambio y pasa después; la validación distingue el éxito, el fallo con salida y el fallo de arranque. El verde confirma que se informa correctamente cada caso, no que las suites preparadas para fallar dejen de hacerlo.
- C-11: Dada una verificación normal con varias suites, cuando finaliza `mem hcd verify`, entonces el resultado global que usa la evidencia de la tarea solo es aprobado si todas las suites pasan. Un fallo de prueba o de arranque impide aprobar la tarea, sin ocultar el estado individual de las suites.
- C-12: Dada una llamada con `--expect-fail` y varias suites, cuando se valida la solicitud, entonces se rechaza antes de iniciar cualquier suite. Se conserva el uso existente de `--expect-fail` con comando único, sin contar un fallo de arranque como rojo esperado, y no cambia el modo `--manual`.

### US3. Acortar el tiempo sin alterar veredictos
Como persona que ejecuta varias suites, quiero que solo las independientes puedan solaparse, para terminar antes sin alterar los resultados de las demás.
- C-06: Dadas suites declaradas independientes explícitamente en la llamada actual y otras sin declaración, cuando se ejecuta el conjunto con concurrencia habilitada, entonces solo las declaradas en esa llamada pueden solaparse dentro del límite. Una suite sin declaración conserva su posición respecto de las demás: no se solapa con otra ni se adelanta una suite posterior para cruzarla. Ninguna declaración anterior autoriza una llamada posterior. Para las suites elegibles, el máximo es de dos simultáneas por defecto. El ajuste explícito acepta solo los enteros `1` y `2` para la llamada actual: `1` ejecuta en serie y `2` no habilita el paralelismo sin una declaración de independencia. Un valor inválido se rechaza antes de iniciar cualquier suite.
- C-07: Dado el mismo conjunto de suites y comandos, con el mismo entorno y condiciones de caché comparables, cuando se mide su ejecución en serie y en paralelo, entonces se registra la duración de cada llamada Bash a `mem hcd verify` desde Pi, del inicio hasta su retorno, junto con los veredictos por suite y el global. En repeticiones comparables con las mismas suites independientes controladas y ejecutadas desde Pi, los veredictos por suite y el global coinciden entre modos, y la mediana del tiempo total en paralelo es menor que la mediana en serie. HCD-054 no se usa como línea base ni se exige demostrar una mejora de ese caso.

## Aclaraciones
- [x] Q1. ¿Dónde y en qué formato debe declarar cada proyecto sus comandos de prueba: en un archivo JSON dedicado en la raíz, en la configuración existente del proyecto si admite una sección de pruebas, o en configuración local del usuario asociada al proyecto? → sin archivo pro proyecto, se debe traer desde gomemmory las deceisiones y constitucion
- [x] Q2. ¿Cómo se autoriza el paralelismo al recuperar las decisiones de GoMemory: con una declaración explícita en cada ejecución, con una regla estructurada y aprobada reutilizable por el proyecto, o admitiendo ambos mecanismos? La Constitución y las decisiones se consultan en las tres opciones. → 1
- [x] Q3. ¿Cómo se fija el máximo de suites simultáneas por ejecución: máximo de 2 por defecto con ajuste explícito y acotado por llamada, o máximo obligatorio en cada llamada que solicita paralelismo? → 1
- [x] Q4. Si GoMemory no permite recuperar la Constitución vigente o las decisiones aplicables, ¿debe bloquearse solo el modo paralelo y conservarse la verificación en serie con aviso, o debe detenerse toda verificación de este flujo hasta recuperar el contexto? → 1
- [x] Q5. ¿Qué entrada desde Pi ejecutará los comandos y recibirá la declaración de independencia: una llamada Bash a mem hcd verify que orquesta y registra la evidencia, o una herramienta de Pi que orquesta y deja a mem hcd verify la ejecución de la comprobación y el registro de evidencia? → 1
- [x] Q6. ¿De dónde obtiene mem hcd verify la lista de comandos: solo de la llamada explícita, de la llamada con respaldo en decisiones de GoMemory, o de la llamada con respaldo en autodetección? En los tres casos, GoMemory sigue siendo la fuente de la Constitución y las decisiones aplicables. → 1
- [x] Q7. Con dos suites simultáneas por defecto, ¿qué rango explícito admite mem hcd verify por llamada: de 1 a 2, de 1 a 4, o de 1 hasta el número de CPU lógicas del entorno (con un mínimo de 2 para conservar el valor por defecto)? Los valores fuera del rango se rechazan antes de ejecutar. → 1
- [x] Q8. ¿Confirmas la regla para GoMemory propuesta tras Splash: una decisión aplicable que prohíba el solapamiento fuerza la serie; una consulta válida sin decisiones solo permite paralelismo si la llamada declara independencia; una consulta fallida fuerza la serie con aviso? → apruebo
- [x] Q9. Para el modo multisuite, ¿conservamos --expect-fail solo para el comando único existente, o también lo admitimos en multisuite exigiendo que todas las suites arranquen y terminen con fallo? En ambos casos, el modo normal solo aprueba si todas pasan, un error de arranque nunca cuenta como rojo esperado y --manual no cambia. → 1
- [x] Q10. ¿Qué proceso se observa: (1) ejecución directa para las suites nuevas y comportamiento actual para el comando único, o (2) ejecución de las suites nuevas mediante Bash y uso de su código de salida? → 1
- [x] Q11. ¿Qué carga y regla prueban el ahorro en C-07: (1) suites independientes controladas ejecutadas desde Pi, con mediana del tiempo total menor en paralelo y veredictos idénticos, o (2) la misma regla y además una mejora en una carga real representativa del caso reportado, cuya independencia debe acreditarse primero? → 1
- [x] Q12. ¿La unidad incluye (1) motor GoMemory e integración Pi en una sola HCD con cambios por repositorio, o (2) una unidad de motor separada y HCD-055 como consumidora? En ambos casos se identifica el binario mem verificado desde Pi. → 1

## Slices
<!-- ### VS-01. comportamiento entregable
Tareas: 1, 2
Criterios: C-01 -->

## Checklist

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->

## Siguiente
Cancelada: no hemos llegada a nada.. cancela la unidad

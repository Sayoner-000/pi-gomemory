# Mostrar progreso por tarea HCD en el panel TDD

ID: HCD-053
Tipo: fix
Carril: fast
Motivo: Fallo de visualización reproducible y alcance acotado al panel; no cambia contratos del motor HCD ni el resultado de las suites normales.

## Intención
Al ejecutar mem hcd verify desde Bash en Pi, mostrar en /tdd una fila por tarea con glifo desde el inicio, actualizarla con el resultado real y conservarla durante los refrescos de Watch.

## Reproducción
En Pi con la extensión de v1.16.0 y `/tdd` visible, activar Watch y ejecutar desde la herramienta Bash varias verificaciones válidas `mem hcd verify --task N --json -- '<comprobación>'`, una por llamada. En HCD-052 se observó que Bash mostró el éxito de las tareas 1 a 3, pero el panel no mostró una fila por tarea durante la ejecución. El listener actúa al recibir `tool_result`, reutiliza una sola fila de Data & Graph y un refresco de Watch puede reemplazarla. HCD-051 incorporó la escucha de resultados y HCD-052 corrigió la atribución de éxitos; esta unidad cubre el progreso visible y su conservación.

## Alcance
Incluido: mostrar en `/tdd` una fila por tarea de la unidad HCD activa que se verifica desde Bash; dibujar `⟳` al iniciar y `✓` o `✗` al finalizar según el resultado real; conservar las filas cuando Watch refresque las suites sin mezclar su estado con Backend ni con Data & Graph; validar la secuencia y el resultado en Pi.
Excluido: cambiar el motor o el formato de evidencia de `mem hcd verify`, modificar `/tasks`, alterar las suites normales o sus totales, conservar historial entre sesiones de Pi, instalar o publicar el cambio.

## ¿Qué pasa si no lo hacemos?
La persona ve el resultado en Bash o en `/tasks`, pero no puede seguir en `/tdd` qué verificación HCD está corriendo ni consultar las anteriores después de un refresco de Watch. El total de Backend no representa esas verificaciones.

## Alternativa más barata
Consultar Bash para ver la llamada en curso y `/tasks` para ver las tareas verificadas. No requiere cambios, pero no ofrece progreso y resultados por tarea en el panel `/tdd`.

Normas aplicables: Constitución §§10, 18 y 21; Playbook HCD-DEF-01, HCD-TDD-01 y HCD-EVIDENCE-01.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. La reproducción ya no falla → re-ejecutar la reproducción

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-19 19:59 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run ^TestHCD053LiveHCDRows$ -count=1` (salida 1)
      --- FAIL: TestHCD053LiveHCDRows (0.15s)
          hcd053_test.go:6: node falló: exit status 1
              node:internal/modules/run_main:111
                  triggerUncaughtException(
                  ^
              AssertionError [ERR_ASSERTION]: debe repintar sin esperar el resultado de Bash
                  at file:///private/var/folders/34/jwxjynms09b21_4rb72hqxk80000gn/T/TestHCD053LiveHCDRows314933195/001/test.mjs:38:8 {
                generatedMessage: false,
                code: 'ERR_ASSERTION',
                actual: false,
                expected: true,
                operator: '==',
                diff: 'simple'
              }
              Node.js v26.13.0
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.500s
      FAIL
- 2026-11-19 20:01 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run ^TestHCD053LiveHCDRows$ -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.388s

## Siguiente
Cerrada con aprobación sobre la revisión b6f72255139f.

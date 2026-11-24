# Registrar excepciones de la constitución para pi-gomemory

ID: HCD-082
Tipo: technical
Carril: fast
Motivo: Cambio documental de gobierno sin decisiones abiertas interdependientes ni cambio de comportamiento

## Intención
Que constitution.md del proyecto registre, según §23 de la constitución de gomemory, las excepciones que aplican a pi-gomemory como harness/instalador (install.sh en la raíz como contrato público, arquitectura hexagonal pendiente, documentos §18 de servicio) y corrija su referencia obsoleta a la versión canónica


## Contexto
La constitución que sirve `mem constitution` (v2.2.0) es la plantilla por defecto de gomemory (`mem docs list` → `constitution · por defecto`), pensada para servicios: contenedores (§2), arquitectura hexagonal (§3, §4), documentos de servicio (§18) y árbol de servicio (§19). pi-gomemory es un harness: un binario instalador con recursos en `embed.FS` más extensiones de Pi. Según §0, el `constitution.md` del proyecto (nivel 3 de precedencia) puede extender la plantilla, pero solo puede relajarla con excepciones que cumplan §23. Hoy no registra ninguna y cita una versión obsoleta («v2.1.1»).

## Alcance
Incluido: reescribir la cabecera de `constitution.md` para explicar las capas (plantilla de gomemory → extensiones y excepciones de este proyecto) sin fijar una versión; añadir la sección «Excepciones» con los campos de §23 para E-01 `install.sh` en la raíz (§19), E-02 recursos embebidos y `embed.go` en la raíz (§19), E-03 arquitectura hexagonal pendiente (§3, §3.1-D, §4, §19 sobre el directorio internal, que aún no existe), E-04 contenerización no aplicable (§2, §19) y E-05 documentos de §18; crear `docs/adr/0005-excepciones-constitucion-harness.md` e indexarlo en `docs/adr/README.md`; añadir `pkg/installer/hcd082_test.go`, que exige los campos de §23 en cada excepción.
Excluido: mover archivos o reestructurar el código Go (migración hexagonal = HCD B, futuro); modificar la plantilla de gomemory (`mem docs import`); CI/CD de §17 y la línea base de herramientas de §1.2 (golangci-lint, gofumpt, Testify, `context.Context` en I/O), que se revisan en otra unidad; renombrar `docs/architecture.md`.

## ¿Qué pasa si no lo hacemos?
El proyecto sigue incumpliendo la plantilla sin dejar constancia. §23 dice que una excepción no registrada o vencida bloquea el merge, y cada ACR o ADM vuelve a reportar como defecto lo que en realidad es una decisión de diseño de un harness. Además, la referencia «v2.1.1» induce a error sobre qué norma rige.

## Alternativa más barata
Adaptar la plantilla con `mem docs import constitution`: es más rápido, pero relaja la norma en la memoria del proyecto sin la trazabilidad de §23 (responsable, vencimiento, ADR) y la deja fuera del repositorio. Se descarta. Tampoco basta con una nota suelta en el README, porque §0 y §23 ubican las excepciones en `constitution.md`.


## Aclaraciones
<!-- - [ ] Q1. pregunta abierta   ·   - [x] Q1. pregunta → respuesta -->


## Checklist
- [x] 1. Añadir la prueba que exige las excepciones E-01 a E-05 con los campos de §23, el ADR-0005 indexado y la cabecera sin «v2.1.1» → `go test ./pkg/installer -run '^TestHCD082' -count=1`, primero en rojo
- [x] 2. Registrar en `constitution.md` la cabecera por capas y las excepciones E-01 a E-05 → `go test ./pkg/installer -run '^TestHCD082ConstitutionExceptions$' -count=1`
- [x] 3. Crear el ADR-0005 e indexarlo en `docs/adr/README.md` → `go test ./pkg/installer -run '^TestHCD082ADR$' -count=1`
- [x] 4. Regresión completa → `go vet ./... && go test ./... -count=1`

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-24 18:40 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD082' -count=1` (salida 1)
      --- FAIL: TestHCD082ConstitutionExceptions (0.00s)
          hcd082_test.go:44: la cabecera no debe fijar la versión obsoleta v2.1.1 de la plantilla
          hcd082_test.go:48: constitution.md debe mencionar "mem constitution"
          hcd082_test.go:48: constitution.md debe mencionar "§23"
          hcd082_test.go:48: constitution.md debe mencionar "## Excepciones"
          hcd082_test.go:74: falta la excepción E-04
          hcd082_test.go:74: falta la excepción E-05
          hcd082_test.go:74: falta la excepción E-01
          hcd082_test.go:74: falta la excepción E-02
          hcd082_test.go:74: falta la excepción E-03
      --- FAIL: TestHCD082ADR (0.00s)
          hcd082_test.go:92: no se pudo leer docs/adr/0005-excepciones-constitucion-harness.md: open ../../docs/adr/0005-excepciones-constitucion-harness.md: no such file or directory
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.333s
      FAIL
- 2026-11-24 18:41 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD082ConstitutionExceptions$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.162s
- 2026-11-24 18:41 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD082ADR$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.180s
- 2026-11-24 18:41 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestHCD082' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.150s
- 2026-11-24 18:41 -05:00 · tarea 4 · verde · caducada · `go vet ./... && go test ./... -count=1` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	49.452s
- 2026-11-24 18:42 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD082ConstitutionExceptions$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.179s
- 2026-11-24 18:42 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestHCD082' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.181s
- 2026-11-24 18:42 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD082ConstitutionExceptions$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.164s
- 2026-11-24 18:42 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD082ADR$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.164s
- 2026-11-24 18:42 -05:00 · tarea 4 · verde · vigente · `go vet ./... && go test ./... -count=1` (salida 0)
      ?   	pi-gomemory	[no test files]
      ok  	pi-gomemory/pkg/installer	49.673s

## Siguiente
Cancelada: borra la constitucion de este y cancela HCD-082

# Auditorías

Los informes conservan la revisión, alcance, reproducciones y límites de cada auditoría. Un resultado sobre un snapshot de desarrollo no certifica una release posterior. El estado de las incidencias se mantiene en [problemas conocidos](../known-issues.md).

## Revisiones recientes

- [ACR y ADM de v1.16.0](2026-11-19-acr-adm-v1.16.0.md): cinco defectos confirmados en `lite`, `/tdd` y `/tasks`. HCD-052 incorpora sus correcciones en v1.17.0; el informe y su veredicto corresponden a v1.16.0.

- [ACR y ADM de v1.15.1](2026-11-18-acr-adm-v1.15.1.md): cinco hallazgos (dos medios y tres bajos) sobre `gomemory-sidebar.ts`, `gomemory-tdd.ts`, `gomemory-tasks.ts`, `uninstall.go` y `skills.go`. Corregidos en v1.15.2 (HCD-048).
- [ACR de v1.15.0](2026-11-17-acr-v1.15.0.md): ocho hallazgos medios reproducidos con el binario de la release, con [reproductor](2026-11-17-acr-v1.15.0-repro.mjs.txt), [salida](2026-11-17-acr-v1.15.0-reproductions.log), [diff revisado](2026-11-17-acr-v1.15.0-target.md) y revisores [A](2026-11-17-acr-v1.15.0-reviewer-a.json) y [B](2026-11-17-acr-v1.15.0-reviewer-b.json). Corregidos en v1.15.1 (HCD-047); el informe describe v1.15.0 y no se ha modificado.
- [HCD-045 en implementación](2026-11-17-acr-adm-hcd045.md): ACR incompleto sobre un snapshot congelado; incluye pruebas y resultados parciales.
- [Implementaciones v1.14](2026-11-17-acr-adm-v1.14.md): defectos medios y observaciones sobre codemode y dependencias.
- [Ronda 4 posterior a los cierres anteriores](2026-11-17-acr-adm-round4.md).
- [Ronda 3](2026-11-17-acr-adm-round3.md), [seguimiento](2026-11-17-acr-adm-followup.md) y [revisión inicial](2026-11-17-acr-adm.md).
- [Auditoría defensiva de v1.13.7](2026-11-17-defensive-qa.md).

## Historial

Los demás informes y sus adjuntos se conservan en esta carpeta. Los archivos de reproducción `.go.txt` y `.mjs.txt` son fixtures: antes de ejecutarlos, revisa las rutas temporales, la revisión requerida y el aislamiento que especifica cada informe. Los logs de fallos son evidencia histórica, no pruebas ejecutadas contra la versión vigente.

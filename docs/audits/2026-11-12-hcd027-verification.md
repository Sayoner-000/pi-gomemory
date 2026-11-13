# Matriz de Verificación y Cierre de Defectos — HCD-027

**Fecha:** 2026-11-12  
**Unidad HCD:** HCD-027 (Carril: Full · Tipo: Fix)  
**Definición Aprobada:** `9747986f1af3` (aprobada con cita «apruebo 9747986f1af3»)  
**Revisión de Trabajo:** `abe8c59a3b92`  
**Base HEAD:** `2d63b8d231522224d30215bbed1205f051afa7c7` (rama main)  
**Digest del Diff:** `0cb6b28e8697480d8a75c6151a1e37029ce13c8083a78890c55d0f7020388f6c`  

---

## 1. Distinción Histórica con la Release v1.12.0

La release **v1.12.0** publicada en el repositorio contiene exclusivamente la funcionalidad de cuota de OpenAI Codex (HCD-026). Las correcciones de los seis defectos reproducidos en la auditoría ACR/ADM de `9807703` corresponden a esta unidad **HCD-027** y residen en el árbol de trabajo actual. **No se publica una versión nueva ni se alteran las notas de versión históricas de v1.12.0**.

---

## 2. Matriz de Correspondencia C-001 a C-006

| Defecto | Clasificación ACR/ADM original | Causa Raíz | Archivo Corregido | Prueba de Reproducción (Rojo/Verde) | Prueba Exhaustiva / Específica | Estado Final |
|---|---|---|---|---|---|---|
| **C-001** | Media (ADM confirmado / ACR unilateral) | `pruneLegacyPiSkills` eliminaba directorios con `isPristineEmbeddedCopy` sin verificar si el usuario había añadido archivos extras | `pkg/installer/hostskills.go` | `TestHCD027Reproductions/C001` | `TestHCD027SkillExtras`, `TestHCD027SkillCleanupCompatibility` | **CERRADO** |
| **C-002** | Alta (Confirmado ACR A+B y ADM) | `writeJSONObject` forzaba `0644` incondicionalmente, ampliando permisos de lectura local en `mcp.json` privados | `pkg/installer/jsonfile.go` | `TestHCD027Reproductions/C002` | `TestHCD027MCPModes`, `TestHCD027JSONAtomicErrors` | **CERRADO** |
| **C-003** | Media (ADM confirmado / ACR unilateral) | `streamLines` no distinguía timeout si el hijo capturaba SIGTERM y salía con código 0; procesos resistentes colgados | `extensions/gomemory-tdd.ts` | `TestHCD027Reproductions/C003` | `TestHCD027ProcessTerminationCauses`, `TestHCD027ProcessGroupCleanup` | **CERRADO** |
| **C-004** | Media (ADM confirmado / ACR unilateral) | `playwrightLineParser` descartaba el nombre del proyecto al extraer `loc`, colapsando pruebas de proyectos distintos en la misma ubicación | `extensions/gomemory-tdd.ts` | `TestHCD027Reproductions/C004` | `TestHCD027PlaywrightIdentity` | **CERRADO** |
| **C-005** | Media (ADM confirmado / ACR unilateral) | `goTestParser` no cerraba pruebas iniciadas con `run` si el proceso terminaba sin evento terminal, dejando `running` en panel | `extensions/gomemory-tdd.ts` | `TestHCD027Reproductions/C005` | `TestHCD027GoPendingResults`, `TestHCD027GoPreservesResults` | **CERRADO** |
| **C-006** | Alta (ADM confirmado / ACR unilateral) | `playwrightLineParser` aprobaba automáticamente cualquier prueba iniciada que no estuviese en la lista de fallos | `extensions/gomemory-tdd.ts` | `TestHCD027Reproductions/C006` | `TestHCD027PlaywrightTerminalResults`, `TestHCD027FlakyContract` | **CERRADO** |

---

## 3. Resultados de los Revisores Independientes

1. **Revisor A (Auditoría Defensiva Multilenguaje / Estructura y Procesos):**
   - Informe en `docs/audits/2026-11-12-hcd027-reviewer-a.md`.
   - Veredicto: RESOLVED para los 6 defectos. Comprobada programación defensiva, ausencia de fugas de recursos y propagación correcta de errores.
2. **Revisor B (Revisión por Consenso / Contratos y Criterios):**
   - Informe en `docs/audits/2026-11-12-hcd027-reviewer-b.md`.
   - Veredicto: SATISFECHO para los 12 criterios C-01 a C-12 y las aclaraciones Q1 (permisos JSON) y Q2 (contrato Flaky).

---

## 4. Límites Declarados

1. **Herramienta Playwright:** Se comprobó mediante el ejecutor real verificado en esta máquina (`Version 1.61.0-alpha-1781023400000`). No se ha ejecutado contra todas las versiones estables de Playwright; el diseño se apoya en el esquema estándar del reporter oficial `json`.
2. **Entornos de Procesos:** La gestión de grupos de procesos (`killProcessGroup` con señal a `-pid` y escalado a `SIGKILL` en <2s) aplica a plataformas POSIX (macOS y Linux).
3. **Sin Despliegue ni Release Host:** Esta verificación cubre exclusivamente el código fuente y las pruebas en el repositorio. No instala ni altera binarios en el host ni publica release de GitHub.

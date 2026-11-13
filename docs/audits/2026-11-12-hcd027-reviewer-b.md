# Informe de Revalidación Independiente — Revisor B (ACR / Contratos y Criterios)

**Fecha:** 2026-11-12  
**Unidad:** HCD-027 (`9747986f1af3` / `abe8c59a3b92`)  
**Base:** HEAD `2d63b8d231522224d30215bbed1205f051afa7c7` (rama main)  
**Digest del Diff:** `0cb6b28e8697480d8a75c6151a1e37029ce13c8083a78890c55d0f7020388f6c`  
**Rol:** Revisor B (Revisión de Contratos, Criterios y Especificación) — Solo Lectura  

---

## 1. Matriz de Criterios de Aceptación (C-01 a C-12)

| Criterio | Descripción | Estado | Evidencia de Verificación |
|---|---|---|---|
| **C-01** | Conservación de archivos/subcarpetas propios en skills legacy marcadas | SATISFECHO | `TestHCD027SkillExtras` y subtest `C001` de `TestHCD027Reproductions`. |
| **C-02** | Retirada de copias legacy sin extras sin afectar enlaces ni carpetas propias | SATISFECHO | `TestHCD027SkillCleanupCompatibility`. |
| **C-03** | Preservación de modo 0600 y campos ajenos en reescritura MCP | SATISFECHO | `TestHCD027MCPModes` y subtest `C002` de `TestHCD027Reproductions`. |
| **C-04** | Modo 0600 en nuevos JSON, atomicidad y propagación de errores | SATISFECHO | `TestHCD027JSONAtomicErrors`. |
| **C-05** | Fallo explícito y no exitoso ante timeout/cancelación aunque hijo salga 0 | SATISFECHO | `TestHCD027ProcessTerminationCauses` y subtest `C003`. |
| **C-06** | Liberación acotada de grupos de procesos en <2s de gracia ante procesos resistentes | SATISFECHO | `TestHCD027ProcessGroupCleanup`. |
| **C-07** | Identidad diferenciada por proyecto en misma ubicación Playwright | SATISFECHO | `TestHCD027PlaywrightIdentity` y subtest `C004`. |
| **C-08** | Preservación de fallos ajenos, reintentos asociados a un único resultado | SATISFECHO | `TestHCD027PlaywrightIdentity` y `TestHCD027FlakyContract`. |
| **C-09** | Cierre de pruebas Go pendientes ante terminación anormal/timeout/aborto con stats.running=0 | SATISFECHO | `TestHCD027GoPendingResults` y subtest `C005`. |
| **C-10** | Preservación de pruebas Go ya concluidas y diagnósticos ante interrupción posterior | SATISFECHO | `TestHCD027GoPreservesResults`. |
| **C-11** | No inventar aprobaciones Playwright ante salidas incompletas; diagnóstico conservado | SATISFECHO | `TestHCD027PlaywrightTerminalResults` y subtest `C006`. |
| **C-12** | Contrato Flaky de Q2: 1 resultado flaky con diagnóstico inicial, contador separado, prevalencia de fallos definitivos, y contraste con runner real | SATISFECHO | `TestHCD027FlakyContract` y `TestHCD027PlaywrightIntegration` con CLI real. |

---

## 2. Validación de Decisiones de Aclaración (Q1 y Q2)

- **Aclaración Q1 (Opción A):**
  - Configuración existente conserva sus permisos (`info.Mode().Perm()`); archivos nuevos adquieren `0600`. No se alteró la política general de enlaces simbólicos ni ejecutables.
- **Aclaración Q2 (Opción A):**
  - Las pruebas que aprueban tras fallar se registran como un único resultado `flaky`.
  - El diagnóstico inicial se preserva en `failure.message` y se expone tanto en el panel como en el informe del agente.
  - El contador `stats.flaky` es independiente (no incrementa `passed`, `failed` ni `queued`).
  - Si existen fallos definitivos, el estado general y de la suite es `failed` (prevalencia de fallos definitivos).

---

## 3. Límites Registrados

1. **Herramienta Playwright Real:** Se verificó la integración real mediante el ejecutor ya instalado `/opt/homebrew/lib/node_modules/@playwright/cli/node_modules/playwright/cli.js` (versión pre-release `1.61.0-alpha-1781023400000`). La compatibilidad con versiones estables publicadas en npm queda como límite no verificado de forma exhaustiva, aunque el esquema del reporter oficial `json` cumple el estándar de Playwright.
2. **Plataformas de Procesos:** La gestión de grupos de procesos (`killProcessGroup` con PID negativo y `detached: true`) aplica a sistemas POSIX (macOS y Linux). En Windows se degrada de forma segura a terminación individual del proceso.

---

## 4. Conclusión de Revisor B
El delta respeta estrictamente el alcance de HCD-027 y la definición aprobada `9747986f1af3`. No se observan discrepancias en los contratos ni regresiones en la suite general (`go test -race ./...`).

# Informe de Revalidación Independiente — Revisor A (ADM / Defensivo)

**Fecha:** 2026-11-12  
**Unidad:** HCD-027 (`9747986f1af3` / `abe8c59a3b92`)  
**Base:** HEAD `2d63b8d231522224d30215bbed1205f051afa7c7` (rama main)  
**Digest del Diff:** `0cb6b28e8697480d8a75c6151a1e37029ce13c8083a78890c55d0f7020388f6c`  
**Rol:** Revisor A (Auditoría Defensiva Multilenguaje / Estructura y Procesos) — Solo Lectura  

---

## 1. Alcance Examinado

- `pkg/installer/jsonfile.go`: Funciones `writeJSONObject` y `writeFileAtomic`.
- `pkg/installer/hostskills.go`: Función `pruneLegacyPiSkills` y predicado `onlySkillFile`.
- `extensions/gomemory-tdd.ts`: `streamLines`, `killProcessGroup`, `goTestParser`, `playwrightLineParser`, `reconcilePlaywrightJson`, métricas `flaky` y `runCheck`.
- Suites de prueba de regresión y cobertura: `hcd027_regressions_test.go`, `hcd027_json_test.go`, `hcd027_skills_test.go`, `hcd027_process_test.go`, `hcd027_tdd_test.go`, `hcd027_playwright_test.go`, `hcd027_playwright_integration_test.go`.

---

## 2. Evaluación Defensiva por Defecto

### C-001 (Pérdida de archivos en skills legacy marcadas)
- **Código:** En `pkg/installer/hostskills.go:250`, la condición de borrado de copias embebidas prístinas ahora exige `isPristineEmbeddedCopy(p) && onlySkillFile(p)`.
- **Análisis:** `onlySkillFile` comprueba que la carpeta no contenga más entradas que `SKILL.md` y `.pi-gomemory`. Si el usuario añadió scripts, notas o subcarpetas, la función devuelve `false` y `os.RemoveAll` nunca se invoca sobre el directorio del usuario.
- **Veredicto defensivo:** RESOLVED. Evita pérdida irreversible de archivos creados por el usuario.

### C-002 (Ampliación de permisos en JSON privados a 0644)
- **Código:** En `pkg/installer/jsonfile.go:37-47`, `writeJSONObject` consulta el modo existente mediante `os.Stat(path)`. Si el archivo existe, preserva `info.Mode().Perm()`; si no existe, asigna `0600`.
- **Análisis:** Previene la ampliación de permisos a 0644 en entornos multiusuario cuando los directorios padre permiten acceso. Los errores de `os.Stat` distintos de `fs.ErrNotExist` detienen la operación sin tocar el archivo.
- **Veredicto defensivo:** RESOLVED.

### C-003 (Falsos positivos de éxito tras SIGTERM y procesos colgados)
- **Código:** En `extensions/gomemory-tdd.ts`:
  1. `streamLines` marca `timedOut: true` o `aborted: true` explícitamente y fuerza código de salida no nulo (`124` o `1`) aun cuando el hijo capture la señal y salga con `process.exit(0)`.
  2. `killProcessGroup` aísla la señalización en POSIX con `detached: true` a `-child.pid`, y programa un temporizador de 800ms que escala a `SIGKILL` y destruye las tuberías `stdout`/`stderr`.
- **Análisis:** Los hijos rebeldes y los descendientes que heredan tuberías son limpiados en menos de 2 segundos. Se valida que no se señalicen grupos ajenos (PID > 1).
- **Veredicto defensivo:** RESOLVED.

### C-004 (Colapso de pruebas Playwright de distintos proyectos)
- **Código:** En `playwrightLineParser`, la identidad de la prueba incluye el proyecto capturado `[${project}]::${loc}::${cleanTitle}`.
- **Análisis:** Dos proyectos (`chromium` y `firefox`) en la misma ubicación física producen dos identificadores separados en el almacén reactivo y no se sobrescriben diagnósticos.
- **Veredicto defensivo:** RESOLVED.

### C-005 (Pruebas Go pendientes en status "running")
- **Código:** En `goTestParser.finish(res)`, el conjunto `runningTests` registra todas las pruebas con `Action: "run"` sin evento terminal. Al concluir el proceso (sea por salida anormal, timeout o aborto), todas las pruebas pendientes se cierran como `failed` con diagnóstico descriptivo de la causa de terminación.
- **Análisis:** `stats.running` queda en 0 y el estado global pasa a `failed`.
- **Veredicto defensivo:** RESOLVED.

### C-006 (Aprobación falsa de pruebas Playwright incompletas)
- **Código:** En `playwrightLineParser.finish(res)`: la autoridad de estados terminales es el informe JSON oficial de Playwright. Si el proceso termina anormalmente o sin informe JSON válido, las pruebas no se marcan `passed`, sino que se cierran como `failed` preservando los diagnósticos.
- **Análisis:** Elimina la inferencia falsa de aprobaciones basada únicamente en no haber visto una cabecera de fallo.
- **Veredicto defensivo:** RESOLVED.

---

## 3. Conclusión de Revisor A
Todas las correcciones abordan directamente la causa raíz de los defectos observados sin introducir dependencias externas ni degradar los mecanismos de seguridad. Los 6 defectos quedan corregidos de forma defensiva y verificada.

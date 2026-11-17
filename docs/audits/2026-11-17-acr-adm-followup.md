# Seguimiento ACR y ADM: cierres H-01 a H-05

- Fecha: 2026-11-17.
- HEAD: `5183867a86a800a88638b4234051810e47d4beb0` + árbol sucio HCD-039/040/041.
- Digest: `8a361933af5b91945ea15ed7d4a316dfd4cb0e61a8ef6b46d33c44550174c607`, idéntico antes de revisiones y después de pruebas.
- [Target y procedimiento](2026-11-17-acr-adm-followup-target.md). Inventario excluye auditorías, HCD e índices. Alcance dirigido, no auditoría exhaustiva.
- Solo lectura del producto: sin correcciones, instalaciones en HOME real, commits ni release. Reproducciones en copia temporal y datos sintéticos.

## 1. Resumen de salud del código

**Tecnologías:** Go en CLI/instalador y pruebas; Shell en gate de release. TS en extensiones y HTML/CSS documental presentes pero fuera de inspección profunda.

**Riesgo general: Medio.** Los cinco reproductores originales pasan. Los cierres locales conservan el comportamiento acordado: pin compilado, configuración CC Extensions sobrevenida preservada, comandos fallidos rechazados, caché ausente/rota no saludable y extracción acotada/cancelable. También pasan las comprobaciones contra npm real.

**Resultado:** dos hallazgos nuevos medios, reproducidos por el coordinador y corroborados por dos revisores; cero críticos, altos o bajos. No se demuestra regresión de los escenarios originales. Hay huecos adyacentes a H-02 y H-04; no equivalen a declarar completo el endurecimiento de todas las retiradas o de toda posible caché. No se demostró fuga de credenciales/PII ni escape de extracción; tampoco se certifica su ausencia general.

### Cobertura y validación

| Comprobación | Resultado |
|---|---|
| `go test -race ./... -count=1` | Pasa: raíz 1,371 s; instalador 58,343 s |
| `go vet ./...` | Pasa |
| `go vet -tags pins ./...` | Pasa |
| `bash -n install.sh scripts/build-release.sh` | Pasa; solo sintaxis |
| `git diff --check` | Pasa |
| `go test ./pkg/installer -run '^(TestACRADM0[1-5]\|TestHCD040\|TestHCD041)' -count=1 -v` | Pasa; incluye cinco reproductores, históricos y fronteras/cancelación |
| `go test -tags pins -count=1 -v ./pkg/installer -run '^(TestPinsCheck\|TestPinsValidate)$'` | PinsCheck pasa con npm real; Validate omitido sin candidato |
| `PINS_CANDIDATE=pi-antigravity@0.14.0 go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$'` | Pasa A1/A3 |
| Mismo comando con `pi-cc-extensions@1.0.20` | Pasa C1 |
| `go test ./pkg/installer -run '^TestFollowup' -count=1 -v` en copia temporal | Falla como esperado: dos versiones no comparables y cuatro sustituciones tardías |
| Dos invocaciones separadas de Splash | Completadas, revisión estática de fuentes; no ejecutaron comandos |
| Grafo | Generación codebase-memory 2026-11-14T12:00:58Z, sin watcher; 11 rutas comprobadas, nuevas/modificadas contrastadas con fuente actual |
| `govulncheck`, `staticcheck`, `shellcheck`, `semgrep`, `gitleaks`, `tsc` | No disponibles en PATH; no se instalaron |
| UI, sidebar, cuota y modelo real; SCA TS | No ejecutado |
| Evidencia oficial HCD nueva | No registrada: es auditoría, no ejecución de tareas ni reaprobación |

Rutas consultadas con grafo: RunDepsUpdate → DepsStatusFor/cachedDeps/liveVersion; probes → runOutput; BuildUninstallPlan → planPlainFile/planManifest y planPackages → planCCExtensions/planPinnedPackages; fetchCandidate → extractCandidate. Revisores leyeron además pins, doctor, plugin, jsonfile, update, gate y pruebas relacionadas. No se ejecutaron instaladores reales ni agotamiento de recursos.

## 2. Hallazgos y fugas detectadas

IDs de este informe son **F-01/F-02**, no sustituyen H-01 a H-05 del informe inicial.

| ID | Severidad | Estado ADM / ACR | Problema |
|---|---|---|---|
| F-01 | Media | CONFIRMADO reproducido / C-002 CONFIRMED | Versiones no comparables producen falso «al día» |
| F-02 | Media | CONFIRMADO reproducido / C-001 CONFIRMED | Cachés/manifiesto eliminan entrada sobrevenida sin revalidación |

#### F-01 — Comparación imposible interpretada como estado saludable

- **Categoría:** Éxito falso / validación de datos.
- **Severidad:** Media: estado sintético no comparable o compilación local `dev`; diagnóstico acotado, sin ejecución arbitraria.
- **Estado:** CONFIRMADO ADM y ACR (A-02 + B-01).
- **Ubicación:** `pkg/installer/deps.go:454-475`; `pkg/installer/update.go:45-67`.
- **Ruta de alcance:** CLI `deps-update` → RunDepsUpdate → DepsStatusFor → Outdated → IsNewer.
- **Análisis:** el guard comprueba Error y cadenas vacías, no si las versiones son comparables. IsNewer devuelve false tanto para «no hay versión superior» como para «no se puede comparar». El caller convierte ambos casos en éxito. `Current=dev` es posible para el componente pi-gomemory construido localmente; no se afirma que npm devuelva hoy `desconocida`.
- **Evidencia:** `TestFollowupInvalidVersionNotHealthy` falla para `[1.0.0 desconocida]` y `[dev 1.0.0]`: `rc=0 stdout="✓ El meta-harness y sus dependencias están al día.\n" stderr=""`.
- **Código corregido propuesto, no aplicado:** añadir este guard a la clasificación de desconocidos, después de resolver las versiones efectivas (pin compilado/local cuando proceda), sin convertir el error de comparación en «no atrasado»:

```go
func comparableVersions(current, latest string) bool {
    _, currentOK := parseSemver(current)
    _, latestOK := parseSemver(latest)
    return currentOK && latestOK
}
```

Usarlo como condición de estado desconocido: `reason = "versiones no comparables"`. Mantener actualización parcial conocida, salida 1 y sugerencia deps-refresh. Este fragmento no es una corrección integrada; debe conectarse al recorrido de estados efectivos para no rechazar un objetivo antiguo que el pin compilado sustituye válidamente.
- **Validación del cierre:** los dos casos del reproductor dejan de anunciar al día, devuelven 1; versiones estables sanas conservan 0, un componente desconocido no impide actualizar otro conocido. Mantener tests HCD-040 de pin y caché antiguos.

#### F-02 — Retirada de cachés y manifiesto sin revalidar entrada

- **Categoría:** Integridad / cambio del sistema de archivos entre plan y acción.
- **Severidad:** Media: se borra enlace o directorio vacío sobrevenido, no el archivo destino ni un directorio no vacío.
- **Estado:** CONFIRMADO ADM y ACR (A-01 + B-02).
- **Ubicación:** `pkg/installer/uninstall.go:642-647,702-703,717-723`.
- **Ruta de alcance:** uninstall → BuildUninstallPlan → planPlainFile (dos cachés) / planManifest → confirmación → apply.
- **Análisis:** Lstat se realiza al crear el plan; apply hace Remove directo. La sustitución ocurre antes de iniciar apply, no durante ella. El manifiesto se retira solo si no fallaron acciones previas, lo que no evita este escenario si todas tuvieron éxito.
- **Evidencia:** `TestFollowupLateEntryPreserved`, cuatro casos cache/manifest × symlink/directory, todos fallan con `apply=<nil>` y `Lstat: no such file or directory`. El destino sintético `SYNTH-PERSONAL` permanece intacto.
- **Matiz de contrato:** el manifiesto de esta auditoría resumió demasiado ampliamente la preservación de entradas. HCD-041 la limita a `pi-cc-extensions.json`: ese cierre sí pasa. F-02 es una variante adyacente, no una regresión demostrada ni una reapertura automática de H-02. La evidencia de borrado es independiente de esa redacción; la decisión de ampliar preservación a cachés/manifiesto requiere acuerdo.
- **Código corregido propuesto, no aplicado:** para acciones cuyo contrato pase a preservar entradas nuevas, capturar FileInfo del archivo regular planificado y usar este helper antes de Remove:

```go
func removeUnchangedRegular(path string, planned os.FileInfo) error {
    now, err := os.Lstat(path)
    if err != nil {
        return goneOr(err)
    }
    if !planned.Mode().IsRegular() || !now.Mode().IsRegular() || !os.SameFile(planned, now) {
        return errChangedAfterPlan
    }
    return os.Remove(path)
}
```

Integrarlo en ambas acciones y acordar qué hacer con enlaces existentes al planificar. No garantiza atomicidad ante sustitución concurrente entre Lstat y Remove; no ampliar esa promesa.
- **Validación del cierre:** cuatro reproducciones verdes, entrada sobrevenida intacta y archivo regular original retirado. Mantener condición ifClean del manifiesto y comportamiento de reintentos.

## 3. Revisión de cierres anteriores

| Hallazgo original | Resultado técnico actual | Límite / estado de unidad |
|---|---|---|
| H-01 pin nuevo oculto por caché | Cerrado el escenario: TestACRADM01 y caché vigente/caducada pasan | HCD-040 continúa activa; cierre funcional local no equivale a cerrar su instalación/UI |
| H-02 CC Extensions sobrevenido | Cerrado el escenario y enlace roto/directorio pasan | No cubre todas las rutas de retirada: F-02 |
| H-03 salida de comando fallido | Cerrado el escenario: fallo stdout/stderr, cancelación y éxito pasan | No se afirmó observar codegraph con actualización disponible real |
| H-04 caché ausente/rota/parcial/error | Cerrados los escenarios acordados y reproducidos | Comparabilidad no vacía sigue abierta como F-01; no afirmar diagnóstico completo |
| H-05 límites y cancelación | Cerrado el escenario: fronteras entradas/bytes/contenido, cancelación durante extracción, limpieza y candidatos reales pasan | No se prueba DoS real ni cancelación dentro de una sola entrada |

**HCD-041:** el motor confirma `closed`, revisión `c3472feedfd3`, aprobación ya registrada. Antes de crear artefactos de esta auditoría conservaba 7/7 verificadas con huella `5183867a86a8+ea4e715cc8ca`. Tras añadir el manifiesto de auditoría, analyze devuelve evidencia no vigente con huella `5183867a86a8+c52a6cb01e37`: la huella HCD incluye cambios documentales que el digest de producto excluye. Los tests actuales verdes son corroboración de auditoría, no sustitución de evidencia oficial ni nuevo cierre. No se modifica el registro histórico.

**HCD-040:** sigue activa en implement, revisión `33363422c686`, tareas 6 y 7 pendientes (instalación en HOME real y observación humana en Pi), resto sin evidencia vigente en la huella actual. PinsCheck y ambos candidatos pasan hoy, pero no se registraron como tareas HCD ni se ejecutó instalación real. La auditoría no puede aprobar por la persona banner/sidebar/modelo. HCD-040 no está lista para cerrar.

## 4. Consenso ACR y límites

- Review ID: `acr_cf822591-fe3c-4086-a087-42a73823a7d6`.
- Seguimiento de `acr_937c93d6-d10a-400a-a907-76cc190c2efc`.
- Delta derivado de alcance explícito, no comparación Git de snapshots: main.go, pkg/installer y gate. Se reexaminaron esos componentes; no hay ahorro de delta verificable.
- Heredados SUSPECT 35/37 (H-04/H-05 originales) no se vuelven a adjudicar como nuevos. Sus escenarios actuales se contrastan con pruebas y lectura. F-01 es otra causa (comparación imposible), no repetición de caché ausente/ilegible.
- Dos invocaciones Splash separadas recibieron solo el mismo documento y lente, no resultados del otro ni historial/memoria. Modelo/proveedor real y aislamiento interno no verificables: independencia **degraded / same-model** según motor. No se afirma diversidad ni aislamiento certificado.
- Los revisores devolvieron etiqueta `Major`; el coordinador normalizó a MEDIUM por impacto acotado y precondiciones específicas. Se normalizaron campos problem→claim y confidence a high, sin promover severidad por la etiqueta editorial. Evidencia dinámica añadida por coordinador, no atribuida a ejecución del revisor.
- Consenso: dos CONFIRMED MEDIUM, cero contradicciones; una ronda inicial, cero correcciones/revalidaciones; `fix_authorized=false`.
- **Veredicto terminal: APPROVED**, derivado por `review_finalize`. Política: no quedan CRITICAL/HIGH confirmados; **los dos MEDIUM siguen abiertos**. No significa ausencia de defectos ni autorización de release o cierre de HCD-040.

## 5. Recomendaciones de resiliencia y próximos cierres

1. Acordar una nueva unidad para F-01 y decidir si se amplía preservación para F-02. No aplicar cambios por el mero resultado de esta revisión.
2. Completar HCD-040: instalación real acordada, comprobaciones de versiones/cuota/doctor y observación de la persona; refrescar evidencia mecánica mediante el motor antes de solicitar cierre sobre revisión exacta.
3. Separar estado comparable, no comparable y atrasado mediante estados explícitos; no usar bool de comparación como prueba de salud.
4. Mantener fronteras pequeñas en pruebas y revalidación simétrica de acciones destructivas. Incorporar scanners faltantes en CI controlada, no instalarlos en host durante auditoría.

## 6. Reproducciones

[Fuente sintética](2026-11-17-acr-adm-followup-repro_test.go.txt), guardada como texto para no alterar suite de producto. Copia usada: `/tmp/pi-acradm-followup-y4y0QQ` (pkg, extensions, scripts, config, themes, go.mod).

Para repetir: en una copia temporal equivalente, poner el adjunto como `pkg/installer/acradm_followup_test.go` y ejecutar `go test ./pkg/installer -run '^TestFollowup' -count=1 -v`. Salida esperada del target auditado: FAIL, dos casos de versiones y cuatro de retirada. Ningún comando de reproducción instala ni desinstala paquetes reales.

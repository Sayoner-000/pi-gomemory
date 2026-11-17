# ACR y ADM — ronda adicional posterior a HCD-042/Q2

- **Fecha:** 2026-11-17.
- **Revisión:** HEAD `5183867a86a800a88638b4234051810e47d4beb0` + árbol sucio actual.
- **Digest de producto:** `0ed83c20464858d16b97d14c536ea4da84ef74f07c23b50c82c6f4a58332343b`, comprobado antes y después de revisiones y reproducciones. Excluye auditorías, documentos HCD e índices.
- **Alcance dirigido:** main.go, pkg/installer y scripts/build-release.sh; prioridad dependencias, manifiesto, retiradas, recursos del plan, pins y extracción. No es revisión exhaustiva del repositorio.
- **Modalidad:** producto solo lectura. Sin correcciones, cambios de pins, instalaciones en HOME real, commits ni release. Reproducciones en copia temporal con datos sintéticos.
- [Target y contratos](2026-11-17-acr-adm-round3-target.md) · [Reproductores](2026-11-17-acr-adm-round3-repro_test.go.txt).

## 1. Resumen de salud del código

**Tecnologías:** Go en CLI/instalador y pruebas; Shell en gate. TypeScript de extensiones y HTML/CSS de documentación fuera de inspección profunda.

**Riesgo general: Medio.** Los cambios de HCD-042 resuelven las reproducciones anteriores y retienen/cerran descriptores como se acordó en Q2; la suite completa con carreras y el análisis estático siguen verdes. La nueva revisión encuentra huecos en entradas anteriores a los helpers y superficies adyacentes, no una rotura demostrada de la retención de recursos.

**Resultado ADM:** cuatro hallazgos medios reproducidos, cero críticos, altos o bajos. **ACR:** uno CONFIRMED y tres SUSPECT (identificación unilateral); la reproducción del coordinador no equivale a consenso de dos revisores. No se ha demostrado que estos defectos hayan sido introducidos por HCD-042: los mecanismos ya existían. No se declara regresión de una causa raíz corregida sin evidencia histórica.

No se demostró fuga de credenciales/PII ni escape de extracción en lo inspeccionado. Esto no certifica su ausencia general.

### Cobertura y validación

| Comprobación | Resultado |
|---|---|
| `go test -race ./... -count=1` | Pasa: raíz 1,443 s, instalador 57,343 s; incluye regresiones y ciclo de vida HCD-042 |
| `go vet ./...` | Pasa |
| `go vet -tags pins ./...` | Pasa |
| `bash -n install.sh scripts/build-release.sh` | Pasa; solo sintaxis |
| `git diff --check` | Pasa |
| `go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsCheck$'` | **Falla correctamente:** npm publica CC Extensions 1.0.21, pin 1.0.20 |
| `PINS_CANDIDATE=pi-antigravity@0.14.0 go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$'` | Pasa A1/A3 |
| Mismo comando con pi-cc-extensions@1.0.20 | Pasa C1; no demuestra compatibilidad de 1.0.21 |
| `go test ./pkg/installer -run '^TestRound3' -count=1 -v` en copia temporal | Cuatro FAIL esperados, uno con tres variantes; helper aislado omitido en ejecución padre |
| Dos Splash separados | Ambos completados; lectura estática, sin ejecución de comandos |
| Grafo y cobertura | codegraph consultado; codebase-memory generación 2026-11-14T12:00:58Z sin watcher. Fuentes nuevas/modificadas contrastadas directamente |
| govulncheck/staticcheck/shellcheck/semgrep/gitleaks/tsc | No disponibles en PATH; no instalados |
| Dependencias TS, UI/sidebar/cuota/modelo real y runtime Linux | No comprobados en esta ronda |
| Comprobaciones oficiales HCD nuevas | No registradas: auditoría no equivale a reaprobación/cierre |

**Cambio upstream:** PinsCheck estaba verde antes y ahora observa `pi-cc-extensions@1.0.21`. El bloqueo de release es el comportamiento correcto del gate, no un quinto defecto del producto. No se validó ni aprobó ese candidato, no se actualizó el pin ni se usó el escape.

**Grafo:** se localizaron Uninstall → BuildUninstallPlan → ReadManifest/planRegularRemoval; Close y rutas de retorno; RunDepsUpdate → effectiveDepState/liveVersion; versionFromCmd → firstSemver; planSelf y RunDoctor. La resolución de nombres comunes como Close sobreaproxima callers; no se tomó como prueba exhaustiva. Ambos revisores leyeron código actual y pruebas; el coordinador obtuvo toda la evidencia dinámica.

## 2. Hallazgos y fugas detectadas

IDs **R3-01 a R3-04**, independientes de H-01..H-05 y F-01/F-02 anteriores.

| ID | Severidad | Estado ADM | Consenso ACR | Problema |
|---|---|---|---|---|
| R3-01 | Media | CONFIRMADO reproducido | C-001 CONFIRMED, A-02+B-01 | Manifiesto FIFO bloquea incluso dry-run |
| R3-02 | Media | CONFIRMADO reproducido | C-004 SUSPECT, solo A-01 | Lector trunca prerelease y permite falso al día |
| R3-03 | Media | CONFIRMADO reproducido | C-002 SUSPECT, solo B-02 | --self borra sustitutos posteriores al plan |
| R3-04 | Media | CONFIRMADO reproducido | C-003 SUSPECT, solo B-03 | Doctor excede timeout por tuberías heredadas |

Las correcciones siguientes son **propuestas no aplicadas**. Los fragmentos ilustran cambios mínimos; su integración debe respetar contratos y pasar rojo/verde antes de considerarse corregidos.

#### R3-01 — Inspección de manifiesto FIFO antes de la protección de retirada

- **Categoría:** Fuga/retención de recursos y bloqueo.
- **Severidad:** Media; requiere una entrada local especial en la ruta de manifiesto.
- **Estado:** CONFIRMADO ADM y ACR.
- **Ubicación:** `pkg/installer/uninstall.go:747-749,790`; `pkg/installer/manifest.go:66-67`.
- **Ruta de alcance:** CLI uninstall, también --dry-run → BuildUninstallPlan → ReadManifest → os.ReadFile.
- **Análisis:** ReadManifest abre y lee antes de llegar a planRegularRemoval. Un FIFO sin escritor bloquea la apertura; la validación posterior de tipos y O_NONBLOCK no llega a ejecutarse. El cierre por defer de Uninstall aún no está instalado porque BuildUninstallPlan no ha devuelto.
- **Evidencia:** TestRound3ManifestFIFO crea FIFO sintético y lanza Uninstall DryRun en proceso aislado. Tras `entered synthetic probe`, supera 1 segundo sin retornar; el padre mata el grupo: `elapsed=1.002836958s`, `signal: killed`. El bloqueo sin escritor se deriva además de la apertura bloqueante, no de asumir que una ejecución lenta ordinaria es infinita.
- **Código corregido propuesto:** abrir de forma no bloqueante/no-follow y validar el descriptor **antes** de leer el manifiesto; siempre cerrarlo. Núcleo para el lector, con imports io/syscall y conservación del tratamiento actual de ausencia/JSON:

```go
f, err := os.OpenFile(ManifestPath(home), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
if err != nil {
    return nil, err
}
defer f.Close()
st, err := f.Stat()
if err != nil {
    return nil, err
}
if !st.Mode().IsRegular() {
    return nil, fmt.Errorf("el manifiesto no es un archivo regular")
}
data, err := io.ReadAll(f)
if err != nil {
    return nil, err
}
```

La integración debe propagar también el error de cierre y acordar el tratamiento de un manifiesto enlazado: no cambiarlo implícitamente si es contrato soportado. No basta un Lstat seguido de una apertura bloqueante, porque el tipo puede cambiar entre ambas operaciones.
- **Validación del cierre:** FIFO sin escritor y otras entradas no regulares no bloquean; salida informa conservación/error; manifiesto regular válido y corrupto mantienen sus contratos y no quedan descriptores abiertos.

#### R3-02 — Pérdida del sufijo de versión en el lector de comandos

- **Categoría:** Éxito falso / validación de borde.
- **Severidad:** Media; requiere un componente con formato no comparable, como prerelease.
- **Estado:** CONFIRMADO ADM, SUSPECT ACR por identificación unilateral.
- **Ubicación:** `pkg/installer/deps.go:77-84,120-127,228-246`.
- **Ruta de alcance:** deps-refresh o relectura local de deps-update → probes compilados (pi/mem/codegraph/codebase-memory) → versionFromCmd → firstSemver → comparación posterior.
- **Análisis:** el regex extrae únicamente X.Y.Z. `tool 1.0.0-rc.1` se convierte en `1.0.0` antes de la validación añadida en HCD-042. La prueba de prerelease con fixedVersion sí pasa, pero no atraviesa el lector de comandos. Es un hueco de entrada relacionado con F-01, no la misma causa de versiones ya recibidas sin comparar.
- **Evidencia:** TestRound3PrereleaseThroughCommand: `reader version="1.0.0" error=<nil>`; RunDepsUpdate devuelve `rc=0`, stdout `✓ El meta-harness y sus dependencias están al día.` y stderr vacío.
- **Código corregido propuesto:** extraer un token completo sin descartar el sufijo y comprobar comparabilidad antes de retornarlo. Ejemplo para esta entrada:

```go
var versionTokenRe = regexp.MustCompile(`v?[0-9]+\.[0-9]+\.[0-9]+[0-9A-Za-z.+-]*`)

func comparableVersionFromText(out string) (string, error) {
    token := versionTokenRe.FindString(out)
    if _, ok := parseSemver(token); !ok {
        return "", fmt.Errorf("versión no comparable")
    }
    return strings.TrimPrefix(token, "v"), nil
}
```

Conectar a versionFromCmd y revisar latestFromCmd por el mismo patrón de captura. No imprimir salida del proceso en errores. Validar fronteras del token y formatos reales antes de integrar, sin ampliar los formatos aceptados por el comparador.
- **Validación del cierre:** prereleases/sufijos por comando real no se convierten en estables; errores quedan desconocidos; las salidas estables reales siguen interpretándose correctamente. Mantener prioridad pin/local y actualización parcial.

#### R3-03 — Retirada --self sin revalidación

- **Categoría:** Integridad / entrada sustituida después del plan.
- **Severidad:** Media; modificación local mientras espera confirmación. No se probó borrado del destino de un enlace.
- **Estado:** CONFIRMADO ADM, SUSPECT ACR por identificación unilateral.
- **Ubicación:** `pkg/installer/uninstall.go:942-965`.
- **Ruta de alcance:** uninstall --self → BuildUninstallPlan → planSelf → confirmación → apply.
- **Análisis:** planSelf valida existencia y BIN_DIR al planificar, pero apply solo hace Remove. La protección Q1/Q2 de cachés/manifiesto no se aplica a --self, recurso excluido del cambio anterior. No se presenta como regresión de ese cierre.
- **Evidencia:** TestRound3SelfReplacement falla para sustitutos regular/enlace/directorio vacío: `action=<nil>` y Lstat posterior devuelve no such file or directory.
- **Código corregido propuesto:** después de comprobar BIN_DIR, reutilizar la retirada regular protegida para ejecutables originalmente regulares:

```go
p.planRegularRemoval(exe, "binario pi-gomemory-setup (--self)", false)
```

Antes de integrar, acordar qué hacer con un ejecutable enlazado desde el inicio: este helper lo conservaría y no debe cambiarse ese contrato por accidente. Actualizar el cierre de recursos de la extensión a --self y su documentación.
- **Validación del cierre:** las tres sustituciones se preservan, el regular original se retira, BIN_DIR ajeno se sigue omitiendo; no fugas al simular/cancelar/fallar, ni borrado del destino del enlace.

#### R3-04 — Timeout de doctor no limita espera de tuberías

- **Categoría:** Retención de procesos/recursos y disponibilidad.
- **Severidad:** Media; requiere un pi que deje descendientes con stdout/stderr heredados.
- **Estado:** CONFIRMADO ADM, SUSPECT ACR por identificación unilateral.
- **Ubicación:** `pkg/installer/doctor.go:88-91`.
- **Ruta de alcance:** CLI doctor/status/check o diagnóstico final de instalación → RunDoctor → pi mcp list --json → Cmd.Output.
- **Análisis:** CommandContext limita al hijo directo, pero Output puede esperar por tuberías heredadas. No tiene WaitDelay, a diferencia de runOutput. Matar al hijo directo a los diez segundos no cierra las tuberías del descendiente.
- **Evidencia:** pi sintético ejecuta `/bin/sleep 30 & wait`. TestRound3DoctorDescendant lanza RunDoctor en grupo aislado, supera presupuesto externo de 12 segundos (`elapsed=12.002690916s`) y el padre termina todo su grupo. No se dejó dormir un descendiente contra HOME real.
- **Código corregido propuesto**, inmediatamente después de CommandContext y antes de Output:

```go
mcpListCmd.WaitDelay = time.Second
```

Esto acota la espera de tuberías y permite devolver WARN; no significa que mate todos los descendientes. Si se exige garantizar su limpieza, diseñar terminación del grupo propio y probarla sin señalizar procesos del anfitrión.
- **Validación del cierre:** el escenario retorna dentro de timeout+WaitDelay+margen, con diagnóstico no saludable; éxito normal conservado. Verificar por separado limpieza del grupo si forma parte del contrato acordado.

## 3. Recomendaciones de resiliencia y revisión de cierres

1. No dar por finalizada la robustez de HCD-042 sin revisar R3-01 (entrada de manifiesto) y R3-02 (entrada de versiones), relacionados con sus promesas. Los escenarios originales y retención/cierre Q2 sí pasan; no afirmar que la última revisión sin hallazgos equivale a ausencia de variantes.
2. R3-03/R3-04 son superficies adyacentes; proponer unidad separada o ampliación explícita, no corregir por conveniencia dentro de una auditoría.
3. Actualizar la evidencia oficial cuando se decida continuar: nuevos documentos cambian huella HCD aunque el digest de producto permanezca igual. No marcar tareas manualmente.
4. Tratar por separado el nuevo candidato CC Extensions 1.0.21 y HCD-040, cuya instalación/observación humana sigue pendiente. No saltar el gate como si fuera un fallo del test.
5. Mantener validación de entrada antes de operaciones bloqueantes, conservar tokens completos y usar pruebas que atraviesen bordes reales además de fakes. Incorporar scanners faltantes en CI controlada.

**Estado de HCD-042:** no cerrada. El intento con cita literal `apruebo cierre` fue rechazado por el motor (`invalid_quote`); no se inventó una cita alternativa con revisión. Conserva work_revision `d94ce297d2d0`, pero tras crear artefactos de esta ronda el motor pasa a verify con evidencia anterior no vigente (huella observada `5183867a86a8+126d4434848f`). El producto auditado no cambió. La auditoría no modifica aprobaciones ni reescribe la evidencia previa.

## 4. ACR: consenso y veredicto

- Review ID: `acr_20990df0-2cf3-4752-a5db-04112ae65aa5`, seguimiento de `acr_cf822591-fe3c-4086-a087-42a73823a7d6`.
- Delta por alcance explícito, no comparación de commits; inherited vacío. No se atribuye ahorro verificable de revisión incremental.
- Dos ejecuciones Splash separadas recibieron solo ruta del target y lente. No se enviaron resultados de uno al otro, historial ni memoria. Modelo/proveedor e aislamiento interno no verificables: independencia **degraded / same-model**, según motor.
- Ambos inspeccionaron fuentes; no verificaron HEAD/digest ni ejecutaron comandos. El coordinador comprobó el digest y ejecutó las reproducciones.
- Salidas editoriales Major normalizadas a MEDIUM por impacto local y condiciones específicas, con categorías y campos normalizados; no se presume severidad HIGH por etiqueta Major.
- Consenso: 1 CONFIRMED MEDIUM, 3 SUSPECT MEDIUM, 0 contradicciones. ADM reproduce los cuatro; eso no cambia la clasificación ACR.
- Una ronda inicial, cero correcciones/revalidaciones; fix_authorized=false.
- **Veredicto terminal del motor: APPROVED.** Significa que no hay CRITICAL/HIGH confirmados pendientes según política; **los cuatro MEDIUM continúan abiertos**. No significa cierre de HCD-042, release autorizada ni código sin defectos.

## 5. Reproducciones y seguridad

Copia usada: `/tmp/pi-acradm-round3-Xfaen3` (pkg, extensions, scripts, config, themes y go.mod). Adjunto como texto para no añadir pruebas fallidas al producto.

Para repetir en copia temporal equivalente: guardar el adjunto como `pkg/installer/acradm_round3_test.go` y ejecutar:

```sh
go test ./pkg/installer -run '^TestRound3' -count=1 -v
```

Esperado: FAIL de cuatro tests de conducta segura, con tres variantes de --self. TestRound3ProbeChild se omite en padre y solo se ejecuta en subprocesos con variables sintéticas. Los dos bloqueos tienen límite externo, grupo privado y SIGKILL de ese grupo; no se provoca bloqueo indefinido ni se instala/desinstala configuración real. Duración observada total 13,386 s. La suite ordinaria verde y estos FAIL intencionales muestran que faltaban casos de borde, no un fallo del harness.

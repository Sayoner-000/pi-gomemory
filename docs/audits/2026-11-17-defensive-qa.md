# ADM — Auditoría Defensiva Multilenguaje de pi-gomemory

- **Fecha:** 2026-11-17 (UTC).
- **Revisión auditada:** `b22a5c4a77607d9fe1860a30e0633dd22b76488e` (v1.13.7).
- **Estado inicial:** árbol de trabajo limpio.
- **Modalidad:** inspección dirigida de superficies de producción y reproducciones en copia temporal; sin cambios de producción, instalaciones de dependencias ni uso de credenciales reales.
- **Alcance:** CLI Go, instalación/actualización/desinstalación, configuración y skills, clientes OAuth/cuotas, extensiones TypeScript TDD/sidebar/plan/Splash/status y scripts de instalación/empaquetado. No equivale a una revisión exhaustiva de cada línea del repositorio.

## 1. Resumen de salud del código

**Nivel de riesgo general observado: MEDIO.**

El proyecto dispone de defensas relevantes: reemplazo atómico de archivos, verificación SHA256 de actualizaciones, límites de descargas y de respuestas de cuotas, timeouts HTTP, ejecución de procesos sin shell en TDD y pruebas de regresión sobre desinstalación, enlaces simbólicos y cancelación de grupos de procesos. La suite existente pasa, incluido el detector de carreras. Estos resultados no cubren todos los contratos negativos: las nuevas pruebas aisladas reproducen dos defectos.

**Resultado: 2 hallazgos CONFIRMADOS — 0 críticos, 0 altos, 2 medios, 0 bajos.** Uno permite crecimiento no acotado de memoria al consumir salida de procesos; otro convierte respuestas vacías de Antigravity en éxito y muestra una etiqueta de plan no sustentada por la respuesta. No se confirmó una fuga de credenciales ni de PII en el alcance inspeccionado; no se afirma su ausencia en todo el producto.

### Lenguajes y tecnologías

| Ámbito | Tecnologías y separación |
|---|---|
| Producción | Go 1.27, biblioteca estándar; extensiones TypeScript para Pi; Bash; configuración JSON |
| Pruebas | Go testing/httptest, ejecución Node de extensiones con stubs de Pi y fixtures sintéticos |
| Documentación | Markdown y guías HTML con estilos; las guías no se auditaron visualmente ni por accesibilidad |
| Distribución | Makefile y `scripts/build-release.sh`, cuatro destinos macOS/Linux |

### Cobertura y validación

| Comprobación | Resultado |
|---|---|
| Congelado con `git rev-parse HEAD` y `git status --short` | SHA registrado; limpio al iniciar |
| `go test -race ./...` | PASA: raíz y `pkg/installer`; raíz desde caché, installer 60,598 s |
| `go test ./... -count=1` | PASA sin caché: raíz 0,231 s; installer 52,364 s |
| `go vet ./...` | PASA |
| `bash -n install.sh scripts/build-release.sh` | PASA |
| `git diff --check` | PASA para cambios rastreados; los archivos nuevos de auditoría se inspeccionan por separado |
| `go test ./pkg/installer -run '^TestADM20261117' -count=1 -v`, en `git archive HEAD` temporal | FALLA como se esperaba: reproduce H-01 y H-02; no es un fallo de la suite preexistente |
| Grafo: llamadas de `streamLines` y `FetchQuotaSummary`, ambos sentidos, profundidad 2 | Consultas completas para ese radio, sin páginas pendientes |
| Cobertura del índice: 22 rutas inspeccionadas | Generación `2026-11-14T12:00:58Z`; sin incidencias registradas. Cinco rutas con metadatos cambiados se contrastaron con fuente actual, no solo con el índice |
| `govulncheck`, `staticcheck`, `shellcheck`, `gitleaks`, `semgrep`, `tsc` | NO DISPONIBLES; no se instalaron |
| `npm audit --omit=dev` | No ejecutado: el repositorio no tiene manifiesto/lockfile npm propio; no se auditaron paquetes instalados en el HOME real |
| HCD: `mem hcd show` y `mem hcd verify` | Sin unidad activa; no hay evidencia HCD registrada ni aprobación implícita |
| Splash independiente sobre este informe y su reproductor | Sin hallazgos en revisión estática; no ejecutó pruebas ni validó las correcciones aplicadas |
| Pruebas autenticadas contra proveedores reales | No ejecutadas; se usó httptest local con token sintético |
| UI real de Pi, Linux nativo y cuatro binarios de distribución | No ejecutados en esta auditoría; pruebas locales en darwin/arm64, Node v26.13.0, Go go1.27.4 |

**Límites del grafo:** cobertura best-effort, no prueba de completitud. El índice no estaba vigilado. Las cinco rutas con metadatos cambiados fueron `pkg/installer/deps.go`, `pkg/installer/harness.go`, `extensions/gomemory-status.ts`, `install.sh` y `scripts/build-release.sh`; se obtuvo su contenido actual mediante codegraph o lectura directa. Las trazas estáticas no demuestran ejecución real de la UI.

**Contexto histórico:** se consultaron cierres y auditorías previas. No se reabren como hallazgos los problemas ya cubiertos por HCD-034 a HCD-037. H-01 es una cota de memoria diferente del defecto histórico de terminación de descendientes; H-02 no es una regresión demostrada de un cierre anterior.

## 2. Hallazgos y fugas detectadas

| ID | Severidad | Estado | Problema |
|---|---|---|---|
| H-01 | Media | CONFIRMADO | `streamLines` limita líneas, pero no bytes: una línea de 8 MiB se acumula y conserva completa |
| H-02 | Media | CONFIRMADO | HTTP 200 con `null` o `{}` se acepta como cuota válida, inventa etiqueta de plan y evita el fallback |

### H-01 — Salida de procesos sin cota de memoria

- **Categoría:** Fuga de recursos / disponibilidad por acumulación no acotada. No se ha demostrado una fuga permanente después de liberar el resultado.
- **Severidad:** Media; requiere que un comando de pruebas genere salida excesiva.
- **Estado:** CONFIRMADO.
- **Ubicación:** `extensions/gomemory-tdd.ts:508-516` (`tail`) y `extensions/gomemory-tdd.ts:579-593` (`buf`).
- **Ruta de alcance:** herramienta `run_test_suite` o `/tdd` → `doRunTestSuite` → `runCheck` → `streamLines` → stdout/stderr de un comando del proyecto.
- **Análisis:** `tail` conserva hasta 40 líneas, sin limitar su tamaño. Antes del salto de línea, `buf += chunk` tampoco tiene cota. Un comando que emite un JSON, un fichero minificado o un volcado grande en una sola línea puede consumir memoria creciente del proceso anfitrión. El timeout de diez minutos limita tiempo, no volumen; un proceso puede emitir mucho antes. El caso con muchas líneas grandes también amplía `tail`.
- **Evidencia:** `TestADM20261117StreamOutputBound` ejecuta Node con `process.stdout.write('x'.repeat(8 * 1024 * 1024))`. Resultado observado:

```text
{"code":0,"lines":1,"retained":8388608}
AssertionError: diagnóstico sin cota de bytes: 8388608
```

La prueba falla contra un presupuesto de diagnóstico propuesto de 64 KiB. Ese umbral es una política de cierre de esta auditoría, no un contrato previo documentado. Se confirmó retención proporcional al volumen; no se provocó un OOM del host.

#### Código corregido propuesto (no aplicado)

Dentro de `streamLines`, limitar tanto el diagnóstico retenido como la línea pendiente y fallar explícitamente ante una línea excesiva. Los siguientes bloques son sustituciones localizadas, no una reescritura completa de la función.

1. Sustituir la declaración de `tail` y `push` por:

```typescript
const MAX_LINE_CHARS = 64 * 1024;
const MAX_TAIL_LINE_CHARS = 1024;
const tail: string[] = [];
let outputLimited = false;
const push = (line: string) => {
  tail.push(line.slice(-MAX_TAIL_LINE_CHARS));
  if (tail.length > 40) tail.shift();
  try {
    onLine(line);
  } catch {
    // Un parser defectuoso no debe tumbar la ejecución.
  }
};
```

2. Sustituir el bucle de suscripción a stdout/stderr por:

```typescript
const rejectLongLine = () => {
  if (outputLimited) return;
  outputLimited = true;
  push("Salida rechazada: una línea excede el límite de 64 Ki caracteres");
  terminate("SIGTERM");
};

for (const stream of [child.stdout, child.stderr]) {
  let buf = "";
  stream?.setEncoding("utf8");
  stream?.on("data", (chunk: string) => {
    if (outputLimited) return;
    buf += chunk;
    let nl: number;
    while ((nl = buf.indexOf("\n")) >= 0) {
      if (nl > MAX_LINE_CHARS) {
        buf = "";
        rejectLongLine();
        return;
      }
      push(buf.slice(0, nl).replace(/\r$/, ""));
      buf = buf.slice(nl + 1);
    }
    if (buf.length > MAX_LINE_CHARS) {
      buf = "";
      rejectLongLine();
    }
  });
  stream?.on("end", () => {
    if (buf && !outputLimited) push(buf);
    buf = "";
  });
}
```

3. Inmediatamente antes de `resolve({ code: exitCode, ... })` en el manejador `close`, añadir:

```typescript
if (outputLimited && exitCode === 0) exitCode = 1;
```

Las cotas son en caracteres UTF-16; el diagnóstico ASCII del reproductor queda por debajo de 64 KiB. Para exigir un presupuesto exacto UTF-8 en cualquier entrada, usar contabilidad por `Buffer.byteLength`. El parser recibe las líneas completas admitidas, mientras que `tail` guarda extractos. La terminación conserva la escalada SIGKILL ya existente.

- **Validación del cierre:** el reproductor debe pasar; añadir casos de muchas líneas grandes, Unicode y stdout/stderr simultáneos. La línea excesiva debe devolver código no cero, sin tests marcados como éxito por truncado silencioso. Ejecutar las pruebas HCD-027/HCD-035 de cancelación y `go test -race ./...`. La propuesta no se aplicó ni se validó como corrección en esta sesión.

### H-02 — Respuestas vacías aceptadas como cuotas reales

- **Categoría:** Éxito falso / integridad de información.
- **Severidad:** Media; requiere una respuesta HTTP 200 sin el esquema esperado, por cambio de API, fallo de upstream o intermediario.
- **Estado:** CONFIRMADO.
- **Ubicación:** `pkg/installer/usage.go:93-108,128` (`FetchQuotaSummary`), `pkg/installer/usage.go:161-166` (`FetchAndFormatUsage`) y `pkg/installer/usage.go:211-213` (etiqueta de respaldo).
- **Ruta de alcance:** `pi-gomemory-setup usage --json` → `FetchQuotaSummary`; `usage`/`quota` → `FetchAndFormatUsage`. El sidebar invoca `usage --json` desde `extensions/gomemory-sidebar.ts:758`; su validador acepta `groups: []` en `extensions/gomemory-sidebar.ts:584-595`.
- **Análisis:** deserializar a una estructura Go no valida la presencia de campos. `null` y `{}` dejan el objeto con valores cero sin error. Ambos flujos devuelven éxito y el primero deja de probar endpoints alternativos. La etiqueta `Google AI Pro (g1-pro-tier)` aparece pese a no haber sido proporcionada por upstream. No se fabrican porcentajes en esta reproducción; sí una identidad de plan y un estado de éxito sin cuotas.
- **Evidencia:** servidor httptest devuelve HTTP 200 con `null`, y después con `{}`; cuenta sintética. En ambos casos:

```text
summary=&{PlanLabel:Google AI Pro (g1-pro-tier) Email: Groups:[]} error=<nil> requests=1
text="\x1b[38;2;34;227;247mGoogle AI Pro (g1-pro-tier)\x1b[0m\n\n" error=<nil> requests=1
```

Se configuraron dos endpoints; solo se consultó el primero en cada llamada. Las cuatro comprobaciones negativas —dos cuerpos por dos funciones— fallan.

#### Código corregido propuesto (no aplicado)

Añadir una validación compartida en `usage.go`:

```go
func parseQuotaResponse(body []byte) (*QuotaSummaryResponse, error) {
    var quota QuotaSummaryResponse
    if err := json.Unmarshal(body, &quota); err != nil {
        return nil, fmt.Errorf("respuesta de cuota ilegible: %w", err)
    }
    buckets := 0
    for _, group := range quota.Groups {
        for _, bucket := range group.Buckets {
            if bucket.RemainingFraction < 0 || bucket.RemainingFraction > 1 {
                return nil, fmt.Errorf("respuesta de cuota con fracción fuera de rango")
            }
            buckets++
        }
    }
    if buckets == 0 {
        return nil, fmt.Errorf("respuesta de cuota sin límites; no se puede confirmar el uso")
    }
    return &quota, nil
}
```

En ambos bucles de endpoints, sustituir la deserialización local por:

```go
quota, err := parseQuotaResponse(body)
if err != nil {
    lastErr = fmt.Errorf("%s: %w", ep, err)
    continue
}
```

En el flujo de texto, el retorno pasa a `return renderQuotaView(quota, account.Email), nil`. En `FetchQuotaSummary`, el acceso `quota.Groups` sigue siendo válido. Sustituir los dos fallbacks de etiqueta por `Plan no informado` evita atribuir un plan específico cuando existen cuotas válidas, pero falta `planLabel`.

- **Validación del cierre:** el reproductor debe pasar. Añadir un caso donde el primer endpoint devuelve `{}` y el segundo una cuota válida para comprobar que funciona el fallback; probar grupos vacíos, cuotas válidas y fracciones fuera de rango. La presencia de `remainingFraction` debe validarse explícitamente si el contrato exige distinguir cero real de campo ausente; esta mejora adicional no está cubierta por el reproductor actual. Ejecutar la suite completa y comprobar que el CLI devuelve salida no cero ante todos los endpoints inválidos. La propuesta no se aplicó ni se validó como corrección en esta sesión.

## 3. Recomendaciones de resiliencia

1. **Prioridad inmediata:** cerrar H-01 y H-02 con pruebas negativas permanentes. No basta con que la suite actual permanezca verde.
2. **Presupuestos de salida:** separar límite por línea, diagnóstico retenido y volumen total. No transportar volcados completos de procesos a mensajes del agente. Definir una política de redacción antes de conservar o devolver salidas que podrían contener secretos; no se ha confirmado aquí una fuga concreta.
3. **Contratos upstream:** compartir parsers semánticos entre CLI y UI y representar datos ausentes como no disponibles. Conservar cero legítimo sin confundirlo con un campo omitido.
4. **Shell:** incorporar tiempos de conexión y total a curl/wget en `install.sh`; actualmente no están configurados. Mantener temporales impredecibles y limpieza con trap también al copiar el ejecutable al destino. Se recomienda como hardening, sin elevarlo a hallazgo reproducido en esta entrega.
5. **Cadena de suministro:** incorporar `govulncheck`, `staticcheck`, `shellcheck` y escaneo de secretos en un entorno de CI preparado. Auditar versiones transitivas de paquetes Pi en un entorno aislado; los checksums no sustituyen una firma de procedencia frente al compromiso del origen.
6. **Cobertura de integración:** complementar los stubs con un smoke test real de Pi y ejecuciones Linux/macOS. No interpretar pruebas de importación o mocks como validación completa del SDK ni de la UI servida.
7. **Segunda revisión:** para autorizar correcciones con mayor confianza, someter los hallazgos a ACR. Esta ADM no representa consenso de dos revisores ni aprobación HCD.

### Revisión independiente del informe

Se ejecutó `splash_review` pasando únicamente la ruta del documento y una lente de revisión, sin historial ni memoria. Resultado leído: **«sin hallazgos»**. El revisor contrastó estáticamente H-01/H-02, su severidad, el reproductor y las correcciones propuestas. No ejecutó pruebas ni correcciones. Esto no cierra los defectos ni constituye consenso ACR o aprobación HCD.

### Reproducción independiente

Archivo: `docs/audits/2026-11-17-adm-repro_test.go.txt`.

Desde el repositorio auditado, sin instalar dependencias:

```bash
ROOT="$PWD"
WORK="$(mktemp -d /tmp/pi-gomemory-adm.XXXXXX)"
git archive b22a5c4a77607d9fe1860a30e0633dd22b76488e | tar -x -C "$WORK"
cp "$ROOT/docs/audits/2026-11-17-adm-repro_test.go.txt" \
   "$WORK/pkg/installer/adm_20261117_test.go"
(cd "$WORK" && go test ./pkg/installer -run '^TestADM20261117' -count=1 -v)
```

**Resultado esperado en la revisión auditada:** salida 1 y los dos tests principales en FAIL. Los fixtures escriben solo en directorios temporales y consultan un servidor local; no leen cuentas reales. Conservar la copia si se necesita investigar o eliminarla después de revisar su ruta.

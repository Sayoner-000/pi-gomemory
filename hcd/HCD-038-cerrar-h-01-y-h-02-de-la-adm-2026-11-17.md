# Cerrar H-01 y H-02 de la ADM 2026-11-17: salida de procesos con cota y cuotas vacías rechazadas

ID: HCD-038
Tipo: fix
Carril: fast
Motivo: Fix con reproducción clara y correcciones locales; las decisiones abiertas son independientes y se resuelven con clarify

## Intención
streamLines deja de retener salida sin cota de bytes, y las respuestas HTTP 200 sin cuotas (null o {}) dejan de contar como éxito, ya no muestran una etiqueta de plan inventada y pasan al endpoint de respaldo

## Reproducción
Origen: ADM 2026-11-17 (`docs/audits/2026-11-17-defensive-qa.md`), revisión b22a5c4 (v1.13.7).

```bash
cp docs/audits/2026-11-17-adm-repro_test.go.txt pkg/installer/adm_20261117_test.go
go test ./pkg/installer -run '^TestADM20261117' -count=1 -v
```

Hoy fallan los dos tests:
- `TestADM20261117StreamOutputBound` (H-01): un hijo emite 8 MiB en una sola línea y `streamLines` la retiene entera en `tail` (`{"code":0,"lines":1,"retained":8388608}`). El búfer `buf` previo al salto de línea (`extensions/gomemory-tdd.ts:579-593`) tampoco tiene cota.
- `TestADM20261117QuotaRejectsMissingData` (H-02, subtests `null` y `{}`): `FetchQuotaSummary` y `FetchAndFormatUsage` (`pkg/installer/usage.go:93-108,161-166`) devuelven éxito con `Groups:[]`, la etiqueta «Google AI Pro (g1-pro-tier)» (`pkg/installer/usage.go:99-102,211-214`) y una sola petición aunque hay dos endpoints.

## Alcance
Incluido: cerrar H-01 (cotas en `streamLines` y el check `hcd` sin depender de `tail`) y H-02 (`parseQuotaResponse` y la etiqueta de respaldo), con pruebas permanentes. En detalle:
- H-01: cota por línea en el búfer de `streamLines`: 64 Ki caracteres por stream (Q2). Una línea que supera el límite termina el proceso con SIGTERM, usando la escalada a SIGKILL que ya existe, deja en el diagnóstico «Salida rechazada: una línea excede el límite de 64 Ki caracteres» y devuelve código 1. Nunca se trunca para seguir (Q1).
- H-01: cota de retención en `tail`: extractos de 1 Ki caracteres por línea, 40 líneas como máximo y un total de 64 KiB medidos en bytes UTF-8 con `Buffer.byteLength` (Q2). El parser sigue recibiendo por `onLine` cada línea completa admitida.
- H-01, defecto que la corrección activaría: el check `hcd` (`runCheck`, `extensions/gomemory-tdd.ts:1396`) parsea `res.tail.join("\n")` como JSON, y `mem hcd show --json` imprime una sola línea. Recortar las líneas de `tail` rompería ese parseo, así que el check `hcd` deja de depender de `tail`: acumula sus líneas completas vía `onLine`, que ya están acotadas por el límite por línea.
- H-02: un parser compartido `parseQuotaResponse` en `pkg/installer/usage.go` que rechaza respuestas sin ningún bucket o con `remainingFraction` fuera de [0,1]. Lo usan los dos bucles, que pasan al endpoint siguiente y, si todos fallan, devuelven error: el CLI sale con código no cero y el sidebar recibe un error en lugar de `groups: []`.
- H-02: cuando upstream no envía `planLabel`, `usage` y `usage --json` muestran «Plan no informado» (Q3). La cabecera fija del sidebar, «▼ Google AI Pro Quota», no cambia.
- Pruebas permanentes: el reproductor pasa a la suite como `pkg/installer/hcd038_test.go`, con estos casos además: muchas líneas grandes, stdout y stderr simultáneos, Unicode, check `hcd` con JSON de más de 1 Ki, fallback del primer endpoint `{}` al segundo con cuota válida, grupos vacíos y fracción fuera de rango.

Excluido: distinguir una fracción ausente de un cero, el límite de volumen total, el hardening de la ADM, commit y release. En detalle:
- Distinguir `remainingFraction` ausente de un cero real (exige cambiar el struct a puntero; la ADM lo marca como mejora no cubierta por el reproductor).
- Un límite de volumen total de salida por proceso (los parsers consumen en streaming; la cota es de retención, no de throughput).
- Las recomendaciones de hardening de la ADM (§3.4–3.6: timeouts de curl en `install.sh`, govulncheck/staticcheck en CI, smoke test real de Pi).
- Commit, release y cualquier cambio en las otras extensiones.

## ¿Qué pasa si no lo hacemos?
- H-01: un comando de pruebas que emite un JSON, un bundle minificado o un volcado en una sola línea hace crecer sin límite la memoria del proceso de Pi mientras dura la ejecución (hasta 10 minutos de timeout), y esa salida completa queda retenida en `tail`.
- H-02: ante un cambio de API, un fallo upstream o un intermediario que responde 200 con `null` o `{}`, el usuario ve un éxito sin cuotas, con un plan que upstream no informó, y el sidebar se queda en «Cargando cuota...» indefinidamente. Además el endpoint de respaldo nunca se prueba.

## Alternativa más barata
- H-02: solo cambiar la etiqueta de respaldo. Es más barato, pero el éxito falso y el fallback perdido se quedan, y el reproductor sigue fallando. Se descarta.
- H-01: limitar solo `tail` sin tocar `buf`. El acumulador previo al salto de línea seguiría sin cota, que es la mitad del defecto. Se descarta.
No hay una alternativa más barata que cierre los dos hallazgos; cada corrección es local a su función.


## Aclaraciones
- [x] Q1. H-01: ¿una línea que supera el límite termina el proceso y fuerza código no cero, o se trunca y la ejecución sigue? → Terminar y fallar: SIGTERM con escalada a SIGKILL ya existente, aviso en el diagnóstico y código de salida 1
- [x] Q2. H-01: ¿límite de 64 Ki caracteres por línea y presupuesto total de tail de 64 KiB, como propone la ADM? → 64 Ki caracteres por línea; tail guarda extractos de 1 Ki por línea, 40 líneas como máximo, con un total de 64 KiB medidos en bytes UTF-8 (Buffer.byteLength)
- [x] Q3. H-02: si upstream no envía planLabel pero sí cuotas válidas, ¿se muestra «Plan no informado» o se conserva «Google AI Pro (g1-pro-tier)»? → «Plan no informado» en usage y usage --json; la cabecera fija del sidebar «▼ Google AI Pro Quota» queda fuera del alcance

## Checklist
- [x] 1. La reproducción ya no falla (reproductor de la ADM copiado como pkg/installer/hcd038_test.go) → go test ./pkg/installer -run '^TestADM20261117' -count=1
- [x] 2. Terminar con código 1 y aviso en el diagnóstico la ejecución cuya línea pendiente supera 64 Ki caracteres en streamLines (extensions/gomemory-tdd.ts) (Q1, Q2) → go test ./pkg/installer -run '^TestHCD038LongLineFails$' -count=1
- [x] 3. Acotar tail a 40 extractos de 1 Ki y 64 KiB UTF-8 con muchas líneas grandes, Unicode y stdout y stderr simultáneos (extensions/gomemory-tdd.ts) (Q2) → go test ./pkg/installer -run '^TestHCD038TailBudget$' -count=1
- [x] 4. Leer el JSON del check hcd desde onLine en lugar de tail, con un mem falso que imprime una línea de más de 1 Ki (extensions/gomemory-tdd.ts) → go test ./pkg/installer -run '^TestHCD038HcdCheckLongJSON$' -count=1
- [x] 5. Rechazar en parseQuotaResponse las respuestas sin buckets o con remainingFraction fuera de [0,1] (pkg/installer/usage.go) → go test ./pkg/installer -run '^TestHCD038QuotaParse$' -count=1
- [x] 6. Pasar al segundo endpoint cuando el primero responde {} y devolver su cuota válida, en usage y usage --json (pkg/installer/usage.go) → go test ./pkg/installer -run '^TestHCD038QuotaFallback$' -count=1
- [x] 7. Mostrar «Plan no informado» cuando falta planLabel, en usage y usage --json (pkg/installer/usage.go) (Q3) → go test ./pkg/installer -run '^TestHCD038PlanLabelMissing$' -count=1
- [x] 8. Mantener verde la suite completa con el detector de carreras (incluye las pruebas de cancelación de HCD-027/HCD-035) → go test -race ./... -count=1
- [x] 9. Mantener limpio el análisis estático → go vet ./...

## Evidencia
<!-- La escribe `mem hcd verify`. Lo escrito a mano aquí no cuenta. -->
- 2026-11-17 06:18 -05:00 · tarea 1 · rojo observado · caducada · `go test ./pkg/installer -run '^TestADM20261117' -count=1` (salida 1)
                expected: true,
                operator: '==',
                diff: 'simple'
              }
              Node.js v26.13.0
      --- FAIL: TestADM20261117QuotaRejectsMissingData (0.00s)
          --- FAIL: TestADM20261117QuotaRejectsMissingData/null (0.00s)
              hcd038_test.go:44: summary=&{PlanLabel:Google AI Pro (g1-pro-tier) Email: Groups:[]} error=<nil> requests=1
              hcd038_test.go:46: respuesta sin cuotas aceptada por FetchQuotaSummary
              hcd038_test.go:50: text="\x1b[38;2;34;227;247mGoogle AI Pro (g1-pro-tier)\x1b[0m\n\n" error=<nil> requests=1
              hcd038_test.go:52: respuesta sin cuotas aceptada por FetchAndFormatUsage
          --- FAIL: TestADM20261117QuotaRejectsMissingData/{} (0.00s)
              hcd038_test.go:44: summary=&{PlanLabel:Google AI Pro (g1-pro-tier) Email: Groups:[]} error=<nil> requests=1
              hcd038_test.go:46: respuesta sin cuotas aceptada por FetchQuotaSummary
              hcd038_test.go:50: text="\x1b[38;2;34;227;247mGoogle AI Pro (g1-pro-tier)\x1b[0m\n\n" error=<nil> requests=1
              hcd038_test.go:52: respuesta sin cuotas aceptada por FetchAndFormatUsage
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.910s
      FAIL
- 2026-11-17 06:19 -05:00 · tarea 2 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD038LongLineFails$' -count=1` (salida 1)
      --- FAIL: TestHCD038LongLineFails (20.96s)
          hcd038_test.go:90: TestHCD038LongLineFails: FAIL línea abierta: debió cortarse por el límite, no por timeout | línea abierta: tardó 20010 ms | línea abierta: falta el aviso en tail: ["xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"] | línea cerrada: código 0, se esperaba 1 | línea cerrada: el parser recibió una línea de más de 64 Ki
      FAIL
      FAIL	pi-gomemory/pkg/installer	21.306s
      FAIL
- 2026-11-17 06:20 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD038LongLineFails$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.233s
- 2026-11-17 06:20 -05:00 · tarea 3 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD038TailBudget$' -count=1` (salida 1)
      --- FAIL: TestHCD038TailBudget (0.22s)
          hcd038_test.go:118: TestHCD038TailBudget: FAIL tail guarda extractos de más de 1 Ki caracteres | tail retiene 585003 bytes, máximo 65536
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.412s
      FAIL
- 2026-11-17 06:20 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD038TailBudget$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.363s
- 2026-11-17 06:20 -05:00 · tarea 4 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD038HcdCheckLongJSON$' -count=1` (salida 1)
      --- FAIL: TestHCD038HcdCheckLongJSON (0.26s)
          hcd038_test.go:136: TestHCD038HcdCheckLongJSON: FAIL el check hcd debe mostrar la unidad HCD-999: ["Unidad HCD activa=passed"]
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.561s
      FAIL
- 2026-11-17 06:20 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD038HcdCheckLongJSON$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.372s
- 2026-11-17 06:21 -05:00 · tarea 5 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD038QuotaParse$' -count=1` (salida 1)
      # pi-gomemory/pkg/installer [pi-gomemory/pkg/installer.test]
      pkg/installer/hcd038_test.go:155:16: undefined: parseQuotaResponse
      pkg/installer/hcd038_test.go:160:13: undefined: parseQuotaResponse
      FAIL	pi-gomemory/pkg/installer [build failed]
      FAIL
- 2026-11-17 06:21 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD038QuotaParse$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.224s
- 2026-11-17 06:21 -05:00 · tarea 6 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD038QuotaFallback$' -count=1` (salida 1)
      --- FAIL: TestHCD038QuotaFallback (0.00s)
          hcd038_test.go:197: FetchQuotaSummary debe devolver la cuota del segundo endpoint: &{PlanLabel:Google AI Pro (g1-pro-tier) Email:synthetic@example.invalid Groups:[]} err=<nil>
          hcd038_test.go:201: FetchAndFormatUsage debe mostrar la cuota del segundo endpoint: "\x1b[38;2;34;227;247mGoogle AI Pro (g1-pro-tier)\x1b[0m\n\x1b[38;2;145;169;197mCuenta: synthetic@example.invalid\x1b[0m\n\n" err=<nil>
          hcd038_test.go:204: cada función debe consultar los dos endpoints: peticiones=[2 0]
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.197s
      FAIL
- 2026-11-17 06:21 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^TestHCD038QuotaFallback$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.202s
- 2026-11-17 06:21 -05:00 · tarea 7 · rojo observado · caducada · `go test ./pkg/installer -run '^TestHCD038PlanLabelMissing$' -count=1` (salida 1)
      --- FAIL: TestHCD038PlanLabelMissing (0.00s)
          hcd038_test.go:214: usage --json debe decir «Plan no informado»: &{PlanLabel:Google AI Pro (g1-pro-tier) Email:synthetic@example.invalid Groups:[{DisplayName:Modelos Buckets:[{DisplayName:Bucket-A RemainingFraction:0.25 RemainingPercent:25 ResetTime: ResetIn:n/a}]}]} err=<nil>
          hcd038_test.go:218: usage debe decir «Plan no informado» sin inventar el plan: "\x1b[38;2;34;227;247mGoogle AI Pro (g1-pro-tier)\x1b[0m\n\x1b[38;2;145;169;197mCuenta: synthetic@example.invalid\x1b[0m\n\n \x1b[38;2;6;193;238mModelos\x1b[0m\n   \x1b[38;2;244;198;107m[#####---------------]\x1b[0m Bucket-A: 25.0% left · resets n/a\n\n" err=<nil>
      FAIL
      FAIL	pi-gomemory/pkg/installer	0.326s
      FAIL
- 2026-11-17 06:22 -05:00 · tarea 7 · verde · caducada · `go test ./pkg/installer -run '^TestHCD038PlanLabelMissing$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.252s
- 2026-11-17 06:22 -05:00 · tarea 1 · verde · caducada · `go test ./pkg/installer -run '^TestADM20261117' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.207s
- 2026-11-17 06:22 -05:00 · tarea 8 · verde · caducada · `go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.209s
      ok  	pi-gomemory/pkg/installer	58.557s
- 2026-11-17 06:23 -05:00 · tarea 9 · verde · caducada · `go vet ./...` (salida 0)
- 2026-11-17 06:23 -05:00 · tarea 2 · verde · caducada · `go test ./pkg/installer -run '^TestHCD038LongLineFails$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.263s
- 2026-11-17 06:23 -05:00 · tarea 3 · verde · caducada · `go test ./pkg/installer -run '^TestHCD038TailBudget$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.357s
- 2026-11-17 06:23 -05:00 · tarea 4 · verde · caducada · `go test ./pkg/installer -run '^TestHCD038HcdCheckLongJSON$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.362s
- 2026-11-17 06:23 -05:00 · tarea 5 · verde · caducada · `go test ./pkg/installer -run '^TestHCD038QuotaParse$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.148s
- 2026-11-17 06:23 -05:00 · tarea 6 · verde · caducada · `go test ./pkg/installer -run '^TestHCD038QuotaFallback$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.143s
- 2026-11-17 06:25 -05:00 · tarea 1 · verde · vigente · `go test ./pkg/installer -run '^TestADM20261117' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.528s
- 2026-11-17 06:25 -05:00 · tarea 2 · verde · vigente · `go test ./pkg/installer -run '^TestHCD038LongLineFails$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	1.229s
- 2026-11-17 06:25 -05:00 · tarea 3 · verde · vigente · `go test ./pkg/installer -run '^TestHCD038TailBudget$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.359s
- 2026-11-17 06:25 -05:00 · tarea 4 · verde · vigente · `go test ./pkg/installer -run '^TestHCD038HcdCheckLongJSON$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.369s
- 2026-11-17 06:25 -05:00 · tarea 5 · verde · vigente · `go test ./pkg/installer -run '^TestHCD038QuotaParse$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.161s
- 2026-11-17 06:25 -05:00 · tarea 6 · verde · vigente · `go test ./pkg/installer -run '^TestHCD038QuotaFallback$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.161s
- 2026-11-17 06:25 -05:00 · tarea 7 · verde · vigente · `go test ./pkg/installer -run '^TestHCD038PlanLabelMissing$' -count=1` (salida 0)
      ok  	pi-gomemory/pkg/installer	0.157s
- 2026-11-17 06:25 -05:00 · tarea 8 · verde · vigente · `go test -race ./... -count=1` (salida 0)
      ok  	pi-gomemory	1.333s
      ok  	pi-gomemory/pkg/installer	58.075s
- 2026-11-17 06:26 -05:00 · tarea 9 · verde · vigente · `go vet ./...` (salida 0)

## Siguiente
Cerrada con aprobación sobre la revisión ffda9ca46b40.

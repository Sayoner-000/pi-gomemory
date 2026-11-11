# Informe ACR + ADM (seguimiento) — pi-gomemory

**Fecha:** 2026-11-11
**Revisión auditada:** `31f814aea5f5dfe9a906851c024d3977e3c9d99f` (v1.7.4) **+ árbol sucio** (10 entradas: `main.go`, `usage.go`, `claude_usage.go`, `gomemory-sidebar.ts`, pruebas HCD-014/015 y sus unidades).
**Digest del conjunto de producción** (`main.go pkg extensions install.sh scripts config Makefile go.mod embed.go`, incluido el árbol sucio): `5ba7d7f57e1341cbe1748417f2343e4a81dbe634fa5f677fb026f15e7b956994`
**Alcance:** repositorio completo (≈7 000 líneas: Go ≈3 900 producción, TS ≈1 000, Bash ≈190). Se excluyen `dist/`, `skills/` (contenido Markdown embebido) y `.memory/`.
**Línea base:** auditoría previa `2026-11-11-acr-adm-review.md` (HEAD `781e39d`, C-001…C-006 cerrados en HCD-012).

---

## 0. Veredicto ACR

| Campo | Valor |
|---|---|
| Independencia | **DEGRADADA**: un único revisor efectivo (Revisor A). |
| Revisor B | **No ejecutó**: `splash_review` falló con «No API key found for antigravity». Pi no tiene extensión de subagentes (memoria 11). |
| Rondas | 1 revisión inicial, 0 correcciones (solo lectura), 0 re-juicios |
| Confirmados por consenso | 0 (no hay segundo revisor) |
| Confirmados por reproducción | 6 (H-01…H-06) |
| Sospechosos | 3 |

**VERDICT: INCOMPLETE** — el protocolo exige dos revisores independientes (§31) y no se sustituye una revisión por dos. Ningún hallazgo CRITICAL/HIGH quedó abierto. Los hallazgos se sostienen por reproducción ejecutable, no por corroboración cruzada. Para llegar a APPROVED, lanzar `mem review` con dos revisores sobre el digest anterior.

---

## 1. Resumen de salud (ADM)

- **Lenguajes:** Go 1.27 (producción: `pkg/installer`, `main.go`; pruebas: 24 `*_test.go`), TypeScript ESM (7 extensiones de Pi), Bash (`install.sh`, `build-release.sh`, launcher generado), JSON de configuración.
- **Riesgo general: MEDIO.** Sin críticos ni altos. Hay un camino de pérdida de datos del usuario (H-01), dos éxitos falsos (H-02, H-03) y un guardarraíl que se evade con facilidad (H-04).
- **Lo que está bien:** escrituras atómicas (`writeFileAtomic`) en todo el instalador salvo dos del log de desinstalación; verificación SHA256 antes de reemplazar el binario; `DepsStatusFor` no ejecuta comandos de la caché editable; desinstalación basada en hashes (no borra lo editado); `readJSONObject` falla cerrado ante JSON inválido; tokens de Antigravity nunca llegan a mensajes de error ni a la salida JSON; `go vet` limpio y `go test -race` verde (cobertura 79,1 %).
- **Cierres previos (C-001…C-006): todos vigentes.** No hay `| bash` en el código; `splash_review` confina con `realpath`; las únicas `os.WriteFile` no atómicas están en `uninstall.go:499,520`; el sidebar dice `Configured`; `bytes.NewReader`; `powershell` en `tools`. Ninguno regresó.
- **Resultado:** 12 hallazgos → **0 críticos, 0 altos, 5 medios, 6 bajos, 1 informativo.** No es una garantía de ausencia de defectos: ver límites.

### Cobertura y validación

| Comprobación | Resultado |
|---|---|
| `go vet ./...` | Pasa |
| `go test -race ./...` | Pasa (`pkg/installer` 4,7 s) |
| `go test -cover ./...` | 79,1 % de `pkg/installer` |
| `gofmt -l .` | **Falla**: `pkg/installer/claude_usage.go` (nuevo, sin commit) y `hcd011_plan_test.go` |
| `bash -n install.sh scripts/build-release.sh` | Pasa |
| `shellcheck`, `staticcheck`, `govulncheck`, `gitleaks`, `semgrep`, `tsc` | **No disponible** en el host (no se instalaron). `govulncheck` queda sin ejecutar: la alcanzabilidad de CVE en la toolchain Go 1.27.4 no está verificada. |
| Secretos en el repositorio | Inspección dirigida (`token`, `Bearer`, `access`): solo lectura de `antigravity-accounts.json` en tiempo de ejecución; sin valores en código. Sin `gitleaks`. |
| Reproducciones | En copia desechable (`/tmp`, ya eliminada); el repositorio no se modificó. Excepto este informe, no se escribió nada. |
| Splash (Revisor B) | **No disponible** — sin credencial del proveedor. |

---

## 2. Hallazgos

| ID | Severidad | Estado | Problema |
|---|---|---|---|
| H-01 | Media | CONFIRMADO | `pruneLegacyPiSkills` borra con `RemoveAll` carpetas de skill con archivos del usuario |
| H-02 | Media | CONFIRMADO | `uninstall` devuelve éxito (exit 0) aunque el plan contenga acciones fallidas |
| H-03 | Media | CONFIRMADO | `doctor` sin timeout y con comprobaciones MCP omitidas en silencio |
| H-04 | Media | CONFIRMADO | Modo `/plan` se evade con `bash -c`, `python3 -c`, `make`, etc. |
| H-05 | Baja | CONFIRMADO | `claude-usage`: «Hoy/Semana» mezcla fecha local con fecha UTC |
| H-06 | Baja | CONFIRMADO | `claude-usage`: atribución de proyecto no determinista con filas duplicadas |
| H-07 | Media | SOSPECHOSO (heredado) | Splash conserva `read/grep/find/ls` sin confinar (riesgo residual de H-02 previo) |
| H-08 | Baja | SOSPECHOSO | Sidebar: error de cuota en JSON crudo y sin límite de reintentos |
| H-09 | Baja | SOSPECHOSO | `SyncSkillRepo` borra enlaces propios; `uninstall --packages` ignora el fallo de `pi remove` |
| H-10 | Baja | CONFIRMADO | Cadena de suministro y proceso de release: sin CI, SHA256SUMS sin firma, `curl \| bash` desde `main` |
| H-11 | Baja | CONFIRMADO | Launcher: `deps-refresh` en red en cada arranque y `skills-sync` destructivo en cada arranque |
| H-12 | Info | CONFIRMADO | Higiene: duplicación, `Expires` ignorado, UA suplantado, árbol sin commit |

---

#### H-01 — `pruneLegacyPiSkills` puede borrar archivos del usuario

- **Categoría:** Integridad / pérdida de datos
- **Severidad:** Media · **Estado:** CONFIRMADO (reproducido)
- **Ubicación:** `pkg/installer/hostskills.go:244-257` (`pruneLegacyPiSkills`)
- **Ruta de alcance:** `pi-gomemory-setup install|skills|skills-sync` → `InstallSkills` → `RegisterSkillRepo`. **El launcher `pi-gomemory` ejecuta `skills-sync` en cada arranque** (`harness.go`, bloque 3c), por lo que el camino se recorre en cada sesión.
- **Análisis:** para carpetas de `~/.pi/agent/skills` sin marca `.pi-gomemory`, se decide borrar solo por igualdad de `SKILL.md` con el del repositorio central, y luego se ejecuta `os.RemoveAll(p)` sobre la carpeta entera. Una skill con `scripts/`, `references/` o notas propias y `SKILL.md` idéntico pierde todo. La desinstalación, en cambio, exige «solo `SKILL.md` + marca» (`isRemovableEmbeddedCopy`): las dos rutas aplican criterios distintos de propiedad. Un `SKILL.md` idéntico es una prueba de origen, no de que la carpeta no tenga contenido añadido.
- **Evidencia (datos sintéticos):** prueba en copia temporal: `~/.pi/agent/skills/foo/{SKILL.md,scripts/mine.sh}` con `SKILL.md` igual al del repositorio → tras `pruneLegacyPiSkills`, `scripts/mine.sh` ya no existe (`PERDIDA DE DATOS: stat …/scripts/mine.sh: no such file or directory`).
- **Corrección mínima:**
  ```go
  // hostskills.go: exigir carpeta con solo SKILL.md (y opcionalmente la marca)
  func onlySkillFile(dir string) bool {
      es, err := os.ReadDir(dir)
      if err != nil { return false }
      for _, e := range es {
          if e.Name() != "SKILL.md" && e.Name() != skillMarkerFile { return false }
      }
      return true
  }
  // en la rama heredada:
  if bytes.Equal(bytes.TrimSpace(repoContent), bytes.TrimSpace(piContent)) && onlySkillFile(p) { remove = true }
  ```
  Si la carpeta tiene archivos extra: conservarla (la skill del repositorio queda en conflicto de nombre y Pi la resolverá con su regla propia).
- **Validación del cierre:** la prueba anterior debe pasar (el archivo sobrevive) y fallar sin el cambio; añadir caso «solo `SKILL.md` idéntico → se retira».

#### H-02 — `uninstall` devuelve éxito con acciones fallidas

- **Categoría:** Éxito falso · **Severidad:** Media · **Estado:** CONFIRMADO (reproducido)
- **Ubicación:** `pkg/installer/uninstall.go:470-473` (`Uninstall`); acciones `ActionFail` creadas en `:197,203,287,344`.
- **Ruta de alcance:** `pi-gomemory-setup uninstall [--yes]`.
- **Análisis:** `Pending()` solo cuenta `ActionRemove`. Si `mcp.json`, `settings.json` o el manifiesto son ilegibles y no hay nada más que retirar, `Pending()==0` imprime «✓ No queda nada que retirar de pi-gomemory.» y retorna `nil` (exit 0) **sin mostrar el plan**. La persona cree que desinstaló, pero el tema, el registro de skills y los servidores MCP siguen en la configuración.
- **Evidencia:** `settings.json="{corrupt"`, sin otros archivos → `err=<nil> out="✓ No queda nada que retirar de pi-gomemory.\n"`.
- **Corrección mínima:**
  ```go
  if plan.Pending() == 0 {
      if n := plan.Count(ActionFail); n > 0 {
          PrintPlan(w, plan)
          return fmt.Errorf("%d comprobaciones fallaron; no se puede confirmar que no quede nada", n)
      }
      fmt.Fprintln(w, "✓ No queda nada que retirar de pi-gomemory.")
      return nil
  }
  ```
  Misma lógica en `--dry-run` (exit distinto de 0 si hay `fallido`).
- **Validación del cierre:** la prueba debe devolver error y listar la acción fallida.

#### H-03 — `doctor`: sin timeout y comprobaciones MCP omitidas en silencio

- **Categoría:** Fuga de recursos / Éxito falso · **Severidad:** Media · **Estado:** CONFIRMADO (reproducido)
- **Ubicación:** `pkg/installer/doctor.go:86-96` (`exec.Command("pi","mcp","list","--json").Output()`).
- **Ruta de alcance:** `pi-gomemory-setup doctor` **y el final de `install`** (`InstallAll` → `PrintDoctor(RunDoctor(opts))`).
- **Análisis:** (a) sin `context`/timeout: un `pi mcp list` colgado bloquea la instalación después de «✓ Instalación completa»; (b) si el comando falla o devuelve JSON inválido, no se emite ninguna fila «MCP …»: el diagnóstico parece sano sin haber comprobado nada.
- **Evidencia:** `pi` falso que imprime `not json` → 0 filas MCP. `pi` falso con `sleep 8` → `RunDoctor tardó 8s` (indefinido con un bloqueo real).
- **Corrección mínima:**
  ```go
  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()
  out, err := exec.CommandContext(ctx, "pi", "mcp", "list", "--json").Output()
  if err != nil || json.Unmarshal(out, &listResp) != nil {
      items = append(items, CheckItem{Name: "MCP en Pi", Status: "WARN", Details: "no se pudo consultar `pi mcp list`"})
  } else { /* filas actuales */ }
  ```
- **Validación del cierre:** pruebas con `pi` falso (JSON roto → fila WARN; `sleep` → retorna en ≈timeout).

#### H-04 — El modo `/plan` se evade con comandos habituales

- **Categoría:** Inseguridad / guardarraíl · **Severidad:** Media · **Estado:** CONFIRMADO (reproducido con node)
- **Ubicación:** `extensions/gomemory-plan.ts:25-29` + `extensions/gomemory-plan-patterns.json` (lista de bloqueo por regex sobre el texto del comando).
- **Ruta de alcance:** cualquier llamada a la herramienta `bash` con el modo plan activo.
- **Análisis:** el bloqueo exige que el verbo vaya precedido de `;`, `&`, `|`, `(` o inicio. Quedan fuera los verbos dentro de comillas y los intérpretes.
- **Evidencia (resultado de `bashBlocked`):**

  | Bloquea | **Permite** |
  |---|---|
  | `rm -rf build`, `echo hi > f`, `sed -i …` | `bash -c 'rm -rf build'`, `sh -c "touch f"`, `python3 -c "open('f','w')…"`, `node -e "…writeFileSync…"`, `perl -pi -e …`, `find . -delete`, `xargs rm`, `env rm f`, `dd of=f`, `git apply`, `patch`, `curl -o`, `wget -O`, `install`, `rsync`, `unlink`, `go build -o`, `make`, `git clean -fd`, `git pull` |

  Cuando el agente «piensa» en un cambio, la ruta natural (`python3 -c`, `make`) ya está permitida.
- **Corrección:** una lista negra de regex no es una frontera de seguridad. Opciones: (1) lista **blanca** de comandos de lectura (`ls, cat, grep, rg, git status|diff|log|show, mem …, go vet|test`) y bloqueo por defecto del resto; (2) como mínimo, bloquear `bash|sh|zsh -c`, `python*|node|perl|ruby -[ce]`, `xargs`, `env`, `make`, `find … -delete|-exec`. Documentar en el README que es un guardarraíl, no un sandbox.
- **Validación del cierre:** tabla de casos anterior como prueba de regresión (`TestHCD012Plan*` ya ejecuta node).

#### H-05 — `claude-usage`: «Hoy/Semana» con fecha local contra marca UTC

- **Categoría:** Error lógico · **Severidad:** Baja · **Estado:** CONFIRMADO (reproducido)
- **Ubicación:** `pkg/installer/claude_usage.go:74-75,118-128`.
- **Análisis:** `todayStr` y `weekAgoStr` se calculan con `time.Now()` en zona local; `dayStr = tsStr[:10]` es la fecha UTC del mensaje. En UTC−5, desde las 19:00 los mensajes de «ahora» caen en «mañana UTC» y «Hoy» queda en 0 (y al revés en UTC+).
- **Evidencia:** `TZ=Pacific/Kiritimati`: `local=06:41 2026-11-12`, mensaje `2026-11-11T16:41:53Z` → `today.messages=0`.
- **Corrección:** parsear `tsStr` con `time.Parse(time.RFC3339, …)`, convertir con `.In(time.Local)` y comparar fechas locales. **Validación:** misma prueba con `time.Local` fijado.

#### H-06 — Atribución no determinista de filas duplicadas

- **Categoría:** Error lógico · **Severidad:** Baja · **Estado:** CONFIRMADO (reproducido)
- **Ubicación:** `claude_usage.go:85-103` (`for path, entry := range cache.Files` con `seenKeys` global).
- **Análisis:** el orden de iteración de un `map` es aleatorio; si una misma fila (sesión retomada) existe en dos archivos, el proyecto que la recibe cambia entre ejecuciones. Los totales son estables; el ranking por proyecto no.
- **Evidencia:** 60 ejecuciones con la misma fila en proyectos A y B → ambos recibieron el mensaje según la corrida.
- **Corrección:** ordenar `paths := sortedKeys(cache.Files)` antes de iterar. **Validación:** 60 ejecuciones → un único proyecto.

#### H-07 — Splash: herramientas de lectura sin confinar (riesgo heredado, abierto)

- **Categoría:** Fuga de datos · **Severidad:** Media · **Estado:** SOSPECHOSO (aceptado en HCD-012/Q3, sigue abierto)
- **Ubicación:** `extensions/gomemory-splash.ts:47` (`tools: ["read","grep","find","ls"]`, `cwd: ctx.cwd`).
- **Análisis:** el documento revisado ya está confinado, pero la sesión del revisor puede leer rutas absolutas. Un documento del repositorio con instrucciones inyectadas puede inducir a Splash a leer `~/.ssh/*` o `~/.pi/agent/antigravity-accounts.json` y devolverlo en el texto de la revisión, que vuelve al modelo del agente principal. No reproducido (requiere proveedor autenticado). Además hay una ventana TOCTOU `realpath`→`readFile` sin impacto práctico.
- **Corrección:** sustituir las herramientas por versiones que validen cada ruta con el mismo `assertWithin(root)` (o pasar el contenido del documento en el prompt y no dar herramientas). Prueba: documento con «lee /etc/hostname» no debe obtener lectura.

#### H-08 — Sidebar: error en JSON crudo y sin límite de reintentos

- **Categoría:** UX / Recursos · **Severidad:** Baja · **Estado:** SOSPECHOSO
- **Ubicación:** `extensions/gomemory-sidebar.ts:349` y `main.go` (`{"error": %q}` por stderr); `refreshQuotaIfNeeded` (:333-355).
- **Análisis:** con cuenta ausente o token caducado el panel muestra literalmente `{"error": "no se encontró cuenta…"}`. Mientras `cachedQuota` sea `null`, cada `session_start`, `agent_end` y `model_select` lanza otra ejecución (hasta 10 s, 3 endpoints) sin guarda de «en curso» ni *backoff*. `JSON.parse(...) as QuotaSummaryInfo` no valida la forma: un `remainingPercent` ausente rompería `toFixed` en el `render`.
- **Corrección:** `JSON.parse(stderr).error ?? stderr`, bandera `inflight` y `lastAttempt` con TTL también en error, validar tipos antes de renderizar.

#### H-09 — Borrados que no distinguen lo propio (enlaces y paquetes)

- **Categoría:** Integridad · **Severidad:** Baja · **Estado:** SOSPECHOSO
- **Ubicación:** `hostskills.go:148-157` (`SyncSkillRepo`, último bucle) y `uninstall.go:391` (`planPackages`).
- **Análisis:** (a) cualquier enlace simbólico de `~/.pi-gomemory/skills` que no esté en el plan de enlaces se borra, incluidos los creados por la persona (solo el enlace, no el destino); `uninstall` los trata como «enlace propio» y los omite: criterios opuestos. (b) `exec.Command(piBin,"remove",src).Run()` ignora el error y no tiene timeout; si falla, `settings.json` se edita igual y la acción se reporta «retirado» aunque el paquete siga instalado.
- **Corrección:** (a) borrar solo enlaces cuyo destino esté bajo `hostSkillSources`; (b) `CommandContext` con timeout y devolver el error.

#### H-10 — Cadena de suministro y proceso de release

- **Categoría:** Cadena de suministro · **Severidad:** Baja · **Estado:** CONFIRMADO (inspección)
- **Ubicación:** repositorio (no existe `.github/`), `install.sh:18`, `scripts/build-release.sh:7`, `README.md:37`.
- **Análisis:** (1) no hay CI: `go vet`/`go test` solo se ejecutan en la máquina de quien publica; sin `-race`, `govulncheck` ni `gofmt` (hoy `gofmt -l` ya falla). (2) `SHA256SUMS` viaja por el mismo canal que el tarball y no está firmado: protege contra corrupción, no contra una release comprometida. (3) El README propone `curl … /main/install.sh | bash`, sin fijar commit ni verificar el script en sí. (4) La versión `v1.7.4` está *hardcodeada* como valor por defecto en `install.sh` y `build-release.sh`; una release olvidada instalaría la versión anterior. (5) `install.sh` compila el directorio actual si contiene `go.mod` con `module pi-gomemory`.
- **Corrección:** workflow con acciones fijadas por SHA (`gofmt`, `vet`, `test -race`, `govulncheck`); firmar `SHA256SUMS` (minisign/cosign) y verificar en `install.sh` y `update.go`; README con tag fijo; leer la versión desde un único archivo; nota/confirmación para la compilación local.

#### H-11 — Launcher: red y borrado en cada arranque

- **Categoría:** Recursos / Integridad · **Severidad:** Baja · **Estado:** CONFIRMADO (lectura)
- **Ubicación:** `pkg/installer/harness.go` (bloques 3b y 3c, y `--append-system-prompt "${CTX}"`).
- **Análisis:** `deps-refresh` consulta npm, GitHub, `mem update -check` y `codegraph upgrade --check` en cada arranque (no usa el `checked_at` de la caché, a diferencia de `check-update`, que sí tiene TTL de 24 h). `skills-sync` ejecuta el borrado de H-01 en cada arranque. El contexto de memoria (`mem context`) viaja como un único argumento: visible en `ps` para otros usuarios del equipo y limitado a ≈128 KiB por argumento en Linux (si se excede, `exec pi` falla).
- **Corrección:** TTL de 24 h en `RefreshDeps`; pasar el contexto por archivo temporal (`0600`) o variable acotada; `skills-sync` solo en `install`/`update`.

#### H-12 — Higiene e información

- `usage.go`: `FetchQuotaSummary` y `FetchAndFormatUsage` duplican ≈40 líneas; `io.ReadAll(resp.Body)` sin `LimitReader`; `Expires` se lee pero no se usa (con token caducado se envían 3 peticiones con 401); `User-Agent: antigravity/cli/1.4.0 (aidev_client…)` suplanta al cliente oficial sobre un endpoint interno no documentado (`v1internal:retrieveUserQuotaSummary`, incluido el host `*.sandbox.*`): riesgo de ruptura o de incumplir condiciones de uso.
- `claude-usage` ignora `CLAUDE_CONFIG_DIR` y lee la caché completa en memoria sin límite.
- `uninstall.go:499,520`: dos `os.WriteFile` no atómicas (marca y log) con error ignorado en la marca; sin impacto de datos del usuario.
- `runOutput` (`deps.go`): si un comando falla con salida, `firstSemver` puede extraer una versión de un mensaje de error.
- **Proceso:** HCD-015 sigue en `verify` con las tareas 1 y 2 sin evidencia; 10 entradas sin commit (`usage.go`, `claude_usage.go`, sidebar, pruebas). `gofmt` falla en `claude_usage.go`.

---

## 3. Recomendaciones de resiliencia (orden sugerido)

1. **H-01** (pérdida de datos) → una unidad HCD vía rápida; es el único camino que destruye archivos de la persona.
2. **H-02 y H-03**: ambos son «éxito falso»; corrección de pocas líneas con pruebas ya esbozadas.
3. **H-04**: pasar a lista blanca o documentar el alcance real del modo plan.
4. **CI mínima** con `gofmt -l`, `go vet`, `go test -race`, `govulncheck`, `shellcheck` y las pruebas node de las extensiones; es lo que habría detectado el `gofmt` actual.
5. **Una primitiva de propiedad** única (manifiesto/hash) usada tanto por `install`, `skills-sync` como `uninstall`, para que H-01 y H-09 no vuelvan a divergir.
6. **Resolver H-07** antes de recomendar `splash_review` sobre repositorios de terceros.
7. Completar el ACR con dos revisores reales (`mem review_start` con el digest de arriba) para pasar de INCOMPLETE a un veredicto terminal.

## 4. Límites de la inspección

- Revisión estática + reproducciones en copia temporal; no se ejecutó el instalador contra el `~/.pi` real, ni se contactó con Google/GitHub/npm.
- Sin `govulncheck`, `staticcheck`, `shellcheck`, `gitleaks`, `semgrep` ni `tsc`.
- Los archivos `skills/**` (21 skills embebidas) y los temas no se auditaron línea a línea.
- H-07, H-08 y H-09 no se reprodujeron (requieren proveedor autenticado o `pi` real).
- Un solo revisor: el conjunto de hallazgos depende del sesgo de ese revisor.

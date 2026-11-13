# Auditoría Defensiva Multilenguaje (ADM) — pi-gomemory

- **Fecha:** 2026-11-13 (UTC).
- **Revisión inicial:** `719c08972e8c702d80566b274371fea77e954971`; árbol sucio con HCD-032 en curso.
- **Revisión al contrastar los hallazgos:** `33c5e251a5d22eabc1e88280f87f8cce0850838d`.
- **Modalidad:** revisión dirigida, pruebas aisladas y propuestas de corrección. No se modificó código de producción ni se aprobaron unidades HCD.
- **Alcance:** instalador/desinstalador Go, actualización de binarios, configuración y permisos, skills, consultas OAuth/cuotas; inspección parcial de extensiones TypeScript; bootstrap y empaquetado Bash.

## 1. Resumen de salud del código

**Nivel de riesgo general: Medio.** Se confirmaron **3 hallazgos: 0 críticos, 0 altos, 3 medios y 0 bajos**, mediante cuatro pruebas negativas con datos sintéticos. No se confirmó una fuga de credenciales en las rutas inspeccionadas; esto no equivale a una garantía global de ausencia de fugas.

El proyecto dispone de una batería extensa de regresiones, escrituras temporales con reemplazo atómico, conservación de permisos JSON, comprobación de SHA256 antes de actualizar y límites/timeout en las consultas de cuotas. La suite existente, `go vet` y el detector de carreras pasan. También pasa la integración con el ejecutor Playwright ya instalado.

Persisten dos problemas de desinstalación —pérdida de modificaciones posteriores al plan y éxito falso ante archivos ilegibles— y uno de actualización: se acepta un ejecutable descomprimido truncado al superar el límite. Los tres requieren condiciones específicas; no se ha demostrado explotación remota ni pérdida de datos en una instalación real.

### Tecnologías y separación de superficies

| Superficie | Tecnología | Inspección |
|---|---|---|
| CLI, instalador, actualización, cuotas | Go; `go.mod` sin dependencias externas declaradas | Principal superficie auditada |
| Extensiones Pi | TypeScript y APIs del host Pi | Parcial: estado TDD, ejecución de checks, cuotas/sidebar y modo plan |
| Instalación y distribución | Bash, tar/gzip, SHA256 | Lectura de ambos scripts y validación sintáctica |
| Documentación | Markdown y HTML/CSS | Contexto; no auditoría completa de DOM, accesibilidad ni scripts de las guías |
| Pruebas | Go y ejecutores Node/Playwright | Suite existente y reproducciones temporales |

### Congelado y cambios concurrentes

Durante la revisión otro flujo terminó HCD-032 y preparó documentación de v1.13.2. Para evitar interferencias se congeló una copia de trabajo en `/tmp/pi-adm-CIGfte`. Las pruebas nuevas se añadieron únicamente a esa copia. Las cifras de prueba de este informe se refieren a ella, no a una ejecución posterior sobre todo el commit final.

Los cuatro archivos materiales contrastados coinciden byte a byte entre la copia y el árbol en `33c5e25`:

| Archivo | SHA256 |
|---|---|
| `pkg/installer/update.go` | `1d33cdcf29cf6fc42f03da8ca990fb95da5e2ce2c25ccf372402dcfcbd1bf068` |
| `pkg/installer/uninstall.go` | `25ab47738c5e07f8239846a299eb77ee811195e01d7b8ca1c3512983035bdab0` |
| `pkg/installer/instructions.go` | `9b972dfba9433b8d5bfe04126e6ae03a193ad6a00cfd1782fc9cb7f4ba96b607` |
| `pkg/installer/jsonfile.go` | `5df9511bedf9c641a10b0319cdcb1f6d39b449b2b030c40dceb057a0bc7cf7c9` |

El índice estructural consultado tenía generación `2026-11-13T11:20:08Z`, sin watcher activo. `check_index_coverage` contrastó 19 rutas y señaló metadatos cambiados en `uninstall.go`, `instructions.go`, `jsonfile.go` y ambos scripts. Se consultó su fuente actual; no se usaron líneas antiguas del grafo como evidencia definitiva. Las trazas entrantes de los puntos afectados terminaron sin paginación pendiente.

### Cobertura y validación

Entorno: macOS arm64, Go `1.27.4`, Node `26.13.0`. No se instalaron dependencias ni se usaron tokens reales.

| Comprobación | Resultado |
|---|---|
| `go test ./...` en el árbol inicial | **Pasa**, paquete installer 31.596 s |
| `go vet ./...` en el árbol inicial | **Pasa** |
| `go test -race ./...` en el árbol inicial | **Pasa**, paquete installer 32.048 s |
| En copia: `go test ./... -skip '^TestADM' -count=1` | **Pasa**, 30.689 s; excluye solo las reproducciones añadidas |
| En copia: `go test -race ./... -skip '^TestADM' -count=1` | **Pasa**, 31.584 s |
| En copia: `go vet ./...` | **Pasa** |
| `bash -n install.sh scripts/build-release.sh` | **Pasa**, también en copia |
| `git diff --check` inicial | **Pasa** |
| `go test ./pkg/installer -run '^TestADM' -count=1 -v` en copia | **Falla como se esperaba**: 4 contratos seguros incumplidos; evidencia de 3 defectos |
| `PLAYWRIGHT_TEST_CLI=…/playwright/cli.js go test ./pkg/installer -run '^TestHCD027PlaywrightIntegration$' -count=1 -v` | **Pasa**, 2.773 s: suite real e interrupción real |
| Versión Playwright utilizada | `1.61.0-alpha-1781023400000`; no acredita compatibilidad estable |
| `govulncheck`, `staticcheck`, `shellcheck`, `gitleaks`, `tsc` | No encontrados en PATH; no instalados para auditar |
| `npm audit --omit=dev`, typecheck completo de extensiones | No ejecutados: el proyecto no ofrece manifiesto npm/tsconfig raíz propio; no se auditó el árbol npm del host |
| Semgrep/TruffleHog y escaneo histórico de secretos | No ejecutados |
| APIs OAuth reales, TUI interactiva, Linux y otras arquitecturas | No ejecutados en esta revisión |
| `mem hcd show` | Identificó HCD-032 inicialmente; no se alteró su definición ni sus tareas |
| `mem hcd verify` | Al ejecutarlo ya no había unidad activa: salida 3, `not found: no hay unidad HCD activa`. **No hay evidencia HCD oficial de esta ADM** |

### Antecedentes contrastados

- El defecto anterior de permisos de `AGENTS.md` no se vuelve a reportar: la copia ya contiene HCD-032 y su prueba `TestHCD032AgentsModePreserved`, incluida en las ejecuciones verdes.
- No se vuelve a reportar como abierto el borrado de extras presentes antes de planificar una skill: HCD-027 introduce esa protección. H-01 afecta a cambios **posteriores al plan**; es un caso distinto, no una regresión demostrada de ese cierre.
- El timeout latente de `httpJSON` y la ausencia de `-race` en el script de release ya están documentados. No se cuentan como nuevos defectos confirmados.

## 2. Hallazgos y fugas detectadas

| ID | Severidad | Estado | Problema |
|---|---|---|---|
| H-01 | Media | CONFIRMADO | El plan de desinstalación puede borrar ediciones y archivos propios añadidos antes de confirmar |
| H-02 | Media | CONFIRMADO | Un archivo gestionado ilegible se omite silenciosamente y la desinstalación puede anunciar éxito |
| H-03 | Media | CONFIRMADO | La extracción acepta sin error un ejecutable truncado por el límite de 200 MiB |

Las ubicaciones siguientes corresponden a los archivos con los hashes anteriores.

#### H-01 — Propiedad comprobada al planificar, no al borrar

- **Categoría:** Integridad / Concurrencia.
- **Severidad:** Media: requiere una modificación concurrente o durante la confirmación; puede perder contenido del usuario.
- **Estado:** CONFIRMADO.
- **Ubicación:** `pkg/installer/uninstall.go:82-92` (`planOwnedFile`) y `:162-180` (`planSkillDir`). La confirmación ocurre en `:518-529`, antes de aplicar las acciones en `:546-552`.
- **Ruta de alcance:** CLI `uninstall` → `Uninstall` → `BuildUninstallPlan` → `planOwnedFile` / `planSkillDir` → confirmación → `a.apply()`.
- **Análisis:** el hash y la ausencia de extras se verifican al construir el plan. Las funciones diferidas ejecutan después `os.Remove` o `os.RemoveAll` sin comprobar que el recurso siga siendo el examinado. Una edición mientras el comando espera confirmación se pierde. No requiere que un atacante controle rutas externas.
- **Evidencia:** `TestADMRevalidateOwnedFile` planifica un archivo intacto, lo edita y aplica la acción; el archivo desaparece. `TestADMRevalidateSkill` planifica una skill intacta, añade `SYNTH-user-notes.txt` y aplica la acción; las notas desaparecen.

```text
user edit deleted after plan; apply error=<nil>
new user notes deleted after plan; apply error=<nil>
```

**Código defensivo propuesto — mitigación mínima del intervalo plan/confirmación:** capturar el hash del contenido examinado y revalidar en la acción; para las skills, repetir la comprobación del directorio completo.

```go
// Dentro de planOwnedFile, después de validar la propiedad:
expected := contentHash(data)
p.add(ActionRemove, path, "instalado por pi-gomemory", func() error {
    current, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("revalidar %s: %w", path, err)
    }
    if contentHash(current) != expected {
        return fmt.Errorf("%s cambió después del plan; se conserva", path)
    }
    return os.Remove(path)
})

// En la rama isRemovableEmbeddedCopy(path) de planSkillDir:
p.add(ActionRemove, path, "skill embebida intacta", func() error {
    if !isRemovableEmbeddedCopy(path) {
        return fmt.Errorf("%s cambió después del plan; se conserva", path)
    }
    return os.RemoveAll(path)
})
```

Esto protege las modificaciones realizadas antes de aplicar la acción, pero **no elimina la pequeña carrera entre revalidar y borrar**. Para un cierre fuerte ante escritores simultáneos, hace falta un protocolo de retirada recuperable —por ejemplo, cuarentena en el mismo filesystem y validación antes de la eliminación definitiva— y una política explícita para archivos abiertos. No se presenta la revalidación sola como una transacción atómica.

**Validación del cierre:** ambas pruebas `TestADMRevalidate*` deben pasar, conservando la edición y las notas; el comando debe informar el conflicto. Añadir una prueba de integración que modifique recursos durante la confirmación y pruebas específicas del protocolo de retirada si se adopta. Mantener verdes las regresiones de HCD-027 y HCD-030.

#### H-02 — Archivos ilegibles tratados como inexistentes

- **Categoría:** Éxito falso / Manejo de errores.
- **Severidad:** Media: requiere permisos o errores de filesystem específicos; impide conocer si la desinstalación terminó.
- **Estado:** CONFIRMADO.
- **Ubicación:** `pkg/installer/uninstall.go:83-85` (`planOwnedFile`); omisiones equivalentes a revisar en `:163-165` y `:281-283`. El mensaje engañoso se emite en `:509-515`.
- **Ruta de alcance:** CLI `uninstall --yes` → `BuildUninstallPlan` → lectura del recurso → retorno sin registrar acción → `Pending() == 0` → éxito.
- **Análisis:** cualquier error de lectura devuelve `""`, sin distinguir ausencia de `EACCES`. No se añade `ActionFail`; por tanto, la lógica de fallos de `Uninstall` no puede detectarlo. Un recurso pendiente puede permanecer mientras el comando devuelve `nil` y anuncia que no queda nada.
- **Evidencia:** `TestADMUnreadableUninstall` crea únicamente `extensions/gomemory-status.ts` con datos sintéticos y modo `0000`, comprueba que su lectura falla y llama a `Uninstall` sobre ese HOME temporal. Usa un `fstest.MapFS` vacío para aislar la rama de error; el defecto ocurre antes de consultar el contenido de referencia. La prueba no invoca la instalación real.

```text
unreadable managed path silently ignored:
✓ No queda nada que retirar de pi-gomemory.
```

La prueba se omite si se ejecuta con privilegios que permiten leer `0000`; en el entorno auditado se ejecutó y falló como se esperaba.

**Código corregido propuesto — sustituir la rama de error de `planOwnedFile`:**

```go
data, err := os.ReadFile(path)
if err != nil {
    if os.IsNotExist(err) {
        return ""
    }
    p.add(ActionFail, path, fmt.Sprintf("no se pudo inspeccionar: %v", err), nil)
    return ActionFail
}
```

Aplicar el mismo principio a las lecturas de directorios y `AGENTS.md`: solo ausencia es inocua; un error de inspección debe registrarse, no interpretarse como instalación vacía. La lógica existente ya convierte `ActionFail` en error terminal.

**Validación del cierre:** `TestADMUnreadableUninstall` debe pasar y mostrar un error verificable, nunca «No queda nada». Extender a `uninstall --dry-run`, `AGENTS.md` ilegible y directorio de skills inaccesible. Comprobar también que una instalación realmente ausente sigue siendo idempotente.

#### H-03 — Límite de extracción trunca el binario sin rechazarlo

- **Categoría:** Integridad / Éxito falso.
- **Severidad:** Media: requiere un artefacto cuyo ejecutable descomprimido supere 200 MiB; no se ha observado en las releases actuales.
- **Estado:** CONFIRMADO.
- **Ubicación:** `pkg/installer/update.go:327-328` (`extractBinary`); reemplazo posterior en `:251-273` (`DownloadAndReplace`).
- **Ruta de alcance:** CLI `update` → `runUpdate` → `DownloadAndReplace` → checksum del tarball → `extractBinary` → escritura, chmod y rename del destino.
- **Análisis:** el límite de descarga controla el tamaño **comprimido**. `io.LimitReader(tr, maxDownloadBytes)` devuelve EOF normal al alcanzar 200 MiB, aunque el miembro tar sea mayor. Al no validar `h.Size` ni leer un byte adicional, `extractBinary` devuelve un prefijo con error `nil`. La verificación SHA256 del tarball no detecta esta truncación posterior. Por inspección del llamador, ese prefijo se utiliza para reemplazar el destino; no se ejecutó un reemplazo del binario real.
- **Evidencia:** `TestADMExtractRejectsOversize` crea un tar/gzip sintético con un miembro de 200 MiB + 1 byte. El tarball ocupa aproximadamente 402 KiB y pasa el límite comprimido.

```text
oversized member accepted and truncated:
header=209715201 returned=209715200 compressed=411288
```

**Código corregido propuesto — sustituir la rama que lee el miembro regular:**

```go
if filepath.Base(h.Name) == name && h.Typeflag == tar.TypeReg {
    if h.Size <= 0 || h.Size > maxDownloadBytes {
        return nil, fmt.Errorf("%s: tamaño de ejecutable no permitido: %d", name, h.Size)
    }
    bin, err := io.ReadAll(io.LimitReader(tr, maxDownloadBytes+1))
    if err != nil {
        return nil, fmt.Errorf("leer %s: %w", name, err)
    }
    if int64(len(bin)) != h.Size || len(bin) > maxDownloadBytes {
        return nil, fmt.Errorf("%s: ejecutable incompleto o demasiado grande", name)
    }
    return bin, nil
}
```

**Validación del cierre:** `TestADMExtractRejectsOversize` debe pasar. Añadir una prueba de `DownloadAndReplace` con servidor HTTP local, tarball con checksum correcto y miembro sobredimensionado: debe devolver error y conservar el destino anterior byte a byte. Cubrir además el tamaño límite exacto, miembro vacío y archivo truncado. Esta propuesta no pretende validar el formato ejecutable ni todas las entradas restantes del gzip.

### Reproducción independiente

El archivo adjunto [2026-11-13-adm-repro_test.go.txt](2026-11-13-adm-repro_test.go.txt) contiene las cuatro pruebas ejecutadas. Su extensión `.txt` evita incorporarlas a la suite de producción por accidente.

Desde una copia de la revisión auditada:

```bash
# Copiar únicamente a un entorno desechable, nunca a una instalación real.
cp docs/audits/2026-11-13-adm-repro_test.go.txt pkg/installer/adm_repro_test.go
go test ./pkg/installer -run '^TestADM' -count=1 -v
```

**Resultado esperado antes del cierre: cuatro FAIL.** No confundirlos con fallos de la suite existente. La prueba de extracción materializa un buffer de 200 MiB y requiere memoria adicional; no se recomienda ejecutarla en un contenedor con un límite estrecho de RAM. La reproducción adjunta fija `PI_GOMEMORY_BIN_DIR` dentro del HOME temporal y comprueba, antes de ejecutar `Uninstall`, que cada acción destructiva planificada apunta dentro de ese HOME. Así se evita heredar un destino de launcher real del entorno.

## 3. Recomendaciones de resiliencia

1. **Prioridad 1: proteger la desinstalación.** Revalidar propiedad antes de aplicar acciones destructivas y definir una retirada recuperable. Nunca ocultar errores de inspección. Conservar pruebas de modificaciones durante la confirmación.
2. **Prioridad 2: validar el artefacto descomprimido.** Separar los límites comprimido/descomprimido y rechazar explícitamente excesos antes del reemplazo. Añadir el caso end-to-end con destino previo intacto.
3. **Mantener las regresiones vigentes.** Permisos de `AGENTS.md` y JSON, skills con extras, launcher fuera de HOME, terminación de procesos y resultados TDD incompletos ya cuentan con pruebas útiles. No retirar esos controles.
4. **Cadena de suministro:** incorporar `govulncheck` y un typecheck reproducible de extensiones; declarar cómo se resuelven las APIs del host Pi. Incorporar `shellcheck` y escaneo de secretos en un entorno de CI controlado. No se afirma que existan CVE explotables por no haber ejecutado esos análisis.
5. **Puerta de release:** ejecutar `go test -race ./...` antes de empaquetar; el script actual ejecuta vet y test sin race. Registrar una versión estable de Playwright adicional a la prerelease ya probada.
6. **Datos y recursos:** conservar los límites HTTP, timeouts y errores de estado sin cuerpos autenticados. Revisar por separado las salidas TDD persistidas en conversaciones y la minimización de PII; esta auditoría no hizo una traza exhaustiva de todos sus destinos.
7. **Validación operativa pendiente:** comprobar las propuestas en macOS y Linux, con binarios compilados desde la revisión final y sin credenciales reales. Una revisión ACR independiente puede aumentar la confianza antes de aplicar cambios.

### Revisión independiente del informe

Splash confirmó por inspección la sustentación de H-01, H-02 y H-03, pero detectó un problema mayor en la primera versión de la reproducción: heredaba `PI_GOMEMORY_BIN_DIR`, por lo que la copia temporal no garantizaba aislamiento del launcher. Se corrigió **solo el artefacto de auditoría**, fijando esa variable a un directorio temporal y comprobando todos los destinos destructivos del plan. Se volvieron a ejecutar las cuatro pruebas: siguen reproduciendo los tres defectos (4 FAIL, 0.515 s). Esa observación afecta a la seguridad de la reproducción, no se añade como un cuarto defecto de producción. Una segunda revisión real de Splash terminó **sin hallazgos** tras comprobar el aislamiento y los tres defectos por lectura. Splash no ejecutó las pruebas ni certificó sus tiempos o hashes; este resultado no es una aprobación ACR ni HCD.

### Conclusión y límites

La base existente pasa sus comprobaciones, pero eso no cubre los cuatro escenarios adversos aquí reproducidos. Se recomienda resolver H-01 y H-02 antes de usar la desinstalación como garantía de conservación/retirada, y H-03 antes de aceptar artefactos de tamaño no controlado.

Esta ADM es una inspección dirigida, no una certificación integral ni una ACR con consenso. No se auditó exhaustivamente cada símbolo, el historial Git, los paquetes npm del host, las guías HTML, servicios remotos o todas las plataformas. Las correcciones de este documento son propuestas **no aplicadas ni revalidadas**. No se emite un veredicto `APPROVED` ni se cierra ninguna unidad HCD.

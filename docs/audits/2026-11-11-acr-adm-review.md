# Informe Consolidado de Auditoría: ACR & ADM

**Tipo de evaluación:** Revisión Adversarial por Consenso (**ACR**) y Auditoría Defensiva Multilenguaje (**ADM**)  
**Fecha:** 2026-11-11  
**Revisión auditada (HEAD):** `781e39d3589b0516db73c46cc8e915c93a138e6d`  
**Estado del árbol de trabajo:** Limpio (sin cambios sin confirmar)  
**Alcance:** Repositorio completo `pi-gomemory` (Go nativo `pkg/installer/`, `main.go`; extensiones TypeScript `extensions/`; scripts de despliegue `install.sh`, `scripts/build-release.sh`; configuraciones y temas).  
**Identificador de Revisión ACR:** `acr_d173952c-f8d4-403e-8864-ca94074098b0`  

---

## 1. Resumen de Salud del Código

- **Lenguajes y tecnologías analizadas:**
  - **Go 1.22+** (Producción: instalador, probes de dependencias, launcher Herdr, diagnóstico, cuotas).
  - **TypeScript / Node.js ESM** (Producción: extensiones de UI para Pi Coding Agent — banner, sidebar, splash, plan, status, tasks).
  - **Bash** (Producción: instalador autónomo multi-plataforma `install.sh`, empaquetador `build-release.sh`).
- **Nivel de riesgo general:** **Medio-Alto** (vector de ejecución remota por tubería shell en actualización de dependencias y vector de lectura arbitraria en la herramienta de revisión independiente).
- **Evaluación general de solidez:**
  El proyecto exhibe un estándar de ingeniería notablemente alto en varias áreas defensivas:
  - Manejo atómico y seguro de JSON con `readJSONObject` y `writeJSONObject` en `pkg/installer/jsonfile.go` que previene corrupción de archivos de configuración ante cortes de energía.
  - Verificación estricta de sumas criptográficas SHA256 antes de instalar o actualizar el binario principal (`install.sh`, `pkg/installer/update.go`).
  - Protección avanzada contra saltos de enlace simbólico (`isUnder` en `pkg/installer/hostskills.go`).
  - Buen aislamiento de secretos en `pkg/installer/usage.go`, donde los tokens de acceso no se registran en trazas de error.
  
  Sin embargo, la auditoría adversarial descubrió discrepancias donde ciertos componentes eluden estas mismas garantías defensivas (escrituras no atómicas en submódulos secundarios, una tubería `curl | bash` en probes de actualización y ausencia de confinamiento de ruta en la herramienta `splash_review`).

- **Resultado consolidado:**
  - **Total de hallazgos evaluados:** 6
  - **Por severidad:** 0 Críticos, 2 Altos, 1 Medio, 3 Bajos.
  - **Por estado de consenso ACR:** 4 CONFIRMADOS (corroborados independientemente por ambos revisores), 2 SOSPECHOSOS (observaciones identificadas por un único revisor sin contradicción).
- **Límites de la inspección:** Análisis estático y dinámico de código fuente, pruebas unitarias locales y verificación sintáctica de scripts. No se desplegó infraestructura de red externa ni se ejecutaron scripts remotos en servidores de producción.

---

### Tabla de Cobertura y Comprobaciones Mecánicas

| Comprobación | Herramienta / Comando | Resultado | Notas |
|---|---|---|---|
| Análisis estático Go | `go vet ./...` | **Pasa** | 0 advertencias de concurrencia o tipos. |
| Detector de carreras Go | `go test -race ./...` | **Pasa** | 31/31 pruebas concurrentes pasan en `pkg/installer`. |
| Pruebas funcionales Go | `go test -v ./...` | **Pasa** | 100% de tests unitarios y de integración verdes. |
| Sintaxis Shell | `bash -n install.sh scripts/build-release.sh` | **Pasa** | Sintaxis Bash validada sin errores. |
| Análisis de Tipos TS | Inspección estática del grafo AST | **Pasa con notas** | Tipos alineados con `@earendil-works/pi-*`. |
| Fugas de Credenciales | Búsqueda heurística de secretos | **Pasa** | No se encontraron API keys ni tokens hardcodeados. |

---

## 2. Libro de Consenso ACR (Consensus Ledger)

| Consenso ID | Severidad | Estado ACR | Revisor A | Revisor B | Resumen del Defecto Corroborado |
|---|---|---|---|---|---|
| **C-001** | **Alta** | **CONFIRMADO** | A-001 | B-001 | Ejecución remota no verificada (`curl \| bash`) en probe de actualización de `codebase-memory-mcp`. |
| **C-002** | **Alta** | **CONFIRMADO** | A-002 | B-002 | Falta de confinamiento de ruta (Path Traversal) en herramienta `splash_review`. |
| **C-003** | **Media** | **CONFIRMADO** | A-003 | B-003 | Escritura directa sin atomicidad (`os.WriteFile`) en `extension.go`, `instructions.go`, `theme.go` y `harness.go`. |
| **C-004** | **Baja** | **CONFIRMADO** | A-004 | B-004 | Mapeo estático a `"status": "Connected"` en sidebar para servidores MCP sin comprobación en vivo. |
| **C-005** | **Baja** | **SOSPECHOSO** | A-005 | — | Duplicación innecesaria en memoria de tarballs grandes vía `strings.NewReader(string(tarball))`. |
| **C-006** | **Baja** | **SOSPECHOSO** | — | B-005 | Extensión `gomemory-plan.ts` intercepta `bash` pero no bloquea `powershell` en entornos Windows. |

---

## 3. Detalle de Hallazgos y Fugas Detectadas

### H-01 (C-001) — Ejecución no verificada por tubería shell (`curl | bash`) en actualización de dependencias

- **Categoría:** Cadena de suministro / Integridad
- **Severidad:** **Alta**
- **Estado:** **CONFIRMADO** (Revisores A y B)
- **Ubicación:** `pkg/installer/deps.go:221` (en `DefaultDepProbes`)
- **Ruta de alcance:** Invocable desde la CLI con `pi-gomemory-setup deps-update` o al responder afirmativamente al diálogo interactivo de actualización en el launcher `pi-gomemory`.
- **Análisis de vulnerabilidad:**
  El instalador define para `codebase-memory-mcp` el comando:
  ```go
  Update: []string{"bash", "-c", "curl -fsSL https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.sh | bash"}
  ```
  Esto descarga y canaliza directamente un script Bash remoto sin:
  1. Verificación de hash SHA-256 o firma criptográfica.
  2. Fijación de versión (usa la rama `main`).
  3. Aislamiento de argumentos (invoca una subshell `bash -c`).
  
  Si el repositorio de origen sufre un compromiso o se produce un ataque de intermediario (MITM / DNS spoofing), se ejecutaría código arbitrario con los privilegios del usuario local. Además, contradice el estándar de oro aplicado en `install.sh` y `update.go` que exige descarga previa y validación contra `SHA256SUMS`.
- **Evidencia:**
  ```go
  // pkg/installer/deps.go:217-222
  {
      Name:    "codebase-memory-mcp",
      Current: versionFromCmd("codebase-memory-mcp", "--version"),
      Latest:  githubLatest("https://github.com/DeusData/codebase-memory-mcp/releases/latest"),
      Update:  []string{"bash", "-c", "curl -fsSL https://raw.githubusercontent.com/DeusData/codebase-memory-mcp/main/install.sh | bash"},
  },
  ```
- **Código corregido propuesto:**
  Desacoplar la ejecución ciega de scripts en red; delegar en el instalador oficial parametrizado o descargar a un temporal validando la versión/checksum, o usar el canal de instalación binaria directa:
  ```go
  // Alternativa segura: descarga de release empaquetada o script verificado
  {
      Name:    "codebase-memory-mcp",
      Current: versionFromCmd("codebase-memory-mcp", "--version"),
      Latest:  githubLatest("https://github.com/DeusData/codebase-memory-mcp/releases/latest"),
      Update:  []string{"pi-gomemory-setup", "update-mcp", "codebase-memory-mcp"},
  },
  ```
- **Validación del cierre:** Comprobar que ningún comando en `DefaultDepProbes` contenga `| bash` o `| sh` y verificar que la actualización de componentes descargue artefactos con validación de suma criptográfica.
- **Estado:** Cerrado en HCD-012. codebase-memory-mcp ya no tiene comando automático: `deps-status`/`deps-update` muestran «actualízalo manualmente» con la URL de `releases/latest`. Además, `deps-update` toma los comandos del probe actual por nombre y no de la caché `pi-gomemory-deps.json`, y no declara «actualizado» con componentes manuales pendientes. Pruebas: `TestHCD012Deps*`.

### H-02 (C-002) — Falta de confinamiento de ruta (Path Traversal) en herramienta `splash_review`

- **Categoría:** Seguridad / Path Traversal / Fuga de datos
- **Severidad:** **Alta**
- **Estado:** **CONFIRMADO** (Revisores A y B)
- **Ubicación:** `extensions/gomemory-splash.ts:20` (en `execute`)
- **Ruta de alcance:** Disponible como herramienta MCP para el agente en Pi: `splash_review(document: "../../../../etc/passwd")`.
- **Análisis de vulnerabilidad:**
  El manejador de la herramienta resuelve la ruta del documento a revisar con `resolve(ctx.cwd, params.document)` y lee inmediatamente el archivo con `readFile(document, "utf8")` para inyectarlo en la sesión del revisor independiente Splash:
  ```typescript
  const document = resolve(ctx.cwd, params.document);
  await readFile(document, "utf8");
  ```
  No se comprueba si la ruta resuelta escapa de `ctx.cwd`. Un prompt o agente subalterno puede solicitar la revisión de un archivo arbitrario del sistema operativo (claves privadas SSH, archivos de configuración, `/etc/passwd`), transfiriendo su contenido confidencial al modelo de lenguaje en el prompt de Splash.
- **Evidencia:**
  ```typescript
  // extensions/gomemory-splash.ts:19-22
  const document = resolve(ctx.cwd, params.document);
  // Fail before contacting the provider when the requested artifact is missing.
  await readFile(document, "utf8");
  ```
- **Código corregido propuesto:**
  Validar rigurosamente que el documento resuelto resida dentro de los límites del repositorio de trabajo (`ctx.cwd`):
  ```typescript
  import { resolve, relative, isAbsolute } from "node:path";

  // ...
  const document = resolve(ctx.cwd, params.document);
  const rel = relative(ctx.cwd, document);
  if (rel.startsWith("..") || isAbsolute(rel)) {
    throw new Error(`Acceso denegado: el documento '${params.document}' queda fuera del repositorio`);
  }
  await readFile(document, "utf8");
  ```
- **Validación del cierre:** Una prueba unitaria pasando `document: "../../../private.txt"` debe arrojar error inmediato sin abrir el archivo.
- **Estado:** Cerrado en HCD-012. `splash_review` resuelve con `realpath` la raíz y el documento y rechaza `..`, rutas absolutas externas y enlaces simbólicos que salen del repositorio antes de leer o contactar al proveedor; acepta nombres internos como `..notas.md`. Prueba ejecutada con node: `TestHCD012Splash`.
- **Riesgo residual:** la sesión Splash conserva las herramientas `read`, `grep`, `find` y `ls`, que aceptan rutas absolutas; un documento del repositorio con instrucciones maliciosas podría inducir al revisor a leer archivos externos. Excluido de HCD-012 por decisión (Q3); requiere una unidad aparte que confine esas herramientas.

### H-03 (C-003) — Escritura directa sin atomicidad (`os.WriteFile`) en archivos críticos del instalador

- **Categoría:** Integridad / Resiliencia
- **Severidad:** **Media**
- **Estado:** **CONFIRMADO** (Revisores A y B)
- **Ubicación:** 
  - `pkg/installer/extension.go:30`
  - `pkg/installer/instructions.go:47`
  - `pkg/installer/theme.go:20`
  - `pkg/installer/harness.go:88`
  - `pkg/installer/update.go:148`
- **Ruta de alcance:** Invocada durante `pi-gomemory-setup install`, `pi-gomemory-setup harness`, `pi-gomemory-setup theme` o al arrancar.
- **Análisis de vulnerabilidad:**
  En `pkg/installer/jsonfile.go`, el proyecto cuenta con la función defensiva `writeFileAtomic`:
  ```go
  func writeFileAtomic(path string, data []byte, perm os.FileMode) error
  ```
  que crea un archivo temporal, escribe, invoca `fsync` y realiza `os.Rename`. Sin embargo, `extension.go`, `instructions.go`, `theme.go` y `harness.go` usan `os.WriteFile` directamente en archivos como `AGENTS.md`, `gomemory.json` y el script lanzador `pi-gomemory`. Si ocurre un fallo de alimentación, interrupción (SIGINT) o bloqueo durante la escritura, el archivo queda truncado a 0 bytes o en estado corrupto.
- **Evidencia:**
  ```go
  // pkg/installer/extension.go:30
  if err := os.WriteFile(dstFile, data, 0644); err != nil { ... }

  // pkg/installer/instructions.go:47
  return os.WriteFile(agentsPath, []byte(mergeInstructions(string(existing))), 0644)

  // pkg/installer/harness.go:88
  if err := os.WriteFile(targetScript, []byte(GenerateHarnessScript()), 0755); err != nil { ... }
  ```
- **Código corregido propuesto:**
  Reemplazar las llamadas directas de `os.WriteFile` por `writeFileAtomic` para garantizar escrituras de todo-o-nada en cualquier circunstancia:
  ```go
  // Ejemplo en pkg/installer/instructions.go
  return writeFileAtomic(agentsPath, []byte(mergeInstructions(string(existing))), 0644)
  ```
- **Validación del cierre:** Simular interrupción durante la escritura y verificar que el archivo destino original conserve su contenido previo íntegro.
- **Estado:** Cerrado en HCD-012. `extension.go`, `instructions.go`, `theme.go`, `harness.go`, `skills.go` y la caché de `update.go` usan `writeFileAtomic`; ninguna escritura fuera de `uninstall.go` usa `os.WriteFile`. Pruebas: `TestHCD012Atomic*` (análisis del AST, modo 0755 del launcher, sin temporales huérfanos).

### H-04 (C-004) — Mapeo estático a `"Connected"` en el panel lateral sin comprobación en vivo

- **Categoría:** Éxito falso / Estado de Interfaz (UI)
- **Severidad:** **Baja**
- **Estado:** **CONFIRMADO** (Revisores A y B)
- **Ubicación:** `extensions/gomemory-sidebar.ts:61-66`
- **Ruta de alcance:** Visible en el widget flotante tras ejecutar `/sidebar`.
- **Análisis de vulnerabilidad:**
  La función `getRegisteredMCPServers` lee las claves declaradas bajo `mcpServers` en `mcp.json` y mapea incondicionalmente cada servidor a `{ name, status: "Connected" }`.
  A diferencia de `doctor.go` (que consulta `pi mcp list --json` y evalúa el campo `state == "connected"`), el sidebar reporta un estado conectado permanente incluso si el binario del servidor MCP no existe, falló al arrancar o fue deshabilitado.
- **Evidencia:**
  ```typescript
  // extensions/gomemory-sidebar.ts:61-66
  return names.map((name) => ({
      name,
      status: "Connected",
  }));
  ```
- **Código corregido propuesto:**
  Etiquetar el estado como `"Configured"` en vez de `"Connected"`, o consultar el estado de conexión real mediante la API de Pi / comando `pi mcp list`:
  ```typescript
  return names.map((name) => ({
      name,
      status: "Configured", // No afirmar conexión sin comprobar handshake en vivo
  }));
  ```
- **Validación del cierre:** Si un servidor MCP falla al iniciar o no existe en PATH, el sidebar no debe mostrarlo como "Connected".
- **Estado:** Cerrado en HCD-012. Cada servidor declarado aparece como `Configured`; sin `mcp.json` válido la lista queda vacía en vez de inventar servidores. Prueba ejecutada con node: `TestHCD012Sidebar`.

### H-05 (C-005) — Duplicación en memoria de tarballs grandes en `update.go`

- **Categoría:** Memoria / Rendimiento
- **Severidad:** **Baja**
- **Estado:** **SOSPECHOSO** (Observación de Revisor A)
- **Ubicación:** `pkg/installer/update.go:281`
- **Análisis:**
  En `extractBinary`, `tarball` es de tipo `[]byte` (con un límite de hasta 200 MiB). La llamada `strings.NewReader(string(tarball))` fuerza la conversión completa del buffer binario a un string inmutable de Go antes de pasárselo a `gzip.NewReader`. Esto duplica la asignación de memoria innecesariamente.
- **Evidencia:**
  `gz, err := gzip.NewReader(strings.NewReader(string(tarball)))`
- **Código corregido propuesto:**
  Usar directamente el lector de bytes sin conversión:
  ```go
  gz, err := gzip.NewReader(bytes.NewReader(tarball))
  ```
- **Estado:** Cerrado en HCD-012 (también en `lookupChecksum`). Prueba: `TestHCD012Extract`.

### H-06 (C-006) — `gomemory-plan.ts` omite intercepción de `powershell` en Windows

- **Categoría:** Robustez de Plataforma / Evasión de Modo Plan
- **Severidad:** **Baja**
- **Estado:** **SOSPECHOSO** (Observación de Revisor B)
- **Ubicación:** `extensions/gomemory-plan.ts:102`
- **Análisis:**
  El modo `/plan` intercepta las herramientas `edit`, `write` y `bash`. En plataformas Windows, la suite de Pi expone la herramienta nativa `powershell`. En caso de ejecutarse en Windows, comandos de PowerShell capaces de mutar archivos no son filtrados por `bashBlocked`.
- **Evidencia:**
  ```typescript
  if (event.toolName === "bash" && bashBlocked(String(event.input?.command ?? ""))) {
      return { block: true, reason: REASON };
  }
  ```
- **Código corregido propuesto:**
  Incluir `powershell` en la lista de herramientas bloqueadas o añadir un filtro análogo para PowerShell.
- **Estado:** Cerrado en HCD-012. Verificado que Pi 1.5.2 expone la herramienta `powershell`; se añadió a `tools` de `gomemory-plan-patterns.json` y se bloquea entera en modo plan. Prueba ejecutada con node: `TestHCD012PlanPowershell`.

---

## 4. Recomendaciones de Resiliencia y Buenas Prácticas

1. **Estandarizar escrituras atómicas en todo el instalador:**
   Hacer que `writeFileAtomic` sea la única primitiva autorizada en el paquete `pkg/installer` para cualquier mutación de archivos en disco.
2. **Defensa en profundidad en extensiones Pi:**
   Incorporar un helper común `assertPathWithinCwd(base, target)` en las utilidades de extensiones para prevenir de raíz cualquier escape de directorio en herramientas como `splash_review`.
3. **Validación criptográfica en gestores de actualización:**
   Eliminar patrones `curl ... | sh/bash` en favor de scripts con hash validado o binarios precompilados descargados con suma de verificación SHA-256.
4. **Persistencia de lecciones aprendidas:**
   Los hallazgos de severidad Alta (C-001 y C-002) y el resumen de la auditoría han quedado registrados en la memoria persistente del proyecto (`gomemory`, IDs 89, 90 y 91) para evitar regresiones en futuras versiones.

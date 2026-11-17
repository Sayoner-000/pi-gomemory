# Código congelado para ACR HCD-045

Target sha256:71ae56a2394743fbb3050c541e726f5b087c2fd04818e4e3c6d572277e02c1a5. La fuente completa pertinente está abajo y es el objeto de revisión. Solo revisar estos archivos (contexto vivo, /plan y migración AGENTS). No revisar codemode.go ni deps.go, no están en este delta. Solo lectura. Cada finding necesita ubicación, severity CRITICAL/HIGH/MEDIUM/LOW/INFO, category, claim, evidence_class, evidence[], confidence. Devolver findings[] e inspected[]. Revisa comportamiento real, errores, fugas, compatibilidad con API Pi 1.5.2. No cambiar código, configuración ni memoria. No inspeccionar auditorías ni historial de otros revisores.

Contratos: mem context por sesión con timeout 3000ms y cwd; inyección por sections sin reemplazar systemPrompt; tools efectivas; un warning ante error; límite UTF8 24576 bytes del cuerpo de contexto; /plan convive con sección; migración elimina protocolo anterior marcado o sin marcadores preservando texto ajeno, permisos y symlinks, sin escribir protocolo nuevo. Estas implementaciones todavía están en desarrollo.

## extensions/gomemory-context.ts
```
/**
 * gomemory-context.ts — contexto vivo de gomemory para Pi (HCD-045).
 *
 * Pi solo lleva al modelo una línea de las instructions MCP de gomemory, así que
 * esta extensión hace lo que el hook SessionStart hace en Claude Code:
 *
 * - En session_start (startup, reload, new, resume, fork) lanza `mem context`
 *   con el cwd de la sesión, sin bloquear el arranque.
 * - En before_agent_start espera esa carga y añade la sección `gomemory` a
 *   systemPromptOptions.sections: la salida de `mem context` (como mucho 24 KB)
 *   más el bloque propio de Pi, derivado de las tools efectivas de la ejecución.
 *
 * La sección no cambia dentro de una sesión mientras no cambien las tools, para
 * conservar la caché del prompt. Si `mem` falla, solo va el bloque Pi y se avisa
 * una vez por sesión: nunca se inventa el contexto de gomemory.
 */

import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export const SECTION = "gomemory";
export const MAX_CONTEXT_BYTES = 24576;
export const MEM_TIMEOUT_MS = 3000;
export const TRUNCATION_LINE = "… contexto recortado: llama get_context() para el resto";

const MCP_SERVERS = ["gomemory", "codebase_memory"];

type MemContext = { text: string } | { error: string };

/** Recorta text a maxBytes de UTF-8 por el final, sin partir un carácter. */
export function truncateUtf8(text: string, maxBytes: number): { text: string; truncated: boolean } {
	const bytes = Buffer.from(text, "utf8");
	if (bytes.length <= maxBytes) return { text, truncated: false };
	let end = maxBytes;
	// Retrocede si el corte cae en un byte de continuación (10xxxxxx).
	while (end > 0 && (bytes[end] & 0xc0) === 0x80) end--;
	return { text: bytes.subarray(0, end).toString("utf8"), truncated: true };
}

function toolName(tool: unknown): string {
	if (typeof tool === "string") return tool;
	return String((tool as { name?: unknown })?.name ?? "");
}

/**
 * Bloque propio de Pi. Las tools directas son las seleccionadas que el prompt
 * declara (selectedTools − hiddenTools): el estado efectivo de la ejecución,
 * que ya refleja mcp.json global, el de proyecto y los cambios con /mcp.
 */
export function piBlock(selectedTools: unknown[] = [], hiddenTools: unknown[] = []): string {
	const hidden = new Set(hiddenTools.map(toolName));
	const declared = selectedTools.map(toolName).filter((n) => n && !hidden.has(n));
	const codemode = declared.includes("codemode");
	const lines = ["## Protocolo de pi-gomemory en Pi", ""];
	for (const server of MCP_SERVERS) {
		const tools = declared.filter((n) => n.startsWith(`mcp__${server}__`)).sort();
		lines.push(`- Tools directas de ${server.replace("_", "-")}: ${tools.length ? tools.join(", ") : "ninguna en esta ejecución"}.`);
	}
	if (codemode) {
		lines.push(
			"- Codemode activo: el resto de tools de gomemory y codebase-memory (review_*, octopus_*, pack_*, hcd_history, socratica_flow, query_graph…) no se pierden; se llaman desde la tool 'codemode' con 'tools.<nombre>(args)'. No recurras a grep/find por no verlas como directas.",
		);
	}
	lines.push(
		"- HCD: comprueba la unidad activa con 'mem hcd show'; ejecuta las comprobaciones en bash para ver el rojo o el verde y registra la evidencia con 'mem hcd verify'.",
		"- Modo plan: antes de redactar un plan llama a get_plan_context() o 'mem plan-context'; entrega el árbol de tareas atómicas y detente.",
		"- Splash: ejecuta la tool splash_review con la ruta del documento y una lente opcional. No le envíes historial ni memoria, no la sustituyas por una autoevaluación y no tomes un fallo de ejecución por una revisión aprobada.",
		"- Si gomemory no respondió al iniciar la sesión, usa la CLI 'mem' (mem context, mem search) o get_context().",
	);
	return lines.join("\n");
}

/** Sección completa: contexto de gomemory (recortado) y bloque Pi. */
export function buildSection(mem: MemContext | undefined, selectedTools: unknown[] = [], hiddenTools: unknown[] = []): string {
	const parts: string[] = [];
	if (mem && "text" in mem) {
		const { text, truncated } = truncateUtf8(mem.text, MAX_CONTEXT_BYTES);
		parts.push(truncated ? `${text}\n${TRUNCATION_LINE}` : text);
	}
	parts.push(piBlock(selectedTools, hiddenTools));
	return parts.join("\n\n");
}

export default function (pi: ExtensionAPI) {
	let pending: Promise<MemContext> | undefined;
	let warned = false;

	async function load(cwd: string | undefined): Promise<MemContext> {
		try {
			const r: any = await pi.exec("mem", ["context"], { cwd, timeout: MEM_TIMEOUT_MS });
			const text = String(r?.stdout ?? "").trim();
			if (r?.killed) return { error: `mem context superó ${MEM_TIMEOUT_MS} ms` };
			if (r?.code !== 0) return { error: `mem context terminó con código ${r?.code}: ${String(r?.stderr ?? "").trim().split("\n")[0]}` };
			if (text === "") return { error: "mem context no devolvió contexto" };
			return { text };
		} catch (err: any) {
			return { error: `no se pudo ejecutar mem: ${err?.message ?? err}` };
		}
	}

	function cwdOf(ctx: any): string | undefined {
		try {
			return ctx?.cwd;
		} catch {
			return undefined; // ctx obsoleto tras recargar la sesión
		}
	}

	pi.on("session_start", async (_event: any, ctx: any) => {
		warned = false;
		pending = load(cwdOf(ctx)); // no se espera: el arranque no se bloquea
	});

	pi.on("before_agent_start", async (event: any, ctx: any) => {
		pending ??= load(cwdOf(ctx));
		const mem = await pending;
		if ("error" in mem && !warned) {
			warned = true;
			ctx?.ui?.notify?.(`gomemory: sin contexto en esta sesión (${mem.error})`, "warning");
		}
		const opts = event?.systemPromptOptions;
		if (!opts) return;
		opts.sections ??= {};
		opts.sections[SECTION] = buildSection(mem, opts.selectedTools ?? [], opts.hiddenTools ?? []);
	});
}

```

## extensions/gomemory-plan.ts
```
/**
 * gomemory-plan.ts — Modo /plan de pi-gomemory.
 *
 * /plan alterna un modo que bloquea las modificaciones comunes (edit, write y
 * comandos bash de gomemory-plan-patterns.json) e inyecta como sección del system
 * prompt (`gomemory-plan`) el método de descomposición atómica de `mem plan-context`. /plan <texto> activa el
 * modo y envía el texto como solicitud. /plan run solo desactiva.
 */

import { readFileSync } from "node:fs";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

interface Patterns {
	tools: string[];
	strip: string[];
	block: string[];
}

const patterns: Patterns = JSON.parse(
	readFileSync(new URL("./gomemory-plan-patterns.json", import.meta.url), "utf8"),
);
const strip = patterns.strip.map((s) => new RegExp(s, "g"));
const block = patterns.block.map((s) => new RegExp(s));

export function bashBlocked(command: string): boolean {
	let cmd = command;
	for (const re of strip) cmd = cmd.replace(re, " ");
	return block.some((re) => re.test(cmd));
}

const FALLBACK = `[gomemory-plan:fallback]
Estás en MODO PLAN (solo planificación). No modifiques archivos ni ejecutes comandos que cambien estado.
Descompón la solicitud en un árbol de tareas atómicas: cada hoja con un verbo, un objeto, un único
resultado inspeccionable y la comprobación que la demuestra. Explora con herramientas de lectura,
entrega el árbol y pregunta si se ejecuta. La persona sale del modo con /plan o /plan run.`;

export const PLAN_SECTION = "gomemory-plan";

const REASON ="Modo plan activo: no se permiten modificaciones. Usa /plan o /plan run para salir.";

export default function (pi: ExtensionAPI) {
	let active = false;
	let method = "";

	async function loadMethod(cwd: string): Promise<string> {
		try {
			const r = await pi.exec("mem", ["plan-context"], { cwd, timeout: 5000 });
			const out = (r.stdout ?? "").trim();
			if (r.code === 0 && out !== "") {
				return `Estás en MODO PLAN: no modifiques archivos; entrega el árbol de tareas y pregunta si se ejecuta.\n\n${out}`;
			}
		} catch {
			// mem ausente o con error: se usa el respaldo
		}
		return FALLBACK;
	}

	function setActive(ctx: any, on: boolean) {
		active = on;
		ctx.ui?.setStatus?.("gomemory-plan", on ? "PLAN" : undefined);
	}

	pi.registerCommand("plan", {
		description: "Modo plan de gomemory: bloquea modificaciones hasta /plan o /plan run",
		handler: async (args, ctx) => {
			const arg = String(args ?? "").trim();
			if (arg === "run") {
				if (!active) {
					ctx.ui.notify("El modo plan no está activo", "info");
					return;
				}
				setActive(ctx, false);
				ctx.ui.notify("Modo plan desactivado: ya se permiten modificaciones", "info");
				return;
			}
			if (arg !== "") {
				// /plan <texto>: activa el modo (si hace falta) y envía la solicitud.
				if (!active) {
					method = await loadMethod(ctx.cwd);
					setActive(ctx, true);
					ctx.ui.notify("Modo plan activo: edit, write y comandos que modifican están bloqueados", "info");
				}
				if (ctx.isIdle?.() === false) {
					pi.sendUserMessage(arg, { deliverAs: "followUp" });
				} else {
					pi.sendUserMessage(arg);
				}
				return;
			}
			if (active) {
				setActive(ctx, false);
				ctx.ui.notify("Modo plan desactivado", "info");
				return;
			}
			method = await loadMethod(ctx.cwd);
			setActive(ctx, true);
			ctx.ui.notify("Modo plan activo: edit, write y comandos que modifican están bloqueados", "info");
		},
	});

	pi.on("tool_call", async (event: any) => {
		if (!active) return;
		if (patterns.tools.includes(event.toolName)) {
			return { block: true, reason: REASON };
		}
		if (event.toolName === "bash" && bashBlocked(String(event.input?.command ?? ""))) {
			return { block: true, reason: REASON };
		}
	});

	// El método va como sección del prompt (HCD-045, Q5): reemplazar systemPrompt
	// tapaba la sección de gomemory y rompía la caché del prompt en cada cambio.
	pi.on("before_agent_start", async (event: any) => {
		const opts = event?.systemPromptOptions;
		if (!opts) return;
		opts.sections ??= {};
		if (active) opts.sections[PLAN_SECTION] = method || FALLBACK;
		else delete opts.sections[PLAN_SECTION];
	});
}

```

## pkg/installer/instructions.go
```
package installer

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// legacyPiInstructions es el protocolo que pi-gomemory escribía en
// ~/.pi/agent/AGENTS.md hasta HCD-045. Está congelado: ya no se instala (el
// contexto llega en vivo por la extensión gomemory-context.ts) y solo sirve
// para reconocer y retirar el formato antiguo sin marcadores.
const legacyPiInstructions = `# Memoria Persistente y Grafo de Código (gomemory) — Protocolo para Pi

Tienes conectados como herramientas directas y CLI en el entorno:
* **gomemory** (tools MCP directas: get_context, get_plan_context, search_memories, get_memory, save_memory, start_session, save_session_summary, end_session, search_code, get_symbol, octopus_route_task, judge_memories; CLI: mem)
* **codegraph** (tool MCP directa: codegraph_explore; CLI: codegraph)
* **codebase-memory** (tools MCP directas: search_graph, trace_path, get_code_snippet, check_index_coverage)

Si el modo codemode está activo ('pi-gomemory-setup codemode status'), el resto de tools de gomemory y codebase-memory (review_*, octopus_*, pack_*, hcd_history, socratica_flow, query_graph…) no se pierden: se llaman desde la tool 'codemode' con 'tools.<nombre>(args)'. No recurras a grep/find por no verlas como directas.

## Reglas Obligatorias de Búsqueda y Ejecución:

1. **Búsqueda Semántica y Memoria (No uses grep para decisiones pasadas):**
   - Llama a la tool directa 'search_memories(query)' o en bash ejecuta 'mem search "tema"'.
   - Para cargar contexto histórico: llama a 'get_context()' o en bash ejecuta 'mem context'.

2. **Exploración de Código (Grafo primero antes que find/grep):**
   - Para entender arquitectura, rutas o relaciones: llama a 'codegraph_explore' o 'search_graph' / 'trace_path'.
   - Usa grep/find en bash SOLO para literales exactos, configs o archivos no indexados.

3. **Guardado Proactivo Obligatorio:**
   - Inmediatamente tras decidir arquitectura, corregir un bug (con causa raíz) o fijar una convención:
     Llama a 'save_memory(title, type, content)' o en bash 'mem save -t "título" -y <decision|bugfix|pattern> "contenido"'.

4. **Metodología HCD:**
   - Comprueba la unidad activa con 'mem hcd show'.
   - Las comprobaciones de tareas ('go test ...', etc.) se ejecutan en 'bash' para observar el estado verde o rojo.
   - Registra la evidencia oficial ejecutando 'mem hcd verify'.

5. **Modo Plan:**
   - Antes de redactar un plan o estrategia: llama a 'get_plan_context()' o 'mem plan-context'.
   - Entrega el árbol de tareas atómicas y detente sin ejecutar código.

6. **Splash (revisor independiente):**
   - Ejecuta la herramienta splash_review con la ruta del documento y una lente opcional.
   - Splash es un rol agnóstico; no depende de ~/.claude/agents ni de otro harness.
   - No envíes historial ni memoria al revisor. No sustituyas la revisión por una autoevaluación.
   - Un fallo de ejecución no equivale a una revisión aprobada. Registra el resultado HCD solo después de leer la revisión real.
`

const (
	instructionsStart = "<!-- pi-gomemory:start -->"
	instructionsEnd   = "<!-- pi-gomemory:end -->"
)

// RetireLegacyInstructions quita de ~/.pi/agent/AGENTS.md el bloque que
// escribían las instalaciones anteriores a HCD-045 y nunca escribe uno nuevo.
// Devuelve true si había bloque. Sin AGENTS.md o sin bloque no toca nada.
func RetireLegacyInstructions(piAgentDir string) (bool, error) {
	path := filepath.Join(piAgentDir, "AGENTS.md")
	target, err := configTarget(path)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(target)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("leer %s: %w", target, err)
	}
	out, ok := stripInstructions(string(data))
	if !ok {
		return false, nil
	}
	if err := writeStrippedAgents(path, target, out); err != nil {
		return false, fmt.Errorf("retirar el protocolo heredado de %s: %w", path, err)
	}
	return true, nil
}

// writeStrippedAgents escribe AGENTS.md ya sin el bloque de pi-gomemory. Lo
// comparten install y uninstall (HCD-045, Q4): si no queda texto, un archivo
// normal se borra y un enlace se conserva con su destino vacío (HCD-036); en
// cualquier otro caso se conservan los permisos (HCD-032).
func writeStrippedAgents(path, target, out string) error {
	if strings.TrimSpace(out) == "" {
		if target == path {
			return os.Remove(path)
		}
		out = ""
	}
	mode, err := modeOrDefault(target, 0644)
	if err != nil {
		return err
	}
	return writeFileAtomic(target, []byte(out), mode)
}

// retireLegacyStep es el paso de install. Un AGENTS.md enlazado que no lleva a
// un archivo aborta la instalación, como el resto de configuraciones de Pi
// (HCD-036, Q6). Cualquier otro fallo solo se avisa y la instalación sigue
// (HCD-045, C-07); doctor lo seguirá avisando.
func retireLegacyStep(piAgentDir string, w io.Writer) error {
	retired, err := RetireLegacyInstructions(piAgentDir)
	var link *linkNotFileError
	switch {
	case errors.As(err, &link):
		return err
	case err != nil:
		fmt.Fprintf(w, "%s    ⚠ No se pudo retirar el protocolo heredado de AGENTS.md: %v. Corrige el permiso y ejecuta pi-gomemory-setup install.%s\n", ColorWarning, err, ColorReset)
	case retired:
		fmt.Fprintf(w, "%s    ✓ Protocolo heredado retirado de AGENTS.md: el contexto llega en vivo con la extensión gomemory-context%s\n", ColorMuted, ColorReset)
	}
	return nil
}

// legacyAgentsItem es la comprobación de doctor: avisa si AGENTS.md conserva
// el bloque heredado y devuelve nil si no hay nada que avisar.
func legacyAgentsItem(piAgentDir string) *CheckItem {
	target, err := configTarget(filepath.Join(piAgentDir, "AGENTS.md"))
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return nil
	}
	if _, ok := instructionsBlock(string(data)); !ok {
		return nil
	}
	return &CheckItem{
		Name:    "AGENTS.md de Pi",
		Status:  "WARN",
		Details: "AGENTS.md conserva el protocolo heredado de pi-gomemory; ejecuta pi-gomemory-setup install para retirarlo",
	}
}

```

## pkg/installer/extension.go
```
package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// tuiResources son las extensiones embebidas y la definición neutral de Splash.
var tuiResources = []struct{ source, destination string }{
	{"extensions/gomemory-status.ts", "gomemory-status.ts"},
	{"extensions/gomemory-splash.ts", "gomemory-splash.ts"},
	{"extensions/gomemory-banner.ts", "gomemory-banner.ts"},
	{"extensions/gomemory-plan.ts", "gomemory-plan.ts"},
	{"extensions/gomemory-plan-patterns.json", "gomemory-plan-patterns.json"},
	{"extensions/gomemory-tasks.ts", "gomemory-tasks.ts"},
	{"extensions/gomemory-sidebar.ts", "gomemory-sidebar.ts"},
	{"extensions/gomemory-tdd.ts", "gomemory-tdd.ts"},
	{"extensions/gomemory-context.ts", "gomemory-context.ts"}, // contexto vivo de gomemory (HCD-045)
	{"config/splash.md", "gomemory-splash-prompt.md"},
}

// InstallTUIExtension instala las extensiones embebidas y la definición neutral de Splash.
func InstallTUIExtension(assets fs.FS, piAgentDir string) error {
	for _, resource := range tuiResources {
		data, err := fs.ReadFile(assets, resource.source)
		if err != nil {
			return fmt.Errorf("leer extensión embebida: %w", err)
		}

		extDir := filepath.Join(piAgentDir, "extensions")
		if err := os.MkdirAll(extDir, 0755); err != nil {
			return fmt.Errorf("crear dir extensions en pi: %w", err)
		}

		dstFile := filepath.Join(extDir, resource.destination)
		if err := writeFileAtomic(dstFile, data, 0644); err != nil {
			return fmt.Errorf("escribir extensión %s: %w", dstFile, err)
		}
	}
	return nil
}

```

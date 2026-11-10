package installer

import (
	"os"
	"path/filepath"
)

// GomemoryPiInstructions contiene el protocolo activo de gomemory optimizado para Pi Coding Agent.
const GomemoryPiInstructions = `# Memoria Persistente y Grafo de Código (gomemory) — Protocolo para Pi

Tienes conectados como herramientas directas y CLI en el entorno:
* **gomemory** (tools MCP directas: search_memories, get_context, save_memory, search_code, get_symbol; CLI: mem)
* **codegraph** (tool MCP directa: codegraph_explore; CLI: codegraph)
* **codebase-memory** (tools MCP directas: search_graph, trace_path, get_code_snippet, check_index_coverage)

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
`

// InstallInstructions guarda las instrucciones en ~/.pi/agent/AGENTS.md
func InstallInstructions(piAgentDir string) error {
	agentsPath := filepath.Join(piAgentDir, "AGENTS.md")
	return os.WriteFile(agentsPath, []byte(GomemoryPiInstructions), 0644)
}

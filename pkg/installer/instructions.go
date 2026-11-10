package installer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

const (
	instructionsStart = "<!-- pi-gomemory:start -->"
	instructionsEnd   = "<!-- pi-gomemory:end -->"
)

// InstallInstructions escribe el protocolo en ~/.pi/agent/AGENTS.md dentro de
// un bloque delimitado. Si el bloque ya existe lo reemplaza; si no, lo añade
// al final. El resto del archivo de la persona se conserva intacto.
func InstallInstructions(piAgentDir string) error {
	if err := os.MkdirAll(piAgentDir, 0755); err != nil {
		return fmt.Errorf("crear %s: %w", piAgentDir, err)
	}
	agentsPath := filepath.Join(piAgentDir, "AGENTS.md")

	existing, err := os.ReadFile(agentsPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("leer %s: %w", agentsPath, err)
	}

	return os.WriteFile(agentsPath, []byte(mergeInstructions(string(existing))), 0644)
}

// mergeInstructions inserta o reemplaza el bloque de pi-gomemory en content.
func mergeInstructions(content string) string {
	block := instructionsStart + "\n" + GomemoryPiInstructions + instructionsEnd + "\n"

	start := strings.Index(content, instructionsStart)
	end := strings.Index(content, instructionsEnd)
	if start >= 0 && end > start {
		return content[:start] + block + strings.TrimPrefix(content[end+len(instructionsEnd):], "\n")
	}

	// Instalaciones ≤ v1.0.0 escribían el protocolo sin marcadores.
	if i := strings.Index(content, GomemoryPiInstructions); i >= 0 {
		return content[:i] + block + strings.TrimPrefix(content[i+len(GomemoryPiInstructions):], "\n")
	}

	if strings.TrimSpace(content) == "" {
		return block
	}
	return strings.TrimRight(content, "\n") + "\n\n" + block
}

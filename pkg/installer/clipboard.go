package installer

import "strings"

// clipboardDiagnostic checks prerequisites without reading or changing clipboard data.
// Tool availability does not prove that a graphical session accepts connections.
func clipboardDiagnostic(platform, display, wayland string, lookup func(string) (string, error)) (string, string) {
	groups := [][]string{}
	hint := ""
	switch platform {
	case "darwin":
		groups = append(groups, []string{"pbcopy", "pbpaste"})
		hint = "se requieren pbcopy y pbpaste del sistema"
	case "linux":
		if wayland != "" {
			groups = append(groups, []string{"wl-copy", "wl-paste"})
		}
		if display != "" {
			groups = append(groups, []string{"xclip"})
		}
		if len(groups) == 0 {
			return "INFO", "sin DISPLAY/ WAYLAND_DISPLAY; verificar portapapeles del terminal (OSC 52) o acceso gráfico"
		}
		hint = "instalar xclip para X11/WSLg o wl-clipboard para Wayland con el gestor de paquetes"
	default:
		return "INFO", "verificar portapapeles nativo del sistema"
	}
	for _, group := range groups {
		available := true
		for _, name := range group {
			if _, err := lookup(name); err != nil {
				available = false
				break
			}
		}
		if available {
			return "INFO", strings.Join(group, ", ") + " disponible; conexión gráfica pendiente de comprobar"
		}
	}
	return "WARN", hint
}

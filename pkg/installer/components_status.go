package installer

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// ComponentStatus es una fila de `pi-gomemory-setup components --json`
// (HCD-076, C-16).
type ComponentStatus struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Required  bool   `json:"required"`
	Installed bool   `json:"installed"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Outdated  bool   `json:"outdated"`
	Error     string `json:"error"`
}

// ComponentsStatus refresca a la fuerza las sondas (reescribiendo la caché) y
// devuelve el estado de cada componente del catálogo, en su orden.
func ComponentsStatus(probes []DepProbe, cachePath string, now time.Time) []ComponentStatus {
	states := map[string]DepState{}
	for _, d := range ForceRefreshDeps(probes, cachePath, now) {
		states[d.Name] = d
	}
	byProbe := map[string]DepProbe{}
	for _, p := range probes {
		byProbe[p.Name] = p
	}
	var out []ComponentStatus
	for _, c := range Catalog() {
		d := states[c.Name]
		if p, ok := byProbe[c.Name]; ok {
			d = effectiveDepState(d, p)
		}
		s := ComponentStatus{Name: c.Name, Kind: string(c.Kind), Required: c.Required, Current: d.Current, Latest: d.Latest, Error: d.Error}
		switch c.Kind {
		case KindTool:
			s.Installed = toolPresent(c.Name)
		default:
			s.Installed = d.Current != ""
		}
		if !s.Installed {
			// Ausente no es un error: el selector y /componentes lo ofrecen para instalar.
			s.Current, s.Error = "", ""
		}
		s.Outdated = s.Installed && s.Error == "" && IsNewer(s.Latest, s.Current)
		out = append(out, s)
	}
	return out
}

// WriteComponentsJSON escribe ComponentsStatus como JSON indentado.
func WriteComponentsJSON(w io.Writer, probes []DepProbe, cachePath string, now time.Time) error {
	data, err := json.MarshalIndent(ComponentsStatus(probes, cachePath, now), "", "  ")
	if err != nil {
		return fmt.Errorf("components --json: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

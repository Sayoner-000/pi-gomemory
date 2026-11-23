package installer

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Tarea 1: sin preferencia, install deja el perfil híbrido y registra que
// +codemode lo añadió pi-gomemory, sin fijar preferencia (HCD-071).
func TestHCD071DefaultHybrid(t *testing.T) {
	t.Run("instalación nueva", func(t *testing.T) {
		opts := sandbox(t)
		var out bytes.Buffer
		if err := applyCodemodeDefault(opts, &out); err != nil {
			t.Fatal(err)
		}
		if s, _ := GetCodemodeStatus(opts.PiAgentDir); !s.Enabled {
			t.Fatalf("install debe dejar codemode híbrido por defecto: %+v", s)
		}
		st := hcd046State(t, opts.HomeDir)
		if !st.AddedPlusCodemode || st.SettingsTarget == "" || st.Preference != "" {
			t.Fatalf("estado tras install por defecto: %+v", st)
		}
		if !strings.Contains(out.String(), "codemode off") {
			t.Fatalf("install no explica cómo volver a directo:\n%s", out.String())
		}
	})
	t.Run("instalación previa en directo sin preferencia pasa a híbrido", func(t *testing.T) {
		opts := sandbox(t)
		if err := reconcileMCPServers(opts.PiAgentDir, false, false); err != nil {
			t.Fatal(err)
		}
		if err := applyCodemodeDefault(opts, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		if s, _ := GetCodemodeStatus(opts.PiAgentDir); !s.Enabled {
			t.Fatalf("update no pasó a híbrido: %+v", s)
		}
	})
	t.Run("+codemode ajeno: híbrido sin atribuirse la propiedad", func(t *testing.T) {
		opts := sandbox(t)
		writeFile(t, filepath.Join(opts.PiAgentDir, "settings.json"), `{"defaultTools":["+codemode"]}`)
		var out bytes.Buffer
		if err := applyCodemodeDefault(opts, &out); err != nil {
			t.Fatal(err)
		}
		if s, _ := GetCodemodeStatus(opts.PiAgentDir); !s.Enabled || s.Partial {
			t.Fatalf("debía quedar híbrido completo: %+v", s)
		}
		if st := hcd046State(t, opts.HomeDir); st.AddedPlusCodemode {
			t.Fatalf("se atribuyó un +codemode ajeno: %+v", st)
		}
		if strings.Contains(out.String(), "parcial") {
			t.Fatalf("aviso de parcial con híbrido completo:\n%s", out.String())
		}
	})
	t.Run("es idempotente", func(t *testing.T) {
		opts := sandbox(t)
		for i := 0; i < 2; i++ {
			if err := applyCodemodeDefault(opts, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
		}
		if tools := hcd046DefaultTools(t, opts.PiAgentDir); len(tools) != 1 || tools[0] != "+codemode" {
			t.Fatalf("defaultTools tras dos install: %v", tools)
		}
		if st := hcd046State(t, opts.HomeDir); !st.AddedPlusCodemode {
			t.Fatalf("la segunda instalación perdió la propiedad: %+v", st)
		}
	})
}

// Tarea 2: codemode off explícito se respeta y un +codemode ajeno queda en
// parcial con aviso (no se retira lo que no añadió pi-gomemory).
func TestHCD071RespectsOff(t *testing.T) {
	t.Run("off se mantiene en directo", func(t *testing.T) {
		opts := sandbox(t)
		if err := SetCodemodePreference(opts, false); err != nil {
			t.Fatal(err)
		}
		if err := applyCodemodeDefault(opts, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		if s, _ := GetCodemodeStatus(opts.PiAgentDir); s.SettingsEnabled || s.MCPEnabled {
			t.Fatalf("install pisó codemode off: %+v", s)
		}
	})
	t.Run("off con +codemode ajeno avisa del parcial", func(t *testing.T) {
		opts := sandbox(t)
		writeFile(t, CodemodeStatePath(opts.HomeDir), `{"preference":"off"}`)
		writeFile(t, filepath.Join(opts.PiAgentDir, "settings.json"), `{"defaultTools":["+codemode"]}`)
		var out bytes.Buffer
		if err := applyCodemodeDefault(opts, &out); err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(hcd046DefaultTools(t, opts.PiAgentDir), "+codemode") {
			t.Fatal("retiró un +codemode ajeno")
		}
		if !strings.Contains(out.String(), "parcial") {
			t.Fatalf("sin aviso de parcial:\n%s", out.String())
		}
	})
}

// Tarea 3: sin preferencia y con lite, gomemory queda híbrido y los servidores
// de grafo gestionados se retiran (composición HCD-052).
func TestHCD071Lite(t *testing.T) {
	opts := sandbox(t)
	writeFile(t, LiteStatePath(opts.HomeDir), `{"enabled":true}`)
	if err := applyCodemodeDefault(opts, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	obj := jsonKeys(t, filepath.Join(opts.PiAgentDir, "mcp.json"))
	var servers map[string]MCPServerDef
	if err := json.Unmarshal(obj["mcpServers"], &servers); err != nil {
		t.Fatal(err)
	}
	if _, ok := servers["codebase-memory"]; ok {
		t.Fatalf("lite debía retirar codebase-memory: %v", servers)
	}
	if _, ok := servers["codegraph"]; ok {
		t.Fatalf("lite debía retirar codegraph: %v", servers)
	}
	if servers["gomemory"].Exposure != "codemode" {
		t.Fatalf("gomemory no quedó híbrido con lite: %+v", servers["gomemory"])
	}
	if !slices.Contains(hcd046DefaultTools(t, opts.PiAgentDir), "+codemode") {
		t.Fatal("falta +codemode con lite por defecto")
	}
}

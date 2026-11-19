package installer

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func hcd046State(t *testing.T, home string) codemodeState {
	t.Helper()
	st, err := readCodemodeState(home)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func hcd046DefaultTools(t *testing.T, piAgentDir string) []string {
	t.Helper()
	obj := jsonKeys(t, filepath.Join(piAgentDir, "settings.json"))
	var tools []string
	if raw, ok := obj["defaultTools"]; ok {
		if err := json.Unmarshal(raw, &tools); err != nil {
			t.Fatal(err)
		}
	}
	return tools
}

// Tarea 1: codemode on/off guardan la preferencia y si pi-gomemory añadió +codemode.
func TestHCD046Preference(t *testing.T) {
	t.Run("on añade y registra la propiedad; off la retira", func(t *testing.T) {
		opts := sandbox(t)
		if err := SetCodemodePreference(opts, true); err != nil {
			t.Fatal(err)
		}
		if st := hcd046State(t, opts.HomeDir); st.Preference != "on" || !st.AddedPlusCodemode {
			t.Fatalf("tras on: %+v", st)
		}
		if err := SetCodemodePreference(opts, false); err != nil {
			t.Fatal(err)
		}
		if st := hcd046State(t, opts.HomeDir); st.Preference != "off" || st.AddedPlusCodemode {
			t.Fatalf("tras off: %+v", st)
		}
		if s, _ := GetCodemodeStatus(opts.PiAgentDir); s.SettingsEnabled || s.MCPEnabled {
			t.Fatalf("off no desactivó codemode: %+v", s)
		}
	})
	t.Run("un +codemode previo de la persona no es de pi-gomemory", func(t *testing.T) {
		opts := sandbox(t)
		writeFile(t, filepath.Join(opts.PiAgentDir, "settings.json"), `{"defaultTools":["+codemode"]}`)
		if err := SetCodemodePreference(opts, true); err != nil {
			t.Fatal(err)
		}
		if st := hcd046State(t, opts.HomeDir); st.AddedPlusCodemode {
			t.Fatalf("se atribuyó a pi-gomemory un +codemode ajeno: %+v", st)
		}
	})
}

// Tarea 2 (HCD-050): install deja modo MCP directo (codemode off) por defecto,
// respeta un codemode on explícito y revierte limpiamente instalaciones v1.15.x
// sin preferencia (respetando +codemode ajeno).
func TestHCD046InstallDefault(t *testing.T) {
	t.Run("instalación nueva en modo directo (codemode off)", func(t *testing.T) {
		opts := sandbox(t)
		if err := InstallAll(repoAssets(t), opts); err != nil {
			t.Fatal(err)
		}
		if s, _ := GetCodemodeStatus(opts.PiAgentDir); s.Enabled || s.SettingsEnabled || s.MCPEnabled {
			t.Fatalf("install debe dejar codemode off por defecto: %+v", s)
		}
		st := hcd046State(t, opts.HomeDir)
		if st.AddedPlusCodemode || st.Preference != "" {
			t.Fatalf("install no debe activar ni fijar preferencia por defecto: %+v", st)
		}
	})
	t.Run("respeta codemode on explícito", func(t *testing.T) {
		opts := sandbox(t)
		if err := SetCodemodePreference(opts, true); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := applyCodemodeDefault(opts, &out); err != nil {
			t.Fatal(err)
		}
		if s, _ := GetCodemodeStatus(opts.PiAgentDir); !s.Enabled || !s.SettingsEnabled || !s.MCPEnabled {
			t.Fatalf("install pisó codemode on: %+v", s)
		}
		if out.Len() == 0 {
			t.Fatal("install no informa de que respeta codemode on")
		}
	})
	t.Run("revierte limpiamente default v1.15.x sin preferencia", func(t *testing.T) {
		opts := sandbox(t)
		// Simula una instalación de v1.15.x: activó codemode por defecto (preference == "")
		// dejando added_plus_codemode=true y settings_target apuntando a settings.json.
		var st codemodeState
		if err := enableCodemode(opts, &st); err != nil {
			t.Fatal(err)
		}
		if err := writeCodemodeState(opts.HomeDir, st); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := applyCodemodeDefault(opts, &out); err != nil {
			t.Fatal(err)
		}
		if s, _ := GetCodemodeStatus(opts.PiAgentDir); s.Enabled || s.SettingsEnabled || s.MCPEnabled || s.Partial {
			t.Fatalf("no revirtió limpiamente a modo directo: %+v", s)
		}
		after := hcd046State(t, opts.HomeDir)
		if after.AddedPlusCodemode || after.SettingsTarget != "" || after.Preference != "" {
			t.Fatalf("el estado no quedó limpio tras revertir v1.15.x: %+v", after)
		}
	})
	t.Run("no toca +codemode de v1.15.0 sin settings_target ni de otro destino", func(t *testing.T) {
		opts := sandbox(t)
		settings := filepath.Join(opts.PiAgentDir, "settings.json")
		writeFile(t, settings, `{"defaultTools":["+codemode"]}`)
		writeFile(t, CodemodeStatePath(opts.HomeDir), `{"added_plus_codemode":true}`)
		var out bytes.Buffer
		if err := applyCodemodeDefault(opts, &out); err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(hcd046DefaultTools(t, opts.PiAgentDir), "+codemode") {
			t.Fatal("retiró +codemode sin settings_target verificable")
		}
		if s, _ := GetCodemodeStatus(opts.PiAgentDir); s.MCPEnabled {
			t.Fatalf("MCP debe quedar en modo directo: %+v", s)
		}
	})
}

// Tarea 3: uninstall retira el perfil directo, el perfil híbrido, el +codemode propio y el estado.
func TestHCD046Uninstall(t *testing.T) {
	t.Run("propio tras codemode on", func(t *testing.T) {
		opts := installSandbox(t)
		if err := SetCodemodePreference(opts, true); err != nil {
			t.Fatal(err)
		}
		plan, err := BuildUninstallPlan(repoAssets(t), opts, UninstallFlags{})
		if err != nil {
			t.Fatal(err)
		}
		hcd042RequireCloser(t, plan)
		mcp := filepath.Join(opts.PiAgentDir, "mcp.json")
		settings := filepath.Join(opts.PiAgentDir, "settings.json")
		for _, target := range []string{
			mcp + " › mcpServers.gomemory",
			mcp + " › mcpServers.codebase-memory",
			settings + " › defaultTools (+codemode)",
			CodemodeStatePath(opts.HomeDir),
		} {
			if a := actionFor(plan, target); a == nil || a.Kind != ActionRemove {
				t.Errorf("%s: quiero retirar, obtengo %+v", target, a)
			}
		}
		var out bytes.Buffer
		if err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true}, &out); err != nil {
			t.Fatalf("%v\n%s", err, out.String())
		}
		if slices.Contains(hcd046DefaultTools(t, opts.PiAgentDir), "+codemode") {
			t.Fatal("uninstall dejó +codemode")
		}
		if _, err := os.Stat(CodemodeStatePath(opts.HomeDir)); !os.IsNotExist(err) {
			t.Fatalf("uninstall dejó el estado de codemode: %v", err)
		}
	})
	t.Run("+codemode ajeno se conserva", func(t *testing.T) {
		opts := sandbox(t)
		writeFile(t, filepath.Join(opts.PiAgentDir, "settings.json"), `{"defaultTools":["+codemode"]}`)
		if err := InstallAll(repoAssets(t), opts); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true}, &out); err != nil {
			t.Fatalf("%v\n%s", err, out.String())
		}
		if !slices.Contains(hcd046DefaultTools(t, opts.PiAgentDir), "+codemode") {
			t.Fatal("uninstall retiró un +codemode que no puso pi-gomemory")
		}
	})
}

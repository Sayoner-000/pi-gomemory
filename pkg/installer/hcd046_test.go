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

// Tarea 2: install deja codemode híbrido salvo un off explícito.
func TestHCD046InstallDefault(t *testing.T) {
	t.Run("instalación nueva en codemode", func(t *testing.T) {
		opts := sandbox(t)
		if err := InstallAll(repoAssets(t), opts); err != nil {
			t.Fatal(err)
		}
		if s, _ := GetCodemodeStatus(opts.PiAgentDir); !s.Enabled {
			t.Fatalf("install no activó codemode: %+v", s)
		}
		st := hcd046State(t, opts.HomeDir)
		if !st.AddedPlusCodemode || st.Preference != "" {
			t.Fatalf("install debe registrar la propiedad sin fijar preferencia: %+v", st)
		}
	})
	t.Run("respeta codemode off", func(t *testing.T) {
		opts := sandbox(t)
		if err := SetCodemodePreference(opts, false); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := applyCodemodeDefault(opts, &out); err != nil {
			t.Fatal(err)
		}
		if s, _ := GetCodemodeStatus(opts.PiAgentDir); s.SettingsEnabled || s.MCPEnabled {
			t.Fatalf("install pisó codemode off: %+v", s)
		}
		if out.Len() == 0 {
			t.Fatal("install no informa de que respeta codemode off")
		}
	})
}

// Tarea 3: uninstall retira el perfil híbrido, el +codemode propio y el estado.
func TestHCD046Uninstall(t *testing.T) {
	t.Run("propio", func(t *testing.T) {
		opts := installSandbox(t)
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

package installer

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// HCD-047: regresión de los ocho hallazgos del ACR de v1.15.0 (C-01 a C-08).
// Solo HOME temporales y datos sintéticos.

func hcd047RawState(t *testing.T, home string) map[string]any {
	t.Helper()
	st := map[string]any{}
	if err := json.Unmarshal([]byte(readFile(t, CodemodeStatePath(home))), &st); err != nil {
		t.Fatalf("codemode.json: %v", err)
	}
	return st
}

func hcd047Tools(t *testing.T, path string) []string {
	t.Helper()
	var tools []string
	if raw, ok := jsonKeys(t, path)["defaultTools"]; ok {
		if err := json.Unmarshal(raw, &tools); err != nil {
			t.Fatal(err)
		}
	}
	return tools
}

func hcd047Servers(t *testing.T, piAgentDir string) map[string]json.RawMessage {
	t.Helper()
	servers := map[string]json.RawMessage{}
	if raw, ok := jsonKeys(t, filepath.Join(piAgentDir, "mcp.json"))["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			t.Fatal(err)
		}
	}
	return servers
}

func hcd047Uninstall(t *testing.T, opts *Options) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Uninstall(repoAssets(t), opts, UninstallFlags{Yes: true}, &out)
	return out.String(), err
}

func hcd047SkipRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root ignora los permisos del sistema de archivos")
	}
}

// hcd047Link apunta settings.json a target, sustituyendo el enlace anterior.
func hcd047Link(t *testing.T, path, target string) {
	t.Helper()
	_ = os.Remove(path)
	mkdir(t, filepath.Dir(path))
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

func TestHCD047Regression(t *testing.T) {
	t.Run("C-01", func(t *testing.T) {
		t.Run("fallo de MCP restaura y el reintento registra la propiedad", func(t *testing.T) {
			opts := sandbox(t)
			settings := filepath.Join(opts.PiAgentDir, "settings.json")
			mcp := filepath.Join(opts.PiAgentDir, "mcp.json")
			const origSettings = `{"defaultTools":["read"]}`
			writeFile(t, settings, origSettings)
			writeFile(t, mcp, "{")
			if err := SetCodemodePreference(opts, true); err == nil {
				t.Fatal("codemode on con mcp.json inválido debe fallar")
			}
			if got := readFile(t, settings); got != origSettings {
				t.Fatalf("el fallo dejó un cambio silencioso en settings.json: %q", got)
			}
			writeFile(t, mcp, "{}")
			if err := SetCodemodePreference(opts, true); err != nil {
				t.Fatal(err)
			}
			if st := hcd047RawState(t, opts.HomeDir); st["added_plus_codemode"] != true {
				t.Fatalf("el reintento perdió la propiedad de +codemode: %v", st)
			}
			if out, err := hcd047Uninstall(t, opts); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			if tools := hcd047Tools(t, settings); slices.Contains(tools, "+codemode") || !slices.Contains(tools, "read") {
				t.Fatalf("uninstall tras el reintento: %v", tools)
			}
		})
		t.Run("fallo del estado restaura y el reintento registra la propiedad", func(t *testing.T) {
			hcd047SkipRoot(t)
			opts := sandbox(t)
			settings := filepath.Join(opts.PiAgentDir, "settings.json")
			mcp := filepath.Join(opts.PiAgentDir, "mcp.json")
			const origSettings, origMCP = `{"defaultTools":["read"]}`, `{"mcpServers":{"mine":{"command":"x"}}}`
			writeFile(t, settings, origSettings)
			writeFile(t, mcp, origMCP)
			statePath := CodemodeStatePath(opts.HomeDir)
			stateDir := filepath.Dir(statePath)
			writeFile(t, statePath, `{"preference":"on"}`)
			if err := os.Chmod(stateDir, 0555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(stateDir, 0755) })
			if err := applyCodemodeDefault(opts, &bytes.Buffer{}); err == nil {
				t.Fatal("install con el estado no escribible debe fallar")
			}
			if got := readFile(t, settings); got != origSettings {
				t.Fatalf("settings.json cambió sin registro de propiedad: %q", got)
			}
			if got := readFile(t, mcp); got != origMCP {
				t.Fatalf("mcp.json cambió sin registro de propiedad: %q", got)
			}
			if err := os.Chmod(stateDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := applyCodemodeDefault(opts, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
			if st := hcd047RawState(t, opts.HomeDir); st["added_plus_codemode"] != true {
				t.Fatalf("el reintento perdió la propiedad de +codemode: %v", st)
			}
			if out, err := hcd047Uninstall(t, opts); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			if slices.Contains(hcd047Tools(t, settings), "+codemode") {
				t.Fatal("uninstall dejó el +codemode que añadió el instalador")
			}
		})
	})

	t.Run("C-02", func(t *testing.T) {
		opts := sandbox(t)
		if err := SetCodemodePreference(opts, false); err != nil {
			t.Fatal(err)
		}
		settings := filepath.Join(opts.PiAgentDir, "settings.json")
		writeFile(t, settings, `{"defaultTools":["read","+codemode"]}`)
		if err := InstallAll(repoAssets(t), opts); err != nil {
			t.Fatal(err)
		}
		if tools := hcd047Tools(t, settings); !slices.Equal(tools, []string{"read", "+codemode"}) {
			t.Fatalf("install cambió la selección ajena de defaultTools: %v", tools)
		}
		servers := hcd047Servers(t, opts.PiAgentDir)
		for name, def := range DefaultMCPServers() {
			want, _ := json.Marshal(def)
			if !sameJSON(servers[name], want) {
				t.Errorf("%s no quedó en modo directo con off: %s", name, servers[name])
			}
		}
		if st := hcd047RawState(t, opts.HomeDir); st["preference"] != "off" {
			t.Fatalf("install no conservó off: %v", st)
		}
	})

	t.Run("C-03", func(t *testing.T) {
		opts := sandbox(t)
		agents := filepath.Join(opts.PiAgentDir, "AGENTS.md")
		original := "SYNTH-USER\n\n" + hcd045Block()
		writeFile(t, agents, original)
		ctx := filepath.Join(opts.PiAgentDir, "extensions", "gomemory-context.ts")
		writeFile(t, filepath.Join(ctx, "ocupa"), "x")
		if err := InstallAll(repoAssets(t), opts); err == nil {
			t.Fatal("install con el destino de la extensión ocupado debe fallar")
		}
		if got := readFile(t, agents); got != original {
			t.Fatalf("el fallo de instalación alteró AGENTS.md: %q", got)
		}
		if err := os.RemoveAll(ctx); err != nil {
			t.Fatal(err)
		}
		if err := InstallAll(repoAssets(t), opts); err != nil {
			t.Fatal(err)
		}
		if readFile(t, ctx) != readAsset(t, "extensions/gomemory-context.ts") {
			t.Fatal("el reintento no instaló gomemory-context.ts")
		}
		if got := readFile(t, agents); got != "SYNTH-USER\n" {
			t.Fatalf("el reintento no retiró solo el protocolo: %q", got)
		}
	})

	t.Run("C-04", func(t *testing.T) {
		block := hcd045Block()
		cases := map[string]string{
			"apertura huérfana":        instructionsStart + "\nSYNTH-USER\n" + block,
			"bloques duplicados":       block + "SYNTH-USER\n" + block,
			"cierre antes de apertura": instructionsEnd + "\nSYNTH-USER\n" + instructionsStart + "\n",
			"ambiguo junto a formato sin marcas": legacyProtocolV1 + "\nSYNTH-USER\n" +
				instructionsStart + "\nSYNTH-MID\n",
		}
		for name, original := range cases {
			t.Run(name, func(t *testing.T) {
				opts := sandbox(t)
				agents := filepath.Join(opts.PiAgentDir, "AGENTS.md")
				writeFile(t, agents, original)
				var out bytes.Buffer
				if err := retireLegacyStep(opts.PiAgentDir, &out); err != nil {
					t.Fatal(err)
				}
				if got := readFile(t, agents); got != original {
					t.Fatalf("install modificó un AGENTS.md ambiguo: %q", got)
				}
				if !strings.Contains(out.String(), "ambiguo") {
					t.Fatalf("install no avisó de la ambigüedad: %q", out.String())
				}
				msg, _ := hcd047Uninstall(t, opts)
				if got := readFile(t, agents); got != original {
					t.Fatalf("uninstall modificó un AGENTS.md ambiguo: %q", got)
				}
				if !strings.Contains(msg, "ambiguo") {
					t.Fatalf("uninstall no avisó de la ambigüedad: %q", msg)
				}
			})
		}
	})

	t.Run("C-05", func(t *testing.T) {
		original := "# SYNTH-HEAD\n\n" + legacyProtocolV1 + "\nSYNTH-MID\n\n" + hcd045Block() +
			"\nSYNTH-TAIL\n\n" + legacyProtocolV2
		for _, linked := range []bool{false, true} {
			t.Run(map[bool]string{false: "archivo", true: "enlace"}[linked], func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "AGENTS.md")
				target := path
				if linked {
					target = linkConfig(t, path, original, 0600)
				} else {
					writeFile(t, path, original)
					if err := os.Chmod(path, 0600); err != nil {
						t.Fatal(err)
					}
				}
				if retired, err := RetireLegacyInstructions(dir); err != nil || !retired {
					t.Fatalf("RetireLegacyInstructions = %v, %v", retired, err)
				}
				got := readFile(t, target)
				for _, p := range legacyPiProtocols {
					if strings.Contains(got, p) {
						t.Fatalf("quedó un protocolo reconocido: %q", got)
					}
				}
				if strings.Contains(got, instructionsStart) || strings.Contains(got, instructionsEnd) {
					t.Fatalf("quedaron marcadores: %q", got)
				}
				h, m, e := strings.Index(got, "# SYNTH-HEAD"), strings.Index(got, "SYNTH-MID"), strings.Index(got, "SYNTH-TAIL")
				if h < 0 || m < h || e < m {
					t.Fatalf("no se conservó el texto propio en orden: %q", got)
				}
				assertMode(t, target, 0600)
				if linked {
					assertLink(t, path, target)
				}
			})
		}
	})

	t.Run("C-06", func(t *testing.T) {
		fixture := readFile(t, filepath.Join("testdata", "hcd047", "mcp-v1.14.0.json"))
		var obj struct {
			MCPServers map[string]json.RawMessage `json:"mcpServers"`
		}
		if err := json.Unmarshal([]byte(fixture), &obj); err != nil {
			t.Fatal(err)
		}
		current, _ := json.Marshal(CodemodeMCPServers()["gomemory"])
		if sameJSON(obj.MCPServers["gomemory"], current) {
			t.Fatal("el fixture de v1.14.0 coincide con el perfil vigente: no prueba el histórico")
		}
		mine := json.RawMessage(`{"command":"synth-own"}`)
		t.Run("perfil exacto se retira", func(t *testing.T) {
			opts := sandbox(t)
			servers := map[string]json.RawMessage{"mine": mine}
			for k, v := range obj.MCPServers {
				servers[k] = v
			}
			data, _ := json.Marshal(map[string]any{"mcpServers": servers})
			writeFile(t, filepath.Join(opts.PiAgentDir, "mcp.json"), string(data))
			if out, err := hcd047Uninstall(t, opts); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			got := hcd047Servers(t, opts.PiAgentDir)
			if len(got) != 1 || !sameJSON(got["mine"], mine) {
				t.Fatalf("quedaron entradas de v1.14.0 o se perdió la propia: %v", got)
			}
		})
		t.Run("entrada propia con el mismo nombre se conserva", func(t *testing.T) {
			opts := sandbox(t)
			var gm map[string]any
			_ = json.Unmarshal(obj.MCPServers["gomemory"], &gm)
			gm["toolExposure"].(map[string]any)["synth_tool"] = "direct"
			edited, _ := json.Marshal(gm)
			data, _ := json.Marshal(map[string]any{"mcpServers": map[string]json.RawMessage{"gomemory": edited}})
			writeFile(t, filepath.Join(opts.PiAgentDir, "mcp.json"), string(data))
			if out, err := hcd047Uninstall(t, opts); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			if got := hcd047Servers(t, opts.PiAgentDir)["gomemory"]; !sameJSON(got, edited) {
				t.Fatalf("se retiró o alteró una entrada personalizada: %s", got)
			}
		})
	})

	t.Run("C-07", func(t *testing.T) {
		for name, unreadable := range map[string]bool{"JSON truncado": false, "error de lectura": true} {
			t.Run(name, func(t *testing.T) {
				if unreadable {
					hcd047SkipRoot(t)
				}
				opts := sandbox(t)
				settings := filepath.Join(opts.PiAgentDir, "settings.json")
				writeFile(t, settings, `{"defaultTools":["+codemode"]}`)
				state := CodemodeStatePath(opts.HomeDir)
				const content = `{"added_plus_codemode":tr`
				writeFile(t, state, content)
				if unreadable {
					if err := os.Chmod(state, 0); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = os.Chmod(state, 0644) })
				}
				out, err := hcd047Uninstall(t, opts)
				if err == nil || !strings.Contains(out, state) {
					t.Fatalf("uninstall no informó del estado inválido (err=%v):\n%s", err, out)
				}
				if unreadable {
					if err := os.Chmod(state, 0644); err != nil {
						t.Fatal(err)
					}
				}
				if got := readFile(t, state); got != content {
					t.Fatalf("no se conservaron los bytes del estado: %q", got)
				}
				if !slices.Contains(hcd047Tools(t, settings), "+codemode") {
					t.Fatal("se retiró un +codemode sin propiedad verificable")
				}
			})
		}
	})

	t.Run("C-08", func(t *testing.T) {
		prepare := func(t *testing.T) (opts *Options, settings, a, b string) {
			opts = sandbox(t)
			settings = filepath.Join(opts.PiAgentDir, "settings.json")
			a, b = filepath.Join(opts.HomeDir, "A.json"), filepath.Join(opts.HomeDir, "B.json")
			writeFile(t, a, `{}`)
			writeFile(t, b, `{"defaultTools":["+codemode"]}`)
			hcd047Link(t, settings, a)
			if err := SetCodemodePreference(opts, true); err != nil {
				t.Fatal(err)
			}
			return opts, settings, a, b
		}
		userKeeps := func(t *testing.T, b string) {
			t.Helper()
			if !slices.Contains(hcd047Tools(t, b), "+codemode") {
				t.Fatal("se retiró el +codemode del usuario en otro destino")
			}
		}
		t.Run("cambio antes del plan", func(t *testing.T) {
			opts, settings, _, b := prepare(t)
			hcd047Link(t, settings, b)
			if out, err := hcd047Uninstall(t, opts); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			userKeeps(t, b)
		})
		t.Run("cambio después del plan", func(t *testing.T) {
			opts, settings, _, b := prepare(t)
			plan, err := BuildUninstallPlan(repoAssets(t), opts, UninstallFlags{})
			if err != nil {
				t.Fatal(err)
			}
			defer plan.Close()
			a := actionFor(plan, settings+" › defaultTools (+codemode)")
			if a == nil || a.apply == nil {
				t.Fatalf("el plan no retira el +codemode propio: %+v", a)
			}
			hcd047Link(t, settings, b)
			_ = a.apply()
			userKeeps(t, b)
		})
		t.Run("cambio antes de reinstalar", func(t *testing.T) {
			opts, settings, _, b := prepare(t)
			hcd047Link(t, settings, b)
			if err := applyCodemodeDefault(opts, &bytes.Buffer{}); err != nil {
				t.Fatal(err)
			}
			if out, err := hcd047Uninstall(t, opts); err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			userKeeps(t, b)
		})
		t.Run("registro heredado sin destino (Q1)", func(t *testing.T) {
			opts := sandbox(t)
			settings := filepath.Join(opts.PiAgentDir, "settings.json")
			writeFile(t, settings, `{"defaultTools":["+codemode"]}`)
			writeFile(t, CodemodeStatePath(opts.HomeDir), `{"added_plus_codemode":true}`)
			out, err := hcd047Uninstall(t, opts)
			if err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			if !slices.Contains(hcd047Tools(t, settings), "+codemode") {
				t.Fatal("se retiró +codemode sin poder comprobar su destino")
			}
			if !strings.Contains(out, "no se pudo comprobar") {
				t.Fatalf("falta el aviso de Q1:\n%s", out)
			}
		})
	})
}

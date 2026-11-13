package installer

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
)

// mutatingReader ejecuta fn justo cuando uninstall pide confirmación, es decir,
// después de construir el plan y antes de aplicarlo.
type mutatingReader struct {
	once sync.Once
	fn   func()
	r    io.Reader
}

func (m *mutatingReader) Read(p []byte) (int, error) {
	m.once.Do(m.fn)
	return m.r.Read(p)
}

// afterPlan simula una terminal interactiva que, antes de responder «s»,
// ejecuta mutate.
func afterPlan(t *testing.T, mutate func()) {
	t.Helper()
	oldTTY, oldIn := stdinIsTerminal, confirmInput
	stdinIsTerminal = func() bool { return true }
	confirmInput = &mutatingReader{fn: mutate, r: strings.NewReader("s\n")}
	t.Cleanup(func() { stdinIsTerminal, confirmInput = oldTTY, oldIn })
}

func runUninstall(t *testing.T, opts *Options, flags UninstallFlags) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Uninstall(repoAssets(t), opts, flags, &out)
	return out.String(), err
}

func line(kind, target, reason string) string {
	return "  [" + kind + "] " + target + " — " + reason + "\n"
}

// planHas comprueba que el plan retira target; sin eso la prueba no ejercita nada.
func planHas(t *testing.T, opts *Options, flags UninstallFlags, target string) {
	t.Helper()
	plan, err := BuildUninstallPlan(repoAssets(t), opts, flags)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range plan.Actions {
		if a.Target == target && a.Kind == ActionRemove {
			return
		}
	}
	t.Fatalf("el plan no retira %s: %+v", target, plan.Actions)
}

func editJSONFile(t *testing.T, path string, fn func(map[string]any)) {
	t.Helper()
	obj := map[string]any{}
	if err := json.Unmarshal([]byte(readFile(t, path)), &obj); err != nil {
		t.Fatal(err)
	}
	fn(obj)
	data, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(data))
}

func selfLoop(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path, path); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

// C-01: un archivo gestionado editado o sustituido por un symlink tras el plan se conserva.
func TestHCD034C01(t *testing.T) {
	for _, verbose := range []bool{false, true} {
		for name, mutate := range map[string]func(t *testing.T, path, home string){
			"editado": func(t *testing.T, path, _ string) { writeFile(t, path, "EDICION-PROPIA") },
			"symlink": func(t *testing.T, path, home string) {
				other := filepath.Join(home, "otro.ts")
				writeFile(t, other, "EDICION-PROPIA")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, path); err != nil {
					t.Fatal(err)
				}
			},
		} {
			t.Run(name, func(t *testing.T) {
				opts := installSandbox(t)
				path := filepath.Join(opts.PiAgentDir, "extensions", "gomemory-status.ts")
				planHas(t, opts, UninstallFlags{}, path)
				afterPlan(t, func() { mutate(t, path, opts.HomeDir) })
				out, err := runUninstall(t, opts, UninstallFlags{Verbose: verbose})
				if err != nil {
					t.Fatalf("uninstall: %v\n%s", err, out)
				}
				if got := readFile(t, path); got != "EDICION-PROPIA" {
					t.Fatalf("se modificó %s: %q", path, got)
				}
				if n := strings.Count(out, line(ActionSkip, path, reasonChanged)); n != 1 {
					t.Fatalf("verbose=%v: la línea de omitir aparece %d veces:\n%s", verbose, n, out)
				}
			})
		}
	}
}

// C-02: una skill embebida con un archivo añadido tras el plan se conserva completa.
func TestHCD034C02(t *testing.T) {
	opts := installSandbox(t)
	dir := filepath.Join(SkillRepoDir(opts.HomeDir), "hcd")
	planHas(t, opts, UninstallFlags{}, dir)
	notes := filepath.Join(dir, "notas.txt")
	afterPlan(t, func() { writeFile(t, notes, "notas propias") })
	out, err := runUninstall(t, opts, UninstallFlags{})
	if err != nil {
		t.Fatalf("uninstall: %v\n%s", err, out)
	}
	for _, p := range []string{notes, filepath.Join(dir, "SKILL.md"), filepath.Join(dir, skillMarkerFile)} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("la skill no se conservó completa: %v", err)
		}
	}
	if !strings.Contains(out, line(ActionSkip, dir, reasonChanged)) {
		t.Fatalf("falta la línea de omitir:\n%s", out)
	}
}

// C-03: las entradas JSON, el bloque de AGENTS.md, packages, pi-cc-extensions.json
// y el enlace a skill del anfitrión que cambian tras el plan se conservan.
func TestHCD034C03(t *testing.T) {
	type tc struct {
		flags  UninstallFlags
		setup  func(t *testing.T, opts *Options)
		target func(opts *Options) string
		mutate func(t *testing.T, opts *Options)
		check  func(t *testing.T, opts *Options)
	}
	settings := func(o *Options) string { return filepath.Join(o.PiAgentDir, "settings.json") }
	mcp := func(o *Options) string { return filepath.Join(o.PiAgentDir, "mcp.json") }
	cases := map[string]tc{
		"mcpServers": {
			target: func(o *Options) string { return mcp(o) + " › mcpServers.gomemory" },
			mutate: func(t *testing.T, o *Options) {
				editJSONFile(t, mcp(o), func(m map[string]any) {
					m["mcpServers"].(map[string]any)["gomemory"] = map[string]any{"command": "propio"}
				})
			},
			check: func(t *testing.T, o *Options) {
				if !strings.Contains(readFile(t, mcp(o)), `"propio"`) {
					t.Fatal("se retiró la entrada MCP cambiada")
				}
			},
		},
		"theme": {
			target: func(o *Options) string { return settings(o) + " › theme" },
			mutate: func(t *testing.T, o *Options) {
				editJSONFile(t, settings(o), func(m map[string]any) { m["theme"] = "oscuro" })
			},
			check: func(t *testing.T, o *Options) {
				if !strings.Contains(readFile(t, settings(o)), `"oscuro"`) {
					t.Fatal("se retiró la clave theme cambiada")
				}
			},
		},
		"registro de skills": {
			target: func(o *Options) string { return settings(o) + " › skills (" + SkillRepoSetting + ")" },
			mutate: func(t *testing.T, o *Options) {
				editJSONFile(t, settings(o), func(m map[string]any) { m["skills"] = map[string]any{"propio": SkillRepoSetting} })
			},
			check: func(t *testing.T, o *Options) {
				if !strings.Contains(readFile(t, settings(o)), `"propio"`) {
					t.Fatal("se modificó el registro de skills cambiado")
				}
			},
		},
		"bloque de AGENTS.md": {
			target: func(o *Options) string { return filepath.Join(o.PiAgentDir, "AGENTS.md") + " › bloque pi-gomemory" },
			mutate: func(t *testing.T, o *Options) {
				p := filepath.Join(o.PiAgentDir, "AGENTS.md")
				writeFile(t, p, strings.Replace(readFile(t, p), instructionsStart, instructionsStart+"\nNOTA-PROPIA", 1))
			},
			check: func(t *testing.T, o *Options) {
				if !strings.Contains(readFile(t, filepath.Join(o.PiAgentDir, "AGENTS.md")), "NOTA-PROPIA") {
					t.Fatal("se retiró el bloque cambiado")
				}
			},
		},
		"packages": {
			flags: UninstallFlags{Packages: true},
			setup: func(t *testing.T, o *Options) {
				editJSONFile(t, settings(o), func(m map[string]any) { m["packages"] = []any{AntigravityPackageSource} })
				// Un `pi` falso deja constancia si se invoca.
				bin := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))[0]
				writeFile(t, filepath.Join(bin, "pi"), "#!/bin/sh\ntouch \""+filepath.Join(o.HomeDir, "pi-invocado")+"\"\n")
				if err := os.Chmod(filepath.Join(bin, "pi"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			target: func(o *Options) string { return settings(o) + " › packages.pi-antigravity" },
			mutate: func(t *testing.T, o *Options) {
				editJSONFile(t, settings(o), func(m map[string]any) { m["packages"] = []any{"npm:pi-antigravity@9.9.9"} })
			},
			check: func(t *testing.T, o *Options) {
				if !strings.Contains(readFile(t, settings(o)), "pi-antigravity@9.9.9") {
					t.Fatal("se retiró la entrada packages cambiada")
				}
				if _, err := os.Stat(filepath.Join(o.HomeDir, "pi-invocado")); err == nil {
					t.Fatal("se invocó pi remove sobre una entrada cambiada")
				}
			},
		},
		"pi-cc-extensions.json": {
			flags: UninstallFlags{Packages: true},
			setup: func(t *testing.T, o *Options) {
				def, err := fs.ReadFile(repoAssets(t), "config/pi-cc-extensions.json")
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(o.PiAgentDir, "pi-cc-extensions.json"), string(def))
			},
			target: func(o *Options) string { return filepath.Join(o.PiAgentDir, "pi-cc-extensions.json") },
			mutate: func(t *testing.T, o *Options) {
				editJSONFile(t, filepath.Join(o.PiAgentDir, "pi-cc-extensions.json"), func(m map[string]any) { m["miClave"] = true })
			},
			check: func(t *testing.T, o *Options) {
				if !strings.Contains(readFile(t, filepath.Join(o.PiAgentDir, "pi-cc-extensions.json")), "miClave") {
					t.Fatal("se retiró pi-cc-extensions.json cambiado")
				}
			},
		},
		"enlace a skill del anfitrión": {
			target: func(o *Options) string { return filepath.Join(SkillRepoDir(o.HomeDir), "demo-anfitrion") },
			mutate: func(t *testing.T, o *Options) {
				link := filepath.Join(SkillRepoDir(o.HomeDir), "demo-anfitrion")
				other := filepath.Join(o.HomeDir, ".claude", "skills", "otra")
				writeFile(t, filepath.Join(other, "SKILL.md"), "otra")
				if err := os.Remove(link); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, link); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, o *Options) {
				cur, err := os.Readlink(filepath.Join(SkillRepoDir(o.HomeDir), "demo-anfitrion"))
				if err != nil || !strings.HasSuffix(cur, "otra") {
					t.Fatalf("se retiró el enlace cambiado: %q %v", cur, err)
				}
			},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			opts := sandbox(t)
			writeFile(t, filepath.Join(opts.HomeDir, ".agents", "skills", "demo-anfitrion", "SKILL.md"), "demo")
			if err := InstallAll(repoAssets(t), opts); err != nil {
				t.Fatal(err)
			}
			if c.setup != nil {
				c.setup(t, opts)
			}
			target := c.target(opts)
			planHas(t, opts, c.flags, target)
			afterPlan(t, func() { c.mutate(t, opts) })
			out, err := runUninstall(t, opts, c.flags)
			if err != nil {
				t.Fatalf("uninstall: %v\n%s", err, out)
			}
			if !strings.Contains(out, line(ActionSkip, target, reasonChanged)) {
				t.Fatalf("falta la línea de omitir para %s:\n%s", target, out)
			}
			c.check(t, opts)
		})
	}
}

// C-04: un recurso, o una clave cuyo archivo sigue, que desaparece tras el plan
// se omite con «ya no existe» y no hace fallar el comando.
func TestHCD034C04(t *testing.T) {
	cases := map[string]struct {
		target func(o *Options) string
		mutate func(t *testing.T, o *Options)
	}{
		"archivo": {
			target: func(o *Options) string { return filepath.Join(o.PiAgentDir, "extensions", "gomemory-status.ts") },
			mutate: func(t *testing.T, o *Options) {
				if err := os.Remove(filepath.Join(o.PiAgentDir, "extensions", "gomemory-status.ts")); err != nil {
					t.Fatal(err)
				}
			},
		},
		"clave con el archivo presente": {
			target: func(o *Options) string { return filepath.Join(o.PiAgentDir, "mcp.json") + " › mcpServers.gomemory" },
			mutate: func(t *testing.T, o *Options) {
				editJSONFile(t, filepath.Join(o.PiAgentDir, "mcp.json"), func(m map[string]any) {
					delete(m["mcpServers"].(map[string]any), "gomemory")
				})
			},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			opts := installSandbox(t)
			target := c.target(opts)
			planHas(t, opts, UninstallFlags{}, target)
			afterPlan(t, func() { c.mutate(t, opts) })
			out, err := runUninstall(t, opts, UninstallFlags{})
			if err != nil {
				t.Fatalf("uninstall: %v\n%s", err, out)
			}
			if !strings.Contains(out, line(ActionSkip, target, reasonGone)) {
				t.Fatalf("falta «ya no existe» para %s:\n%s", target, out)
			}
			if !strings.Contains(out, "✓ pi-gomemory desinstalado. Para reinstalar") {
				t.Fatalf("un recurso desaparecido no cuenta para N:\n%s", out)
			}
		})
	}
}

// C-05: sin cambios el mensaje es el de siempre; con cambios refleja N, que no
// cuenta los desaparecidos.
func TestHCD034C05(t *testing.T) {
	opts := installSandbox(t)
	fakeTerminal(t, false, "")
	out, err := runUninstall(t, opts, UninstallFlags{Yes: true})
	if err != nil || !strings.Contains(out, "✓ pi-gomemory desinstalado. Para reinstalar: pi-gomemory-setup install\n") {
		t.Fatalf("sin cambios: err=%v\n%s", err, out)
	}

	opts = installSandbox(t)
	ext := filepath.Join(opts.PiAgentDir, "extensions")
	afterPlan(t, func() {
		writeFile(t, filepath.Join(ext, "gomemory-status.ts"), "EDIT")
		if err := os.Remove(filepath.Join(opts.PiAgentDir, "themes", "gomemory.json")); err != nil {
			t.Fatal(err)
		}
	})
	out, err = runUninstall(t, opts, UninstallFlags{})
	if err != nil {
		t.Fatalf("con cambios: %v\n%s", err, out)
	}
	if !strings.Contains(out, "✓ pi-gomemory desinstalado; se conservaron 1 elementos que cambiaron después del plan\n") {
		t.Fatalf("N debe contar solo los cambiados:\n%s", out)
	}
	if !strings.Contains(out, "Resumen: ") || !strings.Contains(out, " 2 omitidos") {
		t.Fatalf("la línea Resumen cuenta los dos omitidos:\n%s", out)
	}
}

// C-06: cada inspección de la lista cerrada que falla por algo distinto de la
// ausencia da [fallido], también ejecutando como root.
func TestHCD034C06(t *testing.T) {
	type tc struct {
		flags UninstallFlags
		setup func(t *testing.T, o *Options) string // devuelve el target fallido
		min   int
	}
	cases := map[string]tc{
		"ReadFile extensión": {setup: func(t *testing.T, o *Options) string {
			p := filepath.Join(o.PiAgentDir, "extensions", "gomemory-status.ts")
			mkdir(t, p)
			return p
		}},
		"ReadFile tema": {setup: func(t *testing.T, o *Options) string {
			p := filepath.Join(o.PiAgentDir, "themes", "gomemory.json")
			mkdir(t, p)
			return p
		}},
		"ReadFile launcher respaldado": {setup: func(t *testing.T, o *Options) string {
			p := filepath.Join(o.LocalBinDir, "pi-gomemory")
			mkdir(t, p)
			return p
		}},
		"ReadDir carpeta de skills": {setup: func(t *testing.T, o *Options) string {
			p := SkillRepoDir(o.HomeDir)
			selfLoop(t, p)
			return p
		}},
		"Readlink de una entrada": {setup: func(t *testing.T, o *Options) string {
			p := filepath.Join(SkillRepoDir(o.HomeDir), "enlace")
			mkdir(t, filepath.Dir(p))
			if err := os.Symlink(o.HomeDir, p); err != nil {
				t.Fatal(err)
			}
			old := readLink
			readLink = func(path string) (string, error) {
				if path == p {
					return "", &fs.PathError{Op: "readlink", Path: path, Err: syscall.EIO}
				}
				return old(path)
			}
			t.Cleanup(func() { readLink = old })
			return p
		}},
		"ReadFile de la copia embebida": {setup: func(t *testing.T, o *Options) string {
			p := filepath.Join(SkillRepoDir(o.HomeDir), "copia")
			writeFile(t, filepath.Join(p, skillMarkerFile), "x")
			mkdir(t, filepath.Join(p, "SKILL.md"))
			return p
		}},
		"marca de la skill": {setup: func(t *testing.T, o *Options) string {
			p := filepath.Join(SkillRepoDir(o.HomeDir), "marcada")
			writeFile(t, filepath.Join(p, "SKILL.md"), "x")
			selfLoop(t, filepath.Join(p, skillMarkerFile))
			return p
		}},
		"Stat mcp.json": {setup: func(t *testing.T, o *Options) string {
			p := filepath.Join(o.PiAgentDir, "mcp.json")
			selfLoop(t, p)
			return p
		}},
		"Stat settings.json": {setup: func(t *testing.T, o *Options) string {
			p := filepath.Join(o.PiAgentDir, "settings.json")
			selfLoop(t, p)
			return p
		}},
		"ReadFile AGENTS.md": {setup: func(t *testing.T, o *Options) string {
			p := filepath.Join(o.PiAgentDir, "AGENTS.md")
			mkdir(t, p)
			return p
		}},
		"Lstat caché": {setup: func(t *testing.T, o *Options) string {
			selfLoop(t, o.PiAgentDir)
			return DefaultDepsCachePath(o.PiAgentDir)
		}},
		"Lstat manifiesto": {setup: func(t *testing.T, o *Options) string {
			selfLoop(t, filepath.Dir(ManifestPath(o.HomeDir)))
			return ManifestPath(o.HomeDir)
		}},
		"settings.json con --packages": {flags: UninstallFlags{Packages: true}, min: 2, setup: func(t *testing.T, o *Options) string {
			p := filepath.Join(o.PiAgentDir, "settings.json")
			mkdir(t, p)
			return p
		}},
		"pi-cc-extensions.json con --packages": {flags: UninstallFlags{Packages: true}, setup: func(t *testing.T, o *Options) string {
			p := filepath.Join(o.PiAgentDir, "pi-cc-extensions.json")
			mkdir(t, p)
			return p
		}},
		"os.Executable con --self": {flags: UninstallFlags{Self: true}, setup: func(t *testing.T, o *Options) string {
			old := selfExecutable
			selfExecutable = func() (string, error) { return "", errors.New("sin ejecutable") }
			t.Cleanup(func() { selfExecutable = old })
			return "pi-gomemory-setup (--self)"
		}},
		"Lstat con --self": {flags: UninstallFlags{Self: true}, setup: func(t *testing.T, o *Options) string {
			selfLoop(t, o.LocalBinDir)
			return filepath.Join(o.LocalBinDir, "pi-gomemory-setup")
		}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			for _, mode := range []UninstallFlags{{DryRun: true}, {Yes: true}} {
				opts := sandbox(t)
				t.Setenv("PI_GOMEMORY_BIN_DIR", "")
				fakeTerminal(t, false, "")
				target := c.setup(t, opts)
				flags := c.flags
				flags.DryRun, flags.Yes = mode.DryRun, mode.Yes
				out, err := runUninstall(t, opts, flags)
				if err == nil {
					t.Fatalf("dry-run=%v: debía terminar con error:\n%s", mode.DryRun, out)
				}
				min := c.min
				if min == 0 {
					min = 1
				}
				if n := strings.Count(out, "  ["+ActionFail+"] "+target+" — "); n < min {
					t.Fatalf("dry-run=%v: [fallido] %s aparece %d veces (mín. %d):\n%s", mode.DryRun, target, n, min, out)
				}
				if strings.Contains(out, "No queda nada que retirar") {
					t.Fatalf("dry-run=%v: anunció que no queda nada:\n%s", mode.DryRun, out)
				}
			}
		})
	}
}

// C-07: un launcher ilegible en una carpeta candidata no respaldada se omite
// y se informa aunque no quede nada que retirar.
func TestHCD034C07(t *testing.T) {
	opts := sandbox(t)
	opts.LocalBinDir = filepath.Join(opts.HomeDir, "bin-propio")
	t.Setenv("PI_GOMEMORY_BIN_DIR", "")
	fakeTerminal(t, false, "")
	p := filepath.Join(opts.HomeDir, ".local", "bin", "pi-gomemory")
	mkdir(t, p)
	out, err := runUninstall(t, opts, UninstallFlags{Yes: true})
	if err != nil {
		t.Fatalf("una carpeta candidata no debe hacer fallar: %v\n%s", err, out)
	}
	notice := "  [" + ActionSkip + "] " + p + " — " + reasonUninspectable
	i, j := strings.Index(out, notice), strings.Index(out, "✓ No queda nada que retirar de pi-gomemory.")
	if i < 0 || j < 0 || i > j {
		t.Fatalf("falta la omisión antes de «No queda nada»:\n%s", out)
	}
}

// C-08: un HOME sin nada de pi-gomemory sigue diciendo que no queda nada; un
// archivo donde se espera una carpeta cuenta como ausencia.
func TestHCD034C08(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T, o *Options){
		"vacío":                       func(*testing.T, *Options) {},
		"archivo en lugar de carpeta": func(t *testing.T, o *Options) { writeFile(t, filepath.Join(o.PiAgentDir, "skills"), "x") },
	} {
		t.Run(name, func(t *testing.T) {
			opts := sandbox(t)
			fakeTerminal(t, false, "")
			setup(t, opts)
			out, err := runUninstall(t, opts, UninstallFlags{Yes: true})
			if err != nil || out != "✓ No queda nada que retirar de pi-gomemory.\n" {
				t.Fatalf("err=%v\n%s", err, out)
			}
		})
	}
}

// C-11: un error de relectura al aplicar distinto de la ausencia es [fallido];
// con el tema ilegible, la clave theme se conserva.
func TestHCD034C11(t *testing.T) {
	t.Run("relectura", func(t *testing.T) {
		opts := installSandbox(t)
		mcp := filepath.Join(opts.PiAgentDir, "mcp.json")
		target := mcp + " › mcpServers.gomemory"
		planHas(t, opts, UninstallFlags{}, target)
		afterPlan(t, func() {
			if err := os.Remove(mcp); err != nil {
				t.Fatal(err)
			}
			mkdir(t, mcp)
		})
		out, err := runUninstall(t, opts, UninstallFlags{})
		if err == nil {
			t.Fatalf("debía terminar con error:\n%s", out)
		}
		if !strings.Contains(out, "  ["+ActionFail+"] "+target+" — ") {
			t.Fatalf("falta [fallido] %s:\n%s", target, out)
		}
	})
	t.Run("tema ilegible", func(t *testing.T) {
		opts := installSandbox(t)
		fakeTerminal(t, false, "")
		theme := filepath.Join(opts.PiAgentDir, "themes", "gomemory.json")
		if err := os.Remove(theme); err != nil {
			t.Fatal(err)
		}
		mkdir(t, theme)
		out, err := runUninstall(t, opts, UninstallFlags{Yes: true})
		if err == nil {
			t.Fatalf("debía terminar con error:\n%s", out)
		}
		if !strings.Contains(readFile(t, filepath.Join(opts.PiAgentDir, "settings.json")), `"gomemory"`) {
			t.Fatalf("se retiró theme con el tema ilegible:\n%s", out)
		}
	})
}

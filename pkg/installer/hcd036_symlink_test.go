package installer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const hcd036BrokenLink = "el enlace no apunta a un archivo"

// linkConfig convierte path en un enlace a un archivo de otra carpeta (los
// «dotfiles») con content y mode, y devuelve la ruta del destino.
func linkConfig(t *testing.T, path, content string, mode os.FileMode) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), "dotfiles", filepath.Base(path))
	writeFile(t, target, content)
	if err := os.Chmod(target, mode); err != nil {
		t.Fatal(err)
	}
	mkdir(t, filepath.Dir(path))
	_ = os.Remove(path)
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	return target
}

// moveToLink mueve un archivo instalado a otra carpeta y deja un enlace en su lugar.
func moveToLink(t *testing.T, path string) string {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return linkConfig(t, path, readFile(t, path), info.Mode().Perm())
}

func assertLink(t *testing.T, path, target string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s dejó de ser un enlace", path)
	}
	if target == "" {
		return
	}
	got, err := os.Readlink(path)
	if err != nil || got != target {
		t.Fatalf("%s apunta a %q (%v), quiero %q", path, got, err, target)
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s tiene modo %o, quiero %o", path, got, want)
	}
}

func jsonKeys(t *testing.T, path string) map[string]json.RawMessage {
	t.Helper()
	obj := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(readFile(t, path)), &obj); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return obj
}

// brokenLinkKinds prepara path como enlace roto, bucle o enlace a directorio.
var brokenLinkKinds = map[string]func(t *testing.T, path string){
	"roto": func(t *testing.T, path string) {
		mkdir(t, filepath.Dir(path))
		_ = os.Remove(path)
		if err := os.Symlink(filepath.Join(t.TempDir(), "no-existe.json"), path); err != nil {
			t.Fatal(err)
		}
	},
	"bucle": func(t *testing.T, path string) {
		_ = os.Remove(path)
		selfLoop(t, path)
	},
	"directorio": func(t *testing.T, path string) {
		mkdir(t, filepath.Dir(path))
		_ = os.Remove(path)
		if err := os.Symlink(t.TempDir(), path); err != nil {
			t.Fatal(err)
		}
	},
}

// C-01: mcp.json enlazado recibe los servidores en el destino, conservando
// el enlace, los servidores ajenos y los permisos.
func TestHCD036C01(t *testing.T) {
	opts := sandbox(t)
	path := filepath.Join(opts.PiAgentDir, "mcp.json")
	target := linkConfig(t, path, `{"mcpServers":{"ajeno":{"command":"x"}}}`, 0640)
	if err := InstallMCPServers(opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	assertLink(t, path, target)
	servers := map[string]json.RawMessage{}
	_ = json.Unmarshal(jsonKeys(t, target)["mcpServers"], &servers)
	for _, name := range []string{"ajeno", "gomemory", "codegraph", "codebase-memory"} {
		if _, ok := servers[name]; !ok {
			t.Fatalf("C-01: falta %s en el destino: %s", name, readFile(t, target))
		}
	}
	assertMode(t, target, 0640)
}

// C-02: settings.json enlazado recibe theme, packages y skills en el destino.
func TestHCD036C02(t *testing.T) {
	opts := sandbox(t)
	path := filepath.Join(opts.PiAgentDir, "settings.json")
	target := linkConfig(t, path, `{"propia":true}`, 0600)
	if err := InstallTheme(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	if err := EnsurePinnedPackages(opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	if err := RegisterSkillRepo(opts.PiAgentDir, opts.HomeDir); err != nil {
		t.Fatal(err)
	}
	assertLink(t, path, target)
	obj := jsonKeys(t, target)
	for _, k := range []string{"propia", "theme", "packages", "skills"} {
		if _, ok := obj[k]; !ok {
			t.Fatalf("C-02: falta %s en el destino: %s", k, readFile(t, target))
		}
	}
}

// C-03: AGENTS.md enlazado recibe el bloque en el destino con su contenido y modo.
func TestHCD036C03(t *testing.T) {
	opts := sandbox(t)
	path := filepath.Join(opts.PiAgentDir, "AGENTS.md")
	target := linkConfig(t, path, "# Mis reglas\n", 0640)
	if err := InstallInstructions(opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	assertLink(t, path, target)
	got := readFile(t, target)
	if !strings.Contains(got, "# Mis reglas") || !strings.Contains(got, "Protocolo para Pi") {
		t.Fatalf("C-03: el destino no tiene el contenido propio y el bloque: %q", got)
	}
	assertMode(t, target, 0640)
}

// C-04: pi-cc-extensions.json enlazado recibe las claves por defecto en el destino.
func TestHCD036C04(t *testing.T) {
	opts := sandbox(t)
	path := filepath.Join(opts.PiAgentDir, "pi-cc-extensions.json")
	target := linkConfig(t, path, `{"miClave":true}`, 0644)
	if err := InstallCCExtensions(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	assertLink(t, path, target)
	obj := jsonKeys(t, target)
	if _, ok := obj["miClave"]; !ok {
		t.Fatalf("C-04: se perdió la clave propia: %s", readFile(t, target))
	}
	if _, ok := obj["stickyUserMessage"]; !ok {
		t.Fatalf("C-04: faltan las claves por defecto: %s", readFile(t, target))
	}
}

// C-05: con un enlace relativo hacia otro enlace, el resultado llega al
// destino final, ambos enlaces se conservan y no quedan temporales.
func TestHCD036C05(t *testing.T) {
	opts := sandbox(t)
	root := t.TempDir()
	real := filepath.Join(root, "real", "mcp.json")
	writeFile(t, real, `{}`)
	mid := filepath.Join(root, "enlaces", "mcp.json")
	mkdir(t, filepath.Dir(mid))
	if err := os.Symlink(filepath.Join("..", "real", "mcp.json"), mid); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(opts.PiAgentDir, "mcp.json")
	mkdir(t, opts.PiAgentDir)
	rel, err := filepath.Rel(opts.PiAgentDir, mid)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(rel, path); err != nil {
		t.Fatal(err)
	}
	if err := InstallMCPServers(opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	assertLink(t, path, rel)
	assertLink(t, mid, filepath.Join("..", "real", "mcp.json"))
	if !strings.Contains(readFile(t, real), "gomemory") {
		t.Fatalf("C-05: el destino final no recibió los servidores: %s", readFile(t, real))
	}
	noTemps(t, opts.PiAgentDir)
	noTemps(t, root)
}

// C-07: si la carpeta del destino no admite escritura, falla sin tocar nada.
func TestHCD036C07(t *testing.T) {
	skipIfRoot(t)
	opts := sandbox(t)
	path := filepath.Join(opts.PiAgentDir, "mcp.json")
	target := linkConfig(t, path, `{"mcpServers":{}}`, 0644)
	chmodFor(t, filepath.Dir(target), 0555)
	if err := InstallMCPServers(opts.PiAgentDir); err == nil {
		t.Fatal("C-07: debía fallar con la carpeta del destino sin escritura")
	}
	assertLink(t, path, target)
	if got := readFile(t, target); got != `{"mcpServers":{}}` {
		t.Fatalf("C-07: el destino cambió: %q", got)
	}
}

// C-06: un enlace que no apunta a un archivo aborta install sin tocar nada.
func TestHCD036C06(t *testing.T) {
	for _, file := range []string{"mcp.json", "settings.json", "AGENTS.md", "pi-cc-extensions.json"} {
		for kind, prepare := range brokenLinkKinds {
			t.Run(file+"/"+kind, func(t *testing.T) {
				opts := sandbox(t)
				path := filepath.Join(opts.PiAgentDir, file)
				prepare(t, path)
				before, _ := os.Readlink(path)
				err := InstallAll(repoAssets(t), opts)
				if err == nil || !strings.Contains(err.Error(), hcd036BrokenLink) || !strings.Contains(err.Error(), path) {
					t.Fatalf("C-06: install debía abortar con el enlace roto de %s: %v", path, err)
				}
				assertLink(t, path, before)
				if kind == "roto" {
					if _, err := os.Lstat(before); !os.IsNotExist(err) {
						t.Fatalf("C-06: se creó el destino del enlace roto: %v", err)
					}
				}
			})
		}
	}
}

// C-15: plugin no ejecuta pi install si settings.json o pi-cc-extensions.json
// es un enlace que no apunta a un archivo.
func TestHCD036C15(t *testing.T) {
	cases := map[string]struct {
		file string
		run  func(t *testing.T, o *Options) error
	}{
		"antigravity con settings.json roto": {"settings.json", func(t *testing.T, o *Options) error { return InstallAntigravityPlugin(o.PiAgentDir) }},
		"cc con settings.json roto":          {"settings.json", func(t *testing.T, o *Options) error { return InstallCCExtensions(repoAssets(t), o.PiAgentDir) }},
		"cc con pi-cc-extensions.json roto":  {"pi-cc-extensions.json", func(t *testing.T, o *Options) error { return InstallCCExtensions(repoAssets(t), o.PiAgentDir) }},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			opts := sandbox(t)
			marker := filepath.Join(t.TempDir(), "pi-llamado")
			bin := t.TempDir()
			writeFile(t, filepath.Join(bin, "pi"), "#!/bin/sh\necho llamado >> "+marker+"\n")
			if err := os.Chmod(filepath.Join(bin, "pi"), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin)
			path := filepath.Join(opts.PiAgentDir, c.file)
			brokenLinkKinds["roto"](t, path)
			before, _ := os.Readlink(path)
			err := c.run(t, opts)
			if err == nil || !strings.Contains(err.Error(), hcd036BrokenLink) {
				t.Fatalf("C-15: debía fallar con el enlace roto: %v", err)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatal("C-15: se ejecutó pi install con un enlace roto")
			}
			assertLink(t, path, before)
		})
	}
}

// C-08: uninstall retira las entradas de pi-gomemory en los destinos de
// mcp.json y settings.json sin romper los enlaces.
func TestHCD036C08(t *testing.T) {
	opts := installSandbox(t)
	mcp := filepath.Join(opts.PiAgentDir, "mcp.json")
	settings := filepath.Join(opts.PiAgentDir, "settings.json")
	mcpT, setT := moveToLink(t, mcp), moveToLink(t, settings)
	out, err := runUninstall(t, opts, UninstallFlags{Yes: true})
	if err != nil {
		t.Fatalf("uninstall: %v\n%s", err, out)
	}
	assertLink(t, mcp, mcpT)
	assertLink(t, settings, setT)
	if strings.Contains(readFile(t, mcpT), `"gomemory"`) {
		t.Fatalf("C-08: el destino de mcp.json conserva gomemory: %s", readFile(t, mcpT))
	}
	s := readFile(t, setT)
	if strings.Contains(s, `"theme": "gomemory"`) || strings.Contains(s, SkillRepoSetting) {
		t.Fatalf("C-08: el destino de settings.json conserva lo de pi-gomemory: %s", s)
	}
}

// C-09: AGENTS.md enlazado que queda vacío se vacía en el destino y el enlace se conserva.
func TestHCD036C09(t *testing.T) {
	opts := installSandbox(t)
	path := filepath.Join(opts.PiAgentDir, "AGENTS.md")
	target := moveToLink(t, path)
	out, err := runUninstall(t, opts, UninstallFlags{Yes: true})
	if err != nil {
		t.Fatalf("uninstall: %v\n%s", err, out)
	}
	assertLink(t, path, target)
	if got := readFile(t, target); strings.TrimSpace(got) != "" {
		t.Fatalf("C-09: el destino debía quedar vacío: %q", got)
	}
}

// C-10: pi-cc-extensions.json enlazado se omite con aviso con --packages,
// apunte a los valores por defecto o a una configuración propia.
func TestHCD036C10(t *testing.T) {
	for name, custom := range map[string]bool{"por defecto": false, "personalizada": true} {
		t.Run(name, func(t *testing.T) {
			opts := installSandbox(t)
			path := filepath.Join(opts.PiAgentDir, "pi-cc-extensions.json")
			if custom {
				editJSONFile(t, path, func(o map[string]any) { o["miClave"] = true })
			}
			target := moveToLink(t, path)
			before := readFile(t, target)
			out, err := runUninstall(t, opts, UninstallFlags{Yes: true, Packages: true})
			if err != nil {
				t.Fatalf("uninstall: %v\n%s", err, out)
			}
			if !strings.Contains(out, "  ["+ActionSkip+"] "+path+" — ") || !strings.Contains(out, "enlace") {
				t.Fatalf("C-10: falta el aviso de pi-cc-extensions.json enlazado:\n%s", out)
			}
			assertLink(t, path, target)
			if got := readFile(t, target); got != before {
				t.Fatalf("C-10: el destino cambió: %q", got)
			}
		})
	}
}

// C-11: en uninstall, un enlace que no apunta a un archivo es [fallido], no
// se modifica y conserva el manifiesto.
func TestHCD036C11(t *testing.T) {
	for _, file := range []string{"mcp.json", "settings.json", "AGENTS.md"} {
		for kind, prepare := range brokenLinkKinds {
			t.Run(file+"/"+kind, func(t *testing.T) {
				opts := installSandbox(t)
				path := filepath.Join(opts.PiAgentDir, file)
				prepare(t, path)
				before, _ := os.Readlink(path)
				out, err := runUninstall(t, opts, UninstallFlags{Yes: true})
				if err == nil {
					t.Fatalf("C-11: debía terminar con error:\n%s", out)
				}
				if !strings.Contains(out, "  ["+ActionFail+"] "+path+" — ") || !strings.Contains(out, hcd036BrokenLink) {
					t.Fatalf("C-11: falta [fallido] con el motivo de enlace:\n%s", out)
				}
				assertLink(t, path, before)
				if !manifestExists(t, opts.HomeDir) {
					t.Fatal("C-11: el manifiesto debía conservarse")
				}
			})
		}
	}
}

// C-12: pi-cc-extensions.json roto solo cuenta como fallo con --packages.
func TestHCD036C12(t *testing.T) {
	opts := installSandbox(t)
	path := filepath.Join(opts.PiAgentDir, "pi-cc-extensions.json")
	brokenLinkKinds["roto"](t, path)
	out, err := runUninstall(t, opts, UninstallFlags{DryRun: true})
	if err != nil || strings.Contains(out, "["+ActionFail+"] "+path) {
		t.Fatalf("C-12: sin --packages no debe fallar por %s: %v\n%s", path, err, out)
	}
	out, err = runUninstall(t, opts, UninstallFlags{DryRun: true, Packages: true})
	if err == nil || !strings.Contains(out, "  ["+ActionFail+"] "+path+" — ") || !strings.Contains(out, hcd036BrokenLink) {
		t.Fatalf("C-12: con --packages debe ser [fallido]: %v\n%s", err, out)
	}
}

// C-16: un enlace que se rompe entre el plan y la aplicación cuenta como fallo.
func TestHCD036C16(t *testing.T) {
	for _, file := range []string{"mcp.json", "settings.json", "AGENTS.md"} {
		t.Run(file, func(t *testing.T) {
			opts := installSandbox(t)
			path := filepath.Join(opts.PiAgentDir, file)
			target := moveToLink(t, path)
			afterPlan(t, func() {
				if err := os.Remove(target); err != nil {
					t.Fatal(err)
				}
			})
			out, err := runUninstall(t, opts, UninstallFlags{})
			if err == nil || !strings.Contains(out, "["+ActionFail+"] "+path) {
				t.Fatalf("C-16: el enlace roto al aplicar debía ser [fallido]: %v\n%s", err, out)
			}
			assertLink(t, path, target)
		})
	}
}

// C-14 (guarda sin rojo): el tema, fuera de la política, se sustituye como hoy.
func TestHCD036C14(t *testing.T) {
	opts := sandbox(t)
	theme := filepath.Join(opts.PiAgentDir, "themes", "gomemory.json")
	linkConfig(t, theme, `{}`, 0644)
	if err := InstallTheme(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(theme)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("C-14: el tema debe quedar como archivo normal, como hoy: %v %v", info, err)
	}
}

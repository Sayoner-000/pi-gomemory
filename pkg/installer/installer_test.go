package installer

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// repoAssets reproduce el árbol que main.go embebe (skills, extensions, themes, config).
func repoAssets(t *testing.T) fs.FS {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return os.DirFS(root)
}

// sandbox prepara un HOME aislado, sin `pi` en PATH, y devuelve sus opciones.
func sandbox(t *testing.T) *Options {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", t.TempDir())
	return &Options{
		HomeDir:     home,
		PiAgentDir:  filepath.Join(home, ".pi", "agent"),
		LocalBinDir: filepath.Join(home, ".local", "bin"),
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// D2: los servidores ajenos conservan campos que el struct no conoce.
func TestInstallMCPServersPreservesForeignServers(t *testing.T) {
	opts := sandbox(t)
	mcpPath := filepath.Join(opts.PiAgentDir, "mcp.json")
	foreign := `{"type":"http","url":"https://example.test/mcp","headers":{"X-Key":"k"}}`
	writeFile(t, mcpPath, `{"settings":{"timeout":5},"mcpServers":{"remote":`+foreign+`}}`)

	if err := InstallMCPServers(opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}

	var got struct {
		Settings   json.RawMessage            `json:"settings"`
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(readFile(t, mcpPath)), &got); err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	_ = json.Compact(&compact, got.MCPServers["remote"])
	if compact.String() != foreign {
		t.Errorf("servidor ajeno alterado:\n got %s\nwant %s", compact.String(), foreign)
	}
	if len(got.Settings) == 0 {
		t.Error("se perdió la clave de nivel superior 'settings'")
	}
	for _, name := range []string{"gomemory", "codegraph", "codebase-memory"} {
		if _, ok := got.MCPServers[name]; !ok {
			t.Errorf("falta el servidor %s", name)
		}
	}
}

// D2: un mcp.json ilegible no se sobrescribe.
func TestInstallMCPServersRejectsInvalidJSON(t *testing.T) {
	opts := sandbox(t)
	mcpPath := filepath.Join(opts.PiAgentDir, "mcp.json")
	writeFile(t, mcpPath, `{"mcpServers": {`)

	if err := InstallMCPServers(opts.PiAgentDir); err == nil {
		t.Fatal("se esperaba error con JSON inválido")
	}
	if got := readFile(t, mcpPath); got != `{"mcpServers": {` {
		t.Errorf("el archivo inválido fue modificado: %q", got)
	}
}

// D3: AGENTS.md conserva lo ajeno, es idempotente y migra el formato sin marcadores.
func TestInstallInstructionsPreservesUserContent(t *testing.T) {
	cases := map[string]string{
		"vacío":            "",
		"contenido propio": "# Mis reglas\n\nNo tocar.\n",
		"legado v1.0.0":    "# Mis reglas\n\n" + GomemoryPiInstructions + "\nPie propio.\n",
	}
	for name, initial := range cases {
		t.Run(name, func(t *testing.T) {
			opts := sandbox(t)
			path := filepath.Join(opts.PiAgentDir, "AGENTS.md")
			if initial != "" {
				writeFile(t, path, initial)
			}

			if err := InstallInstructions(opts.PiAgentDir); err != nil {
				t.Fatal(err)
			}
			first := readFile(t, path)
			if err := InstallInstructions(opts.PiAgentDir); err != nil {
				t.Fatal(err)
			}
			second := readFile(t, path)

			if first != second {
				t.Errorf("no es idempotente:\n1ª %q\n2ª %q", first, second)
			}
			if n := strings.Count(second, "Protocolo para Pi"); n != 1 {
				t.Errorf("el protocolo aparece %d veces", n)
			}
			for _, keep := range []string{"# Mis reglas", "No tocar.", "Pie propio."} {
				if strings.Contains(initial, keep) && !strings.Contains(second, keep) {
					t.Errorf("se perdió contenido propio %q", keep)
				}
			}
		})
	}
}

// D4 + D7: la instalación completa no escribe en el cwd y copia los recursos embebidos tal cual.
func TestInstallAllUsesEmbeddedAssetsAndLeavesCwdClean(t *testing.T) {
	assets := repoAssets(t)
	opts := sandbox(t)
	cwd := t.TempDir()
	t.Chdir(cwd)

	ccPath := filepath.Join(opts.PiAgentDir, "pi-cc-extensions.json")
	writeFile(t, ccPath, `{"diffViewMode":"split","miClave":true}`)

	if err := InstallAll(assets, opts); err != nil {
		t.Fatal(err)
	}

	if entries, _ := os.ReadDir(cwd); len(entries) != 0 {
		t.Errorf("InstallAll escribió en el cwd: %v", entries)
	}

	theme, _ := fs.ReadFile(assets, "themes/gomemory.json")
	if got := readFile(t, filepath.Join(opts.PiAgentDir, "themes", "gomemory.json")); got != string(theme) {
		t.Error("el tema instalado difiere del embebido")
	}

	var cc map[string]any
	if err := json.Unmarshal([]byte(readFile(t, ccPath)), &cc); err != nil {
		t.Fatal(err)
	}
	if cc["diffViewMode"] != "split" || cc["miClave"] != true {
		t.Errorf("se pisaron claves propias de pi-cc-extensions: %v", cc)
	}
	if cc["stickyUserMessage"] != true {
		t.Error("no se añadieron los valores por defecto embebidos")
	}

	if info, err := os.Stat(filepath.Join(opts.LocalBinDir, "pi-gomemory")); err != nil || info.Mode()&0111 == 0 {
		t.Errorf("launcher ausente o no ejecutable: %v", err)
	}
	skills, _ := os.ReadDir(SkillRepoDir(opts.HomeDir))
	if len(skills) != 21 {
		t.Errorf("skills instaladas = %d, se esperan 21", len(skills))
	}
}

// D5: sin skills embebidas se falla; no se busca en carpetas de otros agentes.
func TestInstallSkillsFailsWithoutEmbeddedSkills(t *testing.T) {
	opts := sandbox(t)
	other := filepath.Join(opts.HomeDir, ".config", "opencode", "skills", "x", "SKILL.md")
	writeFile(t, other, "---\ndescription: a: b\n---\n")

	if _, err := InstallSkills(fstest.MapFS{}, opts); err == nil {
		t.Fatal("se esperaba error sin skills embebidas")
	}
	if got := readFile(t, other); got != "---\ndescription: a: b\n---\n" {
		t.Errorf("se modificó una skill de otro agente: %q", got)
	}
}

func TestSplashResourcesAreInstalledFromEmbeddedAssets(t *testing.T) {
	opts := sandbox(t)
	assets := repoAssets(t)
	if err := InstallTUIExtension(assets, opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	for source, destination := range map[string]string{
		"extensions/gomemory-splash.ts": "gomemory-splash.ts",
		"config/splash.md":              "gomemory-splash-prompt.md",
	} {
		want, err := fs.ReadFile(assets, source)
		if err != nil {
			t.Fatal(err)
		}
		if got := readFile(t, filepath.Join(opts.PiAgentDir, "extensions", destination)); got != string(want) {
			t.Errorf("%s differs from embedded resource", destination)
		}
	}
}

// D1: si la API falla no se muestran cifras inventadas.
func TestFetchAndFormatUsageNeverFabricates(t *testing.T) {
	opts := sandbox(t)
	writeFile(t, filepath.Join(opts.PiAgentDir, "antigravity-accounts.json"),
		`{"activeAccountId":"a","accounts":{"a":{"access":"tok","email":"persona@example.com"}}}`)

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer failing.Close()

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"planLabel":"Plan X","groups":[{"displayName":"Grupo real","buckets":[{"displayName":"Semanal","remainingFraction":0.42}]}]}`))
	}))
	defer ok.Close()

	saved := endpoints
	t.Cleanup(func() { endpoints = saved })

	endpoints = []string{failing.URL}
	out, err := FetchAndFormatUsage(opts.PiAgentDir)
	if err == nil || out != "" {
		t.Fatalf("con la API caída: out=%q err=%v; se esperaba error sin salida", out, err)
	}

	endpoints = []string{failing.URL, ok.URL}
	out, err = FetchAndFormatUsage(opts.PiAgentDir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Grupo real") || !strings.Contains(out, "42.0%") {
		t.Errorf("salida sin los datos reales: %q", out)
	}
}

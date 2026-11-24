package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// HCD-076 VS-02: sync instala lo que falta y actualiza lo desactualizado.

// withSyncProbes sustituye las sondas que usa sync.
func withSyncProbes(t *testing.T, f func(*Options) []DepProbe) {
	t.Helper()
	old := syncProbes
	syncProbes = f
	t.Cleanup(func() { syncProbes = old })
}

// offlineProbes son las sondas de producción sin red: la última versión
// nunca se conoce, así que nada está desactualizado.
func offlineProbes(o *Options) []DepProbe {
	var out []DepProbe
	for _, p := range DefaultDepProbes("v1.0.0", o.PiAgentDir) {
		p.Latest = func(context.Context) (string, error) { return "", errors.New("sin red en pruebas") }
		p.Published = nil
		out = append(out, p)
	}
	return out
}

// withNode sustituye la lectura de la versión de Node.js y npm.
func withNode(t *testing.T, node string, npm bool) {
	t.Helper()
	old := nodeRuntime
	nodeRuntime = func() (string, bool) { return node, npm }
	t.Cleanup(func() { nodeRuntime = old })
}

// versionedTool escribe un binario que lee su versión de un archivo y que,
// con su subcomando de actualización, escribe newVer.
func versionedTool(t *testing.T, dir, name, ver, upd, newVer string) {
	t.Helper()
	vf := filepath.Join(dir, name+".ver")
	writeFile(t, vf, ver+"\n")
	script := "#!/bin/sh\nif [ \"$1\" = \"" + upd + "\" ]; then echo " + newVer + " > " + vf + "; exit 0; fi\ncat " + vf + "\n"
	writeFile(t, filepath.Join(dir, name), script)
	if err := os.Chmod(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
}

func probeFor(name, latest string, update ...string) DepProbe {
	return DepProbe{
		Name:    name,
		Current: versionFromCmd(name, "--version"),
		Latest:  func(context.Context) (string, error) { return latest, nil },
		Update:  update,
	}
}

func actions(res []SyncResult) map[string]string {
	m := map[string]string{}
	for _, r := range res {
		m[r.Name] = r.Action
	}
	return m
}

// T8 · C-04: una herramienta desactualizada se actualiza sin preguntar aunque
// la caché reciente diga lo contrario, y la caché se reescribe completa.
func TestHCD076SyncUpdatesOutdated(t *testing.T) {
	opts := toolSandbox(t)
	bin := opts.LocalBinDir
	versionedTool(t, bin, "codegraph", "1.0.0", "upgrade", "2.0.0")
	fakeTool(t, bin, "pi")
	srv, _ := scriptServer(t, map[string]string{})
	withToolHooks(t, testHooks(srv))
	withTools(t, []Tool{
		{Name: "pi", Required: true, Script: srv.URL + "/pi.sh", Interp: "sh", Manual: "m"},
		{Name: "codegraph", Script: srv.URL + "/cg.sh", Interp: "sh", Manual: "m"},
	})
	withSyncProbes(t, func(*Options) []DepProbe {
		return []DepProbe{
			{Name: "pi-gomemory", Current: func(context.Context) (string, error) { return "v1.0.0", nil }, Latest: func(context.Context) (string, error) { return "v1.1.0", nil }},
			probeFor("pi", "1.0.0", "pi", "update"),
			probeFor("codegraph", "2.0.0", "codegraph", "upgrade"),
		}
	})
	t.Setenv(updateCheckEnv, "1")
	cache := DefaultDepsCachePath(opts.PiAgentDir)
	fresh, _ := json.Marshal(depsCache{CheckedAt: time.Now(), Deps: []DepState{{Name: "codegraph", Current: "2.0.0", Latest: "2.0.0"}}})
	writeFile(t, cache, string(fresh))

	var out bytes.Buffer
	res, err := Sync(opts, SyncOptions{Yes: true}, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("Sync: %v\n%s", err, out.String())
	}
	if a := actions(res); a["codegraph"] != "updated" || a["pi"] != "current" {
		t.Fatalf("acciones = %v\n%s", a, out.String())
	}
	data, _ := os.ReadFile(filepath.Join(bin, "codegraph.ver"))
	if strings.TrimSpace(string(data)) != "2.0.0" {
		t.Fatalf("codegraph no se actualizó: %q", data)
	}
	var c depsCache
	raw, _ := os.ReadFile(cache)
	_ = json.Unmarshal(raw, &c)
	got := map[string]DepState{}
	for _, d := range c.Deps {
		got[d.Name] = d
	}
	if len(c.Deps) != 3 || got["codegraph"].Current != "2.0.0" || got["pi-gomemory"].Latest != "v1.1.0" {
		t.Fatalf("caché tras sync: %s", raw)
	}
}

// T9 · C-06: solo es fatal no poder instalar un obligatorio que falta.
func TestHCD076SyncFailurePolicy(t *testing.T) {
	opts := toolSandbox(t)
	bin := opts.LocalBinDir
	versionedTool(t, bin, "pi", "1.0.0", "update", "1.0.0")
	writeFile(t, filepath.Join(bin, "pi"), "#!/bin/sh\nif [ \"$1\" = update ]; then exit 4; fi\necho 1.0.0\n")
	srv, _ := scriptServer(t, map[string]string{"/fail.sh": "exit 3\n"})
	withToolHooks(t, testHooks(srv))
	withSyncProbes(t, func(*Options) []DepProbe {
		return []DepProbe{probeFor("pi", "2.0.0", "pi", "update"), probeFor("herdr", "1.0.0", "herdr", "update"), probeFor("mem", "1.0.0")}
	})
	withTools(t, []Tool{
		{Name: "pi", Required: true, Script: srv.URL + "/fail.sh", Interp: "sh", Manual: "manual-pi"},
		{Name: "herdr", Script: srv.URL + "/fail.sh", Interp: "sh", Manual: "manual-herdr"},
	})
	var out bytes.Buffer
	res, err := Sync(opts, SyncOptions{Yes: true}, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("fallar al actualizar pi presente u opcional no es fatal: %v\n%s", err, out.String())
	}
	if a := actions(res); a["pi"] != "failed" || a["herdr"] != "failed" {
		t.Fatalf("acciones = %v", a)
	}
	if !strings.Contains(out.String(), "manual-herdr") || !strings.Contains(out.String(), "manual-pi") {
		t.Fatalf("faltan los comandos manuales:\n%s", out.String())
	}
	withTools(t, []Tool{{Name: "mem", Required: true, Script: srv.URL + "/fail.sh", Interp: "bash", Manual: "manual-mem"}})
	_, err = Sync(opts, SyncOptions{Yes: true}, strings.NewReader(""), &out)
	if err == nil || !strings.Contains(err.Error(), "manual-mem") {
		t.Fatalf("mem obligatorio ausente que no se instala debe ser fatal: %v", err)
	}
}

// T10 · C-03: sin Node.js ≥ 22.19 y npm no se ejecuta el instalador de pi.
func TestHCD076SyncNodeCheck(t *testing.T) {
	opts := toolSandbox(t)
	srv, hits := scriptServer(t, map[string]string{"/pi.sh": "mkdir -p \"$PI_CODING_AGENT_DIR/bin\" && printf '#!/bin/sh\\necho 1.0.0\\n' > \"$PI_CODING_AGENT_DIR/bin/pi\" && chmod +x \"$PI_CODING_AGENT_DIR/bin/pi\"\n"})
	withToolHooks(t, testHooks(srv))
	withSyncProbes(t, func(*Options) []DepProbe { return []DepProbe{probeFor("pi", "1.0.0")} })
	withTools(t, []Tool{{Name: "pi", Required: true, Script: srv.URL + "/pi.sh", Interp: "sh", Env: piEnv, Manual: "manual-pi"}})
	for _, c := range []struct {
		node string
		npm  bool
	}{{"", false}, {"v20.11.0", true}, {"v22.19.0", false}} {
		withNode(t, c.node, c.npm)
		_, err := Sync(opts, SyncOptions{Yes: true}, strings.NewReader(""), &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "Node.js 22.19") || len(*hits) != 0 {
			t.Fatalf("node=%q npm=%v: %v, peticiones %v", c.node, c.npm, err, *hits)
		}
	}
	withNode(t, "v22.19.0", true)
	if _, err := Sync(opts, SyncOptions{Yes: true}, strings.NewReader(""), &bytes.Buffer{}); err != nil || len(*hits) != 1 {
		t.Fatalf("con Node 22.19 y npm debe instalarse pi: %v %v", err, *hits)
	}
}

// T12 · C-12: `sync --json` escribe en stdout una fila por tool y el progreso
// en stderr; un uso incorrecto termina con código 2.
func TestHCD076SyncCommandJSON(t *testing.T) {
	opts := toolSandbox(t)
	fakeTool(t, opts.LocalBinDir, "pi")
	withTools(t, []Tool{
		{Name: "pi", Required: true, Script: "https://invalid.example/pi.sh", Interp: "sh", Manual: "m"},
		{Name: "herdr", Script: "https://invalid.example/h.sh", Interp: "sh", Manual: "m"},
	})
	var stdout, stderr bytes.Buffer
	if code := RunSync(opts, []string{"--json", "--skip=herdr"}, &stdout, &stderr); code != 0 {
		t.Fatalf("código %d:\n%s", code, stderr.String())
	}
	var res []SyncResult
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("stdout no es JSON: %v\n%s", err, stdout.String())
	}
	if a := actions(res); len(res) != 2 || a["pi"] != "current" || a["herdr"] != "skipped" {
		t.Fatalf("resultado = %+v", res)
	}
	if !strings.Contains(stderr.String(), "pi al día") {
		t.Fatalf("el progreso debe ir a stderr:\n%s", stderr.String())
	}
	if code := RunSync(opts, []string{"--only=pi", "--skip=herdr"}, &stdout, &stderr); code != 2 {
		t.Fatalf("--only con --skip: código %d", code)
	}
}

// T13 · C-01, C-05, C-07: InstallAll reconcilia con sync en [1/9] (instala y
// actualiza), los pins siguen en [4/9] y [5/9], y PI_GOMEMORY_SKIP_DEPS lo omite.
func TestHCD076InstallAllUsesSync(t *testing.T) {
	opts := toolSandbox(t)
	versionedTool(t, opts.LocalBinDir, "codegraph", "1.0.0", "upgrade", "2.0.0")
	srv, hits := scriptServer(t, map[string]string{"/pi.sh": "mkdir -p \"$PI_CODING_AGENT_DIR/bin\" && printf '#!/bin/sh\\necho 1.0.0\\n' > \"$PI_CODING_AGENT_DIR/bin/pi\" && chmod +x \"$PI_CODING_AGENT_DIR/bin/pi\"\n"})
	withToolHooks(t, testHooks(srv))
	withTools(t, []Tool{
		{Name: "pi", Required: true, Script: srv.URL + "/pi.sh", Interp: "sh", Env: piEnv, Manual: "m"},
		{Name: "codegraph", Script: srv.URL + "/cg.sh", Interp: "sh", Manual: "m"},
	})
	withSyncProbes(t, func(*Options) []DepProbe {
		return []DepProbe{probeFor("pi", "1.0.0"), probeFor("codegraph", "2.0.0", "codegraph", "upgrade")}
	})
	if err := InstallAll(repoAssets(t), opts); err != nil {
		t.Fatalf("InstallAll: %v", err)
	}
	if len(*hits) != 1 || (*hits)[0] != "/pi.sh" {
		t.Fatalf("pi debe instalarse por su canal: %v", *hits)
	}
	data, _ := os.ReadFile(filepath.Join(opts.LocalBinDir, "codegraph.ver"))
	if strings.TrimSpace(string(data)) != "2.0.0" {
		t.Fatalf("install debe actualizar codegraph sin preguntar: %q", data)
	}
	settings, _ := os.ReadFile(filepath.Join(opts.PiAgentDir, "settings.json"))
	if !strings.Contains(string(settings), AntigravityPackageSource) {
		t.Fatalf("el pin de pi-antigravity debe seguir instalándose en [4/9]:\n%s", settings)
	}
	src, _ := os.ReadFile("installer.go")
	if strings.Contains(string(src), "EnsureTools(") {
		t.Fatal("InstallAll no debe usar EnsureTools: el paso [1/9] es sync")
	}
	t.Setenv(skipDepsEnv, "1")
	versionedTool(t, opts.LocalBinDir, "codegraph", "1.0.0", "upgrade", "2.0.0")
	if err := InstallAll(repoAssets(t), opts); err != nil {
		t.Fatalf("InstallAll con %s: %v", skipDepsEnv, err)
	}
	data, _ = os.ReadFile(filepath.Join(opts.LocalBinDir, "codegraph.ver"))
	if strings.TrimSpace(string(data)) != "1.0.0" || len(*hits) != 1 {
		t.Fatalf("%s no debe tocar herramientas: %q %v", skipDepsEnv, data, *hits)
	}
}

// T15 · C-17: deps-update no actualiza nada dos veces: si pi-gomemory está
// desactualizado solo ejecuta update; si no, delega los tool en un sync.
func TestHCD076DepsUpdateDelegates(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "calls.log")
	self := filepath.Join(dir, "pi-gomemory-setup")
	writeFile(t, self, "#!/bin/sh\necho \"setup $*\" >> "+log+"\n")
	writeFile(t, filepath.Join(dir, "pi"), "#!/bin/sh\necho \"pi $*\" >> "+log+"\n")
	for _, f := range []string{self, filepath.Join(dir, "pi")} {
		if err := os.Chmod(f, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	old := selfExecutable
	selfExecutable = func() (string, error) { return self, nil }
	t.Cleanup(func() { selfExecutable = old })
	t.Setenv("PATH", dir+":/usr/bin:/bin")

	fixed := func(v string) func(context.Context) (string, error) {
		return func(context.Context) (string, error) { return v, nil }
	}
	probes := []DepProbe{
		{Name: "pi-gomemory", Current: fixed("v1.0.0"), Update: []string{"pi-gomemory-setup", "update"}},
		{Name: "mem", Current: fixed("1.0.0"), Update: []string{"mem", "update", "-y"}},
		{Name: "codegraph", Current: fixed("1.0.0"), Update: []string{"codegraph", "upgrade"}},
		{Name: "pi-cc-extensions", Current: fixed("0.0.1"), Update: []string{"pi", "install", CCExtensionsSource}},
	}
	run := func(deps []DepState) string {
		_ = os.Remove(log)
		cache := filepath.Join(dir, "deps.json")
		raw, _ := json.Marshal(depsCache{CheckedAt: time.Now(), Deps: deps})
		writeFile(t, cache, string(raw))
		var out, errOut bytes.Buffer
		if code := RunDepsUpdate(cache, probes, &out, &errOut); code != 0 {
			t.Fatalf("código %d:\n%s%s", code, out.String(), errOut.String())
		}
		data, _ := os.ReadFile(log)
		return string(data)
	}
	all := []DepState{
		{Name: "pi-gomemory", Current: "v1.0.0", Latest: "v1.1.0"},
		{Name: "mem", Current: "1.0.0", Latest: "1.1.0"},
		{Name: "codegraph", Current: "1.0.0", Latest: "1.0.0"},
		{Name: "pi-cc-extensions", Current: "0.0.1", Latest: "0.0.1"},
	}
	if got := run(all); got != "setup update\n" {
		t.Fatalf("con pi-gomemory desactualizado solo debe ejecutarse update:\n%s", got)
	}
	all[0].Latest = "v1.0.0"
	all[2].Latest = "1.1.0"
	got := run(all)
	if !strings.Contains(got, "setup sync --yes --only=mem,codegraph\n") || strings.Count(got, "setup ") != 1 {
		t.Fatalf("los tool deben delegarse en un solo sync:\n%s", got)
	}
	if !strings.Contains(got, "pi install "+CCExtensionsSource) {
		t.Fatalf("el pin debe actualizarse como hoy:\n%s", got)
	}
}

// T11 · C-11, N2: --only, --skip y lite; los obligatorios van siempre.
func TestHCD076SyncFlags(t *testing.T) {
	tools := productionTools()
	sel := func(so SyncOptions, lite bool) string {
		var out []string
		for n, ok := range selectTools(tools, so, lite) {
			if ok {
				out = append(out, n)
			}
		}
		sort.Strings(out)
		return strings.Join(out, ",")
	}
	parse := func(args ...string) SyncOptions {
		so, err := ParseSyncArgs(args)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		return so
	}
	if got := sel(parse("--only=codegraph"), false); got != "codegraph,mem,pi" {
		t.Fatalf("--only=codegraph → %s", got)
	}
	if got := sel(parse("--skip=herdr"), false); got != "codebase-memory-mcp,codegraph,mem,pi" {
		t.Fatalf("--skip=herdr → %s", got)
	}
	if got := sel(parse("--skip=pi"), false); !strings.Contains(got, "pi") {
		t.Fatalf("--skip=pi debe conservar pi → %s", got)
	}
	if got := sel(parse("--yes"), true); got != "mem,pi" {
		t.Fatalf("lite --yes → %s", got)
	}
	if so := parse("--json", "--only=herdr"); !so.JSON || !so.NonInteractive() {
		t.Fatalf("--json --only: %+v", so)
	}
	for _, bad := range [][]string{{"--only=pi", "--skip=herdr"}, {"--only=nada"}, {"--only=pi-gomemory"}, {"--skip=pi-antigravity"}, {"--bogus"}} {
		if _, err := ParseSyncArgs(bad); err == nil {
			t.Fatalf("%v debe rechazarse", bad)
		}
	}
}

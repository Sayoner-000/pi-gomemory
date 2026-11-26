package installer

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// HCD-085: /tdd y la integración Antigravity se retiran del meta-harness.

// Tarea 1: gomemory-tdd.ts ya no está en el repo ni se embebe ni se instala.
func TestHCD085TDDRetired(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", "extensions", "gomemory-tdd.ts")); !os.IsNotExist(err) {
		t.Fatalf("extensions/gomemory-tdd.ts debe haberse borrado: %v", err)
	}
	for _, r := range tuiResources {
		if strings.Contains(r.source, "gomemory-tdd") || strings.Contains(r.destination, "gomemory-tdd") {
			t.Fatalf("tuiResources aún instala %+v", r)
		}
	}
	opts := sandbox(t)
	if err := InstallTUIExtension(repoAssets(t), opts.PiAgentDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(opts.PiAgentDir, "extensions", "gomemory-tdd.ts")); !os.IsNotExist(err) {
		t.Fatalf("InstallTUIExtension no debe escribir gomemory-tdd.ts: %v", err)
	}
}

// Tarea 2: el sidebar no consulta ni pinta la cuota de Antigravity, y /min ya
// no depende del panel TDD.
func TestHCD085SidebarNoAntigravity(t *testing.T) {
	src := readAsset(t, "extensions/gomemory-sidebar.ts")
	for _, gone := range []string{`"usage", "--json"`, "refreshQuotaIfNeeded", "Google AI Pro", "isAntigravityProvider", "dock().tdd", "panelDock().tdd"} {
		if strings.Contains(src, gone) {
			t.Errorf("gomemory-sidebar.ts aún contiene %q", gone)
		}
	}
	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import sidebarExt, { renderSidebar } from "./gomemory-sidebar.ts";
const fails = [];
const calls = [];
const handlers = {};
sidebarExt({
	registerCommand() {},
	on(ev, fn) { handlers[ev] = fn; return () => {}; },
	exec: async (cmd, args) => { calls.push(cmd + " " + (args || []).join(" ")); return { code: 1, stdout: "", stderr: "" }; },
});
const ctx = { cwd: process.cwd(), model: { provider: "antigravity", id: "gemini-agy" }, ui: { notify() {} } };
for (const ev of ["session_start", "turn_end", "agent_end", "model_select"]) await handlers[ev]?.({}, ctx);
await new Promise((r) => setTimeout(r, 20));
if (calls.some((c) => c.startsWith("pi-gomemory-setup usage"))) fails.push("el sidebar aún consulta la cuota de Antigravity: " + JSON.stringify(calls));
const lines = renderSidebar({ title: "x", tokens: 1, contextLimit: 10, cost: 0, servers: [], isAntigravityProvider: true, quota: { planLabel: "P", groups: [] } }, 36, { fg: (_c, s) => s, style: (s) => s });
if (lines.join("\n").includes("Google AI Pro")) fails.push("renderSidebar aún pinta la cuota de Antigravity");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
process.exit(0);
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD085SidebarNoAntigravity: %s", out)
	}
}

// Tarea 3 (complemento en el paquete installer): el launcher ya no reenvía
// `usage`/`quota` a un comando que dejó de existir.
func TestHCD085LauncherNoUsage(t *testing.T) {
	s := GenerateHarnessScript()
	for _, gone := range []string{`"usage"`, `"quota"`, "antigravity.usage", "pi-gomemory-setup usage"} {
		if strings.Contains(s, gone) {
			t.Errorf("el launcher aún contiene %q", gone)
		}
	}
}

// Tarea 4: install ya no fija pi-antigravity y retira de settings.json las
// entradas que escribió pi-gomemory; conserva las de la persona y nunca toca
// antigravity-accounts.json.
func TestHCD085RetireAntigravityPackage(t *testing.T) {
	opts := sandbox(t)
	bin := t.TempDir()
	log := filepath.Join(bin, "pi.log")
	writeExec(t, filepath.Join(bin, "pi"), `echo "$*" >> "`+log+`"`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	settings := filepath.Join(opts.PiAgentDir, "settings.json")
	writeFile(t, settings, `{"theme":"x","packages":["npm:pi-antigravity@0.15.0","npm:pi-antigravity@0.12.0","npm:pi-cc-extensions@`+CCExtensionsVersion+`"]}`)
	accounts := filepath.Join(opts.PiAgentDir, "antigravity-accounts.json")
	writeFile(t, accounts, `{"activeAccountId":"synthetic"}`)

	var out bytes.Buffer
	if err := RetireAntigravityPackage(opts.PiAgentDir, &out); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, settings)
	if strings.Contains(got, "pi-antigravity") || !strings.Contains(got, "pi-cc-extensions@"+CCExtensionsVersion) || !strings.Contains(got, `"theme": "x"`) {
		t.Fatalf("se retiran solo los pins de pi-antigravity y se conserva el resto:\n%s", got)
	}
	calls, _ := os.ReadFile(log)
	for _, want := range []string{"remove npm:pi-antigravity@0.15.0", "remove npm:pi-antigravity@0.12.0"} {
		if !strings.Contains(string(calls), want) {
			t.Errorf("falta «pi %s»; llamadas:\n%s", want, calls)
		}
	}
	if readFile(t, accounts) != `{"activeAccountId":"synthetic"}` {
		t.Fatal("antigravity-accounts.json son credenciales de la persona: no se tocan")
	}
	if !strings.Contains(out.String(), "antigravity-accounts.json") {
		t.Errorf("se debe avisar de que la sesión OAuth de Google sigue en disco:\n%s", out.String())
	}

	// Una entrada que pi-gomemory nunca escribió se conserva con aviso, sin pi remove.
	_ = os.Remove(log)
	writeFile(t, settings, `{"packages":["npm:pi-antigravity@9.9.9"]}`)
	out.Reset()
	if err := RetireAntigravityPackage(opts.PiAgentDir, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readFile(t, settings), "npm:pi-antigravity@9.9.9") || !strings.Contains(out.String(), "no lo instaló pi-gomemory") {
		t.Fatalf("una fuente ajena se conserva con aviso:\n%s\n%s", readFile(t, settings), out.String())
	}
	if calls, _ := os.ReadFile(log); len(calls) != 0 {
		t.Fatalf("no se llama a pi remove sobre una entrada ajena: %s", calls)
	}

	// InstallAll completo no vuelve a registrar pi-antigravity.
	for _, p := range PinnedPackages() {
		if p.Name == "pi-antigravity" {
			t.Fatal("pi-antigravity no debe seguir fijado")
		}
	}
}

// Tarea 5: el workaround antigravity-prompt-guard.js ya no se distribuye.
func TestHCD085PromptGuardRetired(t *testing.T) {
	for _, p := range []string{"extensions/antigravity-prompt-guard.js", "tests/antigravity-prompt-guard"} {
		if _, err := os.Stat(filepath.Join("..", "..", p)); !os.IsNotExist(err) {
			t.Errorf("%s debe haberse borrado: %v", p, err)
		}
	}
	for _, r := range tuiResources {
		if strings.Contains(r.destination, "antigravity") {
			t.Fatalf("tuiResources aún instala %+v", r)
		}
	}
	if _, err := fs.Stat(repoAssets(t), "extensions/antigravity-prompt-guard.js"); err == nil {
		t.Fatal("antigravity-prompt-guard.js no debe estar en los recursos embebidos")
	}
}

// Tarea 6: install borra las copias retiradas intactas según el manifiesto,
// conserva con aviso las editadas o sin huella, y uninstall también las retira.
func TestHCD085RetiredResources(t *testing.T) {
	opts := sandbox(t)
	ext := filepath.Join(opts.PiAgentDir, "extensions")
	tdd := filepath.Join(ext, "gomemory-tdd.ts")
	guard := filepath.Join(ext, "antigravity-prompt-guard.js")
	writeFile(t, tdd, "// tdd instalado por una versión anterior\n")
	writeFile(t, guard, "// guard instalado por una versión anterior\n")
	// Manifiesto de la versión anterior: registra ambas huellas.
	m := Manifest{Version: "v1.30.0", Files: map[string]string{
		tdd:   contentHash([]byte("// tdd instalado por una versión anterior\n")),
		guard: contentHash([]byte("// guard instalado por una versión anterior\n")),
	}}
	if err := os.MkdirAll(filepath.Dir(ManifestPath(opts.HomeDir)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONObject(ManifestPath(opts.HomeDir), m); err != nil {
		t.Fatal(err)
	}
	// La persona editó el guard: se conserva.
	writeFile(t, guard, "// guard editado por la persona\n")

	var out bytes.Buffer
	if err := RetireResources(opts, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(tdd); !os.IsNotExist(err) {
		t.Fatalf("gomemory-tdd.ts intacto debe retirarse: %v\n%s", err, out.String())
	}
	if readFile(t, guard) != "// guard editado por la persona\n" || !strings.Contains(out.String(), "se editó") {
		t.Fatalf("la copia editada se conserva con aviso:\n%s", out.String())
	}

	// Sin huella en el manifiesto no hay prueba de propiedad: se conserva.
	other := sandbox(t)
	orphan := filepath.Join(other.PiAgentDir, "extensions", "gomemory-tdd.ts")
	writeFile(t, orphan, "x")
	if err := RetireResources(other, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphan); err != nil {
		t.Fatalf("sin manifiesto no se borra: %v", err)
	}

	// uninstall retira un gomemory-tdd.ts intacto que dejó una versión anterior.
	writeFile(t, tdd, "// tdd instalado por una versión anterior\n")
	plan, err := BuildUninstallPlan(repoAssets(t), opts, UninstallFlags{})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	if a := actionFor(plan, tdd); a == nil || a.Kind != ActionRemove {
		t.Fatalf("uninstall debe retirar gomemory-tdd.ts intacto: %+v", a)
	}
	if a := actionFor(plan, guard); a == nil || a.Kind != ActionSkip {
		t.Fatalf("uninstall conserva el guard editado: %+v", a)
	}
}

// Tarea 7: README y CHANGELOG documentan la retirada.
func TestHCD085DocsUpdated(t *testing.T) {
	// Las notas de versiones anteriores pueden citar /tdd como historia; lo que
	// no puede quedar es documentación de uso vigente.
	readme := readAsset(t, "README.md")
	for _, gone := range []string{"| `/tdd", "run_test_suite", "pi-gomemory-setup usage", "Google AI Pro Quota", "node --test tests/antigravity-prompt-guard", "npm:pi-antigravity@0.15.0"} {
		if strings.Contains(readme, gone) {
			t.Errorf("README.md aún menciona %q", gone)
		}
	}
	if !strings.Contains(readme, "Retirado en HCD-085") {
		t.Error("README.md debe explicar la retirada")
	}
	changelog := readAsset(t, "CHANGELOG.md")
	head := changelog
	if i := strings.Index(changelog[1:], "\n## "); i > 0 {
		if j := strings.Index(changelog[i+2:], "\n## "); j > 0 {
			head = changelog[:i+2+j]
		}
	}
	for _, want := range []string{"HCD-085", "/tdd", "pi-antigravity", "antigravity-accounts.json"} {
		if !strings.Contains(head, want) {
			t.Errorf("la primera entrada de CHANGELOG.md no menciona %q", want)
		}
	}
}

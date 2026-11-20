package installer

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// C-004: con un título largo, renderSidebar debe conservar el control [−].
func TestHCD021SidebarLongTitle(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-sidebar.ts"}, `
import { renderSidebar } from "./gomemory-sidebar.ts";
const theme = { fg: (_c, s) => s, style: (s) => s };
const state = {
	title: "HCD · HCD-020 · evolution · paso close · v1.9.0", tokens: 1, contextLimit: 10, cost: 0,
	servers: [], providerName: "p", isAntigravityProvider: false, quota: null, isClaudeProvider: false, claudeUsage: null,
};
const fails = [];
for (const width of [36, 24]) {
	const header = renderSidebar(state, width, theme)[1];
	if (!header.includes("[−]")) fails.push("ancho " + width + ": la cabecera perdió [−]: " + JSON.stringify(header));
	if (!header.includes("…")) fails.push("ancho " + width + ": el título largo debe recortarse con …");
	if ([...header].length !== width) fails.push("ancho " + width + ": la fila mide " + [...header].length);
}
const short = renderSidebar({ ...state, title: "HCD · corto" }, 36, theme)[1];
if (!short.includes("HCD · corto") || short.includes("…")) fails.push("un título corto no se recorta");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD021SidebarLongTitle: %s", out)
	}
}

// C-005: fetchQuota no carga en memoria respuestas de más de 1 MiB.
func TestHCD021QuotaBodyLimit(t *testing.T) {
	t.Parallel()

	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		chunk := strings.Repeat("x", 64<<10)
		for i := 0; i < 40; i++ { // 2.5 MiB
			_, _ = w.Write([]byte(chunk))
		}
	}))
	defer big.Close()
	body, err := fetchQuota(big.URL, "token")
	if err == nil {
		t.Fatalf("una respuesta de 2.5 MiB debe dar error; se leyeron %d bytes", len(body))
	}
	if !strings.Contains(err.Error(), "1 MiB") {
		t.Fatalf("el error debe explicar el límite: %v", err)
	}

	small := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"planLabel":"x","groups":[]}`))
	}))
	defer small.Close()
	if body, err := fetchQuota(small.URL, "token"); err != nil || !strings.Contains(string(body), "planLabel") {
		t.Fatalf("una respuesta normal debe leerse entera: %q, %v", body, err)
	}
}

// C-003: durante una ejecución, N turnos en modo watch dejan una sola ejecución
// pendiente (no N), y un /tdd run que queda en cola lo avisa. Desde HCD-056 el
// /tdd run cancela esa ejecución de watch en lugar de esperar a que termine.
func TestHCD021WatchCoalesces(t *testing.T) {
	t.Parallel()

	proj := t.TempDir()
	bin := t.TempDir()
	count := filepath.Join(bin, "count.log")
	writeFile(t, filepath.Join(proj, "go.mod"), "module w\n")
	writeExec(t, filepath.Join(bin, "go"), `echo "$1" >> "`+count+`"
if [ "$1" = "test" ]; then
  sleep 0.4
  echo '{"Action":"run","Package":"w","Test":"TestA"}'
  echo '{"Action":"pass","Package":"w","Test":"TestA","Elapsed":0.01}'
fi
exit 0`)
	writeExec(t, filepath.Join(bin, "mem"), `exit 3`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import { readFileSync } from "node:fs";
import factory from "./gomemory-tdd.ts";
const commands = {}, handlers = {}, notes = [];
let tool;
factory({
	registerCommand: (n, d) => (commands[n] = d),
	registerTool: (d) => (tool = d),
	on: (e, h) => { (handlers[e] ||= []).push(h); return () => {}; },
	exec: async () => { throw new Error("pi.exec no se usa"); },
});
const ctx = { cwd: `+jsStr(proj)+`, hasUI: false, ui: { notify: (m) => notes.push(m) } };
await commands.tdd.handler("watch", ctx);
const first = tool.execute("c1", { suite: "all" }, undefined, undefined, ctx);
await new Promise((r) => setTimeout(r, 50));
const emit = (e, ev) => Promise.all((handlers[e] || []).map((h) => h(ev, ctx)));
const turns = [];
for (let i = 0; i < 5; i++) {
	await emit("tool_execution_end", { toolName: "edit", isError: false });
	turns.push(emit("agent_settled", { aborted: false }));
}
await first;
await Promise.all(turns);
// Se espera a que la única ejecución de watch agrupada esté dentro de go test (0,4 s);
// una espera fija no basta bajo la carga de la suite paralela (HCD-056).
const testRuns = () => { try { return readFileSync(`+jsStr(count)+`, "utf8").split("\n").filter((l) => l === "test").length; } catch { return 0; } };
for (let i = 0; i < 200 && testRuns() < 2; i++) await new Promise((r) => setTimeout(r, 10));
const t0 = Date.now();
await commands.tdd.handler("run", ctx);
const manualMs = Date.now() - t0;
await new Promise((r) => setTimeout(r, 600));
const runs = readFileSync(`+jsStr(count)+`, "utf8").split("\n").filter((l) => l === "test").length;
const fails = [];
// tool + una sola ejecución de watch agrupada (cancelada por /tdd run) + /tdd run manual
if (runs !== 3) fails.push("go test se ejecutó " + runs + " veces; se esperaban 3 (tool, watch agrupado, /tdd run)");
if (!notes.some((m) => /cola/i.test(m))) fails.push("un /tdd run en cola debe avisarlo: " + JSON.stringify(notes));
// /tdd run no espera los ~0,25 s que le quedaban a watch: solo su propia ejecución.
if (manualMs > 1500) fails.push("/tdd run esperó a watch: " + manualMs + " ms");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD021WatchCoalesces: %s", out)
	}
}

// C-006: en un repositorio Go externo (sin pkg/installer) /tdd ejecuta go test ./...
func TestHCD021ExternalGoRepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/ejemplo/servicio\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "internal", "orders", "orders_test.go"), `package orders

import "testing"

func TestTotal(t *testing.T) {}
`)
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, tddToolHarness+`
import { detectSuites } from "./gomemory-tdd.ts";
const plan = detectSuites(`+jsStr(dir)+`);
const cmds = plan.flatMap((p) => p.checks.map((c) => [c.cmd, ...c.args].join(" ")));
if (cmds.some((c) => c.includes("pkg/installer"))) fails.push("no debe asumir pkg/installer: " + cmds.join(" ; "));
if (!cmds.includes("go test -json ./...")) fails.push("backend debe ser go test -json ./...");
const res = await run("backend", `+jsStr(dir)+`);
const total = suiteOf(res.details, "backend")?.tests.find((x) => x.name === "TestTotal");
if (total?.status !== "passed") fails.push("TestTotal del repo externo debe pasar, está " + total?.status);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD021ExternalGoRepo: %s", out)
	}
}

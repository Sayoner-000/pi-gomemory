package installer

import (
	"path/filepath"
	"strings"
	"testing"
)

// Modo watch: una ejecución por prompt que cambió archivos, al asentarse el agente
// y sin bloquearlo. En pi 1.5 turn_end se dispara (y se espera) en cada turno: si
// watch corre ahí, la suite se relanza entre cada tool call y el agente queda en bucle.
func TestTDDWatchRunsOncePerSettledChange(t *testing.T) {
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
const commands = {}, handlers = {};
let tool;
factory({
	registerCommand: (n, d) => (commands[n] = d),
	registerTool: (d) => (tool = d),
	on: (e, h) => { (handlers[e] ||= []).push(h); return () => {}; },
	exec: async () => { throw new Error("pi.exec no se usa"); },
});
const ctx = { cwd: `+jsStr(proj)+`, hasUI: false, ui: { notify: () => {} } };
const emit = async (e, ev) => { for (const h of handlers[e] || []) await h(ev, ctx); };
const runs = () => { try { return readFileSync(`+jsStr(count)+`, "utf8").split("\n").filter((l) => l === "test").length; } catch { return 0; } };
const idle = () => new Promise((r) => setTimeout(r, 1500));
const fails = [];
await commands.tdd.handler("watch", ctx);

// a) turnos y prompts que no tocan archivos: watch no ejecuta nada
for (let i = 0; i < 3; i++) {
	await emit("tool_execution_end", { toolName: "read", isError: false });
	await emit("turn_end", { toolResults: [] });
	await emit("agent_settled", { aborted: false });
}
await idle();
if (runs() !== 0) fails.push("sin cambios watch no debe ejecutar; ejecutó " + runs());

// b) varios turnos con un edit: una sola ejecución, al asentarse y sin bloquear
await emit("tool_execution_end", { toolName: "edit", isError: false });
const t0 = Date.now();
for (let i = 0; i < 3; i++) await emit("turn_end", { toolResults: [] });
await emit("agent_settled", { aborted: false });
const blocked = Date.now() - t0;
if (blocked >= 200) fails.push("los eventos del agente esperan a la suite (" + blocked + "ms)");
await idle();
if (runs() !== 1) fails.push("tras un edit debe ejecutar una vez; ejecutó " + runs());

// c) el agente ya ejecutó run_test_suite all tras su cambio: watch no repite
await emit("tool_execution_end", { toolName: "write", isError: false });
await tool.execute("c1", { suite: "all" }, undefined, undefined, ctx);
const afterTool = runs();
await emit("agent_settled", { aborted: false });
await idle();
if (runs() !== afterTool) fails.push("tras run_test_suite all watch no debe repetir");

// d) un prompt abortado no lanza la suite
await emit("tool_execution_end", { toolName: "edit", isError: false });
await emit("agent_settled", { aborted: true });
await idle();
if (runs() !== afterTool) fails.push("un prompt abortado no debe ejecutar watch");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestTDDWatchRunsOncePerSettledChange: %s", out)
	}
}

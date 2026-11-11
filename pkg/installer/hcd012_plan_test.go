package installer

import "testing"

// C-08: con el modo plan activo, el manejador tool_call bloquea powershell.
func TestHCD012PlanPowershell(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-plan.ts", "extensions/gomemory-plan-patterns.json"}, `
import ext from "./gomemory-plan.ts";
const handlers = {}; let cmd;
const pi = {
  registerCommand: (n, c) => { cmd = c; },
  on: (e, h) => { handlers[e] = h; },
  exec: async () => ({ code: 1, stdout: "" }),
  sendUserMessage: () => {},
};
ext(pi);
const ctx = { cwd: ".", ui: { notify() {}, setStatus() {} } };
const call = (toolName) => handlers.tool_call({ toolName, input: { command: "Remove-Item x" } });
const fails = [];
if (await call("powershell")) fails.push("bloquea con el modo inactivo");
await cmd.handler("", ctx);
const r = await call("powershell");
if (!r || r.block !== true) fails.push("powershell no bloqueado: " + JSON.stringify(r));
if (await call("read")) fails.push("read bloqueado");
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if out != "PASS" {
		t.Fatal(out)
	}
}

package installer

import (
	"strings"
	"testing"
)

// Los tests de detección de suites, parsers y escucha de Bash de /tdd
// (HCD-051) se retiraron con /tdd en HCD-085.

func TestHCD051TasksLivePolling(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import assert from "node:assert/strict";
import registerTasks from "./gomemory-tasks.ts";

const handlers = new Map();
let execCalls = 0;
const fakePi = {
	registerCommand() {},
	on(name, fn) { handlers.set(name, fn); },
	async exec() {
		execCalls++;
		return { code: 0, stdout: JSON.stringify({ ok: true, unit_id: "HCD-051", kind: "feature", stage: "implement", state: "active", tasks: [{ id: "1", text: "t1", status: execCalls > 1 ? "verified" : "pending" }] }) };
	},
};
registerTasks(fakePi);

const onStart = handlers.get("tool_execution_start");
const onEnd = handlers.get("tool_execution_end");
assert.ok(onStart, "debe registrar tool_execution_start");
assert.ok(onEnd, "debe registrar tool_execution_end");

const ctx = { cwd: process.cwd(), ui: { setWidget() {} } };
await onStart({ toolName: "bash", args: { command: "mem hcd verify --task 1 --json -- 'go test'" } }, ctx);
await new Promise((r) => setTimeout(r, 1700));
await onEnd({ toolName: "bash" }, ctx);

assert.ok(execCalls >= 1, "debe haber refrescado durante la ejecución bash: execCalls=" + execCalls);
`)
	t.Log(out)
}

func TestHCD051VerifySingleCallGuard(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import assert from "node:assert/strict";
import registerTasks, { isBundledHcdVerifyCommand } from "./gomemory-tasks.ts";

assert.equal(isBundledHcdVerifyCommand("mem hcd verify --task 1 --json -- 'go test ./...'"), null);
assert.ok(isBundledHcdVerifyCommand("mem hcd verify --task 1 --json -- 'a' && mem hcd verify --task 2 --json -- 'b'"));
assert.ok(isBundledHcdVerifyCommand("v() { mem hcd verify --task $1 --json -- $2; }; v 1 'a'"));
assert.ok(isBundledHcdVerifyCommand("for i in 1 2; do mem hcd verify --task $i --json -- 'ok'; done"));

const handlers = new Map();
registerTasks({ registerCommand() {}, on(name, fn) { handlers.set(name, fn); }, async exec() { return { code: 0, stdout: "" }; } });
const onToolCall = handlers.get("tool_call");
assert.ok(onToolCall, "debe registrar tool_call");

const blocked = await onToolCall({ toolName: "bash", input: { command: "v(){ mem hcd verify --task $1; }; v 1" } });
assert.equal(blocked?.block, true);
const allowed = await onToolCall({ toolName: "bash", input: { command: "mem hcd verify --task 1 --json -- 'go test'" } });
assert.equal(allowed, undefined);
`)
	t.Log(out)
}

func TestHCD051LiteModeAndCodegraphGate(t *testing.T) {
	opts := sandbox(t)
	if err := SetLiteMode(opts, true); err != nil {
		t.Fatalf("SetLiteMode(true) failed: %v", err)
	}
	if !IsLiteMode(opts.HomeDir) {
		t.Fatal("IsLiteMode debe ser true tras SetLiteMode(true)")
	}
	mcp, err := readJSONObject(opts.PiAgentDir + "/mcp.json")
	if err != nil {
		t.Fatal(err)
	}
	servers := string(mcp["mcpServers"])
	if !containsAll(servers, "gomemory") || containsAny(servers, "codegraph", "codebase-memory") {
		t.Fatalf("en modo lite solo debe quedar gomemory en mcpServers, obtenido: %s", servers)
	}

	if err := SetLiteMode(opts, false); err != nil {
		t.Fatalf("SetLiteMode(false) failed: %v", err)
	}
	if IsLiteMode(opts.HomeDir) {
		t.Fatal("IsLiteMode debe ser false tras SetLiteMode(false)")
	}
	mcpFull, err := readJSONObject(opts.PiAgentDir + "/mcp.json")
	if err != nil {
		t.Fatal(err)
	}
	serversFull := string(mcpFull["mcpServers"])
	// Desde v1.28.0 son 2: codebase-memory se retiró (HCD-079, Q2).
	if !containsAll(serversFull, "gomemory", "codegraph") || containsAny(serversFull, "codebase-memory") {
		t.Fatalf("tras SetLiteMode(false) deben restaurarse los 2 servidores, obtenido: %s", serversFull)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

package installer

import (
	"strings"
	"testing"
)

func TestHCD051DetectBackendSuites(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { detectSuites } from "./gomemory-tdd.ts";

const makeDir = () => mkdtempSync(join(tmpdir(), "hcd051-backend-"));

// 1. Python con pyproject.toml + uv.lock + ruff/mypy
{
	const dir = makeDir();
	try {
		writeFileSync(join(dir, "pyproject.toml"), "[project]\nname = \"kolmena\"\n[tool.ruff]\nline-length = 100\n[tool.mypy]\nstrict = true\n");
		writeFileSync(join(dir, "uv.lock"), "");
		const suites = detectSuites(dir);
		const backend = suites.find((s) => s.id === "backend");
		assert.ok(backend && !backend.skipReason, "backend Python debe detectarse");
		assert.ok(backend.checks.some((c) => c.id === "pytest" && c.cmd === "uv" && c.args.includes("pytest")), "debe usar uv run pytest: " + JSON.stringify(backend.checks));
		const ci = suites.find((s) => s.id === "ci");
		assert.ok(ci && ci.checks.some((c) => c.id === "ruff"), "ci debe incluir ruff");
		assert.ok(ci && ci.checks.some((c) => c.id === "mypy"), "ci debe incluir mypy");
	} finally { rmSync(dir, { recursive: true, force: true }); }
}

// 2. Python con .venv/bin/pytest y pytest.ini
{
	const dir = makeDir();
	try {
		mkdirSync(join(dir, ".venv", "bin"), { recursive: true });
		writeFileSync(join(dir, ".venv", "bin", "pytest"), "#!/bin/sh\n");
		writeFileSync(join(dir, "pytest.ini"), "[pytest]\n");
		const suites = detectSuites(dir);
		const backend = suites.find((s) => s.id === "backend");
		assert.ok(backend && backend.checks.some((c) => c.id === "pytest" && c.cmd.endsWith(".venv/bin/pytest")));
	} finally { rmSync(dir, { recursive: true, force: true }); }
}

// 3. PHP con composer.json y vendor/bin/phpunit
{
	const dir = makeDir();
	try {
		mkdirSync(join(dir, "vendor", "bin"), { recursive: true });
		writeFileSync(join(dir, "vendor", "bin", "phpunit"), "#!/bin/sh\n");
		writeFileSync(join(dir, "composer.json"), "{\"name\":\"app/php\"}");
		writeFileSync(join(dir, "phpunit.xml"), "<phpunit/>");
		const suites = detectSuites(dir);
		const backend = suites.find((s) => s.id === "backend");
		assert.ok(backend && !backend.skipReason, "backend PHP debe detectarse");
		assert.ok(backend.checks.some((c) => c.id === "phpunit" && c.cmd.endsWith("vendor/bin/phpunit")));
		const ci = suites.find((s) => s.id === "ci");
		assert.ok(ci && ci.checks.some((c) => c.id === "php-lint"));
	} finally { rmSync(dir, { recursive: true, force: true }); }
}

// 4. Java con pom.xml + mvnw y build.gradle.kts
{
	const dir = makeDir();
	try {
		writeFileSync(join(dir, "pom.xml"), "<project/>");
		writeFileSync(join(dir, "mvnw"), "#!/bin/sh\n");
		const suites = detectSuites(dir);
		const backend = suites.find((s) => s.id === "backend");
		assert.ok(backend && backend.checks.some((c) => c.id === "mvn-test" && c.cmd === "./mvnw"));
	} finally { rmSync(dir, { recursive: true, force: true }); }

	const gdir = makeDir();
	try {
		writeFileSync(join(gdir, "build.gradle.kts"), "");
		const suites = detectSuites(gdir);
		const backend = suites.find((s) => s.id === "backend");
		assert.ok(backend && backend.checks.some((c) => c.id === "gradle-test" && c.cmd === "gradle"));
	} finally { rmSync(gdir, { recursive: true, force: true }); }
}
`)
	t.Log(out)
}

func TestHCD051DetectFrontendAndStaticSuites(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { detectSuites } from "./gomemory-tdd.ts";

const makeDir = () => mkdtempSync(join(tmpdir(), "hcd051-fe-"));

// 1. Vitest sin scripts.test en package.json
{
	const dir = makeDir();
	try {
		writeFileSync(join(dir, "package.json"), "{\"name\":\"web\"}");
		writeFileSync(join(dir, "vitest.config.ts"), "export default {};");
		const suites = detectSuites(dir);
		const fe = suites.find((s) => s.id === "frontend");
		assert.ok(fe && !fe.skipReason, "frontend con vitest.config debe detectarse");
		assert.ok(fe.checks.some((c) => c.id === "vitest"), "debe incluir check vitest: " + JSON.stringify(fe.checks));
	} finally { rmSync(dir, { recursive: true, force: true }); }
}

// 2. HTML y CSS static checks (.htmlhintrc, .stylelintrc.json, eslint.config.mjs)
{
	const dir = makeDir();
	try {
		writeFileSync(join(dir, ".htmlhintrc"), "{}");
		writeFileSync(join(dir, ".stylelintrc.json"), "{}");
		writeFileSync(join(dir, "eslint.config.mjs"), "export default [];");
		const suites = detectSuites(dir);
		const ci = suites.find((s) => s.id === "ci");
		assert.ok(ci && ci.checks.some((c) => c.id === "htmlhint"), "ci debe incluir htmlhint");
		assert.ok(ci && ci.checks.some((c) => c.id === "stylelint"), "ci debe incluir stylelint");
		assert.ok(ci && ci.checks.some((c) => c.id === "eslint"), "ci debe incluir eslint");
	} finally { rmSync(dir, { recursive: true, force: true }); }
}
`)
	t.Log(out)
}

func TestHCD051OutputParsers(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import { createTDDStore, pytestParser, vitestParser, phpunitParser, javaTestParser } from "./gomemory-tdd.ts";

// 1. pytest parser (FAILED + PASSED + summary)
{
	const store = createTDDStore();
	store.initSuites([{ id: "backend", name: "Backend", domain: "backend" }]);
	store.finishSuite("backend", "running");
	const p = pytestParser(store, "backend", "pytest");
	p.onLine("tests/test_api.py::test_ok PASSED");
	p.onLine("FAILED tests/test_api.py::test_fail - AssertionError: expected 200");
	p.onLine("tests/test_api.py:42: AssertionError");
	const failed = p.finish({ code: 1, missing: false, tail: ["FAILED tests/test_api.py::test_fail"] });
	assert.equal(failed, true);
	const suite = store.getState().suites.find((s) => s.id === "backend");
	assert.ok(suite.tests.some((t) => t.name.includes("test_ok") && t.status === "passed"));
	assert.ok(suite.tests.some((t) => t.name.includes("test_fail") && t.status === "failed"));
}

// 2. vitest parser
{
	const store = createTDDStore();
	store.initSuites([{ id: "frontend", name: "Frontend", domain: "frontend" }]);
	store.finishSuite("frontend", "running");
	const p = vitestParser(store, "frontend", "vitest");
	p.onLine(" ✓ src/math.test.ts > add works 5ms");
	p.onLine(" × src/math.test.ts > sub fails 8ms");
	p.onLine("   AssertionError: expected 1 to be 2");
	const failed = p.finish({ code: 1, missing: false, tail: [] });
	assert.equal(failed, true);
	const suite = store.getState().suites.find((s) => s.id === "frontend");
	assert.ok(suite.tests.some((t) => t.name.includes("add works") && t.status === "passed"));
	assert.ok(suite.tests.some((t) => t.name.includes("sub fails") && t.status === "failed"));
}

// 3. phpunit parser
{
	const store = createTDDStore();
	store.initSuites([{ id: "backend", name: "Backend", domain: "backend" }]);
	store.finishSuite("backend", "running");
	const p = phpunitParser(store, "backend", "phpunit");
	p.onLine("PHPUnit 11.0.0");
	p.onLine("1) App\\Tests\\UserTest::testCreate");
	p.onLine("Failed asserting that false is true.");
	p.onLine("Tests: 5, Assertions: 8, Failures: 1.");
	const failed = p.finish({ code: 1, missing: false, tail: [] });
	assert.equal(failed, true);
	const suite = store.getState().suites.find((s) => s.id === "backend");
	assert.ok(suite.tests.some((t) => t.status === "failed" && t.name.includes("UserTest::testCreate")));
}

// 4. javaTestParser (Maven / Gradle)
{
	const store = createTDDStore();
	store.initSuites([{ id: "backend", name: "Backend", domain: "backend" }]);
	store.finishSuite("backend", "running");
	const p = javaTestParser(store, "backend", "mvn test");
	p.onLine("[ERROR] Failures:");
	p.onLine("[ERROR]   OrderServiceTest.rejectsEmpty:34 expected: <true> but was: <false>");
	p.onLine("[ERROR] Tests run: 3, Failures: 1, Errors: 0, Skipped: 0");
	const failed = p.finish({ code: 1, missing: false, tail: [] });
	assert.equal(failed, true);
	const suite = store.getState().suites.find((s) => s.id === "backend");
	assert.ok(suite.tests.some((t) => t.status === "failed" && t.name.includes("OrderServiceTest.rejectsEmpty")));
}
`)
	t.Log(out)
}

func TestHCD051BashTestListener(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import registerTDD, { classifyBashTestCommand, createTDDStore, ingestBashTestResult } from "./gomemory-tdd.ts";

assert.equal(classifyBashTestCommand("uv run pytest -q")?.suiteId, "backend");
assert.equal(classifyBashTestCommand("npx vitest run")?.suiteId, "frontend");
assert.equal(classifyBashTestCommand("mem hcd verify --task 1 --json -- 'go test ./...'")?.suiteId, "data");
assert.equal(classifyBashTestCommand("git status"), null);

const store = createTDDStore();
assert.equal(
	ingestBashTestResult(store, "pytest -v", "tests/test_a.py::test_ok PASSED\nFAILED tests/test_a.py::test_fail - AssertionError", true, process.cwd()),
	true,
);
const backend = store.getState().suites.find((s) => s.id === "backend");
assert.equal(backend?.status, "failed");
assert.ok(backend?.tests.some((t) => t.name.includes("test_fail") && t.status === "failed"));

assert.equal(
	ingestBashTestResult(store, "mem hcd verify --task 1 --json -- 'pytest'", '{"ok":true,"unit_id":"HCD-051","data":{"evidence":{"passed":true}}}', false, process.cwd()),
	true,
);
const data = store.getState().suites.find((s) => s.id === "data");
assert.equal(data?.status, "passed");

const handlers = new Map();
let lastToolOutput = "";
const fakePi = {
	registerCommand() {},
	registerTool() {},
	sendMessage() {},
	on(name, fn) { handlers.set(name, fn); },
};
registerTDD(fakePi);
const onToolResult = handlers.get("tool_result");
assert.ok(onToolResult, "debe registrar listener tool_result");

// Simular pytest fallido por bash y comprobar que /tdd store lo refleja vía run_test_suite o tool_result
await onToolResult({
	toolName: "bash",
	input: { command: "pytest -v" },
	isError: true,
	content: [{ type: "text", text: "FAILED tests/test_a.py::test_x - AssertionError" }],
}, { cwd: process.cwd(), ui: {} });

// También simular mem hcd verify exitoso
await onToolResult({
	toolName: "bash",
	input: { command: "mem hcd verify --task 1 --json -- 'pytest'" },
	isError: false,
	content: [{ type: "text", text: '{"ok":true,"unit_id":"HCD-051","data":{"evidence":{"passed":true}}}' }],
}, { cwd: process.cwd(), ui: {} });
`)
	t.Log(out)
}

func TestHCD051TasksLivePolling(t *testing.T) {
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
	if !containsAll(serversFull, "gomemory", "codegraph", "codebase-memory") {
		t.Fatalf("tras SetLiteMode(false) deben restaurarse los 3 servidores, obtenido: %s", serversFull)
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

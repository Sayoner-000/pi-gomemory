package installer

import (
	"path/filepath"
	"testing"
)

func TestHCD027GoPendingResults(t *testing.T) {
	// Part 1: direct goTestParser check
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import { createTDDStore, goTestParser } from "./gomemory-tdd.ts";

// 1. Salida anormal
{
	const store = createTDDStore();
	store.initSuites([{ id: "backend", name: "Backend", domain: "backend" }]);
	const parser = goTestParser(store, "backend");
	parser.onLine(JSON.stringify({ Action: "run", Package: "synth/pkg", Test: "TestPendingAbnormal" }));
	parser.onLine(JSON.stringify({ Action: "output", Package: "synth/pkg", Test: "TestPendingAbnormal", Output: "running test...\n" }));

	const failed = parser.finish({ code: 1, missing: false, tail: ["fatal error: out of memory"] });
	assert.equal(failed, true, "parser.finish must report failure on abnormal exit");

	const state = store.getState();
	const test = state.suites[0].tests.find(t => t.id === "synth/pkg::TestPendingAbnormal");
	assert.ok(test, "pending test must exist in suite");
	assert.equal(test.status, "failed", "pending test must be closed as failed on abnormal exit");
	assert.equal(state.stats.running, 0, "stats.running must be 0");
	assert.equal(state.status, "failed", "state.status must be failed");
	assert.ok(test.failure?.message, "test must have a diagnostic message");
}

// 2. Timeout
{
	const store = createTDDStore();
	store.initSuites([{ id: "backend", name: "Backend", domain: "backend" }]);
	const parser = goTestParser(store, "backend");
	parser.onLine(JSON.stringify({ Action: "run", Package: "synth/pkg", Test: "TestPendingTimeout" }));

	const failed = parser.finish({ code: 124, missing: false, timedOut: true, tail: [] });
	assert.equal(failed, true, "parser.finish must report failure on timeout");

	const state = store.getState();
	const test = state.suites[0].tests.find(t => t.id === "synth/pkg::TestPendingTimeout");
	assert.ok(test, "pending test must exist in suite");
	assert.equal(test.status, "failed", "pending test must be closed as failed on timeout");
	assert.equal(state.stats.running, 0, "stats.running must be 0 on timeout");
	assert.equal(state.status, "failed", "state.status must be failed on timeout");
	assert.ok(test.failure?.message, "test must have diagnostic on timeout");
}

// 3. Abort
{
	const store = createTDDStore();
	store.initSuites([{ id: "backend", name: "Backend", domain: "backend" }]);
	const parser = goTestParser(store, "backend");
	parser.onLine(JSON.stringify({ Action: "run", Package: "synth/pkg", Test: "TestPendingAbort" }));

	const failed = parser.finish({ code: 1, missing: false, aborted: true, tail: [] });
	assert.equal(failed, true, "parser.finish must report failure on abort");

	const state = store.getState();
	const test = state.suites[0].tests.find(t => t.id === "synth/pkg::TestPendingAbort");
	assert.ok(test, "pending test must exist in suite");
	assert.equal(test.status, "failed", "pending test must be closed as failed on abort");
	assert.equal(state.stats.running, 0, "stats.running must be 0 on abort");
	assert.equal(state.status, "failed", "state.status must be failed on abort");
	assert.ok(test.failure?.message, "test must have diagnostic on abort");
}

console.log("PASS TestHCD027GoPendingResults parser tests");
`)
	t.Log(out)

	// Part 2: End-to-end runner test with fake go tool
	project, bin := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), "module synthetic\n")
	writeExec(t, filepath.Join(bin, "go"), `echo '{"Action":"run","Package":"synthetic","Test":"TestInterruptedE2E"}'
exit 1`)
	outE2E := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"},
		`process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
`+tddToolHarness+`
const result = await run("backend", `+jsStr(project)+`);
const state = result.details;
const test = suiteOf(state, "backend")?.tests.find(t => t.name === "TestInterruptedE2E");
assert.ok(test, "started Go test must exist");
assert.equal(test.status, "failed", "interrupted Go test must be failed");
assert.equal(state.stats.running, 0, "stats.running must be 0");
assert.equal(state.status, "failed", "global state must be failed");
assert.ok(test.failure?.message, "interrupted test must have diagnostic");
console.log("PASS TestHCD027GoPendingResults e2e");
`)
	t.Log(outE2E)
}

func TestHCD027GoPreservesResults(t *testing.T) {
	// Part 1: direct parser check with mixed test lifecycle
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import { createTDDStore, goTestParser } from "./gomemory-tdd.ts";

const store = createTDDStore();
store.initSuites([{ id: "backend", name: "Backend", domain: "backend" }]);
const parser = goTestParser(store, "backend");

// 1. Test that passes
parser.onLine(JSON.stringify({ Action: "run", Package: "mix/pkg", Test: "TestPassed" }));
parser.onLine(JSON.stringify({ Action: "pass", Package: "mix/pkg", Test: "TestPassed", Elapsed: 0.012 }));

// 2. Test that is skipped
parser.onLine(JSON.stringify({ Action: "run", Package: "mix/pkg", Test: "TestSkipped" }));
parser.onLine(JSON.stringify({ Action: "skip", Package: "mix/pkg", Test: "TestSkipped", Elapsed: 0.001 }));

// 3. Test that fails with explicit location and message
parser.onLine(JSON.stringify({ Action: "run", Package: "mix/pkg", Test: "TestAlreadyFailed" }));
parser.onLine(JSON.stringify({ Action: "output", Package: "mix/pkg", Test: "TestAlreadyFailed", Output: "    mix_test.go:99: specific initial failure\n" }));
parser.onLine(JSON.stringify({ Action: "fail", Package: "mix/pkg", Test: "TestAlreadyFailed", Elapsed: 0.055 }));

// 4. Test that starts and is interrupted by abnormal process termination
parser.onLine(JSON.stringify({ Action: "run", Package: "mix/pkg", Test: "TestInterrupted" }));
parser.onLine(JSON.stringify({ Action: "output", Package: "mix/pkg", Test: "TestInterrupted", Output: "in-progress output\n" }));

// Process terminates abnormally with another failure already existing
const failed = parser.finish({ code: 1, missing: false, tail: ["crash in runtime"] });
assert.equal(failed, true, "suite must report failure");

const state = store.getState();
const suite = state.suites[0];
assert.equal(suite.tests.length, 4, "all four tests must exist");

const p = suite.tests.find(t => t.id === "mix/pkg::TestPassed");
assert.ok(p, "TestPassed must exist");
assert.equal(p.status, "passed", "TestPassed must remain passed");
assert.equal(p.durationMs, 12, "TestPassed duration must be preserved");

const s = suite.tests.find(t => t.id === "mix/pkg::TestSkipped");
assert.ok(s, "TestSkipped must exist");
assert.equal(s.status, "skipped", "TestSkipped must remain skipped");

const f = suite.tests.find(t => t.id === "mix/pkg::TestAlreadyFailed");
assert.ok(f, "TestAlreadyFailed must exist");
assert.equal(f.status, "failed", "TestAlreadyFailed must remain failed");
assert.equal(f.durationMs, 55, "TestAlreadyFailed duration must be preserved");
assert.equal(f.failure?.location, "mix_test.go:99", "TestAlreadyFailed location must be preserved");
assert.ok(f.failure?.message.includes("specific initial failure"), "TestAlreadyFailed original message must be preserved");

const i = suite.tests.find(t => t.id === "mix/pkg::TestInterrupted");
assert.ok(i, "TestInterrupted must exist");
assert.equal(i.status, "failed", "TestInterrupted must be closed as failed");
assert.ok(i.failure?.message, "TestInterrupted must have diagnostic");

assert.equal(state.stats.running, 0, "stats.running must be 0");
assert.equal(state.stats.passed, 1, "stats.passed must be 1");
assert.equal(state.stats.failed, 2, "stats.failed must be 2");
assert.equal(state.status, "failed", "state.status must be failed");

console.log("PASS TestHCD027GoPreservesResults parser");
`)
	t.Log(out)

	// Part 2: End-to-end runner test with mixed output
	project, bin := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), "module synthetic\n")
	writeExec(t, filepath.Join(bin, "go"), `printf '%s\n' \
  '{"Action":"run","Package":"synthetic","Test":"TestOK"}' \
  '{"Action":"pass","Package":"synthetic","Test":"TestOK","Elapsed":0.01}' \
  '{"Action":"run","Package":"synthetic","Test":"TestBroken"}' \
  '{"Action":"output","Package":"synthetic","Test":"TestBroken","Output":"    broken_test.go:10: initial fail\n"}' \
  '{"Action":"fail","Package":"synthetic","Test":"TestBroken","Elapsed":0.02}' \
  '{"Action":"run","Package":"synthetic","Test":"TestHanging"}'
exit 1`)
	outE2E := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"},
		`process.env.PATH = `+jsStr(bin)+` + ":" + process.env.PATH;
import assert from "node:assert/strict";
`+tddToolHarness+`
const result = await run("backend", `+jsStr(project)+`);
const state = result.details;
const suite = suiteOf(state, "backend");
assert.ok(suite, "backend suite must exist");
const ok = suite.tests.find(t => t.name === "TestOK");
assert.equal(ok?.status, "passed", "TestOK must be passed");
const broken = suite.tests.find(t => t.name === "TestBroken");
assert.equal(broken?.status, "failed", "TestBroken must be failed");
assert.ok(broken?.failure?.message.includes("initial fail"), "TestBroken diagnostic preserved");
const hanging = suite.tests.find(t => t.name === "TestHanging");
assert.equal(hanging?.status, "failed", "TestHanging must be failed");
assert.equal(state.stats.running, 0, "stats.running must be 0");
assert.equal(state.stats.passed, 1, "stats.passed must be 1");
assert.equal(state.stats.failed, 2, "stats.failed must be 2");
console.log("PASS TestHCD027GoPreservesResults e2e");
`)
	t.Log(outE2E)
}

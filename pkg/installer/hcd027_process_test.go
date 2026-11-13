package installer

import (
	"testing"
)

func TestHCD027ProcessTerminationCauses(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import { streamLines } from "./gomemory-tdd.ts";

// 1. SIGTERM followed by exit 0
{
	const lines = [];
	const child = "process.on('SIGTERM', () => process.exit(0)); console.log('ready'); setInterval(() => {}, 1000);";
	const result = await streamLines(process.execPath, ["-e", child], { timeoutMs: 1000 }, l => lines.push(l));
	assert.ok(lines.includes("ready"), "child must be ready before timeout");
	assert.notEqual(result.code, 0, "timeout with exit 0 must be non-zero");
	assert.equal(result.timedOut, true, "timedOut must be true");
	assert.equal(result.missing, false, "missing must be false on timeout");
}

// 2. Abort prior to spawn
{
	const ac = new AbortController();
	ac.abort();
	const result = await streamLines(process.execPath, ["-e", "console.log('never')"], { signal: ac.signal }, () => {});
	assert.notEqual(result.code, 0, "abort prior to spawn must be non-zero");
	assert.equal(result.aborted, true, "aborted must be true");
	assert.equal(result.missing, false, "abort prior to spawn must not report missing: true");
}

// 3. Abort during execution
{
	const ac = new AbortController();
	const lines = [];
	const child = "console.log('ready'); setInterval(() => {}, 1000);";
	let ready = false;
	const promise = streamLines(process.execPath, ["-e", child], { signal: ac.signal }, l => {
		lines.push(l);
		if (l.includes("ready") && !ready) {
			ready = true;
			ac.abort();
		}
	});
	const result = await promise;
	assert.ok(ready, "child must reach ready before abort");
	assert.notEqual(result.code, 0, "abort during execution must be non-zero");
	assert.equal(result.aborted, true, "aborted must be true");
	assert.equal(result.missing, false, "abort during execution must not report missing");
}

// 4. Missing command distinction preserved
{
	const result = await streamLines("non_existent_binary_synth_987654", [], {}, () => {});
	assert.equal(result.code, 127, "missing binary must return code 127");
	assert.equal(result.missing, true, "missing binary must set missing: true");
	assert.equal(result.timedOut, false, "missing binary must not set timedOut");
	assert.equal(result.aborted, false, "missing binary must not set aborted");
}

console.log("PASS TestHCD027ProcessTerminationCauses");
`)
	t.Log(out)
}

func TestHCD027ProcessGroupCleanup(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import { streamLines } from "./gomemory-tdd.ts";

const isDead = async (pid) => {
	const start = Date.now();
	while (Date.now() - start < 1000) {
		try {
			process.kill(pid, 0);
			await new Promise(r => setTimeout(r, 50));
		} catch (e) {
			if (e.code === "ESRCH") return true;
		}
	}
	return false;
};

// 1. Hijo resistente que ignora SIGTERM y termina por escalado a SIGKILL dentro de 2s
let resistantPid = null;
const resistantChild = "process.on('SIGTERM', () => {}); console.log('RPID:' + process.pid); setInterval(() => {}, 1000);";
const startResistant = Date.now();
const res1 = await streamLines(process.execPath, ["-e", resistantChild], { timeoutMs: 500 }, l => {
	const m = l.match(/RPID:(\d+)/);
	if (m) resistantPid = Number(m[1]);
});
const elapsedResistant = Date.now() - startResistant;
assert.ok(resistantPid, "resistant child PID must be captured");
assert.ok(elapsedResistant < 2000, "resistant child must terminate within 2000ms grace period: took " + elapsedResistant + "ms");
assert.equal(res1.timedOut, true, "timedOut must be true");
assert.notEqual(res1.code, 0, "code must be non-zero");
const resistantDead = await isDead(resistantPid);
assert.ok(resistantDead, "resistant child process must be dead after escalation to SIGKILL");

// 2. Descendiente ordinario (grandchild) que hereda tuberías termina con el grupo
let grandchildPid = null;
const parentChild = "import { spawn } from 'node:child_process'; const sub = spawn(process.execPath, ['-e', 'setInterval(() => {}, 10000)'], { stdio: 'inherit' }); console.log('GPID:' + sub.pid); setInterval(() => {}, 10000);";
const startGroup = Date.now();
const res2 = await streamLines(process.execPath, ["--input-type=module", "-e", parentChild], { timeoutMs: 500 }, l => {
	const m = l.match(/GPID:(\d+)/);
	if (m) grandchildPid = Number(m[1]);
});
const elapsedGroup = Date.now() - startGroup;
assert.ok(grandchildPid, "grandchild PID must be captured");
assert.ok(elapsedGroup < 2000, "process group with descendant must terminate within 2000ms: took " + elapsedGroup + "ms");
const grandchildDead = await isDead(grandchildPid);
assert.ok(grandchildDead, "descendant process must be terminated when process group is killed");

// 3. No se señalizan grupos ajenos (el proceso actual sigue vivo)
assert.doesNotThrow(() => process.kill(process.pid, 0), "current test process group must not be killed");

// 4. La siguiente ejecución en cola empieza y concluye inmediatamente
const nextStart = Date.now();
const res3 = await streamLines(process.execPath, ["-e", "console.log('next queue execution')"], {}, () => {});
const nextElapsed = Date.now() - nextStart;
assert.equal(res3.code, 0, "next execution must succeed");
assert.ok(nextElapsed < 1000, "next queued execution starts immediately without hang");

console.log("PASS TestHCD027ProcessGroupCleanup");
`)
	t.Log(out)
}

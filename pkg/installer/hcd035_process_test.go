package installer

import "testing"

// hcd035ResistantFixture lanza, desde streamLines, un padre que crea un
// descendiente que ignora SIGTERM, permanece en el grupo de la suite y no
// comparte las tuberías del padre (H-03). Devuelve el PID del descendiente.
const hcd035ResistantFixture = `
import assert from "node:assert/strict";
import { streamLines } from "./gomemory-tdd.ts";

const grandchild = "process.on('SIGTERM',()=>{}); process.send('ready'); setInterval(()=>{},1000)";
const parent = "const {spawn}=require('node:child_process'); const c=spawn(process.execPath,['-e',"+JSON.stringify(grandchild)+"],{stdio:['ignore','ignore','ignore','ipc']}); c.on('message',()=>console.log('READY:'+c.pid)); setInterval(()=>{},1000)";

const alive = (pid) => { try { process.kill(pid, 0); return true; } catch (e) { if (e.code !== "ESRCH") throw e; return false; } };
const sleep = (ms) => new Promise(r => setTimeout(r, ms));
`

// C-07: cancelar mata al descendiente resistente 1500 ms después de la
// cancelación y el anfitrión sigue vivo más de 800 ms después del close.
func TestHCD035C07(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, hcd035ResistantFixture+`
const ac = new AbortController();
let pid, abortedAt;
try {
	const res = await streamLines(process.execPath, ["-e", parent], { signal: ac.signal, timeoutMs: 10000 }, line => {
		const m = line.match(/^READY:(\d+)/);
		if (m) { pid = Number(m[1]); abortedAt = Date.now(); ac.abort(); }
	});
	const closedAt = Date.now();
	assert.ok(pid, "el descendiente debe estar listo antes de cancelar");
	assert.equal(res.aborted, true);
	await sleep(Math.max(0, abortedAt + 1500 - Date.now()));
	assert.equal(alive(pid), false, "C-07: el descendiente resistente sigue vivo 1500 ms después de la cancelación");
	await sleep(Math.max(0, closedAt + 900 - Date.now()));
	assert.equal(alive(process.pid), true, "C-07: el anfitrión debe seguir vivo tras la escalada");
} finally { if (pid) { try { process.kill(pid, "SIGKILL"); } catch {} } }
console.log("PASS TestHCD035C07");
`)
	t.Log(out)
}

// C-08: al agotar el tiempo, el descendiente resistente tampoco sobrevive y el
// anfitrión sigue vivo más de 800 ms después del close.
func TestHCD035C08(t *testing.T) {
	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, hcd035ResistantFixture+`
const timeoutMs = 1500;
let pid;
try {
	const startedAt = Date.now();
	const res = await streamLines(process.execPath, ["-e", parent], { timeoutMs }, line => {
		const m = line.match(/^READY:(\d+)/);
		if (m) pid = Number(m[1]);
	});
	const closedAt = Date.now();
	assert.ok(pid, "el descendiente debe estar listo antes del timeout");
	assert.equal(res.timedOut, true);
	await sleep(Math.max(0, startedAt + timeoutMs + 1500 - Date.now()));
	assert.equal(alive(pid), false, "C-08: el descendiente resistente sigue vivo 1500 ms después del timeout");
	await sleep(Math.max(0, closedAt + 900 - Date.now()));
	assert.equal(alive(process.pid), true, "C-08: el anfitrión debe seguir vivo tras la escalada");
} finally { if (pid) { try { process.kill(pid, "SIGKILL"); } catch {} } }
console.log("PASS TestHCD035C08");
`)
	t.Log(out)
}

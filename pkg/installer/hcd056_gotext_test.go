package installer

import (
	"strings"
	"testing"
)

// HCD-056 tarea 13: en Pi real el agente lanza `go test -v` (texto, sin -json).
// Se clasificaba como exit-code: el panel no mostraba filas en vivo y contaba
// el comando entero como un único test (1/1 en vez de 5/5). Con -v cada test
// imprime «=== RUN» y «--- PASS/FAIL», así que se ingiere test a test.
func TestHCD056BashGoVerbose(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import { createTDDStore, createBashLiveIngest, classifyBashTestCommand } from "./gomemory-tdd.ts";
const fails = [];
const upd = (lines) => ({ content: [{ type: "text", text: lines.join("\n") }] });
const suite = (store) => store.getState().suites.find((s) => s.id === "backend");
const test = (store, name) => suite(store)?.tests.find((t) => t.name === name);

if (classifyBashTestCommand("go test -v -count=1 ./...")?.parser !== "go-text") fails.push("go test -v debe usar el parser de texto");
if (classifyBashTestCommand("go test ./...")?.parser !== "exit-code") fails.push("sin -v go test no lista los tests que pasan: sigue exit-code");
if (classifyBashTestCommand("go test -v -json ./...")?.parser !== "go-test") fails.push("con -json manda el parser JSON");

const store = createTDDStore();
const live = createBashLiveIngest(store, process.cwd());
live.start("v1", "go test -v -count=1 ./...");
const lines = [
	"=== RUN   TestSuma",
	"--- PASS: TestSuma (0.00s)",
	"=== RUN   TestLento1",
	"--- PASS: TestLento1 (3.00s)",
	"=== RUN   TestLento2",
];
live.update("v1", upd(lines));
if (test(store, "TestSuma")?.status !== "passed") fails.push("TestSuma debió pasar en vivo: " + test(store, "TestSuma")?.status);
if (test(store, "TestLento1")?.status !== "passed") fails.push("TestLento1 debió pasar en vivo");
const all = [...lines, "    suma_test.go:20: roto", "--- FAIL: TestLento2 (3.00s)", "FAIL", "FAIL\tdemo\t6.0s", ""];
live.update("v1", upd(all));
live.end("v1", true);
if (test(store, "TestLento2")?.status !== "failed") fails.push("TestLento2 debió fallar: " + test(store, "TestLento2")?.status);
if (suite(store)?.tests.length !== 3) fails.push("debe haber 3 filas, hay " + suite(store)?.tests.length);
if (suite(store)?.status !== "failed") fails.push("suite " + suite(store)?.status);
console.log(fails.length ? "FAIL " + fails.join(" | ") : "PASS");
`)
	if !strings.Contains(out, "PASS") {
		t.Fatalf("TestHCD056BashGoVerbose: %s", out)
	}
}

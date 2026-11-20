package installer

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func hcd052Servers(t *testing.T, piAgentDir string) map[string]json.RawMessage {
	t.Helper()
	obj := jsonKeys(t, filepath.Join(piAgentDir, "mcp.json"))
	raw, ok := obj["mcpServers"]
	if !ok {
		return map[string]json.RawMessage{}
	}
	servers := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &servers); err != nil {
		t.Fatal(err)
	}
	return servers
}

func TestHCD052LiteAndMCP(t *testing.T) {
	t.Run("C-003", func(t *testing.T) {
		t.Run("ida y vuelta estandar conserva entrada ajena", func(t *testing.T) {
			opts := sandbox(t)
			mcpPath := filepath.Join(opts.PiAgentDir, "mcp.json")
			foreign := json.RawMessage(`{"command":"synthetic-foreign","args":["--ok"]}`)
			writeFile(t, mcpPath, `{"mcpServers":{"mine":`+string(foreign)+`}}`)
			if err := InstallMCPServers(opts.PiAgentDir); err != nil {
				t.Fatal(err)
			}

			if err := SetLiteMode(opts, true); err != nil {
				t.Fatalf("lite on: %v", err)
			}
			afterOn := hcd052Servers(t, opts.PiAgentDir)
			if _, ok := afterOn["gomemory"]; !ok {
				t.Fatal("lite on debe conservar gomemory")
			}
			if _, ok := afterOn["codegraph"]; ok {
				t.Fatal("lite on debe retirar codegraph gestionado")
			}
			if _, ok := afterOn["codebase-memory"]; ok {
				t.Fatal("lite on debe retirar codebase-memory gestionado")
			}
			if !sameJSON(afterOn["mine"], foreign) {
				t.Fatalf("lite on alteró entrada ajena: %s", afterOn["mine"])
			}

			if err := SetLiteMode(opts, false); err != nil {
				t.Fatalf("lite off: %v", err)
			}
			afterOff := hcd052Servers(t, opts.PiAgentDir)
			for _, name := range []string{"gomemory", "codegraph", "codebase-memory"} {
				if _, ok := afterOff[name]; !ok {
					t.Fatalf("lite off no restauró %s", name)
				}
			}
			if !sameJSON(afterOff["mine"], foreign) {
				t.Fatalf("lite off alteró entrada ajena: %s", afterOff["mine"])
			}
		})

		t.Run("preserva definiciones personalizadas de codegraph y gomemory", func(t *testing.T) {
			opts := sandbox(t)
			mcpPath := filepath.Join(opts.PiAgentDir, "mcp.json")
			customGraph := json.RawMessage(`{"command":"custom-graph","args":["--synthetic-flag"]}`)
			customGM := json.RawMessage(`{"command":"custom-mem","args":["mcp","--synthetic"]}`)
			foreign := json.RawMessage(`{"command":"other-tool"}`)
			if err := InstallMCPServers(opts.PiAgentDir); err != nil {
				t.Fatal(err)
			}
			// Reemplaza codegraph y gomemory por configuraciones personalizadas; codebase-memory sigue gestionado.
			servers := hcd052Servers(t, opts.PiAgentDir)
			servers["codegraph"] = customGraph
			servers["gomemory"] = customGM
			servers["mine"] = foreign
			data, _ := json.MarshalIndent(map[string]any{"mcpServers": servers}, "", "  ")
			writeFile(t, mcpPath, string(data))

			if err := SetLiteMode(opts, true); err != nil {
				t.Fatalf("lite on: %v", err)
			}
			afterOn := hcd052Servers(t, opts.PiAgentDir)
			if !sameJSON(afterOn["codegraph"], customGraph) {
				t.Fatalf("C-003: lite on borró o alteró codegraph personalizado: %s", afterOn["codegraph"])
			}
			if !sameJSON(afterOn["gomemory"], customGM) {
				t.Fatalf("C-003: lite on alteró gomemory personalizado: %s", afterOn["gomemory"])
			}
			if _, ok := afterOn["codebase-memory"]; ok {
				t.Fatal("lite on debió retirar codebase-memory gestionado")
			}
			if !sameJSON(afterOn["mine"], foreign) {
				t.Fatalf("lite on alteró entrada ajena: %s", afterOn["mine"])
			}

			if err := SetLiteMode(opts, false); err != nil {
				t.Fatalf("lite off: %v", err)
			}
			afterOff := hcd052Servers(t, opts.PiAgentDir)
			if !sameJSON(afterOff["codegraph"], customGraph) {
				t.Fatalf("C-003: lite off sobrescribió codegraph personalizado: %s", afterOff["codegraph"])
			}
			if !sameJSON(afterOff["gomemory"], customGM) {
				t.Fatalf("C-003: lite off sobrescribió gomemory personalizado: %s", afterOff["gomemory"])
			}
			if _, ok := afterOff["codebase-memory"]; !ok {
				t.Fatal("lite off debió restaurar codebase-memory gestionado")
			}
			if !sameJSON(afterOff["mine"], foreign) {
				t.Fatalf("lite off alteró entrada ajena: %s", afterOff["mine"])
			}
		})

		t.Run("no pisa ediciones manuales hechas entre lite on y lite off", func(t *testing.T) {
			opts := sandbox(t)
			mcpPath := filepath.Join(opts.PiAgentDir, "mcp.json")
			if err := InstallMCPServers(opts.PiAgentDir); err != nil {
				t.Fatal(err)
			}
			if err := SetLiteMode(opts, true); err != nil {
				t.Fatal(err)
			}
			// El usuario añade manualmente un codegraph personalizado mientras está en lite on.
			editedGraph := json.RawMessage(`{"command":"edited-between","args":["--v2"]}`)
			servers := hcd052Servers(t, opts.PiAgentDir)
			servers["codegraph"] = editedGraph
			data, _ := json.MarshalIndent(map[string]any{"mcpServers": servers}, "", "  ")
			writeFile(t, mcpPath, string(data))

			if err := SetLiteMode(opts, false); err != nil {
				t.Fatal(err)
			}
			afterOff := hcd052Servers(t, opts.PiAgentDir)
			if !sameJSON(afterOff["codegraph"], editedGraph) {
				t.Fatalf("C-003: lite off pisó la edición hecha entre ambos comandos: %s", afterOff["codegraph"])
			}
			if _, ok := afterOff["codebase-memory"]; !ok {
				t.Fatal("lite off debió restaurar codebase-memory ausente")
			}
		})
	})

	t.Run("C-001", func(t *testing.T) {
		t.Run("mcpServers invalido no persiste lite on", func(t *testing.T) {
			opts := sandbox(t)
			mcpPath := filepath.Join(opts.PiAgentDir, "mcp.json")
			const invalid = `{"mcpServers":[]}`
			writeFile(t, mcpPath, invalid)

			if err := SetLiteMode(opts, true); err == nil {
				t.Fatal("lite on con mcpServers lista debe fallar")
			}
			if IsLiteMode(opts.HomeDir) {
				t.Fatal("C-001: lite.json quedó en true tras fallo de mcp.json")
			}
			if _, err := os.Stat(LiteStatePath(opts.HomeDir)); !os.IsNotExist(err) {
				t.Fatalf("C-001: no debió quedar lite.json tras fallo inicial, err=%v", err)
			}
			if got := readFile(t, mcpPath); got != invalid {
				t.Fatalf("mcp.json cambió tras error: %q", got)
			}
		})

		t.Run("error en lite off conserva lite on previo", func(t *testing.T) {
			opts := sandbox(t)
			if err := SetLiteMode(opts, true); err != nil {
				t.Fatal(err)
			}
			mcpPath := filepath.Join(opts.PiAgentDir, "mcp.json")
			writeFile(t, mcpPath, "{")

			if err := SetLiteMode(opts, false); err == nil {
				t.Fatal("lite off con mcp.json inválido debe fallar")
			}
			if !IsLiteMode(opts.HomeDir) {
				t.Fatal("C-001: lite off fallido degradó lite.json a false")
			}
		})

		t.Run("fallo al escribir estado restaura mcp.json", func(t *testing.T) {
			hcd047SkipRoot(t)
			opts := sandbox(t)
			if err := InstallMCPServers(opts.PiAgentDir); err != nil {
				t.Fatal(err)
			}
			mcpPath := filepath.Join(opts.PiAgentDir, "mcp.json")
			origMCP := readFile(t, mcpPath)
			stateDir := filepath.Dir(LiteStatePath(opts.HomeDir))
			mkdir(t, stateDir)
			if err := os.Chmod(stateDir, 0555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(stateDir, 0755) })

			if err := SetLiteMode(opts, true); err == nil {
				t.Fatal("lite on con directorio de estado solo lectura debe fallar")
			}
			if got := readFile(t, mcpPath); got != origMCP {
				t.Fatalf("C-001: mcp.json no se restauró al fallar la escritura de lite.json:\ngot=%s\nwant=%s", got, origMCP)
			}
			if IsLiteMode(opts.HomeDir) {
				t.Fatal("C-001: IsLiteMode no debe ser true tras fallo")
			}
		})

		t.Run("fallo al escribir mcp.json conserva estado anterior", func(t *testing.T) {
			hcd047SkipRoot(t)
			opts := sandbox(t)
			if err := InstallMCPServers(opts.PiAgentDir); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(opts.PiAgentDir, 0555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(opts.PiAgentDir, 0755) })

			if err := SetLiteMode(opts, true); err == nil {
				t.Fatal("lite on con PiAgentDir solo lectura debe fallar")
			}
			if IsLiteMode(opts.HomeDir) {
				t.Fatal("C-001: lite.json quedó en true tras fallo al escribir mcp.json")
			}
		})

		t.Run("lite.json corrupto falla sin mutar mcp.json", func(t *testing.T) {
			opts := sandbox(t)
			if err := InstallMCPServers(opts.PiAgentDir); err != nil {
				t.Fatal(err)
			}
			mcpPath := filepath.Join(opts.PiAgentDir, "mcp.json")
			origMCP := readFile(t, mcpPath)
			writeFile(t, LiteStatePath(opts.HomeDir), "{corrupt")

			if _, err := GetLiteStatus(opts.HomeDir); err == nil {
				t.Fatal("GetLiteStatus debe fallar con lite.json corrupto")
			}
			if err := SetLiteMode(opts, true); err == nil {
				t.Fatal("SetLiteMode debe fallar con lite.json corrupto")
			}
			if err := SetCodemodePreference(opts, true); err == nil {
				t.Fatal("SetCodemodePreference debe fallar con lite.json corrupto")
			}
			if got := readFile(t, mcpPath); got != origMCP {
				t.Fatalf("mcp.json cambió con lite.json corrupto: %s", got)
			}
		})
	})

	t.Run("C-002", func(t *testing.T) {
		t.Run("lite on + codemode on mantiene solo gomemory en codemode", func(t *testing.T) {
			opts := sandbox(t)
			if err := InstallMCPServers(opts.PiAgentDir); err != nil {
				t.Fatal(err)
			}
			if err := SetLiteMode(opts, true); err != nil {
				t.Fatal(err)
			}
			if err := SetCodemodePreference(opts, true); err != nil {
				t.Fatal(err)
			}
			servers := hcd052Servers(t, opts.PiAgentDir)
			if _, ok := servers["codegraph"]; ok {
				t.Fatal("C-002: codemode on reintrodujo codegraph con lite activo")
			}
			if _, ok := servers["codebase-memory"]; ok {
				t.Fatal("C-002: codemode on reintrodujo codebase-memory con lite activo")
			}
			wantGM, _ := json.Marshal(CodemodeMCPServers()["gomemory"])
			if !sameJSON(servers["gomemory"], wantGM) {
				t.Fatalf("C-002: gomemory no quedó en codemode: %s", servers["gomemory"])
			}
			st, err := GetCodemodeStatus(opts.PiAgentDir)
			if err != nil {
				t.Fatal(err)
			}
			if !st.Enabled || !st.MCPEnabled || st.Partial {
				t.Fatalf("C-002: GetCodemodeStatus inconsistente con lite+codemode: %+v", st)
			}

			// Al apagar lite, codegraph y codebase-memory se restauran en perfil codemode.
			if err := SetLiteMode(opts, false); err != nil {
				t.Fatal(err)
			}
			full := hcd052Servers(t, opts.PiAgentDir)
			wantCBM, _ := json.Marshal(CodemodeMCPServers()["codebase-memory"])
			if !sameJSON(full["codebase-memory"], wantCBM) {
				t.Fatalf("C-002: lite off no restauró codebase-memory en codemode: %s", full["codebase-memory"])
			}
			if _, ok := full["codegraph"]; !ok {
				t.Fatal("C-002: lite off no restauró codegraph")
			}
		})

		t.Run("codemode on + lite on + codemode off mantiene lite", func(t *testing.T) {
			opts := sandbox(t)
			if err := SetCodemodePreference(opts, true); err != nil {
				t.Fatal(err)
			}
			if err := SetLiteMode(opts, true); err != nil {
				t.Fatal(err)
			}
			servers := hcd052Servers(t, opts.PiAgentDir)
			wantHybridGM, _ := json.Marshal(CodemodeMCPServers()["gomemory"])
			if !sameJSON(servers["gomemory"], wantHybridGM) {
				t.Fatalf("C-002: lite on perdió exposición codemode de gomemory: %s", servers["gomemory"])
			}
			if err := SetCodemodePreference(opts, false); err != nil {
				t.Fatal(err)
			}
			serversOff := hcd052Servers(t, opts.PiAgentDir)
			if _, ok := serversOff["codegraph"]; ok {
				t.Fatal("C-002: codemode off reintrodujo codegraph con lite activo")
			}
			if _, ok := serversOff["codebase-memory"]; ok {
				t.Fatal("C-002: codemode off reintrodujo codebase-memory con lite activo")
			}
			wantDirectGM, _ := json.Marshal(DefaultMCPServers()["gomemory"])
			if !sameJSON(serversOff["gomemory"], wantDirectGM) {
				t.Fatalf("C-002: gomemory no pasó a direct tras codemode off: %s", serversOff["gomemory"])
			}
		})

		t.Run("install/update (applyCodemodeDefault) respeta lite en direct y codemode", func(t *testing.T) {
			// Direct + lite
			opts := sandbox(t)
			if err := SetLiteMode(opts, true); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := applyCodemodeDefault(opts, &out); err != nil {
				t.Fatal(err)
			}
			servers := hcd052Servers(t, opts.PiAgentDir)
			if _, ok := servers["codegraph"]; ok {
				t.Fatal("C-002: applyCodemodeDefault reintrodujo codegraph con lite activo")
			}
			if _, ok := servers["codebase-memory"]; ok {
				t.Fatal("C-002: applyCodemodeDefault reintrodujo codebase-memory con lite activo")
			}

			// Codemode + lite + entrada ajena
			opts2 := sandbox(t)
			foreign := json.RawMessage(`{"command":"foreign-svc"}`)
			writeFile(t, filepath.Join(opts2.PiAgentDir, "mcp.json"), `{"mcpServers":{"mine":`+string(foreign)+`}}`)
			if err := SetCodemodePreference(opts2, true); err != nil {
				t.Fatal(err)
			}
			if err := SetLiteMode(opts2, true); err != nil {
				t.Fatal(err)
			}
			out.Reset()
			if err := applyCodemodeDefault(opts2, &out); err != nil {
				t.Fatal(err)
			}
			s2 := hcd052Servers(t, opts2.PiAgentDir)
			if _, ok := s2["codegraph"]; ok || s2["codebase-memory"] != nil {
				t.Fatalf("C-002: applyCodemodeDefault rompió lite+codemode: %v", s2)
			}
			wantHybridGM, _ := json.Marshal(CodemodeMCPServers()["gomemory"])
			if !sameJSON(s2["gomemory"], wantHybridGM) {
				t.Fatalf("C-002: gomemory no conservó codemode en install: %s", s2["gomemory"])
			}
			if !sameJSON(s2["mine"], foreign) {
				t.Fatalf("C-002: install alteró servidor ajeno: %s", s2["mine"])
			}

			// Subcomando mcp (InstallMCPServersForOptions) también compone lite + codemode.
			if err := InstallMCPServersForOptions(opts2); err != nil {
				t.Fatal(err)
			}
			s3 := hcd052Servers(t, opts2.PiAgentDir)
			if _, ok := s3["codegraph"]; ok || s3["codebase-memory"] != nil {
				t.Fatalf("C-002: InstallMCPServersForOptions rompió lite+codemode: %v", s3)
			}
			if !sameJSON(s3["gomemory"], wantHybridGM) {
				t.Fatalf("C-002: InstallMCPServersForOptions alteró gomemory: %s", s3["gomemory"])
			}
		})
	})
}

func TestHCD052TDDBashIngest(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tdd.ts"}, `
import assert from "node:assert/strict";
import { classifyBashTestCommand, createTDDStore, ingestBashTestResult } from "./gomemory-tdd.ts";

// 1. Menciones en texto citado, argumentos de printf/echo o comentarios no disparan ingesta
for (const benign of [
	"printf 'go test ./...\\n'",
	"echo \"pytest -q\"",
	"git commit -m 'run vitest and mem hcd verify'",
	"# go test ./...",
	"go test ./... | cat",
	"pytest -q || true",
	"pytest -q; echo done",
]) {
	assert.equal(classifyBashTestCommand(benign), null, "no debe clasificar comando ambiguo o citado: " + benign);
}

// 2. Coincidencia ambigua o citada conserva resultados previos del store sin limpiar la suite
{
	const store = createTDDStore();
	store.initSuites([{ id: "backend", name: "Backend", domain: "backend" }]);
	store.startTest("backend", "prev::t1", "prev_fail");
	store.finishTest("backend", "prev::t1", "failed", 12, { message: "real failure" });
	store.finishSuite("backend", "failed");

	const ingested = ingestBashTestResult(store, "printf 'go test ./...\\n'", "go test ./...", false, process.cwd());
	assert.equal(ingested, false, "printf con go test no debe ingerirse");
	const backend = store.getState().suites.find((s) => s.id === "backend");
	assert.equal(backend.status, "failed", "debe conservar el fallo previo de backend");
	assert.equal(backend.tests.length, 1);
	assert.equal(backend.tests[0].id, "prev::t1");
}

// 3. Ejecución auténtica parcial queda identificada como parcial
{
	const partialGo = classifyBashTestCommand("go test ./pkg/installer -run TestFoo -v");
	assert.ok(partialGo, "go test dirigido debe reconocerse");
	assert.equal(partialGo.partial, true, "go test -run debe marcarse como parcial");

	const fullGo = classifyBashTestCommand("go test ./...");
	assert.ok(fullGo, "go test ./... debe reconocerse");
	assert.equal(fullGo.partial, false, "go test ./... es completo, no parcial");

	const partialPy = classifyBashTestCommand("uv run pytest tests/test_api.py -k test_ok");
	assert.ok(partialPy, "uv run pytest dirigido debe reconocerse");
	assert.equal(partialPy.partial, true, "pytest con ruta/-k debe marcarse como parcial");

	const store = createTDDStore();
	assert.equal(
		ingestBashTestResult(store, "uv run pytest tests/test_api.py -k test_ok", "tests/test_api.py::test_ok PASSED", false, process.cwd()),
		true,
	);
	const backend = store.getState().suites.find((s) => s.id === "backend");
	assert.equal(backend.partial, true, "suite debe reflejar ejecución parcial");
}

// 4. Error externo (isError=true) prevalece sobre evidencia positiva contradictoria en HCD y parsers
{
	const store = createTDDStore();
	const contradictoryHCD = JSON.stringify({
		ok: true,
		unit_id: "HCD-052",
		data: { evidence: { passed: true, excerpt: "ok" } },
	});
	assert.equal(
		ingestBashTestResult(store, "mem hcd verify --task 1 --json -- 'go test ./...'", contradictoryHCD, true, process.cwd()),
		true,
	);
	const data = store.getState().suites.find((s) => s.id === "data");
	assert.equal(data.status, "failed", "isError=true debe prevalecer sobre evidence.passed=true");
	assert.ok(data.tests.some((t) => t.status === "failed"), "debe registrar test fallido ante isError=true");

	// También con pytest devolviendo líneas PASSED pero isError=true
	const store2 = createTDDStore();
	assert.equal(
		ingestBashTestResult(store2, "pytest -v", "tests/test_a.py::test_ok PASSED", true, process.cwd()),
		true,
	);
	const backend = store2.getState().suites.find((s) => s.id === "backend");
	assert.equal(backend.status, "failed", "pytest con isError=true debe dejar backend en failed");
	assert.ok(backend.tests.some((t) => t.status === "failed"), "debe haber al menos un test fallido cuando isError=true");
}
`)
	t.Log(out)
}

func TestHCD052TasksVerifyGuard(t *testing.T) {
	t.Parallel()

	out := runNodeExt(t, []string{"extensions/gomemory-tasks.ts"}, `
import assert from "node:assert/strict";
import { isBundledHcdVerifyCommand } from "./gomemory-tasks.ts";

// 1. Texto entre comillas y comentarios no deben bloquear
for (const allowed of [
	"printf 'mem hcd verify --task 1\\nmem hcd verify --task 2\\n'",
	"echo \"mem hcd verify && mem hcd verify\"",
	"# mem hcd verify --task 1\\n# mem hcd verify --task 2\\nmem hcd verify --task 1 --json -- 'go test ./...'",
	"mem hcd verify --task 1 --json -- 'printf \"mem hcd verify\"'",
	"printf 'for i in 1 2; do echo ok; done\\n' && mem hcd verify --task 1 --json -- 'go test ./...'",
	"for i in 1 2; do echo \"$i\"; done && mem hcd verify --task 1 --json -- 'go test ./...'",
	"echo 'v() { mem hcd verify; }'",
]) {
	assert.equal(isBundledHcdVerifyCommand(allowed), null, "no debe bloquear comando legítimo: " + allowed);
}

// 2. Dos invocaciones efectivas o agrupación estructural siguen bloqueando
for (const blocked of [
	"mem hcd verify --task 1 --json -- 'a' && mem hcd verify --task 2 --json -- 'b'",
	"mem hcd verify --task 1 --json -- 'a'; ./mem hcd verify --task 2 --json -- 'b'",
	"v() { mem hcd verify --task $1 --json -- $2; }; v 1 'a'",
	"function run_verify { mem hcd verify --task $1 --json -- $2; }; run_verify 1 'a'",
	"function run_verify() { mem hcd verify --task $1 --json -- $2; }; run_verify 1 'a'",
	"for i in 1 2; do mem hcd verify --task $i --json -- 'ok'; done",
	"while read t; do mem hcd verify --task $t --json -- 'ok'; done",
]) {
	assert.ok(isBundledHcdVerifyCommand(blocked), "debe bloquear agrupación efectiva: " + blocked);
}
`)
	t.Log(out)
}

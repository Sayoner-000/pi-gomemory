package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// hcd037Cases es la tabla de C-01: la regla de Go que deben seguir el
// launcher, el pie y deps-refresh (HCD-037, Q1).
var hcd037Cases = []struct {
	value    string
	disabled bool
}{
	{"", false}, {" ", false}, {"0", false}, {" 0 ", false},
	{"false", false}, {"FALSE", false}, {"False", false},
	{"1", true}, {"yes", true}, {"true", true}, {"no", true},
}

// C-01 (guarda sin rojo): la regla de referencia en Go.
func TestHCD037GoRule(t *testing.T) {
	for _, c := range hcd037Cases {
		t.Setenv(updateCheckEnv, c.value)
		if got := UpdateCheckDisabled(); got != c.disabled {
			t.Errorf("Go con %q: desactivada=%v, quiero %v", c.value, got, c.disabled)
		}
	}
}

// C-01, C-04: el launcher aplica la misma regla con update_check_disabled.
func TestHCD037Launcher(t *testing.T) {
	s := GenerateHarnessScript()
	if strings.Contains(s, `[ -z "${PI_GOMEMORY_NO_UPDATE_CHECK:-}" ]`) {
		t.Fatal("C-04: el launcher conserva la condición [ -z ], que desactiva con =0")
	}
	start := strings.Index(s, "update_check_disabled() {")
	if start < 0 || !strings.Contains(s, "! update_check_disabled") {
		t.Fatal("C-04: el launcher no define ni usa update_check_disabled")
	}
	end := strings.Index(s[start:], "\n}\n")
	if end < 0 {
		t.Fatal("C-04: no se encuentra el final de update_check_disabled")
	}
	fn := s[start : start+end+3]
	bash := "/bin/bash" // el bash 3.2 de macOS es el mínimo que se admite
	if _, err := os.Stat(bash); err != nil {
		bash = "bash"
	}
	for _, c := range hcd037Cases {
		cmd := exec.Command(bash, "-c", fn+"if update_check_disabled; then echo off; else echo on; fi")
		cmd.Env = append(os.Environ(), updateCheckEnv+"="+c.value)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bash: %v\n%s", err, out)
		}
		want := map[bool]string{true: "off", false: "on"}[c.disabled]
		if got := strings.TrimSpace(string(out)); got != want {
			t.Errorf("C-01: launcher con %q: %s, quiero %s", c.value, got, want)
		}
	}
}

// C-01, C-02: el pie aplica la misma regla y, con =0, consulta deps-status.
func TestHCD037Footer(t *testing.T) {
	table := make([]map[string]any, 0, len(hcd037Cases))
	for _, c := range hcd037Cases {
		table = append(table, map[string]any{"Value": c.value, "Disabled": c.disabled})
	}
	cases, _ := json.Marshal(table)
	out := runNodeExt(t, []string{"extensions/gomemory-status.ts"}, fmt.Sprintf(`
import assert from "node:assert/strict";
import status, { updateCheckDisabled } from "./gomemory-status.ts";

const cases = %s;
for (const c of cases) {
	assert.equal(updateCheckDisabled({ PI_GOMEMORY_NO_UPDATE_CHECK: c.Value }), c.Disabled, "C-01: pie con " + JSON.stringify(c.Value));
}

async function openSession(value) {
	process.env.PI_GOMEMORY_NO_UPDATE_CHECK = value;
	const handlers = {}, calls = [];
	status({
		on: (event, fn) => { handlers[event] = fn; },
		registerCommand() {},
		exec: async (_cmd, args) => { calls.push(args[0]); return { code: 0 }; },
	});
	await handlers.session_start({}, { ui: { setFooter() {}, setStatus() {}, notify() {} }, sessionManager: { getBranch: () => [] } });
	await new Promise((r) => setTimeout(r, 50));
	return calls;
}
assert.ok((await openSession("0")).includes("deps-status"), "C-02: con =0 el pie debe consultar deps-status");
assert.equal((await openSession("1")).length, 0, "C-02: con =1 el pie no debe consultar nada");
console.log("PASS TestHCD037Footer");
`, cases))
	t.Log(out)
}

// C-03: con la comprobación desactivada, RefreshDeps no consulta ni escribe la caché.
func TestHCD037DepsRefresh(t *testing.T) {
	var queries int32
	probes := []DepProbe{{
		Name:    "x",
		Current: func(context.Context) (string, error) { return "v1.0.0", nil },
		Latest: func(context.Context) (string, error) {
			atomic.AddInt32(&queries, 1)
			return "v1.1.0", nil
		},
	}}
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	cache := filepath.Join(t.TempDir(), depsCacheFile)
	old, _ := json.Marshal(depsCache{CheckedAt: now.Add(-2 * time.Hour), Deps: []DepState{{Name: "x", Current: "v1.0.0", Latest: "v1.0.0"}}})
	writeFile(t, cache, string(old))

	t.Setenv(updateCheckEnv, "1")
	states := RefreshDeps(probes, cache, now)
	if queries != 0 {
		t.Fatalf("C-03: con la comprobación desactivada se consultó la red %d veces", queries)
	}
	if got := readFile(t, cache); !bytes.Equal([]byte(got), old) {
		t.Fatalf("C-03: la caché cambió: %s", got)
	}
	if len(states) != 1 || states[0].Name != "x" {
		t.Fatalf("C-03: debía devolver la caché tal cual: %+v", states)
	}
	missing := filepath.Join(t.TempDir(), depsCacheFile)
	if states := RefreshDeps(probes, missing, now); len(states) != 0 || queries != 0 {
		t.Fatalf("C-03: sin caché debía devolver vacío sin consultar: %+v q=%d", states, queries)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("C-03: no debía crearse la caché: %v", err)
	}

	t.Setenv(updateCheckEnv, "0")
	RefreshDeps(probes, cache, now)
	if queries == 0 {
		t.Fatal("C-03: con =0 la comprobación sigue activa y debe consultar")
	}
}

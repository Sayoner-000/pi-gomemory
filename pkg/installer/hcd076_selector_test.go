package installer

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// HCD-076 VS-04: selector de casillas en la terminal.

func withOpenTTY(t *testing.T, input string, calls *int) {
	t.Helper()
	old := openTTY
	openTTY = func() io.ReadCloser {
		*calls++
		return io.NopCloser(strings.NewReader(input))
	}
	t.Cleanup(func() { openTTY = old })
}

// T16 · C-09: la lista muestra nombre, versiones y obligatoriedad; los
// obligatorios no se desmarcan; desmarcar herdr lo deja fuera.
func TestHCD076Selector(t *testing.T) {
	tools := []Tool{{Name: "pi", Required: true}, {Name: "codegraph"}, {Name: "herdr"}}
	states := []DepState{
		{Name: "pi", Current: "1.0.0", Latest: "1.2.0"},
		{Name: "codegraph", Current: "1.0.0", Latest: "1.0.0"},
	}
	sel := map[string]bool{"pi": true, "codegraph": true, "herdr": true}
	var out bytes.Buffer
	got, err := runSelector(tools, sel, states, strings.NewReader("1 3\n\n"), &out)
	if err != nil {
		t.Fatal(err)
	}
	if !got["pi"] || !got["codegraph"] || got["herdr"] {
		t.Fatalf("selección = %v\n%s", got, out.String())
	}
	s := out.String()
	for _, want := range []string{"pi", "1.0.0 → 1.2.0", "obligatorio", "codegraph", "opcional", "herdr", "no instalado", "[x]", "[ ]", "no se puede desmarcar"} {
		if !strings.Contains(s, want) {
			t.Fatalf("falta %q en el selector:\n%s", want, s)
		}
	}
	// Enter sin cambios conserva la selección por defecto; en lite, los
	// opcionales llegan desmarcados y aparecen así.
	out.Reset()
	got, _ = runSelector(tools, map[string]bool{"pi": true, "codegraph": false, "herdr": false}, states, strings.NewReader("\n"), &out)
	if !got["pi"] || got["codegraph"] || got["herdr"] || !strings.Contains(out.String(), "[ ] codegraph") {
		t.Fatalf("lite: %v\n%s", got, out.String())
	}
}

// T17 · C-10, C-13: el selector solo aparece con terminal y sin flags; en
// install (curl | bash) se lee de /dev/tty y desmarcar herdr lo omite.
func TestHCD076SelectorGating(t *testing.T) {
	opts := toolSandbox(t)
	fakeTool(t, opts.LocalBinDir, "pi")
	withTools(t, []Tool{
		{Name: "pi", Required: true, Script: "https://invalid.example/pi.sh", Interp: "sh", Manual: "m"},
		{Name: "herdr", Script: "https://invalid.example/h.sh", Interp: "sh", Manual: "m"},
	})
	calls := 0
	withOpenTTY(t, "2\n\n", &calls)
	if code := RunSync(opts, []string{"--yes"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 || calls != 0 {
		t.Fatalf("--yes no debe abrir la terminal: code=%d calls=%d", code, calls)
	}
	var out bytes.Buffer
	if code := RunSync(opts, nil, &out, &bytes.Buffer{}); code != 0 || calls != 1 {
		t.Fatalf("sin flags con terminal debe mostrar el selector: code=%d calls=%d", code, calls)
	}
	if !strings.Contains(out.String(), "[ ] herdr") {
		t.Fatalf("herdr desmarcado no debe instalarse:\n%s", out.String())
	}
	old := InstallSyncOptions
	t.Cleanup(func() { InstallSyncOptions = old })
	InstallSyncOptions = SyncOptions{}
	if err := InstallAll(repoAssets(t), opts); err != nil || calls != 2 {
		t.Fatalf("install con terminal debe mostrar el selector: err=%v calls=%d", err, calls)
	}
	InstallSyncOptions = SyncOptions{Yes: true}
	if err := InstallAll(repoAssets(t), opts); err != nil || calls != 2 {
		t.Fatalf("install relanzado por update no debe mostrar el selector: err=%v calls=%d", err, calls)
	}
	openTTY = func() io.ReadCloser { return nil }
	var noTTY bytes.Buffer
	res, err := Sync(opts, SyncOptions{}, nil, &noTTY)
	if err != nil || actions(res)["herdr"] != "failed" {
		t.Fatalf("sin terminal se reconcilian todos sin bloquearse: %v %v", actions(res), err)
	}
}

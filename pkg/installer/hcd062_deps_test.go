package installer

import (
	"bytes"
	"strings"
	"testing"
)

// RunDepsInstall informa que codebase-memory-mcp fue retirado y devuelve código 2
// sin descargar nada.
func TestHCD062DepsInstallRetired(t *testing.T) {
	opts := toolSandbox(t)
	srv, hits := scriptServer(t, map[string]string{})
	withToolHooks(t, testHooks(srv))
	for _, args := range [][]string{nil, {"pi"}, {"codegraph"}, {"codebase-memory-mcp"}} {
		var stderr bytes.Buffer
		if code := RunDepsInstall(opts, args, &bytes.Buffer{}, &stderr); code != 2 || !strings.Contains(stderr.String(), "retirado") {
			t.Fatalf("args %v: code %d, stderr %q", args, code, stderr.String())
		}
	}
	if len(*hits) != 0 {
		t.Fatalf("no debe descargar nada: %v", *hits)
	}
}

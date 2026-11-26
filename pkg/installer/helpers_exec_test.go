package installer

import (
	"encoding/json"
	"os"
	"testing"
)

// Ayudantes compartidos por los tests de extensiones; vivían en
// hcd020_tdd_real_test.go hasta que HCD-085 retiró /tdd.

// jsStr devuelve s como literal de cadena JavaScript válido.
func jsStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// writeExec escribe un ejecutable de shell (comando falso en PATH).
func writeExec(t *testing.T, path, body string) {
	t.Helper()
	writeFile(t, path, "#!/bin/sh\n"+body+"\n")
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatal(err)
	}
}

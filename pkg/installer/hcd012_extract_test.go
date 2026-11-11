package installer

import (
	"strings"
	"testing"
)

// C-07: extractBinary no copia el tarball a string.
func TestHCD012Extract(t *testing.T) {
	if strings.Contains(readFile(t, "update.go"), "string(tarball)") {
		t.Fatal("extractBinary todavía convierte el tarball a string")
	}
	bin, err := extractBinary(makeTarball(t, []byte("NEW")), "pi-gomemory-setup")
	if err != nil || string(bin) != "NEW" {
		t.Fatalf("bin=%q err=%v", bin, err)
	}
}

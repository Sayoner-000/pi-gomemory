package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// tarballWithMember construye un tarball cuyo miembro pi-gomemory-setup declara
// size bytes y contiene data; si len(data) < size, el tar queda truncado.
func tarballWithMember(t *testing.T, size int64, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "pi-gomemory-setup", Mode: 0755, Size: size, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(data); err != nil {
		t.Fatal(err)
	}
	if int64(len(data)) == size {
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func withMaxBinaryBytes(t *testing.T, n int64) {
	t.Helper()
	old := maxBinaryBytes
	maxBinaryBytes = n
	t.Cleanup(func() { maxBinaryBytes = old })
}

// C-09: extractBinary acepta exactamente el límite y rechaza más, vacío y truncado.
func TestHCD034C09(t *testing.T) {
	const limit = 1 << 20
	withMaxBinaryBytes(t, limit)

	exact := bytes.Repeat([]byte("a"), limit)
	bin, err := extractBinary(tarballWithMember(t, limit, exact), "pi-gomemory-setup")
	if err != nil || len(bin) != limit {
		t.Fatalf("exactamente el límite: len=%d err=%v", len(bin), err)
	}

	cases := map[string][]byte{
		"mayor que el límite": tarballWithMember(t, limit+1, bytes.Repeat([]byte("a"), limit+1)),
		"vacío":               tarballWithMember(t, 0, nil),
		"truncado":            tarballWithMember(t, 1000, bytes.Repeat([]byte("a"), 10)),
	}
	for name, tb := range cases {
		if bin, err := extractBinary(tb, "pi-gomemory-setup"); err == nil {
			t.Errorf("%s: debía rechazarse, devolvió %d bytes", name, len(bin))
		}
	}
}

// C-10: DownloadAndReplace no toca el binario si el ejecutable supera el límite.
func TestHCD034C10(t *testing.T) {
	const limit = 1 << 20
	withMaxBinaryBytes(t, limit)

	tarball := tarballWithMember(t, limit+1, bytes.Repeat([]byte("a"), limit+1))
	h := sha256.Sum256(tarball)
	srv := newDownloadServer(t, "v9.9.9", tarball, hex.EncodeToString(h[:]))
	defer srv.Close()

	target := filepath.Join(t.TempDir(), "pi-gomemory-setup")
	old := []byte("OLD-BINARY\x00\x01")
	if err := os.WriteFile(target, old, 0755); err != nil {
		t.Fatal(err)
	}
	err := DownloadAndReplace(UpdateConfig{BaseURL: srv.URL, Tag: "v9.9.9", GOOS: "linux", GOARCH: "amd64", Target: target})
	if err == nil {
		t.Fatal("DownloadAndReplace debía fallar con un ejecutable mayor que el límite")
	}
	got, _ := os.ReadFile(target)
	if !bytes.Equal(got, old) {
		t.Fatalf("el binario destino cambió: %q", got)
	}
}

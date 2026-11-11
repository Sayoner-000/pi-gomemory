package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.3.0", "v1.2.0", true},
		{"v1.2.1", "v1.2.0", true},
		{"v2.0.0", "v1.9.9", true},
		{"v1.10.0", "v1.9.0", true},
		{"v1.2.0", "v1.2.0", false},
		{"v1.1.0", "v1.2.0", false},
		{"1.3.0", "v1.2.0", true},
		{"v1.3.0", "dev", false},
		{"garbage", "v1.2.0", false},
		{"v1.3.0-rc1", "v1.2.0", false},
	}
	for _, c := range cases {
		if got := IsNewer(c.latest, c.current); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func newReleaseServer(tag string, hits *int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		fmt.Fprintf(w, `{"tag_name":%q}`, tag)
	}))
}

func TestCheckUpdateNewerAndCache(t *testing.T) {
	var hits int32
	srv := newReleaseServer("v1.3.0", &hits)
	defer srv.Close()
	t.Setenv(updateCheckEnv, "")

	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	cfg := UpdateCheckConfig{
		APIURL:    srv.URL,
		CachePath: filepath.Join(t.TempDir(), updateCacheFile),
		Now:       func() time.Time { return now },
	}

	res := CheckUpdate("v1.2.0", cfg)
	if !res.Newer || res.Latest != "v1.3.0" {
		t.Fatalf("esperaba aviso de v1.3.0, got %+v", res)
	}
	if !strings.Contains(res.Notice(), "pi-gomemory-setup update") {
		t.Fatalf("el aviso debe sugerir el comando update: %q", res.Notice())
	}

	now = now.Add(23 * time.Hour)
	if res := CheckUpdate("v1.2.0", cfg); !res.Newer {
		t.Fatalf("la caché debe conservar el aviso, got %+v", res)
	}
	if hits != 1 {
		t.Fatalf("dentro de 24 h no debe consultar la red: hits=%d", hits)
	}

	now = now.Add(2 * time.Hour)
	CheckUpdate("v1.2.0", cfg)
	if hits != 2 {
		t.Fatalf("pasadas 24 h debe volver a consultar: hits=%d", hits)
	}
}

func TestCheckUpdateUpToDate(t *testing.T) {
	var hits int32
	srv := newReleaseServer("v1.2.0", &hits)
	defer srv.Close()
	t.Setenv(updateCheckEnv, "")
	res := CheckUpdate("v1.2.0", UpdateCheckConfig{APIURL: srv.URL, CachePath: filepath.Join(t.TempDir(), updateCacheFile)})
	if res.Newer || res.Notice() != "" {
		t.Fatalf("no debe avisar si está al día: %+v", res)
	}
}

func TestCheckUpdateNetworkFailureIsSilent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()
	t.Setenv(updateCheckEnv, "")
	start := time.Now()
	res := CheckUpdate("v1.2.0", UpdateCheckConfig{
		APIURL:    srv.URL,
		CachePath: filepath.Join(t.TempDir(), updateCacheFile),
		Timeout:   50 * time.Millisecond,
	})
	if res.Newer || res.Err == nil {
		t.Fatalf("un fallo de red no debe avisar y debe registrar el error: %+v", res)
	}
	if time.Since(start) > 250*time.Millisecond {
		t.Fatalf("debe respetar el timeout")
	}
}

func TestCheckUpdateDisabled(t *testing.T) {
	var hits int32
	srv := newReleaseServer("v9.0.0", &hits)
	defer srv.Close()
	t.Setenv(updateCheckEnv, "1")
	res := CheckUpdate("v1.2.0", UpdateCheckConfig{APIURL: srv.URL, CachePath: filepath.Join(t.TempDir(), updateCacheFile)})
	if res.Newer || hits != 0 {
		t.Fatalf("con %s=1 no debe consultar ni avisar: %+v hits=%d", updateCheckEnv, res, hits)
	}
}

func makeTarball(t *testing.T, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, data := range map[string][]byte{"README.md": []byte("x"), "pi-gomemory-setup": content} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(data))}); err != nil {
			t.Fatal(err)
		}
		tw.Write(data)
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func newDownloadServer(t *testing.T, tag string, tarball []byte, sum string) *httptest.Server {
	name := fmt.Sprintf("pi-gomemory-setup_%s_linux_amd64.tar.gz", tag)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + tag + "/" + name:
			w.Write(tarball)
		case "/" + tag + "/SHA256SUMS":
			fmt.Fprintf(w, "%s  %s\n", sum, name)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestUpdateReplacesBinary(t *testing.T) {
	tarball := makeTarball(t, []byte("NEW-BINARY"))
	h := sha256.Sum256(tarball)
	srv := newDownloadServer(t, "v1.3.0", tarball, hex.EncodeToString(h[:]))
	defer srv.Close()

	target := filepath.Join(t.TempDir(), "pi-gomemory-setup")
	os.WriteFile(target, []byte("OLD"), 0755)

	err := DownloadAndReplace(UpdateConfig{BaseURL: srv.URL, Tag: "v1.3.0", GOOS: "linux", GOARCH: "amd64", Target: target})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "NEW-BINARY" {
		t.Fatalf("binario no reemplazado: %q", got)
	}
	if info, _ := os.Stat(target); info.Mode().Perm()&0100 == 0 {
		t.Fatalf("el binario debe ser ejecutable")
	}
}

func TestUpdateRejectsBadChecksum(t *testing.T) {
	tarball := makeTarball(t, []byte("EVIL"))
	srv := newDownloadServer(t, "v1.3.0", tarball, strings.Repeat("0", 64))
	defer srv.Close()

	target := filepath.Join(t.TempDir(), "pi-gomemory-setup")
	os.WriteFile(target, []byte("OLD"), 0755)

	err := DownloadAndReplace(UpdateConfig{BaseURL: srv.URL, Tag: "v1.3.0", GOOS: "linux", GOARCH: "amd64", Target: target})
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("debe rechazar el checksum incorrecto, err=%v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "OLD" {
		t.Fatalf("no debe tocar el binario si el checksum falla: %q", got)
	}
}

// HCD-004 sustituye check-update por deps-status/deps-refresh, que incluyen pi-gomemory.
func TestHarnessCallsCheckUpdate(t *testing.T) {
	s := GenerateHarnessScript()
	if !strings.Contains(s, "pi-gomemory-setup deps-status") {
		t.Fatalf("el launcher debe llamar a deps-status")
	}
	if !strings.Contains(s, "PI_GOMEMORY_NO_UPDATE_CHECK") {
		t.Fatalf("el launcher debe respetar PI_GOMEMORY_NO_UPDATE_CHECK")
	}
	if !strings.Contains(s, "deps-refresh >/dev/null 2>&1 &) || true") {
		t.Fatalf("deps-refresh debe ir en segundo plano y no impedir el arranque")
	}
	if strings.Index(s, "deps-status") > strings.Index(s, "exec pi ") {
		t.Fatalf("deps-status debe ejecutarse antes de lanzar pi")
	}
}

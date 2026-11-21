package main

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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// HCD-062 T10 · C-10: install.sh lanza `install` con la entrada en /dev/null
// (con `curl | bash` su entrada es el resto del script), conserva su código
// de salida y muestra la nota de PATH también al fallar, solo con los
// directorios donde quedó alguna herramienta.
func TestHCD062ShellInstallScript(t *testing.T) {
	home := t.TempDir()
	setup := `#!/bin/sh
if read line; then echo "stdin:$line" > "$HOME/stdin.txt"; else echo "stdin:eof" > "$HOME/stdin.txt"; fi
mkdir -p "$HOME/.pi/agent/bin" && printf '#!/bin/sh\n' > "$HOME/.pi/agent/bin/pi" && chmod +x "$HOME/.pi/agent/bin/pi"
exit 7
`
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "pi-gomemory-setup", Mode: 0o755, Size: int64(len(setup)), Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte(setup))
	_ = tw.Close()
	_ = gz.Close()
	name := fmt.Sprintf("pi-gomemory-setup_vtest_%s_%s.tar.gz", runtime.GOOS, map[string]string{"amd64": "amd64", "arm64": "arm64"}[runtime.GOARCH])
	sum := sha256.Sum256(buf.Bytes())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + name:
			_, _ = w.Write(buf.Bytes())
		case "/SHA256SUMS":
			fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), name)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	script, err := filepath.Abs("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = t.TempDir() // fuera del repo: descarga en vez de compilar
	cmd.Env = []string{"HOME=" + home, "PATH=/usr/bin:/bin:/usr/sbin:/sbin", "PI_GOMEMORY_VERSION=vtest", "PI_GOMEMORY_BASE_URL=" + srv.URL}
	cmd.Stdin = strings.NewReader("RESTO-DEL-SCRIPT\n")
	out, err := cmd.CombinedOutput()
	ee, ok := err.(*exec.ExitError)
	if !ok || ee.ExitCode() != 7 {
		t.Fatalf("install.sh debe terminar con el código de install (7): %v\n%s", err, out)
	}
	got, _ := os.ReadFile(filepath.Join(home, "stdin.txt"))
	if strings.TrimSpace(string(got)) != "stdin:eof" {
		t.Fatalf("install recibió la entrada del script: %q", got)
	}
	text := string(out)
	for _, want := range []string{home + "/.local/bin", home + "/.pi/agent/bin"} {
		if !strings.Contains(text, want) {
			t.Fatalf("la nota de PATH debe nombrar %s:\n%s", want, text)
		}
	}
	if strings.Contains(text, home+"/.gomemory/bin") {
		t.Fatalf("la nota no debe nombrar directorios sin herramientas:\n%s", text)
	}
}

// C-10: update relanza install sin pasarle su entrada estándar.
func TestHCD062ShellUpdateChildNoStdin(t *testing.T) {
	cmd := installChildCmd("/bin/true", nil)
	if cmd.Stdin != nil {
		t.Fatalf("el install hijo de update no debe heredar stdin: %v", cmd.Stdin)
	}
	if strings.Join(cmd.Args, " ") != "/bin/true install" {
		t.Fatalf("args = %v", cmd.Args)
	}
}

// HCD-064 T7 (H-07 / C-007): scripts/clean-install-check.sh no fija una versión
// obsoleta literal (v1.20.0), sino que lee la versión fijada desde install.sh.
func TestHCD064CleanInstallCheckDynamicVersion(t *testing.T) {
	raw, err := os.ReadFile("scripts/clean-install-check.sh")
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	if strings.Contains(src, "v1.20.0") {
		t.Fatalf("scripts/clean-install-check.sh no debe fijar literalmente v1.20.0")
	}
	if !strings.Contains(src, "PINNED_VER") {
		t.Fatalf("scripts/clean-install-check.sh debe derivar PINNED_VER desde install.sh")
	}
	out, err := exec.Command("bash", "-c", `sed -n 's/^VERSION="${PI_GOMEMORY_VERSION:-\([^}]*\)}"/\1/p' install.sh`).Output()
	if err != nil {
		t.Fatal(err)
	}
	ver := strings.TrimSpace(string(out))
	if !strings.HasPrefix(ver, "v") {
		t.Fatalf("no se pudo extraer VERSION de install.sh: %q", ver)
	}
}

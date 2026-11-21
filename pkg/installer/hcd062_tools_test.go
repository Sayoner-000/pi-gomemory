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
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// HCD-062: instalar pi, mem, codegraph y codebase-memory-mcp desde sus
// canales oficiales cuando faltan.

// toolSandbox prepara un HOME y un PATH con solo los directorios del sistema,
// y devuelve las opciones de instalación.
func toolSandbox(t *testing.T) *Options {
	t.Helper()
	opts := sandbox(t)
	t.Setenv("PATH", "/usr/bin:/bin")
	t.Setenv(skipDepsEnv, "")
	return opts
}

// withToolHooks sustituye los ganchos del paso durante la prueba.
func withToolHooks(t *testing.T, h toolHooks) {
	t.Helper()
	old := hooks
	hooks = h
	t.Cleanup(func() { hooks = old })
}

func withTools(t *testing.T, tools []Tool) {
	t.Helper()
	old := installTools
	installTools = tools
	t.Cleanup(func() { installTools = old })
}

// fakeTool escribe un ejecutable que responde a --version.
func fakeTool(t *testing.T, dir, name string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, name), "#!/bin/sh\necho "+name+" 1.0.0\n")
	if err := os.Chmod(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
}

// scriptServer sirve scripts por HTTPS y registra cada petición.
func scriptServer(t *testing.T, routes map[string]string) (*httptest.Server, *[]string) {
	t.Helper()
	var hits []string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		body, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if strings.HasPrefix(body, "REDIRECT ") {
			http.Redirect(w, r, strings.TrimPrefix(body, "REDIRECT "), http.StatusFound)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func testHooks(srv *httptest.Server) toolHooks {
	h := defaultToolHooks()
	h.Client = srv.Client()
	h.TTY = func() bool { return false }
	h.ScriptTimeout = 20 * time.Second
	h.DownloadTimeout = 5 * time.Second
	h.SystemDirs = nil
	return h
}

// T1 · C-05, N-1: el PATH se amplía con los directorios conocidos, en orden,
// sin duplicar, y se crea el directorio de binarios.
func TestHCD062DetectExtendsPathAndFindsKnownDirs(t *testing.T) {
	opts := toolSandbox(t)
	opts.LocalBinDir = filepath.Join(opts.HomeDir, "custom-bin")
	t.Setenv("PATH", filepath.Join(opts.HomeDir, ".local", "bin")+":/usr/bin:/bin")
	extendToolPath(opts)
	if _, err := os.Stat(opts.LocalBinDir); err != nil {
		t.Fatalf("el directorio de binarios debe existir: %v", err)
	}
	want := []string{
		opts.LocalBinDir,
		filepath.Join(opts.PiAgentDir, "bin"),
		filepath.Join(opts.HomeDir, ".gomemory", "bin"),
		filepath.Join(opts.HomeDir, ".local", "bin"),
		"/usr/bin", "/bin",
	}
	got := strings.Split(os.Getenv("PATH"), ":")
	// /usr/local/bin se añade solo si no estaba: va detrás de los anteriores
	// y antes del PATH original.
	got = removeString(got, "/usr/local/bin")
	if strings.Join(got, ":") != strings.Join(want, ":") {
		t.Fatalf("PATH = %v\nquiero %v", got, want)
	}
	extendToolPath(opts)
	if n := strings.Count(os.Getenv("PATH"), opts.LocalBinDir); n != 1 {
		t.Fatalf("PATH duplicado (%d veces): %s", n, os.Getenv("PATH"))
	}
	fakeTool(t, filepath.Join(opts.PiAgentDir, "bin"), "pi")
	if !toolPresent("pi") {
		t.Fatal("pi en ~/.pi/agent/bin debe contar como presente")
	}
	if toolPresent("codegraph") {
		t.Fatal("codegraph no está en ningún directorio")
	}
}

func removeString(in []string, s string) []string {
	out := in[:0:0]
	for _, v := range in {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}

// T2 · C-14, C-09: el instalador recibe /dev/null, escribe en una tubería,
// corre en el temporal privado y, sin terminal, en su propio grupo.
func TestHCD062RunnerStdinPipeCwdAndGroup(t *testing.T) {
	opts := toolSandbox(t)
	extendToolPath(opts)
	dir := t.TempDir()
	script := filepath.Join(dir, "probe.sh")
	writeFile(t, script, `if read line; then echo "stdin:$line"; else echo stdin:eof; fi
[ -p /dev/stdout ] && echo stdout:pipe
echo "cwd:$(pwd -P)"
echo "pgid:$(ps -o pgid= -p $$ | tr -d ' ')"
`)
	out, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	h := defaultToolHooks()
	h.TTY = func() bool { return false }
	withToolHooks(t, h)
	cwd, _ := filepath.EvalSymlinks(dir)
	if err := runInstaller("sh", []string{script}, nil, dir, out, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out.Name())
	text := string(data)
	for _, want := range []string{"stdin:eof", "stdout:pipe", "cwd:" + cwd} {
		if !strings.Contains(text, want) {
			t.Fatalf("falta %q en la salida:\n%s", want, text)
		}
	}
	own := fmt.Sprint(syscall.Getpgrp())
	if strings.Contains(text, "pgid:"+own+"\n") {
		t.Fatalf("sin terminal el hijo debe ir en su propio grupo:\n%s", text)
	}
	// Con terminal comparte el grupo de primer plano (C-14).
	h.TTY = func() bool { return true }
	withToolHooks(t, h)
	out2, _ := os.CreateTemp(t.TempDir(), "out")
	if err := runInstaller("sh", []string{script}, nil, dir, out2, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(out2.Name())
	if !strings.Contains(string(data), "pgid:"+own+"\n") {
		t.Fatalf("con terminal el hijo debe compartir el grupo %s:\n%s", own, data)
	}
}

// T2 · C-09: un instalador que se cuelga y deja un nieto vivo no bloquea el
// paso: vence el plazo, se mata el grupo y se informa el fallo.
func TestHCD062RunnerTimeoutKillsGroup(t *testing.T) {
	opts := toolSandbox(t)
	extendToolPath(opts)
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "nieto.pid")
	script := filepath.Join(dir, "hang.sh")
	writeFile(t, script, "sleep 60 &\necho $! > "+pidFile+"\nwait\n")
	h := defaultToolHooks()
	h.TTY = func() bool { return false }
	withToolHooks(t, h)
	start := time.Now()
	err := runInstaller("sh", []string{script}, nil, dir, &bytes.Buffer{}, 500*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "plazo") {
		t.Fatalf("debe fallar por plazo: %v", err)
	}
	if d := time.Since(start); d > 8*time.Second {
		t.Fatalf("el paso se quedó colgado %s", d)
	}
	data, _ := os.ReadFile(pidFile)
	var pid int
	fmt.Sscan(strings.TrimSpace(string(data)), &pid)
	time.Sleep(200 * time.Millisecond)
	if pid > 0 && syscall.Kill(pid, 0) == nil {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		t.Fatalf("el nieto %d sigue vivo", pid)
	}
}

// T3 · C-12: solo HTTPS, 200, tamaño acotado.
func TestHCD062DownloadHardening(t *testing.T) {
	big := strings.Repeat("x", 64)
	srv, _ := scriptServer(t, map[string]string{
		"/ok":    "echo ok",
		"/big":   big,
		"/plain": "REDIRECT http://example.invalid/x.sh",
	})
	client := hardenedClient(srv.Client(), 5*time.Second)
	if body, err := fetchHTTPS(client, srv.URL+"/ok", 1024); err != nil || string(body) != "echo ok" {
		t.Fatalf("ok = %q %v", body, err)
	}
	if _, err := fetchHTTPS(client, srv.URL+"/big", 16); err == nil {
		t.Fatal("debe rechazar una respuesta mayor que el límite")
	}
	if _, err := fetchHTTPS(client, srv.URL+"/no", 1024); err == nil {
		t.Fatal("debe exigir HTTP 200")
	}
	if _, err := fetchHTTPS(client, srv.URL+"/plain", 1024); err == nil {
		t.Fatal("debe rechazar una redirección a HTTP")
	}
	if _, err := fetchHTTPS(client, "http://example.invalid/x.sh", 1024); err == nil {
		t.Fatal("debe rechazar una URL que no es HTTPS")
	}
}

// installScriptFor devuelve un script falso que crea el binario name en dir y
// deja constancia del entorno que recibió.
func installScriptFor(name, dir, envVar string) string {
	return fmt.Sprintf(`#!/bin/sh
set -e
echo "%[1]s:%[3]s=${%[3]s:-}" >> %[2]q/../install.log
mkdir -p %[2]q
printf '#!/bin/sh\necho %[1]s 9.9.9\n' > %[2]q/%[1]s
chmod +x %[2]q/%[1]s
`, name, dir, envVar)
}

// T4 · C-01, C-02, C-03, C-05: instala lo que falta por su script, con su
// entorno, y no toca lo presente.
func TestHCD062EnsureInstallsMissingByScript(t *testing.T) {
	opts := toolSandbox(t)
	bin := opts.LocalBinDir
	srv, hits := scriptServer(t, map[string]string{
		"/pi.sh":  installScriptFor("pi", bin, "HOME"),
		"/mem.sh": installScriptFor("mem", bin, "GOMEMORY_BIN_DIR"),
	})
	withToolHooks(t, testHooks(srv))
	withTools(t, []Tool{
		{Name: "pi", Required: true, Script: srv.URL + "/pi.sh", Interp: "sh", Manual: "manual-pi"},
		{Name: "mem", Required: true, Script: srv.URL + "/mem.sh", Interp: "bash", Env: memEnv, Manual: "manual-mem"},
		{Name: "codegraph", Script: srv.URL + "/codegraph.sh", Interp: "sh", Manual: "manual-codegraph"},
	})
	fakeTool(t, filepath.Join(opts.HomeDir, ".gomemory", "bin"), "codegraph")
	var out bytes.Buffer
	if err := EnsureTools(opts, &out); err != nil {
		t.Fatalf("EnsureTools: %v\n%s", err, out.String())
	}
	if strings.Join(*hits, ",") != "/pi.sh,/mem.sh" {
		t.Fatalf("descargas = %v; codegraph ya estaba", *hits)
	}
	for _, n := range []string{"pi", "mem"} {
		if _, err := os.Stat(filepath.Join(bin, n)); err != nil {
			t.Fatalf("%s no quedó instalado: %v", n, err)
		}
	}
	log, _ := os.ReadFile(filepath.Join(bin, "..", "install.log"))
	if !strings.Contains(string(log), "mem:GOMEMORY_BIN_DIR="+bin) {
		t.Fatalf("mem debe recibir GOMEMORY_BIN_DIR=%s:\n%s", bin, log)
	}
	entries, _ := os.ReadDir(bin)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".pi-gomemory-deps-") {
			t.Fatalf("quedó el temporal %s", e.Name())
		}
	}
}

// Hallazgo de la validación real: dentro de una sesión de Pi el entorno trae
// PI_MANAGED_INSTALL_ROOT de la instalación en uso; el instalador de pi no
// debe heredarlo y recibe PI_CODING_AGENT_DIR de esta instalación.
func TestHCD062EnsurePiEnvIsolated(t *testing.T) {
	opts := toolSandbox(t)
	t.Setenv("PI_MANAGED_INSTALL_ROOT", "/otra/instalacion")
	t.Setenv("PI_CODING_AGENT_DIR", "/otra/agent")
	bin := opts.LocalBinDir
	script := installScriptFor("pi", bin, "PI_CODING_AGENT_DIR") + `echo "root=${PI_MANAGED_INSTALL_ROOT:-}" >> "` + bin + `/../install.log"` + "\n"
	srv, _ := scriptServer(t, map[string]string{"/pi.sh": script})
	withToolHooks(t, testHooks(srv))
	pi := installTools0(t, "pi")
	pi.Script = srv.URL + "/pi.sh"
	withTools(t, []Tool{pi})
	if err := EnsureTools(opts, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	log, _ := os.ReadFile(filepath.Join(bin, "..", "install.log"))
	if !strings.Contains(string(log), "pi:PI_CODING_AGENT_DIR="+opts.PiAgentDir) || !strings.Contains(string(log), "root=\n") {
		t.Fatalf("entorno del instalador de pi:\n%s", log)
	}
}

// installTools0 devuelve la definición de producción de una herramienta.
func installTools0(t *testing.T, name string) Tool {
	t.Helper()
	for _, tl := range productionTools() {
		if tl.Name == name {
			return tl
		}
	}
	t.Fatalf("sin herramienta %s", name)
	return Tool{}
}

// T4 · C-06: una obligatoria que falla aborta con el comando manual; una
// opcional que falla avisa y sigue.
func TestHCD062EnsureRequiredAbortsOptionalWarns(t *testing.T) {
	opts := toolSandbox(t)
	srv, _ := scriptServer(t, map[string]string{"/fail.sh": "exit 3\n"})
	withToolHooks(t, testHooks(srv))
	withTools(t, []Tool{
		{Name: "codegraph", Script: srv.URL + "/fail.sh", Interp: "sh", Manual: "manual-codegraph"},
		{Name: "pi", Required: true, Script: srv.URL + "/fail.sh", Interp: "sh", Manual: "manual-pi"},
	})
	var out bytes.Buffer
	err := EnsureTools(opts, &out)
	if err == nil || !strings.Contains(err.Error(), "pi") || !strings.Contains(err.Error(), "manual-pi") {
		t.Fatalf("pi obligatorio debe abortar con su comando manual: %v", err)
	}
	if !strings.Contains(out.String(), "manual-codegraph") {
		t.Fatalf("codegraph opcional debe avisar con su comando manual:\n%s", out.String())
	}
}

// T4 · C-07, N-6: PI_GOMEMORY_SKIP_DEPS omite el paso; lite solo comprueba
// las obligatorias.
func TestHCD062EnsureSkipAndLite(t *testing.T) {
	opts := toolSandbox(t)
	srv, hits := scriptServer(t, map[string]string{})
	withToolHooks(t, testHooks(srv))
	withTools(t, []Tool{
		{Name: "pi", Required: true, Script: srv.URL + "/pi.sh", Interp: "sh", Manual: "m"},
		{Name: "codegraph", Script: srv.URL + "/cg.sh", Interp: "sh", Manual: "m"},
	})
	for _, v := range []string{"1", "yes", " TRUE "} {
		t.Setenv(skipDepsEnv, v)
		if err := EnsureTools(opts, &bytes.Buffer{}); err != nil || len(*hits) != 0 {
			t.Fatalf("%s=%q debe omitir el paso: %v %v", skipDepsEnv, v, err, *hits)
		}
	}
	t.Setenv(skipDepsEnv, "false")
	fakeTool(t, opts.LocalBinDir, "pi")
	if err := SetLiteMode(opts, true); err != nil {
		t.Fatal(err)
	}
	if err := EnsureTools(opts, &bytes.Buffer{}); err != nil || len(*hits) != 0 {
		t.Fatalf("en lite no se instala codegraph: %v %v", err, *hits)
	}
}

// cbmRelease arma un release falso de codebase-memory-mcp.
func cbmRelease(t *testing.T, tag string, binScript string, extra map[string]string) (asset string, tgz []byte, sums string) {
	t.Helper()
	asset, err := cbmAsset(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	files := map[string]string{"codebase-memory-mcp": binScript, "LICENSE": "x", "install.sh": "x", "THIRD_PARTY_NOTICES.md": "x"}
	for k, v := range extra {
		files[k] = v
	}
	for name, body := range files {
		mode := int64(0o644)
		if name == "codebase-memory-mcp" {
			mode = 0o755
		}
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	_ = gz.Close()
	sum := sha256.Sum256(buf.Bytes())
	return asset, buf.Bytes(), hex.EncodeToString(sum[:]) + "  " + asset + "\n"
}

func cbmServer(t *testing.T, tag, asset string, tgz []byte, sums string) (*httptest.Server, *[]string) {
	t.Helper()
	var hits []string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/releases/tag/"+tag, http.StatusFound)
		case "/releases/download/" + tag + "/checksums.txt":
			_, _ = w.Write([]byte(sums))
		case "/releases/download/" + tag + "/" + asset:
			_, _ = w.Write(tgz)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

const cbmFakeBin = `#!/bin/sh
case "$1" in
--version) echo "codebase-memory-mcp 0.12.1" ;;
install)
  for a in "$@"; do case "$a" in --dir=*) d="${a#--dir=}" ;; esac; done
  echo "$@" > "$d/cbm-install-args"
  cp "$0" "$d/codebase-memory-mcp" ;;
esac
`

// T5 · C-04: release verificado, solo el binario, install --skip-config.
func TestHCD062CodebaseMemoryFromRelease(t *testing.T) {
	opts := toolSandbox(t)
	extendToolPath(opts)
	asset, tgz, sums := cbmRelease(t, "v0.12.1", cbmFakeBin, nil)
	srv, _ := cbmServer(t, "v0.12.1", asset, tgz, sums)
	h := testHooks(srv)
	h.CBMReleases = srv.URL + "/releases"
	withToolHooks(t, h)
	if err := installCodebaseMemory(opts, &bytes.Buffer{}); err != nil {
		t.Fatalf("installCodebaseMemory: %v", err)
	}
	args, err := os.ReadFile(filepath.Join(opts.LocalBinDir, "cbm-install-args"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "install -y --force --dir=" + opts.LocalBinDir + " --skip-config"; strings.TrimSpace(string(args)) != want {
		t.Fatalf("args = %q, quiero %q", args, want)
	}
	if _, err := os.Stat(filepath.Join(opts.LocalBinDir, "codebase-memory-mcp")); err != nil {
		t.Fatal(err)
	}
}

// T5 · C-04, N-2: SHA-256 incorrecto, etiqueta inválida o tarball con rutas
// peligrosas no ejecutan nada.
func TestHCD062CodebaseMemoryRejects(t *testing.T) {
	marker := "#!/bin/sh\ntouch \"$HOME/EJECUTADO\"\n"
	cases := map[string]func(t *testing.T) (*httptest.Server, string){
		"sha": func(t *testing.T) (*httptest.Server, string) {
			asset, tgz, _ := cbmRelease(t, "v0.12.1", marker, nil)
			srv, _ := cbmServer(t, "v0.12.1", asset, tgz, strings.Repeat("0", 64)+"  "+asset+"\n")
			return srv, "checksum"
		},
		"sin-entrada": func(t *testing.T) (*httptest.Server, string) {
			asset, tgz, _ := cbmRelease(t, "v0.12.1", marker, nil)
			srv, _ := cbmServer(t, "v0.12.1", asset, tgz, "abc  otro.tar.gz\n")
			return srv, "no encontrado"
		},
		"etiqueta": func(t *testing.T) (*httptest.Server, string) {
			asset, tgz, sums := cbmRelease(t, "v0.12.1", marker, nil)
			srv, _ := cbmServer(t, "../../x", asset, tgz, sums)
			return srv, "etiqueta"
		},
		"ruta": func(t *testing.T) (*httptest.Server, string) {
			asset, tgz, sums := cbmRelease(t, "v0.12.1", "#!/bin/sh\necho no\n", map[string]string{"../codebase-memory-mcp": marker})
			_ = asset
			srv, _ := cbmServer(t, "v0.12.1", asset, tgz, sums)
			return srv, "tarball"
		},
	}
	for name, mk := range cases {
		t.Run(name, func(t *testing.T) {
			opts := toolSandbox(t)
			extendToolPath(opts)
			srv, want := mk(t)
			h := testHooks(srv)
			h.CBMReleases = srv.URL + "/releases"
			withToolHooks(t, h)
			err := installCodebaseMemory(opts, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, quiero %q", err, want)
			}
			if _, err := os.Stat(filepath.Join(opts.HomeDir, "EJECUTADO")); err == nil {
				t.Fatal("se ejecutó contenido no verificado")
			}
		})
	}
}

// Hallazgo de la validación real: el binario de v0.12.1 descomprimido ocupa
// ~335 MB; el límite propio lo admite y sigue rechazando lo desmedido.
func TestHCD062CodebaseBinaryLimit(t *testing.T) {
	if cbmMaxBinaryBytes < 335_000_000 {
		t.Fatalf("el límite %d no admite el binario real (~335 MB)", cbmMaxBinaryBytes)
	}
	_, tgz, _ := cbmRelease(t, "v0.12.1", strings.Repeat("x", 2048), nil)
	if _, err := extractExact(tgz, cbmBinary, 1024); err == nil || !strings.Contains(err.Error(), "fuera de rango") {
		t.Fatalf("debe rechazar un binario mayor que el límite: %v", err)
	}
	if bin, err := extractExact(tgz, cbmBinary, 4096); err != nil || len(bin) != 2048 {
		t.Fatalf("extractExact = %d %v", len(bin), err)
	}
}

func TestHCD062CodebaseAssetNames(t *testing.T) {
	for _, c := range []struct{ os, arch, want string }{
		{"darwin", "arm64", "codebase-memory-mcp-darwin-arm64.tar.gz"},
		{"darwin", "amd64", "codebase-memory-mcp-darwin-amd64.tar.gz"},
		{"linux", "amd64", "codebase-memory-mcp-linux-amd64-portable.tar.gz"},
		{"linux", "arm64", "codebase-memory-mcp-linux-arm64-portable.tar.gz"},
	} {
		if got, err := cbmAsset(c.os, c.arch); err != nil || got != c.want {
			t.Errorf("cbmAsset(%s,%s) = %q %v", c.os, c.arch, got, err)
		}
	}
	if _, err := cbmAsset("windows", "amd64"); err == nil {
		t.Error("windows no está soportado")
	}
}

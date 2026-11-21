package installer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// maxScriptBytes acota un script de instalación descargado (HCD-062, C-12).
const maxScriptBytes = 1 << 20

// depsTempPrefix nombra los temporales del paso; install y uninstall
// retiran los que quedaron de una ejecución interrumpida.
const depsTempPrefix = ".pi-gomemory-deps-"

// hardenedClient copia base con plazo y solo admite redirecciones HTTPS.
func hardenedClient(base *http.Client, timeout time.Duration) *http.Client {
	c := &http.Client{}
	if base != nil {
		*c = *base
	}
	c.Timeout = timeout
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("redirección no HTTPS rechazada: %s", req.URL.Redacted())
		}
		if len(via) >= 10 {
			return errors.New("demasiadas redirecciones")
		}
		return nil
	}
	return c
}

// fetchHTTPS descarga rawURL solo por HTTPS, exige HTTP 200 y como mucho
// limit bytes.
func fetchHTTPS(client *http.Client, rawURL string, limit int64) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" {
		return nil, fmt.Errorf("solo se descarga por HTTPS: %s", u.Redacted())
	}
	return httpGet(client, rawURL, limit)
}

// depsTempDir crea un directorio privado junto al directorio de binarios
// (no en /tmp, que puede ser noexec) y devuelve cómo borrarlo.
func depsTempDir(opts *Options) (string, func(), error) {
	binDir, err := filepath.Abs(opts.LocalBinDir)
	if err != nil {
		return "", nil, fmt.Errorf("resolver directorio de binarios: %w", err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", nil, err
	}
	dir, err := os.MkdirTemp(binDir, depsTempPrefix+"*")
	if err != nil {
		return "", nil, fmt.Errorf("crear temporal: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, err
	}
	// Marca de propiedad entre procesos (HCD-064 / H-03): evita que otra
	// instancia concurrente de install/uninstall borre este temporal vivo.
	_ = writeFileAtomic(filepath.Join(dir, ".pid"), []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600)
	untrack := trackTemp(dir)
	return dir, func() { untrack(); _ = os.RemoveAll(dir) }, nil
}

// pipeOnly oculta el *os.File de un Writer para que exec cree una tubería:
// el instalador nunca ve una terminal en su salida (HCD-062, C-14).
type pipeOnly struct{ w io.Writer }

func (p pipeOnly) Write(b []byte) (int, error) { return p.w.Write(b) }

// resolveInterp busca el intérprete en el PATH y, si no está, en /bin y
// /usr/bin.
func resolveInterp(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	for _, d := range []string{"/bin/", "/usr/bin/"} {
		if _, err := os.Stat(d + name); err == nil {
			return d + name
		}
	}
	return name
}

// runInstaller ejecuta un instalador con la entrada en /dev/null, la salida
// por tubería, dir como directorio de trabajo y un plazo. Con terminal
// comparte el grupo de procesos de primer plano para poder preguntar por
// /dev/tty y recibir Ctrl-C; sin terminal va en su propio grupo, que se mata
// entero al vencer el plazo. WaitDelay impide que un nieto retenga la salida.
func runInstaller(name string, args, env []string, dir string, out io.Writer, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, resolveInterp(name), args...)
	cmd.Dir = dir
	cmd.Env = childEnv(env)
	cmd.Stdin = nil
	cmd.Stdout = pipeOnly{out}
	cmd.Stderr = pipeOnly{out}
	cmd.WaitDelay = 5 * time.Second
	pgidMode := hooks.TTY == nil || !hooks.TTY()
	if pgidMode {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error {
			if cmd.Process == nil {
				return nil
			}
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	} else {
		cmd.Cancel = func() error {
			if cmd.Process == nil {
				return nil
			}
			killProcessTree(cmd.Process.Pid)
			return nil
		}
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	untrack := trackLiveChild(cmd.Process.Pid, pgidMode, cancel)
	err := cmd.Wait()
	untrack()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("superó el plazo de %s", timeout)
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("terminó con código %d", exitErr.ExitCode())
		}
		return err
	}
	return nil
}

// inheritedDrop son variables del entorno que un instalador no debe heredar:
// PI_MANAGED_INSTALL_ROOT la exporta el lanzador de pi y haría que el
// instalador de pi trabaje sobre la instalación en uso (HCD-062).
var inheritedDrop = []string{"PI_MANAGED_INSTALL_ROOT"}

// childEnv es el entorno del proceso sin las variables de inheritedDrop ni
// las que extra redefine, más extra.
func childEnv(extra []string) []string {
	drop := map[string]bool{}
	for _, k := range inheritedDrop {
		drop[k] = true
	}
	for _, kv := range extra {
		k, _, _ := strings.Cut(kv, "=")
		drop[k] = true
	}
	var out []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if !drop[k] {
			out = append(out, kv)
		}
	}
	return append(out, extra...)
}

// runQuiet ejecuta un comando auxiliar y devuelve su salida combinada.
func runQuiet(timeout time.Duration, name string, args ...string) (string, error) {
	var buf strings.Builder
	err := runInstaller(name, args, nil, "", &buf, timeout)
	return buf.String(), err
}

// killProcessTree mata al proceso pid y a sus descendientes (HCD-064 / H-05).
// Se usa cuando el hijo comparte grupo con la terminal (TTY activo) y no se
// puede aplicar Setpgid.
func killProcessTree(root int) {
	if root <= 0 {
		return
	}
	out, err := exec.Command("ps", "-eo", "pid=,ppid=").Output()
	if err == nil {
		children := map[int][]int{}
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) != 2 {
				continue
			}
			p, err1 := strconv.Atoi(fields[0])
			pp, err2 := strconv.Atoi(fields[1])
			if err1 == nil && err2 == nil {
				children[pp] = append(children[pp], p)
			}
		}
		var order []int
		queue := []int{root}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, ch := range children[cur] {
				order = append(order, ch)
				queue = append(queue, ch)
			}
		}
		// Matar de hojas hacia arriba y por último el proceso raíz.
		for i := len(order) - 1; i >= 0; i-- {
			_ = syscall.Kill(order[i], syscall.SIGKILL)
		}
	}
	_ = syscall.Kill(root, syscall.SIGKILL)
}

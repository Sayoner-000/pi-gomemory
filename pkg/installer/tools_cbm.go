package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const cbmBinary = "codebase-memory-mcp"

// cbmMaxBinaryBytes acota el binario descomprimido: el de v0.12.1 ocupa
// unos 335 MB (el .tar.gz, unos 50 MB), por encima del límite de
// pi-gomemory-setup. Variable para las pruebas.
var cbmMaxBinaryBytes int64 = 1 << 30

// cbmTagPattern valida la etiqueta antes de usarla en URL y rutas (N-2).
var cbmTagPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// cbmAsset devuelve el nombre del .tar.gz del release para el SO y la
// arquitectura; Linux usa la variante -portable.
func cbmAsset(goos, goarch string) (string, error) {
	if goos != "darwin" && goos != "linux" {
		return "", fmt.Errorf("%s no publica binario para %s", cbmBinary, goos)
	}
	if goarch != "amd64" && goarch != "arm64" {
		return "", fmt.Errorf("%s no publica binario para %s", cbmBinary, goarch)
	}
	suffix := ""
	if goos == "linux" {
		suffix = "-portable"
	}
	return fmt.Sprintf("%s-%s-%s%s.tar.gz", cbmBinary, goos, goarch, suffix), nil
}

// installCodebaseMemory instala el último release de codebase-memory-mcp en
// el directorio de binarios: etiqueta resuelta una vez por la redirección de
// github.com, SHA-256 contra checksums.txt, extracción exacta del binario,
// firma ad hoc en macOS y el subcomando oficial install --skip-config, que
// no toca la configuración de otros agentes (HCD-062, «Orden seguro»).
func installCodebaseMemory(opts *Options, out io.Writer) error {
	goos, goarch := hooks.GOOS, hooks.GOARCH
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	asset, err := cbmAsset(goos, goarch)
	if err != nil {
		return err
	}
	base := strings.TrimRight(hooks.CBMReleases, "/")
	noRedirect := hardenedClient(hooks.Client, hooks.DownloadTimeout)
	noRedirect.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	tag, err := fetchLatestTagWith(noRedirect, base+"/latest")
	if err != nil {
		return fmt.Errorf("resolver la última versión: %w", err)
	}
	if !cbmTagPattern.MatchString(tag) {
		return fmt.Errorf("etiqueta de release inválida: %q", tag)
	}
	client := hardenedClient(hooks.Client, hooks.DownloadTimeout)
	dl := base + "/download/" + tag + "/"
	sums, err := fetchHTTPS(client, dl+"checksums.txt", 1<<20)
	if err != nil {
		return fmt.Errorf("descargar checksums.txt: %w", err)
	}
	want, err := lookupChecksum(sums, asset)
	if err != nil {
		return err
	}
	tgz, err := fetchHTTPS(client, dl+asset, maxDownloadBytes)
	if err != nil {
		return fmt.Errorf("descargar %s: %w", asset, err)
	}
	got := sha256.Sum256(tgz)
	if !strings.EqualFold(hex.EncodeToString(got[:]), want) {
		return fmt.Errorf("checksum incorrecto para %s: no se ejecuta nada", asset)
	}
	bin, err := extractExact(tgz, cbmBinary, cbmMaxBinaryBytes)
	if err != nil {
		return err
	}
	tmp, cleanup, err := depsTempDir(opts)
	if err != nil {
		return err
	}
	defer cleanup()
	path := filepath.Join(tmp, cbmBinary)
	if err := writeFileAtomic(path, bin, 0o755); err != nil {
		return err
	}
	if goos == "darwin" {
		// Como su instalador oficial: los fallos de xattr o codesign se ignoran.
		_, _ = runQuiet(30*time.Second, "xattr", "-d", "com.apple.quarantine", path)
		_, _ = runQuiet(30*time.Second, "codesign", "--sign", "-", "--force", path)
	}
	if v, err := runQuiet(time.Minute, path, "--version"); err != nil {
		return fmt.Errorf("el binario descargado no se ejecuta: %v %s", err, strings.TrimSpace(v))
	}
	fmt.Fprintf(out, "%s      %s %s verificado (SHA-256)%s\n", ColorMuted, cbmBinary, tag, ColorReset)
	return runInstaller(path, []string{"install", "-y", "--force", "--dir=" + opts.LocalBinDir, "--skip-config"}, nil, tmp, out, hooks.ScriptTimeout)
}

// extractExact extrae del .tar.gz la entrada regular con nombre exacto name.
// Rechaza el archivo entero si contiene rutas absolutas, con «..» o enlaces.
func extractExact(tgz []byte, name string, limit int64) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return nil, fmt.Errorf("abrir tarball: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var bin []byte
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("leer tarball: %w", err)
		}
		clean := strings.TrimPrefix(h.Name, "./")
		if filepath.IsAbs(clean) || clean != filepath.Clean(clean) || strings.Contains(clean, "..") {
			return nil, fmt.Errorf("tarball con ruta no permitida: %q", h.Name)
		}
		if h.Typeflag == tar.TypeSymlink || h.Typeflag == tar.TypeLink {
			return nil, fmt.Errorf("tarball con enlace no permitido: %q", h.Name)
		}
		if clean != name || h.Typeflag != tar.TypeReg {
			continue
		}
		if h.Size <= 0 || h.Size > limit {
			return nil, fmt.Errorf("%s: tamaño %d fuera de rango", name, h.Size)
		}
		if bin, err = io.ReadAll(io.LimitReader(tr, h.Size)); err != nil || int64(len(bin)) != h.Size {
			return nil, fmt.Errorf("leer %s del tarball: %v", name, err)
		}
	}
	if bin == nil {
		return nil, fmt.Errorf("%s no está en el tarball", name)
	}
	return bin, nil
}

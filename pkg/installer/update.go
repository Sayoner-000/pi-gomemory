package installer

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	updateRepo         = "Sayoner-000/pi-gomemory"
	updateCheckEnv     = "PI_GOMEMORY_NO_UPDATE_CHECK"
	updateCacheFile    = "pi-gomemory-update.json"
	updateCheckTTL     = 24 * time.Hour
	updateCheckTimeout = 2 * time.Second
	maxDownloadBytes   = 200 << 20
)

// maxBinaryBytes es el tamaño máximo del ejecutable dentro del tarball. Es una
// variable para que las pruebas lo reduzcan (HCD-034, H-03).
var maxBinaryBytes int64 = 200 << 20

// DefaultReleasesAPI apunta a la redirección de la última release en github.com,
// que no consume el cupo de 60 consultas por hora de api.github.com.
// DefaultReleasesBaseURL es la base de descarga de los tarballs.
var (
	DefaultReleasesAPI     = "https://github.com/" + updateRepo + "/releases/latest"
	DefaultReleasesBaseURL = "https://github.com/" + updateRepo + "/releases/download"
)

// parseSemver acepta "vX.Y.Z" o "X.Y.Z" estables; rechaza prereleases y texto libre.
func parseSemver(tag string) ([3]int, bool) {
	var v [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(tag), "v"), ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

// IsNewer indica si latest es una versión estable estrictamente mayor que current.
// Si alguna de las dos no es semver estable (p. ej. "dev"), devuelve false.
func IsNewer(latest, current string) bool {
	l, ok1 := parseSemver(latest)
	c, ok2 := parseSemver(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// UpdateCheckConfig parametriza CheckUpdate; los campos vacíos usan valores por defecto.
type UpdateCheckConfig struct {
	APIURL    string
	CachePath string
	Timeout   time.Duration
	Now       func() time.Time
}

// UpdateCheckResult es el resultado de comprobar si hay una release nueva.
type UpdateCheckResult struct {
	Current string
	Latest  string
	Newer   bool
	Err     error
}

// Notice devuelve el aviso para el usuario, o "" si no hay nada nuevo.
func (r UpdateCheckResult) Notice() string {
	if !r.Newer {
		return ""
	}
	return fmt.Sprintf("%s↑ pi-gomemory %s disponible (tienes %s). Actualiza con: pi-gomemory-setup update%s",
		ColorBrillo, r.Latest, r.Current, ColorReset)
}

type updateCache struct {
	LatestTag string    `json:"latest_tag"`
	CheckedAt time.Time `json:"checked_at"`
}

// UpdateCheckDisabled indica si la persona desactivó la comprobación.
func UpdateCheckDisabled() bool {
	v := strings.TrimSpace(os.Getenv(updateCheckEnv))
	return v != "" && v != "0" && !strings.EqualFold(v, "false")
}

// DefaultUpdateCachePath devuelve la ruta de la caché dentro de ~/.pi/agent.
func DefaultUpdateCachePath(piAgentDir string) string {
	return filepath.Join(piAgentDir, updateCacheFile)
}

// CheckUpdate consulta la última release como mucho una vez cada 24 h.
// Nunca falla de forma ruidosa: los errores quedan en Result.Err.
func CheckUpdate(current string, cfg UpdateCheckConfig) UpdateCheckResult {
	res := UpdateCheckResult{Current: current}
	if UpdateCheckDisabled() {
		return res
	}
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultReleasesAPI
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = updateCheckTimeout
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	var cache updateCache
	if cfg.CachePath != "" {
		if data, err := os.ReadFile(cfg.CachePath); err == nil {
			_ = json.Unmarshal(data, &cache)
		}
	}

	now := cfg.Now()
	if cache.LatestTag == "" || now.Sub(cache.CheckedAt) >= updateCheckTTL || now.Before(cache.CheckedAt) {
		tag, err := fetchLatestTag(cfg.APIURL, cfg.Timeout)
		if err != nil {
			res.Err = err
			return res
		}
		cache = updateCache{LatestTag: tag, CheckedAt: now}
		if cfg.CachePath != "" {
			if data, err := json.Marshal(cache); err == nil {
				_ = os.MkdirAll(filepath.Dir(cfg.CachePath), 0755)
				_ = writeFileAtomic(cfg.CachePath, data, 0644)
			}
		}
	}

	res.Latest = cache.LatestTag
	res.Newer = IsNewer(res.Latest, current)
	return res
}

// fetchLatestTag obtiene la etiqueta de la última release. Con una URL
// github.com/<repo>/releases/latest lee la cabecera Location de la redirección
// (/releases/tag/<tag>) sin seguirla; con una respuesta JSON lee tag_name.
func fetchLatestTag(apiURL string, timeout time.Duration) (string, error) {
	client := &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pi-gomemory-setup")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		loc := resp.Header.Get("Location")
		const marker = "/releases/tag/"
		i := strings.LastIndex(loc, marker)
		if i < 0 || loc[i+len(marker):] == "" {
			return "", fmt.Errorf("releases: redirección sin etiqueta (%q)", loc)
		}
		return strings.TrimRight(loc[i+len(marker):], "/"), nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("releases: HTTP %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return "", fmt.Errorf("releases API: %w", err)
	}
	if body.TagName == "" {
		return "", errors.New("releases API: tag_name vacío")
	}
	return body.TagName, nil
}

// UpdateConfig parametriza DownloadAndReplace; los campos vacíos usan valores por defecto.
type UpdateConfig struct {
	BaseURL string
	Tag     string
	GOOS    string
	GOARCH  string
	Target  string
	Timeout time.Duration
}

// DownloadAndReplace descarga el tarball de la release, lo verifica contra SHA256SUMS
// y reemplaza el binario Target de forma atómica. Si la verificación falla, Target no cambia.
func DownloadAndReplace(cfg UpdateConfig) error {
	if cfg.Tag == "" || cfg.Target == "" {
		return errors.New("update: falta la versión o el binario destino")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultReleasesBaseURL
	}
	if cfg.GOOS == "" {
		cfg.GOOS = runtime.GOOS
	}
	if cfg.GOARCH == "" {
		cfg.GOARCH = runtime.GOARCH
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 2 * time.Minute
	}
	client := &http.Client{Timeout: cfg.Timeout}
	base := strings.TrimRight(cfg.BaseURL, "/") + "/" + cfg.Tag + "/"
	name := fmt.Sprintf("pi-gomemory-setup_%s_%s_%s.tar.gz", cfg.Tag, cfg.GOOS, cfg.GOARCH)

	sums, err := httpGet(client, base+"SHA256SUMS", 1<<20)
	if err != nil {
		return fmt.Errorf("descargar SHA256SUMS: %w", err)
	}
	want, err := lookupChecksum(sums, name)
	if err != nil {
		return err
	}
	tarball, err := httpGet(client, base+name, maxDownloadBytes)
	if err != nil {
		return fmt.Errorf("descargar %s: %w", name, err)
	}
	got := sha256.Sum256(tarball)
	if !strings.EqualFold(hex.EncodeToString(got[:]), want) {
		return fmt.Errorf("checksum incorrecto para %s: se rechaza la actualización", name)
	}

	bin, err := extractBinary(tarball, "pi-gomemory-setup")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(cfg.Target), ".pi-gomemory-setup-*")
	if err != nil {
		return fmt.Errorf("crear temporal: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return fmt.Errorf("escribir temporal: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0755); err != nil {
		return err
	}
	if err := os.Rename(tmpName, cfg.Target); err != nil {
		return fmt.Errorf("reemplazar %s: %w", cfg.Target, err)
	}
	return nil
}

func httpGet(client *http.Client, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "pi-gomemory-setup")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("respuesta demasiado grande")
	}
	return data, nil
}

func lookupChecksum(sums []byte, name string) (string, error) {
	sc := bufio.NewScanner(bytes.NewReader(sums))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum de %s no encontrado en SHA256SUMS", name)
}

func extractBinary(tarball []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(tarball))
	if err != nil {
		return nil, fmt.Errorf("abrir tarball: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%s no está en el tarball", name)
		}
		if err != nil {
			return nil, fmt.Errorf("leer tarball: %w", err)
		}
		if filepath.Base(h.Name) == name && h.Typeflag == tar.TypeReg {
			if h.Size <= 0 || h.Size > maxBinaryBytes {
				return nil, fmt.Errorf("%s: tamaño %d fuera de rango (1..%d bytes)", name, h.Size, maxBinaryBytes)
			}
			bin, err := io.ReadAll(io.LimitReader(tr, h.Size))
			if err != nil {
				return nil, fmt.Errorf("leer %s: %w", name, err)
			}
			if int64(len(bin)) != h.Size {
				return nil, fmt.Errorf("%s incompleto: %d de %d bytes", name, len(bin), h.Size)
			}
			return bin, nil
		}
	}
}

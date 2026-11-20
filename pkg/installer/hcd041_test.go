package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// HCD-041: cierre de H-02 a H-05 de la ACR/ADM 2026-11-17. Los cuatro primeros
// tests son los reproductores de docs/audits/2026-11-17-acr-adm-repro_test.go.txt;
// el 05 era un testigo y se invierte: más entradas que el límite se rechazan.

func TestACRADM02PreserveSymlinkAfterPlan(t *testing.T) {
	t.Parallel()

	d := t.TempDir()
	defaults := []byte(`{"showStartupHeader":false}`)
	cfg := filepath.Join(d, "pi-cc-extensions.json")
	target := filepath.Join(d, "personal.json")
	if err := os.WriteFile(cfg, defaults, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, defaults, 0600); err != nil {
		t.Fatal(err)
	}
	p := &UninstallPlan{}
	p.planCCExtensions(fstest.MapFS{"config/pi-cc-extensions.json": &fstest.MapFile{Data: defaults}}, d)
	if len(p.Actions) != 1 || p.Actions[0].apply == nil {
		t.Fatalf("no removal planned: %+v", p.Actions)
	}
	if err := os.Remove(cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, cfg); err != nil {
		t.Fatal(err)
	}
	err := p.Actions[0].apply()
	if _, statErr := os.Lstat(cfg); statErr != nil {
		t.Fatalf("new personal symlink removed: apply=%v lstat=%v", err, statErr)
	}
}

func TestACRADM03FailedCommandIsNotVersion(t *testing.T) {
	t.Parallel()

	v, err := versionFromCmd("sh", "-c", "printf '9.0.0\\n'; exit 7")(context.Background())
	if err == nil {
		t.Fatalf("failed command accepted: version=%q error=nil", v)
	}
}

func TestACRADM04MissingCacheIsNotHealthy(t *testing.T) {
	t.Parallel()

	var out, errs bytes.Buffer
	rc := RunDepsUpdate(filepath.Join(t.TempDir(), "absent.json"), nil, &out, &errs)
	if rc == 0 && bytes.Contains(out.Bytes(), []byte("están al día")) {
		t.Fatalf("unknown state reported healthy: rc=%d stdout=%q stderr=%q", rc, out.String(), errs.String())
	}
}

// hcd041EmptyEntries crea un tar.gz con count archivos vacíos.
func hcd041EmptyEntries(t *testing.T, count int) []byte {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	for i := 0; i < count; i++ {
		name := filepath.ToSlash(filepath.Join("package", time.Unix(int64(i), 0).UTC().Format("150405")))
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: 0, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// Tarea 4 (H-04, Q2): deps-update nunca dice «al día» sin conocer el estado.
// Sin caché o ilegible no actualiza nada y sale con 1; con componentes en error
// o sin datos actualiza lo conocido, los lista y sale con 1.
func TestHCD041DepsUpdateUnknown(t *testing.T) {
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")
	bin := t.TempDir()
	log := filepath.Join(bin, "pi.log")
	writeExec(t, filepath.Join(bin, "pi"), `echo "$*" >> "`+log+`"`)
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "npm", "node_modules", "pi-antigravity", "package.json"), `{"version":"0.0.1"}`)
	writeFile(t, filepath.Join(dir, "npm", "node_modules", "pi-cc-extensions", "package.json"), `{"version":"`+CCExtensionsVersion+`"}`)
	probes := append(hcd039PinnedProbes(dir), DepProbe{
		Name:    "codebase-memory-mcp",
		Current: func(context.Context) (string, error) { return "", os.ErrNotExist },
		Manual:  "https://example.invalid",
	})
	run := func(t *testing.T, cache string) (int, string, string, string) {
		t.Helper()
		_ = os.Remove(log)
		var out, errOut bytes.Buffer
		rc := RunDepsUpdate(cache, probes, &out, &errOut)
		calls, _ := os.ReadFile(log)
		return rc, out.String(), errOut.String(), string(calls)
	}
	cacheWith := func(t *testing.T, body string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "deps.json")
		writeFile(t, p, body)
		return p
	}
	healthy := `{"name":"pi-antigravity","current":"` + AntigravityVersion + `","latest":"` + AntigravityVersion + `"},` +
		`{"name":"pi-cc-extensions","current":"` + CCExtensionsVersion + `","latest":"` + CCExtensionsVersion + `"}`

	for name, cache := range map[string]string{
		"ausente":        filepath.Join(t.TempDir(), "no-existe.json"),
		"JSON inválido":  cacheWith(t, "{no es json"),
		"sin checked_at": cacheWith(t, `{"deps":[`+healthy+`]}`),
	} {
		rc, out, errOut, calls := run(t, cache)
		if rc != 1 || strings.Contains(out, "al día") || !strings.Contains(errOut, "deps-refresh") || calls != "" {
			t.Errorf("%s: sin estado conocido no actualiza, pide deps-refresh y sale con 1: rc=%d out=%q err=%q calls=%q", name, rc, out, errOut, calls)
		}
	}

	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "1")
	if rc, _, errOut, _ := run(t, filepath.Join(t.TempDir(), "x.json")); rc != 1 || !strings.Contains(errOut, "PI_GOMEMORY_NO_UPDATE_CHECK") {
		t.Errorf("con la comprobación desactivada explica por qué no hay datos: rc=%d err=%q", rc, errOut)
	}
	t.Setenv("PI_GOMEMORY_NO_UPDATE_CHECK", "")

	// Caché válida: pi-antigravity atrasado, codebase-memory-mcp en error.
	mixed := cacheWith(t, `{"checked_at":"2026-11-17T09:30:00Z","deps":[`+
		`{"name":"pi-antigravity","current":"0.0.1","latest":"`+AntigravityVersion+`"},`+
		`{"name":"pi-cc-extensions","current":"`+CCExtensionsVersion+`","latest":"`+CCExtensionsVersion+`"},`+
		`{"name":"codebase-memory-mcp","error":"codebase-memory-mcp no está en PATH"}]}`)
	rc, out, errOut, calls := run(t, mixed)
	if rc != 1 || strings.Contains(out, "al día") || !strings.Contains(calls, "install "+AntigravityPackageSource) ||
		!strings.Contains(errOut, "codebase-memory-mcp: estado desconocido (codebase-memory-mcp no está en PATH)") {
		t.Errorf("actualiza lo conocido y lista lo desconocido con código 1: rc=%d out=%q err=%q calls=%q", rc, out, errOut, calls)
	}

	// Caché parcial: un probe sin entrada cuenta como desconocido.
	partial := cacheWith(t, `{"checked_at":"2026-11-17T09:30:00Z","deps":[`+healthy+`]}`)
	if rc, out, errOut, _ := run(t, partial); rc != 1 || strings.Contains(out, "al día") || !strings.Contains(errOut, "codebase-memory-mcp: estado desconocido (sin datos en la caché)") {
		t.Errorf("caché parcial: rc=%d out=%q err=%q", rc, out, errOut)
	}

	// Todo conocido y al día: también la versión instalada debe coincidir;
	// DepsStatusFor relee el paquete local, no confía en Current de la caché.
	writeFile(t, filepath.Join(dir, "npm", "node_modules", "pi-antigravity", "package.json"), `{"version":"`+AntigravityVersion+`"}`)
	full := cacheWith(t, `{"checked_at":"2026-11-17T09:30:00Z","deps":[`+healthy+`,{"name":"codebase-memory-mcp","current":"1.0.0","latest":"1.0.0"}]}`)
	if rc, out, _, calls := run(t, full); rc != 0 || !strings.Contains(out, "están al día") || calls != "" {
		t.Errorf("con todo conocido y al día dice «al día» y sale con 0: rc=%d out=%q calls=%q", rc, out, calls)
	}
}

// Tarea 3 (H-03, Q1): un comando que termina con error o se cancela nunca da
// versión, aunque imprima una en stdout o stderr; el éxito se conserva.
func TestHCD041FailedCommand(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	for name, script := range map[string]string{
		"semver en stdout": "printf '9.0.0\\n'; exit 7",
		"semver en stderr": "printf 'error: requiere 9.0.0\\n' >&2; exit 1",
	} {
		if v, err := versionFromCmd("sh", "-c", script)(ctx); err == nil {
			t.Errorf("%s: un fallo no debe dar versión, dio %q", name, v)
		}
	}
	re := regexp.MustCompile(`latest\s+v?(\d+\.\d+\.\d+)`)
	if v, err := latestFromCmd(re, "sh", "-c", "echo 'latest v9.9.9'; exit 2")(ctx); err == nil {
		t.Errorf("latestFromCmd tampoco acepta un fallo con su patrón: %q", v)
	}

	cancelled, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	if v, err := versionFromCmd("sh", "-c", "echo 1.2.3; sleep 5")(cancelled); err == nil {
		t.Errorf("una cancelación con salida previa no debe dar versión: %q", v)
	}

	if v, err := versionFromCmd("sh", "-c", "echo 'tool 1.2.3'")(ctx); err != nil || v != "1.2.3" {
		t.Errorf("un comando correcto sigue dando su versión: %q %v", v, err)
	}
	if v, err := latestFromCmd(regexp.MustCompile(`Disponible:\s*v?(\d+\.\d+\.\d+)`), "sh", "-c", "echo 'Actual: v1.0.0 → Disponible: v1.1.0'")(ctx); err != nil || v != "1.1.0" {
		t.Errorf("latestFromCmd correcto sigue dando la versión: %q %v", v, err)
	}
}

// Tarea 2 (H-02): si pi-cc-extensions.json deja de ser un archivo normal entre
// el plan y la acción, se conserva como «cambió después del plan»; un archivo
// normal con los valores por defecto se sigue retirando.
func TestHCD041CCExtensionsAfterPlan(t *testing.T) {
	t.Parallel()

	defaults := []byte(`{"showStartupHeader":false}`)
	assets := fstest.MapFS{"config/pi-cc-extensions.json": &fstest.MapFile{Data: defaults}}
	plan := func(t *testing.T) (string, string, *UninstallAction) {
		t.Helper()
		d := t.TempDir()
		cfg := filepath.Join(d, "pi-cc-extensions.json")
		writeFile(t, cfg, string(defaults))
		p := &UninstallPlan{}
		p.planCCExtensions(assets, d)
		if len(p.Actions) != 1 || p.Actions[0].Kind != ActionRemove {
			t.Fatalf("debía planificarse la retirada: %+v", p.Actions)
		}
		return d, cfg, &p.Actions[0]
	}

	for name, mutate := range map[string]func(t *testing.T, d, cfg string){
		"enlace a un archivo propio": func(t *testing.T, d, cfg string) {
			target := filepath.Join(d, "personal.json")
			writeFile(t, target, string(defaults))
			_ = os.Remove(cfg)
			if err := os.Symlink(target, cfg); err != nil {
				t.Fatal(err)
			}
		},
		"enlace roto": func(t *testing.T, d, cfg string) {
			_ = os.Remove(cfg)
			if err := os.Symlink(filepath.Join(d, "no-existe.json"), cfg); err != nil {
				t.Fatal(err)
			}
		},
		"directorio": func(t *testing.T, _, cfg string) {
			_ = os.Remove(cfg)
			if err := os.Mkdir(cfg, 0o755); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, cfg, a := plan(t)
			mutate(t, d, cfg)
			if err := a.apply(); err != errChangedAfterPlan {
				t.Errorf("debe devolver errChangedAfterPlan, devolvió %v", err)
			}
			if _, err := os.Lstat(cfg); err != nil {
				t.Errorf("la entrada sobrevenida debe conservarse: %v", err)
			}
			if target := filepath.Join(d, "personal.json"); name == "enlace a un archivo propio" {
				if b, err := os.ReadFile(target); err != nil || string(b) != string(defaults) {
					t.Errorf("el destino del enlace debe quedar intacto: %q %v", b, err)
				}
			}
		})
	}

	_, cfg, a := plan(t)
	if err := a.apply(); err != nil {
		t.Fatalf("un archivo normal por defecto se retira: %v", err)
	}
	if _, err := os.Lstat(cfg); err == nil {
		t.Error("el archivo normal por defecto debía retirarse")
	}
}

// Tarea 5 (H-05, Q3): extractCandidate limita las entradas y el flujo tar
// descomprimido (cabeceras incluidas), respeta la cancelación entre entradas y
// fetchCandidate limpia el directorio temporal tras cualquiera de esos errores.
func TestHCD041ExtractLimits(t *testing.T) {
	if candidateEntryLimit != 10_000 || candidateTarStreamLimit != 256<<20 || candidateExtractLimit != 200<<20 {
		t.Fatalf("límites de producción: entradas=%d flujo=%d contenido=%d", candidateEntryLimit, candidateTarStreamLimit, candidateExtractLimit)
	}
	oldE, oldS, oldRoot, oldContent := candidateEntryLimit, candidateTarStreamLimit, candidateTempRoot, candidateExtractLimit
	t.Cleanup(func() {
		candidateEntryLimit, candidateTarStreamLimit, candidateTempRoot, candidateExtractLimit = oldE, oldS, oldRoot, oldContent
	})
	ctx := context.Background()

	candidateEntryLimit = 50
	dirs := func(n int) []byte {
		var es []hcd039Entry
		for i := 0; i < n; i++ {
			es = append(es, hcd039Entry{name: filepath.ToSlash(filepath.Join("package", "d", time.Unix(int64(i), 0).UTC().Format("150405"))) + "/", typeflag: tar.TypeDir})
		}
		return hcd039Tarball(t, es...)
	}
	if err := extractCandidate(ctx, dirs(50), t.TempDir()); err != nil {
		t.Errorf("50 directorios con límite 50 deben extraerse: %v", err)
	}
	if err := extractCandidate(ctx, dirs(51), t.TempDir()); err == nil {
		t.Error("los directorios vacíos también cuentan como entradas")
	}
	candidateEntryLimit = oldE

	// Cabeceras extendidas voluminosas: no son contenido, pero cuentan en el flujo.
	candidateTarStreamLimit = 64 << 10
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	h := &tar.Header{Name: "package/a", Mode: 0600, Typeflag: tar.TypeReg, PAXRecords: map[string]string{"comment": string(bytes.Repeat([]byte("x"), 128<<10))}}
	if err := tw.WriteHeader(h); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gz.Close()
	if err := extractCandidate(ctx, b.Bytes(), t.TempDir()); err == nil {
		t.Error("una cabecera extendida mayor que el flujo permitido debe rechazarse")
	}
	candidateTarStreamLimit = oldS

	// Fronteras exactas del flujo (incluye cabeceras, relleno y fin del tar).
	boundary := hcd041EmptyEntries(t, 1)
	reader, err := gzip.NewReader(bytes.NewReader(boundary))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	for _, delta := range []int64{-1, 0, 1} {
		candidateTarStreamLimit = int64(len(raw)) + delta
		err := extractCandidate(ctx, boundary, t.TempDir())
		if (err != nil) != (delta < 0) {
			t.Errorf("flujo de %d bytes, límite %d: %v", len(raw), candidateTarStreamLimit, err)
		}
	}
	candidateTarStreamLimit = oldS
	candidateExtractLimit = 16
	for _, size := range []int{15, 16, 17} {
		data := hcd039Tarball(t, hcd039Entry{name: "package/a", body: strings.Repeat("x", size)})
		err := extractCandidate(ctx, data, t.TempDir())
		if (err != nil) != (size > 16) {
			t.Errorf("contenido %d con límite 16: %v", size, err)
		}
	}
	candidateExtractLimit = oldContent

	// Cancela al observar el primer archivo: no depende del reloj ni de carreras.
	dest := t.TempDir()
	mid, stop := context.WithCancel(ctx)
	defer stop()
	observed := &hcd041CancelAfterFile{Context: mid, path: filepath.Join(dest, "package", "first"), cancel: stop}
	data := hcd039Tarball(t, hcd039Entry{name: "package/first"}, hcd039Entry{name: "package/second"})
	if err := extractCandidate(observed, data, dest); !errors.Is(err, context.Canceled) {
		t.Errorf("debe cancelarse entre entradas: %v", err)
	}
	if _, err := os.Stat(observed.path); err != nil {
		t.Fatalf("debió extraerse la primera entrada: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "package", "second")); !os.IsNotExist(err) {
		t.Errorf("no debe extraerse la segunda entrada: %v", err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := extractCandidate(cancelled, hcd041EmptyEntries(t, 3), t.TempDir()); err == nil {
		t.Error("una extracción con el contexto cancelado debe detenerse")
	}

	// fetchCandidate borra el directorio temporal cuando la extracción falla por un límite.
	root := t.TempDir()
	candidateTempRoot = root
	candidateEntryLimit = 2
	data = hcd041EmptyEntries(t, 3)
	if _, _, err := fetchCandidate(ctx, hcd039TarballServer(t, data), hcd039Integrity(data)); err == nil {
		t.Error("fetchCandidate debe propagar el error del límite de entradas")
	}
	if left, _ := os.ReadDir(root); len(left) != 0 {
		t.Errorf("quedó el directorio temporal tras el error: %v", left)
	}
}

// Contexto de prueba que cancela cuando ya se extrajo la primera entrada.
type hcd041CancelAfterFile struct {
	context.Context
	path   string
	cancel context.CancelFunc
}

func (c *hcd041CancelAfterFile) Err() error {
	if _, err := os.Stat(c.path); err == nil {
		c.cancel()
	}
	return c.Context.Err()
}

// Invertido respecto de la auditoría: con el límite de entradas, un tarball que
// lo supera con entradas vacías se rechaza; uno justo en el límite se acepta.
func TestACRADM05EmptyEntriesAreBounded(t *testing.T) {
	old := candidateEntryLimit
	candidateEntryLimit = 100
	t.Cleanup(func() { candidateEntryLimit = old })
	if err := extractCandidate(context.Background(), hcd041EmptyEntries(t, 100), t.TempDir()); err != nil {
		t.Fatalf("100 entradas con límite 100 deben extraerse: %v", err)
	}
	if err := extractCandidate(context.Background(), hcd041EmptyEntries(t, 101), t.TempDir()); err == nil {
		t.Fatal("101 entradas vacías con límite 100 deben rechazarse")
	}
}

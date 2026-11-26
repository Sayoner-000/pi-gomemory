package installer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// HCD-039: seguimiento de los paquetes de Pi fijados frente a npm.

// hcd039Registry sirve el documento abreviado de cada paquete y guarda la
// cabecera Accept de la última petición. RefreshDeps consulta los paquetes en
// paralelo, así que la escritura va protegida.
func hcd039Registry(t *testing.T, docs map[string]string) (*httptest.Server, *string) {
	t.Helper()
	accept := new(string)
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		*accept = r.Header.Get("Accept")
		mu.Unlock()
		doc, ok := docs[strings.TrimPrefix(r.URL.Path, "/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.npm.install-v1+json")
		_, _ = w.Write([]byte(doc))
	}))
	t.Cleanup(srv.Close)
	old := NPMRegistryURL
	NPMRegistryURL = srv.URL
	t.Cleanup(func() { NPMRegistryURL = old })
	return srv, accept
}

// hcd039Doc construye un documento abreviado con las versiones indicadas.
func hcd039Doc(versions ...string) string {
	parts := make([]string, 0, len(versions))
	for _, v := range versions {
		parts = append(parts, `"`+v+`":{"version":"`+v+`","dist":{"tarball":"t","integrity":"sha512-x"}}`)
	}
	return `{"name":"p","versions":{` + strings.Join(parts, ",") + `}}`
}

// Tarea 5 (C-08, C-09, C-10, C-11): build-release.sh ejecuta pins-check sin
// caché y con -v después de go test -race, y si falla se detiene antes de
// borrar dist/ explicando cómo seguir. Se ejecuta el script real con un go falso.
func TestHCD039BuildReleaseGate(t *testing.T) {
	t.Parallel()

	script := readAsset(t, "scripts/build-release.sh")
	run := func(t *testing.T, allowStale bool) (string, string, error) {
		dir, bin := t.TempDir(), t.TempDir()
		log := filepath.Join(bin, "go.log")
		writeFile(t, filepath.Join(dir, "scripts", "build-release.sh"), script)
		for _, f := range []string{"README.md", "LICENSE", "install.sh"} {
			writeFile(t, filepath.Join(dir, f), f)
		}
		writeFile(t, filepath.Join(dir, "dist", "anterior.tar.gz"), "release anterior")
		writeExec(t, filepath.Join(bin, "go"), `echo "$*" >> "`+log+`"
case "$*" in
  *TestPinsCheck*)
    if [ "$PI_GOMEMORY_ALLOW_STALE_PINS" = "1" ]; then echo "aceptado: pi-antigravity: fijado 0.12.0 · npm publica 0.14.0 sin validar"; exit 0; fi
    echo "pi-antigravity: fijado 0.12.0 · npm publica 0.14.0 sin validar"; exit 1 ;;
  build*)
    out=""; prev=""; for a in "$@"; do [ "$prev" = "-o" ] && out="$a"; prev="$a"; done
    mkdir -p "$(dirname "$out")"; echo bin > "$out" ;;
esac
exit 0`)
		cmd := exec.Command("bash", "scripts/build-release.sh", "v0.0.0-test")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"))
		if allowStale {
			cmd.Env = append(cmd.Env, "PI_GOMEMORY_ALLOW_STALE_PINS=1")
		} else {
			cmd.Env = append(cmd.Env, "PI_GOMEMORY_ALLOW_STALE_PINS=")
		}
		out, err := cmd.CombinedOutput()
		calls, _ := os.ReadFile(log)
		if _, statErr := os.Stat(filepath.Join(dir, "dist", "anterior.tar.gz")); statErr == nil {
			out = append(out, []byte("\n[dist intacto]")...)
		}
		return string(out), string(calls), err
	}

	out, calls, err := run(t, false)
	if err == nil {
		t.Fatalf("C-09: con un pin atrasado la release debe detenerse:\n%s", out)
	}
	if !strings.Contains(out, "[dist intacto]") || strings.Contains(calls, "build ") {
		t.Errorf("C-09: debe detenerse antes de borrar dist/ y de compilar:\n%s\n--- go:\n%s", out, calls)
	}
	if !strings.Contains(out, "npm publica 0.14.0") || !strings.Contains(out, "PI_GOMEMORY_ALLOW_STALE_PINS=1") {
		t.Errorf("C-09: debe mostrar el desfase y cómo seguir:\n%s", out)
	}
	pinsCall := "test -tags pins -count=1 -v ./pkg/installer -run ^TestPinsCheck$"
	race, pins := strings.Index(calls, "test -race ./..."), strings.Index(calls, pinsCall)
	if race < 0 || pins < race {
		t.Errorf("C-08: pins-check va con -count=1 -v después de go test -race:\n%s", calls)
	}
	if !strings.Contains(calls, "vet -tags pins ./pkg/installer") {
		t.Errorf("también debe revisar con go vet los archivos de la etiqueta pins:\n%s", calls)
	}

	out, calls, err = run(t, true)
	if err != nil || !strings.Contains(out, "aceptado: pi-antigravity") || !strings.Contains(calls, "build ") {
		t.Errorf("C-11: con PI_GOMEMORY_ALLOW_STALE_PINS=1 empaqueta y deja lo aceptado en la salida (err=%v):\n%s", err, out)
	}
}

// Tarea 4 (C-01, C-02, C-03, C-11): el informe de pins-check falla ante un pin
// atrasado o un error de npm, salvo que se acepte con allowStale, y en ese caso
// deja en la salida lo que se aceptó.
func TestHCD039PinsCheckReport(t *testing.T) {
	join := func(lines []string) string { return strings.Join(lines, "\n") }
	// Desde HCD-085 pi-cc-extensions es el único paquete fijado.
	hcd039Registry(t, map[string]string{"pi-cc-extensions": hcd039Doc(CCExtensionsVersion, "99.0.0")})
	checks := CheckPins(context.Background())
	lines, fail := FormatPinChecks(checks, false)
	out := join(lines)
	if !fail || !strings.Contains(out, "pi-cc-extensions: fijado "+CCExtensionsVersion+" · npm publica 99.0.0") {
		t.Errorf("C-01: el desfase debe fallar y nombrar paquete, pin y publicada:\n%s", out)
	}
	if strings.Contains(out, "pi-antigravity") {
		t.Errorf("HCD-085: pi-antigravity ya no se comprueba:\n%s", out)
	}
	if !strings.Contains(out, "PI_GOMEMORY_ALLOW_STALE_PINS=1") || !strings.Contains(out, "TestPinsValidate") || !strings.Contains(out, "RejectedVersions") {
		t.Errorf("C-09: ante un fallo debe decir cómo seguir:\n%s", out)
	}
	if _, fail := FormatPinChecks(checks, true); fail {
		t.Error("C-11: con allowStale el desfase no debe fallar")
	}
	accepted, _ := FormatPinChecks(checks, true)
	if !strings.Contains(join(accepted), "aceptado: pi-cc-extensions: fijado "+CCExtensionsVersion+" · npm publica 99.0.0") {
		t.Errorf("C-11: con allowStale la salida debe dejar el desfase aceptado:\n%s", join(accepted))
	}

	hcd039Registry(t, map[string]string{"pi-cc-extensions": hcd039Doc(CCExtensionsVersion)})
	lines, fail = FormatPinChecks(CheckPins(context.Background()), false)
	if fail || !strings.Contains(join(lines), "pi-cc-extensions: al día ("+CCExtensionsVersion+")") {
		t.Errorf("C-02: con todo al día no debe fallar y debe decirlo:\n%s", join(lines))
	}

	hcd039Registry(t, map[string]string{})
	errChecks := CheckPins(context.Background())
	lines, fail = FormatPinChecks(errChecks, false)
	if !fail || !strings.Contains(join(lines), "pi-cc-extensions: error consultando npm") || strings.Contains(join(lines), "pi-cc-extensions: al día") {
		t.Errorf("C-03: un error de npm falla y nunca se presenta como al día:\n%s", join(lines))
	}
	lines, fail = FormatPinChecks(errChecks, true)
	if fail || !strings.Contains(join(lines), "no comprobado: error de red") {
		t.Errorf("C-11: con allowStale un error se acepta como «no comprobado: error de red»:\n%s", join(lines))
	}
}

// Tarea 3 (C-07): sin la etiqueta pins, NPMRegistryURL apunta a una dirección
// inválida, así que una prueba sin servidor local falla en vez de salir a npm.
func TestHCD039NoNetworkByDefault(t *testing.T) {
	t.Parallel()

	if pinsLive {
		t.Skip("con la etiqueta pins se consulta el registro real")
	}
	if NPMRegistryURL == "https://registry.npmjs.org" || !strings.HasPrefix(NPMRegistryURL, "http://127.0.0.1:") {
		t.Fatalf("la suite normal no debe apuntar al registro real: NPMRegistryURL = %q", NPMRegistryURL)
	}
	if _, err := registryVersions(context.Background(), "pi-antigravity"); err == nil {
		t.Fatal("sin servidor local, la consulta debe fallar")
	}
}

// Tarea 2 (C-04, C-05): un rechazo nunca oculta una versión intermedia sin
// revisar, y si todo lo que hay por encima del pin está rechazado se informa
// con su motivo. La lista compilada en el binario solo nombra paquetes fijados.
func TestHCD039Rejected(t *testing.T) {
	t.Parallel()

	rejected := []RejectedVersion{{"pi-antigravity", "0.14.0", "rompe A1"}}
	got, err := publishedVersion("pi-antigravity", []string{"0.12.0", "0.13.0", "0.14.0"}, "0.12.0", rejected)
	if err != nil || got != "0.13.0" {
		t.Errorf("C-05: con 0.14.0 rechazada debe quedar 0.13.0, published = %q, %v", got, err)
	}

	all := []RejectedVersion{{"pi-antigravity", "0.13.0", "motivo A"}, {"pi-antigravity", "0.14.0", "motivo B"}}
	got, err = publishedVersion("pi-antigravity", []string{"0.12.0", "0.13.0", "0.14.0"}, "0.12.0", all)
	if err != nil || got != "0.12.0" {
		t.Errorf("C-04: todo rechazado deja published en el pin, = %q, %v", got, err)
	}
	above := rejectedAbove(all, "pi-antigravity", "0.12.0", []string{"0.12.0", "0.13.0", "0.14.0", "0.11.0"})
	if len(above) != 2 || above[0].Reason != "motivo A" || above[1].Version != "0.14.0" {
		t.Errorf("C-04: rejectedAbove debe devolver las dos rechazadas por encima del pin, en orden: %+v", above)
	}
	// Otro paquete con la misma versión no se ve afectado.
	if got, _ := publishedVersion("pi-cc-extensions", []string{"0.12.0", "0.14.0"}, "0.12.0", rejected); got != "0.14.0" {
		t.Errorf("un rechazo de otro paquete no debe contar, published = %q", got)
	}

	pinned := map[string]bool{}
	for _, p := range PinnedPackages() {
		pinned[p.Name] = true
	}
	// HCD-085: pi-antigravity ya no se fija.
	if pinned["pi-antigravity"] || !pinned["pi-cc-extensions"] || len(pinned) != 1 {
		t.Fatalf("PinnedPackages = %+v", PinnedPackages())
	}
	for _, r := range RejectedVersions {
		if _, ok := parseSemver(r.Version); !pinned[r.Package] || !ok || r.Reason == "" {
			t.Errorf("entrada rechazada mal formada: %+v", r)
		}
	}
}

// Tarea 1 (C-01, C-02, C-03, C-06): published es la estable más alta no
// rechazada; sin versions, sin estable o sin el pin es un error.
func TestHCD039Published(t *testing.T) {
	cases := []struct {
		name     string
		versions []string
		pin      string
		want     string
		wantErr  bool
	}{
		{"desfase", []string{"0.12.0", "0.13.0", "0.14.0"}, "0.12.0", "0.14.0", false},
		{"al día", []string{"0.11.0", "0.12.0"}, "0.12.0", "0.12.0", false},
		{"prerelease no cuenta", []string{"0.12.0", "0.15.0-beta.1"}, "0.12.0", "0.12.0", false},
		{"orden numérico, no lexicográfico", []string{"0.9.0", "0.12.0", "0.10.0"}, "0.9.0", "0.12.0", false},
		{"sin versions", nil, "0.12.0", "", true},
		{"sin ninguna estable", []string{"1.0.0-rc.1"}, "0.12.0", "", true},
		{"pin ausente", []string{"0.13.0", "0.14.0"}, "0.12.0", "", true},
	}
	for _, c := range cases {
		got, err := publishedVersion("p", c.versions, c.pin, nil)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("%s: publishedVersion = %q, %v; want %q, error=%v", c.name, got, err, c.want, c.wantErr)
		}
	}

	_, accept := hcd039Registry(t, map[string]string{
		"pi-antigravity": hcd039Doc("0.12.0", "0.14.0"),
		"vacio":          `{"name":"vacio"}`,
	})
	versions, err := registryVersions(context.Background(), "pi-antigravity")
	if err != nil || len(versions) != 2 {
		t.Fatalf("registryVersions = %v, %v", versions, err)
	}
	if *accept != "application/vnd.npm.install-v1+json" {
		t.Errorf("Accept = %q; se pide el documento abreviado", *accept)
	}
	if _, err := registryVersions(context.Background(), "vacio"); err == nil {
		t.Error("un documento sin versions debe ser un error")
	}
	if _, err := registryVersions(context.Background(), "no-existe"); err == nil {
		t.Error("un 404 debe ser un error")
	}

	// Un documento mayor que el límite se rechaza en lugar de truncarse.
	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"versions":{"0.1.0":{"pad":"`))
		_, _ = w.Write([]byte(strings.Repeat("x", maxRegistryDoc)))
		_, _ = w.Write([]byte(`"}}}`))
	}))
	defer big.Close()
	NPMRegistryURL = big.URL
	if _, err := registryVersions(context.Background(), "p"); err == nil || !strings.Contains(err.Error(), "16 MiB") {
		t.Errorf("un documento de más de 16 MiB debe rechazarse explicando el límite, err=%v", err)
	}
}

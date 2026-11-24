package installer

import (
	"bytes"
	"strings"
	"testing"
)

// HCD-076 VS-01: un catálogo único alimenta install, deps, doctor y uninstall.

func catalogNames(kinds ...ComponentKind) []string {
	var out []string
	for _, c := range Catalog() {
		for _, k := range kinds {
			if c.Kind == k {
				out = append(out, c.Name)
				break
			}
		}
	}
	return out
}

// T1 · C-08: orden, tipos y obligatoriedad del catálogo; herdr es un tool
// opcional instalado por su canal oficial.
func TestHCD076CatalogOrder(t *testing.T) {
	want := []struct {
		name     string
		kind     ComponentKind
		required bool
	}{
		{"pi-gomemory", KindSelf, true},
		{"pi", KindTool, true},
		{"mem", KindTool, true},
		{"codegraph", KindTool, false},
		{"herdr", KindTool, false},
		{"pi-antigravity", KindPackage, false},
		{"pi-cc-extensions", KindPackage, false},
	}
	got := Catalog()
	if len(got) != len(want) {
		t.Fatalf("catálogo con %d componentes, quiero %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		c := got[i]
		if c.Name != w.name || c.Kind != w.kind || c.Required != w.required {
			t.Fatalf("componente %d = %s/%s/%v, quiero %s/%s/%v", i, c.Name, c.Kind, c.Required, w.name, w.kind, w.required)
		}
	}
	h := got[4]
	if h.Script != HerdrInstallScriptURL || HerdrInstallScriptURL != "https://herdr.dev/install.sh" || h.Interp != "sh" {
		t.Fatalf("canal de herdr: %+v", h)
	}
	env := h.Env(&Options{LocalBinDir: "/x/bin"})
	if len(env) != 1 || env[0] != "HERDR_INSTALL_DIR=/x/bin" {
		t.Fatalf("entorno de herdr: %v", env)
	}
	if h.Manual != "curl -fsSL https://herdr.dev/install.sh | sh" || h.Uninstall == "" {
		t.Fatalf("manual/desinstalación de herdr: %+v", h)
	}
}

// T2 · C-08: productionTools y los textos con nombres salen de los tool.
func TestHCD076ToolsFromCatalog(t *testing.T) {
	var got []string
	for _, tl := range productionTools() {
		got = append(got, tl.Name)
	}
	want := catalogNames(KindTool)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("productionTools = %v, quiero %v", got, want)
	}
	text := ToolNamesText()
	if text != "pi, mem, codegraph y herdr" {
		t.Fatalf("ToolNamesText = %q", text)
	}
	t.Setenv(skipDepsEnv, "1")
	var out bytes.Buffer
	if err := EnsureTools(sandbox(t), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "no se comprueban "+text) {
		t.Fatalf("aviso de %s sin la lista del catálogo:\n%s", skipDepsEnv, out.String())
	}
	if !strings.Contains(InstallStepToolsText(), text) {
		t.Fatalf("paso [1/9] sin la lista del catálogo: %q", InstallStepToolsText())
	}
	var errOut bytes.Buffer
	RunDepsInstall(sandbox(t), nil, &bytes.Buffer{}, &errOut)
	if !strings.Contains(errOut.String(), "pi, mem, codegraph y herdr") {
		t.Fatalf("uso de deps-install sin los tool por script del catálogo:\n%s", errOut.String())
	}
}

// T3 · C-08: DefaultDepProbes contiene todo el catálogo en su orden, con la
// sonda de herdr.
func TestHCD076ProbesFromCatalog(t *testing.T) {
	var got []string
	var herdr DepProbe
	for _, p := range DefaultDepProbes("v1.0.0", t.TempDir()) {
		got = append(got, p.Name)
		if p.Name == "herdr" {
			herdr = p
		}
	}
	want := catalogNames(KindSelf, KindTool, KindPackage)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("DefaultDepProbes = %v, quiero %v", got, want)
	}
	if strings.Join(herdr.Update, " ") != "herdr update" || herdr.Current == nil || herdr.Latest == nil {
		t.Fatalf("sonda de herdr: %+v", herdr)
	}
}

// T4 · C-08, C-14: doctor informa los tool del catálogo; un opcional ausente
// es WARN y un obligatorio ausente es FAIL.
func TestHCD076DoctorOptionalWarn(t *testing.T) {
	opts := toolSandbox(t)
	withToolHooks(t, toolHooks{})
	status := map[string]string{}
	var order []string
	for _, item := range RunDoctor(opts) {
		if name, ok := strings.CutPrefix(item.Name, "Binario "); ok {
			status[name] = item.Status
			order = append(order, name)
		}
	}
	if strings.Join(order, ",") != strings.Join(catalogNames(KindTool), ",") {
		t.Fatalf("binarios de doctor = %v, quiero %v", order, catalogNames(KindTool))
	}
	for _, c := range Catalog() {
		if c.Kind != KindTool {
			continue
		}
		want := "WARN"
		if c.Required {
			want = "FAIL"
		}
		if status[c.Name] != want {
			t.Fatalf("doctor %s ausente = %s, quiero %s", c.Name, status[c.Name], want)
		}
	}
}

// T5 · C-08: printToolsKept nombra cada tool con su desinstalación.
func TestHCD076UninstallKeptFromCatalog(t *testing.T) {
	var out bytes.Buffer
	printToolsKept(&out)
	s := out.String()
	if !strings.Contains(s, "Se conservan "+ToolNamesText()) {
		t.Fatalf("encabezado sin la lista del catálogo:\n%s", s)
	}
	for _, c := range Catalog() {
		if c.Kind != KindTool {
			continue
		}
		if !strings.Contains(s, c.Name+":") || !strings.Contains(s, c.Uninstall) {
			t.Fatalf("falta %s o su desinstalación %q:\n%s", c.Name, c.Uninstall, s)
		}
	}
}

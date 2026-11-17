package installer

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// HCD-047, tarea 9: los ocho escenarios del ACR de v1.15.0 con el binario
// compilado, en HOME temporales, datos sintéticos y PATH sin Pi. Repite el
// reproductor del informe con las aserciones del comportamiento corregido.

type hcd047Home struct {
	t                                              *testing.T
	bin, home, agent, state, settings, mcp, agents string
}

func (h hcd047Home) run(args ...string) (int, string) {
	h.t.Helper()
	cmd := exec.Command(h.bin, args...)
	cmd.Dir = h.home
	cmd.Env = []string{"HOME=" + h.home, "PATH=/usr/bin:/bin", "PI_GOMEMORY_NO_UPDATE_CHECK=1"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), string(out)
		}
		h.t.Fatalf("%v: %v", args, err)
	}
	return 0, string(out)
}

func (h hcd047Home) mustRun(args ...string) string {
	h.t.Helper()
	code, out := h.run(args...)
	if code != 0 {
		h.t.Fatalf("%v salió %d:\n%s", args, code, out)
	}
	return out
}

func (h hcd047Home) tools(path string) []string {
	h.t.Helper()
	var obj struct {
		DefaultTools []string `json:"defaultTools"`
	}
	if err := json.Unmarshal([]byte(readFile(h.t, path)), &obj); err != nil {
		h.t.Fatalf("%s: %v", path, err)
	}
	return obj.DefaultTools
}

func TestHCD047Binary(t *testing.T) {
	if testing.Short() {
		t.Skip("compila el binario")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "pi-gomemory-setup")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	setup := func(t *testing.T) hcd047Home {
		home := t.TempDir()
		agent := filepath.Join(home, ".pi", "agent")
		mkdir(t, agent)
		return hcd047Home{t: t, bin: bin, home: home, agent: agent,
			state:    CodemodeStatePath(home),
			settings: filepath.Join(agent, "settings.json"),
			mcp:      filepath.Join(agent, "mcp.json"),
			agents:   filepath.Join(agent, "AGENTS.md")}
	}
	block := instructionsStart + "\nSYNTH-PROTOCOL\n" + instructionsEnd + "\n"

	t.Run("off persistido (C-02)", func(t *testing.T) {
		h := setup(t)
		h.mustRun("codemode", "off")
		writeFile(t, h.settings, `{"defaultTools":["+codemode"]}`)
		h.mustRun("install")
		if st := hcd047RawState(t, h.home); st["preference"] != "off" {
			t.Fatalf("preferencia: %v", st)
		}
		if s := hcd047Servers(t, h.agent); !strings.Contains(string(s["gomemory"]), `"direct"`) || strings.Contains(string(s["gomemory"]), "codemode") {
			t.Fatalf("gomemory no quedó directo: %s", s["gomemory"])
		}
		if !slices.Contains(h.tools(h.settings), "+codemode") {
			t.Fatal("install retiró el +codemode de la persona")
		}
	})
	t.Run("propiedad tras reintento (C-01)", func(t *testing.T) {
		h := setup(t)
		writeFile(t, h.mcp, "{")
		if code, _ := h.run("codemode", "on"); code == 0 {
			t.Fatal("codemode on con mcp.json inválido salió 0")
		}
		if _, err := os.Stat(h.settings); !os.IsNotExist(err) {
			t.Fatalf("el fallo dejó settings.json escrito: %v", err)
		}
		writeFile(t, h.mcp, "{}")
		h.mustRun("codemode", "on")
		if st := hcd047RawState(t, h.home); st["added_plus_codemode"] != true {
			t.Fatalf("propiedad: %v", st)
		}
		h.mustRun("uninstall", "--yes")
		if slices.Contains(h.tools(h.settings), "+codemode") {
			t.Fatal("uninstall dejó el +codemode del instalador")
		}
	})
	t.Run("migración después de la extensión (C-03)", func(t *testing.T) {
		h := setup(t)
		ext := filepath.Join(h.agent, "extensions", "gomemory-context.ts")
		mkdir(t, ext)
		original := "SYNTH-USER\n\n" + block
		writeFile(t, h.agents, original)
		if code, _ := h.run("install"); code == 0 {
			t.Fatal("install con el destino ocupado salió 0")
		}
		if readFile(t, h.agents) != original {
			t.Fatal("el fallo alteró AGENTS.md")
		}
		if err := os.Remove(ext); err != nil {
			t.Fatal(err)
		}
		h.mustRun("install")
		if info, err := os.Stat(ext); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("la extensión no se instaló: %v", err)
		}
		if got := readFile(t, h.agents); got != "SYNTH-USER\n" {
			t.Fatalf("AGENTS.md: %q", got)
		}
	})
	t.Run("perfil histórico (C-06)", func(t *testing.T) {
		h := setup(t)
		writeFile(t, h.mcp, readFile(t, filepath.Join("testdata", "hcd047", "mcp-v1.14.0.json")))
		h.mustRun("uninstall", "--yes")
		if s := hcd047Servers(t, h.agent); len(s) != 0 {
			t.Fatalf("quedaron servidores: %v", s)
		}
	})
	t.Run("marca huérfana (C-04)", func(t *testing.T) {
		h := setup(t)
		original := instructionsStart + "\nSYNTH-USER\n" + block
		writeFile(t, h.agents, original)
		out := h.mustRun("install")
		if readFile(t, h.agents) != original || !strings.Contains(out, "ambiguos") {
			t.Fatalf("AGENTS.md ambiguo alterado o sin aviso:\n%s", out)
		}
	})
	t.Run("estado inválido (C-07)", func(t *testing.T) {
		h := setup(t)
		h.mustRun("codemode", "on")
		writeFile(t, h.state, "{")
		if code, out := h.run("uninstall", "--yes"); code == 0 || !strings.Contains(out, h.state) {
			t.Fatalf("uninstall no informó (salida %d):\n%s", code, out)
		}
		if readFile(t, h.state) != "{" || !slices.Contains(h.tools(h.settings), "+codemode") {
			t.Fatal("no se conservaron el estado inválido y el token")
		}
	})
	t.Run("cambio de dotfiles (C-08)", func(t *testing.T) {
		h := setup(t)
		a, b := filepath.Join(h.home, "A.json"), filepath.Join(h.home, "B.json")
		writeFile(t, a, `{}`)
		writeFile(t, b, `{"defaultTools":["+codemode"]}`)
		hcd047Link(t, h.settings, a)
		h.mustRun("codemode", "on")
		hcd047Link(t, h.settings, b)
		h.mustRun("uninstall", "--yes")
		if !slices.Contains(h.tools(b), "+codemode") {
			t.Fatal("uninstall retiró el +codemode del otro destino")
		}
	})
	t.Run("dos formatos (C-05)", func(t *testing.T) {
		h := setup(t)
		historical := readFile(t, filepath.Join("legacy", "protocol-1.txt"))
		writeFile(t, h.agents, historical+"\nSYNTH-USER\n"+block)
		h.mustRun("install")
		got := readFile(t, h.agents)
		if strings.Contains(got, historical) || strings.Contains(got, "SYNTH-PROTOCOL") || !strings.Contains(got, "SYNTH-USER") {
			t.Fatalf("AGENTS.md: %q", got)
		}
	})
}

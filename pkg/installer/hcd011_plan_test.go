package installer

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// snapshotTree registra contenido y destinos de enlaces bajo root.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(p)
			snap[p] = "link:" + target
			return nil
		}
		if d.IsDir() {
			snap[p] = "dir"
			return nil
		}
		data, _ := os.ReadFile(p)
		snap[p] = string(data)
		return nil
	})
	return snap
}

func actionFor(plan *UninstallPlan, target string) *UninstallAction {
	for i := range plan.Actions {
		if plan.Actions[i].Target == target {
			return &plan.Actions[i]
		}
	}
	return nil
}

// Tarea 3 · C-05, C-06, C-07, C-11, C-14: el plan aplica la regla de propiedad.
func TestHCD011Plan(t *testing.T) {
	opts := installSandbox(t)
	ext := filepath.Join(opts.PiAgentDir, "extensions")
	repo := SkillRepoDir(opts.HomeDir)

	writeFile(t, filepath.Join(ext, "gomemory-tasks.ts"), "// editado a mano\n")
	writeFile(t, filepath.Join(repo, "mia", "SKILL.md"), "---\nname: mia\n---\n")
	writeFile(t, filepath.Join(repo, "adversarial-consensus-review", "notas.md"), "mías\n")
	ajena := filepath.Join(opts.HomeDir, "otras", "skill")
	writeFile(t, filepath.Join(ajena, "SKILL.md"), "x")
	if err := os.Symlink(ajena, filepath.Join(repo, "enlace-propio")); err != nil {
		t.Fatal(err)
	}
	hostSkill := filepath.Join(opts.HomeDir, ".claude", "skills", "anfitriona")
	writeFile(t, filepath.Join(hostSkill, "SKILL.md"), "---\nname: anfitriona\n---\n")
	if err := os.Symlink(hostSkill, filepath.Join(repo, "anfitriona")); err != nil {
		t.Fatal(err)
	}
	mcp := filepath.Join(opts.PiAgentDir, "mcp.json")
	writeFile(t, mcp, strings.Replace(readFile(t, mcp), `"mcpServers": {`, `"mcpServers": {
    "ajeno": {"command": "x"},`, 1))
	writeFile(t, mcp, strings.Replace(readFile(t, mcp), `"serve",`, `"serve", "--extra",`, 1))
	agents := filepath.Join(opts.PiAgentDir, "AGENTS.md")
	writeFile(t, agents, "Mis reglas\n\n"+readFile(t, agents))

	plan, err := BuildUninstallPlan(repoAssets(t), opts, UninstallFlags{})
	if err != nil {
		t.Fatal(err)
	}
	expect := map[string]string{
		filepath.Join(ext, "gomemory-status.ts"):                     ActionRemove,
		filepath.Join(ext, "gomemory-tasks.ts"):                      ActionSkip,
		filepath.Join(opts.PiAgentDir, "themes", "gomemory.json"):    ActionRemove,
		filepath.Join(opts.LocalBinDir, "pi-gomemory"):               ActionRemove,
		filepath.Join(repo, "hcd"):                                   ActionRemove,
		filepath.Join(repo, "adversarial-consensus-review"):          ActionSkip,
		filepath.Join(repo, "mia"):                                   ActionSkip,
		filepath.Join(repo, "enlace-propio"):                         ActionSkip,
		filepath.Join(repo, "anfitriona"):                            ActionRemove,
		mcp + " › mcpServers.gomemory":                               ActionRemove,
		mcp + " › mcpServers.codegraph":                              ActionSkip,
		agents + " › bloque pi-gomemory":                             ActionRemove,
		filepath.Join(opts.PiAgentDir, "settings.json") + " › theme": ActionRemove,
	}
	for target, kind := range expect {
		a := actionFor(plan, target)
		if a == nil || a.Kind != kind {
			t.Errorf("%s: quiero %s, obtengo %+v", target, kind, a)
		}
	}
	if a := actionFor(plan, mcp+" › mcpServers.ajeno"); a != nil {
		t.Errorf("el plan toca un servidor ajeno: %+v", a)
	}

	// C-14: un tema distinto de gomemory se conserva.
	settings := filepath.Join(opts.PiAgentDir, "settings.json")
	writeFile(t, settings, strings.Replace(readFile(t, settings), `"theme": "gomemory"`, `"theme": "otro"`, 1))
	plan, _ = BuildUninstallPlan(repoAssets(t), opts, UninstallFlags{})
	if a := actionFor(plan, settings+" › theme"); a != nil && a.Kind == ActionRemove {
		t.Error("se retiraría un tema que no es gomemory")
	}
	// C-14: con el archivo del tema editado, la clave tampoco se retira.
	writeFile(t, settings, strings.Replace(readFile(t, settings), `"theme": "otro"`, `"theme": "gomemory"`, 1))
	writeFile(t, filepath.Join(opts.PiAgentDir, "themes", "gomemory.json"), "{}\n")
	plan, _ = BuildUninstallPlan(repoAssets(t), opts, UninstallFlags{})
	if a := actionFor(plan, settings+" › theme"); a == nil || a.Kind != ActionKeep {
		t.Errorf("con el tema editado la clave debe conservarse: %+v", a)
	}
}

// Tarea 4 · C-01: --dry-run lista el plan y no cambia ningún archivo.
func TestHCD011DryRun(t *testing.T) {
	opts := installSandbox(t)
	before := snapshotTree(t, opts.HomeDir)
	var out bytes.Buffer
	if err := Uninstall(repoAssets(t), opts, UninstallFlags{DryRun: true}, &out); err != nil {
		t.Fatal(err)
	}
	after := snapshotTree(t, opts.HomeDir)
	if len(before) != len(after) {
		t.Fatalf("dry-run cambió el árbol: %d → %d entradas", len(before), len(after))
	}
	for p, v := range before {
		if after[p] != v {
			t.Fatalf("dry-run modificó %s", p)
		}
	}
	for _, want := range []string{"gomemory-status.ts", "mcpServers.gomemory", "bloque pi-gomemory", "dry-run"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("la salida no menciona %q:\n%s", want, out.String())
		}
	}
}

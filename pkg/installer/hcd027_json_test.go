package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestHCD027MCPModes(t *testing.T) {
	operations := []struct {
		name string
		run  func(*Options) error
	}{
		{"mcp", func(opts *Options) error { return InstallMCPServers(opts.PiAgentDir) }},
		{"install", func(opts *Options) error { return InstallAll(repoAssets(t), opts) }},
		{"uninstall-json", func(opts *Options) error {
			plan := &UninstallPlan{}
			plan.planMCP(opts.PiAgentDir)
			if plan.Pending() == 0 {
				return fmt.Errorf("fixture did not schedule MCP removal")
			}
			for _, action := range plan.Actions {
				if action.Kind == ActionRemove {
					if err := action.apply(); err != nil {
						return err
					}
				}
			}
			return nil
		}},
	}
	for _, op := range operations {
		for _, mode := range []os.FileMode{0600, 0640, 0644} {
			t.Run(fmt.Sprintf("%s/%04o", op.name, mode), func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				// InstallAll runs doctor; never start the real agent from this fixture.
				fakeBin := t.TempDir()
				writeExec(t, filepath.Join(fakeBin, "pi"), "exit 0")
				t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
				opts := &Options{
					HomeDir: home, PiAgentDir: filepath.Join(home, ".pi", "agent"),
					LocalBinDir: filepath.Join(home, ".local", "bin"),
				}
				servers := map[string]any{
					"foreign": map[string]any{
						"command": "synthetic-command",
						"env":     map[string]string{"API_KEY": "SYNTH-TOKEN-0001"},
					},
				}
				for name, def := range DefaultMCPServers() {
					servers[name] = def
				}
				data, err := json.Marshal(map[string]any{"mcpServers": servers, "foreignRoot": "keep"})
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(opts.PiAgentDir, "mcp.json")
				writeFile(t, path, string(data))
				if err := os.Chmod(path, mode); err != nil {
					t.Fatal(err)
				}
				if err := op.run(opts); err != nil {
					t.Fatal(err)
				}
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if got := info.Mode().Perm(); got != mode {
					t.Errorf("permissions changed: got %04o, want %04o", got, mode)
				}
				obj, err := readJSONObject(path)
				if err != nil {
					t.Fatal(err)
				}
				var root string
				if err := json.Unmarshal(obj["foreignRoot"], &root); err != nil || root != "keep" {
					t.Fatalf("foreign root field changed: %q, %v", root, err)
				}
				var gotServers map[string]struct {
					Command string            `json:"command"`
					Env     map[string]string `json:"env"`
				}
				if err := json.Unmarshal(obj["mcpServers"], &gotServers); err != nil {
					t.Fatal(err)
				}
				foreign, ok := gotServers["foreign"]
				if !ok || foreign.Command != "synthetic-command" || foreign.Env["API_KEY"] != "SYNTH-TOKEN-0001" {
					t.Fatal("foreign server fields were not preserved")
				}
			})
		}
	}
	t.Run("new-json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.json")
		if err := writeJSONObject(path, map[string]any{"synthetic": true}); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0600 {
			t.Fatalf("new JSON permissions: got %04o, want 0600", got)
		}
	})
	t.Run("new-mcp", func(t *testing.T) {
		dir := t.TempDir()
		if err := InstallMCPServers(dir); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(filepath.Join(dir, "mcp.json"))
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0600 {
			t.Fatalf("new MCP permissions: got %04o, want 0600", got)
		}
	})
}

func TestHCD027JSONAtomicErrors(t *testing.T) {
	t.Run("stat_permission_denied", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("skipping permission test under root")
		}
		dir := t.TempDir()
		subDir := filepath.Join(dir, "no_access")
		if err := os.Mkdir(subDir, 0700); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(subDir, "settings.json")
		originalData := `{"keep": true}`
		if err := os.WriteFile(target, []byte(originalData), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(subDir, 0000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(subDir, 0700)
		})

		err := writeJSONObject(target, map[string]any{"overwrite": true})
		if err == nil {
			t.Fatal("expected error when stat fails due to permissions, got nil")
		}

		if err := os.Chmod(subDir, 0700); err != nil {
			t.Fatal(err)
		}
		data, readErr := os.ReadFile(target)
		if readErr != nil {
			t.Fatalf("could not read target file: %v", readErr)
		}
		if string(data) != originalData {
			t.Fatalf("original file was modified: got %s, want %s", string(data), originalData)
		}

		for _, checkDir := range []string{subDir, dir} {
			entries, err := os.ReadDir(checkDir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.Name() != filepath.Base(target) && entry.Name() != filepath.Base(subDir) {
					t.Errorf("unexpected residual file in %s: %s", checkDir, entry.Name())
				}
			}
		}
	})

	t.Run("read_only_parent_dir", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("skipping permission test under root")
		}
		dir := t.TempDir()
		target := filepath.Join(dir, "config.json")
		originalData := "{\n  \"stable\": \"value\"\n}\n"
		if err := os.WriteFile(target, []byte(originalData), 0640); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dir, 0500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(dir, 0700)
		})

		err := writeJSONObject(target, map[string]any{"stable": "mutated"})
		if err == nil {
			t.Fatal("expected error writing to read-only directory, got nil")
		}

		_ = os.Chmod(dir, 0700)
		data, readErr := os.ReadFile(target)
		if readErr != nil {
			t.Fatalf("could not read target file: %v", readErr)
		}
		if string(data) != originalData {
			t.Fatalf("original file was modified: got %s, want %s", string(data), originalData)
		}
		info, statErr := os.Stat(target)
		if statErr != nil {
			t.Fatal(statErr)
		}
		if perm := info.Mode().Perm(); perm != 0640 {
			t.Fatalf("permissions changed: got %04o, want 0640", perm)
		}

		matches, globErr := filepath.Glob(filepath.Join(dir, ".*.tmp-*"))
		if globErr != nil {
			t.Fatal(globErr)
		}
		if len(matches) > 0 {
			t.Fatalf("residual temp files found: %v", matches)
		}
	})

	t.Run("target_is_directory", func(t *testing.T) {
		dir := t.TempDir()
		targetDir := filepath.Join(dir, "dir_target")
		if err := os.Mkdir(targetDir, 0755); err != nil {
			t.Fatal(err)
		}
		marker := filepath.Join(targetDir, "marker.txt")
		if err := os.WriteFile(marker, []byte("preserve me"), 0644); err != nil {
			t.Fatal(err)
		}

		err := writeJSONObject(targetDir, map[string]any{"data": 123})
		if err == nil {
			t.Fatal("expected error writing JSON over directory, got nil")
		}

		content, readErr := os.ReadFile(marker)
		if readErr != nil {
			t.Fatalf("marker unreadable: %v", readErr)
		}
		if string(content) != "preserve me" {
			t.Fatalf("directory contents disturbed: %s", string(content))
		}

		matches, globErr := filepath.Glob(filepath.Join(dir, ".*.tmp-*"))
		if globErr != nil {
			t.Fatal(globErr)
		}
		if len(matches) > 0 {
			t.Fatalf("residual temp files found: %v", matches)
		}
	})

	t.Run("unserializable_object", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "valid.json")
		originalData := "{\n  \"valid\": true\n}\n"
		if err := os.WriteFile(target, []byte(originalData), 0600); err != nil {
			t.Fatal(err)
		}

		err := writeJSONObject(target, map[string]any{"unsupported": make(chan int)})
		if err == nil {
			t.Fatal("expected error marshaling channel, got nil")
		}

		data, readErr := os.ReadFile(target)
		if readErr != nil {
			t.Fatalf("target unreadable: %v", readErr)
		}
		if string(data) != originalData {
			t.Fatalf("original file modified on marshal error: got %s, want %s", string(data), originalData)
		}

		matches, globErr := filepath.Glob(filepath.Join(dir, ".*.tmp-*"))
		if globErr != nil {
			t.Fatal(globErr)
		}
		if len(matches) > 0 {
			t.Fatalf("residual temp files found: %v", matches)
		}
	})
}

package installer

import (
	"path/filepath"
	"strings"
	"testing"
)

// HCD-062 T9 · M-1: el launcher busca pi también en ~/.pi/agent/bin, donde
// lo deja el instalador oficial si ningún otro candidato está en el PATH.
func TestHCD062LauncherDoctorLauncherPath(t *testing.T) {
	script := GenerateHarnessScript()
	if !strings.Contains(script, `export PATH="${SELF_DIR}:${HOME}/.local/bin:${HOME}/.pi/agent/bin:${HOME}/.gomemory/bin:${PATH}"`) {
		t.Fatalf("el launcher debe añadir ~/.pi/agent/bin a su PATH:\n%s", script)
	}
}

// N-2: doctor amplía el PATH con los directorios conocidos antes de buscar.
func TestHCD062LauncherDoctorFindsKnownDirs(t *testing.T) {
	opts := toolSandbox(t)
	fakeTool(t, filepath.Join(opts.PiAgentDir, "bin"), "pi")
	for _, item := range RunDoctor(opts) {
		if item.Name == "Binario pi" {
			if item.Status != "OK" {
				t.Fatalf("doctor no encuentra pi en ~/.pi/agent/bin: %+v", item)
			}
			return
		}
	}
	t.Fatal("doctor no informa el binario pi")
}

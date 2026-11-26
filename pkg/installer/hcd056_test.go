package installer

import (
	"strings"
	"testing"
)

// Las tareas 1–4 de HCD-056 (filtro, ingesta de Bash y watch de /tdd) se
// retiraron con /tdd en HCD-085.

// HCD-056 tarea 5 (Q3): las skills HCD que llegan a Pi exigen una comprobación
// acotada por tarea, una única regresión final y el rojo en toda tarea con un
// test automatizable, con ejemplos de varios lenguajes.
func TestHCD056SkillsTDD(t *testing.T) {
	t.Parallel()

	tasks := readAsset(t, "skills/hcd-tasks/SKILL.md")
	for _, want := range []string{"comprobación acotada", "regresión completa", "go test ./pkg/x -run", "pytest", "vitest"} {
		if !strings.Contains(tasks, want) {
			t.Errorf("hcd-tasks/SKILL.md no contiene %q", want)
		}
	}
	impl := readAsset(t, "skills/hcd-implement/SKILL.md")
	for _, want := range []string{"Rojo antes del cambio en toda tarea con un test automatizable", "--expect-fail", "comprobación acotada", "No ejecutes la misma comprobación fuera y dentro del motor", "no automatizable"} {
		if !strings.Contains(impl, want) {
			t.Errorf("hcd-implement/SKILL.md no contiene %q", want)
		}
	}
}

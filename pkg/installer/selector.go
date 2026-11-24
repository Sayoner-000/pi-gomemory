package installer

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// runSelector muestra la lista de tool con su estado y deja marcar o
// desmarcar escribiendo sus números; Enter sin texto confirma (HCD-076,
// C-09). Los obligatorios no se pueden desmarcar. Lee de in, que es
// /dev/tty: con `curl | bash` la entrada estándar es el propio script.
func runSelector(tools []Tool, sel map[string]bool, states []DepState, in io.Reader, out io.Writer) (map[string]bool, error) {
	byName := map[string]DepState{}
	for _, d := range states {
		byName[d.Name] = d
	}
	cur := map[string]bool{}
	for k, v := range sel {
		cur[k] = v
	}
	sc := bufio.NewScanner(in)
	for {
		fmt.Fprintf(out, "\n%s    Componentes del meta-harness: escribe números para marcar o desmarcar y pulsa Enter para continuar.%s\n", ColorBrillo, ColorReset)
		for i, t := range tools {
			box := "[ ]"
			if cur[t.Name] {
				box = "[x]"
			}
			kind := "opcional"
			if t.Required {
				kind = "obligatorio"
			}
			fmt.Fprintf(out, "    %d. %s %-20s %-22s %s\n", i+1, box, t.Name, versionText(byName[t.Name]), kind)
		}
		fmt.Fprint(out, "    > ")
		if !sc.Scan() {
			fmt.Fprintln(out)
			return cur, sc.Err()
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			return cur, nil
		}
		for _, f := range strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == ',' }) {
			n, err := strconv.Atoi(f)
			if err != nil || n < 1 || n > len(tools) {
				fmt.Fprintf(out, "%s    ⚠ %q no es un número de la lista%s\n", ColorWarning, f, ColorReset)
				continue
			}
			t := tools[n-1]
			if t.Required {
				fmt.Fprintf(out, "%s    ⚠ %s es obligatorio: no se puede desmarcar%s\n", ColorWarning, t.Name, ColorReset)
				continue
			}
			cur[t.Name] = !cur[t.Name]
		}
	}
}

// versionText resume el estado de un tool para el selector.
func versionText(d DepState) string {
	switch {
	case d.Current == "":
		if d.Latest != "" {
			return "no instalado → " + d.Latest
		}
		return "no instalado"
	case d.Latest == "" || d.Latest == d.Current:
		return d.Current
	}
	return d.Current + " → " + d.Latest
}

//go:build pins

package installer

import (
	"context"
	"os"
	"testing"
)

// TestPinsValidate comprueba un candidato del registro npm real contra los
// contratos de su paquete (HCD-039):
//
//	PINS_CANDIDATE=pi-antigravity@0.14.0 go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsValidate$'
//
// Sin PINS_CANDIDATE se omite. No escribe en ~/.pi/agent.
func TestPinsValidate(t *testing.T) {
	spec := os.Getenv("PINS_CANDIDATE")
	if spec == "" {
		t.Skip("define PINS_CANDIDATE=<paquete>@<versión> para validar un candidato")
	}
	results, err := validateCandidate(context.Background(), spec)
	if err != nil {
		t.Fatalf("pins-validate: %v", err)
	}
	lines, fail := FormatContracts(spec, results)
	for _, l := range lines {
		t.Log(l)
	}
	if fail {
		t.Fatalf("pins-validate: %s rompe algún contrato de pi-gomemory", spec)
	}
}

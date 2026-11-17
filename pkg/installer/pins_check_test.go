//go:build pins

package installer

import (
	"context"
	"os"
	"testing"
)

// TestPinsCheck compara los paquetes fijados con el registro npm real (HCD-039).
// Se ejecuta siempre sin caché y con salida visible:
//
//	go test -tags pins -count=1 -v ./pkg/installer -run '^TestPinsCheck$'
//
// Con PI_GOMEMORY_ALLOW_STALE_PINS=1 informa el desfase o el error y pasa.
func TestPinsCheck(t *testing.T) {
	lines, fail := FormatPinChecks(CheckPins(context.Background()), os.Getenv("PI_GOMEMORY_ALLOW_STALE_PINS") == "1")
	for _, l := range lines {
		t.Log(l)
	}
	if fail {
		t.Fatal("pins-check: hay paquetes fijados atrasados frente a npm o sin comprobar")
	}
}

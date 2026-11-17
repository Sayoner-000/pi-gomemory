package installer

import (
	"os"
	"testing"
)

// blockedNPMRegistry no acepta conexiones (puerto 1 de loopback): una prueba
// que consulte npm sin su propio servidor local falla en vez de salir a internet.
const blockedNPMRegistry = "http://127.0.0.1:1/npm-bloqueado-en-pruebas"

// TestMain aísla la suite normal del registro npm real (HCD-039, C-07). Con la
// etiqueta pins se conserva la URL real para TestPinsCheck y TestPinsValidate.
func TestMain(m *testing.M) {
	if !pinsLive {
		NPMRegistryURL = blockedNPMRegistry
	}
	os.Exit(m.Run())
}

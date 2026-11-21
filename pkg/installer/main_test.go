package installer

import (
	"errors"
	"net/http"
	"os"
	"testing"
)

// blockedTransport rechaza toda petición: una prueba del paso de
// herramientas sin su propio servidor falla en vez de salir a internet.
type blockedTransport struct{}

func (blockedTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("red bloqueada en pruebas (HCD-062)")
}

// blockedNPMRegistry no acepta conexiones (puerto 1 de loopback): una prueba
// que consulte npm sin su propio servidor local falla en vez de salir a internet.
const blockedNPMRegistry = "http://127.0.0.1:1/npm-bloqueado-en-pruebas"

// TestMain aísla la suite normal del registro npm real (HCD-039, C-07). Con la
// etiqueta pins se conserva la URL real para TestPinsCheck y TestPinsValidate.
func TestMain(m *testing.M) {
	if !pinsLive {
		NPMRegistryURL = blockedNPMRegistry
	}
	// HCD-062: ningún test sale a la red a instalar herramientas ni encuentra
	// binarios reales en /usr/local/bin; los tests del paso fijan los suyos.
	installTools = nil
	hooks.SystemDirs = nil
	hooks.Client = &http.Client{Transport: blockedTransport{}}
	os.Exit(m.Run())
}

package installer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// HCD-033 · C-002: httpJSON no depende de que el llamador pase un plazo. Su
// cliente tiene timeout propio (depsProbeTimeout) y una conexión que no
// responde termina con error aunque el contexto no tenga deadline.
func TestHCD033HTTPJSONClientTimeout(t *testing.T) {
	if depsHTTPClient.Timeout != depsProbeTimeout {
		t.Fatalf("timeout del cliente de httpJSON = %v; quiero %v", depsHTTPClient.Timeout, depsProbeTimeout)
	}

	// Mismo cliente con un plazo corto para no esperar 20 s en la prueba.
	hang := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-hang }))
	defer srv.Close()
	defer close(hang)
	prev := depsHTTPClient
	short := *depsHTTPClient
	short.Timeout = 200 * time.Millisecond
	depsHTTPClient = &short
	defer func() { depsHTTPClient = prev }()

	start := time.Now()
	var dst map[string]any
	err := httpJSON(context.Background(), srv.URL, &dst)
	if err == nil {
		t.Fatal("httpJSON devolvió éxito ante un servidor que no responde")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("httpJSON tardó %v: el timeout del cliente no se aplicó", elapsed)
	}
}

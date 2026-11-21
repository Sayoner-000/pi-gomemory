package installer

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

// Temporales del paso de herramientas (HCD-062, M-3 de la cuarta revisión):
// un Ctrl-C mata a pi-gomemory sin ejecutar sus defer, así que mientras haya
// temporales vivos se captura SIGINT/SIGTERM, se borran y se sale con 130.

var (
	tempMu    sync.Mutex
	liveTemps = map[string]bool{}
	sigCh     chan os.Signal
)

// signalExit es la salida tras la limpieza; variable para las pruebas.
var signalExit = func(code int) { os.Exit(code) }

func trackTemp(dir string) func() {
	tempMu.Lock()
	defer tempMu.Unlock()
	liveTemps[dir] = true
	if sigCh == nil {
		sigCh = make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func(ch chan os.Signal) {
			if _, ok := <-ch; ok {
				removeLiveTemps()
				signalExit(130)
			}
		}(sigCh)
	}
	return func() {
		tempMu.Lock()
		defer tempMu.Unlock()
		delete(liveTemps, dir)
		if len(liveTemps) == 0 && sigCh != nil {
			signal.Stop(sigCh)
			close(sigCh)
			sigCh = nil
		}
	}
}

func removeLiveTemps() {
	tempMu.Lock()
	defer tempMu.Unlock()
	for d := range liveTemps {
		_ = os.RemoveAll(d)
		delete(liveTemps, d)
	}
}

// RemoveStaleDepsTemps retira de dir los temporales del paso que quedaron de
// una ejecución que no pudo limpiar (por ejemplo, un SIGKILL).
func RemoveStaleDepsTemps(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var removed []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), depsTempPrefix) {
			p := filepath.Join(dir, e.Name())
			tempMu.Lock()
			live := liveTemps[p]
			tempMu.Unlock()
			if !live && os.RemoveAll(p) == nil {
				removed = append(removed, p)
			}
		}
	}
	return removed
}

package installer

import (
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// Temporales del paso de herramientas (HCD-062, M-3 de la cuarta revisión):
// un Ctrl-C mata a pi-gomemory sin ejecutar sus defer, así que mientras haya
// temporales vivos se captura SIGINT/SIGTERM, se borran y se sale con 130.

var (
	tempMu       sync.Mutex
	liveTemps    = map[string]bool{}
	liveChildren = map[int]liveChild{}
	sigCh        chan os.Signal
)

type liveChild struct {
	pgid   bool
	cancel func()
}

// signalExit es la salida tras la limpieza; variable para las pruebas.
var signalExit = func(code int) { os.Exit(code) }

func trackLiveChild(pid int, pgid bool, cancel func()) func() {
	tempMu.Lock()
	liveChildren[pid] = liveChild{pgid: pgid, cancel: cancel}
	tempMu.Unlock()
	return func() {
		tempMu.Lock()
		delete(liveChildren, pid)
		tempMu.Unlock()
	}
}

func killLiveChildren() {
	tempMu.Lock()
	snapshot := make(map[int]liveChild, len(liveChildren))
	for pid, c := range liveChildren {
		snapshot[pid] = c
	}
	tempMu.Unlock()
	for pid, c := range snapshot {
		if c.cancel != nil {
			c.cancel()
		}
		if c.pgid {
			_ = syscall.Kill(-pid, syscall.SIGKILL)
		} else {
			killProcessTree(pid)
		}
	}
}

func trackTemp(dir string) func() {
	tempMu.Lock()
	defer tempMu.Unlock()
	liveTemps[dir] = true
	if sigCh == nil {
		sigCh = make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func(ch chan os.Signal) {
			if sig, ok := <-ch; ok {
				killLiveChildren()
				removeLiveTemps()
				code := 130
				if s, ok := sig.(syscall.Signal); ok {
					code = 128 + int(s)
				}
				signalExit(code)
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
			if !live && !depsTempOwnedByLiveProcess(p) && os.RemoveAll(p) == nil {
				removed = append(removed, p)
			}
		}
	}
	return removed
}

func depsTempOwnedByLiveProcess(dir string) bool {
	b, err := os.ReadFile(filepath.Join(dir, ".pid"))
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return false
	}
	err = syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

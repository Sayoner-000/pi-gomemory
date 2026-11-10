package installer

import (
	"fmt"
	"strings"
	"testing"
)

func TestClipboardDiagnostic(t *testing.T) {
	for _, tc := range []struct{ name, platform, display, wayland, bins, status, detail string }{
		{"mac", "darwin", "", "", "pbcopy pbpaste", "INFO", "pbcopy"},
		{"mac missing paste", "darwin", "", "", "pbcopy", "WARN", "pbpaste"},
		{"wslg", "linux", ":0", "", "xclip", "INFO", "xclip"},
		{"x11 missing", "linux", ":0", "", "", "WARN", "instalar xclip"},
		{"wayland", "linux", "", "wayland-0", "wl-copy wl-paste", "INFO", "wl-copy"},
		{"wayland missing paste", "linux", "", "wayland-0", "wl-copy", "WARN", "wl-clipboard"},
		{"xwayland fallback", "linux", ":0", "wayland-0", "xclip", "INFO", "xclip"},
		{"headless", "linux", "", "", "xclip", "INFO", "OSC 52"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lookup := func(name string) (string, error) {
				for _, bin := range strings.Fields(tc.bins) {
					if name == bin {
						return "/usr/bin/" + name, nil
					}
				}
				return "", fmt.Errorf("missing %s", name)
			}
			status, detail := clipboardDiagnostic(tc.platform, tc.display, tc.wayland, lookup)
			if status != tc.status || !strings.Contains(detail, tc.detail) {
				t.Fatalf("got %s: %s", status, detail)
			}
		})
	}
}

package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"pi-gomemory/pkg/installer"
)

// HCD-084: update ya al día también revisa las skills externas.
func TestHCD084UpdateRunsSkills(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v0.0.1"}`))
	}))
	defer srv.Close()
	var calls []string
	var out bytes.Buffer
	d := updateDeps{
		apiURL: srv.URL,
		sync:   func() error { calls = append(calls, "sync"); return nil },
		skills: func() error { calls = append(calls, "skills"); return nil },
		out:    &out,
	}
	if err := runUpdateWith(&installer.Options{HomeDir: t.TempDir()}, d); err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "sync,skills" {
		t.Fatalf("update al día llamó %v, quiero sync y después skills", calls)
	}
}

// HCD-084: el binario expone skills-update.
func TestHCD084SkillsUpdateCommand(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), `case "skills-update":`) || !strings.Contains(string(src), "installer.RunSkillsUpdate(EmbeddedAssets, opts, os.Args[2:]") {
		t.Fatal("main no despacha skills-update a installer.RunSkillsUpdate")
	}
}

// HCD-084: la ayuda documenta skills-update y las skills externas.
func TestHCD084Help(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	printHelp()
	os.Stdout = old
	_ = w.Close()
	data, _ := io.ReadAll(r)
	help := string(data)
	for _, want := range []string{"skills-update", "--if-due", "impeccable", "frontend-design", "playwright-cli", "gitlab-release"} {
		if !strings.Contains(help, want) {
			t.Fatalf("la ayuda no menciona %q", want)
		}
	}
}

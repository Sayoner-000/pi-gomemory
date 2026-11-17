package main

import (
	"bytes"
	"strings"
	"testing"

	"pi-gomemory/pkg/installer"
)

func TestCodemodeCLI(t *testing.T) {
	dir := t.TempDir()

	var out bytes.Buffer
	if err := runCodemode(dir, []string{"on"}, &out); err != nil {
		t.Fatalf("runCodemode on: %v", err)
	}
	if !strings.Contains(out.String(), "codemode") {
		t.Fatalf("expected codemode note in output, got %q", out.String())
	}

	st, err := installer.GetCodemodeStatus(dir)
	if err != nil {
		t.Fatalf("GetCodemodeStatus after on: %v", err)
	}
	if !st.Enabled {
		t.Fatalf("expected codemode enabled, got %+v", st)
	}

	out.Reset()
	if err := runCodemode(dir, []string{"status"}, &out); err != nil {
		t.Fatalf("runCodemode status: %v", err)
	}
	if !strings.Contains(out.String(), "on") {
		t.Fatalf("expected status output to mention on, got %q", out.String())
	}

	out.Reset()
	if err := runCodemode(dir, []string{"off"}, &out); err != nil {
		t.Fatalf("runCodemode off: %v", err)
	}
	stOff, err := installer.GetCodemodeStatus(dir)
	if err != nil {
		t.Fatalf("GetCodemodeStatus after off: %v", err)
	}
	if stOff.Enabled || stOff.SettingsEnabled || stOff.MCPEnabled {
		t.Fatalf("expected codemode disabled, got %+v", stOff)
	}

	if err := runCodemode(dir, []string{"invalid"}, &out); err == nil {
		t.Fatal("expected error for invalid subcommand")
	}
}

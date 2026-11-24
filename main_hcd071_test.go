package main

import (
	"os"
	"strings"
	"testing"
)

// HCD-071 tarea 4: el protocolo de Pi pide buscar una tool por codemode antes
// de darla por inexistente (skills copiadas de gomemory, como socratica, se
// rinden si no ven socratica_flow entre las directas).
func TestHCD071ContextProtocol(t *testing.T) {
	data, err := os.ReadFile("extensions/gomemory-context.ts")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, want := range []string{"antes de concluir que una tool no existe", "searchTools"} {
		if !strings.Contains(src, want) {
			t.Fatalf("falta %q en el protocolo de gomemory-context.ts", want)
		}
	}
}

// HCD-079 tarea 4: el protocolo HCD de Pi sigue a gomemory 2.59.0: con tareas
// pendientes una caducada no exige reverificar, y al cierre se revalida con
// `mem hcd verify --stale` sin comando (reejecuta el comando propio de cada una).
func TestHCD079ContextProtocol(t *testing.T) {
	data, err := os.ReadFile("extensions/gomemory-context.ts")
	if err != nil {
		t.Fatal(err)
	}
	src := string(data)
	for _, want := range []string{
		"una tarea caducada (stale) cuenta como progreso",
		"'mem hcd verify --stale' sin comando",
		"reejecuta el último comando verde de cada caducada",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("falta %q en el protocolo de gomemory-context.ts", want)
		}
	}
	if strings.Contains(src, "'mem hcd verify --stale' al cierre para revalidar caducadas en una sola llamada)") {
		t.Fatal("el protocolo conserva la redacción previa a gomemory 2.59.0")
	}
}

// HCD-071 tarea 5: los textos de CLI dicen que el híbrido es el valor por defecto.
func TestHCD071CLIText(t *testing.T) {
	note := codemodeNote()
	if !strings.Contains(note, "por defecto de install y update es codemode híbrido") {
		t.Fatalf("codemodeNote no describe el nuevo valor por defecto:\n%s", note)
	}
	if strings.Contains(note, "por defecto de install y update es directo") {
		t.Fatalf("codemodeNote conserva el valor por defecto anterior:\n%s", note)
	}
	inst, err := os.ReadFile("pkg/installer/installer.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(inst), "en modo codemode híbrido (codemode off para directo)") {
		t.Fatal("el paso [2/9] de install no anuncia el modo híbrido")
	}
}

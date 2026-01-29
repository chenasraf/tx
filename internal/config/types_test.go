package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTmuxWindowInput_UnmarshalYAML_String(t *testing.T) {
	yamlData := `"./src"`

	var w TmuxWindowInput
	err := yaml.Unmarshal([]byte(yamlData), &w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !w.IsString {
		t.Error("expected IsString to be true")
	}
	if w.String != "./src" {
		t.Errorf("expected String to be './src', got %q", w.String)
	}
	if w.Window != nil {
		t.Error("expected Window to be nil")
	}
}

func TestTmuxWindowInput_UnmarshalYAML_Struct(t *testing.T) {
	yamlData := `
name: mywindow
cwd: ./lib
`

	var w TmuxWindowInput
	err := yaml.Unmarshal([]byte(yamlData), &w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.IsString {
		t.Error("expected IsString to be false")
	}
	if w.Window == nil {
		t.Fatal("expected Window to not be nil")
	}
	if w.Window.Name != "mywindow" {
		t.Errorf("expected Name to be 'mywindow', got %q", w.Window.Name)
	}
	if w.Window.Cwd != "./lib" {
		t.Errorf("expected Cwd to be './lib', got %q", w.Window.Cwd)
	}
}

func TestTmuxLayoutInput_UnmarshalYAML_String(t *testing.T) {
	yamlData := `"./src"`

	var l TmuxLayoutInput
	err := yaml.Unmarshal([]byte(yamlData), &l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !l.IsString {
		t.Error("expected IsString to be true")
	}
	if l.String != "./src" {
		t.Errorf("expected String to be './src', got %q", l.String)
	}
}

func TestTmuxLayoutInput_UnmarshalYAML_Array(t *testing.T) {
	yamlData := `
- ./src
- ./lib
- ./test
`

	var l TmuxLayoutInput
	err := yaml.Unmarshal([]byte(yamlData), &l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !l.IsArray {
		t.Error("expected IsArray to be true")
	}
	if len(l.Array) != 3 {
		t.Errorf("expected 3 elements, got %d", len(l.Array))
	}
	expected := []string{"./src", "./lib", "./test"}
	for i, v := range expected {
		if l.Array[i] != v {
			t.Errorf("expected Array[%d] to be %q, got %q", i, v, l.Array[i])
		}
	}
}

func TestTmuxLayoutInput_UnmarshalYAML_PaneLayout(t *testing.T) {
	yamlData := `
cwd: ./src
cmd: npm start
zoom: true
split:
  direction: h
  child:
    cwd: ./lib
`

	var l TmuxLayoutInput
	err := yaml.Unmarshal([]byte(yamlData), &l)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if l.IsString || l.IsArray {
		t.Error("expected neither IsString nor IsArray")
	}
	if l.PaneLayout == nil {
		t.Fatal("expected PaneLayout to not be nil")
	}
	if l.PaneLayout.Cwd != "./src" {
		t.Errorf("expected Cwd to be './src', got %q", l.PaneLayout.Cwd)
	}
	if l.PaneLayout.Cmd != "npm start" {
		t.Errorf("expected Cmd to be 'npm start', got %q", l.PaneLayout.Cmd)
	}
	if !l.PaneLayout.Zoom {
		t.Error("expected Zoom to be true")
	}
	if l.PaneLayout.Split == nil {
		t.Fatal("expected Split to not be nil")
	}
	if l.PaneLayout.Split.Direction != "h" {
		t.Errorf("expected Split.Direction to be 'h', got %q", l.PaneLayout.Split.Direction)
	}
}

func TestConfigFile_UnmarshalYAML(t *testing.T) {
	yamlData := `
myproject:
  root: ~/Dev/myproject
  windows:
    - ./src
    - name: tests
      cwd: ./test

another:
  root: /tmp/another
  blank_window: true
`

	var config ConfigFile
	err := yaml.Unmarshal([]byte(yamlData), &config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(config) != 2 {
		t.Errorf("expected 2 configs, got %d", len(config))
	}

	myproject, ok := config["myproject"]
	if !ok {
		t.Fatal("expected 'myproject' config")
	}
	if myproject.Root != "~/Dev/myproject" {
		t.Errorf("expected Root to be '~/Dev/myproject', got %q", myproject.Root)
	}
	if len(myproject.Windows) != 2 {
		t.Errorf("expected 2 windows, got %d", len(myproject.Windows))
	}

	another, ok := config["another"]
	if !ok {
		t.Fatal("expected 'another' config")
	}
	if !another.BlankWindow {
		t.Error("expected BlankWindow to be true")
	}
}

func TestDefaultEmptyLayout(t *testing.T) {
	if DefaultEmptyLayout.Cwd != "." {
		t.Errorf("expected Cwd to be '.', got %q", DefaultEmptyLayout.Cwd)
	}
	if DefaultEmptyLayout.Split == nil {
		t.Fatal("expected Split to not be nil")
	}
	if DefaultEmptyLayout.Split.Direction != "h" {
		t.Errorf("expected Split.Direction to be 'h', got %q", DefaultEmptyLayout.Split.Direction)
	}
	if DefaultEmptyLayout.Split.Child == nil {
		t.Fatal("expected Split.Child to not be nil")
	}
	if DefaultEmptyLayout.Split.Child.Split == nil {
		t.Fatal("expected nested Split to not be nil")
	}
	if DefaultEmptyLayout.Split.Child.Split.Direction != "v" {
		t.Errorf("expected nested Split.Direction to be 'v', got %q", DefaultEmptyLayout.Split.Child.Split.Direction)
	}
}

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

func TestConfigFile_Get_ExactMatch(t *testing.T) {
	config := ConfigFile{
		"notes": {Root: "/tmp/notes"},
		"work":  {Root: "/tmp/work"},
	}

	item, actualKey, ok := config.Get("notes")
	if !ok {
		t.Fatal("expected to find 'notes'")
	}
	if actualKey != "notes" {
		t.Errorf("expected actualKey 'notes', got %q", actualKey)
	}
	if item.Root != "/tmp/notes" {
		t.Errorf("expected Root '/tmp/notes', got %q", item.Root)
	}
}

func TestConfigFile_Get_CaseInsensitive(t *testing.T) {
	config := ConfigFile{
		"Notes": {Root: "/tmp/notes"},
		"work":  {Root: "/tmp/work"},
	}

	tests := []struct {
		lookup    string
		wantKey   string
		wantFound bool
	}{
		{"Notes", "Notes", true},
		{"notes", "Notes", true},
		{"NOTES", "Notes", true},
		{"nOtEs", "Notes", true},
		{"work", "work", true},
		{"Work", "work", true},
		{"WORK", "work", true},
		{"missing", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.lookup, func(t *testing.T) {
			_, actualKey, ok := config.Get(tt.lookup)
			if ok != tt.wantFound {
				t.Errorf("Get(%q): found=%v, want %v", tt.lookup, ok, tt.wantFound)
			}
			if ok && actualKey != tt.wantKey {
				t.Errorf("Get(%q): actualKey=%q, want %q", tt.lookup, actualKey, tt.wantKey)
			}
		})
	}
}

func TestConfigFile_Get_ExactMatchTakesPrecedence(t *testing.T) {
	// If both "notes" and "Notes" exist, exact match should win
	config := ConfigFile{
		"notes": {Root: "/tmp/lower"},
		"Notes": {Root: "/tmp/upper"},
	}

	_, actualKey, ok := config.Get("notes")
	if !ok {
		t.Fatal("expected to find 'notes'")
	}
	if actualKey != "notes" {
		t.Errorf("expected exact match 'notes', got %q", actualKey)
	}

	_, actualKey, ok = config.Get("Notes")
	if !ok {
		t.Fatal("expected to find 'Notes'")
	}
	if actualKey != "Notes" {
		t.Errorf("expected exact match 'Notes', got %q", actualKey)
	}
}

func TestConfigFile_Get_NotFound(t *testing.T) {
	config := ConfigFile{
		"notes": {Root: "/tmp/notes"},
	}

	_, _, ok := config.Get("missing")
	if ok {
		t.Error("expected not found for 'missing'")
	}
}

func TestConfigFile_Get_EmptyConfig(t *testing.T) {
	config := ConfigFile{}

	_, _, ok := config.Get("anything")
	if ok {
		t.Error("expected not found in empty config")
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

func TestGetNamedLayout(t *testing.T) {
	// Save and restore original state
	originalLayouts := ConfiguredNamedLayouts
	defer func() { ConfiguredNamedLayouts = originalLayouts }()

	// Test when no named layouts are configured
	ConfiguredNamedLayouts = nil
	if layout := GetNamedLayout("test"); layout != nil {
		t.Error("expected nil when no named layouts configured")
	}

	// Test when named layout exists
	ConfiguredNamedLayouts = map[string]*TmuxPaneLayout{
		"dev": {
			Cwd: ".",
			Cmd: "npm run dev",
		},
		"simple": {
			Cwd:   ".",
			Clock: true,
		},
	}

	layout := GetNamedLayout("dev")
	if layout == nil {
		t.Fatal("expected to find 'dev' layout")
	}
	if layout.Cmd != "npm run dev" {
		t.Errorf("expected Cmd to be 'npm run dev', got %q", layout.Cmd)
	}

	layout = GetNamedLayout("simple")
	if layout == nil {
		t.Fatal("expected to find 'simple' layout")
	}
	if !layout.Clock {
		t.Error("expected Clock to be true")
	}

	// Test when named layout doesn't exist
	if layout := GetNamedLayout("nonexistent"); layout != nil {
		t.Error("expected nil for nonexistent layout")
	}
}

func TestGetDefaultLayout(t *testing.T) {
	// Save and restore original state
	originalLayout := ConfiguredDefaultLayout
	defer func() { ConfiguredDefaultLayout = originalLayout }()

	// Test when no custom default layout is configured
	ConfiguredDefaultLayout = nil
	layout := GetDefaultLayout()
	if layout != &DefaultEmptyLayout {
		t.Error("expected DefaultEmptyLayout when no custom layout configured")
	}

	// Test when custom default layout is configured
	customLayout := &TmuxPaneLayout{
		Cwd:   ".",
		Clock: true,
	}
	ConfiguredDefaultLayout = customLayout
	layout = GetDefaultLayout()
	if layout != customLayout {
		t.Error("expected custom layout when configured")
	}
}

func TestGlobalConfig_NamedLayouts(t *testing.T) {
	yamlData := `
shell: /bin/zsh
named_layouts:
  dev:
    cwd: .
    cmd: npm run dev
    split:
      direction: h
      child:
        cwd: .
        cmd: npm test
  simple:
    cwd: .
    clock: true
`

	var cfg GlobalConfig
	err := yaml.Unmarshal([]byte(yamlData), &cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Shell != "/bin/zsh" {
		t.Errorf("expected Shell to be '/bin/zsh', got %q", cfg.Shell)
	}

	if cfg.NamedLayouts == nil {
		t.Fatal("expected NamedLayouts to not be nil")
	}

	if len(cfg.NamedLayouts) != 2 {
		t.Errorf("expected 2 named layouts, got %d", len(cfg.NamedLayouts))
	}

	dev := cfg.NamedLayouts["dev"]
	if dev == nil {
		t.Fatal("expected 'dev' layout to exist")
	}
	if dev.Cmd != "npm run dev" {
		t.Errorf("expected dev.Cmd to be 'npm run dev', got %q", dev.Cmd)
	}
	if dev.Split == nil {
		t.Fatal("expected dev.Split to not be nil")
	}
	if dev.Split.Direction != "h" {
		t.Errorf("expected dev.Split.Direction to be 'h', got %q", dev.Split.Direction)
	}

	simple := cfg.NamedLayouts["simple"]
	if simple == nil {
		t.Fatal("expected 'simple' layout to exist")
	}
	if !simple.Clock {
		t.Error("expected simple.Clock to be true")
	}
}

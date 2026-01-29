package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNameFix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "foo"},
		{"foo.bar", "foo"},
		{"foo.bar.baz", "foo"},
		{".hidden", "hidden"}, // .hidden splits to ["", "hidden"], first non-empty is "hidden"
		{"", ""},
		{"noextension", "noextension"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NameFix(tt.input)
			if result != tt.expected {
				t.Errorf("NameFix(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDirFix(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/foo", filepath.Join(home, "foo")},
		{"~/foo/bar", filepath.Join(home, "foo/bar")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dirFix(tt.input)
			if result != tt.expected {
				t.Errorf("dirFix(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseConfig_Basic(t *testing.T) {
	input := TmuxConfigItemInput{
		Root: "/tmp/myproject",
		Name: "myproject",
		Windows: []TmuxWindowInput{
			{IsString: true, String: "./src"},
			{IsString: true, String: "./lib"},
		},
	}

	result := ParseConfig("myproject", input)

	if result.Name != "myproject" {
		t.Errorf("expected Name to be 'myproject', got %q", result.Name)
	}
	if result.Root != "/tmp/myproject" {
		t.Errorf("expected Root to be '/tmp/myproject', got %q", result.Root)
	}
	if len(result.Windows) != 2 {
		t.Errorf("expected 2 windows, got %d", len(result.Windows))
	}
	if result.Windows[0].Cwd != "/tmp/myproject/src" {
		t.Errorf("expected first window Cwd to be '/tmp/myproject/src', got %q", result.Windows[0].Cwd)
	}
	if result.Windows[1].Cwd != "/tmp/myproject/lib" {
		t.Errorf("expected second window Cwd to be '/tmp/myproject/lib', got %q", result.Windows[1].Cwd)
	}
}

func TestParseConfig_NoWindows(t *testing.T) {
	input := TmuxConfigItemInput{
		Root: "/tmp/myproject",
		Name: "myproject",
	}

	result := ParseConfig("myproject", input)

	// Should have 1 default window
	if len(result.Windows) != 1 {
		t.Errorf("expected 1 default window, got %d", len(result.Windows))
	}
	if result.Windows[0].Name != "myproject" {
		t.Errorf("expected default window name to be 'myproject', got %q", result.Windows[0].Name)
	}
}

func TestParseConfig_BlankWindow(t *testing.T) {
	input := TmuxConfigItemInput{
		Root:        "/tmp/myproject",
		Name:        "myproject",
		BlankWindow: true,
		Windows: []TmuxWindowInput{
			{IsString: true, String: "./src"},
		},
	}

	result := ParseConfig("myproject", input)

	// Should have 2 windows (blank + src)
	if len(result.Windows) != 2 {
		t.Errorf("expected 2 windows (blank + src), got %d", len(result.Windows))
	}
}

func TestParseConfig_NameFromKey(t *testing.T) {
	input := TmuxConfigItemInput{
		Root: "/tmp/myproject",
	}

	result := ParseConfig("fromkey", input)

	if result.Name != "fromkey" {
		t.Errorf("expected Name to be 'fromkey', got %q", result.Name)
	}
}

func TestParseConfig_NameFromRoot(t *testing.T) {
	input := TmuxConfigItemInput{
		Root: "/tmp/myproject",
	}

	result := ParseConfig("", input)

	if result.Name != "myproject" {
		t.Errorf("expected Name to be 'myproject', got %q", result.Name)
	}
}

func TestParseConfig_WindowStruct(t *testing.T) {
	input := TmuxConfigItemInput{
		Root: "/tmp/myproject",
		Name: "myproject",
		Windows: []TmuxWindowInput{
			{
				Window: &TmuxWindow{
					Name: "custom",
					Cwd:  "./custom",
				},
			},
		},
	}

	result := ParseConfig("myproject", input)

	if len(result.Windows) != 1 {
		t.Fatalf("expected 1 window, got %d", len(result.Windows))
	}
	if result.Windows[0].Name != "custom" {
		t.Errorf("expected window name to be 'custom', got %q", result.Windows[0].Name)
	}
	if result.Windows[0].Cwd != "/tmp/myproject/custom" {
		t.Errorf("expected window Cwd to be '/tmp/myproject/custom', got %q", result.Windows[0].Cwd)
	}
}

func TestParseConfig_WithLayout(t *testing.T) {
	input := TmuxConfigItemInput{
		Root: "/tmp/myproject",
		Name: "myproject",
		Windows: []TmuxWindowInput{
			{
				Window: &TmuxWindow{
					Name: "dev",
					Cwd:  "./src",
					Layout: &TmuxLayoutInput{
						PaneLayout: &TmuxPaneLayout{
							Cwd: ".",
							Cmd: "npm start",
							Split: &TmuxSplitLayout{
								Direction: "v",
								Child: &TmuxPaneLayout{
									Cwd: ".",
									Cmd: "npm test",
								},
							},
						},
					},
				},
			},
		},
	}

	result := ParseConfig("myproject", input)

	if len(result.Windows) != 1 {
		t.Fatalf("expected 1 window, got %d", len(result.Windows))
	}

	layout := result.Windows[0].Layout
	if layout.Cmd != "npm start" {
		t.Errorf("expected Cmd to be 'npm start', got %q", layout.Cmd)
	}
	if layout.Split == nil {
		t.Fatal("expected Split to not be nil")
	}
	if layout.Split.Direction != "v" {
		t.Errorf("expected Split.Direction to be 'v', got %q", layout.Split.Direction)
	}
	if layout.Split.Child == nil {
		t.Fatal("expected Split.Child to not be nil")
	}
	if layout.Split.Child.Cmd != "npm test" {
		t.Errorf("expected child Cmd to be 'npm test', got %q", layout.Split.Child.Cmd)
	}
}

func TestParseConfig_ArrayLayout(t *testing.T) {
	input := TmuxConfigItemInput{
		Root: "/tmp/myproject",
		Name: "myproject",
		Windows: []TmuxWindowInput{
			{
				Window: &TmuxWindow{
					Name: "dev",
					Cwd:  ".",
					Layout: &TmuxLayoutInput{
						IsArray: true,
						Array:   []string{"./src", "./lib"},
					},
				},
			},
		},
	}

	result := ParseConfig("myproject", input)

	if len(result.Windows) != 1 {
		t.Fatalf("expected 1 window, got %d", len(result.Windows))
	}

	layout := result.Windows[0].Layout
	if layout.Split == nil {
		t.Fatal("expected Split to not be nil for array layout")
	}
}

func TestParseConfig_TildeExpansion(t *testing.T) {
	home, _ := os.UserHomeDir()

	input := TmuxConfigItemInput{
		Root: "~/myproject",
		Name: "myproject",
	}

	result := ParseConfig("myproject", input)

	if !strings.HasPrefix(result.Root, home) {
		t.Errorf("expected Root to start with home dir %q, got %q", home, result.Root)
	}
}

func TestResolvePath(t *testing.T) {
	tests := []struct {
		root     string
		path     string
		expected string
	}{
		{"/tmp", "src", "/tmp/src"},
		{"/tmp", "./src", "/tmp/src"},
		{"/tmp", "/absolute", "/absolute"},
		{"/tmp", "", "/tmp"},
		{"/tmp", ".", "/tmp"},
	}

	for _, tt := range tests {
		t.Run(tt.root+"_"+tt.path, func(t *testing.T) {
			result := resolvePath(tt.root, tt.path)
			if result != tt.expected {
				t.Errorf("resolvePath(%q, %q) = %q, expected %q", tt.root, tt.path, result, tt.expected)
			}
		})
	}
}

func TestParseConfig_NamedLayout(t *testing.T) {
	// Save and restore original state
	originalLayouts := ConfiguredNamedLayouts
	defer func() { ConfiguredNamedLayouts = originalLayouts }()

	// Configure named layouts
	ConfiguredNamedLayouts = map[string]*TmuxPaneLayout{
		"dev": {
			Cwd: ".",
			Cmd: "npm run dev",
			Split: &TmuxSplitLayout{
				Direction: "h",
				Child: &TmuxPaneLayout{
					Cwd: ".",
					Cmd: "npm run test:watch",
				},
			},
		},
		"simple": {
			Cwd:   ".",
			Clock: true,
		},
	}

	input := TmuxConfigItemInput{
		Root: "/tmp/myproject",
		Name: "myproject",
		Windows: []TmuxWindowInput{
			{
				Window: &TmuxWindow{
					Name: "main",
					Cwd:  "./src",
					Layout: &TmuxLayoutInput{
						IsString: true,
						String:   "dev", // Reference named layout
					},
				},
			},
			{
				Window: &TmuxWindow{
					Name: "logs",
					Cwd:  "./logs",
					Layout: &TmuxLayoutInput{
						IsString: true,
						String:   "simple", // Reference named layout
					},
				},
			},
		},
	}

	result := ParseConfig("myproject", input)

	if len(result.Windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(result.Windows))
	}

	// Check first window uses "dev" layout
	mainLayout := result.Windows[0].Layout
	if mainLayout.Cmd != "npm run dev" {
		t.Errorf("expected first window Cmd to be 'npm run dev', got %q", mainLayout.Cmd)
	}
	if mainLayout.Split == nil {
		t.Fatal("expected first window Split to not be nil")
	}
	if mainLayout.Split.Direction != "h" {
		t.Errorf("expected first window Split.Direction to be 'h', got %q", mainLayout.Split.Direction)
	}
	if mainLayout.Split.Child == nil {
		t.Fatal("expected first window Split.Child to not be nil")
	}
	if mainLayout.Split.Child.Cmd != "npm run test:watch" {
		t.Errorf("expected first window child Cmd to be 'npm run test:watch', got %q", mainLayout.Split.Child.Cmd)
	}

	// Check second window uses "simple" layout
	logsLayout := result.Windows[1].Layout
	if !logsLayout.Clock {
		t.Error("expected second window Clock to be true")
	}
}

func TestParseConfig_NamedLayoutNotFound(t *testing.T) {
	// Save and restore original state
	originalLayouts := ConfiguredNamedLayouts
	defer func() { ConfiguredNamedLayouts = originalLayouts }()

	// No named layouts configured
	ConfiguredNamedLayouts = nil

	input := TmuxConfigItemInput{
		Root: "/tmp/myproject",
		Name: "myproject",
		Windows: []TmuxWindowInput{
			{
				Window: &TmuxWindow{
					Name: "main",
					Cwd:  "./src",
					Layout: &TmuxLayoutInput{
						IsString: true,
						String:   "nonexistent", // Not a named layout, should be treated as path
					},
				},
			},
		},
	}

	result := ParseConfig("myproject", input)

	if len(result.Windows) != 1 {
		t.Fatalf("expected 1 window, got %d", len(result.Windows))
	}

	// Should be treated as a directory path
	layout := result.Windows[0].Layout
	if layout.Cwd != "/tmp/myproject/src/nonexistent" {
		t.Errorf("expected layout Cwd to be '/tmp/myproject/src/nonexistent', got %q", layout.Cwd)
	}
}

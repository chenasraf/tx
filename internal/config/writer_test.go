package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirFixForWrite(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{filepath.Join(home, "projects"), "~/projects"},
		{filepath.Join(home, "Dev", "myproject"), "~/Dev/myproject"},
		{"/tmp/other", "/tmp/other"},
		{"/absolute/path", "/absolute/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := dirFixForWrite(tt.input)
			if result != tt.expected {
				t.Errorf("dirFixForWrite(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDirFixForWriteRelative(t *testing.T) {
	tests := []struct {
		dir      string
		root     string
		expected string
	}{
		{"/tmp/project/src", "/tmp/project", "./src"},
		{"/tmp/project", "/tmp/project", "./"},
		{"/other/path", "/tmp/project", "/other/path"},
	}

	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			result := dirFixForWriteRelative(tt.dir, tt.root)
			if result != tt.expected {
				t.Errorf("dirFixForWriteRelative(%q, %q) = %q, expected %q", tt.dir, tt.root, result, tt.expected)
			}
		})
	}
}

func TestAddSimpleConfigToFile_DryRun(t *testing.T) {
	// Create a temporary directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	initialContent := `
existing:
  root: /tmp/existing
`
	err := os.WriteFile(configPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Set XDG_CONFIG_HOME to temp directory so config is found
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	config := ParsedTmuxConfigItem{
		Name: "newproject",
		Root: "/tmp/newproject",
		Windows: []ParsedTmuxWindow{
			{Name: "main", Cwd: "/tmp/newproject"},
		},
	}

	// Dry run should not modify file
	err = AddSimpleConfigToFile(config, false, true)
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}

	// Verify file wasn't modified
	content, _ := os.ReadFile(configPath)
	if strings.Contains(string(content), "newproject") {
		t.Error("dry run should not have modified file")
	}
}

func TestAddSimpleConfigToFile(t *testing.T) {
	// Create a temporary directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	initialContent := `existing:
  root: /tmp/existing
`
	err := os.WriteFile(configPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Set XDG_CONFIG_HOME to temp directory so config is found
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	config := ParsedTmuxConfigItem{
		Name: "newproject",
		Root: "/tmp/newproject",
		Windows: []ParsedTmuxWindow{
			{Name: "main", Cwd: "/tmp/newproject"},
			{Name: "src", Cwd: "/tmp/newproject/src"},
		},
	}

	err = AddSimpleConfigToFile(config, false, false)
	if err != nil {
		t.Fatalf("failed to add config: %v", err)
	}

	// Verify file was modified
	content, _ := os.ReadFile(configPath)
	contentStr := string(content)

	if !strings.Contains(contentStr, "newproject:") {
		t.Error("expected file to contain 'newproject:'")
	}
	if !strings.Contains(contentStr, "root: /tmp/newproject") {
		t.Error("expected file to contain 'root: /tmp/newproject'")
	}
	if !strings.Contains(contentStr, "windows:") {
		t.Error("expected file to contain 'windows:'")
	}
}

func TestRemoveConfigFromFile(t *testing.T) {
	// Create a temporary directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	initialContent := `first:
  root: /tmp/first
  windows:
    - ./src

second:
  root: /tmp/second

third:
  root: /tmp/third
`
	err := os.WriteFile(configPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Set XDG_CONFIG_HOME to temp directory so config is found
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	err = RemoveConfigFromFile("second", false, false)
	if err != nil {
		t.Fatalf("failed to remove config: %v", err)
	}

	// Verify file was modified correctly
	content, _ := os.ReadFile(configPath)
	contentStr := string(content)

	if !strings.Contains(contentStr, "first:") {
		t.Error("expected file to still contain 'first:'")
	}
	if strings.Contains(contentStr, "second:") {
		t.Error("expected 'second:' to be removed")
	}
	if !strings.Contains(contentStr, "third:") {
		t.Error("expected file to still contain 'third:'")
	}
}

func TestRemoveConfigFromFile_NotFound(t *testing.T) {
	// Create a temporary directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	initialContent := `existing:
  root: /tmp/existing
`
	err := os.WriteFile(configPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Set XDG_CONFIG_HOME to temp directory so config is found
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	err = RemoveConfigFromFile("nonexistent", false, false)
	if err == nil {
		t.Error("expected error when removing nonexistent config")
	}
}

func TestRemoveConfigFromFile_DryRun(t *testing.T) {
	// Create a temporary directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	initialContent := `toremove:
  root: /tmp/toremove
`
	err := os.WriteFile(configPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Set XDG_CONFIG_HOME to temp directory so config is found
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	err = RemoveConfigFromFile("toremove", false, true)
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}

	// Verify file wasn't modified
	content, _ := os.ReadFile(configPath)
	if !strings.Contains(string(content), "toremove:") {
		t.Error("dry run should not have modified file")
	}
}

func TestRemoveConfigFromFile_LastItem(t *testing.T) {
	// Create a temporary directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	initialContent := `first:
  root: /tmp/first

last:
  root: /tmp/last
  windows:
    - ./src
    - ./lib
`
	err := os.WriteFile(configPath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Set XDG_CONFIG_HOME to temp directory so config is found
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	err = RemoveConfigFromFile("last", false, false)
	if err != nil {
		t.Fatalf("failed to remove last config: %v", err)
	}

	// Verify file was modified correctly
	content, _ := os.ReadFile(configPath)
	contentStr := string(content)

	if !strings.Contains(contentStr, "first:") {
		t.Error("expected file to still contain 'first:'")
	}
	if strings.Contains(contentStr, "last:") {
		t.Error("expected 'last:' to be removed")
	}
}

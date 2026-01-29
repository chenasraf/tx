package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunMain_NoConfig(t *testing.T) {
	// Create a temp directory with no config
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Set dry mode to prevent actual tmux operations
	dry = true
	defer func() { dry = false }()

	// Should fail because no config exists
	err := runMain(nil, []string{"nonexistent"})
	if err == nil {
		t.Error("expected error when no config exists")
	}
}

func TestRunMain_WithConfig(t *testing.T) {
	// Create a temp directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	content := `
testproject:
  root: /tmp/test
  windows:
    - ./src
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Set dry mode
	dry = true
	defer func() { dry = false }()

	// Should succeed with valid config key
	err = runMain(nil, []string{"testproject"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunMain_InvalidKey(t *testing.T) {
	// Create a temp directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	content := `
existingproject:
  root: /tmp/test
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	dry = true
	defer func() { dry = false }()

	// Should fail with invalid key
	err = runMain(nil, []string{"nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent key")
	}

	userErr, ok := err.(*UserError)
	if !ok {
		t.Errorf("expected UserError, got %T", err)
	} else if userErr.Message != "tmux config item 'nonexistent' not found" {
		t.Errorf("unexpected error message: %q", userErr.Message)
	}
}

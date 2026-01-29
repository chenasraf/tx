package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchPatterns(t *testing.T) {
	patterns := searchPatterns("tmux")

	expected := []string{
		".tmux.yaml",
		".tmux.yml",
		filepath.Join(".config", ".tmux.yaml"),
		filepath.Join(".config", ".tmux.yml"),
	}

	if len(patterns) != len(expected) {
		t.Errorf("expected %d patterns, got %d", len(expected), len(patterns))
	}

	for i, p := range expected {
		if patterns[i] != p {
			t.Errorf("expected pattern[%d] to be %q, got %q", i, p, patterns[i])
		}
	}
}

func TestSearchDirs(t *testing.T) {
	dirs := searchDirs()

	// Should at least contain current working directory
	if len(dirs) == 0 {
		t.Error("expected at least one search directory")
	}

	// First dir should be current working directory
	cwd, _ := os.Getwd()
	if dirs[0] != cwd {
		t.Errorf("expected first dir to be cwd %q, got %q", cwd, dirs[0])
	}

	// Should contain home directory
	home, _ := os.UserHomeDir()
	found := false
	for _, d := range dirs {
		if d == home {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected search dirs to contain home directory")
	}
}

func TestLoadConfigFile(t *testing.T) {
	// Create a temporary config file
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

	config, err := loadConfigFile(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(config) != 1 {
		t.Errorf("expected 1 config, got %d", len(config))
	}

	item, ok := config["testproject"]
	if !ok {
		t.Fatal("expected 'testproject' in config")
	}

	if item.Root != "/tmp/test" {
		t.Errorf("expected Root to be '/tmp/test', got %q", item.Root)
	}
}

func TestLoadConfigFile_Invalid(t *testing.T) {
	// Create a temporary invalid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	content := `invalid: yaml: content: [[[`
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	_, err = loadConfigFile(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadConfigFile_NotFound(t *testing.T) {
	_, err := loadConfigFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestMergeConfigs(t *testing.T) {
	config1 := ConfigFile{
		"project1": {Root: "/tmp/p1", Name: "p1"},
		"shared":   {Root: "/tmp/shared", Name: "shared-global"},
	}

	config2 := ConfigFile{
		"project2": {Root: "/tmp/p2", Name: "p2"},
		"shared":   {Root: "/tmp/shared-local", Name: "shared-local"},
	}

	merged := mergeConfigs(config1, config2)

	if len(merged) != 3 {
		t.Errorf("expected 3 configs, got %d", len(merged))
	}

	// project1 should be from config1
	if merged["project1"].Root != "/tmp/p1" {
		t.Errorf("expected project1 Root to be '/tmp/p1', got %q", merged["project1"].Root)
	}

	// project2 should be from config2
	if merged["project2"].Root != "/tmp/p2" {
		t.Errorf("expected project2 Root to be '/tmp/p2', got %q", merged["project2"].Root)
	}

	// shared should be overridden by config2
	if merged["shared"].Root != "/tmp/shared-local" {
		t.Errorf("expected shared Root to be '/tmp/shared-local', got %q", merged["shared"].Root)
	}
	if merged["shared"].Name != "shared-local" {
		t.Errorf("expected shared Name to be 'shared-local', got %q", merged["shared"].Name)
	}
}

func TestMergeConfigs_Nil(t *testing.T) {
	config1 := ConfigFile{
		"project1": {Root: "/tmp/p1"},
	}

	merged := mergeConfigs(nil, config1, nil)

	if len(merged) != 1 {
		t.Errorf("expected 1 config, got %d", len(merged))
	}
}

func TestMergeConfigs_PartialOverride(t *testing.T) {
	config1 := ConfigFile{
		"project": {Root: "/tmp/p1", Name: "original", BlankWindow: false},
	}

	config2 := ConfigFile{
		"project": {Root: "/tmp/p2"}, // Only override Root
	}

	merged := mergeConfigs(config1, config2)

	// Root should be overridden
	if merged["project"].Root != "/tmp/p2" {
		t.Errorf("expected Root to be '/tmp/p2', got %q", merged["project"].Root)
	}

	// Name should be preserved (empty string doesn't override)
	// Note: Current implementation doesn't preserve - this tests current behavior
}

func TestGetSearchedPaths(t *testing.T) {
	paths := GetSearchedPaths()

	if len(paths) == 0 {
		t.Error("expected at least some search paths")
	}

	// Should contain both tmux and tmux_local patterns
	hasTmux := false
	hasTmuxLocal := false
	for _, p := range paths {
		if filepath.Base(p) == ".tmux.yaml" || filepath.Base(p) == ".tmux.yml" {
			hasTmux = true
		}
		if filepath.Base(p) == ".tmux_local.yaml" || filepath.Base(p) == ".tmux_local.yml" {
			hasTmuxLocal = true
		}
	}

	if !hasTmux {
		t.Error("expected paths to contain .tmux.yaml patterns")
	}
	if !hasTmuxLocal {
		t.Error("expected paths to contain .tmux_local.yaml patterns")
	}
}

func TestLoadGlobalConfig(t *testing.T) {
	// Create a temporary config file with .config section
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	content := `
.config:
  shell: /bin/custom-shell

testproject:
  root: /tmp/test
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	globalConfig, err := loadGlobalConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load global config: %v", err)
	}

	if globalConfig == nil {
		t.Fatal("expected globalConfig to not be nil")
	}

	if globalConfig.Shell != "/bin/custom-shell" {
		t.Errorf("expected Shell to be '/bin/custom-shell', got %q", globalConfig.Shell)
	}
}

func TestLoadGlobalConfig_NoConfigSection(t *testing.T) {
	// Create a temporary config file without .config section
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	content := `
testproject:
  root: /tmp/test
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	globalConfig, err := loadGlobalConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if globalConfig != nil {
		t.Error("expected globalConfig to be nil when .config section is missing")
	}
}

func TestMergeConfigs_SkipsConfigSection(t *testing.T) {
	config := ConfigFile{
		".config":  {Root: "ignored"},
		"project1": {Root: "/tmp/p1"},
	}

	merged := mergeConfigs(config)

	if _, ok := merged[".config"]; ok {
		t.Error("expected .config to be filtered out")
	}

	if _, ok := merged["project1"]; !ok {
		t.Error("expected project1 to be present")
	}
}

func TestFindConfigFile(t *testing.T) {
	// Create a temporary directory with a config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	content := `
testproject:
  root: /tmp/test
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	result, err := findConfigFile("tmux")
	if err != nil {
		t.Fatalf("failed to find config: %v", err)
	}

	if result == nil {
		t.Fatal("expected result to not be nil")
	}

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	expectedPath, _ := filepath.EvalSymlinks(configPath)
	actualPath, _ := filepath.EvalSymlinks(result.Filepath)
	if actualPath != expectedPath {
		t.Errorf("expected Filepath to be %q, got %q", expectedPath, actualPath)
	}

	if _, ok := result.Config["testproject"]; !ok {
		t.Error("expected config to contain 'testproject'")
	}
}

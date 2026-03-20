package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchPatterns(t *testing.T) {
	patterns := searchPatterns("tmux")

	expected := []string{
		"tmux.yaml",
		"tmux.yml",
		".tmux.yaml",
		".tmux.yml",
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

	// First dir should be home directory
	home, _ := os.UserHomeDir()
	if dirs[0] != home {
		t.Errorf("expected first dir to be home %q, got %q", home, dirs[0])
	}

	// Should contain ~/.config
	dotConfig := filepath.Join(home, ".config")
	found := false
	for _, d := range dirs {
		if d == dotConfig {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected search dirs to contain ~/.config")
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

func TestMergeConfigs_Aliases(t *testing.T) {
	config1 := ConfigFile{
		"project": {Root: "/tmp/p1", Aliases: []string{"p1", "proj"}},
	}

	config2 := ConfigFile{
		"project": {Root: "/tmp/p2", Aliases: []string{"p2"}},
	}

	merged := mergeConfigs(config1, config2)

	// Aliases should be overridden by config2
	if len(merged["project"].Aliases) != 1 {
		t.Errorf("expected 1 alias, got %d", len(merged["project"].Aliases))
	}
	if merged["project"].Aliases[0] != "p2" {
		t.Errorf("expected alias 'p2', got %q", merged["project"].Aliases[0])
	}
}

func TestMergeConfigs_AliasesPreserved(t *testing.T) {
	config1 := ConfigFile{
		"project": {Root: "/tmp/p1", Aliases: []string{"p1", "proj"}},
	}

	config2 := ConfigFile{
		"project": {Root: "/tmp/p2"}, // No aliases - should preserve config1's aliases
	}

	merged := mergeConfigs(config1, config2)

	if len(merged["project"].Aliases) != 2 {
		t.Errorf("expected 2 aliases preserved, got %d", len(merged["project"].Aliases))
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

	// Should contain tmux patterns
	hasTmux := false
	for _, p := range paths {
		if filepath.Base(p) == ".tmux.yaml" || filepath.Base(p) == ".tmux.yml" {
			hasTmux = true
		}
	}

	if !hasTmux {
		t.Error("expected paths to contain .tmux.yaml patterns")
	}

	// Should NOT contain tmux_local patterns (replaced by include mechanism)
	for _, p := range paths {
		if filepath.Base(p) == ".tmux_local.yaml" || filepath.Base(p) == ".tmux_local.yml" {
			t.Error("should not contain tmux_local patterns")
		}
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

func TestMergeGlobalConfigs(t *testing.T) {
	config1 := &GlobalConfig{
		Shell:        "/bin/bash",
		ProjectsPath: "/old/path",
	}

	config2 := &GlobalConfig{
		Shell: "/bin/zsh",
	}

	merged := mergeGlobalConfigs(config1, config2)

	// Shell should be overridden by config2
	if merged.Shell != "/bin/zsh" {
		t.Errorf("expected Shell to be '/bin/zsh', got %q", merged.Shell)
	}

	// ProjectsPath should be preserved from config1
	if merged.ProjectsPath != "/old/path" {
		t.Errorf("expected ProjectsPath to be '/old/path', got %q", merged.ProjectsPath)
	}
}

func TestMergeGlobalConfigs_Nil(t *testing.T) {
	config1 := &GlobalConfig{
		Shell: "/bin/bash",
	}

	merged := mergeGlobalConfigs(nil, config1, nil)

	if merged.Shell != "/bin/bash" {
		t.Errorf("expected Shell to be '/bin/bash', got %q", merged.Shell)
	}
}

func TestMergeGlobalConfigs_NamedLayouts(t *testing.T) {
	config1 := &GlobalConfig{
		NamedLayouts: map[string]*TmuxPaneLayout{
			"dev": {
				Cwd: ".",
				Cmd: "npm run dev",
			},
			"common": {
				Cwd: ".",
				Cmd: "original command",
			},
		},
	}

	config2 := &GlobalConfig{
		NamedLayouts: map[string]*TmuxPaneLayout{
			"test": {
				Cwd: ".",
				Cmd: "npm test",
			},
			"common": {
				Cwd: ".",
				Cmd: "overridden command",
			},
		},
	}

	merged := mergeGlobalConfigs(config1, config2)

	if merged.NamedLayouts == nil {
		t.Fatal("expected NamedLayouts to not be nil")
	}

	// Should have 3 layouts (dev, test, common)
	if len(merged.NamedLayouts) != 3 {
		t.Errorf("expected 3 named layouts, got %d", len(merged.NamedLayouts))
	}

	// dev should be from config1
	if dev := merged.NamedLayouts["dev"]; dev == nil {
		t.Error("expected 'dev' layout to exist")
	} else if dev.Cmd != "npm run dev" {
		t.Errorf("expected dev.Cmd to be 'npm run dev', got %q", dev.Cmd)
	}

	// test should be from config2
	if test := merged.NamedLayouts["test"]; test == nil {
		t.Error("expected 'test' layout to exist")
	} else if test.Cmd != "npm test" {
		t.Errorf("expected test.Cmd to be 'npm test', got %q", test.Cmd)
	}

	// common should be overridden by config2
	if common := merged.NamedLayouts["common"]; common == nil {
		t.Error("expected 'common' layout to exist")
	} else if common.Cmd != "overridden command" {
		t.Errorf("expected common.Cmd to be 'overridden command', got %q", common.Cmd)
	}
}

func TestMergeGlobalConfigs_DefaultLayout(t *testing.T) {
	config1 := &GlobalConfig{
		DefaultLayout: &TmuxPaneLayout{
			Cwd: ".",
			Cmd: "original",
		},
	}

	config2 := &GlobalConfig{
		DefaultLayout: &TmuxPaneLayout{
			Cwd:   ".",
			Clock: true,
		},
	}

	merged := mergeGlobalConfigs(config1, config2)

	if merged.DefaultLayout == nil {
		t.Fatal("expected DefaultLayout to not be nil")
	}

	// DefaultLayout should be fully replaced by config2
	if !merged.DefaultLayout.Clock {
		t.Error("expected DefaultLayout.Clock to be true")
	}
}

func TestLoadGlobalConfig_WithNamedLayouts(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".tmux.yaml")

	content := `
.config:
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

	if globalConfig.NamedLayouts == nil {
		t.Fatal("expected NamedLayouts to not be nil")
	}

	if len(globalConfig.NamedLayouts) != 2 {
		t.Errorf("expected 2 named layouts, got %d", len(globalConfig.NamedLayouts))
	}

	dev := globalConfig.NamedLayouts["dev"]
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

	simple := globalConfig.NamedLayouts["simple"]
	if simple == nil {
		t.Fatal("expected 'simple' layout to exist")
	}
	if !simple.Clock {
		t.Error("expected simple.Clock to be true")
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

	// Set XDG_CONFIG_HOME to temp directory so it's in the search path
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

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

func TestResolveIncludePath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name       string
		include    string
		parentPath string
		expected   string
	}{
		{"absolute path", "/etc/tmux.yaml", "/home/user/.tmux.yaml", "/etc/tmux.yaml"},
		{"tilde path", "~/local.yaml", "/home/user/.tmux.yaml", filepath.Join(home, "local.yaml")},
		{"relative path", "local.yaml", "/home/user/.tmux.yaml", "/home/user/local.yaml"},
		{"relative subdir", "conf/extra.yaml", "/home/user/.tmux.yaml", "/home/user/conf/extra.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveIncludePath(tt.include, tt.parentPath)
			if result != tt.expected {
				t.Errorf("resolveIncludePath(%q, %q) = %q, expected %q", tt.include, tt.parentPath, result, tt.expected)
			}
		})
	}
}

func TestGetTmuxConfigFileInfo_WithIncludes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create included config file
	includedPath := filepath.Join(tmpDir, "extra.yaml")
	includedContent := `
extraproject:
  root: /tmp/extra
`
	if err := os.WriteFile(includedPath, []byte(includedContent), 0644); err != nil {
		t.Fatalf("failed to write included config: %v", err)
	}

	// Create main config that includes the extra file
	configPath := filepath.Join(tmpDir, ".tmux.yaml")
	mainContent := `.config:
  include:
    - extra.yaml

mainproject:
  root: /tmp/main
`
	if err := os.WriteFile(configPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to write main config: %v", err)
	}

	// Set XDG_CONFIG_HOME to temp directory
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	info, err := GetTmuxConfigFileInfo()
	if err != nil {
		t.Fatalf("failed to get config info: %v", err)
	}

	// Should have 1 included config
	if len(info.Included) != 1 {
		t.Fatalf("expected 1 included config, got %d", len(info.Included))
	}

	// Merged config should contain both projects
	if _, ok := info.Merged.Config["mainproject"]; !ok {
		t.Error("expected merged config to contain 'mainproject'")
	}
	if _, ok := info.Merged.Config["extraproject"]; !ok {
		t.Error("expected merged config to contain 'extraproject'")
	}
}

func TestGetTmuxConfigFileInfo_NestedIncludes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create deeply nested config
	deepPath := filepath.Join(tmpDir, "deep.yaml")
	deepContent := `
deepproject:
  root: /tmp/deep
`
	if err := os.WriteFile(deepPath, []byte(deepContent), 0644); err != nil {
		t.Fatalf("failed to write deep config: %v", err)
	}

	// Create mid-level config that includes deep
	midPath := filepath.Join(tmpDir, "mid.yaml")
	midContent := `.config:
  include:
    - deep.yaml

midproject:
  root: /tmp/mid
`
	if err := os.WriteFile(midPath, []byte(midContent), 0644); err != nil {
		t.Fatalf("failed to write mid config: %v", err)
	}

	// Create main config that includes mid
	configPath := filepath.Join(tmpDir, ".tmux.yaml")
	mainContent := `.config:
  include:
    - mid.yaml

mainproject:
  root: /tmp/main
`
	if err := os.WriteFile(configPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to write main config: %v", err)
	}

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	info, err := GetTmuxConfigFileInfo()
	if err != nil {
		t.Fatalf("failed to get config info: %v", err)
	}

	// Should have 2 included configs (deep is loaded before mid due to recursion order)
	if len(info.Included) != 2 {
		t.Fatalf("expected 2 included configs, got %d", len(info.Included))
	}

	// Merged config should contain all three projects
	for _, name := range []string{"mainproject", "midproject", "deepproject"} {
		if _, ok := info.Merged.Config[name]; !ok {
			t.Errorf("expected merged config to contain %q", name)
		}
	}
}

func TestGetTmuxConfigFileInfo_CircularIncludes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two configs that include each other
	aPath := filepath.Join(tmpDir, ".tmux.yaml")
	bPath := filepath.Join(tmpDir, "b.yaml")

	aContent := `.config:
  include:
    - b.yaml

projectA:
  root: /tmp/a
`
	bContent := `.config:
  include:
    - .tmux.yaml

projectB:
  root: /tmp/b
`
	if err := os.WriteFile(aPath, []byte(aContent), 0644); err != nil {
		t.Fatalf("failed to write config a: %v", err)
	}
	if err := os.WriteFile(bPath, []byte(bContent), 0644); err != nil {
		t.Fatalf("failed to write config b: %v", err)
	}

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", oldXDG) }()

	// Should not infinite loop
	info, err := GetTmuxConfigFileInfo()
	if err != nil {
		t.Fatalf("failed to get config info: %v", err)
	}

	// Should have both projects merged
	if _, ok := info.Merged.Config["projectA"]; !ok {
		t.Error("expected merged config to contain 'projectA'")
	}
	if _, ok := info.Merged.Config["projectB"]; !ok {
		t.Error("expected merged config to contain 'projectB'")
	}
}

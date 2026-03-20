package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigKey is the reserved key for global configuration
const ConfigKey = ".config"

// ConfigResult holds a loaded config and its file path
type ConfigResult struct {
	Config   ConfigFile
	Filepath string
}

// ConfigInfo holds global, included, and merged configurations
type ConfigInfo struct {
	Global       *ConfigResult
	Included     []*ConfigResult
	Merged       *ConfigResult
	GlobalConfig *GlobalConfig
}

// ErrNoConfigFound is returned when no config file is found
var ErrNoConfigFound = errors.New("no config file found")

// searchDirs returns the directories to search for config files
func searchDirs() []string {
	var dirs []string

	// Add home directory
	home, err := os.UserHomeDir()
	if err == nil {
		dirs = append(dirs, home)
	}

	// Add XDG config directory
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig != "" {
		dirs = append(dirs, xdgConfig)
	}

	// Add ~/.config as fallback (only if different from XDG_CONFIG_HOME)
	if home != "" {
		dotConfig := filepath.Join(home, ".config")
		if dotConfig != xdgConfig {
			dirs = append(dirs, dotConfig)
		}
	}

	// Add APPDATA if set (Windows)
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		dirs = append(dirs, appdata)
	}

	return dirs
}

// searchPatterns returns the file patterns to search for a given name
func searchPatterns(name string) []string {
	return []string{
		name + ".yaml",
		name + ".yml",
		"." + name + ".yaml",
		"." + name + ".yml",
	}
}

// findConfigFile searches for a config file with the given name
func findConfigFile(name string) (*ConfigResult, error) {
	patterns := searchPatterns(name)
	dirs := searchDirs()

	for _, dir := range dirs {
		for _, pattern := range patterns {
			path := filepath.Join(dir, pattern)
			if _, err := os.Stat(path); err == nil {
				config, err := loadConfigFile(path)
				if err != nil {
					continue
				}
				return &ConfigResult{
					Config:   config,
					Filepath: path,
				}, nil
			}
		}
	}

	return nil, ErrNoConfigFound
}

// loadConfigFile loads a YAML config file from the given path
func loadConfigFile(path string) (ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config, nil
}

// loadGlobalConfig extracts GlobalConfig from a raw YAML file
func loadGlobalConfig(path string) (*GlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse as map to get .config section
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	configSection, ok := raw[ConfigKey]
	if !ok {
		return nil, nil
	}

	// Re-marshal and unmarshal to get typed GlobalConfig
	configData, err := yaml.Marshal(configSection)
	if err != nil {
		return nil, err
	}

	var globalConfig GlobalConfig
	if err := yaml.Unmarshal(configData, &globalConfig); err != nil {
		return nil, err
	}

	return &globalConfig, nil
}

// mergeConfigs merges multiple config files, with later configs taking precedence
func mergeConfigs(configs ...ConfigFile) ConfigFile {
	out := make(ConfigFile)
	for _, config := range configs {
		if config == nil {
			continue
		}
		for key, value := range config {
			// Skip .config section
			if key == ConfigKey {
				continue
			}
			if existing, ok := out[key]; ok {
				// Merge: later values override earlier ones
				merged := existing
				if value.Root != "" {
					merged.Root = value.Root
				}
				if value.Name != "" {
					merged.Name = value.Name
				}
				if value.BlankWindow {
					merged.BlankWindow = value.BlankWindow
				}
				if len(value.Windows) > 0 {
					merged.Windows = value.Windows
				}
				if len(value.Aliases) > 0 {
					merged.Aliases = value.Aliases
				}
				out[key] = merged
			} else {
				out[key] = value
			}
		}
	}
	return out
}

// mergeGlobalConfigs merges GlobalConfig, with later values taking precedence
func mergeGlobalConfigs(configs ...*GlobalConfig) *GlobalConfig {
	result := &GlobalConfig{}
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		if cfg.Shell != "" {
			result.Shell = cfg.Shell
		}
		if cfg.ProjectsPath != "" {
			result.ProjectsPath = cfg.ProjectsPath
		}
		if cfg.DefaultLayout != nil {
			result.DefaultLayout = cfg.DefaultLayout
		}
		if cfg.NamedLayouts != nil {
			if result.NamedLayouts == nil {
				result.NamedLayouts = make(map[string]*TmuxPaneLayout)
			}
			for name, layout := range cfg.NamedLayouts {
				result.NamedLayouts[name] = layout
			}
		}
	}
	return result
}

// resolveIncludePath resolves an include path relative to the config file that contains it.
// Absolute paths are returned as-is. Paths starting with ~ are expanded. Relative paths
// are resolved against the directory of the parent config file.
func resolveIncludePath(includePath, parentConfigPath string) string {
	// Expand ~
	includePath = DirFix(includePath)

	// If absolute, return as-is
	if filepath.IsAbs(includePath) {
		return includePath
	}

	// Resolve relative to parent config file's directory
	parentDir := filepath.Dir(parentConfigPath)
	return filepath.Join(parentDir, includePath)
}

// loadIncludedConfigs loads all configs referenced by the include list in .config.
// It returns the loaded config results and the merged global config from all includes.
// Already-visited paths are tracked to prevent circular includes.
func loadIncludedConfigs(parentPath string, includes []string, visited map[string]bool) ([]*ConfigResult, []*GlobalConfig) {
	var results []*ConfigResult
	var globalConfigs []*GlobalConfig

	for _, inc := range includes {
		resolvedPath := resolveIncludePath(inc, parentPath)

		// Resolve symlinks for dedup
		realPath, err := filepath.EvalSymlinks(resolvedPath)
		if err != nil {
			realPath = resolvedPath
		}

		if visited[realPath] {
			continue
		}
		visited[realPath] = true

		cfg, err := loadConfigFile(resolvedPath)
		if err != nil {
			continue
		}

		result := &ConfigResult{
			Config:   cfg,
			Filepath: resolvedPath,
		}

		gc, _ := loadGlobalConfig(resolvedPath)

		// Recursively load nested includes
		if gc != nil && len(gc.Include) > 0 {
			nestedResults, nestedGCs := loadIncludedConfigs(resolvedPath, gc.Include, visited)
			results = append(results, nestedResults...)
			globalConfigs = append(globalConfigs, nestedGCs...)
		}

		results = append(results, result)
		globalConfigs = append(globalConfigs, gc)
	}

	return results, globalConfigs
}

// GetTmuxConfigFileInfo returns the global, included, and merged configurations
func GetTmuxConfigFileInfo() (*ConfigInfo, error) {
	info := &ConfigInfo{}

	// Search for global config
	if result, err := findConfigFile("tmux"); err == nil {
		info.Global = result
	}

	if info.Global == nil {
		return nil, ErrNoConfigFound
	}

	// Load global config section
	baseGlobalConfig, _ := loadGlobalConfig(info.Global.Filepath)

	// Process includes from .config.include
	var allGlobalConfigs []*GlobalConfig
	allGlobalConfigs = append(allGlobalConfigs, baseGlobalConfig)

	configsToMerge := []ConfigFile{info.Global.Config}

	if baseGlobalConfig != nil && len(baseGlobalConfig.Include) > 0 {
		visited := map[string]bool{}
		// Mark the base config as visited
		if realPath, err := filepath.EvalSymlinks(info.Global.Filepath); err == nil {
			visited[realPath] = true
		} else {
			visited[info.Global.Filepath] = true
		}

		includedResults, includedGCs := loadIncludedConfigs(info.Global.Filepath, baseGlobalConfig.Include, visited)
		info.Included = includedResults

		for _, r := range includedResults {
			configsToMerge = append(configsToMerge, r.Config)
		}
		allGlobalConfigs = append(allGlobalConfigs, includedGCs...)
	}

	// Merge global configs
	info.GlobalConfig = mergeGlobalConfigs(allGlobalConfigs...)

	// Merge session configs
	merged := mergeConfigs(configsToMerge...)
	info.Merged = &ConfigResult{
		Config:   merged,
		Filepath: "merged",
	}

	return info, nil
}

// GetTmuxConfig returns the merged configuration (sessions only, no .config)
func GetTmuxConfig() (ConfigFile, error) {
	info, err := GetTmuxConfigFileInfo()
	if err != nil {
		return nil, err
	}
	return info.Merged.Config, nil
}

// GetGlobalConfig returns the merged global configuration
func GetGlobalConfig() (*GlobalConfig, error) {
	info, err := GetTmuxConfigFileInfo()
	if err != nil {
		return nil, err
	}
	return info.GlobalConfig, nil
}

// FindLegacyLocalConfig searches for a tmux_local config file using the v1.x
// search patterns. Returns the file path if found, or empty string if not.
func FindLegacyLocalConfig() string {
	patterns := searchPatterns("tmux_local")
	dirs := searchDirs()

	for _, dir := range dirs {
		for _, pattern := range patterns {
			path := filepath.Join(dir, pattern)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}

// GetSearchedPaths returns the paths that would be searched for config files
func GetSearchedPaths() []string {
	var paths []string
	dirs := searchDirs()
	patterns := searchPatterns("tmux")
	for _, dir := range dirs {
		for _, pattern := range patterns {
			paths = append(paths, filepath.Join(dir, pattern))
		}
	}
	return paths
}

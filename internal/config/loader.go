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

// ConfigInfo holds global, local, and merged configurations
type ConfigInfo struct {
	Global       *ConfigResult
	Local        *ConfigResult
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

// GetTmuxConfigFileInfo returns the global, local, and merged configurations
func GetTmuxConfigFileInfo() (*ConfigInfo, error) {
	info := &ConfigInfo{}

	var globalGlobalConfig, localGlobalConfig *GlobalConfig

	// Search for global config
	if result, err := findConfigFile("tmux"); err == nil {
		info.Global = result
		// Load global config section
		globalGlobalConfig, _ = loadGlobalConfig(result.Filepath)
	}

	// Search for local config
	if result, err := findConfigFile("tmux_local"); err == nil {
		info.Local = result
		// Load global config section from local
		localGlobalConfig, _ = loadGlobalConfig(result.Filepath)
	}

	if info.Global == nil && info.Local == nil {
		return nil, ErrNoConfigFound
	}

	// Merge global configs
	info.GlobalConfig = mergeGlobalConfigs(globalGlobalConfig, localGlobalConfig)

	// Merge session configs
	var globalConfig, localConfig ConfigFile
	if info.Global != nil {
		globalConfig = info.Global.Config
	}
	if info.Local != nil {
		localConfig = info.Local.Config
	}

	merged := mergeConfigs(globalConfig, localConfig)
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

// GetSearchedPaths returns the paths that would be searched for config files
func GetSearchedPaths() []string {
	var paths []string
	dirs := searchDirs()
	for _, name := range []string{"tmux", "tmux_local"} {
		patterns := searchPatterns(name)
		for _, dir := range dirs {
			for _, pattern := range patterns {
				paths = append(paths, filepath.Join(dir, pattern))
			}
		}
	}
	return paths
}

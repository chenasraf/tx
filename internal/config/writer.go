package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrConfigNotFound is returned when the config file is not found
var ErrConfigNotFound = errors.New("tmux config file not found")

// ErrConfigItemExists is returned when trying to add an item that already exists
var ErrConfigItemExists = errors.New("tmux config item already exists")

// AddSimpleConfigToFile appends a simple config to the config file
func AddSimpleConfigToFile(config ParsedTmuxConfigItem, local bool, dryRun bool) error {
	files, err := GetTmuxConfigFileInfo()
	if err != nil {
		return err
	}

	var file *ConfigResult
	if local {
		file = files.Local
	} else {
		file = files.Global
	}

	if file == nil {
		return ErrConfigNotFound
	}

	// Check if config already exists
	allConfigs, err := GetTmuxConfig()
	if err != nil {
		return err
	}

	if _, exists := allConfigs[config.Name]; exists && !dryRun {
		return fmt.Errorf("%w: '%s'", ErrConfigItemExists, config.Name)
	}

	// Build YAML content
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(config.Name)
	sb.WriteString(":\n")
	sb.WriteString("  root: ")
	sb.WriteString(dirFixForWrite(config.Root))
	sb.WriteString("\n")
	sb.WriteString("  windows:\n")
	for _, w := range config.Windows {
		sb.WriteString("    - ")
		sb.WriteString(dirFixForWriteRelative(w.Cwd, config.Root))
		sb.WriteString("\n")
	}

	if dryRun {
		fmt.Println("Would have saved config to", file.Filepath)
		fmt.Println("Contents:")
		fmt.Println(sb.String())
		return nil
	}

	f, err := os.OpenFile(file.Filepath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	_, err = f.WriteString(sb.String())
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}

// RemoveConfigFromFile removes a config item from the config file
func RemoveConfigFromFile(key string, local bool, dryRun bool) error {
	files, err := GetTmuxConfigFileInfo()
	if err != nil {
		return err
	}

	var file *ConfigResult
	if local {
		file = files.Local
	} else {
		file = files.Global
	}

	if file == nil {
		return ErrConfigNotFound
	}

	// Read file contents
	data, err := os.ReadFile(file.Filepath)
	if err != nil {
		return err
	}

	contents := strings.Split(string(data), "\n")

	// Find the start index of the key
	startIndex := -1
	for i, line := range contents {
		if strings.HasPrefix(line, key+":") {
			startIndex = i
			break
		}
	}

	if startIndex == -1 {
		return fmt.Errorf("tmux config item '%s' not found in config file", key)
	}

	// Find the end index (next key or end of file)
	endIndex := len(contents)
	for i := startIndex + 1; i < len(contents); i++ {
		// Check if line starts with non-whitespace (new key)
		if len(contents[i]) > 0 && contents[i][0] != ' ' && contents[i][0] != '\t' && contents[i][0] != '#' {
			endIndex = i
			break
		}
	}

	// Build new contents
	newContents := append(contents[:startIndex], contents[endIndex:]...)
	result := strings.TrimRight(strings.Join(newContents, "\n"), "\n")

	if dryRun {
		fmt.Println("Would have written to", file.Filepath)
		fmt.Println("New contents:")
		fmt.Println(result)
		return nil
	}

	return os.WriteFile(file.Filepath, []byte(result), 0644)
}

// dirFixForWrite replaces home directory with ~
func dirFixForWrite(dir string) string {
	if home, err := os.UserHomeDir(); err == nil {
		if strings.HasPrefix(dir, home) {
			return "~" + dir[len(home):]
		}
	}
	return dir
}

// dirFixForWriteRelative makes path relative to root and replaces home with ~
func dirFixForWriteRelative(dir, root string) string {
	// First try to make relative to root
	if strings.HasPrefix(dir, root) {
		rel := strings.TrimPrefix(dir, root)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return "./"
		}
		return "./" + rel
	}
	return dirFixForWrite(dir)
}

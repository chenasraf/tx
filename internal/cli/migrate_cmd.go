package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chenasraf/tx/internal/config"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate configuration from v1.x to v2.x format",
	Long: `Migrates your configuration from v1.x to v2.x format.

This finds any tmux_local config file (previously auto-discovered) and adds it
as an explicit include in your main config file's .config section.`,
	RunE: runMigrate,
}

func runMigrate(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	// Find the main config file
	info, err := config.GetTmuxConfigFileInfo()
	if err != nil {
		return NewUserError("no main config file found — nothing to migrate")
	}

	globalPath := info.Global.Filepath

	// Check if there's already an include list
	gc, _ := config.GetGlobalConfig()
	if gc != nil && len(gc.Include) > 0 {
		fmt.Println("Config already has includes — skipping migration.")
		fmt.Println("  includes:")
		for _, inc := range gc.Include {
			fmt.Println("    -", inc)
		}
		return nil
	}

	// Find legacy tmux_local file
	localPath := config.FindLegacyLocalConfig()
	if localPath == "" {
		fmt.Println("No tmux_local config file found — nothing to migrate.")
		return nil
	}

	// Compute the include path relative to the main config's directory
	globalDir := filepath.Dir(globalPath)
	includePath, err := filepath.Rel(globalDir, localPath)
	if err != nil {
		// Fall back to absolute path
		includePath = localPath
	}
	// Prefix with ./ for clarity if relative
	if !filepath.IsAbs(includePath) && !strings.HasPrefix(includePath, ".") {
		includePath = "./" + includePath
	}

	// Read the main config file
	data, err := os.ReadFile(globalPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	content := string(data)
	var newContent string

	if strings.Contains(content, ".config:") {
		// .config section exists — inject include under it
		newContent = injectIncludeIntoConfig(content, includePath)
	} else {
		// No .config section — prepend one
		newContent = fmt.Sprintf(".config:\n  include:\n    - %s\n\n%s", includePath, content)
	}

	if opts.Dry {
		fmt.Println("Would write to", globalPath)
		fmt.Println()
		fmt.Println(newContent)
		return nil
	}

	if err := os.WriteFile(globalPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Println("Migrated successfully!")
	fmt.Println()
	fmt.Printf("  Added include: %s\n", includePath)
	fmt.Printf("  Config file:   %s\n", globalPath)
	fmt.Printf("  Local file:    %s\n", localPath)

	return nil
}

// injectIncludeIntoConfig inserts an include entry into an existing .config section.
func injectIncludeIntoConfig(content, includePath string) string {
	lines := strings.Split(content, "\n")
	var result []string

	configIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == ".config:" {
			configIdx = i
			break
		}
	}

	if configIdx == -1 {
		// Shouldn't happen since caller checks, but be safe
		return fmt.Sprintf(".config:\n  include:\n    - %s\n\n%s", includePath, content)
	}

	// Find the end of the .config block (next top-level key or EOF)
	configEndIdx := len(lines)
	for i := configIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && line[0] != '#' {
			configEndIdx = i
			break
		}
	}

	// Check if include already exists in the block
	hasInclude := false
	for i := configIdx; i < configEndIdx; i++ {
		if strings.TrimSpace(lines[i]) == "include:" {
			hasInclude = true
			break
		}
	}

	if hasInclude {
		// Add entry to existing include list — find the include: line and add after it
		for i := configIdx; i < configEndIdx; i++ {
			result = append(result, lines[i])
			if strings.TrimSpace(lines[i]) == "include:" {
				result = append(result, fmt.Sprintf("    - %s", includePath))
			}
		}
		result = append(result, lines[configEndIdx:]...)
	} else {
		// Add include key right after .config:
		result = append(result, lines[:configIdx+1]...)
		result = append(result, "  include:")
		result = append(result, fmt.Sprintf("    - %s", includePath))
		result = append(result, lines[configIdx+1:]...)
	}

	return strings.Join(result, "\n")
}

package cli

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
	"github.com/chenasraf/tx/internal/fzf"
	"github.com/chenasraf/tx/internal/tmux"
	"github.com/spf13/cobra"
)

var (
	prjSave  bool
	prjLocal bool
)

var prjCmd = &cobra.Command{
	Use:     "prj [name]",
	Aliases: []string{"p"},
	Short:   "Create a new tmux session from project folder",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runPrj,
}

func init() {
	prjCmd.Flags().BoolVarP(&prjSave, "save", "s", false, "Save the session in config file")
	prjCmd.Flags().BoolVarP(&prjLocal, "local", "l", false, "Save the session in local config file")
}

func runPrj(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	var name string
	if len(args) > 0 {
		name = args[0]
	}

	// Get projects from ~/Dev
	projects, err := getProjects()
	if err != nil {
		return err
	}

	// If no name, use fuzzy finder to select from existing projects
	if name == "" {
		selected, err := fzf.Run(projects, fzf.Options{})
		if err != nil {
			return err
		}
		name = selected
	}

	if name == "" {
		return NewUserError("no selection")
	}

	// Build project directory path
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	projectDir := filepath.Join(home, "Dev", name)

	// Create directory if it doesn't exist
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		exec.Log(opts, "Creating dir:", projectDir)
		if !opts.Dry {
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				return err
			}
		}
	}

	// Parse config
	parsed := config.ParseConfig(name, config.TmuxConfigItemInput{
		Name:    config.NameFix(name),
		Root:    projectDir,
		Windows: []config.TmuxWindowInput{{IsString: true, String: "."}},
	})

	// Save if requested
	if prjSave {
		if err := config.AddSimpleConfigToFile(parsed, prjLocal, opts.Dry); err != nil {
			return err
		}
	}

	// Create session
	return tmux.CreateFromConfig(opts, parsed)
}

// getProjects returns directory names in ~/Dev
func getProjects() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	devDir := filepath.Join(home, "Dev")
	entries, err := os.ReadDir(devDir)
	if err != nil {
		return nil, err
	}

	var projects []string
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden directories (dot files)
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			projects = append(projects, name)
		}
	}

	// Case-insensitive sort
	sort.Slice(projects, func(i, j int) bool {
		return strings.ToLower(projects[i]) < strings.ToLower(projects[j])
	})
	return projects, nil
}

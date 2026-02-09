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

// ErrNoProjectsPath is returned when projects_path is not configured
var ErrNoProjectsPath = NewUserError("projects_path not configured. Add to your config file:\n\n.config:\n  projects_path: ~/Dev\n")

// getProjectsPath returns the configured projects path or error if not set
func getProjectsPath() (string, error) {
	// Try to get from config
	globalConfig, err := config.GetGlobalConfig()
	if err != nil || globalConfig == nil || globalConfig.ProjectsPath == "" {
		return "", ErrNoProjectsPath
	}

	// Expand ~ if present
	path := globalConfig.ProjectsPath
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	return path, nil
}

func runPrj(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	var name string
	if len(args) > 0 {
		name = args[0]
	}

	// Get projects path
	projectsPath, err := getProjectsPath()
	if err != nil {
		return err
	}

	// Get projects from projects path
	projects, err := getProjects(projectsPath)
	if err != nil {
		return err
	}

	// If no name, use fuzzy finder to select from existing projects
	if name == "" {
		items := make([]fzf.Item, len(projects))
		for i, p := range projects {
			items[i] = fzf.Item{Key: p, Display: p}
		}
		selected, err := fzf.Run(items, fzf.Options{})
		if err != nil {
			return err
		}
		name = selected
	}

	if name == "" {
		return NewUserError("no selection")
	}

	// Build project directory path
	projectDir := filepath.Join(projectsPath, name)

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

// getProjects returns directory names in the given path
func getProjects(projectsPath string) ([]string, error) {
	entries, err := os.ReadDir(projectsPath)
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

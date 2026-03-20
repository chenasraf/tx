package cli

import (
	"os"
	"path/filepath"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
	"github.com/chenasraf/tx/internal/tmux"
	"github.com/spf13/cobra"
)

var (
	createRootDir    string
	createWindows    []string
	createSave       bool
	createSaveOnly   bool
	createConfigFile string
)

var createCmd = &cobra.Command{
	Use:     "create",
	Aliases: []string{"c"},
	Short:   "Create a new tmux session (temporary)",
	RunE:    runCreate,
}

func init() {
	createCmd.Flags().StringVarP(&createRootDir, "root-dir", "r", "", "Root directory for the session")
	createCmd.Flags().StringArrayVarP(&createWindows, "window", "w", nil, "Add a window with the given directory (relative to root)")
	createCmd.Flags().BoolVarP(&createSave, "save", "s", false, "Save the session to config file")
	createCmd.Flags().BoolVarP(&createSaveOnly, "save-only", "S", false, "Save to config without creating session")
	createCmd.Flags().StringVarP(&createConfigFile, "config", "c", "", "Save to a specific config file")
}

func runCreate(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	exec.Log(opts, "Options:", createRootDir, createWindows, createSave, createSaveOnly)

	// Determine root directory
	rootDir := createRootDir
	if rootDir == "" {
		var err error
		rootDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	// Determine session name
	name := config.NameFix(filepath.Base(rootDir))

	// Build windows list
	windows := createWindows
	if len(windows) == 0 {
		windows = []string{"."}
	}

	// Build window inputs
	windowInputs := make([]config.TmuxWindowInput, 0, len(windows))
	for _, w := range windows {
		windowInputs = append(windowInputs, config.TmuxWindowInput{
			IsString: true,
			String:   w,
		})
	}

	// Parse the config
	parsed := config.ParseConfig(name, config.TmuxConfigItemInput{
		Name:    name,
		Root:    rootDir,
		Windows: windowInputs,
	})

	// Check if session exists
	if tmux.SessionExists(opts, parsed.Name) {
		exec.Log(opts, "Session already exists, attaching")
		return tmux.AttachToSession(opts, parsed.Name)
	}

	// Save if requested
	if createSave || createSaveOnly {
		if err := config.AddSimpleConfigToFile(parsed, createConfigFile, opts.Dry); err != nil {
			return err
		}
	}

	// If save only, we're done
	if createSaveOnly {
		return nil
	}

	// Create the session
	return tmux.CreateFromConfig(opts, parsed)
}

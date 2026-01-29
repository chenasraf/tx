package cli

import (
	"fmt"
	"os"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose bool
	dry     bool
)

// GetOpts returns the current execution options
func GetOpts() exec.Opts {
	return exec.Opts{
		Verbose: verbose,
		Dry:     dry,
	}
}

// UserError is an error type for user-facing errors
type UserError struct {
	Message string
}

func (e *UserError) Error() string {
	return e.Message
}

// NewUserError creates a new UserError
func NewUserError(message string) *UserError {
	return &UserError{Message: message}
}

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "tx [session]",
	Short: "Generate layouts for tmux using presets or on-the-fly args",
	Long: `tx is a tmux session manager that creates sessions from YAML configuration files.

It supports complex pane layouts, fzf selection, and config merging.`,
	Args:              cobra.MaximumNArgs(1),
	PersistentPreRunE: initConfig,
	RunE:              runMain,
}

// initConfig loads global configuration and applies settings
func initConfig(cmd *cobra.Command, args []string) error {
	// Try to load global config (ignore errors - config may not exist)
	globalConfig, err := config.GetGlobalConfig()
	if err == nil && globalConfig != nil {
		// Apply shell from config
		if globalConfig.Shell != "" {
			exec.Shell = globalConfig.Shell
		}
	}
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if _, ok := err.(*UserError); ok {
			fmt.Fprintln(os.Stderr, "Error:", err.Error())
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
	rootCmd.PersistentFlags().BoolVarP(&dry, "dry", "d", false, "Dry run (log commands, don't execute)")

	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(prjCmd)
}

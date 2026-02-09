package cli

import (
	"fmt"
	"os"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
	"github.com/chenasraf/tx/internal/fzf"
	"github.com/spf13/cobra"
)

var (
	// Version is set by main.go from embedded version.txt
	Version string

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
	SilenceErrors:     true, // We handle error printing in Execute()
	SilenceUsage:      true, // Don't print usage on runtime errors
	ValidArgsFunction: completeSessionNames,
}

// completeSessionNames returns session names for shell completion
func completeSessionNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Don't complete if we already have an argument
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := config.GetTmuxConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for name, item := range cfg {
		if name != config.ConfigKey {
			names = append(names, name)
			names = append(names, item.Aliases...)
		}
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// buildFzfItems creates fzf items from a config file
func buildFzfItems(cfg config.ConfigFile) []fzf.Item {
	items := make([]fzf.Item, 0, len(cfg))
	for k, v := range cfg {
		items = append(items, fzf.Item{Key: k, Name: k, Aliases: v.Aliases})
	}
	return items
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
		// Apply default layout from config
		if globalConfig.DefaultLayout != nil {
			config.ConfiguredDefaultLayout = globalConfig.DefaultLayout
		}
		// Apply named layouts from config
		if globalConfig.NamedLayouts != nil {
			config.ConfiguredNamedLayouts = globalConfig.NamedLayouts
		}
	}
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "Verbose logging")
	rootCmd.PersistentFlags().BoolVarP(&dry, "dry", "d", false, "Dry run (log commands, don't execute)")
	rootCmd.Flags().BoolP("version", "v", false, "Print version")

	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(prjCmd)
	rootCmd.AddCommand(killCmd)
}

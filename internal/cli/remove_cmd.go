package cli

import (
	"fmt"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
	"github.com/spf13/cobra"
)

var removeConfigFile string

var removeCmd = &cobra.Command{
	Use:               "remove <key>",
	Aliases:           []string{"rm"},
	Short:             "Remove a tmux workspace from the config file",
	Args:              cobra.ExactArgs(1),
	RunE:              runRemove,
	ValidArgsFunction: completeSessionNames,
}

func init() {
	removeCmd.Flags().StringVarP(&removeConfigFile, "config", "c", "", "Remove from a specific config file")
}

func runRemove(cmd *cobra.Command, args []string) error {
	opts := GetOpts()
	key := args[0]

	// Verify the key exists
	allConfig, err := config.GetTmuxConfig()
	if err != nil {
		return err
	}

	_, actualKey, exists := allConfig.Get(key)
	if !exists {
		return NewUserError("tmux config item '" + key + "' not found")
	}

	err = config.RemoveConfigFromFile(actualKey, removeConfigFile, opts.Dry)
	if err != nil {
		return err
	}

	if !opts.Dry {
		fmt.Printf("Removed tmux config item '%s'\n", key)
	}

	// Log action in verbose/dry mode
	exec.Log(opts, "Removed config item:", key)

	return nil
}

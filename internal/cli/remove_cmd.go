package cli

import (
	"fmt"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
	"github.com/spf13/cobra"
)

var removeLocal bool

var removeCmd = &cobra.Command{
	Use:               "remove <key>",
	Aliases:           []string{"rm"},
	Short:             "Remove a tmux workspace from the config file",
	Args:              cobra.ExactArgs(1),
	RunE:              runRemove,
	ValidArgsFunction: completeSessionNames,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeLocal, "local", "l", false, "Remove from local config file")
}

func runRemove(cmd *cobra.Command, args []string) error {
	opts := GetOpts()
	key := args[0]

	// Verify the key exists
	allConfig, err := config.GetTmuxConfig()
	if err != nil {
		return err
	}

	if _, exists := allConfig[key]; !exists {
		return NewUserError("tmux config item '" + key + "' not found")
	}

	err = config.RemoveConfigFromFile(key, removeLocal, opts.Dry)
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

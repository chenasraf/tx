package cli

import (
	"fmt"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
	"github.com/spf13/cobra"
)

var removeConfigFile string

var removeCmd = &cobra.Command{
	Use:               "remove <key...>",
	Aliases:           []string{"rm"},
	Short:             "Remove tmux workspaces from the config file",
	Args:              cobra.MinimumNArgs(1),
	RunE:              runRemove,
	ValidArgsFunction: completeSessionNamesMulti,
}

func init() {
	removeCmd.Flags().StringVarP(&removeConfigFile, "config", "c", "", "Remove from a specific config file")
}

func runRemove(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	allConfig, err := config.GetTmuxConfig()
	if err != nil {
		return err
	}

	var errs []error
	for _, key := range args {
		_, actualKey, exists := allConfig.Get(key)
		if !exists {
			errs = append(errs, NewUserError("tmux config item '"+key+"' not found"))
			continue
		}

		err = config.RemoveConfigFromFile(actualKey, removeConfigFile, opts.Dry)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if !opts.Dry {
			fmt.Printf("Removed tmux config item '%s'\n", key)
		}

		exec.Log(opts, "Removed config item:", key)
	}

	return joinErrors(errs)
}

package cli

import (
	"os"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
	"github.com/chenasraf/tx/internal/tmux"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:               "attach [key]",
	Aliases:           []string{"a"},
	Short:             "Attach to a tmux session",
	Args:              cobra.MaximumNArgs(1),
	RunE:              runAttach,
	ValidArgsFunction: completeSessionNames,
}

func runAttach(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	var key string
	if len(args) > 0 {
		key = args[0]
	}

	if key != "" {
		// Attach to specific session from config
		allConfig, err := config.GetTmuxConfig()
		if err != nil {
			return err
		}

		item, exists := allConfig[key]
		if !exists {
			return NewUserError("tmux config item '" + key + "' not found")
		}

		parsed := config.ParseConfig(key, item)

		if !tmux.SessionExists(opts, parsed.Name) {
			return NewUserError("tmux session '" + parsed.Name + "' does not exist")
		}

		return tmux.AttachToSession(opts, parsed.Name)
	}

	// No key - attach to last session if not already in tmux
	if os.Getenv("TMUX") != "" {
		exec.Log(opts, "Already in tmux and no key specified, not attaching")
		return nil
	}

	return exec.RunCommand(opts, "tmux attach")
}

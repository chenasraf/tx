package cli

import (
	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
	"github.com/chenasraf/tx/internal/fzf"
	"github.com/chenasraf/tx/internal/tmux"
	"github.com/spf13/cobra"
)

// runMain is the main command handler - opens or creates a session
func runMain(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	var key string
	if len(args) > 0 {
		key = args[0]
	}

	// If no key provided, use fzf to select
	if key == "" {
		info, err := config.GetTmuxConfigFileInfo()
		if err != nil {
			return err
		}

		keys := make([]string, 0, len(info.Merged.Config))
		for k := range info.Merged.Config {
			keys = append(keys, k)
		}

		selected, err := fzf.Run(keys, fzf.Options{})
		if err != nil {
			return err
		}

		if _, actualKey, exists := info.Merged.Config.Get(selected); !exists {
			return NewUserError("tmux config item '" + selected + "' not found")
		} else {
			key = actualKey
		}
	}

	// Get config
	allConfig, err := config.GetTmuxConfig()
	if err != nil {
		return err
	}

	item, actualKey, exists := allConfig.Get(key)
	if !exists {
		return NewUserError("tmux config item '" + key + "' not found")
	}

	parsed := config.ParseConfig(actualKey, item)

	// Check if session exists
	if tmux.SessionExists(opts, parsed.Name) {
		exec.Log(opts, "Session exists, attaching...")
		return tmux.AttachToSession(opts, parsed.Name)
	}

	// Create session
	return tmux.CreateFromConfig(opts, parsed)
}

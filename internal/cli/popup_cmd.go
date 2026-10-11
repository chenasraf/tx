package cli

import (
	"slices"

	"github.com/chenasraf/tx/internal/popup"
	"github.com/chenasraf/tx/internal/tmuxctl"
	"github.com/spf13/cobra"
)

var popupCmd = &cobra.Command{
	Use:   "popup",
	Short: "Open tmux popups that follow the terminal's size",
	Long: `Open tmux popups that follow the terminal's size.

The command runs in a hidden session and the popup holds a client attached to it, so when the
terminal resizes the popup can be reopened at the new size without restarting the command.
Reopening needs a client-resized hook:

  set-hook -g client-resized[10] 'run-shell -b "tx popup resized #{q:hook_client}"'`,
	// no global config needed, and these run on every keypress-bound popup
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
}

var popupOpenCmd = &cobra.Command{
	Use:   "open <client> <name> <dir> [display-popup flags...] -- <command...>",
	Short: "Run a command in a popup on a client",
	Long: `Run a command in <dir> inside a popup on <client>.

Flags before -- are passed to display-popup (e.g. -w 90% -h 90% -b rounded -T ' title ').
A single-word command runs through the shell, as tmux does; several words run directly.

Example:
  bind g run-shell "tx popup open #{q:client_name} lazygit #{q:pane_current_path} -w 90% -h 90% -- lazygit"`,
	// display-popup's flags are passed through untouched
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		split := slices.Index(args, "--")
		if split < 3 || split == len(args)-1 {
			if slices.Contains(args, "-h") || slices.Contains(args, "--help") {
				return cmd.Help()
			}
			return NewUserError("usage: tx popup open <client> <name> <dir> [display-popup flags...] -- <command...>")
		}
		return popup.Open(tmuxctl.Current(), args[0], args[1], args[2], args[3:split], args[split+1:])
	},
}

var popupResizedCmd = &cobra.Command{
	Use:   "resized <client>",
	Short: "Reopen a client's popups at its new size (client-resized hook)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return popup.Resized(tmuxctl.Current(), args[0])
	},
}

func init() {
	popupCmd.AddCommand(popupOpenCmd)
	popupCmd.AddCommand(popupResizedCmd)
}

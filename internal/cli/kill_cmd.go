package cli

import (
	"github.com/chenasraf/tx/internal/exec"
	"github.com/chenasraf/tx/internal/tmux"
	"github.com/spf13/cobra"
)

var killCmd = &cobra.Command{
	Use:               "kill [session]",
	Aliases:           []string{"k"},
	Short:             "Kill a running tmux session (current session if no arg)",
	Args:              cobra.MaximumNArgs(1),
	RunE:              runKill,
	ValidArgsFunction: completeRunningSessions,
}

func runKill(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	if len(args) > 0 {
		sessionName := args[0]
		// Check if session exists
		if !tmux.SessionExists(opts, sessionName) {
			return NewUserError("tmux session '" + sessionName + "' does not exist")
		}
		return tmux.KillSession(opts, sessionName)
	}

	// No arg - kill current session
	return exec.RunCommand(opts, "tmux kill-session")
}

// completeRunningSessions returns running session names for shell completion
func completeRunningSessions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Don't complete if we already have an argument
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return tmux.GetSessionNames(), cobra.ShellCompDirectiveNoFileComp
}

package cli

import (
	"github.com/chenasraf/tx/internal/exec"
	"github.com/chenasraf/tx/internal/tmux"
	"github.com/spf13/cobra"
)

var killCmd = &cobra.Command{
	Use:               "kill [session...]",
	Aliases:           []string{"k"},
	Short:             "Kill running tmux sessions (current session if no arg)",
	RunE:              runKill,
	ValidArgsFunction: completeRunningSessions,
}

func runKill(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	if len(args) > 0 {
		var errs []error
		for _, sessionName := range args {
			if !tmux.SessionExists(opts, sessionName) {
				errs = append(errs, NewUserError("tmux session '"+sessionName+"' does not exist"))
				continue
			}
			if err := tmux.KillSession(opts, sessionName); err != nil {
				errs = append(errs, err)
			}
		}
		return joinErrors(errs)
	}

	// No arg - kill current session
	return exec.RunCommand(opts, "tmux kill-session")
}

// completeRunningSessions returns running session names for shell completion,
// excluding sessions already provided as arguments.
func completeRunningSessions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	all := tmux.GetSessionNames()
	return filterUsed(all, args), cobra.ShellCompDirectiveNoFileComp
}

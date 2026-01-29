package tmux

import (
	"os"

	"github.com/chenasraf/tx/internal/exec"
)

// SessionExists checks if a tmux session with the given name exists
func SessionExists(opts exec.Opts, sessionName string) bool {
	// Always check for real, even in dry mode
	realOpts := exec.Opts{Verbose: opts.Verbose, Dry: false}
	_, code, err := exec.GetCommandOutput(realOpts, "tmux has-session -t "+sessionName)
	if err != nil {
		return false
	}
	return code == 0
}

// AttachToSession attaches to or switches to a tmux session
func AttachToSession(opts exec.Opts, sessionName string) error {
	if os.Getenv("TMUX") != "" {
		// Already inside tmux, use switch-client
		return exec.RunCommand(opts, "tmux switch-client -t "+sessionName)
	}
	// Outside tmux, use attach
	return exec.RunCommand(opts, "tmux attach -t "+sessionName)
}

// ListSessions returns the output of `tmux ls`
func ListSessions(opts exec.Opts) (string, error) {
	output, _, err := exec.GetCommandOutput(opts, "tmux ls")
	return output, err
}

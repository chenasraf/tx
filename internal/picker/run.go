// Package picker is a full-screen picker over tmux sessions and windows, or agent panes, with a
// live preview of what each row holds. It talks to tmux over one control-mode connection.
package picker

import (
	"errors"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/chenasraf/tx/internal/theme"
	"github.com/chenasraf/tx/internal/tmuxctl"
)

// Options configure a picker run.
type Options struct {
	// Client is the tmux client that Enter switches; empty means the one viewing this pane.
	Client         string
	Refresh        time.Duration
	PreviewPercent int
	Colors         map[string]string
}

// ErrNotInTmux is returned when a picker runs outside tmux.
var ErrNotInTmux = errors.New("the picker runs inside tmux")

// Sessions runs the session and window picker.
func Sessions(opts Options) error {
	return run(opts, func(client string, th theme.Theme) source { return newSessions(client, th) })
}

// Agents runs the agent pane picker. commands are process names or /regex/ patterns; mode is
// MatchAuto, MatchName or MatchArgs.
func Agents(opts Options, commands []string, mode string) error {
	switch mode {
	case "", MatchAuto, MatchName, MatchArgs:
	default:
		return fmt.Errorf("unknown agent match mode %q (want auto, name or args)", mode)
	}
	matchers, err := parseMatchers(commands)
	if err != nil {
		return err
	}
	return run(opts, func(client string, th theme.Theme) source { return newAgents(client, th, matchers, mode) })
}

func run(opts Options, newSource func(client string, th theme.Theme) source) error {
	pane := os.Getenv("TMUX_PANE")
	if pane == "" {
		return ErrNotInTmux
	}
	c, err := tmuxctl.Current().Dial(pane)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	th, err := theme.Load(c, opts.Colors)
	if err != nil {
		return err
	}
	client := opts.Client
	if client == "" {
		client = viewingClient(c, pane)
	}
	src := newSource(client, th)
	if err := src.load(c); err != nil {
		return err
	}

	m := newModel(src, c, th, opts.PreviewPercent, opts.Refresh)
	final, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		return err
	}
	fm := final.(*model)
	if fm.chosen != "" {
		return src.accept(c, fm.chosen)
	}
	return fm.err
}

// viewingClient is the terminal client attached to pane's session. Commands sent over the control
// connection otherwise act on the control client itself.
func viewingClient(r tmuxctl.Runner, pane string) string {
	res, err := r.Run([]string{"list-clients", "-t", pane, "-F", "#{client_name}\t#{client_control_mode}"})
	if err != nil {
		return ""
	}
	for _, row := range res[0].Rows() {
		if len(row) == 2 && row[1] == "0" {
			return row[0]
		}
	}
	return ""
}

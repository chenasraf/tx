package picker

import (
	"os/exec"

	"github.com/chenasraf/tx/internal/tmuxctl"
)

const reset = "\x1b[0m"

// Row is one line of the list.
type Row struct {
	// ID is the tmux id the row stands for: a session ($), window (@) or pane (%).
	ID        string
	SessionID string
	Display   string
	// Matched marks a row the query matched, rather than one listed for context.
	Matched bool
}

// source supplies a picker's rows and actions. A picker runs in one goroutine, except
// preview, which runs in the background and must only use its arguments.
type source interface {
	// title names the picker, for the popup frame.
	title() string
	keys() [][]key
	// load reads a fresh snapshot from tmux.
	load(r tmuxctl.Runner) error
	// rows lists the snapshot, filtered by query.
	rows(query string) []Row
	// start is the row to open on.
	start() string
	// digit is the row quick key n (0-9) picks, or "".
	digit(n int) string
	preview(r tmuxctl.Runner, id string, cols, lines int) string
	// describe names what a kill of row id takes down.
	describe(id string) string
	// kill returns the row to land on afterwards and the kill itself, which may take a while and
	// runs in the background.
	kill(id string) (focus string, run func(r tmuxctl.Runner) error)
	accept(r tmuxctl.Runner, id string) error
	// remember is the global option that records the highlighted row, or "".
	remember() string
	// pollList re-lists rows on every refresh tick, for changes tmux sends no notification for.
	pollList() bool
}

// navigation is what h/l and the arrows do in a tree.
type navigation int

const (
	navNone navigation = iota
	navDown
	navReload
)

// tree is a source whose rows expand and collapse.
type tree interface {
	expand(r tmuxctl.Runner, id string) (navigation, string)
	collapse(r tmuxctl.Runner, id string) (navigation, string)
	// reveal makes row id visible once the query is cleared.
	reveal(r tmuxctl.Runner, id string)
}

// peeker is a source whose rows can be worked in from inside the picker.
type peeker interface {
	// peek prepares row id and returns the client to run and a cleanup that reports whether the
	// picker should quit afterwards.
	peek(r tmuxctl.Runner, id string) (*exec.Cmd, func() bool, error)
}

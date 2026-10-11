//go:build windows

package popup

import (
	"errors"

	"github.com/chenasraf/tx/internal/tmuxctl"
)

// Prefix starts the name of every hidden popup session; session lists leave these out.
const Prefix = "_popup_"

var errUnsupported = errors.New("tmux popups are not supported on Windows")

// Open is not supported on Windows.
func Open(s tmuxctl.Server, client, name, dir string, flags, command []string) error {
	return errUnsupported
}

// Resized is not supported on Windows.
func Resized(s tmuxctl.Server, client string) error {
	return errUnsupported
}

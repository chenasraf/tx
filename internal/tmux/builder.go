package tmux

import (
	"fmt"
	"path/filepath"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
)

// CreateFromConfig creates a tmux session from a parsed config
func CreateFromConfig(opts exec.Opts, tmuxConfig config.ParsedTmuxConfigItem) error {
	root := tmuxConfig.Root
	windows := tmuxConfig.Windows
	sessionName := config.NameFix(tmuxConfig.Name)

	exec.Log(opts, "Config:", tmuxConfig)
	exec.Log(opts, "Session name:", sessionName)

	// Check if session already exists
	if SessionExists(opts, sessionName) {
		exec.Log(opts, fmt.Sprintf("tmux session %s already exists, attaching...", sessionName))
		return AttachToSession(opts, sessionName)
	}

	exec.Log(opts, fmt.Sprintf("tmux session %s does not exist, creating...", sessionName))

	var commands []string

	// Create the session
	commands = append(commands, fmt.Sprintf(
		"tmux -f ~/.config/tmux/conf.tmux new-session -d -s %s -n general -c %s; sleep 1",
		sessionName, root,
	))

	// Create all windows
	for i, window := range windows {
		dir := window.Cwd
		windowName := window.Name
		if windowName == "" {
			windowName = config.NameFix(filepath.Base(dir))
		}

		exec.Log(opts, "Creating window:", windowName)

		// Create the window
		commands = append(commands, fmt.Sprintf(
			"tmux new-window -t %s:%d -n %s -c %s",
			sessionName, i+1, windowName, dir,
		))

		// Get pane commands - paneIndex starts at 0 for the window's initial pane
		paneIndex := 0
		paneCommands, timePanes := getPaneCommands(opts, window.Layout, sessionName, windowName, root, &paneIndex)
		commands = append(commands, paneCommands...)

		// Execute clock-mode for panes that have Time: true
		for _, paneIdx := range timePanes {
			commands = append(commands, fmt.Sprintf("tmux clock-mode -t %s:%s.%d", sessionName, windowName, paneIdx))
		}

		// Select first pane
		commands = append(commands, fmt.Sprintf("tmux select-pane -t %s.0", sessionName))
	}

	// Select first window
	commands = append(commands, fmt.Sprintf("tmux select-window -t %s:1", sessionName))

	// Execute all commands
	for _, command := range commands {
		if err := exec.RunCommand(opts, command); err != nil {
			return fmt.Errorf("failed to execute: %s: %w", command, err)
		}
	}

	// Attach to the session
	return AttachToSession(opts, sessionName)
}

// getPaneCommands generates tmux commands for pane layout
// Returns the commands and a list of pane indices that should show time (clock-mode)
func getPaneCommands(opts exec.Opts, pane config.TmuxPaneLayout, sessionName, windowName, rootDir string, paneIndex *int) ([]string, []int) {
	var commands []string
	var timePanes []int

	currentPane := *paneIndex

	// Check if this pane should show time
	if pane.Clock {
		timePanes = append(timePanes, currentPane)
	}

	// Send command if specified
	if pane.Cmd != "" {
		cmd := TransformCmdToTmuxKeys(pane.Cmd)
		if cmd != "" {
			exec.Log(opts, "Sending keys:", cmd)
			commands = append(commands, fmt.Sprintf(
				"tmux send-keys -t %s:%s %s Enter",
				sessionName, windowName, cmd,
			))
		}
	}

	// Handle splits
	if pane.Split != nil {
		direction := pane.Split.Direction
		if direction == "" {
			direction = "h"
		}

		cwd := pane.Cwd
		if cwd == "" {
			cwd = rootDir
		}

		exec.Log(opts, "Splitting pane:", pane.Split, "direction:", direction)

		// Increment pane index for the new pane created by split
		*paneIndex++

		commands = append(commands, fmt.Sprintf(
			"tmux split-window -%s -t %s:%s -c %s",
			direction, sessionName, windowName, cwd,
		))

		// Handle child pane
		if pane.Split.Child != nil {
			exec.Log(opts, "Handling child pane:", pane.Split.Child)
			childCommands, childTimePanes := getPaneCommands(opts, *pane.Split.Child, sessionName, windowName, rootDir, paneIndex)
			commands = append(commands, childCommands...)
			timePanes = append(timePanes, childTimePanes...)
		}

		// Handle zoom
		if pane.Zoom {
			commands = append(commands, fmt.Sprintf(
				"tmux resize-pane -t %s:%s -Z",
				sessionName, windowName,
			))
		}
	}

	return commands, timePanes
}

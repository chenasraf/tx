package tmux

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
)

// CreateFromConfig creates a tmux session from a parsed config.
// If background is true, the session is created but not attached to.
func CreateFromConfig(opts exec.Opts, tmuxConfig config.ParsedTmuxConfigItem, background bool) error {
	root := tmuxConfig.Root
	windows := tmuxConfig.Windows
	sessionName := config.NameFix(tmuxConfig.Name)

	exec.Log(opts, "Config:", tmuxConfig)
	exec.Log(opts, "Session name:", sessionName)

	// Check if session already exists
	if SessionExists(opts, sessionName) {
		if background {
			exec.Log(opts, fmt.Sprintf("tmux session %s already exists (background mode, not attaching)", sessionName))
			return nil
		}
		exec.Log(opts, fmt.Sprintf("tmux session %s already exists, attaching...", sessionName))
		return AttachToSession(opts, sessionName)
	}

	exec.Log(opts, fmt.Sprintf("tmux session %s does not exist, creating...", sessionName))

	// Create the session immediately so subsequent commands can target it
	// and so we can query the actual window/pane base indices the running
	// server is using (show-options can return values that don't match what
	// new-session actually applies, depending on when/how the server started).
	createCmd := fmt.Sprintf("tmux new-session -d -s %s -n general -c %s; sleep 1", sessionName, root)
	if err := exec.RunCommand(opts, createCmd); err != nil {
		return fmt.Errorf("failed to execute: %s: %w", createCmd, err)
	}

	baseIndex := QueryTargetIndex(opts, sessionName+":general", "window_index")
	paneBaseIndex := QueryTargetIndex(opts, sessionName+":general", "pane_index")

	var commands []string

	// Create all windows. The "general" window above lands at baseIndex,
	// so user windows start at baseIndex+1.
	for i, window := range windows {
		windowName := windowNameFor(window)

		exec.Log(opts, "Creating window:", windowName)

		// Create the window
		commands = append(commands, fmt.Sprintf(
			"tmux new-window -t %s:%d -n %s -c %s",
			sessionName, baseIndex+1+i, windowName, window.Cwd,
		))
		commands = append(commands, getWindowCommands(opts, window, sessionName+":"+windowName, root, paneBaseIndex)...)
	}

	// Select initial window — InitialWindow is a 0-based offset from the
	// session's first window, so add baseIndex to get the actual tmux index.
	commands = append(commands, fmt.Sprintf("tmux select-window -t %s:%d", sessionName, baseIndex+tmuxConfig.InitialWindow))

	// Execute all commands
	for _, command := range commands {
		if err := exec.RunCommand(opts, command); err != nil {
			return fmt.Errorf("failed to execute: %s: %w", command, err)
		}
	}

	// Attach to the session unless background mode
	if background {
		return nil
	}
	return AttachToSession(opts, sessionName)
}

// AddToSession appends the config's windows to an existing session instead of
// creating a new one. initial_window keeps its meaning relative to a created
// session: 0 is the "general" window, which here is whichever window is
// already active, so the selection stays put; N selects the Nth added window.
// In background mode no window is selected.
func AddToSession(opts exec.Opts, tmuxConfig config.ParsedTmuxConfigItem, sessionName string, background bool) error {
	exec.Log(opts, "Config:", tmuxConfig)
	exec.Log(opts, "Adding windows to session:", sessionName)

	var targets []string
	for _, window := range tmuxConfig.Windows {
		windowName := windowNameFor(window)
		exec.Log(opts, "Creating window:", windowName)

		// Target the window by ID from here on: the session may already
		// have windows with the same name.
		createCmd := fmt.Sprintf(
			"tmux new-window -d -t %s: -n %s -c %s -P -F '#{window_id}'",
			sessionName, windowName, window.Cwd,
		)
		out, code, err := exec.GetCommandOutput(opts, createCmd)
		if err == nil && code != 0 {
			err = fmt.Errorf("exit status %d", code)
		}
		if err != nil {
			return fmt.Errorf("failed to execute: %s: %w", createCmd, err)
		}
		target := strings.TrimSpace(out)
		if target == "" {
			// Dry run: nothing was created, so there is no ID to target.
			target = sessionName + ":" + windowName
		}
		targets = append(targets, target)

		paneBaseIndex := QueryTargetIndex(opts, target, "pane_index")
		for _, command := range getWindowCommands(opts, window, target, tmuxConfig.Root, paneBaseIndex) {
			if err := exec.RunCommand(opts, command); err != nil {
				return fmt.Errorf("failed to execute: %s: %w", command, err)
			}
		}
	}

	if background || len(targets) == 0 || tmuxConfig.InitialWindow <= 0 {
		return nil
	}
	selected := min(tmuxConfig.InitialWindow, len(targets)) - 1
	return exec.RunCommand(opts, "tmux select-window -t "+targets[selected])
}

// windowNameFor returns the window's configured name, falling back to the
// name of its directory.
func windowNameFor(window config.ParsedTmuxWindow) string {
	if window.Name != "" {
		return window.Name
	}
	return config.NameFix(filepath.Base(window.Cwd))
}

// getWindowCommands generates the commands that lay out the panes of an
// already-created window, addressed by target.
func getWindowCommands(opts exec.Opts, window config.ParsedTmuxWindow, target, rootDir string, paneBaseIndex int) []string {
	paneIndex := paneBaseIndex
	commands, timePanes := getPaneCommands(opts, window.Layout, target, rootDir, &paneIndex)

	for _, paneIdx := range timePanes {
		commands = append(commands, fmt.Sprintf("tmux clock-mode -t %s.%d", target, paneIdx))
	}

	return append(commands, fmt.Sprintf("tmux select-pane -t %s.%d", target, paneBaseIndex))
}

// getPaneCommands generates tmux commands for pane layout
// Returns the commands and a list of pane indices that should show time (clock-mode)
func getPaneCommands(opts exec.Opts, pane config.TmuxPaneLayout, target, rootDir string, paneIndex *int) ([]string, []int) {
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
				"tmux send-keys -t %s %s Enter",
				target, cmd,
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

		splitCmd := fmt.Sprintf(
			"tmux split-window -%s -t %s -c %s",
			direction, target, cwd,
		)
		if pane.Split.Size > 0 && pane.Split.Size <= 100 {
			splitCmd += fmt.Sprintf(" -p %d", pane.Split.Size)
		}
		commands = append(commands, splitCmd)

		// Handle child pane
		if pane.Split.Child != nil {
			exec.Log(opts, "Handling child pane:", pane.Split.Child)
			childCommands, childTimePanes := getPaneCommands(opts, *pane.Split.Child, target, rootDir, paneIndex)
			commands = append(commands, childCommands...)
			timePanes = append(timePanes, childTimePanes...)
		}

		// Handle zoom
		if pane.Zoom {
			commands = append(commands, fmt.Sprintf(
				"tmux resize-pane -t %s -Z",
				target,
			))
		}
	}

	return commands, timePanes
}

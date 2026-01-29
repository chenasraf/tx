package exec

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// Opts contains execution options
type Opts struct {
	Verbose bool
	Dry     bool
}

// DefaultShells is the list of shells to try in order of preference
var DefaultShells = []string{"/bin/zsh", "/bin/bash", "/bin/sh"}

// Shell can be set to override the default shell detection
// Set this from config on startup
var Shell string

// getShell returns the shell to use for command execution
// Priority: Shell variable (from config) > $SHELL env > auto-detect
func getShell() string {
	// Use configured shell if set (from .config section)
	if Shell != "" {
		return Shell
	}

	// Check $SHELL environment variable
	if envShell := os.Getenv("SHELL"); envShell != "" {
		return envShell
	}

	// Auto-detect from default shells
	for _, shell := range DefaultShells {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}

	// Fallback to sh (should always exist)
	return "sh"
}

// Log prints a message if verbose or dry mode is enabled
func Log(opts Opts, args ...any) {
	if opts.Verbose || opts.Dry {
		fmt.Println(args...)
	}
}

// RunCommand executes a command with inherited stdio
func RunCommand(opts Opts, command string) error {
	Log(opts, "$", command)
	if opts.Dry {
		return nil
	}

	cmd := exec.Command(getShell(), "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// GetCommandOutput executes a command and returns its output
func GetCommandOutput(opts Opts, command string) (string, int, error) {
	Log(opts, "$", command)
	if opts.Dry {
		return "", 0, nil
	}

	cmd := exec.Command(getShell(), "-c", command)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", -1, err
		}
	}

	return stdout.String(), exitCode, nil
}

// RunCommandSilent executes a command without output and returns the exit code
func RunCommandSilent(command string) (int, error) {
	cmd := exec.Command(getShell(), "-c", command)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return -1, err
	}
	return 0, nil
}

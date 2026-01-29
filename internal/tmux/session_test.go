package tmux

import (
	"os"
	"testing"

	"github.com/chenasraf/tx/internal/exec"
)

func TestSessionExists_DryMode(t *testing.T) {
	// In dry mode, SessionExists should still check for real
	opts := exec.Opts{Verbose: false, Dry: true}

	// This test checks that the function doesn't crash
	// The actual result depends on whether tmux is running
	_ = SessionExists(opts, "nonexistent-test-session-12345")
}

func TestAttachToSession_InsideTmux(t *testing.T) {
	// Save original TMUX env
	origTmux := os.Getenv("TMUX")
	defer func() { _ = os.Setenv("TMUX", origTmux) }()

	// Set TMUX to simulate being inside tmux
	_ = os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")

	opts := exec.Opts{Verbose: false, Dry: true}
	err := AttachToSession(opts, "testsession")

	// In dry mode, should succeed without actually running
	if err != nil {
		t.Errorf("expected no error in dry mode, got %v", err)
	}
}

func TestAttachToSession_OutsideTmux(t *testing.T) {
	// Save original TMUX env
	origTmux := os.Getenv("TMUX")
	defer func() { _ = os.Setenv("TMUX", origTmux) }()

	// Unset TMUX to simulate being outside tmux
	_ = os.Unsetenv("TMUX")

	opts := exec.Opts{Verbose: false, Dry: true}
	err := AttachToSession(opts, "testsession")

	// In dry mode, should succeed without actually running
	if err != nil {
		t.Errorf("expected no error in dry mode, got %v", err)
	}
}

func TestListSessions_DryMode(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}

	output, err := ListSessions(opts)
	if err != nil {
		t.Errorf("expected no error in dry mode, got %v", err)
	}

	// In dry mode, output should be empty
	if output != "" {
		t.Errorf("expected empty output in dry mode, got %q", output)
	}
}

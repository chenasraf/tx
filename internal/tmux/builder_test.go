package tmux

import (
	"strings"
	"testing"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
)

func TestGetPaneCommands_NoSplit(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}
	pane := config.TmuxPaneLayout{
		Cwd: "/tmp/test",
	}

	paneIndex := 0
	commands, _ := getPaneCommands(opts, pane, "testsession", "testwindow", "/tmp", &paneIndex)

	// No split, no cmd = no commands
	if len(commands) != 0 {
		t.Errorf("expected 0 commands, got %d: %v", len(commands), commands)
	}
}

func TestGetPaneCommands_WithCmd(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}
	pane := config.TmuxPaneLayout{
		Cwd: "/tmp/test",
		Cmd: "npm start",
	}

	paneIndex := 0
	commands, _ := getPaneCommands(opts, pane, "testsession", "testwindow", "/tmp", &paneIndex)

	if len(commands) != 1 {
		t.Fatalf("expected 1 command, got %d: %v", len(commands), commands)
	}

	expected := "tmux send-keys -t testsession:testwindow npm Space start Enter"
	if commands[0] != expected {
		t.Errorf("expected %q, got %q", expected, commands[0])
	}
}

func TestGetPaneCommands_WithSplit(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}
	pane := config.TmuxPaneLayout{
		Cwd: "/tmp/test",
		Split: &config.TmuxSplitLayout{
			Direction: "h",
			Child: &config.TmuxPaneLayout{
				Cwd: "/tmp/test/child",
			},
		},
	}

	paneIndex := 0
	commands, _ := getPaneCommands(opts, pane, "testsession", "testwindow", "/tmp", &paneIndex)

	if len(commands) < 1 {
		t.Fatalf("expected at least 1 command, got %d", len(commands))
	}

	// First command should be split-window
	if commands[0] != "tmux split-window -h -t testsession:testwindow -c /tmp/test" {
		t.Errorf("unexpected split command: %q", commands[0])
	}
}

func TestGetPaneCommands_WithVerticalSplit(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}
	pane := config.TmuxPaneLayout{
		Cwd: "/tmp/test",
		Split: &config.TmuxSplitLayout{
			Direction: "v",
			Child: &config.TmuxPaneLayout{
				Cwd: "/tmp/test/child",
			},
		},
	}

	paneIndex := 0
	commands, _ := getPaneCommands(opts, pane, "testsession", "testwindow", "/tmp", &paneIndex)

	if len(commands) < 1 {
		t.Fatalf("expected at least 1 command, got %d", len(commands))
	}

	// Should use -v for vertical split
	if commands[0] != "tmux split-window -v -t testsession:testwindow -c /tmp/test" {
		t.Errorf("unexpected split command: %q", commands[0])
	}
}

func TestGetPaneCommands_WithZoom(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}
	pane := config.TmuxPaneLayout{
		Cwd:  "/tmp/test",
		Zoom: true,
		Split: &config.TmuxSplitLayout{
			Direction: "h",
			Child: &config.TmuxPaneLayout{
				Cwd: "/tmp/test/child",
			},
		},
	}

	paneIndex := 0
	commands, _ := getPaneCommands(opts, pane, "testsession", "testwindow", "/tmp", &paneIndex)

	// Should have a resize-pane -Z command
	found := false
	for _, cmd := range commands {
		if cmd == "tmux resize-pane -t testsession:testwindow -Z" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected zoom command, commands: %v", commands)
	}
}

func TestGetPaneCommands_NestedSplits(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}
	pane := config.TmuxPaneLayout{
		Cwd: "/tmp/test",
		Split: &config.TmuxSplitLayout{
			Direction: "h",
			Child: &config.TmuxPaneLayout{
				Cwd: "/tmp/test/child1",
				Split: &config.TmuxSplitLayout{
					Direction: "v",
					Child: &config.TmuxPaneLayout{
						Cwd: "/tmp/test/child2",
					},
				},
			},
		},
	}

	paneIndex := 0
	commands, _ := getPaneCommands(opts, pane, "testsession", "testwindow", "/tmp", &paneIndex)

	// Should have at least 2 split commands
	splitCount := 0
	for _, cmd := range commands {
		if strings.HasPrefix(cmd, "tmux split-window") {
			splitCount++
		}
	}

	if splitCount < 2 {
		t.Errorf("expected at least 2 split commands, got %d: %v", splitCount, commands)
	}
}

func TestGetPaneCommands_DefaultDirection(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}
	pane := config.TmuxPaneLayout{
		Cwd: "/tmp/test",
		Split: &config.TmuxSplitLayout{
			Direction: "", // Empty should default to "h"
			Child: &config.TmuxPaneLayout{
				Cwd: "/tmp/test/child",
			},
		},
	}

	paneIndex := 0
	commands, _ := getPaneCommands(opts, pane, "testsession", "testwindow", "/tmp", &paneIndex)

	if len(commands) < 1 {
		t.Fatalf("expected at least 1 command, got %d", len(commands))
	}

	// Should default to -h
	if commands[0] != "tmux split-window -h -t testsession:testwindow -c /tmp/test" {
		t.Errorf("expected default horizontal split, got: %q", commands[0])
	}
}

func TestGetPaneCommands_WithClock(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}
	pane := config.TmuxPaneLayout{
		Cwd:   "/tmp/test",
		Clock: true,
	}

	paneIndex := 0
	_, clockPanes := getPaneCommands(opts, pane, "testsession", "testwindow", "/tmp", &paneIndex)

	if len(clockPanes) != 1 {
		t.Errorf("expected 1 clock pane, got %d", len(clockPanes))
	}

	if len(clockPanes) > 0 && clockPanes[0] != 0 {
		t.Errorf("expected clock pane index 0, got %d", clockPanes[0])
	}
}

func TestGetPaneCommands_NestedClock(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}
	pane := config.TmuxPaneLayout{
		Cwd: "/tmp/test",
		Split: &config.TmuxSplitLayout{
			Direction: "h",
			Child: &config.TmuxPaneLayout{
				Cwd: "/tmp/test/child",
				Split: &config.TmuxSplitLayout{
					Direction: "v",
					Child: &config.TmuxPaneLayout{
						Cwd:   "/tmp/test/child2",
						Clock: true,
					},
				},
			},
		},
	}

	paneIndex := 0
	_, clockPanes := getPaneCommands(opts, pane, "testsession", "testwindow", "/tmp", &paneIndex)

	if len(clockPanes) != 1 {
		t.Errorf("expected 1 clock pane, got %d", len(clockPanes))
	}

	// The clock pane should be index 2 (after 2 splits)
	if len(clockPanes) > 0 && clockPanes[0] != 2 {
		t.Errorf("expected clock pane index 2, got %d", clockPanes[0])
	}
}

func TestCreateFromConfig_DryRun(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}

	tmuxConfig := config.ParsedTmuxConfigItem{
		Name: "testproject",
		Root: "/tmp/testproject",
		Windows: []config.ParsedTmuxWindow{
			{
				Name: "main",
				Cwd:  "/tmp/testproject",
				Layout: config.TmuxPaneLayout{
					Cwd: "/tmp/testproject",
				},
			},
		},
	}

	// In dry mode, this should succeed without actually running tmux
	err := CreateFromConfig(opts, tmuxConfig, false)
	if err != nil {
		t.Errorf("expected no error in dry mode, got %v", err)
	}
}

func TestCreateFromConfig_MultipleWindows(t *testing.T) {
	opts := exec.Opts{Verbose: false, Dry: true}

	tmuxConfig := config.ParsedTmuxConfigItem{
		Name: "testproject",
		Root: "/tmp/testproject",
		Windows: []config.ParsedTmuxWindow{
			{
				Name: "src",
				Cwd:  "/tmp/testproject/src",
				Layout: config.TmuxPaneLayout{
					Cwd: "/tmp/testproject/src",
				},
			},
			{
				Name: "lib",
				Cwd:  "/tmp/testproject/lib",
				Layout: config.TmuxPaneLayout{
					Cwd: "/tmp/testproject/lib",
				},
			},
		},
	}

	err := CreateFromConfig(opts, tmuxConfig, false)
	if err != nil {
		t.Errorf("expected no error in dry mode, got %v", err)
	}
}

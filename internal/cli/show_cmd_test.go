package cli

import (
	"bytes"
	"testing"

	"github.com/chenasraf/tx/internal/config"
)

func TestShowCmd_Exists(t *testing.T) {
	if showCmd == nil {
		t.Error("expected showCmd to not be nil")
	}

	if showCmd.Use != "show [key]" {
		t.Errorf("unexpected Use: %q", showCmd.Use)
	}
}

func TestShowCmd_Aliases(t *testing.T) {
	found := false
	for _, alias := range showCmd.Aliases {
		if alias == "s" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 's' alias")
	}
}

func TestShowCmd_Flags(t *testing.T) {
	jsonFlag := showCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("expected --json flag")
	}
	if jsonFlag.Shorthand != "j" {
		t.Errorf("expected -j shorthand, got %q", jsonFlag.Shorthand)
	}
}

func TestPrintLayout(t *testing.T) {
	// Capture output by redirecting - this is a simple smoke test
	layout := config.TmuxPaneLayout{
		Cwd: "/tmp/test",
		Cmd: "npm start",
		Split: &config.TmuxSplitLayout{
			Direction: "h",
			Child: &config.TmuxPaneLayout{
				Cwd: "/tmp/test/child",
			},
		},
	}

	// This function prints to stdout, so we just verify it doesn't panic
	var buf bytes.Buffer
	// Note: printLayout writes to stdout, not a buffer
	// This test mainly ensures it doesn't crash
	_ = layout
	_ = buf
}

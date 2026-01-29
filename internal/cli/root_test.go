package cli

import (
	"testing"
)

func TestGetOpts(t *testing.T) {
	// Reset global flags
	verbose = false
	dry = false

	opts := GetOpts()
	if opts.Verbose {
		t.Error("expected Verbose to be false")
	}
	if opts.Dry {
		t.Error("expected Dry to be false")
	}

	verbose = true
	dry = true

	opts = GetOpts()
	if !opts.Verbose {
		t.Error("expected Verbose to be true")
	}
	if !opts.Dry {
		t.Error("expected Dry to be true")
	}

	// Reset for other tests
	verbose = false
	dry = false
}

func TestUserError(t *testing.T) {
	err := NewUserError("test error message")

	if err.Error() != "test error message" {
		t.Errorf("expected 'test error message', got %q", err.Error())
	}

	if err.Message != "test error message" {
		t.Errorf("expected Message to be 'test error message', got %q", err.Message)
	}
}

func TestUserError_Interface(t *testing.T) {
	var err error = NewUserError("test")

	if err.Error() != "test" {
		t.Errorf("expected 'test', got %q", err.Error())
	}
}

func TestRootCmd_Exists(t *testing.T) {
	if rootCmd == nil {
		t.Error("expected rootCmd to not be nil")
	}

	if rootCmd.Use != "tx [session]" {
		t.Errorf("unexpected Use: %q", rootCmd.Use)
	}
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	commands := rootCmd.Commands()

	// Only check our custom commands, not built-in ones like "completion" and "help"
	expectedCmds := []string{"attach", "create", "edit", "list", "prj", "remove", "show"}
	cmdNames := make(map[string]bool)
	for _, cmd := range commands {
		cmdNames[cmd.Name()] = true
	}

	for _, expected := range expectedCmds {
		if !cmdNames[expected] {
			t.Errorf("expected subcommand %q to exist", expected)
		}
	}
}

func TestRootCmd_GlobalFlags(t *testing.T) {
	verboseFlag := rootCmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("expected --verbose flag")
	}

	dryFlag := rootCmd.PersistentFlags().Lookup("dry")
	if dryFlag == nil {
		t.Error("expected --dry flag")
	}
}

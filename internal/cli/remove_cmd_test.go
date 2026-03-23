package cli

import (
	"testing"
)

func TestRemoveCmd_Exists(t *testing.T) {
	if removeCmd == nil {
		t.Error("expected removeCmd to not be nil")
	}

	if removeCmd.Use != "remove <key...>" {
		t.Errorf("unexpected Use: %q", removeCmd.Use)
	}
}

func TestRemoveCmd_Aliases(t *testing.T) {
	found := false
	for _, alias := range removeCmd.Aliases {
		if alias == "rm" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'rm' alias")
	}
}

func TestRemoveCmd_Flags(t *testing.T) {
	configFlag := removeCmd.Flags().Lookup("config")
	if configFlag == nil {
		t.Fatal("expected --config flag")
	}
	if configFlag.Shorthand != "c" {
		t.Errorf("expected -c shorthand, got %q", configFlag.Shorthand)
	}
}

func TestRemoveCmd_RequiresArg(t *testing.T) {
	// The command requires exactly 1 argument
	if removeCmd.Args == nil {
		t.Error("expected Args validator")
	}
}

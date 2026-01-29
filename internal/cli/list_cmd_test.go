package cli

import (
	"testing"
)

func TestListCmd_Exists(t *testing.T) {
	if listCmd == nil {
		t.Error("expected listCmd to not be nil")
	}

	if listCmd.Use != "list" {
		t.Errorf("unexpected Use: %q", listCmd.Use)
	}
}

func TestListCmd_Aliases(t *testing.T) {
	if len(listCmd.Aliases) == 0 {
		t.Error("expected aliases")
	}

	found := false
	for _, alias := range listCmd.Aliases {
		if alias == "ls" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'ls' alias")
	}
}

func TestListCmd_Flags(t *testing.T) {
	bareFlag := listCmd.Flags().Lookup("bare")
	if bareFlag == nil {
		t.Error("expected --bare flag")
	}
	if bareFlag.Shorthand != "b" {
		t.Errorf("expected -b shorthand, got %q", bareFlag.Shorthand)
	}

	sessionsFlag := listCmd.Flags().Lookup("sessions")
	if sessionsFlag == nil {
		t.Error("expected --sessions flag")
	}
	if sessionsFlag.Shorthand != "s" {
		t.Errorf("expected -s shorthand, got %q", sessionsFlag.Shorthand)
	}
}

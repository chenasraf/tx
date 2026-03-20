package cli

import (
	"testing"
)

func TestEditCmd_Exists(t *testing.T) {
	if editCmd == nil {
		t.Error("expected editCmd to not be nil")
	}

	if editCmd.Use != "edit" {
		t.Errorf("unexpected Use: %q", editCmd.Use)
	}
}

func TestEditCmd_Aliases(t *testing.T) {
	found := false
	for _, alias := range editCmd.Aliases {
		if alias == "e" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'e' alias")
	}
}

func TestEditCmd_Flags(t *testing.T) {
	configFlag := editCmd.Flags().Lookup("config")
	if configFlag == nil {
		t.Fatal("expected --config flag")
	}
	if configFlag.Shorthand != "c" {
		t.Errorf("expected -c shorthand, got %q", configFlag.Shorthand)
	}
}

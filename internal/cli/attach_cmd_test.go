package cli

import (
	"testing"
)

func TestAttachCmd_Exists(t *testing.T) {
	if attachCmd == nil {
		t.Error("expected attachCmd to not be nil")
	}

	if attachCmd.Use != "attach [key]" {
		t.Errorf("unexpected Use: %q", attachCmd.Use)
	}
}

func TestAttachCmd_Aliases(t *testing.T) {
	found := false
	for _, alias := range attachCmd.Aliases {
		if alias == "a" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'a' alias")
	}
}

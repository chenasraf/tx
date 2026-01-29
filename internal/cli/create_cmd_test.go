package cli

import (
	"testing"
)

func TestCreateCmd_Exists(t *testing.T) {
	if createCmd == nil {
		t.Error("expected createCmd to not be nil")
	}

	if createCmd.Use != "create" {
		t.Errorf("unexpected Use: %q", createCmd.Use)
	}
}

func TestCreateCmd_Aliases(t *testing.T) {
	found := false
	for _, alias := range createCmd.Aliases {
		if alias == "c" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'c' alias")
	}
}

func TestCreateCmd_Flags(t *testing.T) {
	rootDirFlag := createCmd.Flags().Lookup("root-dir")
	if rootDirFlag == nil {
		t.Error("expected --root-dir flag")
	}
	if rootDirFlag.Shorthand != "r" {
		t.Errorf("expected -r shorthand, got %q", rootDirFlag.Shorthand)
	}

	windowFlag := createCmd.Flags().Lookup("window")
	if windowFlag == nil {
		t.Error("expected --window flag")
	}
	if windowFlag.Shorthand != "w" {
		t.Errorf("expected -w shorthand, got %q", windowFlag.Shorthand)
	}

	saveFlag := createCmd.Flags().Lookup("save")
	if saveFlag == nil {
		t.Error("expected --save flag")
	}
	if saveFlag.Shorthand != "s" {
		t.Errorf("expected -s shorthand, got %q", saveFlag.Shorthand)
	}

	saveOnlyFlag := createCmd.Flags().Lookup("save-only")
	if saveOnlyFlag == nil {
		t.Error("expected --save-only flag")
	}
	if saveOnlyFlag.Shorthand != "S" {
		t.Errorf("expected -S shorthand, got %q", saveOnlyFlag.Shorthand)
	}

	localFlag := createCmd.Flags().Lookup("local")
	if localFlag == nil {
		t.Error("expected --local flag")
	}
	if localFlag.Shorthand != "l" {
		t.Errorf("expected -l shorthand, got %q", localFlag.Shorthand)
	}
}

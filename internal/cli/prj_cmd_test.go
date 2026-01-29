package cli

import (
	"strings"
	"testing"
)

func TestPrjCmd_Exists(t *testing.T) {
	if prjCmd == nil {
		t.Error("expected prjCmd to not be nil")
	}

	if prjCmd.Use != "prj [name]" {
		t.Errorf("unexpected Use: %q", prjCmd.Use)
	}
}

func TestPrjCmd_Aliases(t *testing.T) {
	found := false
	for _, alias := range prjCmd.Aliases {
		if alias == "p" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'p' alias")
	}
}

func TestPrjCmd_Flags(t *testing.T) {
	saveFlag := prjCmd.Flags().Lookup("save")
	if saveFlag == nil {
		t.Error("expected --save flag")
	}
	if saveFlag.Shorthand != "s" {
		t.Errorf("expected -s shorthand, got %q", saveFlag.Shorthand)
	}

	localFlag := prjCmd.Flags().Lookup("local")
	if localFlag == nil {
		t.Error("expected --local flag")
	}
	if localFlag.Shorthand != "l" {
		t.Errorf("expected -l shorthand, got %q", localFlag.Shorthand)
	}
}

func TestGetProjects(t *testing.T) {
	// This test depends on the user's ~/Dev directory
	// Just verify it doesn't panic
	projects, err := getProjects()

	// If ~/Dev doesn't exist, that's fine
	if err != nil {
		t.Skip("~/Dev directory doesn't exist, skipping")
	}

	// Projects should be sorted case-insensitively
	for i := 1; i < len(projects); i++ {
		if strings.ToLower(projects[i-1]) > strings.ToLower(projects[i]) {
			t.Errorf("projects not sorted (case-insensitive): %q > %q", projects[i-1], projects[i])
		}
	}

	// No hidden directories should be included
	for _, p := range projects {
		if strings.HasPrefix(p, ".") {
			t.Errorf("hidden directory should be excluded: %q", p)
		}
	}
}

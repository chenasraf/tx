package cli

import (
	"os"
	"path/filepath"
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
	// Create a temp directory with test projects
	tempDir := t.TempDir()

	// Create some test directories
	testDirs := []string{"alpha", "Beta", "charlie", ".hidden"}
	for _, dir := range testDirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}
	}

	projects, err := getProjects(tempDir)
	if err != nil {
		t.Fatalf("getProjects failed: %v", err)
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

	// Should have exactly 3 projects (excluding .hidden)
	if len(projects) != 3 {
		t.Errorf("expected 3 projects, got %d: %v", len(projects), projects)
	}

	// Verify case-insensitive sort order: alpha, Beta, charlie
	expected := []string{"alpha", "Beta", "charlie"}
	for i, name := range expected {
		if i >= len(projects) || projects[i] != name {
			t.Errorf("expected %q at position %d, got %v", name, i, projects)
			break
		}
	}
}

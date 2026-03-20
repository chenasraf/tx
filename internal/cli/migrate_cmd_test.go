package cli

import (
	"strings"
	"testing"
)

func TestMigrateCmd_Exists(t *testing.T) {
	if migrateCmd == nil {
		t.Error("expected migrateCmd to not be nil")
	}

	if migrateCmd.Use != "migrate" {
		t.Errorf("unexpected Use: %q", migrateCmd.Use)
	}
}

func TestInjectIncludeIntoConfig_NoExistingConfig(t *testing.T) {
	content := `myproject:
  root: ~/Dev/myproject
`
	result := injectIncludeIntoConfig(content, "./local.yaml")

	// Should prepend .config with include
	if !strings.Contains(result, ".config:") {
		t.Error("expected .config section to be added")
	}
	if !strings.Contains(result, "include:") {
		t.Error("expected include key")
	}
	if !strings.Contains(result, "- ./local.yaml") {
		t.Error("expected include entry")
	}
}

func TestInjectIncludeIntoConfig_ExistingConfigNoInclude(t *testing.T) {
	content := `.config:
  shell: /bin/zsh

myproject:
  root: ~/Dev/myproject
`
	result := injectIncludeIntoConfig(content, "./local.yaml")

	if !strings.Contains(result, "include:") {
		t.Error("expected include key to be added")
	}
	if !strings.Contains(result, "- ./local.yaml") {
		t.Error("expected include entry")
	}
	// Should preserve existing config
	if !strings.Contains(result, "shell: /bin/zsh") {
		t.Error("expected existing shell config to be preserved")
	}
	if !strings.Contains(result, "myproject:") {
		t.Error("expected myproject to be preserved")
	}
}

func TestInjectIncludeIntoConfig_ExistingInclude(t *testing.T) {
	content := `.config:
  include:
    - ./existing.yaml
  shell: /bin/zsh

myproject:
  root: ~/Dev/myproject
`
	result := injectIncludeIntoConfig(content, "./local.yaml")

	if !strings.Contains(result, "- ./existing.yaml") {
		t.Error("expected existing include to be preserved")
	}
	if !strings.Contains(result, "- ./local.yaml") {
		t.Error("expected new include entry")
	}
}

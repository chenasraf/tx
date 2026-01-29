package exec

import (
	"os"
	"strings"
	"testing"
)

func TestLog_VerboseMode(t *testing.T) {
	// This test mainly ensures Log doesn't panic
	opts := Opts{Verbose: true, Dry: false}
	Log(opts, "test message", 123, "another")
}

func TestLog_DryMode(t *testing.T) {
	opts := Opts{Verbose: false, Dry: true}
	Log(opts, "test message")
}

func TestLog_Silent(t *testing.T) {
	opts := Opts{Verbose: false, Dry: false}
	// Should not print anything
	Log(opts, "test message")
}

func TestRunCommand_DryRun(t *testing.T) {
	opts := Opts{Verbose: false, Dry: true}

	err := RunCommand(opts, "echo hello")
	if err != nil {
		t.Errorf("expected no error in dry mode, got %v", err)
	}
}

func TestRunCommand_Success(t *testing.T) {
	opts := Opts{Verbose: false, Dry: false}

	err := RunCommand(opts, "true")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunCommand_Failure(t *testing.T) {
	opts := Opts{Verbose: false, Dry: false}

	err := RunCommand(opts, "false")
	if err == nil {
		t.Error("expected error for failing command")
	}
}

func TestGetCommandOutput_DryRun(t *testing.T) {
	opts := Opts{Verbose: false, Dry: true}

	output, code, err := GetCommandOutput(opts, "echo hello")
	if err != nil {
		t.Errorf("expected no error in dry mode, got %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0 in dry mode, got %d", code)
	}
	if output != "" {
		t.Errorf("expected empty output in dry mode, got %q", output)
	}
}

func TestGetCommandOutput_Success(t *testing.T) {
	opts := Opts{Verbose: false, Dry: false}

	output, code, err := GetCommandOutput(opts, "echo hello")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if strings.TrimSpace(output) != "hello" {
		t.Errorf("expected 'hello', got %q", output)
	}
}

func TestGetCommandOutput_NonZeroExit(t *testing.T) {
	opts := Opts{Verbose: false, Dry: false}

	_, code, err := GetCommandOutput(opts, "exit 42")
	if err != nil {
		t.Errorf("expected no error for non-zero exit, got %v", err)
	}
	if code != 42 {
		t.Errorf("expected exit code 42, got %d", code)
	}
}

func TestGetCommandOutput_WithPipe(t *testing.T) {
	opts := Opts{Verbose: false, Dry: false}

	output, code, err := GetCommandOutput(opts, "echo 'hello world' | tr ' ' '_'")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if strings.TrimSpace(output) != "hello_world" {
		t.Errorf("expected 'hello_world', got %q", output)
	}
}

func TestRunCommandSilent_Success(t *testing.T) {
	code, err := RunCommandSilent("true")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRunCommandSilent_Failure(t *testing.T) {
	code, err := RunCommandSilent("exit 1")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestRunCommandSilent_CustomExitCode(t *testing.T) {
	code, err := RunCommandSilent("exit 123")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if code != 123 {
		t.Errorf("expected exit code 123, got %d", code)
	}
}

func TestOpts(t *testing.T) {
	opts := Opts{
		Verbose: true,
		Dry:     true,
	}

	if !opts.Verbose {
		t.Error("expected Verbose to be true")
	}
	if !opts.Dry {
		t.Error("expected Dry to be true")
	}
}

func TestGetShell_Default(t *testing.T) {
	// Reset shell config
	oldShell := Shell
	Shell = ""
	defer func() { Shell = oldShell }()

	// Unset env var
	oldEnv := os.Getenv("SHELL")
	os.Unsetenv("SHELL")
	defer os.Setenv("SHELL", oldEnv)

	shell := getShell()
	// Should return one of the default shells or "sh"
	if shell == "" {
		t.Error("expected non-empty shell")
	}
}

func TestGetShell_EnvVar(t *testing.T) {
	// Reset shell config
	oldShell := Shell
	Shell = ""
	defer func() { Shell = oldShell }()

	// Set env var
	oldEnv := os.Getenv("SHELL")
	os.Setenv("SHELL", "/custom/shell")
	defer os.Setenv("SHELL", oldEnv)

	shell := getShell()
	if shell != "/custom/shell" {
		t.Errorf("expected /custom/shell, got %s", shell)
	}
}

func TestGetShell_ConfigOverridesEnv(t *testing.T) {
	// Set shell config (simulates .config section)
	oldShell := Shell
	Shell = "/configured/shell"
	defer func() { Shell = oldShell }()

	// Set env var too
	oldEnv := os.Getenv("SHELL")
	os.Setenv("SHELL", "/env/shell")
	defer os.Setenv("SHELL", oldEnv)

	shell := getShell()
	// Config should take priority over env
	if shell != "/configured/shell" {
		t.Errorf("expected /configured/shell, got %s", shell)
	}
}

func TestDefaultShells(t *testing.T) {
	if len(DefaultShells) == 0 {
		t.Error("expected DefaultShells to have entries")
	}
	// First should be zsh
	if DefaultShells[0] != "/bin/zsh" {
		t.Errorf("expected first shell to be /bin/zsh, got %s", DefaultShells[0])
	}
}

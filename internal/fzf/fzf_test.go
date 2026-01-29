package fzf

import (
	"errors"
	"testing"
)

func TestErrSelectionCancelled(t *testing.T) {
	if ErrSelectionCancelled.Error() != "selection cancelled" {
		t.Errorf("unexpected error message: %s", ErrSelectionCancelled.Error())
	}
}

func TestOptions(t *testing.T) {
	opts := Options{AllowCustom: true}
	if !opts.AllowCustom {
		t.Error("expected AllowCustom to be true")
	}

	opts2 := Options{AllowCustom: false}
	if opts2.AllowCustom {
		t.Error("expected AllowCustom to be false")
	}
}

func TestRun_EmptyInputs(t *testing.T) {
	_, err := Run([]string{}, Options{})
	if err == nil {
		t.Error("expected error for empty inputs")
	}
	if !errors.Is(err, ErrSelectionCancelled) {
		t.Errorf("expected ErrSelectionCancelled, got %v", err)
	}
}

func TestRunWithPreview_EmptyInputs(t *testing.T) {
	_, err := RunWithPreview([]string{}, func(i int) string { return "" })
	if err == nil {
		t.Error("expected error for empty inputs")
	}
	if !errors.Is(err, ErrSelectionCancelled) {
		t.Errorf("expected ErrSelectionCancelled, got %v", err)
	}
}

func TestErrSelectionCancelled_Is(t *testing.T) {
	err := ErrSelectionCancelled
	if !errors.Is(err, ErrSelectionCancelled) {
		t.Error("expected errors.Is to match ErrSelectionCancelled")
	}
}

// Note: Full integration tests for the fuzzy finder require:
// 1. A terminal environment
// 2. A way to simulate user input
//
// The Run and RunWithPreview functions are tested implicitly through
// CLI integration tests. Unit tests focus on edge cases and error handling.

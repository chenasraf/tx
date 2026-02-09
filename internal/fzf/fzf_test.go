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
	_, err := Run([]Item{}, Options{})
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

func TestItem(t *testing.T) {
	item := Item{Key: "my_session", Name: "my_session", Aliases: []string{"ms", "foo-session"}}
	if item.Key != "my_session" {
		t.Errorf("expected Key 'my_session', got %q", item.Key)
	}
	if item.Name != "my_session" {
		t.Errorf("expected Name 'my_session', got %q", item.Name)
	}
	if len(item.Aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(item.Aliases))
	}
}

func TestItem_NoAliases(t *testing.T) {
	item := Item{Key: "simple", Name: "simple"}
	if item.Key != item.Name {
		t.Error("expected Key and Name to be equal for items without aliases")
	}
	if len(item.Aliases) != 0 {
		t.Error("expected no aliases")
	}
}

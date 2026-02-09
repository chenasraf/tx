package fzf

import (
	"errors"

	"github.com/ktr0731/go-fuzzyfinder"
)

// ErrSelectionCancelled is returned when the user cancels selection
var ErrSelectionCancelled = errors.New("selection cancelled")

// Item represents a fuzzy finder item with a key and display string
type Item struct {
	Key     string
	Display string
}

// Options for fuzzy finder
type Options struct {
	AllowCustom bool // Note: go-fuzzyfinder doesn't support custom input like fzf --print-query
}

// Run executes the fuzzy finder with the given items and returns the selected key
func Run(items []Item, opts Options) (string, error) {
	if len(items) == 0 {
		return "", ErrSelectionCancelled
	}

	idx, err := fuzzyfinder.Find(
		items,
		func(i int) string {
			return items[i].Display
		},
	)

	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return "", ErrSelectionCancelled
		}
		return "", err
	}

	return items[idx].Key, nil
}

// RunWithPreview executes the fuzzy finder with a preview function
func RunWithPreview(items []Item, preview func(i int) string) (string, error) {
	if len(items) == 0 {
		return "", ErrSelectionCancelled
	}

	idx, err := fuzzyfinder.Find(
		items,
		func(i int) string {
			return items[i].Display
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i < 0 || i >= len(items) {
				return ""
			}
			return preview(i)
		}),
	)

	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return "", ErrSelectionCancelled
		}
		return "", err
	}

	return items[idx].Key, nil
}

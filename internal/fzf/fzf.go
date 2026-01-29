package fzf

import (
	"errors"

	"github.com/ktr0731/go-fuzzyfinder"
)

// ErrSelectionCancelled is returned when the user cancels selection
var ErrSelectionCancelled = errors.New("selection cancelled")

// Options for fuzzy finder
type Options struct {
	AllowCustom bool // Note: go-fuzzyfinder doesn't support custom input like fzf --print-query
}

// Run executes the fuzzy finder with the given inputs and returns the selected value
func Run(inputs []string, opts Options) (string, error) {
	if len(inputs) == 0 {
		return "", ErrSelectionCancelled
	}

	idx, err := fuzzyfinder.Find(
		inputs,
		func(i int) string {
			return inputs[i]
		},
	)

	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return "", ErrSelectionCancelled
		}
		return "", err
	}

	return inputs[idx], nil
}

// RunWithPreview executes the fuzzy finder with a preview function
func RunWithPreview(inputs []string, preview func(i int) string) (string, error) {
	if len(inputs) == 0 {
		return "", ErrSelectionCancelled
	}

	idx, err := fuzzyfinder.Find(
		inputs,
		func(i int) string {
			return inputs[i]
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i < 0 || i >= len(inputs) {
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

	return inputs[idx], nil
}

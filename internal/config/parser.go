package config

import (
	"os"
	"path/filepath"
	"strings"
)

// dirFix expands ~ to home directory
func dirFix(dir string) string {
	if strings.HasPrefix(dir, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, dir[1:])
		}
	}
	return dir
}

// NameFix strips file extensions from names (e.g., "foo.bar" -> "foo")
// Matches TypeScript: name.split('.').filter(Boolean)[0]
func NameFix(name string) string {
	if name == "" {
		return name
	}
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		// Filter out empty strings and return first non-empty
		for _, part := range parts {
			if part != "" {
				return part
			}
		}
	}
	return name
}

// ParseConfig parses a raw config item into a resolved ParsedTmuxConfigItem
func ParseConfig(key string, item TmuxConfigItemInput) ParsedTmuxConfigItem {
	root := dirFix(item.Root)

	name := item.Name
	if name == "" {
		name = key
	}
	if name == "" {
		name = filepath.Base(root)
	}

	windows := item.Windows
	if len(windows) == 0 || item.BlankWindow {
		// Add default window at the beginning
		defaultWindow := TmuxWindowInput{
			Window: &TmuxWindow{
				Name: name,
				Cwd:  root,
				Layout: &TmuxLayoutInput{
					PaneLayout: &DefaultEmptyLayout,
				},
			},
		}
		windows = append([]TmuxWindowInput{defaultWindow}, windows...)
	}

	parsedWindows := make([]ParsedTmuxWindow, 0, len(windows))
	for _, w := range windows {
		parsedWindows = append(parsedWindows, parseWindow(w, root))
	}

	return ParsedTmuxConfigItem{
		Name:    name,
		Root:    root,
		Windows: parsedWindows,
	}
}

// parseWindow parses a TmuxWindowInput into a ParsedTmuxWindow
func parseWindow(w TmuxWindowInput, root string) ParsedTmuxWindow {
	if w.IsString {
		// Window is just a directory path
		resolvedCwd := dirFix(resolvePath(root, w.String))
		return ParsedTmuxWindow{
			Name:   NameFix(filepath.Base(resolvedCwd)),
			Cwd:    resolvedCwd,
			Layout: parseLayoutWithCwd(&TmuxLayoutInput{PaneLayout: &DefaultEmptyLayout}, resolvedCwd),
		}
	}

	if w.Window == nil {
		// Fallback to default
		return ParsedTmuxWindow{
			Name:   NameFix(filepath.Base(root)),
			Cwd:    root,
			Layout: parseLayoutWithCwd(&TmuxLayoutInput{PaneLayout: &DefaultEmptyLayout}, root),
		}
	}

	// Window is a struct
	resolvedCwd := dirFix(resolvePath(root, w.Window.Cwd))
	windowName := w.Window.Name
	if windowName == "" {
		windowName = NameFix(filepath.Base(resolvedCwd))
	}

	return ParsedTmuxWindow{
		Name:   windowName,
		Cwd:    resolvedCwd,
		Layout: parseLayout(w.Window.Layout, resolvedCwd),
	}
}

// parseLayout parses a TmuxLayoutInput into a TmuxPaneLayout
func parseLayout(layoutInput *TmuxLayoutInput, root string) TmuxPaneLayout {
	if layoutInput == nil {
		return TmuxPaneLayout{
			Cwd:   resolvePath(root, "."),
			Zoom:  DefaultEmptyLayout.Zoom,
			Split: copyTmuxSplitLayout(DefaultEmptyLayout.Split, root),
		}
	}

	if layoutInput.IsString {
		return TmuxPaneLayout{
			Cwd: resolvePath(root, layoutInput.String),
			Cmd: DefaultEmptyPane.Cmd,
		}
	}

	if layoutInput.IsArray {
		// Build split chain from array
		var split *TmuxSplitLayout
		for i := len(layoutInput.Array) - 1; i >= 0; i-- {
			cwd := resolvePath(root, layoutInput.Array[i])
			split = &TmuxSplitLayout{
				Direction: "h",
				Child: &TmuxPaneLayout{
					Cwd:   cwd,
					Split: split,
				},
			}
		}

		baseLayout := parseLayout(&TmuxLayoutInput{PaneLayout: &DefaultEmptyLayout}, root)
		baseLayout.Split = split
		return baseLayout
	}

	if layoutInput.PaneLayout != nil {
		return parsePaneLayout(layoutInput.PaneLayout, root)
	}

	// Default fallback
	return TmuxPaneLayout{
		Cwd: resolvePath(root, "."),
	}
}

// parseLayoutWithCwd is like parseLayout but sets the cwd on the result
func parseLayoutWithCwd(layoutInput *TmuxLayoutInput, cwd string) TmuxPaneLayout {
	layout := parseLayout(layoutInput, cwd)
	layout.Cwd = cwd
	return layout
}

// parsePaneLayout parses a TmuxPaneLayout resolving paths
func parsePaneLayout(pane *TmuxPaneLayout, root string) TmuxPaneLayout {
	result := TmuxPaneLayout{
		Cwd:  resolvePath(root, pane.Cwd),
		Cmd:  pane.Cmd,
		Zoom: pane.Zoom,
	}

	if pane.Split != nil {
		result.Split = &TmuxSplitLayout{
			Direction: pane.Split.Direction,
		}
		if result.Split.Direction == "" {
			result.Split.Direction = "h"
		}
		if pane.Split.Child != nil {
			child := parsePaneLayout(pane.Split.Child, resolvePath(root, pane.Cwd))
			result.Split.Child = &child
		}
	}

	return result
}

// copyTmuxSplitLayout creates a deep copy of a TmuxSplitLayout with resolved paths
func copyTmuxSplitLayout(split *TmuxSplitLayout, root string) *TmuxSplitLayout {
	if split == nil {
		return nil
	}
	result := &TmuxSplitLayout{
		Direction: split.Direction,
	}
	if split.Child != nil {
		child := TmuxPaneLayout{
			Cwd:   resolvePath(root, split.Child.Cwd),
			Cmd:   split.Child.Cmd,
			Zoom:  split.Child.Zoom,
			Split: copyTmuxSplitLayout(split.Child.Split, root),
		}
		result.Child = &child
	}
	return result
}

// resolvePath resolves a path relative to root, or returns path if absolute
func resolvePath(root, path string) string {
	if path == "" {
		path = "."
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

package config

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// GlobalConfig holds global settings from the .config section
type GlobalConfig struct {
	Shell         string                     `yaml:"shell,omitempty"`
	ProjectsPath  string                     `yaml:"projects_path,omitempty"`
	DefaultLayout *TmuxPaneLayout            `yaml:"default_layout,omitempty"`
	NamedLayouts  map[string]*TmuxPaneLayout `yaml:"named_layouts,omitempty"`
	InitialWindow *int                       `yaml:"initial_window,omitempty"`
	Include       []string                   `yaml:"include,omitempty"`
	Picker        *PickerConfig              `yaml:"picker,omitempty"`
}

// PickerConfig holds settings for the session and agent pickers
type PickerConfig struct {
	// Size is the popup size: 0 for borderless full screen, 1-100 for a bordered popup at that
	// percentage of the terminal.
	Size *int `yaml:"size,omitempty"`
	// Refresh is how often the preview re-captures pane contents, as a Go duration ("1s").
	Refresh      string              `yaml:"refresh,omitempty"`
	PreviewWidth PickerPreviewWidths `yaml:"preview_width,omitempty"`
	Agents       PickerAgents        `yaml:"agents,omitempty"`
	// Colors overrides theme colours by role (sky, blue, peach, ...) with #rrggbb values.
	Colors map[string]string `yaml:"colors,omitempty"`
}

// PickerPreviewWidths is each picker's preview width, as a percentage of the popup width
type PickerPreviewWidths struct {
	Sessions int `yaml:"sessions,omitempty"`
	Agents   int `yaml:"agents,omitempty"`
}

// PickerAgents selects which panes the agent picker lists
type PickerAgents struct {
	// Commands are process names, or /regex/ patterns matched against the full command line.
	Commands []string `yaml:"commands,omitempty"`
	// Match is auto (names against the process name, patterns against the command line), name, or
	// args.
	Match string `yaml:"match,omitempty"`
}

// ConfigFile represents the top-level config file: map of session name -> config
type ConfigFile map[string]TmuxConfigItemInput

// Get performs a lookup of a key in the config file.
// It first tries exact key match, then case-insensitive key match,
// then matches against aliases (case-insensitive).
// It returns the config item, the actual key as stored in the config, and whether it was found.
func (c ConfigFile) Get(key string) (TmuxConfigItemInput, string, bool) {
	// Try exact match first
	if item, ok := c[key]; ok {
		return item, key, true
	}
	// Fall back to case-insensitive match
	lower := strings.ToLower(key)
	for k, v := range c {
		if strings.ToLower(k) == lower {
			return v, k, true
		}
	}
	// Fall back to alias match (case-insensitive)
	for k, v := range c {
		for _, alias := range v.Aliases {
			if strings.ToLower(alias) == lower {
				return v, k, true
			}
		}
	}
	return TmuxConfigItemInput{}, "", false
}

// TmuxConfigItemInput represents a single tmux session configuration
type TmuxConfigItemInput struct {
	Root          string            `yaml:"root"`
	Name          string            `yaml:"name,omitempty"`
	Aliases       []string          `yaml:"aliases,omitempty"`
	BlankWindow   bool              `yaml:"blank_window,omitempty"`
	InitialWindow *int              `yaml:"initial_window,omitempty"`
	Windows       []TmuxWindowInput `yaml:"windows,omitempty"`
}

// TmuxWindowInput can be either a string (directory path) or a TmuxWindow struct
type TmuxWindowInput struct {
	IsString bool
	String   string
	Window   *TmuxWindow
}

// UnmarshalYAML implements custom unmarshaling for TmuxWindowInput
func (w *TmuxWindowInput) UnmarshalYAML(value *yaml.Node) error {
	// Try string first
	var s string
	if err := value.Decode(&s); err == nil {
		w.IsString = true
		w.String = s
		return nil
	}

	// Try TmuxWindow struct
	var window TmuxWindow
	if err := value.Decode(&window); err == nil {
		w.IsString = false
		w.Window = &window
		return nil
	}

	return nil
}

// TmuxWindow represents a window configuration with name, cwd, and optional layout
type TmuxWindow struct {
	Name   string           `yaml:"name,omitempty"`
	Cwd    string           `yaml:"cwd"`
	Layout *TmuxLayoutInput `yaml:"layout,omitempty"`
}

// TmuxLayoutInput can be a string, array of strings, or TmuxPaneLayout
type TmuxLayoutInput struct {
	IsString   bool
	String     string
	IsArray    bool
	Array      []string
	PaneLayout *TmuxPaneLayout
}

// UnmarshalYAML implements custom unmarshaling for TmuxLayoutInput
func (l *TmuxLayoutInput) UnmarshalYAML(value *yaml.Node) error {
	// Try string first
	var s string
	if err := value.Decode(&s); err == nil {
		l.IsString = true
		l.String = s
		return nil
	}

	// Try array of strings
	var arr []string
	if err := value.Decode(&arr); err == nil {
		l.IsArray = true
		l.Array = arr
		return nil
	}

	// Try TmuxPaneLayout struct
	var pane TmuxPaneLayout
	if err := value.Decode(&pane); err == nil {
		l.IsString = false
		l.IsArray = false
		l.PaneLayout = &pane
		return nil
	}

	return nil
}

// TmuxPaneLayout represents a pane configuration
type TmuxPaneLayout struct {
	Cwd   string           `yaml:"cwd"`
	Cmd   string           `yaml:"cmd,omitempty"`
	Zoom  bool             `yaml:"zoom,omitempty"`
	Clock bool             `yaml:"clock,omitempty"`
	Split *TmuxSplitLayout `yaml:"split,omitempty"`
}

// TmuxSplitLayout represents a split configuration
type TmuxSplitLayout struct {
	Direction string          `yaml:"direction"`      // "h" or "v"
	Size      int             `yaml:"size,omitempty"` // percentage (1-100) of the split given to the child pane
	Child     *TmuxPaneLayout `yaml:"child"`
}

// ParsedTmuxConfigItem is the resolved/parsed version of TmuxConfigItemInput
type ParsedTmuxConfigItem struct {
	Name          string
	Root          string
	InitialWindow int
	Windows       []ParsedTmuxWindow
}

// ParsedTmuxWindow is the resolved/parsed version of a window
type ParsedTmuxWindow struct {
	Name   string
	Cwd    string
	Layout TmuxPaneLayout
}

// DefaultEmptyPane is the default empty pane configuration
var DefaultEmptyPane = TmuxPaneLayout{
	Cwd: ".",
	Cmd: "",
}

// ConfiguredDefaultLayout holds the user-configured default layout (set from .config)
var ConfiguredDefaultLayout *TmuxPaneLayout

// ConfiguredNamedLayouts holds user-configured named layouts (set from .config)
var ConfiguredNamedLayouts map[string]*TmuxPaneLayout

// ConfiguredInitialWindow holds the user-configured default initial window (set from .config)
var ConfiguredInitialWindow *int

// GetDefaultLayout returns the configured default layout or the hardcoded default
func GetDefaultLayout() *TmuxPaneLayout {
	if ConfiguredDefaultLayout != nil {
		return ConfiguredDefaultLayout
	}
	return &DefaultEmptyLayout
}

// GetNamedLayout returns a named layout by name, or nil if not found
func GetNamedLayout(name string) *TmuxPaneLayout {
	if ConfiguredNamedLayouts == nil {
		return nil
	}
	return ConfiguredNamedLayouts[name]
}

// DefaultEmptyLayout is the default layout with horizontal and vertical splits
var DefaultEmptyLayout = TmuxPaneLayout{
	Cwd:  ".",
	Cmd:  "",
	Zoom: false,
	Split: &TmuxSplitLayout{
		Direction: "h",
		Child: &TmuxPaneLayout{
			Cwd: ".",
			Split: &TmuxSplitLayout{
				Direction: "v",
				Child: &TmuxPaneLayout{
					Cwd:   ".",
					Clock: true,
				},
			},
		},
	},
}

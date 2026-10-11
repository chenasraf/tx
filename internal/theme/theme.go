// Package theme holds the colours the pickers draw with.
//
// Colours come from config overrides first, then the @thm_* options the Catppuccin tmux theme
// sets, then a built-in Catppuccin Mocha palette, so the pickers look right without the theme.
package theme

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chenasraf/tx/internal/tmuxctl"
)

// Theme maps each colour role to a #rrggbb value.
type Theme map[string]string

// Names are the colour roles, as their @thm_ option names end.
var Names = []string{"sky", "overlay_0", "overlay_1", "blue", "green", "subtext_0", "text", "peach", "red"}

// Mocha is the fallback palette.
var Mocha = Theme{
	"sky":       "#89dceb",
	"overlay_0": "#6c7086",
	"overlay_1": "#7f849c",
	"blue":      "#89b4fa",
	"green":     "#a6e3a1",
	"subtext_0": "#a6adc8",
	"text":      "#cdd6f4",
	"peach":     "#fab387",
	"red":       "#f38ba8",
}

// Query is the command that reads every @thm_ option in one line.
func Query() []string {
	parts := make([]string, len(Names))
	for i, n := range Names {
		parts[i] = "#{@thm_" + n + "}"
	}
	return []string{"display-message", "-p", strings.Join(parts, "\t")}
}

// Parse builds a theme from Query's output line.
func Parse(line string, overrides map[string]string) Theme {
	values := strings.Split(line, "\t")
	t := Theme{}
	for i, n := range Names {
		switch {
		case overrides[n] != "":
			t[n] = overrides[n]
		case i < len(values) && values[i] != "":
			t[n] = values[i]
		default:
			t[n] = Mocha[n]
		}
	}
	return t
}

// Load reads the theme from the server.
func Load(r tmuxctl.Runner, overrides map[string]string) (Theme, error) {
	res, err := r.Run(Query())
	if err != nil {
		return nil, err
	}
	return Parse(res[0].Text(), overrides), nil
}

// Fg is the SGR sequence for role's colour as the foreground, or just bold (or nothing) when the
// colour isn't a #rrggbb value.
func (t Theme) Fg(role string, bold bool) string {
	return Fg(t[role], bold)
}

// Fg is the SGR sequence for a #rrggbb foreground.
func Fg(hex string, bold bool) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		if bold {
			return "\x1b[1m"
		}
		return ""
	}
	prefix := ""
	if bold {
		prefix = "1;"
	}
	return fmt.Sprintf("\x1b[%s38;2;%d;%d;%dm", prefix, r, g, b)
}

func parseHex(hex string) (r, g, b uint64, ok bool) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0, false
	}
	var err error
	if r, err = strconv.ParseUint(hex[1:3], 16, 8); err != nil {
		return 0, 0, 0, false
	}
	if g, err = strconv.ParseUint(hex[3:5], 16, 8); err != nil {
		return 0, 0, 0, false
	}
	if b, err = strconv.ParseUint(hex[5:7], 16, 8); err != nil {
		return 0, 0, 0, false
	}
	return r, g, b, true
}

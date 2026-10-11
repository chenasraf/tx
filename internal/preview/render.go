// Package preview draws a tmux window's real pane layout and contents, scaled into a given area,
// without switching any client to it.
package preview

import (
	"fmt"
	"math"
	"strings"

	"github.com/chenasraf/tx/internal/theme"
)

// Pane is one pane's geometry and captured contents (capture-pane -p -e lines).
type Pane struct {
	ID, Index                string
	Left, Top, Width, Height int
	Active, Floating         bool
	Alternate                bool
	Command                  string
	Content                  []string
}

// Tab is one of the session's windows, for the header.
type Tab struct {
	ID, Index, Name string
}

// Window is everything the window preview draws.
type Window struct {
	ID            string
	Width, Height int
	Tabs          []Tab
	// Panes holds only the visible panes: a zoomed window's active pane, at full size.
	Panes []Pane
}

// Single is a single pane under a header naming where it lives.
type Single struct {
	Session, WindowIndex, WindowName, Path string
	Height                                 int
	Content                                []string
}

const (
	up    = 1
	right = 2
	down  = 4
	left  = 8
)

var junctions = map[int]string{
	right | left:             "─",
	up | down:                "│",
	right | down:             "╭",
	left | down:              "╮",
	up | right:               "╰",
	up | left:                "╯",
	up | right | down:        "├",
	up | left | down:         "┤",
	left | right | down:      "┬",
	up | left | right:        "┴",
	up | right | down | left: "┼",
	right:                    "─",
	left:                     "─",
	up:                       "│",
	down:                     "│",
}

type anchor int

const (
	anchorAuto anchor = iota
	anchorBottom
)

// drawContents fills the inside of box (x0, y0)-(x1, y1) with p's contents.
func drawContents(c *canvas, p Pane, x0, y0, x1, y1 int, a anchor) {
	iw, ih := x1-x0-1, y1-y0-1
	if iw <= 0 || ih <= 0 {
		return
	}
	lines := p.Content
	if len(lines) > p.Height {
		lines = lines[:max(p.Height, 0)]
	}
	cells := parseCells(lines)

	// full-screen apps (editors, TUIs) read best from the top; shells from the most recent output,
	// just above the prompt
	fromBottom := a == anchorBottom || !p.Alternate
	var shown [][]cell
	if fromBottom {
		for len(cells) > 0 && isBlank(cells[len(cells)-1]) {
			cells = cells[:len(cells)-1]
		}
		shown = cells[max(len(cells)-ih, 0):]
	} else {
		shown = cells[:min(ih, len(cells))]
	}

	for dy, row := range shown {
		row = row[:min(iw, len(row))]
		for dx, cl := range row {
			if cl.ch != "" && dx == iw-1 && charWidth(firstRune(cl.ch)) == 2 {
				cl.ch = " "
			}
			c.put(x0+1+dx, y0+1+dy, cl.ch, cl.style)
		}
		for dx := len(row); dx < iw; dx++ {
			c.put(x0+1+dx, y0+1+dy, " ", "")
		}
	}
}

// RenderWindow draws w's panes, scaled into cols x lines, under a header listing the session's
// windows.
func RenderWindow(w Window, t theme.Theme, cols, lines int) string {
	c := newCanvas(cols, lines)

	x := 0
	for _, tab := range w.Tabs {
		label := fmt.Sprintf(" %s %s ", tab.Index, tab.Name)
		style := t.Fg("overlay_1", false)
		if tab.ID == w.ID {
			style = t.Fg("sky", true)
		}
		c.text(x, 0, label, style, -1)
		x += runeLen(label) + 1
	}

	// the layout area sits below the header; window cell (x, y) has its borders at x and x+width+1
	const top = 1
	areaW, areaH := cols, lines-1
	sx := float64(areaW-1) / float64(w.Width+1)
	sy := float64(areaH-1) / float64(w.Height+1)
	box := func(p Pane) (int, int, int, int) {
		return round(float64(p.Left) * sx),
			top + round(float64(p.Top)*sy),
			round(float64(p.Left+p.Width+1) * sx),
			top + round(float64(p.Top+p.Height+1)*sy)
	}

	border := t.Fg("overlay_0", false)
	activeBorder := t.Fg("sky", false)
	label := func(p Pane, x0, y0, x1 int, style string) {
		if room := x1 - x0 - 3; room >= 3 {
			c.text(x0+2, y0, fmt.Sprintf(" %s %s ", p.Index, p.Command), style, room)
		}
	}

	var tiled, floating []Pane
	for _, p := range w.Panes {
		if p.Floating {
			floating = append(floating, p)
		} else {
			tiled = append(tiled, p)
		}
	}

	// tiled panes share borders, so their edges merge into proper junctions
	edges := map[[2]int]int{}
	add := func(x, y, bit int) { edges[[2]int{x, y}] |= bit }
	for _, p := range tiled {
		x0, y0, x1, y1 := box(p)
		for x := x0; x < x1; x++ {
			for _, y := range []int{y0, y1} {
				add(x, y, right)
				add(x+1, y, left)
			}
		}
		for y := y0; y < y1; y++ {
			for _, x := range []int{x0, x1} {
				add(x, y, down)
				add(x, y+1, up)
			}
		}
	}
	for pos, mask := range edges {
		ch, ok := junctions[mask]
		if !ok {
			ch = " "
		}
		c.put(pos[0], pos[1], ch, border)
	}

	for _, p := range tiled {
		x0, y0, x1, y1 := box(p)
		drawContents(c, p, x0, y0, x1, y1, anchorAuto)
	}
	for _, p := range tiled {
		if !p.Active {
			continue
		}
		x0, y0, x1, y1 := box(p)
		for _, pos := range perimeter(x0, y0, x1, y1) {
			if c.inside(pos[0], pos[1]) {
				c.cells[pos[1]][pos[0]].style = activeBorder
			}
		}
	}
	for _, p := range tiled {
		x0, y0, x1, _ := box(p)
		style := t.Fg("subtext_0", false)
		if p.Active {
			style = activeBorder
		}
		label(p, x0, y0, x1, style)
	}

	// floating panes sit on top and own their whole box
	for _, p := range floating {
		x0, y0, x1, y1 := box(p)
		style := border
		if p.Active {
			style = activeBorder
		}
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				c.put(x, y, " ", "")
			}
		}
		c.box(x0, y0, x1, y1, style)
		drawContents(c, p, x0, y0, x1, y1, anchorAuto)
		label(p, x0, y0, x1, style)
	}

	return c.render()
}

// RenderSingle draws one pane, bottom-anchored, under a header naming where it lives. Paths under
// home are shown with ~.
func RenderSingle(s Single, home string, t theme.Theme, cols, lines int) string {
	path := s.Path
	if home != "" && (path == home || strings.HasPrefix(path, home+"/")) {
		path = "~" + path[len(home):]
	}

	c := newCanvas(cols, lines)
	c.text(0, 0, " "+s.Session, t.Fg("blue", true), -1)
	c.text(runeLen(s.Session)+2, 0, s.WindowIndex+" "+s.WindowName, t.Fg("overlay_1", false), -1)
	c.text(runeLen(s.Session)+runeLen(s.WindowIndex)+runeLen(s.WindowName)+4, 0, path, t.Fg("overlay_0", false), -1)
	c.box(0, 1, cols-1, lines-1, t.Fg("sky", false))
	// agent TUIs keep their prompt and latest output at the bottom
	p := Pane{Height: s.Height, Alternate: true, Content: s.Content}
	drawContents(c, p, 0, 1, cols-1, lines-1, anchorBottom)
	return c.render()
}

func perimeter(x0, y0, x1, y1 int) [][2]int {
	var out [][2]int
	for x := x0; x <= x1; x++ {
		out = append(out, [2]int{x, y0}, [2]int{x, y1})
	}
	for y := y0 + 1; y < y1; y++ {
		out = append(out, [2]int{x0, y}, [2]int{x1, y})
	}
	return out
}

// round halves to even, so layouts scale the same as Python's round()
func round(f float64) int {
	return int(math.RoundToEven(f))
}

func runeLen(s string) int {
	return len([]rune(s))
}

package preview

import "strings"

type canvas struct {
	w, h  int
	cells [][]cell
}

func newCanvas(w, h int) *canvas {
	c := &canvas{w: max(w, 0), h: max(h, 0)}
	c.cells = make([][]cell, c.h)
	for y := range c.cells {
		row := make([]cell, c.w)
		for x := range row {
			row[x] = cell{ch: " "}
		}
		c.cells[y] = row
	}
	return c
}

func (c *canvas) inside(x, y int) bool {
	return x >= 0 && x < c.w && y >= 0 && y < c.h
}

func (c *canvas) put(x, y int, ch, style string) {
	if c.inside(x, y) {
		c.cells[y][x] = cell{ch: ch, style: style}
	}
}

// text writes s one code point per cell, cut to limit code points when limit >= 0.
func (c *canvas) text(x, y int, s, style string, limit int) {
	i := 0
	for _, r := range s {
		if limit >= 0 && i >= limit {
			break
		}
		c.put(x+i, y, string(r), style)
		i++
	}
}

func (c *canvas) box(x0, y0, x1, y1 int, style string) {
	for x := x0 + 1; x < x1; x++ {
		c.put(x, y0, "─", style)
		c.put(x, y1, "─", style)
	}
	for y := y0 + 1; y < y1; y++ {
		c.put(x0, y, "│", style)
		c.put(x1, y, "│", style)
	}
	c.put(x0, y0, "╭", style)
	c.put(x1, y0, "╮", style)
	c.put(x0, y1, "╰", style)
	c.put(x1, y1, "╯", style)
}

func (c *canvas) render() string {
	var b strings.Builder
	for y, row := range c.cells {
		if y > 0 {
			b.WriteByte('\n')
		}
		cur, started := "", false
		for i, cl := range row {
			if cl.ch == "" {
				continue
			}
			ch := cl.ch
			// a wide char whose right half got overwritten (e.g. by a border) would push the line
			if i+1 < len(row) && row[i+1].ch != "" && charWidth(firstRune(ch)) == 2 {
				ch = " "
			}
			if !started || cl.style != cur {
				b.WriteString("\x1b[0m")
				b.WriteString(cl.style)
				cur, started = cl.style, true
			}
			b.WriteString(ch)
		}
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

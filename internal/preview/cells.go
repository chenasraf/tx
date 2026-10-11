package preview

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/width"
)

// cell is one terminal column. An empty ch marks the right half of a wide character.
type cell struct {
	ch    string
	style string
}

const (
	sgrPattern   = `\x1b\[([0-9;:]*)m`
	otherPattern = `\x1b(?:\[[0-9;:?<>=]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\)|[()][0-9A-Za-z]|[=>])`
)

var anyEscape = regexp.MustCompile(sgrPattern + "|" + otherPattern)

// charWidth is the number of columns r takes: 0 for combining marks, 2 for East Asian wide and
// fullwidth characters.
func charWidth(r rune) int {
	if norm.NFD.PropertiesString(string(r)).CCC() != 0 {
		return 0
	}
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	}
	return 1
}

func firstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

// parseCells turns capture-pane -e output into rows of styled cells. Non-SGR escapes are dropped.
//
// capture-pane carries SGR state across lines, so style does too; it accumulates sequences until
// a reset rather than interpreting them.
func parseCells(lines []string) [][]cell {
	style := ""
	out := make([][]cell, 0, len(lines))
	for _, line := range lines {
		var cells []cell
		pos := 0
		// a sentinel reset at the end flushes the text after the last escape
		padded := line + "\x1b[m"
		for _, m := range anyEscape.FindAllStringSubmatchIndex(padded, -1) {
			for _, r := range strings.ReplaceAll(padded[pos:m[0]], "\t", " ") {
				w := charWidth(r)
				if w == 0 {
					if n := len(cells); n > 0 && cells[n-1].ch != "" {
						cells[n-1].ch += string(r)
					}
					continue
				}
				cells = append(cells, cell{ch: string(r), style: style})
				if w == 2 {
					cells = append(cells, cell{style: style})
				}
			}
			pos = m[1]
			if m[0] >= len(line) {
				break
			}
			if m[2] >= 0 {
				if params := padded[m[2]:m[3]]; params == "" || params == "0" {
					style = ""
				} else {
					style += padded[m[0]:m[1]]
				}
			}
		}
		out = append(out, cells)
	}
	return out
}

func isBlank(cells []cell) bool {
	for _, c := range cells {
		if c.ch != "" && strings.TrimFunc(c.ch, unicode.IsSpace) != "" {
			return false
		}
	}
	return true
}

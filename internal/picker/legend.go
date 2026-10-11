package picker

import (
	"strings"

	"github.com/chenasraf/tx/internal/theme"
)

// key is one legend entry: the keys, then what they do.
type key struct {
	keys, desc string
}

const (
	legendGap    = 4
	legendMargin = 2
)

func (k key) size() int {
	return runeLen(k.keys) + 1 + runeLen(k.desc)
}

func (k key) render(t theme.Theme, pad int) string {
	return t.Fg("sky", false) + k.keys + reset + " " + t.Fg("overlay_1", false) + k.desc + reset + strings.Repeat(" ", pad)
}

// legendLines lays out rows of keys for width columns. Entries line up in columns as long as the
// widest row fits; otherwise each row wraps on its own.
func legendLines(rows [][]key, width int, t theme.Theme) []string {
	ncols := 0
	for _, row := range rows {
		ncols = max(ncols, len(row))
	}
	cols := make([]int, ncols)
	for _, row := range rows {
		for i, k := range row {
			cols[i] = max(cols[i], k.size())
		}
	}
	total := legendMargin + legendGap*(ncols-1)
	for _, c := range cols {
		total += c
	}

	margin := strings.Repeat(" ", legendMargin)
	gap := strings.Repeat(" ", legendGap)
	var lines []string
	if total <= width {
		for _, row := range rows {
			parts := make([]string, len(row))
			for i, k := range row {
				pad := 0
				if i < len(row)-1 {
					pad = cols[i] - k.size()
				}
				parts[i] = k.render(t, pad)
			}
			lines = append(lines, margin+strings.Join(parts, gap))
		}
		return lines
	}

	for _, row := range rows {
		var line []string
		used := legendMargin
		for _, k := range row {
			w := k.size()
			if len(line) > 0 {
				w += legendGap
			}
			if len(line) > 0 && used+w > width {
				lines = append(lines, margin+strings.Join(line, gap))
				line, used, w = nil, legendMargin, k.size()
			}
			line = append(line, k.render(t, 0))
			used += w
		}
		lines = append(lines, margin+strings.Join(line, gap))
	}
	return lines
}

// armedLine replaces the legend while a kill waits for its second d.
func armedLine(what string, t theme.Theme) string {
	return "  " + t.Fg("red", true) + "d" + reset + " " + t.Fg("text", false) + "again to kill" + reset +
		" " + t.Fg("peach", true) + what + reset
}

func runeLen(s string) int {
	return len([]rune(s))
}

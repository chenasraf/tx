// Package table renders simple bordered text tables with headers.
package table

import (
	"strings"
	"unicode/utf8"
)

// Render returns a bordered table string with the given headers and rows.
// Every line in the output is prefixed with indent. Column widths auto-size
// to the widest cell (header included).
func Render(headers []string, rows [][]string, indent string) string {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if w := utf8.RuneCountInString(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	border := func(left, mid, right string) string {
		parts := make([]string, len(widths))
		for i, w := range widths {
			parts[i] = strings.Repeat("─", w+2)
		}
		return left + strings.Join(parts, mid) + right
	}
	rowLine := func(cells []string) string {
		parts := make([]string, len(widths))
		for i := range widths {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			pad := max(widths[i]-utf8.RuneCountInString(cell), 0)
			parts[i] = " " + cell + strings.Repeat(" ", pad) + " "
		}
		return "│" + strings.Join(parts, "│") + "│"
	}

	var b strings.Builder
	b.WriteString(indent + border("┌", "┬", "┐") + "\n")
	b.WriteString(indent + rowLine(headers) + "\n")
	b.WriteString(indent + border("├", "┼", "┤") + "\n")
	for _, row := range rows {
		b.WriteString(indent + rowLine(row) + "\n")
	}
	b.WriteString(indent + border("└", "┴", "┘") + "\n")
	return b.String()
}

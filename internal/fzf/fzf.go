package fzf

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrSelectionCancelled is returned when the user cancels selection
var ErrSelectionCancelled = errors.New("selection cancelled")

// Item represents a fuzzy finder item.
type Item struct {
	Key     string
	Name    string
	Aliases []string
}

// Options for fuzzy finder
type Options struct {
	AllowCustom bool
}

// Styles used for rendering.
var (
	normalStyle   = lipgloss.NewStyle()
	dimStyle      = lipgloss.NewStyle().Faint(true)
	matchStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	dimMatchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Faint(true)
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	promptStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

type model struct {
	items     []Item
	filtered  []filteredItem
	query     string
	cursor    int // index in filtered
	offset    int // scroll offset
	width     int
	height    int
	selected  string
	cancelled bool
	quitting  bool
}

func initialModel(items []Item) model {
	m := model{
		items:  items,
		width:  80,
		height: 24,
	}
	m.filtered = filterAndSort(items, "")
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampScroll()
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = m.filtered[m.cursor].item.Key
			} else {
				m.cancelled = true
			}
			m.quitting = true
			return m, tea.Quit

		case tea.KeyBackspace:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.refilter()
			}
			return m, nil

		case tea.KeyCtrlU:
			m.query = ""
			m.refilter()
			return m, nil

		case tea.KeyUp, tea.KeyCtrlK:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.clampScroll()
			}
			return m, nil

		case tea.KeyDown, tea.KeyCtrlJ:
			if m.cursor > 0 {
				m.cursor--
				m.clampScroll()
			}
			return m, nil

		case tea.KeyRunes:
			m.query += string(msg.Runes)
			m.refilter()
			return m, nil
		}
	}

	return m, nil
}

func (m *model) refilter() {
	m.filtered = filterAndSort(m.items, m.query)
	m.cursor = 0
	m.offset = 0
}

func (m *model) clampScroll() {
	// Available rows for items (height minus prompt row)
	maxVisible := m.height - 1
	if maxVisible < 1 {
		maxVisible = 1
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+maxVisible {
		m.offset = m.cursor - maxVisible + 1
	}
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	// Available rows for items (height minus prompt row)
	maxVisible := m.height - 1
	if maxVisible < 1 {
		maxVisible = 1
	}

	end := m.offset + maxVisible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	// Build item rows (reversed: highest index on top, lowest near prompt)
	var rows []string
	for i := m.offset; i < end; i++ {
		fi := m.filtered[i]
		rows = append(rows, m.renderItem(fi, i == m.cursor))
	}

	var b strings.Builder

	// Pad with empty lines so items stick to the bottom
	emptyLines := maxVisible - len(rows)
	for i := 0; i < emptyLines; i++ {
		b.WriteString("\n")
	}

	// Render items in reverse order (first match closest to prompt)
	for i := len(rows) - 1; i >= 0; i-- {
		b.WriteString(rows[i])
		b.WriteString("\n")
	}

	// Prompt line at the bottom
	b.WriteString(promptStyle.Render("> "))
	b.WriteString(cursorStyle.Render(m.query))
	b.WriteString(cursorStyle.Render("█"))

	return b.String()
}

// renderItem renders a single filtered item row.
func (m model) renderItem(fi filteredItem, isCursor bool) string {
	// Max width for the row content (leave space for cursor indicator)
	maxW := m.width - 2
	if maxW < 10 {
		maxW = 10
	}

	var b strings.Builder

	// Cursor indicator
	if isCursor {
		b.WriteString(promptStyle.Render("▸ "))
	} else {
		b.WriteString("  ")
	}

	// Render name with highlights
	nameStr := highlightMatches(fi.item.Name, fi.namePositions, normalStyle, matchStyle)

	// Render aliases if present
	aliasStr := ""
	if len(fi.item.Aliases) > 0 {
		var parts []string
		for i, alias := range fi.item.Aliases {
			if i < len(fi.aliasMatches) && fi.aliasMatches[i].positions != nil {
				parts = append(parts, highlightMatches(alias, fi.aliasMatches[i].positions, dimStyle, dimMatchStyle))
			} else {
				parts = append(parts, dimStyle.Render(alias))
			}
		}
		aliasStr = dimStyle.Render(" (") + strings.Join(parts, dimStyle.Render(", ")) + dimStyle.Render(")")
	}

	row := nameStr + aliasStr

	// Truncate if needed (approximate — styled strings contain escape codes)
	// We just cap visible chars loosely; Lip Gloss handles the rest.

	if isCursor {
		// Apply reverse to the whole content portion
		b.Reset()
		content := nameStr + aliasStr
		b.WriteString(selectedStyle.Render("▸ " + stripStyle(fi, maxW)))
		_ = content // use styled version only in non-selected
		return b.String()
	}

	b.WriteString(row)
	return b.String()
}

// stripStyle produces a plain-text version of the item for reverse-video rendering.
func stripStyle(fi filteredItem, maxW int) string {
	s := fi.item.Name
	if len(fi.item.Aliases) > 0 {
		s += " (" + strings.Join(fi.item.Aliases, ", ") + ")"
	}
	// Truncate to maxW runes
	runes := []rune(s)
	if len(runes) > maxW {
		runes = runes[:maxW]
	}
	return string(runes)
}

// highlightMatches renders text with certain character positions styled differently.
func highlightMatches(text string, positions []int, base, highlight lipgloss.Style) string {
	if len(positions) == 0 {
		return base.Render(text)
	}

	posSet := make(map[int]bool, len(positions))
	for _, p := range positions {
		posSet[p] = true
	}

	var b strings.Builder
	runes := []rune(text)
	i := 0
	for i < len(runes) {
		if posSet[i] {
			// Collect consecutive highlighted runes
			j := i
			for j < len(runes) && posSet[j] {
				j++
			}
			b.WriteString(highlight.Render(string(runes[i:j])))
			i = j
		} else {
			// Collect consecutive normal runes
			j := i
			for j < len(runes) && !posSet[j] {
				j++
			}
			b.WriteString(base.Render(string(runes[i:j])))
			i = j
		}
	}
	return b.String()
}

// Run executes the fuzzy finder with the given items and returns the selected key.
func Run(items []Item, opts Options) (string, error) {
	if len(items) == 0 {
		return "", ErrSelectionCancelled
	}

	m := initialModel(items)
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	final := result.(model)
	if final.cancelled {
		return "", ErrSelectionCancelled
	}
	return final.selected, nil
}

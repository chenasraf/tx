package picker

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/chenasraf/tx/internal/theme"
	"github.com/chenasraf/tx/internal/tmuxctl"
)

// fzf's default 256-colour scheme, which the list matches
const (
	sgrPointer   = "\x1b[1m\x1b[38;5;161m"
	sgrCursorBg  = "\x1b[48;5;236m"
	sgrCursorRow = "\x1b[1m\x1b[38;5;254m"
	sgrGutter    = "\x1b[38;5;236m"
	sgrBorder    = "\x1b[38;5;59m"
	sgrPrompt    = "\x1b[38;5;110m"
	sgrInfo      = "\x1b[38;5;144m"
	gutterGlyph  = "▌"
	borderGlyph  = "│"
	ellipsis     = ".."
	doubleClick  = 400 * time.Millisecond
	settleDelay  = 30 * time.Millisecond
	killTimeout  = time.Second
)

// notifications that mean the tree or a layout changed
var structural = map[string]bool{
	"sessions-changed":        true,
	"session-renamed":         true,
	"session-window-changed":  true,
	"window-add":              true,
	"window-close":            true,
	"window-renamed":          true,
	"window-pane-changed":     true,
	"unlinked-window-add":     true,
	"unlinked-window-close":   true,
	"unlinked-window-renamed": true,
	"layout-change":           true,
	"pane-mode-changed":       true,
}

// conn is what the picker needs from its tmux connection.
type conn interface {
	tmuxctl.Runner
	Send(cmds ...[]string) error
	Notifications() <-chan tmuxctl.Notification
	Done() <-chan struct{}
}

type armed struct {
	target, what string
	at           time.Time
	token        int
}

type previewReq struct {
	id           string
	cols, lines  int
	refreshToken int
}

type (
	previewMsg struct {
		req  previewReq
		text string
	}
	noteMsg    tmuxctl.Notification
	closedMsg  struct{}
	refreshMsg struct{}
	tickMsg    struct{}
	disarmMsg  struct{ token int }
	killedMsg  struct{ err error }
	peekedMsg  struct{ quit bool }
)

type model struct {
	src     source
	conn    conn
	th      theme.Theme
	pct     int
	refresh time.Duration
	now     func() time.Time

	width, height int

	rows   []Row
	total  int
	cursor int
	offset int
	// focus is a row to put the cursor on once it shows up in the list
	focus string

	input bool
	query []rune
	qpos  int

	armed    *armed
	armToken int
	// the legend's height while not armed, kept so the list doesn't jump while the prompt shows
	legendHeight int

	previewText string
	previewFor  previewReq
	previewBusy bool
	previewWant previewReq
	ticks       int

	refreshPending bool
	remembered     string

	clickRow int
	clickAt  time.Time

	chosen string
	err    error
}

func newModel(src source, c conn, th theme.Theme, pct int, refresh time.Duration) *model {
	m := &model{src: src, conn: c, th: th, pct: pct, refresh: refresh, now: time.Now, clickRow: -1}
	m.focus = src.start()
	m.setRows(src.rows(""))
	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.waitNote(), m.tick())
}

func (m *model) waitNote() tea.Cmd {
	return func() tea.Msg {
		select {
		case n := <-m.conn.Notifications():
			return noteMsg(n)
		case <-m.conn.Done():
			return closedMsg{}
		}
	}
}

func (m *model) tick() tea.Cmd {
	if m.refresh <= 0 {
		return nil
	}
	return tea.Tick(m.refresh, func(time.Time) tea.Msg { return tickMsg{} })
}

// ---- rows and cursor

func (m *model) current() string {
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		return m.rows[m.cursor].ID
	}
	return ""
}

func (m *model) indexOf(id string) int {
	for i, r := range m.rows {
		if r.ID == id {
			return i
		}
	}
	return -1
}

// setRows replaces the list, keeping the cursor on the pending focus, else the same row, else the
// same position.
func (m *model) setRows(rows []Row) {
	prev := m.current()
	m.rows = rows
	m.total = len(m.src.rows(""))
	switch {
	case m.focus != "" && m.indexOf(m.focus) >= 0:
		m.cursor = m.indexOf(m.focus)
		m.focus = ""
	case prev != "" && m.indexOf(prev) >= 0:
		m.cursor = m.indexOf(prev)
	default:
		m.focus = ""
		m.cursor = min(m.cursor, len(rows)-1)
	}
	m.cursor = max(m.cursor, 0)
	m.clampScroll()
}

func (m *model) reload() {
	if err := m.src.load(m.conn); err != nil {
		m.err = err
		return
	}
	m.setRows(m.src.rows(string(m.query)))
}

// requery filters for a changed query: the cursor lands on the first match so Enter goes straight
// to it.
func (m *model) requery() {
	m.setRows(m.src.rows(string(m.query)))
	if len(m.query) == 0 {
		return
	}
	for i, r := range m.rows {
		if r.Matched {
			m.cursor = i
			break
		}
	}
	m.clampScroll()
}

func (m *model) move(delta int) {
	if len(m.rows) == 0 {
		return
	}
	m.cursor = max(0, min(len(m.rows)-1, m.cursor+delta))
	m.clampScroll()
}

func (m *model) clampScroll() {
	h := max(m.listHeight(), 1)
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	m.offset = max(0, min(m.offset, max(len(m.rows)-h, 0)))
}

// afterChange follows any cursor or list change: remember the row and bring the preview up to date.
func (m *model) afterChange() tea.Cmd {
	id := m.current()
	if opt := m.src.remember(); opt != "" && id != "" && id != m.remembered {
		m.remembered = id
		_ = m.conn.Send([]string{"set-option", "-g", opt, id})
	}
	return m.requestPreview(false)
}

// ---- layout

func (m *model) legend() []string {
	if m.armed != nil {
		lines := make([]string, max(m.legendHeight, 1))
		lines[0] = armedLine(m.armed.what, m.th)
		return lines
	}
	lines := legendLines(m.src.keys(), m.width, m.th)
	m.legendHeight = len(lines)
	return lines
}

func (m *model) bodyHeight() int {
	h := m.legendHeight
	if h == 0 {
		h = len(legendLines(m.src.keys(), m.width, m.th))
	}
	return max(m.height-h-1, 1)
}

func (m *model) listTop() int {
	if m.input {
		return 2
	}
	return 0
}

func (m *model) listHeight() int {
	return m.bodyHeight() - m.listTop()
}

func (m *model) listWidth() int {
	return m.width - m.width*m.pct/100
}

func (m *model) previewSize() (int, int) {
	return max(m.width-m.listWidth()-2, 1), m.bodyHeight()
}

// ---- preview

func sameView(a, b previewReq) bool {
	return a.id == b.id && a.cols == b.cols && a.lines == b.lines
}

// requestPreview asks for the cursor row's preview; force re-captures one already shown.
func (m *model) requestPreview(force bool) tea.Cmd {
	cols, lines := m.previewSize()
	want := previewReq{id: m.current(), cols: cols, lines: lines, refreshToken: m.ticks}
	if !force && sameView(want, m.previewWant) {
		return nil
	}
	m.previewWant = want
	if want.id == "" {
		m.previewText, m.previewFor = "", want
		return nil
	}
	return m.fetchPreview()
}

// fetchPreview runs one fetch at a time in the background; requests made meanwhile collapse into
// the latest, which is fetched once the running one lands.
func (m *model) fetchPreview() tea.Cmd {
	want := m.previewWant
	if m.previewBusy || m.width == 0 || want.id == "" {
		return nil
	}
	m.previewBusy = true
	src, c := m.src, m.conn
	return func() tea.Msg {
		return previewMsg{req: want, text: src.preview(c, want.id, want.cols, want.lines)}
	}
}

// ---- update

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.legend()
		m.clampScroll()
		return m, m.requestPreview(false)

	case previewMsg:
		m.previewBusy = false
		// an older capture of the wanted view still beats a stale or empty one
		if sameView(msg.req, m.previewWant) {
			m.previewText, m.previewFor = msg.text, msg.req
		}
		if msg.req != m.previewWant {
			return m, m.fetchPreview()
		}
		return m, nil

	case noteMsg:
		cmds := []tea.Cmd{m.waitNote()}
		if structural[msg.Name] && !m.refreshPending {
			m.refreshPending = true
			cmds = append(cmds, tea.Tick(settleDelay, func(time.Time) tea.Msg { return refreshMsg{} }))
		}
		return m, tea.Batch(cmds...)

	case refreshMsg:
		m.refreshPending = false
		m.reload()
		return m, m.requestPreview(true)

	case tickMsg:
		m.ticks++
		if m.src.pollList() {
			m.reload()
		}
		return m, tea.Batch(m.tick(), m.afterChange(), m.requestPreview(true))

	case closedMsg:
		m.err = fmt.Errorf("lost the connection to tmux")
		return m, tea.Quit

	case disarmMsg:
		if m.armed != nil && m.armed.token == msg.token {
			m.armed = nil
		}
		return m, nil

	case killedMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.reload()
		return m, m.afterChange()

	case peekedMsg:
		if msg.quit {
			return m, tea.Quit
		}
		m.reload()
		return m, m.requestPreview(true)

	case tea.MouseMsg:
		return m, m.mouse(msg)

	case tea.KeyMsg:
		if m.input {
			return m, m.searchKey(msg)
		}
		return m, m.listKey(msg)
	}
	return m, nil
}

func (m *model) accept(id string) tea.Cmd {
	if id == "" {
		return tea.Quit
	}
	m.chosen = id
	return tea.Quit
}

func (m *model) listKey(msg tea.KeyMsg) tea.Cmd {
	k := msg.String()
	switch k {
	case "ctrl+c", "q", "esc":
		return tea.Quit
	case "j", "down", "ctrl+j", "ctrl+n":
		m.move(1)
	case "k", "up", "ctrl+k", "ctrl+p":
		m.move(-1)
	case "pgdown":
		m.move(m.listHeight())
	case "pgup":
		m.move(-m.listHeight())
	case "enter":
		return m.accept(m.current())
	case "/":
		m.input = true
		m.clampScroll()
		return m.requestPreview(false)
	case "d":
		return m.killKey()
	case "l", "right", "h", "left":
		t, ok := m.src.(tree)
		if !ok || m.current() == "" {
			return nil
		}
		var nav navigation
		var focus string
		if k == "l" || k == "right" {
			nav, focus = t.expand(m.conn, m.current())
		} else {
			nav, focus = t.collapse(m.conn, m.current())
		}
		switch nav {
		case navDown:
			m.move(1)
		case navReload:
			m.focus = focus
			m.reload()
		}
	case "i":
		p, ok := m.src.(peeker)
		if !ok || m.current() == "" {
			return nil
		}
		cmd, done, err := p.peek(m.conn, m.current())
		if err != nil {
			m.err = err
			return nil
		}
		return tea.ExecProcess(cmd, func(error) tea.Msg { return peekedMsg{quit: done()} })
	default:
		if len(k) == 1 && k[0] >= '0' && k[0] <= '9' {
			if id := m.src.digit(int(k[0] - '0')); id != "" {
				return m.accept(id)
			}
		}
		return nil
	}
	return m.afterChange()
}

func (m *model) searchKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		return tea.Quit
	case "enter":
		return m.accept(m.current())
	case "esc":
		// leaving search keeps the cursor on its row, which stays visible in the full list
		id := m.current()
		if t, ok := m.src.(tree); ok && id != "" {
			t.reveal(m.conn, id)
			_ = m.src.load(m.conn)
		}
		m.focus = id
		m.input, m.query, m.qpos = false, nil, 0
		m.setRows(m.src.rows(""))
	case "down", "ctrl+j", "ctrl+n":
		m.move(1)
	case "up", "ctrl+k", "ctrl+p":
		m.move(-1)
	case "pgdown":
		m.move(m.listHeight())
	case "pgup":
		m.move(-m.listHeight())
	case "left", "ctrl+b":
		m.qpos = max(m.qpos-1, 0)
		return nil
	case "right", "ctrl+f":
		m.qpos = min(m.qpos+1, len(m.query))
		return nil
	case "home", "ctrl+a":
		m.qpos = 0
		return nil
	case "end", "ctrl+e":
		m.qpos = len(m.query)
		return nil
	case "backspace", "ctrl+h":
		if m.qpos == 0 {
			return nil
		}
		m.query = append(m.query[:m.qpos-1], m.query[m.qpos:]...)
		m.qpos--
		m.requery()
	case "delete", "ctrl+d":
		if m.qpos >= len(m.query) {
			return nil
		}
		m.query = append(m.query[:m.qpos], m.query[m.qpos+1:]...)
		m.requery()
	case "ctrl+u":
		m.query = append([]rune{}, m.query[m.qpos:]...)
		m.qpos = 0
		m.requery()
	case "ctrl+w":
		i := m.qpos
		for i > 0 && m.query[i-1] == ' ' {
			i--
		}
		for i > 0 && m.query[i-1] != ' ' {
			i--
		}
		m.query = append(m.query[:i], m.query[m.qpos:]...)
		m.qpos = i
		m.requery()
	default:
		var runes []rune
		switch msg.Type {
		case tea.KeyRunes:
			runes = msg.Runes
		case tea.KeySpace:
			runes = []rune{' '}
		default:
			return nil
		}
		q := append([]rune{}, m.query[:m.qpos]...)
		q = append(q, runes...)
		m.query = append(q, m.query[m.qpos:]...)
		m.qpos += len(runes)
		m.requery()
	}
	return m.afterChange()
}

// killKey handles d, vim style: the first d arms a kill of the row under the cursor and swaps the
// legend for a prompt; a second d on the same row within killTimeout confirms.
func (m *model) killKey() tea.Cmd {
	target := m.current()
	if target == "" {
		return nil
	}
	if a := m.armed; a != nil && a.target == target && m.now().Sub(a.at) < killTimeout {
		m.armed = nil
		focus, run := m.src.kill(target)
		if focus != "" {
			m.focus = focus
		}
		c := m.conn
		return func() tea.Msg { return killedMsg{err: run(c)} }
	}
	m.armToken++
	token := m.armToken
	m.armed = &armed{target: target, what: m.src.describe(target), at: m.now(), token: token}
	return tea.Tick(killTimeout, func(time.Time) tea.Msg { return disarmMsg{token: token} })
}

func (m *model) mouse(msg tea.MouseMsg) tea.Cmd {
	if msg.X >= m.listWidth() {
		return nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelDown:
		m.move(1)
		return m.afterChange()
	case tea.MouseButtonWheelUp:
		m.move(-1)
		return m.afterChange()
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		y := msg.Y - m.listTop()
		if y < 0 || y >= m.listHeight() {
			return nil
		}
		i := m.offset + y
		if i >= len(m.rows) {
			return nil
		}
		now := m.now()
		if i == m.clickRow && now.Sub(m.clickAt) < doubleClick {
			return m.accept(m.rows[i].ID)
		}
		m.clickRow, m.clickAt = i, now
		m.cursor = i
		return m.afterChange()
	}
	return nil
}

// ---- view

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	listW := m.listWidth()
	itemW := max(listW-3, 1)
	bodyH := m.bodyHeight()
	pcols, _ := m.previewSize()
	previewLines := strings.Split(m.previewText, "\n")

	var b strings.Builder
	for y := 0; y < bodyH; y++ {
		b.WriteString(m.listLine(y, listW, itemW))
		b.WriteString(sgrBorder + borderGlyph + reset + " ")
		if y < len(previewLines) && m.previewText != "" {
			b.WriteString(ansi.Truncate(previewLines[y], pcols, ""))
		}
		b.WriteString(reset + "\n")
	}
	b.WriteString(m.th.Fg("overlay_0", false) + strings.Repeat("─", m.width) + reset)
	for _, l := range m.legend() {
		b.WriteString("\n" + ansi.Truncate(l, m.width, "") + reset)
	}
	return b.String()
}

func (m *model) listLine(y, listW, itemW int) string {
	if m.input {
		switch y {
		case 0:
			return m.promptLine(listW)
		case 1:
			return sgrBorder + strings.Repeat("─", max(listW-1, 0)) + reset + " "
		}
	}
	i := m.offset + y - m.listTop()
	if i < 0 || i >= len(m.rows) {
		return strings.Repeat(" ", listW)
	}
	content := ansi.Truncate(m.rows[i].Display, itemW, ellipsis)
	pad := strings.Repeat(" ", max(itemW-ansi.StringWidth(content), 0))
	if i == m.cursor {
		// the cursor row is bold on a highlight, and the row's own resets would drop both
		content = sgrCursorRow + strings.ReplaceAll(content, reset, reset+sgrCursorBg+sgrCursorRow)
		return sgrPointer + sgrCursorBg + gutterGlyph + reset + sgrCursorBg + " " + content + pad + reset + " "
	}
	return sgrGutter + gutterGlyph + reset + " " + content + pad + reset + " "
}

func (m *model) promptLine(listW int) string {
	info := fmt.Sprintf("%d/%d", len(m.rows), m.total)
	var q strings.Builder
	for i, r := range m.query {
		if i == m.qpos {
			q.WriteString("\x1b[7m" + string(r) + "\x1b[27m")
		} else {
			q.WriteRune(r)
		}
	}
	if m.qpos >= len(m.query) {
		q.WriteString("\x1b[7m \x1b[27m")
	}
	query := ansi.Truncate(q.String(), max(listW-3-len(info)-1, 1), "")
	used := 2 + ansi.StringWidth(query)
	pad := strings.Repeat(" ", max(listW-1-used-len(info), 1))
	return "\x1b[1m" + sgrPrompt + "/" + reset + sgrPrompt + " " + reset + "\x1b[1m" + query + reset +
		pad + sgrInfo + info + reset + " "
}

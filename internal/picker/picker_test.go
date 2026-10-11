package picker

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/chenasraf/tx/internal/theme"
	"github.com/chenasraf/tx/internal/tmuxctl"
)

var sgr = regexp.MustCompile(`\x1b\[[0-9;:]*m`)

func plain(s string) string { return sgr.ReplaceAllString(s, "") }

// fakeTmux answers the picker's queries from fixed tables and records every other command.
type fakeTmux struct {
	sessions []string // list-sessions rows
	windows  []string // list-windows -a rows
	clients  []string // list-clients rows
	last     string
	ran      [][]string
	notes    chan tmuxctl.Notification
	done     chan struct{}
}

func newFake() *fakeTmux {
	return &fakeTmux{
		sessions: []string{
			"$1\talpha\t2\t1",
			"$2\tbeta\t1\t",
			"$10\tgamma\t2\t",
			"$3\t_popup_sessions_ttys001\t1\t",
		},
		windows: []string{
			"$1\t@1\t1\tzsh\t1",
			"$1\t@2\t2\teditor\t2",
			"$2\t@3\t1\tzsh\t1",
			"$10\t@4\t1\tlogs\t1",
			"$10\t@5\t2\tbuild\t1",
			"$3\t@6\t1\ttx\t1",
		},
		clients: []string{
			"/dev/ttys001\t$2\talpha\t0",
			"client-99\t$3\t\t1",
		},
		notes: make(chan tmuxctl.Notification),
		done:  make(chan struct{}),
	}
}

func (f *fakeTmux) Run(cmds ...[]string) ([]tmuxctl.Result, error) {
	out := make([]tmuxctl.Result, len(cmds))
	for i, c := range cmds {
		var lines []string
		switch c[0] {
		case "list-sessions":
			lines = f.sessions
		case "list-windows":
			lines = f.windows
		case "list-clients":
			lines = f.clients
		case "show-options":
			lines = []string{f.last}
		default:
			f.ran = append(f.ran, c)
		}
		out[i] = tmuxctl.Result{Lines: append([]string{}, lines...)}
	}
	return out, nil
}

func (f *fakeTmux) Send(cmds ...[]string) error {
	f.ran = append(f.ran, cmds...)
	return nil
}

func (f *fakeTmux) Notifications() <-chan tmuxctl.Notification { return f.notes }
func (f *fakeTmux) Done() <-chan struct{}                      { return f.done }

func loaded(t *testing.T, f *fakeTmux) *sessions {
	t.Helper()
	s := newSessions("/dev/ttys001", theme.Mocha)
	if err := s.load(f); err != nil {
		t.Fatal(err)
	}
	return s
}

func ids(rows []Row) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.ID
	}
	return out
}

func TestSessionRows_Tree(t *testing.T) {
	s := loaded(t, newFake())
	rows := s.rows("")
	// sorted numerically by id, popup sessions hidden, only alpha expanded
	if got, want := ids(rows), []string{"$1", "@1", "@2", "$2", "$10"}; !slices.Equal(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	want := []string{
		"0 ▾ alpha 2 windows",
		"      1 zsh",
		"      2 editor 2 panes",
		"1 ▸ beta 1 window ●",
		"2 ▸ gamma 2 windows",
	}
	for i, w := range want {
		if got := plain(rows[i].Display); got != w {
			t.Errorf("row %d = %q, want %q", i, got, w)
		}
	}
}

func TestSessionRows_Query(t *testing.T) {
	s := loaded(t, newFake())
	rows := s.rows("ed")
	// a matching window brings its session row; quick keys keep their numbers
	if got, want := ids(rows), []string{"$1", "@2"}; !slices.Equal(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	if rows[0].Matched || !rows[1].Matched {
		t.Errorf("matched = %v %v, want false true", rows[0].Matched, rows[1].Matched)
	}
	if got := ids(s.rows("bu")); !slices.Equal(got, []string{"$10", "@5"}) {
		t.Errorf("subsequence query: %v", got)
	}
	if got := ids(s.rows("BETA")); len(got) != 0 {
		t.Errorf("smart case: uppercase should be case-sensitive, got %v", got)
	}
}

func TestSessionStart(t *testing.T) {
	f := newFake()
	f.last = "@5" // under gamma, which is collapsed
	if got := loaded(t, f).start(); got != "$10" {
		t.Errorf("hidden window: start = %s, want $10", got)
	}
	f.last = "@99"
	if got := loaded(t, f).start(); got != "$2" {
		t.Errorf("gone: start = %s, want the client's session $2", got)
	}
	f.last = "@2"
	if got := loaded(t, f).start(); got != "@2" {
		t.Errorf("visible window: start = %s, want @2", got)
	}
	f.last = `\$10` // as tmux 3.4 prints it
	if got := loaded(t, f).start(); got != "$10" {
		t.Errorf("escaped session id: start = %s, want $10", got)
	}
}

func TestSessionKill(t *testing.T) {
	s := loaded(t, newFake())

	// a window of a multi-window session: just the window, landing on the row above
	focus, run := s.kill("@2")
	if focus != "@1" {
		t.Errorf("focus = %s, want @1", focus)
	}
	f := newFake()
	if err := run(f); err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(f.ran, [][]string{{"kill-window", "-t", "@2"}}, slices.Equal) {
		t.Errorf("ran %v", f.ran)
	}

	// the client's own session: switch the client to its last session first
	focus, run = s.kill("$2")
	if focus != "@2" {
		t.Errorf("focus = %s, want @2", focus)
	}
	f = newFake()
	_ = run(f)
	want := [][]string{{"switch-client", "-c", "/dev/ttys001", "-t", "$1"}, {"kill-session", "-t", "$2"}}
	if !slices.EqualFunc(f.ran, want, slices.Equal) {
		t.Errorf("ran %v, want %v", f.ran, want)
	}

	focus, _ = s.kill("@1")
	if focus != "$1" {
		t.Errorf("window @1: focus = %s, want $1", focus)
	}
	// the first session's rows all go: land below
	focus, _ = s.kill("$1")
	if focus != "$2" {
		t.Errorf("session $1: focus = %s, want $2", focus)
	}
	if got := s.describe("@3"); got != "session beta" {
		t.Errorf("only window describes as its session, got %q", got)
	}
	if got := s.describe("@5"); got != "window 2 build" {
		t.Errorf("describe = %q", got)
	}
}

func TestSessionFallback(t *testing.T) {
	s := loaded(t, newFake())
	s.clientLast = "gone"
	if got := s.fallback("$2"); got != "$1" {
		t.Errorf("previous: %s", got)
	}
	if got := s.fallback("$1"); got != "$2" {
		t.Errorf("next: %s", got)
	}
}

func TestSessionDigit(t *testing.T) {
	s := loaded(t, newFake())
	if got := s.digit(2); got != "$10" {
		t.Errorf("digit 2 = %s", got)
	}
	if got := s.digit(5); got != "" {
		t.Errorf("digit past the end = %s", got)
	}
}

func TestLegendLines(t *testing.T) {
	s := newSessions("", theme.Mocha)
	wide := legendLines(s.keys(), 120, theme.Mocha)
	want := []string{
		"  ↑↓/jk move    →←/lh expand/collapse    enter switch    0-9 switch to session",
		"  / search      esc leave search         dd kill         q quit",
	}
	if len(wide) != 2 || plain(wide[0]) != want[0] || plain(wide[1]) != want[1] {
		t.Errorf("wide = %q", wide)
	}
	narrow := legendLines(s.keys(), 60, theme.Mocha)
	want = []string{
		"  ↑↓/jk move    →←/lh expand/collapse    enter switch",
		"  0-9 switch to session",
		"  / search    esc leave search    dd kill    q quit",
	}
	if len(narrow) != 3 {
		t.Fatalf("narrow = %q", narrow)
	}
	for i := range want {
		if plain(narrow[i]) != want[i] {
			t.Errorf("narrow %d = %q, want %q", i, plain(narrow[i]), want[i])
		}
	}
}

func TestFuzzy(t *testing.T) {
	pos, ok := fuzzy("dit", "editor")
	if !ok || !pos[1] || !pos[3] || len(pos) != 3 {
		t.Errorf("substring: %v %v", pos, ok)
	}
	pos, ok = fuzzy("etr", "editor")
	if !ok || !pos[0] || !pos[3] || !pos[5] || len(pos) != 3 {
		t.Errorf("subsequence: %v %v", pos, ok)
	}
	if _, ok := fuzzy("x", "editor"); ok {
		t.Error("no match matched")
	}
}

// ---- model

func newTestModel(t *testing.T) (*model, *fakeTmux) {
	t.Helper()
	f := newFake()
	s := loaded(t, f)
	m := newModel(s, f, theme.Mocha, 78, 0)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	return m, f
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func press(m *model, keys ...string) tea.Cmd {
	var cmd tea.Cmd
	for _, k := range keys {
		_, cmd = m.Update(keyMsg(k))
	}
	return cmd
}

func TestModel_StartsOnClientSession(t *testing.T) {
	m, _ := newTestModel(t)
	if got := m.current(); got != "$2" {
		t.Errorf("current = %s, want $2", got)
	}
}

func TestModel_MoveRemembersRow(t *testing.T) {
	m, f := newTestModel(t)
	press(m, "j")
	if got := m.current(); got != "$10" {
		t.Fatalf("current = %s", got)
	}
	if !slices.ContainsFunc(f.ran, func(c []string) bool { return slices.Equal(c, []string{"set-option", "-g", lastOpt, "$10"}) }) {
		t.Errorf("last row not recorded: %v", f.ran)
	}
}

func TestModel_SearchLandsOnFirstMatch(t *testing.T) {
	m, _ := newTestModel(t)
	press(m, "/", "e", "d")
	if string(m.query) != "ed" || m.current() != "@2" {
		t.Errorf("query %q current %s", string(m.query), m.current())
	}
	// q and d type while searching
	press(m, "q")
	if string(m.query) != "edq" {
		t.Errorf("query = %q", string(m.query))
	}
	press(m, "backspace")
	// leaving search on a window expands its session and stays on the window
	press(m, "esc")
	if m.input || len(m.query) != 0 || m.current() != "@2" {
		t.Errorf("after esc: input %v query %q current %s", m.input, string(m.query), m.current())
	}
}

func TestModel_ExpandCollapse(t *testing.T) {
	m, f := newTestModel(t)
	press(m, "l") // on beta, collapsed
	if !slices.ContainsFunc(f.ran, func(c []string) bool { return slices.Equal(c, []string{"set-option", "-t", "$2", expandedOpt, "1"}) }) {
		t.Errorf("expand not set: %v", f.ran)
	}
	press(m, "k") // @2, under expanded alpha
	if m.current() != "@2" {
		t.Fatalf("current = %s", m.current())
	}
	press(m, "h")
	if !slices.ContainsFunc(f.ran, func(c []string) bool { return slices.Equal(c, []string{"set-option", "-u", "-t", "$1", expandedOpt}) }) {
		t.Errorf("collapse not unset: %v", f.ran)
	}
}

func TestModel_KillArmsThenConfirms(t *testing.T) {
	m, f := newTestModel(t)
	clock := time.Unix(1000, 0)
	m.now = func() time.Time { return clock }

	press(m, "d")
	if m.armed == nil || m.armed.what != "session beta" {
		t.Fatalf("armed = %+v", m.armed)
	}
	if !strings.Contains(plain(m.View()), "d again to kill session beta") {
		t.Error("legend doesn't show the armed prompt")
	}

	// too late: re-arms instead of killing
	clock = clock.Add(2 * time.Second)
	if cmd := press(m, "d"); cmd == nil || m.armed == nil {
		t.Fatal("expected a re-arm")
	}
	clock = clock.Add(500 * time.Millisecond)
	cmd := press(m, "d")
	if m.armed != nil || cmd == nil {
		t.Fatal("expected the kill")
	}
	msg := cmd()
	if !slices.ContainsFunc(f.ran, func(c []string) bool { return slices.Equal(c, []string{"kill-session", "-t", "$2"}) }) {
		t.Errorf("not killed: %v", f.ran)
	}
	if _, ok := msg.(killedMsg); !ok {
		t.Errorf("msg = %T", msg)
	}
}

func TestModel_DigitAccepts(t *testing.T) {
	m, _ := newTestModel(t)
	press(m, "2")
	if m.chosen != "$10" {
		t.Errorf("chosen = %s", m.chosen)
	}
}

func TestModel_ViewFillsScreen(t *testing.T) {
	m, _ := newTestModel(t)
	lines := strings.Split(m.View(), "\n")
	if len(lines) != 30 {
		t.Fatalf("view has %d lines, want 30", len(lines))
	}
	if got := plain(lines[0]); !strings.HasPrefix(got, "▌ 0 ▾ alpha 2 windows ") || !strings.Contains(got, "│") {
		t.Errorf("first line = %q", got)
	}
}

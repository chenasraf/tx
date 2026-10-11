package picker

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/chenasraf/tx/internal/popup"
	"github.com/chenasraf/tx/internal/preview"
	"github.com/chenasraf/tx/internal/theme"
	"github.com/chenasraf/tx/internal/tmuxctl"
)

// Expansion is a session option rather than picker state, so it carries over between launches and
// goes away with the session.
const expandedOpt = "@tx_picker_expanded"

// lastOpt records the last highlighted row, so the next launch reopens where you left off.
const lastOpt = "@tx_picker_last"

type sessionInfo struct {
	ID, Name string
	Windows  int
	Attached bool
	Expanded bool
}

type windowInfo struct {
	SessionID, ID, Index, Name, Panes string
}

type sessions struct {
	client string
	th     theme.Theme

	list    []sessionInfo
	windows []windowInfo
	last    string
	// the client's current and previous session
	clientSession, clientLast string
}

func newSessions(client string, th theme.Theme) *sessions {
	return &sessions{client: client, th: th}
}

func (s *sessions) title() string { return "sessions" }

func (s *sessions) remember() string { return lastOpt }

func (s *sessions) pollList() bool { return false }

func (s *sessions) keys() [][]key {
	return [][]key{
		{{"↑↓/jk", "move"}, {"→←/lh", "expand/collapse"}, {"enter", "switch"}, {"0-9", "switch to session"}},
		{{"/", "search"}, {"esc", "leave search"}, {"dd", "kill"}, {"q", "quit"}},
	}
}

func (s *sessions) load(r tmuxctl.Runner) error {
	res, err := r.Run(
		[]string{"list-sessions", "-F", "#{session_id}\t#{session_name}\t#{session_windows}\t#{" + expandedOpt + "}"},
		[]string{"list-windows", "-a", "-F", "#{session_id}\t#{window_id}\t#{window_index}\t#{window_name}\t#{window_panes}"},
		// session_attached counts control-mode clients, this picker's included
		[]string{"list-clients", "-F", "#{client_name}\t#{session_id}\t#{client_last_session}\t#{client_control_mode}"},
		[]string{"show-options", "-gqv", lastOpt},
	)
	if err != nil {
		return err
	}

	attached := map[string]bool{}
	s.clientSession, s.clientLast = "", ""
	for _, c := range res[2].Rows() {
		if len(c) < 4 || c[3] == "1" {
			continue
		}
		attached[c[1]] = true
		if c[0] == s.client {
			s.clientSession, s.clientLast = c[1], c[2]
		}
	}

	s.list = s.list[:0]
	for _, row := range res[0].Rows() {
		// popup sessions hold popups (this picker included), not work
		if len(row) < 4 || strings.HasPrefix(row[1], popup.Prefix) {
			continue
		}
		n, _ := strconv.Atoi(row[2])
		s.list = append(s.list, sessionInfo{
			ID: row[0], Name: row[1], Windows: n, Attached: attached[row[0]], Expanded: row[3] == "1",
		})
	}
	slices.SortFunc(s.list, func(a, b sessionInfo) int { return idNum(a.ID) - idNum(b.ID) })

	s.windows = s.windows[:0]
	for _, row := range res[1].Rows() {
		if len(row) >= 5 {
			s.windows = append(s.windows, windowInfo{SessionID: row[0], ID: row[1], Index: row[2], Name: row[3], Panes: row[4]})
		}
	}
	s.last = res[3].Text()
	return nil
}

// idNum orders ids like $12 and @3 numerically, which is creation order.
func idNum(id string) int {
	n, _ := strconv.Atoi(strings.TrimLeft(id, "$@%"))
	return n
}

// quickKey labels rows 1-10 with the keys 0-9.
func quickKey(n int, th theme.Theme) string {
	if n <= 10 {
		return th.Fg("sky", false) + strconv.Itoa(n-1) + reset
	}
	return " "
}

// rows lists sessions in creation order. Without a query, windows are listed only under expanded
// sessions; with one, matching sessions and windows are listed, and a matching window always
// brings its session row along.
func (s *sessions) rows(query string) []Row {
	th := s.th
	matchStyle := th.Fg("peach", true)
	var out []Row
	for n, sess := range s.list {
		var own []windowInfo
		for _, w := range s.windows {
			if w.SessionID == sess.ID {
				own = append(own, w)
			}
		}

		type shownWindow struct {
			w   windowInfo
			pos map[int]bool
		}
		var shown []shownWindow
		var sPos map[int]bool
		sMatched := false
		if query != "" {
			sPos, sMatched = fuzzy(query, sess.Name)
			for _, w := range own {
				if p, ok := fuzzy(query, w.Name); ok {
					shown = append(shown, shownWindow{w, p})
				}
			}
			if !sMatched && len(shown) == 0 {
				continue
			}
		} else if sess.Expanded {
			for _, w := range own {
				shown = append(shown, shownWindow{w, nil})
			}
		}

		arrow := "▸"
		if len(shown) > 0 {
			arrow = "▾"
		}
		dot := ""
		if sess.Attached {
			dot = " " + th.Fg("green", false) + "●" + reset
		}
		count := fmt.Sprintf("%d window", sess.Windows)
		if sess.Windows != 1 {
			count += "s"
		}
		out = append(out, Row{
			ID:        sess.ID,
			SessionID: sess.ID,
			Display: quickKey(n+1, th) + " " + th.Fg("overlay_1", false) + arrow + reset + " " +
				highlight(sess.Name, sPos, th.Fg("blue", true), matchStyle) + " " +
				th.Fg("overlay_0", false) + count + reset + dot,
			Matched: query != "" && sMatched,
		})
		for _, sw := range shown {
			extra := ""
			if sw.w.Panes != "1" {
				extra = " " + th.Fg("overlay_0", false) + sw.w.Panes + " panes" + reset
			}
			out = append(out, Row{
				ID:        sw.w.ID,
				SessionID: sess.ID,
				Display: "      " + th.Fg("overlay_1", false) + sw.w.Index + reset + " " +
					highlight(sw.w.Name, sw.pos, th.Fg("text", false), matchStyle) + extra,
				Matched: query != "",
			})
		}
	}
	return out
}

func (s *sessions) session(id string) *sessionInfo {
	for i := range s.list {
		if s.list[i].ID == id {
			return &s.list[i]
		}
	}
	return nil
}

func (s *sessions) window(id string) *windowInfo {
	for i := range s.windows {
		if s.windows[i].ID == id {
			return &s.windows[i]
		}
	}
	return nil
}

func (s *sessions) sessionOf(id string) string {
	if strings.HasPrefix(id, "$") {
		return id
	}
	if w := s.window(id); w != nil {
		return w.SessionID
	}
	return ""
}

// start reopens on the last highlighted row; a window hidden under a since-collapsed session
// falls back to that session, and anything gone to the client's current session.
func (s *sessions) start() string {
	rows := s.rows("")
	has := func(id string) bool {
		return id != "" && slices.ContainsFunc(rows, func(r Row) bool { return r.ID == id })
	}
	for _, c := range []string{s.last, s.sessionOf(s.last), s.clientSession} {
		if has(c) {
			return c
		}
	}
	return ""
}

func (s *sessions) digit(n int) string {
	if n < len(s.list) {
		return s.list[n].ID
	}
	return ""
}

func (s *sessions) preview(r tmuxctl.Runner, id string, cols, lines int) string {
	w, err := preview.FetchWindow(r, id)
	if err != nil {
		return ""
	}
	return preview.RenderWindow(w, s.th, cols, lines)
}

// killScope is the session or window a kill on row id takes down: a session's only window is
// the session.
func (s *sessions) killScope(id string) string {
	if strings.HasPrefix(id, "$") {
		return id
	}
	sid := s.sessionOf(id)
	if sess := s.session(sid); sess != nil && sess.Windows == 1 {
		return sid
	}
	return id
}

func (s *sessions) describe(id string) string {
	scope := s.killScope(id)
	if sess := s.session(scope); sess != nil {
		return "session " + sess.Name
	}
	if w := s.window(scope); w != nil {
		return "window " + w.Index + " " + w.Name
	}
	return scope
}

func (s *sessions) kill(id string) (string, func(r tmuxctl.Runner) error) {
	scope := s.killScope(id)
	rows := s.rows("")
	doomed := map[string]bool{}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
		if row.ID == scope || row.SessionID == scope {
			doomed[row.ID] = true
		}
	}
	i := max(slices.Index(ids, scope), 0)
	focus := ""
	for _, c := range append(reversed(ids[:i]), ids[min(i+1, len(ids)):]...) {
		if !doomed[c] {
			focus = c
			break
		}
	}

	if !strings.HasPrefix(scope, "$") {
		return focus, func(r tmuxctl.Runner) error {
			return firstErr(r.Run([]string{"kill-window", "-t", scope}))
		}
	}
	// killing the client's own session would detach it, so move it off first
	fallback := ""
	if s.client != "" && s.clientSession == scope {
		fallback = s.fallback(scope)
	}
	return focus, func(r tmuxctl.Runner) error {
		if fallback != "" {
			if err := firstErr(r.Run([]string{"switch-client", "-c", s.client, "-t", fallback})); err != nil {
				return err
			}
		}
		return firstErr(r.Run([]string{"kill-session", "-t", scope}))
	}
}

// fallback is the client's previous session, else the one before sid in the list, else the one
// after.
func (s *sessions) fallback(sid string) string {
	for _, sess := range s.list {
		if sess.Name == s.clientLast && sess.ID != sid {
			return sess.ID
		}
	}
	ids := make([]string, len(s.list))
	for i, sess := range s.list {
		ids[i] = sess.ID
	}
	i := max(slices.Index(ids, sid), 0)
	others := append(reversed(ids[:i]), ids[min(i+1, len(ids)):]...)
	if len(others) > 0 {
		return others[0]
	}
	return ""
}

func (s *sessions) accept(r tmuxctl.Runner, id string) error {
	return firstErr(r.Run([]string{"switch-client", "-c", s.client, "-t", id}))
}

func (s *sessions) expand(r tmuxctl.Runner, id string) (navigation, string) {
	sess := s.session(id)
	if sess == nil {
		return navNone, ""
	}
	if sess.Expanded {
		return navDown, ""
	}
	_, _ = r.Run([]string{"set-option", "-t", id, expandedOpt, "1"})
	return navReload, id
}

// collapse folds the row's session; from a window it lands on the session, since the window row
// goes away.
func (s *sessions) collapse(r tmuxctl.Runner, id string) (navigation, string) {
	sid := s.sessionOf(id)
	sess := s.session(sid)
	if sess == nil || !sess.Expanded {
		return navNone, ""
	}
	_, _ = r.Run([]string{"set-option", "-u", "-t", sid, expandedOpt})
	return navReload, sid
}

// reveal expands a window's session, so the window stays visible in the tree after a search.
func (s *sessions) reveal(r tmuxctl.Runner, id string) {
	if strings.HasPrefix(id, "@") {
		if sid := s.sessionOf(id); sid != "" {
			_, _ = r.Run([]string{"set-option", "-t", sid, expandedOpt, "1"})
		}
	}
}

func reversed(s []string) []string {
	out := slices.Clone(s)
	slices.Reverse(out)
	return out
}

// firstErr folds a batch's connection error and the first rejected command into one error.
func firstErr(res []tmuxctl.Result, err error) error {
	if err != nil {
		return err
	}
	for _, r := range res {
		if r.Err != nil {
			return r.Err
		}
	}
	return nil
}

package picker

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/chenasraf/tx/internal/popup"
	"github.com/chenasraf/tx/internal/preview"
	"github.com/chenasraf/tx/internal/theme"
	"github.com/chenasraf/tx/internal/tmuxctl"
)

// Match modes for agent detection.
const (
	// MatchAuto checks plain entries against the pane's process name and /regex/ entries against
	// its foreground command line.
	MatchAuto = "auto"
	// MatchName checks every entry against the pane's process name.
	MatchName = "name"
	// MatchArgs checks every entry against the pane's foreground command line.
	MatchArgs = "args"
)

// SIGTERM lets an agent shut down cleanly; SIGKILL only if it outlasts this.
const killGrace = 3 * time.Second

// Working in an agent pane from the picker: a nested client attaches to a throwaway session
// grouped with the pane's session, so it can show the pane's window without moving the user's
// own client. That session's key table makes C-c detach and C-d detach and flag the picker to
// quit too; every other key goes to the pane.
const (
	peekTable      = "tx-peek"
	peekRestoreOpt = "@tx_peek_restore"
	peekQuitOpt    = "@tx_peek_quit"
)

// matcher is one configured agent: a process name, or a pattern over the command line.
type matcher struct {
	name string
	re   *regexp.Regexp
}

// parseMatchers turns configured entries into matchers; /.../ entries are regular expressions.
func parseMatchers(entries []string) ([]matcher, error) {
	var out []matcher
	for _, e := range entries {
		if len(e) >= 2 && strings.HasPrefix(e, "/") && strings.HasSuffix(e, "/") {
			re, err := regexp.Compile(e[1 : len(e)-1])
			if err != nil {
				return nil, fmt.Errorf("agent pattern %s: %w", e, err)
			}
			out = append(out, matcher{re: re})
			continue
		}
		out = append(out, matcher{name: e})
	}
	return out, nil
}

// proc is a process from ps.
type proc struct {
	tty              string
	pid, tpgid, pgid int
	args, argv0      string
}

// foreground reports whether p is in its terminal's foreground process group.
func (p proc) foreground() bool {
	return p.pgid == p.tpgid
}

func parsePS(out string) []proc {
	var procs []proc
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) < 5 {
			continue
		}
		pid, _ := strconv.Atoi(f[1])
		tpgid, _ := strconv.Atoi(f[2])
		pgid, _ := strconv.Atoi(f[3])
		procs = append(procs, proc{
			tty: f[0], pid: pid, tpgid: tpgid, pgid: pgid,
			args: strings.Join(f[4:], " "), argv0: filepath.Base(f[4]),
		})
	}
	return procs
}

func listProcs() ([]proc, error) {
	out, err := exec.Command("ps", "-A", "-o", "tty=,pid=,tpgid=,pgid=,args=").Output()
	if err != nil {
		return nil, err
	}
	return parsePS(string(out)), nil
}

// matchProc is the agent name process p matches as, or "".
func (m matcher) matchProc(p proc, mode string) string {
	if m.re != nil {
		return m.re.FindString(p.args)
	}
	if p.argv0 == m.name {
		return m.name
	}
	// in args mode a plain name also finds agents started through an interpreter, like node
	if mode == MatchArgs {
		for _, a := range strings.Fields(p.args)[1:] {
			if filepath.Base(a) == m.name {
				return m.name
			}
		}
	}
	return ""
}

// matchCommand is the agent name a pane's process name matches as, or "".
func (m matcher) matchCommand(command string) string {
	if m.re != nil {
		return m.re.FindString(command)
	}
	if command == m.name {
		return m.name
	}
	return ""
}

type agentPane struct {
	sessionID       string
	window, index   int
	id, path, title string
	tty             string
	// agent is what the pane matched as
	agent string
	label string
}

type agents struct {
	client   string
	th       theme.Theme
	matchers []matcher
	mode     string
	home     string

	panes []agentPane
}

func newAgents(client string, th theme.Theme, matchers []matcher, mode string) *agents {
	home, _ := os.UserHomeDir()
	if mode == "" {
		mode = MatchAuto
	}
	return &agents{client: client, th: th, matchers: matchers, mode: mode, home: home}
}

func (a *agents) title() string { return "agents" }

func (a *agents) remember() string { return "" }

// a pane starting or quitting an agent sends no notification
func (a *agents) pollList() bool { return true }

func (a *agents) keys() [][]key {
	return [][]key{
		{{"↑↓/jk", "move"}, {"enter", "switch"}, {"0-9", "switch to pane"}, {"i", "interact"}, {"^c", "back to list"}, {"^d", "close"}},
		{{"/", "search"}, {"esc", "leave search"}, {"dd", "kill"}, {"q", "quit"}},
	}
}

// needsProcs reports whether matching reads command lines, which takes a ps.
func (a *agents) needsProcs() bool {
	if a.mode == MatchArgs {
		return true
	}
	return a.mode == MatchAuto && slices.ContainsFunc(a.matchers, func(m matcher) bool { return m.re != nil })
}

// match is what a pane runs as an agent, or "".
func (a *agents) match(command string, fg []proc) string {
	for _, m := range a.matchers {
		useArgs := a.mode == MatchArgs || (a.mode == MatchAuto && m.re != nil)
		if !useArgs {
			if name := m.matchCommand(command); name != "" {
				return name
			}
			continue
		}
		for _, p := range fg {
			if name := m.matchProc(p, a.mode); name != "" {
				return name
			}
		}
	}
	return ""
}

func (a *agents) load(r tmuxctl.Runner) error {
	res, err := r.Run([]string{"list-panes", "-a", "-F",
		"#{session_id}\t#{window_index}\t#{pane_index}\t#{pane_id}\t#{pane_current_command}" +
			"\t#{pane_current_path}\t#{pane_title}\t#{pane_tty}\t#{session_name}"})
	if err != nil {
		return err
	}
	var procs []proc
	if a.needsProcs() {
		if procs, err = listProcs(); err != nil {
			return err
		}
	}
	foreground := map[string][]proc{}
	for _, p := range procs {
		if p.foreground() {
			foreground["/dev/"+p.tty] = append(foreground["/dev/"+p.tty], p)
		}
	}

	// grouped sessions list the same pane once per session; the popup and peek sessions are
	// groups of real ones, so the real session's listing wins
	rows := res[0].Rows()
	slices.SortStableFunc(rows, func(x, y []string) int {
		return boolInt(len(x) > 8 && strings.HasPrefix(x[8], popup.Prefix)) -
			boolInt(len(y) > 8 && strings.HasPrefix(y[8], popup.Prefix))
	})
	seen := map[string]bool{}
	a.panes = a.panes[:0]
	for _, row := range rows {
		if len(row) < 9 || seen[row[3]] {
			continue
		}
		agent := a.match(row[4], foreground[row[7]])
		if agent == "" {
			continue
		}
		seen[row[3]] = true
		w, _ := strconv.Atoi(row[1])
		i, _ := strconv.Atoi(row[2])
		a.panes = append(a.panes, agentPane{
			sessionID: row[0], window: w, index: i, id: row[3],
			path: row[5], title: row[6], tty: row[7], agent: agent,
		})
	}
	slices.SortFunc(a.panes, func(x, y agentPane) int {
		if c := idNum(x.sessionID) - idNum(y.sessionID); c != 0 {
			return c
		}
		if c := x.window - y.window; c != 0 {
			return c
		}
		return x.index - y.index
	})

	// panes are titled by directory, numbered when several share one
	total := map[string]int{}
	for i := range a.panes {
		a.panes[i].label = dirName(a.panes[i].path)
		total[a.panes[i].label]++
	}
	numbered := map[string]int{}
	for i := range a.panes {
		name := a.panes[i].label
		if total[name] > 1 {
			numbered[name]++
			a.panes[i].label = fmt.Sprintf("%s #%d", name, numbered[name])
		}
	}
	return nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func dirName(path string) string {
	if base := filepath.Base(strings.TrimRight(path, "/")); base != "." && base != "/" && base != "" {
		return base
	}
	return "/"
}

func (a *agents) rows(query string) []Row {
	th := a.th
	label, match := th.Fg("blue", true), th.Fg("peach", true)
	showAgent := len(a.matchers) > 1
	var out []Row
	n := 0
	for _, p := range a.panes {
		var lPos, tPos, aPos map[int]bool
		if query != "" {
			var ok bool
			if lPos, ok = fuzzy(query, p.label); !ok {
				if tPos, ok = fuzzy(query, p.title); !ok {
					if aPos, ok = fuzzy(query, p.agent); !ok || !showAgent {
						continue
					}
				}
			}
		}
		n++
		display := quickKey(n, th) + " " + highlight(p.label, lPos, label, match)
		if showAgent {
			display += " " + highlight(p.agent, aPos, th.Fg("overlay_1", false), match)
		}
		if p.title != "" {
			display += " " + highlight(p.title, tPos, th.Fg("overlay_0", false), match)
		}
		out = append(out, Row{ID: p.id, SessionID: p.sessionID, Display: display, Matched: query != ""})
	}
	return out
}

func (a *agents) start() string { return "" }

func (a *agents) digit(n int) string {
	if n < len(a.panes) {
		return a.panes[n].id
	}
	return ""
}

func (a *agents) pane(id string) *agentPane {
	for i := range a.panes {
		if a.panes[i].id == id {
			return &a.panes[i]
		}
	}
	return nil
}

func (a *agents) preview(r tmuxctl.Runner, id string, cols, lines int) string {
	s, err := preview.FetchSingle(r, id)
	if err != nil {
		return ""
	}
	return preview.RenderSingle(s, a.home, a.th, cols, lines)
}

func (a *agents) describe(id string) string {
	if p := a.pane(id); p != nil {
		return p.agent + " in " + dirName(p.path)
	}
	return id
}

// kill stops the agent in pane id, waiting for it to exit since the list reloads right after
// and would still show it.
func (a *agents) kill(id string) (string, func(r tmuxctl.Runner) error) {
	p := a.pane(id)
	if p == nil {
		return "", func(tmuxctl.Runner) error { return nil }
	}
	tty, matchers, mode := strings.TrimPrefix(p.tty, "/dev/"), a.matchers, a.mode
	return "", func(tmuxctl.Runner) error {
		procs, err := listProcs()
		if err != nil {
			return err
		}
		pid := agentPID(procs, tty, matchers, mode)
		if pid == 0 {
			return nil
		}
		return terminate(pid)
	}
}

// agentPID is the agent process on tty, preferring the one leading the foreground group.
func agentPID(procs []proc, tty string, matchers []matcher, mode string) int {
	best, bestFG := 0, false
	for _, p := range procs {
		if p.tty != tty {
			continue
		}
		hit := false
		for _, m := range matchers {
			if m.matchProc(p, mode) != "" {
				hit = true
				break
			}
		}
		if !hit {
			continue
		}
		fg := p.pid == p.tpgid
		if best == 0 || (fg && !bestFG) || (fg == bestFG && p.pid < best) {
			best, bestFG = p.pid, fg
		}
	}
	return best
}

func terminate(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return err
	}
	deadline := time.Now().Add(killGrace)
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		if proc.Signal(syscall.Signal(0)) != nil {
			return nil
		}
	}
	if err := proc.Signal(syscall.SIGKILL); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}

func (a *agents) accept(r tmuxctl.Runner, id string) error {
	return firstErr(r.Run(
		[]string{"select-window", "-t", id},
		[]string{"select-pane", "-t", id},
		[]string{"switch-client", "-c", a.client, "-t", id},
	))
}

func (a *agents) peek(r tmuxctl.Runner, id string) (*exec.Cmd, func() bool, error) {
	res, err := r.Run(
		[]string{"display-message", "-p", "-t", id, "#{session_id}\t#{window_id}\t#{window_zoomed_flag}"},
		[]string{"list-panes", "-t", id, "-F", "#{pane_active}\t#{pane_id}"},
	)
	if err != nil {
		return nil, nil, err
	}
	info := strings.Split(res[0].Text(), "\t")
	if len(info) < 3 {
		return nil, nil, fmt.Errorf("pane %s not found", id)
	}
	sid, wid, zoomed := info[0], info[1], info[2]
	active := id
	for _, row := range res[1].Rows() {
		if len(row) == 2 && row[0] == "1" {
			active = row[1]
		}
	}

	name := fmt.Sprintf("%speek_%d", popup.Prefix, os.Getpid())
	target := "=" + name + ":"
	cmds := [][]string{
		{"new-session", "-d", "-s", name, "-t", sid},
		{"set-option", "-t", target, "status", "off"},
		{"set-option", "-t", target, "key-table", peekTable},
		{"set-option", "-t", target, peekRestoreOpt, wid + " " + active + " " + zoomed},
		{"bind-key", "-T", peekTable, "C-c", "detach-client"},
		// one argument, so the ; separates commands inside the binding
		{"bind-key", "-T", peekTable, "C-d", "set-option " + peekQuitOpt + " 1 ; detach-client"},
		// zoom is per window, so the pane's window is zoomed onto it here and restored afterwards
		{"select-window", "-t", "=" + name + ":" + wid},
	}
	if zoomed == "1" {
		cmds = append(cmds, []string{"resize-pane", "-Z", "-t", active})
	}
	cmds = append(cmds, []string{"select-pane", "-t", id}, []string{"resize-pane", "-Z", "-t", id})
	if err := firstErr(r.Run(cmds...)); err != nil {
		unpeek(r, name)
		return nil, nil, err
	}

	attach := tmuxctl.Current().Command("attach-session", "-t", "="+name)
	done := func() bool {
		res, _ := r.Run([]string{"show-options", "-qv", "-t", target, peekQuitOpt})
		quit := len(res) > 0 && res[0].Text() == "1"
		unpeek(r, name)
		return quit
	}
	return attach, done, nil
}

// unpeek puts the pane's window back as it was and drops the peek session.
func unpeek(r tmuxctl.Runner, name string) {
	target := "=" + name + ":"
	res, err := r.Run([]string{"show-options", "-qv", "-t", target, peekRestoreOpt})
	if err == nil {
		if f := strings.Fields(res[0].Text()); len(f) == 3 {
			wid, active, zoomed := f[0], f[1], f[2]
			now, _ := r.Run([]string{"display-message", "-p", "-t", wid, "#{window_zoomed_flag}"})
			var cmds [][]string
			if len(now) > 0 && now[0].Text() == "1" {
				cmds = append(cmds, []string{"resize-pane", "-Z", "-t", wid})
			}
			cmds = append(cmds, []string{"select-pane", "-t", active})
			if zoomed == "1" {
				cmds = append(cmds, []string{"resize-pane", "-Z", "-t", active})
			}
			_, _ = r.Run(cmds...)
		}
	}
	_, _ = r.Run([]string{"kill-session", "-t", "=" + name})
}

package picker

import (
	"slices"
	"testing"

	"github.com/chenasraf/tx/internal/theme"
	"github.com/chenasraf/tx/internal/tmuxctl"
)

type panesFake struct{ rows []string }

func (f panesFake) Run(cmds ...[]string) ([]tmuxctl.Result, error) {
	out := make([]tmuxctl.Result, len(cmds))
	for i := range cmds {
		out[i] = tmuxctl.Result{Lines: f.rows}
	}
	return out, nil
}

func mustMatchers(t *testing.T, entries ...string) []matcher {
	t.Helper()
	m, err := parseMatchers(entries)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

const psOutput = `ttys001   501   501   501 -zsh
ttys002   600   700   600 -zsh
ttys002   700   700   700 node /opt/homebrew/bin/gemini --yolo
ttys003   800   810   800 -zsh
ttys003   810   810   810 claude
ttys003   811   810   810 claude --child
??          1     0     1 /sbin/launchd
`

func TestParsePS(t *testing.T) {
	procs := parsePS(psOutput)
	if len(procs) != 7 {
		t.Fatalf("got %d procs", len(procs))
	}
	g := procs[2]
	if g.tty != "ttys002" || g.pid != 700 || !g.foreground() || g.argv0 != "node" || g.args != "node /opt/homebrew/bin/gemini --yolo" {
		t.Errorf("gemini proc = %+v", g)
	}
	if procs[1].foreground() {
		t.Error("background shell counted as foreground")
	}
}

func TestMatch_Modes(t *testing.T) {
	procs := parsePS(psOutput)
	fg := func(tty string) []proc {
		var out []proc
		for _, p := range procs {
			if p.tty == tty && p.foreground() {
				out = append(out, p)
			}
		}
		return out
	}

	auto := newAgents("", theme.Mocha, mustMatchers(t, "claude", "/gemini|codex/"), MatchAuto)
	if got := auto.match("claude", nil); got != "claude" {
		t.Errorf("auto, by name: %q", got)
	}
	if got := auto.match("node", fg("ttys002")); got != "gemini" {
		t.Errorf("auto, by pattern: %q", got)
	}
	if got := auto.match("zsh", fg("ttys001")); got != "" {
		t.Errorf("auto, shell: %q", got)
	}

	name := newAgents("", theme.Mocha, mustMatchers(t, "gemini"), MatchName)
	if got := name.match("node", fg("ttys002")); got != "" {
		t.Errorf("name mode shouldn't see through node: %q", got)
	}

	args := newAgents("", theme.Mocha, mustMatchers(t, "gemini", "claude"), MatchArgs)
	if got := args.match("node", fg("ttys002")); got != "gemini" {
		t.Errorf("args mode, interpreter: %q", got)
	}
	if got := args.match("claude", fg("ttys003")); got != "claude" {
		t.Errorf("args mode, direct: %q", got)
	}
}

func TestAgentPID_PrefersForegroundLeader(t *testing.T) {
	procs := parsePS(psOutput)
	if got := agentPID(procs, "ttys003", mustMatchers(t, "claude"), MatchAuto); got != 810 {
		t.Errorf("pid = %d, want 810", got)
	}
	if got := agentPID(procs, "ttys002", mustMatchers(t, "/gemini/"), MatchAuto); got != 700 {
		t.Errorf("pid = %d, want 700", got)
	}
	if got := agentPID(procs, "ttys001", mustMatchers(t, "claude"), MatchAuto); got != 0 {
		t.Errorf("pid = %d, want none", got)
	}
}

func TestAgentRows(t *testing.T) {
	f := panesFake{rows: []string{
		// a peek session groups with $2 and lists its panes too; the real session wins
		"$9\t1\t0\t%5\tclaude\t/home/u/api\t\t/dev/ttys005\t_popup_peek_1",
		"$2\t1\t0\t%5\tclaude\t/home/u/api\t\t/dev/ttys005\twork",
		"$2\t2\t0\t%6\tzsh\t/home/u\t\t/dev/ttys006\twork",
		"$1\t1\t1\t%3\tclaude\t/home/u/api\tfix tests\t/dev/ttys003\tmain",
		"$1\t1\t0\t%2\tclaude\t/\t\t/dev/ttys002\tmain",
	}}
	a := newAgents("", theme.Mocha, mustMatchers(t, "claude"), MatchAuto)
	if err := a.load(f); err != nil {
		t.Fatal(err)
	}
	rows := a.rows("")
	if got, want := ids(rows), []string{"%2", "%3", "%5"}; !slices.Equal(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	wantText := []string{"0 /", "1 api #1 fix tests", "2 api #2"}
	for i, w := range wantText {
		if got := plain(rows[i].Display); got != w {
			t.Errorf("row %d = %q, want %q", i, got, w)
		}
	}
	if a.pane("%5").sessionID != "$2" {
		t.Errorf("deduped pane kept the peek session: %+v", a.pane("%5"))
	}
	if got := ids(a.rows("tests")); !slices.Equal(got, []string{"%3"}) {
		t.Errorf("title search: %v", got)
	}
	if got := a.digit(2); got != "%5" {
		t.Errorf("digit 2 = %s", got)
	}
	if got := a.describe("%3"); got != "claude in api" {
		t.Errorf("describe = %q", got)
	}
}

func TestParseMatchers_BadPattern(t *testing.T) {
	if _, err := parseMatchers([]string{"/(/"}); err == nil {
		t.Error("expected an error")
	}
}

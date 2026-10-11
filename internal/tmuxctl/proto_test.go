package tmuxctl

import (
	"reflect"
	"strings"
	"testing"
)

func feedAll(lines string) []event {
	var p parser
	var out []event
	for _, l := range strings.Split(strings.TrimSuffix(lines, "\n"), "\n") {
		if ev := p.feed(l); ev.kind != evNone {
			out = append(out, ev)
		}
	}
	return out
}

func TestParser_Transcript(t *testing.T) {
	events := feedAll(`%begin 1791724196 321 0
%end 1791724196 321 0
%session-changed $0 main
%begin 1791724196 323 1
main	x
%end 1791724196 323 1
%begin 1791724196 326 1
parse error: unknown command: bogus-cmd
%error 1791724196 326 1
%begin 1791724196 327 1
%end 1791724196 327 1
%window-renamed @0 my window
%exit
`)

	if len(events) != 6 {
		t.Fatalf("got %d events: %+v", len(events), events)
	}

	if events[0].kind != evNotify || events[0].note != (Notification{Name: "session-changed", Raw: "$0 main"}) {
		t.Errorf("event 0: %+v", events[0])
	}
	if events[1].kind != evReply || !reflect.DeepEqual(events[1].reply.Lines, []string{"main\tx"}) || events[1].reply.Err != nil {
		t.Errorf("event 1: %+v", events[1])
	}
	if events[2].kind != evReply || events[2].reply.Err == nil ||
		events[2].reply.Err.Error() != "tmux: parse error: unknown command: bogus-cmd" {
		t.Errorf("event 2: %+v", events[2])
	}
	if events[3].kind != evReply || len(events[3].reply.Lines) != 0 || events[3].reply.Lines == nil {
		t.Errorf("event 3: %+v", events[3])
	}
	if events[4].note != (Notification{Name: "window-renamed", Raw: "@0 my window"}) {
		t.Errorf("event 4: %+v", events[4])
	}
	if events[5].kind != evExit {
		t.Errorf("event 5: %+v", events[5])
	}
}

func TestParser_OutputThatLooksLikeProtocol(t *testing.T) {
	// pane contents can contain lines starting with %; only the block's own tag ends it
	events := feedAll(`%begin 1 5 1
%end 1 4 1
%window-add @9
%begin 2 6 1
%end 1 5 1
`)
	if len(events) != 1 {
		t.Fatalf("got %d events: %+v", len(events), events)
	}
	want := []string{"%end 1 4 1", "%window-add @9", "%begin 2 6 1"}
	if !reflect.DeepEqual(events[0].reply.Lines, want) {
		t.Errorf("lines = %q, want %q", events[0].reply.Lines, want)
	}
}

func TestQuote(t *testing.T) {
	tests := map[string]string{
		"plain":      `"plain"`,
		`a"b`:        `"a\"b"`,
		`back\slash`: `"back\\slash"`,
		"$HOME":      `"\$HOME"`,
		"~/x;y":      `"~/x;y"`,
		"#{name}":    `"#{name}"`,
		"a\nb\tc\rd": `"a\nb\tc\rd"`,
		"esc\x1b[0m": `"esc\033[0m"`,
		"漢字 é":       `"漢字 é"`,
		"":           `""`,
	}
	for in, want := range tests {
		if got := Quote(in); got != want {
			t.Errorf("Quote(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestResultRows(t *testing.T) {
	r := Result{Lines: []string{"a\tb", "", "c"}}
	want := [][]string{{"a", "b"}, {"c"}}
	if got := r.Rows(); !reflect.DeepEqual(got, want) {
		t.Errorf("Rows() = %q, want %q", got, want)
	}
}

func TestEscapeTrailingSemicolon(t *testing.T) {
	if got := escapeTrailingSemicolon("echo a;"); got != `echo a\;` {
		t.Errorf("got %q", got)
	}
	if got := escapeTrailingSemicolon("a;b"); got != "a;b" {
		t.Errorf("got %q", got)
	}
}

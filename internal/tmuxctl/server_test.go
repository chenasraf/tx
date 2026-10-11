package tmuxctl

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

// testServer starts a private tmux server with one session named "main".
func testServer(t *testing.T) Server {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	s := Server{Flags: []string{"-L", fmt.Sprintf("tmuxctl-test-%d-%d", os.Getpid(), time.Now().UnixNano())}}
	if out, err := s.Command("-f", "/dev/null", "new-session", "-d", "-s", "main", "-x", "80", "-y", "24").CombinedOutput(); err != nil {
		t.Fatalf("start server: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = s.Command("kill-server").Run() })
	return s
}

func TestExec_Batch(t *testing.T) {
	s := testServer(t)
	res, err := s.Exec(
		[]string{"display-message", "-p", "#{session_name}"},
		[]string{"set-option", "-g", "@x", "1"},
		[]string{"list-sessions", "-F", "#{session_name}\t#{session_windows}"},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"main"}, {}, {"main\t1"}}
	if len(res) != len(want) {
		t.Fatalf("got %d results: %+v", len(res), res)
	}
	for i := range want {
		if !reflect.DeepEqual(res[i].Lines, want[i]) {
			t.Errorf("result %d = %q, want %q", i, res[i].Lines, want[i])
		}
	}
}

func TestExec_TrailingSemicolonIsLiteral(t *testing.T) {
	s := testServer(t)
	if _, err := s.Exec([]string{"set-option", "-g", "@x", "a;"}); err != nil {
		t.Fatal(err)
	}
	res, err := s.Exec([]string{"show-options", "-gv", "@x"})
	if err != nil {
		t.Fatal(err)
	}
	if got := res[0].Text(); got != "a;" {
		t.Errorf("@x = %q, want %q", got, "a;")
	}
}

func TestExec_Error(t *testing.T) {
	s := testServer(t)
	_, err := s.Exec([]string{"kill-session", "-t", "=nope"})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestConn_RunAndQuoting(t *testing.T) {
	s := testServer(t)
	c, err := s.Dial("=main")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	// @want is set from argv, which tmux takes literally; @v goes through the command language.
	// tmux compares them itself, since some versions print option values escaped.
	value := "x\"y\\z $HOME ~ #{session_name}; q\ttab\nline2 漢字 \x1b[1m it's '' \"quoted\" \\n"
	if _, err := s.Exec([]string{"set-option", "-g", "@want", value}); err != nil {
		t.Fatal(err)
	}
	res, err := c.Run(
		[]string{"set-option", "-g", "@v", value},
		[]string{"display-message", "-p", "#{==:#{@v},#{@want}}"},
		[]string{"bogus-command"},
		[]string{"display-message", "-p", "#{session_name}"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Err != nil {
		t.Errorf("set: %v", res[0].Err)
	}
	if got := res[1].Text(); got != "1" {
		t.Errorf("quoted value differs from the literal one")
	}
	if res[2].Err == nil {
		t.Error("bogus command: expected an error")
	}
	if got := res[3].Text(); got != "main" {
		t.Errorf("after an error = %q, want main", got)
	}
}

func TestConn_Notifications(t *testing.T) {
	s := testServer(t)
	c, err := s.Dial("=main")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()

	if _, err := c.Run([]string{"new-session", "-d", "-s", "other"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(2 * time.Second)
	for {
		select {
		case n := <-c.Notifications():
			if n.Name == "sessions-changed" {
				return
			}
		case <-deadline:
			t.Fatal("no sessions-changed notification")
		}
	}
}

func TestConn_ClosedWhenServerExits(t *testing.T) {
	s := testServer(t)
	c, err := s.Dial("=main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Exec([]string{"kill-server"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-c.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("connection did not end")
	}
	if _, err := c.Run([]string{"display-message", "-p", "x"}); err != ErrClosed {
		t.Errorf("Run after exit: err = %v, want ErrClosed", err)
	}
}

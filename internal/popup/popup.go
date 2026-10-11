//go:build !windows

// Package popup opens tmux popups that follow the terminal's size.
//
// tmux sizes a popup once, when it opens. So the command runs in a hidden session and the popup
// only holds a client attached to it; on resize the popup is closed and reopened around the same
// session, and the app inside just sees its terminal resize, keeping all its state.
package popup

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/chenasraf/tx/internal/tmuxctl"
)

// Prefix starts the name of every hidden popup session; session lists leave these out.
const Prefix = "_popup_"

// an empty key table with no prefix: every key goes to the app, none to tmux bindings
const keyTable = "popup"

const (
	optClient = "@popup_client"
	optFlags  = "@popup_flags"
	optOwner  = "@popup_owner"
)

// resizes arrive in bursts while a window is dragged; only the last one reopens the popup
const resizeSettle = 200 * time.Millisecond

const attachTimeout = 2 * time.Second

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]`)

// SessionName is the hidden session that holds popup name on client.
func SessionName(name, client string) string {
	return Prefix + name + "_" + nonAlnum.ReplaceAllString(filepath.Base(client), "")
}

// target addresses a session exactly. set/show take a pane target, where an exact session match
// needs the trailing colon.
func target(session string) string {
	return "=" + session + ":"
}

// Open runs command in dir inside a popup on client, passing flags to display-popup, and returns
// once the popup has closed.
func Open(s tmuxctl.Server, client, name, dir string, flags, command []string) error {
	session := SessionName(name, client)
	encodedFlags, err := encodeFlags(flags)
	if err != nil {
		return err
	}

	// the stale-session kill goes last: it fails when there is none, and a failure skips the rest
	res, err := s.Exec(
		[]string{"display-message", "-p", "-c", client, "#{client_width}\t#{client_height}\t#{socket_path}"},
		[]string{"kill-session", "-t", "=" + session},
	)
	if len(res) == 0 || len(res[0].Rows()) == 0 {
		if err == nil {
			err = fmt.Errorf("client %s not found", client)
		}
		return err
	}
	info := res[0].Rows()[0]
	if len(info) < 3 {
		return fmt.Errorf("client %s not found", client)
	}
	width, height, socket := info[0], info[1], info[2]

	token := newToken()
	newSession := append([]string{"new-session", "-d", "-s", session, "-x", width, "-y", height, "-c", dir}, command...)
	cmds := [][]string{newSession}
	for _, o := range [][2]string{
		{"status", "off"},
		{"prefix", "None"},
		{"prefix2", "None"},
		{"key-table", keyTable},
		{optClient, client},
		{optFlags, encodedFlags},
		{optOwner, token},
	} {
		cmds = append(cmds, []string{"set-option", "-t", target(session), o[0], o[1]})
	}
	if _, err := s.Exec(cmds...); err != nil {
		return err
	}

	unlock, err := lock(session)
	if err != nil {
		return err
	}
	closed, err := show(s, client, session, socket, flags)
	unlock()
	if err != nil {
		return err
	}
	finish(s, closed, session, token)
	return nil
}

// Resized reopens client's popups at the client's new size.
func Resized(s tmuxctl.Server, client string) error {
	res, err := s.Exec([]string{"list-sessions", "-F", "#{session_name}\t#{" + optClient + "}\t#{" + optFlags + "}\t#{socket_path}"})
	if err != nil {
		return err
	}
	for _, row := range res[0].Rows() {
		if len(row) < 4 || !strings.HasPrefix(row[0], Prefix) || row[1] != client {
			continue
		}
		session, socket := row[0], row[3]
		flags, err := decodeFlags(row[2])
		if err != nil {
			continue
		}

		// whichever launcher last claimed the session owns it: only the owner reopens the popup or
		// kills the session once its popup is gone, so of a burst of resizes only the last acts
		token := newToken()
		if _, err := s.Exec([]string{"set-option", "-t", target(session), optOwner, token}); err != nil {
			continue
		}
		time.Sleep(resizeSettle)

		unlock, err := lock(session)
		if err != nil {
			return err
		}
		if owner(s, session) != token {
			unlock()
			continue
		}
		_, _ = s.Exec([]string{"display-popup", "-C", "-c", client})
		closed, err := show(s, client, session, socket, flags)
		unlock()
		if err != nil {
			return err
		}
		finish(s, closed, session, token)
	}
	return nil
}

// Flags are stored as base64 of their JSON: some tmux versions print option values with $, quotes
// and control characters escaped, and the URL alphabet has none of those.
func encodeFlags(flags []string) (string, error) {
	data, err := json.Marshal(flags)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeFlags(s string) ([]string, error) {
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var flags []string
	err = json.Unmarshal(data, &flags)
	return flags, err
}

func newToken() string {
	return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
}

// show opens the popup around session, returning once its client has attached. The returned
// channel closes when the popup does.
func show(s tmuxctl.Server, client, session, socket string, flags []string) (<-chan struct{}, error) {
	args := append([]string{"display-popup", "-E", "-c", client}, flags...)
	// several arguments, so tmux runs the attach directly rather than through default-shell
	args = append(args, "env", "-u", "TMUX", "tmux", "-S", socket, "attach-session", "-t", "="+session)
	popup := s.Command(args...)
	if err := popup.Start(); err != nil {
		return nil, err
	}

	closed := make(chan struct{})
	go func() {
		_ = popup.Wait()
		close(closed)
	}()
	deadline := time.Now().Add(attachTimeout)
	for time.Now().Before(deadline) && !attached(s, session) {
		select {
		case <-closed:
			return closed, nil
		case <-time.After(20 * time.Millisecond):
		}
	}
	return closed, nil
}

// attached reports whether a terminal client is attached to session. Control-mode clients don't
// count: a picker running in the session holds one, and it would keep the session looking in use.
func attached(s tmuxctl.Server, session string) bool {
	res, err := s.Exec([]string{"list-clients", "-t", "=" + session, "-F", "#{client_control_mode}"})
	if err != nil || len(res) == 0 {
		return false
	}
	return slices.Contains(res[0].Lines, "0")
}

func owner(s tmuxctl.Server, session string) string {
	res, err := s.Exec([]string{"show-options", "-qv", "-t", target(session), optOwner})
	if err != nil || len(res) == 0 {
		return ""
	}
	return res[0].Text()
}

// finish waits for the popup to close. If the app didn't exit, either a resize reopened the popup
// (and that launcher owns the session now) or the client went away, which leaves the session
// orphaned.
func finish(s tmuxctl.Server, closed <-chan struct{}, session, token string) {
	<-closed
	if owner(s, session) == token && !attached(s, session) {
		_, _ = s.Exec([]string{"kill-session", "-t", "=" + session})
	}
}

// lock serialises closing and reopening the popup, so one reopen can't close another's
// half-open popup.
func lock(session string) (func(), error) {
	f, err := os.OpenFile(filepath.Join(os.TempDir(), "tmux-"+session+".lock"), os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() { _ = f.Close() }, nil
}

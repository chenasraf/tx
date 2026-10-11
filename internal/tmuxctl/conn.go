package tmuxctl

import (
	"bufio"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// ErrClosed is returned for commands on a connection that has ended.
var ErrClosed = errors.New("tmux control connection closed")

// Conn is a control-mode client. Commands are pipelined: Run writes a whole batch before reading
// any reply, so a batch costs one round trip to the server however many commands it holds.
type Conn struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser

	mu      sync.Mutex
	waiters []chan Result
	closed  bool

	notes chan Notification
	done  chan struct{}
}

// Dial attaches a control client to session target. The client neither receives pane output
// nor takes part in sizing windows, so it doesn't disturb the sessions it reads.
func (s Server) Dial(target string) (*Conn, error) {
	cmd := s.Command("-C", "attach-session", "-t", target, "-f", "no-output,ignore-size")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	c := &Conn{
		cmd:   cmd,
		stdin: stdin,
		notes: make(chan Notification, 64),
		done:  make(chan struct{}),
	}
	go c.read(stdout)
	return c, nil
}

func (c *Conn) read(r io.Reader) {
	defer c.shutdown()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	var p parser
	for sc.Scan() {
		ev := p.feed(sc.Text())
		switch ev.kind {
		case evReply:
			c.mu.Lock()
			var w chan Result
			if len(c.waiters) > 0 {
				w, c.waiters = c.waiters[0], c.waiters[1:]
			}
			c.mu.Unlock()
			if w != nil {
				w <- ev.reply
			}
		case evNotify:
			// a full buffer already holds a pending refresh, so dropping loses nothing
			select {
			case c.notes <- ev.note:
			default:
			}
		case evExit:
			return
		}
	}
}

func (c *Conn) shutdown() {
	c.mu.Lock()
	c.closed = true
	waiters := c.waiters
	c.waiters = nil
	c.mu.Unlock()
	for _, w := range waiters {
		w <- Result{Err: ErrClosed}
	}
	close(c.done)
	_ = c.cmd.Wait()
}

// send queues cmds and returns a channel per command that receives its reply.
func (c *Conn) send(cmds [][]string) ([]chan Result, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, ErrClosed
	}
	var b strings.Builder
	chans := make([]chan Result, len(cmds))
	for i, argv := range cmds {
		b.WriteString(Line(argv))
		b.WriteByte('\n')
		chans[i] = make(chan Result, 1)
	}
	// waiters are queued under the same lock as the write, so replies pair up in order
	c.waiters = append(c.waiters, chans...)
	if _, err := io.WriteString(c.stdin, b.String()); err != nil {
		c.waiters = c.waiters[:len(c.waiters)-len(chans)]
		return nil, err
	}
	return chans, nil
}

// Run sends cmds and waits for every reply. A command tmux rejects carries its error in its
// Result; the returned error is only for a broken connection.
func (c *Conn) Run(cmds ...[]string) ([]Result, error) {
	chans, err := c.send(cmds)
	if err != nil {
		return nil, err
	}
	out := make([]Result, len(chans))
	for i, ch := range chans {
		out[i] = <-ch
		if errors.Is(out[i].Err, ErrClosed) {
			return out, ErrClosed
		}
	}
	return out, nil
}

// Send queues cmds without waiting for their replies.
func (c *Conn) Send(cmds ...[]string) error {
	_, err := c.send(cmds)
	return err
}

// Notifications delivers asynchronous messages such as %sessions-changed.
func (c *Conn) Notifications() <-chan Notification {
	return c.notes
}

// Done is closed once the connection has ended.
func (c *Conn) Done() <-chan struct{} {
	return c.done
}

// Close detaches the client and waits for it to exit.
func (c *Conn) Close() error {
	_ = c.stdin.Close()
	<-c.done
	return nil
}

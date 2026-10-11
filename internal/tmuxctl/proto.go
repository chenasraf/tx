// Package tmuxctl talks to a tmux server: a long-lived control-mode connection for interactive
// tools that issue many commands, and batched one-shot execution for everything else.
package tmuxctl

import (
	"fmt"
	"strconv"
	"strings"
)

// Result is the output of one command.
type Result struct {
	Lines []string
	// Err is set when tmux answered with %error; Lines then hold the message.
	Err error
}

// Text joins the output lines.
func (r Result) Text() string {
	return strings.Join(r.Lines, "\n")
}

// Rows splits each output line on tabs, skipping empty lines.
func (r Result) Rows() [][]string {
	out := make([][]string, 0, len(r.Lines))
	for _, l := range r.Lines {
		if l != "" {
			out = append(out, strings.Split(l, "\t"))
		}
	}
	return out
}

// CommandError is a command tmux rejected.
type CommandError struct {
	Message string
}

func (e *CommandError) Error() string {
	return "tmux: " + e.Message
}

// Notification is an asynchronous control-mode message such as %sessions-changed.
type Notification struct {
	// Name without the leading %, e.g. "window-add".
	Name string
	// Raw is everything after the name, unparsed: some notifications end in free text.
	Raw string
}

type eventKind int

const (
	evNone eventKind = iota
	evReply
	evNotify
	evExit
)

type event struct {
	kind  eventKind
	reply Result
	note  Notification
}

// parser turns control-mode output lines into replies and notifications.
//
// Every command's output is framed by %begin and %end (or %error) lines carrying the same
// "time number flags" tag. Blocks whose flags lack bit 1 come from tmux itself (the attach
// command, for one) rather than from commands this client sent, so they are not replies.
type parser struct {
	inBlock bool
	tag     string
	ours    bool
	lines   []string
}

func (p *parser) feed(line string) event {
	if p.inBlock {
		if rest, ok := strings.CutPrefix(line, "%end "); ok && rest == p.tag {
			return p.finish(nil)
		}
		if rest, ok := strings.CutPrefix(line, "%error "); ok && rest == p.tag {
			return p.finish(&CommandError{Message: strings.Join(p.lines, "\n")})
		}
		p.lines = append(p.lines, line)
		return event{}
	}

	if rest, ok := strings.CutPrefix(line, "%begin "); ok {
		p.inBlock = true
		p.tag = rest
		p.ours = blockFlags(rest)&1 != 0
		p.lines = nil
		return event{}
	}
	if line == "%exit" || strings.HasPrefix(line, "%exit ") {
		return event{kind: evExit}
	}
	if name, ok := strings.CutPrefix(line, "%"); ok {
		name, raw, _ := strings.Cut(name, " ")
		return event{kind: evNotify, note: Notification{Name: name, Raw: raw}}
	}
	return event{}
}

func (p *parser) finish(err error) event {
	p.inBlock = false
	if !p.ours {
		return event{}
	}
	lines := p.lines
	if lines == nil {
		lines = []string{}
	}
	return event{kind: evReply, reply: Result{Lines: lines, Err: err}}
}

func blockFlags(tag string) int {
	fields := strings.Fields(tag)
	if len(fields) < 3 {
		return 0
	}
	n, _ := strconv.Atoi(fields[2])
	return n
}

// Quote makes s a single tmux command-language argument, taken literally: no environment
// expansion, no ~, no command separators.
func Quote(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"', '\\', '$':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if c < 0x20 || c == 0x7f {
				fmt.Fprintf(&b, `\%03o`, c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// Line renders argv as one tmux command line.
func Line(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = Quote(a)
	}
	return strings.Join(parts, " ")
}

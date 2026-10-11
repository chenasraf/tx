package tmuxctl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Server identifies a tmux server by the global flags that select it (-L name, -S socket).
type Server struct {
	Flags []string
}

// Current is the server the calling process runs under, from $TMUX, or tmux's default server
// outside tmux. The socket is passed explicitly so nested clients, which need $TMUX unset, still
// reach the same server.
func Current() Server {
	if socket, _, _ := strings.Cut(os.Getenv("TMUX"), ","); socket != "" {
		return Server{Flags: []string{"-S", socket}}
	}
	return Server{}
}

// Command builds a tmux invocation against this server with $TMUX removed from its environment.
func (s Server) Command(args ...string) *exec.Cmd {
	cmd := exec.Command("tmux", append(append([]string{}, s.Flags...), args...)...)
	cmd.Env = withoutTMUX(os.Environ())
	return cmd
}

func withoutTMUX(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		if !strings.HasPrefix(kv, "TMUX=") {
			out = append(out, kv)
		}
	}
	return out
}

// Exec runs cmds in one tmux process, chained like `a \; b \; c`, and returns each one's output.
//
// tmux skips the rest of a chain once a command fails, so a failure returns the outputs that
// completed and the error; commands that are expected to fail belong in their own Exec.
func (s Server) Exec(cmds ...[]string) ([]Result, error) {
	if len(cmds) == 0 {
		return nil, nil
	}
	marker := fmt.Sprintf("__tmuxctl_%d__", os.Getpid())
	var args []string
	for i, c := range cmds {
		if i > 0 {
			args = append(args, ";", "display-message", "-p", marker, ";")
		}
		for _, a := range c {
			args = append(args, escapeTrailingSemicolon(a))
		}
	}

	cmd := s.Command(args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	chunks := strings.Split(stdout.String(), marker+"\n")
	results := make([]Result, 0, len(chunks))
	for _, chunk := range chunks {
		results = append(results, Result{Lines: splitLines(chunk)})
	}
	if runErr != nil {
		return results, &CommandError{Message: strings.TrimSpace(stderr.String())}
	}
	return results, nil
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}

// tmux treats an argument ending in ; as a command separator unless the ; is escaped.
func escapeTrailingSemicolon(a string) string {
	if strings.HasSuffix(a, ";") {
		return a[:len(a)-1] + `\;`
	}
	return a
}

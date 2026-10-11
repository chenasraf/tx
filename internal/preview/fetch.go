package preview

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chenasraf/tx/internal/tmuxctl"
)

const (
	windowInfoFormat = "#{session_id}\t#{window_id}\t#{window_width}\t#{window_height}\t#{window_zoomed_flag}"
	tabsFormat       = "#{session_id}\t#{window_id}\t#{window_index}\t#{window_name}"
	panesFormat      = "#{pane_id}\t#{pane_index}\t#{pane_left}\t#{pane_top}\t#{pane_width}\t#{pane_height}" +
		"\t#{pane_active}\t#{pane_floating_flag}\t#{alternate_on}\t#{pane_current_command}"
	singleFormat = "#{session_name}\t#{window_index}\t#{window_name}\t#{pane_current_path}\t#{pane_height}"
)

func capture(id string) []string {
	return []string{"capture-pane", "-p", "-e", "-t", id}
}

// FetchWindow reads what RenderWindow needs for target: a session (its current window) or a
// window.
func FetchWindow(r tmuxctl.Runner, target string) (Window, error) {
	res, err := r.Run([]string{"display-message", "-p", "-t", target, windowInfoFormat})
	if err != nil {
		return Window{}, err
	}
	info := fields(res[0], 5)
	if info == nil {
		return Window{}, fmt.Errorf("%s not found", target)
	}
	sid, wid := info[0], info[1]
	w := Window{ID: wid, Width: atoi(info[2]), Height: atoi(info[3])}
	zoomed := info[4] == "1"

	res, err = r.Run(
		[]string{"list-windows", "-t", sid, "-F", tabsFormat},
		[]string{"list-panes", "-t", wid, "-F", panesFormat},
	)
	if err != nil {
		return Window{}, err
	}
	for _, row := range res[0].Rows() {
		if len(row) >= 4 {
			w.Tabs = append(w.Tabs, Tab{ID: row[1], Index: row[2], Name: row[3]})
		}
	}
	for _, row := range res[1].Rows() {
		if len(row) < 10 {
			continue
		}
		p := Pane{
			ID: row[0], Index: row[1],
			Left: atoi(row[2]), Top: atoi(row[3]), Width: atoi(row[4]), Height: atoi(row[5]),
			Active: row[6] == "1", Floating: row[7] == "1", Alternate: row[8] == "1",
			Command: row[9],
		}
		if zoomed {
			// geometry of hidden panes is stale while zoomed, so show only what is actually visible
			if !p.Active {
				continue
			}
			p.Left, p.Top, p.Width, p.Height = 0, 0, w.Width, w.Height
		}
		w.Panes = append(w.Panes, p)
	}

	if len(w.Panes) == 0 {
		return w, nil
	}
	cmds := make([][]string, len(w.Panes))
	for i, p := range w.Panes {
		cmds[i] = capture(p.ID)
	}
	res, err = r.Run(cmds...)
	if err != nil {
		return Window{}, err
	}
	for i := range w.Panes {
		w.Panes[i].Content = res[i].Lines
	}
	return w, nil
}

// FetchSingle reads what RenderSingle needs for pane target.
func FetchSingle(r tmuxctl.Runner, target string) (Single, error) {
	res, err := r.Run(
		[]string{"display-message", "-p", "-t", target, singleFormat},
		capture(target),
	)
	if err != nil {
		return Single{}, err
	}
	info := fields(res[0], 5)
	if info == nil {
		return Single{}, fmt.Errorf("%s not found", target)
	}
	return Single{
		Session: info[0], WindowIndex: info[1], WindowName: info[2], Path: info[3],
		Height: atoi(info[4]), Content: res[1].Lines,
	}, nil
}

func fields(r tmuxctl.Result, n int) []string {
	if r.Err != nil || len(r.Lines) == 0 {
		return nil
	}
	f := strings.Split(r.Lines[0], "\t")
	if len(f) < n {
		return nil
	}
	return f
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

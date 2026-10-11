package preview

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenasraf/tx/internal/theme"
	"github.com/chenasraf/tx/internal/tmuxctl"
)

// Fixtures in testdata hold the tmux calls a preview made against a throwaway server (argv and
// stdout) and, in <name>.want, the expected rendering for that preview area.
type fixture struct {
	Target string `json:"target"`
	Cols   int    `json:"cols"`
	Lines  int    `json:"lines"`
	Home   string `json:"home"`
	Calls  []struct {
		Args   []string `json:"args"`
		Stdout string   `json:"stdout"`
	} `json:"calls"`
}

// replay answers commands from recorded calls.
type replay map[string]string

var aliases = map[string]string{"display": "display-message"}

func callKey(args []string) string {
	args = append([]string{}, args...)
	if full, ok := aliases[args[0]]; ok {
		args[0] = full
	}
	return strings.Join(args, "\x1f")
}

func (r replay) Run(cmds ...[]string) ([]tmuxctl.Result, error) {
	out := make([]tmuxctl.Result, len(cmds))
	for i, c := range cmds {
		stdout, ok := r[callKey(c)]
		if !ok {
			return nil, fmt.Errorf("unrecorded command %q", c)
		}
		lines := []string{}
		if s := strings.TrimSuffix(stdout, "\n"); s != "" {
			lines = strings.Split(s, "\n")
		}
		out[i] = tmuxctl.Result{Lines: lines}
	}
	return out, nil
}

func loadFixture(t *testing.T, path string) (fixture, replay, string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f fixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	r := replay{}
	for _, c := range f.Calls {
		r[callKey(c.Args)] = c.Stdout
	}
	want, err := os.ReadFile(strings.TrimSuffix(path, ".json") + ".want")
	if err != nil {
		t.Fatal(err)
	}
	return f, r, string(want)
}

func render(t testing.TB, f fixture, r tmuxctl.Runner) string {
	th, err := theme.Load(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(f.Target, "%") {
		s, err := FetchSingle(r, f.Target)
		if err != nil {
			t.Fatal(err)
		}
		return RenderSingle(s, f.Home, th, f.Cols, f.Lines)
	}
	w, err := FetchWindow(r, f.Target)
	if err != nil {
		t.Fatal(err)
	}
	return RenderWindow(w, th, f.Cols, f.Lines)
}

func TestFixtures(t *testing.T) {
	paths, _ := filepath.Glob("testdata/*.json")
	if len(paths) == 0 {
		t.Fatal("no fixtures")
	}
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".json")
		t.Run(name, func(t *testing.T) {
			f, r, want := loadFixture(t, path)
			got := render(t, f, r)
			if got == want {
				return
			}
			gotLines, wantLines := strings.Split(got, "\n"), strings.Split(want, "\n")
			for i := 0; i < max(len(gotLines), len(wantLines)); i++ {
				var g, w string
				if i < len(gotLines) {
					g = gotLines[i]
				}
				if i < len(wantLines) {
					w = wantLines[i]
				}
				if g != w {
					t.Fatalf("line %d differs\n got: %q\nwant: %q", i, g, w)
				}
			}
		})
	}
}

func BenchmarkRenderWindow(b *testing.B) {
	data, _ := os.ReadFile("testdata/three-panes.json")
	var f fixture
	if err := json.Unmarshal(data, &f); err != nil {
		b.Fatal(err)
	}
	r := replay{}
	for _, c := range f.Calls {
		r[callKey(c.Args)] = c.Stdout
	}
	th, _ := theme.Load(r, nil)
	w, err := FetchWindow(r, f.Target)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = RenderWindow(w, th, 120, 40)
	}
}

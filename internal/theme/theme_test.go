package theme

import "testing"

func TestParse_Precedence(t *testing.T) {
	// sky from @thm_, blue overridden by config, the rest missing so Mocha
	line := "#000001\t\t\t#000002"
	th := Parse(line, map[string]string{"blue": "#ffffff"})
	if th["sky"] != "#000001" {
		t.Errorf("sky = %s", th["sky"])
	}
	if th["blue"] != "#ffffff" {
		t.Errorf("blue = %s", th["blue"])
	}
	if th["red"] != Mocha["red"] {
		t.Errorf("red = %s", th["red"])
	}
}

func TestFg(t *testing.T) {
	tests := []struct {
		hex  string
		bold bool
		want string
	}{
		{"#89dceb", false, "\x1b[38;2;137;220;235m"},
		{"#89dceb", true, "\x1b[1;38;2;137;220;235m"},
		{"red", true, "\x1b[1m"},
		{"", false, ""},
		{"#zzzzzz", false, ""},
	}
	for _, tt := range tests {
		if got := Fg(tt.hex, tt.bold); got != tt.want {
			t.Errorf("Fg(%q, %v) = %q, want %q", tt.hex, tt.bold, got, tt.want)
		}
	}
}

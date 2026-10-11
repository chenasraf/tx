//go:build !windows

package popup

import (
	"slices"
	"strings"
	"testing"
)

func TestFlagsRoundTrip(t *testing.T) {
	flags := []string{"-w", "90%", "-T", ` it's "lazygit" $x `, "-b", "rounded"}
	enc, err := encodeFlags(flags)
	if err != nil {
		t.Fatal(err)
	}
	// nothing any tmux version escapes when printing an option value
	if strings.ContainsAny(enc, `$"'\%`) {
		t.Errorf("encoded flags %q hold characters tmux may escape", enc)
	}
	got, err := decodeFlags(enc)
	if err != nil || !slices.Equal(got, flags) {
		t.Errorf("decoded %q, %v", got, err)
	}
}

func TestSessionName(t *testing.T) {
	if got := SessionName("lazygit", "/dev/ttys001"); got != "_popup_lazygit_ttys001" {
		t.Errorf("got %q", got)
	}
	if got := SessionName("sessions", "client-1.2"); got != "_popup_sessions_client12" {
		t.Errorf("got %q", got)
	}
}

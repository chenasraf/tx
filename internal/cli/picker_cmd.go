package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/picker"
	"github.com/chenasraf/tx/internal/popup"
	"github.com/chenasraf/tx/internal/tmuxctl"
	"github.com/spf13/cobra"
)

const defaultRefresh = time.Second

// pickerFlags are shared by the session and agent pickers.
type pickerFlags struct {
	client  string
	popup   bool
	size    int
	refresh time.Duration
}

func addPickerFlags(cmd *cobra.Command, f *pickerFlags) {
	cmd.Flags().StringVar(&f.client, "client", "", "tmux client that Enter switches (default: the one viewing this pane)")
	cmd.Flags().BoolVar(&f.popup, "popup", false, "open in a popup that follows the terminal's size")
	cmd.Flags().IntVar(&f.size, "size", 0, "popup size: 0 for borderless full screen, 1-100 for a percentage (default: $TMUX_SWITCHER_SIZE, then config)")
	cmd.Flags().DurationVar(&f.refresh, "refresh", defaultRefresh, "how often the preview re-captures pane contents")
	// pickers load only the global config, and no layouts
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error { return nil }
}

func pickerConfig() *config.PickerConfig {
	gc, err := config.GetGlobalConfig()
	if err != nil || gc == nil || gc.Picker == nil {
		return &config.PickerConfig{}
	}
	return gc.Picker
}

// popupSize resolves --size, then $TMUX_SWITCHER_SIZE, then config, clamped to 0-100.
func popupSize(cmd *cobra.Command, f *pickerFlags, pc *config.PickerConfig) int {
	size := 0
	if cmd.Flags().Changed("size") {
		size = f.size
	} else if env, err := strconv.Atoi(os.Getenv("TMUX_SWITCHER_SIZE")); err == nil {
		size = env
	} else if pc.Size != nil {
		size = *pc.Size
	}
	return max(0, min(100, size))
}

func refreshInterval(cmd *cobra.Command, f *pickerFlags, pc *config.PickerConfig) (time.Duration, error) {
	if cmd.Flags().Changed("refresh") || pc.Refresh == "" {
		return f.refresh, nil
	}
	d, err := time.ParseDuration(pc.Refresh)
	if err != nil {
		return 0, NewUserError(fmt.Sprintf("invalid picker refresh %q: %v", pc.Refresh, err))
	}
	return d, nil
}

// openInPopup reruns the picker command inside a popup on the client.
func openInPopup(cmd *cobra.Command, f *pickerFlags, pc *config.PickerConfig, title string, extra []string) error {
	if f.client == "" {
		return NewUserError("--popup needs --client, e.g. --client #{client_name}")
	}
	var frame []string
	if size := popupSize(cmd, f, pc); size > 0 {
		pct := fmt.Sprintf("%d%%", size)
		frame = []string{"-w", pct, "-h", pct, "-b", "rounded", "-T", " " + title + " "}
	} else {
		frame = []string{"-w", "100%", "-h", "100%", "-B"}
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	inner := []string{self, cmd.Name(), "--client", f.client}
	if cmd.Flags().Changed("refresh") {
		inner = append(inner, "--refresh", f.refresh.String())
	}
	inner = append(inner, extra...)
	home, _ := os.UserHomeDir()
	return popup.Open(tmuxctl.Current(), f.client, cmd.Name(), home, frame, inner)
}

func pickerError(err error) error {
	if err == picker.ErrNotInTmux {
		return NewUserError(err.Error() + "; use --popup --client to open it from a binding")
	}
	return err
}

var sessionsFlags pickerFlags

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Pick a tmux session or window, with a live preview",
	Long: `Pick a tmux session or window from a tree, with a live preview of its pane layout and contents.

Keys: ↑↓/jk move, →←/lh expand/collapse, enter switch, 0-9 switch to the numbered session,
/ search, esc leave search, dd kill, q quit.

Bind it in tmux.conf:
  bind s run-shell "tx sessions --popup --client #{client_name}"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		pc := pickerConfig()
		if sessionsFlags.popup {
			return openInPopup(cmd, &sessionsFlags, pc, "sessions", nil)
		}
		refresh, err := refreshInterval(cmd, &sessionsFlags, pc)
		if err != nil {
			return err
		}
		pct := pc.PreviewWidth.Sessions
		if pct <= 0 || pct >= 100 {
			pct = 78
		}
		return pickerError(picker.Sessions(picker.Options{
			Client:         sessionsFlags.client,
			Refresh:        refresh,
			PreviewPercent: pct,
			Colors:         pc.Colors,
		}))
	},
}

func init() {
	addPickerFlags(sessionsCmd, &sessionsFlags)
}

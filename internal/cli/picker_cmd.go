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
	_ = cmd.RegisterFlagCompletionFunc("client", completeClients)
}

func completeClients(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	res, err := tmuxctl.Current().Exec([]string{"list-clients", "-F", "#{client_name}"})
	if err != nil || len(res) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return res[0].Lines, cobra.ShellCompDirectiveNoFileComp
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

var (
	agentsFlags    pickerFlags
	agentsCommands []string
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Pick a pane running a coding agent, with a live preview",
	Long: `Pick a pane running a coding agent (Claude Code by default), with a live preview of it.

Keys: ↑↓/jk move, enter switch, 0-9 switch to the numbered pane, i work in the pane from the
picker (^c back to the list, ^d close), / search, esc leave search, dd kill the agent, q quit.

Which panes count as agents comes from --command, then .config.picker.agents.commands:
process names (claude) match the pane's process; /regex/ entries match its foreground command
line, for agents started through an interpreter (/gemini|codex/).

Bind it in tmux.conf:
  bind a run-shell "tx agents --popup --client #{client_name}"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		pc := pickerConfig()
		if agentsFlags.popup {
			var extra []string
			if cmd.Flags().Changed("command") {
				for _, c := range agentsCommands {
					extra = append(extra, "--command", c)
				}
			}
			return openInPopup(cmd, &agentsFlags, pc, "agents", extra)
		}
		refresh, err := refreshInterval(cmd, &agentsFlags, pc)
		if err != nil {
			return err
		}
		pct := pc.PreviewWidth.Agents
		if pct <= 0 || pct >= 100 {
			pct = 70
		}
		commands := pc.Agents.Commands
		if cmd.Flags().Changed("command") || len(commands) == 0 {
			commands = agentsCommands
		}
		return pickerError(picker.Agents(picker.Options{
			Client:         agentsFlags.client,
			Refresh:        refresh,
			PreviewPercent: pct,
			Colors:         pc.Colors,
		}, commands, pc.Agents.Match))
	},
}

func init() {
	addPickerFlags(sessionsCmd, &sessionsFlags)
	addPickerFlags(agentsCmd, &agentsFlags)
	agentsCmd.Flags().StringArrayVar(&agentsCommands, "command", []string{"claude"}, "agent process name or /regex/ over the command line (repeatable)")
}

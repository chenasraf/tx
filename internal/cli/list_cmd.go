package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/tmux"
	"github.com/spf13/cobra"
)

var (
	listBare     bool
	listSessions bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all tmux configurations and sessions",
	RunE:    runList,
}

func init() {
	listCmd.Flags().BoolVarP(&listBare, "bare", "b", false, "Show only configuration names (useful for scripting)")
	listCmd.Flags().BoolVarP(&listSessions, "sessions", "s", false, "Show only tmux sessions")
}

func runList(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	configInfo, err := config.GetTmuxConfigFileInfo()
	if err != nil {
		return err
	}

	rawConfig, err := config.GetTmuxConfig()
	if err != nil {
		return err
	}

	// Get sorted keys
	keys := make([]string, 0, len(rawConfig))
	for k := range rawConfig {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})

	// Bare mode - just print keys
	if listBare {
		for _, k := range keys {
			fmt.Println(k)
		}
		return nil
	}

	// Get sessions info
	sessionsOutput, err := tmux.ListSessions(opts)
	sessionsStr := ""
	if err == nil && sessionsOutput != "" {
		// Format sessions output
		lines := strings.Split(strings.TrimSpace(sessionsOutput), "\n")
		for _, line := range lines {
			if line != "" {
				sessionsStr += "  " + line + "\n"
			}
		}
	} else {
		sessionsStr = "  No tmux sessions\n"
	}

	// Sessions only mode
	if listSessions {
		fmt.Println(sessionsStr)
		return nil
	}

	// Full output
	fmt.Println("tmux sessions:")
	fmt.Println()
	fmt.Print(sessionsStr)
	fmt.Println()

	fmt.Println("tmux config files:")
	fmt.Println()
	if configInfo.Global != nil {
		fmt.Println("  global:", configInfo.Global.Filepath)
	}
	if configInfo.Local != nil {
		fmt.Println("  local:", configInfo.Local.Filepath)
	}
	fmt.Println()

	fmt.Println("tmux configurations:")
	fmt.Println()
	for _, k := range keys {
		item := rawConfig[k]
		if len(item.Aliases) > 0 {
			fmt.Printf("  - %s \033[2m%s\033[0m\n", k, strings.Join(item.Aliases, ", "))
		} else {
			fmt.Println("  -", k)
		}
	}

	return nil
}

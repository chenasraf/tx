package cli

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/table"
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
		sessionsStr = formatSessionsTable(sessionsOutput, "  ")
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
	for _, inc := range configInfo.Included {
		fmt.Println("  included:", inc.Filepath)
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

var sessionLineRe = regexp.MustCompile(`^([^:]+):\s*(\d+)\s+windows?\s*\(created\s+([^)]+)\)\s*(.*)$`)

// formatSessionsTable parses `tmux ls` output and renders it as a bordered
// table with headers. Each output line is prefixed with indent.
func formatSessionsTable(raw, indent string) string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	rows := make([][]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		m := sessionLineRe.FindStringSubmatch(line)
		if m == nil {
			// Fallback: dump the whole line into the Name column
			rows = append(rows, []string{line, "", "", ""})
			continue
		}
		status := strings.TrimSpace(m[4])
		status = strings.TrimPrefix(status, "(")
		status = strings.TrimSuffix(status, ")")
		rows = append(rows, []string{m[1], m[2], m[3], status})
	}
	return table.Render([]string{"Name", "Windows", "Created", "Status"}, rows, indent)
}

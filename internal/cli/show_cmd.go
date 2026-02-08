package cli

import (
	"encoding/json"
	"fmt"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/fzf"
	"github.com/spf13/cobra"
)

var showJSON bool

var showCmd = &cobra.Command{
	Use:               "show [key]",
	Aliases:           []string{"s"},
	Short:             "Show the tmux configuration for a specific key",
	Args:              cobra.MaximumNArgs(1),
	RunE:              runShow,
	ValidArgsFunction: completeSessionNames,
}

func init() {
	showCmd.Flags().BoolVarP(&showJSON, "json", "j", false, "Output as JSON")
}

func runShow(cmd *cobra.Command, args []string) error {
	allConfig, err := config.GetTmuxConfig()
	if err != nil {
		return err
	}

	var key string
	if len(args) > 0 {
		key = args[0]
	}

	// If no key, use fzf
	if key == "" {
		keys := make([]string, 0, len(allConfig))
		for k := range allConfig {
			keys = append(keys, k)
		}

		selected, err := fzf.Run(keys, fzf.Options{})
		if err != nil {
			return err
		}
		key = selected
	}

	item, actualKey, exists := allConfig.Get(key)
	if !exists {
		return NewUserError("tmux config item '" + key + "' not found")
	}

	parsed := config.ParseConfig(actualKey, item)

	if showJSON {
		data, err := json.Marshal(parsed)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	} else {
		// Pretty print
		fmt.Printf("Name: %s\n", parsed.Name)
		fmt.Printf("Root: %s\n", parsed.Root)
		fmt.Println("Windows:")
		for _, w := range parsed.Windows {
			fmt.Printf("  - %s (%s)\n", w.Name, w.Cwd)
			printLayout(w.Layout, "      ")
		}
	}

	return nil
}

func printLayout(layout config.TmuxPaneLayout, indent string) {
	if layout.Cmd != "" {
		fmt.Printf("%sCmd: %s\n", indent, layout.Cmd)
	}
	if layout.Zoom {
		fmt.Printf("%sZoom: true\n", indent)
	}
	if layout.Split != nil {
		fmt.Printf("%sSplit: %s\n", indent, layout.Split.Direction)
		if layout.Split.Child != nil {
			printLayout(*layout.Split.Child, indent+"  ")
		}
	}
}

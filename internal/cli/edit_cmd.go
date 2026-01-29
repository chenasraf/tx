package cli

import (
	"os"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
	"github.com/spf13/cobra"
)

var editLocal bool

var editCmd = &cobra.Command{
	Use:     "edit",
	Aliases: []string{"e"},
	Short:   "Edit the tmux configuration file",
	RunE:    runEdit,
}

func init() {
	editCmd.Flags().BoolVarP(&editLocal, "local", "l", false, "Edit the local config file")
}

func runEdit(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	configInfo, err := config.GetTmuxConfigFileInfo()
	if err != nil {
		return err
	}

	var filepath string
	if editLocal {
		if configInfo.Local == nil {
			return NewUserError("local config file not found")
		}
		filepath = configInfo.Local.Filepath
	} else {
		if configInfo.Global == nil {
			return NewUserError("global config file not found")
		}
		filepath = configInfo.Global.Filepath
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	return exec.RunCommand(opts, editor+" "+filepath)
}

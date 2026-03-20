package cli

import (
	"os"

	"github.com/chenasraf/tx/internal/config"
	"github.com/chenasraf/tx/internal/exec"
	"github.com/spf13/cobra"
)

var editConfigFile string

var editCmd = &cobra.Command{
	Use:     "edit",
	Aliases: []string{"e"},
	Short:   "Edit the tmux configuration file",
	RunE:    runEdit,
}

func init() {
	editCmd.Flags().StringVarP(&editConfigFile, "config", "c", "", "Edit a specific config file")
}

func runEdit(cmd *cobra.Command, args []string) error {
	opts := GetOpts()

	var filepath string
	if editConfigFile != "" {
		filepath = config.DirFix(editConfigFile)
	} else {
		configInfo, err := config.GetTmuxConfigFileInfo()
		if err != nil {
			return err
		}
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

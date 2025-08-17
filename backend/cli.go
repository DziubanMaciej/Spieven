package backend

import (
	"time"

	"github.com/spf13/cobra"
)

func CreateCliCommand() *cobra.Command {
	var (
		frequentTrim           bool
		remote                 bool
		displayKillGracePeriod int
	)
	command := &cobra.Command{
		Use:   "serve [OPTIONS...]",
		Short: "Launch Spieven backend engine.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			displayKillGracePeriod := time.Millisecond * time.Duration(displayKillGracePeriod)
			return RunServer(frequentTrim, remote, displayKillGracePeriod)
		},
	}
	command.Flags().BoolVarP(&frequentTrim, "frequent-trim", "t", false, "Enable very frequent resource trimming. This flag should only be used for testing purposes")
	command.Flags().BoolVarP(&remote, "remote", "r", false, "Allow connections from remote addresses")
	command.Flags().IntVarP(&displayKillGracePeriod, "display-kill-grace-period", "g", 1000, "Delay in milliseconds before killing all tasks related to a display that has been closed")
	return command
}

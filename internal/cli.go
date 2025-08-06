package internal

import (
	"fmt"
	"supervisor/common"

	"github.com/spf13/cobra"
)

func CreateCliCommands() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "internal",
		Args:          cobra.ExactArgs(0),
		SilenceErrors: true,
		Hidden:        true,
	}

	watchxorgCmd := &cobra.Command{
		Use:  "watchxorg DISPLAY",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dpyName := args[0]
			dpy := common.TryConnectXorg(dpyName)
			if dpy == nil {
				return fmt.Errorf("could not connect to xorg %v", dpyName)
			}

			fmt.Printf("Connected to xorg %v\n", dpyName)
			common.WatchXorgActive(dpy)
			common.DisconnectXorg(dpy)
			fmt.Printf("Disconnected from xorg %v\n", dpyName)
			return nil
		},
	}

	rootCmd.AddCommand(watchxorgCmd)

	return rootCmd
}

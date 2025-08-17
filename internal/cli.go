package internal

import (
	"fmt"
	"spieven/common"

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
			err := common.LoadXorgLibs()
			if err != nil {
				return err
			}
			defer common.UnloadXorgLibs()

			dpyName := args[0]
			dpy, err := common.TryConnectXorg(dpyName)
			if err != nil {
				return err
			}
			defer common.DisconnectXorg(dpy)

			fmt.Printf("Connected to xorg %v\n", dpyName)
			common.WatchXorgActive(dpy)
			fmt.Printf("Disconnected from xorg %v\n", dpyName)
			return nil
		},
	}

	rootCmd.AddCommand(watchxorgCmd)

	return rootCmd
}

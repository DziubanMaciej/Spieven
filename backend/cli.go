package backend

import "github.com/spf13/cobra"

func CreateCliCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "serve",
		Short: "Launch Spieven backend engine",
		RunE: func(cmd *cobra.Command, args []string) error {
			frequentTrim, err := cmd.Flags().GetBool("frequent-trim")
			if err != nil {
				return err
			}
			remote, err := cmd.Flags().GetBool("remote")
			if err != nil {
				return err
			}

			return RunServer(frequentTrim, remote)
		},
	}
	command.Flags().BoolP("frequent-trim", "t", false, "Enable very frequent resource trimming. This flag should only be used for testing purposes")
	command.Flags().BoolP("remote", "r", false, "Allow connections from remote addresses")
	return command
}

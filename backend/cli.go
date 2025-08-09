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

			return RunServer(frequentTrim)
		},
	}
	command.Flags().BoolP("frequent-trim", "t", false, "Enable very frequent resource trimming. This flag should only be used for testing purposes")
	return command
}

package backend

import "github.com/spf13/cobra"

func CreateCliCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "Spieven [command] [OPTIONS]",
		Short: "Spieven - a process supervisor for Linux",
		RunE: func(cmd *cobra.Command, args []string) error {
			frequentTrim, err := cmd.Flags().GetBool("frequentTrim")
			if err != nil {
				return err
			}

			return RunServer(frequentTrim)
		},
	}
	command.Flags().BoolP("frequentTrim", "t", false, "Enable very frequent resource trimming. This flag should only be used for testing purposes")
	return command
}

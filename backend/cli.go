package backend

import "github.com/spf13/cobra"

func CreateCliCommand() *cobra.Command {
	var (
		frequentTrim bool
		remote       bool
	)
	command := &cobra.Command{
		Use:   "serve",
		Short: "Launch Spieven backend engine",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunServer(frequentTrim, remote)
		},
	}
	command.Flags().BoolVarP(&frequentTrim, "frequent-trim", "t", false, "Enable very frequent resource trimming. This flag should only be used for testing purposes")
	command.Flags().BoolVarP(&remote, "remote", "r", false, "Allow connections from remote addresses")
	return command
}

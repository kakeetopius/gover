package cmd

import "github.com/spf13/cobra"

func CheckCommand() *cobra.Command {
	checkCommand := &cobra.Command{
		Use:          "check",
		Short:        "Check if there are newer go releases.",
		SilenceUsage: true,
	}

	return checkCommand
}

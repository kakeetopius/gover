package cmd

import "github.com/spf13/cobra"

func UpdateCommand() *cobra.Command {
	checkCommand := &cobra.Command{
		Use:          "update",
		Short:        "Update the go version if there are newer releases.",
		SilenceUsage: true,
	}

	return checkCommand
}

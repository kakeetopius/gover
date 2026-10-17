package cmd

import "github.com/spf13/cobra"

func Command() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "gover",
		Short:        "A tool for managing go versions.",
		SilenceUsage: true,
	}

	rootCmd.AddCommand(
		CheckCommand(),
		UpdateCommand(),
	)

	return rootCmd
}

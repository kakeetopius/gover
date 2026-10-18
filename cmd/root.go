// Package cmd contains gover's version management logic.
package cmd

import "github.com/spf13/cobra"

func Command() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "gover",
		Short:        "A tool for managing go versions.",
		SilenceUsage: true,
	}

	rootCmd.AddCommand(
		UpdateCommand(),
		SwitchCommand(),
		ListCommand(),
	)

	return rootCmd
}

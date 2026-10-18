// Package cmd contains gover's version management logic.
package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

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
		PurgeCommand(),
	)

	return rootCmd
}

// versionStringAndNumber accepts a string that MUST be in the form go1.22.8 or 1.22.8
// in either case the first string returned will be go1.22.8 and the second will be 1.22.8
// Note: no checks are done to determine if the string s is valid go version.
func versionStringAndNumber(s string) (string, string) {
	versionNum := s
	var versionStr string

	if strings.HasPrefix(versionNum, "go") {
		versionStr = versionNum
		versionNum = strings.TrimPrefix(versionNum, "go")
	} else {
		versionStr = fmt.Sprintf("go%v", versionNum)
	}

	return versionStr, versionNum
}

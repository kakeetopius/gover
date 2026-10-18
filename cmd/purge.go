package cmd

import (
	"fmt"
	"go/version"

	"github.com/kakeetopius/gover/util"
	"github.com/spf13/cobra"
)

func PurgeCommand() *cobra.Command {
	purgeCommand := &cobra.Command{
		Use:          "purge version",
		Short:        "Remove a particular go version.",
		SilenceUsage: true,
		RunE:         runPurge,
	}

	purgeCommand.Flags().BoolP("all", "a", false, "Remove all go versions installed by gover.")

	return purgeCommand
}

func runPurge(cmd *cobra.Command, args []string) error {
	purgeAll, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}

	toPurge := args
	if purgeAll {
		toPurge, err = util.GetInstalledVersions()
		if err != nil {
			return err
		}
	}

	if len(toPurge) == 0 {
		if purgeAll {
			return fmt.Errorf("no go versions found that were installed by gover.")
		}
		return fmt.Errorf("at least one go version is required")
	}

	for _, v := range toPurge {
		versionStr, versionNum := versionStringAndNumber(v)

		if !version.IsValid(versionStr) {
			fmt.Printf("invalid go version: %v\n", versionNum)
			continue
		}

		isInstalled, err := util.CheckIfGoVersionIsInstalled(versionNum)
		if err != nil {
			return fmt.Errorf("could not purge go version %v: %w", versionNum, err)
		}
		if !isInstalled {
			fmt.Printf("go version %v is not installed\n", versionNum)
			continue
		}

		err = util.DeleteVersionDirectory(versionNum)
		if err != nil {
			return fmt.Errorf("could not purge go version %v: %w", versionNum, err)
		}

		fmt.Println("Successfully purged go version", versionNum)
	}

	return nil
}

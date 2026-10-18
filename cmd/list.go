package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/kakeetopius/gover/releases"
	"github.com/kakeetopius/gover/util"
	"github.com/spf13/cobra"
)

func ListCommand() *cobra.Command {
	listCommand := &cobra.Command{
		Use:          "list",
		Short:        "List available go versions. By default it lists only versions available offline.",
		SilenceUsage: true,
		RunE:         runList,
	}

	listCommand.Flags().BoolP("all", "a", false, "List all available go versions even those not installed locally.")

	return listCommand
}

func runList(cmd *cobra.Command, args []string) error {
	all, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}

	var versions []string
	var heading string
	if all {
		heading = "All Go Versions"
		fmt.Println("Getting all go versions...")
		releases, err := releases.Get(context.Background(), true)
		if err != nil {
			return err
		}
		versions = releases.VersionsSlice()
	} else {
		heading = "Go Versions Installed by gover"
		versions, err = util.GetInstalledVersions()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	if len(versions) == 0 {
		fmt.Println("No go versions found that were installed by gover.")
		return nil
	}

	fmt.Println(heading)
	for _, v := range versions {
		fmt.Println(v)
	}

	return nil
}

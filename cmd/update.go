package cmd

import (
	"context"
	"errors"
	"fmt"
	"go/version"
	"path/filepath"

	"github.com/kakeetopius/gover/releases"
	"github.com/kakeetopius/gover/util"
	"github.com/spf13/cobra"
)

func UpdateCommand() *cobra.Command {
	updateCommand := &cobra.Command{
		Use:          "update",
		Short:        "Update or Install to the latest stable go version",
		SilenceUsage: true,
		RunE:         runUpdate,
	}

	updateCommand.Flags().Bool("unstable", false, "Allow updating to an unstable version if it is the latest one.")

	return updateCommand
}

func runUpdate(cmd *cobra.Command, args []string) error {
	allowUnstable, err := cmd.Flags().GetBool("unstable")
	if err != nil {
		return err
	}

	fmt.Println("Getting go versions...")
	allReleases, err := releases.Get(context.Background(), false)
	if err != nil {
		return err
	}
	if !allowUnstable {
		allReleases = allReleases.Filter(releases.FilterStable)
	}

	goos, goarch := util.OsAndArch()

	releease := allReleases.First() // returns the latest version
	if releease == nil {
		return fmt.Errorf("could not get the latest go release for %v/%v", goos, goarch)
	}

	releaseFile := releease.
		Filter(releases.FilterOS(goos)).
		Filter(releases.FilterArch(goarch)).
		Filter(releases.FilterFileKind("archive")).
		FirstFile()

	if releaseFile == nil {
		return fmt.Errorf("could not get the latest go release for %v/%v", goos, goarch)
	}
	versionNum := releaseFile.VersionNum()

	fmt.Printf("latest go version is %v\n", versionNum)

	currentVersion, err := util.GetCurrentGoVersion()
	if err != nil && !errors.Is(err, util.ErrCouldNotGetVersion) && !errors.Is(err, util.ErrGoNotInstalled) {
		return err
	}

	binDir, err := util.GetDefaultBinDirectory()
	if err != nil {
		return err
	}

	installDir, err := util.GetInstallDirectory()
	if err != nil {
		return err
	}

	if version.Compare(currentVersion, releease.Version) >= 0 {
		fmt.Println("Already at the latest go version. Nothing to update.")
		return nil
	}

	alreadyInstalled, err := util.CheckIfGoVersionIsInstalled(versionNum)
	if err != nil {
		return err
	}
	if alreadyInstalled {
		err = util.SymLinkGoBinary(versionNum, binDir)
		if err != nil {
			return err
		}
		fmt.Printf("The latest go version %v is now active.\n", versionNum)
		fmt.Printf("Make sure to add %v to the environment variable PATH.\n", binDir)
		return nil
	}

	fmt.Println("Downloading...")
	err = releaseFile.Download(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Download Complete.")

	fmt.Println("Verifying Cheksum...")
	err = releaseFile.VerifyChecksum()
	if err != nil {
		return err
	}
	fmt.Println("Cheksum okay.")

	fmt.Println("Extracting go source...")
	err = releaseFile.ExtractTo(installDir)
	if err != nil {
		return err
	}
	fmt.Println("Extraction Complete")

	err = util.SymLinkGoBinary(versionNum, binDir)
	if err != nil {
		return err
	}
	fmt.Println("The latest go source has been installed to:", filepath.Join(installDir, versionNum))
	fmt.Println("The latest go binary has been symlinked to the directory:", binDir)
	fmt.Println("Make sure to prepend it to the PATH environment variable.")

	return nil
}

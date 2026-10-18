package cmd

import (
	"context"
	"errors"
	"fmt"
	"go/version"

	"github.com/kakeetopius/gover/releases"
	"github.com/kakeetopius/gover/util"
	"github.com/spf13/cobra"
)

func SwitchCommand() *cobra.Command {
	switchCommand := &cobra.Command{
		Use:          "switch version",
		Short:        "Switch to a specific go version.",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE:         runSwitch,
	}

	return switchCommand
}

func runSwitch(cmd *cobra.Command, args []string) error {
	currentVersion, err := util.GetCurrentGoVersion()
	if err != nil && !errors.Is(err, util.ErrCouldNotGetVersion) && !errors.Is(err, util.ErrGoNotInstalled) {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("no new version given")
	}
	wantedVersion, wantedVersionNum := versionStringAndNumber(args[0])

	if !version.IsValid(wantedVersion) {
		return fmt.Errorf("invalid go version: %v", wantedVersionNum)
	}

	if version.Compare(wantedVersion, currentVersion) == 0 {
		fmt.Printf("The version go %v is already the active version.\n", wantedVersionNum)
		return nil
	}

	binDir, err := util.GetDefaultBinDirectory()
	if err != nil {
		return err
	}
	installDir, err := util.GetInstallDirectory()
	if err != nil {
		return err
	}
	alreadyInstalled, err := util.CheckIfGoVersionIsInstalled(wantedVersionNum)
	if err != nil {
		return err
	}

	if alreadyInstalled {
		err = util.SymLinkGoBinary(wantedVersionNum, "")
		if err != nil {
			return err
		}
		fmt.Printf("go version %v is now active\n", wantedVersionNum)
		fmt.Printf("Make sure to prepend %v to the PATH environment variable.\n", binDir)
		return nil
	}

	fmt.Printf("go version %v not found offline\n", wantedVersionNum)

	fmt.Println("Getting go versions...")
	allReleases, err := releases.Get(context.Background(), true)
	if err != nil {
		return err
	}

	goos, goarch := util.OsAndArch()

	release := allReleases.
		Filter(releases.FilterVersion(wantedVersion)).
		First()

	if release == nil {
		return fmt.Errorf("Could not find go version %v for %v/%v", wantedVersionNum, goos, goarch)
	}

	releaseArchive := release.
		Filter(releases.FilterOS(goos)).
		Filter(releases.FilterArch(goarch)).
		Filter(releases.FilterFileKind("archive")).
		FirstFile()
	if releaseArchive == nil {
		return fmt.Errorf("Could not find go version %v for %v/%v", wantedVersionNum, goos, goarch)
	}

	fmt.Printf("go version %v found online\n", wantedVersionNum)

	fmt.Println("Downloading...")
	err = releaseArchive.Download(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Download Complete.")

	fmt.Println("Verifying Cheksum...")
	err = releaseArchive.VerifyChecksum()
	if err != nil {
		return err
	}
	fmt.Println("Cheksum okay.")

	fmt.Println("Extracting go source...")
	err = releaseArchive.ExtractTo(installDir)
	if err != nil {
		return err
	}
	fmt.Println("Extraction Complete")

	err = util.SymLinkGoBinary(wantedVersionNum, binDir)
	if err != nil {
		return err
	}

	fmt.Printf("go version %v is now active\n", wantedVersionNum)
	fmt.Printf("Make sure to prepend %v to the PATH environment variable.\n", binDir)
	return nil
}

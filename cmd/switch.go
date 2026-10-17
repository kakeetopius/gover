package cmd

import (
	"cmp"
	"context"
	"fmt"
	"go/version"
	"os"
	"runtime"

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
	currentVersion, err := GetCurrentGoVersion()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("no new version given")
	}
	wantedVersionNum := args[0]
	wantedVersion := fmt.Sprintf("go%v", wantedVersionNum)

	if !version.IsValid(wantedVersion) {
		return fmt.Errorf("invalid go version: %v", wantedVersionNum)
	}

	if version.Compare(wantedVersion, currentVersion) == 0 {
		fmt.Printf("The version go %v is already the active version.\n", wantedVersionNum)
		return nil
	}

	binDir, err := GetDefaultBinDirectory()
	if err != nil {
		return err
	}
	installDir, err := GetInstallDirectory()
	if err != nil {
		return err
	}
	alreadyInstalled, err := CheckIfGoVersionIsInstalled(wantedVersionNum)
	if err != nil {
		return err
	}

	if alreadyInstalled {
		err = SymLinkGoBinary(wantedVersionNum, "")
		if err != nil {
			return err
		}
		fmt.Printf("go version %v is now active\n", wantedVersionNum)
		fmt.Printf("Make sure to prepend %v to the PATH environment variable.\n", binDir)
		return nil
	}

	fmt.Printf("go version %v not found offline\n", wantedVersionNum)

	fmt.Println("Getting go versions...")
	releases, err := GetReleases(context.Background(), true)
	if err != nil {
		return err
	}

	goos := cmp.Or(os.Getenv("GOOS"), runtime.GOOS)
	goarch := cmp.Or(os.Getenv("GOARCH"), runtime.GOARCH)
	if goos == "" {
		return fmt.Errorf("could not GOOS environment variable. make sure to set it")
	}
	if goarch == "" {
		return fmt.Errorf("could not GOARCH environment variable. make sure to set it")
	}

	release := releases.
		Filter(FilterVersion(wantedVersionNum)).
		First()

	if release == nil {
		return fmt.Errorf("Could not find go version %v", wantedVersionNum)
	}

	releaseArchive := release.
		Filter(FilterOS(goos)).
		Filter(FilterArch(goarch)).
		Filter(FilterFileKind("archive")).
		FirstFile()
	if releaseArchive == nil {
		return fmt.Errorf("Could not find go version %v", wantedVersionNum)
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

	err = SymLinkGoBinary(wantedVersionNum, binDir)
	if err != nil {
		return err
	}

	fmt.Printf("go version %v is now active\n", wantedVersionNum)
	fmt.Printf("Make sure to prepend %v to the PATH environment variable.\n", binDir)
	return nil
}

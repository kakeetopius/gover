package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"go/version"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

func UpdateCommand() *cobra.Command {
	updateCommand := &cobra.Command{
		Use:          "update",
		Short:        "Update or Install to the latest stable go version",
		SilenceUsage: true,
		RunE:         runUpdate,
	}

	return updateCommand
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println("Getting go versions...")
	releases, err := GetReleases(context.Background(), false)
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

	releease := releases.
		Filter(FilterReleaseStable).
		First()
	if releease == nil {
		return fmt.Errorf("could not get the latest go release")
	}

	releaseFile := releease.
		Filter(FilterOS(goos)).
		Filter(FilterArch(goarch)).
		Filter(FilterFileKind("archive")).
		FirstFile()

	if releease == nil {
		return fmt.Errorf("could not get the latest go release")
	}
	versionNum := releaseFile.CleanedVersion()

	fmt.Printf("latest go version is %v\n", versionNum)

	currentVersion, err := GetCurrentGoVersion()
	if err != nil && !errors.Is(err, ErrCouldNotGetVersion) && !errors.Is(err, ErrGoNotInstalled) {
		return err
	}

	binDir, err := GetDefaultBinDirectory()
	if err != nil {
		return err
	}

	installDir, err := GetInstallDirectory()
	if err != nil {
		return err
	}

	if version.Compare(currentVersion, releease.Version) >= 0 {
		fmt.Println("Already at the latest go version. Nothing to update.")
		return nil
	}

	alreadyInstalled, err := CheckIfGoVersionIsInstalled(versionNum)
	if err != nil {
		return err
	}
	if alreadyInstalled {
		err = SymLinkGoBinary(versionNum, binDir)
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

	err = SymLinkGoBinary(versionNum, binDir)
	if err != nil {
		return err
	}
	fmt.Println("The latest go source has been installed to:", filepath.Join(installDir, versionNum))
	fmt.Println("The latest go binary has been symlinked to the directory:", binDir)
	fmt.Println("Make sure to prepend it to the PATH environment variable.")

	return nil
}

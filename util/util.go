// Package util contains some utility functions for gover.
package util

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"go/version"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"text/template"
)

var (
	ErrGoNotInstalled     = errors.New("go not installed on the system or not in environments' PATH variable")
	ErrCouldNotGetVersion = errors.New("could not get the current go version")
)

// GetCurrentGoVersion returns the version of first go binary found in the environments' PATH variable in the format go1.20.3
// It returns [ErrGoNotInstalled] if no go binary is found or [ErrCouldNotGetVersion] if the output from
// the command 'go version' is not the expected one.
func GetCurrentGoVersion() (string, error) {
	_, err := exec.LookPath("go")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrGoNotInstalled
		}
		return "", err
	}

	cmd := exec.Command(
		"go", "version",
	)
	out := bytes.NewBuffer([]byte{})
	cmd.Stdout = out
	cmd.Env = setEnv(cmd.Environ(), "GOTOOLCHAIN", "local") // prevents go command from automatically changing the version

	err = cmd.Run()
	if err != nil {
		return "", ErrCouldNotGetVersion
	}
	re, err := regexp.Compile(`go version (go\d+\.\d+(?:\.\d+)?)`)
	if err != nil {
		return "", err
	}

	matches := re.FindStringSubmatch(out.String())
	if len(matches) > 1 {
		return matches[1], nil
	}
	return "", ErrCouldNotGetVersion
}

// GetInstallDirectory returns the directory to install go versions in.
func GetInstallDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", errors.New("LOCALAPPDATA is not set")
		}
		return filepath.Join(base, "Programs", "Gover"), nil
	default:
		return filepath.Join(home, ".local", "gover"), nil
	}
}

// GetDefaultBinDirectory returns the directory to symlink the go binary to.
func GetDefaultBinDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", errors.New("LOCALAPPDATA is not set")
		}
		return filepath.Join(base, "Programs", "Gover", "bin"), nil
	default:
		return filepath.Join(home, ".local", "bin"), nil
	}
}

// CheckIfGoVersionIsInstalled checks if a particular go version is installed within the directory returned by [GetInstallDirectory]
func CheckIfGoVersionIsInstalled(version string) (bool, error) {
	installDir, err := GetInstallDirectory()
	if err != nil {
		return false, err
	}

	info, err := os.Stat(filepath.Join(installDir, version))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	if info.IsDir() {
		return true, nil
	}

	return false, nil
}

// SymLinkGoBinary creates a symlink of the go binary from the version's directory within the installation directory returned by [GetInstallDirectory] to the
// destination directory 'dstDir' or if dst is empty to the bin directory returned by [GetDefaultBinDirectory]
func SymLinkGoBinary(version string, dstDir string) error {
	installDir, err := GetInstallDirectory()
	if err != nil {
		return err
	}

	_, err = CheckIfGoVersionIsInstalled(version)
	if err != nil {
		return err
	}

	oldPath := filepath.Join(installDir, version, "bin", getBinaryName("go"))
	info, err := os.Stat(oldPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("could not find go version %v binary in the directory %v", version, filepath.Join(installDir, version))
		}
		return err
	}

	if info.IsDir() {
		return fmt.Errorf("the path %v should be a file not a directory", oldPath)
	}

	if dstDir == "" {
		binDir, err := GetDefaultBinDirectory()
		if err != nil {
			return err
		}
		dstDir = binDir
	}

	err = os.MkdirAll(dstDir, 0o755)
	if err != nil {
		return err
	}
	newPath := filepath.Join(dstDir, getBinaryName("go"))

	err = os.Remove(newPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("error installing the go binary to the destination directory: %w", err)
	}

	if runtime.GOOS == "windows" {
		return createWindowsLauncher(oldPath, newPath)
	}

	err = os.Symlink(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("error installing the go binary to the destination directory: %w", err)
	}

	return nil
}

func OsAndArch() (string, string) {
	return cmp.Or(os.Getenv("GOOS"), runtime.GOOS), cmp.Or(os.Getenv("GOARCH"), runtime.GOARCH)
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(env)+1)

	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			result = append(result, e)
		}
	}

	return append(result, prefix+value)
}

// GetInstalledVersions returns all versions installed in the directory returned by [GetInstallDirectory]
func GetInstalledVersions() ([]string, error) {
	installDir, err := GetInstallDirectory()
	if err != nil {
		return nil, err
	}

	dirInfo, err := os.ReadDir(installDir)
	if err != nil {
		return nil, err
	}

	installedVersions := make([]string, 0, len(dirInfo))
	for _, entry := range dirInfo {
		if !entry.IsDir() {
			continue
		}
		// directory names are in the form 1.26.0 but required format is go1.26.0
		name := "go" + entry.Name()

		if !version.IsValid(name) {
			continue
		}

		installedVersions = append(installedVersions, name)
	}

	return SortVersions(installedVersions), nil
}

// SortVersions sorts the versions from newest release to oldest.
func SortVersions(versions []string) []string {
	slices.SortFunc(versions, func(a, b string) int {
		return -version.Compare(a, b)
	})

	return versions
}

// DeleteVersionDirectory deletes a go version installed in the directory returned by [GetInstallDirectory]
func DeleteVersionDirectory(version string) error {
	installDir, err := GetInstallDirectory()
	if err != nil {
		return err
	}
	versionDir := filepath.Join(installDir, version)

	return os.RemoveAll(versionDir)
}

func getBinaryName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}

	return name
}

func replaceExtension(fileName string, newExt string) string {
	ext := filepath.Ext(fileName)
	return strings.TrimSuffix(fileName, ext) + newExt
}

func createWindowsLauncher(goBinaryPath string, launcherPath string) error {
	tmpl, err := template.New("launcher").Parse(windowsLauncher)
	if err != nil {
		return err
	}
	launcherPath = replaceExtension(launcherPath, ".cmd")

	launcher, err := os.OpenFile(launcherPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o777)
	if err != nil {
		return err
	}
	defer launcher.Close()

	err = tmpl.Execute(launcher, BinaryPath{
		Path: goBinaryPath,
	})
	if err != nil {
		return err
	}

	return nil
}

type BinaryPath struct {
	Path string
}

var windowsLauncher = `@echo off
setlocal

set "APP_PATH={{.Path}}"

if not exist "%APP_PATH%" (
    echo Error: Binary not found: "%APP_PATH%"
    exit /b 1
)

"%APP_PATH%" %*
exit /b %ERRORLEVEL%
`

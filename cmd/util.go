package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var (
	ErrGoNotInstalled     = errors.New("go not installed on the system or not in environments' PATH variable")
	ErrCouldNotGetVersion = errors.New("could not get the current go version")
)

// GetCurrentGoVersion returns the version of first go binary found in the environments' PATH variable
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
		return "", err
	}
	re, err := regexp.Compile(`go version (go\d+\.\d+\.\d+)`)
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
		return filepath.Join(base, "Programs", "Go"), nil
	default:
		return filepath.Join(home, ".local", "go"), nil
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
		return filepath.Join(base, "Programs", "Go", "bin"), nil
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
// destination directory 'dst' or if dst is empty to the bin directory returned by [GetDefaultBinDirectory]
func SymLinkGoBinary(version string, dst string) error {
	installDir, err := GetInstallDirectory()
	if err != nil {
		return err
	}

	_, err = CheckIfGoVersionIsInstalled(version)
	if err != nil {
		return err
	}

	oldPath := filepath.Join(installDir, version, "bin", "go")
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

	if dst == "" {
		binDir, err := GetDefaultBinDirectory()
		if err != nil {
			return err
		}
		dst = binDir
	}
	newPath := filepath.Join(dst, "go")

	err = os.Remove(newPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("error installing the go binary to the destination directory: %w", err)
	}

	err = os.Symlink(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("error installing the go binary to the destination directory: %w", err)
	}

	return nil
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

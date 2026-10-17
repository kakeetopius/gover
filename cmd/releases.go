package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kakeetopius/net-tools/pkg/httpclient"
)

type GoReleases []GoRelease

type GoRelease struct {
	Version string          `json:"version"`
	Stable  bool            `json:"stable"`
	Files   []GoReleaseFile `json:"files"`
}

type GoReleaseFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	SHA256   string `json:"sha256"`
	Size     int    `json:"size"`
	Kind     string `json:"kind"`

	downloadedFile []byte
}

type (
	ReleasesFilterFunc     func(GoRelease) bool
	ReleaseFilesFilterFunc func(GoReleaseFile) bool
)

const (
	goDownloadEndpoint = "https://go.dev/dl"
)

type queryParams struct {
	Mode    string `url:"mode,omitempty"`
	Include string `url:"include,omitempty"`
}

func GetReleases(ctx context.Context, includeAll bool) (GoReleases, error) {
	params := queryParams{
		Mode: "json",
	}
	if includeAll {
		params.Include = "all"
	}
	var releases GoReleases

	err := httpclient.
		New().
		WithBaseURL(goDownloadEndpoint).
		GetJSON(ctx, "/", params, &releases)
	if err != nil {
		return nil, err
	}

	return releases, nil
}

func (r GoReleases) Filter(f ReleasesFilterFunc) GoReleases {
	filtered := make([]GoRelease, 0)
	for _, release := range r {
		if !f(release) {
			continue
		}
		filtered = append(filtered, release)
	}

	return filtered
}

func (r GoReleases) Latest() *GoRelease {
	if len(r) > 0 {
		return &r[0]
	}

	return nil
}

func (r GoReleases) Version(version string) *GoRelease {
	r = r.Filter(FilterVersion(version))
	if len(r) > 0 {
		return &r[0]
	}

	return nil
}

func (r *GoRelease) Filter(f ReleaseFilesFilterFunc) *GoRelease {
	newFiles := make([]GoReleaseFile, 0)

	for _, file := range r.Files {
		if !f(file) {
			continue
		}
		newFiles = append(newFiles, file)
	}

	r.Files = newFiles
	return r
}

func (r *GoRelease) FirstFile() *GoReleaseFile {
	if len(r.Files) > 0 {
		return &r.Files[0]
	}

	return nil
}

func FilterReleaseStable(r GoRelease) bool {
	return r.Stable
}

func FilterReleaseUnstable(r GoRelease) bool {
	return !r.Stable
}

func FilterVersion(version string) ReleasesFilterFunc {
	return func(r GoRelease) bool {
		return r.Version == "go"+version
	}
}

func FilterFileKind(kind string) ReleaseFilesFilterFunc {
	return func(f GoReleaseFile) bool {
		return f.Kind == kind
	}
}

func FilterOS(os string) ReleaseFilesFilterFunc {
	return func(f GoReleaseFile) bool {
		return f.OS == os
	}
}

func FilterArch(arch string) ReleaseFilesFilterFunc {
	return func(f GoReleaseFile) bool {
		return f.Arch == arch
	}
}

func (f *GoReleaseFile) Download(ctx context.Context) error {
	resp, err := httpclient.
		New().
		WithBaseURL(goDownloadEndpoint).
		Get(ctx, "/"+f.Filename, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f.downloadedFile, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if len(f.downloadedFile) != f.Size {
		return fmt.Errorf("received an incomplete or corrupted release file.")
	}

	return nil
}

func (f *GoReleaseFile) CleanedVersion() string {
	// f.Version is in format go27.0.2 so we remove the word go
	return strings.TrimPrefix(f.Version, "go")
}

func (f *GoReleaseFile) Save(dstDir string) error {
	if f.downloadedFile == nil {
		return fmt.Errorf("Release file not yet downloaded")
	}

	if dstDir == "" {
		dstDir = os.TempDir()
	}

	dst, err := os.OpenFile(filepath.Join(dstDir, f.Filename), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o760)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, bytes.NewReader(f.downloadedFile))

	return err
}

type ErrCheckSumMismatch struct {
	Expected string
	Got      string
}

func (e ErrCheckSumMismatch) Error() string {
	return fmt.Sprintf("checksums for the downloaded release do not match, expected %v got %v", e.Expected, e.Got)
}

func (f *GoReleaseFile) VerifyChecksum() error {
	if f.downloadedFile == nil {
		return fmt.Errorf("Release file not yet downloaded")
	}

	sha := sha256.New()

	_, err := io.Copy(sha, bytes.NewReader(f.downloadedFile)) // stream downloaded content to sha
	if err != nil {
		return err
	}
	got := hex.EncodeToString(sha.Sum(nil))

	result := subtle.ConstantTimeCompare([]byte(strings.ToLower(f.SHA256)), []byte(strings.ToLower(got))) // compare the two hashes safely with subtle package
	if result != 1 {
		return ErrCheckSumMismatch{Expected: f.SHA256, Got: got}
	}

	return nil
}

func (f *GoReleaseFile) ExtractTo(dstDir string) error {
	if f.Kind != "archive" {
		return fmt.Errorf("the file '%v' is not an archive", f.Filename)
	}

	if f.downloadedFile == nil {
		return fmt.Errorf("Release file not yet downloaded")
	}

	if dstDir == "" {
		return fmt.Errorf("no destination directory given to extract archive.")
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(f.downloadedFile))
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	if err != nil {
		return err
	}

	version := f.CleanedVersion()

	for {
		hdr, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}

		// this makes sure files are saved under the a directory named after the version instead of the directory go
		fileName := strings.Replace(hdr.Name, "go", version, 1)

		dst := filepath.Join(dstDir, fileName)

		// prevent paths in archive from escaping dstDir
		rel, err := filepath.Rel(dstDir, dst)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("invalid archive path: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(dst, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
		case tar.TypeReg:
			err = os.MkdirAll(filepath.Dir(dst), 0o755)
			if err != nil {
				return err
			}

			err = func() error {
				out, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(hdr.Mode))
				if err != nil {
					return err
				}
				defer out.Close()

				_, err = io.Copy(out, tarReader)
				if err != nil {
					return err
				}

				return nil
			}()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

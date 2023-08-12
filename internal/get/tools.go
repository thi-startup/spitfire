package get

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/moby/moby/pkg/archive"
	"github.com/thi-startup/spitfire/utils"
)

const amd64tar = "amd64.tar.gz"

func DownloadInitFromGithub(destDir string) error {
	gh := NewGithubDownloader("thi-startup", "init").WithPattern(amd64tar)
	return downloadToolFromGithub(gh, destDir)
}

func DownloadFirectlFromGithub(destDir string) error {
	gh := NewGithubDownloader("thi-startup", "firectl").WithPattern(amd64tar)
	return downloadToolFromGithub(gh, destDir)
}

func downloadToolFromGithub(gh *Github, destDir string) error {
	tmp, err := utils.Mktemp(gh.Repo())
	if err != nil {
		return fmt.Errorf("error creating tmp directory for init: %w", err)
	}

	defer os.RemoveAll(tmp)

	if !utils.IsDir(destDir) {
		return fmt.Errorf("destination must be a directory")
	}

	out := filepath.Join(tmp, fmt.Sprintf("%s.tar.gz", gh.Repo()))

	if err := gh.DownloadAssetByTag(context.TODO(), out); err != nil {
		return fmt.Errorf("error downloading asset: %w", err)
	}

	tar, err := os.Open(out)
	if err != nil {
		return fmt.Errorf("error opening tarball: %w", err)
	}

	defer tar.Close()

	err = archive.Untar(tar, destDir, &archive.TarOptions{
		NoLchown: true,
	})
	if err != nil {
		return fmt.Errorf("error unpacking tar archive: %w", err)
	}

	return nil
}

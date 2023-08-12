package get

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/go-github/v53/github"
	"github.com/thi-startup/spitfire/utils"
)

type Github struct {
	owner   string
	repo    string
	pattern string
	tag     string
}

func NewGithubDownloader(owner, repo string) *Github {
	return &Github{
		owner:   owner,
		repo:    repo,
		pattern: "tar.gz",
		tag:     "latest",
	}
}

func (g Github) Repo() string {
	return g.repo
}

func (g Github) Owner() string {
	return g.owner
}

func (g Github) WithPattern(pattern string) *Github {
	g.pattern = pattern
	return &g
}

func (g Github) WithTag(tag string) *Github {
	g.tag = tag
	return &g
}

func (g Github) DownloadAssetByTag(ctx context.Context, dest string) error {
	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetReleaseByTag(ctx, g.owner, g.repo, g.tag)
	if err != nil {
		return fmt.Errorf("error getting release: %w", err)
	}

	assets, _, err := client.Repositories.ListReleaseAssets(ctx, g.owner, g.repo, release.GetID(), nil)
	if err != nil {
		return fmt.Errorf("error listing release assets: %w", err)
	}

	asset, err := FilterAssetsByPattern(assets, g.pattern)
	if err != nil {
		return fmt.Errorf("error filtering assets: %w", err)
	}

	rc, _, err := client.Repositories.DownloadReleaseAsset(ctx, g.owner, g.repo, asset.GetID(), client.Client())
	if err != nil {
		return fmt.Errorf("error downloading asset: %w", err)
	}

	if rc == nil {
		return fmt.Errorf("nothing was downloaded")
	}

	defer rc.Close()

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	bar := utils.NewProgressBar(int64(asset.GetSize()))

	_, err = io.Copy(io.MultiWriter(file, bar), rc)
	if err != nil {
		return err
	}

	return nil
}

// TODO: rewrite the filter logic to account for scenarios where multiple releases meet the filter params.
// it should return an error if multiple assets matches the pattern. As the code is now, it works just fine
// for pulling the init.
func FilterAssetsByPattern(assets []*github.ReleaseAsset, pattern string) (*github.ReleaseAsset, error) {
	var asset *github.ReleaseAsset
	for _, a := range assets {
		if name := a.GetName(); strings.Contains(name, pattern) {
			asset = a
		}
	}

	if asset == nil {
		return nil, fmt.Errorf("error asset cannot be found")
	}

	return asset, nil
}

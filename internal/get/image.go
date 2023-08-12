package get

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/thi-startup/spitfire/utils"
)

type Image struct {
	cacheDir string
}

func (p Image) WithCacheDir(dir string) *Image {
	p.cacheDir = dir
	return &p
}

func NewImage() (*Image, error) {
	cacheDir, err := utils.ImageCache()
	if err != nil {
		return nil, err
	}

	return &Image{
		cacheDir: cacheDir,
	}, nil
}

// Pull downloads the image layers and extracts them into a the destination
// directory.
func (p Image) Pull(name, dest string) (*v1.Config, error) {
	img, err := crane.Pull(name)
	if err != nil {
		return nil, fmt.Errorf("error pulling image: %w", err)
	}

	cachedImage := cache.Image(img, cache.NewFilesystemCache(p.cacheDir))

	if err := Unpack(cachedImage, dest); err != nil {
		return nil, fmt.Errorf("error unpacking image: %w", err)
	}

	config, err := cachedImage.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("error getting config file: %w", err)
	}

	return &config.Config, nil
}

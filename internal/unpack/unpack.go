package unpack

import (
	"fmt"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/moby/moby/pkg/archive"
	"github.com/thi-startup/spitfire/utils"
)

func Unpack(image v1.Image, dest string) error {
	if _, err := utils.CreateNotExist(dest); err != nil {
		return fmt.Errorf("error creating destdir: %w", err)
	}

	layers, err := image.Layers()
	if err != nil {
		return fmt.Errorf("error getting layers: %w", err)
	}

	for _, l := range layers {
		compressed, err := l.Compressed()
		if err != nil {
			return fmt.Errorf("error getting compressed image: %w", err)
		}

		err = archive.Untar(compressed, dest, &archive.TarOptions{
			NoLchown: true,
		})
		if err != nil {
			return fmt.Errorf("error unpacking tar archive: %w", err)
		}
	}

	return nil
}

func stripSHA(digest string) string {
	strip := strings.Split(digest, ":")
	if len(strip) == 1 {
		return digest
	}
	return strip[1]
}

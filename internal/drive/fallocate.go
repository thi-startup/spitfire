package drive

import (
	"fmt"
	"os"

	"github.com/detailyang/go-fallocate"
)

func Fallocate(name string, length int64) error {
	f, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}

	defer f.Close()

	if err := fallocate.Fallocate(f, 0, length); err != nil {
		return fmt.Errorf("error fallocatng file: %w", err)
	}

	return nil
}

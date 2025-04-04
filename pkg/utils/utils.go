package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thi-startup/spitfire/internal/config"
)

func Exists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

func CreateNotExist(file string) (string, error) {
	if !Exists(file) {
		if err := os.MkdirAll(file, 0774); err != nil {
			return "", err
		}
	}
	return file, nil
}

func HomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return CreateNotExist(filepath.Join(home, ".spitfire"))
}

func ImageCache() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", nil
	}
	return CreateNotExist(filepath.Join(home, "images"))
}

func VerifyBinary(path string) error {
	finfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("binary %q does not exist", path)
	}
	if err != nil {
		return fmt.Errorf("stat binary %q: %w", path, err)
	}
	if finfo.IsDir() {
		return fmt.Errorf("binary %q is a directory", path)
	}
	if finfo.Mode()&config.ExecutableMask == 0 {
		return fmt.Errorf("binary %q is not executable", path)
	}
	return nil
}

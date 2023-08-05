package utils

import (
	"os"
	"path/filepath"
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

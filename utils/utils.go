package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
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
	return CreateNotExist("/opt/spitfire")
}

func ImageCache() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", nil
	}
	return CreateNotExist(filepath.Join(home, "images"))
}

func AssetsCache() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", nil
	}

	return CreateNotExist(filepath.Join(home, "assets"))
}

func MicroVmCache() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", nil
	}

	return CreateNotExist(filepath.Join(home, "microvms"))
}

func Mktemp(pattern string) (string, error) {
	return os.MkdirTemp(os.TempDir(), pattern)
}

func IsDir(path string) bool {
	fi, err := os.Stat(path)

	return err == nil && fi.IsDir()
}

func CopyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	fi, err := srcFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE, fi.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}

	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("failed to get stats for %s", src)
	}

	if err := destFile.Chown(int(stat.Uid), int(stat.Gid)); err != nil {
		return fmt.Errorf("chown failed: %v", err)
	}

	return nil
}

func SymlinkVmlinux(dest string) error {
	assets, err := AssetsCache()
	if err != nil {
		return err
	}

	vmlinux := filepath.Join(assets, "vmlinux")
	newName := filepath.Join(dest, "vmlinux")

	if err := os.Symlink(vmlinux, newName); err != nil {
		return err
	}

	return nil
}

func NewProgressBar(maxBytes int64, description ...string) *progressbar.ProgressBar {
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}

	writer := ansi.NewAnsiStdout()
	return progressbar.NewOptions64(
		maxBytes,
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetWriter(writer),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}

func NewProgressReader(reader io.Reader, bar *progressbar.ProgressBar) progressbar.Reader {
	return progressbar.NewReader(reader, bar)
}

package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/docker/docker/daemon/graphdriver/copy"
)

const (
	mkfsPath      = "/sbin/mkfs"
	fallocatePath = "/usr/bin/fallocate"
	mountPath     = "/usr/bin/mount"
	umountPath    = "/usr/bin/umount"
)

type mkfs struct {
	fscmd     string
	size      string
	name      string
	mountArgs string
	from      string
	makeInit  bool
	initFrom  string
}

func (m *mkfs) Size(s string) *mkfs {
	m.size = s
	return m
}

func (m *mkfs) MakeInit(p bool, from string) *mkfs {
	m.makeInit = p
	m.initFrom = from
	return m
}

// specify file to copy from
func (m *mkfs) DirPath(f string) *mkfs {
	m.from = f
	return m
}

func (m *mkfs) Type(t string) *mkfs {
	m.fscmd = strings.Join([]string{m.fscmd, t}, ".")
	return m
}

func (m *mkfs) MountFlags(s string) *mkfs {
	return m
}

func (m *mkfs) Name(name string) *mkfs {
	m.name = name
	return m
}

func mktemp() (string, func() error, error) {
	name, err := os.MkdirTemp(".", "tmpdir")
	if err != nil {
		return "", nil, err
	}

	rm := func() error {
		if err := os.RemoveAll(name); err != nil {
			return err
		}
		return nil
	}

	return name, rm, nil
}

func getGoPath() (string, error) {
	path, err := exec.LookPath("go")

	if err != nil && !Exists("/usr/local/go/bin/go") {
		return "", fmt.Errorf("failed to get go path: %v", err)
	} else if path == "" {
		return "/usr/local/go/bin/go", nil
	}

	return path, nil
}

func installInitFromGithub() (initPath string, runConfig []byte, err error) {
	repo := "github.com/thi-startup/init@latest"

	goPath, err := getGoPath()
	if err != nil {
		return
	}

	install := exec.Command(goPath, "install", repo)
	if err := install.Run(); err != nil {
		return "", nil, fmt.Errorf("failed to install thi-startup init: %v", err)
	}

	initPath = filepath.Join(os.Getenv("HOME"), "go", "bin", "init")
	if !Exists(initPath) {
		return "", nil, fmt.Errorf("failed to install thi-startup init")
	}

	runConfig, err = exec.Command("curl", "-s", "https://raw.githubusercontent.com/thi-startup/init/main/run.json").CombinedOutput()
	if err != nil {
		return "", nil, fmt.Errorf("failed to download run.json from init repo: %v", err)
	}
	return
}

func cdWithRetFunc(dir string) (func() error, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get pwd: %v", err)
	}

	chDir := func(d string) error {
		if err := os.Chdir(d); err != nil {
			return fmt.Errorf("failed to cd into %s: %v", d, err)
		}
		return nil
	}

	if err := chDir(dir); err != nil {
		return nil, err
	}

	return func() error { return chDir(pwd) }, nil
}

func useLocalInit(repo string) (initPath string, runConfig []byte, err error) {
	goPath, err := getGoPath()
	if err != nil {
		return
	}

	goBack, err := cdWithRetFunc(repo)
	if err != nil {
		return
	}
	defer goBack()

	install := exec.Command(goPath, "build", ".")
	if err := install.Run(); err != nil {
		return "", nil, fmt.Errorf("failed to build thi-startup init: %v", err)
	}

	if initPath, err = filepath.Abs("./init"); err != nil {
		return
	}

	if runConfig, err = os.ReadFile("./run.json"); err != nil {
		return
	}

	return
}

func (m *mkfs) Execute() error {
	fallocate := exec.Command(fallocatePath, "-l", m.size, m.name)
	if err := fallocate.Run(); err != nil {
		return nil
	}

	if !Exists(m.fscmd) {
		return fmt.Errorf("could not find '%s'", m.fscmd)
	}
	mkfs := exec.Command(m.fscmd, m.name)
	if err := mkfs.Run(); err != nil {
		return fmt.Errorf("failed to create filesystem: %v", err)
	}

	if m.makeInit {
		var (
			initPath string
			runJson  []byte
			err      error
		)

		if !Exists(m.initFrom) {
			if initPath, runJson, err = installInitFromGithub(); err != nil {
				return err
			}
		} else {
			abs, err := filepath.Abs(m.initFrom)
			if err != nil {
				return fmt.Errorf("failed get absolute path of local repo: %v", err)
			}

			if initPath, runJson, err = useLocalInit(abs); err != nil {
				return err
			}
		}

		initDir, rmDir, err := mktemp()
		if err != nil {
			return err
		}

		defer func() {
			if err := rmDir(); err != nil {
				log.Fatalf("failed to remove temp dir: '%s': %v", initDir, err)
			}
		}()

		if err := copyFile(initPath, filepath.Join(initDir, "init")); err != nil {
			return fmt.Errorf("failed to copy init: %v", err)
		}

		configDir := filepath.Join(initDir, "thi")

		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config dir in init drive: %v", err)
		}

		if err := os.WriteFile(filepath.Join(configDir, "run.json"), runJson, 0755); err != nil {
			return fmt.Errorf("failed to create run.json file: %v", err)
		}
		m.from = initDir
	}

	if m.makeInit || Exists(m.from) {
		temp, rmDir, err := mktemp()
		if err != nil {
			return fmt.Errorf("failed to make temporary directory: %v", err)
		}

		defer func() {
			if err := rmDir(); err != nil {
				log.Fatalf("failed to remove temp dir: '%s': %v", temp, err)
			}
		}()

		mount := exec.Command(mountPath, "-o", m.mountArgs, m.name, temp)
		if err := mount.Run(); err != nil {
			return fmt.Errorf("failed to mount '%s': %v", m.name, err)
		}

		if err := copy.DirCopy(m.from, temp, copy.Content, true); err != nil {
			return fmt.Errorf("failed to copy from '%s' to '%s': %v", m.from, temp, err)
		}

		umount := exec.Command(umountPath, temp)
		if err := umount.Run(); err != nil {
			return fmt.Errorf("failed to unmount '%s': %v", temp, err)
		}
	}

	return nil
}

func copyFile(src, dest string) error {
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

func Exists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

func Mkfs() (*mkfs, error) {
	if !Exists(fallocatePath) {
		return nil, fmt.Errorf("could not find '%s'", fallocatePath)
	}
	if !Exists(mountPath) {
		return nil, fmt.Errorf("could not find '%s'", mountPath)
	}
	return &mkfs{
		fscmd:     mkfsPath,
		mountArgs: "loop,noatime",
	}, nil
}

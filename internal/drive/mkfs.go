package drive

import (
	"context"
	"io"
	"os"
	"os/exec"
)

// TODO: add the ability to add filesystem specific arguments
type Mkfs struct {
	fs     string
	device string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func NewMkfsCommand(fs, device string) Mkfs {
	return Mkfs{
		fs:     fs,
		device: device,
	}.WithStderr(nil)
}

func (m Mkfs) WithStdin(f io.Reader) Mkfs {
	m.stdin = os.Stdin
	if f != nil {
		m.stdin = f
	}
	return m
}

func (m Mkfs) Stdin() io.Reader {
	return m.stdin
}

func (m Mkfs) WithStdout(f io.Writer) Mkfs {
	m.stdout = os.Stdout
	if f != nil {
		m.stdout = f
	}
	return m
}

func (m Mkfs) Stdout() io.Writer {
	return m.stdout
}

func (m Mkfs) WithStderr(f io.Writer) Mkfs {
	m.stderr = os.Stderr
	if f != nil {
		m.stderr = f
	}
	return m
}

func (m Mkfs) Stderr() io.Writer {
	return m.stderr
}

func (m Mkfs) Args() []string {
	args := []string{}
	if len(m.fs) != 0 {
		args = append(args, "-t", m.fs)
	}

	args = append(args, m.device)
	return args
}

func (m Mkfs) Build(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(
		ctx,
		"mkfs",
		m.Args()...,
	)

	if stdin := m.Stdin(); stdin != nil {
		cmd.Stdin = stdin
	}

	if stdout := m.Stdout(); stdout != nil {
		cmd.Stdout = stdout
	}

	if stderr := m.Stderr(); stderr != nil {
		cmd.Stderr = stderr
	}

	return cmd
}

package drive

import (
	"context"
	"io"
	"os/exec"
)

type mount struct {
	source string
	target string
	mkdir  bool
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func NewMountCommand(source, dest string) mount {
	return mount{
		source: source,
		target: dest,
	}
}

func (m mount) Mkdir() mount {
	m.mkdir = true
	return m
}

func (m mount) WithStdin(f io.Reader) mount {
	m.stdin = f
	return m
}

func (m mount) Stdin() io.Reader {
	return m.stdin
}

func (m mount) WithStdout(f io.Writer) mount {
	m.stdout = f
	return m
}

func (m mount) Stdout() io.Writer {
	return m.stdout
}

func (m mount) WithStderr(f io.Writer) mount {
	m.stderr = f
	return m
}

func (m mount) Stderr() io.Writer {
	return m.stderr
}

func (m mount) MountArgs() []string {
	args := []string{}

	if m.mkdir {
		args = append(args, "--mkdir")
	}

	args = append(args, "--source", m.source)
	args = append(args, "--target", m.target)

	return args
}

func (m mount) UmountArgs() []string {
	args := []string{}

	args = append(args, m.target)
	return args
}

func (m mount) build(ctx context.Context, name string, args []string) *exec.Cmd {
	cmd := exec.CommandContext(
		ctx,
		name,
		args...,
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

func (m mount) BuildMountCmd(ctx context.Context) *exec.Cmd {
	return m.build(ctx, "mount", m.MountArgs())
}

func (m mount) BuildUmountCmd(ctx context.Context) *exec.Cmd {
	return m.build(ctx, "umount", m.UmountArgs())
}

package firectl

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Firectl struct {
	binary           string
	kernel           string
	kernelOpts       []string
	cni              string
	rootDrive        string
	additionalDrives []string
	stdin            io.Reader
	stdout           io.Writer
	stderr           io.Writer
}

func NewFirectlCommand() Firectl {
	return Firectl{
		kernel:     "vmlinux",
		kernelOpts: []string{"ro", "console=ttyS0", "noapic", "reboot=k", "panic=1", "pci=off", "nomodules"},
	}.WithStdin(os.Stdin).WithStdout(os.Stdout).WithStderr(os.Stderr).WithBinary("firectl")
}

func (f Firectl) WithBinary(binary string) Firectl {
	f.binary = binary
	return f
}

func (f Firectl) WithRootDrive(drive string) Firectl {
	f.rootDrive = drive
	return f
}

func (f Firectl) WithCNI(cni string) Firectl {
	f.cni = cni
	return f
}

func (f Firectl) WithAdditionalDrives(drives ...string) Firectl {
	f.additionalDrives = append(f.additionalDrives, drives...)
	return f
}

func (f Firectl) WithKernelImage(kernel string) Firectl {
	f.kernel = kernel
	return f
}

func (f Firectl) WithKernelOpts(opts ...string) Firectl {
	f.kernelOpts = append(f.kernelOpts, opts...)
	return f
}

func (f Firectl) WithStdin(stdin io.Reader) Firectl {
	f.stdin = os.Stdin
	if stdin != nil {
		f.stdin = stdin
	}
	return f
}

func (f Firectl) Stdin() io.Reader {
	return f.stdin
}

func (f Firectl) WithStdout(stdout io.Writer) Firectl {
	f.stdout = os.Stdout
	if stdout != nil {
		f.stdout = stdout
	}
	return f
}

func (f Firectl) Stdout() io.Writer {
	return f.stdout
}

func (f Firectl) WithStderr(stderr io.Writer) Firectl {
	f.stderr = os.Stderr
	if stderr != nil {
		f.stderr = stderr
	}
	return f
}

func (f Firectl) Stderr() io.Writer {
	return f.stderr
}

func (f Firectl) Args() []string {
	args := []string{}
	args = append(args, "--kernel", f.kernel)

	if len(f.cni) != 0 {
		args = append(args, "--cni-net", f.cni)
	}

	if len(f.rootDrive) != 0 {
		args = append(args, "--root-drive", f.rootDrive)
	}

	for _, drive := range f.additionalDrives {
		args = append(args, "--add-drive", drive)
	}

	kernelOpts := strings.Join(f.kernelOpts, " ")
	args = append(args, "--kernel-opts", kernelOpts)

	return args
}

func (f Firectl) Build(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(
		ctx,
		f.binary,
		f.Args()...,
	)

	if stdin := f.Stdin(); stdin != nil {
		cmd.Stdin = stdin
	}

	if stdout := f.Stdout(); stdout != nil {
		cmd.Stdout = stdout
	}

	if stderr := f.Stderr(); stderr != nil {
		cmd.Stderr = stderr
	}

	return cmd
}

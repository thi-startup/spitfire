package drive

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/c2h5oh/datasize"
)

type Drive struct {
	opts opts
}

type opts struct {
	name   string
	fstype string
	size   string
	target string
	path   string
}

type optFunc func(*opts)

func defaultOpts() opts {
	return opts{
		name:   "rootfs.ext4",
		fstype: "ext4",
		size:   "400M",
	}
}

func NewDrive(opts ...optFunc) *Drive {
	o := defaultOpts()
	for _, fn := range opts {
		fn(&o)
	}

	return &Drive{
		opts: o,
	}
}

func WithPath(path string) optFunc {
	return func(opts *opts) {
		opts.path = path
	}
}

func WithName(name string) optFunc {
	return func(opts *opts) {
		if len(name) != 0 {
			opts.name = name
		}
		opts.name = filepath.Join(opts.path, opts.name)
	}
}

func WithFstype(fs string) optFunc {
	return func(opts *opts) {
		if len(fs) != 0 {
			opts.fstype = fs
		}
	}
}

func WithSize(size string) optFunc {
	return func(opts *opts) {
		if len(size) != 0 {
			opts.size = size
		}
	}
}

func WithTarget(target string) optFunc {
	return func(opts *opts) {
		opts.target = target
	}
}

func (d Drive) CreateLoopDrive(ctx context.Context) error {
	size, err := datasize.ParseString(d.opts.size)
	if err != nil {
		return fmt.Errorf("error parsing size: %w", err)
	}

	if err := Fallocate(d.opts.name, int64(size)); err != nil {
		return fmt.Errorf("error creating drive: %w", err)
	}

	if err := NewMkfsCommand(d.opts.fstype, d.opts.name).Build(ctx).Run(); err != nil {
		return fmt.Errorf("error running mkfs command: %w", err)
	}

	return nil
}

func (d Drive) MountLoopDrive(ctx context.Context) error {
	if len(d.opts.target) == 0 {
		return fmt.Errorf("error target cannot be empty")
	}

	if err := NewMountCommand(d.opts.name, d.opts.target).BuildMountCmd(ctx).Run(); err != nil {
		return fmt.Errorf("error running mount command: %w", err)
	}

	return nil
}

func (d Drive) UmountLoopDrive(ctx context.Context) error {
	if len(d.opts.target) == 0 {
		return fmt.Errorf("error target cannot be empty")
	}

	if err := NewMountCommand(d.opts.name, d.opts.target).BuildUmountCmd(ctx).Run(); err != nil {
		return fmt.Errorf("error running umount command: %w", err)
	}

	return nil
}

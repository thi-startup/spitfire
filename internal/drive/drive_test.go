package drive

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/thi-startup/spitfire/utils"
)

func Test_CreateLoopDrive(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "test.ext4")

	drive := NewDrive(
		WithName(name),
		WithFstype("ext4"),
		WithSize("4M"),
	)

	if err := drive.CreateLoopDrive(context.TODO()); err != nil {
		t.Fatal(err)
	}

	if !utils.Exists(name) {
		t.Fatal(fmt.Errorf("loop drive has not been created"))
	}
}

package drive

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
)

func Test_MkfsCmd(t *testing.T) {
	mkfs := NewMkfsCommand("ext4", "test.ext4")
	cmd := mkfs.Build(context.TODO())

	abs, err := exec.LookPath("mkfs")
	if err != nil {
		t.Fatal(err)
	}

	expected := fmt.Sprintf("%s -t ext4 test.ext4", abs)
	got := cmd.String()

	if expected != got {
		t.Fatalf("expected %s: got %s", expected, got)
	}
}

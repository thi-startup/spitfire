package drive

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/thi-startup/spitfire/utils"
)

func TestFallocate(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "testfile")

	size, err := datasize.ParseString("1M")
	if err != nil {
		t.Fatal(err)
	}

	if err := Fallocate(name, int64(size)); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(name)

	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}

	if !utils.Exists(name) {
		t.Fatal(fmt.Errorf("file doesn't exist"))
	}

	newSize := info.Size()

	if newSize != int64(size) {
		t.Fatalf("expected: %d, got: %d", size, newSize)
	}
}

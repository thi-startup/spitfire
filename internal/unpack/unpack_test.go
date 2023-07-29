package unpack

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/random"
)

// TODO: Add more robust tests
func Test_Unpack(t *testing.T) {
	dir := t.TempDir()

	img, err := random.Image(10, 5)
	if err != nil {
		t.Fatal(err)
	}

	if err := Unpack(img, dir); err != nil {
		t.Fatal(err)
	}
}

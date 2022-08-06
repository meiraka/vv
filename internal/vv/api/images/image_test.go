package images

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResizeImage(t *testing.T) {
	for _, filename := range []string{
		"app.jpg",
		"app.png",
		"app.webp",
	} {
		t.Run(filename, func(t *testing.T) {
			f, err := os.Open(filepath.Join("testdata", filename))
			if err != nil {
				t.Fatalf("testfile: %s: %v", filename, err)
			}
			defer f.Close()
			if _, err := resizeImage(f, 8, 8); err != nil {
				t.Errorf("got error %v; want <nil>", err)
			}
		})
	}

}

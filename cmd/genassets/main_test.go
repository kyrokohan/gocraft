package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyrokohan/gocraft/internal/tile"
)

func readPNG(t *testing.T, name string) *image.NRGBA {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "internal", "assets", "img", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	// A picture with no see-through pixels is saved without its alpha and read back as
	// another kind of image, so it is copied into the kind the generator draws.
	got := image.NewNRGBA(img.Bounds())
	draw.Draw(got, got.Bounds(), img, img.Bounds().Min, draw.Src)
	return got
}

// The pictures in the repo are what the generator draws now. If this fails after a
// change to the generator, run `make assets`.
func TestCommittedPicturesAreCurrent(t *testing.T) {
	for name, want := range map[string]*image.NRGBA{
		"blocks.png": drawBlocks(), "title.png": drawTitle(), "mossback.png": drawMossback(),
	} {
		got := readPNG(t, name)
		if got.Bounds() != want.Bounds() {
			t.Errorf("%s is %v, the generator draws %v", name, got.Bounds(), want.Bounds())
			continue
		}
		for i := range want.Pix {
			if got.Pix[i] != want.Pix[i] {
				t.Errorf("%s differs from what the generator draws: run `make assets`", name)
				break
			}
		}
	}
}

func TestOnlyLeavesAndGlassHaveHoles(t *testing.T) {
	for id, paint := range painters {
		holes := 0
		for y := range size {
			for x := range size {
				if _, ok := paint(x, y); !ok {
					holes++
				}
			}
		}
		seeThrough := tile.ID(id) == tile.Leaves || tile.ID(id) == tile.Glass
		if seeThrough && holes == 0 {
			t.Errorf("tile %d should have holes and has none", id)
		}
		if !seeThrough && holes != 0 {
			t.Errorf("tile %d has %d holes and should be solid", id, holes)
		}
	}
}

func TestTheMossbacksFaceIsTheSameBothWaysRound(t *testing.T) {
	for y := range size {
		for x := range size {
			a, _ := mossFace(x, y)
			b, _ := mossFace(size-1-x, y)
			if a != b {
				t.Fatalf("the face differs at %d,%d from its mirror", x, y)
			}
		}
	}
}

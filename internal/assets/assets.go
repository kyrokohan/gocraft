// Package assets holds gocraft's pictures and reads them once at start-up. The pictures
// are drawn by cmd/genassets; run `make assets` to draw them again.
package assets

import (
	"bytes"
	"embed"
	"fmt"

	"glonk.dev/bitmap"

	"github.com/kyrokohan/gocraft/internal/tile"
)

//go:embed img/blocks.png img/title.png
var files embed.FS

// Levels is how many sizes each tile is kept in: 16, 8, 4, 2 and 1 pixels across. A
// block far away is drawn from a smaller one, so that it does not flicker as the
// camera moves.
const Levels = 5

// Texture is one tile in every size, the largest first. A pixel is 0xAARRGGBB, where AA
// is 0 for a hole and 255 for a color. Rows run from the top.
type Texture [Levels][]uint32

// Side returns the width and height of a level in pixels.
func Side(level int) int {
	return tile.Size >> level
}

// Set is every picture the game uses.
type Set struct {
	// Tiles are the block textures, indexed by tile.ID.
	Tiles [tile.Count]Texture

	// Title is the name of the game, for the title screen.
	Title *bitmap.Bitmap
}

// Load reads the pictures built into the program.
func Load() (*Set, error) {
	strip, err := decode("img/blocks.png")
	if err != nil {
		return nil, err
	}
	if strip.Width() != tile.Size*int(tile.Count) || strip.Height() != tile.Size {
		return nil, fmt.Errorf("assets: blocks.png is %d by %d, wanted %d by %d",
			strip.Width(), strip.Height(), tile.Size*int(tile.Count), tile.Size)
	}

	set := &Set{}
	for id := range set.Tiles {
		full := make([]uint32, tile.Size*tile.Size)
		for y := range tile.Size {
			for x := range tile.Size {
				if c := strip.At(id*tile.Size+x, y); !c.IsDefault() {
					r, g, b := c.RGB()
					full[y*tile.Size+x] = 0xFF000000 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
				}
			}
		}
		set.Tiles[id][0] = full
		for level := 1; level < Levels; level++ {
			set.Tiles[id][level] = halve(set.Tiles[id][level-1], Side(level-1))
		}
	}

	if set.Title, err = decode("img/title.png"); err != nil {
		return nil, err
	}
	return set, nil
}

func decode(name string) (*bitmap.Bitmap, error) {
	data, err := files.ReadFile(name)
	if err != nil {
		return nil, err
	}
	bm, err := bitmap.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("assets: %s: %w", name, err)
	}
	return bm, nil
}

// halve returns a picture half as wide and tall as src, which is side pixels across.
// Each pixel is the average of the colors among the four it stands for, and a hole if
// fewer than two of them are colors: leaves close up from far away and a pane of glass
// keeps its frame.
func halve(src []uint32, side int) []uint32 {
	n := side / 2
	out := make([]uint32, n*n)
	for y := range n {
		for x := range n {
			var r, g, b, count uint32
			for _, p := range [4]uint32{
				src[2*y*side+2*x], src[2*y*side+2*x+1],
				src[(2*y+1)*side+2*x], src[(2*y+1)*side+2*x+1],
			} {
				if p>>24 == 0 {
					continue
				}
				r += p >> 16 & 0xFF
				g += p >> 8 & 0xFF
				b += p & 0xFF
				count++
			}
			if count >= 2 {
				out[y*n+x] = 0xFF000000 | r/count<<16 | g/count<<8 | b/count
			}
		}
	}
	return out
}

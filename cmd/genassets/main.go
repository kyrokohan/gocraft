// Command genassets draws gocraft's pictures: the strip of block tiles, the title and
// the creature.
// Everything is drawn by the code in this file, from a fixed hash, so the pictures are
// the same every run and nothing is downloaded or copied from anywhere.
//
//	go run ./cmd/genassets                     # writes internal/assets/img/*.png
//	go run ./cmd/genassets -preview sheet.png  # and an enlarged sheet to look at
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/kyrokohan/gocraft/internal/tile"
)

const size = tile.Size

// hash returns a number from 0 up to 1 that depends only on its arguments, so a
// picture drawn from it comes out the same on every machine and Go version.
func hash(x, y, seed int) float64 {
	h := uint32(x)*0x9E3779B1 ^ uint32(y)*0x85EBCA77 ^ uint32(seed)*0xC2B2AE3D
	h ^= h >> 15
	h *= 0x2C1B3C6D
	h ^= h >> 12
	h *= 0x297A2D39
	h ^= h >> 15
	return float64(h) / (1 << 32)
}

// blotch is soft noise from 0 to 1 that changes over about cell pixels and wraps at the
// edge of a tile, so that tiles laid side by side show no seam.
func blotch(x, y, cell, seed int) float64 {
	n := size / cell
	fx, fy := float64(x)/float64(cell), float64(y)/float64(cell)
	x0, y0 := int(fx), int(fy)
	tx, ty := fx-float64(x0), fy-float64(y0)
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	at := func(i, j int) float64 { return hash(i%n, j%n, seed) }
	top := at(x0, y0)*(1-tx) + at(x0+1, y0)*tx
	bottom := at(x0, y0+1)*(1-tx) + at(x0+1, y0+1)*tx
	return top*(1-ty) + bottom*ty
}

type rgb struct{ r, g, b float64 }

func (c rgb) scale(f float64) rgb { return rgb{c.r * f, c.g * f, c.b * f} }

func (c rgb) nrgba() color.NRGBA {
	clamp := func(v float64) uint8 { return uint8(math.Round(min(max(v, 0), 255))) }
	return color.NRGBA{R: clamp(c.r), G: clamp(c.g), B: clamp(c.b), A: 255}
}

// grain is base made a little lighter or darker at each pixel, by up to amount.
func grain(base rgb, amount float64, x, y, seed int) rgb {
	return base.scale(1 + amount*(hash(x, y, seed)*2-1))
}

// painter says what one pixel of a tile looks like. ok is false for a hole.
type painter func(x, y int) (c rgb, ok bool)

var painters = [tile.Count]painter{
	tile.GrassTop:  grassTop,
	tile.GrassSide: grassSide,
	tile.Dirt:      dirt,
	tile.Stone:     stone,
	tile.Sand:      sand,
	tile.LogSide:   logSide,
	tile.LogTop:    logTop,
	tile.Leaves:    leaves,
	tile.Planks:    planks,
	tile.Brick:     brick,
	tile.Glass:     glass,
	tile.Water:     water,
	tile.Snow:      snow,
	tile.Deepstone: deepstone,

	tile.Gravel:        gravel,
	tile.Clay:          clay,
	tile.SandstoneTop:  sandstoneTop,
	tile.SandstoneSide: sandstoneSide,
	tile.Cobble:        cobble,
	tile.CoalOre:       ore(rgb{44, 44, 50}, 50),
	tile.CopperOre:     ore(rgb{204, 122, 70}, 51),
	tile.IronOre:       ore(rgb{206, 172, 142}, 52),
	tile.CrystalOre:    ore(rgb{140, 228, 236}, 53),
	tile.PineLogSide:   pineLogSide,
	tile.PineLogTop:    pineLogTop,
	tile.PineNeedles:   pineNeedles,
	tile.PaleLogSide:   paleLogSide,
	tile.PaleLogTop:    paleLogTop,
	tile.PaleLeaves:    paleLeaves,
	tile.CactusSide:    cactusSide,
	tile.CactusTop:     cactusTop,
	tile.Ice:           ice,
	tile.TallGrass:     tallGrass,
	tile.RedFlower:     flower(rgb{214, 58, 52}, rgb{244, 208, 72}),
	tile.YellowFlower:  flower(rgb{244, 206, 60}, rgb{226, 132, 44}),
	tile.Bush:          bush,
	tile.Reed:          reed,
	tile.DryBush:       dryBush,
	tile.Mushroom:      mushroom,
	tile.Torch:         torch,
}

func grassTop(x, y int) (rgb, bool) {
	c := grain(rgb{86, 160, 70}, 0.10, x, y, 1)
	switch h := hash(x, y, 2); {
	case h < 0.10:
		c = c.scale(1.2) // a blade catching the light
	case h > 0.90:
		c = c.scale(0.8)
	}
	return c, true
}

func dirt(x, y int) (rgb, bool) {
	c := grain(rgb{134, 96, 67}, 0.09, x, y, 3)
	switch h := hash(x, y, 4); {
	case h < 0.08:
		c = c.scale(0.72) // a pebble
	case h > 0.94:
		c = c.scale(1.16)
	}
	return c, true
}

func grassSide(x, y int) (rgb, bool) {
	depth := 3 + int(hash(x, 0, 5)*3) // the turf hangs 3 to 5 pixels down
	if y >= depth {
		return dirt(x, y)
	}
	c, _ := grassTop(x, y)
	if y == depth-1 {
		c = c.scale(0.78)
	}
	return c, true
}

func stone(x, y int) (rgb, bool) {
	c := rgb{128, 128, 134}.scale(0.86 + 0.28*blotch(x, y, 4, 6))
	c = grain(c, 0.04, x, y, 7)
	if hash(x, y, 8) < 0.05 {
		c = c.scale(0.7) // a chip
	}
	return c, true
}

func sand(x, y int) (rgb, bool) {
	c := grain(rgb{219, 207, 150}, 0.045, x, y, 9)
	if hash(x, y, 10) < 0.06 {
		c = c.scale(0.86)
	}
	return c, true
}

func logSide(x, y int) (rgb, bool) {
	c := rgb{104, 78, 48}.scale(0.8 + 0.4*hash(x, 0, 11)) // bark runs up and down
	return grain(c, 0.07, x, y, 12), true
}

func logTop(x, y int) (rgb, bool) {
	d := max(math.Abs(float64(x)-7.5), math.Abs(float64(y)-7.5)) // square rings
	if d > 6.6 {
		return grain(rgb{96, 72, 44}, 0.08, x, y, 13), true
	}
	c := rgb{188, 152, 98}
	if int(d)%2 == 1 {
		c = rgb{158, 122, 74}
	}
	return grain(c, 0.04, x, y, 14), true
}

func leaves(x, y int) (rgb, bool) {
	if hash(x, y, 15) < 0.2 {
		return rgb{}, false
	}
	c := grain(rgb{58, 124, 50}, 0.12, x, y, 16)
	switch h := hash(x, y, 17); {
	case h < 0.12:
		c = c.scale(1.28)
	case h > 0.86:
		c = c.scale(0.72)
	}
	return c, true
}

func planks(x, y int) (rgb, bool) {
	board := y / 4
	seam := rgb{112, 84, 50}
	if y%4 == 3 || x == (board*7+3)%size {
		return grain(seam, 0.05, x, y, 18), true
	}
	c := rgb{172, 136, 86}.scale(0.92 + 0.16*hash(board, 0, 19))
	return grain(c, 0.05, x/3, y, 20), true
}

func brick(x, y int) (rgb, bool) {
	row := y / 4
	shifted := x + row%2*4 // every other course is moved half a brick
	if y%4 == 3 || shifted%8 == 7 {
		return grain(rgb{196, 190, 178}, 0.04, x, y, 21), true
	}
	c := rgb{158, 74, 58}.scale(0.86 + 0.28*hash(shifted/8%2, row, 22))
	return grain(c, 0.05, x, y, 23), true
}

func glass(x, y int) (rgb, bool) {
	if x == 0 || y == 0 || x == size-1 || y == size-1 {
		return rgb{206, 228, 236}, true
	}
	if (x+y == 9 || x+y == 10) && x >= 3 && x <= 7 {
		return rgb{238, 247, 252}, true // a glint
	}
	return rgb{}, false
}

func water(x, y int) (rgb, bool) {
	c := grain(rgb{52, 104, 196}, 0.05, x, y, 24)
	wave := int(math.Round(1.5 * math.Sin(float64(x)*math.Pi/8)))
	if (y+wave)%5 == 0 && hash(x/2, y, 25) < 0.7 {
		c = c.scale(1.28)
	}
	return c, true
}

func snow(x, y int) (rgb, bool) {
	if hash(x, y, 26) < 0.07 {
		return rgb{212, 224, 242}, true
	}
	return grain(rgb{238, 242, 248}, 0.025, x, y, 27), true
}

func deepstone(x, y int) (rgb, bool) {
	if hash(x, y, 28) < 0.06 {
		return rgb{96, 72, 126}, true // a fleck
	}
	c := rgb{52, 50, 62}.scale(0.8 + 0.5*blotch(x, y, 4, 29))
	return grain(c, 0.05, x, y, 30), true
}

// The mossback is gocraft's one creature: a squat animal with moss growing on its back.
// Its picture is four tiles side by side, which the renderer lays on the boxes it is
// built from, in this order: the back, the hide of its sides, its face, and a leg.
var mossbackTiles = [...]painter{mossBack, mossHide, mossFace, mossLeg}

var (
	moss  = rgb{78, 122, 62}
	cream = rgb{224, 208, 172}
)

func mossBack(x, y int) (rgb, bool) {
	c := moss.scale(0.84 + 0.32*blotch(x, y, 4, 40))
	c = grain(c, 0.06, x, y, 41)
	switch h := hash(x, y, 42); {
	case h < 0.03:
		return grain(cream, 0.04, x, y, 43), true // a bare patch
	case h < 0.13:
		c = c.scale(1.26) // a tuft
	}
	return c, true
}

func mossHide(x, y int) (rgb, bool) {
	hang := 4 + int(hash(x, 0, 44)*4) // the moss hangs 4 to 7 pixels down the side
	if y < hang {
		c, _ := mossBack(x, y)
		if y == hang-1 {
			c = c.scale(0.74)
		}
		return c, true
	}
	c := grain(cream, 0.05, x, y, 45)
	if hash(x, y, 46) < 0.05 {
		c = c.scale(0.86)
	}
	return c, true
}

// mossFace is the same from left to right as from right to left, so that it does not
// matter which way round a face of a box takes its picture.
func mossFace(x, y int) (rgb, bool) {
	mx := min(x, size-1-x) // the distance from the nearer side
	switch {
	case y < 3:
		c, _ := mossBack(mx, y)
		return c, true // a fringe of moss over the brow
	case (mx == 4 || mx == 5) && y >= 6 && y <= 8:
		if mx == 4 && y == 6 {
			return rgb{236, 238, 244}, true // a glint
		}
		return rgb{40, 36, 48}, true // an eye
	case mx == 7 && y == 11:
		return rgb{176, 112, 112}, true // the nose
	case mx >= 6 && y == 13:
		return rgb{150, 128, 100}, true // the mouth
	}
	return grain(cream, 0.04, mx, y, 47), true
}

func mossLeg(x, y int) (rgb, bool) {
	if y >= size-4 {
		return grain(rgb{70, 60, 52}, 0.08, x, y, 48), true // a hoof
	}
	return grain(rgb{186, 160, 120}, 0.06, x, y, 49), true
}

// drawMossback draws the creature's picture: its tiles side by side.
func drawMossback() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size*len(mossbackTiles), size))
	for i, paint := range mossbackTiles {
		for y := range size {
			for x := range size {
				if c, ok := paint(x, y); ok {
					img.SetNRGBA(i*size+x, y, c.nrgba())
				}
			}
		}
	}
	return img
}

func gravel(x, y int) (rgb, bool) {
	// Pebbles two pixels across, each its own gray, with dark gaps between some.
	c := rgb{132, 128, 126}.scale(0.72 + 0.5*hash(x/2, y/2, 60))
	if hash(x, y, 61) < 0.12 {
		c = c.scale(0.68)
	}
	return grain(c, 0.05, x, y, 62), true
}

func clay(x, y int) (rgb, bool) {
	c := rgb{150, 158, 170}.scale(0.95 + 0.08*hash(0, y/3, 63)) // laid down in faint bands
	return grain(c, 0.03, x, y, 64), true
}

func sandstoneTop(x, y int) (rgb, bool) {
	return grain(rgb{208, 190, 136}, 0.04, x, y, 65), true
}

func sandstoneSide(x, y int) (rgb, bool) {
	c := rgb{208, 190, 136}.scale(0.9 + 0.16*hash(0, y/3, 66)) // beds of sand, one on another
	if y%6 == 5 {
		c = c.scale(0.86)
	}
	return grain(c, 0.035, x, y, 67), true
}

func cobble(x, y int) (rgb, bool) {
	// Stones four across in courses three high, every other course shifted, with
	// mortar in the joints.
	row := y / 4
	shifted := x + row%2*2
	if y%4 == 3 || shifted%4 == 3 {
		return grain(rgb{78, 78, 84}, 0.06, x, y, 68), true
	}
	c := rgb{136, 136, 142}.scale(0.82 + 0.34*hash(shifted/4, row, 69))
	return grain(c, 0.05, x, y, 70), true
}

// ore returns the painter of stone with lumps of something in it.
func ore(lump rgb, seed int) painter {
	return func(x, y int) (rgb, bool) {
		if blotch(x, y, 2, seed) > 0.7 && hash(x, y, seed+100) > 0.15 {
			return grain(lump, 0.1, x, y, seed+200), true
		}
		return stone(x, y)
	}
}

func pineLogSide(x, y int) (rgb, bool) {
	c := rgb{86, 60, 46}.scale(0.78 + 0.44*hash(x, 0, 71)) // bark runs up and down
	return grain(c, 0.08, x, y, 72), true
}

func pineLogTop(x, y int) (rgb, bool) {
	d := max(math.Abs(float64(x)-7.5), math.Abs(float64(y)-7.5))
	if d > 6.6 {
		return grain(rgb{78, 54, 42}, 0.08, x, y, 73), true
	}
	c := rgb{196, 142, 96}
	if int(d)%2 == 1 {
		c = rgb{166, 110, 72}
	}
	return grain(c, 0.04, x, y, 74), true
}

func pineNeedles(x, y int) (rgb, bool) {
	if hash(x, y, 75) < 0.15 {
		return rgb{}, false
	}
	c := grain(rgb{40, 88, 62}, 0.12, x, y, 76)
	if (x+2*y)%5 == 0 {
		c = c.scale(1.3) // needles lie in rows
	}
	return c, true
}

func paleLogSide(x, y int) (rgb, bool) {
	if hash(x/3, y, 77) < 0.1 && y%3 != 0 {
		return grain(rgb{62, 58, 54}, 0.1, x, y, 78), true // a dark scar across the bark
	}
	return grain(rgb{216, 210, 198}, 0.04, x, y, 79), true
}

func paleLogTop(x, y int) (rgb, bool) {
	d := max(math.Abs(float64(x)-7.5), math.Abs(float64(y)-7.5))
	if d > 6.6 {
		return grain(rgb{206, 200, 188}, 0.05, x, y, 80), true
	}
	c := rgb{226, 206, 160}
	if int(d)%2 == 1 {
		c = rgb{204, 180, 132}
	}
	return grain(c, 0.04, x, y, 81), true
}

func paleLeaves(x, y int) (rgb, bool) {
	if hash(x, y, 82) < 0.22 {
		return rgb{}, false
	}
	c := grain(rgb{134, 170, 74}, 0.12, x, y, 83)
	if hash(x, y, 84) < 0.12 {
		c = c.scale(1.2)
	}
	return c, true
}

func cactusSide(x, y int) (rgb, bool) {
	c := rgb{76, 142, 78}
	if x%4 == 0 {
		c = c.scale(0.74) // a furrow between two ribs
	}
	if x%4 == 2 && hash(x, y, 85) < 0.22 {
		return rgb{226, 226, 190}, true // a spine
	}
	return grain(c, 0.05, x, y, 86), true
}

func cactusTop(x, y int) (rgb, bool) {
	d := max(math.Abs(float64(x)-7.5), math.Abs(float64(y)-7.5))
	c := rgb{76, 142, 78}.scale(1.25 - 0.05*d)
	return grain(c, 0.05, x, y, 87), true
}

func ice(x, y int) (rgb, bool) {
	c := grain(rgb{168, 208, 240}, 0.03, x, y, 88)
	if (x+y)%7 == 0 && hash(x/2, y/2, 89) < 0.6 {
		c = rgb{226, 240, 252} // a crack that catches the light
	}
	return c, true
}

// The plants are drawn as they stand: the bottom row of the tile is the ground. Most of
// a plant's tile is holes.

func tallGrass(x, y int) (rgb, bool) {
	height := 5 + int(hash(x, 0, 90)*9) // each column is a blade of its own height
	if hash(x, 1, 91) < 0.3 || size-1-y >= height {
		return rgb{}, false
	}
	c := rgb{84, 150, 62}.scale(0.8 + 0.5*float64(size-1-y)/float64(size)) // lighter toward the tip
	return grain(c, 0.08, x, y, 92), true
}

// flower returns the painter of a flower with petals of one color round a middle of
// another, on a stem with two leaves.
func flower(petal, middle rgb) painter {
	return func(x, y int) (rgb, bool) {
		dx, dy := x-7, y-5 // the middle of the bloom
		switch {
		case dx == 0 && dy == 0, dx == 1 && dy == 0:
			return middle, true
		case dx >= -2 && dx <= 3 && dy >= -2 && dy <= 2 && !(abs(2*dx-1) >= 5 && abs(dy) == 2):
			return grain(petal, 0.08, x, y, 93), true
		case (x == 7 || x == 8) && y > 7:
			return rgb{66, 128, 58}, true // the stem
		case y == 11 && (x == 5 || x == 6), y == 12 && (x == 9 || x == 10):
			return rgb{84, 150, 62}, true // a leaf
		}
		return rgb{}, false
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func bush(x, y int) (rgb, bool) {
	// A round mound of leaves on the ground, with berries in it.
	dx, dy := float64(x)-7.5, float64(y)-10
	if dx*dx/49+dy*dy/30 > 1 || hash(x, y, 94) < 0.1 {
		return rgb{}, false
	}
	if hash(x, y, 95) < 0.06 {
		return rgb{196, 52, 70}, true // a berry
	}
	return grain(rgb{52, 110, 56}, 0.14, x, y, 96), true
}

func reed(x, y int) (rgb, bool) {
	// A few stalks of different heights, each with a brown head.
	for i, stalk := range []int{2, 6, 9, 13} {
		top := 1 + int(hash(i, 0, 97)*5)
		if x != stalk || y < top {
			continue
		}
		if y < top+3 {
			return rgb{122, 86, 52}, true
		}
		return grain(rgb{128, 164, 84}, 0.06, x, y, 98), true
	}
	return rgb{}, false
}

func dryBush(x, y int) (rgb, bool) {
	// Bare twigs that spread as they rise from one foot.
	spread := (size - 1 - y) / 2
	dx := abs(x - 7)
	if y < 5 || dx > spread || (dx != spread && dx != spread/2 && dx != 0) || hash(x, y, 99) < 0.15 {
		return rgb{}, false
	}
	return grain(rgb{132, 100, 66}, 0.1, x, y, 100), true
}

func mushroom(x, y int) (rgb, bool) {
	switch {
	case y >= 12 && (x == 7 || x == 8):
		return rgb{226, 216, 196}, true // the stem
	case y >= 8 && y < 12 && abs(2*x-15) <= 9-2*(11-y):
		if hash(x, y, 101) < 0.18 {
			return rgb{236, 226, 206}, true // a spot on the cap
		}
		return grain(rgb{166, 84, 62}, 0.06, x, y, 102), true
	}
	return rgb{}, false
}

func torch(x, y int) (rgb, bool) {
	// A stick standing on the ground with a flame on its top: white in the heart, then
	// yellow, then orange at the edge.
	switch {
	case y >= 7 && (x == 7 || x == 8):
		return grain(rgb{118, 82, 48}, 0.08, x, y, 103), true
	case y == 6 && (x == 7 || x == 8):
		return rgb{70, 44, 30}, true // charred where it burns
	}
	dx, dy := float64(x)-7.5, float64(y)-4.2
	switch d := dx*dx/2.6 + dy*dy/5.2; {
	case d < 0.35:
		return rgb{255, 250, 222}, true
	case d < 0.75:
		return rgb{255, 214, 86}, true
	case d < 1.15:
		return rgb{246, 138, 44}, true
	}
	return rgb{}, false
}

// drawBlocks draws the strip: every tile side by side, in the order of package tile.
func drawBlocks() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size*int(tile.Count), size))
	for id, paint := range painters {
		for y := range size {
			for x := range size {
				if c, ok := paint(x, y); ok {
					img.SetNRGBA(id*size+x, y, c.nrgba())
				}
			}
		}
	}
	return img
}

// font is the letters of the title, 5 by 7.
var font = map[rune][7]string{
	'G': {" ### ", "#   #", "#    ", "# ###", "#   #", "#   #", " ### "},
	'O': {" ### ", "#   #", "#   #", "#   #", "#   #", "#   #", " ### "},
	'C': {" ### ", "#   #", "#    ", "#    ", "#    ", "#   #", " ### "},
	'R': {"#### ", "#   #", "#   #", "#### ", "# #  ", "#  # ", "#   #"},
	'A': {" ### ", "#   #", "#   #", "#####", "#   #", "#   #", "#   #"},
	'F': {"#####", "#    ", "#    ", "#### ", "#    ", "#    ", "#    "},
	'T': {"#####", "  #  ", "  #  ", "  #  ", "  #  ", "  #  ", "  #  "},
}

const (
	titleText  = "GOCRAFT"
	titleBlock = 4 // pixels across each little block a letter is built from

	// titleCols and titleRows are the size of the title counted in those blocks.
	titleCols = len(titleText)*6 - 1
	titleRows = 7
)

// titleTiles is what each letter of the title is built from.
var titleTiles = [len(titleText)]tile.ID{
	tile.GrassTop, tile.Dirt, tile.Stone, tile.Brick, tile.Planks, tile.LogSide, tile.Sand,
}

// drawTitle draws the name of the game in letters built from little blocks, each letter
// out of a different tile, on a see-through ground. A block is lit along its top and left
// and dark along its bottom and right.
func drawTitle() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, titleCols*titleBlock, titleRows*titleBlock))
	for i, letter := range titleText {
		paint := painters[titleTiles[i]]
		for row, line := range font[letter] {
			for col, mark := range line {
				if mark != '#' {
					continue
				}
				for dy := range titleBlock {
					for dx := range titleBlock {
						x, y := (i*6+col)*titleBlock+dx, row*titleBlock+dy
						c, _ := paint(x%size, y%size)
						switch {
						case dx == titleBlock-1 || dy == titleBlock-1:
							c = c.scale(0.68)
						case dx == 0 || dy == 0:
							c = c.scale(1.18)
						}
						img.SetNRGBA(x, y, c.nrgba())
					}
				}
			}
		}
	}
	return img
}

// enlarge returns src with every pixel k pixels across, laid over a dark ground so that
// holes show.
func enlarge(src *image.NRGBA, k int) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()*k, b.Dy()*k))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.NRGBA{R: 30, G: 30, B: 38, A: 255}), image.Point{}, draw.Src)
	for y := range b.Dy() * k {
		for x := range b.Dx() * k {
			if p := src.NRGBAAt(x/k, y/k); p.A != 0 {
				out.SetNRGBA(x, y, p)
			}
		}
	}
	return out
}

// inRows lays the tiles of a strip out in rows of a number of tiles, so that a long
// strip can be looked at.
func inRows(strip *image.NRGBA, perRow int) *image.NRGBA {
	tiles := strip.Bounds().Dx() / size
	out := image.NewNRGBA(image.Rect(0, 0, perRow*size, (tiles+perRow-1)/perRow*size))
	for i := range tiles {
		at := image.Rect(i%perRow*size, i/perRow*size, i%perRow*size+size, i/perRow*size+size)
		draw.Draw(out, at, strip, image.Pt(i*size, 0), draw.Src)
	}
	return out
}

// drawPreview is every picture, enlarged, one above the other.
func drawPreview() *image.NRGBA {
	pictures := []*image.NRGBA{enlarge(inRows(drawBlocks(), 13), 6), enlarge(drawTitle(), 6), enlarge(drawMossback(), 12)}
	w, h := 0, 0
	for _, p := range pictures {
		w, h = max(w, p.Bounds().Dx()), h+p.Bounds().Dy()+12
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.NRGBA{R: 30, G: 30, B: 38, A: 255}), image.Point{}, draw.Src)
	y := 0
	for _, p := range pictures {
		draw.Draw(out, p.Bounds().Add(image.Pt(0, y)), p, image.Point{}, draw.Src)
		y += p.Bounds().Dy() + 12
	}
	return out
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func run() error {
	out := flag.String("out", "internal/assets/img", "folder to write the pictures into")
	preview := flag.String("preview", "", "also write an enlarged sheet of everything to this file, to look at")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(*out, "blocks.png"), drawBlocks()); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(*out, "title.png"), drawTitle()); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(*out, "mossback.png"), drawMossback()); err != nil {
		return err
	}
	if *preview != "" {
		return writePNG(*preview, drawPreview())
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Command genassets draws gocraft's pictures: the strip of block tiles and the title.
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

// drawPreview is the strip and the title, enlarged, one above the other.
func drawPreview() *image.NRGBA {
	blocks, title := enlarge(drawBlocks(), 6), enlarge(drawTitle(), 6)
	w := max(blocks.Bounds().Dx(), title.Bounds().Dx())
	out := image.NewNRGBA(image.Rect(0, 0, w, blocks.Bounds().Dy()+title.Bounds().Dy()+12))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.NRGBA{R: 30, G: 30, B: 38, A: 255}), image.Point{}, draw.Src)
	draw.Draw(out, blocks.Bounds(), blocks, image.Point{}, draw.Src)
	draw.Draw(out, title.Bounds().Add(image.Pt(0, blocks.Bounds().Dy()+12)), title, image.Point{}, draw.Src)
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

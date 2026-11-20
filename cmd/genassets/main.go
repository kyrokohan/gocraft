// Command genassets draws Skyspear's pictures: the strip of block tiles, the title and
// the creatures; and it makes the game's sounds.
// Everything is drawn by the code in this file, from a fixed hash, so the pictures are
// the same every run and nothing is downloaded or copied from anywhere.
//
//	go run ./cmd/genassets                     # writes internal/assets/img/*.png and sound/*.wav
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

	"github.com/kyrokohan/skyspear/internal/tile"
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

// mix is the color a part t of the way from c to o.
func (c rgb) mix(o rgb, t float64) rgb {
	return rgb{c.r + (o.r-c.r)*t, c.g + (o.g-c.g)*t, c.b + (o.b-c.b)*t}
}

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
	tile.TurfTop:   turfTop,
	tile.TurfSide:  turfSide,
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
	tile.FlintSeam:     flintSeam,
	tile.IronOre:       ore(rgb{206, 172, 142}, 52),
	tile.CrystalOre:    ore(rgb{140, 228, 236}, 53),
	tile.PineLogSide:   pineLogSide,
	tile.PineLogTop:    pineLogTop,
	tile.PineNeedles:   pineNeedles,
	tile.PaleLogSide:   paleLogSide,
	tile.PaleLogTop:    paleLogTop,
	tile.PaleLeaves:    paleLeaves,
	tile.StumpPalmSide: stumpPalmSide,
	tile.StumpPalmTop:  stumpPalmTop,
	tile.Ice:           ice,
	tile.Fern:          frond(&fernPic, rgb{50, 104, 50}, rgb{88, 154, 66}, 92),
	tile.RedFlower:     flower(rgb{214, 58, 52}, rgb{244, 208, 72}),
	tile.YellowFlower:  flower(rgb{244, 206, 60}, rgb{226, 132, 44}),
	tile.Bush:          bush,
	tile.Horsetail:     horsetail,
	tile.DryBush:       dryBush,
	tile.Mushroom:      mushroom,
	tile.Torch:         torch,

	tile.Berries: berries,
	tile.Cracks:  cracks,

	tile.KnappingTop:  knappingTop,
	tile.KnappingSide: knappingSide,
	tile.HearthTop:    hearthTop,
	tile.HearthSide:   hearthSide,
	tile.PotTop:       potTop,
	tile.PotSide:      potSide,
	tile.FurBed:       furBed,

	tile.TallFern:      frond(&tallFernPic, rgb{46, 100, 48}, rgb{80, 148, 62}, 310),
	tile.PalmCrown:     palmCrown,
	tile.FernTrunkSide: fernTrunkSide,
	tile.FernTrunkTop:  fernTrunkTop,
	tile.Fronds:        frondLeaves,
	tile.FanLeaves:     fanLeaves,
	tile.RedwoodSide:   redwoodSide,
	tile.RedwoodTop:    redwoodTop,
	tile.Bone:          bone,
	tile.RedrockTop:    redrockTop,
	tile.RedrockSide:   redrockSide,

	tile.Stick:        stick,
	tile.Coal:         coal,
	tile.CopperBar:    bar(rgb{214, 128, 74}),
	tile.IronBar:      bar(rgb{206, 208, 214}),
	tile.CrystalShard: shard,
	tile.Fruit:        fruit,
	tile.BerryLoaf:    berryLoaf,
	tile.WoodPick:     tool(toolPick, woodHead),
	tile.WoodAxe:      tool(toolAxe, woodHead),
	tile.WoodShovel:   tool(toolShovel, woodHead),
	tile.StonePick:    tool(toolPick, stoneHead),
	tile.StoneAxe:     tool(toolAxe, stoneHead),
	tile.StoneShovel:  tool(toolShovel, stoneHead),
	tile.CopperPick:   tool(toolPick, copperHead),
	tile.CopperAxe:    tool(toolAxe, copperHead),
	tile.CopperShovel: tool(toolShovel, copperHead),
	tile.IronPick:     tool(toolPick, ironHead),
	tile.IronAxe:      tool(toolAxe, ironHead),
	tile.IronShovel:   tool(toolShovel, ironHead),
	tile.WoodBlade:    blade(woodHead),
	tile.StoneBlade:   blade(stoneHead),
	tile.CopperBlade:  blade(copperHead),
	tile.IronBlade:    blade(ironHead),
	tile.Moss:         mossClump,
	tile.MossCake:     mossCake,
	tile.Meat:         meat,
	tile.Fiber:        fiber,
	tile.Cord:         cord,
	tile.Flint:        flint,
	tile.Hide:         hide,
	tile.HandAxe:      handAxe,
	tile.FlintPick:    tool(toolPick, flintDark.scale(1.3)),
	tile.FlintKnife:   blade(flintDark.scale(1.3)),
	tile.DiggingStick: diggingStick,
	tile.Club:         club,
	tile.Spear:        spear,

	tile.Fire:          fire,
	tile.HearthLitTop:  hearthLitTop,
	tile.HearthLitSide: hearthLitSide,
	tile.CharredSide:   charredSide,
	tile.CharredTop:    charredTop,
	tile.PitCoverTop:   pitCoverTop,
	tile.PitCoverSide:  pitCoverSide,
	tile.Ash:           ashGround,
	tile.Basalt:        basalt,
	tile.Lava:          lava,
	tile.ObsidianRock:  obsidianRock,
	tile.FirestoneRock: firestoneRock,
	tile.SinterRust:    sinter(rgb{196, 104, 52}, 790),
	tile.SinterGold:    sinter(rgb{222, 190, 84}, 793),
	tile.SinterGreen:   sinter(rgb{104, 152, 96}, 796),
	tile.Obsidian:      obsidianShard,
	tile.Firestone:     firestonePiece,
	tile.ObsidianKnife: blade(obsidianGlass.scale(1.5)),
	tile.ObsidianPick:  tool(toolPick, obsidianGlass.scale(1.5)),
	tile.ObsidianSpear: obsidianSpear,
	tile.Striker:       striker,
	tile.CookedMeat:    cookedMeat,
	tile.PalmSeeds:     seeds(rgb{214, 84, 44}, rgb{246, 170, 96}),
	tile.RoastSeeds:    seeds(rgb{122, 78, 44}, rgb{64, 40, 26}),
	tile.Resin:         resin,
	tile.UnlitTorch:    unlitTorch,
	tile.FireDrill:     fireDrill,
	tile.HardSpear:     hardSpear,
	tile.SoftPot:       softPot,
	tile.Pebble:        pebble,
	tile.Sling:         sling,
}

func turfTop(x, y int) (rgb, bool) {
	// Cushions of moss, with a dead frond lying on them here and there. There is no
	// grass in the old lands.
	c := rgb{72, 126, 60}.scale(0.84 + 0.34*blotch(x, y, 4, 1))
	c = grain(c, 0.07, x, y, 2)
	switch h := hash(x, y, 300); {
	case h < 0.045:
		return grain(rgb{116, 92, 60}, 0.08, x, y, 301), true
	case h > 0.92:
		c = c.scale(1.22) // a tip catching the light
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

func turfSide(x, y int) (rgb, bool) {
	depth := 3 + int(hash(x, 0, 5)*3) // the moss hangs 3 to 5 pixels down
	if y >= depth {
		return dirt(x, y)
	}
	c, _ := turfTop(x, y)
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

// The mossback is a squat animal with moss growing on its back. Its picture is six tiles
// side by side, which the renderer lays on the boxes it is built from, in this order: the
// back, the hide of its sides, its face, a leg, and the back and the sides once its moss
// is shorn.
var mossbackTiles = []painter{mossBack, mossHide, mossFace, mossLeg, mossBare, mossBareHide}

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

// mossBare is the back of a shorn mossback: its hide, with a stubble of moss.
func mossBare(x, y int) (rgb, bool) {
	if hash(x, y, 172) < 0.14 {
		return grain(moss.scale(0.86), 0.06, x, y, 173), true
	}
	return grain(cream.scale(0.96), 0.05, x, y, 174), true
}

// mossBareHide is the side of a shorn mossback: the hide, with a line of stubble along
// the top.
func mossBareHide(x, y int) (rgb, bool) {
	if y == 0 {
		return mossBare(x, y)
	}
	c, _ := mossHide(x, size-1) // the hide where no moss hangs
	return grain(c, 0.03, x, y, 175), true
}

// drawTiles draws a creature's picture: its tiles side by side.
func drawTiles(tiles []painter) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size*len(tiles), size))
	for i, paint := range tiles {
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

func stumpPalmSide(x, y int) (rgb, bool) {
	// The trunk is armored with the feet of old leaves, in diamonds.
	c := rgb{126, 96, 62}
	switch up, down := (x+y)%8, (x-y+size)%8; {
	case up == 0 || down == 0:
		c = c.scale(0.6) // the line between two of them
	case up >= 3 && up <= 5 && down >= 3 && down <= 5:
		c = c.scale(1.18) // the middle of a diamond
	}
	return grain(c, 0.07, x, y, 85), true
}

func stumpPalmTop(x, y int) (rgb, bool) {
	d := max(math.Abs(float64(x)-7.5), math.Abs(float64(y)-7.5))
	if d > 5.6 {
		return grain(rgb{104, 78, 50}, 0.08, x, y, 86), true // the armor, seen end on
	}
	c := rgb{206, 186, 132}.scale(1.1 - 0.04*d) // soft pith
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

// A fern's picture is a few fronds that arch from one foot, in the middle of the bottom
// row. Each frond is a rib with leaflets along it.
const (
	noFrond = iota
	frondRib
	frondLeaf
)

// fronds returns the picture of a fern whose fronds end at tips.
func fronds(tips [][2]float64) (pic [size][size]uint8) {
	const footX, footY = 7.5, 15.5
	inside := func(x, y int) bool { return x >= 0 && x < size && y >= 0 && y < size }
	for _, tip := range tips {
		// The rib leaves the foot going up, and leans out toward its tip.
		bendX, bendY := footX+(tip[0]-footX)*0.25, tip[1]-1
		for i := range 49 {
			t := float64(i) / 48
			a, b, c := (1-t)*(1-t), 2*(1-t)*t, t*t
			x, y := int(a*footX+b*bendX+c*tip[0]), int(a*footY+b*bendY+c*tip[1])
			if !inside(x, y) {
				continue
			}
			pic[y][x] = frondRib
			if t < 0.3 {
				continue // the stalk is bare near the foot
			}
			// Leaflets stand off the rib on both sides: above and below it where it
			// runs level, left and right of it where it climbs.
			run := 2*(1-t)*(bendX-footX) + 2*t*(tip[0]-bendX)
			rise := 2*(1-t)*(bendY-footY) + 2*t*(tip[1]-bendY)
			for _, side := range [2]int{-1, 1} {
				lx, ly := x, y+side
				if math.Abs(rise) > math.Abs(run) {
					lx, ly = x+side, y
				}
				if inside(lx, ly) && pic[ly][lx] == noFrond && (lx+ly)%2 == 0 {
					pic[ly][lx] = frondLeaf
				}
			}
		}
	}
	return pic
}

var fernPic = fronds([][2]float64{{1, 10}, {3.5, 6.5}, {6.5, 5}, {9.5, 5}, {12.5, 6.5}, {15, 10}})

// frond returns the painter of a fern's picture in two greens, lighter toward the tips.
func frond(pic *[size][size]uint8, rib, leaf rgb, seed int) painter {
	return func(x, y int) (rgb, bool) {
		c := leaf
		switch pic[y][x] {
		case noFrond:
			return rgb{}, false
		case frondRib:
			c = rib
		}
		c = c.scale(0.82 + 0.4*float64(size-1-y)/float64(size))
		return grain(c, 0.07, x, y, seed), true
	}
}

var (
	tallFernPic  = fronds([][2]float64{{0.5, 6}, {2.5, 2}, {5.5, 0.5}, {8, 3}, {10.5, 0.5}, {13.5, 2}, {15.5, 6}})
	palmCrownPic = fronds([][2]float64{{0.5, 9}, {1.5, 4}, {5, 1}, {11, 1}, {14.5, 4}, {15.5, 9}})
)

func palmCrown(x, y int) (rgb, bool) {
	// Stiff dark leaves spread from the top of the trunk, round a cone in their middle.
	if (x == 7 || x == 8) && y >= 10 || (x == 6 || x == 9) && y >= 12 && y <= 14 {
		return grain(rgb{184, 128, 62}, 0.1, x, y, 311), true
	}
	return frond(&palmCrownPic, rgb{40, 92, 54}, rgb{62, 132, 70}, 312)(x, y)
}

func fernTrunkSide(x, y int) (rgb, bool) {
	// A tree fern's trunk is a mat of dark roots, with the stubs of old fronds in it.
	c := rgb{82, 60, 44}.scale(0.76 + 0.48*hash(x, 0, 313))
	if hash(x/2, y/2, 314) < 0.1 {
		c = rgb{128, 100, 70} // a stub
	}
	return grain(c, 0.1, x, y, 315), true
}

func fernTrunkTop(x, y int) (rgb, bool) {
	d := max(math.Abs(float64(x)-7.5), math.Abs(float64(y)-7.5))
	if d > 5.6 {
		return grain(rgb{74, 54, 40}, 0.1, x, y, 316), true
	}
	c := rgb{150, 122, 84}
	if (x+y)%3 == 0 {
		c = rgb{112, 86, 58} // strands, seen end on
	}
	return grain(c, 0.06, x, y, 317), true
}

func frondLeaves(x, y int) (rgb, bool) {
	// Leaflets in rows that slant away from a rib on both sides of it.
	rib := x%8 == 3
	slant := (x + abs(x%8-3) + 2*y) % 4
	if !rib && (slant == 3 || hash(x, y, 318) < 0.12) {
		return rgb{}, false
	}
	c := grain(rgb{70, 144, 58}, 0.1, x, y, 319)
	switch {
	case rib:
		c = c.scale(0.7)
	case slant == 0:
		c = c.scale(1.24)
	}
	return c, true
}

func fanLeaves(x, y int) (rgb, bool) {
	// Small fans, gold where the year has turned them and green where it has not.
	if hash(x, y, 320) < 0.2 {
		return rgb{}, false
	}
	c := rgb{216, 178, 54}.mix(rgb{150, 170, 62}, ramp(0.45, 0.75, blotch(x, y, 4, 321)))
	c = grain(c, 0.08, x, y, 322)
	switch h := hash(x, y, 323); {
	case h < 0.14:
		c = c.scale(1.2)
	case h > 0.88:
		c = c.scale(0.78)
	}
	return c, true
}

// ramp returns 0 below lo, 1 above hi, and a straight rise between.
func ramp(lo, hi, v float64) float64 {
	return min(max((v-lo)/(hi-lo), 0), 1)
}

func redwoodSide(x, y int) (rgb, bool) {
	// Thick red bark in ridges two pixels wide, with a deep furrow between them.
	c := rgb{134, 70, 46}.scale(0.84 + 0.3*hash(x/2, 0, 324))
	if x%4 == 3 {
		c = c.scale(0.58)
	}
	return grain(c, 0.08, x, y, 325), true
}

func redwoodTop(x, y int) (rgb, bool) {
	d := max(math.Abs(float64(x)-7.5), math.Abs(float64(y)-7.5))
	if d > 6.6 {
		return grain(rgb{112, 58, 40}, 0.08, x, y, 326), true
	}
	c := rgb{202, 124, 86}
	if int(d)%2 == 1 {
		c = rgb{172, 96, 66}
	}
	return grain(c, 0.04, x, y, 327), true
}

func bone(x, y int) (rgb, bool) {
	// Old ivory, with faint grooves along it and a pit here and there.
	c := rgb{226, 218, 194}.scale(0.94 + 0.1*hash(x, 0, 328))
	switch h := hash(x, y, 329); {
	case h < 0.04:
		c = c.scale(0.78)
	case (x+3*y)%16 == 5 && h < 0.7:
		c = c.scale(0.86) // a crack
	}
	return grain(c, 0.025, x, y, 330), true
}

func redrockTop(x, y int) (rgb, bool) {
	// Dry red earth, cracked.
	c := grain(rgb{178, 98, 64}, 0.07, x, y, 334)
	switch {
	case (x+y/3)%7 == 0 && hash(x/2, y/2, 335) < 0.7, (y+x/4)%8 == 0 && hash(x/3, y, 336) < 0.5:
		c = c.scale(0.68)
	case hash(x, y, 337) < 0.06:
		c = c.scale(1.16)
	}
	return c, true
}

// strata are the beds of the red rock, from the top of a tile to its bottom.
var strata = [size]rgb{
	{178, 98, 64}, {178, 98, 64}, {178, 98, 64}, {226, 198, 152},
	{208, 144, 90}, {208, 144, 90}, {208, 144, 90}, {134, 72, 58},
	{134, 72, 58}, {178, 98, 64}, {178, 98, 64}, {178, 98, 64},
	{226, 198, 152}, {208, 144, 90}, {208, 144, 90}, {208, 144, 90},
}

func redrockSide(x, y int) (rgb, bool) {
	// The beds rise and fall a little along the face, and meet again at its edges.
	wave := int(math.Round(0.8 * math.Sin(float64(x)*math.Pi/8)))
	return grain(strata[(y+wave+size)%size], 0.05, x, y, 338), true
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

func horsetail(x, y int) (rgb, bool) {
	// Three jointed stalks that run the height of the tile, so that one horsetail
	// stands on another. At each joint a ring of thin leaves points up and out.
	for _, stalk := range []int{3, 8, 13} {
		joint := (y + stalk) % 4 // each stalk has its joints at its own heights
		switch d := abs(x - stalk); {
		case d == 0 && joint == 0:
			return rgb{58, 84, 52}, true
		case d == 0:
			return grain(rgb{112, 162, 86}, 0.06, x, y, 98), true
		case d == 1 && joint == 3, d == 2 && joint == 2:
			return rgb{136, 182, 96}, true // a leaf of the ring
		}
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

func berries(x, y int) (rgb, bool) {
	// A bunch of round berries hanging from a twig with a leaf.
	switch {
	case y <= 3 && x == 8, y == 4 && (x == 7 || x == 9):
		return rgb{108, 82, 52}, true // the twig
	case y >= 1 && y <= 3 && x >= 9 && x <= 11 && !(y == 1 && x == 11):
		return grain(rgb{72, 138, 64}, 0.1, x, y, 104), true // the leaf
	}
	// The pixel belongs to the nearest berry, whose dark edge keeps it from the next.
	d, dx, dy := math.Inf(1), 0.0, 0.0
	for _, b := range [][2]float64{{6, 8}, {10.5, 8}, {8.5, 12}, {4, 12.5}, {12.5, 12.5}} {
		if ex, ey := float64(x)+0.5-b[0], float64(y)+0.5-b[1]; ex*ex+ey*ey < d {
			d, dx, dy = ex*ex+ey*ey, ex, ey
		}
	}
	switch {
	case d < 0.8 && dx < 0 && dy < 0:
		return rgb{246, 170, 190}, true // where the light catches it
	case d < 2.2:
		return grain(rgb{186, 40, 80}, 0.08, x, y, 105), true
	case d < 4.4:
		return rgb{100, 18, 46}, true
	}
	return rgb{}, false
}

// crackSteps is how many steps out from the middle each pixel of the cracks is: lines
// that wander out from the middle and fork. 0 is no crack.
var crackSteps = func() (steps [size][size]int) {
	var walk func(x, y, angle float64, step, seed int)
	walk = func(x, y, angle float64, step, seed int) {
		for ; step < 14; step++ {
			px, py := int(math.Floor(x)), int(math.Floor(y))
			if px < 0 || py < 0 || px >= size || py >= size {
				return
			}
			if steps[py][px] == 0 {
				steps[py][px] = step
			}
			angle += (hash(step, seed, 106) - 0.5) * 1.2
			if step == 5 && hash(seed, 0, 107) < 0.7 {
				walk(x, y, angle+0.9, step+1, seed+10) // a fork
			}
			x, y = x+math.Cos(angle), y+math.Sin(angle)
		}
	}
	for i := range 5 {
		walk(7.5, 7.5, float64(i)*2*math.Pi/5+hash(i, 1, 108), 1, i)
	}
	return steps
}()

func cracks(x, y int) (rgb, bool) {
	if crackSteps[y][x] == 0 {
		return rgb{}, false
	}
	// The red says how far a block must be broken for the pixel to show: the middle
	// first, the ends last. The renderer draws it dark whatever its color.
	return rgb{float64(crackSteps[y][x]) * 255 / 14, 0, 0}, true
}

// The things that are not blocks are drawn on holes, as they would lie in a hand.

func stick(x, y int) (rgb, bool) {
	switch {
	case x+y == 15 && x >= 3 && x <= 12:
		return grain(rgb{150, 110, 66}, 0.08, x, y, 119), true
	case x+y == 16 && x >= 4 && x <= 12:
		return grain(rgb{104, 74, 44}, 0.08, x, y, 120), true
	}
	return rgb{}, false
}

func coal(x, y int) (rgb, bool) {
	dx, dy := float64(x)-7.5, float64(y)-8.5
	if dx*dx/30+dy*dy/20 > 1 || blotch(x, y, 4, 121) < 0.25 && dx*dx+dy*dy > 12 {
		return rgb{}, false
	}
	if hash(x, y, 122) < 0.12 {
		return rgb{120, 120, 132}, true // where it catches the light
	}
	return grain(rgb{36, 36, 42}, 0.2, x, y, 123), true
}

// bar returns the painter of a bar of metal of a color, seen from above at a slant.
func bar(metal rgb) painter {
	return func(x, y int) (rgb, bool) {
		switch {
		case y >= 5 && y <= 7 && x >= 4+(7-y) && x <= 11+(7-y):
			return grain(metal.scale(1.15), 0.04, x, y, 124), true // the top
		case y >= 8 && y <= 11 && x >= 2 && x <= 13:
			return grain(metal.scale(0.85), 0.04, x, y, 125), true // the side
		case y >= 8 && y <= 11 && x == 14 && y <= 10:
			return metal.scale(0.6), true
		}
		return rgb{}, false
	}
}

func shard(x, y int) (rgb, bool) {
	dx, dy := math.Abs(float64(x)-7.5), math.Abs(float64(y)-7.5)
	if dx/3.2+dy/7 > 1 {
		return rgb{}, false
	}
	if float64(x) < 7.5 && dx/3.2+dy/7 > 0.55 {
		return rgb{224, 252, 252}, true
	}
	return grain(rgb{120, 220, 230}, 0.08, x, y, 126), true
}

func fruit(x, y int) (rgb, bool) {
	// A round fruit with a stem and a leaf.
	switch {
	case x == 8 && y >= 2 && y <= 4:
		return rgb{96, 66, 38}, true
	case y >= 2 && y <= 3 && x >= 9 && x <= 11:
		return grain(rgb{78, 148, 62}, 0.1, x, y, 127), true
	}
	dx, dy := float64(x)-7.5, float64(y)-9.5
	switch d := dx*dx + dy*dy; {
	case d < 2.5 && dx < -1 && dy < -1:
		return rgb{254, 226, 150}, true
	case d < 30:
		return grain(rgb{232, 150, 52}.scale(1-0.02*max(dx+dy, 0)), 0.05, x, y, 128), true
	}
	return rgb{}, false
}

func berryLoaf(x, y int) (rgb, bool) {
	dx, dy := float64(x)-7.5, float64(y)-9
	if dx*dx/49+dy*dy/20 > 1 {
		return rgb{}, false
	}
	switch {
	case hash(x, y, 129) < 0.12 && dy < 2:
		return rgb{176, 40, 76}, true // a berry baked in
	case dy < -1.5:
		return grain(rgb{200, 144, 80}, 0.06, x, y, 130), true // the crust on top
	}
	return grain(rgb{164, 104, 56}, 0.06, x, y, 131), true
}

// The kinds of tool, and the colors of their heads by what they are made of.
const (
	toolPick = iota
	toolAxe
	toolShovel
)

var (
	woodHead   = rgb{172, 132, 80}
	stoneHead  = rgb{136, 136, 144}
	copperHead = rgb{214, 128, 74}
	ironHead   = rgb{216, 218, 226}
)

// tool returns the painter of a tool: a handle from the bottom left to the top right,
// with a head of its kind at the top.
func tool(kind int, head rgb) painter {
	return func(x, y int) (rgb, bool) {
		// a runs along the handle, up to the right, and p across it, down to the right.
		fx, fy := float64(x)+0.5, float64(y)+0.5
		a, p := fx-fy, fx+fy-16
		on := false
		switch kind {
		case toolPick:
			// A head across the top of the handle, its two points bent back.
			on = math.Abs(p) < 7.5 && math.Abs(a-(8-0.07*p*p)) < 1.4-0.12*math.Abs(p)
		case toolAxe:
			// A blade out to one side of the top of the handle, its edge curved.
			on = p > 0.5 && a > 6.5-0.6*p && a < 8.5+0.25*p && p < 6.5-0.08*(a-6)*(a-6)
		case toolShovel:
			// A rounded blade on the end of the handle.
			w := 2.8
			if a > 10 {
				w *= math.Sqrt(max(1-(a-10)*(a-10)/5, 0))
			}
			on = a > 5.5 && a < 12.3 && math.Abs(p) < w
		}
		if on {
			return grain(head.scale(1.05-0.03*(fx-fy)/4), 0.06, x, y, 132+kind), true
		}
		switch {
		case x+y == 15 && x >= 2 && x <= 11:
			return grain(rgb{150, 110, 66}, 0.08, x, y, 119), true
		case x+y == 16 && x >= 3 && x <= 11:
			return grain(rgb{104, 74, 44}, 0.08, x, y, 120), true
		}
		return rgb{}, false
	}
}

// blade returns the painter of a blade: a grip at the bottom left with a knob on its
// end, a guard across it, and a straight blade up to a point at the top right.
func blade(head rgb) painter {
	return func(x, y int) (rgb, bool) {
		fx, fy := float64(x)+0.5, float64(y)+0.5
		a, p := fx-fy, fx+fy-16
		switch {
		case a > -2.5 && a < 12.5 && math.Abs(p) < 1.7*min(1, (12.5-a)/3.5):
			c := head
			if p < -0.5 {
				c = c.scale(1.2) // the edge catches the light
			}
			return grain(c, 0.05, x, y, 160), true
		case a >= -4.5 && a <= -2.5 && math.Abs(p) < 3.6:
			return grain(head.scale(0.72), 0.06, x, y, 161), true
		case a > -11.5 && a < -4.5 && p >= -0.5 && p <= 1.5:
			return grain(rgb{104, 74, 44}, 0.08, x, y, 162), true // the grip, wound in leather
		case a >= -13.5 && a <= -11.5 && math.Abs(p-0.5) < 1.6:
			return grain(head.scale(0.8), 0.06, x, y, 163), true
		}
		return rgb{}, false
	}
}

// mossClump is moss shorn from a mossback: a soft heap, tufted along its top.
func mossClump(x, y int) (rgb, bool) {
	dx, dy := float64(x)-7.5, float64(y)-9.5
	if dx*dx/42+dy*dy/20 > 1 || dy < -3.6+1.4*math.Sin(float64(x)*1.9) {
		return rgb{}, false
	}
	c := moss.scale(1.12 - 0.05*dy + 0.3*(blotch(x, y, 3, 164)-0.5))
	if hash(x, y, 165) < 0.1 {
		c = c.scale(1.25) // a tuft
	}
	return grain(c, 0.06, x, y, 166), true
}

// meat is a cut of raw meat: red, marbled with fat, with the end of a bone in it.
func meat(x, y int) (rgb, bool) {
	dx, dy := (float64(x)-7.5)/6.5, (float64(y)-8.5)/4.8
	switch d := dx*dx + dy*dy; {
	case x >= 12 && y >= 3 && y <= 6 && x <= 14:
		return rgb{232, 224, 204}, true // the bone
	case d > 1:
		return rgb{}, false
	case d > 0.78:
		return grain(rgb{224, 206, 190}, 0.04, x, y, 171), true // a rim of fat
	case blotch(x, y, 4, 172) > 0.66:
		return grain(rgb{226, 150, 140}, 0.05, x, y, 173), true // marbling
	}
	return grain(rgb{178, 52, 50}, 0.08, x, y, 174), true
}

// mossCake is a round cake seen a little from above: a sponge with moss on its top that
// runs down its side here and there.
func mossCake(x, y int) (rgb, bool) {
	dx := (float64(x) - 7.5) / 6.5
	if math.Abs(dx) >= 1 {
		return rgb{}, false
	}
	round := 2 * math.Sqrt(1-dx*dx) // how far the round top and bottom bulge from their middle lines
	fy := float64(y) + 0.5
	top, rim, bottom := 6-round, 6+round, 11.5+0.6*round
	switch {
	case fy < top || fy > bottom:
		return rgb{}, false
	case fy < rim || fy < rim+1+2.5*hash(x, 0, 167):
		return grain(moss.scale(1.15+0.25*(blotch(x, y, 3, 168)-0.5)), 0.05, x, y, 169), true
	case fy > bottom-1:
		return grain(rgb{170, 134, 88}, 0.05, x, y, 170), true // its baked bottom
	}
	return grain(rgb{222, 196, 140}, 0.05, x, y, 171), true
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
	'S': {" ####", "#    ", "#    ", " ### ", "    #", "    #", "#### "},
	'K': {"#   #", "#  # ", "# #  ", "##   ", "# #  ", "#  # ", "#   #"},
	'Y': {"#   #", "#   #", " # # ", "  #  ", "  #  ", "  #  ", "  #  "},
	'P': {"#### ", "#   #", "#   #", "#### ", "#    ", "#    ", "#    "},
	'E': {"#####", "#    ", "#    ", "#### ", "#    ", "#    ", "#####"},
	'A': {" ### ", "#   #", "#   #", "#####", "#   #", "#   #", "#   #"},
	'R': {"#### ", "#   #", "#   #", "#### ", "# #  ", "#  # ", "#   #"},
}

const (
	titleText  = "SKYSPEAR"
	titleBlock = 4 // pixels across each little block a letter is built from

	// titleCols and titleRows are the size of the title counted in those blocks.
	titleCols = len(titleText)*6 - 1
	titleRows = 7
)

// titleTiles is what each letter of the title is built from: SKY out of cold tiles,
// SPEAR out of earth.
var titleTiles = [len(titleText)]tile.ID{
	tile.Ice, tile.Snow, tile.Stone, tile.SandstoneSide, tile.Sand, tile.Dirt, tile.LogSide, tile.TurfTop,
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
	pictures := []*image.NRGBA{enlarge(inRows(drawBlocks(), 13), 6), enlarge(drawTitle(), 6), enlarge(drawTiles(mossbackTiles), 12), enlarge(inRows(drawTiles(personTiles()), 7), 12), enlarge(drawTiles(buglerTiles), 12), enlarge(drawTiles(nipperTiles), 12), enlarge(drawTiles(quickclawTiles), 12),
		enlarge(drawTiles(shieldbrowTiles), 12), enlarge(drawTiles(thornbackTiles), 12), enlarge(drawTiles(boulderbackTiles), 12), enlarge(drawTiles(cloudgrazerTiles), 12), enlarge(drawTiles(dreadTiles), 12)}
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
	sounds := flag.String("sounds", "internal/assets/sound", "folder to write the sounds into")
	flag.Parse()

	if err := writeSounds(*sounds); err != nil {
		return err
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(*out, "blocks.png"), drawBlocks()); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(*out, "title.png"), drawTitle()); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(*out, "mossback.png"), drawTiles(mossbackTiles)); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(*out, "person.png"), drawTiles(personTiles())); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(*out, "bugler.png"), drawTiles(buglerTiles)); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(*out, "nipper.png"), drawTiles(nipperTiles)); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(*out, "quickclaw.png"), drawTiles(quickclawTiles)); err != nil {
		return err
	}
	for name, tiles := range giantTiles {
		if err := writePNG(filepath.Join(*out, name+".png"), drawTiles(tiles)); err != nil {
			return err
		}
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

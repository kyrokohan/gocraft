// Package assets holds Skyspear's pictures and sounds and reads them once at start-up.
// They are made by cmd/genassets; run `make assets` to make them again.
package assets

import (
	"bytes"
	"embed"
	"encoding/binary"
	"fmt"

	"glonk.dev/bitmap"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/tile"
)

//go:embed img/*.png sound/*.wav
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
	// Tiles are the block textures, indexed by tile.ID, as the renderer reads them.
	Tiles [tile.Count]Texture

	// Icons are the same tiles as pictures glonk can draw, for the hotbar: each is a
	// view of its part of the strip.
	Icons [tile.Count]*bitmap.Bitmap

	// Strip is the whole strip of tiles, for a glonk mesh that has a tile on each face.
	Strip *bitmap.Bitmap

	// Title is the name of the game, for the title screen.
	Title *bitmap.Bitmap

	// Mossback is the picture laid on the boxes the creature is built from: MossbackTiles
	// square tiles side by side, its back, its hide, its face, a leg, and its back and its
	// hide once it is shorn.
	Mossback *bitmap.Bitmap

	// Bugler is the same for the bugler: BuglerTiles tiles, its back, its side, its
	// belly, the side of its head, the end of its bill, a leg and its crest.
	Bugler *bitmap.Bitmap

	// Nipper is the same for the nipper: NipperTiles tiles, its back, its side, the side
	// of its head and a leg.
	Nipper *bitmap.Bitmap

	// Quickclaw is the same for the quickclaw: QuickclawTiles tiles, its back, its side,
	// the side of its head, a leg and an arm.
	Quickclaw *bitmap.Bitmap

	// Shieldbrow, Thornback and Boulderback are the same for the three armed
	// plant-eaters, with ShieldbrowTiles, ThornbackTiles and BoulderbackTiles tiles: for
	// each its back, its side, its belly, the side of its head, its face, a leg, and
	// what it is armed with.
	Shieldbrow, Thornback, Boulderback *bitmap.Bitmap

	// Snowtusk and Slowclaw are the same for the beasts of the cold steppe, laid out as
	// those are, with SnowtuskTiles and SlowclawTiles tiles.
	Snowtusk, Slowclaw *bitmap.Bitmap

	// Howler and Daggercat are the same for the hunters of the steppe, with HowlerTiles
	// and DaggercatTiles tiles: the last two are its tail, and an ear or a fang.
	Howler, Daggercat *bitmap.Bitmap

	// Rattlewing and Logcrawler are the same for the beasts of the coal swamp, with
	// RattlewingTiles and LogcrawlerTiles tiles.
	Rattlewing, Logcrawler *bitmap.Bitmap

	// Sunback and Sailjaw are the same for the two beasts with a sail on the back, with
	// SunbackTiles and SailjawTiles tiles: the last two are the sail and the tail.
	Sunback, Sailjaw *bitmap.Bitmap

	// Lakeneck, Leapfin and Deepmaw are the same for the beasts of the sea, with
	// LakeneckTiles, LeapfinTiles and DeepmawTiles tiles: the leg is a flipper, and the
	// last two are a lakeneck's neck and tail, a leapfin's fin and tail, and a deepmaw's
	// fin and the side of its lower jaw.
	Lakeneck, Leapfin, Deepmaw *bitmap.Bitmap

	// Galewing and Stiltwing are the same for the two great fliers, with GalewingTiles
	// and StiltwingTiles tiles: the face is the beak, and the last two are the skin of a
	// wing, and a galewing's crest or a stiltwing's neck.
	Galewing, Stiltwing *bitmap.Bitmap

	// Cloudgrazer is the same for the cloudgrazer, with CloudgrazerTiles tiles: the last
	// two are the bands of its neck and the end of its tail.
	Cloudgrazer *bitmap.Bitmap

	// Dread is the same for the dread, with DreadTiles tiles: the last two are an arm
	// and the inside of its mouth.
	Dread *bitmap.Bitmap

	// Person is the same for the cave person, once for each look, one look after
	// another: PersonTiles tiles each, the face, the hair, the front, the back and a
	// side of the hide, an arm and a leg.
	Person *bitmap.Bitmap

	// Sounds are the game's sounds, by their names: samples from -1 to 1 at sound.Rate.
	Sounds [sound.Count][]float32
}

// How many tiles the pictures of the creatures have.
const (
	MossbackTiles = 6
	BuglerTiles   = 7
	NipperTiles   = 4

	QuickclawTiles = 5
	PersonTiles    = 7

	ShieldbrowTiles  = 8
	ThornbackTiles   = 8
	BoulderbackTiles = 8
	CloudgrazerTiles = 8
	DreadTiles       = 8
	SnowtuskTiles    = 8
	SlowclawTiles    = 8
	HowlerTiles      = 8
	DaggercatTiles   = 8
	RattlewingTiles  = 8
	LogcrawlerTiles  = 8
	SunbackTiles     = 8
	SailjawTiles     = 8
	LakeneckTiles    = 8
	LeapfinTiles     = 8
	DeepmawTiles     = 8
	GalewingTiles    = 8
	StiltwingTiles   = 8
)

// PersonLooks is how many looks a cave person can have.
const PersonLooks = 6

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

	set := &Set{Strip: strip}
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
		set.Icons[id] = strip.Sub(id*tile.Size, 0, tile.Size, tile.Size)
		for level := 1; level < Levels; level++ {
			set.Tiles[id][level] = halve(set.Tiles[id][level-1], Side(level-1))
		}
	}

	if set.Title, err = decode("img/title.png"); err != nil {
		return nil, err
	}
	if set.Mossback, err = creature("img/mossback.png", MossbackTiles); err != nil {
		return nil, err
	}
	if set.Bugler, err = creature("img/bugler.png", BuglerTiles); err != nil {
		return nil, err
	}
	if set.Nipper, err = creature("img/nipper.png", NipperTiles); err != nil {
		return nil, err
	}
	if set.Quickclaw, err = creature("img/quickclaw.png", QuickclawTiles); err != nil {
		return nil, err
	}
	if set.Person, err = creature("img/person.png", PersonTiles*PersonLooks); err != nil {
		return nil, err
	}
	for _, giant := range []struct {
		into  **bitmap.Bitmap
		file  string
		tiles int
	}{
		{&set.Shieldbrow, "img/shieldbrow.png", ShieldbrowTiles},
		{&set.Thornback, "img/thornback.png", ThornbackTiles},
		{&set.Boulderback, "img/boulderback.png", BoulderbackTiles},
		{&set.Cloudgrazer, "img/cloudgrazer.png", CloudgrazerTiles},
		{&set.Dread, "img/dread.png", DreadTiles},
		{&set.Snowtusk, "img/snowtusk.png", SnowtuskTiles},
		{&set.Slowclaw, "img/slowclaw.png", SlowclawTiles},
		{&set.Howler, "img/howler.png", HowlerTiles},
		{&set.Daggercat, "img/daggercat.png", DaggercatTiles},
		{&set.Rattlewing, "img/rattlewing.png", RattlewingTiles},
		{&set.Logcrawler, "img/logcrawler.png", LogcrawlerTiles},
		{&set.Sunback, "img/sunback.png", SunbackTiles},
		{&set.Sailjaw, "img/sailjaw.png", SailjawTiles},
		{&set.Lakeneck, "img/lakeneck.png", LakeneckTiles},
		{&set.Leapfin, "img/leapfin.png", LeapfinTiles},
		{&set.Deepmaw, "img/deepmaw.png", DeepmawTiles},
		{&set.Galewing, "img/galewing.png", GalewingTiles},
		{&set.Stiltwing, "img/stiltwing.png", StiltwingTiles},
	} {
		if *giant.into, err = creature(giant.file, giant.tiles); err != nil {
			return nil, err
		}
	}
	for n := range sound.Count {
		if set.Sounds[n], err = readWAV("sound/" + n.File()); err != nil {
			return nil, err
		}
	}
	return set, nil
}

// readWAV reads a sound as cmd/genassets writes it: a WAV file of one channel, 16 bits
// a sample, at sound.Rate.
func readWAV(name string) ([]float32, error) {
	data, err := files.ReadFile(name)
	if err != nil {
		return nil, err
	}
	le := binary.LittleEndian
	if len(data) < 44 || string(data[:4]) != "RIFF" || string(data[8:16]) != "WAVEfmt " || string(data[36:40]) != "data" ||
		le.Uint16(data[20:]) != 1 || le.Uint16(data[22:]) != 1 || le.Uint32(data[24:]) != sound.Rate || le.Uint16(data[34:]) != 16 {
		return nil, fmt.Errorf("assets: %s is not a WAV file of one channel, 16 bits a sample, at %d a second", name, sound.Rate)
	}
	pcm := data[44:]
	if n := int(le.Uint32(data[40:])); n <= len(pcm) {
		pcm = pcm[:n]
	}
	out := make([]float32, len(pcm)/2)
	for i := range out {
		out[i] = float32(int16(le.Uint16(pcm[2*i:]))) / 32767
	}
	return out, nil
}

// creature reads the picture of a creature, which is a number of square tiles side by side.
func creature(name string, tiles int) (*bitmap.Bitmap, error) {
	bm, err := decode(name)
	if err != nil {
		return nil, err
	}
	if bm.Width() != tiles*bm.Height() {
		return nil, fmt.Errorf("assets: %s is %d by %d, wanted %d square tiles side by side",
			name, bm.Width(), bm.Height(), tiles)
	}
	return bm, nil
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

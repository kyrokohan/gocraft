package main

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/tile"
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
	pictures := map[string]*image.NRGBA{
		"blocks.png": drawBlocks(), "title.png": drawTitle(), "mossback.png": drawTiles(mossbackTiles),
		"person.png": drawTiles(personTiles()), "bugler.png": drawTiles(buglerTiles),
		"nipper.png": drawTiles(nipperTiles), "quickclaw.png": drawTiles(quickclawTiles),
	}
	for name, tiles := range giantTiles {
		pictures[name+".png"] = drawTiles(tiles)
	}
	for name, want := range pictures {
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

func TestOnlyLeavesGlassPlantsAndPicturesOnNoBlockHaveHoles(t *testing.T) {
	for id, paint := range painters {
		holes := 0
		for y := range size {
			for x := range size {
				if _, ok := paint(x, y); !ok {
					holes++
				}
			}
		}
		seeThrough := slices.Contains([]tile.ID{
			tile.Leaves, tile.Glass, tile.PineNeedles, tile.PaleLeaves, tile.Fronds, tile.FanLeaves, tile.TallFern, tile.PalmCrown,
			tile.Fern, tile.RedFlower, tile.YellowFlower, tile.Bush, tile.Horsetail, tile.DryBush, tile.Mushroom, tile.Torch, tile.FurBed,
			tile.Berries, tile.Cracks, tile.Stick, tile.Coal, tile.CopperBar, tile.IronBar, tile.CrystalShard, tile.Fruit, tile.BerryLoaf,
			tile.WoodPick, tile.WoodAxe, tile.WoodShovel, tile.StonePick, tile.StoneAxe, tile.StoneShovel,
			tile.CopperPick, tile.CopperAxe, tile.CopperShovel, tile.SkyIronPick, tile.SkyIronAxe, tile.IronShovel,
			tile.WoodBlade, tile.StoneBlade, tile.CopperBlade, tile.SkyIronKnife, tile.Moss, tile.MossCake, tile.Meat,
			tile.Fiber, tile.Cord, tile.Flint, tile.Hide, tile.HandAxe, tile.FlintPick, tile.FlintKnife, tile.DiggingStick, tile.Club, tile.Spear,
			tile.Fire, tile.CookedMeat, tile.PalmSeeds, tile.RoastSeeds, tile.Resin, tile.UnlitTorch, tile.FireDrill, tile.HardSpear, tile.SoftPot, tile.Pebble, tile.Sling,
			tile.Obsidian, tile.Firestone, tile.ObsidianKnife, tile.ObsidianPick, tile.ObsidianSpear, tile.Striker, tile.SkyIron, tile.SkyIronSpear, tile.FurCloak, tile.Tusk, tile.Amber, tile.ScaleCrown, tile.BuglerEgg, tile.ShieldbrowEgg, tile.GalewingEgg,
			tile.Ochre, tile.Chalk, tile.RedPaint, tile.BlackPaint, tile.WhitePaint,
		}, tile.ID(id))
		if seeThrough && holes == 0 {
			t.Errorf("tile %d should have holes and has none", id)
		}
		if !seeThrough && holes != 0 {
			t.Errorf("tile %d has %d holes and should be solid", id, holes)
		}
	}
}

func TestACreaturesFaceIsTheSameBothWaysRound(t *testing.T) {
	for name, face := range map[string]painter{"mossback": mossFace} {
		for y := range size {
			for x := range size {
				a, _ := face(x, y)
				b, _ := face(size-1-x, y)
				if a != b {
					t.Fatalf("the %s's face differs at %d,%d from its mirror", name, x, y)
				}
			}
		}
	}
}

func TestEverySoundIsHeardAndNeverPastFull(t *testing.T) {
	for n := range sound.Count {
		s := makeSound(n)
		if len(s) < samples(0.05) || len(s) > samples(8) {
			t.Errorf("%s is %d samples long", n.File(), len(s))
			continue
		}
		peak, mean := 0.0, 0.0
		for _, v := range s {
			if math.IsNaN(v) {
				t.Fatalf("%s has a sample that is not a number", n.File())
			}
			peak, mean = max(peak, math.Abs(v)), mean+v/float64(len(s))
		}
		if peak < 0.2 || peak > 0.95 {
			t.Errorf("%s peaks at %.3f, wanted it heard and short of full", n.File(), peak)
		}
		if math.Abs(mean) > 0.02 {
			t.Errorf("%s sits %.3f off the middle", n.File(), mean)
		}
		// Where a loop goes round it moves no more than from one sample to the next
		// anywhere in it.
		steepest := 0.0
		for i := 1; i < len(s); i++ {
			steepest = max(steepest, math.Abs(s[i]-s[i-1]))
		}
		if n.Loops() && math.Abs(s[0]-s[len(s)-1]) > steepest {
			t.Errorf("%s jumps from %.3f to %.3f where it goes round", n.File(), s[len(s)-1], s[0])
		}
		if !n.Loops() && math.Abs(s[len(s)-1]) > 1e-3 {
			t.Errorf("%s ends at %.3f, with a click", n.File(), s[len(s)-1])
		}
	}
}

func TestASoundComesOutTheSameEveryTime(t *testing.T) {
	for _, n := range []sound.Name{sound.Break(sound.Glass), sound.Call, sound.Wind} {
		if !bytes.Equal(wav(makeSound(n)), wav(makeSound(n))) {
			t.Errorf("%s came out two ways", n.File())
		}
	}
}

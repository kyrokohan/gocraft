package world

import (
	"math"
	"testing"
)

func TestTarPitsLieInTheSwampAndTheBadlandsAndAreTheSameEveryTime(t *testing.T) {
	const seed = 7
	s := uint32(seed)
	in := map[Biome]int{}
	var first tarPit
	for gz := -40; gz <= 40; gz++ {
		for gx := -40; gx <= 40; gx++ {
			p, ok := tarIn(gx, gz, s)
			if again, also := tarIn(gx, gz, s); again != p || also != ok {
				t.Fatalf("the square %d, %d was asked twice for its tar pit, and the answers differ", gx, gz)
			}
			if !ok {
				continue
			}
			land := landAt(p.x, p.z, s).biome
			if in[land]++; land != Swamp && land != Badlands {
				t.Fatalf("a tar pit lies at %d, %d in land %d", p.x, p.z, land)
			}
			if p.wide < 3 || p.wide > 5 || p.x < gx*markSquare+markReach || p.x >= (gx+1)*markSquare-markReach {
				t.Fatalf("the tar pit at %d, %d reaches %.1f from its middle, in the square %d, %d", p.x, p.z, p.wide, gx, gz)
			}
			if _, bones := landmarkIn(gx, gz, s); bones {
				t.Fatalf("the square %d, %d holds a skeleton and a tar pit", gx, gz)
			}
			if first == (tarPit{}) && land == Swamp {
				first = p
			}
		}
	}
	if in[Swamp] < 10 || in[Badlands] < 10 {
		t.Fatalf("%d tar pits in the swamp and %d in the badlands of 6,561 squares, wanted some of each", in[Swamp], in[Badlands])
	}

	// One of them, built: a round black pool three deep, level with the ground at its
	// middle, with a bank all round it, nothing over it, and bones in it.
	w := Generate(seed)
	w.Follow(float64(first.x), float64(first.z))
	w.LoadAll()
	tar, bones := 0, 0
	for dz := -7; dz <= 7; dz++ {
		for dx := -7; dx <= 7; dx++ {
			x, z := first.x+dx, first.z+dz
			far := math.Hypot(float64(dx), float64(dz))
			top := w.At(x, first.y, z)
			switch {
			case far <= first.wide-0.5:
				if top != Tar && top != Bone || w.At(x, first.y-2, z) != Tar && w.At(x, first.y-2, z) != Bone || w.At(x, first.y-3, z) == Tar {
					t.Fatalf("%d, %d from the middle of the tar pit lies %s over %s over %s: wanted tar three deep", dx, dz,
						top.Info().Name, w.At(x, first.y-2, z).Info().Name, w.At(x, first.y-3, z).Info().Name)
				}
				if over := w.At(x, first.y+2, z); over != Air && over != Bone {
					t.Fatalf("over the tar at %d, %d from its middle is %s", dx, dz, over.Info().Name)
				}
			case far > first.wide+0.5 && far <= first.wide+1.5:
				if !top.Info().Solid || top == Water {
					t.Fatalf("the bank %d, %d from the middle of the tar pit is %s, wanted solid ground to its level", dx, dz, top.Info().Name)
				}
			}
			for y := first.y - 2; y <= first.y+2; y++ {
				switch w.At(x, y, z) {
				case Tar:
					tar++
				case Bone:
					bones++
				}
			}
		}
	}
	if tar < 60 || bones < 2 {
		t.Errorf("the tar pit at %d, %d holds %d blocks of tar and %d of bone", first.x, first.z, tar, bones)
	}
}

func TestTarIsNoSolidThingAndIsTakenByHand(t *testing.T) {
	info := Tar.Info()
	if info.Solid || info.Fixed || info.Plant || info.Glow != 0 || info.Shade != MaxLight {
		t.Errorf("tar is %+v: wanted no solid thing, that can be taken, dark, and lets no light through", info)
	}
	if s, yields := Tar.DigWith(NoItem); !yields || s <= 0 || s > 3 {
		t.Errorf("tar is taken by hand in %v s, and leaves something: %v", s, yields)
	}
	if item, n := Tar.Drop(); item != Tar.Item() || n != 1 || Tar.Item().Name() != "tar" {
		t.Errorf("tar leaves %d of %q", n, item.Name())
	}
	if Tar != ScaleCrown+1 {
		t.Errorf("the numbers moved: tar is %d", Tar)
	}
}

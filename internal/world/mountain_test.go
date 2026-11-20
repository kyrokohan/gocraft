package world

import (
	"math"
	"slices"
	"testing"
)

func TestTheFirstMountainLiesAJourneyFromWhereTheWorldBegins(t *testing.T) {
	for seed := uint32(1); seed <= 20; seed++ {
		x, z := mountPlace(0, 0, seed)
		if far := math.Hypot(float64(x), float64(z)); far < mountNear-1 || far > mountFar+1 {
			t.Errorf("seed %d: the first mountain is %.0f blocks from where the world begins, wanted %d to %d", seed, far, mountNear, mountFar)
		}
		if again, againZ := mountPlace(0, 0, seed); again != x || againZ != z {
			t.Fatalf("seed %d: the seed gave two places for it", seed)
		}
	}
	// It is not in the same place in every world.
	ax, az := mountPlace(0, 0, 1)
	bx, bz := mountPlace(0, 0, 2)
	if ax == bx && az == bz {
		t.Error("two seeds put their first mountain in the same place")
	}
}

func TestThereIsOneMountainToASquareWellInsideIt(t *testing.T) {
	for gz := -6; gz <= 6; gz++ {
		for gx := -6; gx <= 6; gx++ {
			x, z := mountPlace(gx, gz, 7)
			x0, z0 := gx*mountSquare-mountSquare/2, gz*mountSquare-mountSquare/2
			if x-mountAsh < x0 || x+mountAsh >= x0+mountSquare || z-mountAsh < z0 || z+mountAsh >= z0+mountSquare {
				t.Errorf("the mountain of square %d,%d at %d,%d reaches out of it", gx, gz, x, z)
			}
			if m, far, ok := mountainNear(x, z, 7); !ok || far != 0 || m.x != x || m.z != z {
				t.Errorf("the middle of the mountain of square %d,%d is not on it: %+v, %v, %v", gx, gz, m, far, ok)
			}
			if _, _, ok := mountainNear(x+mountAsh+1, z, 7); ok {
				t.Errorf("a column %d from the mountain of square %d,%d is on it", mountAsh+1, gx, gz)
			}
		}
	}
}

func TestACanStandsAboutFortyOverTheLandWithACrater(t *testing.T) {
	for seed := uint32(1); seed <= 12; seed++ {
		x, z := mountPlace(0, 0, seed)
		m, _, _ := mountainNear(x, z, seed)

		round := func(far float64) (lowest, highest int) {
			lowest, highest = Height, 0
			for i := range 24 {
				a := 2 * math.Pi * float64(i) / 24
				l := landAt(x+int(math.Round(far*math.Cos(a))), z+int(math.Round(far*math.Sin(a))), seed)
				lowest, highest = min(lowest, l.top), max(highest, l.top)
			}
			return lowest, highest
		}
		if low, high := round(craterWide); low < m.rim-3 || high > m.rim {
			t.Errorf("seed %d: the rim stands from %d to %d, wanted it at about %d all round", seed, low, high, m.rim)
		}
		footLow, footHigh := round(mountFoot + 6)
		if over := m.rim - (footLow+footHigh)/2; over < 18 || over > 58 {
			t.Errorf("seed %d: the rim at %d is %d over the land at its foot (%d to %d), wanted about %d", seed, m.rim, over, footLow, footHigh, mountHigh)
		}
		middle := landAt(x, z, seed)
		if m.rim-middle.top < craterDeep-1 || middle.lava != m.lava() || middle.biome != Ashlands || !middle.cone {
			t.Errorf("seed %d: the middle of the crater is %d under the rim, with lava to %d: %+v", seed, m.rim-middle.top, middle.lava, middle)
		}
		// It climbs all the way: no step of the cone is higher than a wall.
		for far := mountFoot; far > craterWide; far-- {
			a, b := landAt(x+far, z, seed), landAt(x+far-1, z, seed)
			if b.top-a.top > 6 || b.top < a.top-2 {
				t.Errorf("seed %d: %d from the middle the ground goes from %d to %d in a block", seed, far, a.top, b.top)
			}
		}
		if rim := landAt(x+craterWide, z, seed); rim.lava != 0 {
			t.Errorf("seed %d: there is lava on the rim", seed)
		}
		if out := landAt(x+mountAsh+3, z, seed); out.biome == Ashlands || out.cone {
			t.Errorf("seed %d: the land %d from the middle is ashlands", seed, mountAsh+3)
		}
	}
}

// A mountain is far wider than a chunk, and every chunk it lies in builds its part from
// the seed alone.
func TestAMountainIsTheSameWhicheverChunkIsMadeFirstAndHoldsLava(t *testing.T) {
	const seed = 7
	x, z := mountPlace(0, 0, seed)
	m, _, _ := mountainNear(x, z, seed)
	gen := Terrain(seed)

	var chunks [][2]int
	for cz := floorDiv(z-craterWide-2, ChunkSize); cz <= floorDiv(z+craterWide+2, ChunkSize); cz++ {
		for cx := floorDiv(x-craterWide-2, ChunkSize); cx <= floorDiv(x+craterWide+2, ChunkSize); cx++ {
			chunks = append(chunks, [2]int{cx, cz})
		}
	}
	first := map[[2]int][]Block{}
	for _, c := range chunks {
		first[c] = made(gen, c[0], c[1])
	}
	slices.Reverse(chunks)
	for _, c := range chunks {
		if !slices.Equal(made(gen, c[0], c[1]), first[c]) {
			t.Errorf("chunk %v came out different when its neighbors were made first", c)
		}
	}

	w := newWorld(seed, gen, "", float64(x), float64(z))
	lake, high := 0, 0
	for dz := -mountAsh; dz <= mountAsh; dz++ {
		for dx := -mountAsh; dx <= mountAsh; dx++ {
			for y := range Height {
				if w.At(x+dx, y, z+dz) != Lava {
					continue
				}
				lake, high = lake+1, max(high, y)
				if far := math.Hypot(float64(dx), float64(dz)); far >= craterWide {
					t.Fatalf("there is lava %.1f from the middle, outside the crater", far)
				}
			}
		}
	}
	if lake < 60 || high != m.lava() {
		t.Errorf("the crater holds %d blocks of lava, up to layer %d: wanted a lake up to %d", lake, high, m.lava())
	}
	if w.At(x, m.lava()+1, z) != Air || w.LightAt(x, m.lava()+1, z).Block() != MaxLight-1 {
		t.Errorf("over the lake there is %v, lit %d: wanted air, lit by the lava", w.At(x, m.lava()+1, z), w.LightAt(x, m.lava()+1, z).Block())
	}
	if crater, far, ok := w.Mountain(float64(x)+100.5, float64(z)+0.5); !ok || crater != (Cell{x, m.rim, z}) || math.Abs(far-100) > 0.01 {
		t.Errorf("the mountain nearest a place 100 blocks from this one is %+v, %.1f off, %v", crater, far, ok)
	}
	if _, _, ok := New().Mountain(0, 0); ok {
		t.Error("a world that is not made from a seed has a mountain")
	}
}

func TestAshlandsAreAshAndDarkRockWhereNothingGrows(t *testing.T) {
	const seed = 7
	x, z := mountPlace(0, 0, seed)
	w := newWorld(seed, Terrain(seed), "", float64(x), float64(z))
	ground := map[Block]int{}
	for dz := -mountAsh + 4; dz <= mountAsh-4; dz++ {
		for dx := -mountAsh + 4; dx <= mountAsh-4; dx++ {
			l := landAt(x+dx, z+dz, seed)
			if l.biome != Ashlands || l.lava != 0 {
				continue // the sea, where the mountain stands in it, and the lake
			}
			top := w.At(x+dx, l.top, z+dz)
			ground[top]++
			switch top {
			case Ash, Basalt, ObsidianRock, FirestoneRock, Air: // air: the mouth of a cave
			default:
				t.Fatalf("the ground at %d,%d is %v, wanted ash, dark rock or what lies in it", x+dx, z+dz, top)
			}
			if over := w.At(x+dx, l.top+1, z+dz); over != Air && over != CharredLog {
				t.Fatalf("on the ashlands at %d,%d there is %v, wanted nothing or a charred trunk", x+dx, z+dz, over)
			}
		}
	}
	if ground[Ash] < 2000 || ground[Basalt] < 500 {
		t.Errorf("the ashlands are %d ash and %d dark rock, wanted plenty of both", ground[Ash], ground[Basalt])
	}
}

func TestLavaGlowsAndIsNoSolidThingAndCannotBeTaken(t *testing.T) {
	info := Lava.Info()
	if info.Glow != MaxLight || info.Solid || !info.Fixed || info.Shade != 0 || info.Stands() {
		t.Errorf("lava: glow %d, solid %v, fixed %v, shade %d", info.Glow, info.Solid, info.Fixed, info.Shade)
	}
	if !Basalt.Info().Solid || !Ash.Info().Solid || Ash.Info().Dig >= Basalt.Info().Dig {
		t.Error("ash and dark rock are not solid, or ash is no softer than rock")
	}
	if Ash != PitCover+1 || Basalt != Ash+1 || Lava != Basalt+1 || Ashlands != Highland+1 {
		t.Errorf("the numbers moved: ash %d, dark rock %d, lava %d, ashlands %d", Ash, Basalt, Lava, Ashlands)
	}
}

func TestObsidianAndFirestoneAreFoundOnAFireMountainAndNowhereElse(t *testing.T) {
	const seed = 7
	x, z := mountPlace(0, 0, seed)
	w := newWorld(seed, Terrain(seed), "", float64(x), float64(z))
	obsidian, firestone, far := 0, 0, 0.0
	for dz := -mountAsh; dz <= mountAsh; dz++ {
		for dx := -mountAsh; dx <= mountAsh; dx++ {
			out := math.Hypot(float64(dx), float64(dz))
			for y := SeaLevel; y < Height; y++ {
				switch w.At(x+dx, y, z+dz) {
				case ObsidianRock:
					if obsidian++; out >= mountFoot {
						t.Fatalf("obsidian lies %.0f from the middle, off the cone", out)
					}
				case FirestoneRock:
					firestone++
					far = max(far, math.Abs(out-craterWide))
					if landAt(x+dx, z+dz, seed).lava != 0 {
						t.Fatalf("firestone lies under the lake, at %d,%d,%d", x+dx, y, z+dz)
					}
				}
			}
		}
	}
	if obsidian < 60 || firestone < 12 {
		t.Errorf("the mountain holds %d blocks of obsidian and %d of firestone, wanted veins of the one and patches of the other", obsidian, firestone)
	}
	if far > firestoneRing+1.5 {
		t.Errorf("firestone lies %.1f from the rim of the crater, wanted it within %v", far, firestoneRing)
	}

	// Round where the world begins there is neither.
	home := Generate(seed)
	for dz := -80; dz <= 80; dz++ {
		for dx := -80; dx <= 80; dx++ {
			for y := range Height {
				if b := home.At(dx, y, dz); b == ObsidianRock || b == FirestoneRock || b == Lava {
					t.Fatalf("there is %v at %d,%d,%d, far from any mountain", b, dx, y, dz)
				}
			}
		}
	}
}

func TestObsidianNeedsAFlintPickAndFirestoneBreaksByHand(t *testing.T) {
	if _, yields := ObsidianRock.DigWith(NoItem); yields {
		t.Error("obsidian breaks out by hand")
	}
	if _, yields := ObsidianRock.DigWith(DiggingStick); yields {
		t.Error("obsidian breaks out with a digging stick")
	}
	if _, yields := ObsidianRock.DigWith(FlintPick); !yields {
		t.Error("a flint pick does not break obsidian out")
	}
	if item, n := ObsidianRock.Drop(); item != Obsidian || n != 1 {
		t.Errorf("a vein of obsidian leaves %d of %v", n, item)
	}
	if _, yields := FirestoneRock.DigWith(NoItem); !yields {
		t.Error("firestone does not break by hand")
	}
	if item, n := FirestoneRock.Drop(); item != Firestone || n != 1 {
		t.Errorf("firestone leaves %d of %v", n, item)
	}
}

func TestToolsOfObsidianAreTheSharpestAndTheFirstToBreak(t *testing.T) {
	for _, pair := range [][2]Item{{ObsidianKnife, FlintKnife}, {ObsidianPick, FlintPick}, {ObsidianSpear, HardSpear}} {
		glass, other := pair[0], pair[1]
		if glass.Uses() != 40 || glass.Uses() >= other.Uses()/2 || glass.StackSize() != 1 {
			t.Errorf("%s lasts %d uses and %s %d: wanted 40, and well under half", glass.Name(), glass.Uses(), other.Name(), other.Uses())
		}
		if glass.Damage() < other.Damage() {
			t.Errorf("%s hits for %v and %s for %v", glass.Name(), glass.Damage(), other.Name(), other.Damage())
		}
	}
	if ObsidianKnife.Damage() != 5 || !ObsidianKnife.Carves() || ObsidianSpear.Damage() != 5 || ObsidianSpear.Reach() != 1 || !ObsidianSpear.Thrown() {
		t.Errorf("knife: %v, carves %v; spear: %v, reach %v, thrown %v", ObsidianKnife.Damage(), ObsidianKnife.Carves(), ObsidianSpear.Damage(), ObsidianSpear.Reach(), ObsidianSpear.Thrown())
	}
	glass, _ := Stone.DigWith(ObsidianPick)
	flint, _ := Stone.DigWith(FlintPick)
	if glass >= flint {
		t.Errorf("an obsidian pick digs stone in %v s and a flint one in %v", glass, flint)
	}
	// Crystal, which nothing could break out, now can be.
	if _, yields := CrystalOre.DigWith(FlintPick); yields {
		t.Error("a flint pick breaks out crystal")
	}
	if _, yields := CrystalOre.DigWith(ObsidianPick); !yields {
		t.Error("an obsidian pick does not break out crystal")
	}
	if !Spear.Thrown() || !HardSpear.Thrown() || Club.Thrown() || ObsidianKnife.Thrown() {
		t.Error("what is thrown: wanted the three spears and nothing else")
	}
	if Striker.Uses() != 30 || Striker.StackSize() != 1 || Obsidian.StackSize() != StackSize || Firestone.StackSize() != StackSize {
		t.Errorf("a striker lasts %d fires, %d to a slot", Striker.Uses(), Striker.StackSize())
	}
	if Obsidian != Sling+1 || Striker != Obsidian+5 || ObsidianRock != Lava+1 || FirestoneRock != ObsidianRock+1 {
		t.Errorf("the numbers moved: obsidian %d, striker %d, obsidian rock %d, firestone rock %d", Obsidian, Striker, ObsidianRock, FirestoneRock)
	}
}

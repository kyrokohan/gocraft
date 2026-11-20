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
				if _, floor := m.tube(seed); y <= floor+4 {
					continue // the lava of the tube, deep in the cone: its pool, and what hangs in its roof
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
			case Water, SinterRust, SinterGold, SinterGreen: // a hot spring
			default:
				t.Fatalf("the ground at %d,%d is %v, wanted ash, dark rock or what lies in it", x+dx, z+dz, top)
			}
			if m, _, _ := mountainNear(x, z, seed); slices.ContainsFunc(m.springs(seed), func(s spring) bool {
				return math.Hypot(float64(x+dx-s.x), float64(z+dz-s.z)) <= springReach
			}) {
				continue // a hot spring, whose ground is made level
			}
			if way, _ := mountainOf(x, z, seed).tube(seed); math.Abs(float64(dx)*way[1]-float64(dz)*way[0]) < 3 {
				continue // the mouth of the lava tube, which is cut into the ground
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
					if _, chamber, _ := w.Tube(float64(x), float64(z)); math.Hypot(float64(x+dx-chamber.X), float64(z+dz-chamber.Z)) < chamberWide+2 {
						continue // in the wall of the chamber of the lava tube, which is deep in the cone
					}
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

func TestHotSpringsLieOnTheAshRoundAMountainTheSameEveryTime(t *testing.T) {
	geysers := 0
	for seed := uint32(1); seed <= 12; seed++ {
		x, z := mountPlace(0, 0, seed)
		m, _, _ := mountainNear(x, z, seed)
		all := m.springs(seed)
		if again := m.springs(seed); !slices.Equal(again, all) {
			t.Fatalf("seed %d: the seed gave two sets of springs", seed)
		}
		if len(all) > 4 {
			t.Errorf("seed %d: %d springs, wanted four at the most", seed, len(all))
		}
		for i, s := range all {
			far := math.Hypot(float64(s.x-x), float64(s.z-z))
			if far < springNear-1 || far > springFar+1 || landAt(s.x, s.z, seed).biome != Ashlands || landAt(s.x, s.z, seed).cone {
				t.Errorf("seed %d: spring %d lies %.0f from the middle, in land %d: wanted it on the ash off the cone", seed, i, far, landAt(s.x, s.z, seed).biome)
			}
			if s.geyser {
				geysers++
			}
			if got, ok := springAt(s.x, s.z, seed); !ok || got != s {
				t.Errorf("seed %d: the middle of spring %d is not in its own water", seed, i)
			}
			if _, ok := springAt(s.x+int(s.wide)+2, s.z, seed); ok {
				t.Errorf("seed %d: a column on the rings of spring %d is in its water", seed, i)
			}
		}
		if len(all) > 0 && !all[0].geyser {
			t.Errorf("seed %d: the first spring has no geyser", seed)
		}
	}
	if geysers < 12 {
		t.Errorf("%d geysers in twelve worlds, wanted one or more to a mountain", geysers)
	}
	if _, ok := springAt(0, 0, 7); ok {
		t.Error("where the world begins is in a hot spring")
	}
}

func TestAHotSpringIsAPoolInRingsOfColoredStone(t *testing.T) {
	const seed = 7
	x, z := mountPlace(0, 0, seed)
	m, _, _ := mountainNear(x, z, seed)
	w := newWorld(seed, Terrain(seed), "", float64(x), float64(z))
	for i, s := range m.springs(seed) {
		if w.At(s.x, s.y, s.z) != Water || !w.At(s.x, s.y-1, s.z).Info().Solid || w.At(s.x, s.y+1, s.z) != Air {
			t.Errorf("spring %d: its middle is %v over %v under %v, wanted water one deep, on a floor, under the air", i, w.At(s.x, s.y, s.z), w.At(s.x, s.y-1, s.z), w.At(s.x, s.y+1, s.z))
		}
		rings := map[Block]int{}
		for dz := -springReach; dz <= springReach; dz++ {
			for dx := -springReach; dx <= springReach; dx++ {
				b := w.At(s.x+dx, s.y, s.z+dz)
				rings[b]++
				if far := math.Hypot(float64(dx), float64(dz)); far <= s.wide+3.4 && w.At(s.x+dx, s.y+1, s.z+dz) != Air {
					t.Fatalf("spring %d: over its ground at %d,%d there is %v", i, dx, dz, w.At(s.x+dx, s.y+1, s.z+dz))
				}
			}
		}
		if rings[Water] < 12 || rings[SinterRust] < 8 || rings[SinterGold] < 12 || rings[SinterGreen] < 12 {
			t.Errorf("spring %d: %d water, %d rust, %d gold, %d green: wanted a pool in three rings", i, rings[Water], rings[SinterRust], rings[SinterGold], rings[SinterGreen])
		}
		if !w.Spring(float64(s.x)+0.5, float64(s.z)+0.5) || w.Spring(float64(s.x)+s.wide+2.5, float64(s.z)+0.5) {
			t.Errorf("spring %d: its middle is in its water %v, and its rings %v", i, w.Spring(float64(s.x)+0.5, float64(s.z)+0.5), w.Spring(float64(s.x)+s.wide+2.5, float64(s.z)+0.5))
		}
	}
	vents := w.Geysers(float64(x), float64(z), 80)
	if len(vents) == 0 || w.At(vents[0].X, vents[0].Y, vents[0].Z) != Water {
		t.Errorf("the geysers of the mountain spout from %v", vents)
	}
	if far := w.Geysers(float64(x)+300, float64(z), 20); len(far) != 0 {
		t.Errorf("300 blocks from the mountain there are %d geysers within 20", len(far))
	}
	if New().Spring(0, 0) || len(New().Geysers(0, 0, 100)) != 0 {
		t.Error("a world that is not made from a seed has springs")
	}
}

func TestALavaTubeRunsIntoTheConeTwoBlocksWideToAChamber(t *testing.T) {
	for _, seed := range []int64{7, 11, 2024} {
		s := uint32(seed)
		x, z := mountPlace(0, 0, s)
		m, _, _ := mountainNear(x, z, s)
		w := newWorld(seed, Terrain(seed), "", float64(x), float64(z))
		mouth, chamber, ok := w.Tube(float64(x), float64(z))
		if !ok {
			t.Fatalf("seed %d: the mountain has no tube", seed)
		}
		way, floor := m.tube(s)
		lit := 0

		// Along it, from the chamber out: three blocks of air over a floor, rock two
		// blocks to each side, and a roof, with lava in it here and there.
		for far := tubeFrom + 5.0; far <= tubeTo-8; far += 1 { // from where it opens into the chamber
			cx, cz := float64(x)+0.5+far*way[0], float64(z)+0.5+far*way[1]
			at := func(aside float64) (int, int) {
				return int(math.Floor(cx + aside*way[1])), int(math.Floor(cz - aside*way[0]))
			}
			if ax, az := at(0); w.At(ax, floor+1, az) != Air || w.At(ax, floor+3, az) != Air || !w.At(ax, floor, az).Info().Solid {
				t.Fatalf("seed %d: %.0f along the tube there is %v on %v, wanted air on a floor", seed, far, w.At(ax, floor+1, az), w.At(ax, floor, az))
			}
			for _, aside := range []float64{-2.6, 2.6} {
				if ax, az := at(aside); far > tubeFrom+6 && !w.At(ax, floor+1, az).Info().Solid && !w.At(ax, floor+2, az).Info().Solid {
					t.Fatalf("seed %d: %.0f along the tube it is open %.1f to the side: wanted it two blocks wide", seed, far, aside)
				}
			}
			if ax, az := at(0); far < tubeTo-12 {
				switch roof := w.At(ax, floor+4, az); {
				case roof == Lava:
					lit++
				case !roof.Info().Solid:
					t.Fatalf("seed %d: %.0f along the tube it has no roof", seed, far)
				}
			}
		}
		if lit < 1 {
			t.Errorf("seed %d: lava hangs in the roof of the tube in %d places, wanted some to light it", seed, lit)
		}
		// The chamber: room, a pool of lava in its floor, and what the mountain keeps in its walls.
		for far := tubeFrom + 2.0; far < tubeFrom+5; far++ { // and the way from the pool to there is open
			if ax, az := int(math.Floor(float64(x)+0.5+far*way[0])), int(math.Floor(float64(z)+0.5+far*way[1])); w.At(ax, floor+1, az) != Air || w.At(ax, floor+2, az) != Air {
				t.Errorf("seed %d: %.0f along, between the chamber and the tube, there is %v under %v", seed, far, w.At(ax, floor+1, az), w.At(ax, floor+2, az))
			}
		}
		if w.At(chamber.X, floor, chamber.Z) != Lava || w.At(chamber.X, floor+2, chamber.Z) != Air {
			t.Errorf("seed %d: the middle of the chamber is %v under %v, wanted lava under air", seed, w.At(chamber.X, floor, chamber.Z), w.At(chamber.X, floor+2, chamber.Z))
		}
		kept := 0
		for dz := -7; dz <= 7; dz++ {
			for dx := -7; dx <= 7; dx++ {
				for y := floor + 1; y <= floor+3; y++ {
					if b := w.At(chamber.X+dx, y, chamber.Z+dz); b == ObsidianRock || b == FirestoneRock {
						kept++
					}
				}
			}
		}
		if kept < 10 {
			t.Errorf("seed %d: the walls of the chamber hold %d blocks of obsidian and firestone", seed, kept)
		}
		// Its mouth is open to the sky, on the side of the cone.
		if w.At(mouth.X, mouth.Y, mouth.Z) != Air || w.LightAt(mouth.X, mouth.Y+1, mouth.Z).Sky() < 10 {
			t.Errorf("seed %d: the mouth of the tube at %+v is %v, with the sky's light at %d", seed, mouth, w.At(mouth.X, mouth.Y, mouth.Z), w.LightAt(mouth.X, mouth.Y+1, mouth.Z).Sky())
		}
	}
}

func TestTheSpringsAndTheTubeAreTheSameWhicheverChunkIsMadeFirst(t *testing.T) {
	const seed = 7
	x, z := mountPlace(0, 0, seed)
	gen := Terrain(seed)
	var chunks [][2]int
	for cz := floorDiv(z-mountAsh, ChunkSize); cz <= floorDiv(z+mountAsh, ChunkSize); cz++ {
		for cx := floorDiv(x-mountAsh, ChunkSize); cx <= floorDiv(x+mountAsh, ChunkSize); cx++ {
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
			t.Fatalf("chunk %v came out different when its neighbors were made first", c)
		}
	}
}

func TestAFireMountainEruptsEveryFourToSevenDaysByTheSeed(t *testing.T) {
	for seed := int64(1); seed <= 8; seed++ {
		w := Generate(seed)
		crater, _, _ := w.Mountain(0, 0)
		x, z := float64(crater.X), float64(crater.Z)
		if began, _, ok := w.Erupted(x, z, 3.99); ok {
			t.Errorf("seed %d: the mountain began to erupt on day %.2f, before the fourth day", seed, began)
		}
		last, count := 0.0, 0
		for day := 0.0; day < 80; day += 0.01 {
			began, next, ok := w.Erupted(x, z, day)
			if then, _, _ := w.Erupted(x, z, next); next <= day || then != next {
				t.Fatalf("seed %d: on day %.2f the next eruption is to begin on day %.2f, and on that day the last began on day %.2f", seed, day, next, then)
			}
			if !ok {
				continue
			}
			if again, _, _ := w.Erupted(x, z, day); again != began || began > day {
				t.Fatalf("seed %d: on day %.2f the last eruption began on day %.2f, and then on day %.2f", seed, day, began, again)
			}
			if began != last {
				if gap := began - last; gap < 4 || gap > 7 {
					t.Errorf("seed %d: an eruption began on day %.2f, %.2f days after the last: wanted four to seven", seed, began, gap)
				}
				last, count = began, count+1
			}
		}
		if count < 11 || count > 20 {
			t.Errorf("seed %d: %d eruptions in 80 days, wanted 11 to 20", seed, count)
		}
	}

	// Two mountains of one world do not keep the same days.
	w := Generate(7)
	first := func(x, z float64) float64 {
		for day := 4.0; day < 8; day += 0.01 {
			if began, _, ok := w.Erupted(x, z, day); ok {
				return began
			}
		}
		t.Fatalf("the mountain near %.0f, %.0f did not erupt in a week", x, z)
		return 0
	}
	a, _, _ := w.Mountain(0, 0)
	b, _, _ := w.Mountain(float64(a.X)+mountSquare, float64(a.Z))
	if a == b || first(float64(a.X), float64(a.Z)) == first(float64(b.X), float64(b.Z)) {
		t.Errorf("the mountains at %v and %v first erupt on the same day", a, b)
	}
	if _, _, ok := New().Erupted(0, 0, 100); ok {
		t.Error("a world that is not made from a seed has a mountain that erupted")
	}
}

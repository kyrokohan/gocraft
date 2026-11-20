package world

import (
	"math"
	"slices"
	"testing"
)

// made returns the chunk a generator makes at a place.
func made(gen Generator, cx, cz int) []Block {
	blocks := make([]Block, chunkBlocks)
	gen(cx, cz, blocks)
	return blocks
}

// held returns the blocks a world holds for a chunk, read a block at a time.
func held(w *World, cx, cz int) []Block {
	blocks := make([]Block, 0, chunkBlocks)
	for y := range Height {
		for z := range ChunkSize {
			for x := range ChunkSize {
				blocks = append(blocks, w.At(cx*ChunkSize+x, y, cz*ChunkSize+z))
			}
		}
	}
	return blocks
}

func TestSetAndAt(t *testing.T) {
	w := New()

	w.Set(1, 2, 3, Brick)
	w.Set(-40, 100, -7, Glass) // the world goes on below 0 on x and z

	if got := w.At(1, 2, 3); got != Brick {
		t.Errorf("got %v, wanted Brick", got)
	}
	if got := w.At(-40, 100, -7); got != Glass {
		t.Errorf("got %v at a place below 0, wanted Glass", got)
	}
	if got := w.At(3, 2, 1); got != Air {
		t.Errorf("got %v, wanted Air", got)
	}
}

func TestAboveBelowAndFarAwayIsAirAndCannotBeSet(t *testing.T) {
	w := New()

	for _, p := range [][3]int{
		{0, -1, 0}, {0, Height, 0}, // below and above the world
		{4000, 10, 0}, {0, 10, -4000}, // where the world is not loaded
	} {
		w.Set(p[0], p[1], p[2], Stone)
		if got := w.At(p[0], p[1], p[2]); got != Air {
			t.Errorf("At(%v) is %v, wanted Air", p, got)
		}
	}
}

func TestHeightIsAboveTheHighestBlock(t *testing.T) {
	w := New()
	if got := w.Height(); got != 0 {
		t.Errorf("an empty world is %d high, wanted 0", got)
	}

	w.Set(4, 30, 4, Stone)
	w.Set(4, 7, 4, Stone)

	if got := w.Height(); got != 31 {
		t.Errorf("got %d, wanted 31", got)
	}
}

func TestSurfaceIsTheHighestBlockOfAColumn(t *testing.T) {
	w := New()
	w.Set(2, 5, 2, Stone)
	w.Set(2, 9, 2, Leaves)

	if got := w.Surface(2, 2); got != 9 {
		t.Errorf("got %d, wanted 9", got)
	}
	if got := w.Surface(3, 3); got != -1 {
		t.Errorf("an empty column is %d, wanted -1", got)
	}
}

func TestTheSameSeedMakesTheSameChunk(t *testing.T) {
	for _, at := range [][2]int{{0, 0}, {3, -2}, {-17, 40}, {1000, -1000}} {
		a, b, c := made(Terrain(7), at[0], at[1]), made(Terrain(7), at[0], at[1]), made(Terrain(8), at[0], at[1])

		if !slices.Equal(a, b) {
			t.Errorf("chunk %v: the same seed made two different chunks", at)
		}
		if slices.Equal(a, c) {
			t.Errorf("chunk %v: two seeds made the same chunk", at)
		}
	}
}

// A chunk is made from the seed and its place alone, so it is the same whether it is
// made by itself or as part of a world, whatever was made before it. That is what lets
// a tree stand on the line between two chunks.
func TestAChunkDoesNotDependOnItsNeighbors(t *testing.T) {
	w := Generate(5)
	gen := Terrain(5)

	for cz := -LoadRadius; cz <= LoadRadius; cz++ {
		for cx := -LoadRadius; cx <= LoadRadius; cx++ {
			if !slices.Equal(held(w, cx, cz), made(gen, cx, cz)) {
				t.Fatalf("chunk %d,%d in a world is not the chunk the seed makes by itself", cx, cz)
			}
		}
	}
}

// find returns a column of a kind of country in the world of a seed, well inside it:
// with the same country a little way off on every side.
func find(t *testing.T, seed int64, want Biome) (x, z int) {
	t.Helper()
	for reach := 0; reach < 6000; reach += 48 {
		for z := -reach; z <= reach; z += 48 {
			for x := -reach; x <= reach; x += 48 {
				if max(abs(x), abs(z)) != reach {
					continue
				}
				inside := true
				for _, d := range [][2]int{{0, 0}, {24, 0}, {-24, 0}, {0, 24}, {0, -24}} {
					inside = inside && BiomeAt(seed, x+d[0], z+d[1]) == want
				}
				if inside {
					return x, z
				}
			}
		}
	}
	t.Fatalf("seed %d has no country of kind %d within 6,000 blocks", seed, want)
	return 0, 0
}

// counted returns how many of each kind of block the loaded part of a world holds.
func counted(w *World, x, z int) map[Block]int {
	count := map[Block]int{}
	span := LoadRadius * ChunkSize
	for dz := -span; dz < span; dz++ {
		for dx := -span; dx < span; dx++ {
			for y := 1; y < Height; y++ {
				count[w.At(x+dx, y, z+dz)]++
			}
		}
	}
	return count
}

func TestTreesAreWholeAcrossChunkEdges(t *testing.T) {
	// A redwood's needles are told from an umbrella pine's by the trunk they are near.
	trunksOf := map[Block][]Block{
		Leaves: {Log}, PineNeedles: {PineLog, RedwoodLog}, FanLeaves: {PaleLog}, Fronds: {FernTrunk},
	}

	for _, country := range []Biome{Jungle, Highland, Tundra} {
		w, x0, z0 := loadedAt(t, 5, country)

		// Every leaf is within reach of a log of its own tree: a tree cut off at the
		// edge of a chunk would leave leaves with no trunk.
		leaves, orphans := 0, 0
		span := (LoadRadius - 1) * ChunkSize
		for z := z0 - span; z < z0+span; z++ {
			for x := x0 - span; x < x0+span; x++ {
				for y := SeaLevel; y < Height; y++ {
					trunks, isLeaf := trunksOf[w.At(x, y, z)]
					if !isLeaf {
						continue
					}
					leaves++
					near := false
					for dz := -treeReach; dz <= treeReach && !near; dz++ {
						for dx := -treeReach; dx <= treeReach && !near; dx++ {
							for dy := -3; dy <= 2 && !near; dy++ {
								near = slices.Contains(trunks, w.At(x+dx, y+dy, z+dz))
							}
						}
					}
					if !near {
						orphans++
					}
				}
			}
		}
		if leaves < 200 || orphans != 0 {
			t.Errorf("country %d: %d leaves, %d of them with no trunk in reach", country, leaves, orphans)
		}
	}
}

func TestTheBottomIsDeepstoneAndTheTopIsAir(t *testing.T) {
	w := Generate(1)

	span := LoadRadius * ChunkSize
	for z := -span; z < span; z++ {
		for x := -span; x < span; x++ {
			if !w.Loaded(x, z) {
				t.Fatalf("the column at %d,%d is not loaded", x, z)
			}
			if got := w.At(x, 0, z); got != Deepstone {
				t.Fatalf("the bottom at %d,%d is %v, wanted Deepstone", x, z, got)
			}
			if w.At(x, Height-1, z) != Air {
				t.Fatalf("the top layer at %d,%d is not air", x, z)
			}
			for y := 1; y < Height; y++ {
				if w.At(x, y, z) == Deepstone {
					t.Fatalf("deepstone above the bottom layer at %d,%d,%d", x, y, z)
				}
			}
		}
	}
}

func TestEveryKindOfCountryHasItsOwnGroundAndGrowth(t *testing.T) {
	for _, tc := range []struct {
		name    string
		country Biome
		holds   []Block
		lacks   []Block
	}{
		{"sea", Sea, []Block{Water, Sand}, nil},
		{"prairie", Prairie, []Block{Turf, Dirt, Fern, TallFern, Horsetail, StumpPalm, PalmCrown}, []Block{Sandstone, Redrock, PineLog}},
		{"jungle", Jungle, []Block{Turf, FernTrunk, Fronds, Log, Leaves, TallFern, Bush, RedFlower}, []Block{Redrock, PineLog}},
		{"highland", Highland, []Block{Turf, PineLog, PineNeedles, PaleLog, FanLeaves, RedwoodLog}, []Block{Redrock, FernTrunk}},
		{"badlands", Badlands, []Block{Redrock, Sandstone, Sand}, []Block{Log, PineLog, Turf, StumpPalm}},
		{"tundra", Tundra, []Block{Snow, PineLog, PineNeedles}, []Block{StumpPalm, Redrock}},
		{"crags", Crags, []Block{Stone}, nil},
	} {
		w, x, z := loadedAt(t, 7, tc.country)

		// Only the columns of this country: others may reach into what is loaded.
		count := map[Block]int{}
		span := LoadRadius * ChunkSize
		for dz := -span; dz < span; dz++ {
			for dx := -span; dx < span; dx++ {
				if BiomeAt(7, x+dx, z+dz) != tc.country {
					continue
				}
				for y := 1; y < Height; y++ {
					count[w.At(x+dx, y, z+dz)]++
				}
			}
		}
		for _, b := range tc.holds {
			if count[b] == 0 {
				t.Errorf("the %s round %d,%d has no %s", tc.name, x, z, b.Info().Name)
			}
		}
		for _, b := range tc.lacks {
			if count[b] != 0 {
				t.Errorf("the %s round %d,%d has %d of %s", tc.name, x, z, count[b], b.Info().Name)
			}
		}
	}
}

func TestTheWorldIsAMixOfEveryKindOfCountry(t *testing.T) {
	for seed := int64(1); seed <= 3; seed++ {
		share := map[Biome]float64{}
		n := 0.0
		for z := -6000; z < 6000; z += 61 {
			for x := -6000; x < 6000; x += 61 {
				share[BiomeAt(seed, x, z)]++
				n++
			}
		}
		for b := Sea; b < biomeCount; b++ {
			least := 0.01
			if b == Ashlands {
				least = 0.002 // only round the fire mountains, of which there are few
			}
			if part := share[b] / n; part < least || part > 0.6 {
				t.Errorf("seed %d: country %d is %.1f%% of the world, wanted between %v and 60", seed, b, part*100, least*100)
			}
		}
		if ash := share[Ashlands] / n; ash > 0.012 {
			t.Errorf("seed %d: %.1f%% of the world is ashlands, wanted about half of one", seed, ash*100)
		}
		if sea := share[Sea] / n; sea < 0.12 || sea > 0.5 {
			t.Errorf("seed %d: %.0f%% of the world is sea, wanted between 12 and 50", seed, sea*100)
		}
	}
}

func TestHorsetailsStandAtTheWatersEdgeAndIceLiesOnColdWater(t *testing.T) {
	w, x, z := loadedAt(t, 7, Sea)

	span := LoadRadius * ChunkSize
	for dz := -span + 1; dz < span-1; dz++ {
		for dx := -span + 1; dx < span-1; dx++ {
			cx, cz := x+dx, z+dz
			if w.At(cx, SeaLevel+2, cz) == Horsetail {
				wet := w.At(cx+1, SeaLevel, cz) == Water || w.At(cx-1, SeaLevel, cz) == Water ||
					w.At(cx, SeaLevel, cz+1) == Water || w.At(cx, SeaLevel, cz-1) == Water
				if !wet || w.At(cx, SeaLevel+1, cz) != Sand {
					t.Fatalf("a horsetail at %d,%d does not stand on sand beside water", cx, cz)
				}
			}
			if w.At(cx, SeaLevel, cz) == Ice && w.At(cx, SeaLevel-1, cz) != Water && w.At(cx, SeaLevel-1, cz).Info().Plant {
				t.Fatalf("ice at %d,%d lies on %v", cx, cz, w.At(cx, SeaLevel-1, cz))
			}
		}
	}
}

func TestSpawnStandsOnDryGroundWithRoomAbove(t *testing.T) {
	for seed := int64(1); seed <= 5; seed++ {
		w := Generate(seed)

		p := w.Spawn()

		x, y, z := int(p.X-0.5), int(p.Y), int(p.Z-0.5)
		if !w.At(x, y-1, z).Info().Solid {
			t.Errorf("seed %d: spawn at %v does not stand on solid ground", seed, p)
		}
		// A plant at its feet is no obstacle; its head needs air.
		if feet := w.At(x, y, z); (feet != Air && !feet.Info().Plant) || w.At(x, y+1, z) != Air {
			t.Errorf("seed %d: spawn at %v has no room for the player", seed, p)
		}
	}
}

// The window wraps round, so a far place shares its part of the list with a near one.
// After a move, what the world holds must be the far place's own ground.
func TestAFarPlaceHoldsItsOwnGround(t *testing.T) {
	w := Generate(3)
	s := uint32(3)

	for _, to := range [][2]float64{{10_000, -7_000}, {-123_456.5, 99.5}, {8, 8}, {263, 8}} {
		w.Follow(to[0], to[1])
		w.LoadAll()

		for _, d := range [][2]int{{0, 0}, {-90, 40}, {95, -95}, {-1, -1}} {
			x, z := int(to[0])+d[0], int(to[1])+d[1]
			l := landAt(x, z, s)
			top := l.top
			if holeAt(x, top, z, l, s) {
				continue // the mouth of a cave: the ground here is lower than the land says
			}
			want := top
			if top < SeaLevel {
				want = SeaLevel // water lies over low ground
			}
			// A tree may stand on the column, so the ground is checked, not the top block.
			if got := w.At(x, top, z); got == Air || got == Water || w.Surface(x, z) < want {
				t.Fatalf("round %v the column at %d,%d has %v at its ground level %d and its surface at %d",
					to, x, z, got, top, w.Surface(x, z))
			}
			if w.At(x, top+1, z) == Stone {
				t.Fatalf("round %v the column at %d,%d has stone above its ground", to, x, z)
			}
		}
	}
}

func TestAChangedChunkComesBackAfterItLeftTheWindow(t *testing.T) {
	w := Generate(2)
	y := w.Surface(5, 5) + 3
	w.Set(5, y, 5, Brick)
	unchanged := held(w, 3, 3)

	w.Follow(5_000, 0) // far enough that the chunk leaves the window
	w.LoadAll()
	if w.Loaded(5, 5) || w.At(5, y, 5) != Air {
		t.Fatal("the chunk is still in the window after the move")
	}

	w.Follow(5, 5)
	w.LoadAll()
	if got := w.At(5, y, 5); got != Brick {
		t.Errorf("the block that was placed is %v after coming back, wanted Brick", got)
	}
	if !slices.Equal(held(w, 3, 3), unchanged) {
		t.Error("a chunk that was not changed came back different")
	}
}

// A player who runs or flies in a straight line never sees a place that is not loaded,
// with one chunk brought in a frame.
func TestWalkingFarNeverOutrunsTheWorld(t *testing.T) {
	w := Generate(4)
	const step = 10.0 / 30 // flying, at 30 frames a second

	for x := 0.0; x < 2000; x += step {
		w.Follow(x, 0.5)
		w.Load(1)

		// As far as anything is drawn, every way.
		for _, d := range [][2]int{{64, 0}, {-64, 0}, {0, 64}, {0, -64}, {64, 64}, {-64, -64}} {
			if !w.Loaded(int(x)+d[0], d[1]) {
				t.Fatalf("at x=%.1f the column %d, %d away is not loaded", x, d[0], d[1])
			}
		}
	}
}

func TestLoadBringsInTheNearestChunksFirst(t *testing.T) {
	w := Generate(4)
	w.Follow(3_000, 3_000) // a place with nothing loaded

	left := w.Load(1)
	if !w.Loaded(3_000, 3_000) {
		t.Error("the first chunk loaded is not the one the player is in")
	}
	if want := (2*LoadRadius+1)*(2*LoadRadius+1) - 1; left != want {
		t.Errorf("%d chunks left to load, wanted %d", left, want)
	}

	w.Load(8)
	for _, d := range [][2]int{{-16, 0}, {16, 0}, {0, -16}, {0, 16}, {16, 16}} {
		if !w.Loaded(3_000+d[0], 3_000+d[1]) {
			t.Errorf("the chunk %v from the player is not among the first nine loaded", d)
		}
	}
	if w.Load(1_000) != 0 {
		t.Error("chunks are still missing after loading them all")
	}
}

// One chunk, made from the seed.
func BenchmarkChunk(b *testing.B) {
	gen := Terrain(1)
	blocks := make([]Block, chunkBlocks)
	i := 0
	for b.Loop() {
		clear(blocks)
		gen(i%64, i/64, blocks)
		i++
	}
}

// What the world does in a frame while the player flies in a straight line: follow,
// and bring one chunk in.
func BenchmarkFollowAndLoad(b *testing.B) {
	w := Generate(1)
	x := 0.0
	for b.Loop() {
		x += 10.0 / 30
		w.Follow(x, 0)
		w.Load(1)
	}
}

func TestGroundIsUnderThePlantsAndTheMats(t *testing.T) {
	w := New()
	w.Set(2, 5, 2, Turf)
	w.Set(2, 6, 2, Fern)

	if got := w.Ground(2, 2); got != 5 {
		t.Errorf("the ground under a plant is %d, wanted 5", got)
	}
	if got := w.Surface(2, 2); got != 6 {
		t.Errorf("the surface of a column with a plant is %d, wanted the plant at 6", got)
	}
	if got := w.Ground(3, 3); got != -1 {
		t.Errorf("an empty column has ground at %d, wanted -1", got)
	}
	w.Set(4, 5, 4, Stone)
	w.Set(4, 6, 4, FurBed)
	if got := w.Ground(4, 4); got != 5 {
		t.Errorf("the ground under a bedroll is %d, wanted 5", got)
	}
}

func TestEveryBlockHasANameAndPlantsAreNotSolid(t *testing.T) {
	for b := Turf; b < blockCount; b++ {
		info := b.Info()
		if info.Name == "" {
			t.Errorf("block %d has no name", b)
		}
		if info.Plant && info.Solid {
			t.Errorf("%s is a plant and solid", info.Name)
		}
	}
}

// loadedAt returns the world of a seed with the chunks round a column of one kind of
// country loaded.
func loadedAt(t *testing.T, seed int64, country Biome) (w *World, x, z int) {
	t.Helper()
	x, z = find(t, seed, country)
	return newWorld(seed, Terrain(seed), "", float64(x), float64(z)), x, z
}

func TestThereAreCavesUnderTheLand(t *testing.T) {
	w, x0, z0 := loadedAt(t, 7, Prairie)
	s := uint32(7)

	under, hollow := 0, 0
	span := (LoadRadius - 1) * ChunkSize
	for z := z0 - span; z < z0+span; z++ {
		for x := x0 - span; x < x0+span; x++ {
			top := landAt(x, z, s).top
			for y := 5; y < top-4; y++ {
				under++
				if w.At(x, y, z) == Air {
					hollow++
				}
			}
		}
	}
	if share := float64(hollow) / float64(under); share < 0.01 || share > 0.2 {
		t.Errorf("%.1f%% of the ground under the prairie is cave, wanted between 1 and 20", share*100)
	}
}

func TestCavesLeaveTheSeaItsBed(t *testing.T) {
	w, x0, z0 := loadedAt(t, 7, Sea)

	span := (LoadRadius - 1) * ChunkSize
	for z := z0 - span; z < z0+span; z++ {
		for x := x0 - span; x < x0+span; x++ {
			for y := 1; y <= SeaLevel; y++ {
				if b := w.At(x, y, z); b == Water || b == Ice {
					// Water lies on water or on ground, and has water or ground beside it.
					for _, d := range [][3]int{{0, -1, 0}, {1, 0, 0}, {-1, 0, 0}, {0, 0, 1}, {0, 0, -1}} {
						if w.At(x+d[0], y+d[1], z+d[2]) == Air {
							t.Fatalf("the water at %d,%d,%d has air beside or under it", x, y, z)
						}
					}
				}
			}
		}
	}
}

func TestNothingGrowsOverAHole(t *testing.T) {
	trunks := map[Block]bool{Log: true, PaleLog: true, PineLog: true, StumpPalm: true, FernTrunk: true, RedwoodLog: true}

	for _, country := range []Biome{Prairie, Jungle, Highland, Tundra} {
		w, x0, z0 := loadedAt(t, 7, country)

		span := (LoadRadius - 1) * ChunkSize
		for z := z0 - span; z < z0+span; z++ {
			for x := x0 - span; x < x0+span; x++ {
				for y := 1; y < Height; y++ {
					b, below := w.At(x, y, z), w.At(x, y-1, z)
					// A horsetail may stand on a horsetail.
					if (b.Info().Plant && b != below || (trunks[b] && !trunks[below])) && !below.Info().Solid {
						t.Fatalf("country %d: a %s at %d,%d,%d stands on %s", country, b.Info().Name, x, y, z, below.Info().Name)
					}
				}
			}
		}
	}
}

func TestEveryOreIsInTheStoneAtItsOwnDepth(t *testing.T) {
	w, x0, z0 := loadedAt(t, 7, Crags) // high ground: the most stone
	s := uint32(7)

	count := map[Block]int{}
	stone := 0
	span := (LoadRadius - 1) * ChunkSize
	for z := z0 - span; z < z0+span; z++ {
		for x := x0 - span; x < x0+span; x++ {
			top := landAt(x, z, s).top
			for y := 1; y < Height; y++ {
				switch b := w.At(x, y, z); b {
				case Stone:
					stone++
				case IronOre:
					t.Fatalf("iron ore in the ground at %d,%d,%d: iron only falls from the sky", x, y, z)
				case CoalOre, FlintSeam, CrystalOre:
					count[b]++
					if y > top {
						t.Fatalf("%s above the ground at %d,%d,%d", b.Info().Name, x, y, z)
					}
					if b == CrystalOre && y >= 20 {
						t.Fatalf("crystal ore as high as layer %d", y)
					}
				}
			}
		}
	}
	ores := 0
	for _, b := range []Block{CoalOre, FlintSeam, CrystalOre} {
		if count[b] == 0 {
			t.Errorf("there is no %s", b.Info().Name)
		}
		ores += count[b]
	}
	if share := float64(ores) / float64(ores+stone); share < 0.004 || share > 0.05 {
		t.Errorf("%.2f%% of the stone is ore, wanted between 0.4 and 5", share*100)
	}
	if count[CoalOre] < count[CrystalOre] {
		t.Errorf("%d coal and %d crystal: coal should be the common one", count[CoalOre], count[CrystalOre])
	}
}

func TestHeightNearIsTheHighestGroundInReach(t *testing.T) {
	w := New()
	w.Set(3, 20, 3, Stone)
	w.Set(80, 90, 3, Stone) // a tower five chunks along x

	if got := w.HeightNear(3, 3, 30); got != 21 {
		t.Errorf("within 30 blocks the world is %d high, wanted 21", got)
	}
	if got := w.HeightNear(3, 3, 64); got != 21 {
		t.Errorf("within 64 blocks the world is %d high, wanted 21: the tower is 77 away", got)
	}
	if got := w.HeightNear(3, 3, 80); got != 91 {
		t.Errorf("within 80 blocks the world is %d high, wanted 91", got)
	}
	if got := w.Height(); got != 91 {
		t.Errorf("the world is %d high, wanted 91", got)
	}

	w.Set(3, 40, 3, Brick)
	if got := w.HeightNear(3, 3, 10); got != 41 {
		t.Errorf("after building up, the world near is %d high, wanted 41", got)
	}
}

// A chunk looks treeReach blocks outside itself for trees whose leaves reach into it. A
// tree that reached farther would be cut off at the edge of a chunk.
func TestNoTreeReachesFartherThanChunksLookForIt(t *testing.T) {
	const x, y, z = 8, 60, 8 // the foot of the tree, in the middle of chunk 0,0
	for name, stand := range map[string]func(terrain){
		"broad tree":    func(t terrain) { t.broadTree(x, y, z, 10) },
		"pine":          func(t terrain) { t.pine(x, y, z, 10) },
		"umbrella pine": func(t terrain) { t.umbrellaPine(x, y, z, 15) },
		"fan-leaf tree": func(t terrain) { t.fanTree(x, y, z, 8) },
		"redwood":       func(t terrain) { t.redwood(x, y, z, 33) },
		"tree fern":     func(t terrain) { t.treeFern(x, y, z, 7) },
		"stump palm":    func(t terrain) { t.stumpPalm(x, y, z, 3) },
	} {
		blocks, reach := 0, 0
		for cz := -1; cz <= 1; cz++ {
			for cx := -1; cx <= 1; cx++ {
				chunk := make([]Block, ChunkSize*ChunkSize*Height)
				stand(terrain{seed: 1, cx: cx, cz: cz, blocks: chunk})
				for i, b := range chunk {
					if b == Air || i/(ChunkSize*ChunkSize) < y {
						continue // air, or the foot of a trunk let down into the ground
					}
					blocks++
					bx, bz := cx*ChunkSize+i%ChunkSize, cz*ChunkSize+i/ChunkSize%ChunkSize
					reach = max(reach, abs(bx-x), abs(bz-z))
				}
			}
		}
		if blocks == 0 || reach > treeReach {
			t.Errorf("a %s of %d blocks reaches %d from its foot, and chunks look %d", name, blocks, reach, treeReach)
		}
	}
}

// A new player should see the world, and not the trunk or the bone it stands beside.
func TestSpawnStandsClearOfWhatIsRoundIt(t *testing.T) {
	for seed := int64(1); seed <= 12; seed++ {
		w := Generate(seed)
		p := w.Spawn()
		x, y, z := int(math.Floor(p.X)), int(p.Y), int(math.Floor(p.Z))
		for dz := -1; dz <= 1; dz++ {
			for dx := -1; dx <= 1; dx++ {
				for dy := range 2 {
					if b := w.At(x+dx, y+dy, z+dz); b.Info().Solid {
						t.Errorf("seed %d: %s stands at %d,%d,%d, beside the spawn at %d,%d,%d", seed, b.Info().Name, x+dx, y+dy, z+dz, x, y, z)
					}
				}
			}
		}
	}
}

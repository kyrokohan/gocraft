package world

import (
	"slices"
	"testing"
)

// firstStruck returns a tree of a seed's world that lightning struck, the nearest its
// origin.
func firstStruck(t *testing.T, seed uint32) struck {
	t.Helper()
	for ring := range 60 {
		for gz := -ring; gz <= ring; gz++ {
			for gx := -ring; gx <= ring; gx++ {
				if s, ok := struckIn(gx, gz, seed); ok && max(abs(gx), abs(gz)) == ring {
					return s
				}
			}
		}
	}
	t.Fatalf("seed %d has no struck tree within 60 squares", seed)
	return struck{}
}

func TestAStruckTreeStandsOnDryLandInASquareWithNoSkeleton(t *testing.T) {
	found := 0
	for gz := -40; gz <= 40; gz++ {
		for gx := -40; gx <= 40; gx++ {
			s, ok := struckIn(gx, gz, 7)
			if again, okAgain := struckIn(gx, gz, 7); again != s || okAgain != ok {
				t.Fatalf("square %d,%d: the seed gave two different answers", gx, gz)
			}
			if !ok {
				continue
			}
			found++
			if _, bones := landmarkIn(gx, gz, 7); bones {
				t.Errorf("square %d,%d holds a skeleton and a struck tree", gx, gz)
			}
			if s.x-markReach < gx*markSquare || s.x+markReach >= (gx+1)*markSquare ||
				s.z-markReach < gz*markSquare || s.z+markReach >= (gz+1)*markSquare {
				t.Errorf("the struck tree at %d,%d is not well inside square %d,%d", s.x, s.z, gx, gz)
			}
			if l := landAt(s.x, s.z, 7); l.biome == Sea || l.biome == Shore || l.top != s.y {
				t.Errorf("the struck tree at %d,%d stands in country %d at %d, on ground at %d", s.x, s.z, l.biome, s.y, l.top)
			}
		}
	}
	if found < 1000 {
		t.Errorf("%d of 6,561 squares hold a struck tree, wanted a good part of them", found)
	}
}

// A struck tree is built from the seed alone, by every chunk it lies in.
func TestAStruckTreeIsTheSameWhicheverChunkIsMadeFirstAndBurns(t *testing.T) {
	for _, seed := range []int64{7, 11, 2024} {
		s := firstStruck(t, uint32(seed))
		gen := Terrain(seed)

		var chunks [][2]int
		for cz := floorDiv(s.z-struckReach, ChunkSize); cz <= floorDiv(s.z+struckReach, ChunkSize); cz++ {
			for cx := floorDiv(s.x-struckReach, ChunkSize); cx <= floorDiv(s.x+struckReach, ChunkSize); cx++ {
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
				t.Errorf("seed %d: chunk %v came out different when its neighbors were made first", seed, c)
			}
		}

		w := newWorld(seed, gen, "", float64(s.x), float64(s.z))
		for y := s.y + 1; y <= s.y+s.tall; y++ {
			if b := w.At(s.x, y, s.z); b != CharredLog {
				t.Errorf("seed %d: the trunk at %d,%d,%d is %v, wanted a charred log", seed, s.x, y, s.z, b)
			}
		}
		fires := w.Struck(float64(s.x), float64(s.z), 8)
		if len(fires) != 2 {
			t.Fatalf("seed %d: the struck tree at %d,%d has %d fires, wanted one on its top and one at its foot", seed, s.x, s.z, len(fires))
		}
		if top := (Cell{s.x, s.y + s.tall + 1, s.z}); fires[0] != top {
			t.Errorf("seed %d: the first fire is at %v, wanted the top of the trunk at %v", seed, fires[0], top)
		}
		for _, f := range fires {
			if w.At(f.X, f.Y, f.Z) != Fire || w.At(f.X, f.Y-1, f.Z) != CharredLog {
				t.Errorf("seed %d: at %v there is %v on %v, wanted fire on a charred log",
					seed, f, w.At(f.X, f.Y, f.Z), w.At(f.X, f.Y-1, f.Z))
			}
			if got := w.LightAt(f.X, f.Y, f.Z).Block(); got != MaxLight {
				t.Errorf("seed %d: the fire at %v gives light %d, wanted %d", seed, f, got, MaxLight)
			}
		}
		// The fire at the foot is low enough to light a torch at from the ground.
		if foot := fires[1]; foot.Y-landAt(foot.X, foot.Z, uint32(seed)).top != 2 {
			t.Errorf("seed %d: the fire at the foot is %d over its ground, wanted 2", seed, foot.Y-landAt(foot.X, foot.Z, uint32(seed)).top)
		}
	}
}

func TestNoStruckTreeIsFoundFarFromOne(t *testing.T) {
	s := firstStruck(t, 7)
	w := newWorld(7, Terrain(7), "", float64(s.x), float64(s.z))
	if got := w.Struck(float64(s.x)+40, float64(s.z), 8); len(got) != 0 {
		t.Errorf("40 blocks from a struck tree, %d fires are within 8 blocks", len(got))
	}
	if got := New().Struck(0, 0, 500); len(got) != 0 {
		t.Errorf("a world that is not made from a seed has %d struck trees", len(got))
	}
}

func TestWhatTheThingsOfFireLeave(t *testing.T) {
	for b, want := range map[Block]Stack{
		Fire:       {},
		CharredLog: {Item: Coal, Count: 1},
		HearthLit:  {Item: Hearth.Item(), Count: 1},
		Hearth:     {Item: Hearth.Item(), Count: 1},
		ClayPot:    {Item: ClayPot.Item(), Count: 1},
		PalmCrown:  {Item: PalmSeeds, Count: 2},
	} {
		if item, n := b.Drop(); (Stack{Item: item, Count: n}) != want {
			t.Errorf("%v leaves %d of %v, wanted %d of %v", b, n, item, want.Count, want.Item)
		}
	}
}

func TestFireAndALitHearthGlowAndAColdHearthDoesNot(t *testing.T) {
	for b, want := range map[Block]uint8{Fire: MaxLight, HearthLit: 14, Torch: 14, Hearth: 0, CharredLog: 0} {
		if got := b.Info().Glow; got != want {
			t.Errorf("%v glows at %d, wanted %d", b, got, want)
		}
	}
	if info := Fire.Info(); info.Solid || !info.Stands() {
		t.Error("fire stops the player or stands on nothing: wanted it walked into, on what it burns")
	}
}

func TestPineAndRedwoodLogsHoldResinNowAndThenAndAlwaysTheSameOnes(t *testing.T) {
	for _, log := range []Block{PineLog, RedwoodLog} {
		with := 0
		for i := range 2000 {
			x, y, z := i%40-20, 30+i/40%10, i/400-2
			item, n := log.AlsoAt(x, y, z)
			if again, m := log.AlsoAt(x, y, z); again != item || m != n {
				t.Fatalf("%v at %d,%d,%d gave two answers", log, x, y, z)
			}
			if n > 0 {
				if with++; item != Resin || n != 1 {
					t.Fatalf("%v holds %d of %v, wanted one piece of resin", log, n, item)
				}
			}
			if main, k := log.DropAt(x, y, z); main != log.Item() || k != 1 {
				t.Fatalf("%v leaves %d of %v, wanted itself whether it holds resin or not", log, k, main)
			}
		}
		if with < 250 || with > 550 {
			t.Errorf("%d of 2,000 %vs hold resin, wanted about one in five", with, log)
		}
	}
	for _, b := range []Block{Log, PaleLog, FernTrunk, Stone, Leaves} {
		for i := range 300 {
			if _, n := b.AlsoAt(i, 40, -i); n > 0 {
				t.Fatalf("%v holds something besides what it leaves", b)
			}
		}
	}
}

func TestTheThingsOfFire(t *testing.T) {
	if Meat.Food() >= CookedMeat.Food() || CookedMeat.Food() != 12 {
		t.Errorf("raw meat gives %d and cooked %d, wanted 12 cooked and less raw", Meat.Food(), CookedMeat.Food())
	}
	if PalmSeeds.Poison() == 0 || RoastSeeds.Poison() != 0 || RoastSeeds.Food() == 0 {
		t.Errorf("raw seeds take %v and roast ones take %v and give %d: wanted poison raw, and food roast",
			PalmSeeds.Poison(), RoastSeeds.Poison(), RoastSeeds.Food())
	}
	for _, i := range []Item{Meat, CookedMeat, Berries, Fruit, RoastSeeds} {
		if i.Poison() != 0 {
			t.Errorf("%v is poison", i)
		}
	}
	if HardSpear.Damage() <= Spear.Damage() || HardSpear.Uses() <= Spear.Uses() || HardSpear.Reach() != Spear.Reach() {
		t.Errorf("a hardened spear hits for %v, lasts %d and reaches %v; a spear %v, %d and %v: wanted harder, longer and as far",
			HardSpear.Damage(), HardSpear.Uses(), HardSpear.Reach(), Spear.Damage(), Spear.Uses(), Spear.Reach())
	}
	if FireDrill.Uses() == 0 || FireDrill.StackSize() != 1 {
		t.Errorf("a fire drill lasts %d uses, %d to a slot: wanted it to wear, one to a slot", FireDrill.Uses(), FireDrill.StackSize())
	}
	for _, i := range []Item{UnlitTorch, Resin, SoftPot, PalmSeeds, CookedMeat} {
		if !i.Known() || i.Name() == "" || i.StackSize() != StackSize {
			t.Errorf("item %d is %q, known %v, %d to a slot", i, i.Name(), i.Known(), i.StackSize())
		}
	}
	// The numbers of what a save may hold do not move.
	if Spear != 292 || CookedMeat != Spear+1 || Fire != Redrock+1 || Hearth != KnappingStone+1 || ClayPot != Hearth+1 {
		t.Errorf("the numbers moved: spear %d, cooked meat %d, fire %d, hearth %d, clay pot %d", Spear, CookedMeat, Fire, Hearth, ClayPot)
	}
}

// A light that is carried lights the world as a torch in its cell would, and is no block.
func TestALightThatIsCarriedLightsItsCellAndGoesWithIt(t *testing.T) {
	hall := func() *World {
		w := small(3)
		for x := 0; x <= 20; x++ {
			for y := 9; y <= 11; y++ {
				for z := 9; z <= 11; z++ {
					if x == 0 || x == 20 || y != 10 || z != 10 {
						w.Set(x, y, z, Stone)
					}
				}
			}
		}
		return w
	}
	w := hall()
	dark := plainLight(w)

	w.Carry(3, 10, 10, 14)
	for x, want := range map[int]int{3: 14, 4: 13, 10: 7, 16: 1, 17: 0} {
		if got := w.LightAt(x, 10, 10).Block(); got != want {
			t.Errorf("%d along the hall the carried light is %d, wanted %d", x, got, want)
		}
	}
	if w.At(3, 10, 10) != Air {
		t.Errorf("the cell the light is carried in holds %v, wanted air", w.At(3, 10, 10))
	}

	// Carried along the hall, the light is what a torch standing there would give.
	for _, x := range []int{4, 5, 9, 15} {
		w.Carry(x, 10, 10, 14)
		with := hall()
		with.Set(x, 10, 10, Torch)
		sameLight(t, w, plainLight(with), "carried to another cell")
	}

	// A block put down or taken away beside it leaves it burning.
	w.Set(14, 10, 10, Stone)
	w.Set(14, 10, 10, Air)
	if got := w.LightAt(15, 10, 10).Block(); got != 14 {
		t.Errorf("after a block changed beside it the carried light is %d, wanted 14", got)
	}

	w.Carry(15, 10, 10, 0)
	sameLight(t, w, dark, "the light put out")
}

func TestALightCarriedInTheOpenAirAboveEveryBlockStillLights(t *testing.T) {
	w := small(3) // all air: every cell is above the highest block
	w.Carry(5, 40, 5, 14)
	for dx, want := range map[int]int{0: 14, 1: 13, 5: 9, 13: 1, 14: 0} {
		if got := w.LightAt(5+dx, 40, 5).Block(); got != want {
			t.Errorf("%d from a light carried in the open air it is %d, wanted %d", dx, got, want)
		}
	}
	if got := w.LightAt(5, 40, 5).Sky(); got != MaxLight {
		t.Errorf("the sky's light where a light is carried is %d, wanted %d", got, MaxLight)
	}
	w.Carry(5, Height+3, 5, 14) // out of the world: it lights nothing
	sameLight(t, w, plainLight(w), "the light carried out of the world")
}

// A chunk that comes into the window after the light is given its light too.
func TestACarriedLightReachesAChunkThatComesInLater(t *testing.T) {
	w := emptyWorld(0, nil, "")
	w.load(key{1, 1})
	w.Carry(ChunkSize, 60, ChunkSize, 14)
	w.settle()
	for _, k := range []key{{1, 0}, {0, 1}, {0, 0}} {
		w.load(k)
	}
	if got := w.LightAt(ChunkSize-1, 60, ChunkSize-1).Block(); got != 12 {
		t.Errorf("in the corner of the last chunk the carried light is %d, wanted 12", got)
	}
	with := emptyWorld(0, nil, "")
	for _, k := range []key{{1, 1}, {1, 0}, {0, 1}, {0, 0}} {
		with.load(k)
	}
	with.Set(ChunkSize, 60, ChunkSize, Torch)
	sameLight(t, w, plainLight(with), "the last chunk in")
}

// What it costs to carry a light a cell on over the ground and light the world again.
func BenchmarkCarry(b *testing.B) {
	w := Generate(1)
	i := 0
	for b.Loop() {
		x, z := i%64, i/64%8
		w.Carry(x, w.Ground(x, z)+2, z, 14)
		w.settle()
		i++
	}
}

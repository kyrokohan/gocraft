package world

import (
	"math"
	"slices"
	"testing"
)

// firstFire returns the fire of the camp within a walk of where a seed's world begins.
func firstFire(t *testing.T, seed int64) Cell {
	t.Helper()
	for _, f := range Generate(seed).Camps(0, 0, campFar+1) {
		if far := math.Hypot(float64(f.X), float64(f.Z)); far >= campNear-1 && far <= campFar+1 {
			return f
		}
	}
	t.Fatalf("seed %d has no camp %d to %d blocks from where it begins", seed, campNear, campFar)
	return Cell{}
}

// around returns the blocks of the chunks that a camp reaches into, by chunk.
func around(seed int64, fire Cell) map[[2]int][]Block {
	gen := Terrain(seed)
	chunks := map[[2]int][]Block{}
	for cz := floorDiv(fire.Z-campReach, ChunkSize); cz <= floorDiv(fire.Z+campReach, ChunkSize); cz++ {
		for cx := floorDiv(fire.X-campReach, ChunkSize); cx <= floorDiv(fire.X+campReach, ChunkSize); cx++ {
			chunks[[2]int{cx, cz}] = made(gen, cx, cz)
		}
	}
	return chunks
}

// blockIn returns the block at a place from chunks made by around.
func blockIn(chunks map[[2]int][]Block, x, y, z int) Block {
	cx, cz := floorDiv(x, ChunkSize), floorDiv(z, ChunkSize)
	return chunks[[2]int{cx, cz}][(y*ChunkSize+z-cz*ChunkSize)*ChunkSize+x-cx*ChunkSize]
}

func TestEveryWorldHasACampWithinAWalkOfWhereItBegins(t *testing.T) {
	for seed := int64(1); seed <= 10; seed++ {
		fire := firstFire(t, seed)
		if b := BiomeAt(seed, fire.X, fire.Z); !grassLand(b) {
			t.Errorf("seed %d: the first camp, at %+v, is in country %d", seed, fire, b)
		}
		if again := firstFire(t, seed); again != fire {
			t.Errorf("seed %d: the first camp is at %+v and then at %+v", seed, fire, again)
		}

		chunks := around(seed, fire)
		if b := blockIn(chunks, fire.X, fire.Y, fire.Z); b != Fire {
			t.Errorf("seed %d: the camp's fire at %+v is %v", seed, fire, b)
		}
		tents := 0
		for _, blocks := range chunks {
			for _, b := range blocks {
				if b == Tent {
					tents++
				}
			}
		}
		if tents < 2*hutBlocks {
			t.Errorf("seed %d: the camp at %+v has %d blocks of tent, wanted two huts at least", seed, fire, tents)
		}
	}
}

func TestACampLiesInsideItsSquareInGrassLandAndNotNearTheFirst(t *testing.T) {
	const seed = 7
	first, _ := firstCamp(seed)
	found := 0
	for gz := -20; gz <= 20; gz++ {
		for gx := -20; gx <= 20; gx++ {
			x0, z0 := gx*campSquare, gz*campSquare
			var in []camp
			campsNear(seed, first, true, x0, z0, x0+campSquare-1, z0+campSquare-1, -campReach, func(c camp) {
				if c != first {
					in = append(in, c)
				}
			})
			if len(in) > 1 {
				t.Fatalf("square %d,%d holds %d camps", gx, gz, len(in))
			}
			for _, c := range in {
				found++
				if c.x-campReach < x0 || c.x+campReach >= x0+campSquare || c.z-campReach < z0 || c.z+campReach >= z0+campSquare {
					t.Errorf("the camp at %d,%d reaches out of square %d,%d", c.x, c.z, gx, gz)
				}
				if b := landAt(c.x, c.z, seed).biome; !grassLand(b) {
					t.Errorf("the camp at %d,%d is in country %d", c.x, c.z, b)
				}
				if abs(c.x-first.x) < campApart && abs(c.z-first.z) < campApart {
					t.Errorf("the camp at %d,%d is %d,%d from the first", c.x, c.z, c.x-first.x, c.z-first.z)
				}
			}
		}
	}
	if found < 100 || found > 1000 {
		t.Errorf("%d of 1,681 squares hold a camp, wanted some and not most", found)
	}
}

func TestNothingGrowsOrLiesInACampButItsFireAndHuts(t *testing.T) {
	for seed := int64(1); seed <= 4; seed++ {
		fire := firstFire(t, seed)
		chunks := around(seed, fire)
		s := uint32(seed) ^ uint32(seed>>32)
		for z := fire.Z - campReach; z <= fire.Z+campReach; z++ {
			for x := fire.X - campReach; x <= fire.X+campReach; x++ {
				for y := landAt(x, z, s).top + 1; y < Height; y++ {
					if b := blockIn(chunks, x, y, z); !slices.Contains([]Block{Air, Fire, Tent, Dirt, Turf, Cobble, Water}, b) {
						t.Fatalf("seed %d: a %s stands in the camp at %d, %d, %d", seed, b.Info().Name, x, y, z)
					}
				}
			}
		}
	}
}

func TestATentBreaksIntoHide(t *testing.T) {
	if item, n := Tent.Drop(); item != Hide || n != 1 {
		t.Errorf("a tent drops %d of %v, wanted a hide", n, item)
	}
	if !Tent.Info().Solid || Tent.Info().Fixed {
		t.Error("a tent is not a solid block that can be broken")
	}
}

func TestAWorldNotMadeFromASeedHasNoCamps(t *testing.T) {
	if fires := New().Camps(0, 0, 1000); fires != nil {
		t.Errorf("an empty world has camps: %v", fires)
	}
}

func TestTheGroundRoundACampsFireIsLevel(t *testing.T) {
	for seed := int64(1); seed <= 10; seed++ {
		fire := firstFire(t, seed)
		chunks := around(seed, fire)
		for z := fire.Z - yardReach; z <= fire.Z+yardReach; z++ {
			for x := fire.X - yardReach; x <= fire.X+yardReach; x++ {
				under, at := blockIn(chunks, x, fire.Y-1, z), blockIn(chunks, x, fire.Y, z)
				if !under.Info().Solid || at != Air && at != Fire {
					t.Fatalf("seed %d: by the fire, at %d, %d the ground is %s under %s", seed, x, z, under.Info().Name, at.Info().Name)
				}
			}
		}
	}
}

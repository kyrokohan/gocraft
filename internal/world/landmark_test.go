package world

import (
	"slices"
	"testing"
)

// firstLandmark returns a landmark of a seed's world, the nearest its origin.
func firstLandmark(t *testing.T, seed uint32) landmark {
	t.Helper()
	for ring := range 60 {
		for gz := -ring; gz <= ring; gz++ {
			for gx := -ring; gx <= ring; gx++ {
				if m, ok := landmarkIn(gx, gz, seed); ok && max(abs(gx), abs(gz)) == ring {
					return m
				}
			}
		}
	}
	t.Fatalf("seed %d has no landmark within 60 squares", seed)
	return landmark{}
}

func TestALandmarkLiesWellInsideItsSquareAndInItsOwnCountry(t *testing.T) {
	found := 0
	for gz := -40; gz <= 40; gz++ {
		for gx := -40; gx <= 40; gx++ {
			m, ok := landmarkIn(gx, gz, 7)
			if again, okAgain := landmarkIn(gx, gz, 7); again != m || okAgain != ok {
				t.Fatalf("square %d,%d: the seed gave two different answers", gx, gz)
			}
			if !ok {
				continue
			}
			found++
			if m.x-markReach < gx*markSquare || m.x+markReach >= (gx+1)*markSquare ||
				m.z-markReach < gz*markSquare || m.z+markReach >= (gz+1)*markSquare {
				t.Errorf("the landmark at %d,%d reaches out of square %d,%d", m.x, m.z, gx, gz)
			}
			if b := landAt(m.x, m.z, 7).biome; b != Badlands && b != Prairie {
				t.Errorf("the landmark at %d,%d lies in country %d", m.x, m.z, b)
			}
		}
	}
	if found < 100 || found > 3000 {
		t.Errorf("%d of 6,561 squares hold a landmark, wanted some and not most", found)
	}
}

// A great skeleton is longer than a chunk is wide, so it is always built by several
// chunks, each from the seed alone.
func TestAGreatSkeletonIsWholeAcrossChunkEdges(t *testing.T) {
	const seed = 7
	m := firstLandmark(t, seed)
	gen := Terrain(seed)

	var chunks [][2]int
	for cz := floorDiv(m.z-markReach, ChunkSize); cz <= floorDiv(m.z+markReach, ChunkSize); cz++ {
		for cx := floorDiv(m.x-markReach, ChunkSize); cx <= floorDiv(m.x+markReach, ChunkSize); cx++ {
			chunks = append(chunks, [2]int{cx, cz})
		}
	}

	bones, holders := 0, 0
	first := map[[2]int][]Block{}
	for _, c := range chunks {
		first[c] = made(gen, c[0], c[1])
		n := 0
		for i, b := range first[c] {
			if b != Bone {
				continue
			}
			n++
			x, z := c[0]*ChunkSize+i%ChunkSize, c[1]*ChunkSize+i/ChunkSize%ChunkSize
			if abs(x-m.x) > markReach || abs(z-m.z) > markReach {
				t.Fatalf("a bone at %d,%d is farther than %d from the skeleton's middle at %d,%d", x, z, markReach, m.x, m.z)
			}
		}
		bones += n
		if n > 0 {
			holders++
		}
	}
	if bones < 60 || holders < 2 {
		t.Errorf("the skeleton at %d,%d has %d bones in %d chunks, wanted 60 or more in 2 or more", m.x, m.z, bones, holders)
	}

	// Made again in the other order, every chunk is the same.
	slices.Reverse(chunks)
	for _, c := range chunks {
		if !slices.Equal(made(gen, c[0], c[1]), first[c]) {
			t.Errorf("chunk %v came out different when its neighbors were made first", c)
		}
	}
}

// The ribs stand as arches with room to walk under the spine.
func TestTheRibsOfASkeletonAreAHall(t *testing.T) {
	const seed = 7
	m := firstLandmark(t, seed)
	w := newWorld(seed, Terrain(seed), "", float64(m.x), float64(m.z))

	// Somewhere along it there is a rib: bone on both sides and overhead, air between.
	halls := 0
	for d := -markReach; d <= markReach; d++ {
		x, z := m.x+d, m.z
		sideX, sideZ := 0, 3
		if !m.along {
			x, z, sideX, sideZ = m.x, m.z+d, 3, 0
		}
		y := landAt(x, z, seed).top + 1 // the ground, not the spine over it
		if w.At(x, y, z) == Air && w.At(x, y+1, z) == Air &&
			w.At(x+sideX, y, z+sideZ) == Bone && w.At(x-sideX, y, z-sideZ) == Bone &&
			slices.Contains([]Block{w.At(x, y+3, z), w.At(x, y+4, z), w.At(x, y+5, z), w.At(x, y+6, z)}, Bone) {
			halls++
		}
	}
	if halls < 3 {
		t.Errorf("the skeleton at %d,%d has %d ribs to walk under, wanted 3 or more", m.x, m.z, halls)
	}
}

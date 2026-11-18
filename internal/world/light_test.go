package world

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

// plainLight works the light of every loaded cell out again the slow, plain way, and
// returns it laid out as the window holds it: first the light straight down each column,
// and then each cell given the most that any neighbor can give it, again and again until
// nothing changes. A chunk that is not loaded lets no light through.
func plainLight(w *World) []Light {
	out := make([]Light, len(w.light))
	var columns [][2]int
	top := 0
	for at, ok := range w.loaded {
		if !ok {
			continue
		}
		k := w.owner[at]
		top = max(top, w.tops[at])
		for z := range ChunkSize {
			for x := range ChunkSize {
				columns = append(columns, [2]int{k.x*ChunkSize + x, k.z*ChunkSize + z})
			}
		}
	}

	for _, c := range columns {
		level := MaxLight
		for y := Height - 1; y >= 0; y-- {
			info := w.At(c[0], y, c[1]).Info()
			level = max(level-int(info.Shade), 0)
			out[index(c[0], y, c[1])] = Light(level<<4 | int(info.Glow))
		}
	}

	// Light from a glowing block at the top can climb lightReach above it; past that
	// every cell is lit by the open sky alone.
	steps := [6][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
	for changed := true; changed; {
		changed = false
		for _, c := range columns {
			for y := range min(top+lightReach+1, Height) {
				info := w.At(c[0], y, c[1]).Info()
				if info.Shade >= MaxLight {
					continue
				}
				cost := max(int(info.Shade), 1)
				i := index(c[0], y, c[1])
				sky, block := out[i].Sky(), out[i].Block()
				for _, s := range steps {
					x, up, z := c[0]+s[0], y+s[1], c[1]+s[2]
					if up < 0 || up >= Height || !w.Loaded(x, z) {
						continue
					}
					n := out[index(x, up, z)]
					sky, block = max(sky, n.Sky()-cost), max(block, n.Block()-cost)
				}
				if l := Light(sky<<4 | block); l != out[i] {
					out[i], changed = l, true
				}
			}
		}
	}
	return out
}

// sameLight fails the test where the light of a loaded cell is not what want says.
func sameLight(t *testing.T, w *World, want []Light, when string) {
	t.Helper()
	w.settle()
	wrong := 0
	for at, ok := range w.loaded {
		if !ok {
			continue
		}
		k := w.owner[at]
		for y := range Height {
			for z := k.z * ChunkSize; z < (k.z+1)*ChunkSize; z++ {
				for x := k.x * ChunkSize; x < (k.x+1)*ChunkSize; x++ {
					i := index(x, y, z)
					if got := w.light[i]; got != want[i] {
						if wrong++; wrong <= 5 {
							t.Errorf("%s: light at %d,%d,%d is sky %d block %d, wanted sky %d block %d (%v there)",
								when, x, y, z, got.Sky(), got.Block(), want[i].Sky(), want[i].Block(), w.At(x, y, z))
						}
					}
				}
			}
		}
	}
	if wrong > 5 {
		t.Errorf("%s: and %d cells more", when, wrong-5)
	}
}

// cellHash mixes the place of a cell into a number from 0 to 99.
func cellHash(x, y, z int) int {
	h := uint32(x)*0x9E3779B1 ^ uint32(y)*0x85EBCA77 ^ uint32(z)*0xC2B2AE3D
	h ^= h >> 15
	h *= 0x2C1B3C6D
	h ^= h >> 13
	return int(h % 100)
}

// jumble makes chunks of everything that light meets, in heaps: holes in the ground,
// leaves and water to dim it, glass, glowing blocks, and roofs to shut the sky out.
func jumble(cx, cz int, blocks []Block) {
	for z := range ChunkSize {
		for x := range ChunkSize {
			wx, wz := cx*ChunkSize+x, cz*ChunkSize+z
			ground := 6 + cellHash(wx, 0, wz)%10
			for y := range 24 {
				r := cellHash(wx, y, wz)
				var b Block
				switch {
				case y < ground:
					b = pick(r, []Block{Stone, Air, Leaves, Water, Glass, CrystalOre, Torch}, []int{55, 15, 6, 6, 3, 3, 2})
				case y == 20 && (cx+cz)&1 == 0 && r < 85:
					b = Stone // a roof over most of every other chunk
				default:
					b = pick(r, []Block{Stone, Leaves, Torch, Water}, []int{6, 3, 1, 1})
				}
				blocks[(y*ChunkSize+z)*ChunkSize+x] = b
			}
		}
	}
}

// pick returns the block that a number from 0 to 99 falls on, where each block takes as
// many numbers as its weight, and air takes the rest.
func pick(r int, blocks []Block, weights []int) Block {
	for i, w := range weights {
		if r < w {
			return blocks[i]
		}
		r -= w
	}
	return Air
}

// chunksAround returns the chunks of a square n chunks across, in an order shuffled by rng.
func chunksAround(rng *rand.Rand, n int) []key {
	var keys []key
	for z := range n {
		for x := range n {
			keys = append(keys, key{x - n/2, z - n/2})
		}
	}
	rng.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	return keys
}

func TestTheOpenSkyIsFullyLitAndAShutRoomIsDark(t *testing.T) {
	w := New()
	if l := w.LightAt(3, 40, 3); l.Sky() != MaxLight || l.Block() != 0 {
		t.Errorf("the open air is lit sky %d block %d, wanted sky %d and no other", l.Sky(), l.Block(), MaxLight)
	}
	room(w, 10, 20, 10, Stone)
	if l := w.LightAt(10, 20, 10); l != 0 {
		t.Errorf("inside a shut room the light is sky %d block %d, wanted dark", l.Sky(), l.Block())
	}
	if l := w.LightAt(3, Height+5, 3); l.Sky() != MaxLight {
		t.Errorf("above the world the sky is %d, wanted %d", l.Sky(), MaxLight)
	}
}

// room walls the cell at x, y, z in on every side with b, and leaves the cell itself empty.
func room(w *World, x, y, z int, b Block) {
	for dy := -1; dy <= 1; dy++ {
		for dz := -1; dz <= 1; dz++ {
			for dx := -1; dx <= 1; dx++ {
				if dx != 0 || dy != 0 || dz != 0 {
					w.Set(x+dx, y+dy, z+dz, b)
				}
			}
		}
	}
}

// small returns a world of air with only the n by n chunks round the origin loaded: few
// enough for plainLight.
func small(n int) *World {
	w := emptyWorld(0, nil, "")
	for _, k := range chunksAround(rand.New(rand.NewPCG(1, 1)), n) {
		w.load(k)
	}
	return w
}

func TestATorchLightsItsPlaceAndTakingItAwayLeavesItDark(t *testing.T) {
	w := small(3)
	// A long hall with a roof, shut at both ends.
	for x := 0; x <= 20; x++ {
		for y := 9; y <= 11; y++ {
			for z := 9; z <= 11; z++ {
				if x == 0 || x == 20 || y != 10 || z != 10 {
					w.Set(x, y, z, Stone)
				}
			}
		}
	}
	before := plainLight(w)
	w.Set(3, 10, 10, Torch)

	for x, want := range map[int]int{3: 14, 4: 13, 10: 7, 16: 1, 17: 0} {
		if got := w.LightAt(x, 10, 10).Block(); got != want {
			t.Errorf("%d along the hall the torch's light is %d, wanted %d", x, got, want)
		}
	}
	if got := w.LightAt(10, 10, 10).Sky(); got != 0 {
		t.Errorf("the sky's light in the hall is %d, wanted none", got)
	}

	w.Set(3, 10, 10, Air)
	sameLight(t, w, before, "the torch taken away")
}

func TestTheSkysLightFallsStraightAndLosesALevelAStepAside(t *testing.T) {
	w := New()
	// A roof over the ground with an open edge at x = 20, and a floor under it.
	for x := -30; x < 20; x++ {
		for z := -30; z <= 30; z++ {
			w.Set(x, 30, z, Stone)
			w.Set(x, 25, z, Stone)
		}
	}
	for x := 20; x < 40; x++ {
		for z := -30; z <= 30; z++ {
			w.Set(x, 25, z, Stone)
		}
	}
	for _, c := range []struct{ x, want int }{{25, 15}, {20, 15}, {19, 14}, {15, 10}, {6, 1}, {5, 0}} {
		if got := w.LightAt(c.x, 26, 0).Sky(); got != c.want {
			t.Errorf("at x=%d under the roof the sky's light is %d, wanted %d", c.x, got, c.want)
		}
	}
	// Far down under the floor, which the sky reaches only past its end at x = 40.
	for _, c := range []struct{ x, want int }{{40, 15}, {30, 5}, {25, 0}} {
		if got := w.LightAt(c.x, 5, 0).Sky(); got != c.want {
			t.Errorf("at x=%d far under the floor the sky's light is %d, wanted %d", c.x, got, c.want)
		}
	}
}

// The light of a torch high over low ground reaches a chunk that comes in after it, from
// the chunk at its corner, over the two beside both.
func TestTorchLightComesInOverLowGroundFromACorner(t *testing.T) {
	w := emptyWorld(0, nil, "")
	w.load(key{1, 1})
	w.Set(ChunkSize, 60, ChunkSize, Torch) // in the corner nearest chunk 0, 0
	for _, k := range []key{{1, 0}, {0, 1}, {0, 0}} {
		w.load(k)
	}
	if got := w.LightAt(ChunkSize-1, 60, ChunkSize-1).Block(); got != 12 {
		t.Errorf("in the corner of the last chunk the torch's light is %d, wanted 12", got)
	}
	sameLight(t, w, plainLight(w), "the last chunk in")
}

func TestLeavesAndWaterDimTheLightThatComesThrough(t *testing.T) {
	w := New()
	for y := 30; y < 33; y++ {
		w.Set(5, y, 5, Leaves)
		w.Set(9, y, 9, Water)
	}
	// Walls round both columns, so that light comes only from above.
	for y := 20; y < 33; y++ {
		for _, c := range [][2]int{{5, 5}, {9, 9}} {
			for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				w.Set(c[0]+d[0], y, c[1]+d[1], Stone)
			}
		}
	}
	if got := w.LightAt(5, 29, 5).Sky(); got != 12 {
		t.Errorf("under three leaves the sky's light is %d, wanted 12", got)
	}
	if got := w.LightAt(9, 31, 9).Sky(); got != 13 {
		t.Errorf("two deep in water the sky's light is %d, wanted 13", got)
	}
	if got := w.LightAt(9, 25, 9).Sky(); got != 12 {
		t.Errorf("in the air under the water it is %d, wanted 12", got)
	}
}

func TestCrystalOreGlowsFaintly(t *testing.T) {
	w := New()
	room(w, 10, 20, 10, Stone)
	w.Set(10, 21, 10, CrystalOre) // the roof of the room
	if got := w.LightAt(10, 20, 10).Block(); got != 7 {
		t.Errorf("under crystal ore in a shut room the light is %d, wanted 7", got)
	}
}

// However the chunks come in, the light is what the blocks make it.
func TestLightIsTheSameWhateverOrderTheChunksComeIn(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	for round := range 3 {
		w := emptyWorld(1, jumble, "")
		keys := chunksAround(rng, 3)
		for i, k := range keys {
			w.load(k)
			if i == len(keys)/2 || i == len(keys)-1 {
				sameLight(t, w, plainLight(w), fmt.Sprintf("round %d after %d chunks", round, i+1))
			}
		}
	}
}

func TestLightFollowsTheBlocksAsTheyChange(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 6))
	w := emptyWorld(1, jumble, "")
	for _, k := range chunksAround(rng, 3) {
		w.load(k)
	}
	kinds := []Block{Air, Air, Stone, Torch, Leaves, Water, Glass, CrystalOre}
	for round := range 6 {
		// Sometimes one change, and sometimes a heap of them before the light is asked for.
		for range 1 + round%3*7 {
			x, y, z := rng.IntN(48)-24, rng.IntN(26), rng.IntN(48)-24
			w.Set(x, y, z, kinds[rng.IntN(len(kinds))])
		}
		sameLight(t, w, plainLight(w), fmt.Sprintf("after the changes of round %d", round))
	}
}

// Light that crosses into a chunk from one beside it is taken away again when what let it
// through is closed.
func TestShuttingAWayAcrossAChunkEdgeTakesItsLightAway(t *testing.T) {
	w := small(3)
	// A shut hall across the edge between chunks 0 and 1, with a torch at one end.
	for x := 8; x <= 24; x++ {
		room(w, x, 10, 10, Stone)
	}
	for x := 9; x <= 23; x++ {
		w.Set(x, 10, 10, Air)
	}
	w.Set(9, 10, 10, Torch)
	if got := w.LightAt(17, 10, 10).Block(); got != 6 {
		t.Fatalf("across the edge the torch's light is %d, wanted 6", got)
	}
	w.Set(15, 10, 10, Stone)
	if got := w.LightAt(17, 10, 10).Block(); got != 0 {
		t.Errorf("behind the wall the torch's light is %d, wanted none", got)
	}
	sameLight(t, w, plainLight(w), "the hall shut")
}

// The light round a place is what it would be had the player never left, after going
// away and coming back. A short trip leaves some of the chunks round the place in the
// window, with light in them from those that left, and a long one takes them all out.
func TestTheLightIsTheSameAfterGoingAwayAndComingBack(t *testing.T) {
	x, z := 6, 6
	built := func(w *World) *World {
		y := w.Ground(x, z) + 1
		w.Set(x, y, z, Torch)
		w.Set(x+2, y+2, z, Brick)
		return w
	}
	fresh := built(Generate(7))

	for _, away := range []float64{150, 3_000} {
		w := built(Generate(7))
		w.Follow(away, 0)
		w.LoadAll()
		w.Follow(float64(x), float64(z))
		w.LoadAll()

		for yy := range Height {
			for zz := -2 * ChunkSize; zz < 3*ChunkSize; zz++ {
				for xx := -2 * ChunkSize; xx < 3*ChunkSize; xx++ {
					if got, want := w.LightAt(xx, yy, zz), fresh.LightAt(xx, yy, zz); got != want {
						t.Fatalf("%v away and back, the light at %d,%d,%d is %v, wanted %v", away, xx, yy, zz, got, want)
					}
				}
			}
		}
	}
}

// What a chunk costs to bring in, made from the seed and lit.
func BenchmarkLoadALitChunk(b *testing.B) {
	w := Generate(1)
	i := 0
	for b.Loop() {
		// A chunk at the edge of the window, which is not loaded, and one off the far
		// edge, so that the next can be loaded in the same place.
		k := key{w.cx + LoadRadius + 1, w.cz - LoadRadius + i%(2*LoadRadius+1)}
		w.load(k)
		w.evict(place(k))
		i++
	}
}

// What it costs to dig out a block of the ground and light what is under it.
func BenchmarkRelight(b *testing.B) {
	w := Generate(1)
	i := 0
	for b.Loop() {
		x, z := i%32, i/32%32
		y := w.Ground(x, z)
		was := w.At(x, y, z)
		w.Set(x, y, z, Air)
		w.settle()
		w.Set(x, y, z, was)
		w.settle()
		i++
	}
}

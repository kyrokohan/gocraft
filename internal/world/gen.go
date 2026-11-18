package world

import "math"

// Layers of the terrain, counted as the layer of a column's top block.
const (
	rockLine = 74  // from here up the ground is bare stone
	snowLine = 84  // and from here up it is snow
	maxTop   = 112 // no ground is higher, which leaves room for a tree
)

// treeSquare is the side of the squares of a grid over the ground, each of which holds
// one tree or none, so that no two trees stand in each other.
const treeSquare = 7

// lattice returns a number from 0 up to 1 fixed by a point of a grid and a seed.
func lattice(x, z int, seed uint32) float64 {
	h := uint32(x)*0x9E3779B1 ^ uint32(z)*0x85EBCA77 ^ seed*0xC2B2AE3D
	h ^= h >> 15
	h *= 0x2C1B3C6D
	h ^= h >> 12
	h *= 0x297A2D39
	h ^= h >> 15
	return float64(h) / (1 << 32)
}

// smooth returns noise from 0 to 1 that changes gently from one whole number to the next.
func smooth(x, z float64, seed uint32) float64 {
	x0, z0 := math.Floor(x), math.Floor(z)
	tx, tz := x-x0, z-z0
	tx, tz = tx*tx*(3-2*tx), tz*tz*(3-2*tz)
	i, j := int(x0), int(z0)
	near := lattice(i, j, seed)*(1-tx) + lattice(i+1, j, seed)*tx
	far := lattice(i, j+1, seed)*(1-tx) + lattice(i+1, j+1, seed)*tx
	return near*(1-tz) + far*tz
}

// hills returns noise from 0 to 1 with detail at four sizes, each half the size and half
// the strength of the one before.
func hills(x, z float64, seed uint32) float64 {
	sum, strength, total := 0.0, 1.0, 0.0
	for octave := range uint32(4) {
		sum += strength * smooth(x, z, seed+octave*101)
		total += strength
		x, z, strength = x*2, z*2, strength/2
	}
	return sum / total
}

// floorDiv divides and rounds down, where Go rounds toward 0: the square of a grid that
// a coordinate is in, on either side of 0.
func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// Generate returns the world a seed makes, with the chunks round its origin loaded.
// The same seed always makes the same world.
func Generate(seed int64) *World {
	return newWorld(seed, Terrain(seed), "", 0, 0)
}

// Terrain returns the Generator of the world a seed makes: rolling ground with a few
// peaks, water up to SeaLevel, beaches, and trees on the grass.
func Terrain(seed int64) Generator {
	s := uint32(seed) ^ uint32(seed>>32)
	return func(cx, cz int, blocks []Block) {
		terrain{seed: s, cx: cx, cz: cz, blocks: blocks}.build()
	}
}

// terrain is one chunk being made. Everything it puts down is worked out from the seed
// and the place alone, never from what is in a chunk beside it, so a chunk comes out
// the same whenever and in whatever order it is made, and a tree that stands on the
// line between two chunks is built half by each.
type terrain struct {
	seed   uint32
	cx, cz int
	blocks []Block
}

// put sets the block at a place in the world, if the place is in this chunk. Leaves
// and the like only take the place of air.
func (t terrain) put(x, y, z int, b Block, onlyAir bool) {
	lx, lz := x-t.cx*ChunkSize, z-t.cz*ChunkSize
	if uint(lx) >= ChunkSize || uint(lz) >= ChunkSize || uint(y) >= Height {
		return
	}
	at := (y*ChunkSize+lz)*ChunkSize + lx
	if onlyAir && t.blocks[at] != Air {
		return
	}
	t.blocks[at] = b
}

func (t terrain) build() {
	x0, z0 := t.cx*ChunkSize, t.cz*ChunkSize
	for z := z0; z < z0+ChunkSize; z++ {
		for x := x0; x < x0+ChunkSize; x++ {
			t.column(x, z, groundTop(x, z, t.seed))
		}
	}

	// The trees whose leaves can reach into this chunk: those rooted in it or within
	// two blocks of it.
	for gz := floorDiv(z0-2, treeSquare); gz <= floorDiv(z0+ChunkSize+1, treeSquare); gz++ {
		for gx := floorDiv(x0-2, treeSquare); gx <= floorDiv(x0+ChunkSize+1, treeSquare); gx++ {
			if lattice(gx, gz, t.seed+7001) > 0.38 {
				continue
			}
			x := gx*treeSquare + 2 + int(lattice(gx, gz, t.seed+7002)*3)
			z := gz*treeSquare + 2 + int(lattice(gx, gz, t.seed+7003)*3)
			if top := groundTop(x, z, t.seed); grassy(top) {
				t.tree(x, top+1, z, 4+int(lattice(gx, gz, t.seed+7004)*3))
			}
		}
	}
}

// groundTop is the layer of the top block of the ground in a column.
func groundTop(x, z int, seed uint32) int {
	fx, fz := float64(x), float64(z)
	h := float64(SeaLevel) + (hills(fx/72, fz/72, seed)-0.42)*62
	if peak := hills(fx/40, fz/40, seed+977); peak > 0.55 {
		h += (peak - 0.55) * 130
	}
	return min(max(int(h), 1), maxTop)
}

// grassy reports whether the ground of a column with its top at a layer is grass.
func grassy(top int) bool {
	return top > SeaLevel+1 && top < rockLine
}

// column builds the ground of one column up to the layer top, and the water over it.
func (t terrain) column(x, z, top int) {
	surface, under := Grass, Dirt
	switch {
	case top >= snowLine:
		surface, under = Snow, Stone
	case top >= rockLine:
		surface, under = Stone, Stone
	case top <= SeaLevel+1:
		surface, under = Sand, Sand // a beach, or the bed of the water
	}

	t.put(x, 0, z, Deepstone, false)
	for y := 1; y < top; y++ {
		b := Stone
		if y >= top-3 {
			b = under
		}
		t.put(x, y, z, b, false)
	}
	t.put(x, top, z, surface, false)
	for y := top + 1; y <= SeaLevel; y++ {
		t.put(x, y, z, Water, false)
	}
}

// tree stands a tree with its lowest log at (x, y, z) and a trunk of the given height.
func (t terrain) tree(x, y, z, trunk int) {
	for i := range trunk {
		t.put(x, y+i, z, Log, false)
	}
	crown := y + trunk - 2 // the leaves start two below the top of the trunk
	for layer := range 4 {
		reach := 2
		if layer >= 2 {
			reach = 1
		}
		for dz := -reach; dz <= reach; dz++ {
			for dx := -reach; dx <= reach; dx++ {
				corner := abs(dx) == reach && abs(dz) == reach
				// The top is a plus, and the lower corners are there or not by chance.
				if corner && (layer == 3 || lattice(x+dx+layer*31, z+dz, t.seed+7005) < 0.5) {
					continue
				}
				t.put(x+dx, crown+layer, z+dz, Leaves, true)
			}
		}
	}
}

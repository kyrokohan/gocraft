package world

import "math"

// Layers of the terrain, counted as the layer of a column's top block.
const (
	rockLine = 76  // from here up the ground is bare stone
	snowLine = 86  // and from here up it is snow
	maxTop   = 112 // no ground is higher, which leaves room for a tree
)

// treeSquare is the side of the squares of a grid over the ground, each of which holds
// one tree or none, so that no two trees stand in each other.
const treeSquare = 7

// Biome is a kind of country: what the ground is made of and what grows on it.
type Biome uint8

const (
	Sea    Biome = iota // ground under the water
	Shore               // sand at the water's edge
	Plains              // grass, with flowers and few trees
	Forest              // grass under many trees
	Desert              // sand over sandstone, with cactus
	Tundra              // snow, with pines
	Crags               // bare stone, high up, with snow higher still
)

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

// ramp returns 0 below lo, 1 above hi, and a smooth rise between.
func ramp(lo, hi, v float64) float64 {
	t := min(max((v-lo)/(hi-lo), 0), 1)
	return t * t * (3 - 2*t)
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

// Terrain returns the Generator of the world a seed makes: seas and land, flat country
// and high, warm and cold, wet and dry, each with what grows there.
func Terrain(seed int64) Generator {
	s := uint32(seed) ^ uint32(seed>>32)
	return func(cx, cz int, blocks []Block) {
		terrain{seed: s, cx: cx, cz: cz, blocks: blocks}.build()
	}
}

// land is what the seed says of one column of the world: how high its ground is and
// what country it is in. It is worked out from the seed and the place alone.
type land struct {
	top   int // the layer of the top block of the ground
	biome Biome
	cold  bool // cold enough for ice to lie on water
}

// landAt returns the land of the column at x, z.
func landAt(x, z int, seed uint32) land {
	fx, fz := float64(x), float64(z)

	// Four slow noises: how far above the sea the country stands, how rough it is,
	// how warm and how wet. And one quick one, for the lie of the ground itself.
	stand := hills(fx/300, fz/300, seed+1)
	rough := ramp(0.44, 0.72, hills(fx/170, fz/170, seed+2))
	warm := hills(fx/380, fz/380, seed+3) + (lattice(x, z, seed+6)-0.5)*0.02
	wet := hills(fx/330, fz/330, seed+4) + (lattice(x, z, seed+7)-0.5)*0.02
	lie := hills(fx/52, fz/52, seed+5) - 0.5

	dry := warm > 0.60 && wet < 0.46
	swell := 6 + 50*rough
	if dry {
		swell = 4 + 14*rough // dunes, not mountains
	}
	h := float64(SeaLevel) + (stand-0.44)*90 + lie*swell
	l := land{top: min(max(int(math.Floor(h)), 2), maxTop), cold: warm < 0.38}

	switch {
	case l.top < SeaLevel:
		l.biome = Sea
	case l.top <= SeaLevel+1:
		l.biome = Shore
	case l.top >= rockLine:
		l.biome = Crags
	case dry:
		l.biome = Desert
	case l.cold:
		l.biome = Tundra
	case wet > 0.55:
		l.biome = Forest
	default:
		l.biome = Plains
	}
	return l
}

// BiomeAt returns the kind of country the column at x, z is in, in the world of a seed.
func BiomeAt(seed int64, x, z int) Biome {
	return landAt(x, z, uint32(seed)^uint32(seed>>32)).biome
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
			l := landAt(x, z, t.seed)
			t.column(x, z, l)
			t.grow(x, z, l)
		}
	}

	// The trees whose leaves can reach into this chunk: those rooted in it or within
	// two blocks of it. A tree takes the place of whatever small thing grew at its foot.
	for gz := floorDiv(z0-2, treeSquare); gz <= floorDiv(z0+ChunkSize+1, treeSquare); gz++ {
		for gx := floorDiv(x0-2, treeSquare); gx <= floorDiv(x0+ChunkSize+1, treeSquare); gx++ {
			x := gx*treeSquare + 2 + int(lattice(gx, gz, t.seed+7002)*3)
			z := gz*treeSquare + 2 + int(lattice(gx, gz, t.seed+7003)*3)
			t.plant(x, z, landAt(x, z, t.seed), lattice(gx, gz, t.seed+7001), lattice(gx, gz, t.seed+7004))
		}
	}
}

// column builds the ground of one column, and the water over it.
func (t terrain) column(x, z int, l land) {
	// What the top block is, what the few blocks under it are, and how many of those.
	surface, under, deep := Grass, Dirt, 3
	switch l.biome {
	case Sea:
		surface, under = Sand, Sand
		switch patch := smooth(float64(x)/9, float64(z)/9, t.seed+31); {
		case patch > 0.68:
			surface, under = Gravel, Gravel
		case patch < 0.24:
			surface, under = Clay, Clay
		}
	case Shore:
		surface, under = Sand, Sand
	case Desert:
		surface, under, deep = Sand, Sandstone, 6
	case Tundra:
		surface = Snow
	case Crags:
		surface, under = Stone, Stone
		if l.top >= snowLine {
			surface = Snow
		}
	}

	t.put(x, 0, z, Deepstone, false)
	for y := 1; y < l.top; y++ {
		b := Stone
		if y >= l.top-deep {
			b = under
		}
		t.put(x, y, z, b, false)
	}
	if l.biome == Desert {
		for y := l.top - 2; y < l.top; y++ {
			t.put(x, y, z, Sand, false) // loose sand over the sandstone
		}
	}
	t.put(x, l.top, z, surface, false)

	for y := l.top + 1; y <= SeaLevel; y++ {
		t.put(x, y, z, Water, false)
	}
	if l.cold && l.top < SeaLevel {
		t.put(x, SeaLevel, z, Ice, false)
	}
}

// grow puts the small plant, if any, that stands on a column.
func (t terrain) grow(x, z int, l land) {
	chance := lattice(x, z, t.seed+8001)
	var b Block
	switch l.biome {
	case Plains:
		switch {
		case chance < 0.16:
			b = TallGrass
		case chance < 0.19:
			// Flowers of one color grow together.
			b = RedFlower
			if smooth(float64(x)/18, float64(z)/18, t.seed+8002) < 0.5 {
				b = YellowFlower
			}
		}
	case Forest:
		switch {
		case chance < 0.07:
			b = TallGrass
		case chance < 0.095:
			b = Bush
		case chance < 0.1:
			b = Mushroom
		}
	case Desert:
		if chance < 0.015 {
			b = DryBush
		}
	case Tundra:
		if chance < 0.008 {
			b = DryBush
		}
	case Shore:
		// Reeds stand at the water's edge: on sand a step above the water, beside a
		// column that is under it.
		if chance < 0.3 && l.top == SeaLevel+1 && t.byWater(x, z) {
			b = Reed
		}
	}
	if b != Air {
		t.put(x, l.top+1, z, b, true)
	}
}

// byWater reports whether a column beside the one at x, z is under water.
func (t terrain) byWater(x, z int) bool {
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if n := landAt(x+d[0], z+d[1], t.seed); n.top < SeaLevel && !n.cold {
			return true
		}
	}
	return false
}

// plant stands the tree, if any, that grows on the column at x, z. chance says whether
// one does and kind what it is like, each a number from 0 to 1 of the tree's own.
func (t terrain) plant(x, z int, l land, chance, kind float64) {
	y := l.top + 1
	switch l.biome {
	case Forest:
		if chance < 0.66 {
			if kind < 0.3 {
				t.paleTree(x, y, z, 5+int(kind*10))
			} else {
				t.broadTree(x, y, z, 4+int(kind*3))
			}
		}
	case Plains:
		if chance < 0.07 {
			if kind < 0.4 {
				t.paleTree(x, y, z, 5+int(kind*5))
			} else {
				t.broadTree(x, y, z, 4+int(kind*3))
			}
		}
	case Tundra:
		if chance < 0.32 {
			t.pine(x, y, z, 6+int(kind*4))
		}
	case Desert:
		if chance < 0.1 {
			for i := range 1 + int(kind*3) {
				t.put(x, y+i, z, Cactus, false)
			}
		}
	}
}

// crown puts one layer of a tree's leaves round its trunk: a square reach blocks from
// the trunk each way, with its corners there or not by chance, or never.
func (t terrain) crown(x, y, z, reach int, leaves Block, corners bool) {
	for dz := -reach; dz <= reach; dz++ {
		for dx := -reach; dx <= reach; dx++ {
			if abs(dx) == reach && abs(dz) == reach && reach > 0 &&
				(!corners || lattice(x+dx+y*31, z+dz, t.seed+7005) < 0.5) {
				continue
			}
			t.put(x+dx, y, z+dz, leaves, true)
		}
	}
}

// broadTree stands a broad-leaved tree with its lowest log at (x, y, z): a short trunk
// under a wide round crown.
func (t terrain) broadTree(x, y, z, trunk int) {
	for i := range trunk {
		t.put(x, y+i, z, Log, false)
	}
	top := y + trunk
	t.crown(x, top-2, z, 2, Leaves, true)
	t.crown(x, top-1, z, 2, Leaves, true)
	t.crown(x, top, z, 1, Leaves, true)
	t.crown(x, top+1, z, 1, Leaves, false)
}

// paleTree stands a pale tree: a tall thin trunk with a small crown at the top.
func (t terrain) paleTree(x, y, z, trunk int) {
	for i := range trunk {
		t.put(x, y+i, z, PaleLog, false)
	}
	top := y + trunk
	t.crown(x, top-2, z, 1, PaleLeaves, true)
	t.crown(x, top-1, z, 2, PaleLeaves, false)
	t.crown(x, top, z, 1, PaleLeaves, true)
	t.crown(x, top+1, z, 0, PaleLeaves, false)
}

// pine stands a pine: a tall trunk with rings of needles that narrow to a point.
func (t terrain) pine(x, y, z, trunk int) {
	for i := range trunk {
		t.put(x, y+i, z, PineLog, false)
	}
	top := y + trunk
	t.crown(x, top, z, 0, PineNeedles, false) // the tip
	for down, reach := range []int{1, 2, 1, 2, 2} {
		if at := top - 1 - down; at >= y+2 { // the foot of the trunk is bare
			t.crown(x, at, z, reach, PineNeedles, false)
		}
	}
}

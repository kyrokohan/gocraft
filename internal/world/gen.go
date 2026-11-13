package world

import "math"

// Layers of the terrain, counted as the layer of a column's top block.
const (
	rockLine = 34 // from here up the ground is bare stone
	snowLine = 38 // and from here up it is snow
	maxTop   = 50 // no ground is higher, which leaves room for a tree
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

// Generate returns the world a seed makes: rolling ground with a few peaks, water up to
// SeaLevel, beaches, and trees on the grass. The same seed always makes the same world.
func Generate(seed int64) *World {
	w := New()
	w.Seed = seed
	s := uint32(seed) ^ uint32(seed>>32)

	for z := range SizeZ {
		for x := range SizeX {
			top := groundTop(x, z, s)
			w.fillColumn(x, z, top)
		}
	}

	// A tree in some of the squares of a grid, so that no two stand in each other.
	const square = 7
	for gz := range SizeZ / square {
		for gx := range SizeX / square {
			if lattice(gx, gz, s+7001) > 0.38 {
				continue
			}
			x := gx*square + 2 + int(lattice(gx, gz, s+7002)*3)
			z := gz*square + 2 + int(lattice(gx, gz, s+7003)*3)
			if x < 3 || z < 3 || x >= SizeX-3 || z >= SizeZ-3 {
				continue
			}
			y := w.Surface(x, z)
			if w.At(x, y, z) != Grass {
				continue
			}
			w.plantTree(x, y+1, z, 4+int(lattice(gx, gz, s+7004)*3), s)
		}
	}
	return w
}

// groundTop is the layer of the top block of the ground in a column.
func groundTop(x, z int, seed uint32) int {
	fx, fz := float64(x), float64(z)
	h := float64(SeaLevel) + (hills(fx/72, fz/72, seed)-0.42)*52
	if peak := hills(fx/40, fz/40, seed+977); peak > 0.55 {
		h += (peak - 0.55) * 70
	}
	return min(max(int(h), 1), maxTop)
}

// fillColumn builds the ground of one column up to the layer top, and the water over it.
func (w *World) fillColumn(x, z, top int) {
	surface, under := Grass, Dirt
	switch {
	case top >= snowLine:
		surface, under = Snow, Stone
	case top >= rockLine:
		surface, under = Stone, Stone
	case top <= SeaLevel+1:
		surface, under = Sand, Sand // a beach, or the bed of the water
	}

	w.Set(x, 0, z, Deepstone)
	for y := 1; y < top; y++ {
		b := Stone
		if y >= top-3 {
			b = under
		}
		w.Set(x, y, z, b)
	}
	w.Set(x, top, z, surface)
	for y := top + 1; y <= SeaLevel; y++ {
		w.Set(x, y, z, Water)
	}
}

// plantTree stands a tree with its lowest log at (x, y, z) and a trunk of the given
// height. Leaves only take the place of air.
func (w *World) plantTree(x, y, z, trunk int, seed uint32) {
	for i := range trunk {
		w.Set(x, y+i, z, Log)
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
				if corner && (layer == 3 || lattice(x+dx+layer*31, z+dz, seed+7005) < 0.5) {
					continue
				}
				if w.At(x+dx, crown+layer, z+dz) == Air {
					w.Set(x+dx, crown+layer, z+dz, Leaves)
				}
			}
		}
	}
}

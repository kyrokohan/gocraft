package world

import "math"

// A fire mountain is a cone of dark rock with a crater at its top and a lake of lava in
// the crater, in a ring of ashlands. The ground is cut into great squares, and the seed
// puts one mountain in each, well inside it: so a column asks only about the mountain of
// the square it lies in, and a chunk builds its part of a mountain from the seed alone,
// as it does everything.
//
// The squares are set so that the world begins in the middle of one, and that one puts
// its mountain a journey from there: far enough to set out for, and near enough to see
// the smoke of.
const (
	mountSquare = 1600 // the side of a square: a mountain about every 1,500 blocks
	mountFoot   = 40   // how far from its middle the cone reaches: it is 80 across
	mountAsh    = 70   // how far from its middle the ground is ash
	mountHigh   = 40   // how high over the land it stands on the rim of its crater is
	craterWide  = 9    // how far from its middle the rim is
	craterDeep  = 10   // how far under the rim the middle of the crater's floor is
	lavaUnder   = 4    // how far under the rim the lava of the lake stands

	mountNear = 500 // the least the first mountain lies from where the world begins
	mountFar  = 800 // and the most
	mountEdge = mountAsh + 2

	firestoneRing = 3.0 // how far each side of the rim of the crater firestone is found

	eruptLeast = 4.0 // the fewest days from one eruption of a mountain to its next
	eruptMost  = 7.0 // and the most
)

// mountain is one fire mountain: the column at its middle, and how high the rim of its
// crater is.
type mountain struct {
	x, z int
	rim  int
}

// mountPlace returns the column at the middle of the mountain of a square of the grid.
func mountPlace(gx, gz int, seed uint32) (x, z int) {
	if gx == 0 && gz == 0 {
		// The square the world begins in: at a bearing the seed picks, a journey out,
		// and kept inside the square as every mountain is.
		bearing := 2 * math.Pi * lattice(0, 0, seed+9101)
		out := mountNear + (mountFar-mountNear)*lattice(0, 0, seed+9102)
		most := float64(mountSquare/2 - mountEdge)
		return int(min(max(out*math.Cos(bearing), -most), most)), int(min(max(out*math.Sin(bearing), -most), most))
	}
	room := float64(mountSquare - 2*mountEdge)
	return gx*mountSquare - mountSquare/2 + mountEdge + int(lattice(gx, gz, seed+9101)*room),
		gz*mountSquare - mountSquare/2 + mountEdge + int(lattice(gx, gz, seed+9102)*room)
}

// mountainNear returns the mountain whose ashlands the column at x, z lies in, how far
// the column is from its middle, and whether there is one.
func mountainNear(x, z int, seed uint32) (m mountain, far float64, ok bool) {
	gx, gz := floorDiv(x+mountSquare/2, mountSquare), floorDiv(z+mountSquare/2, mountSquare)
	mx, mz := mountPlace(gx, gz, seed)
	if far = math.Hypot(float64(x-mx), float64(z-mz)); far >= mountAsh {
		return mountain{}, far, false
	}
	// It stands on the land as that lies on the whole, without its hills, and no lower
	// than the shore; and its rim is no higher than ground can be.
	stand := hills(float64(mx)/300, float64(mz)/300, seed+1)
	base := max(float64(SeaLevel)+(stand-0.44)*90, SeaLevel+2)
	return mountain{x: mx, z: mz, rim: min(int(base)+mountHigh, maxTop-4)}, far, true
}

// shape returns how high the ground of a column is that lies far from the middle of the
// mountain, where the land without the mountain would be h high: the cone rises from
// the land at its foot to the rim, steeper as it goes, and inside the rim the crater
// falls away as a bowl.
func (m mountain) shape(h, far float64) float64 {
	switch {
	case far >= mountFoot:
		return h
	case far >= craterWide:
		up := (mountFoot - far) / (mountFoot - craterWide)
		return h + (float64(m.rim)-h)*math.Pow(up, 1.4)
	}
	in := far / craterWide
	return float64(m.rim) - craterDeep*(1-in*in)
}

// lava returns the layer the lava of the mountain's lake stands to.
func (m mountain) lava() int {
	return m.rim - lavaUnder
}

// Mountain returns the middle of the crater of the fire mountain nearest the column at
// x, z, at the height of its rim, and how far off it is along the ground. A world that
// is not made from a seed has none.
func (w *World) Mountain(x, z float64) (crater Cell, far float64, ok bool) {
	if w.gen == nil {
		return Cell{}, 0, false
	}
	seed := uint32(w.Seed) ^ uint32(w.Seed>>32)
	far = math.Inf(1)
	// The nearest is in the square the column lies in, or in one beside it.
	gx, gz := floorDiv(int(x)+mountSquare/2, mountSquare), floorDiv(int(z)+mountSquare/2, mountSquare)
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			mx, mz := mountPlace(gx+dx, gz+dz, seed)
			if d := math.Hypot(float64(mx)+0.5-x, float64(mz)+0.5-z); d < far {
				m, _, _ := mountainNear(mx, mz, seed)
				crater, far = Cell{X: mx, Y: m.rim, Z: mz}, d
			}
		}
	}
	return crater, far, true
}

// Erupted returns the day on which the fire mountain nearest the column at x, z last
// began to erupt, as of a day of the world, and the day on which it next will: ok is
// false while it has not erupted yet. A mountain erupts by the seed and the clock alone,
// every eruptLeast to eruptMost days and each mountain on its own days, so nothing of
// when is kept in a save.
func (w *World) Erupted(x, z, day float64) (began, next float64, ok bool) {
	crater, _, found := w.Mountain(x, z)
	if !found {
		return 0, math.Inf(1), false
	}
	seed := uint32(w.Seed) ^ uint32(w.Seed>>32)
	for k := 0; ; k++ {
		next = began + eruptLeast + (eruptMost-eruptLeast)*lattice(crater.X+k, crater.Z, seed+9301)
		if next > day {
			return began, next, k > 0
		}
		began = next
	}
}

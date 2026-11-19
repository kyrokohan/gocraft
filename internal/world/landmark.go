package world

// A landmark is a thing bigger than a tree that the seed puts down: a great skeleton.
// The ground is cut into big squares, and each holds one landmark or none, well inside
// it. So a chunk only asks about the squares it lies in, and builds its part of a
// landmark from the seed alone, as it does a tree.
const (
	markSquare = 160 // the side of a square
	markReach  = 17  // a landmark reaches no farther than this from its middle
)

type landmark struct {
	x, z  int  // its middle
	along bool // it lies along x, and not along z
	ribs  int  // how many blocks of its length are ribs
}

// landmarkIn returns the landmark of a square of the grid, and whether it has one.
func landmarkIn(gx, gz int, seed uint32) (landmark, bool) {
	room := float64(markSquare - 2*markReach)
	m := landmark{
		x:     gx*markSquare + markReach + int(lattice(gx, gz, seed+9002)*room),
		z:     gz*markSquare + markReach + int(lattice(gx, gz, seed+9003)*room),
		along: lattice(gx, gz, seed+9004) < 0.5,
		ribs:  12 + 2*int(lattice(gx, gz, seed+9005)*5),
	}
	// Most of the dead lie in the badlands, and a few on the prairie.
	chance := lattice(gx, gz, seed+9001)
	switch landAt(m.x, m.z, seed).biome {
	case Badlands:
		return m, chance < 0.8
	case Prairie:
		return m, chance < 0.3
	}
	return m, false
}

// landmarks builds the part that lies in this chunk of every landmark near it.
func (t terrain) landmarks() {
	x0, z0 := t.cx*ChunkSize, t.cz*ChunkSize
	for gz := floorDiv(z0, markSquare); gz <= floorDiv(z0+ChunkSize-1, markSquare); gz++ {
		for gx := floorDiv(x0, markSquare); gx <= floorDiv(x0+ChunkSize-1, markSquare); gx++ {
			m, ok := landmarkIn(gx, gz, t.seed)
			if ok && m.x+markReach >= x0 && m.x-markReach < x0+ChunkSize &&
				m.z+markReach >= z0 && m.z-markReach < z0+ChunkSize {
				t.skeleton(m)
			}
		}
	}
}

// The parts of a great skeleton, in blocks along it, and how high over the ground at
// its middle the spine of the shortest runs. A longer one stands higher.
const (
	skullLong = 5
	neckLong  = 2
	tailLong  = 6
	spineHigh = 5
)

// skeleton builds the bones of a great beast that died long ago: a skull, a spine held
// up by ribs that stand as arches wide and high enough to walk through, and a tail that
// sinks to the ground. A bone only takes the place of air, so where the ground has risen
// the skeleton lies half buried in it.
func (t terrain) skeleton(m landmark) {
	// a counts along the skeleton from its middle and c across it.
	at := func(a, c int) (x, z int) {
		if m.along {
			return m.x + a, m.z + c
		}
		return m.x + c, m.z + a
	}
	ground := func(a, c int) int {
		x, z := at(a, c)
		return landAt(x, z, t.seed).top
	}
	bone := func(a, y, c int) {
		x, z := at(a, c)
		t.put(x, y, z, Bone, true)
	}

	a := -(skullLong + neckLong + m.ribs + tailLong) / 2
	high := spineHigh + (m.ribs-12)/4
	spine := ground(0, 0) + high

	// The skull lies on its own ground: a hollow box with a narrower, lower snout, and
	// a hole on each side for an eye.
	head := ground(a+2, 0)
	for da := range skullLong {
		for c := -2; c <= 2; c++ {
			for h := 1; h <= 4; h++ {
				outsideSnout := da < 2 && (abs(c) == 2 || h == 4)
				eye := da == 3 && abs(c) == 2 && h == 3
				hollow := da > 0 && da < skullLong-1 && abs(c) < 2 && h > 1 && h < 4
				if !outsideSnout && !eye && !hollow {
					bone(a+da, head+h, c)
				}
			}
		}
	}
	a += skullLong

	// The neck climbs from the top of the skull to the spine.
	for i := range neckLong {
		bone(a+i, head+4+(spine-head-4)*(i+1)/(neckLong+1), 0)
	}
	a += neckLong

	// Under every other block of the spine, a rib on each side: out and down from the
	// spine, then straight down to the ground.
	for i := range m.ribs {
		bone(a+i, spine, 0)
		if i%2 == 1 {
			continue
		}
		for _, side := range [2]int{-1, 1} {
			bone(a+i, spine, side)
			bone(a+i, spine-1, 2*side)
			for y := spine - 2; y > ground(a+i, 3*side) && y > spine-2-8; y-- {
				bone(a+i, y, 3*side)
			}
		}
	}
	a += m.ribs

	for i := range tailLong {
		bone(a+i, spine-(i+1)*(high-1)/tailLong, 0)
	}
}

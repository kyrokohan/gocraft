package world

import (
	"math"
	"slices"
)

// What lies under the ground, caves and ores, comes from noise in three dimensions.
// Working that out for every block of a chunk would take longer than making the rest
// of it, so it is worked out only at the points of a coarse grid, underStep blocks
// apart, and a block takes the blend of the eight points round it. The grid is the
// same for the whole world, so what a block is does not depend on which chunk is
// being made: a tree in the next chunk can ask whether the ground at its foot is a
// hole and get the answer that chunk will give.
const (
	underStep = 4
	nodesXZ   = ChunkSize/underStep + 1
	nodesY    = Height/underStep + 1
)

// The noises under the ground.
const (
	tunnelA = iota // tunnels run where two noises are both near their middle,
	tunnelB        // which is along a line that winds through the ground
	cavern         // a wide chamber is where a slower noise is high
	lode           // and an ore is where a quick one is high
	underFields
)

// caveFloor is the lowest layer a cave reaches: under it the stone is whole down to
// the bottom of the world.
const caveFloor = 5

// lattice3 returns a number from 0 up to 1 fixed by a point of a grid in space.
func lattice3(x, y, z int, seed uint32) float64 {
	h := uint32(x)*0x9E3779B1 ^ uint32(y)*0xD1B54A35 ^ uint32(z)*0x85EBCA77 ^ seed*0xC2B2AE3D
	h ^= h >> 15
	h *= 0x2C1B3C6D
	h ^= h >> 12
	h *= 0x297A2D39
	h ^= h >> 15
	return float64(h) / (1 << 32)
}

// noise3 returns noise from 0 to 1 in three dimensions that changes gently from one
// whole number to the next.
func noise3(x, y, z float64, seed uint32) float64 {
	x0, y0, z0 := math.Floor(x), math.Floor(y), math.Floor(z)
	tx, ty, tz := x-x0, y-y0, z-z0
	tx, ty, tz = tx*tx*(3-2*tx), ty*ty*(3-2*ty), tz*tz*(3-2*tz)
	i, j, k := int(x0), int(y0), int(z0)
	var corner [8]float64
	for n := range corner {
		corner[n] = lattice3(i+n&1, j+n>>1&1, k+n>>2&1, seed)
	}
	return blend(&corner, tx, ty, tz)
}

// blend returns the value part of the way between eight corners of a box: corner n is
// on the far side along x, y and z when bit 1, 2 and 4 of n is set.
func blend(c *[8]float64, tx, ty, tz float64) float64 {
	x00 := c[0] + (c[1]-c[0])*tx
	x10 := c[2] + (c[3]-c[2])*tx
	x01 := c[4] + (c[5]-c[4])*tx
	x11 := c[6] + (c[7]-c[6])*tx
	y0 := x00 + (x10-x00)*ty
	y1 := x01 + (x11-x01)*ty
	return y0 + (y1-y0)*tz
}

// node returns one of the noises under the ground at a point of the coarse grid, which
// is counted in steps of underStep blocks.
func node(field, ix, iy, iz int, seed uint32) float64 {
	x, y, z := float64(ix*underStep), float64(iy*underStep), float64(iz*underStep)
	switch field {
	case tunnelA:
		return noise3(x/18, y/12, z/18, seed+9001) // tunnels are wider than they are tall
	case tunnelB:
		return noise3(x/18, y/12, z/18, seed+9002)
	case cavern:
		return noise3(x/34, y/20, z/34, seed+9003)
	}
	return noise3(x/7, y/7, z/7, seed+9004)
}

// hollow reports whether the noises at a place make it part of a cave.
func hollow(a, b, c float64) bool {
	return c > cavernLevel || (math.Abs(a-0.5) < tunnelWidth && math.Abs(b-0.5) < tunnelWidth)
}

// carves reports whether a cave may take a block at a layer of a column: not the
// bottom of the world, and not the ground that water lies on or near, which keeps a
// crust five thick so that no sea has a hole in its bed.
func carves(l land, y int) bool {
	if y < caveFloor || y > l.top {
		return false
	}
	return l.top >= SeaLevel+4 || y <= l.top-5
}

// holeAt reports whether the block at a place in the world is taken by a cave. It is
// what a chunk works out for its own blocks from its grid, worked out here for one
// block anywhere, from the same points in the same way.
func holeAt(x, y, z int, l land, seed uint32) bool {
	if !carves(l, y) {
		return false
	}
	ix, iy, iz := floorDiv(x, underStep), y/underStep, floorDiv(z, underStep)
	tx := float64(x-ix*underStep) / underStep
	ty := float64(y-iy*underStep) / underStep
	tz := float64(z-iz*underStep) / underStep
	var value [3]float64
	for field := range value {
		var corner [8]float64
		for n := range corner {
			corner[n] = node(field, ix+n&1, iy+n>>1&1, iz+n>>2&1, seed)
		}
		value[field] = blend(&corner, tx, ty, tz)
	}
	return hollow(value[tunnelA], value[tunnelB], value[cavern])
}

// under is the noises under the ground of one chunk: their values at the points of the
// coarse grid that the chunk's blocks lie between, up to the highest ground in it.
type under struct {
	grid [underFields][nodesY][nodesXZ][nodesXZ]float64

	// may says, for each box of eight points, what the blocks inside it can be. A
	// blend lies between the least and the greatest of its corners, so most boxes can
	// be seen to hold no cave and no ore from their corners alone, and their blocks
	// need no blending at all.
	may [nodesY - 1][nodesXZ - 1][nodesXZ - 1]uint8
}

// What a box of the coarse grid may hold.
const (
	mayTunnel = 1 << iota
	mayCavern
	mayLode
)

// The noises that make a tunnel are near their middle within this much of it, a
// chamber is where its noise is above cavernLevel, and an ore where its is above
// lodeLevel.
const (
	tunnelWidth = 0.07
	cavernLevel = 0.8
	lodeLevel   = 0.78
)

// fill works out the grid for the chunk at (cx, cz), as high as the layer top, the
// highest ground in the chunk.
func (u *under) fill(cx, cz, top int, seed uint32) {
	ix0, iz0 := cx*ChunkSize/underStep, cz*ChunkSize/underStep
	layers := min(top/underStep+2, nodesY)
	for field := range u.grid {
		for iy := range layers {
			for iz := range nodesXZ {
				for ix := range nodesXZ {
					u.grid[field][iy][iz][ix] = node(field, ix0+ix, iy, iz0+iz, seed)
				}
			}
		}
	}

	for iy := range layers - 1 {
		for iz := range nodesXZ - 1 {
			for ix := range nodesXZ - 1 {
				var lo, hi [underFields]float64
				for field := range u.grid {
					c := u.corners(field, ix, iy, iz)
					lo[field], hi[field] = slices.Min(c[:]), slices.Max(c[:])
				}
				var may uint8
				if lo[tunnelA] < 0.5+tunnelWidth && hi[tunnelA] > 0.5-tunnelWidth &&
					lo[tunnelB] < 0.5+tunnelWidth && hi[tunnelB] > 0.5-tunnelWidth {
					may |= mayTunnel
				}
				if hi[cavern] > cavernLevel {
					may |= mayCavern
				}
				if hi[lode] >= lodeLevel {
					may |= mayLode
				}
				u.may[iy][iz][ix] = may
			}
		}
	}
}

// corners returns the values of a noise at the eight points of a box of the grid.
func (u *under) corners(field, ix, iy, iz int) [8]float64 {
	g := &u.grid[field]
	return [8]float64{
		g[iy][iz][ix], g[iy][iz][ix+1], g[iy+1][iz][ix], g[iy+1][iz][ix+1],
		g[iy][iz+1][ix], g[iy][iz+1][ix+1], g[iy+1][iz+1][ix], g[iy+1][iz+1][ix+1],
	}
}

// at returns a noise at a block of the chunk, counted from the chunk's corner.
func (u *under) at(field, lx, y, lz int) float64 {
	ix, iy, iz := lx/underStep, y/underStep, lz/underStep
	corner := u.corners(field, ix, iy, iz)
	return blend(&corner,
		float64(lx-ix*underStep)/underStep, float64(y-iy*underStep)/underStep, float64(lz-iz*underStep)/underStep)
}

// hole reports whether a block of the chunk is taken by a cave.
func (u *under) hole(lx, y, lz int, l land) bool {
	if !carves(l, y) {
		return false
	}
	may := u.may[y/underStep][lz/underStep][lx/underStep]
	if may&mayCavern != 0 && u.at(cavern, lx, y, lz) > cavernLevel {
		return true
	}
	return may&mayTunnel != 0 &&
		math.Abs(u.at(tunnelA, lx, y, lz)-0.5) < tunnelWidth &&
		math.Abs(u.at(tunnelB, lx, y, lz)-0.5) < tunnelWidth
}

// ore returns the ore that lies at a block of stone in the chunk, or Stone if there
// is none. Ores lie in lumps. What a lump is made of depends on how deep it is: coal
// and copper high up, iron lower, and crystal only near the bottom of the world.
func (u *under) ore(x, y, z, lx, lz int, seed uint32) Block {
	if u.may[y/underStep][lz/underStep][lx/underStep]&mayLode == 0 || u.at(lode, lx, y, lz) < lodeLevel {
		return Stone
	}
	// One kind for the whole lump: the lumps are a few blocks across, so every block
	// of one falls in the same box of eight.
	kind := lattice3(floorDiv(x, 8), y/8, floorDiv(z, 8), seed+9005)
	switch {
	case y < 18:
		switch {
		case kind < 0.4:
			return CrystalOre
		case kind < 0.8:
			return IronOre
		}
	case y < 42:
		switch {
		case kind < 0.4:
			return IronOre
		case kind < 0.7:
			return CopperOre
		}
	default:
		if kind < 0.3 {
			return CopperOre
		}
	}
	return CoalOre
}

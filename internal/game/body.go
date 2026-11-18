package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/gocraft/internal/world"
)

// A body is a box that stands on blocks and is stopped by them: the player's, and a
// creature's. It is given as the point in the middle of its bottom, its width (it is as
// deep as it is wide) and its height. The functions here move one, and they are all
// that knows how blocks stop a thing, so the player and the creatures cannot disagree.

// maxStep is the farthest a body moves before blocks are looked at again. It is under
// half a block, so no frame is long enough to carry one through a wall.
const maxStep = 0.4

// gap is how far from a block a body is left when stopped by it. Without it, rounding
// could leave the body a hair inside the block.
const gap = 1e-7

// solid reports whether a cell stops a body. The bottom of the world is a floor; above
// it there is open air, and along the ground it has no end.
func solid(w *world.World, x, y, z int) bool {
	if y < 0 {
		return true
	}
	return w.At(x, y, z).Info().Solid
}

// boxOf returns the low and high corner of a body.
func boxOf(pos geometry.Vector3, width, height float64) (lo, hi [3]float64) {
	half := width / 2
	return [3]float64{pos.X - half, pos.Y, pos.Z - half},
		[3]float64{pos.X + half, pos.Y + height, pos.Z + half}
}

// inWater reports whether the feet of a body are in water.
func inWater(w *world.World, pos geometry.Vector3) bool {
	return w.At(int(math.Floor(pos.X)), int(math.Floor(pos.Y+0.1)), int(math.Floor(pos.Z))) == world.Water
}

// travel moves a body by move and reports on which axes (0 is x, 1 is y, 2 is z) a
// block stopped it. It goes a part at a time, each under half a block, and along one
// axis at a time, which is what lets a body slide along a wall it meets at an angle.
func travel(w *world.World, pos *geometry.Vector3, width, height float64, move geometry.Vector3) (stopped [3]bool) {
	steps := max(1, int(math.Ceil(max(math.Abs(move.X), math.Abs(move.Y), math.Abs(move.Z))/maxStep)))
	part := move.Scale(1 / float64(steps))
	for range steps {
		for _, axis := range [3]int{0, 2, 1} { // along the ground first, then up or down
			delta := [3]float64{part.X, part.Y, part.Z}[axis]
			if slide(w, pos, width, height, axis, delta) {
				stopped[axis] = true
			}
		}
	}
	return stopped
}

// slide moves a body along one axis and, if that takes its box into a solid block, puts
// it back against the block's face. It reports whether the body was stopped.
func slide(w *world.World, pos *geometry.Vector3, width, height float64, axis int, delta float64) bool {
	if delta == 0 {
		return false
	}
	coord := [3]*float64{&pos.X, &pos.Y, &pos.Z}[axis]
	*coord += delta

	lo, hi := boxOf(*pos, width, height)
	var from, to [3]int
	for a := range 3 {
		from[a], to[a] = int(math.Floor(lo[a])), int(math.Ceil(hi[a]))-1
	}

	// The face that stops the body is the nearest one it has gone past.
	stopped := false
	face := math.Inf(1)
	if delta < 0 {
		face = math.Inf(-1)
	}
	for y := from[1]; y <= to[1]; y++ {
		for z := from[2]; z <= to[2]; z++ {
			for x := from[0]; x <= to[0]; x++ {
				if !solid(w, x, y, z) {
					continue
				}
				stopped = true
				cell := float64([3]int{x, y, z}[axis])
				if delta > 0 {
					face = min(face, cell)
				} else {
					face = max(face, cell+1)
				}
			}
		}
	}
	if !stopped {
		return false
	}
	if delta > 0 {
		*coord = face - (hi[axis] - *coord) - gap
	} else {
		*coord = face + (*coord - lo[axis]) + gap
	}
	return true
}

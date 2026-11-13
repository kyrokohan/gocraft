package world

import (
	"math"

	"glonk.dev/geometry"
)

// Face is a side of a block, named by the way it faces.
type Face uint8

const (
	FaceXNeg Face = iota // faces -x
	FaceXPos             // faces +x
	FaceYNeg             // faces down
	FaceYPos             // faces up
	FaceZNeg             // faces -z
	FaceZPos             // faces +z
)

// Normal returns the direction the face looks, one step along one axis.
func (f Face) Normal() (x, y, z int) {
	sign := -1
	if f&1 == 1 {
		sign = 1
	}
	switch f >> 1 {
	case 0:
		return sign, 0, 0
	case 1:
		return 0, sign, 0
	}
	return 0, 0, sign
}

// Hit is a block a ray ran into.
type Hit struct {
	// X, Y and Z are the cell of the block.
	X, Y, Z int

	// Face is the side of the block the ray came in through.
	Face Face

	// T is how far along the ray that side is: the ray meets it at origin + direction*T,
	// so T is a distance in blocks only if the direction is one long.
	T float64

	// Block is what is in the cell.
	Block Block
}

// Ray walks the grid one cell at a time along a line, visiting every cell the line
// passes through, in order. Make one with NewRay and ask World.Next for the blocks on
// its way.
type Ray struct {
	x, y, z                int     // the cell the ray is in
	stepX, stepY, stepZ    int     // which way it leaves a cell on each axis: -1, 0 or 1
	nextX, nextY, nextZ    float64 // how far along the ray the next cell wall is, on each axis
	deltaX, deltaY, deltaZ float64 // how far along the ray one whole cell is, on each axis
	entryX, entryY, entryZ Face    // the face a block is entered through when the ray crosses on that axis
}

// NewRay returns a ray that starts at origin and runs along direction, which can be any
// length. A direction that is zero or not a number goes nowhere and meets nothing.
func NewRay(origin, direction geometry.Vector3) Ray {
	var r Ray
	r.x, r.stepX, r.nextX, r.deltaX = axis(origin.X, direction.X)
	r.y, r.stepY, r.nextY, r.deltaY = axis(origin.Y, direction.Y)
	r.z, r.stepZ, r.nextZ, r.deltaZ = axis(origin.Z, direction.Z)
	r.entryX, r.entryY, r.entryZ = entry(FaceXNeg, r.stepX), entry(FaceYNeg, r.stepY), entry(FaceZNeg, r.stepZ)
	return r
}

// axis sets a ray up along one axis: the cell it starts in, which way it steps, how far
// along the ray the first cell wall is and how far apart the walls are after that.
func axis(origin, direction float64) (cell, step int, next, delta float64) {
	cell = int(math.Floor(origin))
	switch {
	case direction > 0:
		return cell, 1, (float64(cell) + 1 - origin) / direction, 1 / direction
	case direction < 0:
		return cell, -1, (origin - float64(cell)) / -direction, -1 / direction
	}
	return cell, 0, math.Inf(1), math.Inf(1)
}

// entry is the face a block is entered through by a ray stepping one way along an axis,
// whose face toward the negative side is neg: stepping up the axis comes in through that
// one, stepping down through the one opposite.
func entry(neg Face, step int) Face {
	if step < 0 {
		return neg + 1
	}
	return neg
}

// Next moves the ray on to the next cell that is not air and returns it, or false once
// the ray is more than maxT along or has left the world. The cell the ray starts in is
// never returned. Calling Next again goes on from the last hit, so a caller can look
// through water or the holes in leaves.
func (w *World) Next(r *Ray, maxT float64) (Hit, bool) {
	for {
		// Cross the nearest cell wall.
		var t float64
		var face Face
		switch {
		case r.nextX <= r.nextY && r.nextX <= r.nextZ:
			t, face = r.nextX, r.entryX
			r.x += r.stepX
			r.nextX += r.deltaX
			if (r.stepX > 0 && r.x >= SizeX) || (r.stepX < 0 && r.x < 0) {
				return Hit{}, false
			}
		case r.nextY <= r.nextZ:
			t, face = r.nextY, r.entryY
			r.y += r.stepY
			r.nextY += r.deltaY
			if (r.stepY > 0 && r.y >= SizeY) || (r.stepY < 0 && r.y < 0) {
				return Hit{}, false
			}
		default:
			t, face = r.nextZ, r.entryZ
			r.z += r.stepZ
			r.nextZ += r.deltaZ
			if (r.stepZ > 0 && r.z >= SizeZ) || (r.stepZ < 0 && r.z < 0) {
				return Hit{}, false
			}
		}
		// Written so that a NaN ends the ray. An infinite t is a ray with no wall left to
		// cross, which would otherwise stand still for ever.
		if !(t <= maxT) || math.IsInf(t, 1) {
			return Hit{}, false
		}
		if !inside(r.x, r.y, r.z) {
			continue // not in the world yet
		}
		if b := w.blocks[index(r.x, r.y, r.z)]; b != Air {
			return Hit{X: r.x, Y: r.y, Z: r.z, Face: face, T: t, Block: b}, true
		}
	}
}

// Target returns the first block along a ray that can be broken or built on, up to reach
// along it. Water is passed through.
func (w *World) Target(origin, direction geometry.Vector3, reach float64) (Hit, bool) {
	r := NewRay(origin, direction)
	for {
		hit, ok := w.Next(&r, reach)
		if !ok {
			return Hit{}, false
		}
		if hit.Block != Water {
			return hit, true
		}
	}
}

package render

import (
	"math"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/gocraft/internal/assets"
	"github.com/kyrokohan/gocraft/internal/world"
)

// The size of a mossback's body in blocks: the box that blocks stop. It is as deep as
// it is wide, though the animal is longer than that: a box cannot turn, and the animal
// does.
const (
	MossbackWidth  = 0.8
	MossbackHeight = 0.95
)

// The parts of the animal, in blocks, with its feet at 0 and its front toward +z.
const (
	legLength  = 0.35
	legWidth   = 0.2
	legAside   = 0.22 // from the middle to a leg
	legAhead   = 0.3  // from the middle to a front or back leg
	bodyWidth  = 0.7
	bodyHeight = 0.5
	bodyLength = 1.0
	headWidth  = 0.5
	headHeight = 0.45
	headLength = 0.4
	legSwing   = 0.5 // how far a leg swings each way, in radians
)

// The tiles of the creature's picture, from the left, as cmd/genassets draws them.
const (
	tileBack = iota
	tileHide
	tileFace
	tileLeg
	mossbackTiles
)

// Mossback is where one creature is and how it stands: all that drawing it needs.
type Mossback struct {
	// Pos is the middle of the bottom of its body: the point between its feet.
	Pos geometry.Vector3

	// Yaw is the way it faces, as a camera's Yaw is.
	Yaw float64

	// Stride is how far through a step its legs are, in radians: they swing as its sine.
	Stride float64

	// Light is how bright the light it stands in is, from 0 to 1: the Sky's Brightness of
	// the light in its cell.
	Light float64
}

// Mossbacks draws the creatures. They are glonk meshes: boxes with the creature's
// picture on their faces, lit by glonk and faded by the camera's fog, and drawn into
// the canvas after the blocks, whose depth hides them where a block is nearer.
type Mossbacks struct {
	body, head, leg mesh.Mesh
	lights          [2]mesh.Light
}

// NewMossbacks builds the boxes a creature is made of, with the picture from set.
func NewMossbacks(set *assets.Set) *Mossbacks {
	look := mesh.Material{Color: engine.RGB(224, 208, 172), Texture: set.Mossback}

	// Each box says which tile goes on which of its faces, in the order of world.Face.
	hide := [6]int{tileHide, tileHide, tileHide, tileBack, tileHide, tileHide}
	face := hide
	face[world.FaceZPos] = tileFace
	leg := [6]int{tileLeg, tileLeg, tileLeg, tileLeg, tileLeg, tileLeg}

	return &Mossbacks{
		body: box(-bodyWidth/2, legLength, -bodyLength/2, bodyWidth, bodyHeight, bodyLength, hide, look),
		head: box(-headWidth/2, legLength+0.15, bodyLength/2, headWidth, headHeight, headLength, face, look),
		// A leg hangs from its own origin, so that turning it there swings it from the hip.
		leg: box(-legWidth/2, -legLength, -legWidth/2, legWidth, legLength, legWidth, leg, look),
	}
}

// box returns a mesh of a box with one corner at (x, y, z) and the given size, with a
// tile of the picture on each face.
func box(x, y, z, w, h, d float64, tiles [6]int, look mesh.Material) mesh.Mesh {
	m := mesh.Mesh{Material: look}
	// Corner i is on the far side along x, y and z when bit 1, 2 and 4 of i is set, as
	// in glonk's Cube, whose faces these are: each wound so that it faces out.
	for i := range 8 {
		m.Vertices = append(m.Vertices, geometry.Vector3{
			X: x + w*float64(i&1), Y: y + h*float64(i>>1&1), Z: z + d*float64(i>>2&1),
		})
	}
	quads := [6][4]int{
		world.FaceXNeg: {0, 4, 6, 2}, world.FaceXPos: {1, 3, 7, 5},
		world.FaceYNeg: {0, 1, 5, 4}, world.FaceYPos: {2, 6, 7, 3},
		world.FaceZNeg: {0, 2, 3, 1}, world.FaceZPos: {4, 5, 7, 6},
	}

	// place says where on the picture a corner of a face is: across the face along one
	// of its two axes and down it along the other, inside the face's tile. It stays a
	// little inside the tile, so that a pixel on the edge of a face is not taken from
	// the tile beside it.
	place := func(side world.Face, corner int) geometry.Vector2 {
		v := m.Vertices[corner]
		fx, fy, fz := (v.X-x)/w, (v.Y-y)/h, (v.Z-z)/d
		var across, down float64
		switch side >> 1 {
		case 0:
			across, down = fz, 1-fy
		case 1:
			across, down = fx, fz
		default:
			across, down = fx, 1-fy
		}
		inside := func(f float64) float64 { return 0.02 + 0.96*f }
		return geometry.Vector2{X: (float64(tiles[side]) + inside(across)) / mossbackTiles, Y: inside(down)}
	}
	for side, q := range quads {
		side := world.Face(side)
		m.Faces = append(m.Faces, [3]int{q[0], q[1], q[2]}, [3]int{q[0], q[2], q[3]})
		m.FaceTexCoords = append(m.FaceTexCoords,
			[3]geometry.Vector2{place(side, q[0]), place(side, q[1]), place(side, q[2])},
			[3]geometry.Vector2{place(side, q[0]), place(side, q[2]), place(side, q[3])},
		)
	}
	m.UpdateBounds() // so that glonk can skip it when it is off screen
	return m
}

// Draw draws the creatures as the camera sees them, over what is in the canvas. Call
// it after the blocks, and before anything that is seen through.
//
// A creature is lit about as the blocks round it are: its top as bright as the light it
// stands in, and its side away from the sun or the moon less.
func (m *Mossbacks) Draw(c *engine.Canvas, cam mesh.Camera, all []Mossback, sky *Sky) {
	for _, one := range all {
		// The ambient light lights every face, and the other fills in part of what that
		// leaves dark: so a face toward the sun is as bright as the light, and one turned
		// away from it seven tenths of that.
		lit := min(max(one.Light, 0.01), 1)
		m.lights = [2]mesh.Light{
			mesh.Ambient{Level: 0.7 * lit},
			mesh.Directional{Direction: sky.giver().Neg(), Intensity: 0.3 * lit / (1 - 0.7*lit)},
		}
		turned := mesh.Camera{Yaw: one.Yaw}
		ahead, aside := turned.Forward(), turned.Right()

		// put draws a part with its origin a way to the animal's right, up, and ahead
		// of the point between its feet, tipped by pitch.
		put := func(part mesh.Mesh, right, up, front, pitch float64) {
			at := one.Pos.Add(aside.Scale(right)).Add(ahead.Scale(front))
			at.Y += up
			part.Transform = mesh.Transform{Position: at, Yaw: one.Yaw, Pitch: pitch}
			part.DrawShadedPixels(c, cam, m.lights[:]...)
		}
		put(m.body, 0, 0, 0, 0)
		put(m.head, 0, 0, 0, 0)

		// The legs at opposite corners swing together, as a walking animal's do.
		swing := legSwing * math.Sin(one.Stride)
		put(m.leg, -legAside, legLength, legAhead, swing)
		put(m.leg, legAside, legLength, legAhead, -swing)
		put(m.leg, -legAside, legLength, -legAhead, -swing)
		put(m.leg, legAside, legLength, -legAhead, swing)
	}
}

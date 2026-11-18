package render

import (
	"math"

	"glonk.dev/bitmap"
	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/gocraft/internal/assets"
	"github.com/kyrokohan/gocraft/internal/world"
)

// The size of a gloam's body in blocks: the box that blocks stop, as tall as the player
// is, so that it goes wherever the player goes.
const (
	GloamWidth  = 0.6
	GloamHeight = 1.7
)

// The parts of a gloam, in blocks, with the point under it at 0 and its front toward
// +z. It hangs a little over the ground, as smoke does, and its arms are long.
const (
	gloamFloat    = 0.3  // from the ground to the bottom of its body
	gloamBodyW    = 0.6  // wide
	gloamBodyH    = 0.95 // tall
	gloamBodyD    = 0.4  // deep
	gloamHeadW    = 0.5
	gloamHeadH    = 0.45
	gloamHeadD    = 0.45
	gloamHeadUp   = 1.22 // the bottom of the head
	gloamHeadFore = 0.11 // how far ahead of the body the head pokes: it stoops
	gloamArmW     = 0.14
	gloamArmLong  = 0.95
	gloamShoulder = 1.2  // the height the arms hang from
	gloamArmAside = 0.38 // from the middle to an arm
	gloamEyeW     = 0.08
	gloamEyeH     = 0.06
	gloamEyeAside = 0.094 // from the middle of the face to an eye
	gloamEyeUp    = 1.475
	gloamBob      = 0.08 // how far it drifts up and down
)

// The tiles of the gloam's picture, from the left, as cmd/genassets draws them.
const (
	tileSmoke = iota
	tileGloamFace
	tileArm
)

// gloamEyes is the color its eyes glow.
var gloamEyes = engine.RGB(214, 255, 150)

// Gloam is where one gloam is and how it is: all that drawing it needs.
type Gloam struct {
	// Pos is the point on the ground under the middle of it.
	Pos geometry.Vector3

	// Yaw is the way it faces, as a camera's Yaw is.
	Yaw float64

	// Age is how many seconds it has been about, which it drifts and sways by.
	Age float64

	// Reach is how far its arms reach out ahead of it, from 0, hanging, to 1.
	Reach float64

	// Fade is how far it has faded away, from 0, all there, to 1, gone.
	Fade float64

	// Light is how bright the light it is in is, from 0 to 1, as a Mossback's is. Its
	// eyes glow whatever the light.
	Light float64

	// Hurt is true while it shows a hit.
	Hurt bool
}

// Gloams draws the gloams, as glonk meshes like the mossbacks. One that fades is seen
// through, so draw them after everything solid.
type Gloams struct {
	body, head, arm, eye mesh.Mesh
	hurt                 *bitmap.Bitmap
	lights               [2]mesh.Light
}

// NewGloams builds the boxes a gloam is made of, with the picture from set.
func NewGloams(set *assets.Set) *Gloams {
	look := mesh.Material{Color: engine.RGB(40, 36, 58), Texture: set.Gloam}
	smoke := [6]int{tileSmoke, tileSmoke, tileSmoke, tileSmoke, tileSmoke, tileSmoke}
	face := smoke
	face[world.FaceZPos] = tileGloamFace
	arm := [6]int{tileArm, tileArm, tileArm, tileArm, tileArm, tileArm}

	const n = assets.GloamTiles
	return &Gloams{
		body: box(-gloamBodyW/2, gloamFloat, -gloamBodyD/2, gloamBodyW, gloamBodyH, gloamBodyD, smoke, n, look),
		head: box(-gloamHeadW/2, gloamHeadUp, gloamHeadFore-gloamHeadD/2, gloamHeadW, gloamHeadH, gloamHeadD, face, n, look),
		// An arm hangs from its own origin, so that it turns at the shoulder.
		arm: box(-gloamArmW/2, -gloamArmLong, -gloamArmW/2, gloamArmW, gloamArmLong, gloamArmW, arm, n, look),
		// The eyes have no picture: they are one color, lit in full.
		eye:  box(-gloamEyeW/2, -gloamEyeH/2, 0, gloamEyeW, gloamEyeH, 0.02, smoke, n, mesh.Material{Color: gloamEyes}),
		hurt: reddened(set.Gloam),
	}
}

// Draw draws the gloams as the camera sees them, over what is in the canvas.
func (g *Gloams) Draw(c *engine.Canvas, cam mesh.Camera, all []Gloam, sky *Sky) {
	glow := []mesh.Light{mesh.Ambient{Level: 1}}
	for _, one := range all {
		if one.Fade >= 0.98 {
			continue // nothing is left of it to see
		}
		g.lights = lightsIn(one.Light, sky)
		turned := mesh.Camera{Yaw: one.Yaw}
		ahead, aside := turned.Forward(), turned.Right()
		lift := gloamBob * math.Sin(one.Age*2)
		var opacity float64 // solid, to glonk
		if one.Fade > 0 {
			opacity = 1 - one.Fade
		}

		put := func(part mesh.Mesh, right, up, front, pitch float64, lights []mesh.Light) {
			at := one.Pos.Add(aside.Scale(right)).Add(ahead.Scale(front))
			at.Y += up + lift
			part.Transform = mesh.Transform{Position: at, Yaw: one.Yaw, Pitch: pitch}
			part.Material.Opacity = opacity
			if one.Hurt && part.Material.Texture != nil {
				part.Material.Texture = g.hurt
			}
			part.DrawShadedPixels(c, cam, lights...)
		}
		put(g.body, 0, 0, 0, 0, g.lights[:])
		put(g.head, 0, 0, 0, 0, g.lights[:])

		// The arms sway as it drifts, out of step with each other, and reach for what it
		// follows as it comes near.
		reach := 1.3 * one.Reach
		put(g.arm, -gloamArmAside, gloamShoulder, 0, reach+0.12*math.Sin(one.Age*1.7), g.lights[:])
		put(g.arm, gloamArmAside, gloamShoulder, 0, reach+0.12*math.Sin(one.Age*1.7+2), g.lights[:])

		front := gloamHeadFore + gloamHeadD/2 + 0.005
		put(g.eye, -gloamEyeAside, gloamEyeUp, front, 0, glow)
		put(g.eye, gloamEyeAside, gloamEyeUp, front, 0, glow)
	}
}

package render

import (
	"math"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/tile"
	"github.com/kyrokohan/skyspear/internal/world"
)

// Held is what the player holds, as it is drawn low on the right of the view.
type Held struct {
	Item world.Item

	// Swing is how far through a swing of the hand it is, from 0 to 1, as a block is dug,
	// broken or placed, a creature hit or food eaten. 0 is still.
	Swing float64

	// Bob is how far the hand has risen with the player's steps, from 0 to 1.
	Bob float64

	// Light is how bright the light at the player's head is, from 0 to 1, as for a Lying.
	Light float64
}

// handScale is how much smaller than it looks the hand is made, and as much nearer the
// eye: so near that no block, not a ceiling just over the head, can come between.
const handScale = 0.07

// handView is the camera the hand is drawn with: its own, at the origin and facing +z,
// so the hand stays where it is on the screen however the view turns.
var handView = mesh.Camera{FOV: 70 * math.Pi / 180, Near: 0.01, Far: 1}

// plate returns a thin square with an item's picture on its faces, one across, with the
// middle of its bottom at its origin: a tool or a thing that is not a block, as it is held.
func (d *Drops) plate(item world.Item) *mesh.Mesh {
	if m, ok := d.plates[item]; ok {
		return m
	}
	t := int(item.Icon())
	m := box(-0.5, 0, -0.005, 1, 1, 0.01, [6]int{t, t, t, t, t, t}, int(tile.Count), mesh.Material{Color: engine.RGB(200, 200, 200), Texture: d.strip.Strip})
	// Only its front and back, two triangles each in box's order of faces: its edges are
	// too thin to see, and would be drawn as dots.
	z := 2 * int(world.FaceZNeg)
	m.Faces, m.FaceTexCoords = m.Faces[z:z+4], m.FaceTexCoords[z:z+4]
	d.plates[item] = &m
	return &m
}

// DrawHand draws what the player holds over the view, after everything in the world: a
// block as a small block turned to show three faces, and anything else as its picture,
// held up. Nothing is drawn for an empty hand.
func (d *Drops) DrawHand(c *engine.Canvas, h Held) {
	if !h.Item.Known() || c.Height() == 0 {
		return
	}
	// Where it is held: low on the right, as a part of half the view's height and width
	// at the depth it is held at, whatever the shape of the canvas.
	const depth = 1.2
	up := math.Tan(handView.FOV/2) * depth
	across := up * float64(c.Width()) / float64(c.Height())
	swing := math.Sin(math.Pi * h.Swing)
	at := geometry.Vector3{X: (0.75 - 0.12*swing) * across, Y: -0.8*up + 0.05*h.Bob - 0.06*swing, Z: depth + 0.2*swing}

	// A swing tips the top of what is held away, as a stroke forward does.
	m, size := d.plate(h.Item), 0.55
	turn := mesh.Transform{Yaw: -0.25, Roll: 0.35, Pitch: 0.1 - 0.7*swing}
	if _, ok := h.Item.Block(); ok {
		m, size = d.cube(h.Item), 0.3
		turn = mesh.Transform{Yaw: 0.75, Pitch: 0.35 - 0.5*swing}
	}
	turn.Position = at.Add(geometry.Vector3{Y: -size / 2}).Scale(handScale)
	turn.Scale = size * handScale
	m.Transform = turn

	lit := min(max(h.Light, 0.01), 1)
	d.lights = [2]mesh.Light{
		mesh.Ambient{Level: 0.7 * lit},
		mesh.Directional{Direction: geometry.Vector3{X: 0.4, Y: -1, Z: 0.5}, Intensity: 0.3 * lit / (1 - 0.7*lit)},
	}
	m.DrawShadedPixels(c, handView, d.lights[:]...)
}

package render

import (
	"math"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/tile"
	"github.com/kyrokohan/skyspear/internal/world"
)

// Lying is a thing lying on the ground, as it is drawn.
type Lying struct {
	Item world.Item

	// Pos is the middle of its bottom, and Size how far across it is.
	Pos  geometry.Vector3
	Size float64

	// Age is how long it has lain there, in seconds, which turns it and bobs it.
	Age float64

	// Light is how bright the light it lies in is, from 0 to 1, as for a Mossback.
	Light float64
}

// Drops draws things lying on the ground: each a small block with the tiles of its item
// on its faces, turning slowly and bobbing, lit as a creature is.
type Drops struct {
	strip  *assets.Set
	cubes  map[world.Item]*mesh.Mesh // a cube for each item, made the first time it is drawn
	plates map[world.Item]*mesh.Mesh // and a plate, for an item held that is not a block
	shafts map[world.Item]*mesh.Mesh // and a shaft, for a long item in flight
	lights [2]mesh.Light
}

// NewDrops returns a drawer of things on the ground with the tiles of set.
func NewDrops(set *assets.Set) *Drops {
	return &Drops{strip: set, cubes: map[world.Item]*mesh.Mesh{}, plates: map[world.Item]*mesh.Mesh{}, shafts: map[world.Item]*mesh.Mesh{}}
}

// cube returns the cube of an item, one block across, with its bottom in the middle of
// its origin: a block's own tiles on its faces, or an item's picture on each face.
func (d *Drops) cube(item world.Item) *mesh.Mesh {
	if m, ok := d.cubes[item]; ok {
		return m
	}
	t := int(item.Icon())
	tiles := [6]int{t, t, t, t, t, t}
	if b, ok := item.Block(); ok {
		info := b.Info()
		side, top, bottom := int(info.Side), int(info.Top), int(info.Bottom)
		tiles = [6]int{side, side, bottom, top, side, side}
	}
	m := box(-0.5, 0, -0.5, 1, 1, 1, tiles, int(tile.Count), mesh.Material{Color: engine.RGB(200, 200, 200), Texture: d.strip.Strip})
	d.cubes[item] = &m
	return &m
}

// Draw draws the things as the camera sees them, over what is in the canvas, after the
// blocks.
func (d *Drops) Draw(c *engine.Canvas, cam mesh.Camera, all []Lying, sky *Sky) {
	for _, one := range all {
		m := d.cube(one.Item)
		d.lights = lightsIn(one.Light, sky)
		at := one.Pos
		at.Y += one.Size * 0.3 * (1 + math.Sin(one.Age*2.5)) // a bob, never into the ground
		m.Transform = mesh.Transform{Position: at, Yaw: one.Age * 1.2, Scale: one.Size}
		m.DrawShadedPixels(c, cam, d.lights[:]...)
	}
}

// Bit is a small piece of a broken block, flying from where it broke, as it is drawn.
type Bit struct {
	Item world.Item

	// Pos is its middle, Size how far across it is, and Spin how far it has turned as
	// it tumbles, in radians.
	Pos  geometry.Vector3
	Size float64
	Spin float64

	// Light is how bright the light it is in is, from 0 to 1, as for a Lying.
	Light float64
}

// DrawBits draws the bits of broken blocks as the camera sees them, each a tiny block
// with the tiles of its item, tumbling. Call it after the blocks.
func (d *Drops) DrawBits(c *engine.Canvas, cam mesh.Camera, bits []Bit, sky *Sky) {
	for _, one := range bits {
		m := d.cube(one.Item)
		d.lights = lightsIn(one.Light, sky)
		at := one.Pos
		at.Y -= one.Size / 2 // the cube's origin is the middle of its bottom
		m.Transform = mesh.Transform{Position: at, Yaw: one.Spin, Pitch: 0.7 * one.Spin, Scale: one.Size}
		m.DrawShadedPixels(c, cam, d.lights[:]...)
	}
}

// Flying is a thing in flight, or one that sticks where it came down, as it is drawn.
type Flying struct {
	Item world.Item

	// Pos is its middle and Size how far across it is. Along is the way it flies or
	// flew, of any length: a Long thing is drawn as its picture, lying along that way
	// point first, and any other as a small block.
	Pos   geometry.Vector3
	Along geometry.Vector3
	Long  bool
	Size  float64

	// Light is how bright the light it is in is, from 0 to 1, as for a Lying.
	Light float64
}

// shaft returns the picture of a long item laid along the z axis, point first, one
// across by its picture's side, about its middle. A long thing is drawn in its picture
// from the lower left corner to the upper right, so the picture is turned to bring that
// line onto the axis; and it is there twice, upright and flat, each seen from both of
// its sides, so that the thing is seen from wherever it is looked at.
func (d *Drops) shaft(item world.Item) *mesh.Mesh {
	if m, ok := d.shafts[item]; ok {
		return m
	}
	const half = math.Sqrt2 / 2 // from the middle of the picture to a corner of it
	t, count := float64(item.Icon()), float64(tile.Count)
	at := func(u, v float64) geometry.Vector2 { // a corner of the picture, a little inside its tile
		return geometry.Vector2{X: (t + 0.02 + 0.96*u) / count, Y: 0.02 + 0.96*v}
	}
	tail, tip, left, right := at(0, 1), at(1, 0), at(0, 0), at(1, 1)
	m := mesh.Mesh{
		Material: mesh.Material{Color: engine.RGB(200, 200, 200), Texture: d.strip.Strip},
		Vertices: []geometry.Vector3{
			{Z: -half}, {Z: half}, // the tail and the tip
			{Y: half}, {Y: -half}, // the other two corners, upright
			{X: half}, {X: -half}, // and flat
		},
	}
	for _, q := range [2][2]int{{2, 3}, {4, 5}} {
		// Each half of the picture, on the one side of the axis and on the other, wound
		// both ways.
		for _, tri := range [4][3]int{{0, 1, q[0]}, {0, q[0], 1}, {0, q[1], 1}, {0, 1, q[1]}} {
			m.Faces = append(m.Faces, tri)
			var uv [3]geometry.Vector2
			for i, v := range tri {
				uv[i] = map[int]geometry.Vector2{0: tail, 1: tip, q[0]: left, q[1]: right}[v]
			}
			m.FaceTexCoords = append(m.FaceTexCoords, uv)
		}
	}
	m.UpdateBounds()
	d.shafts[item] = &m
	return &m
}

// DrawFlying draws what flies as the camera sees it, over what is in the canvas, after
// the blocks.
func (d *Drops) DrawFlying(c *engine.Canvas, cam mesh.Camera, all []Flying, sky *Sky) {
	for _, one := range all {
		d.lights = lightsIn(one.Light, sky)
		m, at, turn := d.cube(one.Item), one.Pos, mesh.Transform{}
		if one.Long {
			// The shaft lies along z: tipped up and turned to lie along its way.
			m = d.shaft(one.Item)
			turn.Pitch = math.Atan2(one.Along.Y, math.Hypot(one.Along.X, one.Along.Z))
			turn.Yaw = math.Atan2(one.Along.X, one.Along.Z)
		} else {
			at.Y -= one.Size / 2 // the cube's origin is the middle of its bottom
		}
		turn.Position, turn.Scale = at, one.Size
		m.Transform = turn
		m.DrawShadedPixels(c, cam, d.lights[:]...)
	}
}

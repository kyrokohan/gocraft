package render

import (
	"math"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/gocraft/internal/assets"
	"github.com/kyrokohan/gocraft/internal/tile"
	"github.com/kyrokohan/gocraft/internal/world"
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
	lights [2]mesh.Light
}

// NewDrops returns a drawer of things on the ground with the tiles of set.
func NewDrops(set *assets.Set) *Drops {
	return &Drops{strip: set, cubes: map[world.Item]*mesh.Mesh{}}
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

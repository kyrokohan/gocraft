// Package render draws the block world into a glonk Canvas, one ray for each pixel.
//
// The blocks are not glonk meshes. Every pixel sends a ray through the grid and takes
// the color of the tile it lands on, which is quick, lets a far block be drawn from a
// smaller copy of its tile so that it does not flicker, and lets water be seen into.
// The rays are the ones glonk's Camera gives for each pixel, and they write glonk's
// depth, so anything glonk draws into the Canvas afterwards with a depth (a mesh, a
// line) is hidden by the blocks in front of it and hides those behind. The camera's far
// plane and fog are followed too, so that such a thing fades with the blocks round it.
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

// ViewDistance is how deep into the view the world is drawn, in blocks: the Far of the
// camera the game uses. It is also how deep the rays go for a camera with no Far, since
// a ray has to end somewhere.
const ViewDistance = 64.0

// Fog is the fog of the camera the game uses: the last two fifths of ViewDistance fade
// into the color the sky has at the horizon, so that the world ends in haze and not in
// a line. Blocks and glonk's meshes both fade by it.
var Fog = &mesh.Fog{Color: skyHorizon.engine(), Start: 0.6 * ViewDistance, End: ViewDistance}

// faceLight is how bright each face of a block is drawn, out of 256, indexed by
// world.Face: the top is fully lit, the sides less, and the bottom least. It is what
// gives the blocks their shape, as there are no shadows.
var faceLight = [6]uint32{205, 205, 128, 256, 166, 166}

// color is a color with a channel in each field, from 0 to 255, for mixing. The pixel
// loop mixes whole numbers with it, which is quicker there than engine.Mix on packed
// colors.
type color struct{ r, g, b uint32 }

var (
	skyHorizon = color{190, 220, 245}
	skyZenith  = color{70, 130, 220}
	sunColor   = color{255, 244, 200}
	deepWater  = color{24, 60, 130}
)

// sunDirection is where the sun stands, one long.
var sunDirection = geometry.Vector3{X: 0.35, Y: 0.8, Z: 0.45}.Normalize()

// mix returns the color part of the way from c to o, where part runs from 0 (c) to 256 (o).
func (c color) mix(o color, part uint32) color {
	keep := 256 - part
	return color{(c.r*keep + o.r*part) >> 8, (c.g*keep + o.g*part) >> 8, (c.b*keep + o.b*part) >> 8}
}

// lit returns the color at a brightness out of 256.
func (c color) lit(light uint32) color {
	return color{c.r * light >> 8, c.g * light >> 8, c.b * light >> 8}
}

func (c color) engine() engine.Color {
	return engine.RGB(uint8(c.r), uint8(c.g), uint8(c.b))
}

func unpack(pixel uint32) color {
	return color{pixel >> 16 & 0xFF, pixel >> 8 & 0xFF, pixel & 0xFF}
}

// part turns a number from 0 to 1 into a part out of 256 for mix, clamped.
func part(v float64) uint32 {
	return uint32(min(max(v, 0), 1) * 256)
}

// rays is where every pixel of a canvas looks from a camera.
//
// glonk's Camera.RayPixels gives the ray through one pixel, and works out the camera's
// angles each time it is asked. The direction changes by the same amount from each
// pixel to the next, so three of its rays are enough for a whole frame: the one through
// the first pixel, and the ones through its neighbors across and down.
type rays struct {
	eye          geometry.Vector3
	first        geometry.Vector3 // the direction through the middle of pixel (0, 0)
	across, down geometry.Vector3 // what is added to it for each pixel across and down
}

// newRays returns the rays of a camera on a canvas. ok is false for a camera that sees
// nothing.
func newRays(cam mesh.Camera, c *engine.Canvas) (r rays, ok bool) {
	eye, first, ok0 := cam.RayPixels(0.5, 0.5, c)
	_, right, ok1 := cam.RayPixels(1.5, 0.5, c)
	_, below, ok2 := cam.RayPixels(0.5, 1.5, c)
	if !ok0 || !ok1 || !ok2 {
		return rays{}, false
	}
	return rays{eye: eye, first: first, across: right.Sub(first), down: below.Sub(first)}, true
}

// through returns the direction of the ray through the middle of a pixel. Like the
// directions glonk gives, it is one deep along the camera's Look and not one long, so a
// point t along it is at the depth t.
func (r rays) through(px, py int) geometry.Vector3 {
	return r.first.Add(r.across.Scale(float64(px))).Add(r.down.Scale(float64(py)))
}

// plantLight is how bright a plant is drawn, out of 256: nearly as bright as the top
// of a block, as it stands in the open.
const plantLight = 238

// Renderer draws worlds. Make one with New and keep it.
type Renderer struct {
	tiles    *[tile.Count]assets.Texture
	faceTile [256][6]tile.ID // the tile on each face of each kind of block
	plant    [256]bool       // the kinds of block that are plants: two crossed pictures, and not a cube
}

// New returns a renderer that draws blocks with the tiles of set.
func New(set *assets.Set) *Renderer {
	r := &Renderer{tiles: &set.Tiles}
	for b := range r.faceTile {
		info := world.Block(b).Info()
		r.faceTile[b] = [6]tile.ID{
			world.FaceXNeg: info.Side, world.FaceXPos: info.Side,
			world.FaceYNeg: info.Bottom, world.FaceYPos: info.Top,
			world.FaceZNeg: info.Side, world.FaceZPos: info.Side,
		}
		r.plant[b] = info.Plant
	}
	return r
}

// cross finds where a ray from eye along d meets a plant in the cell of hit. A plant
// is two pictures standing corner to corner in its cell, crossing in the middle, so
// that it is seen from every side. The ray takes the nearer picture that it meets at a
// pixel which is not a hole: t is how far along the ray that is, and ok is false if it
// passes through the holes of both.
func (r *Renderer) cross(hit world.Hit, eye, d geometry.Vector3, level int) (t float64, pixel uint32, ok bool) {
	// The eye, counted from the corner of the cell.
	ox, oy, oz := eye.X-float64(hit.X), eye.Y-float64(hit.Y), eye.Z-float64(hit.Z)
	side := assets.Side(level)
	picture := r.tiles[r.faceTile[hit.Block][world.FaceXNeg]][level]

	t = math.Inf(1)
	for _, plane := range [2]struct{ num, den float64 }{
		{oz - ox, d.X - d.Z},     // the picture from one corner to the far one: where x is z
		{1 - ox - oz, d.X + d.Z}, // and the one across it: where x and z add up to 1
	} {
		if plane.den == 0 {
			continue // the ray runs along the picture, edge on
		}
		at := plane.num / plane.den
		if !(at > 0) || at >= t {
			continue
		}
		// Across the picture is how far along x the ray meets it, and down it is how
		// far below the top of the cell.
		u, v := ox+d.X*at, 1-(oy+d.Y*at)
		if u < 0 || u >= 1 || v < 0 || v >= 1 {
			continue // the ray meets the plane outside the cell
		}
		if p := picture[int(v*float64(side))*side+int(u*float64(side))]; p>>24 != 0 {
			t, pixel, ok = at, p, true
		}
	}
	return t, pixel, ok
}

// onFace returns where on the face of a block a ray met it at the point p: u across
// and v down, each from 0 to 1, as a tile is laid on the face.
func onFace(hit world.Hit, p geometry.Vector3) (u, v float64) {
	fx, fy, fz := p.X-float64(hit.X), p.Y-float64(hit.Y), p.Z-float64(hit.Z)
	switch hit.Face >> 1 {
	case 0:
		return fz, 1 - fy
	case 1:
		return fx, fz
	}
	return fx, 1 - fy
}

// texel returns the pixel of a block's tile at a place on one of its faces, from the
// tile's copy at the given level, 0 for a hole.
func (r *Renderer) texel(hit world.Hit, u, v float64, level int) uint32 {
	// The place is on the face only to within rounding, so it can be a hair outside.
	side := assets.Side(level)
	ui := min(max(int(u*float64(side)), 0), side-1)
	vi := min(max(int(v*float64(side)), 0), side-1)
	return r.tiles[r.faceTile[hit.Block][hit.Face]][level][vi*side+ui]
}

// sky returns the color of the sky along a ray, given how steeply it climbs (the sine of
// its angle above level ground) and how nearly it points at the sun (the cosine of the
// angle between them).
func sky(climb, toSun float64) color {
	c := skyHorizon.mix(skyZenith, part(climb*1.8))
	switch {
	case toSun > 0.9985:
		return sunColor
	case toSun > 0.985:
		return c.mix(sunColor, part((toSun-0.985)/0.0135*0.5)) // a glow round it
	}
	return c
}

// Draw draws the world as the camera sees it into every pixel of the canvas. Blocks are
// written with their depth and the sky without one, so that it is behind everything.
// The world is drawn as deep as the camera's Far (ViewDistance if it has none) and
// fades by the camera's Fog, if it has one.
func (r *Renderer) Draw(c *engine.Canvas, w *world.World, cam mesh.Camera) {
	v, ok := newRays(cam, c)
	if !ok {
		return
	}
	eye := v.eye
	underwater := w.At(int(math.Floor(eye.X)), int(math.Floor(eye.Y)), int(math.Floor(eye.Z))) == world.Water
	top := float64(w.Height())
	far := cam.Far
	if far == 0 {
		far = ViewDistance
	}
	fog := cam.Fog

	// How far away a tile has to be before a smaller copy of it is used: where its
	// pixels become smaller than one and a half pixels of the canvas.
	pixelsPerBlock := 1 / v.across.Length() // of a block one away, seen face on
	var farther [assets.Levels - 1]float64
	for level := range farther {
		farther[level] = pixelsPerBlock / tile.Size * 1.5 * float64(int(1)<<level)
	}
	levelAt := func(dist float64) int {
		level := 0
		for level < len(farther) && dist > farther[level] {
			level++
		}
		return level
	}

	for py := range c.Height() {
		row := v.first.Add(v.down.Scale(float64(py)))
		for px := range c.Width() {
			d := row.Add(v.across.Scale(float64(px)))
			unit := 1 / d.Length()
			above := sky(d.Y*unit, d.Dot(sunDirection)*unit)

			// A ray is one deep for each 1 along it, so the far plane is where it ends.
			// Above the highest block there is only sky, so it ends there too.
			reach := far
			if d.Y > 0 {
				reach = min(reach, (top-eye.Y)/d.Y)
			}

			ray := world.NewRay(eye, d)
			var surface world.Hit
			var pixel, light uint32
			var depth float64 // of the block or plant the ray stops at, if found
			found, wet := false, false
			for {
				hit, ok := w.Next(&ray, reach)
				if !ok {
					break
				}
				if hit.Block == world.Water {
					// The first water a ray meets is the surface it sees. From under
					// water there is none: the view is tinted as a whole.
					if !wet && !underwater {
						surface, wet = hit, true
					}
					continue
				}
				if r.plant[hit.Block] {
					if at, p, ok := r.cross(hit, eye, d, levelAt(hit.T/unit)); ok {
						pixel, light, depth, found = p, plantLight, at, true
						break
					}
					continue // through the holes of the plant
				}
				fu, fv := onFace(hit, eye.Add(d.Scale(hit.T)))
				if pixel = r.texel(hit, fu, fv, levelAt(hit.T/unit)); pixel>>24 != 0 {
					light, depth, found = faceLight[hit.Face], hit.T, true
					break
				}
			}

			if !found && !wet {
				if underwater {
					above = above.mix(deepWater, 230)
				}
				c.Set(px, py, above.engine())
				continue
			}

			// depth is what is stored for the pixel, and seen is how deep the first
			// thing the ray shows is, which is what fades. They differ under water:
			// water is seen through, so what is stored is the ground under it, and a
			// line drawn on a block down there is not hidden by the surface.
			col, seen := above, depth
			if found {
				col = unpack(pixel).lit(light)
			}
			if wet {
				// Water is drawn over what lies under it, more strongly the more of it
				// the ray went through.
				through := ViewDistance
				if found {
					through = (depth - surface.T) / unit
				}
				su, sv := onFace(surface, eye.Add(d.Scale(surface.T)))
				ripple := unpack(r.texel(surface, su, sv, levelAt(surface.T/unit))).lit(faceLight[surface.Face])
				col = col.mix(ripple, part(0.45+0.12*through)*230>>8)
				seen = surface.T
				if !found {
					depth = surface.T // nothing under it in reach: the water is all there is
				}
			}

			out := col.engine()
			if fog != nil && seen > fog.Start {
				out = fog.Apply(out, seen)
			}
			if underwater {
				red, green, blue := out.RGB()
				tinted := color{uint32(red), uint32(green), uint32(blue)}
				out = tinted.mix(deepWater, part(0.3+0.05*seen/unit)*235>>8).engine()
			}
			c.SetDepth(px, py, depth, out)
		}
	}
}

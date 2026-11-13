// Package render draws the block world into a glonk Canvas, one ray for each pixel.
//
// glonk's meshes are filled one color to a triangle and cannot be seen through, so the
// blocks are not meshes: every pixel sends a ray through the grid and takes the color of
// the tile it lands on. The rays use the same projection as glonk's Camera and write the
// same depth, so anything glonk draws into the Canvas afterwards with a depth (a mesh, a
// wireframe) is hidden by the blocks in front of it and hides those behind.
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

// ViewDistance is how far the world is drawn, in blocks. The last part of it fades into
// the sky.
const ViewDistance = 64.0

// fogStart is the part of ViewDistance that is clear, before the fade begins.
const fogStart = 0.6

// faceLight is how bright each face of a block is drawn, out of 256, indexed by
// world.Face: the top is fully lit, the sides less, and the bottom least. It is what
// gives the blocks their shape, as there are no shadows.
var faceLight = [6]uint32{205, 205, 128, 256, 166, 166}

// color is a color with a channel in each field, from 0 to 255, for mixing.
type color struct{ r, g, b uint32 }

var (
	skyHorizon = color{190, 220, 245}
	skyZenith  = color{70, 130, 220}
	sunColor   = color{255, 244, 200}
	deepWater  = color{24, 60, 130}
	outline    = color{255, 214, 64} // round the block the player aims at
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

// View is a camera worked out for a canvas of one size: where each pixel looks.
type View struct {
	eye             geometry.Vector3
	look, right, up geometry.Vector3
	w, h            int
	kx, ky          float64 // how far a ray leans for each pixel away from the middle
}

// NewView returns the view of a camera on a canvas w pixels wide and h tall. ok is false
// for a camera that sees nothing. A FOV or CellAspect of 0 means what it means to glonk:
// a quarter turn, and cells twice as tall as wide.
func NewView(cam mesh.Camera, w, h int) (v View, ok bool) {
	fov, aspect := cam.FOV, cam.CellAspect
	if fov == 0 {
		fov = math.Pi / 2
	}
	if aspect == 0 {
		aspect = 2
	}
	if !(fov > 0 && fov < math.Pi && aspect > 0) || math.IsInf(aspect, 0) || w <= 0 || h <= 0 {
		return View{}, false
	}

	// glonk puts a point at cell x = w/2 + focal*vx/vz * cells/2 * aspect and cell
	// y = cells/2 - focal*vy/vz * cells/2, where cells is the height in cells, half the
	// height in pixels. Turned round, that is how far a ray leans at each pixel.
	focal := 1 / math.Tan(fov/2)
	cells := float64(h) / 2
	look, right := cam.Look(), cam.Right()
	return View{
		eye:  cam.Position,
		look: look, right: right, up: look.Cross(right),
		w: w, h: h,
		kx: 1 / (focal * cells / 2 * aspect),
		ky: 1 / (focal * cells),
	}, true
}

// Direction returns the way the ray through a point of the canvas runs. x and y are in
// pixels and can fall between them: the middle of pixel (3, 5) is (3.5, 5.5). The
// direction is not one long. It reaches one block deep into the view, so a point t along
// it is at the depth t that glonk gives a mesh there.
func (v View) Direction(x, y float64) geometry.Vector3 {
	sx := (x - float64(v.w)/2) * v.kx
	sy := (float64(v.h)/2 - y) * v.ky
	return v.look.Add(v.right.Scale(sx)).Add(v.up.Scale(sy))
}

// onOutline reports whether the ray along d, which met a block at (fu, fv) on one of
// its faces, is on the outline of that face: a band along each edge, one pixel wide and
// never thinner than a twentieth of the block.
//
// A pixel covers more of a face the shallower the angle the face is seen at, a long
// strip of it in the end, so the width of the band is worked out on the face itself: it
// is how far across the face the rays through the next pixel over and the next pixel
// down land. A band that wide always has a pixel in it, so no edge goes missing.
func (v View) onOutline(hit world.Hit, d geometry.Vector3, fu, fv float64) bool {
	axis := hit.Face >> 1
	plane := float64([3]int{hit.X, hit.Y, hit.Z}[axis] + int(hit.Face&1))
	eye := [3]float64{v.eye.X, v.eye.Y, v.eye.Z}[axis]

	bandU, bandV := 0.05, 0.05
	for _, next := range [2]geometry.Vector3{d.Add(v.right.Scale(v.kx)), d.Sub(v.up.Scale(v.ky))} {
		t := (plane - eye) / [3]float64{next.X, next.Y, next.Z}[axis]
		u, w := onFace(hit, v.eye.Add(next.Scale(t)))
		// A quarter over, as the rays spread and the next pixel is not quite as far
		// along the face everywhere.
		bandU, bandV = max(bandU, 1.25*math.Abs(u-fu)), max(bandV, 1.25*math.Abs(w-fv))
	}
	bandU, bandV = min(bandU, 0.5), min(bandV, 0.5)
	return fu < bandU || fu > 1-bandU || fv < bandV || fv > 1-bandV
}

// Renderer draws worlds. Make one with New and keep it.
type Renderer struct {
	tiles    *[tile.Count]assets.Texture
	faceTile [256][6]tile.ID // the tile on each face of each kind of block
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
	}
	return r
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
//
// aim, if it is not nil, is a block to mark: the edges of its faces are drawn as an
// outline. The rays draw it themselves. A wireframe drawn by glonk with a depth lies on
// the very faces it outlines, and comes out in dashes wherever a face is seen at a
// shallow angle.
func (r *Renderer) Draw(c *engine.Canvas, w *world.World, cam mesh.Camera, aim *world.Hit) {
	v, ok := NewView(cam, c.Width(), c.Height())
	if !ok {
		return
	}
	eye := v.eye
	underwater := w.At(int(math.Floor(eye.X)), int(math.Floor(eye.Y)), int(math.Floor(eye.Z))) == world.Water
	top := float64(w.Height())

	// How far away a tile has to be before a smaller copy of it is used: where its
	// pixels become smaller than one and a half pixels of the canvas.
	pixelsPerBlock := 1 / v.kx // of a block one away, seen face on
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

	for py := range v.h {
		sy := (float64(v.h)/2 - (float64(py) + 0.5)) * v.ky
		row := v.look.Add(v.up.Scale(sy))
		for px := range v.w {
			sx := (float64(px) + 0.5 - float64(v.w)/2) * v.kx
			d := row.Add(v.right.Scale(sx))
			unit := 1 / math.Sqrt(1+sx*sx+sy*sy) // 1 / the length of d
			above := sky(d.Y*unit, d.Dot(sunDirection)*unit)

			// Above the highest block there is only sky, so a ray stops there.
			reach := ViewDistance * unit
			if d.Y > 0 {
				reach = min(reach, (top-eye.Y)/d.Y)
			}

			ray := world.NewRay(eye, d)
			var solid, surface world.Hit
			var pixel uint32
			found, wet, outlined := false, false, false
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
				dist := hit.T / unit
				fu, fv := onFace(hit, eye.Add(d.Scale(hit.T)))
				if aim != nil && hit.X == aim.X && hit.Y == aim.Y && hit.Z == aim.Z && v.onOutline(hit, d, fu, fv) {
					solid, found, outlined = hit, true, true
					break
				}
				if pixel = r.texel(hit, fu, fv, levelAt(dist)); pixel>>24 != 0 {
					solid, found = hit, true
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

			col, depth := above, solid.T
			switch {
			case outlined:
				col = outline
			case found:
				col = unpack(pixel).lit(faceLight[solid.Face])
			}
			if wet {
				// Water is drawn over what lies under it, more strongly the more of it
				// the ray went through.
				through := ViewDistance
				if found {
					through = (solid.T - surface.T) / unit
				}
				su, sv := onFace(surface, eye.Add(d.Scale(surface.T)))
				ripple := unpack(r.texel(surface, su, sv, levelAt(surface.T/unit))).lit(faceLight[surface.Face])
				col = col.mix(ripple, part(0.45+0.12*through)*230>>8)
				depth = surface.T
			}
			dist := depth / unit
			col = col.mix(above, part((dist/ViewDistance-fogStart)/(1-fogStart)))
			if underwater {
				col = col.mix(deepWater, part(0.3+0.05*dist)*235>>8)
			}
			c.SetDepth(px, py, depth, col.engine())
		}
	}
}

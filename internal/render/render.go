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

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/tile"
	"github.com/kyrokohan/skyspear/internal/world"
)

// ViewDistance is how deep into the view the world is drawn, in blocks: the Far of the
// camera the game uses. It is also how deep the rays go for a camera with no Far, since
// a ray has to end somewhere.
const ViewDistance = 64.0

// faceLight is how bright each face of a block is drawn, out of 256, indexed by
// world.Face: the top is fully lit, the sides less, and the bottom least. It is what
// gives the blocks their shape, as there are no shadows.
var faceLight = [6]uint32{205, 205, 128, 256, 166, 166}

// color is a color with a channel in each field, from 0 to 255, for mixing. The pixel
// loop mixes whole numbers with it, which is quicker there than engine.Mix on packed
// colors.
type color struct{ r, g, b uint32 }

// mix returns the color part of the way from c to o, where part runs from 0 (c) to 256 (o).
func (c color) mix(o color, part uint32) color {
	keep := 256 - part
	return color{(c.r*keep + o.r*part) >> 8, (c.g*keep + o.g*part) >> 8, (c.b*keep + o.b*part) >> 8}
}

// lit returns the color at a brightness out of 256.
func (c color) lit(light uint32) color {
	return color{c.r * light >> 8, c.g * light >> 8, c.b * light >> 8}
}

// times returns the color lit by a light with a brightness out of 256 in each channel.
func (c color) times(light color) color {
	return color{c.r * light.r >> 8, c.g * light.g >> 8, c.b * light.b >> 8}
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

// plantLight is how bright a plant is drawn, out of 256, in the light of its cell:
// nearly as bright as the top of a block.
const plantLight = 238

// Renderer draws worlds. Make one with New and keep it.
type Renderer struct {
	// Crack is the block being broken, if any, which is drawn with cracks on its faces.
	Crack Crack

	// Frost is how white the tops of blocks that are open to the sky are drawn, from 0,
	// as they are, to 1: the ground of a winter. No block is changed by it.
	Frost float64

	tiles    *[tile.Count]assets.Texture
	faceTile [256][6]tile.ID  // the tile on each face of each kind of block
	plant    [256]bool        // the kinds of block that are plants: two crossed pictures, and not a cube
	mat      [256]bool        // the kinds of block that are mats: one picture lying flat
	glow     [256]world.Light // the light each kind of block gives off
	shuts    [256]bool        // the kinds of block that cast the soft shadows where blocks meet

	wet []wetPixel // the pixels of the last Draw that show water, for Wash
}

// sunkPart is how much of the water's color lies over what an eye under water sees
// through so many blocks of it, out of 256.
func sunkPart(blocks float64) uint32 {
	thick := part(0.3+0.05*blocks) * 235 >> 8
	if blocks > underClear {
		thick += uint32((256 - 235) * min((blocks-underClear)/(underMurk-underClear), 1))
	}
	return thick
}

// wetPixel is a pixel in which Draw saw water: how deep the top of the water is there,
// the color of that water as it is lit and faded, and what Draw left in the pixel. From
// an eye that is under water every pixel is one, and its top is under 0: the water
// begins at the eye.
type wetPixel struct {
	x, y  int
	top   float64
	water color
	drew  engine.Color
}

// Water is seen into, and not through: what lies under it shows less the more water a
// ray went through to reach it, least at clearTo blocks of it, and from there the water
// thickens until, at murk blocks, it is all there is. A ray goes no farther into it, so
// a view over deep sea costs no more than one over a pond.
//
// An eye under water sees farther in it, as one that looks along the water and not down
// into it: all that the water round it lets through at underClear blocks, and nothing
// at underMurk.
const (
	clearTo    = (1 - 0.45) / 0.12
	murk       = 10.0
	underClear = (1 - 0.3) / 0.05
	underMurk  = 20.0
)

// washPart is how much of the water's color lies over what Wash finds under the water,
// out of 256: more than over the floor of a pool one deep, so that what lies in a pool
// is a dim shape in it.
const washPart = 180

// bogWater is the color of water that stands on peat.
var bogWater = color{24, 22, 20}

// frostWhite is the color of frost, and frostMost how much of a face's own color the most
// frost hides: the ground still shows through it.
var frostWhite = color{236, 242, 250}

const frostMost = 0.78

// Crack is a block that is being broken, and how far the breaking has gone, from 0, not
// at all, to 1, broken. The cracks grow out from the middle of each face as it goes.
type Crack struct {
	X, Y, Z int
	Part    float64
}

// crackShade is how bright a crack is, out of 256, of what is under it.
const crackShade = 64

// cracked returns a pixel of a face of the block being broken, with the cracks on it.
func (r *Renderer) cracked(pixel uint32, u, v float64, level int) uint32 {
	side := assets.Side(level)
	ui := min(max(int(u*float64(side)), 0), side-1)
	vi := min(max(int(v*float64(side)), 0), side-1)
	crack := r.tiles[tile.Cracks][level][vi*side+ui]
	if crack>>24 == 0 || float64(crack>>16&0xFF) > r.Crack.Part*255 {
		return pixel // no crack here, or not yet
	}
	c := unpack(pixel).lit(crackShade)
	return 0xFF000000 | c.r<<16 | c.g<<8 | c.b
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
		r.mat[b] = info.Mat
		r.glow[b] = world.Light(min(info.Glow, world.MaxLight))
		r.shuts[b] = info.Solid && !info.Stands() && info.Shade == world.MaxLight
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

// matHeight is how far above the floor of its cell a mat lies, in blocks.
const matHeight = 1.0 / 16

// flat finds where a ray from eye along d meets a mat in the cell of hit: its top tile,
// lying a little above the floor of the cell, the way a block's top is laid. ok is false
// if the ray misses it, or meets it at a hole.
func (r *Renderer) flat(hit world.Hit, eye, d geometry.Vector3, level int) (t float64, pixel uint32, ok bool) {
	if d.Y == 0 {
		return 0, 0, false // the ray runs along the mat, edge on
	}
	t = (float64(hit.Y) + matHeight - eye.Y) / d.Y
	u, v := eye.X+d.X*t-float64(hit.X), eye.Z+d.Z*t-float64(hit.Z)
	if !(t > 0) || u < 0 || u >= 1 || v < 0 || v >= 1 {
		return 0, 0, false
	}
	side := assets.Side(level)
	pixel = r.tiles[r.faceTile[hit.Block][world.FaceYPos]][level][int(v*float64(side))*side+int(u*float64(side))]
	return t, pixel, pixel>>24 != 0
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

// lightOn returns the light that falls on the face of a block a ray met, in each
// channel out of 256: that of the cell the face looks into, or the block's own glow if
// that is more, shaded for the way the face points.
func (r *Renderer) lightOn(w *world.World, sky *Sky, hit world.Hit) color {
	nx, ny, nz := hit.Face.Normal()
	l := w.LightAt(hit.X+nx, hit.Y+ny, hit.Z+nz)
	if g := r.glow[hit.Block]; g > l&world.MaxLight {
		l = l&^world.MaxLight | g
	}
	return sky.shade[l].lit(faceLight[hit.Face])
}

// The soft shadows where blocks meet. A face is darker where the cells beside the one in
// front of it are shut, and darkest in a corner shut on both sides, as less of the open
// air round it reaches it there. It is worked out at the four corners of a face and
// blended across it, and only on faces near enough for it to be seen: it fades out from
// shadeFrom to shadeTo blocks away.
var cornerShade = [4]uint32{150, 185, 220, 256} // out of 256, by how many of the three cells round a corner are open

const shadeFrom, shadeTo = 24.0, 32.0

// faceAxes are, for the faces across x, y and z, the ways onFace measures u and v along:
// a step of one along each, as a step from cell to cell.
var faceAxes = [3][2][3]int{
	{{0, 0, 1}, {0, -1, 0}}, // u along z, and v down
	{{1, 0, 0}, {0, 0, 1}},
	{{1, 0, 0}, {0, -1, 0}},
}

// corners returns how lit each corner of the face a ray met is by the soft shadows, out
// of 256: the corner where u and v are 0, then where u is 1, then v, then both.
func (r *Renderer) corners(w *world.World, hit world.Hit) (c [4]uint32) {
	nx, ny, nz := hit.Face.Normal()
	fx, fy, fz := hit.X+nx, hit.Y+ny, hit.Z+nz // the cell in front of the face
	u, v := faceAxes[hit.Face>>1][0], faceAxes[hit.Face>>1][1]
	shut := func(du, dv int) int {
		if r.shuts[w.At(fx+du*u[0]+dv*v[0], fy+du*u[1]+dv*v[1], fz+du*u[2]+dv*v[2])] {
			return 1
		}
		return 0
	}
	for i, d := range [4][2]int{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
		side, other := shut(d[0], 0), shut(0, d[1])
		open := 3 - side - other - shut(d[0], d[1])
		if side+other == 2 {
			open = 0 // shut on both sides, the corner is as dark whatever is in it
		}
		c[i] = cornerShade[open]
	}
	return c
}

// blend returns the shade at a place on a face from the shades of its corners.
func blend(c [4]uint32, u, v float64) uint32 {
	u, v = min(max(u, 0), 1), min(max(v, 0), 1)
	near := float64(c[0])*(1-u) + float64(c[1])*u
	far := float64(c[2])*(1-u) + float64(c[3])*u
	return uint32(near*(1-v) + far*v)
}

// Draw draws the world as the camera sees it into every pixel of the canvas. Blocks are
// written with their depth and the sky without one, so that it is behind everything.
// The world is drawn as deep as the camera's Far (ViewDistance if it has none), and fades
// by the camera's Fog, if it has one: the game's camera has the sky's. The sky is the
// one at the time of day, which also says how bright the light of each cell looks.
func (r *Renderer) Draw(c *engine.Canvas, w *world.World, cam mesh.Camera, sky *Sky) {
	r.wet = r.wet[:0]
	v, ok := newRays(cam, c)
	if !ok {
		return
	}
	eye := v.eye
	ex, ey, ez := int(math.Floor(eye.X)), int(math.Floor(eye.Y)), int(math.Floor(eye.Z))
	underwater := w.At(ex, ey, ez) == world.Water
	over := ey // the highest block of the water the eye is in
	for underwater && w.At(ex, over+1, ez) == world.Water {
		over++
	}
	deep := deepWater.times(sky.shade[w.LightAt(ex, ey, ez)]) // the water round the eye, as lit there
	far := cam.Far
	if far == 0 {
		far = ViewDistance
	}
	// Above the highest block that is near enough to be seen there is only sky.
	top := float64(w.HeightNear(eye.X, eye.Z, min(far, 4*ViewDistance)))
	fog := cam.Fog

	// How far away a tile has to be before a smaller copy of it is used: where its
	// pixels become smaller than one and a half pixels of the canvas.
	pixelsPerBlock := 1 / v.across.Length() // of a block one away, seen face on
	var farther [assets.Levels - 1]float64
	for level := range farther {
		farther[level] = pixelsPerBlock / tile.Size * 1.5 * float64(int(1)<<level)
	}
	var frost uint32 // how much of the frost is on the face the ray of this pixel met, out of 256
	var bog bool     // that face is peat: water over it is black
	// The face whose corners were last worked out, as the next pixel is likely on it too.
	var shadedAt world.Hit
	var shades [4]uint32
	shaded := false

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
			above := sky.color(d.Scale(unit))

			// A ray is one deep for each 1 along it, so the far plane is where it ends.
			// Above the highest block there is only sky, so it ends there too.
			reach := far
			if d.Y > 0 {
				reach = min(reach, (top-eye.Y)/d.Y)
			}
			// From under water a ray that does not come up out of it soon ends in it.
			sunk := underwater && (d.Y <= 0 || (float64(over+1)-eye.Y)/d.Y > underMurk*unit)
			if sunk {
				reach = min(reach, underMurk*unit)
			}

			ray := world.NewRay(eye, d)
			var surface world.Hit
			var pixel uint32
			var light color
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
						reach = min(reach, hit.T+murk*unit) // past that the water hides everything
					}
					continue
				}
				if r.plant[hit.Block] {
					if at, p, ok := r.cross(hit, eye, d, levelAt(hit.T/unit)); ok {
						pixel, depth, found = p, at, true
						light = sky.shade[w.LightAt(hit.X, hit.Y, hit.Z)].lit(plantLight)
						break
					}
					continue // through the holes of the plant
				}
				if r.mat[hit.Block] {
					if at, p, ok := r.flat(hit, eye, d, levelAt(hit.T/unit)); ok {
						pixel, depth, found = p, at, true
						light = sky.shade[w.LightAt(hit.X, hit.Y, hit.Z)].lit(plantLight)
						break
					}
					continue // over it, or through its holes
				}
				fu, fv := onFace(hit, eye.Add(d.Scale(hit.T)))
				level := levelAt(hit.T / unit)
				if pixel = r.texel(hit, fu, fv, level); pixel>>24 != 0 {
					if crack := r.Crack; crack.Part > 0 && hit.X == crack.X && hit.Y == crack.Y && hit.Z == crack.Z {
						pixel = r.cracked(pixel, fu, fv, level)
					}
					light, depth, found = r.lightOn(w, sky, hit), hit.T, true
					frost, bog = 0, hit.Block == world.Peat
					if r.Frost > 0 && hit.Face == world.FaceYPos && r.glow[hit.Block] == 0 && w.LightAt(hit.X, hit.Y+1, hit.Z).Sky() == world.MaxLight {
						frost = part(frostMost * r.Frost)
					}
					if dist := hit.T / unit; dist < shadeTo {
						if !shaded || hit.X != shadedAt.X || hit.Y != shadedAt.Y || hit.Z != shadedAt.Z || hit.Face != shadedAt.Face {
							shades, shadedAt, shaded = r.corners(w, hit), hit, true
						}
						a := blend(shades, fu, fv)
						if dist > shadeFrom {
							a += uint32(float64(256-a) * (dist - shadeFrom) / (shadeTo - shadeFrom))
						}
						light = light.lit(a)
					}
					break
				}
			}

			if !found && !wet {
				switch {
				case sunk:
					// The water is all there is, and hides what is deeper in it.
					above = deep
					c.SetDepth(px, py, underMurk*unit, above.engine())
				case underwater:
					// The sky, through the water over the eye: as much of it as that
					// much water lets through, so that it meets the murk with no seam.
					above = above.mix(deep, sunkPart((float64(over+1)-eye.Y)/d.Y/unit))
					c.Set(px, py, above.engine())
				default:
					c.Set(px, py, above.engine())
				}
				if underwater {
					r.wet = append(r.wet, wetPixel{x: px, y: py, top: -1, water: deep, drew: above.engine()})
				}
				continue
			}

			// depth is what is stored for the pixel, and seen is how deep the first
			// thing the ray shows is, which is what fades. They differ under water:
			// water is seen through, so what is stored is the ground under it, and a
			// line drawn on a block down there is not hidden by the surface.
			col, seen := above, depth
			if found {
				col = unpack(pixel).mix(frostWhite, frost).times(light)
			}
			var water color // the water the ray went through, as it is lit
			if wet {
				// Water is drawn over what lies under it, more strongly the more of it
				// the ray went through.
				through := ViewDistance
				if found {
					through = (depth - surface.T) / unit
				}
				su, sv := onFace(surface, eye.Add(d.Scale(surface.T)))
				ripple := unpack(r.texel(surface, su, sv, levelAt(surface.T/unit)))
				if found && bog {
					ripple = ripple.mix(bogWater, 190) // the pools of a swamp are black
				}
				ripple = ripple.times(r.lightOn(w, sky, surface))
				water = ripple
				thick := part(0.45+0.12*through) * 230 >> 8
				if through > clearTo {
					thick += uint32((256 - 230) * min((through-clearTo)/(murk-clearTo), 1))
				}
				col = col.mix(ripple, thick)
				seen = surface.T
				if !found {
					// Nothing under it in reach: the water is all there is, and what is
					// drawn deeper in it than a ray sees is hidden.
					depth = surface.T + murk*unit
				}
			}

			out := col.engine()
			if fog != nil && seen > fog.Start {
				out = fog.Apply(out, seen)
			}
			if underwater {
				red, green, blue := out.RGB()
				tinted := color{uint32(red), uint32(green), uint32(blue)}
				out = tinted.mix(deep, sunkPart(seen/unit)).engine()
				r.wet = append(r.wet, wetPixel{x: px, y: py, top: -1, water: deep, drew: out})
			}
			c.SetDepth(px, py, depth, out)
			if wet {
				if fog != nil && seen > fog.Start {
					red, green, blue := fog.Apply(water.engine(), seen).RGB()
					water = color{uint32(red), uint32(green), uint32(blue)}
				}
				r.wet = append(r.wet, wetPixel{x: px, y: py, top: surface.T, water: water, drew: out})
			}
		}
	}
}

// Wash draws the water over what has been drawn under its top since Draw: a beast that
// lies in a pool is seen through the water, as the floor of the pool is. What is over the
// water, or in front of it, stays as it is. From an eye that is under water all that has
// been drawn is in the water, or seen through it. Call it after the solid things that
// may be in water, and before anything that is seen through.
func (r *Renderer) Wash(c *engine.Canvas) {
	for i := range r.wet {
		p := &r.wet[i]
		now := c.At(p.x, p.y)
		if now == p.drew {
			continue // nothing has been drawn here
		}
		red, green, blue := now.RGB()
		washed := color{uint32(red), uint32(green), uint32(blue)}.mix(p.water, washPart).engine()
		switch {
		case p.top < 0:
			c.Set(p.x, p.y, washed)
			p.drew = washed
		case c.SetDepth(p.x, p.y, p.top, washed):
			// The top of the water is nearer than what is under it, and than nothing
			// else: the depth says which this is.
			p.drew = washed
		default:
			p.drew = now // it is over the water, or in front of it
		}
	}
}

package render

import (
	"math"
	"sync"

	"glonk.dev/bitmap"
	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/world"
)

// Paintings on the faces of rock. A picture is a mask: how much paint each pixel of a
// face has. The masks are drawn by code when the renderer is made, twice as fine as a
// tile, so that the line of a painting is finer than the grain of the stone under it,
// with smaller copies for faces far off, as a tile has.

// maskSide is how many pixels across the largest copy of a picture is.
const maskSide = 32

// mask is a picture in every size, the largest first: how much paint each pixel has, out
// of 255. Rows run from the top.
type mask [assets.Levels][]uint8

// sprayReach is how far out from a hand, in pixels of the largest mask, the paint blown
// round it reaches.
const sprayReach = 6.0

// masks are every picture, drawn once: they are the same for every renderer.
var (
	masks     [world.Pictures]mask
	masksOnce sync.Once
)

// makeMasks returns every picture: the hand, the spiral and the comet drawn by code, and
// each kind of beast from its own body, as Beasts draws it.
func makeMasks(set *assets.Set) [world.Pictures]mask {
	masksOnce.Do(func() {
		masks[world.Hand] = drawMask(handPaint)
		masks[world.Spiral] = drawMask(spiralPaint)
		masks[world.Comet] = drawMask(cometPaint)
		beasts := NewBeasts(set)
		for k := range Kinds {
			masks[world.FirstBeast+world.Picture(k)] = drawMask(silhouette(beasts, k))
		}
	})
	return masks
}

// drawMask makes a mask from how much paint there is at each point of a face, from 0 to
// 1, where x and y run from 0 to maskSide; and its smaller copies, each pixel the mean of
// the four it covers.
func drawMask(paint func(x, y float64) float64) (m mask) {
	m[0] = make([]uint8, maskSide*maskSide)
	for i := range m[0] {
		x, y := float64(i%maskSide)+0.5, float64(i/maskSide)+0.5
		m[0][i] = uint8(math.Round(255 * min(max(paint(x, y), 0), 1)))
	}
	for level := 1; level < assets.Levels; level++ {
		side, big := maskSide>>level, m[level-1]
		m[level] = make([]uint8, side*side)
		for i := range m[level] {
			at := 2*(i/side)*2*side + 2*(i%side) // the top left of the four
			m[level][i] = uint8((int(big[at]) + int(big[at+1]) + int(big[at+2*side]) + int(big[at+2*side+1])) / 4)
		}
	}
	return m
}

// handPaint is a stencil of a hand, as it is made in caves: paint blown round a hand held
// flat on the stone, thick at its edge and thinning out, and none on the hand.
func handPaint(x, y float64) float64 {
	d := handEdge(x, y)
	if d <= 0 {
		return 0
	}
	return min(1.4*(1-d/sprayReach), 1) * (0.8 + 0.2*speck(x, y)) // in grains, as blown paint is
}

// handEdge is how far a point is from the edge of a hand, fingers up: less than 0 on it.
func handEdge(x, y float64) float64 {
	d := (math.Hypot((x-16)/5.2, (y-19.5)/5.8) - 1) * 5.2 // the palm
	for _, f := range [...][5]float64{
		{12.6, 15, 11.6, 6.5, 1.5},  // the fingers, from the palm to their tips
		{15.6, 14, 15.4, 4.5, 1.5},  //
		{18.6, 14.5, 19.4, 6, 1.5},  //
		{20.6, 17, 22.8, 10, 1.4},   //
		{11.5, 20, 6.8, 14, 1.7},    // the thumb
		{16, 24, 16, maskSide, 4.2}, // the wrist, to the edge of the face
	} {
		d = min(d, capsule(x, y, f[0], f[1], f[2], f[3], f[4]))
	}
	return d
}

// capsule is how far a point is from the edge of a line from (ax, ay) to (bx, by) that
// is r thick on each side: less than 0 in it.
func capsule(x, y, ax, ay, bx, by, r float64) float64 {
	dx, dy := bx-ax, by-ay
	t := min(max(((x-ax)*dx+(y-ay)*dy)/(dx*dx+dy*dy), 0), 1)
	return math.Hypot(x-ax-t*dx, y-ay-t*dy) - r
}

// spiralPaint is a line of paint that winds three times out from the middle of a face.
func spiralPaint(x, y float64) float64 {
	const turns, outer, half, soft = 3.0, 13.0, 1.4, 0.6
	dx, dy := x-maskSide/2, y-maskSide/2
	r, a := math.Hypot(dx, dy), math.Atan2(dy, dx)
	if a < 0 {
		a += 2 * math.Pi
	}
	near := math.Inf(1) // how far the point is from the nearest turn
	for k := range int(turns) {
		near = min(near, math.Abs(r-outer/turns*(a/(2*math.Pi)+float64(k))))
	}
	return (half + soft - near) / soft
}

// cometPaint is the Skyspear: a round head high on the left, and a tail that widens and
// fades down to the right.
func cometPaint(x, y float64) float64 {
	const head, long = 3.5, 22.0
	dx, dy := x-9, y-9
	if math.Hypot(dx, dy) < head {
		return 1
	}
	along, across := (dx+dy)/math.Sqrt2, math.Abs(dx-dy)/math.Sqrt2
	if along < 0 || along > long {
		return 0
	}
	wide := 2 + 4*along/long
	return min((wide-across)/0.8, 1) * (1 - 0.5*along/long)
}

// shapeSide is how many pixels across a beast is drawn before it is made a mask: enough
// that a pixel of the mask is the mean of many.
const shapeSide = 128

// shapeMargin is how many pixels of the mask are left bare round a beast.
const shapeMargin = 2

// silhouette draws a kind of beast side on and far off, and returns how much of each
// point of a mask its body covers, as a solid shape: as large as it fits, in the middle.
func silhouette(b *Beasts, k Kind) func(x, y float64) float64 {
	fb, err := engine.NewFrameBuffer(shapeSide, shapeSide/2)
	if err != nil {
		panic(err) // the size is fixed and fine
	}
	c := fb.Canvas()
	width, height := Size(k, false)
	back, front := Span(k, false)
	long := max(back+front, height, width)
	var x0, y0, x1, y1 int
	// Some of a beast can reach past its box, a neck or a tail: one that runs off the
	// edge of the view is drawn again from twice as far.
	for try := 0; try < 4; try, long = try+1, 2*long {
		far := 10 * long
		cam := mesh.Camera{Position: geometry.Vector3{Y: height / 2, Z: -far}, FOV: 2 * math.Atan(0.75*long/far), Far: 2 * far}
		c.Clear()
		b.Draw(c, cam, []Beast{{Kind: k, Yaw: math.Pi / 2, Light: 1}}, NewSky(Noon))
		x0, y0, x1, y1 = shapeSide, shapeSide, 0, 0 // the box round what was drawn
		for y := range shapeSide {
			for x := range shapeSide {
				if !c.At(x, y).IsDefault() {
					x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x+1), max(y1, y+1)
				}
			}
		}
		if x0 > 0 && y0 > 0 && x1 < shapeSide && y1 < shapeSide {
			break
		}
	}
	cover := make([]float64, maskSide*maskSide)
	if x1 <= x0 {
		return func(x, y float64) float64 { return 0 }
	}
	scale := float64(maskSide-2*shapeMargin) / float64(max(x1-x0, y1-y0)) // pixels of the mask to one drawn
	ox := (maskSide - scale*float64(x1-x0)) / 2
	oy := (maskSide - scale*float64(y1-y0)) / 2
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if !c.At(x, y).IsDefault() {
				mx, my := int(ox+scale*(float64(x-x0)+0.5)), int(oy+scale*(float64(y-y0)+0.5))
				cover[my*maskSide+mx] += scale * scale
			}
		}
	}
	// A pixel of the mask half covered is painted in full, so that the shape is solid.
	return func(x, y float64) float64 { return 2 * cover[int(y)*maskSide+int(x)] }
}

// speck is a number from 0 to 1 that is the same for a pixel each time and unlike its
// neighbours': the grains of paint.
func speck(x, y float64) float64 {
	h := uint32(int(x))*73856093 ^ uint32(int(y))*19349663
	h ^= h >> 13
	h *= 0x5bd1e995
	h ^= h >> 15
	return float64(h&0xFF) / 255
}

// paintColor is the color a paint goes on in.
func paintColor(paint world.Item) color {
	switch paint {
	case world.RedPaint:
		return color{164, 66, 38}
	case world.BlackPaint:
		return color{38, 34, 32}
	}
	return color{230, 226, 212} // chalk
}

// painted returns a pixel of the stone of a face, at a place on it, with the painting on
// the face over it, from the picture's copy at the given level. The paint takes the grain
// of the stone, darker where the stone is dark; the face's light falls on both after.
func (r *Renderer) painted(pixel uint32, p world.Painting, u, v float64, level int) uint32 {
	side := maskSide >> level
	ui := min(max(int(u*float64(side)), 0), side-1)
	vi := min(max(int(v*float64(side)), 0), side-1)
	a := uint32(r.masks[p.Picture][level][vi*side+ui])
	if a == 0 {
		return pixel
	}
	stone := unpack(pixel)
	grain := (stone.r*77 + stone.g*150 + stone.b*29) >> 8
	c := stone.mix(paintColor(p.Paint).lit(128+grain/2), a+a>>7) // 255 is all paint
	return 0xFF000000 | c.r<<16 | c.g<<8 | c.b
}

// PictureSide is how many pixels across Picture makes a painting when it is small: for
// the picker, which shows many. A painting's card shows it twice as large.
const PictureSide = maskSide / 2

// Picture returns a painting as it looks on a kind of block, face on and with no light
// on it, PictureSide pixels across or, large, twice that.
func (r *Renderer) Picture(p world.Painting, on world.Block, large bool) *bitmap.Bitmap {
	side, level := PictureSide, 1
	if large {
		side, level = maskSide, 0
	}
	pic := bitmap.New(side, side)
	hit := world.Hit{Block: on, Face: p.Face}
	for i := range side * side {
		x, y := i%side, i/side
		u, v := (float64(x)+0.5)/float64(side), (float64(y)+0.5)/float64(side)
		pic.Set(x, y, unpack(r.painted(r.texel(hit, u, v, 0), p, u, v, level)).engine())
	}
	return pic
}

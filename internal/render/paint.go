package render

import (
	"math"

	"glonk.dev/bitmap"

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

// makeMasks draws every picture.
func makeMasks() (m [world.Pictures]mask) {
	m[world.Hand] = drawMask(handPaint)
	m[world.Spiral] = drawMask(spiralPaint)
	return m
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

// PictureSide is how many pixels across Picture makes a painting.
const PictureSide = maskSide / 2

// Picture returns a painting as it looks on a kind of block, face on and with no light
// on it: for the picker, which shows what can be painted.
func (r *Renderer) Picture(p world.Painting, on world.Block) *bitmap.Bitmap {
	pic := bitmap.New(PictureSide, PictureSide)
	hit := world.Hit{Block: on, Face: p.Face}
	for i := range PictureSide * PictureSide {
		x, y := i%PictureSide, i/PictureSide
		u, v := (float64(x)+0.5)/PictureSide, (float64(y)+0.5)/PictureSide
		pic.Set(x, y, unpack(r.painted(r.texel(hit, u, v, 0), p, u, v, 1)).engine())
	}
	return pic
}

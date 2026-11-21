package render

import (
	"testing"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/world"
)

// paintedAt returns a painting of a picture in red ochre on the side of the block at x, y,
// z that faces the camera.
func paintedAt(x, y, z int, p world.Picture) world.Painting {
	return world.Painting{Side: world.Side{Cell: world.Cell{X: x, Y: y, Z: z}, Face: world.FaceZNeg}, Picture: p, Paint: world.RedPaint}
}

// coverage is the part of a picture's largest mask that has paint on it.
func coverage(m mask) float64 {
	n := 0
	for _, a := range m[0] {
		if a > 0 {
			n++
		}
	}
	return float64(n) / float64(len(m[0]))
}

// The paint goes on only where the picture is: elsewhere the stone is as it was, and
// where the picture is whole the stone takes the paint's color.
func TestPaintIsDrawnOnlyWhereThePictureIs(t *testing.T) {
	r := newRenderer(t)
	for p := range world.Pictures {
		for level := range assets.Levels {
			m, side := r.masks[p][level], maskSide>>level
			if len(m) != side*side {
				t.Fatalf("the %s at level %d has %d pixels, wanted %d", p.Name(), level, len(m), side*side)
			}
			for i, a := range m {
				u, v := (float64(i%side)+0.5)/float64(side), (float64(i/side)+0.5)/float64(side)
				stone := r.texel(world.Hit{Block: world.Stone, Face: world.FaceZNeg}, u, v, level)
				got := r.painted(stone, paintedAt(0, 0, 0, p), u, v, level)
				switch {
				case a == 0 && got != stone:
					t.Fatalf("the %s at level %d painted %x at %d, where it has no paint", p.Name(), level, got, i)
				case a == 255:
					c, was := unpack(got), unpack(stone)
					if int(c.r)-int(c.g) <= max(int(was.r)-int(was.g), 0) {
						t.Fatalf("the %s at level %d left %x at %d, wanted it red: the stone was %x", p.Name(), level, got, i, stone)
					}
				}
			}
		}
	}
}

// A hand is a stencil: paint all round it, and none on it.
func TestAHandIsLeftBareInACloudOfPaint(t *testing.T) {
	m := newRenderer(t).masks[world.Hand]
	palm := m[0][19*maskSide+16]
	finger := m[0][6*maskSide+15]
	beside := m[0][14*maskSide+4] // left of the thumb and the fingers
	if palm != 0 || finger != 0 || beside == 0 {
		t.Errorf("paint on the palm %d, on a finger %d, beside the hand %d: wanted none, none and some", palm, finger, beside)
	}
	if c := coverage(m); c < 0.3 || c > 0.8 {
		t.Errorf("the paint round the hand covers %.2f of the face", c)
	}
	if c := coverage(newRenderer(t).masks[world.Spiral]); c < 0.15 || c > 0.5 {
		t.Errorf("the spiral covers %.2f of the face", c)
	}
}

// A painting is seen on its face of the wall, and not when a block stands in front of it,
// nor when it is on the far side.
func TestAPaintingShowsOnItsFaceAndABlockInFrontHidesIt(t *testing.T) {
	w := world.New()
	wall(w, 6, world.Stone)
	r := newRenderer(t)
	bare := sum(t, r, w)

	r.Paintings = map[world.Side]world.Painting{}
	for y := 10; y < 15; y++ {
		for x := 10; x < 15; x++ {
			p := paintedAt(x, y, 6, world.Spiral)
			r.Paintings[p.Side] = p
		}
	}
	if painted := sum(t, r, w); painted == bare {
		t.Error("the paintings on the wall changed nothing")
	}

	back := map[world.Side]world.Painting{}
	for _, p := range r.Paintings {
		p.Face = world.FaceZPos
		back[p.Side] = p
	}
	r.Paintings = back
	if far := sum(t, r, w); far != bare {
		t.Errorf("paintings on the far side of the wall changed the frame: %d, wanted %d", far, bare)
	}

	front := map[world.Side]world.Painting{}
	for _, p := range back {
		p.Face = world.FaceZNeg
		front[p.Side] = p
	}
	in := world.New()
	wall(in, 6, world.Stone)
	wall(in, 5, world.Stone) // a wall put up in front of the painted one
	r.Paintings = nil
	covered := sum(t, r, in)
	r.Paintings = front
	if hidden := sum(t, r, in); hidden != covered {
		t.Errorf("paintings behind a wall changed the frame: %d, wanted %d", hidden, covered)
	}
}

// Only rock takes paint: a painting left on a block that is no longer rock is not drawn.
func TestAPaintingIsDrawnOnlyOnRock(t *testing.T) {
	w := world.New()
	wall(w, 6, world.Planks)
	r := newRenderer(t)
	bare := sum(t, r, w)
	r.Paintings = map[world.Side]world.Painting{}
	p := paintedAt(12, 12, 6, world.Hand)
	r.Paintings[p.Side] = p
	if got := sum(t, r, w); got != bare {
		t.Errorf("a painting on planks changed the frame: %d, wanted %d", got, bare)
	}
	if !r.rock[world.Stone] || r.rock[world.Planks] || r.rock[world.Dirt] {
		t.Error("the renderer's rock is not the world's")
	}
}

// One frame of a wall of 200 paintings, face on, at 240 by 67 cells.
func BenchmarkPaintedWall(b *testing.B) {
	w := world.New()
	wall(w, 6, world.Stone)
	r := newRenderer(b)
	r.Paintings = map[world.Side]world.Painting{}
	for y := 4; y < 14; y++ {
		for x := 2; x < 22; x++ {
			p := paintedAt(x, y, 6, world.Picture(x%int(world.Pictures)))
			r.Paintings[p.Side] = p
		}
	}
	c := newCanvas(b, 240, 67)
	for b.Loop() {
		c.Clear()
		r.Draw(c, w, facing, noon)
	}
}

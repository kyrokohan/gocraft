package render

import (
	"math"
	"testing"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/gocraft/internal/world"
)

func TestAFaceIsDarkerInTheCornersOfTheBlocksBesideIt(t *testing.T) {
	r := newRenderer(t)
	w := world.New()
	w.Set(5, 10, 5, world.Stone)
	top := world.Hit{X: 5, Y: 10, Z: 5, Face: world.FaceYPos, Block: world.Stone}
	if got := r.corners(w, top); got != [4]uint32{256, 256, 256, 256} {
		t.Errorf("in the open: %v, wanted no shadow", got)
	}

	w.Set(6, 11, 5, world.Stone) // beside the cell over it, along +x, which is u
	if got, want := r.corners(w, top), [4]uint32{256, 220, 256, 220}; got != want {
		t.Errorf("a block beside along u: %v, wanted %v", got, want)
	}
	w.Set(5, 11, 6, world.Stone) // and along +z, which is v: the corner with both is shut
	if got, want := r.corners(w, top), [4]uint32{256, 220, 220, 150}; got != want {
		t.Errorf("blocks beside along u and v: %v, wanted %v", got, want)
	}
	w.Set(4, 11, 5, world.Glass) // glass lets the air through, and casts none
	if got, want := r.corners(w, top), [4]uint32{256, 220, 220, 150}; got != want {
		t.Errorf("with glass along -u: %v, wanted %v", got, want)
	}
	if got := blend([4]uint32{256, 220, 220, 150}, 0.5, 0.5); got != (256+220+220+150)/4 {
		t.Errorf("the middle is %d, wanted the mean of the corners", got)
	}
}

func TestTheFloorOfAPitIsDarkerThanTheGroundRoundIt(t *testing.T) {
	w := world.New()
	for x := range 21 {
		for z := range 21 {
			for y := range 11 {
				w.Set(x, y, z, world.Stone)
			}
		}
	}
	w.Set(10, 10, 10, world.Air) // a pit a block deep, its floor the top of 10, 9, 10
	c := newCanvas(t, 60, 30)
	down := mesh.Camera{Position: geometry.Vector3{X: 10.5, Y: 16, Z: 10.5}, Pitch: -math.Pi/2 + 0.001, FOV: math.Pi / 3, Far: ViewDistance}
	newRenderer(t).Draw(c, w, down, noon)

	mean := func(x0, y0 int) float64 {
		sum := 0
		for y := y0; y < y0+4; y++ {
			for x := x0; x < x0+4; x++ {
				r, g, b := c.At(x, y).RGB()
				sum += int(r) + int(g) + int(b)
			}
		}
		return float64(sum) / 16
	}
	pit, ground := mean(28, 28), mean(50, 28)
	if ratio := pit / ground; ratio < 0.45 || ratio > 0.75 {
		t.Errorf("the pit's floor is %.0f and the ground %.0f: a ratio of %.2f, wanted about %.2f", pit, ground, ratio, 150.0/256)
	}
}

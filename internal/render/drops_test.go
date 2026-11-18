package render

import (
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/gocraft/internal/assets"
	"github.com/kyrokohan/gocraft/internal/world"
)

// sum is the brightness of a canvas, added up over every pixel.
func sum(t *testing.T, r *Renderer, w *world.World) (total int) {
	t.Helper()
	c := newCanvas(t, 60, 30)
	r.Draw(c, w, facing, noon)
	for y := range c.Height() {
		for x := range c.Width() {
			red, green, blue := c.At(x, y).RGB()
			total += int(red) + int(green) + int(blue)
		}
	}
	return total
}

func TestABlockBeingBrokenIsDrawnCrackedAsTheBreakingGoesOn(t *testing.T) {
	w := world.New()
	wall(w, 6, world.Sand)
	r := newRenderer(t)
	whole := sum(t, r, w)

	r.Crack = Crack{X: 12, Y: 12, Z: 6, Part: 0.3}
	begun := sum(t, r, w)
	r.Crack.Part = 1
	broken := sum(t, r, w)
	r.Crack.Z = 7 // a block of air behind the wall
	elsewhere := sum(t, r, w)

	if !(broken < begun && begun < whole) {
		t.Errorf("the wall is %d whole, %d cracked a little and %d cracked through: wanted it darker as it goes", whole, begun, broken)
	}
	if elsewhere != whole {
		t.Errorf("cracks on a block not in view changed the frame: %d, wanted %d", elsewhere, whole)
	}
}

func TestAThingOnTheGroundIsDrawnAsASmallBlock(t *testing.T) {
	set, err := assets.Load()
	if err != nil {
		t.Fatal(err)
	}
	r, d := New(set), NewDrops(set)
	w := world.New()
	wall(w, 8, world.Stone)
	c := newCanvas(t, 60, 30)
	r.Draw(c, w, facing, noon)
	before := c.At(30, 30)

	d.Draw(c, facing, []Lying{{Item: world.Brick.Item(), Pos: geometry.Vector3{X: 12.5, Y: 12.3, Z: 4.5}, Size: 0.5, Light: 1}}, noon)

	if after := c.At(30, 30); after == before {
		t.Errorf("the middle of the view is still %v with a brick lying in front of it", after)
	}
	if red, green, _ := c.At(30, 30).RGB(); int(red) < int(green)+20 {
		t.Errorf("the brick is drawn red %d green %d, wanted it red", red, green)
	}
}

func TestABitIsDrawnAsATinyBlockWhereItIs(t *testing.T) {
	set, err := assets.Load()
	if err != nil {
		t.Fatal(err)
	}
	r, d := New(set), NewDrops(set)
	w := world.New()
	wall(w, 8, world.Stone)
	c := newCanvas(t, 60, 30)
	r.Draw(c, w, facing, noon)
	before, corner := c.At(30, 30), c.At(5, 5)

	d.DrawBits(c, facing, []Bit{{Item: world.Brick.Item(), Pos: geometry.Vector3{X: 12.5, Y: 12.5, Z: 4}, Size: 0.15, Spin: 1, Light: 1}}, noon)
	if red, green, _ := c.At(30, 30).RGB(); c.At(30, 30) == before || int(red) < int(green)+20 {
		t.Errorf("the middle of the view is %v with a bit of brick in front of it, was %v", c.At(30, 30), before)
	}
	if c.At(5, 5) != corner {
		t.Error("a tiny bit in the middle changed the corner of the view")
	}
}

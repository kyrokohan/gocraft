package render

import (
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/assets"
	"github.com/kyrokohan/gocraft/internal/world"
)

func TestWhatIsHeldIsDrawnLowOnTheRightOverAWallJustAhead(t *testing.T) {
	set, err := assets.Load()
	if err != nil {
		t.Fatal(err)
	}
	r, d := New(set), NewDrops(set)
	w := world.New()
	wall(w, 3, world.Stone) // its face at z 3, a fifth of a block from the eye
	near := facing
	near.Position.Z = 2.8
	c := newCanvas(t, 60, 30)
	r.Draw(c, w, near, noon)
	before := snapshot(c)

	d.DrawHand(c, Held{Light: 1})
	if snapshot(c) != before {
		t.Error("an empty hand changed the view")
	}
	for _, item := range []world.Item{world.Planks.Item(), world.StonePick} {
		c := newCanvas(t, 60, 30)
		r.Draw(c, w, near, noon)
		d.DrawHand(c, Held{Item: item, Light: 1})
		changed := func(x0, y0, x1, y1 int) (n int) {
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					if c.At(x, y) != before[y*60+x] {
						n++
					}
				}
			}
			return n
		}
		if right, left := changed(40, 35, 60, 60), changed(0, 0, 30, 30); right < 10 || left != 0 {
			t.Errorf("%s in hand changed %d pixels low on the right and %d high on the left", item.Name(), right, left)
		}
	}
}

// snapshot is every pixel of a canvas 60 wide, row by row.
func snapshot(c *engine.Canvas) (all [60 * 60]engine.Color) {
	for y := range c.Height() {
		for x := range c.Width() {
			all[y*60+x] = c.At(x, y)
		}
	}
	return all
}

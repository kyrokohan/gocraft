package render

import (
	"math"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/gocraft/internal/assets"
	"github.com/kyrokohan/gocraft/internal/world"
)

func newMossbacks(t testing.TB) *Mossbacks {
	t.Helper()
	set, err := assets.Load()
	if err != nil {
		t.Fatal(err)
	}
	return NewMossbacks(set)
}

// ahead is a mossback standing in front of the facing camera, turned by yaw, at a depth.
func ahead(depth, yaw float64) Mossback {
	return Mossback{
		Pos:   geometry.Vector3{X: facing.Position.X, Y: facing.Position.Y - 0.6, Z: facing.Position.Z + depth},
		Yaw:   yaw,
		Light: 1, // in the full light of day
	}
}

// with draws a world and then creatures over it, and returns that and the world alone.
func with(t *testing.T, w *world.World, all ...Mossback) (bare, drawn *engine.Canvas) {
	t.Helper()
	rend, moss := newRenderer(t), newMossbacks(t)
	bare, drawn = newCanvas(t, 80, 40), newCanvas(t, 80, 40)
	rend.Draw(bare, w, facing, noon)
	rend.Draw(drawn, w, facing, noon)
	moss.Draw(drawn, facing, all, noon)
	return bare, drawn
}

// dark counts the pixels that are nearly black: an eye, or a hoof.
func dark(c *engine.Canvas) int {
	n := 0
	for y := range c.Height() {
		for x := range c.Width() {
			if p := c.At(x, y); !p.IsDefault() {
				if r, g, b := p.RGB(); r < 45 && g < 45 && b < 50 {
					n++
				}
			}
		}
	}
	return n
}

func TestAMossbackIsDrawnInItsOwnColors(t *testing.T) {
	bare, drawn := with(t, world.New(), ahead(3, math.Pi/2)) // side on

	n := differing(bare, drawn)
	if n < 150 {
		t.Fatalf("the creature changed %d pixels at 3 away, wanted a body", n)
	}
	// Seen from the side it is moss above and cream below: green, and something lighter.
	green, cream := 0, 0
	for y := range drawn.Height() {
		for x := range drawn.Width() {
			if bare.At(x, y) == drawn.At(x, y) {
				continue
			}
			switch r, g, b := drawn.At(x, y).RGB(); {
			case g > r && g > b:
				green++
			case r > 120 && g > 110 && r >= b:
				cream++
			}
		}
	}
	if green < n/10 || cream < n/10 {
		t.Errorf("of %d pixels, %d are moss and %d are hide: wanted both", n, green, cream)
	}
}

func TestBlocksHideAMossbackAndItHidesBlocks(t *testing.T) {
	near, far := world.New(), world.New()
	wall(near, 4, world.Stone) // between the camera and the creature
	wall(far, 9, world.Stone)  // behind the creature

	if bare, drawn := with(t, near, ahead(4, 0)); differing(bare, drawn) != 0 {
		t.Errorf("the creature shows through a wall in %d pixels", differing(bare, drawn))
	}
	if bare, drawn := with(t, far, ahead(4, 0)); differing(bare, drawn) < 80 {
		t.Errorf("the creature changed %d pixels in front of a wall, wanted it drawn over it", differing(bare, drawn))
	}
}

func TestTheFaceIsOnTheFrontOfTheHead(t *testing.T) {
	_, towards := with(t, world.New(), ahead(2.5, math.Pi)) // facing the camera
	_, away := with(t, world.New(), ahead(2.5, 0))

	if eyes, none := dark(towards), dark(away); eyes < none+4 {
		t.Errorf("%d dark pixels with the creature facing the camera and %d facing away: wanted its eyes", eyes, none)
	}
}

func TestTheLegsSwingWithTheStride(t *testing.T) {
	standing := ahead(3, math.Pi/2)
	stepping := standing
	stepping.Stride = math.Pi / 2

	_, a := with(t, world.New(), standing)
	_, b := with(t, world.New(), stepping)

	if differing(a, b) == 0 {
		t.Error("the picture is the same in the middle of a step as standing")
	}
}

func TestAFarMossbackFadesIntoTheFog(t *testing.T) {
	bare, drawn := with(t, world.New(), ahead(ViewDistance-3, math.Pi/2))

	fr, fg, fb := noon.Fog.Color.RGB()
	for y := range drawn.Height() {
		for x := range drawn.Width() {
			if bare.At(x, y) == drawn.At(x, y) {
				continue
			}
			r, g, b := drawn.At(x, y).RGB()
			if off := max(int(fr)-int(r), int(fg)-int(g), int(fb)-int(b)); off > 40 {
				t.Fatalf("pixel %d,%d of a creature at the end of the view is %v, wanted nearly the fog's %v",
					x, y, drawn.At(x, y), noon.Fog.Color)
			}
		}
	}
}

// Twelve creatures in view, over a frame of 240 by 67 cells.
func BenchmarkMossbacks(b *testing.B) {
	moss := newMossbacks(b)
	c := newCanvas(b, 240, 67)
	var all []Mossback
	for i := range 12 {
		m := ahead(4+float64(i), float64(i))
		m.Pos.X += float64(i%5) - 2
		all = append(all, m)
	}

	for b.Loop() {
		c.Clear()
		moss.Draw(c, facing, all, noon)
	}
}

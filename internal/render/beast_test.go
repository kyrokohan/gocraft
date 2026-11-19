package render

import (
	"math"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/world"
)

func newBeasts(t testing.TB) *Beasts {
	t.Helper()
	set, err := assets.Load()
	if err != nil {
		t.Fatal(err)
	}
	return NewBeasts(set)
}

// ahead is a mossback standing in front of the facing camera, turned by yaw, at a depth.
func ahead(depth, yaw float64) Beast {
	return Beast{
		Pos:   geometry.Vector3{X: facing.Position.X, Y: facing.Position.Y - 0.6, Z: facing.Position.Z + depth},
		Yaw:   yaw,
		Light: 1, // in the full light of day
	}
}

// with draws a world and then creatures over it, and returns that and the world alone.
func with(t *testing.T, w *world.World, all ...Beast) (bare, drawn *engine.Canvas) {
	t.Helper()
	rend, moss := newRenderer(t), newBeasts(t)
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
	moss := newBeasts(b)
	c := newCanvas(b, 240, 67)
	var all []Beast
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

// ahead is a mossback, so a bugler is made from one.
func bugler(depth, yaw float64) Beast {
	b := ahead(depth, yaw)
	b.Kind = Bugler
	b.Pos.Y = facing.Position.Y - 1.6
	return b
}

func TestABuglerStandsThreeBlocksTallWithARedCrest(t *testing.T) {
	bare, drawn := with(t, world.New(), bugler(9, math.Pi/2)) // side on

	if n := differing(bare, drawn); n < 150 {
		t.Fatalf("the bugler changed %d pixels at 9 away, wanted a body", n)
	}
	top, bottom, left, right := span(bare, drawn)
	// Side on it is longer than it is tall: five and a half blocks to three and a half.
	if tall, long := bottom-top+1, right-left+1; long < tall*5/4 || long > tall*5/2 {
		t.Errorf("the bugler is %d pixels tall and %d long, wanted it about half as long again as tall", tall, long)
	}
	red := 0
	for y := range drawn.Height() {
		for x := range drawn.Width() {
			if r, g, b := drawn.At(x, y).RGB(); bare.At(x, y) != drawn.At(x, y) && r > 120 && int(r) > int(g)+45 && b < 80 {
				red++
			}
		}
	}
	if red < 3 {
		t.Errorf("%d pixels of the bugler are red, wanted its crest", red)
	}
}

func TestAYoungBeastIsTheSameShapeHalfTheSize(t *testing.T) {
	bare, grown := with(t, world.New(), bugler(9, math.Pi/2))
	young := bugler(9, math.Pi/2)
	young.Young = true
	_, small := with(t, world.New(), young)

	top, bottom, _, _ := span(bare, grown)
	smallTop, smallBottom, _, _ := span(bare, small)
	if tall, low := bottom-top+1, smallBottom-smallTop+1; low*2 < tall-3 || low*2 > tall+3 {
		t.Errorf("a young bugler is %d pixels tall and a grown one %d, wanted half", low, tall)
	}
	if w, h := Size(Bugler, true); w != sizes[Bugler][0]/2 || h != sizes[Bugler][1]/2 {
		t.Errorf("the box of a young bugler is %v by %v, wanted half a grown one's", w, h)
	}
}

func TestADeadBeastSinksAndFadesAway(t *testing.T) {
	bare, alive := with(t, world.New(), bugler(9, math.Pi/2))
	falling := bugler(9, math.Pi/2)
	falling.Fall = 0.5
	_, half := with(t, world.New(), falling)
	falling.Fall = 1
	_, gone := with(t, world.New(), falling)

	top, _, _, _ := span(bare, alive)
	halfTop, _, _, _ := span(bare, half)
	if halfTop <= top {
		t.Errorf("the top of a falling bugler is at row %d and of a standing one at %d, wanted it lower", halfTop, top)
	}
	if n := differing(bare, gone); n != 0 {
		t.Errorf("a bugler that has fallen all the way still changes %d pixels", n)
	}
}

func TestEveryKindHasANameAShapeAndASize(t *testing.T) {
	beasts := newBeasts(t)
	for k := range Kinds {
		if named, ok := KindNamed(k.String()); !ok || named != k || k.String() == "beast" {
			t.Errorf("kind %d is named %q, which does not name it back", k, k.String())
		}
		if w, h := Size(k, false); w <= 0 || h <= 0 {
			t.Errorf("a %s has a box of %v by %v", k, w, h)
		}
		if s := beasts.shapes[k]; len(s.parts) == 0 || s.hurt == nil || s.stand <= 0 {
			t.Errorf("a %s has no shape to draw", k)
		}
	}
	if _, ok := KindNamed("gryphon"); ok {
		t.Error("there is a kind named gryphon")
	}
}

// Thirty beasts in view, two thirds of them buglers, over a frame of 240 by 67 cells.
func BenchmarkBeasts(b *testing.B) {
	beasts := newBeasts(b)
	c := newCanvas(b, 240, 67)
	var all []Beast
	for i := range 30 {
		one := bugler(8+float64(i%10)*2, float64(i))
		if i%3 == 0 {
			one = ahead(4+float64(i%10), float64(i))
		}
		one.Pos.X += float64(i%7)*2.5 - 7
		one.Stride, one.Young = float64(i), i%5 == 4
		all = append(all, one)
	}

	for b.Loop() {
		c.Clear()
		beasts.Draw(c, facing, all, noon)
	}
}

package render

import (
	"math"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/world"
)

// backTo is a cave person standing with its back to the facing camera, at a depth.
func backTo(depth float64) Person {
	return Person{
		Pos:   geometry.Vector3{X: facing.Position.X, Y: facing.Position.Y - 1.2, Z: facing.Position.Z + depth},
		Light: 1, // in the full light of day
	}
}

// withPerson draws a world and then a cave person over it, and returns that and the
// world alone.
func withPerson(t *testing.T, w *world.World, one Person) (bare, drawn *engine.Canvas) {
	t.Helper()
	set, err := assets.Load()
	if err != nil {
		t.Fatal(err)
	}
	rend, people, held := newRenderer(t), NewPeople(set), NewDrops(set)
	bare, drawn = newCanvas(t, 80, 80), newCanvas(t, 80, 80)
	rend.Draw(bare, w, facing, noon)
	rend.Draw(drawn, w, facing, noon)
	people.Draw(drawn, facing, []Person{one}, noon, held)
	return bare, drawn
}

// span returns the rows and columns in which two canvases differ: the first and the
// last of each.
func span(a, b *engine.Canvas) (top, bottom, left, right int) {
	top, left = a.Height(), a.Width()
	bottom, right = -1, -1
	for y := range a.Height() {
		for x := range a.Width() {
			if a.At(x, y) != b.At(x, y) {
				top, bottom, left, right = min(top, y), max(bottom, y), min(left, x), max(right, x)
			}
		}
	}
	return top, bottom, left, right
}

func TestACavePersonStandsTallerThanItIsWide(t *testing.T) {
	bare, drawn := withPerson(t, world.New(), backTo(4))

	if n := differing(bare, drawn); n < 150 {
		t.Fatalf("the cave person changed %d pixels at 4 away, wanted a body", n)
	}
	top, bottom, left, right := span(bare, drawn)
	tall, wide := bottom-top+1, right-left+1
	if tall < 2*wide || tall < 20 {
		t.Errorf("the cave person is %d pixels tall and %d wide, wanted it more than twice as tall as wide", tall, wide)
	}
}

func TestEachLookHasItsOwnColorsOnTheSameBody(t *testing.T) {
	_, first := withPerson(t, world.New(), backTo(4))
	for look := 1; look < assets.PersonLooks; look++ {
		one := backTo(4)
		one.Look = look
		bare, drawn := withPerson(t, world.New(), one)

		if n := differing(first, drawn); n < 60 {
			t.Errorf("look %d differs from the first look in %d pixels, wanted its own colors", look, n)
		}
		a, b, c, d := span(bare, first)
		if e, f, g, h := span(bare, drawn); a != e || b != f || c != g || d != h {
			t.Errorf("look %d has another shape than the first look", look)
		}
	}
}

func TestAWallHidesTheCavePerson(t *testing.T) {
	w := world.New()
	for y := range 12 {
		for x := range 12 {
			w.Set(int(facing.Position.X)-6+x, int(facing.Position.Y)-6+y, int(facing.Position.Z)+2, world.Stone)
		}
	}
	if bare, drawn := withPerson(t, w, backTo(4)); differing(bare, drawn) != 0 {
		t.Errorf("a cave person behind a wall changed %d pixels", differing(bare, drawn))
	}
}

func TestACrouchedCavePersonIsLowerAndAHurtOneRedder(t *testing.T) {
	bare, stood := withPerson(t, world.New(), backTo(4))
	low := backTo(4)
	low.Crouched = true
	_, crouched := withPerson(t, world.New(), low)

	standTop, _, _, _ := span(bare, stood)
	crouchTop, _, _, _ := span(bare, crouched)
	if crouchTop < standTop+2 {
		t.Errorf("the top of a crouched cave person is at row %d and of a standing one at %d, wanted it lower", crouchTop, standTop)
	}

	sore := backTo(4)
	sore.Hurt = true
	_, hurt := withPerson(t, world.New(), sore)
	red := func(c *engine.Canvas) (sum int) {
		for y := range c.Height() {
			for x := range c.Width() {
				if bare.At(x, y) != c.At(x, y) {
					r, g, _ := c.At(x, y).RGB()
					sum += int(r) - int(g)
				}
			}
		}
		return sum
	}
	if red(hurt) < red(stood)+1000 {
		t.Errorf("a hurt cave person is red by %d and a well one by %d, wanted it flushed", red(hurt), red(stood))
	}
}

func TestASeatedCavePersonSitsAstrideAsHighAsOneThatStands(t *testing.T) {
	bare, stood := withPerson(t, world.New(), backTo(4))
	astride := backTo(4)
	astride.Seated, astride.Stride = true, math.Pi/2 // it rides, and its legs do not walk
	_, seated := withPerson(t, world.New(), astride)

	standTop, standBottom, standLeft, standRight := span(bare, stood)
	top, bottom, left, right := span(bare, seated)
	if top != standTop {
		t.Errorf("the head of a seated cave person is at row %d and of a standing one at %d, wanted them level", top, standTop)
	}
	if right-left < standRight-standLeft+6 || bottom > standBottom-2 {
		t.Errorf("a seated cave person is %d pixels wide down to row %d, and a standing one %d wide down to %d: wanted its feet out to its sides, and higher",
			right-left+1, bottom, standRight-standLeft+1, standBottom)
	}
}

func TestTheLimbsSwingAndTheHandHoldsWhatIsHeld(t *testing.T) {
	_, still := withPerson(t, world.New(), backTo(4))

	walking := backTo(4)
	walking.Stride = math.Pi / 2
	if _, drawn := withPerson(t, world.New(), walking); differing(still, drawn) < 20 {
		t.Errorf("a stride changed %d pixels, wanted the legs and arms to swing", differing(still, drawn))
	}

	striking := backTo(4)
	striking.Swing = 0.5
	if _, drawn := withPerson(t, world.New(), striking); differing(still, drawn) < 10 {
		t.Errorf("a swing changed %d pixels, wanted the arm to rise", differing(still, drawn))
	}

	for _, item := range []world.Item{world.Stone.Item(), world.StonePick} {
		holding := backTo(4)
		holding.Held = item
		if _, drawn := withPerson(t, world.New(), holding); differing(still, drawn) < 8 {
			t.Errorf("holding %s changed %d pixels, wanted it in the hand", item.Name(), differing(still, drawn))
		}
	}
}

func TestTheHeadTipsWithTheView(t *testing.T) {
	_, level := withPerson(t, world.New(), backTo(3))
	up := backTo(3)
	up.Tilt = 0.6
	if _, drawn := withPerson(t, world.New(), up); differing(level, drawn) < 6 {
		t.Errorf("looking up changed %d pixels, wanted the head to tip", differing(level, drawn))
	}
}

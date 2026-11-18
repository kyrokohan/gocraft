package game

import (
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/gocraft/internal/world"
)

// lookingDown is a game on the test floor whose player looks at the floor a little
// ahead: at the top of the block at 20, 5, 21.
func lookingDown(t *testing.T) *Game {
	t.Helper()
	g := onFloor(t)
	g.player.Pitch = -0.9
	hold(t, g, 1)
	return g
}

func TestThePlayerAimsAtTheBlockInTheMiddleOfTheView(t *testing.T) {
	g := lookingDown(t)

	want := world.Hit{X: 20, Y: 5, Z: 21, Face: world.FaceYPos, Block: world.Stone}
	got := g.target
	got.T = 0
	if !g.aimed || got != want {
		t.Errorf("aimed %v at %+v, wanted %+v", g.aimed, got, want)
	}
}

func TestNothingIsAimedAtBeyondTheReach(t *testing.T) {
	g := onFloor(t)
	g.player.Pitch = 1.2 // at the sky

	hold(t, g, 1, 'q')
	hold(t, g, 1, 'e')

	if g.aimed {
		t.Errorf("aimed at %+v, wanted nothing", g.target)
	}
}

func TestQBreaksTheBlockAimedAt(t *testing.T) {
	g := lookingDown(t)

	hold(t, g, 1, 'q')

	if got := g.world.At(20, 5, 21); got != world.Air {
		t.Errorf("the block is %v, wanted it gone", got.Info().Name)
	}
	// With that block gone the player aims through the hole, at the side of the next one.
	want := world.Hit{X: 20, Y: 5, Z: 22, Face: world.FaceZNeg, Block: world.Stone}
	got := g.target
	got.T = 0
	if !g.aimed || got != want {
		t.Errorf("after breaking, aimed %v at %+v, wanted %+v", g.aimed, got, want)
	}
}

func TestHoldingQKeepsDigging(t *testing.T) {
	g := lookingDown(t)

	// A held key repeats, and each repeat is a press.
	kb := engine.NewKeyboard()
	for range 3 {
		kb.Press('q')
		if err := g.Update(frame, kb.Input()); err != nil {
			t.Fatal(err)
		}
		kb.EndFrame()
	}

	dug := 0
	for z := range 40 {
		for x := range 40 {
			for y := range 6 {
				if g.world.At(x, y, z) == world.Air {
					dug++
				}
			}
		}
	}
	if dug != 3 {
		t.Errorf("%d blocks of the floor are gone, wanted 3", dug)
	}
}

func TestDeepstoneCannotBeBroken(t *testing.T) {
	g := lookingDown(t)
	g.world.Set(20, 5, 21, world.Deepstone)

	hold(t, g, 1, 'Q')

	if got := g.world.At(20, 5, 21); got != world.Deepstone {
		t.Errorf("the block is %v, wanted deepstone still", got.Info().Name)
	}
}

func TestEPlacesTheSelectedBlockOnTheFaceAimedAt(t *testing.T) {
	g := lookingDown(t)

	hold(t, g, 1, '8')
	hold(t, g, 1, 'e')

	if got := g.world.At(20, 6, 21); got != world.Brick {
		t.Errorf("the block on top is %v, wanted brick", got.Info().Name)
	}
}

func TestABlockCannotBePlacedWhereThePlayerStands(t *testing.T) {
	g := onFloor(t)
	g.player.Pitch = -maxPitch // at the floor under the feet
	hold(t, g, 1)
	if !g.aimed || g.target.X != 20 || g.target.Z != 20 {
		t.Fatalf("aimed %v at %+v, wanted the block underfoot", g.aimed, g.target)
	}

	hold(t, g, 1, 'e')

	if got := g.world.At(20, 6, 20); got != world.Air {
		t.Errorf("a block of %v was placed inside the player", got.Info().Name)
	}
}

func TestABlockCanBePlacedInWater(t *testing.T) {
	g := lookingDown(t)
	g.world.Set(20, 6, 21, world.Water)

	hold(t, g, 1, 'e')

	if got := g.world.At(20, 6, 21); got != hotbar[0] {
		t.Errorf("the block is %v, wanted %v in place of the water", got.Info().Name, hotbar[0].Info().Name)
	}
}

func TestNumberKeysAndTabPickTheBlock(t *testing.T) {
	g := onFloor(t)

	for i := range hotbar {
		hold(t, g, 1, engine.Key('1'+rune(i)))
		if g.selected != i {
			t.Errorf("key %c picked slot %d, wanted %d", '1'+rune(i), g.selected, i)
		}
	}

	hold(t, g, 1, engine.KeyTab)
	if g.selected != 0 {
		t.Errorf("tab on the last slot picked %d, wanted it to go round to 0", g.selected)
	}
	hold(t, g, 1, engine.KeyTab)
	if g.selected != 1 {
		t.Errorf("tab picked %d, wanted 1", g.selected)
	}
}

func TestEveryHotbarBlockCanBeStoodOnAndBroken(t *testing.T) {
	for _, b := range hotbar {
		if info := b.Info(); !info.Solid || info.Fixed || info.Name == "" {
			t.Errorf("%v is in the hotbar: %+v", b, info)
		}
	}
}

// outlined counts the pixels of a frame that are the color of the aim outline, and
// says whether all of them are inside a rectangle of pixels.
func outlined(c *engine.Canvas, x0, y0, x1, y1 int) (n int, inside bool) {
	inside = true
	for y := range c.Height() {
		for x := range c.Width() {
			if c.At(x, y) != hudPick.Foreground {
				continue
			}
			n++
			if x < x0 || x > x1 || y < y0 || y > y1 {
				inside = false
			}
		}
	}
	return n, inside
}

func TestTheBlockAimedAtIsOutlined(t *testing.T) {
	g := lookingDown(t)
	fb, _ := engine.NewFrameBuffer(120, 40)
	c := fb.Canvas()
	cam := g.player.Camera()

	g.rend.Draw(c, g.world, cam)
	if n, _ := outlined(c, 0, 0, c.Width(), c.Height()); n != 0 {
		t.Fatalf("%d pixels of the world are the color of the outline; the test needs another color", n)
	}
	g.Draw(fb)

	// The top of the block aimed at, 20,5,21, is a square on the floor. Where its four
	// corners are drawn bounds the outline.
	x0, y0, x1, y1 := c.Width(), c.Height(), 0, 0
	for _, corner := range [][2]float64{{20, 21}, {21, 21}, {20, 22}, {21, 22}} {
		x, y, _, ok := cam.ProjectPixels(geometry.Vector3{X: corner[0], Y: 6, Z: corner[1]}, c)
		if !ok {
			t.Fatal("a corner of the block is not in view")
		}
		x0, y0 = min(x0, int(x)-1), min(y0, int(y)-1)
		x1, y1 = max(x1, int(x)+1), max(y1, int(y)+1)
	}
	n, inside := outlined(c, x0, y0, x1, y1)
	if n < 2*(x1-x0) {
		t.Errorf("%d pixels are outline, wanted the four edges of a square %d across", n, x1-x0)
	}
	if !inside {
		t.Errorf("the outline goes outside the block, which is at %d,%d to %d,%d", x0, y0, x1, y1)
	}
}

func TestNothingIsOutlinedWhenNothingIsAimedAt(t *testing.T) {
	g := onFloor(t)
	g.player.Pitch = 1.2 // at the sky
	hold(t, g, 1)
	fb, _ := engine.NewFrameBuffer(120, 40)

	g.Draw(fb)

	if n, _ := outlined(fb.Canvas(), 0, 0, 120, 80); n != 0 {
		t.Errorf("%d pixels are outline with nothing aimed at", n)
	}
}

func TestAPlantDoesNotStopThePlayerAndCanBePicked(t *testing.T) {
	g := lookingDown(t)
	g.world.Set(20, 6, 21, world.TallGrass) // on the block the player aims at
	hold(t, g, 1)
	if !g.aimed || g.target.Block != world.TallGrass {
		t.Fatalf("aimed %v at %+v, wanted the plant", g.aimed, g.target)
	}

	hold(t, g, 30, 'w') // a second of walking, straight through it
	if g.player.Pos.Z < 22 {
		t.Errorf("the player is at z=%v: a plant stopped it", g.player.Pos.Z)
	}

	g = lookingDown(t)
	g.world.Set(20, 6, 21, world.TallGrass)
	hold(t, g, 1, 'q')
	if got := g.world.At(20, 6, 21); got != world.Air {
		t.Errorf("after Q the plant is %v, wanted it picked", got.Info().Name)
	}
	if got := g.world.At(20, 5, 21); got != world.Stone {
		t.Errorf("picking the plant broke the block under it: %v", got.Info().Name)
	}
}

func TestABlockPlacedOnAPlantTakesItsPlace(t *testing.T) {
	g := lookingDown(t)
	g.world.Set(20, 6, 21, world.RedFlower)

	hold(t, g, 1, '8')
	hold(t, g, 1, 'e')

	if got := g.world.At(20, 6, 21); got != world.Brick {
		t.Errorf("the cell of the flower holds %v, wanted the brick that was placed", got.Info().Name)
	}
	if got := g.world.At(20, 7, 21); got != world.Air {
		t.Errorf("a block of %v was placed above the flower", got.Info().Name)
	}
}

func TestBreakingTheGroundTakesThePlantOnIt(t *testing.T) {
	g := onFloor(t)
	g.world.Set(20, 6, 22, world.Dirt) // a block standing on the floor ahead of the player
	g.world.Set(20, 7, 22, world.Bush) // with a bush on it
	g.player.Pitch = -0.45             // at the side of the block, under the bush
	hold(t, g, 1)
	if want := (world.Hit{X: 20, Y: 6, Z: 22, Face: world.FaceZNeg, Block: world.Dirt}); !g.aimed ||
		g.target.X != want.X || g.target.Y != want.Y || g.target.Z != want.Z || g.target.Block != want.Block {
		t.Fatalf("aimed %v at %+v, wanted the side of the block under the bush", g.aimed, g.target)
	}

	hold(t, g, 1, 'q')

	if g.world.At(20, 6, 22) != world.Air || g.world.At(20, 7, 22) != world.Air {
		t.Errorf("after the block was broken it is %v with %v on it, wanted both gone",
			g.world.At(20, 6, 22).Info().Name, g.world.At(20, 7, 22).Info().Name)
	}
}

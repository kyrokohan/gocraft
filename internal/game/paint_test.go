package game

import (
	"path/filepath"
	"strings"
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/skyspear/internal/world"
)

// wallAhead is inSurvival with a block at eye height ahead of the player, aimed at on its
// face toward the player, and three red paints in hand. It returns that face.
func wallAhead(t *testing.T, b world.Block) (*Game, world.Side) {
	t.Helper()
	g := inSurvival(t)
	g.world.Set(20, 7, 22, b)
	holding(g, world.RedPaint, 3)
	hold(t, g, 1)
	if !g.aimed || g.target.X != 20 || g.target.Y != 7 || g.target.Z != 22 || g.target.Face != world.FaceZNeg {
		t.Fatalf("aimed %v at %+v, wanted the face of 20, 7, 22 toward the player", g.aimed, g.target)
	}
	return g, world.Side{Cell: world.Cell{X: 20, Y: 7, Z: 22}, Face: world.FaceZNeg}
}

func TestEWithPaintOnAFaceOfStoneOpensThePickerAndEnterPaintsThere(t *testing.T) {
	g, side := wallAhead(t, world.Stone)

	press(g, 'e')
	if g.screen != picking {
		t.Fatalf("E with paint on stone: screen %d, wanted the picker", g.screen)
	}
	if text, _ := drawn(t, g); !strings.Contains(text, "hand") || !strings.Contains(text, "Enter") {
		t.Errorf("the picker does not show the hand chosen, or how to paint it:\n%s", text)
	}
	press(g, engine.KeyRight)
	if text, _ := drawn(t, g); !strings.Contains(text, "spiral") {
		t.Errorf("Right did not choose the spiral:\n%s", text)
	}
	press(g, engine.KeyEnter)

	want := world.Painting{Side: side, Picture: world.Spiral, Paint: world.RedPaint}
	if got, ok := g.paintings[side]; !ok || got != want || g.screen != playing {
		t.Errorf("Enter painted %+v (%v) and left screen %d, wanted %+v and play", got, ok, g.screen, want)
	}
	if n := g.items[g.selected].Count; n != 2 {
		t.Errorf("painting left %d paints of 3, wanted 2", n)
	}

	// Painting over it changes it.
	holding(g, world.WhitePaint, 1)
	press(g, 'e')
	press(g, engine.KeyEnter)
	if got := g.paintings[side]; got.Picture != world.Hand || got.Paint != world.WhitePaint || len(g.paintings) != 1 {
		t.Errorf("painted over, the face has %+v, and there are %d paintings", got, len(g.paintings))
	}
	if g.items[g.selected].Count != 0 {
		t.Errorf("the last paint was not used: %+v", g.items[g.selected])
	}
}

func TestEscapeLeavesThePickerAndPaintsNothing(t *testing.T) {
	g, _ := wallAhead(t, world.Stone)
	press(g, 'e')
	press(g, engine.KeyEscape)
	if g.screen != playing || len(g.paintings) != 0 || g.items[g.selected].Count != 3 {
		t.Errorf("Escape left screen %d, %d paintings and %d paints", g.screen, len(g.paintings), g.items[g.selected].Count)
	}
}

func TestOnlyRockCanBePainted(t *testing.T) {
	for _, b := range []world.Block{world.Stone, world.Cobble, world.Redrock, world.Sandstone, world.Basalt, world.ChalkSeam, world.Deepstone} {
		g, _ := wallAhead(t, b)
		if press(g, 'e'); g.screen != picking {
			t.Errorf("%s cannot be painted", b.Info().Name)
		}
	}
	for _, b := range []world.Block{world.Dirt, world.Log, world.Leaves, world.Planks} {
		g, _ := wallAhead(t, b)
		if press(g, 'e'); g.screen != playing || g.items[g.selected].Count != 3 {
			t.Errorf("E with paint on %s: screen %d and %d paints left, wanted nothing done", b.Info().Name, g.screen, g.items[g.selected].Count)
		}
	}
}

func TestASaveKeepsEveryPainting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := inSurvival(t)
	g.opts.SavePath = path
	for _, p := range []world.Painting{
		{Side: world.Side{Cell: world.Cell{X: 1, Y: 5, Z: 2}, Face: world.FaceYPos}, Picture: world.Hand, Paint: world.WhitePaint},
		{Side: world.Side{Cell: world.Cell{X: 1, Y: 5, Z: 2}, Face: world.FaceXNeg}, Picture: world.Spiral, Paint: world.BlackPaint},
	} {
		g.paintings[p.Side] = p
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)

	if len(back.paintings) != 2 {
		t.Fatalf("%d paintings came back, wanted 2", len(back.paintings))
	}
	for side, p := range g.paintings {
		if back.paintings[side] != p {
			t.Errorf("%+v came back as %+v", p, back.paintings[side])
		}
	}
}

// Breaking the block takes its painting away. A block put against it only hides it.
func TestBreakingTheBlockTakesItsPaintingAway(t *testing.T) {
	g, side := wallAhead(t, world.Stone)
	g.paintings[side] = world.Painting{Side: side, Picture: world.Hand, Paint: world.RedPaint}

	g.world.Set(20, 7, 21, world.Cobble) // against the painted face
	hold(t, g, 1)
	g.world.Set(20, 7, 21, world.Air)
	hold(t, g, 1)
	if _, ok := g.paintings[side]; !ok {
		t.Fatal("a block put against the painting and taken away again took the painting with it")
	}

	g.world.Set(20, 7, 22, world.Air)
	hold(t, g, 1)
	if _, ok := g.paintings[side]; ok {
		t.Error("the painting is still there with its stone gone")
	}

	// Far away, where its stone is not loaded, a painting is kept.
	far := world.Side{Cell: world.Cell{X: 5000, Y: 6, Z: 5000}, Face: world.FaceZNeg}
	g.paintings[far] = world.Painting{Side: far, Picture: world.Spiral, Paint: world.RedPaint}
	hold(t, g, 1)
	if _, ok := g.paintings[far]; !ok {
		t.Error("a painting where the world is not loaded was taken away")
	}
}

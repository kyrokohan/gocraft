package game

import (
	"strings"
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// paintedAhead is wallAhead with a picture painted on the face toward the player, and a
// block of cobble in hand.
func paintedAhead(t *testing.T, p world.Picture) (*Game, world.Side) {
	t.Helper()
	g, side := wallAhead(t, world.Stone)
	g.paintings[side] = world.Painting{Side: side, Picture: p, Paint: world.RedPaint}
	holding(g, world.Cobble.Item(), 5)
	return g, side
}

func TestEOnAPaintedBeastShowsWhatYouKnowOfIt(t *testing.T) {
	g, _ := paintedAhead(t, beastPicture(render.Daggercat))

	press(g, 'e')
	if g.screen != reading {
		t.Fatalf("E on a painting: screen %d, wanted its card", g.screen)
	}
	text, _ := drawn(t, g)
	if !strings.Contains(text, "daggercat") {
		t.Errorf("the card does not name the daggercat:\n%s", text)
	}
	for _, line := range breeds[render.Daggercat].lore {
		if !strings.Contains(text, line) {
			t.Errorf("the card does not say %q:\n%s", line, text)
		}
	}
	if g.world.At(20, 7, 21) != world.Air || g.items[g.selected].Count != 5 {
		t.Error("E on a painting with a block in hand built against it")
	}

	press(g, 'e')
	if g.screen != playing {
		t.Errorf("E on the card left screen %d, wanted play", g.screen)
	}
	press(g, 'e')
	press(g, engine.KeyEscape)
	if g.screen != playing {
		t.Errorf("Esc on the card left screen %d, wanted play", g.screen)
	}
}

func TestEveryKindOfBeastHasLinesForItsCard(t *testing.T) {
	for k := range render.Kinds {
		lore := breeds[k].lore
		if len(lore) < 2 || len(lore) > 4 {
			t.Errorf("a %s has %d lines, wanted 2 to 4", k, len(lore))
		}
		for _, line := range lore {
			if line == "" || len(line) > cardLine {
				t.Errorf("a %s's line %q is empty or longer than %d", k, line, cardLine)
			}
		}
	}
}

func TestTheCometsCardCountsTheNightsUntilItFalls(t *testing.T) {
	for _, c := range []struct {
		night int
		want  string
	}{{3, "it falls in 9 nights"}, {11, "it falls next night"}} {
		g, _ := paintedAhead(t, world.Comet)
		g.days = float64(c.night) - firstMorning - 0.1 // the evening of that night
		press(g, 'e')
		if text, _ := drawn(t, g); !strings.Contains(text, c.want) {
			t.Errorf("on night %d the comet's card does not say %q:\n%s", c.night, c.want, text)
		}
	}
}

func TestAHandSaysYouWereHere(t *testing.T) {
	g, _ := paintedAhead(t, world.Hand)
	press(g, 'e')
	if text, _ := drawn(t, g); !strings.Contains(text, "you were here") {
		t.Errorf("the hand's card:\n%s", text)
	}
}

func TestWithPaintInHandEPaintsOverAPainting(t *testing.T) {
	g, side := paintedAhead(t, world.Hand)
	holding(g, world.BlackPaint, 1)
	press(g, 'e')
	if g.screen != picking {
		t.Fatalf("E with paint on a painting: screen %d, wanted the picker", g.screen)
	}
	press(g, engine.KeyRight)
	press(g, engine.KeyEnter)
	if p := g.paintings[side]; p.Picture != world.Spiral || p.Paint != world.BlackPaint {
		t.Errorf("painted over, the face has %+v", p)
	}
}

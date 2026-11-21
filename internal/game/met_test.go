package game

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// cat is a daggercat standing on the test floor.
func cat(x, z float64) beast {
	return beast{kind: render.Daggercat, Pos: geometry.Vector3{X: x, Y: 6, Z: z}, health: breeds[render.Daggercat].health}
}

func TestABeastInViewIsMetOnceWithOneMessage(t *testing.T) {
	g := onFloor(t)
	g.fauna.all = []beast{cat(20.5, 32.5)} // 12 blocks ahead
	daggercat := world.FirstBeast + world.Picture(render.Daggercat)

	g.lookRound(meetEvery)
	if !g.met[daggercat] || !strings.Contains(g.notice, "you have seen a daggercat: you can paint it now") {
		t.Fatalf("a daggercat 12 blocks ahead: met %v, told %q", g.met[daggercat], g.notice)
	}
	g.notice = ""
	g.lookRound(meetEvery)
	if g.notice != "" {
		t.Errorf("met again, told %q", g.notice)
	}
	if !slices.Contains(g.known(), daggercat) {
		t.Errorf("the picker does not have the daggercat: %v", g.known())
	}
}

func TestABeastNotInViewIsNotMet(t *testing.T) {
	for name, place := range map[string]func(g *Game){
		"behind a wall": func(g *Game) {
			for y := 6; y < 12; y++ {
				for x := 15; x < 26; x++ {
					g.world.Set(x, y, 26, world.Stone)
				}
			}
			g.fauna.all = []beast{cat(20.5, 32.5)}
		},
		"40 blocks off":     func(g *Game) { g.fauna.all = []beast{cat(20.5, 60.5)} },
		"behind the player": func(g *Game) { g.fauna.all = []beast{cat(20.5, 10.5)} },
	} {
		g := onFloor(t)
		place(g)
		g.lookRound(meetEvery)
		if g.met[world.FirstBeast+world.Picture(render.Daggercat)] || g.notice != "" {
			t.Errorf("a daggercat %s was met, and the game told %q", name, g.notice)
		}
	}
}

func TestABeastThatBitesOrIsHitIsMet(t *testing.T) {
	g := atNight(t)
	g.fauna.all = []beast{claw(20.5, 19.5)} // close behind the player
	for range 90 {
		if hold(t, g, 1); g.health < maxHealth || g.screen != playing {
			break
		}
	}
	if g.health == maxHealth {
		t.Fatal("the quickclaw did not bite")
	}
	if !g.met[world.FirstBeast+world.Picture(render.Quickclaw)] {
		t.Error("a quickclaw that bit the player from behind was not met")
	}

	g = onFloor(t)
	g.fauna.all = []beast{cat(20.5, 19.5)}
	g.wound(0, 1, 1, false)
	if !g.met[world.FirstBeast+world.Picture(render.Daggercat)] {
		t.Error("a daggercat hit from behind was not met")
	}
}

func TestTheCometIsMetTheFirstNightItCanBeSeen(t *testing.T) {
	g := onFloor(t)
	g.days = 3.2 // by day
	drawn(t, g)
	g.lookRound(meetEvery)
	if g.met[world.Comet] {
		t.Fatal("the comet was met by day")
	}
	g.days = 3.6 // that night
	drawn(t, g)
	g.lookRound(meetEvery)
	if !g.met[world.Comet] || !strings.Contains(g.notice, "you have seen the comet") {
		t.Errorf("at night: met %v, told %q", g.met[world.Comet], g.notice)
	}
}

func TestThePickerHasWhatWasMetInOrder(t *testing.T) {
	g := onFloor(t)
	if got := g.known(); !slices.Equal(got, []world.Picture{world.Hand, world.Spiral}) {
		t.Fatalf("at the start the picker has %v, wanted the hand and the spiral", got)
	}
	g.meet(world.FirstBeast + world.Picture(render.Daggercat))
	g.meet(world.Comet)
	want := []world.Picture{world.Hand, world.Spiral, world.Comet, world.FirstBeast + world.Picture(render.Daggercat)}
	if got := g.known(); !slices.Equal(got, want) {
		t.Errorf("the picker has %v, wanted %v", got, want)
	}

	// With everything met the picker is wider than the frame: it shows the part round
	// the picture chosen.
	g, side := wallAhead(t, world.Stone)
	for p := range world.Pictures {
		g.meet(p)
	}
	press(g, 'e')
	press(g, engine.KeyLeft) // from the first round to the last
	if text, _ := drawn(t, g); !strings.Contains(text, "stiltwing") {
		t.Errorf("Left from the first does not show the last:\n%s", text)
	}
	press(g, engine.KeyEnter)
	if p := g.paintings[side]; p.Picture != world.FirstBeast+world.Picture(render.Stiltwing) {
		t.Errorf("painted %+v, wanted a stiltwing", p)
	}
}

func TestASaveKeepsWhatWasMet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := inSurvival(t)
	g.opts.SavePath = path
	g.meet(world.Comet)
	g.meet(world.FirstBeast + world.Picture(render.Dread))
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)

	if back.met != g.met {
		t.Errorf("what was met came back as %v, wanted %v", back.met, g.met)
	}
}

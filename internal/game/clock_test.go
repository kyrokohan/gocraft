package game

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

func TestANewWorldBeginsInTheMorning(t *testing.T) {
	g := onFloor(t)
	if g.days != 0 || g.sky.Time() != firstMorning {
		t.Errorf("a new world is %v days old at %v, wanted 0 at %v", g.days, g.sky.Time(), firstMorning)
	}
	if got := clockText(g.days); got != "day 1, 07:12" {
		t.Errorf("the clock of a new world says %q", got)
	}
}

func TestTheClockSaysTheDayAndTheHour(t *testing.T) {
	for _, c := range []struct {
		days float64
		want string
	}{
		{0, "day 1, 07:12"},
		{0.2, "day 1, 12:00"},
		{0.7, "day 2, 00:00"},
		{2.75, "day 4, 01:12"},
	} {
		if got := clockText(c.days); got != c.want {
			t.Errorf("after %v days the clock says %q, wanted %q", c.days, got, c.want)
		}
	}
	if got := timeOfDay(0.2); math.Abs(got-render.Noon) > 1e-9 {
		t.Errorf("0.2 days in it is %v, wanted noon", got)
	}
}

func TestTheClockRunsInPlayAndStandsStillInAPause(t *testing.T) {
	g := onFloor(t)
	hold(t, g, int(dayLength/frame/10)) // a tenth of a day
	if math.Abs(g.days-0.1) > 1e-9 {
		t.Errorf("after a tenth of a day of play the clock is at %v days", g.days)
	}

	g.show(paused)
	was := g.days
	hold(t, g, 100)
	if g.days != was {
		t.Errorf("the clock went from %v to %v in a pause", was, g.days)
	}
	g.show(onTitle)
	hold(t, g, 100)
	if g.days != was {
		t.Errorf("the clock went from %v to %v on the title", was, g.days)
	}
}

func TestTMovesTheClockOnAQuarterOfADayOnce(t *testing.T) {
	g := onFloor(t)
	kb := engine.NewKeyboard()
	held(t, g, kb, 8, 't')
	if want := quarterDay + 8*frame/dayLength; math.Abs(g.days-want) > 1e-9 {
		t.Errorf("after holding T the clock is at %v days, wanted %v: once and no more", g.days, want)
	}
}

func TestTheSkyIsDrawnAtTheHourOfTheClock(t *testing.T) {
	g := onFloor(t)
	drawn(t, g)
	_, day, _ := g.sky.Fog.Color.RGB()

	g.days = 0.7 // midnight
	drawn(t, g)
	if at := g.sky.Time(); min(at, 1-at) > 1e-9 {
		t.Errorf("the sky is at %v, wanted midnight", at)
	}
	if _, night, _ := g.sky.Fog.Color.RGB(); int(night)*3 > int(day) {
		t.Errorf("the fog's green is %d at midnight and %d by day: wanted the night darker", night, day)
	}
}

func TestTheClockIsSavedWithTheWorld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := savedGame(t, path)
	startNew(t, g, true)
	g.days = 1.6
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)
	if back.days != 1.6 || back.sky.Time() != timeOfDay(1.6) {
		t.Errorf("the world came back %v days old at %v, wanted 1.6 days at %v", back.days, back.sky.Time(), timeOfDay(1.6))
	}
	text, _ := drawn(t, back)
	if strings.Contains(text, "panic") {
		t.Fatal(text)
	}
}

func TestATorchStandsOnTheGroundAndLightsIt(t *testing.T) {
	g := lookingDown(t)
	hold(t, g, 1, '6')
	if hotbar[g.selected] != world.Torch {
		t.Fatalf("slot 6 holds %v, wanted a torch", hotbar[g.selected].Info().Name)
	}
	hold(t, g, 1, 'e')

	if got := g.world.At(20, 6, 21); got != world.Torch {
		t.Fatalf("the block on the floor is %v, wanted a torch", got.Info().Name)
	}
	if got := g.world.LightAt(20, 7, 21).Block(); got != 13 {
		t.Errorf("over the torch its light is %d, wanted 13", got)
	}
}

func TestATorchIsNotPlacedInWaterOrInTheAir(t *testing.T) {
	g := lookingDown(t)
	g.world.Set(20, 6, 21, world.Water)
	hold(t, g, 1, '6')
	hold(t, g, 1, 'e')
	if got := g.world.At(20, 6, 21); got != world.Water {
		t.Errorf("a torch aimed at water left %v there, wanted the water", got.Info().Name)
	}

	// The side of a block, with nothing under where the torch would go.
	g = onFloor(t)
	g.world.Set(20, 9, 23, world.Stone)
	g.player.Pitch = 0.6
	hold(t, g, 1, '6')
	if !g.aimed || g.target.Face != world.FaceZNeg {
		t.Fatalf("aimed %v at %+v, wanted the near side of the block in the air", g.aimed, g.target)
	}
	hold(t, g, 1, 'e')
	if got := g.world.At(20, 9, 22); got != world.Air {
		t.Errorf("a torch went on the side of a block in the air: %v", got.Info().Name)
	}
}

func TestACreatureInTheDarkIsDrawnDark(t *testing.T) {
	g := onFloor(t)
	// One in the open and one shut in a hut of stone.
	g.fauna.all = []beast{{Pos: geometry.Vector3{X: 10.5, Y: 6, Z: 10.5}}, {Pos: geometry.Vector3{X: 30.5, Y: 6, Z: 30.5}}}
	for y := 6; y <= 8; y++ {
		for z := 29; z <= 31; z++ {
			for x := 29; x <= 31; x++ {
				if y == 8 || x != 30 || z != 30 {
					g.world.Set(x, y, z, world.Stone)
				}
			}
		}
	}
	drawn(t, g)
	if open, shut := g.poses[0].Light, g.poses[1].Light; open < 0.95 || shut >= 0.2 {
		t.Errorf("the creature in the open is lit %v and the one shut away %v: wanted 1 and dark", open, shut)
	}
}

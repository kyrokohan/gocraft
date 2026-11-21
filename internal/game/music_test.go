package game

import (
	"testing"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// nightCamp is floorCamp at midnight, its people seated round the fire, with nothing
// heard yet.
func nightCamp(t *testing.T) (*Game, world.Cell) {
	t.Helper()
	g, fire := floorCamp(t)
	g.days = 1 - firstMorning // midnight
	hold(t, g, seconds(12))
	for _, one := range g.people {
		if !one.seated {
			t.Fatal("at midnight the people of the camp are not seated round the fire")
		}
	}
	hush(g)
	return g, fire
}

// music returns how many drum beats and flute notes have been heard.
func music(g *Game) (drums, flute int) {
	for n, k := range heard(g) {
		switch {
		case n == sound.Drum || n == sound.Tap:
			drums += k
		case n >= sound.Note(0) && n <= sound.Note(sound.Notes-1):
			flute += k
		}
	}
	return drums, flute
}

func TestAtNightTwoOrMoreAtAFireDrumAndPlayAFlute(t *testing.T) {
	g, _ := nightCamp(t)
	swings := map[float64]bool{}
	piped := false
	for range seconds(4) {
		hold(t, g, 1)
		swings[g.people[0].pose().Swing] = true
		piped = piped || g.people[1].pose().Swing > 0
	}
	if drums, flute := music(g); drums < 3 || flute < 2 {
		t.Errorf("in 4 seconds at the fire at night, %d drum beats and %d notes of the flute were heard", drums, flute)
	}
	if len(swings) < 3 || !piped {
		t.Errorf("the drummer's arm took %d places, and the flute was held up: %v", len(swings), piped)
	}
}

func TestByDayOrWithOneAtAFireThereIsNoMusic(t *testing.T) {
	g, _ := floorCamp(t)
	g.days = 0.2 // noon
	hold(t, g, seconds(4))
	hush(g)
	hold(t, g, seconds(4))
	if drums, flute := music(g); drums+flute != 0 {
		t.Errorf("by day %d drum beats and %d notes were heard", drums, flute)
	}

	g, _ = nightCamp(t)
	g.people = g.people[:1]
	hold(t, g, seconds(4))
	if drums, flute := music(g); drums+flute != 0 {
		t.Errorf("with one at the fire, %d drum beats and %d notes were heard", drums, flute)
	}
	if g.people[0].pose().Swing != 0 {
		t.Error("one alone at the fire drums")
	}
}

func TestTheMusicIsHeardNearItsFireAndNotFar(t *testing.T) {
	g, _ := nightCamp(t)
	g.player.Flying = true
	g.player.Pos.Z -= 40
	hush(g)
	hold(t, g, seconds(4))
	if drums, flute := music(g); drums+flute != 0 {
		t.Errorf("47 blocks from the fire, %d drum beats and %d notes were heard", drums, flute)
	}
}

func TestTheTribePlaysAtYourFireToo(t *testing.T) {
	g := inSurvival(t)
	g.world.Set(floorHearth.X, floorHearth.Y, floorHearth.Z, world.Hearth)
	g.people = []person{member(world.Cell{X: 300, Y: 70}, 1), member(world.Cell{X: 600, Y: 70}, 2)}
	g.people[1].seat = 3
	g.days = 1 - firstMorning // midnight
	hold(t, g, seconds(12))
	hush(g)
	hold(t, g, seconds(4))
	if drums, flute := music(g); drums == 0 || flute == 0 {
		t.Errorf("at night at the player's fire, %d drum beats and %d notes were heard", drums, flute)
	}
}

func TestAFireKeepsItsTuneForASeedAndChangesItALittleEachNight(t *testing.T) {
	fire := world.Cell{X: 120, Y: 70, Z: -40}
	one := tuneOf(7, fire, 3)
	if one != tuneOf(7, fire, 3) {
		t.Fatal("a fire's tune came out two ways")
	}
	if one == tuneOf(7, world.Cell{X: 360, Y: 70, Z: -40}, 3) || one == tuneOf(8, fire, 3) {
		t.Error("another fire, or another world, plays the same tune")
	}
	changed := false
	for night := 4; night < 9; night++ {
		other := tuneOf(7, fire, night)
		flute, drums := 0, 0
		for i := range one.flute {
			if one.flute[i] != other.flute[i] {
				flute++
			}
		}
		for i := range one.drums {
			if one.drums[i] != other.drums[i] {
				drums++
			}
		}
		if other.beat != one.beat || flute > 4 || drums > 4 {
			t.Errorf("on night %d the tune has another beat, or %d notes and %d drums of it changed", night, flute, drums)
		}
		changed = changed || flute+drums > 0
	}
	if !changed {
		t.Error("the tune is the same every night")
	}
	if one.beat < 60.0/112/2-1e-9 || one.beat > 60.0/84/2+1e-9 {
		t.Errorf("a step of the tune is %.3f seconds", one.beat)
	}
}

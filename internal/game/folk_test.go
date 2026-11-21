package game

import (
	"math"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// campGame is a game of a seed's world with the player flying 12 blocks from the fire of
// the camp within a walk of where it begins, and the chunks round it loaded.
func campGame(t *testing.T, seed int64) (*Game, world.Cell) {
	t.Helper()
	g, err := New(Options{Seed: seed})
	if err != nil {
		t.Fatal(err)
	}
	g.Play()
	g.begin(false) // creative, so nothing hunts and nothing hurts
	for _, fire := range g.world.Camps(0, 0, 201) {
		if far := math.Hypot(float64(fire.X), float64(fire.Z)); far >= 119 {
			g.player.Pos = geometry.Vector3{X: float64(fire.X) + 12.5, Y: float64(fire.Y) + 3, Z: float64(fire.Z) + 0.5}
			g.player.Flying = true
			g.world.Follow(g.player.Pos.X, g.player.Pos.Z)
			g.world.LoadAll()
			hush(g)
			return g, fire
		}
	}
	t.Fatalf("seed %d has no camp within a walk of where it begins", seed)
	return nil, world.Cell{}
}

// campers returns the people of the camp whose fire is at a cell.
func campers(g *Game, fire world.Cell) []person {
	var them []person
	for _, one := range g.people {
		if one.fire == fire {
			them = append(them, one)
		}
	}
	return them
}

// apartFrom is how far a point is from the middle of a cell, along the ground.
func apartFrom(p geometry.Vector3, c world.Cell) float64 {
	return math.Hypot(p.X-float64(c.X)-0.5, p.Z-float64(c.Z)-0.5)
}

func TestThePeopleOfACampLiveRoundItsFireEachWithALookOfTheirOwn(t *testing.T) {
	g, fire := campGame(t, 1)
	hold(t, g, seconds(3))
	them := campers(g, fire)
	if len(them) < 3 || len(them) > 5 {
		t.Fatalf("the camp at %+v has %d people, wanted 3 to 5", fire, len(them))
	}
	seen := map[int]bool{}
	for _, one := range them {
		if seen[one.look] {
			t.Errorf("two people of the camp have look %d", one.look)
		}
		seen[one.look] = true
		if one.look < 0 || one.look >= assets.PersonLooks {
			t.Errorf("a person has look %d", one.look)
		}
		if far := apartFrom(one.Pos, fire); far > campRoam+2 {
			t.Errorf("a person is %.1f blocks from the camp's fire", far)
		}
	}
	again, _ := campGame(t, 1)
	hold(t, again, seconds(1))
	for i, one := range campers(again, fire) {
		if one.look != them[i].look || one.want != them[i].want {
			t.Errorf("person %d of the camp is another in the same world made again", i)
		}
	}
}

func TestACampWhoseFireIsOutHasNobody(t *testing.T) {
	g, fire := campGame(t, 1)
	g.world.Set(fire.X, fire.Y, fire.Z, world.Air)
	hold(t, g, seconds(2))
	if n := len(campers(g, fire)); n != 0 {
		t.Fatalf("a camp with no fire has %d people", n)
	}

	g.world.Set(fire.X, fire.Y, fire.Z, world.Fire)
	hold(t, g, seconds(2))
	if len(campers(g, fire)) == 0 {
		t.Fatal("the camp has nobody once its fire burns again")
	}
	g.world.Set(fire.X, fire.Y, fire.Z, world.Air)
	hold(t, g, seconds(2))
	if n := len(campers(g, fire)); n != 0 {
		t.Errorf("%d people stay at a camp whose fire was put out", n)
	}
}

func TestPeopleAreLetGoWhenTheirCampIsLeftBehind(t *testing.T) {
	g, fire := campGame(t, 1)
	hold(t, g, seconds(2))
	if len(campers(g, fire)) == 0 {
		t.Fatal("the camp has nobody")
	}
	g.player.Pos.X += 150
	hold(t, g, seconds(2))
	if n := len(campers(g, fire)); n != 0 {
		t.Errorf("%d people are kept of a camp 150 blocks behind", n)
	}
}

// floorCamp is onFloor with a fire on the floor 8 blocks ahead of the player, and the
// people of a camp round it.
func floorCamp(t *testing.T) (*Game, world.Cell) {
	t.Helper()
	g := onFloor(t)
	fire := world.Cell{X: 20, Y: 6, Z: 28}
	g.world.Set(fire.X, fire.Y, fire.Z, world.Fire)
	g.people = campPeople(g.world, fire)
	return g, fire
}

func TestByDayPeopleWalkAboutTheirCampAndByNightTheySitRoundTheFire(t *testing.T) {
	g, fire := floorCamp(t)
	g.days = 0.2 // noon
	walked := false
	for range 30 {
		hold(t, g, seconds(1))
		for _, one := range g.people {
			walked = walked || one.walking
			if far := apartFrom(one.Pos, fire); far > campRoam+1.5 || one.seated {
				t.Fatalf("by day a person is %.1f blocks from the fire, seated %v", far, one.seated)
			}
		}
	}
	if !walked {
		t.Error("in 30 seconds of day nobody walked")
	}

	g.days = 1 - firstMorning // midnight
	hold(t, g, seconds(12))
	for _, one := range g.people {
		if far := apartFrom(one.Pos, fire); !one.seated || math.Abs(far-seatOut) > 0.6 {
			t.Errorf("at night a person is %.1f blocks from the fire, seated %v", far, one.seated)
		}
	}
}

// sayAt counts the pixels of the pale ground a picture is shown on, in a box round the
// place over a person's head where the picture is drawn.
func sayAt(t *testing.T, g *Game, one person) int {
	t.Helper()
	fb, err := engine.NewFrameBuffer(240, 67)
	if err != nil {
		t.Fatal(err)
	}
	g.Draw(fb)
	c := fb.Canvas()
	cam, _ := g.viewed()
	x, y, _, ok := cam.ProjectPixels(one.over(), c)
	if !ok {
		t.Fatal("the place over the person's head is not in view")
	}
	n := 0
	for py := int(y) - 30; py < int(y)+4; py++ {
		for px := int(x) - 16; px < int(x)+16; px++ {
			if c.At(px, py) == sayPale {
				n++
			}
		}
	}
	return n
}

func TestAPersonNearShowsWhatTheyWantAndAWallHidesIt(t *testing.T) {
	g := onFloor(t)
	one := person{Pos: geometry.Vector3{X: 20.5, Y: 6, Z: 24.5}, Yaw: math.Pi, want: world.Amber}
	g.people = []person{one}
	if n := sayAt(t, g, one); n < 10 {
		t.Errorf("a person 4 blocks ahead shows %d pixels of a picture, wanted one", n)
	}

	for y := 6; y < 12; y++ {
		for x := 15; x < 26; x++ {
			g.world.Set(x, y, 22, world.Stone)
		}
	}
	if n := sayAt(t, g, one); n != 0 {
		t.Errorf("a wall between does not hide the picture: %d pixels of it", n)
	}

	g.world = floor()
	g.people[0].Pos.Z = 20.5 + talkReach + 2
	if n := sayAt(t, g, g.people[0]); n != 0 {
		t.Errorf("a person farther than %v blocks shows a picture: %d pixels", talkReach, n)
	}
}

func TestACampFireSmokesAndCrackles(t *testing.T) {
	g, fire := campGame(t, 1)
	hold(t, g, seconds(4))
	over := 0
	for _, b := range g.bits {
		if b.smoke && b.pos.Y > float64(fire.Y)+1.5 && apartFrom(b.pos, fire) < 2 {
			over++
		}
	}
	if over < 3 {
		t.Errorf("%d puffs of smoke stand over the camp's fire after four seconds", over)
	}
	if heard(g)[sound.Crackle] == 0 {
		t.Error("the camp's fire does not crackle")
	}
}

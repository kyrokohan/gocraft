package game

import (
	"math"
	"math/rand/v2"
	"path/filepath"
	"slices"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// The player's bed and hearth on the test floor, 4 blocks apart, away from the camp
// facingPerson makes.
var (
	floorBed    = world.Cell{X: 6, Y: 6, Z: 6}
	floorHearth = world.Cell{X: 6, Y: 6, Z: 10}
)

// askers counts the people who show a fire over their head: who ask to come to the
// player's fire.
func askers(g *Game) int {
	n := 0
	for _, one := range g.people {
		if one.say == asking {
			n++
		}
	}
	return n
}

// ofTribe returns who has come to the player's fire, or is on their way.
func ofTribe(g *Game) []*person {
	var them []*person
	for i := range g.people {
		if g.people[i].tribe {
			them = append(them, &g.people[i])
		}
	}
	return them
}

// asked is facingPerson with an empty hand, and the camp trusting the player enough
// that the person ahead asks to come.
func asked(t *testing.T) (*Game, *person) {
	t.Helper()
	g, one := facingPerson(t, 0, 0)
	g.trust[one.fire] = askTrust
	hold(t, g, seconds(1.1))
	if one.say != asking {
		t.Fatalf("at a trust of %d the person ahead shows %+v", askTrust, one.say)
	}
	return g, one
}

// joined is asked, with a bed and a hearth on the floor, and E said to the one who asks.
func joined(t *testing.T) (*Game, *person) {
	t.Helper()
	g, one := asked(t)
	g.world.Set(floorBed.X, floorBed.Y, floorBed.Z, world.FurBed)
	g.world.Set(floorHearth.X, floorHearth.Y, floorHearth.Z, world.Hearth)
	g.bed = &floorBed
	press(g, 'e')
	if !one.tribe || !one.follows {
		t.Fatal("E with an empty hand, with a bed and a hearth, did not bring the one who asks")
	}
	return g, one
}

// member is one who came to the player's fire from a camp.
func member(from world.Cell, look int) person {
	return person{
		tribe: true, camp: from, fire: floorHearth, look: look,
		Pos: geometry.Vector3{X: 6.5, Y: 6, Z: 13.5}, rng: rand.New(rand.NewPCG(1, uint64(look))),
	}
}

func TestACampThatTrustsYouEnoughAsksOneOfItsPeopleToComeToYourFire(t *testing.T) {
	g, one := facingPerson(t, 0, 0)
	g.trust[one.fire] = askTrust - 1
	hold(t, g, seconds(1.1))
	if n := askers(g); n != 0 {
		t.Fatalf("at a trust of %d, %d ask to come", askTrust-1, n)
	}
	g.trust[one.fire] = askTrust
	hold(t, g, seconds(1.1))
	if n := askers(g); n != 1 || one.say != asking {
		t.Errorf("at a trust of %d, %d ask to come, and the first of the camp shows %+v", askTrust, n, one.say)
	}
}

func TestWithNoFireOneWhoAsksShowsABedAndThenAHearth(t *testing.T) {
	g, one := asked(t)
	press(g, 'e')
	if one.say != (saying{item: world.FurBed.Item()}) || one.tribe {
		t.Fatalf("with no bed, E shows %+v, and they come: %v", one.say, one.tribe)
	}

	g.world.Set(floorBed.X, floorBed.Y, floorBed.Z, world.FurBed)
	g.bed = &floorBed
	g.world.Set(floorBed.X, floorBed.Y, floorBed.Z+homeReach+1, world.Hearth) // too far from the bed
	press(g, 'e')
	if one.say != (saying{item: world.Hearth.Item()}) || one.tribe {
		t.Fatalf("with a bed and no hearth by it, E shows %+v, and they come: %v", one.say, one.tribe)
	}

	g.world.Set(floorHearth.X, floorHearth.Y, floorHearth.Z, world.Hearth)
	press(g, 'e')
	if !one.tribe || !one.follows || one.fire != floorHearth || one.say == asking {
		t.Errorf("with a hearth by the bed, E: they come %v, follow %v, to the fire at %+v, and show %+v", one.tribe, one.follows, one.fire, one.say)
	}
}

func TestOneWhoComesFollowsYouHomeAndLivesThere(t *testing.T) {
	g, one := joined(t)
	g.player.Pos = geometry.Vector3{X: 20.5, Y: 6, Z: 8.5} // 14 blocks back
	hold(t, g, seconds(8))
	if far := math.Hypot(one.Pos.X-g.player.Pos.X, one.Pos.Z-g.player.Pos.Z); far > followFar+1 || !one.follows {
		t.Fatalf("one who follows is %.1f blocks off after 8 seconds, following %v", far, one.follows)
	}

	g.days = 0.2                                           // noon
	g.player.Pos = geometry.Vector3{X: 9.5, Y: 6, Z: 10.5} // by the hearth
	hold(t, g, seconds(10))
	if one.follows {
		t.Fatal("one who followed the player to their fire still follows")
	}
	for range 10 {
		hold(t, g, seconds(1))
		if far := apartFrom(one.Pos, floorHearth); far > campRoam+1.5 || one.seated {
			t.Fatalf("by day one who lives at the fire is %.1f blocks from it, seated %v", far, one.seated)
		}
	}
	g.days = 1 - firstMorning // midnight
	g.player.Pos = geometry.Vector3{X: 20.5, Y: 6, Z: 20.5}
	hold(t, g, seconds(12))
	if far := apartFrom(one.Pos, floorHearth); !one.seated || math.Abs(far-seatOut) > 0.4 {
		t.Errorf("at night one who lives at the fire is %.1f blocks from it, seated %v", far, one.seated)
	}
}

func TestOneAsksAtACampFarFromYourFireAndComes(t *testing.T) {
	g, one := asked(t)
	bed, home := world.Cell{X: 500, Y: 6, Z: 500}, world.Cell{X: 500, Y: 6, Z: 504}
	g.bed, g.home = &bed, &home // seen when the player was there; that ground is not loaded now
	press(g, 'e')
	if !one.tribe || one.fire != home {
		t.Errorf("at a camp far from the player's fire, E: they come %v, to the fire at %+v, and show %+v", one.tribe, one.fire, one.say)
	}
}

func TestOneLeftFarBehindIsAtYourFireByTheNextMorning(t *testing.T) {
	g, _ := joined(t)
	g.player.Flying = true
	g.player.Pos.X += 150
	hold(t, g, seconds(1.5))
	them := ofTribe(g)
	if len(them) != 1 || them[0].away == 0 {
		t.Fatal("one who follows is not on their own way once 150 blocks behind")
	}
	morning := them[0].away
	if morning <= g.days || morning > g.days+1 || math.Abs(timeOfDay(morning)-render.Sunrise) > 1e-9 {
		t.Fatalf("one left behind arrives at day %.3f, after day %.3f", morning, g.days)
	}

	g.player.Pos.X -= 150
	g.days = morning - 0.01
	hold(t, g, seconds(1.5))
	if one := ofTribe(g)[0]; one.away == 0 {
		t.Fatal("one left behind came before the morning")
	}
	g.days = morning + 0.001
	hold(t, g, seconds(1.5))
	if one := ofTribe(g)[0]; one.away != 0 || one.follows || apartFrom(one.Pos, floorHearth) > seatOut+1 {
		t.Errorf("by the morning one left behind is %.1f blocks from the fire, away %v, following %v", apartFrom(one.Pos, floorHearth), one.away, one.follows)
	}
}

func TestNoMoreThanSixComeToYourFireAndNoMoreThanTwoFromACamp(t *testing.T) {
	g, one := facingPerson(t, 0, 0)
	camp := one.fire
	g.people = append(g.people, member(camp, 15))
	g.trust[camp] = 2*askTrust - 1
	hold(t, g, seconds(1.1))
	if n := askers(g); n != 0 {
		t.Fatalf("one has come from the camp, and at a trust of %d, %d more ask", 2*askTrust-1, n)
	}
	g.trust[camp] = 2 * askTrust
	hold(t, g, seconds(1.1))
	if n := askers(g); n != 1 {
		t.Fatalf("one has come from the camp, and at a trust of %d, %d more ask", 2*askTrust, n)
	}

	g.people = append(g.people, member(camp, 14))
	g.trust[camp] = 1000
	hold(t, g, seconds(1.1))
	if n := askers(g); n != 0 {
		t.Fatalf("two have come from the camp, and %d more ask", n)
	}

	g.people = slices.DeleteFunc(g.people, func(one person) bool { return one.tribe })
	for i := range maxTribe {
		g.people = append(g.people, member(world.Cell{X: 300 * (i + 1), Y: 70, Z: 0}, i))
	}
	hold(t, g, seconds(1.1))
	if n := askers(g); n != 0 {
		t.Fatalf("%d live at the fire, and %d more ask", maxTribe, n)
	}
	g.people = g.people[:len(g.people)-1]
	hold(t, g, seconds(1.1))
	if n := askers(g); n != 1 {
		t.Errorf("%d live at the fire, and %d more ask", maxTribe-1, n)
	}
}

func TestOneWhoCameToYourFireIsNotMadeAtTheirCampAgain(t *testing.T) {
	g, fire := campGame(t, 1)
	hold(t, g, seconds(2))
	before := campers(g, fire)
	i := slices.IndexFunc(g.people, func(one person) bool { return one.fire == fire })
	look := g.people[i].look
	g.join(i, world.Cell{X: fire.X + 40, Y: fire.Y, Z: fire.Z})

	g.player.Pos.X += 150
	hold(t, g, seconds(2))
	g.player.Pos.X -= 150
	hold(t, g, seconds(2))
	after := campers(g, fire)
	if len(after) != len(before)-1 || slices.ContainsFunc(after, func(one person) bool { return one.look == look }) {
		t.Errorf("the camp had %d people, and once one came away it has %d, the one who came among them", len(before), len(after))
	}
	if n := len(ofTribe(g)); n != 1 {
		t.Errorf("%d have come to the player's fire, wanted 1", n)
	}
}

func TestASaveKeepsWhoCameToYourFire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := inSurvival(t)
	g.opts.SavePath = path
	home, camp := floorHearth, world.Cell{X: -120, Y: 70, Z: 44}
	g.home = &home
	follower, lost := member(camp, 3), member(camp, 9)
	follower.follows, follower.Pos, follower.Yaw = true, geometry.Vector3{X: 18.5, Y: 6, Z: 20.5}, 1
	lost.follows, lost.away = true, 3.25
	g.people = append(campPeople(g.world, world.Cell{X: 20, Y: 6, Z: 28}), member(camp, 7), follower, lost)
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)
	if back.home == nil || *back.home != home {
		t.Errorf("the player's fire came back as %v", back.home)
	}
	them := ofTribe(back)
	if len(back.people) != 3 || len(them) != 3 {
		t.Fatalf("%d people came back, %d of them of the tribe; wanted the 3 of the tribe", len(back.people), len(them))
	}
	for i, want := range []person{member(camp, 7), follower, lost} {
		got := them[i]
		if got.camp != want.camp || got.look != want.look || got.Pos != want.Pos || got.Yaw != want.Yaw ||
			got.follows != want.follows || got.away != want.away || got.fire != home {
			t.Errorf("one of the tribe came back as %+v, wanted %+v", *got, want)
		}
	}
}

func TestANewWorldHasNoneOfTheTribeOfTheLast(t *testing.T) {
	g, _ := joined(t)
	g.newWorld(5)
	if len(g.people) != 0 || g.home != nil {
		t.Errorf("a new world begins with %d people and the fire %v of the last", len(g.people), g.home)
	}
}

func TestALitHearthIsYourFireToo(t *testing.T) {
	g := inSurvival(t)
	g.world.Set(floorBed.X, floorBed.Y, floorBed.Z, world.FurBed)
	g.world.Set(floorHearth.X, floorHearth.Y, floorHearth.Z, world.HearthLit)
	g.bed = &floorBed
	if home := g.homeFire(); home == nil || *home != floorHearth {
		t.Errorf("with a lit hearth by the bed, the player's fire is %v", home)
	}
}

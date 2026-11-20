package game

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// arrival is the day of the world on which the first Skyspear arrives: the midnight of
// its night.
const arrival = cometNight - firstMorning

// fallAt sets the clock of a game to so many seconds after the first Great Fall began:
// fewer than none is before it.
func fallAt(g *Game, into float64) {
	g.days = arrival + (into-fallLasts/2)/dayLength
	g.fallen = g.days
}

func TestTheGreatFallComesRoundTheMidnightTheSkyspearArrivesAndNotBefore(t *testing.T) {
	g := underStars(t)
	for _, tc := range []struct {
		days float64
		on   bool
	}{
		{5 - firstMorning, false},       // the midnight of another night
		{arrival - 0.2, false},          // the evening of its night
		{arrival - 91/dayLength, false}, // a second before
		{arrival - 89/dayLength, true},
		{arrival, true},
		{arrival + 89/dayLength, true},
		{arrival + 91/dayLength, false}, // over
		{arrival + 1, false},
		{arrival + cometEvery, true}, // and the next
	} {
		g.days = tc.days
		if since := g.greatFall(); (since >= 0 && since < fallLasts) != tc.on {
			t.Errorf("on day %.3f the Great Fall began %.0f s ago: wanted it on: %v", tc.days, since, tc.on)
		}
	}
	// While it falls the comet is all here, and when the Skyspear has struck it is gone.
	fallAt(g, fallLasts-1)
	if near, _ := g.comet(); near != 1 {
		t.Errorf("a second before the Skyspear strikes the comet is %.2f near, wanted all of it", near)
	}
	fallAt(g, fallLasts+1)
	if near, _ := g.comet(); near > 0.01 {
		t.Errorf("a second after the Skyspear struck the comet is %.2f near, wanted it gone", near)
	}
}

func TestInTheGreatFallStarsComeDownThickAndNoneNearTheBed(t *testing.T) {
	g := underStars(t)
	p := g.player.Pos
	g.bed = &world.Cell{X: int(math.Floor(p.X)) + 30, Y: int(p.Y), Z: int(math.Floor(p.Z))}

	// A minute before, on the night itself: it is quiet.
	fallAt(g, -60)
	hold(t, g, seconds(5))
	if starInSky(g) || len(cores(g)) != 0 {
		t.Fatalf("a minute before the Great Fall a star is in the sky, or %d have struck", len(cores(g)))
	}

	fallAt(g, -1)
	most, fell, last := 0, 0, 0
	for range seconds(fallLasts + 8) {
		hold(t, g, 1)
		n := 0
		for _, m := range g.missiles {
			if fliers[m.kind].falls {
				n++
			}
		}
		if n > last {
			fell += n - last // a new one: no two strike in the frame another comes
		}
		most, last = max(most, n), n
	}
	if fell < 80 || fell > 91 {
		t.Errorf("%d stars fell in the three minutes of the Great Fall, wanted one every two seconds", fell)
	}
	if starInSky(g) {
		t.Error("a star is still in the sky seven seconds after the Great Fall was over")
	}
	if most < 2 || most > 4 {
		t.Errorf("at the most %d stars were in the sky at once, wanted two or three", most)
	}
	struck := cores(g)
	if len(struck) < 45 || len(struck) > fell {
		t.Errorf("%d of the %d stars left craters, wanted half of them or more", len(struck), fell)
	}
	nearest := math.Inf(1)
	for _, c := range struck {
		mid := cellMid(c.X, c.Y, c.Z)
		if far := flatFrom(mid, *g.bed); far < bedSafe {
			t.Errorf("a star struck %.0f blocks from the bed, wanted none within %v", far, bedSafe)
		}
		if far := math.Hypot(mid.X-p.X, mid.Z-p.Z); far > fallFar+25 {
			t.Errorf("a star struck %.0f blocks from the player, wanted none past %v or a little more", far, fallFar)
		} else {
			nearest = min(nearest, far)
		}
	}
	if nearest > starNear {
		t.Errorf("the nearest star of the Great Fall struck %.0f blocks from the player: wanted them nearer than the %v of a lone star", nearest, starNear)
	}
}

// cellar is a game in survival on a block of stone nine deep, with a room two high dug
// in it at 30, 30 under so many blocks of roof, the player standing in the room, and a
// star struck on the roof over it.
func cellar(t *testing.T, roof int) *Game {
	t.Helper()
	g := among(t, 0.5)
	for y := range 10 {
		for z := 20; z <= 40; z++ {
			for x := 20; x <= 40; x++ {
				g.world.Set(x, y, z, world.Stone)
			}
		}
	}
	top := 9
	feet := top - roof - 1
	g.world.Set(30, feet, 30, world.Air)
	g.world.Set(30, feet+1, 30, world.Air)
	stand(g, geometry.Vector3{X: 30.5, Y: float64(feet), Z: 30.5})
	g.impact(geometry.Vector3{X: 30.5, Y: float64(top) + 1.5, Z: 30.5})
	return g
}

func TestThreeBlocksOfStoneOverheadKeepAStarOff(t *testing.T) {
	g := cellar(t, 3)
	if g.health != maxHealth {
		t.Errorf("a star that struck the roof took %v points from a player under three blocks of stone", maxHealth-g.health)
	}
	if g.world.At(30, 9, 30) != world.Air || g.world.At(30, 7, 30) != world.Air {
		t.Errorf("the roof over the room is %v at its top and %v at its bottom: wanted the star to have broken it", g.world.At(30, 9, 30), g.world.At(30, 7, 30))
	}
	// It broke through: its core is not in the air of the room, but in the floor.
	if air, floor := g.world.At(30, 6, 30), g.world.At(30, 4, 30); air != world.Air || floor != world.HotCore || len(cores(g)) != 1 {
		t.Errorf("in the room, under the hole, is %v, and in its floor %v, with %d cores kept: wanted air, and the core in the floor", air, floor, len(cores(g)))
	}

	if thin := cellar(t, 2); maxHealth-thin.health != blastHarm {
		t.Errorf("a star took %v points from a player under two blocks of stone, wanted %v", maxHealth-thin.health, blastHarm)
	}
	// A beast is as safe as a person, and as hurt.
	g = among(t, 0.5, giant(render.Bugler, 30.5, 30.5))
	g.fauna.all[0].Pos.Y = 1 // four under the plain, as in a cave
	g.impact(geometry.Vector3{X: 30.5, Y: 6.5, Z: 30.5})
	if c := g.fauna.all[0]; c.health != breeds[render.Bugler].health {
		t.Errorf("a star took %v points from a bugler four blocks under where it struck", breeds[render.Bugler].health-c.health)
	}
}

func TestBeastsRunFromWhereAStarStrikes(t *testing.T) {
	g := among(t, 0.5, giant(render.Mossback, 30.5, 50.5), giant(render.Mossback, 30.5, -30.5))
	g.impact(geometry.Vector3{X: 30.5, Y: 6.5, Z: 30.5})
	near, far := g.fauna.all[0], g.fauna.all[1]
	if near.flee <= 0 || facing(near, geometry.Vector3{X: 30.5, Y: 6, Z: 30.5}) < 2 {
		t.Errorf("a mossback twenty blocks from where a star struck runs for %.1f s, turned %.1f from it: wanted it to run away", near.flee, facing(near, geometry.Vector3{X: 30.5, Y: 6, Z: 30.5}))
	}
	if far.flee > 0 {
		t.Errorf("a mossback sixty blocks from where a star struck runs for %.1f s", far.flee)
	}
}

func TestTheSkyspearStrikesFarOffAndAshClimbsTheSky(t *testing.T) {
	g := underStars(t)
	g.begin(true)
	ash := func() float64 {
		a, _ := g.ash()
		return a
	}
	fallAt(g, fallLasts-2)
	hold(t, g, seconds(1))
	g.missiles, g.booms = nil, nil // the stars of the Great Fall are not what is looked at
	hush(g)
	if g.white != 0 || ash() != 0 {
		t.Fatalf("a second before the Skyspear strikes the sky is white by %.2f and holds %.2f of ash", g.white, ash())
	}
	hold(t, g, seconds(1.5))
	if g.white <= 0 || g.health != maxHealth {
		t.Fatalf("half a second after the Skyspear struck the sky is white by %.2f, and the player has lost %v points", g.white, maxHealth-g.health)
	}
	drawn(t, g) // with the sky white

	// Its boom is twenty seconds on its way, and the ground shakes hard when it comes.
	var great []boom
	for _, b := range g.booms {
		if b.great {
			great = append(great, b)
		}
	}
	if g.booms, g.missiles = great, nil; len(great) != 1 {
		t.Fatalf("the Skyspear struck and %d great booms are on their way", len(great))
	}
	hold(t, g, seconds(skyspearBoom-2))
	if heard(g)[sound.Boom] != 0 || g.white != 0 {
		t.Errorf("%v s after the Skyspear struck, %d booms were heard and the sky is white by %.2f", skyspearBoom-1.5, heard(g)[sound.Boom], g.white)
	}
	g.quake = 0
	hold(t, g, seconds(2.5))
	if heard(g)[sound.Boom] == 0 || g.quake < 0.8 {
		t.Errorf("%v s after the Skyspear struck, %d booms were heard and the ground shakes by %.2f: wanted its boom, and a hard shake", skyspearBoom+1, heard(g)[sound.Boom], g.quake)
	}
	if g.health != maxHealth {
		t.Errorf("the boom of the Skyspear took %v points from the player", maxHealth-g.health)
	}
	hold(t, g, seconds(shockTime+2))
	if g.quake > 0.05 {
		t.Errorf("the ground still shakes by %.2f, %v s after the boom", g.quake, shockTime+3)
	}

	// Ash climbs the sky in a minute, stays for the winter, and clears after it.
	for _, tc := range []struct {
		after    float64 // seconds after the Skyspear struck
		low, top float64
	}{
		{10, 0.02, 0.3},
		{ashClimb + 5, winterAsh, winterAsh},
		{2 * dayLength, winterAsh, winterAsh},
		{winterDays*dayLength - 5, winterAsh, winterAsh},
		{(winterDays + ashAfter/2) * dayLength, 0.45 * winterAsh, 0.55 * winterAsh},
		{(winterDays+ashAfter)*dayLength + 5, 0, 0},
	} {
		fallAt(g, fallLasts+tc.after)
		if a := ash(); a < tc.low-1e-9 || a > tc.top+1e-9 {
			t.Errorf("%.0f s after the Skyspear struck the sky holds %.2f of ash, wanted %.2f to %.2f", tc.after, a, tc.low, tc.top)
		}
	}
	fallAt(g, fallLasts+2*dayLength)
	drawn(t, g)
	if g.sky.Fog.Color == render.NewSky(timeOfDay(g.days)).Fog.Color {
		t.Error("two days after the Great Fall the frame is as clear as it ever was")
	}
}

func TestThePlayerCannotSleepThroughTheGreatFall(t *testing.T) {
	g := underStars(t)
	g.begin(true)
	bed := world.Cell{X: int(math.Floor(g.player.Pos.X)), Y: int(g.player.Pos.Y), Z: int(math.Floor(g.player.Pos.Z)) + 1}
	lie := func(days float64) (slept bool, said string) {
		g.days, g.bed = days, nil
		g.lieDown(bed)
		return g.days != days, g.notice
	}
	if slept, said := lie(arrival - 0.2); slept || !strings.Contains(said, "Skyspear") {
		t.Errorf("in the evening of the night the Skyspear arrives the player slept: %v, and was told %q", slept, said)
	}
	if slept, said := lie(arrival + 30/dayLength); slept || !strings.Contains(said, "Skyspear") {
		t.Errorf("in the Great Fall the player slept: %v, and was told %q", slept, said)
	}
	if slept, _ := lie(arrival + 120/dayLength); !slept {
		t.Error("with the Great Fall over the player cannot sleep")
	}
	if slept, _ := lie(arrival - 1); !slept {
		t.Error("the night before the Skyspear arrives the player cannot sleep")
	}
}

func TestAGreatFallSavedHalfWayGoesOnAndOneThatIsOverDoesNotComeAgain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := underStars(t)
	g.opts.SavePath = path
	fallAt(g, 60)
	hold(t, g, seconds(20))
	before := len(cores(g))
	if before < 5 {
		t.Fatalf("twenty seconds of the Great Fall left %d craters", before)
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)
	back.Play()
	if len(cores(back)) != before {
		t.Fatalf("the world came back with %d cores kept, wanted %d", len(cores(back)), before)
	}
	hold(t, back, seconds(20))
	if more := len(cores(back)) - before; more < 5 {
		t.Errorf("twenty seconds after it was opened %d more stars had struck: wanted the Great Fall to go on", more)
	}

	// Opened the day after: it is over, and no star falls by day.
	back.days = arrival + 0.5
	if err := back.Save(); err != nil {
		t.Fatal(err)
	}
	after := savedGame(t, path)
	after.Play()
	n := len(cores(after))
	hold(t, after, seconds(10))
	if starInSky(after) || len(cores(after)) != n || after.white != 0 {
		t.Errorf("a world opened the day after the Great Fall has a star in the sky: %v, %d more craters, and a sky white by %.2f", starInSky(after), len(cores(after))-n, after.white)
	}
}

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

// underStars is a game in creative in the world of seed 7, where it begins, with no
// beasts and none to come, and the land brought in round its player.
func underStars(t *testing.T) *Game {
	t.Helper()
	g, err := New(Options{Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	g.Play()
	g.begin(false)
	g.fauna.all, g.fauna.timer = nil, math.Inf(1)
	g.world.LoadAll()
	hush(g)
	return g
}

// fallNight returns the first night from a night on that a star falls on, and the day
// of the world it falls at.
func fallNight(t *testing.T, g *Game, from int) (night int, when float64) {
	t.Helper()
	for night = from; night < from+60; night++ {
		if when, ok := g.starFall(night); ok {
			return night, when
		}
	}
	t.Fatalf("no star falls in the sixty nights from night %d", from)
	return 0, 0
}

// cores returns the cells of the hot cores that are kept to cool.
func cores(g *Game) (at []world.Cell) {
	for _, e := range g.embers {
		if e.Into == world.SkyIronCore {
			at = append(at, e.At)
		}
	}
	return at
}

// starInSky reports whether a star is on its way down.
func starInSky(g *Game) bool {
	for _, m := range g.missiles {
		if fliers[m.kind].falls {
			return true
		}
	}
	return false
}

func TestAStarFallsOnlyByNightAtAMomentTheSeedGives(t *testing.T) {
	g := underStars(t)
	falls := 0
	for night := 1; night <= 300; night++ {
		when, ok := g.starFall(night)
		if !ok {
			continue
		}
		falls++
		if at := timeOfDay(when); at > 0.16 && at < 0.84 {
			t.Fatalf("on night %d a star falls at %.2f of the day, wanted in the dark", night, at)
		}
		if again, _ := g.starFall(night); again != when || int(math.Floor(when+firstMorning+0.5)) != night {
			t.Fatalf("the star of night %d falls on day %.3f, and then on day %.3f", night, when, again)
		}
	}
	if falls < 70 || falls > 130 {
		t.Errorf("a star falls on %d of 300 nights, wanted about one in three", falls)
	}

	// F3 says when the next one does.
	g.debug = true
	if text, _ := drawn(t, g); !strings.Contains(text, "star    falls in ") {
		t.Errorf("the frame numbers do not say when a star falls next:\n%s", text)
	}
}

func TestAStarComesDownFortyBlocksOffOrMoreAndNeverNearTheBed(t *testing.T) {
	g := underStars(t)
	p := g.player.Pos
	g.bed = &world.Cell{X: int(math.Floor(p.X)) + 50, Y: int(p.Y), Z: int(math.Floor(p.Z))}
	night, when := fallNight(t, g, 1)

	// By day, and on the night before its moment, none falls.
	g.days = when - 0.3
	g.fallen = g.days
	hold(t, g, seconds(3))
	if starInSky(g) || len(cores(g)) != 0 {
		t.Fatalf("a star falls on day %.2f, and the star of night %d is to fall on day %.2f", g.days, night, when)
	}

	// The same moment comes eight times: each star picks its own place.
	places := map[world.Cell]bool{}
	for i := range 8 {
		g.days = when - 1/dayLength
		g.fallen = g.days
		before, flew, bright := len(cores(g)), false, 0.0
		for range seconds(starFlight + 3) {
			hold(t, g, 1)
			_, _, ball := g.fireball()
			flew, bright = flew || starInSky(g), max(bright, ball)
		}
		if !flew || bright < 1 || starInSky(g) {
			t.Fatalf("star %d: one was seen in the sky: %v, at a brightness of %.2f, and one is still there: %v", i, flew, bright, starInSky(g))
		}
		all := cores(g)
		if len(all) != before+1 {
			t.Fatalf("star %d left %d cores, wanted one", i, len(all)-before)
		}
		core := all[len(all)-1]
		places[core] = true
		if b := g.world.At(core.X, core.Y, core.Z); b != world.HotCore {
			t.Errorf("star %d: where its core is kept lies %v", i, b)
		}
		mid := cellMid(core.X, core.Y, core.Z)
		if far := math.Hypot(mid.X-p.X, mid.Z-p.Z); far < starNear-2 || far > starFar+25 {
			t.Errorf("star %d struck %.0f blocks from the player, wanted %v to %v or a little more", i, far, starNear, starFar)
		}
		if far := flatFrom(mid, *g.bed); far < bedSafe {
			t.Errorf("star %d struck %.0f blocks from the bed, wanted none within %v", i, far, bedSafe)
		}
	}
	if len(places) < 7 {
		t.Errorf("eight stars struck in %d places", len(places))
	}

	// A night slept through: its star is not seen in the morning.
	g.days = when - 1/dayLength
	g.fallen = g.days
	g.days = float64(night) - firstMorning + 0.4
	hold(t, g, seconds(2))
	if starInSky(g) {
		t.Error("a star whose moment passed in the night falls in the morning")
	}
	// The clock moved on a quarter of a day past its moment, and it is still night: it falls.
	g.days = when - 0.1
	g.fallen = g.days
	g.days = when + 0.02
	hold(t, g, 2)
	if !starInSky(g) {
		t.Error("a star whose moment the clock was moved past, in the night, does not fall")
	}
}

// crater is a game in survival on a deep floor of sand, with a star struck at 30.5, 30.5
// on it, and the cell of the ground the star met. Three columns of the floor, on one
// side of the middle, are stone.
func crater(t *testing.T) (*Game, world.Cell) {
	t.Helper()
	g := among(t, 0.5)
	for y := range 6 {
		for z := 18; z <= 42; z++ {
			for x := 18; x <= 42; x++ {
				g.world.Set(x, y, z, world.Sand)
			}
		}
	}
	for y := range 6 {
		for x := 27; x <= 29; x++ {
			g.world.Set(x, y, 30, world.Stone) // but for a strip of stone on one side
		}
	}
	g.impact(geometry.Vector3{X: 30.5, Y: 6.5, Z: 30.5})
	return g, world.Cell{X: 30, Y: 5, Z: 30}
}

func TestTheBoomOfAStarComesASecondLaterForEachFortyBlocks(t *testing.T) {
	for _, far := range []float64{40, 72} {
		g := among(t, 0.5)
		g.impact(geometry.Vector3{X: 0.5, Y: 6.5, Z: 0.5 + far})
		if _, _, flash := g.fireball(); flash <= 0 {
			t.Errorf("a star struck %v blocks off and no flash is seen at once", far)
		}
		wait := far / boomPace
		hold(t, g, seconds(wait-0.2))
		if heard(g)[sound.Boom] != 0 || g.quake != 0 {
			t.Errorf("the boom of a star %v blocks off was heard before %.1f s were up", far, wait)
		}
		hold(t, g, seconds(0.4))
		if heard(g)[sound.Boom] != 1 || g.quake <= 0 {
			t.Errorf("%.1f s after a star struck %v blocks off, %d booms were heard and the ground shakes by %.2f", wait+0.2, far, heard(g)[sound.Boom], g.quake)
		}
		hold(t, g, seconds(2))
		if _, _, flash := g.fireball(); flash != 0 || heard(g)[sound.Boom] != 1 {
			t.Errorf("two seconds on the flash is %.2f and %d booms were heard", flash, heard(g)[sound.Boom])
		}
	}
}

func TestAStarLeavesACraterWithGlassFiresAndAHotCoreOfSkyIron(t *testing.T) {
	g, c := crater(t)
	at := func(dx, dy, dz int) world.Block { return g.world.At(c.X+dx, c.Y+dy, c.Z+dz) }

	// A bowl seven across and three deep, with the core under its middle.
	for _, open := range [][3]int{{0, 0, 0}, {0, -1, 0}, {0, -2, 0}, {3, 0, 0}, {-3, 0, 0}, {0, 0, 3}, {0, 0, -3}, {2, -1, 2}, {2, -2, 0}} {
		if b := at(open[0], open[1], open[2]); b != world.Air {
			t.Errorf("%v from the middle of the crater lies %v, wanted the bowl open there", open, b)
		}
	}
	for _, whole := range [][3]int{{5, 0, 0}, {-5, 0, 0}, {0, 0, 5}, {4, 0, 4}, {0, -4, 0}, {3, -3, 0}} {
		if b := at(whole[0], whole[1], whole[2]); !b.Info().Solid {
			t.Errorf("%v from the middle of the crater lies %v, wanted the ground whole there", whole, b)
		}
	}
	if b := at(0, -3, 0); b != world.HotCore {
		t.Fatalf("under the middle of the crater lies %v, wanted the hot core", b)
	}
	// The sand of the bowl is glass, and the sand away from it is sand.
	glass := 0
	for dz := -4; dz <= 4; dz++ {
		for dx := -4; dx <= 4; dx++ {
			for dy := -4; dy <= 0; dy++ {
				if at(dx, dy, dz) == world.Glass {
					glass++
				}
			}
		}
	}
	if glass < 30 || at(2, -3, 0) != world.Glass || at(8, 0, 0) != world.Sand || at(0, -5, 0) != world.Sand {
		t.Errorf("%d blocks of the bowl are glass; its floor is %v, the ground eight off %v, and deep under it %v", glass, at(2, -3, 0), at(8, 0, 0), at(0, -5, 0))
	}
	// Where the floor of the bowl was stone it is burnt to dark rock.
	if at(-2, -3, 0) != world.Basalt || at(-3, -2, 0) != world.Basalt || at(-2, -4, 0) != world.Stone {
		t.Errorf("the stone floor of the bowl is %v and %v, and the stone under it %v: wanted dark rock over stone", at(-2, -3, 0), at(-3, -2, 0), at(-2, -4, 0))
	}
	// Fires stand round it, and they and the core are kept.
	fires := 0
	for _, e := range g.embers {
		switch b := g.world.At(e.At.X, e.At.Y, e.At.Z); {
		case e.Into == world.SkyIronCore:
			if e.At != (world.Cell{X: c.X, Y: c.Y - 3, Z: c.Z}) || math.Abs(e.Left-coreHot) > 1 {
				t.Errorf("the core is kept at %v with %.0f s to cool, wanted under the crater with %v", e.At, e.Left, coreHot)
			}
		case b != world.Fire || e.Into != world.Air || math.Abs(e.Left-starFire) > 1:
			t.Errorf("a fire of the crater is %v at %v, with %.0f s left, to turn into %v", b, e.At, e.Left, e.Into)
		default:
			fires++
			if far := flatFrom(cellMid(e.At.X, e.At.Y, e.At.Z), c); far < bowlWide || far > bowlWide+3 {
				t.Errorf("a fire of the crater burns %.1f blocks from its middle, wanted on its rim", far)
			}
		}
	}
	if fires != craterFires || len(cores(g)) != 1 {
		t.Errorf("%d fires burn round the crater with %d cores kept, wanted %d and one", fires, len(cores(g)), craterFires)
	}

	// The core burns what stands on it, lights a torch, and cools in two minutes.
	stand(g, geometry.Vector3{X: 30.5, Y: float64(c.Y - 2), Z: 30.5})
	g.health, g.food = maxHealth, mendFood-1
	hold(t, g, seconds(1))
	if lost := maxHealth - g.health; lost < fireHurt-0.5 || lost > fireHurt+0.5 {
		t.Errorf("a second on the hot core cost %.1f points, wanted about %v", lost, fireHurt)
	}
	if !burns(world.HotCore) || burns(world.SkyIronCore) {
		t.Error("a torch is not lit at a hot core, or is lit at a dark one")
	}
	stand(g, geometry.Vector3{X: 0.5, Y: 6, Z: 0.5})
	hold(t, g, seconds(coreHot-3))
	if at(0, -3, 0) != world.HotCore {
		t.Errorf("the core went dark before its %v seconds were up", coreHot)
	}
	hold(t, g, seconds(3))
	if at(0, -3, 0) != world.SkyIronCore || len(cores(g)) != 0 || len(g.embers) != craterFires {
		t.Errorf("%v seconds on the core is %v, with %d cores and %d fires kept", coreHot+1, at(0, -3, 0), len(cores(g)), len(g.embers))
	}
	// Its fires burn for a day.
	hold(t, g, seconds(starFire-coreHot-5))
	if len(g.embers) != craterFires {
		t.Errorf("%d fires of the crater burn near the end of their day, wanted all %d", len(g.embers), craterFires)
	}
	hold(t, g, seconds(8))
	if len(g.embers) != 0 {
		t.Errorf("%d fires of the crater still burn after their day", len(g.embers))
	}
}

func TestAStarHurtsWhatItStrikesAndSparesWhatMustBeKept(t *testing.T) {
	// A beast and a player in the blast.
	g := among(t, 0.5, giant(render.Bugler, 30.5, 32.5), giant(render.Bugler, 30.5, 40.5))
	stand(g, geometry.Vector3{X: 33.5, Y: 6, Z: 30.5})
	g.impact(geometry.Vector3{X: 30.5, Y: 6.5, Z: 30.5})
	if lost := maxHealth - g.health; lost != blastHarm {
		t.Errorf("a star that struck three blocks from the player took %v points, wanted %v", lost, blastHarm)
	}
	if near, far := g.fauna.all[0], g.fauna.all[1]; near.health != breeds[render.Bugler].health-blastHarm || far.health != breeds[render.Bugler].health {
		t.Errorf("a bugler two blocks from the star has %v points left and one ten off %v", near.health, far.health)
	}

	// Near the bed, a pot or a hearth it burns up: a flash and a boom, and no crater.
	for name, keep := range map[string]func(g *Game){
		"the bed":  func(g *Game) { g.bed = &world.Cell{X: 40, Y: 6, Z: 30} },
		"a pot":    func(g *Game) { g.pot(world.Cell{X: 35, Y: 6, Z: 30}) },
		"a hearth": func(g *Game) { g.hearth(world.Cell{X: 30, Y: 6, Z: 36}) },
	} {
		g := among(t, 0.5)
		keep(g)
		g.impact(geometry.Vector3{X: 30.5, Y: 6.5, Z: 30.5})
		if _, _, flash := g.fireball(); g.world.At(30, 5, 30) != world.Stone || len(g.embers) != 0 || flash <= 0 || len(g.booms) != 1 {
			t.Errorf("a star that came down by %s left %v where it struck and %d fires, with a flash of %.2f and %d booms to come", name, g.world.At(30, 5, 30), len(g.embers), flash, len(g.booms))
		}
	}
}

func TestACraterSavedWithItsCoreHotComesBackAndTheCoreStillCools(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := underStars(t)
	g.opts.SavePath = path
	p := g.player.Pos
	x, z := int(math.Floor(p.X))+12, int(math.Floor(p.Z))
	g.impact(cellMid(x, g.world.Ground(x, z)+1, z))
	all := cores(g)
	if len(all) != 1 {
		t.Fatalf("a star that struck twelve blocks from where the world begins left %d cores", len(all))
	}
	hold(t, g, seconds(20))
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	back := savedGame(t, path)
	back.Play()
	if got := cores(back); len(got) != 1 || got[0] != all[0] || len(back.embers) != len(g.embers) {
		t.Fatalf("the world came back with its cores at %v and %d things kept, wanted %v and %d", got, len(back.embers), all, len(g.embers))
	}
	hold(t, back, seconds(coreHot-25))
	if b := back.world.At(all[0].X, all[0].Y, all[0].Z); b != world.HotCore {
		t.Errorf("the core came back as %v, wanted it hot still", b)
	}
	hold(t, back, seconds(10))
	if b := back.world.At(all[0].X, all[0].Y, all[0].Z); b != world.SkyIronCore {
		t.Errorf("%v seconds after it struck, across a save, the core is %v", coreHot+5, b)
	}
}

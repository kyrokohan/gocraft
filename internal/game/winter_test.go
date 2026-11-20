package game

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// winterAt is a game in survival where the world of seed 7 begins, with no beasts and
// none to come, so many days after the Skyspear struck: half a day is the noon of the
// first day of the winter, and one day its first midnight.
func winterAt(t *testing.T, days float64) *Game {
	t.Helper()
	g := underStars(t)
	g.begin(true)
	fallAt(g, fallLasts+days*dayLength)
	return g
}

func TestInTheLongWinterWarmthIsGoneInTwoMinutesAndThenHealthGoes(t *testing.T) {
	g := winterAt(t, 0.5)
	if !g.winter() || !g.cold() || g.warmth != maxWarmth {
		t.Fatalf("at the first noon after the Great Fall it is winter: %v, cold: %v, and the player has %v of warmth", g.winter(), g.cold(), g.warmth)
	}
	hold(t, g, seconds(coldTime/2))
	if math.Abs(g.warmth-maxWarmth/2) > 0.5 || g.health != maxHealth {
		t.Errorf("after a minute of the cold the player has %.1f of warmth and has lost %v points of health: wanted half its warmth, and no hurt", g.warmth, maxHealth-g.health)
	}
	hold(t, g, seconds(coldTime/2+1))
	if g.warmth != 0 || maxHealth-g.health > 1 {
		t.Errorf("after two minutes of the cold the player has %.1f of warmth and has lost %v points of health", g.warmth, maxHealth-g.health)
	}
	// With none left the cold takes a point every four seconds, though the player is fed.
	g.health, g.food = maxHealth, maxFood
	hold(t, g, seconds(5*chillTime+1))
	if lost := maxHealth - g.health; lost != 5 {
		t.Errorf("%v seconds with no warmth left cost %v points of health, wanted 5", 5*chillTime+1, lost)
	}
	// And it can make the player faint.
	g.health = 1
	hold(t, g, seconds(chillTime+1))
	if g.screen != fainted {
		t.Errorf("a player with one point of health and no warmth is on screen %d after %v seconds of the cold, wanted fainted", g.screen, chillTime+1)
	}
	// Waking, it is warm again.
	g.wake()
	if g.warmth != maxWarmth {
		t.Errorf("the player woke with %v of warmth", g.warmth)
	}
	// In creative there is no cold.
	c := underStars(t)
	fallAt(c, fallLasts+0.5*dayLength)
	c.warmth = maxWarmth
	hold(t, c, seconds(30))
	if c.warmth != maxWarmth {
		t.Errorf("in creative the winter took %.1f of warmth", maxWarmth-c.warmth)
	}
}

func TestAFireNearAndAHotSpringGiveWarmthBack(t *testing.T) {
	fire := func(off int) float64 {
		g := winterAt(t, 0.5)
		p := g.player.Pos
		x, z := int(math.Floor(p.X))+off, int(math.Floor(p.Z))
		g.world.Set(x, int(p.Y), z, world.Fire) // at the height of the player's feet
		g.warmth = 5
		hold(t, g, seconds(10))
		return g.warmth
	}
	if near, far := fire(4), fire(9); near < 5+0.8*10*maxWarmth/warmTime || far >= 5 {
		t.Errorf("ten seconds by a fire four blocks off left %.1f of warmth, and by one nine blocks off %.1f: wanted the near one to warm, and the far one not", near, far)
	}
	g := winterAt(t, 0.5)
	p := g.player.Pos
	x, z := int(math.Floor(p.X))+3, int(math.Floor(p.Z))
	g.world.Set(x, int(p.Y), z, world.Fire) // at the height of the player's feet
	g.warmth = 1
	hold(t, g, seconds(warmTime+2))
	if g.warmth != maxWarmth {
		t.Errorf("%v seconds by a fire left %.1f of warmth, wanted all of it", warmTime+2, g.warmth)
	}

	// A hot spring warms, in the winter too.
	s, vent := atMountain(t)
	s.days = arrival + (fallLasts/2+0.5*dayLength)/dayLength
	s.days += math.Mod(8-s.round(vent)+geyserEvery, geyserEvery) / dayLength // its geyser is quiet for a while
	s.fallen = s.days
	if !s.winter() {
		t.Fatal("the hot spring is not in the winter")
	}
	s.warmth = 5
	hold(t, s, seconds(4))
	if s.warmth < 5+0.8*4*maxWarmth/warmTime {
		t.Errorf("four seconds in a hot spring in the winter left %.1f of warmth, wanted it to warm", s.warmth)
	}
}

func TestAFurCloakKeepsTheColdOfTheDayOffAndHalfThatOfTheNight(t *testing.T) {
	lost := func(days float64, cloak bool) float64 {
		g := winterAt(t, days)
		if cloak {
			g.items[5] = world.Stack{Item: world.FurCloak, Count: 1}
		}
		hold(t, g, seconds(coldTime/2))
		return maxWarmth - g.warmth
	}
	if day := lost(0.5, true); day != 0 {
		t.Errorf("a minute of the winter's day took %.1f of warmth from a player who carries a fur cloak", day)
	}
	bare, cloaked := lost(1, false), lost(1, true)
	if math.Abs(bare-maxWarmth/2) > 0.5 || math.Abs(cloaked-bare/2) > 0.3 {
		t.Errorf("a minute of the winter's night took %.1f of warmth, and %.1f from a player who carries a fur cloak: wanted half as much", bare, cloaked)
	}

	// It is made by hand, of four hides and two cords.
	g := inSurvival(t)
	g.items[0] = world.Stack{Item: world.Hide, Count: 4}
	g.items[1] = world.Stack{Item: world.Cord, Count: 2}
	r, ok := recipeFor(g, world.FurCloak)
	if !ok || r.at != byHand || !g.craft(r) || g.items.count([]world.Item{world.FurCloak}) != 1 || g.items.count([]world.Item{world.Hide, world.Cord}) != 0 {
		t.Errorf("a fur cloak could not be made by hand of four hides and two cords: the player carries %v", g.items)
	}
}

// coldLand returns a place in a cold land of a game's world, on the ground, with the
// world brought in round it.
func coldLand(t *testing.T, g *Game) geometry.Vector3 {
	t.Helper()
	for far := 100.0; far < 6000; far += 60 {
		for turn := range 12 {
			x := g.player.Pos.X + far*math.Cos(float64(turn)*math.Pi/6)
			z := g.player.Pos.Z + far*math.Sin(float64(turn)*math.Pi/6)
			if !g.world.Cold(x, z) {
				continue
			}
			g.world.Follow(x, z)
			g.world.LoadAll()
			if y := g.world.Ground(int(math.Floor(x)), int(math.Floor(z))); g.world.At(int(math.Floor(x)), y, int(math.Floor(z))).Info().Solid {
				return geometry.Vector3{X: x, Y: float64(y) + 1, Z: z}
			}
		}
	}
	t.Fatal("no cold land within 6,000 blocks")
	return geometry.Vector3{}
}

func TestOutOfTheWinterOnlyTheColdLandsTakeWarmth(t *testing.T) {
	g := underStars(t)
	g.begin(true)
	g.days = 5.2 // the noon of a day before any Great Fall
	g.fallen = g.days
	if g.winter() || g.cold() {
		t.Fatalf("on day 5, where the world begins, it is winter: %v, and cold: %v", g.winter(), g.cold())
	}
	g.warmth = 12
	hold(t, g, seconds(10))
	if want := 12 + 10*maxWarmth/backTime; math.Abs(g.warmth-want) > 0.5 {
		t.Errorf("ten seconds in a land that is not cold left %.1f of warmth, wanted the %.1f it comes back to", g.warmth, want)
	}

	stand(g, coldLand(t, g))
	g.warmth = maxWarmth
	hold(t, g, seconds(coldTime/4))
	if lost := maxWarmth - g.warmth; math.Abs(lost-maxWarmth/4) > 0.5 || !g.cold() || g.winter() {
		t.Errorf("half a minute in a cold land took %.1f of warmth, wanted a quarter of it", lost)
	}
}

func TestInTheWinterHerdsAreThinAndQuickclawsComeByDay(t *testing.T) {
	f := &fauna{}
	for k, want := range map[render.Kind][2]int{
		render.Bugler: {16, 8}, render.Mossback: {12, 6}, render.Nipper: {10, 5},
		render.Shieldbrow: {4, 3}, render.Cloudgrazer: {2, 2}, render.Quickclaw: {3, 3}, render.Dread: {1, 1},
	} {
		f.winter = false
		usual := f.most(k)
		f.winter = true
		if thin := f.most(k); usual != want[0] || thin != want[1] {
			t.Errorf("there may be %d of kind %d round the player, and %d in the winter: wanted %d and %d", usual, k, thin, want[0], want[1])
		}
	}

	// The beasts that come round a player in a minute and a half: fewer in the winter.
	come := func(days float64) (grazers, claws int) {
		g := winterAt(t, days)
		g.fauna.timer = 0
		for range seconds(90) {
			hold(t, g, 1)
			claws = max(claws, g.fauna.count(render.Quickclaw))
		}
		for k := range render.Kinds {
			if !breeds[k].visits {
				grazers += g.fauna.count(render.Kind(k))
			}
		}
		return grazers, claws
	}
	summer, _ := func() (int, int) {
		g := underStars(t)
		g.begin(true)
		g.days, g.fauna.timer = 5.2, 0
		g.fallen = g.days
		claws := 0
		for range seconds(90) {
			hold(t, g, 1)
			claws = max(claws, g.fauna.count(render.Quickclaw))
		}
		n := 0
		for k := range render.Kinds {
			if !breeds[k].visits {
				n += g.fauna.count(render.Kind(k))
			}
		}
		if claws != 0 {
			t.Errorf("%d quickclaws came at noon of a day that is not in the winter", claws)
		}
		return n, claws
	}()
	winter, claws := come(0.5)
	if summer < 6 || winter > summer*2/3 {
		t.Errorf("%d beasts came round the player in the winter and %d out of it: wanted far fewer", winter, summer)
	}
	if claws == 0 {
		t.Error("no quickclaw came at noon in the winter")
	}
}

func TestAfterFourDaysTheWinterIsOver(t *testing.T) {
	g := winterAt(t, winterDays-0.01)
	if !g.winter() || g.frost() != 1 {
		t.Errorf("at the end of the fourth day it is winter: %v, with %.2f of the frost", g.winter(), g.frost())
	}
	fallAt(g, fallLasts+(winterDays+0.01)*dayLength)
	if g.winter() || g.cold() {
		t.Errorf("after the fourth day it is winter: %v, and cold: %v", g.winter(), g.cold())
	}
	g.warmth = 10
	hold(t, g, seconds(10))
	if g.warmth <= 10 {
		t.Errorf("ten seconds of the spring left %.1f of warmth of 10: wanted it to come back", g.warmth)
	}
	stepBeastsOnce := func() bool {
		hold(t, g, 1)
		return g.fauna.winter
	}
	if stepBeastsOnce() || g.fauna.most(render.Bugler) != breeds[render.Bugler].most {
		t.Error("in the spring the beasts still keep the winter")
	}

	// The frost comes in the first minutes, lies through the winter, and goes in half a
	// day after it; and the frame is drawn with it.
	for _, tc := range []struct{ days, low, top float64 }{
		{-0.01, 0, 0}, {frostTime / 2 / dayLength, 0.45, 0.55}, {1, 1, 1}, {winterDays - 0.01, 1, 1},
		{winterDays + ashAfter/2, 0.45, 0.55}, {winterDays + ashAfter + 0.01, 0, 0},
	} {
		fallAt(g, fallLasts+tc.days*dayLength)
		drawn(t, g)
		if f := g.frost(); f < tc.low-1e-9 || f > tc.top+1e-9 || g.rend.Frost != f {
			t.Errorf("%.2f days after the Skyspear struck there is %.2f of the frost, and %.2f in the frame: wanted %.2f to %.2f", tc.days, f, g.rend.Frost, tc.low, tc.top)
		}
	}
}

func TestWarmthShowsOnlyWhenSomeIsLostAndIsSaved(t *testing.T) {
	g := winterAt(t, 0.5)
	if text, _ := drawn(t, g); strings.Contains(text, "☼") {
		t.Errorf("with all its warmth the player is shown a bar of it:\n%s", text)
	}
	g.warmth = 7
	if text, _ := drawn(t, g); !strings.Contains(text, "☼") {
		t.Errorf("with some of its warmth lost the player is shown no bar of it:\n%s", text)
	}

	path := filepath.Join(t.TempDir(), "world")
	g.opts.SavePath = path
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	if back := savedGame(t, path); back.warmth != 7 {
		t.Errorf("the player came back with %v of warmth, wanted 7", back.warmth)
	}
	// A save from before there was warmth is whole.
	g.warmth = maxWarmth
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	if back := savedGame(t, path); back.warmth != maxWarmth {
		t.Errorf("a save that says nothing of the cold came back with %v of warmth", back.warmth)
	}
}

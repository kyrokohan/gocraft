package game

import (
	"math"
	"path/filepath"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// nested is a game in survival on the plain by day with a herd of a kind round x = 0.5
// and the z given, so many grown and so many young, and a nest of one egg in the middle
// of it. The player stands at z = 0.5.
func nested(t *testing.T, kind render.Kind, z float64, grown, young int) *Game {
	t.Helper()
	var all []beast
	for i := range grown + young {
		angle := 2 * math.Pi * float64(i) / float64(grown+young)
		c := giant(kind, 0.5+3.5*math.Cos(angle), z+3.5*math.Sin(angle))
		c.herd, c.young = 1, i >= grown
		all = append(all, c)
	}
	g := among(t, 0.5, all...)
	g.world = wide()
	g.fauna.herds = 1
	g.fauna.eggs = []egg{{kind: kind, Pos: geometry.Vector3{X: 0.5, Y: 6, Z: z}, herd: 1, wild: true, ring: true}}
	return g
}

// carried is how many of a thing the player has, in the bag and lying about.
func carried(g *Game, item world.Item) int {
	n := g.items.count([]world.Item{item})
	for _, d := range g.drops {
		if d.Stack.Item == item {
			n += d.Stack.Count
		}
	}
	return n
}

func TestNestsArePutWithAHerdByTheSeedsChance(t *testing.T) {
	for kind, want := range map[render.Kind]world.Item{render.Bugler: world.BuglerEgg, render.Shieldbrow: world.ShieldbrowEgg, render.Galewing: world.GalewingEgg} {
		if b := breeds[kind]; b.nests <= 0 || b.nests > 1 || b.egg != want || world.Item(want).Food() != 0 {
			t.Errorf("a %s keeps a nest %v times in one, of the egg %v: wanted %v, which is no food", kind, b.nests, b.egg, want)
		}
	}
	if breeds[render.Snowtusk].nests != 0 || breeds[render.Mossback].nests != 0 || breeds[render.Quickclaw].nests != 0 {
		t.Error("a snowtusk, a mossback or a quickclaw keeps a nest")
	}

	herds, nests := 0, 0
	for seed := int64(1); seed <= 12; seed++ {
		w := realWorld(seed)
		f := newFauna(w, w.Spawn())
		again := newFauna(w, w.Spawn())
		if len(f.eggs) != len(again.eggs) {
			t.Fatalf("seed %d: %d eggs, and %d when asked again", seed, len(f.eggs), len(again.eggs))
		}
		seen := map[int]bool{}
		for i := range f.all {
			if c := &f.all[i]; c.kind == render.Bugler && !seen[c.herd] {
				seen[c.herd] = true
				herds++
			}
		}
		of := map[int]int{}
		for i, e := range f.eggs {
			if e != again.eggs[i] {
				t.Errorf("seed %d: egg %d is %+v, and %+v when asked again", seed, i, e, again.eggs[i])
			}
			of[e.herd]++
			near, kin := math.Inf(1), false
			for j := range f.all {
				if c := &f.all[j]; c.herd == e.herd {
					kin = kin || c.kind == e.kind
					near = min(near, c.Pos.Distance(e.Pos))
				}
			}
			x, z := int(math.Floor(e.Pos.X)), int(math.Floor(e.Pos.Z))
			if !kin || !e.wild || e.herd == 0 || e.warm != 0 || near > 8 && e.kind != render.Galewing || math.Abs(e.Pos.Y-float64(w.Ground(x, z)+1)) > 0.01 {
				t.Errorf("seed %d: an egg of a %s lies at %+v, of herd %d, %.1f blocks from the nearest of it: wanted it wild, on the ground, among its own", seed, e.kind, e.Pos, e.herd, near)
			}
		}
		for herd, n := range of {
			if n < 1 || n > 2 {
				t.Errorf("seed %d: the nest of herd %d holds %d eggs, wanted one or two", seed, herd, n)
			}
			if seen[herd] {
				nests++
			}
		}
	}
	if herds < 12 || nests*8 < herds || nests*2 > herds {
		t.Errorf("of %d herds of buglers in twelve worlds %d keep a nest: wanted about one in three", herds, nests)
	}
}

func TestAnEggIsStolenWithAHandAndCarried(t *testing.T) {
	for _, key := range []rune{'q', 'e'} {
		g := nested(t, render.Bugler, 3.5, 0, 0)
		g.player.Pitch = -0.45 // down at the egg, three blocks ahead
		hold(t, g, 2, engine.Key(key))
		if len(g.fauna.eggs) != 0 || carried(g, world.BuglerEgg) != 1 {
			t.Fatalf("with %c on an egg three blocks off, %d eggs lie there and the player has %d", key, len(g.fauna.eggs), carried(g, world.BuglerEgg))
		}
		// It is not eaten, and it does not go.
		g.selected = 0
		g.items[0] = world.Stack{Item: world.BuglerEgg, Count: 1}
		g.food = maxFood / 2
		hold(t, g, seconds(3), 'e')
		if carried(g, world.BuglerEgg) != 1 || g.food > maxFood/2 {
			t.Errorf("three seconds of E with an egg in hand left %d eggs and food at %v of %v", carried(g, world.BuglerEgg), g.food, maxFood/2)
		}
	}
	// One out of reach is not taken.
	g := nested(t, render.Bugler, 12.5, 0, 0)
	g.player.Pitch = -0.1
	hold(t, g, 2, 'q')
	if len(g.fauna.eggs) != 1 || carried(g, world.BuglerEgg) != 0 {
		t.Errorf("an egg twelve blocks off was taken: %d lie there", len(g.fauna.eggs))
	}
	drawn(t, g)
}

func TestBuglersRunAtAThiefAndTheirYoungDoNot(t *testing.T) {
	g := nested(t, render.Bugler, 9.5, 4, 2)
	g.takeEgg(0)
	if carried(g, world.BuglerEgg) != 1 {
		t.Fatal("the egg was not taken")
	}
	came, nearest := 0, math.Inf(1)
	near := make([]float64, len(g.fauna.all))
	for i := range near {
		near[i] = math.Inf(1)
	}
	for range seconds(8) {
		hold(t, g, 1)
		for i := range g.fauna.all {
			near[i] = min(near[i], g.fauna.all[i].Pos.Distance(g.player.Pos))
		}
	}
	for i, c := range g.fauna.all {
		if nearest = min(nearest, near[i]); !c.young && near[i] < 2.5 {
			came++
		}
		if c.young && near[i] < 4 {
			t.Errorf("a young bugler came within %.1f blocks of the thief", near[i])
		}
	}
	if came < 2 || g.health >= maxHealth {
		t.Errorf("%d of four grown buglers came within 2.5 blocks of one who took their egg, the nearest of all to %.1f, and the player lost %v points: wanted two or more to run it down", came, nearest, maxHealth-g.health)
	}
	// When they are done with it they are a herd again, and run from a player.
	hold(t, g, seconds(thiefTime))
	stand(g, geometry.Vector3{X: 0.5, Y: 6, Z: -30.5})
	hold(t, g, seconds(alarmTime+2))
	from := g.fauna.all[0].Pos
	stand(g, from.Add(geometry.Vector3{Z: -5}))
	hold(t, g, seconds(2))
	if c := g.fauna.all[0]; c.flee <= 0 || c.Pos.Distance(g.player.Pos) < 5 {
		t.Errorf("a bugler whose herd is done with a thief has a player five blocks off and does not run: it is %.1f blocks off", c.Pos.Distance(g.player.Pos))
	}
}

func TestShieldbrowsGuardAnEggAndChargeAThief(t *testing.T) {
	b := breeds[render.Shieldbrow]
	// A player who comes near the egg is warned, as one who comes near their young is.
	g := nested(t, render.Shieldbrow, 20.5, 2, 0)
	for i := range g.fauna.all {
		g.fauna.all[i].Pos.Z += 6 // the family stands beyond its nest
	}
	stand(g, geometry.Vector3{X: 0.5, Y: 6, Z: 20.5 - b.warns + 1})
	hold(t, g, seconds(2))
	warned := false
	for _, c := range g.fauna.all {
		warned = warned || c.lower || c.ire > 0
	}
	if !warned {
		t.Errorf("a player %.0f blocks from a shieldbrow's egg was not warned", b.warns-1)
	}

	// One who takes it is charged, from where they stand.
	g = nested(t, render.Shieldbrow, 20.5, 2, 0)
	g.takeEgg(0)
	charged := false
	for range seconds(10) {
		hold(t, g, 1)
		for _, c := range g.fauna.all {
			charged = charged || c.ire > 0
		}
	}
	if !charged || g.health >= maxHealth {
		t.Errorf("shieldbrows twenty blocks from one who took their egg charged: %v, and the player lost %v points", charged, maxHealth-g.health)
	}
}

func TestGalewingsNestOnHighGroundAndDiveAtOneWhoComesNear(t *testing.T) {
	// A pair over the plain, with a hill of stone fifteen square and fifteen high near.
	hill := func(g *Game) {
		for z := 34; z < 49; z++ {
			for x := 34; x < 49; x++ {
				for y := 6; y <= 20; y++ {
					g.world.Set(x, y, z, world.Stone)
				}
			}
		}
	}
	pair := func() []beast {
		one, two := soarer(20.5, 20.5, 6), soarer(24.5, 22.5, 6)
		one.herd, two.herd = 1, 1
		return []beast{one, two}
	}
	g := hunting(t, -30.5, pair()...)
	hill(g)
	g.fauna.herds = 1
	if !g.fauna.nest(g.world, render.Galewing, 1, geometry.Vector3{X: 20.5, Y: 20, Z: 20.5}) || len(g.fauna.eggs) < 1 || len(g.fauna.eggs) > 2 {
		t.Fatalf("a pair of galewings by a hill made %d eggs", len(g.fauna.eggs))
	}
	nest := g.fauna.eggs[0].Pos
	if nest.Y != 21 || nest.X < 34 || nest.X > 49 || nest.Z < 34 || nest.Z > 49 {
		t.Fatalf("the nest of a pair of galewings by a hill is at %+v: wanted it on the top of the hill, at height 21", nest)
	}
	// Over flat ground a pair has none.
	flat := hunting(t, -30.5, pair()...)
	flat.fauna.herds = 1
	if flat.fauna.nest(flat.world, render.Galewing, 1, geometry.Vector3{X: 20.5, Y: 20, Z: 20.5}) || len(flat.fauna.eggs) != 0 {
		t.Errorf("over flat ground a pair of galewings made a nest of %d eggs", len(flat.fauna.eggs))
	}

	// A player fifty blocks from the nest is let be. One on the hill is dived at.
	hold(t, g, seconds(20))
	if g.health != maxHealth {
		t.Fatalf("a player 80 blocks from a galewing's nest lost %v points", maxHealth-g.health)
	}
	stand(g, nest.Add(geometry.Vector3{X: 3}))
	passes, was := 0, g.health
	for range seconds(40) {
		hold(t, g, 1)
		if g.health < was {
			if was-g.health != galeDive {
				t.Fatalf("a galewing's dive took %v points, wanted %v", was-g.health, galeDive)
			}
			passes, was = passes+1, g.health
			g.health, was = maxHealth, maxHealth
		}
	}
	if passes < 2 || passes > 14 {
		t.Errorf("in forty seconds on their nest's hill two galewings dived at the player %d times: wanted two or more, and not one a second", passes)
	}
	// At night they sleep by their nest.
	stand(g, geometry.Vector3{X: 0.5, Y: 6, Z: 0.5})
	g.days = 2 - firstMorning
	hold(t, g, seconds(90))
	for _, c := range g.fauna.all {
		if c.aloft || c.Pos.Distance(nest) > 4 {
			t.Errorf("at midnight a galewing with a nest is %.1f blocks from it, in the air: %v", c.Pos.Distance(nest), c.aloft)
		}
	}
}

func TestASaveHoldsAnEgg(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := savedGame(t, path)
	startNew(t, g, true)
	wild := egg{kind: render.Shieldbrow, Pos: geometry.Vector3{X: 3.5, Y: 70, Z: 4.5}, herd: 4, wild: true, ring: true}
	mine := egg{kind: render.Galewing, Pos: geometry.Vector3{X: 6.5, Y: 70, Z: 4.5}, warm: 120}
	g.fauna.eggs = []egg{wild, mine}
	g.fauna.herds = 4
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	again := savedGame(t, path)
	again.Play()
	if got := again.fauna.eggs; len(got) != 2 || got[0] != wild || got[1] != mine {
		t.Errorf("a save of %+v and %+v opened with %+v", wild, mine, got)
	}
	if again.fauna.herds < 4 {
		t.Errorf("a save with an egg of herd 4 opened with %d herds counted: the next herd would take its number", again.fauna.herds)
	}
}

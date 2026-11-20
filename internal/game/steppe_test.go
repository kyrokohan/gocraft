package game

import (
	"math"
	"math/rand/v2"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// onSteppe returns a world of a seed with the land brought in round a place well inside
// a stretch of its steppe, and that place.
func onSteppe(t *testing.T, seed int64) (*world.World, geometry.Vector3) {
	t.Helper()
	w := world.Generate(seed)
	for far := 0; far < 8000; far += 80 {
		for turn := range 16 {
			x := int(float64(far) * math.Cos(float64(turn)*math.Pi/8))
			z := int(float64(far) * math.Sin(float64(turn)*math.Pi/8))
			inside := true
			for _, d := range [][2]int{{0, 0}, {48, 0}, {-48, 0}, {0, 48}, {0, -48}, {34, 34}, {-34, -34}, {34, -34}, {-34, 34}} {
				inside = inside && world.BiomeAt(seed, x+d[0], z+d[1]) == world.Tundra
			}
			if !inside {
				continue
			}
			w.Follow(float64(x), float64(z))
			w.LoadAll()
			return w, geometry.Vector3{X: float64(x) + 0.5, Y: float64(w.Ground(x, z) + 1), Z: float64(z) + 0.5}
		}
	}
	t.Fatalf("seed %d has no stretch of steppe within 8,000 blocks", seed)
	return nil, geometry.Vector3{}
}

func TestSnowtusksAndSlowclawsLiveOnTheSteppeAndNowhereElse(t *testing.T) {
	herds, alone := 0, 0
	for seed := int64(1); seed <= 5; seed++ {
		w, at := onSteppe(t, seed)
		f := newFauna(w, at)
		grown, young := map[int]int{}, map[int]int{}
		for _, c := range f.all {
			if c.kind != render.Snowtusk && c.kind != render.Slowclaw {
				continue
			}
			x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
			if land := world.BiomeAt(seed, x, z); land != world.Tundra {
				t.Errorf("seed %d: a %s stands in land %d, wanted the steppe", seed, c.kind, land)
			}
			if c.health != breeds[c.kind].health {
				t.Errorf("seed %d: a %s has %v health, wanted %v", seed, c.kind, c.health, breeds[c.kind].health)
			}
			switch {
			case c.kind == render.Slowclaw:
				if alone++; c.herd != 0 || c.young {
					t.Errorf("seed %d: a slowclaw is of herd %d, young %v: wanted it grown and alone", seed, c.herd, c.young)
				}
			case c.young:
				young[c.herd]++
			default:
				grown[c.herd]++
			}
		}
		for herd, n := range grown {
			if herds++; n != 3 || young[herd] > 2 || herd == 0 {
				t.Errorf("seed %d: a herd of snowtusks has %d grown and %d young, wanted 3 and up to 2", seed, n, young[herd])
			}
		}
		for _, k := range []render.Kind{render.Snowtusk, render.Slowclaw} {
			if n := f.count(k); n > breeds[k].most {
				t.Errorf("seed %d: %d of %s, and the most is %d", seed, n, k, breeds[k].most)
			}
		}
		// Round where the world begins they are on the steppe too, if any is there.
		home := realWorld(seed)
		for _, c := range newFauna(home, home.Spawn()).all {
			land := world.BiomeAt(seed, int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z)))
			if steppe := c.kind == render.Snowtusk || c.kind == render.Slowclaw; steppe && land != world.Tundra {
				t.Errorf("seed %d: a %s stands where the world begins, in land %d", seed, c.kind, land)
			}
		}
	}
	if herds < 3 || alone < 3 {
		t.Errorf("%d herds of snowtusks and %d slowclaws were found on the steppe of five worlds, wanted some in most", herds, alone)
	}
}

func TestAHerdOfSnowtusksChargesTogetherWhatComesNearItsYoung(t *testing.T) {
	bull, cow, calf := giant(render.Snowtusk, -3.5, 30.5), giant(render.Snowtusk, 4.5, 30.5), giant(render.Snowtusk, 0.5, 33.5)
	bull.herd, cow.herd, calf.herd, calf.young = 1, 1, 1, true
	g := among(t, 3.5, bull, cow, calf) // 30 blocks from the young one
	b := breeds[render.Snowtusk]

	hold(t, g, seconds(2))
	for _, c := range g.fauna.all[:2] {
		if heard(g)[sound.Trumpet] != 0 || c.ire > 0 || c.lower {
			t.Fatalf("a snowtusk 27 blocks off took notice: trumpets %d, charging %v, head down %v", heard(g)[sound.Trumpet], c.ire > 0, c.lower)
		}
	}
	g.player.Pos.Z = 33.5 - b.warns + 1 // within warning of the young one
	hold(t, g, seconds(2))
	if c := g.fauna.all[0]; heard(g)[sound.Trumpet] == 0 || !c.lower || c.ire > 0 || g.health != maxHealth {
		t.Errorf("a snowtusk warned: trumpets %d, head down %v, charging %v, the player's health %v", heard(g)[sound.Trumpet], c.lower, c.ire > 0, g.health)
	}
	drawn(t, g) // with their heads down

	g.player.Pos.Z = 33.5 - b.charges + 1
	both := false
	for range seconds(6) {
		hold(t, g, 1)
		both = both || g.fauna.all[0].ire > 0 && g.fauna.all[1].ire > 0
		if g.health < maxHealth {
			break
		}
	}
	if lost := maxHealth - g.health; lost != b.blow && lost != 2*b.blow {
		t.Fatalf("the charge of the herd took %v points, wanted the %v of one snowtusk, or of two", lost, b.blow)
	}
	if !both {
		t.Error("the two grown snowtusks of the herd did not charge together")
	}
}

func TestASlowclawTakesNoNoticeUntilItIsHitAndThenStandsUpAndStrikes(t *testing.T) {
	g := among(t, 21, giant(render.Slowclaw, 0.5, 23.5)) // two and a half blocks from it
	hold(t, g, seconds(5))
	c := &g.fauna.all[0]
	if c.ire > 0 || heard(g)[sound.Bellow] != 0 || g.health != maxHealth {
		t.Fatalf("a slowclaw took notice of a player beside it: charging %v, cries %d, the player's health %v", c.ire > 0, heard(g)[sound.Bellow], g.health)
	}
	// It stands up when it has stopped, and goes on all fours.
	if c.walking || c.Pose().Lower != 1 {
		t.Errorf("a slowclaw that stands still is walking: %v, and stands up by %v", c.walking, c.Pose().Lower)
	}
	c.walking, c.timer = true, 3
	hold(t, g, 2)
	if c.Pose().Lower != 0 {
		t.Error("a slowclaw that walks stands up")
	}

	c.walking, c.timer = false, math.Inf(1)
	g.wound(0, 3, 1, false)
	hold(t, g, 2)
	if c.ire <= 0 || c.flee > 0 || c.health != breeds[render.Slowclaw].health-3 {
		t.Fatalf("a slowclaw that was hit: coming %v, running %v, health %v", c.ire > 0, c.flee > 0, c.health)
	}
	stood := false
	for range seconds(chargeTime) {
		hold(t, g, 1)
		stood = stood || c.Pose().Lower == 1
		if g.health < maxHealth {
			break
		}
	}
	if lost := maxHealth - g.health; lost != breeds[render.Slowclaw].blow || lost != 6 || !stood {
		t.Errorf("a slowclaw that was hit took %v points from the player beside it, standing up: %v: wanted 6", lost, stood)
	}
	// It does not come after one who goes away.
	g.player.Pos.Z = 5
	g.health = maxHealth
	hold(t, g, seconds(12))
	if g.health != maxHealth || c.Pos.Z < 18 {
		t.Errorf("twelve seconds on, the slowclaw is at z = %.1f and the player who went away has lost %v points", c.Pos.Z, maxHealth-g.health)
	}
}

func TestWhatTheBeastsOfTheSteppeLeave(t *testing.T) {
	for _, tc := range []struct {
		kind              render.Kind
		young             bool
		meat, hide, tusks int
	}{
		{render.Snowtusk, false, 10, 4, 2},
		{render.Snowtusk, true, 10, 4, 0},
		{render.Slowclaw, false, 6, 3, 0},
	} {
		one := giant(tc.kind, 0.5, 30.5)
		one.young = tc.young
		g := among(t, -30.5, one)
		for hits := 0; hits < 200 && !g.fauna.all[0].dead; hits++ {
			g.wound(0, 6, 1, false)
		}
		hold(t, g, seconds(fallTime+0.5))
		left := map[world.Item]int{}
		for _, d := range g.drops {
			left[d.Stack.Item] += d.Stack.Count
		}
		if len(g.fauna.all) != 0 || left[world.Meat] != tc.meat || left[world.Hide] != tc.hide || left[world.Tusk] != tc.tusks {
			t.Errorf("a dead %s, young %v, left %v: wanted %d meat, %d hides and %d tusks", tc.kind, tc.young, left, tc.meat, tc.hide, tc.tusks)
		}
	}
	// Four hides: one snowtusk is a fur cloak.
	if breeds[render.Snowtusk].hide < 4 || breeds[render.Snowtusk].health <= breeds[render.Shieldbrow].health {
		t.Errorf("a snowtusk has %v health and leaves %d hides", breeds[render.Snowtusk].health, breeds[render.Snowtusk].hide)
	}
}

func TestTheBeastsOfTheSteppeDoNotStickOnIt(t *testing.T) {
	walked, stuck := 0, 0
	for seed := int64(1); seed <= 5; seed++ {
		w, at := onSteppe(t, seed)
		f := newFauna(w, at)
		f.timer = math.Inf(1)                  // none come or go
		sky := at.Add(geometry.Vector3{Y: 60}) // a player too far off to be a threat
		start, farthest := map[*rand.Rand]geometry.Vector3{}, map[*rand.Rand]float64{}
		for i := range f.all {
			if c := &f.all[i]; !c.young && (c.kind == render.Snowtusk || c.kind == render.Slowclaw) {
				start[c.rng] = c.Pos
			}
		}
		for range seconds(120) {
			f.step(w, sky, frame)
			for i := range f.all {
				c := &f.all[i]
				if from, ok := start[c.rng]; ok {
					farthest[c.rng] = max(farthest[c.rng], math.Hypot(c.Pos.X-from.X, c.Pos.Z-from.Z))
				}
			}
		}
		for who := range start {
			if walked++; farthest[who] < 3 {
				stuck++
			}
		}
	}
	t.Logf("%d beasts of the steppe walked for two minutes in five worlds, and %d never got three blocks from where they began", walked, stuck)
	if walked < 8 || stuck*8 > walked {
		t.Errorf("%d of %d beasts of the steppe stuck where they began, wanted fewer than one in eight of ten or so", stuck, walked)
	}
}

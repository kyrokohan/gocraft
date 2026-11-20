package game

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// giants are the armed plant-eaters.
var giants = []render.Kind{render.Shieldbrow, render.Thornback, render.Boulderback}

// giant is a grown beast of a kind on the plain at x, z, that stands still and faces +z
// until something makes it do otherwise.
func giant(kind render.Kind, x, z float64) beast {
	return beast{
		kind: kind, health: breeds[kind].health, timer: math.Inf(1),
		Pos: geometry.Vector3{X: x, Y: 6, Z: z}, rng: rand.New(rand.NewPCG(7, uint64(kind))),
	}
}

// among is a game in survival on the plain, by day, with the beasts given and none to
// come, whose player stands at x = 0.5 and the z given, facing +z.
func among(t *testing.T, z float64, beasts ...beast) *Game {
	t.Helper()
	g := inSurvival(t)
	g.world = plain()
	g.player.Pos = geometry.Vector3{X: 0.5, Y: 6, Z: z}
	g.fauna.all, g.fauna.timer = beasts, math.Inf(1)
	hush(g)
	return g
}

// facing is how far a beast's Yaw is from the way toward a point, in radians.
func facing(c beast, at geometry.Vector3) float64 {
	return math.Abs(math.Remainder(c.Yaw-yawOf(at.Sub(c.Pos)), 2*math.Pi))
}

func TestTheArmedPlantEatersAreFoundInTheirLandsAndGroups(t *testing.T) {
	found := map[render.Kind]int{}
	for seed := int64(1); seed <= 10; seed++ {
		w := realWorld(seed)
		f := newFauna(w, w.Spawn())
		grown, young := map[int]int{}, map[int]int{}
		for _, c := range f.all {
			if !slices.Contains(giants, c.kind) {
				continue
			}
			found[c.kind]++
			b := breeds[c.kind]
			x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
			if land := world.BiomeAt(seed, x, z); !slices.Contains(b.lands, land) {
				t.Errorf("seed %d: a %s stands in land %d", seed, c.kind, land)
			}
			if c.health != b.health {
				t.Errorf("seed %d: a %s has %v health, wanted %v", seed, c.kind, c.health, b.health)
			}
			switch {
			case c.kind != render.Shieldbrow:
				if c.herd != 0 || c.young {
					t.Errorf("seed %d: a %s is of herd %d, young %v: wanted it grown and alone", seed, c.kind, c.herd, c.young)
				}
			case c.young:
				young[c.herd]++
			default:
				grown[c.herd]++
			}
		}
		for herd, n := range grown {
			if n != 2 || young[herd] < 1 || young[herd] > 2 {
				t.Errorf("seed %d: a family of shieldbrows has %d grown and %d young, wanted 2 and 1 or 2", seed, n, young[herd])
			}
		}
		for _, k := range giants {
			if n := f.count(k); n > breeds[k].most {
				t.Errorf("seed %d: %d of %s, and the most is %d", seed, n, k, breeds[k].most)
			}
		}
	}
	for _, k := range giants {
		if found[k] == 0 {
			t.Errorf("no %s was found in ten worlds", k)
		}
	}
}

func TestAShieldbrowWarnsThenChargesAndGoesBackToItsYoung(t *testing.T) {
	grown, calf := giant(render.Shieldbrow, 0.5, 30.5), giant(render.Shieldbrow, 0.5, 33.5)
	grown.herd, calf.herd, calf.young = 1, 1, true
	g := among(t, 5.5, grown, calf) // 28 blocks from the young one
	b := breeds[render.Shieldbrow]

	hold(t, g, seconds(2))
	if c := g.fauna.all[0]; heard(g)[sound.Bellow] != 0 || c.ire > 0 || c.lower {
		t.Fatalf("a shieldbrow 25 blocks off took notice: bellows %d, charging %v, head down %v", heard(g)[sound.Bellow], c.ire > 0, c.lower)
	}

	g.player.Pos.Z = 33.5 - b.warns + 1 // within warning of the young one, and not yet where it charges
	hold(t, g, seconds(2))
	c := g.fauna.all[0]
	if heard(g)[sound.Bellow] == 0 || !c.lower || facing(c, g.player.Pos) > 0.2 {
		t.Errorf("a shieldbrow warned: bellows %d, head down %v, %.2f rad off facing the player", heard(g)[sound.Bellow], c.lower, facing(c, g.player.Pos))
	}
	if c.ire > 0 || g.health != maxHealth || c.Pos.Distance(grown.Pos) > 0.1 {
		t.Fatalf("a shieldbrow that only warns charged: ire %v, health %v, moved %.1f", c.ire, g.health, c.Pos.Distance(grown.Pos))
	}
	drawn(t, g) // with its head down

	g.player.Pos.Z = 33.5 - b.charges + 1
	from := g.player.Pos
	for range seconds(6) {
		if hold(t, g, 1); g.health < maxHealth {
			break
		}
	}
	if lost := maxHealth - g.health; lost != b.blow {
		t.Fatalf("a shieldbrow's charge took %v points, wanted %v", lost, b.blow)
	}
	if thrown := g.player.Pos.Distance(from); thrown < 1 {
		t.Errorf("the charge threw the player %.1f blocks, wanted a block or more", thrown)
	}
	if c := g.fauna.all[0]; c.ire != 0 || c.rest <= 0 {
		t.Errorf("after it hit, the shieldbrow still charges: ire %v, rest %v", c.ire, c.rest)
	}

	g.player.Pos = geometry.Vector3{X: 0.5, Y: 6, Z: -30.5} // gone
	hold(t, g, seconds(40))
	if c, young := g.fauna.all[0], g.fauna.all[1]; c.Pos.Distance(young.Pos) > herdReach+3 || young.dead {
		t.Errorf("forty seconds on the shieldbrow is %.1f from its young, wanted it back with it", c.Pos.Distance(young.Pos))
	}
}

// A charge cannot turn fast: once the beast runs, a player well out of its way is missed.
func TestAChargingShieldbrowMissesAPlayerWhoGetsWellAside(t *testing.T) {
	grown := giant(render.Shieldbrow, 0.5, 30.5)
	grown.Yaw = math.Pi // it faces the player
	g := among(t, 30.5-breeds[render.Shieldbrow].charges+0.5, grown)
	for range seconds(3) {
		if hold(t, g, 1); g.fauna.all[0].ire > 0 && g.fauna.all[0].ire < chargeTime {
			break
		}
	}
	if c := g.fauna.all[0]; c.ire <= 0 || c.ire == chargeTime || !c.walking {
		t.Fatalf("a shieldbrow with a player well within its charge does not run at it: ire %v, walking %v", c.ire, c.walking)
	}
	g.player.Pos.X += 9 // out of its way, faster than it can turn
	hold(t, g, seconds(chargeTime+1))
	if g.health != maxHealth {
		t.Errorf("a player nine blocks aside of the charge lost %v points", maxHealth-g.health)
	}
	if c := g.fauna.all[0]; c.ire != 0 || c.rest <= 0 && c.Pos.Distance(grown.Pos) < 5 {
		t.Errorf("the charge did not run its course: ire %v, and it is %.1f from where it began", c.ire, c.Pos.Distance(grown.Pos))
	}
}

func TestAShieldbrowChargesAHunterThatComesAtItsYoung(t *testing.T) {
	grown, calf := giant(render.Shieldbrow, 0.5, 30.5), giant(render.Shieldbrow, 0.5, 33.5)
	grown.herd, calf.herd, calf.young = 1, 1, true
	g := among(t, -30.5, grown, calf, claw(3.5, 36.5)) // a quickclaw by the young one
	g.days = midnight                                  // when it hunts, and does not go
	hold(t, g, seconds(10))
	if left := only(g.fauna.all, render.Quickclaw); len(left) != 0 && left[0].health == breeds[render.Quickclaw].health {
		t.Errorf("a quickclaw by a shieldbrow's young was not charged: it has %v health", left[0].health)
	}
	if g.health != maxHealth {
		t.Errorf("the player, far away, lost %v points", maxHealth-g.health)
	}
}

func TestAShieldbrowThatIsHitChargesFromFarAndIsNotKnockedBack(t *testing.T) {
	g := among(t, 5.5, giant(render.Shieldbrow, 0.5, 23.5)) // 18 blocks off: it takes no notice
	hold(t, g, seconds(1))
	if g.fauna.all[0].ire > 0 {
		t.Fatal("a shieldbrow 18 blocks off charged a player who did nothing")
	}
	g.wound(0, 5, 1, false)
	hold(t, g, 2)
	c := g.fauna.all[0]
	if c.ire <= 0 || c.flee > 0 || c.push.Length() > 0 || c.health != breeds[render.Shieldbrow].health-5 {
		t.Fatalf("a shieldbrow that was hit: charging %v, running %v, knocked %v, health %v", c.ire > 0, c.flee > 0, c.push, c.health)
	}
	for range seconds(chargeTime) {
		if hold(t, g, 1); g.health < maxHealth {
			break
		}
	}
	if g.health == maxHealth {
		t.Error("the shieldbrow did not reach the player who hit it from 18 blocks")
	}
}

func TestAThornbackTurnsItsTailAndStrikesAndDoesNotRun(t *testing.T) {
	b := breeds[render.Thornback]
	g := among(t, 30.5-b.warns+1, giant(render.Thornback, 0.5, 30.5))
	hold(t, g, seconds(3))
	c := g.fauna.all[0]
	if away := math.Abs(math.Remainder(c.Yaw-yawOf(c.Pos.Sub(g.player.Pos)), 2*math.Pi)); away > 0.2 {
		t.Errorf("a thornback a player came near is %.2f rad off turning its tail to it", away)
	}
	if heard(g)[sound.Rattle] == 0 || g.health != maxHealth {
		t.Errorf("warned %d times, and the player has %v health: wanted a warning and no blow yet", heard(g)[sound.Rattle], g.health)
	}

	// Within reach of the tail, which is behind it: toward the player.
	tail := c.Pos.Add(mesh.Camera{Yaw: c.Yaw}.Forward().Scale(-2))
	g.player.Pos = geometry.Vector3{X: tail.X, Y: 6, Z: tail.Z - 1.2}
	from := g.player.Pos
	hold(t, g, seconds(1))
	if lost := maxHealth - g.health; lost != b.blow {
		t.Fatalf("a second in reach of a thornback's tail cost %v points, wanted one blow of %v", lost, b.blow)
	}
	if g.player.Pos.Distance(from) < 1 || g.player.Pos.Z >= from.Z {
		t.Errorf("the blow threw the player from %v to %v, wanted it thrown away from the beast", from, g.player.Pos)
	}
	if pose := g.fauna.all[0].Pose(); g.fauna.all[0].lash > 0 && pose.Lash == 0 {
		t.Error("the tail is not seen to swing")
	}

	// Hit, it stands where it is.
	at := g.fauna.all[0].Pos
	g.wound(0, 5, 1, false)
	hold(t, g, seconds(3))
	if c := g.fauna.all[0]; c.Pos.Distance(at) > 0.6 || c.flee > 0 || c.health != b.health-5 {
		t.Errorf("a thornback that was hit moved %.1f, running %v, health %v: wanted it to stand", c.Pos.Distance(at), c.flee > 0, c.health)
	}
}

func TestABoulderbacksArmorTakesThreePointsOffEveryHit(t *testing.T) {
	b := breeds[render.Boulderback]
	g := among(t, -20.5, giant(render.Boulderback, 0.5, 30.5))
	for _, hit := range []struct{ points, left float64 }{
		{world.NoItem.Damage(), b.health},   // a hand
		{world.Spear.Damage(), b.health},    // a stab with a spear: 3
		{world.Club.Damage(), b.health - 1}, // a club: 4
		{world.HardSpear.Damage() + thrownMore, b.health - 1 - 3},
	} {
		g.wound(0, hit.points, 1, false)
		if c := g.fauna.all[0]; c.health != hit.left {
			t.Errorf("after a hit of %v the boulderback has %v health, wanted %v", hit.points, c.health, hit.left)
		}
	}
	if b.armor != 3 || breeds[render.Shieldbrow].armor != 0 || breeds[render.Thornback].armor != 0 {
		t.Errorf("armor: boulderback %v, shieldbrow %v, thornback %v", b.armor, breeds[render.Shieldbrow].armor, breeds[render.Thornback].armor)
	}
}

func TestEachOfThemDiesOfEnoughSpearsAndLeavesItsMeatAndHides(t *testing.T) {
	for _, kind := range giants {
		b := breeds[kind]
		g := among(t, -30.5, giant(kind, 0.5, 30.5))
		spears := 0
		for ; spears < 100 && !g.fauna.all[0].dead; spears++ {
			g.wound(0, world.HardSpear.Damage()+thrownMore, 1, false)
		}
		if spears < 8 || spears > 20 {
			t.Errorf("a %s died of %d thrown spears, wanted work: 8 to 20", kind, spears)
		}
		hold(t, g, seconds(fallTime+0.5))
		left := map[world.Item]int{}
		for _, d := range g.drops {
			left[d.Stack.Item] += d.Stack.Count
		}
		if len(g.fauna.all) != 0 || left[world.Meat] != b.meat || left[world.Hide] != b.hide {
			t.Errorf("a dead %s left %v with %d beasts still there, wanted %d meat and %d hides", kind, left, len(g.fauna.all), b.meat, b.hide)
		}
		if b.meat < 6 || b.health < 2*breeds[render.Bugler].health {
			t.Errorf("a %s has %v health and %d meat: wanted far more than a bugler", kind, b.health, b.meat)
		}
	}
}

func TestAGiantIsMetAllAlongItsBodyAndNotBesideIt(t *testing.T) {
	g := among(t, -30.5, giant(render.Shieldbrow, 0.5, 30.5)) // it faces +z
	back, front := render.Span(render.Shieldbrow, false)
	for _, c := range []struct {
		x, z float64
		hit  bool
		what string
	}{
		{0.5, 30.5, true, "its middle"},
		{0.5, 30.5 + front - 0.3, true, "its head"},
		{0.5, 30.5 + back + 0.3, true, "its tail"},
		{0.5, 30.5 + front + 1, false, "ahead of its head"},
		{0.5 + 2.2, 30.5 + front - 0.3, false, "beside its head"},
		{0.5 - 2.2, 30.5, false, "beside its middle"},
	} {
		if _, hit := g.beastAt(geometry.Vector3{X: c.x, Y: 7, Z: c.z}); hit != c.hit {
			t.Errorf("a point at %s meets the shieldbrow: %v, wanted %v", c.what, hit, c.hit)
		}
	}

	// Turned to face +x, it lies along x.
	g.fauna.all[0].Yaw = math.Pi / 2
	if _, hit := g.beastAt(geometry.Vector3{X: 0.5 + front - 0.3, Y: 7, Z: 30.5}); !hit {
		t.Error("turned, its head is not where it faces")
	}
	if _, hit := g.beastAt(geometry.Vector3{X: 0.5, Y: 7, Z: 30.5 + front - 0.3}); hit {
		t.Error("turned, a point where its head was still meets it")
	}
	// And a ray across its tail, from the side, aims at it.
	origin := geometry.Vector3{X: 0.5 + back + 0.4, Y: 7, Z: 24.5}
	if _, ok := g.aimCreature(origin, geometry.Vector3{Z: 1}, 20); !ok {
		t.Error("a ray across its tail does not aim at it")
	}

	// A beast no longer than its box is one box, as before.
	one := giant(render.Bugler, 0.5, 10.5)
	if _, n := one.body(); n != 1 {
		t.Errorf("a bugler's body is %d boxes, wanted 1", n)
	}
}

func TestAGiantClimbsAStepOfTwoBlocksAndABuglerOnlyOne(t *testing.T) {
	climbs := func(kind render.Kind, high int) float64 {
		w := plain()
		for z := 15; z < 60; z++ {
			for x := -10; x <= 10; x++ {
				for y := 6; y < 6+high; y++ {
					w.Set(x, y, z, world.Stone)
				}
			}
		}
		c := giant(kind, 0.5, 9.5)
		c.walking = true
		for range seconds(12) {
			c.walking, c.timer, c.Yaw = true, math.Inf(1), 0 // it keeps on, straight at the step
			c.Step(w, frame)
		}
		return c.Pos.Y - 6
	}
	for _, kind := range giants {
		if up := climbs(kind, 2); !near(up, 2) {
			t.Errorf("a %s at a step of two blocks is %.2f up, wanted on top of it", kind, up)
		}
		if up := climbs(kind, 3); up > 0.01 {
			t.Errorf("a %s climbed a step of three blocks: %.2f up", kind, up)
		}
	}
	if up := climbs(render.Bugler, 1); !near(up, 1) {
		t.Errorf("a bugler at a step of one block is %.2f up, wanted on top of it", up)
	}
	if up := climbs(render.Bugler, 2); up > 0.01 {
		t.Errorf("a bugler climbed a step of two blocks: %.2f up", up)
	}
}

// The risk named in the design: a big beast that sticks on the land. Each giant found
// in ten real worlds is left to walk for two minutes, and must get somewhere.
func TestGiantsDoNotStickOnTheLand(t *testing.T) {
	walked, stuck := 0, 0
	for seed := int64(1); seed <= 10; seed++ {
		w := realWorld(seed)
		spawn := w.Spawn()
		f := newFauna(w, spawn)
		f.all = slices.DeleteFunc(f.all, func(c beast) bool { return !slices.Contains(giants, c.kind) }) // only they are walked: it is them the test is about
		f.timer = math.Inf(1)                                                                            // none come or go
		sky := spawn.Add(geometry.Vector3{Y: 60})                                                        // a player too far off to be a threat, and near enough for them to be awake
		start, farthest := map[*rand.Rand]geometry.Vector3{}, map[*rand.Rand]float64{}
		for i := range f.all {
			if c := &f.all[i]; slices.Contains(giants, c.kind) && !c.young {
				start[c.rng] = c.Pos
			}
		}
		for range seconds(120) {
			f.step(w, sky, frame)
			for i := range f.all {
				c := &f.all[i]
				if at, ok := start[c.rng]; ok {
					farthest[c.rng] = max(farthest[c.rng], math.Hypot(c.Pos.X-at.X, c.Pos.Z-at.Z))
				}
			}
		}
		for who := range start {
			if walked++; farthest[who] < 3 {
				stuck++
			}
		}
	}
	t.Logf("%d giants walked for two minutes in ten worlds, and %d never got three blocks from where they began", walked, stuck)
	if walked < 10 {
		t.Fatalf("only %d giants were found in ten worlds", walked)
	}
	if stuck*10 > walked {
		t.Errorf("%d of %d giants stuck where they began, wanted fewer than one in ten", stuck, walked)
	}
}

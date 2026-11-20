package game

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// onSwamp returns a world of a seed with the land brought in round a place well inside a
// stretch of its coal swamp, and that place.
func onSwamp(t *testing.T, seed int64) (*world.World, geometry.Vector3) {
	t.Helper()
	w := world.Generate(seed)
	for far := 0; far < 9000; far += 60 {
		for turn := range 16 {
			x := int(float64(far) * math.Cos(float64(turn)*math.Pi/8))
			z := int(float64(far) * math.Sin(float64(turn)*math.Pi/8))
			inside := true
			for _, d := range [][2]int{{0, 0}, {30, 0}, {-30, 0}, {0, 30}, {0, -30}} {
				inside = inside && world.BiomeAt(seed, x+d[0], z+d[1]) == world.Swamp
			}
			if !inside {
				continue
			}
			w.Follow(float64(x), float64(z))
			w.LoadAll()
			return w, geometry.Vector3{X: float64(x) + 0.5, Y: float64(w.Ground(x, z) + 1), Z: float64(z) + 0.5}
		}
	}
	t.Fatalf("seed %d has no stretch of swamp within 9,000 blocks", seed)
	return nil, geometry.Vector3{}
}

func TestPeatBurnsThreeTimesAsLongAsALogAndAScaleTreeIsALog(t *testing.T) {
	if peat, log := burnTime(world.Peat.Item()), burnTime(world.PineLog.Item()); peat != 3*log || log <= 0 {
		t.Errorf("peat burns for %v s and a log for %v, wanted three times as long", peat, log)
	}
	if !slices.Contains(logs, world.ScaleTrunk.Item()) || burnTime(world.ScaleTrunk.Item()) != burnTime(world.PineLog.Item()) {
		t.Error("the trunk of a scale tree is no log: it makes no sticks, or does not burn as one")
	}
	if burnTime(world.Amber) != 0 || burnTime(world.ScaleCrown.Item()) != 0 {
		t.Error("amber burns, or the crown of a scale tree")
	}
}

// over returns how high a beast is over what is under it: ground, or water.
func over(w *world.World, c *beast) float64 {
	x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
	for y := int(math.Floor(c.Pos.Y)); y > 0; y-- {
		if b := w.At(x, y, z); b != world.Air && !b.Info().Stands() {
			return c.Pos.Y - float64(y+1)
		}
	}
	return math.Inf(1)
}

func TestRattlewingsFlyLowOverTheSwampAndNeverComeDown(t *testing.T) {
	flown := 0
	for seed := int64(1); seed <= 5; seed++ {
		w, at := onSwamp(t, seed)
		f := newFauna(w, at)
		f.timer = math.Inf(1)
		sky := at.Add(geometry.Vector3{Y: 60})
		start := map[*rand.Rand]geometry.Vector3{}
		for i := range f.all {
			c := &f.all[i]
			if c.kind != render.Rattlewing {
				continue
			}
			start[c.rng] = c.Pos
			if land := world.BiomeAt(seed, int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))); land != world.Swamp {
				t.Errorf("seed %d: a rattlewing is over land %d as the world is made", seed, land)
			}
		}
		if n := f.count(render.Rattlewing); n > breeds[render.Rattlewing].most {
			t.Errorf("seed %d: %d rattlewings, and the most is %d", seed, n, breeds[render.Rattlewing].most)
		}
		far := map[*rand.Rand]float64{}
		for range seconds(60) {
			f.step(w, sky, frame)
			for i := range f.all {
				c := &f.all[i]
				from, ok := start[c.rng]
				if !ok {
					continue
				}
				far[c.rng] = max(far[c.rng], math.Hypot(c.Pos.X-from.X, c.Pos.Z-from.Z))
				if high := over(w, c); high < 0.3 || high > 4.5 || c.OnGround || inWater(w, c.Pos) {
					t.Fatalf("seed %d: a rattlewing is %.1f over what is under it, on the ground: %v, in water: %v", seed, high, c.OnGround, inWater(w, c.Pos))
				}
				if land := world.BiomeAt(seed, int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))); land != world.Swamp && land != world.Sea && land != world.Shore {
					t.Fatalf("seed %d: a rattlewing has flown out over land %d", seed, land)
				}
			}
		}
		for who := range start {
			if flown++; far[who] < 4 {
				t.Errorf("seed %d: a rattlewing got no farther than %.1f blocks in a minute", seed, far[who])
			}
		}
	}
	if flown < 4 {
		t.Errorf("%d rattlewings were found over the swamp of five worlds", flown)
	}

	// It is harmless, a hand's hit sends it off, and a spear's kills it.
	g := among(t, 0.5, giant(render.Rattlewing, 0.5, 2.5))
	g.fauna.all[0].Pos.Y = 7.5
	hold(t, g, seconds(8))
	if g.health != maxHealth {
		t.Errorf("a rattlewing took %v points from a player beside it", maxHealth-g.health)
	}
	g.fauna.all[0].Pos = geometry.Vector3{X: 0.5, Y: 7.5, Z: 2.5}
	g.wound(0, 1, 1, false)
	if c := g.fauna.all[0]; c.dead || c.flee <= 0 {
		t.Errorf("a rattlewing hit by a hand is dead: %v, and darts off for %.1f s", c.dead, c.flee)
	}
	g.wound(0, 3, 1, false)
	if !g.fauna.all[0].dead {
		t.Error("a rattlewing lived through a spear")
	}
}

func TestALogcrawlerCurlsUpWhenItIsHitAndNothingGetsThroughItsPlates(t *testing.T) {
	g := among(t, 0.5, giant(render.Logcrawler, 0.5, 2.5))
	b := breeds[render.Logcrawler]
	hold(t, g, seconds(6))
	if g.health != maxHealth || g.fauna.all[0].Pose().Lower != 0 {
		t.Fatalf("a logcrawler beside a player took %v points from it, and is curled by %v", maxHealth-g.health, g.fauna.all[0].Pose().Lower)
	}
	g.wound(0, 3, 1, false)
	c := &g.fauna.all[0]
	if c.health != b.health-3 || c.curl <= 0 || c.flee > 0 || c.Pose().Lower != 1 {
		t.Fatalf("a logcrawler that was hit has %v health, is curled for %.1f s and runs for %.1f s", c.health, c.curl, c.flee)
	}
	at := c.Pos
	for range 5 {
		g.wound(0, 6, 1, false)
		hold(t, g, seconds(0.5))
	}
	if c.health != b.health-3 || c.dead || c.Pos.Distance(at) > 0.2 {
		t.Errorf("five hits on a curled logcrawler left it %v health, dead: %v, and moved it %.1f: wanted nothing to get through", c.health, c.dead, c.Pos.Distance(at))
	}
	hold(t, g, seconds(b.curls))
	if c.curl > 0 || c.Pose().Lower != 0 {
		t.Errorf("%v seconds on it is still curled", b.curls+2.5)
	}
	g.wound(0, 6, 1, false)
	if !c.dead {
		t.Errorf("a logcrawler that had uncurled lived through a hit of 6 with %v health", c.health)
	}
	hold(t, g, seconds(fallTime+0.5))
	meat := 0
	for _, d := range g.drops {
		if d.Stack.Item == world.Meat {
			meat += d.Stack.Count
		}
	}
	if meat != b.meat || b.meat != 2 {
		t.Errorf("a dead logcrawler left %d pieces of meat, wanted 2", meat)
	}
}

func TestLogcrawlersCreepOnThePeatOfTheSwampAndDoNotStick(t *testing.T) {
	walked, stuck := 0, 0
	for seed := int64(1); seed <= 5; seed++ {
		w, at := onSwamp(t, seed)
		f := newFauna(w, at)
		f.timer = math.Inf(1)
		sky := at.Add(geometry.Vector3{Y: 60})
		start, farthest := map[*rand.Rand]geometry.Vector3{}, map[*rand.Rand]float64{}
		for i := range f.all {
			c := &f.all[i]
			if c.kind != render.Logcrawler {
				continue
			}
			start[c.rng] = c.Pos
			x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
			if on, land := w.At(x, int(c.Pos.Y)-1, z), world.BiomeAt(seed, x, z); on != world.Peat || land != world.Swamp {
				t.Errorf("seed %d: a logcrawler stands on %s in land %d", seed, on.Info().Name, land)
			}
		}
		for range seconds(180) {
			f.step(w, sky, frame)
			for i := range f.all {
				c := &f.all[i]
				if from, ok := start[c.rng]; ok {
					farthest[c.rng] = max(farthest[c.rng], math.Hypot(c.Pos.X-from.X, c.Pos.Z-from.Z))
					if inWater(w, c.Pos) {
						t.Fatalf("seed %d: a logcrawler is in the water", seed)
					}
				}
			}
		}
		for who := range start {
			if walked++; farthest[who] < 1.5 {
				stuck++
			}
		}
	}
	t.Logf("%d logcrawlers crept for three minutes in five worlds, and %d never got a block and a half from where they began", walked, stuck)
	if walked < 4 || stuck*3 > walked {
		t.Errorf("%d of %d logcrawlers stuck where they began", stuck, walked)
	}
}

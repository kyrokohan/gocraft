package game

import (
	"math"
	"slices"
	"testing"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// soarer is a galewing in the air over x, z, as high over the height given as its kind
// soars, that goes as it will.
func soarer(x, z, over float64) beast {
	c := giant(render.Galewing, x, z)
	c.Pos.Y, c.timer, c.aloft = over+breeds[render.Galewing].hover, 0, true
	return c
}

// under is the height of what is under a beast: the top of the first block below it that
// is not air, water too.
func under(g *Game, c *beast) float64 {
	x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
	for y := int(math.Floor(c.Pos.Y)); y >= 0; y-- {
		if g.world.At(x, y, z) != world.Air {
			return float64(y + 1)
		}
	}
	return 0
}

func TestAGalewingKeepsInTheAirCirclesAndDipsToTheWater(t *testing.T) {
	b := breeds[render.Galewing]
	if b.moves != gaitSoar || b.hover < 10 || b.wary != 0 || b.temper == hunter {
		t.Fatalf("a galewing moves in way %d, %v over what is under it, wary at %v", b.moves, b.hover, b.wary)
	}
	if !slices.Contains(b.lands, world.Sea) || !slices.Contains(b.lands, world.Shore) {
		t.Errorf("a galewing's lands are %v, wanted the sea and the shore", b.lands)
	}
	// The plain is all sea, one block deep, but for the rock the player stands on.
	g := hunting(t, 0.5, soarer(20.5, 20.5, 6))
	for z := -40; z < 80; z++ {
		for x := -40; x < 80; x++ {
			if x != 0 || z != 0 {
				g.world.Set(x, 4, z, world.Stone)
				g.world.Set(x, 5, z, world.Water)
			}
		}
	}
	c := &g.fauna.all[0]
	from, yaw := c.Pos, c.Yaw
	turned, low, dipped, back := 0.0, math.Inf(1), false, false
	for range seconds(90) {
		hold(t, g, 1)
		turned += math.Abs(math.Remainder(c.Yaw-yaw, 2*math.Pi))
		yaw = c.Yaw
		high := c.Pos.Y - under(g, c)
		low = min(low, high)
		if !c.aloft || c.OnGround || high < 0.6 || high > b.hover+10 || c.dead || c.flee > 0 {
			t.Fatalf("a galewing is at %+v, %.1f over what is under it, in the air: %v, running: %v", c.Pos, high, c.aloft, c.flee > 0)
		}
		if far := math.Hypot(c.Pos.X-from.X, c.Pos.Z-from.Z); far > 50 {
			t.Fatalf("a galewing that circles is %.0f blocks from where it began", far)
		}
		if high < 2.5 && g.world.At(int(math.Floor(c.Pos.X)), 5, int(math.Floor(c.Pos.Z))) == world.Water {
			dipped = true
		}
		back = back || dipped && high > b.hover-3
	}
	if turned < 4*math.Pi {
		t.Errorf("in a minute and a half a galewing turned %.1f times round, wanted two or more", turned/(2*math.Pi))
	}
	if !dipped || !back {
		t.Errorf("a galewing over the sea came down to the water: %v, and went up again: %v: it was %.1f over what was under it at the lowest", dipped, back, low)
	}
	if g.health != maxHealth {
		t.Errorf("a galewing took %v points from the player", maxHealth-g.health)
	}
}

func TestAGalewingSleepsOnHighGroundAndFliesAtSunrise(t *testing.T) {
	// A hill of stone fifteen blocks square and fifteen high, twenty blocks from where
	// it circles, at midnight.
	g := atHour(t, 0, 0.5, soarer(20.5, 20.5, 6))
	for z := 34; z < 49; z++ {
		for x := 34; x < 49; x++ {
			for y := 6; y <= 20; y++ {
				g.world.Set(x, y, z, world.Stone)
			}
		}
	}
	c := &g.fauna.all[0]
	down := -1.0
	for i := range seconds(90) {
		hold(t, g, 1)
		if !c.aloft {
			down = float64(i) * frame
			break
		}
	}
	if down < 0 || math.Abs(c.Pos.Y-21) > 0.2 || c.Pos.X < 34 || c.Pos.X > 49 || c.Pos.Z < 34 || c.Pos.Z > 49 {
		t.Fatalf("at midnight a galewing by a hill came down: %v, and is at %+v: wanted it on the top of the hill, at height 21", down >= 0, c.Pos)
	}
	at := c.Pos
	hold(t, g, seconds(20))
	if c.aloft || c.Pos.Distance(at) > 0.3 {
		t.Errorf("a galewing on its hill at night moved %.1f blocks in twenty seconds, in the air: %v", c.Pos.Distance(at), c.aloft)
	}
	if pose := c.Pose(); pose.Aloft {
		t.Error("a galewing on its hill is drawn with its wings out")
	}
	// At sunrise it flies.
	g.days = 2 + 0.3 - firstMorning
	hold(t, g, seconds(8))
	if high := c.Pos.Y - under(g, c); !c.aloft || high < 5 || !c.Pose().Aloft {
		t.Errorf("eight seconds into the morning a galewing is %.1f over what is under it, in the air: %v: wanted it up from its hill", high, c.aloft)
	}

	// With no high ground near it keeps in the air all night.
	g = atHour(t, 0, 0.5, soarer(20.5, 20.5, 6))
	c = &g.fauna.all[0]
	for range seconds(60) {
		hold(t, g, 1)
		if !c.aloft || c.Pos.Y-under(g, c) < 5 {
			t.Fatalf("over flat ground at night a galewing is at %+v, in the air: %v", c.Pos, c.aloft)
		}
	}
}

func TestAStiltwingTakesOffWhenAPlayerComesNearAndLandsAgainFarOff(t *testing.T) {
	b := breeds[render.Stiltwing]
	if _, height := render.Size(render.Stiltwing, false); height < 4 || b.takes < 6 || b.wary < 8 || !slices.Equal(b.lands, []world.Biome{world.Badlands}) {
		t.Fatalf("a stiltwing stands %.1f tall, flies %v over the ground, is wary at %v, in lands %v", height, b.takes, b.wary, b.lands)
	}
	one := giant(render.Stiltwing, 0.5, -40.5)
	one.herd = 1
	g := hunting(t, -64.5, one)
	g.world = wide()
	c := &g.fauna.all[0]
	hold(t, g, seconds(5))
	if c.aloft || !c.OnGround || c.flee > 0 || c.Pose().Aloft {
		t.Fatalf("a stiltwing with a player 24 blocks off is in the air: %v, on the ground: %v", c.aloft, c.OnGround)
	}

	// The player walks at it. It takes alarm, and is in the air.
	left := geometry.Vector3{}
	for range seconds(10) {
		hold(t, g, 1, 'w')
		if c.aloft {
			left = c.Pos
			break
		}
	}
	if !c.aloft || c.flee <= 0 || c.Pos.Distance(g.player.Pos) > b.wary+1 {
		t.Fatalf("a stiltwing a player walked up to is in the air: %v, with the player %.1f blocks off: wanted it to take off at %v", c.aloft, c.Pos.Distance(g.player.Pos), b.wary)
	}
	hold(t, g, seconds(3))
	if high := c.Pos.Y - 6; !c.aloft || high < 5 || !c.Pose().Aloft {
		t.Errorf("three seconds after it took off a stiltwing is %.1f over the ground, in the air: %v", high, c.aloft)
	}
	// It comes down far off, and stands.
	landed := -1.0
	for i := range seconds(40) {
		hold(t, g, 1)
		if !c.aloft && c.OnGround {
			landed = float64(i) * frame
			break
		}
	}
	far := math.Hypot(c.Pos.X-left.X, c.Pos.Z-left.Z)
	if landed < 0 || far < 30 || math.Abs(c.Pos.Y-6) > 0.01 || c.health != b.health || c.dead {
		t.Fatalf("a stiltwing that took off came down: %v, %.0f blocks from where it left the ground, at height %.2f with %v health: wanted it down thirty blocks off or more, unhurt", landed >= 0, far, c.Pos.Y, c.health)
	}
	if g.health != maxHealth {
		t.Errorf("a stiltwing took %v points from the player", maxHealth-g.health)
	}

	// It does not come down on water: it flies on over it.
	one = giant(render.Stiltwing, 0.5, 30.5)
	one.aloft, one.Pos.Y, one.Yaw = true, 6+b.takes, yawOf(geometry.Vector3{Z: 1})
	g = hunting(t, 50.5, one)
	g.world, g.player.Pos.X = wide(), 40.5 // forty blocks to a side of its way
	for z := 20; z <= 70; z++ {
		for x := -20; x <= 20; x++ {
			g.world.Set(x, 5, z, world.Water)
			g.world.Set(x, 4, z, world.Stone)
		}
	}
	c = &g.fauna.all[0]
	hold(t, g, seconds(30))
	if c.aloft || !c.OnGround || inWater(g.world, c.Pos) || c.Pos.Z < 70 {
		t.Errorf("a stiltwing that was over water is at %+v, in the air: %v, in water: %v: wanted it down on the land beyond", c.Pos, c.aloft, inWater(g.world, c.Pos))
	}
}

// onLand returns a world of a seed with all of it loaded round a place well inside a
// land, and that place.
func onLand(t *testing.T, seed int64, land world.Biome) (*world.World, geometry.Vector3) {
	t.Helper()
	w := world.Generate(seed)
	for far := 0; far < 9000; far += 60 {
		for turn := range 16 {
			x := int(float64(far) * math.Cos(float64(turn)*math.Pi/8))
			z := int(float64(far) * math.Sin(float64(turn)*math.Pi/8))
			inside := true
			for _, d := range [][2]int{{0, 0}, {30, 0}, {-30, 0}, {0, 30}, {0, -30}} {
				inside = inside && world.BiomeAt(seed, x+d[0], z+d[1]) == land
			}
			if !inside {
				continue
			}
			w.Follow(float64(x), float64(z))
			w.LoadAll()
			return w, geometry.Vector3{X: float64(x) + 0.5, Y: float64(w.Ground(x, z) + 1), Z: float64(z) + 0.5}
		}
	}
	t.Fatalf("seed %d has no stretch of land %d within 9,000 blocks", seed, land)
	return nil, geometry.Vector3{}
}

func TestGalewingsAreFoundOverTheSeaAndStiltwingsInTheBadlands(t *testing.T) {
	gales, stilts := 0, 0
	for seed := int64(1); seed <= 4; seed++ {
		// Over the sea, by day: galewings, in the air, and they keep there.
		w, at := atSea(t, seed)
		f := newFauna(w, at)
		f.sun, f.timer = 1, math.Inf(1)
		for i := range f.all {
			c := &f.all[i]
			if c.kind != render.Galewing {
				continue
			}
			gales++
			x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
			if land := world.BiomeAt(seed, x, z); !c.aloft || land != world.Sea && land != world.Shore || c.Pos.Y < float64(w.Ground(x, z))+breeds[c.kind].hover-1 {
				t.Errorf("seed %d: a galewing was put at %+v, over land %d, in the air: %v", seed, c.Pos, land, c.aloft)
			}
		}
		if n := f.count(render.Galewing); n > breeds[render.Galewing].most {
			t.Errorf("seed %d: %d galewings", seed, n)
		}
		sky := at.Add(geometry.Vector3{Y: 90})
		for range seconds(120) {
			f.sun = 1
			f.step(w, sky, frame)
			for i := range f.all {
				c := &f.all[i]
				if c.kind != render.Galewing || c.dead {
					continue
				}
				x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
				if high := c.Pos.Y - float64(w.Ground(x, z)+1); !c.aloft || high < 0.5 || inWater(w, c.Pos) {
					t.Fatalf("seed %d: a galewing is at %+v, %.1f over what is under it, in the air: %v", seed, c.Pos, high, c.aloft)
				}
			}
		}

		// In the badlands: stiltwings, on foot, in ones and twos.
		w, at = onLand(t, seed, world.Badlands)
		f = newFauna(w, at)
		for i := range f.all {
			c := &f.all[i]
			if c.kind != render.Stiltwing {
				continue
			}
			stilts++
			x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
			if land := world.BiomeAt(seed, x, z); c.aloft || land != world.Badlands || math.Abs(c.Pos.Y-float64(w.Ground(x, z)+1)) > 0.01 {
				t.Errorf("seed %d: a stiltwing was put at %+v, in land %d, in the air: %v", seed, c.Pos, land, c.aloft)
			}
		}
		if n := f.count(render.Stiltwing); n > breeds[render.Stiltwing].most {
			t.Errorf("seed %d: %d stiltwings", seed, n)
		}
	}
	if gales < 2 || stilts < 2 {
		t.Errorf("%d galewings were found over the seas of four worlds, and %d stiltwings in their badlands", gales, stilts)
	}
}

func TestAGalewingBeyondTheViewIsGivenToTheSkyToDraw(t *testing.T) {
	near, far := soarer(0.5, 30.5, 6), soarer(0.5, 80.5, 6)
	roosted := soarer(40.5, 80.5, 6)
	roosted.aloft = false
	g := hunting(t, 0.5, near, far, roosted, giant(render.Bugler, 0.5, 90.5))
	g.world = wide()
	cam, _ := g.viewed()
	ways := g.farWings(cam)
	if len(ways) != 1 {
		t.Fatalf("the sky was given %d galewings to draw: wanted the one that is in the air beyond the view, and no other", len(ways))
	}
	want := g.fauna.all[1].Pos.Sub(cam.Position)
	if ways[0].Sub(want).Length() > 1.5 {
		t.Errorf("the sky draws a galewing %+v from the eye, and it is %+v from it", ways[0], want)
	}
	drawn(t, g)
}

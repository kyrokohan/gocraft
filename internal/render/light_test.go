package render

import (
	"math"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/world"
)

// shut builds a box of stone round the facing camera and the wall in front of it, so
// that no light from the sky comes in.
func shut(w *world.World) {
	for z := 0; z <= 6; z++ {
		for y := -1; y <= 24; y++ {
			for x := -1; x <= 24; x++ {
				if x == -1 || x == 24 || y == -1 || y == 24 || z == 0 || z == 6 {
					w.Set(x, y+1, z, world.Stone) // one up, so that the floor is in the world
				}
			}
		}
	}
}

// brightness returns the mean of the brightest channel of the pixels of a canvas.
func brightness(c *engine.Canvas) float64 {
	sum := 0
	for y := range c.Height() {
		for x := range c.Width() {
			r, g, b := c.At(x, y).RGB()
			sum += int(max(r, g, b))
		}
	}
	return float64(sum) / float64(c.Width()*c.Height())
}

// drawn returns the picture of a world from the facing camera, in the sky of a time of day.
func drawn(t *testing.T, w *world.World, time float64) *engine.Canvas {
	t.Helper()
	c := newCanvas(t, 40, 20)
	cam := facing
	cam.Fog = &NewSky(time).Fog
	newRenderer(t).Draw(c, w, cam, NewSky(time))
	return c
}

func TestAWallShutAwayFromTheSkyIsDark(t *testing.T) {
	open, closed := world.New(), world.New()
	wall(open, 5, world.Stone)
	wall(closed, 5, world.Stone)
	shut(closed)

	lit, dark := brightness(drawn(t, open, Noon)), brightness(drawn(t, closed, Noon))
	if dark > lit/4 {
		t.Errorf("a wall shut away is %.0f bright and one in the open %.0f: wanted it far darker", dark, lit)
	}
	if dark == 0 {
		t.Error("a wall shut away is black: the darkest place should still show a little")
	}
}

func TestATorchLightsTheDark(t *testing.T) {
	dark, lit := world.New(), world.New()
	for _, w := range []*world.World{dark, lit} {
		wall(w, 5, world.Stone)
		shut(w)
	}
	lit.Set(12, 13, 4, world.Torch)

	without, with := drawn(t, dark, Noon), drawn(t, lit, Noon)
	if b, d := brightness(with), brightness(without); b < 3*d {
		t.Errorf("the room is %.0f bright with a torch and %.0f without: wanted it lit", b, d)
	}
	// Torchlight is warm: the wall by the torch is redder than it is blue.
	if r, _, b := with.At(28, 18).RGB(); r <= b+10 {
		t.Errorf("the wall by the torch is %v, wanted it warm", with.At(28, 18))
	}
}

func TestCrystalOreGlowsInTheDark(t *testing.T) {
	stone, crystal := world.New(), world.New()
	wall(stone, 5, world.Stone)
	wall(crystal, 5, world.CrystalOre)
	shut(stone)
	shut(crystal)
	if s, c := brightness(drawn(t, stone, Noon)), brightness(drawn(t, crystal, Noon)); c < 1.5*s {
		t.Errorf("in the dark crystal ore is %.0f bright and stone %.0f: wanted it to glow", c, s)
	}
}

func TestTheWorldIsDarkAndBlueAtNight(t *testing.T) {
	w := world.New()
	wall(w, 5, world.Sand)
	day, night := drawn(t, w, Noon), drawn(t, w, Midnight)
	if d, n := brightness(day), brightness(night); n > d/3 {
		t.Errorf("sand is %.0f bright at night and %.0f by day: wanted it dark", n, d)
	}
	if r, _, b := night.At(20, 20).RGB(); b <= r {
		t.Errorf("sand at night is %v, wanted the blue of the moon's light", night.At(20, 20))
	}
	if r, _, b := day.At(20, 20).RGB(); b >= r {
		t.Errorf("sand by day is %v, wanted its own yellow", day.At(20, 20))
	}
}

// up returns the picture of the sky straight overhead at a time of day.
func up(t *testing.T, time float64) *engine.Canvas {
	t.Helper()
	c := newCanvas(t, 80, 40)
	cam := facing
	cam.Pitch = math.Pi / 2.2
	cam.Fog = &NewSky(time).Fog
	newRenderer(t).Draw(c, world.New(), cam, NewSky(time))
	return c
}

// stars counts the pixels far brighter than the sky round them.
func stars(c *engine.Canvas) int {
	n := 0
	for y := range c.Height() {
		for x := 1; x < c.Width(); x++ {
			_, g, _ := c.At(x, y).RGB()
			_, before, _ := c.At(x-1, y).RGB()
			if int(g) > int(before)+60 {
				n++
			}
		}
	}
	return n
}

func TestTheNightSkyIsDarkWithStarsAndTheDaySkyHasNone(t *testing.T) {
	night, day := up(t, Midnight), up(t, Noon)
	if n, d := brightness(night), brightness(day); n > d/4 {
		t.Errorf("the sky overhead is %.0f bright at night and %.0f by day", n, d)
	}
	if n := stars(night); n < 5 {
		t.Errorf("%d stars overhead at night, wanted some", n)
	}
	// By day, nowhere in the sky: the sun is overhead, so this looks at directions.
	for _, time := range []float64{Noon, Sunrise + 0.05, Sunset - 0.05} {
		s := NewSky(time)
		starless := *s
		starless.stars = 0
		for i := range 5000 {
			// Directions spread evenly over the half of the sky above the horizon.
			climb := float64(i) / 5000
			around := float64(i) * math.Pi * (3 - math.Sqrt(5))
			flat := math.Sqrt(1 - climb*climb)
			d := geometry.Vector3{X: flat * math.Cos(around), Y: climb, Z: flat * math.Sin(around)}
			if s.color(d) != starless.color(d) {
				t.Fatalf("at %v there is a star toward %v", time, d)
			}
		}
	}
}

func TestTheStarsTurnWithTheSky(t *testing.T) {
	a, b := up(t, Midnight), up(t, Midnight+0.02)
	if differing(a, b) == 0 {
		t.Error("the stars stood still for half an hour of the night")
	}
}

func TestTheSunCrossesTheSkyAndTheMoonIsAcrossFromIt(t *testing.T) {
	for _, c := range []struct {
		time float64
		sun  geometry.Vector3
	}{
		{Sunrise, geometry.Vector3{X: 1}},
		{Noon, geometry.Vector3{Y: 1}},
		{Sunset, geometry.Vector3{X: -1}},
		{Midnight, geometry.Vector3{Y: -1}},
	} {
		s := NewSky(c.time)
		if s.sun.Dot(c.sun) < 0.9 {
			t.Errorf("at %v the sun is toward %v, wanted about %v", c.time, s.sun, c.sun)
		}
		if s.moon != s.sun.Neg() {
			t.Errorf("at %v the moon is toward %v, and not across from the sun at %v", c.time, s.moon, s.sun)
		}
	}
	noon, midnight := NewSky(Noon), NewSky(Midnight)
	if got := noon.color(noon.sun); got != noon.sunDisc {
		t.Errorf("looking at the sun shows %v, wanted the sun", got)
	}
	away := geometry.Vector3{X: 0.3, Y: 0.9, Z: 0.3}.Normalize()
	if m, s := midnight.color(midnight.moon), midnight.color(away); m.g < s.g+80 {
		t.Errorf("looking at the moon shows %v, and the sky beside it %v: wanted the moon bright", m, s)
	}
}

func TestTheSkyGlowsRedTowardTheSunAsItSets(t *testing.T) {
	s := NewSky(Sunset)
	low := geometry.Vector3{X: -1, Y: 0.05}.Normalize() // just over the horizon, under the sun
	if c := s.color(low); c.r <= c.b {
		t.Errorf("the sky under the setting sun is %v, wanted it red", c)
	}
	if c := NewSky(Noon).color(low); c.r >= c.b {
		t.Errorf("the same sky at noon is %v, wanted it blue", c)
	}
}

func TestTheFogTakesTheColorOfTheHorizonAtEveryHour(t *testing.T) {
	_, dayGreen, _ := NewSky(Noon).Fog.Color.RGB()
	_, nightGreen, _ := NewSky(Midnight).Fog.Color.RGB()
	if nightGreen*3 > dayGreen {
		t.Errorf("the fog's green is %d at night and %d by day: wanted it dark at night", nightGreen, dayGreen)
	}
}

func TestTimeGoesRoundTheClock(t *testing.T) {
	for _, c := range []struct{ in, want float64 }{
		{0.3, 0.3}, {1.25, 0.25}, {-0.25, 0.75}, {7, 0}, {math.NaN(), Noon}, {math.Inf(1), Noon},
	} {
		if got := NewSky(c.in).Time(); got != c.want {
			t.Errorf("a sky set to %v is at %v, wanted %v", c.in, got, c.want)
		}
	}
}

func TestHowBrightLightLooks(t *testing.T) {
	noon, midnight := NewSky(Noon), NewSky(Midnight)
	full := world.Light(world.MaxLight << 4)
	torch := world.Light(14)
	for _, s := range []*Sky{noon, midnight} {
		if got := s.Brightness(0); got != darkest {
			t.Errorf("at %v the dark is %v bright, wanted %v", s.Time(), got, darkest)
		}
		if got := s.Brightness(torch); math.Abs(got-0.8) > 1e-9 {
			t.Errorf("at %v a torch's light is %v bright, wanted 0.8 at any hour", s.Time(), got)
		}
	}
	if got := noon.Brightness(full); got != 1 {
		t.Errorf("the open sky at noon is %v bright, wanted 1", got)
	}
	if got := midnight.Brightness(full); got > 0.4 {
		t.Errorf("the open sky at midnight is %v bright, wanted it dim", got)
	}
	// Each level is dimmer than the one above it.
	for l := 1; l <= world.MaxLight; l++ {
		if a, b := noon.Brightness(world.Light(l<<4)), noon.Brightness(world.Light((l-1)<<4)); a < b {
			t.Errorf("level %d of the sky's light is %v bright, less than level %d at %v", l, a, l-1, b)
		}
	}
}

// The smoke of a fire mountain stands in the sky toward it: dark by day and red at its
// foot by night, and nowhere else.
func TestTheSmokeOfAMountainStandsOnTheHorizonTowardIt(t *testing.T) {
	bright := func(c color) uint32 { return c.r + c.g + c.b }
	toward := geometry.Vector3{X: 0.6, Z: 0.8} // the way to the mountain
	low := toward.Add(geometry.Vector3{Y: 0.12}).Normalize()
	away := geometry.Vector3{X: -0.6, Y: 0.12, Z: -0.8}.Normalize()
	high := geometry.Vector3{X: 0.2, Y: 0.9, Z: 0.3}.Normalize()

	day := NewSky(Noon)
	clear, clearAway, clearHigh := day.color(low), day.color(away), day.color(high)
	day.SetPlume(toward.Scale(600), 0.8)
	if smoke := day.color(low); bright(smoke) >= bright(clear)*3/4 {
		t.Errorf("by day the sky toward the mountain is %v with its smoke and %v without: wanted it a good deal darker", smoke, clear)
	}
	if day.color(away) != clearAway || day.color(high) != clearHigh {
		t.Error("the smoke shows away from the mountain, or high overhead")
	}

	night := NewSky(Midnight)
	dark := night.color(low)
	night.SetPlume(toward, 0.8)
	if glow := night.color(low); glow.r < dark.r+40 || glow.r < 2*glow.b {
		t.Errorf("by night the sky at the foot of the smoke is %v, and %v without: wanted the red of the fire in it", glow, dark)
	}

	// From too far off there is none, and it is fainter from far than from near.
	day.SetPlume(toward, 0)
	if day.color(low) != clear {
		t.Error("smoke of no strength darkens the sky")
	}
	day.SetPlume(toward, 0.2)
	faint := day.color(low)
	day.SetPlume(toward, 1)
	if strong := day.color(low); bright(strong) >= bright(faint) || bright(faint) >= bright(clear) {
		t.Errorf("the smoke is %v from near and %v from far, against %v of clear sky: wanted darker the nearer", strong, faint, clear)
	}
	day.SetPlume(geometry.Vector3{}, 1)
	if day.color(low) != clear {
		t.Error("smoke with no way to it shows")
	}
}

// Ash in the sky darkens it and the light of day, and not the light of a fire. While
// the mountain throws out its fire, the foot of its smoke is red by day too.
func TestAshDarkensTheSkyAndTheLightOfDay(t *testing.T) {
	bright := func(c color) uint32 { return c.r + c.g + c.b }
	high := geometry.Vector3{X: 0.2, Y: 0.9, Z: 0.3}.Normalize()
	low := geometry.Vector3{X: -0.6, Y: 0.12, Z: -0.8}.Normalize()
	day, torch := world.Light(world.MaxLight<<4), world.Light(14)

	clear := NewSky(Noon)
	ashen := NewSky(Noon)
	ashen.SetAsh(1, 0)
	ashen.Set(Noon)
	for name, way := range map[string]geometry.Vector3{"overhead": high, "at the horizon": low} {
		if with, without := ashen.color(way), clear.color(way); bright(with) >= bright(without)*3/4 {
			t.Errorf("the sky %s is %v under ash and %v clear: wanted it a good deal darker", name, with, without)
		}
	}
	if with, without := ashen.Brightness(day), clear.Brightness(day); with >= 0.8*without || with < 0.45 {
		t.Errorf("the light of day is %.2f under ash and %.2f clear: wanted it darker, and day still", with, without)
	}
	if ashen.Brightness(torch) != clear.Brightness(torch) {
		t.Error("ash in the sky changes the light of a torch")
	}

	// Half as much ash darkens half as much, and none is no change.
	half := NewSky(Noon)
	half.SetAsh(0.5, 0)
	half.Set(Noon)
	if b := half.Brightness(day); b <= ashen.Brightness(day) || b >= clear.Brightness(day) {
		t.Errorf("the light of day under half the ash is %.2f, wanted between %.2f and %.2f", b, ashen.Brightness(day), clear.Brightness(day))
	}
	half.SetAsh(0, 0)
	half.Set(Noon)
	if half.color(high) != clear.color(high) || half.Brightness(day) != clear.Brightness(day) {
		t.Error("a sky the ash has left is not as it was")
	}

	// The fire at the foot of the smoke, by day.
	toward := geometry.Vector3{X: 0.6, Z: 0.8}
	foot := toward.Add(geometry.Vector3{Y: 0.05}).Normalize()
	smoking, erupting := NewSky(Noon), NewSky(Noon)
	smoking.SetPlume(toward.Scale(200), 1)
	erupting.SetPlume(toward.Scale(200), 1)
	erupting.SetAsh(0, 1)
	erupting.Set(Noon)
	if fire, smoke := erupting.color(foot), smoking.color(foot); fire.r < smoke.r+60 || fire.r < 2*fire.b {
		t.Errorf("by day the foot of the smoke of a mountain that erupts is %v, and %v of one that does not: wanted the red of its fire", fire, smoke)
	}
}

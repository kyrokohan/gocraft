package render

import (
	"math"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/world"
)

// The time of day runs from 0 to 1: midnight at 0, the sun rising at a quarter, noon at
// a half and the sun setting at three quarters.
const (
	Midnight = 0.0
	Sunrise  = 0.25
	Noon     = 0.5
	Sunset   = 0.75
)

var (
	dayHorizon   = color{190, 220, 245}
	dayZenith    = color{70, 130, 220}
	nightHorizon = color{22, 30, 56}
	nightZenith  = color{4, 7, 22}
	duskGlow     = color{255, 136, 72} // the low sky toward the sun as it rises and sets
	sunColor     = color{255, 244, 200}
	moonColor    = color{214, 222, 236}
	starColor    = color{230, 236, 255}
	deepWater    = color{24, 60, 130}

	dayCloud   = color{250, 252, 255}
	nightCloud = color{46, 52, 78}
	duskCloud  = color{255, 176, 140}
)

// The light the sky gives, in each channel, as a part of the full light of day: white by
// day, blue and dim by night, and warm as the sun goes down. A torch's is warm always.
var (
	nightTint = [3]float64{0.15, 0.18, 0.3}
	duskTint  = [3]float64{1, 0.82, 0.66}
	torchTint = [3]float64{1, 0.8, 0.56}
)

// darkest is how bright the darkest place is drawn, as a part of full light: dark, but
// not black, so that a cave can still be found the way out of.
const darkest = 0.12

// starGrid is how many cells of the sky's star field there are along each way from its
// middle, in a direction one long. A cell is about as wide as a pixel of the view.
const starGrid = 150.0

// Sky is the sky at one time of day: where the sun, the moon and the stars are, the
// colors of the sky, its fog and its clouds, and how bright its light makes the world.
// Make one with NewSky and move it on with Set.
type Sky struct {
	// Fog is the fog of the camera at this hour: it fades the world into the color of
	// the horizon.
	Fog mesh.Fog

	time        float64
	sun, moon   geometry.Vector3 // the way to each, one long
	along       geometry.Vector3 // with across and the z axis, the axes of the turning sky the stars are fixed to
	across      geometry.Vector3
	day         float64 // how much of the day's light there is: 0 at night and 1 by day
	dusk        float64 // how near the sun is to the horizon, from 0 to 1 when it is on it
	stars       float64 // how bright the stars are, from 0 to 1
	horizon     color
	zenith      color
	sunDisc     color
	cloud       color
	cloudFog    mesh.Fog
	eyeFog      mesh.Fog   // the last FogIn gave
	shade       [256]color // for each world.Light, the brightness of each channel it gives, out of 256
	brightness  [256]float64
	skyChannels [3]float64 // the sky's light in each channel, as a part of the full light of day

	// The smoke of the nearest fire mountain, which stands on the horizon in the way to
	// it from farther off than the world is drawn: the way to the mountain along the
	// ground, one long, and how much of the smoke is seen, from 0, none, to 1.
	toMount geometry.Vector3
	plume   float64
	foot    float64 // how high over the horizon the smoke begins: the top of the mountain, as the sine of the angle

	// How much ash the sky holds, from 0 to 1, and how much of the fire of a mountain
	// that is throwing it out shows at the foot of its smoke.
	ash, fire float64

	// A shooting star, if one streaks the sky now: where its head and the end of its
	// tail are, one long each, and how bright it is, from 0, none, to 1.
	streakHead, streakTail geometry.Vector3
	streak                 float64

	// Fireballs: how bright they are and the land under them, from 0 to 1, and for each
	// that is seen the ways to its head and to the end of its tail, one long each.
	balls [maxBalls][2]geometry.Vector3
	seen  int
	ball  float64

	// The great fliers that are too far off to be drawn as they are: for each, the way
	// to it, one long, the ways to its right and up from there, and half of how wide
	// its wings look from here.
	wings  [maxWings]farWing
	winged int

	// The comet: how near it is, from 0, none, to 1; where its head is, one long, and
	// the way its tail goes from there, one long and square to that; how long the tail
	// is, in radians; and how much of the comet is seen at this hour, from 0 to 1.
	comet                float64
	cometHead, cometBack geometry.Vector3
	cometLong, cometSeen float64
}

// How a shooting star and a fireball stand in the sky. Widths are in a direction one
// long, so about radians.
const (
	streakTime = 0.9   // seconds a shooting star is seen
	streakLong = 0.35  // how far across the sky it goes
	streakWide = 0.008 // half its width: about a pixel
	ballWide   = 0.045 // half the width of a fireball's head
	ballHalo   = 0.3   // how far round its head the sky glows
)

var (
	ballColor = color{255, 240, 190}
	ballTrail = color{255, 150, 60}
	ballLight = [3]float64{0.7, 0.6, 0.48} // the light a fireball gives the land: as bright as dusk, and warm
)

// SetStreaks has shooting stars streak the night sky, so many a minute, by a clock in
// seconds that runs on from night to night: which streak is in the sky, and where, is
// worked out from the clock alone. None is seen where no stars are.
func (s *Sky) SetStreaks(clock, perMinute float64) {
	s.streak = 0
	if perMinute <= 0 {
		return
	}
	every := 60 / perMinute
	slot := math.Floor(clock / every)
	n := int(slot)
	into := (clock - slot*every - puff(n, 4, 71)*(every-streakTime)) / streakTime
	if into < 0 || into >= 1 {
		return
	}
	// It begins high, and goes across and down.
	yaw, up, turn := 2*math.Pi*puff(n, 1, 71), 0.35+0.5*puff(n, 2, 71), 2*math.Pi*puff(n, 3, 71)
	flat := math.Sqrt(1 - up*up)
	start := geometry.Vector3{X: flat * math.Cos(yaw), Y: up, Z: flat * math.Sin(yaw)}
	way := geometry.Vector3{X: -math.Sin(yaw) * math.Cos(turn), Y: -0.2 - 0.5*math.Abs(math.Sin(turn)), Z: math.Cos(yaw) * math.Cos(turn)}
	s.streakHead = start.Add(way.Scale(streakLong * into)).Normalize()
	s.streakTail = start.Add(way.Scale(streakLong * max(into-0.4, 0))).Normalize()
	s.streak = math.Sin(math.Pi * into)
}

// maxBalls is how many fireballs the sky shows at once.
const maxBalls = 4

// farWing is a great flier seen from far off, as the sky draws it.
type farWing struct {
	to, right, up geometry.Vector3
	half          float64
}

// How the sky draws a flier that is far off: a dark shape of two wings held up a little.
const (
	maxWings  = 4     // how many it shows at once
	wingSpan  = 8.0   // how far its wings are from tip to tip, in blocks
	wingRise  = 0.3   // how far a wing rises toward its tip, of its length
	wingThick = 0.16  // how thick a wing is at the body, of its length
	wingLeast = 0.005 // the least a wing is thick, as an angle, so that a far one is not lost between pixels
	wingHaze  = 0.6   // how much of its dark shows through the air between
)

// wingDark is the color of a flier against the sky.
var wingDark = color{42, 40, 46}

// SetWings has great fliers stand in the sky that are too far off to be drawn as they
// are: for each, the way from the eye to it, as long as it is far.
func (s *Sky) SetWings(ways []geometry.Vector3) {
	s.winged = 0
	for _, w := range ways {
		far := w.Length()
		if s.winged == maxWings || far < 1 {
			continue
		}
		to := w.Scale(1 / far)
		right := geometry.Vector3{X: to.Z, Z: -to.X}
		if right.Length() < 1e-6 {
			continue // straight overhead, or under
		}
		right = right.Normalize()
		s.wings[s.winged] = farWing{to: to, right: right, up: to.Cross(right).Normalize(), half: wingSpan / 2 / far}
		s.winged++
	}
}

// SetFireballs has fireballs stand in the sky: for each, the ways from the eye to it and
// to the end of its tail, of any length; and how bright they are, from 0, none, to 1.
// They light the land under them as dusk does, from the next Set. With none there is no
// ball, and the light alone: the flash of one that has struck.
func (s *Sky) SetFireballs(ways [][2]geometry.Vector3, bright float64) {
	s.ball, s.seen = min(max(bright, 0), 1), 0
	for _, w := range ways {
		if s.ball > 0 && s.seen < maxBalls && w[0].Length() > 1e-9 && w[1].Length() > 1e-9 {
			s.balls[s.seen] = [2]geometry.Vector3{w[0].Normalize(), w[1].Normalize()}
			s.seen++
		}
	}
}

// How the comet stands in the sky. It is fixed among the stars, on the line from the
// point they turn about to the top of the sky at midnight: so it rises at dusk lying
// along the horizon, stands upright at midnight, and sets at dawn.
const (
	cometPole = 0.7   // how far from the point the stars turn about its head is, in radians
	cometMost = 1.1   // how long its tail is when it is nearest: a third of the sky
	cometWide = 0.016 // half the width of its tail at its head, when it is far off
	cometFan  = 3.5   // how many times wider than that the end of its tail is
	cometGlow = 3.0   // how many times wider than that the glow round its head is
)

// cometOmen is how near the comet is when the sky begins to redden: in its last hours.
const cometOmen = 0.97

var (
	cometColor = color{206, 228, 255}
	omenRed    = color{150, 34, 26}
)

// SetComet has the comet stand among the stars, as near as near says, from 0, none, to
// 1, at which it is biggest. It shows from the next Set.
func (s *Sky) SetComet(near float64) {
	s.comet = min(max(near, 0), 1)
}

// along returns where the point nearest to p lies on the line from a to b, from 0 at a
// to 1 at b, and how far from p it is.
func along(p, a, b geometry.Vector3) (t, off float64) {
	ab := b.Sub(a)
	if l := ab.Dot(ab); l > 1e-12 {
		t = min(max(p.Sub(a).Dot(ab)/l, 0), 1)
	}
	return t, p.Sub(a.Add(ab.Scale(t))).Length()
}

// How ash in the sky looks by day and by night, and how much of the light of day the
// most of it takes.
var (
	ashDay   = color{92, 84, 80}
	ashNight = color{20, 17, 18}
)

const ashDims = 0.45

// SetAsh has the sky hold ash, from 0, none, to 1, and the fire of a mountain that is
// throwing it out, from 0 to 1. Ash greys the sky, its clouds and its fog, hides the
// stars and takes from the light of day; the fire reddens the foot of the mountain's
// smoke by day as the night does. Both show from the next Set.
func (s *Sky) SetAsh(ash, fire float64) {
	s.ash, s.fire = min(max(ash, 0), 1), min(max(fire, 0), 1)
}

// How the smoke of a fire mountain stands on the horizon.
const (
	plumeTop  = 0.5  // how high it climbs, as the sine of the angle over the horizon
	plumeWide = 0.05 // half its width at its foot, in radians, when it is far off
	plumeLean = 0.12 // how far it drifts to a side by its top
)

var (
	smokeDay   = color{74, 70, 72}
	smokeNight = color{22, 20, 24}
	lavaGlow   = color{236, 84, 30}
)

// SetPlume has the smoke of a fire mountain stand in the sky in the way to it, as much
// of it as near says, from 0, none, to 1, at which it is widest. The way is from the eye
// to the top of the mountain, of any length: the smoke rises from there, which is on
// the horizon from far off and high in the sky from the mountain's foot.
func (s *Sky) SetPlume(to geometry.Vector3, near float64) {
	s.plume = 0
	if flat := math.Hypot(to.X, to.Z); flat > 1e-9 && near > 0 {
		s.toMount, s.plume = geometry.Vector3{X: to.X / flat, Z: to.Z / flat}, min(near, 1)
		s.foot = max(to.Y/to.Length(), 0)
	}
}

// NewSky returns the sky at a time of day.
func NewSky(time float64) *Sky {
	s := &Sky{}
	s.Set(time)
	return s
}

// smooth goes from 0 at a to 1 at b, easing in and out.
func smooth(a, b, v float64) float64 {
	t := min(max((v-a)/(b-a), 0), 1)
	return t * t * (3 - 2*t)
}

// levels is how bright each level of light looks, as a part of the full light: each level
// down is four fifths of the one above it, and level 0 is no light at all.
var levels = func() (out [world.MaxLight + 1]float64) {
	for l := 1; l <= world.MaxLight; l++ {
		out[l] = math.Pow(0.8, float64(world.MaxLight-l))
	}
	return out
}()

// Set moves the sky to a time of day, from 0 to 1; any other number is taken round the
// clock to one.
func (s *Sky) Set(time float64) {
	time -= math.Floor(time)
	if !(time >= 0 && time < 1) {
		time = Noon // not a number
	}
	s.time = time

	// The sun rises along +x and sets along -x, leaning a little to +z, and turns
	// round the z axis at an even pace; the moon is always across from it.
	angle := (time - Sunrise) * 2 * math.Pi
	s.sun = geometry.Vector3{X: math.Cos(angle), Y: math.Sin(angle), Z: 0.3}.Normalize()
	s.moon = s.sun.Neg()
	s.along = geometry.Vector3{X: math.Cos(angle), Y: math.Sin(angle)}
	s.across = geometry.Vector3{X: -math.Sin(angle), Y: math.Cos(angle)}

	height := s.sun.Y
	s.day = smooth(-0.12, 0.22, height)
	s.dusk = 1 - smooth(0, 0.3, math.Abs(height))
	s.stars = 1 - smooth(-0.25, 0.02, height)

	s.horizon = nightHorizon.mix(dayHorizon, part(s.day))
	s.zenith = nightZenith.mix(dayZenith, part(s.day))
	s.sunDisc = sunColor.mix(duskGlow, part(s.dusk*0.6))
	s.cloud = nightCloud.mix(dayCloud, part(s.day)).mix(duskCloud, part(s.dusk*0.45))
	if s.ash > 0 {
		grey, much := ashNight.mix(ashDay, part(s.day)), part(0.8*s.ash)
		s.horizon, s.zenith, s.cloud = s.horizon.mix(grey, much), s.zenith.mix(grey, much), s.cloud.mix(grey, much)
		s.sunDisc = s.sunDisc.mix(duskGlow, much) // the sun is red through it
		s.stars *= 1 - s.ash
	}

	fog := s.horizon.mix(duskGlow, part(s.dusk*0.25))
	s.Fog = mesh.Fog{Color: fog.engine(), Start: 0.6 * ViewDistance, End: ViewDistance}
	s.cloudFog = mesh.Fog{Color: fog.engine(), Start: ViewDistance, End: 3 * ViewDistance}

	if s.cometSeen = 0; s.comet > 0 {
		// Among the stars a place is so far round from the point they turn about, +z.
		at := func(round float64) geometry.Vector3 {
			return s.along.Scale(-math.Sin(round)).Add(geometry.Vector3{Z: math.Cos(round)})
		}
		s.cometHead, s.cometBack = at(cometPole), at(cometPole+math.Pi/2)
		s.cometLong = 0.04 + cometMost*s.comet*s.comet
		// A near one shows before the stars do: through the dusk and the dawn.
		early := 0.3 * smooth(0.5, 1, s.comet)
		s.cometSeen = (1 - smooth(-0.25+early, 0.02+early, height)) * (1 - s.ash) * (0.45 + 0.55*s.comet)
		// On its last night the low sky is red with it.
		omen := smooth(cometOmen, 1, s.comet)
		s.horizon, s.zenith = s.horizon.mix(omenRed, part(0.6*omen)), s.zenith.mix(omenRed, part(0.2*omen))
	}

	for c := range s.skyChannels {
		s.skyChannels[c] = (nightTint[c] + (1-nightTint[c])*s.day) * (1 - (1-duskTint[c])*s.dusk*s.day) * (1 - ashDims*s.ash)
		s.skyChannels[c] = max(s.skyChannels[c], ballLight[c]*s.ball)
	}
	for l := range s.shade {
		light := world.Light(l)
		sky, block := levels[light.Sky()], levels[light.Block()]
		var ch [3]float64
		for c := range ch {
			ch[c] = max(sky*s.skyChannels[c], block*torchTint[c], darkest)
		}
		s.shade[l] = color{uint32(ch[0] * 256), uint32(ch[1] * 256), uint32(ch[2] * 256)}
		s.brightness[l] = max(ch[0], ch[1], ch[2])
	}
}

// FogIn returns the fog for an eye in a cell with the light l: the sky's Fog, darker as
// less of the sky's light reaches the eye, so that a cave fades into the dark and not
// into the color of the sky outside. It stays the same until the next call.
func (s *Sky) FogIn(l world.Light) *mesh.Fog {
	s.eyeFog = s.Fog
	if l.Sky() < world.MaxLight {
		dim := uint32(max(levels[l.Sky()], darkest) * 256)
		red, green, blue := s.Fog.Color.RGB()
		s.eyeFog.Color = color{uint32(red), uint32(green), uint32(blue)}.lit(dim).engine()
	}
	return &s.eyeFog
}

// Time returns the time of day the sky is at.
func (s *Sky) Time() float64 {
	return s.time
}

// Brightness returns how bright a light makes what it falls on, from 0 to 1, in the
// brightest of its channels.
func (s *Sky) Brightness(l world.Light) float64 {
	return s.brightness[l]
}

// giver returns the way the light comes from that shapes things: from the sun, and from
// the moon once the sun has set.
func (s *Sky) giver() geometry.Vector3 {
	if s.sun.Y > -0.05 {
		return s.sun
	}
	return s.moon
}

// color returns the color of the sky along a ray one long.
func (s *Sky) color(d geometry.Vector3) color {
	climb := d.Y
	c := s.horizon.mix(s.zenith, part(climb*1.8))

	toSun := d.Dot(s.sun)
	if s.dusk > 0 && toSun > 0 {
		// The low sky toward the sun goes red as it rises and sets.
		glow := s.dusk * toSun * toSun * toSun * (1 - min(max(climb*2.2, 0), 1))
		c = c.mix(duskGlow, part(glow*0.8))
	}
	if s.stars > 0 && climb > -0.05 {
		// The stars are fixed to the sky, which turns with the sun.
		a, b, p := d.Dot(s.along), d.Dot(s.across), d.Z
		x, y, z := int(math.Floor(a*starGrid)), int(math.Floor(b*starGrid)), int(math.Floor(p*starGrid))
		if twinkle := puff(x+y*977, z, 91); twinkle < 0.0035 {
			c = c.mix(starColor, part(s.stars*(0.45+150*twinkle)))
		}
	}

	if flat := math.Hypot(d.X, d.Z); s.plume > 0 && climb > s.foot-0.02 && climb < s.foot+plumeTop && flat > 1e-9 {
		// The smoke: a column that widens and drifts as it climbs, dark by day, and at
		// night all but lost in the dark, with the red of the fire under it at its foot.
		ahead := (d.X*s.toMount.X + d.Z*s.toMount.Z) / flat
		aside := (d.X*s.toMount.Z - d.Z*s.toMount.X) / flat
		up := max(climb-s.foot, 0) / plumeTop
		wide := plumeWide * (1 + 2.5*s.plume*s.plume) * (0.5 + 1.6*up)
		if edge := math.Abs(aside-plumeLean*up*up) / wide; ahead > 0 && edge < 1 {
			thick := s.plume * (1 - edge*edge) * (1 - 0.8*up*up)
			c = c.mix(smokeDay.mix(smokeNight, part(1-s.day)), part(0.85*thick))
			if glow := max(1-s.day, s.fire) * (1 - 1.6*up); glow > 0 {
				c = c.mix(lavaGlow, part(0.9*thick*glow))
			}
		}
	}

	if s.streak > 0 && s.stars > 0 && climb > 0 {
		if t, off := along(d, s.streakTail, s.streakHead); off < streakWide {
			c = c.mix(starColor, part(s.stars*s.streak*(0.3+0.7*t)))
		}
	}
	if s.cometSeen > 0 && climb > -0.05 {
		// The comet: how far behind its head the ray is, round the sky, and how far to a
		// side of its tail. The tail widens and fades as it goes back.
		ahead, back := d.Dot(s.cometHead), d.Dot(s.cometBack)
		head := cometWide * (0.5 + s.comet)
		if side := math.Sqrt(max(1-ahead*ahead-back*back, 0)); side < head*(1+cometFan) {
			behind := math.Atan2(back, ahead)
			if far, wide := behind/s.cometLong, head*(1+cometFan*behind/s.cometLong); far >= 0 && far < 1 && side < wide {
				faint := 1 - far
				c = c.mix(cometColor.mix(duskGlow, part(smooth(cometOmen, 1, s.comet))), part(s.cometSeen*(1-side/wide)*(0.12+0.88*faint*faint)))
			}
			// Its head: a glow, and a bright middle to it.
			if off := d.Sub(s.cometHead).Length(); off < cometGlow*head {
				glow := 1 - off/(cometGlow*head)
				c = c.mix(cometColor, part(s.cometSeen*glow*glow)).mix(starColor, part(s.cometSeen*smooth(0.55, 0.9, glow)))
			}
		}
	}
	for _, f := range s.wings[:s.winged] {
		// A flier far off: two wings, each thinner toward its tip and held a little up.
		off := d.Sub(f.to)
		if side := math.Abs(off.Dot(f.right)); side < f.half && d.Dot(f.to) > 0.9 {
			if high := math.Abs(off.Dot(f.up) - wingRise*side); high < max(f.half*wingThick*(1.2-side/f.half), wingLeast) {
				c = c.mix(wingDark, part(wingHaze))
			}
		}
	}
	for _, b := range s.balls[:s.seen] {
		// A fireball: a head, a tail that narrows behind it, and a glow round the head.
		t, off := along(d, b[1], b[0])
		if wide := ballWide * (0.15 + 0.85*t*t); off < wide {
			c = c.mix(ballTrail.mix(ballColor, part(t*t)), part(s.ball*(0.35+0.65*t)*(1-0.6*off/wide)))
		} else if far := d.Sub(b[0]).Length(); far < ballHalo {
			c = c.mix(duskGlow, part(0.6*s.ball*(1-far/ballHalo)*(1-far/ballHalo)))
		}
	}

	if s.sun.Y > -0.1 {
		switch {
		case toSun > 0.9985:
			return s.sunDisc
		case toSun > 0.985:
			return c.mix(s.sunDisc, part((toSun-0.985)/0.0135*0.5)) // a glow round it
		}
	}
	if toMoon := -toSun; s.moon.Y > -0.1 && toMoon > 0.9992 {
		return c.mix(moonColor, part(0.4+0.6*s.stars))
	}
	return c
}

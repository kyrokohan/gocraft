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

	fog := s.horizon.mix(duskGlow, part(s.dusk*0.25))
	s.Fog = mesh.Fog{Color: fog.engine(), Start: 0.6 * ViewDistance, End: ViewDistance}
	s.cloudFog = mesh.Fog{Color: fog.engine(), Start: ViewDistance, End: 3 * ViewDistance}

	for c := range s.skyChannels {
		s.skyChannels[c] = (nightTint[c] + (1-nightTint[c])*s.day) * (1 - (1-duskTint[c])*s.dusk*s.day)
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
			if glow := (1 - s.day) * (1 - 1.6*up); glow > 0 {
				c = c.mix(lavaGlow, part(0.9*thick*glow))
			}
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

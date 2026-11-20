package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"

	"github.com/kyrokohan/skyspear/internal/sound"
)

// The sounds are made here too, from the same fixed hash: noise shaped by filters and
// swells, and tones, so they come out the same every run and nothing is recorded or
// copied. Each is written as a WAV file, to be listened to on its own.

const rate = float64(sound.Rate)

// noise is white noise from -1 to 1, one value for each sample of a sound.
func noise(i, seed int) float64 {
	return hash(i, seed, 900)*2 - 1
}

// lowpass is a one-pole filter that lets through what is below a frequency.
type lowpass struct{ a, y float64 }

func newLowpass(hz float64) lowpass { return lowpass{a: 1 - math.Exp(-2*math.Pi*hz/rate)} }

func (f *lowpass) at(x float64) float64 {
	f.y += f.a * (x - f.y)
	return f.y
}

// highpass lets through what is above a frequency.
type highpass struct{ low lowpass }

func newHighpass(hz float64) highpass { return highpass{newLowpass(hz)} }

func (f *highpass) at(x float64) float64 { return x - f.low.at(x) }

// band is a two-pole filter that rings at a frequency, and lets through what is near it:
// q is how near.
type band struct{ b0, a1, a2, x1, x2, y1, y2 float64 }

func newBand(hz, q float64) band {
	w := 2 * math.Pi * hz / rate
	alpha := math.Sin(w) / (2 * q)
	a0 := 1 + alpha
	return band{b0: alpha / a0, a1: -2 * math.Cos(w) / a0, a2: (1 - alpha) / a0}
}

func (f *band) at(x float64) float64 {
	y := f.b0*x - f.b0*f.x2 - f.a1*f.y1 - f.a2*f.y2
	f.x2, f.x1, f.y2, f.y1 = f.x1, x, f.y1, y
	return y
}

// swell is how loud a sound is t seconds into it: up over attack, and then dying away,
// to a part of a third of itself in each decay.
func swell(t, attack, decay float64) float64 {
	switch {
	case t < 0:
		return 0
	case t < attack:
		return t / attack
	}
	return math.Exp(-(t - attack) / decay)
}

// samples is how many samples a sound of a length in seconds has.
func samples(seconds float64) int {
	return int(seconds * rate)
}

// peakTo scales a sound so that its loudest sample is at a level, and fades out its last
// few milliseconds, so that it does not end with a click.
func peakTo(s []float64, level float64) []float64 {
	fade := samples(0.005)
	for i := range s {
		if left := len(s) - 1 - i; left < fade {
			s[i] *= float64(left) / float64(fade)
		}
	}
	return scaleTo(s, level)
}

// scaleTo scales a sound so that its loudest sample is at a level.
func scaleTo(s []float64, level float64) []float64 {
	peak := 0.0
	for _, v := range s {
		peak = max(peak, math.Abs(v))
	}
	for i := range s {
		s[i] *= level / max(peak, 1e-9)
	}
	return s
}

// voiceOf is how a material sounds: noise between low and cut, the higher the brighter,
// with a ring at a frequency, mixed in by ring, and grains that die away in decay
// seconds.
type voiceOf struct {
	low, cut float64
	ring     float64 // Hz, 0 for none
	ringMix  float64
	decay    float64
}

var materialVoices = [...]voiceOf{
	sound.Earth: {low: 60, cut: 900, decay: 0.03},
	sound.Stone: {low: 300, cut: 3200, ring: 1500, ringMix: 0.35, decay: 0.02},
	sound.Sand:  {low: 900, cut: 5000, decay: 0.035},
	sound.Wood:  {low: 80, cut: 1600, ring: 240, ringMix: 0.8, decay: 0.04},
	sound.Plant: {low: 1500, cut: 6000, decay: 0.05},
	sound.Glass: {low: 1500, cut: 9000, ring: 3100, ringMix: 0.7, decay: 0.05},
	sound.Snow:  {low: 300, cut: 2200, decay: 0.045},
}

// grains is the sound of a material struck n times within span seconds, in a sound of a
// length: noise in its band, each grain dying away at its pace times slow.
func grains(m sound.Material, n int, span, length, slow float64, seed int) []float64 {
	v := materialVoices[m]
	lo, hi, hi2, ring := newHighpass(v.low), newLowpass(v.cut), newLowpass(v.cut), newBand(max(v.ring, 1), 12)
	starts := make([]float64, n)
	for g := range starts {
		starts[g] = span * hash(g, seed, 901) * float64(min(g, 1)) // the first at once
	}
	out := make([]float64, samples(length))
	for i := range out {
		t := float64(i) / rate
		env := 0.0
		for g, at := range starts {
			env += swell(t-at, 0.002, v.decay*slow) * (1 - 0.5*hash(g, seed, 902))
		}
		x := hi2.at(hi.at(lo.at(noise(i, seed)))) * env // two poles, so that a dull one is dull
		if v.ring > 0 {
			x = x*(1-v.ringMix) + ring.at(x)*v.ringMix*6
		}
		out[i] = x
	}
	return out
}

// thump is a low tone falling from one frequency to another, as a knock on the ground.
func thump(from, to, length, decay float64) []float64 {
	out := make([]float64, samples(length))
	phase := 0.0
	for i := range out {
		t := float64(i) / rate
		phase += 2 * math.Pi * (to + (from-to)*math.Exp(-t/decay)) / rate
		out[i] = math.Sin(phase) * swell(t, 0.002, decay)
	}
	return out
}

// add adds b into a, scaled, from a sample on.
func add(a, b []float64, scale float64, from int) {
	for i, v := range b {
		if from+i < len(a) {
			a[from+i] += v * scale
		}
	}
}

// voiced is the sound of a throat: a buzz at a pitch that moves as pitch says, through
// two formants that make it a vowel, swelling and dying as loud says.
func voiced(length float64, pitch, loud func(t float64) float64, f1, f2 float64) []float64 {
	out := make([]float64, samples(length))
	b1, b2 := newBand(f1, 4), newBand(f2, 6)
	phase := 0.0
	for i := range out {
		t := float64(i) / rate
		phase += 2 * math.Pi * pitch(t) / rate
		buzz := 0.0
		for k := 1.0; k <= 10; k++ {
			buzz += math.Sin(k*phase) / k
		}
		buzz += 0.15 * noise(i, 903) // breath
		out[i] = (b1.at(buzz) + 0.6*b2.at(buzz)) * loud(t)
	}
	return out
}

// loop makes a sound of a length that goes round without a seam: what made runs on past
// the end is faded into the start.
func loop(length float64, made func(n int) []float64) []float64 {
	n, fade := samples(length), samples(0.5)
	y := made(n + fade)
	out := make([]float64, n)
	copy(out, y)
	for i := range fade {
		w := float64(i) / float64(fade)
		out[i] = y[i]*w + y[n+i]*(1-w)
	}
	mean := 0.0
	for _, v := range out {
		mean += v / float64(n)
	}
	for i := range out {
		out[i] -= mean
	}
	return out
}

// makeSound makes one sound, from -1 to 1.
func makeSound(n sound.Name) []float64 {
	for m := range sound.Material(len(materialVoices)) {
		seed := 910 + 10*int(m)
		switch n {
		case sound.Step(m):
			return peakTo(grains(m, 2, 0.03, 0.12, 1, seed), 0.35)
		case sound.Dig(m):
			return peakTo(grains(m, 1, 0, 0.09, 1, seed+1), 0.5)
		case sound.Break(m):
			return peakTo(grains(m, 7, 0.18, 0.35, 1.6, seed+2), 0.8)
		case sound.Place(m):
			s := grains(m, 1, 0, 0.14, 1.3, seed+3)
			add(s, thump(140, 70, 0.14, 0.04), 0.6, 0)
			return peakTo(s, 0.6)
		}
	}
	switch n {
	case sound.Splash:
		s := make([]float64, samples(0.6))
		f := newLowpass(4000)
		for i := range s {
			t := float64(i) / rate
			f.a = 1 - math.Exp(-2*math.Pi*(600+3400*math.Exp(-t/0.12))/rate)
			s[i] = f.at(noise(i, 990)) * swell(t, 0.005, 0.18)
		}
		for b := range 5 { // bubbles, rising
			at := 0.1 + 0.35*hash(b, 0, 991)
			add(s, chirp(500+500*hash(b, 1, 991), 2, 0.04), 0.25, samples(at))
		}
		return peakTo(s, 0.7)
	case sound.Pickup:
		s := chirp(700, 2, 0.12)
		add(s, chirp(1400, 2, 0.12), 0.25, 0)
		return peakTo(s, 0.35)
	case sound.Hurt:
		s := thump(150, 70, 0.25, 0.08)
		add(s, grains(sound.Earth, 1, 0, 0.25, 1, 992), 0.5, 0)
		return peakTo(s, 0.75)
	case sound.Eat:
		s := make([]float64, samples(0.45))
		for bite := range 3 {
			add(s, crunch(993+bite), 1, samples(0.15*float64(bite)))
		}
		return peakTo(s, 0.55)
	case sound.Call:
		// Two hums, the second falling: a mossback to itself.
		pitch := func(t float64) float64 {
			vib := 1 + 0.02*math.Sin(2*math.Pi*5*t)
			if t < 0.42 {
				return 150 * vib
			}
			return (150 - 50*(t-0.42)) * vib
		}
		loud := func(t float64) float64 {
			part := math.Mod(t, 0.45) / 0.42
			return math.Pow(math.Sin(math.Pi*min(part, 1)), 2)
		}
		return peakTo(voiced(0.9, pitch, loud, 450, 1100), 0.6)
	case sound.Cry:
		pitch := func(t float64) float64 { return 260 + 80*math.Sin(math.Pi*t/0.35) - 60*t }
		loud := func(t float64) float64 { return math.Sin(math.Pi * t / 0.35) }
		return peakTo(voiced(0.35, pitch, loud, 700, 1500), 0.65)
	case sound.Horn:
		// A long low note through a hollow crest, that swells, wavers and falls away: a
		// bugler to its herd.
		pitch := func(t float64) float64 {
			return (92 + 10*math.Sin(math.Pi*t/1.6)) * (1 + 0.012*math.Sin(2*math.Pi*4.5*t))
		}
		loud := func(t float64) float64 { return math.Pow(math.Sin(math.Pi*min(t/1.6, 1)), 0.7) }
		return peakTo(voiced(1.6, pitch, loud, 240, 520), 0.7)
	case sound.Alarm:
		// Two short blasts, higher and harder: a bugler that has seen something.
		pitch := func(t float64) float64 { return 168 + 30*math.Sin(math.Pi*math.Mod(t, 0.32)/0.28) }
		loud := func(t float64) float64 {
			part := math.Mod(t, 0.32) / 0.28
			return math.Pow(math.Sin(math.Pi*min(part, 1)), 0.6)
		}
		return peakTo(voiced(0.62, pitch, loud, 420, 900), 0.75)
	case sound.Chirp:
		// Three quick rising squeaks: a nipper.
		s := make([]float64, samples(0.3))
		for i := range 3 {
			add(s, chirp(1500+200*float64(i), 1.5, 0.07), 1, samples(0.1*float64(i)))
		}
		return peakTo(s, 0.5)
	case sound.Growl:
		// A low rattle from the throat that rises and dies: a quickclaw that hunts.
		pitch := func(t float64) float64 { return 58 + 14*math.Sin(math.Pi*t/0.9) }
		loud := func(t float64) float64 {
			return math.Sin(math.Pi*min(t/0.9, 1)) * (0.7 + 0.3*math.Sin(2*math.Pi*23*t)) // it rattles
		}
		return peakTo(voiced(0.9, pitch, loud, 300, 1100), 0.55)
	case sound.Screech:
		// A harsh cry that leaps up and breaks off, with breath in it.
		s := voiced(0.4, func(t float64) float64 { return 520 + 900*t - 1400*t*t },
			func(t float64) float64 { return math.Pow(math.Sin(math.Pi*t/0.4), 0.5) }, 1500, 2900)
		f := newHighpass(2200)
		for i := range s {
			s[i] += 0.5 * f.at(noise(i, 996)) * swell(float64(i)/rate, 0.02, 0.25)
		}
		return peakTo(s, 0.7)
	case sound.Bellow:
		// A deep note from a great chest that swells, roughens and drops: a shieldbrow
		// that has seen what it does not like.
		pitch := func(t float64) float64 { return 74 + 26*math.Sin(math.Pi*min(t/0.5, 1)) - 22*max(t-0.6, 0) }
		loud := func(t float64) float64 {
			return math.Pow(math.Sin(math.Pi*min(t/1.1, 1)), 0.6) * (0.8 + 0.2*math.Sin(2*math.Pi*17*t))
		}
		return peakTo(voiced(1.1, pitch, loud, 360, 780), 0.8)
	case sound.Grunt:
		// Two low short breaths through the nose: a great beast at its feed.
		pitch := func(t float64) float64 { return 66 - 14*math.Mod(t, 0.34)/0.3 }
		loud := func(t float64) float64 {
			part := math.Mod(t, 0.34) / 0.3
			return math.Pow(math.Sin(math.Pi*min(part, 1)), 1.5)
		}
		return peakTo(voiced(0.64, pitch, loud, 280, 640), 0.5)
	case sound.Rattle:
		// Dry plates shaken one on another, quicker and quicker, over a hiss of breath.
		s := make([]float64, samples(0.8))
		hiss := newHighpass(1800)
		for i := range s {
			s[i] = 0.25 * hiss.at(noise(i, 993)) * swell(float64(i)/rate, 0.15, 0.6)
		}
		at := 0.0
		for k := range 11 {
			knock := make([]float64, samples(0.03))
			f := newBand(700+500*hash(k, 0, 994), 4)
			for i := range knock {
				x := 0.0
				if i == 0 {
					x = 1
				}
				knock[i] = f.at(x) * swell(float64(i)/rate, 0, 0.008)
			}
			add(s, knock, 3, samples(at))
			at += 0.11 - 0.006*float64(k)
		}
		return peakTo(s, 0.6)
	case sound.Thud:
		// The foot of a giant coming down: a deep blow that sinks, with the ground's
		// own rumble after it.
		s := thump(64, 26, 0.7, 0.16)
		rumble := newLowpass(90)
		for i := range s {
			s[i] += 0.8 * rumble.at(noise(i, 992)) * swell(float64(i)/rate, 0.01, 0.3)
		}
		return peakTo(s, 0.85)
	case sound.Drone:
		// A note too low to place, held long, that swells and sinks: a cloudgrazer to
		// its mate, across a plain.
		pitch := func(t float64) float64 { return (44 + 7*math.Sin(math.Pi*t/2.6)) * (1 + 0.01*math.Sin(2*math.Pi*3*t)) }
		loud := func(t float64) float64 { return math.Pow(math.Sin(math.Pi*min(t/2.6, 1)), 0.8) }
		return peakTo(voiced(2.6, pitch, loud, 180, 420), 0.7)
	case sound.Roar:
		// The dread has seen what it wants: a roar from a throat as wide as a cave, that
		// opens, tears and falls away, with its breath in it.
		pitch := func(t float64) float64 { return 96 + 64*math.Sin(math.Pi*min(t/0.5, 1)) - 46*max(t-0.7, 0) }
		loud := func(t float64) float64 {
			return math.Pow(math.Sin(math.Pi*min(t/1.9, 1)), 0.5) * (0.72 + 0.28*math.Sin(2*math.Pi*31*t))
		}
		s := voiced(1.9, pitch, loud, 520, 1250)
		breath := newBand(900, 1.2)
		for i := range s {
			s[i] += 0.5 * breath.at(noise(i, 991)) * swell(float64(i)/rate, 0.2, 1.2)
		}
		return peakTo(s, 0.9)
	case sound.Rumble:
		// The dread, to itself: breath drawn through a great chest, with a slow rattle
		// under it, more felt than heard.
		pitch := func(t float64) float64 { return 38 + 6*math.Sin(math.Pi*t/1.4) }
		loud := func(t float64) float64 {
			return math.Sin(math.Pi*min(t/1.4, 1)) * (0.6 + 0.4*math.Sin(2*math.Pi*11*t))
		}
		return peakTo(voiced(1.4, pitch, loud, 200, 520), 0.65)
	case sound.Geyser:
		// A rush that comes up out of the ground and roars for as long as it spouts,
		// then falls back: steam, with the weight of water under it.
		s := make([]float64, samples(3.2))
		hiss, deep := newBand(2600, 0.8), newLowpass(140)
		for i := range s {
			t := float64(i) / rate
			swell := math.Min(1, t/0.25) * math.Min(1, (3.2-t)/0.7)
			s[i] = (0.7*hiss.at(noise(i, 989)) + 1.4*deep.at(noise(i, 990))) * swell * (0.85 + 0.15*math.Sin(2*math.Pi*7*t))
		}
		return peakTo(s, 0.8)
	case sound.Tremor:
		// The ground under a fire mountain that is about to break: very low, with a slow
		// roll in it, and even enough that one can follow another.
		s := make([]float64, samples(2.4))
		deep, grit := newLowpass(70), newBand(180, 0.9)
		for i := range s {
			t := float64(i) / rate
			fade := math.Min(1, t/0.5) * math.Min(1, (2.4-t)/0.5)
			s[i] = (1.6*deep.at(noise(i, 993)) + 0.25*grit.at(noise(i, 994))) * fade * (0.75 + 0.25*math.Sin(2*math.Pi*3*t))
		}
		return peakTo(s, 0.85)
	case sound.Boom:
		// A star that struck, far off: one deep blow, and the roll of it over the land
		// after, dying away.
		s := make([]float64, samples(3))
		deep, roll := newLowpass(90), newLowpass(220)
		for i := range s {
			t := float64(i) / rate
			blow := math.Exp(-t / 0.12)
			s[i] = 2.2*deep.at(noise(i, 995))*(blow+0.5*math.Exp(-t/0.9)) + 0.5*roll.at(noise(i, 996))*math.Exp(-t/0.6)*(0.7+0.3*math.Sin(2*math.Pi*5*t))
		}
		return peakTo(s, 0.95)
	case sound.Trumpet:
		// A blast through a long trunk: it leaps up, holds, and falls away, brassy and
		// rough.
		pitch := func(t float64) float64 { return 300 + 260*math.Sin(math.Pi*min(t/0.25, 0.5)) - 120*max(t-0.7, 0) }
		loud := func(t float64) float64 {
			return math.Pow(math.Sin(math.Pi*min(t/1.0, 1)), 0.5) * (0.82 + 0.18*math.Sin(2*math.Pi*23*t))
		}
		return peakTo(voiced(1.0, pitch, loud, 900, 2100), 0.8)
	case sound.Howl:
		// One voice that climbs, holds a long note and sinks: it carries a long way.
		pitch := func(t float64) float64 {
			return 290 + 210*math.Sin(math.Pi/2*min(t/0.7, 1)) - 150*max(t-1.7, 0)/0.7
		}
		loud := func(t float64) float64 {
			return math.Pow(math.Sin(math.Pi*min(t/2.4, 1)), 0.7) * (0.9 + 0.1*math.Sin(2*math.Pi*6*t))
		}
		return peakTo(voiced(2.4, pitch, loud, 620, 1400), 0.75)
	case sound.Snarl:
		// A cat's snarl: harsh, from the throat, quick to rise and gone.
		pitch := func(t float64) float64 { return 150 + 90*math.Sin(math.Pi*min(t/0.3, 1)) }
		loud := func(t float64) float64 {
			return math.Pow(math.Sin(math.Pi*min(t/0.6, 1)), 0.5) * (0.55 + 0.45*math.Sin(2*math.Pi*38*t))
		}
		s := voiced(0.6, pitch, loud, 700, 2300)
		rasp := newBand(2600, 1.5)
		for i := range s {
			s[i] += 0.45 * rasp.at(noise(i, 997)) * swell(float64(i)/rate, 0.05, 0.4)
		}
		return peakTo(s, 0.8)
	case sound.Whirr:
		// Stiff wings that beat forty times a second: a dry rattle, that comes and goes.
		s := make([]float64, samples(0.7))
		dry := newBand(2400, 1.2)
		for i := range s {
			t := float64(i) / rate
			beat := math.Pow(0.5+0.5*math.Sin(2*math.Pi*40*t), 4)
			s[i] = dry.at(noise(i, 998)) * beat * swell(t, 0.08, 0.5)
		}
		return peakTo(s, 0.7)
	case sound.Hiss:
		// Breath let out slowly through a row of teeth: it swells, and stops short.
		s := make([]float64, samples(0.9))
		high, low := newHighpass(3000), newBand(900, 1.5)
		for i := range s {
			t := float64(i) / rate
			swell := math.Pow(math.Sin(math.Pi*math.Min(t/0.9, 1)), 0.6) * math.Min(1, (0.9-t)/0.05)
			s[i] = (0.8*high.at(noise(i, 999)) + 0.35*low.at(noise(i, 1001))) * swell
		}
		return peakTo(s, 0.6)
	case sound.Moan:
		// A hollow note that rises a little and falls a long way: a lakeneck, with its
		// head up, over still water.
		pitch := func(t float64) float64 {
			return (150 + 26*math.Sin(math.Pi*min(t/0.8, 1)) - 22*max(t-0.8, 0)) * (1 + 0.008*math.Sin(2*math.Pi*5*t))
		}
		loud := func(t float64) float64 { return math.Pow(math.Sin(math.Pi*min(t/2.2, 1)), 0.7) }
		return peakTo(voiced(2.2, pitch, loud, 300, 700), 0.6)
	case sound.Crackle:
		// A low roar, even all through so that it can follow itself, with pops in it.
		s := make([]float64, samples(1))
		roar := newLowpass(250)
		for i := range s {
			t := float64(i) / rate
			s[i] = roar.at(noise(i, 996)) * 1.5 * math.Min(1, math.Min(t, 1-t)/0.05)
		}
		for p := range 9 {
			pop := make([]float64, samples(0.02))
			f := newBand(2000+2000*hash(p, 0, 997), 3)
			for i := range pop {
				x := 0.0
				if i == 0 {
					x = 1
				}
				pop[i] = f.at(x) * swell(float64(i)/rate, 0, 0.004)
			}
			add(s, pop, 4+4*hash(p, 1, 997), samples(0.05+0.9*hash(p, 2, 997)))
		}
		return peakTo(s, 0.5)
	case sound.Drip:
		s := make([]float64, samples(0.25))
		phase := 0.0
		for i := range s {
			t := float64(i) / rate
			phase += 2 * math.Pi * (900 + 900*math.Exp(-t/0.01)) / rate
			s[i] = math.Sin(phase) * swell(t, 0.001, 0.04)
		}
		add(s, append([]float64(nil), s[:samples(0.1)]...), 0.3, samples(0.07)) // its echo
		return peakTo(s, 0.4)
	case sound.Wind:
		return scaleTo(loop(6, func(n int) []float64 {
			s := make([]float64, n)
			f := newLowpass(500)
			for i := range s {
				t := float64(i) / rate
				cut := 500 + 350*math.Sin(2*math.Pi*t/6) + 150*math.Sin(2*math.Pi*t/2+1)
				f.a = 1 - math.Exp(-2*math.Pi*cut/rate)
				s[i] = f.at(noise(i, 998)) * (0.6 + 0.4*math.Sin(2*math.Pi*t/3+0.5))
			}
			return s
		}), 0.45)
	case sound.Cave:
		return scaleTo(loop(6, func(n int) []float64 {
			s := make([]float64, n)
			f, g := newLowpass(120), newLowpass(120)
			for i := range s {
				t := float64(i) / rate
				s[i] = g.at(f.at(noise(i, 999)))*4 + 0.3*math.Sin(2*math.Pi*55*t)
			}
			return s
		}), 0.35)
	}
	return nil
}

// chirp is a short tone that rises by a ratio over its length.
func chirp(from, ratio, length float64) []float64 {
	s := make([]float64, samples(length))
	phase := 0.0
	for i := range s {
		t := float64(i) / rate
		phase += 2 * math.Pi * from * math.Pow(ratio, t/length) / rate
		s[i] = math.Sin(phase) * swell(t, 0.003, length/3)
	}
	return s
}

// crunch is one bite: a few grains of noise in a band, close together.
func crunch(seed int) []float64 {
	s := make([]float64, samples(0.1))
	f := newBand(1800, 1.5)
	for i := range s {
		t := float64(i) / rate
		env := 0.0
		for g := range 4 {
			env += swell(t-0.02*hash(g, seed, 904)*float64(g), 0.002, 0.012)
		}
		s[i] = f.at(noise(i, seed)) * env
	}
	return s
}

// retune moves a band to another frequency, keeping what is ringing in it.
func retune(f band, hz float64) band {
	g := newBand(hz, 6)
	g.x1, g.x2, g.y1, g.y2 = f.x1, f.x2, f.y1, f.y2
	return g
}

// wav is a sound as a WAV file: mono, 16 bits a sample, at sound.Rate.
func wav(s []float64) []byte {
	var b bytes.Buffer
	put := func(v any) { binary.Write(&b, binary.LittleEndian, v) }
	b.WriteString("RIFF")
	put(uint32(36 + 2*len(s)))
	b.WriteString("WAVEfmt ")
	put(uint32(16))
	put(uint16(1)) // PCM
	put(uint16(1)) // one channel
	put(uint32(sound.Rate))
	put(uint32(2 * sound.Rate))
	put(uint16(2))
	put(uint16(16))
	b.WriteString("data")
	put(uint32(2 * len(s)))
	for _, v := range s {
		put(int16(math.Round(min(max(v, -1), 1) * 32767)))
	}
	return b.Bytes()
}

// writeSounds makes every sound and writes it into a folder.
func writeSounds(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for n := range sound.Count {
		if err := os.WriteFile(filepath.Join(dir, n.File()), wav(makeSound(n)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

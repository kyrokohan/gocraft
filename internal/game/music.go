package game

import (
	"math"
	"math/rand/v2"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// The music of the tribe. At night, at any fire where two or more people sit, one
// drums and one plays a bone flute: at a camp, and at the player's fire. Each fire has a
// tune of its own from the seed, which changes a little each night, and it is heard as
// any sound is, from the fire, so only near it.
const (
	tuneSteps  = 16   // steps of a tune, each an eighth of a beat of four to a bar: two bars
	drumGain   = 0.45 // how loud a drum is at the fire
	fluteGain  = 0.4  // and a note of the flute
	fluteSwing = 0.45
)

// drumSounds are the drums a tune's steps strike: low and high.
var drumSounds = [2]sound.Name{sound.Drum, sound.Tap}

// tune is what is played at one fire on one night: how long a step is, in seconds,
// which drum each step strikes, and which note of the flute each beat plays, -1 for
// none.
type tune struct {
	beat  float64
	drums [tuneSteps]int
	flute [tuneSteps / 2]int
}

// tuneOf returns the tune of a fire on a night, as the seed makes it. The fire's tune
// keeps its pace from night to night; each night two of its notes and two of its drums
// may be others.
func tuneOf(seed int64, fire world.Cell, night int) tune {
	at := uint64(uint32(fire.X))<<32 | uint64(uint32(fire.Z))
	rng := rand.New(rand.NewPCG(uint64(seed)^0x6d75736963, at))
	t := tune{beat: 60 / (84 + 28*rng.Float64()) / 2}
	for i := range t.drums {
		switch {
		case i%8 == 0:
			t.drums[i] = 0 // the low drum begins each bar
		case i%4 == 0:
			t.drums[i] = rng.IntN(2)
		case rng.Float64() < 0.35:
			t.drums[i] = 1
		default:
			t.drums[i] = -1
		}
	}
	note := 2 * rng.IntN(2)
	for i := range t.flute {
		switch {
		case i == len(t.flute)-1:
			t.flute[i] = 0 // it comes home at the end
		case rng.Float64() < 0.25:
			t.flute[i] = -1
		default:
			note = min(max(note+[]int{-2, -1, -1, 0, 1, 1, 2}[rng.IntN(7)], 0), sound.Notes-1)
			t.flute[i] = note
		}
	}
	tonight := rand.New(rand.NewPCG(uint64(seed)^0x6e69676874, at^uint64(night)))
	for range 2 {
		t.flute[tonight.IntN(len(t.flute)-1)] = tonight.IntN(sound.Notes+1) - 1
		if i := 2*tonight.IntN(tuneSteps/2) + 1; t.drums[i] < 0 {
			t.drums[i] = 1
		} else {
			t.drums[i] = -1
		}
	}
	return t
}

// nightOf is which night it is after days of play: it begins at sunset.
func nightOf(days float64) int {
	return int(math.Floor(firstMorning + days + 0.25))
}

// playMusic plays the music at each fire where two or more sit, dt seconds on, and
// moves the drummer's arm to it and holds up the flute. The first who sits at a fire
// drums, and the next plays.
func (g *Game) playMusic(dt float64) {
	before := g.music
	g.music += dt
	for i := range g.people {
		g.people[i].swing = 0
	}
	if g.fauna.sun >= sunUp {
		return
	}
	first := map[world.Cell]int{}
	for i := range g.people {
		one := &g.people[i]
		if !one.seated || one.away > 0 || one.follows {
			continue
		}
		drummer, ok := first[one.fire]
		if !ok {
			first[one.fire] = i
			continue
		}
		if drummer < 0 {
			continue // the music there has its two already
		}
		first[one.fire] = -1
		t := tuneOf(g.world.Seed, one.fire, nightOf(g.days))
		step := int(math.Floor(g.music / t.beat))
		at := cellMid(one.fire.X, one.fire.Y, one.fire.Z)
		if step > int(math.Floor(before/t.beat)) {
			s := step % tuneSteps
			if d := t.drums[s]; d >= 0 {
				g.ears.Play(drumSounds[d], at, drumGain)
			}
			if f := t.flute[s/2]; s%2 == 0 && f >= 0 {
				g.ears.Play(sound.Note(f), at, fluteGain)
			}
		}
		if t.drums[step%tuneSteps] >= 0 {
			g.people[drummer].swing = g.music/t.beat - float64(step)
		}
		one.swing = fluteSwing
	}
}

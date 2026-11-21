package sound

import (
	"encoding/binary"
	"math"
	"math/rand/v2"
	"sync"

	"glonk.dev/geometry"
)

// The mixer's numbers.
const (
	// Voices is the most sounds that play at once: a new one past that takes the place
	// of the one nearest its end. The music at a fire takes a few of them.
	Voices = 24

	// Hearing is how far away a sound can be heard, in blocks. It fades to nothing there.
	Hearing = 24.0

	// slew is how many seconds a loop takes to come up to the loudness it is set to, or
	// down to it, so that it never jumps.
	slew = 0.6

	// spread is how much the pitch of a sound is moved, up or down at random, each time
	// it plays, so that one played again and again does not sound the same each time:
	// but for the music, which keeps in tune.
	spread = 0.06

	// boost is how much louder the mix is made at the end. The clips leave room above
	// them, and a sound loses loudness to its distance and to the two ears' share of it,
	// so without it the game is far quieter than other programs; limit takes what goes
	// past full.
	boost = 3.0
)

// voice is a sound playing.
type voice struct {
	clip        Name
	at          float64 // in samples of its clip, with a part between two
	rate        float64 // samples of the clip to a sample made
	left, right float64 // how loud in each ear
}

// Mixer mixes the sounds playing into one stream of stereo samples, for a speaker to read.
// Its methods may be called from any goroutine.
type Mixer struct {
	mu     sync.Mutex
	clips  [Count][]float32
	voices []voice
	rng    *rand.Rand

	ear, right geometry.Vector3 // where the listener is, and the way to its right
	volume     float64

	// The two loops: how far into each, how loud each is, and how loud it is to be.
	loopAt    [2]float64
	loopGain  [2]float64
	loopAimed [2]float64
}

// NewMixer returns a mixer of the clips, each at Rate, playing nothing, at full volume,
// with its listener at the origin facing +z.
func NewMixer(clips [Count][]float32) *Mixer {
	return &Mixer{
		clips: clips, rng: rand.New(rand.NewPCG(1, 2)),
		right: geometry.Vector3{X: -1}, volume: 1,
	}
}

// Listen puts the listener at a place, with its right ear the way right points.
func (m *Mixer) Listen(at, right geometry.Vector3) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ear, m.right = at, right
}

// SetVolume sets how loud everything is, from 0 to 1.
func (m *Mixer) SetVolume(v float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.volume = min(max(v, 0), 1)
}

// Ambient sets how loud the wind and the air of caves are to be, each from 0 to 1. They
// come to it over a little while.
func (m *Mixer) Ambient(wind, cave float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loopAimed = [2]float64{min(max(wind, 0), 1), min(max(cave, 0), 1)}
}

// Play plays a sound from a place in the world, as the listener hears it from where it
// is: not at all from Hearing or farther. gain is how loud it is where it is made.
func (m *Mixer) Play(n Name, at geometry.Vector3, gain float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	left, right := m.ears(at, Hearing)
	m.start(n, left*gain, right*gain)
}

// PlayFar is Play for a sound that carries: it is heard from as far as hearing, and
// fades to nothing there. A call across a valley is one.
func (m *Mixer) PlayFar(n Name, at geometry.Vector3, gain, hearing float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	left, right := m.ears(at, hearing)
	m.start(n, left*gain, right*gain)
}

// PlayNear plays a sound the listener makes itself, in both ears alike.
func (m *Mixer) PlayNear(n Name, gain float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.start(n, gain*math.Sqrt2/2, gain*math.Sqrt2/2)
}

// ears returns how loud a sound at a place is in each ear, if it can be heard from as
// far as hearing: less the farther it is, and more in the ear on its side, with the two
// together as loud wherever it is.
func (m *Mixer) ears(at geometry.Vector3, hearing float64) (left, right float64) {
	to := at.Sub(m.ear)
	d := to.Length()
	if d >= hearing {
		return 0, 0
	}
	near := 1 - d/hearing
	gain := near * near
	side := 0.0 // from -1, to the left, to 1, to the right
	if flat := (geometry.Vector3{X: to.X, Z: to.Z}); flat.Length() > 1e-9 {
		side = flat.Normalize().Dot(m.right)
	}
	angle := (side + 1) * math.Pi / 4
	return gain * math.Cos(angle), gain * math.Sin(angle)
}

// start starts a voice, if it would be heard.
func (m *Mixer) start(n Name, left, right float64) {
	if n >= Count || len(m.clips[n]) == 0 || left+right < 1e-4 {
		return
	}
	v := voice{clip: n, rate: 1 + spread*(2*m.rng.Float64()-1), left: left, right: right}
	if n.Tuned() {
		v.rate = 1
	}
	if len(m.voices) < Voices {
		m.voices = append(m.voices, v)
		return
	}
	// The one with the least left to play gives way.
	last, least := 0, math.Inf(1)
	for i, o := range m.voices {
		if left := float64(len(m.clips[o.clip])) - o.at; left < least {
			last, least = i, left
		}
	}
	m.voices[last] = v
}

// Playing returns the sounds playing, but for the loops.
func (m *Mixer) Playing() []Name {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]Name, len(m.voices))
	for i, v := range m.voices {
		names[i] = v.clip
	}
	return names
}

// Read mixes the sounds into p as stereo samples, 32-bit floats, little end first, the
// left before the right: what oto plays. It fills the whole of p but a part of a sample.
func (m *Mixer) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	frames := len(p) / 8
	step := 1 / (slew * Rate)
	for f := range frames {
		var left, right float64
		kept := m.voices[:0]
		for _, v := range m.voices {
			clip := m.clips[v.clip]
			i := int(v.at)
			if i >= len(clip)-1 {
				continue // it has played
			}
			part := v.at - float64(i)
			s := float64(clip[i])*(1-part) + float64(clip[i+1])*part
			left, right = left+s*v.left, right+s*v.right
			v.at += v.rate
			kept = append(kept, v)
		}
		m.voices = kept

		for l, n := range [2]Name{Wind, Cave} {
			g := &m.loopGain[l]
			*g += max(min(m.loopAimed[l]-*g, step), -step)
			clip := m.clips[n]
			if len(clip) == 0 {
				continue
			}
			s := float64(clip[int(m.loopAt[l])]) * *g * math.Sqrt2 / 2
			left, right = left+s, right+s
			if m.loopAt[l]++; int(m.loopAt[l]) >= len(clip) {
				m.loopAt[l] = 0
			}
		}

		gain := boost * m.volume
		binary.LittleEndian.PutUint32(p[8*f:], math.Float32bits(float32(limit(left*gain))))
		binary.LittleEndian.PutUint32(p[8*f+4:], math.Float32bits(float32(limit(right*gain))))
	}
	return frames * 8, nil
}

// limit keeps a sample from going past full: it is left as it is up to 0.8, and bent
// smoothly towards 1 above that, so that many sounds at once do not crack.
func limit(s float64) float64 {
	const knee = 0.8
	if a := math.Abs(s); a > knee {
		return math.Copysign(knee+(1-knee)*math.Tanh((a-knee)/(1-knee)), s)
	}
	return s
}

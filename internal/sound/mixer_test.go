package sound

import (
	"encoding/binary"
	"math"
	"slices"
	"testing"

	"glonk.dev/geometry"
)

// tone is a clip of n samples all at one level.
func tone(n int, level float32) []float32 {
	c := make([]float32, n)
	for i := range c {
		c[i] = level
	}
	return c
}

// mixer is a mixer whose every clip is a tenth of a second at half level, but for any
// given, with its listener at the origin and its right ear to -x.
func mixer(clips map[Name][]float32) *Mixer {
	var all [Count][]float32
	for n := range all {
		all[n] = tone(Rate/10, 0.5)
		if c, ok := clips[Name(n)]; ok {
			all[n] = c
		}
	}
	return NewMixer(all)
}

// read reads n stereo samples from the mixer and returns the left and the right.
func read(m *Mixer, n int) (left, right []float64) {
	p := make([]byte, 8*n)
	if got, err := m.Read(p); err != nil || got != len(p) {
		panic("a short read")
	}
	for i := range n {
		left = append(left, float64(math.Float32frombits(binary.LittleEndian.Uint32(p[8*i:]))))
		right = append(right, float64(math.Float32frombits(binary.LittleEndian.Uint32(p[8*i+4:]))))
	}
	return left, right
}

func noLoops() map[Name][]float32 {
	return map[Name][]float32{Wind: nil, Cave: nil}
}

func TestEverySoundHasAFileOfItsOwn(t *testing.T) {
	seen := map[string]bool{}
	for n := range Count {
		f := n.File()
		if seen[f] || len(f) < 5 {
			t.Errorf("sound %d has the file %q, which is not one of its own", n, f)
		}
		seen[f] = true
	}
	if Step(Wood).File() != "step-wood.wav" || Break(Glass).File() != "break-glass.wav" || Cave.File() != "cave.wav" {
		t.Errorf("files: %s, %s, %s", Step(Wood).File(), Break(Glass).File(), Cave.File())
	}
}

func TestASoundIsLouderInTheEarOnItsSideAndFadesWithDistance(t *testing.T) {
	m := mixer(noLoops())
	right := geometry.Vector3{X: -1}

	l, r := m.ears(geometry.Vector3{X: -4})
	if r <= l || l < 0 {
		t.Errorf("a sound to the right: left %.3f, right %.3f", l, r)
	}
	l, r = m.ears(geometry.Vector3{X: 4})
	if l <= r {
		t.Errorf("a sound to the left: left %.3f, right %.3f", l, r)
	}
	l, r = m.ears(geometry.Vector3{Z: 4})
	if math.Abs(l-r) > 1e-9 {
		t.Errorf("a sound ahead: left %.3f, right %.3f, wanted the same", l, r)
	}
	nearL, nearR := m.ears(right.Scale(2))
	farL, farR := m.ears(right.Scale(12))
	if !(farL*farL+farR*farR < nearL*nearL+nearR*nearR) {
		t.Errorf("a sound 12 away is as loud as one 2 away")
	}
	if l, r := m.ears(geometry.Vector3{Z: Hearing}); l != 0 || r != 0 {
		t.Errorf("a sound at the edge of hearing is heard: %.3f, %.3f", l, r)
	}
	// Wherever it is round the listener, at one distance it is as loud in all.
	for _, at := range []geometry.Vector3{{X: 5}, {X: -5}, {Z: 5}, {X: 3, Z: 4}} {
		l, r := m.ears(at)
		if p := l*l + r*r; math.Abs(p-nearPower(5)) > 1e-9 {
			t.Errorf("at %v the sound has a power of %.4f, wanted %.4f", at, p, nearPower(5))
		}
	}
}

func nearPower(d float64) float64 {
	g := (1 - d/Hearing) * (1 - d/Hearing)
	return g * g
}

func TestASoundPlaysOnceThroughAndIsGone(t *testing.T) {
	m := mixer(noLoops())
	m.voices = append(m.voices, voice{clip: Pickup, rate: 1, left: 1, right: 1})

	left, right := read(m, Rate/10+50)
	if math.Abs(left[10]-0.5) > 1e-6 || math.Abs(right[10]-0.5) > 1e-6 {
		t.Errorf("while it plays: %.3f, %.3f, wanted 0.5 in each", left[10], right[10])
	}
	if left[Rate/10+20] != 0 || len(m.Playing()) != 0 {
		t.Errorf("after its end: %.3f, and %v still playing", left[Rate/10+20], m.Playing())
	}
}

func TestNoMoreThanVoicesPlayAndTheOldestGivesWay(t *testing.T) {
	m := mixer(map[Name][]float32{Wind: nil, Cave: nil, Hurt: tone(Rate, 0.1)})
	for range Voices {
		m.PlayNear(Pickup, 1)
	}
	read(m, 100) // they are all a little way on
	m.PlayNear(Hurt, 1)

	playing := m.Playing()
	if len(playing) != Voices || !slices.Contains(playing, Hurt) {
		t.Errorf("playing %v, wanted %d, the new one among them", playing, Voices)
	}
}

func TestManySoundsAtOnceDoNotGoPastFull(t *testing.T) {
	m := mixer(map[Name][]float32{Wind: nil, Cave: nil, Pickup: tone(Rate, 1)})
	for range Voices {
		m.PlayNear(Pickup, 1)
	}
	left, right := read(m, 200)
	for i := range left {
		if math.Abs(left[i]) > 1 || math.Abs(right[i]) > 1 {
			t.Fatalf("sample %d is %.3f, %.3f: past full", i, left[i], right[i])
		}
	}
	if left[100] < 0.95 {
		t.Errorf("at full, a sample is %.3f, wanted near 1", left[100])
	}
	if limit(0.5) != 0.5 || limit(-0.7) != -0.7 {
		t.Error("a quiet sample was changed")
	}
}

func TestTheLoopsComeUpSlowlyAndGoRound(t *testing.T) {
	wind := make([]float32, 100)
	for i := range wind {
		wind[i] = float32(i%2)*2 - 1 // +1 and -1 by turns
	}
	m := mixer(map[Name][]float32{Wind: wind, Cave: nil})
	m.Ambient(1, 0)

	left, _ := read(m, Rate)
	if math.Abs(left[1]) > 0.01 {
		t.Errorf("the wind is at %.3f at once, wanted it to come up", left[1])
	}
	if a := math.Abs(left[Rate-1]); a < 0.65 {
		t.Errorf("a second on the wind is at %.3f, wanted it up", a)
	}
	if m.loopAt[0] >= 100 {
		t.Errorf("the loop is %v into a clip of 100", m.loopAt[0])
	}
}

func TestVolumeScalesEverything(t *testing.T) {
	m := mixer(noLoops())
	m.SetVolume(0.5)
	m.voices = append(m.voices, voice{clip: Pickup, rate: 1, left: 1, right: 1})
	left, _ := read(m, 10)
	if math.Abs(left[5]-0.25) > 1e-6 {
		t.Errorf("at half volume a half-level sound is %.3f, wanted 0.25", left[5])
	}
	m.SetVolume(0)
	m.PlayNear(Eat, 1)
	if left, _ := read(m, 10); left[5] != 0 {
		t.Errorf("at no volume: %.3f", left[5])
	}
}

func TestASoundTooFarAwayIsNotStarted(t *testing.T) {
	m := mixer(noLoops())
	m.Play(Call, geometry.Vector3{Z: Hearing + 1}, 1)
	if len(m.Playing()) != 0 {
		t.Errorf("playing %v from beyond hearing", m.Playing())
	}
	m.Listen(geometry.Vector3{Z: 10}, geometry.Vector3{X: -1})
	m.Play(Call, geometry.Vector3{Z: Hearing + 1}, 1)
	if len(m.Playing()) != 1 {
		t.Errorf("playing %v once the listener came near", m.Playing())
	}
}

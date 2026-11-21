package main

import (
	"math"
	"testing"

	"github.com/kyrokohan/skyspear/internal/sound"
)

// strength returns how strong a frequency is in a sound: the power of the one bin of a
// Fourier transform at it.
func strength(s []float64, hz float64) float64 {
	c := 2 * math.Cos(2*math.Pi*hz/rate)
	var s1, s2 float64
	for _, x := range s {
		s1, s2 = x+c*s1-s2, s1
	}
	return s1*s1 + s2*s2 - c*s1*s2
}

func TestTheFluteIsInTuneInOneFiveNoteScale(t *testing.T) {
	for i, semis := range []float64{0, 3, 5, 7, 10} {
		hz := fluteHz(i)
		if want := fluteHz(0) * math.Pow(2, semis/12); math.Abs(hz-want) > 1e-9 {
			t.Errorf("note %d of the flute is %.2f Hz, wanted %.2f, %v semitones over the first", i, hz, want, semis)
		}
		s := makeSound(sound.Note(i))
		held := s[len(s)/4 : 3*len(s)/4]
		at := strength(held, hz)
		for _, off := range []float64{-1, 1} {
			if near := strength(held, hz*math.Pow(2, off/12)); at < 10*near {
				t.Errorf("note %d is not in tune: %.0f Hz is %.1f times as strong as a semitone off it", i, hz, at/near)
			}
		}
	}
}

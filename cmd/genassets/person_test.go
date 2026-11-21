package main

import (
	"math"
	"testing"

	"github.com/kyrokohan/skyspear/internal/assets"
)

// apart is how far two colors are, as the sum of how far each of their parts is.
func apart(a, b rgb) float64 {
	return math.Abs(a.r-b.r) + math.Abs(a.g-b.g) + math.Abs(a.b-b.b)
}

func TestThereAreAsManyLooksAsTheGameDrawsAndNoTwoAreAlike(t *testing.T) {
	if len(looks) != assets.PersonLooks {
		t.Fatalf("%d looks are drawn, and the game has %d", len(looks), assets.PersonLooks)
	}
	for i, a := range looks {
		for j, b := range looks[:i] {
			if apart(a.skin, b.skin) < 40 && apart(a.hair, b.hair) < 40 && apart(a.hide, b.hide) < 40 &&
				a.spots == b.spots && a.beard == b.beard {
				t.Errorf("looks %d and %d are alike", j+1, i+1)
			}
		}
	}
}

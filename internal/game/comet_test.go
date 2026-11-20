package game

import (
	"math"
	"strings"
	"testing"
)

func TestTheSkyspearGrowsEachNightAndComesAgain(t *testing.T) {
	g := underStars(t)
	last := 0.0
	for night := 1; night < cometNight; night++ {
		g.days = float64(night) - firstMorning // its midnight
		near, left := g.comet()
		if near <= last || near >= 1 || math.Abs(left-float64(cometNight-night)) > 1e-9 {
			t.Errorf("at the midnight of night %d the comet is %.3f near with %.2f days to come: wanted nearer than the %.3f of the night before, and %d days", night, near, left, last, cometNight-night)
		}
		last = near
	}
	if last < 0.9 {
		t.Errorf("on the night before it arrives the comet is %.2f near, wanted all but here", last)
	}
	if near, _ := cometAt(0); near != 0 {
		t.Errorf("as a world begins the comet is %.2f near, wanted a speck", near)
	}

	// The midnight it arrives it is here; then it is gone, and the next is a speck that
	// arrives so many days on.
	arrives := float64(cometNight) - firstMorning
	if near, left := cometAt(arrives); near != 1 || left != 0 {
		t.Errorf("at the midnight of night %d the comet is %.3f near with %.3f days to come", cometNight, near, left)
	}
	if near, left := cometAt(arrives + 0.01); near > 0.001 || math.Abs(left-(cometEvery-0.01)) > 1e-9 {
		t.Errorf("just after it arrived the comet is %.3f near, and the next comes in %.2f days: wanted a speck, and %d days", near, left, cometEvery)
	}
	if near, left := cometAt(arrives + cometEvery - 0.1); near < 0.99 || math.Abs(left-0.1) > 1e-9 {
		t.Errorf("a tenth of a day before the second comet it is %.3f near with %.2f days to come", near, left)
	}

	// F3 says how long it is.
	g.days, g.debug = 4, true
	if text, _ := drawn(t, g); !strings.Contains(text, "comet   in 7.70 days") {
		t.Errorf("on day 4 the frame numbers do not say the comet comes in 7.70 days:\n%s", text)
	}
}

func TestStarsFallMoreOftenAsTheSkyspearNears(t *testing.T) {
	g := underStars(t)
	// In the three nights before it arrives a star falls near the player every night.
	for _, night := range []int{9, 10, 11, 33, 34, 35} {
		if when, ok := g.starFall(night); !ok || int(math.Floor(when+firstMorning+0.5)) != night {
			t.Errorf("no star falls on night %d, in the last three before the comet arrives", night)
		}
	}
	// On other nights it is still about one in three.
	falls := 0
	for night := 13; night <= 32; night++ {
		if _, ok := g.starFall(night); ok {
			falls++
		}
	}
	if falls < 2 || falls > 12 {
		t.Errorf("a star falls on %d of the twenty nights after the comet has gone, wanted about one in three", falls)
	}
	if _, ok := among(t, 0.5).starFall(10); ok {
		t.Error("a star falls in a world that is not made from a seed, because a comet is near")
	}

	// More shooting stars each night, and the same number all through one night.
	rate := func(night int, off float64) float64 {
		g.days = float64(night) - firstMorning + off
		return g.streaks()
	}
	first, mid, eve := rate(1, 0), rate(6, 0), rate(cometNight-1, 0)
	if first < streaksAMinute || mid <= first || eve <= 2*mid || eve > streaksMost {
		t.Errorf("%.1f shooting stars a minute on night 1, %.1f on night 6 and %.1f on the night before the comet: wanted more and more, up to %v", first, mid, eve, streaksMost)
	}
	if rate(6, -0.2) != mid || rate(6, 0.2) != mid {
		t.Errorf("on night 6 there are %.2f shooting stars a minute in the evening, %.2f at midnight and %.2f before dawn: wanted the same", rate(6, -0.2), mid, rate(6, 0.2))
	}
	if after := rate(cometNight+1, 0); after > first+1 {
		t.Errorf("the night after the comet arrived there are %.1f shooting stars a minute, wanted few again", after)
	}
}

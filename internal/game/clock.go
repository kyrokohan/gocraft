package game

import (
	"fmt"
	"math"
)

// The world's clock. It runs while the game is played, and stands still on the title and
// in a pause.
const (
	dayLength    = 600.0 // seconds of play in a day of the game
	firstMorning = 0.3   // the time of day a world begins at: a little after sunrise
	quarterDay   = 0.25  // how far T moves the clock on
)

// timeOfDay returns the time of day, as render.Sky takes it, after days of play.
func timeOfDay(days float64) float64 {
	t := firstMorning + days
	return t - math.Floor(t)
}

// clockText says which day of the world it is after days of play, and the hour, as the
// player is shown it: "day 1, 07:12".
func clockText(days float64) string {
	// Counted in whole minutes, with a hair over for numbers like 0.3 + 0.7 that come
	// out a hair under what they are.
	minutes := int(math.Floor((firstMorning+days)*24*60 + 1e-6))
	day, minute := minutes/(24*60), minutes%(24*60)
	return fmt.Sprintf("day %d, %02d:%02d", day+1, minute/60, minute%60)
}

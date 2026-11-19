package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// A bedroll is where the player lies down: E on one makes it where the player wakes, from
// a faint too, and at night sleeps the player through to the morning, unless a hunter is
// near enough to come for a sleeper.

// sleepHunter is how near a hunter keeps the player from sleeping, in blocks.
const sleepHunter = 8

// noticeTime is how many seconds a notice stays on the screen.
const noticeTime = 3.0

// tell shows the player a line for a while, over play.
func (g *Game) tell(text string) {
	g.notice, g.noticeLeft = text, noticeTime
}

// lieDown lies the player down on the bedroll at a cell.
func (g *Game) lieDown(c world.Cell) {
	t := timeOfDay(g.days)
	day := t >= render.Sunrise && t <= render.Sunset
	if day && g.bed != nil && *g.bed == c {
		return // E held down: it has been said
	}
	g.bed = &c
	switch {
	case day:
		g.tell("you will wake here")
	case g.hunterWithin(sleepHunter):
		g.tell("you will wake here, but you cannot sleep with a hunter near")
	default:
		// The clock goes on to the next morning, the hour a world begins at, and the
		// night's hunters are gone with it.
		g.days += math.Mod(firstMorning-t+1, 1)
		g.fauna.dismiss()
		g.tell("you slept till morning, and will wake here")
	}
}

// bedPlace returns where the player wakes at its furBed, if it has one that is still
// there with room over it to stand: the world is brought in round it first. A bedroll
// that is gone is forgotten.
func (g *Game) bedPlace() (geometry.Vector3, bool) {
	if g.bed == nil {
		return geometry.Vector3{}, false
	}
	b := *g.bed
	g.world.Follow(float64(b.X), float64(b.Z))
	g.world.LoadAll()
	if g.world.At(b.X, b.Y, b.Z) != world.FurBed || solid(g.world, b.X, b.Y+1, b.Z) {
		g.bed = nil
		g.tell("your bedroll is gone, so you woke where you first began")
		return geometry.Vector3{}, false
	}
	return geometry.Vector3{X: float64(b.X) + 0.5, Y: float64(b.Y), Z: float64(b.Z) + 0.5}, true
}

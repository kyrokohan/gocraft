package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// What the player has met, and so can paint: a beast seen near with nothing in the way,
// or one that hit the player or was hit; and the comet, seen in the night sky. Each is
// told once, and kept in the save.

const (
	meetEvery = 0.25 // seconds between looks round for what has not been met
	meetReach = 32.0 // how near a beast is seen well enough to be painted
	cometMet  = 0.3  // how much of the comet is seen in the sky when it is met

	// A view of this many cells, of the shape of most terminals, is what is in view.
	viewCols, viewRows = 160, 45
)

// beastPicture is the picture of a kind of beast.
func beastPicture(k render.Kind) world.Picture {
	return world.FirstBeast + world.Picture(k)
}

// nameOf returns what a picture is called.
func nameOf(p world.Picture) string {
	if p < world.FirstBeast {
		return p.Name()
	}
	return render.Kind(p - world.FirstBeast).String()
}

// known returns the pictures the player can paint: a hand and a spiral, and what it has
// met, in the order of the pictures.
func (g *Game) known() []world.Picture {
	known := []world.Picture{world.Hand, world.Spiral}
	for p := world.Comet; p < world.Pictures; p++ {
		if g.met[p] {
			known = append(known, p)
		}
	}
	return known
}

// meet has the player meet what a picture shows, and tells it so the first time.
func (g *Game) meet(p world.Picture) {
	if p < world.Comet || p >= world.Pictures || g.met[p] {
		return // a hand and a spiral are known from the start
	}
	g.met[p] = true
	what := "the comet"
	if p >= world.FirstBeast {
		what = "a " + nameOf(p)
	}
	g.tell("you have seen " + what + ": you can paint it now")
}

// lookRound counts down dt, and each meetEvery meets the beasts not yet met that are in
// view, near and with nothing in the way; and the comet, when it is out in the night sky
// over the player.
func (g *Game) lookRound(dt float64) {
	if g.meetWait -= dt; g.meetWait > 0 {
		return
	}
	g.meetWait = meetEvery
	cam := g.eyes()
	for _, c := range g.fauna.all {
		p := beastPicture(c.kind)
		if g.met[p] || c.dead {
			continue
		}
		_, height := render.Size(c.kind, c.young)
		mid := c.Pos.Add(geometry.Vector3{Y: height / 2})
		if mid.Sub(cam.Position).Length() > meetReach {
			continue
		}
		if x, y, ok := cam.Project(mid, viewCols, viewRows); !ok || x < 0 || x >= viewCols || y < 0 || y >= viewRows {
			continue
		}
		if clearSight(g.world, cam.Position, mid) {
			g.meet(p)
		}
	}
	eye := cam.Position
	open := g.world.LightAt(int(math.Floor(eye.X)), int(math.Floor(eye.Y)), int(math.Floor(eye.Z))).Sky() == world.MaxLight
	if open && g.sky.CometSeen() > cometMet {
		g.meet(world.Comet)
	}
}

// savedMet returns what was met as a save holds it: by name.
func (g *Game) savedMet() (names []string) {
	for p := world.Comet; p < world.Pictures; p++ {
		if g.met[p] {
			names = append(names, nameOf(p))
		}
	}
	return names
}

// loadMet takes what was met from a save. A name this game does not know is let go.
func (g *Game) loadMet(names []string) {
	g.met = [world.Pictures]bool{}
	for _, name := range names {
		if name == nameOf(world.Comet) {
			g.met[world.Comet] = true
		} else if k, ok := render.KindNamed(name); ok {
			g.met[beastPicture(k)] = true
		}
	}
}

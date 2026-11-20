package game

import (
	"fmt"
	"math"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// The two views of the world: through the player's eyes, and from behind its right
// shoulder, where the player's own body is seen.
const (
	viewEyes   = "eyes"
	viewBehind = "behind"
)

// Where the camera of the view from behind is, in blocks.
const (
	behindBack  = 4.0  // behind the shoulder, along the way the player looks
	behindRight = 0.6  // to the right of the eyes, so the body does not hide what is aimed at
	behindUp    = 0.35 // and above them
	behindGap   = 0.3  // how far short of a block it stops
	behindLeast = 1.0  // with less room than this behind the player, the eyes see
)

// switchView changes from one view to the other, and keeps the choice between runs.
func (g *Game) switchView() {
	if g.settings.View == viewBehind {
		g.settings.View = viewEyes
	} else {
		g.settings.View = viewBehind
	}
	if err := g.settings.save(g.opts.Settings); err != nil {
		g.note = "the settings were not saved: " + reason(err)
	}
}

// How the view shakes at the step of a giant.
const (
	shakeTilt = 0.035 // how far the view tips each way at the hardest, in radians
	shakeDrop = 0.06  // and how far it moves up and down, in blocks
	shakeBeat = 9.0   // how many times a second it trembles
	shakeFade = 0.3   // how long a shake takes to die to about a third, in seconds
)

// viewed returns the camera the world is seen through, without fog, and whether it is
// the one behind the player, with the player's body in view.
//
// The camera from behind looks the way the eyes do, from behind the right shoulder. It
// is never in a block: a wall behind the player pulls it in, and where there is too
// little room for it the eyes see.
func (g *Game) viewed() (mesh.Camera, bool) {
	cam := g.eyes()
	if g.quake > 0 && !g.settings.NoShake {
		// The ground shakes under a giant's step: the view trembles, up and down.
		beat := math.Sin(2 * math.Pi * shakeBeat * g.shook)
		cam.Pitch += shakeTilt * g.quake * beat
		cam.Position.Y += shakeDrop * g.quake * beat
	}
	if g.settings.View != viewBehind {
		return cam, false
	}
	side := cam.Right().Scale(behindRight).Add(geometry.Vector3{Y: behindUp})
	out := side.Length()
	shoulder := cam.Position.Add(side.Scale(g.room(cam.Position, side.Scale(1/out), out) / out))
	back := g.room(shoulder, cam.Look().Neg(), behindBack)
	if back < behindLeast {
		return cam, false
	}
	cam.Position = shoulder.Add(cam.Look().Scale(-back))
	return cam, true
}

// room returns how far a camera can go from a point along a direction of length 1, up
// to far: it stops behindGap short of the first solid block. A plant or water does not
// stop it.
func (g *Game) room(from, along geometry.Vector3, far float64) float64 {
	r := world.NewRay(from, along)
	for {
		hit, ok := g.world.Next(&r, far+behindGap)
		if !ok {
			return far
		}
		if hit.Block.Info().Solid {
			return max(min(hit.T-behindGap, far), 0)
		}
	}
}

// nextLook changes the look the player's body wears to the one after it, and says
// which it is: through the eyes the body is not seen.
func (g *Game) nextLook() {
	g.worn = (g.worn + 1) % assets.PersonLooks
	g.tell(fmt.Sprintf("look %d of %d", g.worn+1, assets.PersonLooks))
}

// pace notes how far the player walked this frame from where it was, for the swing of
// its body's legs.
func (g *Game) pace(from geometry.Vector3) {
	moved := math.Hypot(g.player.Pos.X-from.X, g.player.Pos.Z-from.Z)
	if g.paced = moved > 1e-6 && g.player.OnGround; g.paced {
		g.strode += moved
	}
}

// body returns the player's body as it is drawn from behind: where the player is, doing
// what the player does.
func (g *Game) body() render.Person {
	p := &g.player
	one := render.Person{
		Pos: p.Pos, Yaw: p.Yaw, Tilt: p.Pitch, Look: g.worn,
		Crouched: p.Crouched, Hurt: g.flash > 0,
		Light: g.sky.Brightness(g.lightAt(p.head())),
		Held:  g.items[g.selected].Item,
	}
	if g.paced {
		one.Stride = math.Pi * g.strode / stepLength // a leg forward for each step heard
	}
	if g.swing > 0 {
		one.Swing = 1 - g.swing/swingTime
	}
	return one
}

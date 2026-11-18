package game

import (
	"math"
	"math/rand/v2"
	"slices"

	"glonk.dev/geometry"

	"github.com/kyrokohan/gocraft/internal/render"
	"github.com/kyrokohan/gocraft/internal/world"
)

// The gloam is a creature of the dark, in survival only. It gathers where there is no
// light, at night under the sky or in a cave, drifts after the player and hurts on touch.
// Light undoes it: in daylight or by a torch it fades away. It is not saved, as it does
// not last. Distances are in blocks, times in seconds, and these are first guesses.
const (
	gloamMost     = 3    // about at once
	gloamLook     = 5.0  // how often a place is looked for to gather a new one
	gloamNear     = 12.0 // the least distance from the player it gathers at
	gloamFar      = 28.0 // and the most
	gloamLost     = 64.0 // the distance it is let go at
	gloamSpeed    = 2.6  // slower than a walk, so that it can be got away from
	gloamHealth   = 12.0 // points: a hand takes 12 hits, an iron blade 2
	gloamBite     = 2.0  // points a touch takes from the player
	gloamBiteWait = 1.0  // between touches that hurt
	gloamTouch    = 0.9  // how near along the ground it is to touch the player
	gloamDark     = 0.32 // the most light it gathers in, as Sky.Brightness gives it: a night under the sky
	gloamLit      = 0.42 // the light it fades in: a torch three blocks away, or the day
	gloamFadeTime = 1.5  // how long it takes to fade away in the light, or to gather in the dark
	gloamDieTime  = 0.5  // how long it takes to fade away once it has no health left
	gloamBold     = 0.3  // the most it can have faded and still hurt
	gloamSearch   = 8    // blocks above and below the player's feet that a place is looked for in
)

// gloam is one gloam.
type gloam struct {
	pos      geometry.Vector3 // the point on the ground under it
	yaw      float64
	velY     float64
	onGround bool
	push     geometry.Vector3 // the way a hit sends it, and how fast
	age      float64          // how long it has been about
	fade     float64          // from 0, all there, to 1, gone: it gathers from 1
	health   float64
	hurt     float64 // how long it still shows a hit
	bite     float64 // until it can hurt again
	dying    bool    // it has no health left, and fades
}

// stepGloams runs the gloams for dt seconds, gathers new ones in the dark now and then,
// and lets go of those that have faded or been left far behind. It is called while
// survival is played, after the player has moved.
func (g *Game) stepGloams(dt float64) {
	if !g.survival {
		g.gloams = nil
		return
	}
	g.sky.Set(timeOfDay(g.days)) // the light it fades by is the light of this hour
	player := g.player.Pos
	for i := range g.gloams {
		m := &g.gloams[i]
		m.age += dt
		m.hurt = max(m.hurt-dt, 0)
		m.bite = max(m.bite-dt, 0)
		switch lit := g.gloamLit(m); {
		case m.dying:
			m.fade = min(m.fade+dt/gloamDieTime, 1)
		case lit:
			m.fade = min(m.fade+dt/gloamFadeTime, 1)
		default:
			m.fade = max(m.fade-dt/gloamFadeTime, 0) // back in the dark, it gathers again
		}
		m.step(g.world, player, dt)
		if !m.dying && m.fade <= gloamBold && m.bite == 0 && m.touches(player) {
			m.bite = gloamBiteWait
			g.hurt(gloamBite)
		}
	}
	g.gloams = slices.DeleteFunc(g.gloams, func(m gloam) bool {
		gone := m.fade >= 1 && (m.dying || g.gloamLit(&m))
		return gone || m.pos.Distance(player) > gloamLost
	})

	if g.gloamWait -= dt; g.gloamWait <= 0 {
		g.gloamWait = gloamLook
		if len(g.gloams) < gloamMost {
			g.gatherGloam()
		}
	}
}

// gloamLit reports whether a gloam is in light enough to fade it.
func (g *Game) gloamLit(m *gloam) bool {
	return g.sky.Brightness(g.lightAt(m.pos.Add(geometry.Vector3{Y: 1}))) > gloamLit
}

// gatherGloam tries one place for a new gloam, between gloamNear and gloamFar from the
// player along the ground and not far above or below it: a floor in the dark with room
// over it. It reports whether the place would do.
func (g *Game) gatherGloam() bool {
	if g.gloamRng == nil {
		g.gloamRng = rand.New(rand.NewPCG(uint64(g.world.Seed), 0x676c6f61))
	}
	r, p := g.gloamRng, g.player.Pos
	angle, away := r.Float64()*2*math.Pi, gloamNear+r.Float64()*(gloamFar-gloamNear)
	x := int(math.Floor(p.X + math.Cos(angle)*away))
	z := int(math.Floor(p.Z + math.Sin(angle)*away))
	if !g.world.Loaded(x, z) {
		return false
	}
	feet := int(math.Floor(p.Y))
	for y := feet + gloamSearch; y >= feet-gloamSearch; y-- {
		if !solid(g.world, x, y-1, z) || !g.roomFor(x, y, z) {
			continue
		}
		if g.sky.Brightness(g.world.LightAt(x, y+1, z)) > gloamDark {
			return false // the floor nearest the player's height there is lit
		}
		g.gloams = append(g.gloams, gloam{
			pos:    geometry.Vector3{X: float64(x) + 0.5, Y: float64(y), Z: float64(z) + 0.5},
			yaw:    r.Float64()*2*math.Pi - math.Pi,
			fade:   1,
			health: gloamHealth,
		})
		return true
	}
	return false
}

// roomFor reports whether a body as tall as the player fits in a cell and the one over
// it: air or a plant, and not water.
func (g *Game) roomFor(x, y, z int) bool {
	for _, b := range []world.Block{g.world.At(x, y, z), g.world.At(x, y+1, z)} {
		if b == world.Water || b.Info().Solid {
			return false
		}
	}
	return true
}

// step moves a gloam for dt seconds: after the player along the ground, up a block where
// one is in the way, and down by its own weight. One that fades stands where it is.
func (m *gloam) step(w *world.World, player geometry.Vector3, dt float64) {
	to := player.Sub(m.pos)
	to.Y = 0
	var move geometry.Vector3
	if d := to.Length(); d > gloamTouch*0.8 && m.fade == 0 && !m.dying {
		way := to.Scale(1 / d)
		m.yaw = yawOf(way)
		move = way.Scale(gloamSpeed * dt)
	}
	if inWater(w, m.pos) {
		m.velY = 1.5
	} else {
		m.velY = max(m.velY-gravity*dt, -fallSpeed)
	}
	move = move.Add(m.push.Scale(dt))
	m.push = m.push.Scale(max(1-knockDrag*dt, 0))
	move.Y = m.velY * dt

	stopped := travel(w, &m.pos, render.GloamWidth, render.GloamHeight, move)
	m.onGround = stopped[1] && move.Y < 0
	if stopped[1] {
		m.velY = 0
	}
	if (stopped[0] || stopped[2]) && m.onGround {
		m.velY = creatureHop // a block in the way: it climbs
	}
}

// touches reports whether a gloam is near enough to the player standing at p to hurt it.
func (m *gloam) touches(p geometry.Vector3) bool {
	along := geometry.Vector3{X: p.X - m.pos.X, Z: p.Z - m.pos.Z}
	return along.Length() < gloamTouch && p.Y < m.pos.Y+render.GloamHeight && p.Y+playerHeight > m.pos.Y
}

// hit is a gloam being hit by the player standing at from, by points of its health. It
// is knocked away, and at no health it fades.
func (m *gloam) hit(from geometry.Vector3, points float64) {
	m.push = awayFrom(m.pos, from, m.yaw).Scale(knockSpeed)
	m.hurt = hurtShow
	if m.health -= points; m.health <= 0 {
		m.dying = true
	}
}

// pose returns how a gloam is drawn.
func (m *gloam) pose(player geometry.Vector3) render.Gloam {
	reach := 1 - (m.pos.Distance(player)-1)/4 // its arms come up as it nears the player
	return render.Gloam{
		Pos: m.pos, Yaw: m.yaw, Age: m.age, Fade: m.fade, Hurt: m.hurt > 0,
		Reach: min(max(reach, 0), 1),
	}
}

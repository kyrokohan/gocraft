package game

import (
	"math"

	"glonk.dev/geometry"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// What the game sounds like: steps on what the player walks on, digging, breaking and
// placing by what a block is made of, the player's body, the creatures, hearths, water
// dripping in caves, and the wind outside or the air of a cave all the while.

// The pace of the sounds that come again and again.
const (
	stepLength   = 1.6  // blocks walked from one step to the next
	crouchedStep = 0.4  // how loud a crouched player's step is, of a walking one's
	digStroke    = 0.25 // seconds from one stroke of digging to the next
	crackleTime  = 1.0  // seconds of a hearth's crackle, which is played again as it ends
	crackleReach = 24.0 // how far off a tree that burns is still heard, in blocks
	callEvery    = 14.0 // seconds a mossback is quiet for, as a rule
	dripEvery    = 4.0  // and the water in a cave
	splashSpeed  = 2.0  // blocks a second down, the least to splash into water at
	landSpeed    = 6.0  // and to be heard landing on the ground at
)

// Mixer returns the game's mixer, for a speaker to play.
func (g *Game) Mixer() *sound.Mixer { return g.ears }

// NoSound tells the player once, on the title, that there is no sound, and why.
func (g *Game) NoSound(err error) {
	g.note = "no sound: " + reason(err)
}

// materialOf is what a block sounds like.
func materialOf(b world.Block) sound.Material {
	switch b {
	case world.Turf, world.Dirt, world.Clay:
		return sound.Earth
	case world.Sand, world.Gravel, world.Ash:
		return sound.Sand
	case world.Log, world.PineLog, world.PaleLog, world.Planks, world.StumpPalm,
		world.FernTrunk, world.RedwoodLog, world.CharredLog, world.PitCover:
		return sound.Wood
	case world.Leaves, world.PineNeedles, world.PaleLeaves, world.Fronds, world.FanLeaves:
		return sound.Plant
	case world.Glass, world.Ice:
		return sound.Glass
	case world.Snow:
		return sound.Snow
	}
	if b.Info().Stands() {
		return sound.Plant // the plants, the torch and the bedroll
	}
	return sound.Stone
}

// cellMid is the middle of a block's cell, where its sounds are made.
func cellMid(x, y, z int) geometry.Vector3 {
	return geometry.Vector3{X: float64(x) + 0.5, Y: float64(y) + 0.5, Z: float64(z) + 0.5}
}

// listen puts the listener in the player's head, facing as the player does, and sets the
// loops as ambience has them.
func (g *Game) listen() {
	cam := g.player.Camera()
	g.ears.Listen(cam.Position, cam.Right())
	g.ears.Ambient(g.ambience())
}

// ambience returns how loud the two loops are to be: the wind as loud as the sky's light
// at the player's head, and the air of a cave where it is dark. Nothing loops in a pause,
// and the wind blows quietly over the title.
func (g *Game) ambience() (wind, cave float64) {
	s := g.screen
	if s == settingsScreen {
		s = g.back
	}
	switch s {
	case paused, fainted:
		return 0, 0
	case playing, inBag:
		open := float64(g.lightAt(g.player.Camera().Position).Sky()) / world.MaxLight
		return 0.4 * open, 0.6 * (1 - open)
	}
	return 0.3, 0
}

// stepSounds plays the player's steps on the ground, a landing, and a splash where it
// falls into water. It is given where the player was before it moved this frame, whether
// it was in water, and how fast it was falling.
func (g *Game) stepSounds(from geometry.Vector3, wasWet bool, fell float64) {
	p := g.player
	wet := inWater(g.world, p.Pos)
	if wet && !wasWet && fell > splashSpeed {
		g.ears.PlayNear(sound.Splash, 0.8)
	}
	under := g.world.At(int(math.Floor(p.Pos.X)), int(math.Floor(p.Pos.Y-0.05)), int(math.Floor(p.Pos.Z)))
	if !p.OnGround || p.Flying || wet || !under.Info().Solid {
		return
	}
	if fell > landSpeed {
		g.ears.PlayNear(sound.Step(materialOf(under)), 1.3)
	}
	if g.walked += math.Hypot(p.Pos.X-from.X, p.Pos.Z-from.Z); g.walked >= stepLength {
		g.walked = 0
		gain := 1.0
		if p.Crouched {
			gain = crouchedStep
		}
		g.ears.PlayNear(sound.Step(materialOf(under)), gain)
	}
}

// digSound plays a stroke of digging the block aimed at, as digging begins and then
// every digStroke seconds: it is given how long the block had been dug, and how long now.
func (g *Game) digSound(before, now float64) {
	if before == 0 || int(before/digStroke) != int(now/digStroke) {
		t := g.target
		g.ears.Play(sound.Dig(materialOf(t.Block)), cellMid(t.X, t.Y, t.Z), 0.7)
		g.swing = swingTime // a stroke of the hand
	}
}

// worldSounds plays what makes sounds by itself, dt seconds on: the creatures, now and
// then, the hearths that are burning, and water dripping in a cave.
func (g *Game) worldSounds(dt float64) {
	for i := range g.fauna.all {
		c := &g.fauna.all[i]
		if b := &breeds[c.kind]; !c.dead && g.noises.Float64() < dt/b.callEvery {
			g.voice(c, b.call, 0.8)
		}
	}
	if g.crackle -= dt; g.crackle <= 0 {
		g.crackle = crackleTime
		for c, k := range g.hearths {
			if k.Burn > 0 {
				g.ears.Play(sound.Crackle, cellMid(c.X, c.Y, c.Z), 0.8)
			}
		}
		for _, c := range g.world.Struck(g.player.Pos.X, g.player.Pos.Z, crackleReach) {
			if g.world.At(c.X, c.Y, c.Z) == world.Fire {
				g.ears.Play(sound.Crackle, cellMid(c.X, c.Y, c.Z), 0.8)
			}
		}
	}
	eye := g.player.Camera().Position
	if g.lightAt(eye).Sky() == 0 && g.noises.Float64() < dt/dripEvery {
		at := eye.Add(geometry.Vector3{X: 12*g.noises.Float64() - 6, Y: 3, Z: 12*g.noises.Float64() - 6})
		g.ears.Play(sound.Drip, at, 0.7)
	}
}

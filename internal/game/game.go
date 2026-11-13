// Package game is the gocraft game itself: what engine.Run drives.
package game

import (
	"math"
	"time"

	"glonk.dev/engine"
	"glonk.dev/widget"

	"github.com/kyrokohan/gocraft/internal/assets"
	"github.com/kyrokohan/gocraft/internal/render"
	"github.com/kyrokohan/gocraft/internal/world"
)

const (
	fieldOfView = math.Pi / 3 // from the top of the view to the bottom
	eyeHeight   = 1.6         // above the player's feet
)

// Options are what a new game is told.
type Options struct {
	// Seed is the number a new world is made from. 0 takes one from the clock, so that
	// every new world is a different one.
	Seed int64
}

// screen is what the game is showing.
type screen uint8

const (
	onTitle screen = iota // the name of the game and a menu, over a view of the world
	playing
	paused // the world standing still under a menu
)

// The lines of the menus. What a line does is decided by its text, so the order of a
// menu can change without anything else changing.
const (
	itemNew    = "New world"
	itemResume = "Resume"
	itemQuit   = "Quit"
)

// Game is the whole game. It implements engine.Game.
type Game struct {
	opts Options
	set  *assets.Set
	rend *render.Renderer

	screen screen
	menu   widget.Menu
	spin   float64 // how far the view behind the title has turned
	fps    float64

	world  *world.World
	player Player

	selected int       // the slot of the hotbar in use
	target   world.Hit // the block the player aims at, if aimed is set
	aimed    bool
}

// New returns a game showing its title, ready for engine.Run.
func New(opts Options) (*Game, error) {
	set, err := assets.Load()
	if err != nil {
		return nil, err
	}
	g := &Game{opts: opts, set: set, rend: render.New(set), fps: 30}
	g.newWorld()
	g.show(onTitle)
	return g, nil
}

// newWorld makes a world and stands the player in it.
func (g *Game) newWorld() {
	seed := g.opts.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	g.world = world.Generate(seed)
	g.player = Player{Pos: g.world.Spawn()}
	g.selected, g.aimed = 0, false
}

// show changes what the game is showing, and gives the menu the lines of that screen.
func (g *Game) show(s screen) {
	g.screen = s
	g.menu = widget.Menu{Style: hudText}
	switch s {
	case onTitle:
		g.menu.Items = []string{itemNew, itemQuit}
	case paused:
		g.menu.Items = []string{itemResume, itemQuit}
	}
}

// Play leaves the title and starts the game in the world that is behind it.
func (g *Game) Play() {
	g.show(playing)
}

// Update advances the game by dt seconds.
func (g *Game) Update(dt float64, in *engine.Input) error {
	if dt > 0 {
		g.fps = 0.9*g.fps + 0.1/dt
	}

	if g.screen == playing {
		if in.Pressed(engine.KeyEscape) {
			g.show(paused)
			return nil
		}
		g.player.Step(g.world, readControls(in), dt)
		g.build(in)
		return nil
	}

	// A menu: the title's, or the pause menu's.
	if g.screen == onTitle {
		g.spin += 0.12 * dt
	}
	if in.Pressed(engine.KeyEscape) {
		if g.screen == onTitle {
			return engine.ErrQuit
		}
		g.show(playing)
		return nil
	}
	choice, chosen := g.menu.Update(in)
	if !chosen {
		return nil
	}
	switch g.menu.Items[choice] {
	case itemNew, itemResume:
		g.show(playing)
	case itemQuit:
		return engine.ErrQuit
	}
	return nil
}

// Draw renders the game into fb.
func (g *Game) Draw(fb *engine.FrameBuffer) {
	c := fb.Canvas()
	if g.screen == onTitle {
		// The world from a little above where the player will stand, turning slowly.
		cam := g.player.Camera()
		cam.Position.Y += 4
		cam.Yaw, cam.Pitch = g.spin, 0.15
		g.rend.Draw(c, g.world, cam, nil)
		g.drawTitle(fb, c)
		return
	}

	var aim *world.Hit
	if g.aimed {
		aim = &g.target
	}
	g.rend.Draw(c, g.world, g.player.Camera(), aim)
	g.drawHotbarIcons(fb, c)
	c.Compose()
	g.drawHUD(fb)
	if g.screen == paused {
		g.drawPause(fb)
	}
}

// Package game is the gocraft game itself: what engine.Run drives.
package game

import (
	"errors"
	"io/fs"
	"math"
	"time"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"
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

	// SavePath is the file the world is kept in between runs. It is read when the game
	// is made, and written by Save. Empty means the world is not kept.
	SavePath string
}

// screen is what the game is showing.
type screen uint8

const (
	onTitle screen = iota // the name of the game and a menu, over a view of the world
	playing
	paused     // the world standing still under a menu
	confirming // the title, asking whether a new world may take the place of the saved one
)

// The lines of the menus. What a line does is decided by its text, so the order of a
// menu can change without anything else changing.
const (
	itemContinue = "Continue"
	itemNew      = "New world"
	itemQuit     = "Quit"
	itemKeep     = "No, keep it"
	itemReplace  = "Yes, start a new one"
	itemResume   = "Resume"
	itemSave     = "Save and quit"
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

	world   *world.World
	player  Player
	saved   bool // the world is the one in the save file
	started bool // play has begun, so there is something worth saving

	selected int       // the slot of the hotbar in use
	target   world.Hit // the block the player aims at, if aimed is set
	aimed    bool
	box      mesh.Mesh // the outline drawn round that block: the edges of a cube its size
}

// New returns a game showing its title, ready for engine.Run.
func New(opts Options) (*Game, error) {
	set, err := assets.Load()
	if err != nil {
		return nil, err
	}
	g := &Game{opts: opts, set: set, rend: render.New(set), fps: 30, box: mesh.Cube(1)}
	if err := g.load(); err != nil {
		return nil, err
	}
	g.show(onTitle)
	return g, nil
}

// load takes the world from the save file if there is one, and makes a new world if
// there is not. A save file that is there and cannot be used is an error: carrying on
// would write a new world over it when the game ends.
func (g *Game) load() error {
	if g.opts.SavePath != "" {
		s, err := world.Read(g.opts.SavePath)
		switch {
		case err == nil:
			g.world = s.World
			g.player = Player{Pos: s.Pos, Yaw: s.Yaw, Pitch: s.Pitch, Flying: s.Flying}
			g.selected = min(max(s.Slot, 0), len(hotbar)-1)
			g.saved = true
			return nil
		case !errors.Is(err, fs.ErrNotExist):
			return err
		}
	}
	g.newWorld()
	return nil
}

// Save writes the world and the player to the save file. It does nothing if the game
// has no save file, or if play never began: a world nobody has stood in is not worth
// keeping, and must not take the place of one that is.
func (g *Game) Save() error {
	if g.opts.SavePath == "" || !g.started {
		return nil
	}
	return world.Write(g.opts.SavePath, world.Save{
		World:  g.world,
		Pos:    g.player.Pos,
		Yaw:    g.player.Yaw,
		Pitch:  g.player.Pitch,
		Flying: g.player.Flying,
		Slot:   g.selected,
	})
}

// newWorld makes a world and stands the player in it.
func (g *Game) newWorld() {
	seed := g.opts.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	g.world = world.Generate(seed)
	g.player = Player{Pos: g.world.Spawn()}
	g.selected, g.aimed, g.saved = 0, false, false
}

// show changes what the game is showing, and gives the menu the lines of that screen.
func (g *Game) show(s screen) {
	g.screen = s
	g.menu = widget.Menu{Style: hudText}
	switch s {
	case onTitle:
		g.menu.Items = []string{itemNew, itemQuit}
		if g.saved {
			g.menu.Items = []string{itemContinue, itemNew, itemQuit}
		}
	case confirming:
		g.menu.Items = []string{itemKeep, itemReplace}
	case paused:
		g.menu.Items = []string{itemResume, itemSave}
	case playing:
		g.started = true
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

	// Escape is a switch on every screen, so it is read with Started: held, it must
	// not open the pause menu and close it again, or leave the question and then quit.
	escape := in.Started(engine.KeyEscape)

	if g.screen == playing {
		if escape {
			g.show(paused)
			return nil
		}
		g.player.Step(g.world, readControls(in), dt)
		g.build(in)
		return nil
	}

	// A menu: the title's, its question, or the pause menu's.
	if g.screen != paused {
		g.spin += 0.12 * dt
	}
	if escape {
		switch g.screen {
		case onTitle:
			return engine.ErrQuit
		case confirming:
			g.show(onTitle)
		case paused:
			g.show(playing)
		}
		return nil
	}
	choice, chosen := g.menu.Update(in)
	if !chosen {
		return nil
	}
	switch g.menu.Items[choice] {
	case itemContinue, itemResume:
		g.show(playing)
	case itemNew:
		if g.saved {
			g.show(confirming) // a new world would take the place of the saved one
		} else {
			g.show(playing)
		}
	case itemKeep:
		g.show(onTitle)
	case itemReplace:
		g.newWorld()
		g.show(playing)
	case itemQuit, itemSave:
		// The world is saved by whoever runs the game, once Run has returned: that
		// covers Ctrl-C and a closing window as well as this menu.
		return engine.ErrQuit
	}
	return nil
}

// Draw renders the game into fb.
func (g *Game) Draw(fb *engine.FrameBuffer) {
	c := fb.Canvas()
	if g.screen == onTitle || g.screen == confirming {
		// The world from a little above where the player will stand, turning slowly.
		cam := g.player.Camera()
		cam.Position.Y += 4
		cam.Yaw, cam.Pitch = g.spin, 0.15
		g.rend.Draw(c, g.world, cam)
		g.drawTitle(fb, c)
		return
	}

	cam := g.player.Camera()
	g.rend.Draw(c, g.world, cam)
	if g.aimed {
		// The outline lies on the faces of the block, which the rays have drawn with
		// their depth, so glonk draws it whole there and hides it behind what is nearer.
		g.box.Transform.Position = geometry.Vector3{
			X: float64(g.target.X) + 0.5, Y: float64(g.target.Y) + 0.5, Z: float64(g.target.Z) + 0.5,
		}
		g.box.DrawOnSurfacePixels(c, cam, hudPick.Foreground)
	}
	g.drawHotbarIcons(fb, c)
	c.Compose()
	g.drawHUD(fb)
	if g.screen == paused {
		g.drawPause(fb)
	}
}

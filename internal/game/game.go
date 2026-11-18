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

	// chunksAFrame is how many chunks of the world are brought in each frame as the
	// player moves. Flying flat out crosses a chunk in 48 frames and needs 13 new ones.
	chunksAFrame = 2
)

// Options are what a new game is told.
type Options struct {
	// Seed is the number a new world is made from. 0 takes one from the clock, so that
	// every new world is a different one.
	Seed int64

	// SavePath is the folder the world is kept in between runs. It is read when the
	// game is made, and written by Save. Empty means the world is not kept.
	SavePath string

	// OldSave is the file an earlier version of the game kept its one world in, if it
	// is still there. This version cannot play it and leaves it alone, and says so.
	OldSave string

	// world is a world to play in as it is, in place of one made from the seed. Tests
	// set it: making a world takes time, and most tests want a bare floor.
	world *world.World
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
	spin   float64      // how far the view behind the title has turned
	stats  engine.Stats // how the last frame went, as glonk reports it
	debug  bool         // whether the frame numbers are shown: F3
	exact  bool         // whether the terminal reports key releases, so that two keys can be held

	world   *world.World
	clouds  *render.Clouds
	herd    *herd
	moss    *render.Mossbacks
	poses   []render.Mossback // the creatures as they are drawn, kept between frames
	player  Player
	saved   bool // the world is the one in the save file
	started bool // play has begun, so there is something worth saving

	selected int       // the slot of the hotbar in use
	target   world.Hit // the block the player aims at, if aimed is set
	aimed    bool

	cols, rows     int        // the size of the frame in cells, once glonk has said
	pointing       bool       // the pointer aims, and not the middle of the view
	mouseX, mouseY int        // the cell the pointer was last in
	clickWait      [3]float64 // for each mouse button, the time until it acts again while held
	pressX, pressY int        // where the left button went down
	dragging       bool       // the left button is dragging the view round
	dug            bool       // the left button, held still, has broken a block since it went down
	box            mesh.Mesh  // the outline drawn round that block: the edges of a cube its size
}

// New returns a game showing its title, ready for engine.Run.
func New(opts Options) (*Game, error) {
	set, err := assets.Load()
	if err != nil {
		return nil, err
	}
	g := &Game{
		opts: opts, set: set, rend: render.New(set), moss: render.NewMossbacks(set),
		exact: true, box: mesh.Cube(1),
	}
	if err := g.load(); err != nil {
		return nil, err
	}
	g.show(onTitle)
	return g, nil
}

// load takes the world from its folder if one is saved there, and makes a new world if
// not. A saved world that is there and cannot be used is an error: carrying on would
// write a new world over it when the game ends.
func (g *Game) load() error {
	if g.opts.SavePath != "" {
		w, s, err := world.Open(g.opts.SavePath)
		switch {
		case err == nil:
			g.world = w
			g.clouds = render.NewClouds(g.world.Seed)
			g.player = Player{Pos: s.Pos, Yaw: s.Yaw, Pitch: s.Pitch, Flying: s.Flying}
			g.herd = newHerd(g.world, g.player.Pos) // they are not in the save yet
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

// Save writes the world and the player to the world's folder. It does nothing if the
// game has no folder for it, or if play never began: a world nobody has stood in is not
// worth keeping, and must not take the place of one that is.
func (g *Game) Save() error {
	if g.opts.SavePath == "" || !g.started {
		return nil
	}
	return g.world.Save(g.opts.SavePath, world.Meta{
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
	if g.world = g.opts.world; g.world == nil {
		g.world = world.Generate(seed)
	}
	g.clouds = render.NewClouds(seed)
	g.player = Player{Pos: g.world.Spawn()}
	g.herd = newHerd(g.world, g.player.Pos)
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

// Stats is called by glonk after every frame with how the frame went. It makes the game
// an engine.StatsReceiver.
func (g *Game) Stats(s engine.Stats) {
	g.stats = s
}

// Update advances the game by dt seconds.
func (g *Game) Update(dt float64, in *engine.Input) error {
	g.exact = in.Exact()
	if in.Started(engine.KeyF3) {
		g.debug = !g.debug
	}

	// Escape is a switch on every screen, so it is read with Started: held, it must
	// not open the pause menu and close it again, or leave the question and then quit.
	escape := in.Started(engine.KeyEscape)

	if g.screen != paused {
		g.clouds.Advance(dt) // the sky moves behind the title too, and stands still in a pause

		// The world is held round the player, and comes in a few chunks a frame as the
		// player moves: making a chunk takes time that a frame has little of.
		g.world.Follow(g.player.Pos.X, g.player.Pos.Z)
		g.world.Load(chunksAFrame)
	}

	if g.screen == playing {
		if escape {
			g.show(paused)
			return nil
		}
		g.player.Step(g.world, readControls(in), dt)
		g.herd.step(g.world, g.player.Pos, dt)
		g.build(in, dt)
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

// drawThings draws what glonk draws into the world, over the blocks: the creatures,
// which are solid, and then the clouds, which are seen through and so come last.
func (g *Game) drawThings(c *engine.Canvas, cam mesh.Camera) {
	g.poses = g.poses[:0]
	for i := range g.herd.all {
		g.poses = append(g.poses, g.herd.all[i].Pose())
	}
	g.moss.Draw(c, cam, g.poses)
	g.clouds.Draw(c, cam)
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
		g.drawThings(c, cam)
		g.drawTitle(fb, c)
		return
	}

	// The blocks first, by rays, and then what glonk draws into the same canvas: the
	// depth the rays stored hides it where a block is nearer.
	cam := g.player.Camera()
	g.rend.Draw(c, g.world, cam)
	g.drawThings(c, cam)
	if g.aimed {
		// The outline lies on the faces of the block, which the rays have drawn with
		// their depth, so glonk draws it whole there and hides it behind what is nearer.
		g.box.Transform.Position = geometry.Vector3{
			X: float64(g.target.X) + 0.5, Y: float64(g.target.Y) + 0.5, Z: float64(g.target.Z) + 0.5,
		}
		g.box.DrawOnSurfacePixels(c, cam, hudPick.Foreground)
	}
	if g.screen == paused {
		c.BlendRect(0, 0, c.Width(), c.Height(), engine.RGB(0, 0, 0), 0.5) // the world dims behind the menu
	}
	c.Compose()
	g.drawHUD(fb, c)
	if g.screen == paused {
		g.drawPause(fb)
	}
}

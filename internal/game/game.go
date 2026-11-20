// Package game is the Skyspear game itself: what engine.Run drives.
package game

import (
	"errors"
	"io/fs"
	"math"
	"math/rand/v2"
	"path/filepath"
	"time"

	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"
	"glonk.dev/widget"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

const (
	eyeHeight = 1.6 // above the player's feet

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

	// Worlds is a folder of worlds, each a folder of its own, for the title to list. When
	// it is set, SavePath is not read: it becomes the folder of the world chosen.
	Worlds string

	// Settings is the file the player's settings are kept in. Empty means they are not.
	Settings string

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
	choosing   // the title, asking how the new world is to be played
	fainted    // the player has no health left, and wakes from a menu
	inBag      // what the player carries, over the world standing still

	// The title's screens for a list of worlds.
	worldChosen  // what to do with the world chosen in the list
	deleting     // whether to delete it
	deletingSure // whether to, for good
	naming       // the name of a new world being typed
	seeding      // and its seed

	settingsScreen // the settings, over the title or the paused world
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
	itemSurvival = "Survival"
	itemCreative = "Creative"
	itemWake     = "Wake up"
	itemSettings = "Settings"
	itemBack     = "Back"
	itemPlay     = "Play"
	itemDelete   = "Delete"
	itemDeleteIt = "Yes, delete it"
	itemForGood  = "Yes, delete it for good"
	itemLeave    = "Save and leave"
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

	world  *world.World
	clouds *render.Clouds
	sky    *render.Sky // at the time of day, which days says
	days   float64     // how long the world has been played, in days of the game
	fauna  *fauna
	beasts *render.Beasts
	poses  []render.Beast // the beasts as they are drawn, kept between frames

	// The player's body, which is seen from behind: what draws it, the body as it is
	// drawn, kept between frames, which look it wears, how far the player has walked
	// in all, and whether it walked this frame.
	folk   *render.People
	cast   []render.Person
	worn   int
	strode float64
	paced  bool

	player  Player
	saved   bool // the world is the one in the save file
	started bool // play has begun, so there is something worth saving

	survival bool // the world is played in survival, and not in creative
	items    inventory
	drops    []drop
	look     *render.Drops
	lying    []render.Lying // the drops as they are drawn, kept between frames

	// The bits flying from blocks broken, the chances that throw them, and the bits as
	// they are drawn, kept between frames.
	bits    []bit
	scatter *rand.Rand
	pieces  []render.Bit

	swing float64 // seconds left of a swing of the hand

	// quake is how hard the ground shakes under the player, from 0 to 1, since the step
	// of a giant: it dies away, and shook is how long it has shaken, which is what the
	// view trembles by.
	quake, shook float64

	// The fires that burn near the player, as they were last looked for, and how long
	// until they are looked for again.
	burning  []geometry.Vector3
	fireWait float64

	// The player's body, in survival.
	health, food, air float64
	hunger, mend      float64 // seconds since food last went down, and since the body last mended
	peak              float64 // the highest the player has been since last on the ground
	flash             float64 // seconds the view still shows red, for a hurt

	selected int       // the slot of the hotbar in use
	target   world.Hit // the block the player aims at, if aimed is set
	aimed    bool
	prey     prey // the creature the player aims at, if preyed is set, and then no block is
	preyed   bool
	hitWait  float64 // until the player can hit, or throw, again

	// Where the player aims from and along, as of the last frame, what has been thrown
	// and is in flight or sticks where it landed, and those things as they are drawn,
	// kept between frames.
	from, along geometry.Vector3
	missiles    []missile
	thrown      []render.Flying

	embers []world.Ember // the fires that the rocks of a fire mountain left, which burn out

	digAt   [3]int  // the block being dug, in survival
	digging float64 // for how long it has been dug

	// Fire: the hearth a fire drill is being turned in and for how long it has been, how
	// many seconds the torch in hand has burned, and until the fires near smoke again.
	drillAt  world.Cell
	drilling float64
	torch    float64
	smoking  float64

	// The pots and the hearths of the world, by their cells, the one open in the bag, if
	// any, and the bedroll the player wakes at.
	pots    map[world.Cell]*world.PotStore
	hearths map[world.Cell]*world.HearthStore
	opened  world.Block // a pot or a hearth, or Air while the bag is the player's alone
	at      world.Cell  // where that is
	bed     *world.Cell

	notice     string  // a line over play, of what was just done
	noticeLeft float64 // seconds it stays

	hand    world.Stack // what is held while things are moved in the bag
	cursor  int         // the slot of the bag the keys are on
	inList  bool        // the keys are on the list of what can be made, and not on a slot
	recipe  int         // the line of that list the keys are on
	listTop int         // the first line of it shown, when it is longer than its panel

	settings Settings
	back     screen // the screen the settings go back to

	// The sounds: the mixer they are played into, the chances they are played by, how far
	// the player has walked since its last step, and until the hearths crackle again.
	ears    *sound.Mixer
	noises  *rand.Rand
	walked  float64
	crackle float64

	// The list of worlds on the title, the one chosen, and the new one being made.
	list    []savedWorld
	chosen  int
	field   field
	newName string
	newSeed int64
	note    string // a line on the title of what went wrong, or what was done

	menuX, menuY, menuW, menuRows int // where the menu was last drawn: its first line, and how many show

	cols, rows     int        // the size of the frame in cells, once glonk has said
	pointing       bool       // the pointer aims, and not the middle of the view
	mouseX, mouseY int        // the cell the pointer was last in
	clickWait      [3]float64 // for each mouse button, the time until it acts again while held
	pressX, pressY int        // where the left button went down
	dragging       bool       // the left button is dragging the view round
	dug            bool       // the left button, held still, has broken a block since it went down
	box            mesh.Mesh  // the outline drawn round that block: the edges of a cube its size
	fog            mesh.Fog   // the fog of the view, as the camera of this frame has it
}

// New returns a game showing its title, ready for engine.Run.
func New(opts Options) (*Game, error) {
	set, err := assets.Load()
	if err != nil {
		return nil, err
	}
	g := &Game{
		opts: opts, set: set, rend: render.New(set), beasts: render.NewBeasts(set), look: render.NewDrops(set),
		folk:  render.NewPeople(set),
		exact: true, box: mesh.Cube(1), settings: loadSettings(opts.Settings),
		ears: sound.NewMixer(set.Sounds), noises: rand.New(rand.NewPCG(1, 0x736f756e)),
		scatter: rand.New(rand.NewPCG(2, 0x62697473)),
	}
	g.ears.SetVolume(g.settings.volume())
	if err := g.load(); err != nil {
		return nil, err
	}
	g.sky = render.NewSky(timeOfDay(g.days))
	g.show(onTitle)
	return g, nil
}

// load takes the world from its folder if one is saved there, and makes a new world if
// not. A saved world that is there and cannot be used is an error: carrying on would
// write a new world over it when the game ends.
func (g *Game) load() error {
	if g.opts.Worlds != "" {
		g.opts.SavePath = "" // the world behind the title is a new one, till one is chosen
	}
	if g.opts.SavePath != "" {
		w, s, err := world.Open(g.opts.SavePath)
		switch {
		case err == nil:
			g.use(w, s)
			return nil
		case !errors.Is(err, fs.ErrNotExist):
			return err
		}
	}
	g.newWorld(g.opts.Seed)
	return nil
}

// use plays a saved world, with the player as it was saved.
func (g *Game) use(w *world.World, s world.Meta) {
	g.world = w
	g.clouds = render.NewClouds(g.world.Seed)
	g.player = Player{Pos: s.Pos, Yaw: s.Yaw, Pitch: s.Pitch, Flying: s.Flying}
	g.fauna = faunaFrom(g.world, s, g.player.Pos)
	g.bits, g.missiles, g.embers = nil, nil, s.Embers
	g.selected = min(max(s.Slot, 0), hotbarSlots-1)
	g.days = s.Days
	g.saved = true
	g.loadStores(s) // before the player, which may wake at its bed
	g.loadPlayer(s)
}

// Save writes the world and the player to the world's folder. It does nothing if the
// game has no folder for it, or if play never began: a world nobody has stood in is not
// worth keeping, and must not take the place of one that is.
func (g *Game) Save() error {
	if g.opts.SavePath == "" || !g.started {
		return nil
	}
	mode := world.Creative
	if g.survival {
		mode = world.Survival
	}
	m := world.Meta{
		Pos:    g.player.Pos,
		Yaw:    g.player.Yaw,
		Pitch:  g.player.Pitch,
		Flying: g.player.Flying,
		Slot:   g.selected,
		Days:   g.days,
		Mode:   mode,
		Items:  g.items[:],
		Health: g.health,
		Food:   g.food,
		Air:    g.air,
		Look:   g.worn,
		Torch:  g.torch,
		Embers: g.embers,
	}
	for _, d := range g.drops {
		m.Drops = append(m.Drops, d.Drop)
	}
	for _, t := range g.missiles {
		if t.leaves {
			m.Drops = append(m.Drops, world.Drop{Stack: t.stack, Pos: t.pos}) // in flight or stuck, it is kept as a thing lying there
		}
	}
	m.Beasts = g.fauna.saved()
	m.Pots, m.Hearths = g.savedStores()
	m.Bed = g.bed
	return g.world.Save(g.opts.SavePath, m)
}

// loadPlayer takes what the player carries and how its body is from a save. A save from
// before there were modes is creative, with the hotbar a creative world starts with, and
// a player saved with no health left wakes as from a faint.
func (g *Game) loadPlayer(s world.Meta) {
	g.survival = s.Mode == world.Survival
	g.items = inventory{}
	copy(g.items[:], s.Items)
	if s.Items == nil && !g.survival {
		g.items = creativeStart()
	}
	g.health, g.food, g.air = min(s.Health, maxHealth), min(max(s.Food, 0), maxFood), min(max(s.Air, 0), maxAir)
	g.worn = min(max(s.Look, 0), assets.PersonLooks-1)
	g.torch = s.Torch
	g.peak = g.player.Pos.Y
	g.drops = g.drops[:0]
	for _, d := range s.Drops {
		g.drops = append(g.drops, drop{Drop: d})
	}
	if !g.survival {
		g.renew()
		return
	}
	g.player.Flying = false
	if g.health <= 0 {
		g.wake()
	}
}

// begin makes the world behind the title one played in survival, with nothing carried,
// or in creative, with the hotbar full.
func (g *Game) begin(survival bool) {
	g.survival = survival
	g.items = creativeStart()
	if survival {
		g.items = inventory{}
	}
	g.player.Flying = false
	g.renew()
}

// newWorld makes a world from a seed and stands the player in it. Seed 0 takes one from
// the clock.
func (g *Game) newWorld(seed int64) {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	if g.world = g.opts.world; g.world == nil {
		g.world = world.Generate(seed)
	}
	g.clouds = render.NewClouds(seed)
	g.player = Player{Pos: g.world.Spawn()}
	g.fauna = newFauna(g.world, g.player.Pos)
	g.selected, g.aimed, g.saved, g.days, g.torch = 0, false, false, 0, 0
	g.drops, g.bits, g.missiles, g.embers = nil, nil, nil, nil
	g.loadStores(world.Meta{})
	g.begin(false)
}

// show changes what the game is showing, and gives the menu the lines of that screen.
func (g *Game) show(s screen) {
	g.screen = s
	g.menu = widget.Menu{Style: hudText}
	g.menuRows = 0 // till the menu of this screen is drawn, a click is on no line of it
	switch s {
	case onTitle:
		g.menu.Items = g.titleItems()
	case worldChosen:
		g.menu.Items = []string{itemPlay, itemDelete, itemBack}
	case deleting:
		g.menu.Items = []string{itemKeep, itemDeleteIt}
	case deletingSure:
		g.menu.Items = []string{itemKeep, itemForGood}
	case naming:
		g.field = nameField(g.freeName())
	case settingsScreen:
		g.menu.Items = g.settings.lines()
	case confirming:
		g.menu.Items = []string{itemKeep, itemReplace}
	case choosing:
		g.menu.Items = []string{itemSurvival, itemCreative}
	case paused:
		g.menu.Items = []string{itemResume, itemSettings, itemSave}
		if g.listing() {
			g.menu.Items[2] = itemLeave
		}
	case fainted:
		g.menu.Items = []string{itemWake}
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
	defer g.listen() // the listener moves with the player, after the frame has moved it
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
		if keyDown(in.Started, 'i') {
			g.openBag()
			return nil
		}
		g.days += dt / dayLength
		if keyDown(in.Started, 't') {
			g.days += quarterDay
		}
		if keyDown(in.Started, 'v') {
			g.switchView()
		}
		if keyDown(in.Started, 'l') {
			g.nextLook()
		}
		c := readControls(in)
		c.toggleFly = c.toggleFly && !g.survival // only in creative can the player fly
		from, wasWet, fell := g.player.Pos, inWater(g.world, g.player.Pos), -g.player.velY
		g.player.Step(g.world, c, dt)
		g.pace(from)
		g.stepSounds(from, wasWet, fell)
		if g.stepBeasts(dt); g.screen != playing {
			return nil // a hunter made the player faint
		}
		g.worldSounds(dt)
		g.flash = max(g.flash-dt, 0)
		g.swing = max(g.swing-dt, 0)
		if g.shook += dt; g.quake > 0 {
			if g.quake *= math.Exp(-dt / shakeFade); g.quake < 0.01 {
				g.quake = 0
			}
		}
		if g.survival {
			if g.live(dt); g.screen != playing {
				return nil // the player fainted
			}
		}
		g.stepDrops(dt)
		g.stepMissiles(dt)
		g.geysers(dt)
		g.erupt(dt)
		g.smoke(dt)
		g.stepBits(dt)
		g.stepHearths(dt)
		g.noticeLeft = max(g.noticeLeft-dt, 0)
		g.build(in, dt)
		g.carry(dt)
		return nil
	}
	if g.screen == inBag {
		g.stepHearths(dt) // a hearth that is open is seen to cook
		g.useBag(in, escape)
		return nil
	}

	// A menu: the title's, its questions, the pause menu's, the settings, or the one to
	// wake from. A new world's name and seed are typed.
	if g.titleShown() {
		g.spin += 0.12 * dt
	}
	if escape {
		switch g.screen {
		case onTitle:
			return engine.ErrQuit
		case paused:
			g.show(playing)
		case settingsScreen:
			g.show(g.back)
		case fainted:
		default:
			g.note = ""
			g.show(onTitle)
		}
		return nil
	}
	if g.screen == naming || g.screen == seeding {
		g.typeIn(in)
		return nil
	}
	if g.screen == settingsScreen {
		// Left and Right change the setting the menu is on.
		for key, by := range map[engine.Key]int{engine.KeyLeft: -1, engine.KeyRight: 1} {
			if in.Pressed(key) && g.menu.Selected < settingLines {
				g.changeSetting(g.menu.Selected, by)
			}
		}
	}
	choice, chosen := g.menu.Update(in)
	if c, ok := g.pointMenu(in); ok {
		choice, chosen = c, true
	}
	if !chosen {
		return nil
	}
	g.note = ""
	switch {
	case g.screen == onTitle && g.listing() && choice < len(g.list):
		g.chosen = choice
		g.show(worldChosen)
		return nil
	case g.screen == settingsScreen && choice < settingLines:
		g.changeSetting(choice, 1)
		return nil
	}
	switch g.menu.Items[choice] {
	case itemContinue, itemResume:
		g.show(playing)
	case itemNew:
		switch {
		case g.listing():
			g.show(naming)
		case g.saved:
			g.show(confirming) // a new world would take the place of the saved one
		default:
			g.show(choosing)
		}
	case itemBack:
		if g.screen == settingsScreen {
			g.show(g.back)
		} else {
			g.show(onTitle)
		}
	case itemKeep:
		g.show(onTitle)
	case itemReplace:
		g.show(choosing)
	case itemSurvival, itemCreative:
		switch {
		case g.listing():
			g.newWorld(g.newSeed)
			g.opts.SavePath = filepath.Join(g.opts.Worlds, g.newName)
		case g.saved:
			g.newWorld(g.opts.Seed) // the world behind the title is the saved one
		}
		g.begin(g.menu.Items[choice] == itemSurvival)
		g.show(playing)
	case itemPlay:
		g.playWorld(g.list[g.chosen])
	case itemDelete:
		g.show(deleting)
	case itemDeleteIt:
		g.show(deletingSure)
	case itemForGood:
		g.deleteWorld(g.list[g.chosen])
		g.show(onTitle)
	case itemSettings:
		g.back = g.screen
		g.show(settingsScreen)
	case itemWake:
		g.wake()
		g.show(playing)
	case itemLeave:
		if err := g.Save(); err != nil {
			g.note = "the world was not saved: " + reason(err) // so it stays open
			return nil
		}
		g.show(onTitle)
	case itemQuit, itemSave:
		// The world is saved by whoever runs the game, once Run has returned: that
		// covers Ctrl-C and a closing window as well as this menu.
		return engine.ErrQuit
	}
	return nil
}

// titleShown reports whether the screen is the title's or one of its questions, over the
// world turning slowly behind it.
func (g *Game) titleShown() bool {
	switch g.screen {
	case onTitle, confirming, choosing, worldChosen, deleting, deletingSure, naming, seeding:
		return true
	case settingsScreen:
		return g.back != paused
	}
	return false
}

// changeSetting moves line i of the settings by one value either way, and keeps it.
func (g *Game) changeSetting(i, by int) {
	g.settings.step(i, by)
	g.menu.Items = g.settings.lines()
	g.ears.SetVolume(g.settings.volume())
	if err := g.settings.save(g.opts.Settings); err != nil {
		g.note = "the settings were not saved: " + reason(err)
	}
}

// drawThings draws what glonk draws into the world, over the blocks: the creatures,
// which are solid, and then the clouds, which are seen through and so come last. The
// player's own body is among them when the view is from behind it.
func (g *Game) drawThings(c *engine.Canvas, cam mesh.Camera, body bool) {
	g.poses = g.poses[:0]
	for i := range g.fauna.all {
		pose := g.fauna.all[i].Pose()
		pose.Light = g.sky.Brightness(g.lightAt(pose.Pos.Add(geometry.Vector3{Y: 0.5})))
		pose.Glow = pose.Light < glowBelow
		g.poses = append(g.poses, pose)
	}
	g.beasts.Draw(c, cam, g.poses, g.sky)

	if g.cast = g.cast[:0]; body {
		g.cast = append(g.cast, g.body())
	}
	g.folk.Draw(c, cam, g.cast, g.sky, g.look)

	g.lying = g.lying[:0]
	for _, d := range g.drops {
		if d.Pos.Sub(cam.Position).Length() > render.ViewDistance {
			continue
		}
		g.lying = append(g.lying, render.Lying{
			Item: d.Item, Pos: d.Pos, Size: dropSize, Age: d.Age,
			Light: g.sky.Brightness(g.lightAt(d.Pos.Add(geometry.Vector3{Y: dropSize / 2}))),
		})
	}
	g.look.Draw(c, cam, g.lying, g.sky)
	g.look.DrawFlying(c, cam, g.flying(), g.sky)
	g.look.DrawBits(c, cam, g.drawnBits(), g.sky)

	g.clouds.Draw(c, cam, g.sky)
}

// The smoke of the nearest fire mountain is seen in the sky from plumeFar blocks off, and
// all of it from plumeNear.
const (
	plumeNear = 250.0
	plumeFar  = 1200.0
)

// plume has the sky show the smoke of the nearest fire mountain, in the way to it from
// where the player is: more of it the nearer the mountain.
func (g *Game) plume() {
	p := g.player.Pos
	crater, far, ok := g.world.Mountain(p.X, p.Z)
	if !ok {
		g.sky.SetPlume(geometry.Vector3{}, 0)
		return
	}
	eye := g.player.head()
	to := geometry.Vector3{X: float64(crater.X) + 0.5 - eye.X, Y: float64(crater.Y) - eye.Y, Z: float64(crater.Z) + 0.5 - eye.Z}
	g.sky.SetPlume(to, 1-(far-plumeNear)/(plumeFar-plumeNear))
}

// lightAt returns the light in the cell a point is in.
func (g *Game) lightAt(p geometry.Vector3) world.Light {
	return g.world.LightAt(int(math.Floor(p.X)), int(math.Floor(p.Y)), int(math.Floor(p.Z)))
}

// camera returns the view from the player's eyes, with the fog of the hour: darkened
// where the light of the sky does not reach the eye, so that a cave fades into the dark
// and not into the color of the sky outside it.
func (g *Game) camera() mesh.Camera {
	cam, _ := g.viewed()
	cam.Fog = g.fogAt(cam)
	return cam
}

// eyes returns the view from the player's eyes as the settings have it: as wide, and as
// deep, as they say.
func (g *Game) eyes() mesh.Camera {
	cam := g.player.Camera()
	cam.FOV, cam.Far = g.settings.fov(), float64(g.settings.ViewDistance)
	return cam
}

// fogAt returns the sky's fog for a camera, which ends where the camera's view does.
func (g *Game) fogAt(cam mesh.Camera) *mesh.Fog {
	g.fog = *g.sky.FogIn(g.lightAt(cam.Position))
	g.fog.Start, g.fog.End = 0.6*cam.Far, cam.Far
	return &g.fog
}

// Draw renders the game into fb.
func (g *Game) Draw(fb *engine.FrameBuffer) {
	c := fb.Canvas()
	g.sky.SetAsh(g.ash())
	g.sky.Set(timeOfDay(g.days))
	g.plume()
	g.rend.Crack = render.Crack{}
	if g.titleShown() {
		// The world from a little above where the player will stand, turning slowly.
		cam := g.eyes()
		cam.Position.Y += 4
		cam.Fog = g.fogAt(cam)
		cam.Yaw, cam.Pitch = g.spin, 0.15
		g.rend.Draw(c, g.world, cam, g.sky)
		g.drawThings(c, cam, false)
		g.drawTitle(fb, c)
		return
	}

	// The blocks first, by rays, and then what glonk draws into the same canvas: the
	// depth the rays stored hides it where a block is nearer.
	cam, behind := g.viewed()
	cam.Fog = g.fogAt(cam)
	if g.digging > 0 && g.aimed {
		seconds, _ := g.digTime(g.target.Block)
		g.rend.Crack = render.Crack{X: g.target.X, Y: g.target.Y, Z: g.target.Z, Part: g.digging / seconds}
	}
	g.rend.Draw(c, g.world, cam, g.sky)
	g.drawThings(c, cam, behind)
	if g.aimed && (g.screen == playing || g.screen == paused) {
		// The outline lies on the faces of the block, which the rays have drawn with
		// their depth, so glonk draws it whole there and hides it behind what is nearer.
		g.box.Transform.Position = geometry.Vector3{
			X: float64(g.target.X) + 0.5, Y: float64(g.target.Y) + 0.5, Z: float64(g.target.Z) + 0.5,
		}
		g.box.DrawOnSurfacePixels(c, cam, hudPick.Foreground)
	}
	if g.screen != inBag && !behind {
		// Last, over everything in the world. From behind it is in the body's hand.
		g.look.DrawHand(c, g.inHand())
	}
	if eye := cam.Position; g.lightAt(eye) != 0 && g.world.At(int(math.Floor(eye.X)), int(math.Floor(eye.Y)), int(math.Floor(eye.Z))) == world.Lava {
		c.BlendRect(0, 0, c.Width(), c.Height(), engine.RGB(255, 120, 30), 0.75) // the eye is in lava
	}
	if g.flash > 0 {
		c.BlendRect(0, 0, c.Width(), c.Height(), engine.RGB(200, 20, 20), 0.4*g.flash/hurtFlash)
	}
	switch g.screen {
	case paused, inBag, settingsScreen:
		c.BlendRect(0, 0, c.Width(), c.Height(), engine.RGB(0, 0, 0), 0.5) // the world dims behind the menu
	case fainted:
		c.BlendRect(0, 0, c.Width(), c.Height(), engine.RGB(60, 0, 0), 0.6)
	}
	c.Compose()
	switch g.screen {
	case inBag:
		g.drawBag(fb, c)
		return
	case fainted:
		w := menuWidth
		g.drawMenu(fb, "you fainted", (fb.Width()-w)/2, max(fb.Height()/2-2, 0), w, 0)
		return
	}
	g.drawHUD(fb, c)
	switch g.screen {
	case paused:
		g.drawPause(fb)
	case settingsScreen:
		g.drawSettings(fb)
	}
}

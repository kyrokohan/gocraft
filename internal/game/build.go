package game

import (
	"math"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/world"
)

// hotbar is the kinds of block the player can place, on the keys 1 to 9.
var hotbar = [9]world.Block{
	world.Grass, world.Dirt, world.Stone, world.Sand, world.Log,
	world.Leaves, world.Planks, world.Brick, world.Glass,
}

// reach is how far away the player can break and place blocks.
const reach = 5.0

// A held mouse button acts again after clickDelay, and then every clickRepeat, as a held
// key does by the OS's own repeat.
const (
	clickDelay  = 0.35
	clickRepeat = 0.2
)

// dragSpeed is how far a drag turns the view, as a multiple of the angle the pointer
// moved through: at 1 the world would stay under the pointer, and a little more saves
// the hand some travel.
const dragSpeed = 1.5

// Resize is called by glonk with the size of the frame in cells, which says where on
// the screen the pointer is. It makes the game an engine.Resizer.
func (g *Game) Resize(width, height int) {
	g.cols, g.rows = width, height
}

// point follows the mouse. The pointer aims from the moment it moves or a button goes
// down, and the middle of the view aims again from the moment an arrow key turns the
// view: whichever was used last. It reports the hotbar slot under the pointer, or -1.
func (g *Game) point(in *engine.Input) (slot int) {
	x, y, ok := in.MousePos()
	clicked := in.MouseStarted(engine.MouseLeft) || in.MouseStarted(engine.MouseRight)
	if ok && (x != g.mouseX || y != g.mouseY || clicked) {
		g.pointing = true
	}
	if ok {
		g.mouseX, g.mouseY = x, y
	}
	for _, k := range []engine.Key{engine.KeyLeft, engine.KeyRight, engine.KeyUp, engine.KeyDown} {
		if in.IsDown(k) {
			g.pointing = false
		}
	}

	if !g.pointing || g.cols <= 0 || g.rows <= 0 {
		return -1 // the middle aims, or the size of the frame is not known yet
	}
	x0, y0 := barOrigin(g.cols, g.rows)
	if g.mouseY < y0 || g.mouseX < x0 || g.mouseX >= x0+barWidth {
		return -1
	}
	return min((g.mouseX-x0)/slotPitch, len(hotbar)-1)
}

// aim finds the block the player aims at, if one is in reach: the one under the
// pointer while the mouse is in use, and the one in the middle of the view otherwise.
func (g *Game) aim() {
	cam := g.player.Camera()
	direction := cam.Look()
	if g.pointing {
		// glonk's ray through the pointer's cell reaches one deep, and the reach is a
		// distance, so the ray is made one long.
		if _, d, ok := cam.Ray(float64(g.mouseX)+0.5, float64(g.mouseY)+0.5, g.cols, g.rows); ok {
			direction = d.Normalize()
		}
	}
	g.target, g.aimed = g.world.Target(cam.Position, direction, reach)
}

// drag reads the left mouse button, which does two things, and reports whether it
// breaks a block in this frame.
//
// Pressed and let go where it is, it is a click, and breaks. Held still, it digs on, as
// a held key does. Pressed and moved, it drags the view round and breaks nothing: so a
// press cannot break at once, since it may be the start of a drag. The pointer moved
// from (fromX, fromY) in this frame.
func (g *Game) drag(in *engine.Input, dt float64, fromX, fromY int) (breaks bool) {
	const b = engine.MouseLeft
	if in.MouseStarted(b) {
		g.pressX, g.pressY = g.mouseX, g.mouseY
		g.dragging, g.dug = false, false
		g.clickWait[b] = clickDelay
	}

	if in.MouseDown(b) {
		// A hand is not perfectly still in a click, so a drag begins two pixels from
		// where the button went down: two cells across, or one up or down. A button
		// that has begun to dig goes on digging wherever the pointer goes.
		dx, dy := g.mouseX-g.pressX, g.mouseY-g.pressY
		if !g.dragging && !g.dug && dx*dx+4*dy*dy >= 4 {
			g.dragging = true
			fromX, fromY = g.pressX, g.pressY // the turn takes in the move that began it
		}
		switch {
		case g.dragging:
			g.turn(g.mouseX-fromX, g.mouseY-fromY)
		default:
			if g.clickWait[b] -= dt; g.clickWait[b] <= 0 {
				g.clickWait[b] = clickRepeat
				g.dug, breaks = true, true
			}
		}
	}

	if in.MouseReleased(b) {
		// Let go with no drag and nothing dug: a click.
		breaks = breaks || (!g.dragging && !g.dug)
		g.dragging = false
	}
	return breaks
}

// turn turns the view by a move of the pointer, in cells: right looks right and down
// looks down. A cell turns the view by the angle it covers at the middle of the view,
// times dragSpeed; it is half as wide as it is tall, so a cell across turns half as far
// as a cell up. Until the size of the frame is known, that angle is not, and nothing turns.
func (g *Game) turn(dx, dy int) {
	if g.rows <= 0 {
		return
	}
	cell := dragSpeed * 2 * math.Tan(fieldOfView/2) / float64(g.rows)
	g.player.Turn(float64(dx)*cell/2, -float64(dy)*cell)
}

// held reports whether a mouse button acts in this frame: when it goes down, and then
// again and again while it stays down.
func (g *Game) held(in *engine.Input, b engine.MouseButton, dt float64) bool {
	wait := &g.clickWait[b]
	switch {
	case in.MouseStarted(b):
		*wait = clickDelay
		return true
	case !in.MouseDown(b):
		return false
	}
	if *wait -= dt; *wait > 0 {
		return false
	}
	*wait = clickRepeat
	return true
}

// build handles the keys and the mouse that pick, break and place blocks. A held key
// repeats, so holding Q digs on and holding E builds on, and so does a held button.
func (g *Game) build(in *engine.Input, dt float64) {
	for i := range hotbar {
		if in.Pressed(engine.Key('1' + rune(i))) {
			g.selected = i
		}
	}
	if in.Pressed(engine.KeyTab) {
		g.selected = (g.selected + 1) % len(hotbar)
	}
	if _, steps := in.Wheel(); steps != 0 {
		n := len(hotbar)
		g.selected = ((g.selected+steps)%n + n) % n
	}

	fromX, fromY := g.mouseX, g.mouseY
	slot := g.point(in)
	breaks := g.drag(in, dt, fromX, fromY)
	places := g.held(in, engine.MouseRight, dt)
	if slot >= 0 {
		// The pointer is on the hotbar: a click picks the slot, and no block behind the
		// bar is aimed at.
		if breaks || places {
			g.selected = slot
		}
		g.aimed = false
		return
	}
	breaks = breaks || keyDown(in.Pressed, 'q')
	places = places || keyDown(in.Pressed, 'e')

	g.aim()
	if !g.aimed {
		return
	}
	t := g.target
	switch {
	case breaks:
		if t.Block.Info().Fixed {
			return
		}
		g.world.Set(t.X, t.Y, t.Z, world.Air)
		if g.world.At(t.X, t.Y+1, t.Z).Info().Plant {
			g.world.Set(t.X, t.Y+1, t.Z, world.Air) // a plant goes with the ground it stood on
		}
	case places:
		// A block goes against the face aimed at, or in the place of a plant aimed at.
		x, y, z := t.X, t.Y, t.Z
		if !t.Block.Info().Plant {
			nx, ny, nz := t.Face.Normal()
			x, y, z = x+nx, y+ny, z+nz
		}
		if b := g.world.At(x, y, z); (b != world.Air && b != world.Water && !b.Info().Plant) || g.player.overlaps(x, y, z) {
			return
		}
		g.world.Set(x, y, z, hotbar[g.selected])
	default:
		return
	}
	g.aim() // the world changed, so what is aimed at may have too
}

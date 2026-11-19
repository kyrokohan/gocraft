package game

import (
	"math"

	"glonk.dev/engine"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// reach is how far away the player can break and place blocks.
const reach = 5.0

// minDig is the least time digging takes in survival, even for a plant: long enough
// that a press of the button that turns out to be the start of a drag breaks nothing.
const minDig = 0.1

// A held mouse button acts again after clickDelay, and then every clickRepeat, as a held
// key does by the OS's own repeat.
const (
	clickDelay  = 0.35
	clickRepeat = 0.2
)

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
	return min((g.mouseX-x0)/slotPitch, hotbarSlots-1)
}

// aim finds the block or the creature the player aims at, if one is in reach: under the
// pointer while the mouse is in use, and in the middle of the view otherwise.
func (g *Game) aim() {
	cam, behind := g.viewed()
	direction := cam.Look()
	if g.pointing {
		// glonk's ray through the pointer's cell reaches one deep, and the reach is a
		// distance, so the ray is made one long.
		if _, d, ok := cam.Ray(float64(g.mouseX)+0.5, float64(g.mouseY)+0.5, g.cols, g.rows); ok {
			direction = d.Normalize()
		}
	}
	origin := cam.Position
	if behind {
		// The ray is the camera's, so that what is aimed at is what is seen under the
		// crosshair or the pointer. But it starts where it passes the player: nothing
		// between the camera and the player is aimed at, and the reach is the player's.
		passes := g.player.head().Sub(origin).Dot(cam.Look()) / direction.Dot(cam.Look())
		origin = origin.Add(direction.Scale(passes))
	}
	// A spear reaches a beast from farther than a hand reaches a block, so the ray goes
	// as far as that, and a block beyond the hand's reach is only in the way.
	far := reach + g.items[g.selected].Item.Reach()
	g.from, g.along = origin, direction // what is thrown goes the same way
	g.target, g.aimed = g.world.Target(origin, direction, far)

	// A creature in front of the block, or in reach where there is none, is aimed at in
	// its place.
	limit := far
	if g.aimed {
		limit = g.target.T
		g.aimed = g.target.T <= reach
	}
	if g.prey, g.preyed = g.aimCreature(origin, direction, limit); g.preyed {
		g.aimed = false
	}
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
// times the mouse speed of the settings; it is half as wide as it is tall, so a cell across turns half as far
// as a cell up. Until the size of the frame is known, that angle is not, and nothing turns.
func (g *Game) turn(dx, dy int) {
	if g.rows <= 0 {
		return
	}
	cell := g.settings.MouseSpeed * 2 * math.Tan(g.settings.fov()/2) / float64(g.rows)
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

// dig goes on breaking the block aimed at while the player digs, in survival, and
// reports whether it breaks in this frame. Moving on to another block starts again.
func (g *Game) dig(digging bool, dt float64) bool {
	at := [3]int{g.target.X, g.target.Y, g.target.Z}
	if !digging || !g.aimed || at != g.digAt {
		g.digAt, g.digging = at, 0
	}
	if !digging || !g.aimed {
		return false
	}
	g.digSound(g.digging, g.digging+dt)
	g.digging += dt
	if seconds, _ := g.digTime(g.target.Block); g.digging < seconds {
		return false
	}
	g.digging = 0
	g.dug = true // a held button that has broken a block digs on wherever the pointer goes
	return true
}

// digTime returns how many seconds a block takes to dig in survival with what is in hand,
// and whether it leaves anything when it breaks.
func (g *Game) digTime(b world.Block) (seconds float64, yields bool) {
	seconds, yields = b.DigWith(g.items[g.selected].Item)
	return max(seconds, minDig), yields
}

// wear wears the tool in hand by a block broken, which is gone once it is worn out. A
// block that breaks at once, as a plant does, does not wear it.
func (g *Game) wear(b world.Block) {
	if b.Info().Dig > 0 {
		g.wearOut()
	}
}

// wearOut wears the tool in hand by a use, which is gone once it is worn out. Anything
// else in hand does not wear.
func (g *Game) wearOut() {
	s := &g.items[g.selected]
	if uses := s.Item.Uses(); uses > 0 {
		if s.Wear++; s.Wear >= uses {
			*s = world.Stack{}
		}
	}
}

// build handles the keys and the mouse that pick, break and place blocks. In creative a
// block breaks at once, and a held key or button repeats, so holding Q digs on. In
// survival a block breaks once it has been dug for its time, which the right tool makes
// shorter, and leaves what it is made of, if it was dug with a tool it needs. Holding E,
// or the right button, builds on.
func (g *Game) build(in *engine.Input, dt float64) {
	for i := range hotbarSlots {
		if in.Pressed(engine.Key('1' + rune(i))) {
			g.selected = i
		}
	}
	if in.Pressed(engine.KeyTab) {
		g.selected = (g.selected + 1) % hotbarSlots
	}
	if _, steps := in.Wheel(); steps != 0 {
		g.selected = ((g.selected+steps)%hotbarSlots + hotbarSlots) % hotbarSlots
	}

	fromX, fromY := g.mouseX, g.mouseY
	slot := g.point(in)
	clicked := g.drag(in, dt, fromX, fromY)
	places := g.held(in, engine.MouseRight, dt)
	if slot >= 0 {
		// The pointer is on the hotbar: a click picks the slot, and no block behind the
		// bar is aimed at.
		if clicked || places {
			g.selected = slot
		}
		g.aimed, g.preyed, g.digging = false, false, 0
		return
	}
	places = places || keyDown(in.Pressed, 'e')
	g.hitWait = max(g.hitWait-dt, 0)

	g.aim()
	if keyDown(in.IsDown, 'r') && g.hitWait == 0 && g.throw() {
		return
	}
	if g.preyed {
		// A creature is aimed at, and no block: Q or a click hits it, again while held,
		// and E or a right click shears it. Food in hand is eaten as anywhere else.
		g.digging = 0
		switch {
		case (clicked || keyDown(in.IsDown, 'q')) && g.hitWait == 0:
			g.strike(g.prey)
		case places && !g.feed(g.prey) && !g.shear(g.prey):
			g.eat()
		}
		return
	}
	if g.drill(keyDown(in.IsDown, 'e') || in.MouseDown(engine.MouseRight), dt) {
		return
	}
	if places && g.aimed && g.useBlock(g.target) {
		return
	}
	if places && g.eat() {
		return
	}
	breaks := clicked || keyDown(in.Pressed, 'q')
	if g.survival {
		breaks = g.dig(keyDown(in.IsDown, 'q') || in.MouseDown(engine.MouseLeft) && !g.dragging, dt)
	}
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
		g.ears.Play(sound.Break(materialOf(t.Block)), cellMid(t.X, t.Y, t.Z), 1)
		g.shatter(t.X, t.Y, t.Z, t.Block)
		g.swing = swingTime
		if _, yields := g.digTime(t.Block); yields {
			g.leave(t.X, t.Y, t.Z, t.Block)
		}
		g.spill(world.Cell{X: t.X, Y: t.Y, Z: t.Z})
		if g.survival {
			g.wear(t.Block)
		}
		// A plant or a mat goes with the ground it stood on, and so does a plant that
		// stood on that one: a stand of horsetails falls whole.
		for y := t.Y + 1; ; y++ {
			above := g.world.At(t.X, y, t.Z)
			if !above.Info().Stands() {
				break
			}
			g.world.Set(t.X, y, t.Z, world.Air)
			g.leave(t.X, y, t.Z, above)
		}
	case places:
		b, ok := g.items[g.selected].Item.Block()
		if !ok {
			return // an empty slot, or a thing that is not a block
		}
		// A block goes against the face aimed at, or in the place of a plant aimed at,
		// but for a fire, which nothing takes the place of.
		gives := func(b world.Block) bool { return b.Info().Plant && !b.Info().Fixed }
		x, y, z := t.X, t.Y, t.Z
		if !gives(t.Block) {
			nx, ny, nz := t.Face.Normal()
			x, y, z = x+nx, y+ny, z+nz
		}
		if in := g.world.At(x, y, z); (in != world.Air && in != world.Water && !gives(in)) || g.player.overlaps(x, y, z) {
			return
		}
		// A plant, a torch or a mat stands on solid ground, and not in water.
		if b.Info().Stands() && (g.world.At(x, y, z) == world.Water || !g.world.At(x, y-1, z).Info().Solid) {
			return
		}
		g.world.Set(x, y, z, b)
		g.ears.Play(sound.Place(materialOf(b)), cellMid(x, y, z), 0.9)
		g.swing = swingTime
		if g.survival {
			g.items.takeOne(g.selected)
		}
	default:
		return
	}
	g.aim() // the world changed, so what is aimed at may have too
}

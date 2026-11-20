package game

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"glonk.dev/engine"
	"glonk.dev/widget"

	"github.com/kyrokohan/skyspear/internal/world"
)

// The look of everything drawn over the world: light text and frames on a dark ground,
// and one bright color for what is picked.
var (
	hudGround = engine.RGB(24, 26, 34)
	hudText   = engine.Cell{Foreground: engine.RGB(236, 236, 240), Background: hudGround}
	hudFrame  = engine.Cell{Foreground: engine.RGB(140, 146, 166), Background: hudGround}
	hudPick   = engine.Cell{Foreground: engine.RGB(255, 214, 64), Background: hudGround}
	hudFill   = engine.Cell{Character: ' ', Background: hudGround}
)

// The hotbar: nine slots side by side along the bottom, each with a picture of its
// block. A slot is slotSize cells wide and half as many tall, which is slotSize pixels
// each way, so the picture is square.
const (
	slotSize     = 6
	slotPitch    = slotSize + 1              // from one slot to the next, with the line between
	barWidth     = hotbarSlots*slotPitch + 1 // with the frame
	barHeight    = slotSize/2 + 2            // with the frame
	titleUnit    = 4                         // pixels of the title picture to each little block of its letters
	menuWidth    = 22                        // cells, with the frame
	listWide     = 46                        // the menu of the list of worlds, with the frame
	settingsWide = 36                        // the settings, with the frame
	fieldWide    = 30                        // a field a new world's name or seed is typed in, with the frame
	helpWidth    = 54                        // cells, with the frame
	numbersWidth = 32                        // cells, with the frame
	menuMost     = 10                        // the most lines a menu shows: a longer one scrolls

	hint         = " Up, Down or the mouse: choose   Enter or a click: pick " // under the title's menu
	settingsHint = " Left, Right or a click: change   Esc: back "
	typeHint     = " type it   Enter: next   Esc: back "
)

// help is the keys, shown on the pause menu.
var help = []string{
	"W A S D   walk         Arrows, drag    look",
	"Space     jump, up     X       down",
	"F         fly (creative)  Esc  this menu",
	"F3        frame info   T       6 hours on",
	"Q, click  break        E, right click  place, eat",
	"1 to 9    pick slot    Tab, wheel      next slot",
	"I         your things  V       view from behind",
	"C         crouch       L       next look",
	"R         throw a spear, or sling a pebble",
}

// barOrigin returns the top left cell of the hotbar in a frame of a size in cells: along
// the bottom, in the middle.
func barOrigin(width, height int) (x, y int) {
	return (width - barWidth) / 2, height - barHeight
}

// drawHUD draws what lies over the world while playing: the crosshair, a line of
// numbers, and the hotbar with the name of the block in use. It comes after
// Canvas.Compose, so the world is in the frame already.
func (g *Game) drawHUD(fb *engine.FrameBuffer, c *engine.Canvas) {
	// The crosshair marks the middle of the view while that is what aims. It keeps the
	// color of the world under it as its background.
	if !g.pointing {
		cx, cy := fb.Width()/2, fb.Height()/2
		cross := fb.At(cx, cy)
		cross.Character, cross.Foreground, cross.Attrs = '+', hudText.Foreground, engine.Bold
		fb.Set(cx, cy, cross)
	}

	mode := ""
	switch {
	case g.player.Flying:
		mode = "  flying"
	case g.hidden():
		mode = "  hidden"
	case g.player.Crouched:
		mode = "  crouched"
	}
	p := g.player.Pos
	status := fmt.Sprintf(" %.0f fps  %d %d %d%s  Esc: menu ", g.stats.FPS,
		int(math.Floor(p.X)), int(math.Floor(p.Y)), int(math.Floor(p.Z)), mode)
	widget.Label{Text: status, Style: hudText}.Draw(fb, 0, 0, fb.Width())
	row := 1
	if !g.exact {
		// Walking and turning at once needs two held keys, which this terminal cannot tell.
		widget.Label{Text: " no key releases here: one key at a time ", Style: hudPick}.Draw(fb, 0, row, fb.Width())
		row++
	}
	if g.world.Fault() != nil {
		// A chunk's file could not be read. The chunk was made again from the seed and
		// the file is left as it is, so what is built in that chunk will not be kept.
		widget.Label{Text: " a saved chunk could not be read: what is built there is not kept ", Style: hudPick}.Draw(fb, 0, row, fb.Width())
		row++
	}
	if g.noticeLeft > 0 {
		widget.Label{Text: " " + g.notice + " ", Style: hudPick}.Draw(fb, 0, row, fb.Width())
		row++
	}
	if g.debug {
		g.drawNumbers(fb, row)
	}

	x0, y0 := barOrigin(fb.Width(), fb.Height())
	g.drawRow(fb, c, x0, y0, g.items[:hotbarSlots])
	for i := range hotbarSlots {
		number := hudFrame
		number.Character = rune('1' + i)
		fb.Set(x0+i*slotPitch+slotPitch/2, y0, number)
	}

	// The slot in use has a bright frame of its own, and the name of what it holds over
	// the bar.
	x := x0 + g.selected*slotPitch
	widget.Panel{Style: hudPick}.Draw(fb, x, y0, slotPitch+1, barHeight)
	number := hudPick
	number.Character = rune('1' + g.selected)
	fb.Set(x+slotPitch/2, y0, number)
	if name := g.items[g.selected].Item.Name(); name != "" {
		widget.Label{Text: " " + name + " ", Align: widget.Center, Style: hudText}.Draw(fb, x0, y0-1, barWidth)
	}
	if g.survival {
		g.drawBody(fb, x0, y0-2)
	}
}

// drawRow draws slots side by side with the top left corner at (x, y): a frame round
// them with a line between each, and a picture of what each holds, with how many of it
// there are in survival. It comes after Canvas.Compose, as the HUD does.
func (g *Game) drawRow(fb *engine.FrameBuffer, c *engine.Canvas, x, y int, row []world.Stack) {
	w := len(row)*slotPitch + 1
	widget.Panel{Style: hudFrame, Fill: hudFill}.Draw(fb, x, y, w, barHeight)
	line := hudFrame
	line.Character = '│'
	for i := 1; i < len(row); i++ {
		for r := 1; r < barHeight-1; r++ {
			fb.Set(x+i*slotPitch, y+r, line)
		}
	}

	// The pictures are pixels put down over the cells of the panel. The world's pixels
	// under the row are rubbed out first, or they would be put down with them.
	c.FillRect(x, 2*y, w, 2*barHeight, engine.Color{})
	for i, s := range row {
		if s.Count == 0 {
			continue
		}
		px, py := x+1+i*slotPitch, 2*(y+1)
		g.set.Icons[s.Item.Icon()].DrawScaled(c, px, py, slotSize, slotSize)
		if uses := s.Item.Uses(); s.Wear > 0 && uses > 0 {
			// A worn tool has a bar along the bottom of its picture of how much is left.
			left := 1 - float64(s.Wear)/float64(uses)
			c.FillRect(px, py+slotSize-1, slotSize, 1, wearGround)
			c.FillRect(px, py+slotSize-1, max(int(math.Round(left*slotSize)), 1), 1, wearColor(left))
		}
	}
	c.ComposeRect(x, 2*y, w, 2*barHeight)

	if !g.survival {
		return // in creative there are as many as are wanted
	}
	for i, s := range row {
		if s.Count > 1 {
			n := strconv.Itoa(s.Count)
			widget.Label{Text: n, Style: hudText}.Draw(fb, x+(i+1)*slotPitch-len(n), y+barHeight-2, len(n))
		}
	}
}

// wearGround is under the bar of a worn tool.
var wearGround = engine.RGB(30, 30, 30)

// wearColor is the color of the bar of a worn tool with a part of it left: green when it
// is new, and red as it is worn out.
func wearColor(left float64) engine.Color {
	return engine.RGB(uint8(230*(1-left)+20), uint8(200*left+30), 40)
}

// The colors of the player's health, food and air.
var (
	healthColor = engine.RGB(226, 56, 64)
	foodColor   = engine.RGB(222, 150, 62)
	airColor    = engine.RGB(110, 182, 240)
)

// meterWidth is how many cells a meter of the body takes: a point to a cell for health
// and food.
const meterWidth = 20

// drawBody draws the player's health at the left over the hotbar, the row y, and its
// food at the right, with its air over the food while it is short of breath.
func (g *Game) drawBody(fb *engine.FrameBuffer, x0, y int) {
	meter := func(x, y int, symbol rune, value float64, color engine.Color) {
		r, gr, b := color.RGB()
		dim := engine.RGB(r/4, gr/4, b/4)
		fb.Set(x, y, engine.Cell{Character: symbol, Foreground: color, Background: hudGround})
		fb.Set(x+1, y, hudFill)
		widget.Bar{
			Value: value,
			Full:  engine.Cell{Character: '▄', Foreground: color, Background: hudGround},
			Empty: engine.Cell{Character: '▄', Foreground: dim, Background: hudGround},
		}.Draw(fb, x+2, y, meterWidth)
	}
	right := x0 + barWidth - meterWidth - 2
	meter(x0, y, '♥', g.health/maxHealth, healthColor)
	meter(right, y, '●', g.food/maxFood, foodColor)
	if g.air < maxAir {
		meter(right, y-1, '○', g.air/maxAir, airColor)
	}
}

// drawNumbers draws the frame numbers glonk reports, in a panel at the left from the
// given row down: where the time of a frame goes, and how much is sent to the terminal.
func (g *Game) drawNumbers(fb *engine.FrameBuffer, y int) {
	ms := func(d time.Duration) string {
		return fmt.Sprintf("%.1f ms", float64(d.Microseconds())/1000)
	}
	lines := []string{
		"update  " + ms(g.stats.Update),
		"draw    " + ms(g.stats.Draw),
		"write   " + ms(g.stats.Write),
		fmt.Sprintf("sent    %.1f KB a frame", float64(g.stats.Bytes)/1000),
		fmt.Sprintf("seed    %d", g.world.Seed),
		"time    " + clockText(g.days),
	}
	_, left := g.comet()
	lines = append(lines, fmt.Sprintf("comet   in %.2f days", left))
	if when, ok := g.nextStar(); ok {
		lines = append(lines, fmt.Sprintf("star    falls in %.2f days", when-g.days))
	}
	if _, far, ok := g.world.Mountain(g.player.Pos.X, g.player.Pos.Z); ok && far < eruptReach {
		_, next, _ := g.world.Erupted(g.player.Pos.X, g.player.Pos.Z, g.days)
		lines = append(lines, fmt.Sprintf("erupts  in %.2f days", next-g.days))
	}
	widget.Panel{Title: "frame", Style: hudFrame, Fill: hudFill}.Draw(fb, 0, y, numbersWidth, len(lines)+2)
	for i, line := range lines {
		widget.Label{Text: line, Style: hudText}.Draw(fb, 2, y+1+i, numbersWidth-4)
	}
}

// drawMenu draws the menu in a panel w cells wide with its top left corner at (x, y),
// and returns the row under it. It keeps where the lines went, for the mouse.
func (g *Game) drawMenu(fb *engine.FrameBuffer, title string, x, y, w, extra int) int {
	rows := min(len(g.menu.Items), menuMost)
	h := rows + extra + 2
	widget.Panel{Title: title, Style: hudFrame, Fill: hudFill}.Draw(fb, x, y, w, h)
	g.menu.Draw(fb, x+2, y+1, w-4, rows)
	g.menuX, g.menuY, g.menuW, g.menuRows = x+2, y+1, w-4, rows
	return y + h
}

// question is what the title's screen asks, over its menu, and how wide that menu is.
func (g *Game) question() (string, int) {
	name := func() string { return g.list[g.chosen].name }
	switch g.screen {
	case confirming:
		return "replace your saved world?", 0
	case choosing:
		return "how will this world be played?", 0
	case worldChosen:
		return name(), 0
	case deleting:
		return "delete " + name() + "?", 0
	case deletingSure:
		return name() + " cannot be brought back", 0
	case naming:
		return "name the new world", fieldWide
	case seeding:
		return "its seed (none: any)", fieldWide
	case settingsScreen:
		return "settings", settingsWide
	case onTitle:
		if g.listing() {
			return "worlds", listWide
		}
	}
	return "", menuWidth
}

// drawTitle draws the title screen over the world: the name of the game as pixels,
// then the menu under it.
func (g *Game) drawTitle(fb *engine.FrameBuffer, c *engine.Canvas) {
	// The picture is drawn at a whole number of pixels to each little block of its
	// letters, the most that fits, so every letter keeps its shape.
	cols, rows := g.set.Title.Width()/titleUnit, g.set.Title.Height()/titleUnit
	scale := min(max((c.Width()-4)/cols, 1), titleUnit)
	w, h := cols*scale, rows*scale
	top := max(c.Height()/6, 1)
	g.set.Title.DrawScaled(c, (c.Width()-w)/2, top, w, h)
	c.Compose()

	question, width := g.question()
	if width == 0 {
		width = max(engine.TextWidth(question)+8, menuWidth)
	}
	x, y := (fb.Width()-width)/2, min((top+h)/2+2, fb.Height()-min(len(g.menu.Items), menuMost)-3)
	var under int
	keys := hint
	switch g.screen {
	case naming, seeding:
		g.field.draw(fb, question, x, y, width)
		under, keys = y+3, typeHint
	case settingsScreen:
		under, keys = g.drawMenu(fb, question, x, y, width, 0), settingsHint
	default:
		under = g.drawMenu(fb, question, x, y, width, 0)
	}
	widget.Label{Text: keys, Align: widget.Center, Style: hudText}.Draw(fb, 0, under+1, fb.Width())
	g.drawNote(fb, under+2)
	if g.screen == choosing {
		for i, line := range []string{
			" Survival: find what you need, dig it, and stay alive ",
			" Creative: every block without end, and nothing hurts ",
		} {
			widget.Label{Text: line, Align: widget.Center, Style: hudFrame}.Draw(fb, 0, under+3+i, fb.Width())
		}
	}
	if g.opts.OldSave != "" && !g.saved && g.screen == onTitle {
		// The one world of the game's first version. It is kept, and cannot be played.
		note := " the world saved by an older gocraft is kept, but this version cannot play it "
		widget.Label{Text: note, Align: widget.Center, Style: hudFrame}.Draw(fb, 0, under+3, fb.Width())
	}
}

// drawNote draws on row y what last went wrong, or what was done, if anything was.
func (g *Game) drawNote(fb *engine.FrameBuffer, y int) {
	if g.note != "" {
		widget.Label{Text: " " + g.note + " ", Align: widget.Center, Style: hudPick}.Draw(fb, 0, y, fb.Width())
	}
}

// drawSettings draws the settings over the paused world.
func (g *Game) drawSettings(fb *engine.FrameBuffer) {
	x, y := (fb.Width()-settingsWide)/2, max((fb.Height()-len(g.menu.Items)-2)/2, 0)
	under := g.drawMenu(fb, "settings", x, y, settingsWide, 0)
	widget.Label{Text: settingsHint, Align: widget.Center, Style: hudText}.Draw(fb, 0, under+1, fb.Width())
	g.drawNote(fb, under+2)
}

// drawPause draws the pause menu over the world, with the keys under it.
func (g *Game) drawPause(fb *engine.FrameBuffer) {
	h := len(g.menu.Items) + len(help) + 3
	x, y := (fb.Width()-helpWidth)/2, max((fb.Height()-h)/2, 0)
	under := g.drawMenu(fb, "paused", x, y, helpWidth, len(help)+1)
	for i, line := range help {
		widget.Label{Text: line, Style: hudFrame}.Draw(fb, x+2, y+len(g.menu.Items)+2+i, helpWidth-4)
	}
	g.drawNote(fb, under+1)
}

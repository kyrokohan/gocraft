package game

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"glonk.dev/engine"
	"glonk.dev/widget"

	"github.com/kyrokohan/skyspear/internal/world"
)

// The title lists the worlds in Options.Worlds, the last played first: one is chosen to
// play or to delete, or a new one is made with a name, a seed and a way to play it. Each
// world is a folder of that name.

// savedWorld is a world in the list.
type savedWorld struct {
	name  string
	path  string
	about string    // how it is played and which day it is, or that it cannot be read
	when  time.Time // when it was last saved
}

// readWorlds returns the worlds in a folder, the last saved first. A folder with no
// world in it is not one; a world that cannot be read is listed, so that it can be
// deleted.
func readWorlds(dir string) []savedWorld {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var list []savedWorld
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := os.Stat(filepath.Join(path, "world.json"))
		if err != nil {
			continue
		}
		w := savedWorld{name: e.Name(), path: path, when: info.ModTime(), about: "cannot be read"}
		if m, err := world.Peek(path); err == nil {
			mode := m.Mode
			if mode == "" {
				mode = world.Creative
			}
			w.about = fmt.Sprintf("%s, day %d", mode, int(math.Floor(firstMorning+m.Days))+1)
		}
		list = append(list, w)
	}
	slices.SortStableFunc(list, func(a, b savedWorld) int { return b.when.Compare(a.when) })
	return list
}

// listing reports whether the title lists worlds to choose from, and not the one world
// of Options.SavePath.
func (g *Game) listing() bool {
	return g.opts.Worlds != ""
}

// titleItems are the lines of the title's menu: the worlds, and then what else can be
// done there.
func (g *Game) titleItems() []string {
	if !g.listing() {
		if g.saved {
			return []string{itemContinue, itemNew, itemSettings, itemQuit}
		}
		return []string{itemNew, itemSettings, itemQuit}
	}
	g.list = readWorlds(g.opts.Worlds)
	var items []string
	for _, w := range g.list {
		items = append(items, fmt.Sprintf("%-*s %s", nameLength, w.name, w.about))
	}
	return append(items, itemNew, itemSettings, itemQuit)
}

// playWorld opens a world of the list and plays it.
func (g *Game) playWorld(w savedWorld) {
	opened, s, err := world.Open(w.path)
	if err != nil {
		g.note = "that world cannot be read: " + reason(err)
		g.show(onTitle)
		return
	}
	g.opts.SavePath = w.path
	g.use(opened, s)
	g.show(playing)
}

// reason is the last part of an error: what is wrong, without where.
func reason(err error) string {
	text := err.Error()
	if i := strings.LastIndex(text, ": "); i >= 0 {
		return text[i+2:]
	}
	return text
}

// deleteWorld deletes a world of the list for good. It deletes nothing but a folder in
// the folder of worlds, and if it is the world being played, it is not saved again.
func (g *Game) deleteWorld(w savedWorld) {
	if filepath.Dir(w.path) != filepath.Clean(g.opts.Worlds) || w.name != filepath.Base(w.path) {
		return
	}
	if err := os.RemoveAll(w.path); err != nil {
		g.note = "it could not be deleted: " + reason(err)
		return
	}
	if g.opts.SavePath == w.path {
		g.opts.SavePath, g.saved, g.started = "", false, false
	}
	g.note = w.name + " is deleted"
}

// nameLength is the most letters a world's name has, and seedLength a seed's.
const (
	nameLength = 20
	seedLength = 19
)

// field is a line of text being typed.
type field struct {
	text  string
	most  int
	allow func(rune) bool
}

// update types into the field what was typed or pasted, and takes the last letter off
// for Backspace.
func (f *field) update(in *engine.Input) {
	for _, r := range in.Text() + in.Paste() {
		if f.allow(r) && utf8.RuneCountInString(f.text) < f.most {
			f.text += string(r)
		}
	}
	if in.Pressed(engine.KeyBackspace) && f.text != "" {
		_, last := utf8.DecodeLastRuneInString(f.text)
		f.text = f.text[:len(f.text)-last]
	}
}

// draw draws the field in a panel w cells wide with its top left corner at (x, y), with
// a mark where the next letter goes.
func (f *field) draw(fb *engine.FrameBuffer, title string, x, y, w int) {
	widget.Panel{Title: title, Style: hudFrame, Fill: hudFill}.Draw(fb, x, y, w, 3)
	widget.Label{Text: f.text + "▏", Style: hudPick}.Draw(fb, x+2, y+1, w-4)
}

// nameField and seedField are the fields a new world is made with.
func nameField(text string) field {
	return field{text: text, most: nameLength, allow: func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' || r == '_'
	}}
}

func seedField() field {
	return field{most: seedLength, allow: func(r rune) bool { return r == '-' || ('0' <= r && r <= '9') }}
}

// freeName returns a name for a new world that no world in the list has.
func (g *Game) freeName() string {
	for n := 1; ; n++ {
		name := "world " + strconv.Itoa(n)
		if n == 1 {
			name = "world"
		}
		if _, err := os.Stat(filepath.Join(g.opts.Worlds, name)); errors.Is(err, fs.ErrNotExist) {
			return name
		}
	}
}

// typeIn reads the keys while a new world's name or seed is typed: Enter goes on to the
// next question, if what was typed will do.
func (g *Game) typeIn(in *engine.Input) {
	g.field.update(in)
	if !in.Started(engine.KeyEnter) {
		return
	}
	g.note = ""
	text := strings.TrimSpace(g.field.text)
	switch g.screen {
	case naming:
		_, err := os.Stat(filepath.Join(g.opts.Worlds, text))
		switch {
		case !strings.ContainsFunc(text, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }):
			g.note = "a name needs a letter or a number"
		case err == nil || !errors.Is(err, fs.ErrNotExist):
			g.note = "there is a world of that name"
		default:
			g.newName, g.field = text, seedField()
			if g.opts.Seed != 0 {
				g.field.text = strconv.FormatInt(g.opts.Seed, 10) // the seed the game was started with
			}
			g.show(seeding)
		}
	case seeding:
		g.newSeed = 0
		if text != "" {
			seed, err := strconv.ParseInt(text, 10, 64)
			if err != nil || seed == 0 {
				g.note = "a seed is a whole number, not 0, and not too long"
				return
			}
			g.newSeed = seed
		}
		g.show(choosing)
	}
}

// pointMenu reads the mouse over the menu where it was last drawn: moving onto a line
// picks it, and a click chooses it.
func (g *Game) pointMenu(in *engine.Input) (choice int, chosen bool) {
	x, y, ok := in.MousePos()
	if !ok || g.menuRows == 0 {
		return -1, false
	}
	moved := x != g.mouseX || y != g.mouseY
	g.mouseX, g.mouseY = x, y
	if x < g.menuX || x >= g.menuX+g.menuW || y < g.menuY || y >= g.menuY+g.menuRows {
		return -1, false
	}
	// The rows the menu shows, as widget.Menu scrolls them.
	n, top := len(g.menu.Items), 0
	if n > g.menuRows {
		top = min(max(g.menu.Selected-g.menuRows/2, 0), n-g.menuRows)
	}
	i := top + y - g.menuY
	if i >= n {
		return -1, false
	}
	if moved && n <= g.menuRows {
		g.menu.Selected = i // a list that scrolls would move from under the pointer
	}
	if in.MouseStarted(engine.MouseLeft) {
		g.menu.Selected = i
		return i, true
	}
	return -1, false
}

package game

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/world"
)

// listing is a game whose title lists the worlds in a folder of the test's own, with
// settings kept in a file of its own. The world behind the title is the test floor.
func listing(t *testing.T, dir string) *Game {
	t.Helper()
	g, err := New(Options{Seed: 1, Worlds: dir, Settings: filepath.Join(t.TempDir(), "settings.json"), world: floor()})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// saveWorld saves a world into a folder of dir, as a game of that mode on that day
// would, and dates the save.
func saveWorld(t *testing.T, dir, name, mode string, days float64, when time.Time) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := floor().Save(path, world.Meta{Pos: standing().Pos, Mode: mode, Days: days, Health: maxHealth, Food: maxFood, Air: maxAir}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(path, "world.json"), when, when); err != nil {
		t.Fatal(err)
	}
}

// typeKeys runs one frame with text typed and then the keys pressed.
func typeKeys(t *testing.T, g *Game, text string, keys ...engine.Key) {
	t.Helper()
	kb := engine.NewKeyboard()
	kb.Type(text)
	kb.Press(keys...)
	if err := g.Update(frame, kb.Input()); err != nil {
		t.Fatal(err)
	}
}

// erase takes every letter out of the field being typed in.
func erase(t *testing.T, g *Game) {
	t.Helper()
	for g.field.text != "" {
		typeKeys(t, g, "", engine.KeyBackspace)
	}
}

func TestTheTitleListsTheWorldsTheLastSavedFirst(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	saveWorld(t, dir, "old one", world.Creative, 0, now.Add(-time.Hour))
	saveWorld(t, dir, "new one", world.Survival, 2, now)
	if err := os.Mkdir(filepath.Join(dir, "not a world"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "broken"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken", "world.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	g := listing(t, dir)
	text, _ := drawn(t, g)

	var names []string
	for _, w := range g.list {
		names = append(names, w.name)
	}
	if strings.Join(names, ",") != "new one,broken,old one" && strings.Join(names, ",") != "broken,new one,old one" {
		t.Errorf("listed %q, wanted the new one before the old one and no folder without a world", names)
	}
	for _, want := range []string{"worlds", "new one", "survival, day 3", "old one", "creative, day 1", "cannot be read", itemNew, itemSettings} {
		if !strings.Contains(text, want) {
			t.Errorf("the list does not show %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "not a world") {
		t.Errorf("a folder with no world in it is listed:\n%s", text)
	}
}

func TestANewWorldIsNamedSeededAndSavedInTheList(t *testing.T) {
	dir := t.TempDir()
	g, err := New(Options{Seed: 1, Worlds: dir}) // a world made from its seed
	if err != nil {
		t.Fatal(err)
	}

	press(g, engine.KeyEnter) // New world, the only line above Settings and Quit
	if g.screen != naming || g.field.text != "world" {
		t.Fatalf("screen %d with %q typed, wanted the name asked for with world offered", g.screen, g.field.text)
	}
	if text, _ := drawn(t, g); !strings.Contains(text, "name the new world") {
		t.Errorf("the name is not asked for:\n%s", text)
	}
	erase(t, g)
	typeKeys(t, g, "home/2", engine.KeyEnter) // a slash cannot be in a folder's name
	if g.screen != seeding || g.newName != "home2" {
		t.Fatalf("screen %d named %q, wanted the seed asked for of home2", g.screen, g.newName)
	}
	if g.field.text != "1" {
		t.Errorf("the seed offered is %q, wanted the one the game was started with", g.field.text)
	}
	erase(t, g)
	typeKeys(t, g, "4x2", engine.KeyEnter)
	if g.screen != choosing || g.newSeed != 42 {
		t.Fatalf("screen %d with seed %d, wanted the mode asked for with seed 42", g.screen, g.newSeed)
	}
	press(g, engine.KeyEnter) // Survival

	if g.screen != playing || !g.survival || g.world.Seed != 42 {
		t.Fatalf("screen %d, survival %v, seed %d: wanted play in survival in world 42", g.screen, g.survival, g.world.Seed)
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	if list := readWorlds(dir); len(list) != 1 || list[0].name != "home2" || list[0].about != "survival, day 1" {
		t.Errorf("the list is %+v, wanted home2 in survival on day 1", list)
	}
}

func TestANameThatIsTakenOrHasNoLetterAndABadSeedAreRefused(t *testing.T) {
	dir := t.TempDir()
	saveWorld(t, dir, "home", world.Creative, 0, time.Now())
	g := listing(t, dir)
	press(g, engine.KeyDown, engine.KeyEnter) // New world, under home

	for _, name := range []string{"home", " - _ "} {
		erase(t, g)
		typeKeys(t, g, name, engine.KeyEnter)
		if g.screen != naming || g.note == "" {
			t.Errorf("named %q: screen %d with the note %q, wanted the name asked for again, and why", name, g.screen, g.note)
		}
	}
	if text, _ := drawn(t, g); !strings.Contains(text, g.note) {
		t.Errorf("the note %q is not shown:\n%s", g.note, text)
	}
	erase(t, g)
	typeKeys(t, g, strings.Repeat("a", nameLength+5), engine.KeyEnter)
	if g.newName != strings.Repeat("a", nameLength) {
		t.Errorf("named %q, wanted the name cut at %d letters", g.newName, nameLength)
	}
	for _, seed := range []string{"0", "-", "99999999999999999999"} {
		erase(t, g)
		typeKeys(t, g, seed, engine.KeyEnter)
		if g.screen != seeding || g.note == "" {
			t.Errorf("seed %q: screen %d with the note %q, wanted the seed asked for again", seed, g.screen, g.note)
		}
	}
	erase(t, g)
	typeKeys(t, g, "", engine.KeyEnter)
	if g.screen != choosing || g.newSeed != 0 {
		t.Errorf("no seed: screen %d with seed %d, wanted the mode asked for and a seed from the clock", g.screen, g.newSeed)
	}
	if press(g, engine.KeyEscape); g.screen != onTitle {
		t.Errorf("Escape went to screen %d, wanted the title", g.screen)
	}
}

func TestAWorldIsPlayedLeftAndDeletedOnlyWhenAskedTwice(t *testing.T) {
	dir := t.TempDir()
	saveWorld(t, dir, "home", world.Survival, 0, time.Now())
	path := filepath.Join(dir, "home")
	g := listing(t, dir)

	press(g, engine.KeyEnter) // home
	if text, _ := drawn(t, g); g.screen != worldChosen || !strings.Contains(text, "home") || !strings.Contains(text, itemPlay) {
		t.Fatalf("screen %d, wanted home asked about:\n%s", g.screen, text)
	}
	press(g, engine.KeyEnter) // Play
	if g.screen != playing || g.opts.SavePath != path || !g.survival {
		t.Fatalf("screen %d in %q, wanted home played in survival", g.screen, g.opts.SavePath)
	}
	g.player.Pos.X += 3
	press(g, engine.KeyEscape)
	if text, _ := drawn(t, g); !strings.Contains(text, itemLeave) {
		t.Errorf("the pause menu has no %q:\n%s", itemLeave, text)
	}
	press(g, engine.KeyUp, engine.KeyEnter) // Save and leave
	if m, err := world.Peek(path); g.screen != onTitle || err != nil || m.Pos.X != g.player.Pos.X {
		t.Fatalf("screen %d, saved at %v (%v): wanted the title, and home saved where the player left it", g.screen, m.Pos, err)
	}

	press(g, engine.KeyEnter)                 // home
	press(g, engine.KeyDown, engine.KeyEnter) // Delete
	press(g, engine.KeyEnter)                 // Keep it
	if _, err := os.Stat(path); g.screen != onTitle || err != nil {
		t.Fatalf("kept: screen %d, %v: wanted the title and home still there", g.screen, err)
	}

	press(g, engine.KeyEnter)
	press(g, engine.KeyDown, engine.KeyEnter)
	press(g, engine.KeyDown, engine.KeyEnter) // Yes, delete it
	if text, _ := drawn(t, g); g.screen != deletingSure || !strings.Contains(text, "cannot be brought back") {
		t.Fatalf("screen %d, wanted to be asked again:\n%s", g.screen, text)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("home is gone after the first yes: %v", err)
	}
	press(g, engine.KeyDown, engine.KeyEnter) // Yes, delete it for good

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) || g.screen != onTitle {
		t.Fatalf("screen %d, %v: wanted the title and home gone", g.screen, err)
	}
	if text, _ := drawn(t, g); !strings.Contains(text, "home is deleted") || strings.Contains(text, "survival, day") {
		t.Errorf("wanted the list empty and a note of what was done:\n%s", text)
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the world it was playing was saved again after it was deleted: %v", err)
	}
}

func TestOnlyAFolderOfTheListIsDeleted(t *testing.T) {
	dir, outside := t.TempDir(), t.TempDir()
	g := listing(t, dir)

	g.deleteWorld(savedWorld{name: filepath.Base(outside), path: outside})
	g.deleteWorld(savedWorld{name: "..", path: filepath.Join(dir, "..")})

	if _, err := os.Stat(outside); err != nil {
		t.Errorf("a folder outside the list was deleted: %v", err)
	}
}

func TestSettingsChangeAreKeptAndAreUsed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	g, err := New(Options{Seed: 1, Settings: path, world: floor()})
	if err != nil {
		t.Fatal(err)
	}

	press(g, engine.KeyDown, engine.KeyEnter) // Settings, under New world
	if text, _ := drawn(t, g); g.screen != settingsScreen || !strings.Contains(text, "Mouse speed") || !strings.Contains(text, "1.5") {
		t.Fatalf("screen %d, wanted the settings:\n%s", g.screen, text)
	}
	for _, key := range []engine.Key{
		engine.KeyRight,                // mouse speed 1.5 to 2
		engine.KeyDown, engine.KeyLeft, // view distance 64 to 48
		engine.KeyDown, engine.KeyLeft, // field of view 60 to 50
		engine.KeyDown, engine.KeyLeft, // volume 70 to 60
		engine.KeyDown, engine.KeyEnter, // Back
	} {
		press(g, key)
	}

	want := Settings{MouseSpeed: 2, ViewDistance: 48, FieldOfView: 50, Volume: 60}
	if g.settings != want || g.screen != onTitle {
		t.Fatalf("settings %+v on screen %d, wanted %+v and the title", g.settings, g.screen, want)
	}
	if kept := loadSettings(path); kept != want {
		t.Errorf("kept %+v, wanted %+v", kept, want)
	}
	g.Play()
	if cam := g.camera(); !near(cam.FOV, 50*math.Pi/180) || cam.Far != 48 || cam.Fog.End != 48 {
		t.Errorf("the view is %v wide and %v deep, with fog to %v: wanted the settings", cam.FOV, cam.Far, cam.Fog.End)
	}

	// From the pause menu, the settings go back to it.
	press(g, engine.KeyEscape)
	press(g, engine.KeyDown, engine.KeyEnter)
	if text, _ := drawn(t, g); g.screen != settingsScreen || !strings.Contains(text, "48 blocks") {
		t.Fatalf("screen %d, wanted the settings over the paused world:\n%s", g.screen, text)
	}
	if press(g, engine.KeyEscape); g.screen != paused {
		t.Errorf("Escape from the settings went to screen %d, wanted the pause menu", g.screen)
	}
}

func TestSettingsReadFromAFileAreOnesTheGameHas(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	for _, tc := range []struct {
		name string
		path string
		want Settings
	}{
		{"none", filepath.Join(dir, "none.json"), defaultSettings},
		{"not JSON", write("bad.json", "{"), defaultSettings},
		{"near ones", write("near.json", `{"mouseSpeed": 1.3, "viewDistance": 1000, "fieldOfView": -5, "volume": 77}`), Settings{1.25, 64, 50, 80}},
		{"some", write("some.json", `{"viewDistance": 32}`), Settings{1.5, 32, 60, 70}},
	} {
		if got := loadSettings(tc.path); got != tc.want {
			t.Errorf("%s: %+v, wanted %+v", tc.name, got, tc.want)
		}
	}

	s := Settings{MouseSpeed: 3, ViewDistance: 64, FieldOfView: 50, Volume: 0}
	s.step(0, 1)
	s.step(1, 1)
	s.step(2, -1)
	s.step(3, -1)
	if want := (Settings{MouseSpeed: 0.5, ViewDistance: 32, FieldOfView: 90, Volume: 100}); s != want {
		t.Errorf("past the ends: %+v, wanted %+v", s, want)
	}
}

func TestAMenuFollowsThePointerAndTakesAClick(t *testing.T) {
	g := newGame(t)
	kb := engine.NewKeyboard()
	drawn(t, g) // the menu is where it was last drawn

	kb.MouseMove(g.menuX+1, g.menuY+1) // Settings
	step(t, g, kb, 1)
	if g.menu.Selected != 1 {
		t.Fatalf("the pointer on line 1 picked line %d", g.menu.Selected)
	}
	kb.MouseMove(g.menuX+g.menuW, g.menuY+2) // beside Quit
	step(t, g, kb, 1)
	if g.menu.Selected != 1 {
		t.Errorf("the pointer beside the menu picked line %d", g.menu.Selected)
	}

	kb.MouseMove(g.menuX, g.menuY+2)
	kb.MouseDown(engine.MouseLeft)
	if err := g.Update(frame, kb.Input()); !errors.Is(err, engine.ErrQuit) {
		t.Errorf("a click on Quit returned %v, wanted ErrQuit", err)
	}
}

func TestAFieldTakesWhatIsTypedAndPasted(t *testing.T) {
	f := nameField("")
	kb := engine.NewKeyboard()
	kb.Type("my/wörld")
	kb.Paste(" two\n")
	f.update(kb.Input())
	if f.text != "mywörld two" {
		t.Errorf("typed %q, wanted mywörld two", f.text)
	}
	kb.EndFrame()
	kb.Press(engine.KeyBackspace)
	f.update(kb.Input())
	if f.text != "mywörld tw" {
		t.Errorf("after Backspace %q, wanted mywörld tw", f.text)
	}
}

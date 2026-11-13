package game

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/world"
)

// savedGame is a game with a save file, in a folder of the test's own.
func savedGame(t *testing.T, path string) *Game {
	t.Helper()
	g, err := New(Options{Seed: 1, SavePath: path})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// playAndSave starts a new world, changes it and the player, and saves. It returns
// the cell of a block that was broken.
func playAndSave(t *testing.T, path string) (x, y, z int) {
	t.Helper()
	g := savedGame(t, path)
	if err := press(g, engine.KeyEnter); err != nil { // New world
		t.Fatal(err)
	}
	x, z = 100, 90
	y = g.world.Surface(x, z)
	g.world.Set(x, y, z, world.Air)
	g.player.Pos.X += 0.25
	g.player.Yaw, g.player.Pitch, g.player.Flying, g.selected = 1.25, -0.5, true, 4
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	return x, y, z
}

func TestWithNoSaveTheTitleOffersOnlyANewWorld(t *testing.T) {
	g := savedGame(t, filepath.Join(t.TempDir(), "world.sav"))

	text, _ := drawn(t, g)

	if strings.Contains(text, itemContinue) || !strings.Contains(text, itemNew) {
		t.Errorf("wanted a new world and nothing to continue:\n%s", text)
	}
}

func TestContinueBringsTheWorldAndThePlayerBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world.sav")
	x, y, z := playAndSave(t, path)
	first := savedGame(t, path) // only to know where the player was left

	g := savedGame(t, path)
	text, _ := drawn(t, g)
	if !strings.Contains(text, itemContinue) {
		t.Fatalf("the title does not offer to continue:\n%s", text)
	}
	if err := press(g, engine.KeyEnter); err != nil { // Continue is the first line
		t.Fatal(err)
	}

	if g.screen != playing {
		t.Fatalf("screen %d, wanted play", g.screen)
	}
	if got := g.world.At(x, y, z); got != world.Air {
		t.Errorf("the broken block came back as %v", got.Info().Name)
	}
	p := g.player
	if p.Pos != first.player.Pos || p.Yaw != 1.25 || p.Pitch != -0.5 || !p.Flying || g.selected != 4 {
		t.Errorf("the player came back as %+v with slot %d", p, g.selected)
	}
}

func TestAWorldNobodyPlayedIsNotSaved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world.sav")
	g := savedGame(t, path)

	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a save was written from the title screen (stat: %v)", err)
	}
}

func TestAGameWithNoSaveFileSavesNothing(t *testing.T) {
	g := onFloor(t)

	if err := g.Save(); err != nil {
		t.Errorf("got %v, wanted nothing to happen", err)
	}
}

func TestANewWorldAsksBeforeItReplacesTheSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world.sav")
	x, y, z := playAndSave(t, path)

	// No: back to the title, and nothing is written.
	g := savedGame(t, path)
	press(g, engine.KeyDown, engine.KeyEnter) // New world
	text, _ := drawn(t, g)
	if g.screen != confirming || !strings.Contains(text, "replace your saved world?") {
		t.Fatalf("screen %d, wanted the question:\n%s", g.screen, text)
	}
	press(g, engine.KeyEnter) // No, keep it
	if g.screen != onTitle || g.world.At(x, y, z) != world.Air {
		t.Errorf("screen %d: wanted the title again with the saved world behind it", g.screen)
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	// Escape is a no as well.
	press(g, engine.KeyDown, engine.KeyEnter)
	press(g, engine.KeyEscape)
	if g.screen != onTitle {
		t.Errorf("screen %d after Escape, wanted the title", g.screen)
	}

	// Yes: a new world, played at once, that takes the place of the save when saved.
	press(g, engine.KeyDown, engine.KeyEnter)
	press(g, engine.KeyDown, engine.KeyEnter) // Yes, start a new one
	if g.screen != playing || g.world.At(x, y, z) == world.Air {
		t.Fatalf("screen %d: wanted play in a new world", g.screen)
	}
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}
	if again := savedGame(t, path); again.world.At(x, y, z) == world.Air {
		t.Error("the save still holds the old world")
	}
}

func TestASaveThatCannotBeReadStopsTheGameAndIsLeftAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world.sav")
	if err := os.WriteFile(path, []byte("not a world"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := New(Options{Seed: 1, SavePath: path}); err == nil {
		t.Fatal("wanted an error for a save that cannot be read")
	}

	if data, _ := os.ReadFile(path); string(data) != "not a world" {
		t.Error("the file was changed")
	}
}

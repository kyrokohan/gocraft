package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"glonk.dev/engine"

	"github.com/kyrokohan/skyspear/internal/assets"
)

func TestLStepsThroughTheLooksAndWrapsRound(t *testing.T) {
	g := fromBehind(t, onFloor(t))
	for i := 1; i <= assets.PersonLooks; i++ {
		hold(t, g, 3, 'l') // held for three frames: only its going down counts
		want := i % assets.PersonLooks
		text, _ := drawn(t, g)
		if g.worn != want || g.cast[0].Look != want {
			t.Fatalf("after %d presses of L the look is %d and the body's %d, wanted %d", i, g.worn, g.cast[0].Look, want)
		}
		if said := fmt.Sprintf("look %d of %d", want+1, assets.PersonLooks); !strings.Contains(text, said) {
			t.Errorf("the screen does not say %q:\n%s", said, text)
		}
	}
}

func TestTheLookIsSavedWithTheWorld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world")
	g := savedGame(t, path)
	startNew(t, g, false)
	if g.worn != 0 {
		t.Fatalf("a new world's player wears look %d, wanted the first", g.worn)
	}
	g.worn = 4
	if err := g.Save(); err != nil {
		t.Fatal(err)
	}

	again := savedGame(t, path)
	press(again, engine.KeyEnter) // Continue is the first line
	if again.worn != 4 {
		t.Errorf("the world came back with look %d, wanted 4", again.worn)
	}

	// A look the game does not have, in a file changed by hand, is the nearest it has.
	file := filepath.Join(path, "world.json")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(data), `"look": 4`, `"look": 99`, 1)
	if changed == string(data) {
		t.Fatal("the test did not change the file")
	}
	if err := os.WriteFile(file, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	odd := savedGame(t, path)
	press(odd, engine.KeyEnter)
	if odd.worn != assets.PersonLooks-1 {
		t.Errorf("a world with look 99 came back with look %d, wanted the last", odd.worn)
	}
}

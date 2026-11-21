package world

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"glonk.dev/geometry"
)

// played is a world with a few things changed from what its seed makes, and what its
// player was doing. One change is in a chunk that has since left the window.
func played(t *testing.T) (*World, Meta) {
	t.Helper()
	w := Generate(3)
	w.Set(10, 100, 10, Brick)
	w.Set(11, 100, 10, Glass)
	w.Set(-20, 101, 7, Planks) // another chunk, below 0
	w.Follow(700, 0)
	w.LoadAll() // the three changes are far away now, and set aside
	w.Set(705, 102, 3, Sand)
	return w, Meta{
		Pos:    geometry.Vector3{X: 702.25, Y: 90, Z: 1.5},
		Yaw:    1.5,
		Pitch:  -0.25,
		Flying: true,
		Slot:   7,
		Mode:   Survival,
		Items:  []Stack{{Item: Dirt.Item(), Count: 12}, {}, {Item: Berries, Count: 3}, {Item: StonePick, Count: 1, Wear: 7}},
		Health: 15,
		Food:   11.5,
		Air:    4,
		Drops:  []Drop{{Stack: Stack{Item: Cobble.Item(), Count: 1}, Pos: geometry.Vector3{X: 703.5, Y: 91, Z: 2.5}, Age: 12}},
		Mossbacks: []Mossback{
			{Pos: geometry.Vector3{X: 710.5, Y: 88, Z: 4.5}, Yaw: 0.5, Regrow: 120},
			{Pos: geometry.Vector3{X: 690.5, Y: 89, Z: -3.5}, Yaw: -2},
		},
		Pots: []PotStore{{Cell: Cell{X: 705, Y: 102, Z: 4}, Items: []Stack{{}, {Item: Coal, Count: 5}}}},
		Hearths: []HearthStore{{
			Cell: Cell{X: 706, Y: 102, Z: 3},
			In:   Stack{Item: IronOre.Item(), Count: 2}, Fuel: Stack{Item: Coal, Count: 1}, Out: Stack{Item: IronBar, Count: 4},
			Burn: 20.5, Burns: 48, Cooked: 2.25,
		}},
		Bed:       &Cell{X: 704, Y: 103, Z: 3},
		Paintings: []Painting{{Side: Side{Cell: Cell{X: 705, Y: 101, Z: 3}, Face: FaceZNeg}, Picture: Spiral, Paint: RedPaint}},
		Met:       []string{"comet", "daggercat"},
		Camps:     []Camp{{Fire: Cell{X: -120, Y: 70, Z: 44}, Trust: 6}},
	}
}

func chunkFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, chunkFolder))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestASavedWorldComesBackAsItWas(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "worlds", "one")
	w, meta := played(t)

	if err := w.Save(dir, meta); err != nil {
		t.Fatal(err)
	}
	got, back, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	if back.Seed != 3 || got.Seed != 3 || back.Version != saveVersion {
		t.Errorf("seed %d and %d, version %d: wanted 3, 3 and %d", back.Seed, got.Seed, back.Version, saveVersion)
	}
	if back.Pos != meta.Pos || back.Yaw != meta.Yaw || back.Pitch != meta.Pitch || !back.Flying || back.Slot != 7 {
		t.Errorf("the player came back as %+v", back)
	}
	if back.Mode != Survival || back.Health != 15 || back.Food != 11.5 || back.Air != 4 ||
		!slices.Equal(back.Items, meta.Items) || !slices.Equal(back.Drops, meta.Drops) ||
		!slices.Equal(back.Mossbacks, meta.Mossbacks) {
		t.Errorf("the player's body and things came back as %+v", back)
	}
	if !reflect.DeepEqual(back.Pots, meta.Pots) || !slices.Equal(back.Hearths, meta.Hearths) || back.Bed == nil || *back.Bed != *meta.Bed {
		t.Errorf("the pots, the hearths and the bed came back as %+v, %+v and %v", back.Pots, back.Hearths, back.Bed)
	}
	if !slices.Equal(back.Paintings, meta.Paintings) || !slices.Equal(back.Met, meta.Met) {
		t.Errorf("the paintings and what was met came back as %+v and %v", back.Paintings, back.Met)
	}
	// The world opens round the player, with what was built there.
	if !got.Loaded(702, 1) || got.At(705, 102, 3) != Sand {
		t.Errorf("round the player: loaded %v, the sand is %v", got.Loaded(702, 1), got.At(705, 102, 3))
	}
	// And what was built far away is there when the player goes back.
	got.Follow(10, 10)
	got.LoadAll()
	if got.At(10, 100, 10) != Brick || got.At(11, 100, 10) != Glass || got.At(-20, 101, 7) != Planks {
		t.Error("what was built at the start did not come back")
	}
	// A ray stops at the height, so it must not be under the highest block.
	if got.Height() < 102 {
		t.Errorf("the world came back %d high, with a block at layer 101", got.Height())
	}
}

func TestOnlyChangedChunksAreSaved(t *testing.T) {
	dir := t.TempDir()
	w, meta := played(t)

	if err := w.Save(dir, meta); err != nil {
		t.Fatal(err)
	}

	// Three chunks were changed: 0,0 and -2,0 at the start, and 44,0 by the player.
	if names := chunkFiles(t, dir); len(names) != 3 {
		t.Errorf("the chunks folder holds %v, wanted three files", names)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Errorf("the world's folder holds %d things, wanted world.json and chunks", len(entries))
	}
}

func TestSavingAgainWritesOnlyWhatChangedSince(t *testing.T) {
	dir := t.TempDir()
	w, meta := played(t)
	if err := w.Save(dir, meta); err != nil {
		t.Fatal(err)
	}
	before := map[string]os.FileInfo{}
	for _, name := range chunkFiles(t, dir) {
		before[name], _ = os.Stat(filepath.Join(dir, chunkFolder, name))
	}

	w.Set(725, 103, 0, Log) // one more chunk, the next one along
	if err := w.Save(dir, meta); err != nil {
		t.Fatal(err)
	}

	names := chunkFiles(t, dir)
	if len(names) != 4 {
		t.Fatalf("the chunks folder holds %v after a second save, wanted four files", names)
	}
	for _, name := range names {
		was, old := before[name]
		now, _ := os.Stat(filepath.Join(dir, chunkFolder, name))
		if old && !now.ModTime().Equal(was.ModTime()) {
			t.Errorf("%s was written again though it did not change", name)
		}
	}
}

func TestOpeningAFolderWithNoWorldSaysSo(t *testing.T) {
	_, _, err := Open(filepath.Join(t.TempDir(), "none"))

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("got %v, wanted fs.ErrNotExist", err)
	}
}

func TestASaveThatCannotHoldThePlayerIsRefused(t *testing.T) {
	dir := t.TempDir()
	w, meta := played(t)
	meta.Pos.X = nan()

	if err := w.Save(dir, meta); err == nil {
		t.Fatal("saving a player who is nowhere should fail")
	}
	if _, err := os.Stat(filepath.Join(dir, metaName)); !errors.Is(err, fs.ErrNotExist) {
		t.Error("a world.json was written for a save that failed")
	}
}

func nan() float64 {
	var zero float64
	return zero / zero
}

func TestABrokenWorldFileIsAnErrorAndNeverAPanic(t *testing.T) {
	good := t.TempDir()
	w, meta := played(t)
	if err := w.Save(good, meta); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(good, metaName))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)

	for name, content := range map[string]string{
		"empty":              "",
		"not json":           "hello, this is not a world",
		"cut short":          text[:len(text)/2],
		"another kind":       `[1, 2, 3]`,
		"old version":        strings.Replace(text, `"version": 3`, `"version": 1`, 1),
		"no version":         strings.Replace(text, `"version": 3,`, ``, 1),
		"player nowhere":     strings.Replace(text, `"X": 702.25`, `"X": 1e999`, 1),
		"player too far":     strings.Replace(text, `"X": 702.25`, `"X": 1e15`, 1),
		"a word for yaw":     strings.Replace(text, `"yaw": 1.5`, `"yaw": "left"`, 1),
		"another mode":       strings.Replace(text, `"mode": "survival"`, `"mode": "peaceful"`, 1),
		"unknown item":       strings.Replace(text, `"item": 2,`, `"item": 999,`, 1),
		"too many":           strings.Replace(text, `"count": 12`, `"count": 65`, 1),
		"none of a kind":     strings.Replace(text, `"count": 12`, `"count": 0`, 1),
		"body nowhere":       strings.Replace(text, `"health": 15`, `"health": 1e999`, 1),
		"an empty drop":      strings.Replace(text, `"count": 1,`, `"count": 0,`, 1),
		"worn through":       strings.Replace(text, `"wear": 7`, `"wear": 9999`, 1),
		"moss ungrown":       strings.Replace(text, `"regrow": 120`, `"regrow": -5`, 1),
		"overfull pot":       strings.Replace(text, `"count": 5`, `"count": 65`, 1),
		"unburnt hearth":     strings.Replace(text, `"burn": 20.5`, `"burn": -1`, 1),
		"hearth out cold":    strings.Replace(text, `"cooked": 2.25`, `"cooked": 1e999`, 1),
		"bed in the sky":     strings.Replace(text, `"y": 103`, `"y": 500`, 1),
		"painted no face":    strings.Replace(text, `"face": 4`, `"face": 9`, 1),
		"no such picture":    strings.Replace(text, `"picture": 1`, `"picture": 99`, 1),
		"less than no trust": strings.Replace(text, `"trust": 6`, `"trust": -1`, 1),
		"a camp in the sky":  strings.Replace(text, `"y": 70`, `"y": 900`, 1),
	} {
		if content == text {
			t.Fatalf("%s: the test did not change the file", name)
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, metaName), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		if _, _, err := Open(dir); err == nil {
			t.Errorf("%s: opened without an error", name)
		}
	}
}

// A world saved by an older version of the game is whole, and the game cannot play it.
// That is told apart from a world that is damaged, so the title can say which.
func TestAWorldOfAnOlderVersionIsToldApartFromADamagedOne(t *testing.T) {
	dir := t.TempDir()
	w, meta := played(t)
	if err := w.Save(dir, meta); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, metaName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	for name, c := range map[string]struct {
		content string
		older   bool
	}{
		"version 2":  {strings.Replace(string(data), `"version": 3`, `"version": 2`, 1), true},
		"no version": {strings.Replace(string(data), `"version": 3,`, ``, 1), true},
		"version 99": {strings.Replace(string(data), `"version": 3`, `"version": 99`, 1), false},
		"damaged":    {"{", false},
	} {
		if c.content == string(data) {
			t.Fatalf("%s: the test did not change the file", name)
		}
		if err := os.WriteFile(path, []byte(c.content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Peek(dir); err == nil || errors.Is(err, ErrOlder) != c.older {
			t.Errorf("%s: Peek said %v, wanted an error that is of an older version: %v", name, err, c.older)
		}
		if _, _, err := Open(dir); err == nil || errors.Is(err, ErrOlder) != c.older {
			t.Errorf("%s: Open said %v, wanted an error that is of an older version: %v", name, err, c.older)
		}
	}
}

// A chunk file that cannot be read does not stop the game: the chunk is made again
// from the seed, the game is told, and the file is never written over, so that
// whatever can still be got out of it is not lost.
func TestADamagedChunkIsMadeAfreshAndItsFileLeftAlone(t *testing.T) {
	dir := t.TempDir()
	w := Generate(3)
	w.Set(10, 100, 10, Brick) // in the chunk at the origin, where the player is
	if err := w.Save(dir, Meta{Pos: geometry.Vector3{X: 8, Y: 90, Z: 8}}); err != nil {
		t.Fatal(err)
	}
	path := chunkPath(dir, key{0, 0})
	fresh := made(Terrain(3), 0, 0)

	for name, damage := range map[string]func([]byte) []byte{
		"empty":           func([]byte) []byte { return nil },
		"not a chunk":     func([]byte) []byte { return []byte("hello, this is not a chunk") },
		"another version": func(d []byte) []byte { return append([]byte("GCCHUNK9"), d[8:]...) },
		"cut short":       func(d []byte) []byte { return d[:len(d)/2] },
		"a flipped byte":  func(d []byte) []byte { d = bytes.Clone(d); d[len(d)/2] ^= 0xFF; return d },
		"junk at the end": func(d []byte) []byte { return append(bytes.Clone(d), d[len(d)-40:]...) },
		"too many blocks": func([]byte) []byte { return pack(make([]Block, chunkBlocks+64)) },
		"an unknown block": func([]byte) []byte {
			blocks := make([]Block, chunkBlocks)
			blocks[100] = blockCount + 3
			return pack(blocks)
		},
		"far too large": func([]byte) []byte { return make([]byte, maxChunkFile+10) },
	} {
		good, _ := os.ReadFile(path)
		bad := damage(good)
		if err := os.WriteFile(path, bad, 0o644); err != nil {
			t.Fatal(err)
		}

		got, at, err := Open(dir)
		if err != nil {
			t.Fatalf("%s: the world did not open: %v", name, err)
		}
		if got.Fault() == nil {
			t.Errorf("%s: the game was not told the chunk could not be read", name)
		}
		if h := held(got, 0, 0); !bytes.Equal(asBytes(h), asBytes(fresh)) {
			t.Errorf("%s: the chunk is not the one the seed makes", name)
		}

		got.Set(3, 110, 3, Brick) // build in it, and save
		if err := got.Save(dir, at); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if after, _ := os.ReadFile(path); !bytes.Equal(after, bad) {
			t.Errorf("%s: the damaged file was written over", name)
		}

		if err := os.WriteFile(path, good, 0o644); err != nil { // put it right for the next case
			t.Fatal(err)
		}
	}
}

func asBytes(blocks []Block) []byte {
	out := make([]byte, len(blocks))
	for i, b := range blocks {
		out[i] = byte(b)
	}
	return out
}

func TestAWriteLeavesNoOtherFilesBehind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "thing")

	for range 2 { // the second time over a file that is there
		if err := writeFile(path, []byte("hello")); err != nil {
			t.Fatal(err)
		}
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "thing" {
		t.Errorf("the folder holds %v, wanted only the file", entries)
	}
}

// A new world saved where another was takes its place whole. Chunks of the old world
// left in the folder would be read as the new one's own.
func TestAWorldSavedOverAnotherTakesItsPlace(t *testing.T) {
	dir := t.TempDir()
	old, meta := played(t)
	if err := old.Save(dir, meta); err != nil {
		t.Fatal(err)
	}
	stray := filepath.Join(dir, chunkFolder, "notes.txt")
	if err := os.WriteFile(stray, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}

	fresh := Generate(9)
	fresh.Set(40, 105, 40, Brick)
	if err := fresh.Save(dir, Meta{}); err != nil {
		t.Fatal(err)
	}

	got, back, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if back.Seed != 9 {
		t.Fatalf("the folder holds seed %d, wanted the new world's 9", back.Seed)
	}
	if got.At(10, 100, 10) != Air || got.At(-20, 101, 7) != Air {
		t.Error("what was built in the old world is in the new one")
	}
	if got.At(40, 105, 40) != Brick {
		t.Error("what was built in the new world is not there")
	}
	if names := chunkFiles(t, dir); len(names) != 2 {
		t.Errorf("the chunks folder holds %v, wanted the new world's one chunk and the stray file", names)
	}
	if data, _ := os.ReadFile(stray); string(data) != "mine" {
		t.Error("a file that is not a chunk was removed")
	}
}

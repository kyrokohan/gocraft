package world

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"testing"

	"glonk.dev/geometry"
)

// played is a save with a few things changed from what the seed makes.
func played() Save {
	w := Generate(3)
	w.Set(10, 40, 10, Brick)
	w.Set(11, 40, 10, Glass)
	w.Set(128, w.Surface(128, 128), 128, Air)
	return Save{
		World:  w,
		Pos:    geometry.Vector3{X: 12.25, Y: 41, Z: 10.5},
		Yaw:    1.5,
		Pitch:  -0.25,
		Flying: true,
		Slot:   7,
	}
}

func TestASaveComesBackAsItWas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world.sav")
	want := played()

	if err := Write(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got.World.Bytes(), want.World.Bytes()) {
		t.Error("the blocks changed")
	}
	if got.World.Seed != 3 {
		t.Errorf("seed %d, wanted 3", got.World.Seed)
	}
	if got.Pos != want.Pos || got.Yaw != want.Yaw || got.Pitch != want.Pitch || !got.Flying || got.Slot != 7 {
		t.Errorf("the player came back as %+v", got)
	}
	// A ray stops at the height, so it must not be under the highest block.
	if got.World.Height() < 41 {
		t.Errorf("the world came back %d high, with a block at layer 40", got.World.Height())
	}
}

func TestWriteMakesTheFolderAndLeavesNothingElseInIt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b")
	path := filepath.Join(dir, "world.sav")

	for range 2 { // the second time over a save that is there
		if err := Write(path, played()); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "world.sav" {
		t.Errorf("the folder holds %v, wanted only world.sav", entries)
	}
}

func TestAWriteThatFailsKeepsTheOldSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world.sav")
	if err := Write(path, played()); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)

	if err := Write(path, Save{}); err == nil { // no world to write
		t.Fatal("writing a save with no world should fail")
	}

	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Error("the old save was changed by a write that failed")
	}
}

func TestReadingASaveThatIsNotThereSaysSo(t *testing.T) {
	_, err := Read(filepath.Join(t.TempDir(), "none.sav"))

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("got %v, wanted fs.ErrNotExist", err)
	}
}

func TestABrokenSaveIsAnErrorAndNeverAPanic(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.sav")
	if err := Write(good, played()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(good)
	if err != nil {
		t.Fatal(err)
	}

	// A save whose world holds a block this game does not know.
	strange := played()
	strange.World.Set(5, 5, 5, blockCount+3)
	// A save whose player is nowhere.
	lost := played()
	lost.Pos.X = math.NaN()

	for name, write := range map[string]func(path string){
		"empty":           func(p string) { os.WriteFile(p, nil, 0o644) },
		"not a save":      func(p string) { os.WriteFile(p, []byte("hello, this is not a world"), 0o644) },
		"wrong version":   func(p string) { os.WriteFile(p, append([]byte("GOCRAFT9"), data[8:]...), 0o644) },
		"cut short":       func(p string) { os.WriteFile(p, data[:len(data)/2], 0o644) },
		"header only":     func(p string) { os.WriteFile(p, data[:40], 0o644) },
		"junk at the end": func(p string) { os.WriteFile(p, append(bytes.Clone(data), data[len(data)-64:]...), 0o644) },
		"a damaged block": func(p string) {
			damaged := bytes.Clone(data)
			damaged[len(damaged)/2] ^= 0xFF
			os.WriteFile(p, damaged, 0o644)
		},
		"too many blocks": func(p string) {
			var buf bytes.Buffer
			buf.Write(data[:len(magic)+binary.Size(header{})])
			zw := gzip.NewWriter(&buf)
			zw.Write(append(played().World.Bytes(), make([]byte, 64)...))
			zw.Close()
			os.WriteFile(p, buf.Bytes(), 0o644)
		},
		"unknown block":  func(p string) { writeUnchecked(t, p, strange) },
		"player nowhere": func(p string) { writeUnchecked(t, p, lost) },
	} {
		path := filepath.Join(dir, name+".sav")
		write(path)

		if _, err := Read(path); err == nil {
			t.Errorf("%s: read without an error", name)
		}
	}
}

// writeUnchecked writes a save that Write would refuse, to test that Read refuses it too.
func writeUnchecked(t *testing.T, path string, s Save) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := encode(f, s); err != nil {
		t.Fatal(err)
	}
}

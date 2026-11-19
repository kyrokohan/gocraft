package assets

import (
	"testing"

	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/tile"
)

func holes(pixels []uint32) int {
	n := 0
	for _, p := range pixels {
		if p>>24 == 0 {
			n++
		}
	}
	return n
}

func TestLoadReadsEveryTileInEverySize(t *testing.T) {
	set, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	for id, tex := range set.Tiles {
		for level, pixels := range tex {
			if want := Side(level) * Side(level); len(pixels) != want {
				t.Errorf("tile %d level %d has %d pixels, wanted %d", id, level, len(pixels), want)
			}
		}
	}
	if set.Title.Width() == 0 || set.Title.Height() == 0 {
		t.Error("the title picture is empty")
	}
}

func TestHolesAreKeptAndCloseUpWithDistance(t *testing.T) {
	set, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if holes(set.Tiles[tile.Leaves][0]) == 0 {
		t.Error("leaves should have holes close up")
	}
	if holes(set.Tiles[tile.Leaves][Levels-1]) != 0 {
		t.Error("leaves should be one solid pixel at the smallest size")
	}
	if holes(set.Tiles[tile.Glass][1]) == 0 {
		t.Error("glass should still be see-through one size down")
	}
	for level, pixels := range set.Tiles[tile.Stone] {
		if holes(pixels) != 0 {
			t.Errorf("stone has holes at level %d", level)
		}
	}
}

func TestHalveAveragesColorsAndIgnoresHoles(t *testing.T) {
	red, blue := uint32(0xFFFF0000), uint32(0xFF0000FF)

	got := halve([]uint32{
		red, blue, red, 0,
		0, 0, 0, 0,
		0, 0, red, red,
		0, 0, red, red,
	}, 4)

	want := []uint32{0xFF7F007F, 0, 0, red}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pixel %d is %08X, wanted %08X", i, got[i], want[i])
		}
	}
}

func TestEveryTileHasAnIconOfItsOwnPixels(t *testing.T) {
	set, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	for id, icon := range set.Icons {
		if icon.Width() != tile.Size || icon.Height() != tile.Size {
			t.Fatalf("icon %d is %d by %d, wanted %d by %d", id, icon.Width(), icon.Height(), tile.Size, tile.Size)
		}
		for _, at := range [][2]int{{0, 0}, {5, 9}, {15, 15}} {
			want := set.Tiles[id][0][at[1]*tile.Size+at[0]]
			got := icon.At(at[0], at[1])
			if hole := want>>24 == 0; hole != got.IsDefault() {
				t.Errorf("icon %d pixel %v: hole %v in the tile, %v in the icon", id, at, hole, got.IsDefault())
				continue
			}
			if r, g, b := got.RGB(); want>>24 != 0 && (uint32(r) != want>>16&0xFF || uint32(g) != want>>8&0xFF || uint32(b) != want&0xFF) {
				t.Errorf("icon %d pixel %v is %v, the tile has %06X", id, at, got, want&0xFFFFFF)
			}
		}
	}
}

func TestEverySoundIsRead(t *testing.T) {
	set, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for n, s := range set.Sounds {
		peak := float32(0)
		for _, v := range s {
			peak = max(peak, v, -v)
		}
		if len(s) < sound.Rate/20 || peak < 0.2 || peak > 1 {
			t.Errorf("%s: %d samples, peaking at %.3f", sound.Name(n).File(), len(s), peak)
		}
	}
	if len(set.Sounds[sound.Wind]) != 6*sound.Rate {
		t.Errorf("the wind is %d samples, wanted six seconds", len(set.Sounds[sound.Wind]))
	}
}

func TestAFileThatIsNotASoundOfTheGameIsRefused(t *testing.T) {
	if _, err := readWAV("img/title.png"); err == nil {
		t.Error("a picture was read as a sound")
	}
}

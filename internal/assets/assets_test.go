package assets

import (
	"testing"

	"github.com/kyrokohan/gocraft/internal/tile"
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

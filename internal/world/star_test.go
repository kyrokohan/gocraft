package world

import "testing"

func TestAStarFallsNearThePlayerAboutOneNightInThreeByTheSeed(t *testing.T) {
	for seed := int64(1); seed <= 6; seed++ {
		w := Generate(seed)
		falls := 0
		for night := 1; night <= 600; night++ {
			part, ok := w.StarFall(night)
			if again, also := w.StarFall(night); again != part || also != ok {
				t.Fatalf("seed %d: night %d was asked twice, and the answers differ", seed, night)
			}
			if part < 0 || part >= 1 {
				t.Fatalf("seed %d: on night %d a star falls at part %v of the night, wanted 0 to 1", seed, night, part)
			}
			if ok {
				falls++
			}
		}
		if falls < 160 || falls > 240 {
			t.Errorf("seed %d: a star falls on %d of 600 nights, wanted about one in three", seed, falls)
		}
	}
	a, b := Generate(1), Generate(2)
	same := 0
	for night := 1; night <= 200; night++ {
		_, x := a.StarFall(night)
		if _, y := b.StarFall(night); x == y {
			same++
		}
	}
	if same > 150 {
		t.Errorf("two seeds agree on %d of 200 nights: wanted each its own", same)
	}
	if _, ok := New().StarFall(3); ok {
		t.Error("a star falls in a world that is not made from a seed")
	}
}

func TestACoreOfSkyIronIsTooHotToBreakAndThenNeedsAFlintPick(t *testing.T) {
	hot, dark := HotCore.Info(), SkyIronCore.Info()
	if !hot.Fixed || hot.Glow < 10 || !hot.Solid {
		t.Errorf("a hot core is %+v: wanted it solid, glowing, and not to be broken", hot)
	}
	if item, n := HotCore.Drop(); n != 0 {
		t.Errorf("a hot core leaves %d of %v, wanted nothing", n, item)
	}
	if dark.Fixed || dark.Glow != 0 || !dark.Solid {
		t.Errorf("a core gone dark is %+v: wanted it solid, dark, and to be broken", dark)
	}
	for hand, want := range map[Item]bool{NoItem: false, DiggingStick: false, HandAxe: false, FlintPick: true, ObsidianPick: true} {
		if _, yields := SkyIronCore.DigWith(hand); yields != want {
			t.Errorf("a core of sky iron dug with %q leaves something: %v, wanted %v", hand.Name(), yields, want)
		}
	}
	slow, _ := SkyIronCore.DigWith(FlintPick)
	if stone, _ := Stone.DigWith(FlintPick); slow < 3 || slow < 2*stone {
		t.Errorf("a flint pick takes %.1f s to break out a core of sky iron and %.1f s for stone: wanted it slow", slow, stone)
	}
	counts := map[int]int{}
	for x := range 200 {
		item, n := SkyIronCore.DropAt(x, 7, -x*3)
		if item != SkyIron || n < 2 || n > 4 {
			t.Fatalf("a core of sky iron leaves %d of %q, wanted two to four lumps of sky iron", n, item.Name())
		}
		counts[n]++
	}
	if len(counts) != 3 {
		t.Errorf("200 cores left lumps in these numbers: %v, wanted twos, threes and fours", counts)
	}
	if SkyIron.Name() != "sky iron" || SkyIron.StackSize() != StackSize || !SkyIron.Known() {
		t.Errorf("sky iron is %q, %d to a slot", SkyIron.Name(), SkyIron.StackSize())
	}
}

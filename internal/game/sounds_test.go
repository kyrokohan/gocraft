package game

import (
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"

	"glonk.dev/engine"
	"glonk.dev/geometry"

	"github.com/kyrokohan/gocraft/internal/sound"
	"github.com/kyrokohan/gocraft/internal/world"
)

// hush gives the game a new mixer, playing nothing and listening where the player is, so
// that a test hears only what comes after.
func hush(g *Game) {
	g.ears = sound.NewMixer(g.set.Sounds)
	g.listen()
}

// heard returns how many times each sound has been started since the mixer was hushed.
func heard(g *Game) map[sound.Name]int {
	n := map[sound.Name]int{}
	for _, s := range g.ears.Playing() {
		n[s]++
	}
	return n
}

func TestABlockSoundsAsWhatItIsMadeOf(t *testing.T) {
	for b, want := range map[world.Block]sound.Material{
		world.Grass: sound.Earth, world.Clay: sound.Earth,
		world.Stone: sound.Stone, world.CoalOre: sound.Stone, world.Brick: sound.Stone, world.Kiln: sound.Stone,
		world.Gravel: sound.Sand, world.Sand: sound.Sand,
		world.PineLog: sound.Wood, world.Crate: sound.Wood, world.Workbench: sound.Wood,
		world.PineNeedles: sound.Plant, world.TallGrass: sound.Plant, world.Torch: sound.Plant, world.Bedroll: sound.Plant,
		world.Ice: sound.Glass, world.Glass: sound.Glass,
		world.Snow: sound.Snow,
	} {
		if got := materialOf(b); got != want {
			t.Errorf("%s sounds as material %d, wanted %d", b.Info().Name, got, want)
		}
	}
}

func TestBreakingAndPlacingSoundAsTheBlock(t *testing.T) {
	g := lookingDown(t)
	hush(g)
	hold(t, g, 1, 'q')
	if heard(g)[sound.Break(sound.Stone)] != 1 {
		t.Errorf("breaking stone: heard %v", g.ears.Playing())
	}

	g.items[g.selected] = world.Stack{Item: world.Planks.Item(), Count: 1}
	hush(g)
	hold(t, g, 1, 'e')
	if g.world.At(20, 5, 21) != world.Planks || heard(g)[sound.Place(sound.Wood)] != 1 {
		t.Errorf("placing planks: heard %v", g.ears.Playing())
	}
}

func TestDiggingSoundsInStrokes(t *testing.T) {
	g := digging(t, world.Stone)
	hush(g)
	hold(t, g, seconds(1), 'q')
	if n := heard(g)[sound.Dig(sound.Stone)]; n < 4 || n > 5 {
		t.Errorf("a second of digging stone: %d strokes, wanted one at the start and one every %v s", n, digStroke)
	}
}

func TestWalkingSoundsStepsOnTheGroundAndFlyingDoesNot(t *testing.T) {
	g := onFloor(t)
	hush(g)
	hold(t, g, seconds(2), 'w')
	if n := heard(g)[sound.Step(sound.Stone)]; n < 3 {
		t.Errorf("two seconds of walking on stone: %d steps", n)
	}

	g.player = *standing()
	g.player.Flying = true
	hush(g)
	hold(t, g, seconds(2), 'w')
	if n := heard(g)[sound.Step(sound.Stone)]; n != 0 {
		t.Errorf("flying: %d steps", n)
	}
}

func TestTheBodyIsHeard(t *testing.T) {
	g := inSurvival(t)
	hush(g)
	g.hurt(1)
	g.food = 1
	g.items[g.selected] = world.Stack{Item: world.Berries, Count: 1}
	g.eat()
	g.drops = []drop{{Drop: world.Drop{Stack: world.Stack{Item: world.Dirt.Item(), Count: 1}, Pos: g.player.Pos, Age: pickWait}}}
	hold(t, g, 1)
	if h := heard(g); h[sound.Hurt] != 1 || h[sound.Eat] != 1 || h[sound.Pickup] != 1 {
		t.Errorf("hurt, eating and a pickup: heard %v", g.ears.Playing())
	}
}

func TestCreaturesAndKilnsAreHeard(t *testing.T) {
	g := inSurvival(t)
	withMossback(t, g)
	hush(g)
	g.strike(g.prey)
	g.gloams = []gloam{{pos: geometry.Vector3{X: 22, Y: 6, Z: 20}, health: gloamHealth}}
	g.strike(prey{gloam: true})
	if h := heard(g); h[sound.Cry] != 1 || h[sound.Hiss] != 1 {
		t.Errorf("a mossback and a gloam hit: heard %v", g.ears.Playing())
	}

	// Each is as sure to be heard as can be in as long as it is quiet for, as a rule.
	g.kilns = map[world.Cell]*world.KilnStore{{X: 18, Y: 6, Z: 20}: {Burn: 5}}
	hush(g)
	g.worldSounds(callEvery)
	if h := heard(g); h[sound.Call] != 1 || h[sound.Moan] != 1 || h[sound.Crackle] != 1 {
		t.Errorf("a mossback, a gloam and a kiln left be: heard %v", g.ears.Playing())
	}
}

func TestTheWindBlowsOutsideAndCaveAirIsHeardInTheDark(t *testing.T) {
	g := onFloor(t)
	if wind, cave := g.ambience(); !near(wind, 0.4) || cave != 0 {
		t.Errorf("outside: wind %.2f, cave %.2f", wind, cave)
	}
	for x := 18; x <= 22; x++ { // a stone box round the player's head, and over it
		for z := 18; z <= 22; z++ {
			for y := 6; y <= 9; y++ {
				if x == 18 || x == 22 || z == 18 || z == 22 || y == 9 {
					g.world.Set(x, y, z, world.Stone)
				}
			}
		}
	}
	if wind, cave := g.ambience(); wind != 0 || !near(cave, 0.6) {
		t.Errorf("shut in: wind %.2f, cave %.2f", wind, cave)
	}
	press(g, engine.KeyEscape)
	if wind, cave := g.ambience(); wind != 0 || cave != 0 {
		t.Errorf("paused: wind %.2f, cave %.2f", wind, cave)
	}
}

func TestTheVolumeSettingIsHowLoudTheMixerIs(t *testing.T) {
	g := newGame(t)
	loudest := func() float64 {
		hush(g)
		g.ears.SetVolume(g.settings.volume())
		g.ears.PlayNear(sound.Pickup, 1)
		p := make([]byte, 8*sound.Rate/4)
		g.ears.Read(p)
		top := 0.0
		for i := 0; i < len(p); i += 4 {
			top = max(top, math.Abs(float64(math.Float32frombits(binary.LittleEndian.Uint32(p[i:])))))
		}
		return top
	}
	full := loudest()
	for g.settings.Volume != 0 {
		g.changeSetting(3, -1)
	}
	if full < 0.1 || loudest() != 0 {
		t.Errorf("a pickup at %d%% peaks at %.3f, and at 0%% at %.3f", defaultSettings.Volume, full, loudest())
	}
}

func TestTheTitleSaysWhenThereIsNoSound(t *testing.T) {
	g := newGame(t)
	g.NoSound(errors.New("no device"))
	if text, _ := drawn(t, g); !strings.Contains(text, "no sound: no device") {
		t.Errorf("the title does not say there is no sound:\n%s", text)
	}
}

// loudness plays the game for a number of frames with keys held, reading what the mixer
// makes as a speaker would, and returns how loud it was on the whole and at its loudest,
// in decibels of full.
func loudness(t *testing.T, g *Game, frames int, keys ...engine.Key) (rms, peak float64) {
	t.Helper()
	kb := engine.NewKeyboard()
	kb.Press(keys...)
	p := make([]byte, 8*sound.Rate/30)
	var sum float64
	n := 0
	for range frames {
		if err := g.Update(frame, kb.Input()); err != nil {
			t.Fatal(err)
		}
		kb.EndFrame()
		g.ears.Read(p)
		for i := 0; i < len(p); i += 4 {
			s := float64(math.Float32frombits(binary.LittleEndian.Uint32(p[i:])))
			sum, peak, n = sum+s*s, max(peak, math.Abs(s)), n+1
		}
	}
	return 20 * math.Log10(math.Sqrt(sum/float64(n))), 20 * math.Log10(peak)
}

func TestPlayIsLoudEnoughAndStepsStandOutOfTheWind(t *testing.T) {
	still, stillPeak := loudness(t, onFloor(t), seconds(3))
	_, walkPeak := loudness(t, onFloor(t), seconds(3), 'w')
	_, breakPeak := loudness(t, lookingDown(t), seconds(1), 'q')
	if still < -30 || still > -18 {
		t.Errorf("the wind outside is at %.1f dB, wanted it between -30 and -18", still)
	}
	if walkPeak < stillPeak+4 {
		t.Errorf("steps peak at %.1f dB over wind that peaks at %.1f: lost in it", walkPeak, stillPeak)
	}
	if breakPeak > 0 || breakPeak < -6 {
		t.Errorf("a block breaking peaks at %.1f dB, wanted it loud and short of full", breakPeak)
	}
}

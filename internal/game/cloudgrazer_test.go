package game

import (
	"math"
	"path/filepath"
	"slices"
	"testing"

	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/sound"
	"github.com/kyrokohan/skyspear/internal/world"
)

// strider is a cloudgrazer on the plain at x, z that walks on, straight along +z.
func strider(x, z float64) beast {
	c := giant(render.Cloudgrazer, x, z)
	c.walking = true
	return c
}

func TestCloudgrazersGoInPairsOnOpenLandAndTowerOverEverything(t *testing.T) {
	b := breeds[render.Cloudgrazer]
	found := 0
	for seed := int64(1); seed <= 10; seed++ {
		w := realWorld(seed)
		all := only(newFauna(w, w.Spawn()).all, render.Cloudgrazer)
		if len(all) != 0 && len(all) != 2 {
			t.Errorf("seed %d: %d cloudgrazers, wanted a pair or none", seed, len(all))
		}
		for _, c := range all {
			found++
			x, z := int(math.Floor(c.Pos.X)), int(math.Floor(c.Pos.Z))
			if land := world.BiomeAt(seed, x, z); !slices.Contains(b.lands, land) {
				t.Errorf("seed %d: a cloudgrazer stands in land %d", seed, land)
			}
			if c.young || c.herd == 0 {
				t.Errorf("seed %d: a cloudgrazer is young %v, of herd %d: wanted a grown one of a pair", seed, c.young, c.herd)
			}
		}
	}
	if found == 0 {
		t.Error("no cloudgrazer was found in ten worlds")
	}
	_, front := render.Span(render.Cloudgrazer, false)
	back, _ := render.Span(render.Cloudgrazer, false)
	if _, high := render.Size(render.Bugler, false); front-back < 5*high {
		t.Errorf("a cloudgrazer is %.1f long, wanted many times a bugler's height of %.1f", front-back, high)
	}
}

func TestACloudgrazerIsNotHurtAndDoesNotRun(t *testing.T) {
	g := among(t, 5.5, giant(render.Cloudgrazer, 0.5, 20.5))
	at := g.fauna.all[0].Pos
	for range 5 {
		g.wound(0, world.HardSpear.Damage()+thrownMore, 1, false)
	}
	hold(t, g, seconds(3))
	c := g.fauna.all[0]
	if c.dead || c.flee > 0 || c.hurt > 0 || c.push.Length() > 0 || c.Pos.Distance(at) > 0.01 {
		t.Errorf("a cloudgrazer hit five times: dead %v, running %v, hurt %v, pushed %v, moved %.2f", c.dead, c.flee > 0, c.hurt > 0, c.push, c.Pos.Distance(at))
	}

	// A spear thrown at it meets it and lies there, and it takes no notice.
	g.player.Pitch = 0.3
	holding(g, world.Spear, 1)
	hold(t, g, 1, 'r')
	flies(t, g)
	hold(t, g, seconds(1))
	if len(g.drops) != 1 || g.fauna.all[0].flee > 0 {
		t.Errorf("a spear thrown at a cloudgrazer: %d lying, and it runs: %v", len(g.drops), g.fauna.all[0].flee > 0)
	}

	// Its mate does not run either when it is hit.
	one, two := giant(render.Cloudgrazer, 0.5, 20.5), giant(render.Cloudgrazer, 8.5, 24.5)
	one.herd, two.herd = 1, 1
	g = among(t, 5.5, one, two)
	g.wound(0, 9, 1, false)
	if g.fauna.all[1].flee > 0 || g.fauna.alarms[1].left > 0 && g.fauna.all[1].walking {
		t.Error("the mate of a cloudgrazer that was hit runs")
	}
}

func TestACloudgrazersFootHurtsAPlayerUnderItAndNotOneBesideIt(t *testing.T) {
	b := breeds[render.Cloudgrazer]
	f := &fauna{all: []beast{giant(render.Cloudgrazer, 0.5, 20.5)}}
	c := &f.all[0]
	turned := mesh.Camera{Yaw: c.Yaw}
	for pair := 1; pair <= 2; pair++ {
		side := float64(3 - 2*pair)
		foot := c.Pos.Add(turned.Right().Scale(side * b.foot[0])).Add(turned.Forward().Scale(b.foot[1]))
		for _, tc := range []struct {
			at   geometry.Vector3
			hurt bool
			what string
		}{
			{foot, true, "under the foot"},
			{foot.Add(geometry.Vector3{X: 0.6}), true, "by the edge of the foot"},
			{foot.Add(geometry.Vector3{X: side * 2.5}), false, "beside it"},
			{c.Pos, false, "under its belly"},
			{foot.Add(geometry.Vector3{Y: 5}), false, "on its back"},
		} {
			f.bitten, f.shove, f.treads = 0, geometry.Vector3{}, nil
			c.trod = pair
			f.tread(0, tc.at)
			if got := f.bitten == b.stamp; got != tc.hurt || !tc.hurt && f.bitten != 0 {
				t.Errorf("pair %d, a player %s: %v points, wanted hurt %v", pair, tc.what, f.bitten, tc.hurt)
			}
			if tc.hurt && f.shove.Length() < 1 {
				t.Errorf("pair %d, a player %s was thrown %.2f, wanted a block or more", pair, tc.what, f.shove.Length())
			}
			if len(f.treads) != 1 || f.treads[0].name != sound.Thud {
				t.Errorf("the step left %+v to be heard, wanted one thud", f.treads)
			}
		}
	}
	if b.stamp < 6 {
		t.Errorf("its foot takes %v points, wanted a blow to fear", b.stamp)
	}

	// In play: a player it walks over is hurt, and thrown clear.
	g := among(t, 30.5, strider(0.5-b.foot[0], 12.5))
	for range seconds(40) {
		g.fauna.all[0].walking, g.fauna.all[0].timer, g.fauna.all[0].Yaw = true, math.Inf(1), 0
		g.player.Pos = geometry.Vector3{X: 0.5, Y: 6, Z: g.fauna.all[0].Pos.Z + b.foot[1] + 0.5} // just ahead of a front foot
		if hold(t, g, 1); g.health < maxHealth {
			break
		}
	}
	if g.health > maxHealth-b.stamp {
		t.Errorf("a player kept under a cloudgrazer's front foot for forty seconds has %v health", g.health)
	}
}

func TestTheGroundShakesMoreTheNearerTheStepAndNotAtAllFromFar(t *testing.T) {
	b := breeds[render.Cloudgrazer]
	shake := func(away float64, still bool) (peak, tilt float64, thuds int) {
		g := among(t, 0.5, strider(30.5, 0.5+away)) // well to the side: its feet come nowhere near
		g.player.Pos.X = 0.5
		g.fauna.all[0].Pos.X = 0.5 + away
		g.fauna.all[0].Pos.Z = -10.5
		g.settings.NoShake = still
		level := g.eyes()
		for range seconds(12) {
			g.fauna.all[0].walking, g.fauna.all[0].timer, g.fauna.all[0].Yaw = true, math.Inf(1), 0
			hold(t, g, 1)
			cam, _ := g.viewed()
			peak, tilt = max(peak, g.quake), max(tilt, math.Abs(cam.Pitch-level.Pitch))
		}
		return peak, tilt, heard(g)[sound.Thud]
	}
	near5, tilt5, thuds := shake(5, false)
	near25, tilt25, _ := shake(25, false)
	far, tiltFar, thudsFar := shake(b.quake+10, false)
	if thuds < 3 || thudsFar < 3 {
		t.Errorf("twelve seconds of its walk were %d and %d thuds, wanted a step every few seconds, heard from %v blocks too", thuds, thudsFar, b.quake+10)
	}
	if !(near5 > near25 && near25 > 0 && far == 0) {
		t.Errorf("the ground shakes %.2f at 5 blocks, %.2f at 25 and %.2f at %v: wanted more the nearer, and none past %v", near5, near25, far, b.quake+10, b.quake)
	}
	if !(tilt5 > tilt25 && tilt25 > 0 && tiltFar == 0) || tilt5 > shakeTilt+1e-9 {
		t.Errorf("the view tips %.4f at 5 blocks, %.4f at 25 and %.4f from far: wanted more the nearer, never past %.3f", tilt5, tilt25, tiltFar, shakeTilt)
	}
	if peak, tilt, _ := shake(5, true); peak == 0 || tilt != 0 {
		t.Errorf("with the shaking turned off the ground shakes %.2f and the view tips %.4f: wanted the view still", peak, tilt)
	}

	// A shake dies away.
	g := among(t, 0.5)
	g.quake = 1
	hold(t, g, seconds(3))
	if g.quake != 0 {
		t.Errorf("three seconds after a step the ground still shakes: %.3f", g.quake)
	}
}

func TestAGiantsStepIsHeardFromEightyBlocksAndABuglersIsNot(t *testing.T) {
	g := among(t, -35.5, strider(0.5, 44.5)) // 80 blocks off, and it walks across, not away
	for range seconds(8) {
		g.fauna.all[0].walking, g.fauna.all[0].timer, g.fauna.all[0].Yaw = true, math.Inf(1), math.Pi/2
		hold(t, g, 1)
	}
	if d := g.fauna.all[0].Pos.Distance(g.player.Pos); d < 80 {
		t.Fatalf("the cloudgrazer is %.1f off, wanted 80 or more", d)
	}
	if heard(g)[sound.Thud] == 0 {
		t.Errorf("no step of a cloudgrazer 80 blocks off was heard: %v", g.ears.Playing())
	}
	if breeds[render.Cloudgrazer].carries < 80 {
		t.Errorf("its step carries %v blocks", breeds[render.Cloudgrazer].carries)
	}

	g = among(t, 0.5, giant(render.Bugler, 0.5, 10.5))
	for range seconds(8) {
		g.fauna.all[0].walking, g.fauna.all[0].timer, g.fauna.all[0].Yaw = true, math.Inf(1), 0
		hold(t, g, 1)
	}
	if heard(g)[sound.Thud] != 0 || g.quake != 0 {
		t.Errorf("a bugler's steps thud %d times and shake the ground %.2f", heard(g)[sound.Thud], g.quake)
	}
}

func TestACloudgrazerWalksThroughLeavesAndTurnsAtATrunk(t *testing.T) {
	walks := func(kind render.Kind, wall world.Block) float64 {
		w := plain()
		for x := -12; x <= 12; x++ {
			for y := 6; y < 22; y++ {
				w.Set(x, y, 20, wall)
				w.Set(x, y, 21, wall)
			}
		}
		c := giant(kind, 0.5, 9.5)
		for range seconds(30) {
			c.walking, c.timer, c.Yaw = true, math.Inf(1), 0
			c.Step(w, frame)
		}
		return c.Pos.Z
	}
	if z := walks(render.Cloudgrazer, world.Leaves); z < 25 {
		t.Errorf("a cloudgrazer at a wall of leaves got to z=%.1f, wanted it through the wall at 20", z)
	}
	if z := walks(render.Cloudgrazer, world.Log); z > 19 {
		t.Errorf("a cloudgrazer at a wall of trunks got to z=%.1f, wanted it stopped at 20", z)
	}
	if z := walks(render.Bugler, world.Leaves); z > 19.5 {
		t.Errorf("a bugler walked through leaves, to z=%.1f", z)
	}
}

func TestTheShakingIsASettingThatIsKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := defaultSettings
	if s.NoShake {
		t.Fatal("a new player starts with the shaking off")
	}
	lines := s.lines()
	if len(lines) != settingLines+1 || lines[4] == "" {
		t.Fatalf("the settings have the lines %q, wanted %d and Back", lines, settingLines)
	}
	s.step(4, 1)
	if !s.NoShake || s.lines()[4] == lines[4] {
		t.Errorf("stepping the line turned the shaking off: %v, and the line reads %q", s.NoShake, s.lines()[4])
	}
	if err := s.save(path); err != nil {
		t.Fatal(err)
	}
	if kept := loadSettings(path); !kept.NoShake {
		t.Error("the setting was not kept")
	}
	s.step(4, -1)
	if s.NoShake {
		t.Error("stepping it back did not turn the shaking on again")
	}
}

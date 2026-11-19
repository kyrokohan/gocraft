package game

import "testing"

func TestCCrouchesAndStandsAgain(t *testing.T) {
	g := onFloor(t)
	hold(t, g, 3, 'c') // held for three frames: only its going down counts
	if !g.player.Crouched {
		t.Fatal("C did not crouch the player")
	}
	hold(t, g, 1, 'C') // a terminal may send a capital
	if g.player.Crouched {
		t.Error("C again did not stand the player up")
	}
}

func TestCrouchedThePlayerWalksSlowerAndSeesFromLower(t *testing.T) {
	stood := onFloor(t)
	hold(t, stood, seconds(1), 'w')

	crouched := onFloor(t)
	hold(t, crouched, 1, 'c')
	hold(t, crouched, seconds(1), 'w')

	far, near := stood.player.Pos.Z-standing().Pos.Z, crouched.player.Pos.Z-standing().Pos.Z
	if near < 0.25*far || near > 0.4*far {
		t.Errorf("crouched the player walked %.2f blocks in a second and standing %.2f, wanted about a third", near, far)
	}
	if lower := stood.eyes().Position.Y - crouched.eyes().Position.Y; lower < 0.3 || lower > 0.6 {
		t.Errorf("crouched the eyes are %.2f lower, wanted between 0.3 and 0.6", lower)
	}
}

func TestAJumpFlyingAndWaterStandTheCrouchedPlayerUp(t *testing.T) {
	g := onFloor(t)
	hold(t, g, 1, 'c')
	hold(t, g, 1, ' ')
	if g.player.Crouched || g.player.Pos.Y <= standing().Pos.Y {
		t.Errorf("a jump left the player crouched (%v) or on the ground at %v", g.player.Crouched, g.player.Pos.Y)
	}

	p := standing()
	p.Crouched = true
	run(p, floor(), controls{toggleFly: true}, frame)
	if p.Crouched {
		t.Error("flying left the player crouched")
	}

	w := floor()
	p = standing()
	w.Set(20, 6, 20, 10) // world.Water, where the player stands
	p.Crouched = true
	run(p, w, controls{}, frame)
	if p.Crouched {
		t.Error("the player is crouched in water")
	}
}

func TestCrouchedStepsAreQuieter(t *testing.T) {
	_, walk := loudness(t, onFloor(t), seconds(3), 'w')

	g := onFloor(t)
	hold(t, g, 1, 'c')
	_, sneak := loudness(t, g, seconds(8), 'w') // long enough for several slow steps
	if sneak > walk-3 {
		t.Errorf("crouched steps peak at %.1f dB and walking ones at %.1f, wanted them 3 or more quieter", sneak, walk)
	}
}

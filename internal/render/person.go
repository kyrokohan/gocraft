package render

import (
	"math"

	"glonk.dev/bitmap"
	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/world"
)

// The parts of a cave person, in blocks, with its feet at 0 and its front toward +z. It
// stands 1.8 high: as high as the box that blocks stop.
const (
	limbLength  = 0.76 // a leg, from the hip to the ground
	limbWidth   = 0.2
	limbAside   = 0.13 // from the middle to a leg
	trunkWidth  = 0.5
	trunkHeight = 0.62
	trunkDepth  = 0.26
	armLength   = 0.6
	armWidth    = 0.16
	armAside    = trunkWidth/2 + armWidth/2 // from the middle to an arm
	shoulder    = trunkHeight - 0.02        // above the hip
	skullSize   = 0.42

	stride      = 0.6  // how far a leg or an arm swings each way as it walks, in radians
	carry       = 0.5  // how far forward the arm that holds something is held
	strike      = 1.1  // and how much farther a swing lifts it
	crouchSplay = 0.85 // how far a crouched person's legs are set, one ahead and one behind
	crouchLean  = 0.5  // and how far its body leans forward
	seatSplay   = 1.05 // how far a seated person's legs are set out to its sides
	seatAhead   = 0.45 // and forward
	maxTilt     = 0.7  // how far the head tips up or down
)

// The tiles of one look of the cave person's picture, from the left, as cmd/genassets
// draws them.
const (
	personFace = iota
	personHair
	personHideFront
	personHideBack
	personHideSide
	personArm
	personLeg
)

// SeatHeight is how high a seated cave person's seat is over the point it is drawn at:
// as high as its hip is when it stands, so that it sees from as high over its seat.
const SeatHeight = limbLength

// Person is where a cave person is and how it stands: all that drawing it needs.
type Person struct {
	// Pos is the point between its feet, and Yaw the way it faces, as a camera's Yaw is.
	Pos geometry.Vector3
	Yaw float64

	// Look is which of the looks it has, from 0.
	Look int

	// Tilt is how far its head is tipped up, in radians, as a camera's Pitch is.
	Tilt float64

	// Stride is how far through its steps it is, in radians: its legs and arms swing as
	// the sine of it. Swing is how far through a swing of its right arm it is, from 0
	// to 1, as it digs, hits or eats.
	Stride, Swing float64

	// Crouched is true while it goes bent low, Seated while it rides, astride what it
	// sits on, and Hurt while it shows a hit.
	Crouched, Seated, Hurt bool

	// Light is how bright the light it stands in is, from 0 to 1, as for a Mossback.
	Light float64

	// Held is what is in its right hand, if anything.
	Held world.Item
}

// body is the boxes of one look.
type body struct {
	head, trunk, arm, leg mesh.Mesh
}

// People draws cave people. Like the creatures they are glonk meshes: boxes with a
// picture on their faces, drawn into the canvas after the blocks.
type People struct {
	looks  [assets.PersonLooks]body
	hurt   *bitmap.Bitmap
	lights [2]mesh.Light
}

// NewPeople builds the boxes a cave person is made of, once for each look.
func NewPeople(set *assets.Set) *People {
	p := &People{hurt: reddened(set.Person)}
	skin := mesh.Material{Color: engine.RGB(198, 142, 100), Texture: set.Person}
	const n = assets.PersonTiles * assets.PersonLooks
	for look := range p.looks {
		// Each box says which tile goes on which of its faces, in the order of world.Face.
		tile := func(t int) int { return look*assets.PersonTiles + t }
		head := [6]int{tile(personHair), tile(personHair), tile(personHair), tile(personHair), tile(personHair), tile(personFace)}
		trunk := [6]int{tile(personHideSide), tile(personHideSide), tile(personHideBack), tile(personHideSide), tile(personHideBack), tile(personHideFront)}
		arm := [6]int{tile(personArm), tile(personArm), tile(personArm), tile(personArm), tile(personArm), tile(personArm)}
		leg := [6]int{tile(personLeg), tile(personLeg), tile(personLeg), tile(personLeg), tile(personLeg), tile(personLeg)}
		p.looks[look] = body{
			// The head turns about the neck and the trunk about the hip: their origins.
			head:  box(-skullSize/2, 0, -skullSize/2, skullSize, skullSize, skullSize, head, n, skin),
			trunk: box(-trunkWidth/2, 0, -trunkDepth/2, trunkWidth, trunkHeight, trunkDepth, trunk, n, skin),
			// An arm hangs from the shoulder and a leg from the hip.
			arm: box(-armWidth/2, -armLength, -armWidth/2, armWidth, armLength, armWidth, arm, n, skin),
			leg: box(-limbWidth/2, -limbLength, -limbWidth/2, limbWidth, limbLength, limbWidth, leg, n, skin),
		}
	}
	return p
}

// Draw draws cave people as the camera sees them, over what is in the canvas. Call it
// after the blocks, and before anything that is seen through. held makes the picture of
// what is in a hand.
func (p *People) Draw(c *engine.Canvas, cam mesh.Camera, all []Person, sky *Sky, held *Drops) {
	for _, one := range all {
		b := &p.looks[min(max(one.Look, 0), len(p.looks)-1)]
		p.lights = lightsIn(one.Light, sky)
		turned := mesh.Camera{Yaw: one.Yaw}
		ahead, aside := turned.Forward(), turned.Right()

		// spot is a point a way to the person's right, up, and ahead of the point
		// between its feet.
		spot := func(right, up, front float64) geometry.Vector3 {
			at := one.Pos.Add(aside.Scale(right)).Add(ahead.Scale(front))
			at.Y += up
			return at
		}
		// put draws a part with its origin there, tipped by pitch and leaned by roll: a
		// positive pitch tips the top of the part back, and so swings a hanging limb
		// forward, and a positive roll swings it out to the left.
		put := func(part mesh.Mesh, right, up, front, pitch, roll float64) {
			part.Transform = mesh.Transform{Position: spot(right, up, front), Yaw: one.Yaw, Pitch: pitch, Roll: roll}
			if one.Hurt {
				part.Material.Texture = p.hurt
			}
			part.DrawShadedPixels(c, cam, p.lights[:]...)
		}

		// A standing person's hip is a leg above the ground and its body upright. A
		// crouched one sets its legs apart, which lowers the hip, and leans forward. A
		// seated one has its legs out to its sides and forward, and still.
		hip, lean, apart := limbLength, 0.0, 0.0
		forth, out := 0.0, 0.0
		step := stride * math.Sin(one.Stride)
		switch {
		case one.Seated:
			forth, out, step = seatAhead, seatSplay, 0
		case one.Crouched:
			hip, lean, apart = limbLength*math.Cos(crouchSplay), crouchLean, crouchSplay
			step /= 3
		}
		// A point of the leaning trunk, a height above the hip.
		upOf := func(h float64) float64 { return hip + h*math.Cos(lean) }
		aheadOf := func(h float64) float64 { return h * math.Sin(lean) }

		put(b.leg, -limbAside, hip, 0, forth+apart+step, out)
		put(b.leg, limbAside, hip, 0, forth-apart-step, -out)
		put(b.trunk, 0, hip, 0, -lean, 0)
		put(b.head, 0, upOf(trunkHeight), aheadOf(trunkHeight), min(max(one.Tilt, -maxTilt), maxTilt), 0)

		// The arms swing against the legs. The right one is held forward when there is
		// something in its hand, and lifted through a swing.
		right := step
		if one.Held.Known() || one.Swing > 0 {
			right = carry + strike*math.Sin(math.Pi*one.Swing)
		}
		put(b.arm, -armAside, upOf(shoulder), aheadOf(shoulder), -step, 0)
		put(b.arm, armAside, upOf(shoulder), aheadOf(shoulder), right, 0)

		if one.Held.Known() {
			// The hand is at the end of the arm, which hangs from the shoulder and has
			// swung forward by its pitch.
			hand := spot(armAside, upOf(shoulder)-armLength*math.Cos(right), aheadOf(shoulder)+armLength*math.Sin(right))
			held.drawHeld(c, cam, one.Held, hand, one.Yaw, right, p.lights[:])
		}
	}
}

// The size of what a cave person holds: a tool as its picture, a smaller thing as a
// smaller picture, and a block as a small block.
const (
	heldTool  = 0.7
	heldThing = 0.42
	heldCube  = 0.22
)

// flat reports whether an item is held as its picture, and not as a small block: a
// thing that is not a block, or a block that is drawn as a plant is, as a torch is.
func flat(item world.Item) bool {
	b, ok := item.Block()
	return !ok || b.Info().Plant
}

// drawHeld draws an item in a hand at a place in the world: a block as a small block,
// and anything else as its picture, standing up out of the hand and tipped forward as
// the arm that holds it is.
func (d *Drops) drawHeld(c *engine.Canvas, cam mesh.Camera, item world.Item, hand geometry.Vector3, yaw, arm float64, lights []mesh.Light) {
	m, size, at := d.cube(item), heldCube, hand
	switch _, tier := item.Tool(); {
	case !flat(item):
	case item == world.Torch.Item() || item == world.UnlitTorch:
		// A torch stands in the middle of its picture, so the hand holds it by its foot.
		m, size = d.plate(item), heldTool
	default:
		// The picture is across the way the person faces, so that it is seen whole from
		// behind and from in front, with its lower corner in the hand: a tool is drawn
		// from there up and out, and so it stands clear of the body.
		m, size = d.plate(item), heldThing
		if tier > 0 {
			size = heldTool
		}
		at = hand.Add(mesh.Camera{Yaw: yaw}.Right().Scale(size / 2))
	}
	m.Transform = mesh.Transform{Position: at, Yaw: yaw, Pitch: 0.3 - arm, Scale: size}
	m.DrawShadedPixels(c, cam, lights...)
}

package game

import (
	"math"
	"math/rand/v2"
	"slices"
	"strconv"

	"glonk.dev/bitmap"
	"glonk.dev/engine"
	"glonk.dev/geometry"
	"glonk.dev/mesh"

	"github.com/kyrokohan/skyspear/internal/assets"
	"github.com/kyrokohan/skyspear/internal/render"
	"github.com/kyrokohan/skyspear/internal/world"
)

// The people of the camps. They are made from the seed when their camp comes near, and
// let go when it is left behind, as the beasts are, and nothing of them is saved. A camp
// whose fire is not there has nobody: one put out, or one in ground made before there
// were camps.
const (
	campRoam   = 9.0  // how far from the fire they go about by day
	seatOut    = 2.5  // how far from the fire they sit at night
	seatLift   = 0.2  // how high off the ground a seated one's hips are
	personWalk = 1.4  // how fast they walk, in blocks a second
	talkReach  = 10.0 // how near the player must be to see what they say
	saySize    = 0.6  // how wide the picture over a head is, in blocks
	sayOver    = 0.2  // and how far over the top of the head it is shown
	sideTime   = 0.8  // how long a person goes aside round what is in their way, in seconds
)

// person is one of the people of a camp, or one who came from it to the player's fire.
type person struct {
	Pos         geometry.Vector3
	Yaw, Stride float64
	look        int
	camp        world.Cell // the fire of their camp
	fire        world.Cell // and the fire they live at: their camp's, or the player's
	seat        float64    // the way from the fire to their place by it, in radians
	want        world.Item // what they want, which they show over their head
	say         saying     // what they show now
	sayFor      float64    // and for how much longer, before they show their want again

	tribe   bool    // they came to the player's fire, or are on their way
	follows bool    // they follow the player there
	asks    bool    // they ask to come, as their camp trusts the player enough
	away    float64 // one left behind is on their own way, and at the fire at this day
	hurry   bool    // they run, to catch up
	swing   float64 // how far through a stroke of the drum their arm is, or held up with a flute

	job      job                     // the work the player gave one of the tribe
	workIn   float64                 // how long before a keeper sees to the hearth again
	errand   int                     // what a gatherer is on the way to do
	goal     geometry.Vector3        // and where
	goalCell world.Cell              // and the cell of it
	missed   []world.Cell            // the cells of what they could not reach since they were home
	carry    [gatherLoad]world.Stack // what they carry home
	picks    int                     // how many things they have picked since

	walking, seated, onGround bool
	velY, timer               float64
	aside                     float64          // how much longer they go aside, round what is in their way
	to                        geometry.Vector3 // where they walk to
	rng                       *rand.Rand
}

// campPeople returns the people of the camp whose fire is at a cell, as the seed makes
// them: three to five, no two alike, each standing a few steps from the fire.
func campPeople(w *world.World, fire world.Cell) []person {
	rng := rand.New(rand.NewPCG(uint64(w.Seed), uint64(uint32(fire.X))<<32|uint64(uint32(fire.Z))))
	them := make([]person, 3+rng.IntN(3))
	looks, wants := rng.Perm(assets.PersonLooks), campMarket(w.Seed, fire).wants
	for i := range them {
		seat := 2 * math.Pi * (float64(i) + 0.4*rng.Float64()) / float64(len(them))
		at := cellMid(fire.X, fire.Y, fire.Z).Add(geometry.Vector3{X: 4 * math.Cos(seat), Z: 4 * math.Sin(seat)})
		at.Y = groundAt(w, at, fire.Y)
		want := wants[rng.IntN(len(wants))]
		them[i] = person{
			Pos: at, Yaw: rng.Float64() * 2 * math.Pi, look: looks[i], camp: fire, fire: fire, seat: seat,
			want: want, say: saying{item: want},
			rng: rand.New(rand.NewPCG(uint64(w.Seed), rng.Uint64())),
		}
	}
	return them
}

// groundAt returns the height of the ground under a point, looked for from 3 blocks over
// a layer to 3 under it: the layer itself when there is none.
func groundAt(w *world.World, at geometry.Vector3, y int) float64 {
	for top := y + 3; top > y-4; top-- {
		if solid(w, int(math.Floor(at.X)), top-1, int(math.Floor(at.Z))) {
			return float64(top)
		}
	}
	return float64(y)
}

// stepPeople moves the people near the player on for dt seconds, and once a second
// brings in those of camps that came near and lets go of those of camps left behind.
func (g *Game) stepPeople(dt float64) {
	if g.peopleWait -= dt; g.peopleWait <= 0 {
		g.peopleWait = 1
		g.settle()
	}
	night := g.fauna.sun < sunUp
	for i := range g.people {
		one := &g.people[i]
		if one.away > 0 {
			continue
		}
		if one.sayFor -= dt; one.sayFor <= 0 {
			one.say = one.idle()
		}
		if one.Pos.Distance(g.player.Pos) < creatureAwake {
			switch {
			case g.work(one, night, dt):
			case one.follows:
				one.follow(g.player.Pos, dt)
			default:
				one.think(night, dt)
			}
			one.step(g.world, dt)
		}
	}
}

// idle is what a person shows when nothing has been said to them: what they want, that
// they would come to the player's fire, or, for one who has come, nothing.
func (one *person) idle() saying {
	switch {
	case one.tribe:
		return saying{}
	case one.asks:
		return asking
	}
	return saying{item: one.want}
}

// settle lets go of the people of camps that are far off or whose fire is out, and makes
// those of the camps near the player whose fire burns, but for any who came to the
// player's fire: those are kept wherever they are. Then it settles the tribe.
func (g *Game) settle() {
	p := g.player.Pos
	burns := func(fire world.Cell) bool { return g.world.At(fire.X, fire.Y, fire.Z) == world.Fire }
	g.people = slices.DeleteFunc(g.people, func(one person) bool {
		return !one.tribe && (cellMid(one.fire.X, one.fire.Y, one.fire.Z).Distance(p) > creatureGone || !burns(one.fire))
	})
	for _, fire := range g.world.Camps(p.X, p.Z, creatureAwake) {
		if cellMid(fire.X, fire.Y, fire.Z).Distance(p) < creatureAwake && burns(fire) &&
			!slices.ContainsFunc(g.people, func(one person) bool { return !one.tribe && one.camp == fire }) {
			g.people = append(g.people, slices.DeleteFunc(campPeople(g.world, fire), func(one person) bool {
				return slices.ContainsFunc(g.people, func(m person) bool { return m.tribe && m.camp == fire && m.look == one.look })
			})...)
		}
	}
	g.settleTribe()
}

// think has a person make up their mind, dt seconds on. By day they stand a while, then
// walk to somewhere in their camp, and so on. By night they go to their place by the fire
// and sit there, facing it.
func (one *person) think(night bool, dt float64) {
	mid := cellMid(one.fire.X, one.fire.Y, one.fire.Z)
	if one.aside > 0 {
		one.aside -= dt
		return
	}
	if night {
		seat := mid.Add(geometry.Vector3{X: seatOut * math.Cos(one.seat), Z: seatOut * math.Sin(one.seat)})
		to := seat.Sub(one.Pos)
		to.Y = 0
		if one.seated = to.Length() < 0.3; one.seated {
			one.walking, one.Yaw = false, yawOf(mid.Sub(one.Pos))
			return
		}
		one.walking, one.Yaw = true, yawOf(to)
		return
	}
	one.seated = false
	one.timer -= dt
	if one.walking {
		to := one.to.Sub(one.Pos)
		to.Y = 0
		if to.Length() < 0.7 || one.timer <= 0 {
			one.walking, one.timer = false, 2+4*one.rng.Float64()
		} else {
			one.Yaw = yawOf(to)
		}
		return
	}
	if one.timer > 0 {
		return
	}
	a, far := 2*math.Pi*one.rng.Float64(), 2+(campRoam-2)*one.rng.Float64()
	one.to = mid.Add(geometry.Vector3{X: far * math.Cos(a), Z: far * math.Sin(a)})
	one.walking, one.timer = true, 10
}

// step moves a person's body on for dt seconds: they walk where they face, and step up a
// block that is in their way, or down one, so that they can always come back. At anything
// higher, at water, at a deeper drop and at a fire, they turn to one side or the other
// and go that way a moment, round it.
func (one *person) step(w *world.World, dt float64) {
	one.velY = max(one.velY-gravity*dt, -fallSpeed)
	var move geometry.Vector3
	if one.walking {
		ahead := mesh.Camera{Yaw: one.Yaw}.Forward()
		front := one.Pos.Add(ahead.Scale(playerWidth/2 + lookAhead))
		fire := w.At(int(math.Floor(front.X)), int(math.Floor(one.Pos.Y+0.1)), int(math.Floor(front.Z))) == world.Fire
		switch climb, blocked := lookAheadOf(w, one.Pos, ahead, playerWidth, playerHeight, 1, 2, true, solid); {
		case climb > 0 && !fire:
			if one.onGround {
				one.velY = creatureHop
			}
		case (blocked || fire) && one.onGround:
			one.goAside()
			ahead = mesh.Camera{Yaw: one.Yaw}.Forward()
		}
		if one.walking {
			speed := personWalk
			if one.hurry {
				speed = personRun
			}
			move = ahead.Scale(speed * dt)
			one.Stride += math.Pi * speed * dt / stepLength
		}
	}
	if !one.walking {
		one.Stride = 0
	}
	move.Y = one.velY * dt
	stopped := travel(w, &one.Pos, playerWidth, playerHeight, move)
	if one.onGround = stopped[1] && move.Y < 0; stopped[1] {
		one.velY = 0
	}
	if (stopped[0] || stopped[2]) && one.walking && one.onGround && one.aside <= 0 {
		one.goAside() // a corner the look ahead did not see
	}
}

// goAside turns a person a quarter of the way round, to one side or the other, to walk
// that way a moment and then make up their mind again.
func (one *person) goAside() {
	one.Yaw += math.Pi / 2 * float64(1-2*one.rng.IntN(2))
	one.aside = sideTime
}

// pose is how a person is drawn. A seated one sits on the ground, which is lower than a
// rider sits.
func (one *person) pose() render.Person {
	p := render.Person{Pos: one.Pos, Yaw: one.Yaw, Look: one.look, Stride: one.Stride, Swing: one.swing, Seated: one.seated}
	if one.seated {
		p.Pos.Y -= render.SeatHeight - seatLift
	}
	return p
}

// over is the place over a person's head where what they say is shown from.
func (one *person) over() geometry.Vector3 {
	top := one.Pos.Add(geometry.Vector3{Y: playerHeight + sayOver})
	if one.seated {
		top.Y -= render.SeatHeight - seatLift
	}
	return top
}

// The colors of the pale ground a person's picture is shown on, and of its edge.
var (
	sayPale = engine.RGB(236, 228, 208)
	sayEdge = engine.RGB(74, 58, 44)
)

// sayings draws over the head of each person near the player what they say, as a picture
// on a pale ground with a tail down to them, as wide as saySize at their distance. It is
// drawn at their depth, so what stands in front of them hides it.
func (g *Game) sayings(c *engine.Canvas, cam mesh.Camera) {
	fov := cam.FOV
	if fov == 0 {
		fov = math.Pi / 2
	}
	for i := range g.people {
		one := &g.people[i]
		if one.away > 0 || one.say == (saying{}) || one.Pos.Distance(g.player.Pos) > talkReach {
			continue
		}
		x, y, depth, ok := cam.ProjectPixels(one.over(), c)
		if !ok {
			continue
		}
		pic := g.bubble(one.say)
		w := int(math.Round(saySize / (2 * depth * math.Tan(fov/2)) * float64(c.Height())))
		if w < 6 {
			continue // too small to make out
		}
		h := w * pic.Height() / pic.Width()
		pic.DrawScaledDepth(c, int(math.Floor(x))-w/2, int(math.Floor(y))-h, w, h, depth)
	}
}

// The size of a picture with its ground and tail, in its own pixels.
const (
	bubbleSide = 20
	bubbleTail = 3
)

// bubble returns what a person says as a picture on a pale ground with a dark edge, and
// a tail under it: a thing, with how many in its corner when there are more than one, or
// a hand stenciled in red on stone, for thanks. Each is made once.
func (g *Game) bubble(s saying) *bitmap.Bitmap {
	if b := g.bubbles[s]; b != nil {
		return b
	}
	b := bitmap.New(bubbleSide, bubbleSide+bubbleTail)
	const last = bubbleSide - 1
	for y := range bubbleSide {
		for x := range bubbleSide {
			switch edge := x == 0 || y == 0 || x == last || y == last; {
			case (x == 0 || x == last) && (y == 0 || y == last):
				// the corners are round
			case edge && !(y == last && x >= 9 && x <= 10):
				b.Set(x, y, sayEdge)
			default:
				b.Set(x, y, sayPale)
			}
		}
	}
	for i := range bubbleTail {
		b.Set(8+i, bubbleSide+i, sayEdge)
		b.Set(11-i, bubbleSide+i, sayEdge)
		for x := 9 + i; x <= 10-i; x++ {
			b.Set(x, bubbleSide+i, sayPale)
		}
	}
	icon := g.set.Icons[s.item.Icon()]
	if s.hand {
		icon = g.rend.Picture(world.Painting{Picture: world.Hand, Paint: world.RedPaint}, world.Stone, false)
	}
	for y := range icon.Height() {
		for x := range icon.Width() {
			if p := icon.At(x, y); !p.IsDefault() {
				b.Set(2+x, 2+y, p)
			}
		}
	}
	if s.count > 1 {
		count(b, s.count, bubbleSide-3, bubbleSide-3)
	}
	g.bubbles[s] = b
	return b
}

// digits are the figures a count is drawn in, three pixels wide and five high.
var digits = [10][5]string{
	{"###", "#.#", "#.#", "#.#", "###"}, {".#.", "##.", ".#.", ".#.", "###"},
	{"###", "..#", "###", "#..", "###"}, {"###", "..#", "###", "..#", "###"},
	{"#.#", "#.#", "###", "..#", "..#"}, {"###", "#..", "###", "..#", "###"},
	{"###", "#..", "###", "#.#", "###"}, {"###", "..#", ".#.", ".#.", ".#."},
	{"###", "#.#", "###", "#.#", "###"}, {"###", "#.#", "###", "..#", "###"},
}

// count draws a number into a picture with its last figure's lower right corner at x, y:
// dark figures on a pale ground a pixel wider all round, so it reads over the thing.
func count(b *bitmap.Bitmap, n, x, y int) {
	text := strconv.Itoa(n)
	left := x - 4*len(text) + 2
	for py := y - 5; py <= y+1; py++ {
		for px := left - 1; px <= x+1; px++ {
			b.Set(px, py, sayPale)
		}
	}
	for i, d := range text {
		for row, line := range digits[d-'0'] {
			for col, c := range line {
				if c == '#' {
					b.Set(left+4*i+col, y-4+row, sayEdge)
				}
			}
		}
	}
}

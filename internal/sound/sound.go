// Package sound names the game's sounds and mixes them: each from where it is in the world,
// quieter with distance and louder in the ear it is nearer, over two loops that are heard
// all the while, the wind and the air of caves. The sounds are made by cmd/genassets as
// WAV files in internal/assets/sound, which internal/assets reads.
package sound

// Rate is how many samples a second every sound has, and the mixer makes.
const Rate = 22050

// Material is what a block sounds like when it is walked on, dug, broken and placed.
type Material uint8

const (
	Earth Material = iota
	Stone
	Sand
	Wood
	Plant
	Glass
	Snow
	materials
)

// Name is one of the game's sounds.
type Name uint8

// The four sounds of each material come first, in the order of the materials: its step,
// a stroke of digging it, its breaking and its placing.
const (
	stepOf Name = iota
	digOf
	breakOf
	placeOf
	perMaterial
)

const (
	Splash Name = Name(materials)*perMaterial + iota // falling into water
	Pickup                                           // something picked up
	Hurt                                             // the player hurt
	Eat
	Call    // a mossback, to itself
	Cry     // a mossback hit
	Horn    // a bugler, to its herd: a long low call through its crest
	Alarm   // a bugler that takes alarm, or is hit
	Chirp   // a nipper
	Growl   // a quickclaw that hunts: low, and from the throat
	Screech // a quickclaw hit, or one that has seen its prey
	Bellow  // a shieldbrow that warns, charges or is hit: deep, and from the chest
	Grunt   // a giant plant-eater, to itself
	Rattle  // a thornback or a boulderback that warns: its plates shaken, and a hiss
	Thud    // the foot of a giant coming down, which is heard from far and felt
	Drone   // a cloudgrazer, to its mate: long, and very low
	Roar    // the dread, when it has seen its prey
	Rumble  // the dread, to itself: breath, and a rattle under it
	Geyser  // a geyser that spouts: water and steam, up out of the ground
	Tremor  // the ground under a fire mountain that is about to erupt, and while it does
	Boom    // a star that struck, heard from far off and after
	Trumpet // a snowtusk that warns, charges or is hit: a long blast through its trunk
	Howl    // a pack of howlers, far off: it rises, holds and falls
	Snarl   // a daggercat that springs, or is hit
	Whirr   // a rattlewing: its wings, dry and loud
	Crackle // a hearth's fire
	Drip    // water in a cave
	Wind    // a loop
	Cave    // a loop
	Count
)

// Step, Dig, Break and Place are the sounds of a material.
func Step(m Material) Name  { return Name(m)*perMaterial + stepOf }
func Dig(m Material) Name   { return Name(m)*perMaterial + digOf }
func Break(m Material) Name { return Name(m)*perMaterial + breakOf }
func Place(m Material) Name { return Name(m)*perMaterial + placeOf }

var materialNames = [materials]string{"earth", "stone", "sand", "wood", "plant", "glass", "snow"}

var otherNames = [Count - Splash]string{
	"splash", "pickup", "hurt", "eat", "call", "cry", "horn", "alarm", "chirp", "growl", "screech", "bellow", "grunt", "rattle", "thud", "drone", "roar", "rumble", "geyser", "tremor", "boom", "trumpet", "howl", "snarl", "whirr", "crackle", "drip", "wind", "cave",
}

// File is the name of the sound's file: step-wood.wav, say, or splash.wav.
func (n Name) File() string {
	if n < Splash {
		kind := [perMaterial]string{"step", "dig", "break", "place"}[n%perMaterial]
		return kind + "-" + materialNames[n/perMaterial] + ".wav"
	}
	return otherNames[n-Splash] + ".wav"
}

// Loops reports whether the sound is played round and round, as the wind and the air of
// caves are.
func (n Name) Loops() bool {
	return n == Wind || n == Cave
}

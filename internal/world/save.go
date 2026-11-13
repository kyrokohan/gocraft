package world

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"

	"glonk.dev/geometry"
)

// Save is what a save file holds: the world, and the player in it.
type Save struct {
	World *World

	// Pos is the point between the player's feet, and Yaw and Pitch where it looks.
	Pos        geometry.Vector3
	Yaw, Pitch float64
	Flying     bool

	// Slot is the hotbar slot in use, from 0.
	Slot int
}

// magic opens every save file. Its last character is the version of the layout: change
// it when the layout changes, and old files are refused instead of misread.
const magic = "GOCRAFT1"

// header follows the magic, in little-endian byte order. The blocks follow it, one
// byte each in the order World keeps them, compressed with gzip.
type header struct {
	Seed                int64
	SizeX, SizeY, SizeZ uint32
	X, Y, Z             float64
	Yaw, Pitch          float64
	Flying              uint8
	Slot                uint8
}

// errBadSave is wrapped by every error Read returns for a file that is not a save this
// game can use.
var errBadSave = errors.New("not a usable save file")

func finite(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// encode writes a save without looking at what is in it.
func encode(w io.Writer, s Save) error {
	if _, err := io.WriteString(w, magic); err != nil {
		return err
	}
	h := header{
		Seed:  s.World.Seed,
		SizeX: SizeX, SizeY: SizeY, SizeZ: SizeZ,
		X: s.Pos.X, Y: s.Pos.Y, Z: s.Pos.Z,
		Yaw: s.Yaw, Pitch: s.Pitch,
		Slot: uint8(s.Slot),
	}
	if s.Flying {
		h.Flying = 1
	}
	if err := binary.Write(w, binary.LittleEndian, h); err != nil {
		return err
	}
	zw := gzip.NewWriter(w)
	if _, err := zw.Write(s.World.Bytes()); err != nil {
		return err
	}
	return zw.Close()
}

// Write saves s at path, making the folder if it is not there. The save is written to
// another file beside it first and then put in its place, so a write that fails part
// way, or a machine that stops, leaves the save that was there before.
func Write(path string, s Save) (err error) {
	if s.World == nil {
		return errors.New("save: no world to write")
	}
	if !finite(s.Pos.X, s.Pos.Y, s.Pos.Z, s.Yaw, s.Pitch) || s.Slot < 0 || s.Slot > math.MaxUint8 {
		return errors.New("save: the player is nowhere a save can hold")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".saving-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()

	buffered := bufio.NewWriter(tmp)
	if err = encode(buffered, s); err != nil {
		return err
	}
	if err = buffered.Flush(); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Read loads the save at path. A file that is not there gives an error that wraps
// fs.ErrNotExist. A file that is not a save, is cut short, was written by another
// version, or holds something this game cannot use gives an error and never a panic:
// what is in it is checked before it is used.
func Read(path string) (Save, error) {
	f, err := os.Open(path)
	if err != nil {
		return Save{}, err
	}
	defer f.Close()

	bad := func(why string) (Save, error) {
		return Save{}, fmt.Errorf("%s: %w: %s", path, errBadSave, why)
	}

	r := bufio.NewReader(f)
	start := make([]byte, len(magic))
	if _, err := io.ReadFull(r, start); err != nil || string(start) != magic {
		return bad("it does not start as a gocraft save of this version does")
	}
	var h header
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		return bad("it is cut short")
	}
	if h.SizeX != SizeX || h.SizeY != SizeY || h.SizeZ != SizeZ {
		return bad(fmt.Sprintf("its world is %d by %d by %d blocks, and this game's is %d by %d by %d",
			h.SizeX, h.SizeY, h.SizeZ, SizeX, SizeY, SizeZ))
	}
	if !finite(h.X, h.Y, h.Z, h.Yaw, h.Pitch) {
		return bad("its player is nowhere")
	}

	zr, err := gzip.NewReader(r)
	if err != nil {
		return bad("its blocks cannot be read")
	}
	w := New()
	w.Seed = h.Seed
	// The blocks are read into the world's own list, so a file can never make the game
	// hold more than one world, however much it claims to unpack to.
	raw := make([]byte, len(w.blocks))
	if _, err := io.ReadFull(zr, raw); err != nil {
		return bad("its blocks are cut short")
	}
	// The end of the blocks must be the end of the file. Reading on to it is also what
	// makes gzip check the blocks against the checksum it keeps after them.
	if n, err := zr.Read(make([]byte, 1)); n != 0 || err != io.EOF {
		return bad("its blocks are damaged, or there is more after them")
	}
	for i, b := range raw {
		if !Block(b).Known() {
			return bad(fmt.Sprintf("it holds a block of kind %d, which this game does not have", b))
		}
		w.blocks[i] = Block(b)
		if b != byte(Air) {
			w.height = max(w.height, i/(SizeX*SizeZ)+1)
		}
	}

	return Save{
		World:  w,
		Pos:    geometry.Vector3{X: h.X, Y: h.Y, Z: h.Z},
		Yaw:    h.Yaw,
		Pitch:  h.Pitch,
		Flying: h.Flying != 0,
		Slot:   int(h.Slot),
	}, nil
}

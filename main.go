// Command skyspear is a block-building game that runs in the terminal.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"glonk.dev/engine"

	"github.com/kyrokohan/skyspear/internal/game"
	"github.com/kyrokohan/skyspear/internal/sound"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "skyspear:", err)
		os.Exit(1)
	}
}

func run() error {
	seed := flag.Int64("seed", 0, "the number a new world is made from; 0 takes one from the clock")
	worldPath := flag.String("world", "", "play only the world saved in this folder, with no list of worlds")
	worldsPath := flag.String("worlds", "", "the folder of the worlds the title lists (default: skyspear/worlds in your config folder)")
	mouse := flag.Bool("mouse", true, "aim, break and place with the mouse; false leaves the pointer to the terminal, to select text")
	shot := flag.String("shot", "", "draw one frame of play into this PNG file and quit; nothing is loaded or saved")
	mute := flag.Bool("mute", false, "play with no sound, and leave the sound device alone")
	flag.Parse()

	if *shot != "" {
		g, err := game.New(game.Options{Seed: *seed})
		if err != nil {
			return err
		}
		return writeShot(g, *shot)
	}

	if *worldPath != "" && *worldsPath != "" {
		return errors.New("-world and -worlds cannot both be given")
	}
	opts := game.Options{Seed: *seed, SavePath: *worldPath, Worlds: *worldsPath}
	dir, dirErr := os.UserConfigDir()
	if dirErr == nil {
		opts.Settings = filepath.Join(dir, "skyspear", "settings.json")
	}
	if opts.SavePath == "" && opts.Worlds == "" {
		if dirErr != nil {
			return fmt.Errorf("no folder to save worlds in (%w): name one with -worlds", dirErr)
		}
		opts.Worlds = filepath.Join(dir, "skyspear", "worlds")

		// The first version of the game, then called gocraft, kept its one world in a
		// file in its own folder. It is not read and not touched; the title says that
		// it is there.
		if old := filepath.Join(dir, "gocraft", "world.sav"); isFile(old) {
			opts.OldSave = old
		}
	}
	g, err := game.New(opts)
	if err != nil {
		return err
	}
	if !*mute {
		// A machine with no sound device is played on in silence; the title says why.
		if speaker, err := sound.Open(g.Mixer()); err != nil {
			g.NoSound(err)
		} else {
			defer speaker.Close()
		}
	}

	// Run returns when the player quits, and also on Ctrl-C, kill and a closing window,
	// so saving here keeps the world however the game ends.
	err = engine.Run(context.Background(), g, engine.Config{FPS: 30, Mouse: *mouse})
	if saveErr := g.Save(); saveErr != nil {
		err = errors.Join(err, fmt.Errorf("the world was not saved: %w", saveErr))
	}
	return err
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// writeShot draws one frame of play as a terminal of 240 by 67 cells would show it and
// saves it as a picture, each pixel four across, so the frame can be looked at without
// one. glonk draws the picture: a letter is a mark in its color where it stands.
func writeShot(g *game.Game, path string) error {
	fb, err := engine.NewFrameBuffer(240, 67)
	if err != nil {
		return err
	}
	g.Play()
	// A second of play with no keys, so that the player has landed and taken aim.
	keys := engine.NewKeyboard()
	for range 30 {
		if err := g.Update(1.0/30, keys.Input()); err != nil {
			return err
		}
	}
	g.Draw(fb)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, fb.Image(engine.ImageOptions{Scale: 4})); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

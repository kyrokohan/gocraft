// Command gocraft is a block-building sandbox that runs in the terminal.
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

	"github.com/kyrokohan/gocraft/internal/game"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gocraft:", err)
		os.Exit(1)
	}
}

func run() error {
	seed := flag.Int64("seed", 0, "the number a new world is made from; 0 takes one from the clock")
	worldPath := flag.String("world", "", "the file the world is saved in (default: gocraft/world.sav in your config folder)")
	mouse := flag.Bool("mouse", true, "aim, break and place with the mouse; false leaves the pointer to the terminal, to select text")
	shot := flag.String("shot", "", "draw one frame of play into this PNG file and quit; nothing is loaded or saved")
	flag.Parse()

	if *shot != "" {
		g, err := game.New(game.Options{Seed: *seed})
		if err != nil {
			return err
		}
		return writeShot(g, *shot)
	}

	path := *worldPath
	if path == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return fmt.Errorf("no folder to save the world in (%w): name a file with -world", err)
		}
		path = filepath.Join(dir, "gocraft", "world.sav")
	}
	g, err := game.New(game.Options{Seed: *seed, SavePath: path})
	if err != nil {
		return err
	}

	// Run returns when the player quits, and also on Ctrl-C, kill and a closing window,
	// so saving here keeps the world however the game ends.
	err = engine.Run(context.Background(), g, engine.Config{FPS: 30, Mouse: *mouse})
	if saveErr := g.Save(); saveErr != nil {
		err = errors.Join(err, fmt.Errorf("the world was not saved: %w", saveErr))
	}
	return err
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

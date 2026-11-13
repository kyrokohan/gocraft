// Command gocraft is a block-building sandbox that runs in the terminal.
package main

import (
	"context"
	"flag"
	"fmt"
	"image/png"
	"os"

	"glonk.dev/engine"

	"github.com/kyrokohan/gocraft/internal/game"
	"github.com/kyrokohan/gocraft/internal/render"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	seed := flag.Int64("seed", 0, "the number a new world is made from; 0 takes one from the clock")
	shot := flag.String("shot", "", "draw one frame into this PNG file and quit, instead of playing")
	flag.Parse()

	g, err := game.New(game.Options{Seed: *seed})
	if err != nil {
		return err
	}
	if *shot != "" {
		return writeShot(g, *shot)
	}
	return engine.Run(context.Background(), g, engine.Config{FPS: 30})
}

// writeShot draws one frame of play as a terminal of 240 by 67 cells would show it and
// saves its pixels, each four across, so the picture can be looked at without one. Text
// is not in it: it is drawn in cells, not pixels.
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
	if err := png.Encode(f, render.Picture(fb.Canvas(), 4)); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

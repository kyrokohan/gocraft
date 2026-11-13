# gocraft

A block-building sandbox that runs in the terminal, built on [glonk](https://glonk.dev).

Walk a world of hills, water and trees made from a seed, break blocks and place your own, and find it as you left it the next time. The art is original and drawn by code in this repo.

> An MVP: one world of a fixed size, creative play only. See [Not in yet](#not-in-yet).

## Run

```
go run .
```

Requires Go 1.26 or newer.

- **Use a terminal that reports key releases:** Ghostty, kitty, WezTerm, foot, iTerm2 or Alacritty. There you can walk and turn at once. In other terminals (Apple's Terminal, for one) only one key stays held at a time, which is playable but clumsy. This is how glonk reads keys; see its README.
- **A bigger window is a sharper picture.** Every cell is two pixels, one above the other, so 200 by 60 cells is a picture of 200 by 120 pixels.
- **True color looks best.** Inside tmux or over ssh, `export COLORTERM=truecolor` if your terminal can show it.

## Keys

| Keys | Action |
|---|---|
| W A S D | walk, strafe |
| Arrows | look |
| Space | jump; swim up; rise when flying |
| X | sink when flying |
| F | flying on and off |
| Q / E | break / place the block aimed at (hold to repeat) |
| 1 to 9, Tab | pick a block; next block |
| Esc | pause menu, with these keys on it |

There is no mouse: glonk reads the keyboard only. Letters work in either case, and Ctrl-C quits (and saves).

## Saving

The world is saved when you choose **Save and quit**, and also on Ctrl-C or when the window closes. Next time the title offers **Continue**. **New world** asks before it takes the place of a saved one.

- The file is `gocraft/world.sav` in your config folder (`~/Library/Application Support` on macOS, `~/.config` on Linux), or the one named with `-world`.
- A save is written beside the old one and then put in its place, so a write that fails leaves the old save.
- A save file that cannot be read stops the game with an error instead of being written over. Move it away, or name another with `-world`.

## Flags

| Flag | Meaning |
|---|---|
| `-seed n` | the number a new world is made from; the same number makes the same world. 0, the default, takes one from the clock |
| `-world file` | where the world is saved |
| `-shot file.png` | draw one frame of play into a PNG and quit, to look at the picture without a terminal (text is not in it) |

## How it is drawn

glonk's meshes are filled one color to a triangle and cannot be seen through, so the blocks are not meshes. Every pixel of glonk's half-block `Canvas` sends a ray through the grid of blocks and takes the color of the tile it lands on, which gives textures to the pixel, water you can see into, and holes in leaves and glass. A frame of 240 by 134 pixels takes 1 to 3 ms on one core of an Apple M5 Pro.

The rays use the projection of glonk's `mesh.Camera` and write glonk's depth with `Canvas.SetDepth`, so a mesh drawn by glonk into the same canvas is hidden by nearer blocks; a test holds the two together. glonk also runs the loop and the terminal, reads the keys, draws the title picture (`bitmap`) and the menus and HUD (`widget`), and runs every test without a terminal.

## Pictures

`internal/assets/img` holds the two pictures the game uses: a strip of 14 block tiles and the title. Both are drawn by `cmd/genassets` from a fixed hash, so they come out the same every time:

```
make assets                                   # draw them again
go run ./cmd/genassets -preview sheet.png     # and an enlarged sheet to look at
```

A test fails if the pictures in the repo are not what the generator draws.

## Development

```
make check    # formatting, vet, and tests with the race detector
make bench    # the time to draw a frame
```

The race detector needs cgo. Where there is no C compiler, run `make check RACE=`.

| Package | Purpose |
|---|---|
| `internal/world` | the grid of blocks, terrain from a seed, rays through the grid, save files |
| `internal/render` | the world drawn into a glonk `Canvas`, one ray to a pixel |
| `internal/game` | the game glonk runs: the player, building, the HUD, the title and pause menus |
| `internal/assets` | the pictures, read once at start-up, each tile in five sizes |
| `internal/tile` | the names of the tiles in the strip |
| `cmd/genassets` | draws the pictures |

glonk is pinned to a commit on its main branch, because the `widget` and `bitmap` packages are newer than its latest tag.

## Not in yet

- An endless world, day and night, flowing water, light and shadows.
- Creatures, crafting, counted blocks, health.
- Mouse look and sound.

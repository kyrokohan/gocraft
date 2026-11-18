# gocraft

A block-building sandbox that runs in the terminal, built on [glonk](https://glonk.dev).

Walk a world of hills, water and trees made from a seed, under drifting clouds and among grazing mossbacks. Break blocks and place your own, with the keys or the mouse, look round by dragging, and find the world as you left it the next time. The art is original and drawn by code in this repo.

> An MVP: one world of a fixed size, creative play only. See [Not in yet](#not-in-yet).

## Run

```
go run .
```

Requires Go 1.26 or newer.

- **Use a terminal that reports key releases:** Ghostty, kitty, WezTerm, foot, iTerm2 or Alacritty. There you can walk and turn at once. In other terminals (Apple's Terminal, for one) only one key stays held at a time, which is playable but clumsy, and the game says so on screen. This is how glonk reads keys; see its README.
- **A bigger window is a sharper picture.** Every cell is two pixels, one above the other, so 200 by 60 cells is a picture of 200 by 120 pixels.
- **True color looks best.** Inside tmux or over ssh, `export COLORTERM=truecolor` if your terminal can show it.

## Keys and mouse

| Keys | Action |
|---|---|
| W A S D | walk, strafe |
| Arrows | look |
| Space | jump; swim up; rise when flying |
| X | sink when flying |
| F | flying on and off |
| Q / E | break / place the block aimed at (hold to repeat) |
| 1 to 9, Tab | pick a block; next block |
| F3 | frame numbers: where the time of a frame goes and how much is sent |
| Esc | pause menu, with these keys on it |

| Mouse | Action |
|---|---|
| Move | aim at the block under the pointer |
| Hold left and drag | turn the view |
| Left click / right click | break / place (hold still to repeat) |
| Wheel | pick a block |
| Click a hotbar slot | pick that block |

- **Whichever you used last aims.** Move or click the mouse and the pointer aims. Press an arrow key and the middle of the view aims again, with its crosshair.
- **Drag to look.** A terminal reports the pointer in cells and cannot lock it, so the view cannot follow the mouse as it does in a window. Holding the left button and dragging turns it, a cell at a time; a press let go where it is stays a click. The arrows turn it too.
- While the game runs, the terminal cannot select text with the mouse. `-mouse=false` gives the pointer back to it.
- Letters work in either case, and Ctrl-C quits (and saves).

## Saving

The world is saved when you choose **Save and quit**, and also on Ctrl-C or when the window closes. Next time the title offers **Continue**. **New world** asks before it takes the place of a saved one.

- The file is `gocraft/world.sav` in your config folder (`~/Library/Application Support` on macOS, `~/.config` on Linux), or the one named with `-world`.
- A save is written beside the old one and then put in its place, so a write that fails leaves the old save.
- A save file that cannot be read stops the game with an error instead of being written over. Move it away, or name another with `-world`.
- The mossbacks are not in the save file yet: they start again round the spawn each time a world loads.

## Flags

| Flag | Meaning |
|---|---|
| `-seed n` | the number a new world is made from; the same number makes the same world. 0, the default, takes one from the clock |
| `-world file` | where the world is saved |
| `-mouse=false` | leave the pointer to the terminal |
| `-shot file.png` | draw one frame of play into a PNG and quit, to look at it without a terminal. A letter is a mark in its color where it stands |

## How it is drawn

Two things draw into one frame, and glonk holds them together.

- **The blocks are rays.** Every pixel of glonk's half-block `Canvas` sends a ray through the grid of blocks and takes the color of the tile it lands on. That is quick, it lets a far block be drawn from a smaller copy of its tile so that it does not flicker, and it does water exactly: you see the bottom through it. The rays are the ones `Camera.RayPixels` gives, they end at the camera's `Far`, they fade by `Fog.Apply`, and they store glonk's depth with `Canvas.SetDepth`.
- **The things are glonk meshes.** The clouds are see-through slabs (`Material.Opacity`) and a mossback is six boxes with a picture on their faces (`Material.Texture`). glonk lights them, fades them by the same fog, and hides them behind any block that is nearer, through the depth the rays stored. The outline round the block you aim at is a cube's edges drawn with `DrawOnSurfacePixels`.

On one core of an Apple M5 Pro, at 240 by 134 pixels, the blocks take 1 to 3 ms, the clouds 0.3 ms and twelve mossbacks 0.1 ms.

glonk also runs the loop and the terminal, reads the keys and the mouse, draws the title picture (`bitmap`), the menus and the HUD (`widget`), puts the hotbar's pictures down over its panel (`Canvas.ComposeRect`), reports the frame numbers (`Stats`), draws the `-shot` picture (`FrameBuffer.Image`), and runs every test without a terminal.

## Pictures

`internal/assets/img` holds the three pictures the game uses: a strip of 14 block tiles, the title, and the mossback's four tiles. All are drawn by `cmd/genassets` from a fixed hash, so they come out the same every time:

```
make assets                                   # draw them again
go run ./cmd/genassets -preview sheet.png     # and an enlarged sheet to look at
```

A test fails if the pictures in the repo are not what the generator draws.

## Development

```
make check    # formatting, vet, and tests with the race detector
make bench    # the time to draw the blocks, the clouds, the creatures and a whole frame
```

The race detector needs cgo. Where there is no C compiler, run `make check RACE=`.

| Package | Purpose |
|---|---|
| `internal/world` | the grid of blocks, terrain from a seed, rays through the grid, save files |
| `internal/render` | the blocks drawn by rays into a glonk `Canvas`, and the clouds and creatures drawn into it as glonk meshes |
| `internal/game` | the game glonk runs: the player, the creatures, building, the mouse, the HUD, the title and pause menus |
| `internal/assets` | the pictures, read once at start-up, each block tile in five sizes |
| `internal/tile` | the names of the tiles in the strip |
| `cmd/genassets` | draws the pictures |

gocraft uses glonk v0.12.0.

## Not in yet

- An endless world, day and night, flowing water, light and shadows.
- More kinds of creature, creatures in the save file, anything to do to them.
- Crafting, counted blocks, health.
- A view that follows the mouse without a button held, and sound.

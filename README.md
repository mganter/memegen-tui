# memegen

Graphical CLI meme generator in Go — place captions on an image with your
**mouse**, right in the terminal. Like imgflip's meme generator, but a TUI.

## Install / run

```sh
go install github.com/mganter/memegen-tui/cmd/memegen@latest  # or build locally:
go build -o memegen ./cmd/memegen
./memegen                              # no image → mouse-driven file browser opens
./memegen path/to/image.png            # writes path/to/image-meme.png on save
./memegen -o out.png path/to/image.jpg # custom output
```

Run with no image and a file browser opens at the current directory: click a
folder to descend, `..` to go up, click an image to open it in the editor.

Inputs: PNG, JPEG, GIF. Output: PNG (also copy to clipboard).

## Controls

| Input | Action |
|-------|--------|
| **Click** empty area | add a caption there |
| **Click** a caption | select it |
| **Drag** | move selected caption |
| `Enter` | edit selected caption text (`Esc` to finish) |
| `Tab` | cycle selection |
| `+` / `-` | grow / shrink font |
| arrows | nudge selected caption |
| `d` | delete selected |
| `s` | save PNG |
| `c` | copy PNG to clipboard |
| `q` / `Ctrl+C` | quit |

The image preview uses Unicode half-blocks with ANSI truecolor, so it works in
any color terminal (no sixel/kitty required). Captions render with a black
outline in the bundled bold font; the preview shows the real flattened result.

## Layout

Follows [golang-standards/project-layout](https://github.com/golang-standards/project-layout):

- `cmd/memegen/`      — CLI entry point.
- `pkg/canvas/`       — domain model: base image + text boxes, hit-test, move (pure).
- `pkg/render/`       — burns captions onto the image → flattened `image.Image`.
- `pkg/preview/`      — image → half-block ANSI string + cell↔pixel mapping (mouse).
- `pkg/memeio/`       — load image, encode/save PNG, clipboard copy.
- `internal/browser/` — mouse file picker shown when no image is passed.
- `internal/ui/`      — bubbletea model wiring mouse/keys to canvas ops.

## Test

```sh
go test ./...
```

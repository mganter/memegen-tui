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

The top rows open searchable browsers over online meme template catalogs:

- **★ browse KnowYourMeme templates** — the
  [KnowYourMeme dataset](https://www.kaggle.com/datasets/adityaaggarwal27/knowyourmeme-memes)
  (~43k entries). Fetched from Kaggle on first use and cached; later runs reuse
  it via an HTTP conditional request (ETag/If-Modified-Since) and only
  re-download when the dataset changed.
- **★ browse Imgflip templates** — the ~100 most popular templates from the
  [imgflip API](https://api.imgflip.com/get_memes). Cached as CSV after first
  fetch.

Type to filter by title, `↑`/`↓` to move, `Enter` to pick — the template image
is downloaded and opened in the editor. A live preview of the highlighted
template renders beside the list (fetched async and cached). On terminals that
speak the **Kitty graphics protocol** (kitty, Ghostty) the preview is drawn as
real pixels; elsewhere it falls back to Unicode half-blocks. Force the fallback
with `MEMEGEN_NO_GRAPHICS=1`. Both catalogs cache under the user cache dir
(`~/Library/Caches/memegen` on macOS, `$XDG_CACHE_HOME/memegen` on Linux) and
work offline once primed.

Inputs: PNG, JPEG, GIF. Output: PNG (also copy to clipboard). Base images larger
than 1080p (1920×1080) are downscaled to fit on load (aspect preserved), so the
editor, preview, and saved meme stay at a sane resolution.

## Controls

| Input | Action |
|-------|--------|
| **Click** empty area | add a caption there |
| **Click** a caption | select it |
| **Drag** | move selected caption |
| `Enter` | edit selected caption text (`Esc` to finish) |
| `Tab` | cycle selection |
| `+` / `-` | grow / shrink font |
| `,` / `.` | narrow / widen text box (wrap width) |
| arrows | nudge selected caption |
| `Shift`+`↑` / `↓` | jump caption to top / bottom |
| `d` | delete selected |
| `s` | save PNG |
| `c` | copy PNG to clipboard |
| `q` / `Ctrl+C` | quit |

The editor preview shows the real flattened result (captions render with a black
outline in the bundled bold font). On terminals that speak the **Kitty graphics
protocol** (kitty, Ghostty) it is drawn as real pixels; otherwise it falls back
to Unicode half-blocks with ANSI truecolor, which works in any color terminal.
`MEMEGEN_NO_GRAPHICS=1` forces the half-block fallback. Mouse caption placement
works identically in both modes (the graphics image occupies the same cell grid).

## Layout

Follows [golang-standards/project-layout](https://github.com/golang-standards/project-layout):

- `cmd/memegen/`      — CLI entry point.
- `pkg/canvas/`       — domain model: base image + text boxes, hit-test, move (pure).
- `pkg/render/`       — burns captions onto the image → flattened `image.Image`.
- `pkg/preview/`      — image → half-block ANSI (cell↔pixel mapping for mouse) + Kitty graphics renderer.
- `pkg/memeio/`       — load image (file/URL), encode/save PNG, clipboard copy, image download.
- `pkg/memecat/`      — source-agnostic meme catalog: Template/Catalog, title search, CSV cache.
- `pkg/kym/`          — KnowYourMeme dataset loader: CSV parse + fetch/conditional cache.
- `pkg/imgflip/`      — imgflip API loader: get_memes → catalog + CSV cache.
- `internal/browser/` — mouse file picker shown when no image is passed.
- `internal/templates/` — searchable browser (with live preview) over a meme catalog.
- `internal/ui/`      — bubbletea model wiring mouse/keys to canvas ops.

## Test

```sh
go test ./...
```

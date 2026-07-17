# ghost-captcha

[![Go Reference](https://pkg.go.dev/badge/github.com/rohitkhatri1st/ghost-captcha.svg)](https://pkg.go.dev/github.com/rohitkhatri1st/ghost-captcha)
[![test](https://github.com/rohitkhatri1st/ghost-captcha/actions/workflows/test.yml/badge.svg)](https://github.com/rohitkhatri1st/ghost-captcha/actions/workflows/test.yml)
[![License](https://img.shields.io/github/license/rohitkhatri1st/ghost-captcha)](LICENSE)

A Go library that renders text as an animated "ghost text" CAPTCHA: the
letterforms are invisible in any single frame and only become readable once
the animation plays.

![demo](assets/demo.gif)

## How it works

Every pixel — inside the letterforms and outside them — is filled with noise
drawn from the same color distribution and kept in constant motion, so a
single frame is uniform static: there's no static patch of pixels for a
frame-differencing pass to key on. The noise inside the letterforms scrolls
one direction; the noise around them scrolls the opposite direction. That
contrast is invisible in a still frame but reads instantly to a human eye
once the animation plays. The animation is also exactly one seamless loop —
the last frame flows straight back into the first with no jump.

## Why

An ordinary CAPTCHA is a single static image, so a bot — or an AI model with
vision — sees exactly what a human sees and can attack it in one shot: OCR,
frame analysis, a single call to a vision model. Because ghost-captcha's
letterforms don't exist in any one frame, an automated solver has to
reconstruct motion across dozens of frames instead of reading a single
image, while a human just watches it play.

This is **not** a guarantee against every automated solver, now or ever — no
CAPTCHA scheme is. As of this writing, it takes considerably more effort for
even a capable AI model to defeat than a static text CAPTCHA, and simpler
solvers (plain OCR, single-frame vision models) can't solve it at all. Treat
it as raising the cost of automation, not as an absolute guarantee, and pair
it with other abuse-prevention measures (rate limiting, behavioral signals)
for anything security-critical.

## Install

```sh
go get github.com/rohitkhatri1st/ghost-captcha
```

## Quick start

```go
import ghostcaptcha "github.com/rohitkhatri1st/ghost-captcha"

text, data, err := ghostcaptcha.GenerateCaptcha(ghostcaptcha.CaptchaOptions{})
// text is the random string the caller solved for; data is the encoded image.
```

`GenerateCaptcha` draws random text from a charset and renders it in one
call. To render specific text instead of a random captcha, call
`GenerateGhost` directly:

```go
data, err := ghostcaptcha.GenerateGhost("HELLO", &ghostcaptcha.GhostOptions{
	TextDrift: ghostcaptcha.TextDriftEllipse,
})
```

See the [package examples](https://pkg.go.dev/github.com/rohitkhatri1st/ghost-captcha#pkg-examples)
for more, including multi-line text and custom encoding, or run the
runnable demo in [examples/basic](examples/basic/main.go):

```sh
go run ./examples/basic
```

## Try it in your browser

No Go install needed: [wasm/main.go](wasm/main.go) compiles this library to
WebAssembly, and [docs/index.html](docs/index.html) is a small static page
that calls it — type text, pick options, get a GIF, entirely client-side
(only GIF works this way; WebM/MP4 need real ffmpeg, which WASM can't run).

To host it on GitHub Pages: Settings → Pages → Source: "GitHub Actions".
[.github/workflows/pages.yml](.github/workflows/pages.yml) then builds
`docs/main.wasm` and deploys `docs/` on every push to `main` — the compiled
binary is never committed, so `go get`-ing this package never downloads it.
To build and try it locally instead:

```sh
GOOS=js GOARCH=wasm go build -o docs/main.wasm ./wasm
```

## Output formats

`GhostOptions.Format` selects the container:

| Format | Dependencies | Notes |
| --- | --- | --- |
| `FormatGIF` (default) | none | Larger files, but no external dependency |
| `FormatWebM` | `ffmpeg` on `PATH` | Smaller/faster than GIF |
| `FormatMP4` | `ffmpeg` on `PATH` | For players/browsers without WebM |

If you need frames in a format this package doesn't encode to, call
`GenerateGhostFrames` to get the rendered frames directly, or set
`GhostOptions.Encoder` to plug your own encoding step into `GenerateGhost`
while still reusing its rendering.

## Options

`GhostOptions` controls every visual and timing detail. All fields are
optional — anything left at its zero value gets a sensible default (see the
[godoc](https://pkg.go.dev/github.com/rohitkhatri1st/ghost-captcha#GhostOptions)
for exact defaults):

| Field | Controls |
| --- | --- |
| `FontSize`, `FontBytes` | Font and its point size |
| `LetterSpacing` | Extra horizontal gap between characters |
| `Width`, `Height` | Canvas size (defaults to fit the text itself) |
| `NoiseColorA`, `NoiseColorB` | The two ends of the noise's color range |
| `BackgroundCellSize`, `TextCellSize` | Noise grain size, background vs. letterforms |
| `Frames`, `FrameDelay` | Animation length and per-frame delay |
| `Loop` | GIF loop count (0 = forever) |
| `TextDrift` | How the letterforms wander from frame to frame |
| `Format` | Output container: GIF, WebM, or MP4 |
| `Encoder` | Override the default GIF/WebM/MP4 encoding step |

`CaptchaOptions` (for `GenerateCaptcha`) additionally controls the generated
text itself via `Length` and `Charset`, plus an embedded `Ghost GhostOptions`
for every rendering option above.

## Testing

```sh
go test ./...
```

Tests that exercise WebM/MP4 output skip automatically if `ffmpeg` isn't on
`PATH`.

## License

[Apache License 2.0](LICENSE)

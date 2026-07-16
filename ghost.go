package ghostcaptcha

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"os/exec"
	"runtime"
	"sync"

	"golang.org/x/image/font"
)

// FrameEncoder turns rendered ghost frames into final output bytes. opts is
// whatever GhostOptions GenerateGhost was called with, so an encoder can
// read Width, Height, FrameDelay, Loop, Format, or any options a custom
// encoder cares about. Setting GhostOptions.Encoder overrides GenerateGhost's
// own GIF/WebM/MP4 encoding with this function instead.
type FrameEncoder func(frames []*image.Paletted, delays []int, opts *GhostOptions) ([]byte, error)

// GenerateGhost renders text as an animated "ghost text" GIF or video. The
// letterforms are filled with noise that scrolls one direction, surrounded
// by noise that scrolls the opposite direction. A single frame is just
// uniform static — every cell, inside the letterforms or out, is drawn from
// the same distribution and keeps moving every frame, so there's no static
// patch of pixels for a frame-differencing pass to pick the text out by —
// but once the animation plays, the two opposing motions make the
// letterforms stand out to a human eye.
//
// The animation is exactly one seamless loop: the last frame flows straight
// back into the first with no jump. opts.Format selects the output file
// format; the video formats (FormatWebM, FormatMP4) need ffmpeg installed
// and on PATH, unless opts.Encoder is set to something that doesn't need it.
//
// GenerateGhost is just GenerateGhostFrames plus an encoding step. Call
// GenerateGhostFrames directly to get the raw frames and encode them
// yourself, or set opts.Encoder to reuse GenerateGhost's rendering with your
// own encoding logic in place of its default GIF/WebM/MP4 encoders.
func GenerateGhost(text string, opts *GhostOptions) ([]byte, error) {
	opts.setDefaults()

	encode := opts.Encoder
	if encode == nil {
		encode = defaultFrameEncoder(opts.Format)

		// Check ffmpeg is available before spending time rendering frames,
		// so a missing dependency fails fast with a clear error instead of
		// surfacing deep inside encodeVideo after all that work. Only
		// GenerateGhost's own video encoders need ffmpeg; a custom Encoder
		// is responsible for checking its own dependencies.
		if opts.Format != FormatGIF {
			if _, err := exec.LookPath("ffmpeg"); err != nil {
				return nil, fmt.Errorf("ghostfont: %s output requires ffmpeg on PATH: %w", opts.Format, err)
			}
		}
	}

	frames, delays, err := GenerateGhostFrames(text, opts)
	if err != nil {
		return nil, err
	}

	data, err := encode(frames, delays, opts)
	if err != nil {
		return nil, fmt.Errorf("ghostfont: encoding %s: %w", opts.Format, err)
	}
	return data, nil
}

// GenerateGhostFrames renders the raw ghost-text animation frames for text,
// without encoding them into any container format. delays holds each
// frame's display duration in centiseconds (1/100s), matching image/gif's
// Delay field convention, at index i for frames[i]. Use this directly when
// you want to encode the animation yourself — into GIF/WebM/MP4 with
// different settings than GenerateGhost's, or into a format it doesn't
// support at all.
func GenerateGhostFrames(text string, opts *GhostOptions) (frames []*image.Paletted, delays []int, err error) {
	if text == "" {
		return nil, nil, errors.New("ghostfont: text must not be empty")
	}
	text = lineEndingReplacer.Replace(text)
	opts.setDefaults()

	face, err := loadFace(opts.FontBytes, opts.FontSize)
	if err != nil {
		return nil, nil, fmt.Errorf("ghostfont: loading font: %w", err)
	}
	defer face.Close()

	frames, delays = renderGhostFrames(face, text, opts)
	return frames, delays, nil
}

// defaultFrameEncoder returns the FrameEncoder GenerateGhost uses when
// opts.Encoder is left unset: encodeGIF for FormatGIF, encodeVideoFrames
// (ffmpeg-backed) for everything else.
func defaultFrameEncoder(format Format) FrameEncoder {
	if format == FormatGIF {
		return encodeGIF
	}
	return encodeVideoFrames
}

// encodeGIF is the default FrameEncoder for FormatGIF: a paletted, looping
// animated GIF using only the Go standard library.
func encodeGIF(frames []*image.Paletted, delays []int, opts *GhostOptions) ([]byte, error) {
	g := &gif.GIF{Image: frames, Delay: delays, LoopCount: opts.Loop}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, fmt.Errorf("encoding gif: %w", err)
	}
	return buf.Bytes(), nil
}

// encodeVideoFrames is the default FrameEncoder for FormatWebM and
// FormatMP4: it pipes frames to ffmpeg via encodeVideo.
func encodeVideoFrames(frames []*image.Paletted, delays []int, opts *GhostOptions) ([]byte, error) {
	fps := fmt.Sprintf("100/%d", opts.FrameDelay)
	return encodeVideo(frames, opts.Width, opts.Height, fps, opts.Format)
}

func renderGhostFrames(face font.Face, text string, opts *GhostOptions) ([]*image.Paletted, []int) {
	bgDir := scrollDown
	textDir := opposite(bgDir)
	frameCount := backgroundFrameCount(opts, bgDir)

	palette := buildNoisePalette(opts.NoiseColorA, opts.NoiseColorB, noisePaletteSteps)

	bgCols, bgRows := cellGrid(opts.Width, opts.Height, opts.BackgroundCellSize)
	bgTile := newNoiseTile(bgCols, bgRows, len(palette))

	textCols, textRows := cellGrid(opts.Width, opts.Height, opts.TextCellSize)
	textTile := newNoiseTile(textCols, textRows, len(palette))

	shape := textShape(face, text, opts.LetterSpacing)
	anchors := textFrameAnchors(opts.Width, opts.Height, shape.Rect.Dx(), shape.Rect.Dy(), frameCount, opts.TextDrift)

	frames := make([]*image.Paletted, frameCount)
	delays := make([]int, frameCount)

	// Frames are independent given the shared, read-only shape/tiles/
	// anchors computed above, so render them concurrently: each goroutine
	// only ever writes its own frames[i], and the worker count is capped
	// at GOMAXPROCS so this doesn't oversubscribe the CPU.
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i := range frames {
		delays[i] = opts.FrameDelay
		sem <- struct{}{}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			frames[i] = renderGhostFrame(shape, anchors[i], bgTile, textTile, palette, opts, i, bgDir, textDir)
		}(i)
	}
	wg.Wait()
	return frames, delays
}

// renderGhostFrame draws one frame: canvas pixels the letterform mask
// covers (shape anchored at anchor) sample textTile scrolling in textDir;
// every other pixel samples bgTile scrolling in bgDir. Both tiles draw
// from the same palette, so on its own a single frame is indistinguishable
// static — only the animation reveals the letterforms.
func renderGhostFrame(shape *image.Alpha, anchor image.Point, bgTile, textTile *noiseTile, palette color.Palette, opts *GhostOptions, shift int, bgDir, textDir scrollDirection) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, opts.Width, opts.Height), palette)
	for y := 0; y < opts.Height; y++ {
		rowOff := y * img.Stride
		for x := 0; x < opts.Width; x++ {
			if isTextPixel(shape, anchor, x, y) {
				img.Pix[rowOff+x] = sampleTile(textTile, x, y, opts.TextCellSize, shift, textDir)
			} else {
				img.Pix[rowOff+x] = sampleTile(bgTile, x, y, opts.BackgroundCellSize, shift, bgDir)
			}
		}
	}
	return img
}

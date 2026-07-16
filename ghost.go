package ghostcaptcha

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"

	"golang.org/x/image/font"
)

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
// format; the video formats need ffmpeg installed and on PATH.
func GenerateGhost(text string, opts *GhostOptions) ([]byte, error) {
	if text == "" {
		return nil, errors.New("ghostfont: text must not be empty")
	}
	text = lineEndingReplacer.Replace(text)
	opts.setDefaults()

	face, err := loadFace(opts.FontBytes, opts.FontSize)
	if err != nil {
		return nil, fmt.Errorf("ghostfont: loading font: %w", err)
	}
	defer face.Close()

	frames, delays := renderGhostFrames(face, text, opts)

	g := &gif.GIF{Image: frames, Delay: delays, LoopCount: opts.Loop}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, fmt.Errorf("ghostfont: encoding gif: %w", err)
	}
	return buf.Bytes(), nil
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
	for i := range frames {
		frames[i] = renderGhostFrame(shape, anchors[i], bgTile, textTile, palette, opts, i, bgDir, textDir)
		delays[i] = opts.FrameDelay
	}
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
